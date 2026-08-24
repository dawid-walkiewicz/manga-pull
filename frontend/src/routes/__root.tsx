import { TanStackDevtools } from "@tanstack/react-devtools";
import { type QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRootRouteWithContext, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { AppSidebar } from "#/components/sidebar";
import {
	SidebarInset,
	SidebarProvider,
	SidebarTrigger,
} from "#/components/ui/sidebar";
import { TooltipProvider } from "#/components/ui/tooltip";
import TanStackQueryDevtools from "../integrations/tanstack-query/devtools";

interface MyRouterContext {
	queryClient: QueryClient;
}

export const Route = createRootRouteWithContext<MyRouterContext>()({
	head: () => ({
		meta: [
			{
				title: "Manga Pull",
			},
		],
	}),
	component: RootLayout,
});

function RootLayout() {
	const { queryClient } = Route.useRouteContext();

	return (
		<QueryClientProvider client={queryClient}>
			<div className="font-sans antialiased wrap-anywhere selection:bg-[rgba(79,184,178,0.24)]">
				<TooltipProvider>
					<SidebarProvider>
						<AppSidebar />
						<SidebarInset>
							<SidebarTrigger />
							<Outlet />
						</SidebarInset>
					</SidebarProvider>
				</TooltipProvider>
				<TanStackDevtools
					config={{
						position: "bottom-right",
					}}
					plugins={[
						{
							name: "Tanstack Router",
							render: <TanStackRouterDevtoolsPanel />,
						},
						TanStackQueryDevtools,
					]}
				/>
			</div>
		</QueryClientProvider>
	);
}
