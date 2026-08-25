import { createFileRoute, redirect } from "@tanstack/react-router";
import { Pin } from "lucide-react";
import { useState } from "react";
import { Button } from "#/components/ui/button";
import { getPluginTitle, savePluginTitle } from "#/lib/api";

export const Route = createFileRoute("/browse/$pluginId/$titleId")({
	loader: async ({ params }) => {
		const title = await getPluginTitle(params.pluginId, params.titleId);

		if (title.savedId) {
			throw redirect({
				to: "/library/$titleId",
				params: {
					titleId: title.savedId,
				},
			});
		}

		return title;
	},
	component: RouteComponent,
});

function RouteComponent() {
	const { pluginId, titleId } = Route.useParams();
	const titleDetails = Route.useLoaderData();

	const navigate = Route.useNavigate();
	const [saving, setSaving] = useState(false);

	async function handleSave() {
		try {
			setSaving(true);

			const response = await savePluginTitle(pluginId, titleId);

			await navigate({
				to: "/library/$titleId",
				params: {
					titleId: response.id,
				},
			});
		} finally {
			setSaving(false);
		}
	}

	return (
		<div className="grid grid-cols-2 gap-6 p-8">
			<div className="flex flex-col gap-4">
				<span className="text-2xl w-full font-bold">{titleDetails.title}</span>
				<div className="flex flex-row gap-x-4">
					{titleDetails.cover ? (
						<img src={titleDetails.cover} alt={`${titleDetails.title} cover`} />
					) : null}
					<div className="flex flex-col">
						<span>Author: {titleDetails.author}</span>
						<span>Artist: {titleDetails.artist}</span>
						<span>Status: {titleDetails.status}</span>
						<span className="">
							Alternative titles: {titleDetails.alternativeTitles.join(", ")}
						</span>
					</div>
				</div>
				<div>
					<Button variant="outline" onClick={handleSave} className="w-32">
						<Pin data-icon="inline-start" /> {saving ? "Saving" : "Save"}
					</Button>
				</div>
				<span>{titleDetails.description}</span>
			</div>
			<div className="flex flex-col gap-4">
				{titleDetails.chapters.map((chapter) => (
					<span key={chapter.id}>
						{chapter.number}:{chapter.title}
					</span>
				))}
			</div>
		</div>
	);
}
