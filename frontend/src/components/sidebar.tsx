import { Link } from "@tanstack/react-router";
import { BookOpen, Compass, Plug } from "lucide-react";
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarGroupContent,
	SidebarGroupLabel,
	SidebarHeader,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
} from "@/components/ui/sidebar";

const navigationItems = [
	{ title: "Library", to: "/library", icon: BookOpen },
	{ title: "Browse", to: "/browse", icon: Compass },
	{ title: "Plugins", to: "/plugins", icon: Plug },
] as const;

export function AppSidebar() {
	return (
		<Sidebar>
			<SidebarHeader className="px-4 py-3">
				<div className="font-semibold text-sidebar-foreground text-sm">
					Manga Pull
				</div>
			</SidebarHeader>
			<SidebarContent>
				<SidebarGroup>
					<SidebarGroupLabel>Navigation</SidebarGroupLabel>
					<SidebarGroupContent>
						<SidebarMenu>
							{navigationItems.map((item) => (
								<SidebarMenuItem key={item.to}>
									<SidebarMenuButton asChild tooltip={item.title}>
										<Link
											activeOptions={{ exact: true }}
											activeProps={{
												className:
													"bg-sidebar-accent font-medium text-sidebar-accent-foreground",
											}}
											to={item.to}
										>
											<item.icon />
											<span>{item.title}</span>
										</Link>
									</SidebarMenuButton>
								</SidebarMenuItem>
							))}
						</SidebarMenu>
					</SidebarGroupContent>
				</SidebarGroup>
			</SidebarContent>
			<SidebarFooter />
		</Sidebar>
	);
}
