import { createFileRoute } from "@tanstack/react-router";
import { Download, ExternalLink, Pin, RotateCw } from "lucide-react";
import { useState } from "react";
import { Button, buttonVariants } from "#/components/ui/button";
import { Card } from "#/components/ui/card";
import { Spinner } from "#/components/ui/spinner";
import {
	deleteTitle,
	downloadChapter,
	getTitle,
	refreshTitle,
	type SavedChapter,
} from "#/lib/api";
import { formatTime } from "#/lib/date";

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

function ChapterItem({ chapter }: { chapter: SavedChapter }) {
	const [isLoading, setIsLoading] = useState(false);

	const handleClick = async () => {
		setIsLoading(true);
		try {
			await downloadChapter(chapter.id);
		} catch (err) {
			console.log(err);
		} finally {
			setIsLoading(false);
		}
	};

	return (
		<Card key={chapter.id} className="px-4 py-4 gap-0">
			<div className="flex flex-row justify-between">
				<div className="flex flex-col">
					<span className="font-semibold">{`Chapter ${chapter.number}${chapter.title ? `: ${chapter.title}` : ""}`}</span>
					<span className="text-xs">
						{chapter.publishedAt && formatTime(chapter.publishedAt)}{" "}
						{chapter.downloaded && "Downloaded"}
					</span>
				</div>
				{!chapter.downloaded &&
					(isLoading ? (
						<Spinner />
					) : (
						<Button size="icon" variant="ghost" onClick={handleClick}>
							<Download />
						</Button>
					))}
			</div>
		</Card>
	);
}

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
			const details = await refreshTitle(title.id);
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
				<div className="flex flex-col gap-2">
					{title.chapters.map((chapter) => (
						<ChapterItem key={chapter.id} chapter={chapter} />
					))}
				</div>
			</div>
		</div>
	);
}
