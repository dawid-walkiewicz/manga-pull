import { createFileRoute } from "@tanstack/react-router";
import { ExternalLink, Pin, RotateCw } from "lucide-react";
import { useState } from "react";
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

	const [title, setTitle] = useState(titleDetails);

	async function handleDelete() {
		try {
			await deleteTitle(titleDetails.id);
			await navigate({
				to: "/browse/$pluginId/$titleId",
				params: {
					pluginId: title.pluginId,
					titleId: title.remoteId,
				},
			});
		} catch (err) {
			console.log(err);
		}
	}

	async function handleRefresh() {
		try {
			const details = await refreshPluginTitle(title.pluginId, title.remoteId);
			setTitle(details);
		} catch (err) {
			console.log(err);
		}
	}

	return (
		<div className="grid grid-cols-2 gap-6 p-8">
			<div className="flex flex-col gap-4">
				<span className="text-2xl w-full font-bold">{title.title}</span>
				<div className="flex flex-row gap-x-4">
					{title.cover ? (
						<img src={title.cover} alt={`${title.title} cover`} />
					) : null}
					<div className="flex flex-col">
						<span>Author: {title.author}</span>
						<span>Artist: {title.artist}</span>
						<span>Status: {title.status}</span>
						<span className="">
							Alternative titles: {title.alternativeTitles.join(", ")}
						</span>
					</div>
				</div>
				<div>
					<Button variant="destructive" onClick={handleDelete} className="w-32">
						<Pin data-icon="inline-start" /> Saved
					</Button>
					{title.url && (
						<a
							href={title.url}
							className={buttonVariants({ variant: "ghost", size: "icon" })}
						>
							<ExternalLink />
						</a>
					)}
				</div>
				<span>{title.description}</span>
			</div>
			<div>
				<div className="flex flex-row justify-end">
					<Button variant="ghost" size="icon" onClick={handleRefresh}>
						<RotateCw />
					</Button>
				</div>
				<div className="flex flex-col gap-4">
					{title.chapters.map((chapter) => (
						<span key={chapter.id}>
							{chapter.number}:{chapter.title}
						</span>
					))}
				</div>
			</div>
		</div>
	);
}
