import type { FC } from "react";
import {
	Sidebar as BaseSidebar,
	SettingsSidebarNavItem as SidebarNavItem,
} from "#/components/Sidebar/Sidebar";
import type { Permissions } from "#/modules/permissions";

interface AISettingsSidebarViewProps {
	/** Site-wide permissions. */
	permissions: Permissions;
}

const AISettingsSidebarView: FC<AISettingsSidebarViewProps> = ({
	permissions,
}) => {
	return (
		<BaseSidebar>
			<div className="flex flex-col gap-1">
				{permissions.viewDeploymentConfig && (
					<SidebarNavItem href="/ai/settings/governance">
						AI Governance
					</SidebarNavItem>
				)}
				{permissions.viewAIGatewayKeys && (
					<SidebarNavItem href="/ai/settings/gateway-keys">
						AI Gateway keys
					</SidebarNavItem>
				)}
				{permissions.viewAnyAIProvider && (
					<SidebarNavItem href="/ai/settings" end>
						Providers
					</SidebarNavItem>
				)}
				{permissions.editDeploymentConfig && (
					<>
						<SidebarNavItem href="/ai/settings/agents">
							Coder Agents
						</SidebarNavItem>
						<div className="flex flex-col gap-1 pl-3">
							<SidebarNavItem href="/ai/settings/agent-settings">
								Agent settings
							</SidebarNavItem>
							<SidebarNavItem href="/ai/settings/models">Models</SidebarNavItem>
							<SidebarNavItem href="/ai/settings/mcp-servers">
								MCP servers
							</SidebarNavItem>
							<SidebarNavItem href="/ai/settings/templates">
								Templates
							</SidebarNavItem>
							<SidebarNavItem href="/ai/settings/spend">Spend</SidebarNavItem>
							<SidebarNavItem href="/ai/settings/instructions">
								Instructions
							</SidebarNavItem>
							<SidebarNavItem href="/ai/settings/lifecycle">
								Lifecycle
							</SidebarNavItem>
						</div>
					</>
				)}
			</div>
		</BaseSidebar>
	);
};

export default AISettingsSidebarView;
