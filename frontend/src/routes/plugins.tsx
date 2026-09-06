import { createFileRoute } from "@tanstack/react-router";
import { RotateCw, TriangleAlert } from "lucide-react";
import { Button } from "#/components/ui/button";
import { Switch } from "#/components/ui/switch";
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "#/components/ui/tooltip";
import {
	disablePlugin,
	enablePlugin,
	getPluginIconUrl,
	listPlugins,
	scanPlugins,
} from "#/lib/api";

export const Route = createFileRoute("/plugins")({
	loader: () => listPlugins(),
	component: PluginsPage,
});

function PluginsPage() {
	const plugins = Route.useLoaderData();

	return (
		<div className="p-8">
			<Button variant="ghost" size="icon" onClick={scanPlugins}>
				<RotateCw />
			</Button>
			{plugins.map((plugin) => (
				<div className="flex flex-row justify-between p-2" key={plugin.id}>
					<div className="flex flex-row gap-4 items-center">
						<img
							src={getPluginIconUrl(plugin.id)}
							alt={`${plugin.name} icon`}
							className="size-8"
						/>
						<span className="text-xl">{plugin.name}</span>
						<span className="text-sm">{plugin.version}</span>
					</div>
					<div className="flex flex-row gap-2 items-center">
						{plugin.error && (
							<Tooltip>
								<TooltipTrigger>
									<TriangleAlert className="text-red-500" />
								</TooltipTrigger>
								<TooltipContent side="bottom">{plugin.error}</TooltipContent>
							</Tooltip>
						)}
						<Switch
							defaultChecked={plugin.enabled}
							disabled={plugin.error !== null}
							onCheckedChange={(checked: boolean) => {
								if (checked) {
									enablePlugin(plugin.id);
								} else {
									disablePlugin(plugin.id);
								}
							}}
						/>
					</div>
				</div>
			))}
		</div>
	);
}
