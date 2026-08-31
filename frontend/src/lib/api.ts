export const API_BASE_URL =
	import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8000";

export type ApiErrorResponse = {
	error: string;
};

export type PluginManifest = {
	id: string;
	name: string;
	version: string;
	apiVersion: number;
	description: string;
	transport: string;
	domains: string[];
};

export type Plugin = PluginManifest & {
	path: string;
	enabled: boolean;
};

export type PluginTitleSummary = {
	id: string;
	title: string;
	cover: string;
};

export type PluginChapter = {
	id: string;
	number: number;
	volume: number | null;
	season: number | null;
	title: string | null;
	groupName: string | null;
	language: string | null;
	publishedAt: string | null;
	url: string | null;
};

export type PluginTitleDetails = {
	id: string;
	title: string;
	cover: string | null;
	alternativeTitles: string[];
	author: string | null;
	artist: string | null;
	status: string | null;
	description: string | null;
	url: string | null;
	chapters: PluginChapter[];
	savedId: number | null;
};

export type SavedTitleSummary = {
	id: number;
	pluginId: string;
	remoteId: string;
	title: string;
	cover: string | null;
};

export type SavedTitleResponse = {
	id: number;
};

export type SavedChapter = {
	id: number;
	savedTitleId: number;
	remoteId: string;
	number: number;
	volume: number | null;
	season: number | null;
	title: string | null;
	groupName: string | null;
	url: string | null;
	publishedAt: string | null;
	downloaded: boolean;
};

export type SavedTitleDetails = {
	id: number;
	pluginId: string;
	remoteId: string;
	title: string;
	alternativeTitles: string[];
	author: string | null;
	artist: string | null;
	status: string | null;
	description: string | null;
	cover: string | null;
	groupFilter: string[];
	url: string | null;
	directoryName: string;
	chapterNameTemplate: string;
	lastRefreshedAt: string | null;
	chapters: SavedChapter[];
};

export type JobStatus =
	| "queued"
	| "running"
	| "paused"
	| "retrying"
	| "completed"
	| "failed"
	| "cancelled";

export type JobType = "refresh_title" | "download_chapter";

export type Job = {
	id: number;
	jobType: JobType | string;
	status: JobStatus | string;
	savedTitleId: number | null;
	chapterId: number | null;
	attempt: number;
	progress: string;
	errorMessage: string | null;
	createdAt: string;
	startedAt: string | null;
	finishedAt: string | null;
};

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
	const response = await fetch(`${API_BASE_URL}${path}`, init);
	const text = await response.text();

	if (!response.ok) {
		let message = `Request failed with status ${response.status}`;

		if (text) {
			try {
				const error = JSON.parse(text) as ApiErrorResponse;
				message = error.error || message;
			} catch {
				message =
					text.startsWith("<!doctype") || text.startsWith("<html")
						? `Request returned HTML instead of JSON: ${path}`
						: message;
			}
		}

		throw new Error(message);
	}

	if (response.status === 204) {
		return undefined as T;
	}

	if (!text) {
		return undefined as T;
	}

	if (text.startsWith("<!doctype") || text.startsWith("<html")) {
		throw new Error(`Request returned HTML instead of JSON: ${path}`);
	}

	return JSON.parse(text) as T;
}

const encodePathPart = (value: string | number) =>
	encodeURIComponent(String(value));

export function getPluginIconUrl(pluginId: string): string {
	return `${API_BASE_URL}/api/plugins/${encodePathPart(pluginId)}/icon`;
}

export function listTitles(): Promise<SavedTitleSummary[]> {
	return apiRequest("/api/titles");
}

export function getTitle(id: number): Promise<SavedTitleDetails> {
	return apiRequest(`/api/titles/${encodePathPart(id)}`);
}

export function deleteTitle(id: number): Promise<void> {
	return apiRequest(`/api/titles/${encodePathPart(id)}`, { method: "DELETE" });
}

export function listPlugins(): Promise<Plugin[]> {
	return apiRequest("/api/plugins");
}

export function scanPlugins(): Promise<void> {
	return apiRequest("/api/plugins/scan", { method: "POST" });
}

export function enablePlugin(id: string): Promise<void> {
	return apiRequest(`/api/plugins/${encodePathPart(id)}/enable`, {
		method: "POST",
	});
}

export function disablePlugin(id: string): Promise<void> {
	return apiRequest(`/api/plugins/${encodePathPart(id)}/disable`, {
		method: "POST",
	});
}

export function searchPluginTitles(
	pluginId: string,
	title: string,
): Promise<PluginTitleSummary[]> {
	const search = new URLSearchParams({ title });
	return apiRequest(
		`/api/plugins/${encodePathPart(pluginId)}/search?${search.toString()}`,
	);
}

export function browsePluginTitles(
	pluginId: string,
): Promise<PluginTitleSummary[]> {
	return apiRequest(`/api/plugins/${encodePathPart(pluginId)}/titles`);
}

export function getPluginTitle(
	pluginId: string,
	titleId: string,
): Promise<PluginTitleDetails> {
	return apiRequest(
		`/api/plugins/${encodePathPart(pluginId)}/titles/${encodePathPart(titleId)}`,
	);
}

export function savePluginTitle(
	pluginId: string,
	titleId: string,
): Promise<SavedTitleResponse> {
	return apiRequest(
		`/api/plugins/${encodePathPart(pluginId)}/titles/${encodePathPart(titleId)}/save`,
		{ method: "POST" },
	);
}

export function refreshPluginTitle(
	pluginId: string,
	titleId: string,
): Promise<SavedTitleDetails> {
	return apiRequest(
		`/api/plugins/${encodePathPart(pluginId)}/titles/${encodePathPart(titleId)}/refresh`,
		{ method: "POST" },
	);
}

export function listJobs(): Promise<Job[]> {
	return apiRequest("/api/jobs");
}

export function cancelJob(jobId: number): Promise<void> {
	return apiRequest(`/api/jobs/${encodePathPart(jobId)}/cancel`, {
		method: "POST",
	});
}

export function pauseJob(jobId: number): Promise<void> {
	return apiRequest(`/api/jobs/${encodePathPart(jobId)}/pause`, {
		method: "POST",
	});
}

export function resumeJob(jobId: number): Promise<void> {
	return apiRequest(`/api/jobs/${encodePathPart(jobId)}/resume`, {
		method: "POST",
	});
}

export function retryJob(jobId: number): Promise<void> {
	return apiRequest(`/api/jobs/${encodePathPart(jobId)}/retry`, {
		method: "POST",
	});
}

export function queueTitleRefresh(titleId: number): Promise<void> {
	return apiRequest(`/api/jobs/refresh/${encodePathPart(titleId)}`, {
		method: "POST",
	});
}
