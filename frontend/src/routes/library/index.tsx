import { createFileRoute } from "@tanstack/react-router";
import Gallery, { type GalleryTitle } from "#/components/gallery";
import { listTitles } from "#/lib/api";

export const Route = createFileRoute("/library/")({
	loader: () => listTitles(),
	component: RouteComponent,
});

function RouteComponent() {
	const summaries = Route.useLoaderData();

	return (
		<div className="p-8">
			<Gallery
				titles={summaries.map((summary) => {
					return {
						id: summary.id,
						title: summary.title,
						cover: summary.cover,
					} as GalleryTitle<number>;
				})}
				getLink={(title) => ({
					to: "/library/$titleId",
					params: {
						titleId: title.id,
					},
				})}
			/>
		</div>
	);
}
