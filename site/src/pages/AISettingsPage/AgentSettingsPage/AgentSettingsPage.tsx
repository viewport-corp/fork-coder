import type { FC } from "react";
import {
	SettingsHeader,
	SettingsHeaderTitle,
} from "#/components/SettingsHeader/SettingsHeader";
import { useAuthenticated } from "#/hooks/useAuthenticated";
import { RequirePermission } from "#/modules/permissions/RequirePermission";
import { pageTitle } from "#/utils/page";

const AgentSettingsPage: FC = () => {
	const { permissions } = useAuthenticated();

	return (
		<RequirePermission isFeatureVisible={permissions.editDeploymentConfig}>
			<title>{pageTitle("Agent settings", "AI Settings")}</title>
			<SettingsHeader>
				<SettingsHeaderTitle>Agent settings</SettingsHeaderTitle>
			</SettingsHeader>
		</RequirePermission>
	);
};

export default AgentSettingsPage;
