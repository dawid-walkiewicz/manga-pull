import { createFileRoute } from "@tanstack/react-router";
import { RotateCw } from "lucide-react";
import { Button } from "#/components/ui/button";
import { Switch } from "#/components/ui/switch";
import {
	API_BASE_URL,
	disablePlugin,
	enablePlugin,
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
							src={`${API_BASE_URL}/api/plugins/${plugin.id}/icon`}
							alt={`${plugin.name} icon`}
							className="size-8"
						/>
						<span className="text-xl">{plugin.name}</span>
						<span className="text-sm">{plugin.version}</span>
					</div>
					<Switch
						defaultChecked={plugin.enabled}
						onCheckedChange={(checked: boolean) => {
							if (checked) {
								enablePlugin(plugin.id);
							} else {
								disablePlugin(plugin.id);
							}
						}}
					/>
				</div>
			))}
		</div>
	);
}
