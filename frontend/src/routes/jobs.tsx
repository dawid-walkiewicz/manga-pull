import { createFileRoute, useRouter } from "@tanstack/react-router";
import { CirclePause, CirclePlay, CircleX, RotateCw } from "lucide-react";
import { Button } from "#/components/ui/button";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "#/components/ui/table";
import { cancelJob, listJobs, pauseJob, resumeJob, retryJob } from "#/lib/api";

export const Route = createFileRoute("/jobs")({
	loader: () => listJobs(),
	component: RouteComponent,
});

function RouteComponent() {
	const router = useRouter();
	const jobs = Route.useLoaderData();

	function formatJobType(jobType: string): string {
		switch (jobType) {
			case "refresh_title":
				return "Refresh Title";
			case "download_chapter":
				return "Download Chapter";
			default:
				return "Unknown Job Type";
		}
	}

	async function handleCancel(jobId: number) {
		try {
			await cancelJob(jobId);
			await router.invalidate({ sync: true });
		} catch (err) {
			console.log(err);
		}
	}

	async function handlePause(jobId: number) {
		try {
			await pauseJob(jobId);
			await router.invalidate({ sync: true });
		} catch (err) {
			console.log(err);
		}
	}

	async function handleResume(jobId: number) {
		try {
			await resumeJob(jobId);
			await router.invalidate({ sync: true });
		} catch (err) {
			console.log(err);
		}
	}

	async function handleRetry(jobId: number) {
		try {
			await retryJob(jobId);
			await router.invalidate({ sync: true });
		} catch (err) {
			console.log(err);
		}
	}

	function canCancel(jobStatus: string): boolean {
		return ["queued", "running", "paused", "retrying"].includes(jobStatus);
	}

	function canPause(jobStatus: string): boolean {
		return ["queued", "running", "retrying"].includes(jobStatus);
	}

	function canRetry(jobStatus: string): boolean {
		return ["cancelled", "failed"].includes(jobStatus);
	}

	return (
		<div className="p-8">
			<Table>
				<TableHeader>
					<TableRow>
						<TableHead>ID</TableHead>
						<TableHead>Status</TableHead>
						<TableHead>Type</TableHead>
						<TableHead>Attempt</TableHead>
						<TableHead>Progress</TableHead>
						<TableHead>Created</TableHead>
						<TableHead>Started</TableHead>
						<TableHead>Finished</TableHead>
						<TableHead>Error</TableHead>
						<TableHead className="text-right">Actions</TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{jobs.map((job) => (
						<TableRow key={job.id}>
							<TableCell>{job.id}</TableCell>
							<TableCell>{job.status}</TableCell>
							<TableCell>{formatJobType(job.jobType)}</TableCell>
							<TableCell>{job.attempt}</TableCell>
							<TableCell>{job.progress}</TableCell>
							<TableCell>
								<span>
									{new Date(job.createdAt).toLocaleString(undefined, {
										dateStyle: "medium",
										timeStyle: "short",
									})}
								</span>
							</TableCell>
							<TableCell>
								<span>
									{job.startedAt &&
										new Date(job.startedAt).toLocaleString(undefined, {
											dateStyle: "medium",
											timeStyle: "short",
										})}
								</span>
							</TableCell>
							<TableCell>
								<span>
									{job.finishedAt &&
										new Date(job.finishedAt).toLocaleString(undefined, {
											dateStyle: "medium",
											timeStyle: "short",
										})}
								</span>
							</TableCell>
							<TableCell>{job.errorMessage}</TableCell>
							<TableCell className="text-right">
								{job.status === "paused" && (
									<Button
										variant="ghost"
										size="icon"
										className="size-8"
										onClick={() => handleResume(job.id)}
									>
										<CirclePlay />
										<span className="sr-only">Resume job</span>
									</Button>
								)}
								{canPause(job.status) && (
									<Button
										variant="ghost"
										size="icon"
										className="size-8"
										disabled={!canPause(job.status)}
										onClick={() => handlePause(job.id)}
									>
										<CirclePause />
										<span className="sr-only">Pause job</span>
									</Button>
								)}
								{canRetry(job.status) && (
									<Button
										variant="ghost"
										size="icon"
										className="size-8"
										disabled={!canRetry(job.status)}
										onClick={() => handleRetry(job.id)}
									>
										<RotateCw />
										<span className="sr-only">Retry job</span>
									</Button>
								)}

								{canCancel(job.status) && (
									<Button
										variant="ghost"
										size="icon"
										className="size-8 text-destructive hover:bg-destructive/10 hover:text-destructive"
										onClick={() => handleCancel(job.id)}
									>
										<CircleX />
										<span className="sr-only">Cancel job</span>
									</Button>
								)}
							</TableCell>
						</TableRow>
					))}
				</TableBody>
			</Table>
		</div>
	);
}
