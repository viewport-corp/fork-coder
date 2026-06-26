# Coder — Viewport Deploy Overlay

Self-hosted [Coder](https://github.com/coder/coder) for the **Viewport Infrastructure**
department. Sub-issue **#416**, part of **#404**.

This overlay is `deploy/docker-compose.yml`, derived verbatim from upstream
`coder/coder` `compose.yaml` (pinned at **v2.33.9**) with Viewport guardrail
adjustments. Upstream code is untouched; everything lives under `deploy/`.

## Images
- App: `ghcr.io/coder/coder:v2.33.9` (latest release; pinned via `CODER_VERSION`).
- DB: `postgres:17` (Coder requires Postgres >= 13).

## Ports
- Coder listens on **7080** inside the container (`CODER_HTTP_ADDRESS=0.0.0.0:7080`);
  compose publishes `7080:7080`. (Repo compose authoritatively uses 7080, not the
  older 3000 default — health checks + access URL must target 7080.)
- Postgres 5432 is internal-only (not published; no :80/:443/DNS in scope).

## Required environment (set in Dokploy Environment tab — names only, never commit)
| Var | Notes |
| --- | --- |
| `CODER_VERSION` | Pinned tag. Default `v2.33.9`. |
| `CODER_ACCESS_URL` | REQUIRED, no default. External URL workspaces/users reach Coder at. Must NOT be localhost/127.0.0.1. Internal-only here, e.g. `http://194.163.153.171:7080`. |
| `POSTGRES_USER` | Postgres user. |
| `POSTGRES_PASSWORD` | Postgres password (Dokploy **secret**). |
| `POSTGRES_DB` | Postgres db (default `coder`). |

`CODER_PG_CONNECTION_URL` is built inside the compose from the POSTGRES_* vars and
points at the internal `database` service.

## Volumes
- `coder_data:/var/lib/postgresql/data` — Postgres data, **must persist**.
- `coder_home:/home/coder` — optional in prod (resettable).

## ⚠ Sam-gated blocker — Docker socket mount
Upstream mounts the **default** socket (`/var/run/docker.sock`) so Coder can
provision Docker-based workspace templates. Under Viewport guardrails the default
socket is the **OLD engine = STRICTLY READ-ONLY**. The mount is therefore
**commented out** in this overlay → Coder deploys **control-plane only** (no Docker
workspace templates until a workspace backend is chosen).

Decision needed before enabling workspaces (pick one):
- **(a)** leave commented — control-plane only (current safe default).
- **(b)** mount the **NEW-engine** socket `/var/run/docker-viewport.sock` so Coder
  provisions workspace containers on the new engine only.

**Never** mount `/var/run/docker.sock` (old engine).

## Deploy (Dokploy — verified method)
1. New Dokploy at `http://194.163.153.171:3001/` → **Viewport Infrastructure**
   project → Create Service → **Compose** (Docker Compose, not Stack/Swarm).
2. Provider = Git → this fork (`viewport-corp/fork-coder`), branch `viewport/deploy`,
   compose path `deploy/docker-compose.yml`.
3. Set env vars in the Dokploy Environment tab (Dokploy writes UI vars to `.env`;
   the compose consumes them via `${VAR}` substitution). Password as a Dokploy secret.
4. Ensure the project/server is bound to the **NEW engine**
   (`DOCKER_HOST=unix:///var/run/docker-viewport.sock`) before deploying.

## Health verification (internal only)
- App: `curl -fsS http://127.0.0.1:7080/healthz` → `200 OK`.
- DB: Postgres healthcheck (`pg_isready`) reports healthy before Coder starts.

## Sources (docs-first)
- Upstream `coder/coder` `compose.yaml` (verbatim) + latest release `v2.33.9`.
- coder.com/docs/install/docker; admin setup (`CODER_HTTP_ADDRESS`, `CODER_ACCESS_URL`);
  health-check (`/healthz` -> 200).
- docs.dokploy.com/docs/core/docker-compose (Compose service type; `${VAR}` substitution;
  UI vars -> `.env`).
