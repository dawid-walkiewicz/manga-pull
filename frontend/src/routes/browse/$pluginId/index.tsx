import { createFileRoute } from "@tanstack/react-router";
import Gallery, { type GalleryTitle } from "#/components/gallery";
import { browsePluginTitles } from "#/lib/api";

export const Route = createFileRoute("/browse/$pluginId/")({
	loader: async ({ params }) => {
		return browsePluginTitles(params.pluginId);
	},
	component: RouteComponent,
});

function RouteComponent() {
	const { pluginId } = Route.useParams();
	const titleSummaries = Route.useLoaderData();

	return (
		<div className="p-8">
			<Gallery
				titles={titleSummaries.map((summary) => {
					return {
						id: summary.id,
						title: summary.title,
						cover: summary.cover,
					} as GalleryTitle<string>;
				})}
				getLink={(title) => ({
					to: "/browse/$pluginId/$titleId",
					params: {
						pluginId: pluginId,
						titleId: title.id,
					},
				})}
			/>
		</div>
	);
}
