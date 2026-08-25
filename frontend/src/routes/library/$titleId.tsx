import { createFileRoute } from "@tanstack/react-router";
import { ExternalLink, Pin, RotateCw } from "lucide-react";
import { Button, buttonVariants } from "#/components/ui/button";
import { deleteTitle, getTitle, refreshPluginTitle } from "#/lib/api";

export const Route = createFileRoute("/library/$titleId")({
	params: {
		parse: (params) => ({
			...params,
			titleId: Number(params.titleId),
		}),
		stringify: (params) => ({
			titleId: String(params.titleId),
		}),
	},
	loader: async ({ params }) => {
		return getTitle(params.titleId);
	},
	component: RouteComponent,
});

function RouteComponent() {
	const titleDetails = Route.useLoaderData();
	const navigate = Route.useNavigate();

	async function handleDelete() {
		try {
			await deleteTitle(titleDetails.id);
			await navigate({
				to: "/browse/$pluginId/$titleId",
				params: {
					pluginId: titleDetails.pluginId,
					titleId: titleDetails.remoteId,
				},
			});
		} catch (err) {
			console.log(err);
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
					<Button variant="destructive" onClick={handleDelete} className="w-32">
						<Pin data-icon="inline-start" /> Saved
					</Button>
					{titleDetails.url && (
						<a
							href={titleDetails.url}
							className={buttonVariants({ variant: "ghost", size: "icon" })}
						>
							<ExternalLink />
						</a>
					)}
				</div>
				<span>{titleDetails.description}</span>
			</div>
			<div>
				<div className="flex flex-row justify-end">
					<Button
						variant="ghost"
						size="icon"
						onClick={() =>
							refreshPluginTitle(titleDetails.pluginId, titleDetails.remoteId)
						}
					>
						<RotateCw />
					</Button>
				</div>
				<div className="flex flex-col gap-4">
					{titleDetails.chapters.map((chapter) => (
						<span key={chapter.id}>
							{chapter.number}:{chapter.title}
						</span>
					))}
				</div>
			</div>
		</div>
	);
}
