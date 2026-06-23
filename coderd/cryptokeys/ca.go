package cryptokeys

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"math/big"
	"time"

	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/database/dbtime"
)

const (
	caCertPEMBlockType = "CERTIFICATE"
	caKeyPEMBlockType  = "EC PRIVATE KEY"

	// clockSkewTolerance backdates the CA certificate's NotBefore and extends
	// its NotAfter so that replicas with mildly skewed clocks still accept it.
	clockSkewTolerance = time.Hour
)

// NATSCA is the parsed state of the nats_ca crypto key feature at one point in
// time. The CA signs the ephemeral leaf certificates that replicas use for
// NATS cluster mTLS.
//
// Callers that need to react to CA rotation (re-minting leaves and reloading
// the NATS server config) should poll FetchOrCreateInitialNATSCA and compare
// Sequence to detect when the active CA has changed.
type NATSCA struct {
	// Sequence is the crypto_keys sequence of the active row.
	Sequence int32
	// Cert is the active CA certificate used to sign leaf certificates.
	Cert *x509.Certificate
	// Key is the active CA private key.
	Key crypto.Signer
	// TrustBundle contains the certificates of all currently valid CA rows,
	// including Cert. During a rotation overlap window it has two entries;
	// installing the full bundle as the trust root lets replicas on either
	// side of a rotation verify each other.
	TrustBundle []*x509.Certificate
}

// FetchOrCreateInitialNATSCA returns the current NATS cluster CA, creating the
// initial CA if no valid one exists. The NATS pubsub is constructed before the
// key rotator starts, so on fresh deployments the CA row will not exist at
// first fetch; this function bootstraps it under an advisory lock so the
// rotator has a CA to take over.
//
// Ordering constraint: the FIRST call (the one that creates the CA) must run to
// completion before starting the key rotator (cryptokeys.StartRotator). The
// bootstrap insert is only collision-free with respect to the rotator under
// that ordering; see the lock comment below. In production the server satisfies
// this by fetching the CA before calling StartRotator. Once a CA exists this
// function is a read-only fast path, so later callers (e.g. polling for
// rotation) may run concurrently with the rotator safely.
//
// keyDuration sizes the bootstrap certificate's validity window and must match
// the rotator's key duration (database.DefaultKeyDuration in production) so the
// bootstrap CA and rotator-minted CAs have consistent lifetimes.
func FetchOrCreateInitialNATSCA(ctx context.Context, db database.Store, keyDuration time.Duration) (*NATSCA, error) {
	//nolint:gocritic // The CA accessor requires the same crypto key access as the rotator.
	ctx = dbauthz.AsKeyRotator(ctx)

	now := dbtime.Now()

	keys, err := db.GetCryptoKeysByFeature(ctx, database.CryptoKeyFeatureNatsCa)
	if err != nil {
		return nil, xerrors.Errorf("get crypto keys by feature: %w", err)
	}

	ca, ok, err := parseNATSCAKeys(keys, now)
	if err != nil {
		return nil, err
	}
	if ok {
		return ca, nil
	}

	// No active CA exists, so create one under the rotator's advisory lock
	// (LockIDCryptoKeyRotation). The lock plus a read-committed re-check after
	// acquiring it keeps this startup path collision-free: it either inserts
	// the first nats_ca row or sees a row a peer (or the rotator) already
	// committed.
	//
	// This does NOT make the rotator itself collision-proof against this path.
	// The rotator reads under Repeatable Read with a snapshot fixed when it
	// acquires the lock, so a row committed here while it waits is invisible to
	// it and its insert would hit a unique violation (which self-heals on its
	// next cycle). We rely on the ordering constraint documented above
	// (bootstrap completes before StartRotator) so the two never actually
	// contend; the shared lock protects the startup path, not the rotator.
	err = db.InTx(func(tx database.Store) error {
		err := tx.AcquireLock(ctx, database.LockIDCryptoKeyRotation)
		if err != nil {
			return xerrors.Errorf("acquire lock: %w", err)
		}

		keys, err = tx.GetCryptoKeysByFeature(ctx, database.CryptoKeyFeatureNatsCa)
		if err != nil {
			return xerrors.Errorf("get crypto keys by feature: %w", err)
		}

		// Recompute now after acquiring the lock: a concurrent creator may
		// have committed a row with a StartsAt later than the time captured
		// before we blocked on the lock.
		now = dbtime.Now()
		var ok bool
		ca, ok, err = parseNATSCAKeys(keys, now)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}

		secret, err := generateCASecret(now, keyDuration)
		if err != nil {
			return xerrors.Errorf("generate CA secret: %w", err)
		}

		latestKey, err := tx.GetLatestCryptoKeyByFeature(ctx, database.CryptoKeyFeatureNatsCa)
		if err != nil && !xerrors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("get latest key: %w", err)
		}

		newKey, err := tx.InsertCryptoKey(ctx, database.InsertCryptoKeyParams{
			Feature:  database.CryptoKeyFeatureNatsCa,
			Sequence: latestKey.Sequence + 1,
			Secret: sql.NullString{
				String: secret,
				Valid:  true,
			},
			// Set by dbcrypt if it's required.
			SecretKeyID: sql.NullString{},
			StartsAt:    now,
		})
		if err != nil {
			return xerrors.Errorf("insert crypto key: %w", err)
		}

		ca, ok, err = parseNATSCAKeys([]database.CryptoKey{newKey}, now)
		if err != nil {
			return err
		}
		if !ok {
			return xerrors.New("inserted NATS CA is not usable for signing")
		}
		return nil
	}, &database.TxOptions{
		// Read committed (the default) is required here: with repeatable
		// read, the snapshot is taken before the advisory lock is granted,
		// so the post-lock re-check would not see a row committed by a
		// concurrent creator and we would insert a duplicate.
		TxIdentifier: "fetch_nats_ca",
	})
	if err != nil {
		return nil, err
	}
	return ca, nil
}

