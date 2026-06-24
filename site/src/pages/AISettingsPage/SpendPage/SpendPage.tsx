import type { FC } from "react";
import {
	SettingsHeader,
	SettingsHeaderTitle,
} from "#/components/SettingsHeader/SettingsHeader";
import { useAuthenticated } from "#/hooks/useAuthenticated";
import { RequirePermission } from "#/modules/permissions/RequirePermission";
import { pageTitle } from "#/utils/page";

const SpendPage: FC = () => {
	const { permissions } = useAuthenticated();

	return (
		<RequirePermission isFeatureVisible={permissions.editDeploymentConfig}>
			<title>{pageTitle("Spend", "AI Settings")}</title>
			<SettingsHeader>
				<SettingsHeaderTitle>Spend</SettingsHeaderTitle>
			</SettingsHeader>
		</RequirePermission>
	);
};

export default SpendPage;
