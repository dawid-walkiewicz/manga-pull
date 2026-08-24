import { createFileRoute } from "@tanstack/react-router";
import { RotateCw } from "lucide-react";
import { Button } from "#/components/ui/button";
import { Switch } from "#/components/ui/switch";
import {
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
				<div className="flex flex-row justify-between" key={plugin.id}>
					<div className="flex gap-4">
						<span className="text-xl">{plugin.name}</span>
						<span className="self-end text-sm">{plugin.version}</span>
					</div>
					<Switch
						defaultChecked={plugin.Enabled}
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
