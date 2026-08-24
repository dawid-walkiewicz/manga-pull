import { Link, type LinkProps } from "@tanstack/react-router";
import { Tooltip, TooltipContent, TooltipTrigger } from "./ui/tooltip";

export type GalleryTitle<TId extends string | number> = {
	id: TId;
	title: string;
	cover: string;
};

export type GalleryProps<TId extends string | number> = {
	titles: GalleryTitle<TId>[];
	getLink: (title: GalleryTitle<TId>) => LinkProps;
};

export default function Gallery<TId extends string | number>({
	titles,
	getLink,
}: GalleryProps<TId>) {
	return (
		<div className="grid grid-cols-[repeat(auto-fill,minmax(10rem,1fr))] gap-x-10 gap-y-4">
			{titles.map((title) => {
				const link = getLink(title);

				return (
					<Link
						className="relative block overflow-hidden w-48"
						key={title.id}
						{...link}
					>
						{title.cover ? (
							<img
								src={title.cover}
								alt={`${title.title} cover`}
								className="aspect-3/4 w-full object-cover"
							/>
						) : null}
						<div className="absolute inset-x-0 bottom-0 bg-linear-to-t from-black/80 to-transparent px-2 pt-10">
							<Tooltip>
								<TooltipTrigger>
									<span className="line-clamp-2 text-white text-sm text-left">
										{title.title}
									</span>
								</TooltipTrigger>
								<TooltipContent>
									<p>{title.title}</p>
								</TooltipContent>
							</Tooltip>
						</div>
					</Link>
				);
			})}
		</div>
	);
}