// parseNATSCAKeys builds a NATSCA from the database rows for the nats_ca
// feature. Rows must be ordered by sequence descending (the order returned by
// GetCryptoKeysByFeature). The active CA is the newest row that is usable for
// signing; the trust bundle contains the certificates of every row that is
// still valid for verification. The boolean reports whether a row could act
// as the active CA.
func parseNATSCAKeys(keys []database.CryptoKey, now time.Time) (*NATSCA, bool, error) {
	ca := &NATSCA{}
	for _, key := range keys {
		if !key.CanVerify(now) {
			continue
		}
		cert, signer, err := parseCASecret(key.Secret.String)
		if err != nil {
			return nil, false, xerrors.Errorf("parse CA secret for sequence %d: %w", key.Sequence, err)
		}
		ca.TrustBundle = append(ca.TrustBundle, cert)
		if ca.Cert == nil && key.CanSign(now) {
			ca.Sequence = key.Sequence
			ca.Cert = cert
			ca.Key = signer
		}
	}
	if ca.Cert == nil {
		return nil, false, nil
	}
	return ca, true, nil
}

// generateCASecret generates a new self-signed CA certificate and private key
// for signing NATS cluster leaf certificates, PEM-encoded into a single
// bundle for storage in the crypto_keys secret column.
//
// anchorTime is the key row's starts_at (which may be in the future for a
// rotated-in key). keyDuration is the rotator's key duration: the row stays the
// active signer for that long. The certificate must outlive that window plus
// the longest leaf it could sign (NATSCALeafValidity) plus clock-skew slack, so
// leaves minted just before rotation still chain to a valid CA.
func generateCASecret(anchorTime time.Time, keyDuration time.Duration) (string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", xerrors.Errorf("generate key: %w", err)
	}

	// 128-bit random serial per CA/Browser Forum conventions.
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", xerrors.Errorf("generate serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "coder-nats-ca",
		},
		NotBefore:             anchorTime.Add(-clockSkewTolerance),
		NotAfter:              anchorTime.Add(keyDuration + NATSCALeafValidity + clockSkewTolerance),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if err != nil {
		return "", xerrors.Errorf("create certificate: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", xerrors.Errorf("marshal private key: %w", err)
	}

	var secret []byte
	secret = append(secret, pem.EncodeToMemory(&pem.Block{Type: caCertPEMBlockType, Bytes: der})...)
	secret = append(secret, pem.EncodeToMemory(&pem.Block{Type: caKeyPEMBlockType, Bytes: keyDER})...)
	return string(secret), nil
}

// parseCASecret parses a PEM bundle produced by generateCASecret back into
// the CA certificate and private key.
func parseCASecret(secret string) (*x509.Certificate, crypto.Signer, error) {
	var (
		cert *x509.Certificate
		key  *ecdsa.PrivateKey
	)
	rest := []byte(secret)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		switch block.Type {
		case caCertPEMBlockType:
			if cert != nil {
				return nil, nil, xerrors.New("multiple certificates in CA secret")
			}
			var err error
			cert, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, xerrors.Errorf("parse certificate: %w", err)
			}
		case caKeyPEMBlockType:
			if key != nil {
				return nil, nil, xerrors.New("multiple private keys in CA secret")
			}
			var err error
			key, err = x509.ParseECPrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, xerrors.Errorf("parse private key: %w", err)
			}
		default:
			return nil, nil, xerrors.Errorf("unexpected PEM block type: %q", block.Type)
		}
	}
	if cert == nil {
		return nil, nil, xerrors.New("no certificate in CA secret")
	}
	if key == nil {
		return nil, nil, xerrors.New("no private key in CA secret")
	}
	if !key.PublicKey.Equal(cert.PublicKey) {
		return nil, nil, xerrors.New("private key does not match certificate")
	}
	return cert, key, nil
}
