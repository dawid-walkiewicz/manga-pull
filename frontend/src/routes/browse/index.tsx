import { createFileRoute, Link } from "@tanstack/react-router";
import { listPlugins } from "#/lib/api";

export const Route = createFileRoute("/browse/")({
	loader: () => listPlugins(),
	component: RouteComponent,
});

function RouteComponent() {
	const plugins = Route.useLoaderData();

	return (
		<div className="p-8">
			<div className="flex flex-col gap-2">
				{plugins.map((plugin) => (
					<Link
						className="flex flex-row justify-between rounded-2xl p-2 bg-primary"
						key={plugin.id}
						to={"/browse/$pluginId"}
						params={{ pluginId: plugin.id }}
					>
						<span className="text-xl">{plugin.name}</span>
					</Link>
				))}
			</div>
		</div>
	);
}
