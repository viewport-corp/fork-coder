import type { FC } from "react";
import {
	SettingsHeader,
	SettingsHeaderTitle,
} from "#/components/SettingsHeader/SettingsHeader";
import { useAuthenticated } from "#/hooks/useAuthenticated";
import { RequirePermission } from "#/modules/permissions/RequirePermission";
import { pageTitle } from "#/utils/page";

const CoderAgentsPage: FC = () => {
	const { permissions } = useAuthenticated();

	return (
		<RequirePermission isFeatureVisible={permissions.editDeploymentConfig}>
			<title>{pageTitle("Coder Agents", "AI Settings")}</title>
			<SettingsHeader>
				<SettingsHeaderTitle>Coder Agents</SettingsHeaderTitle>
			</SettingsHeader>
		</RequirePermission>
	);
};

export default CoderAgentsPage;
