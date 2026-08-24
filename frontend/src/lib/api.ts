export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "";

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
	chapters: PluginChapter[];
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
	language: string | null;
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
	directoryName: string;
	chapterNameTemplate: string;
	lastRefreshedAt: string | null;
	chapters: SavedChapter[];
};

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
	const response = await fetch(`${API_BASE_URL}${path}`, init);

	if (!response.ok) {
		let message = `Request failed with status ${response.status}`;

		try {
			const error = (await response.json()) as ApiErrorResponse;
			message = error.error || message;
		} catch (error) {
			console.log(error);
		}

		throw new Error(message);
	}

	if (response.status === 204) {
		return undefined as T;
	}

	const text = await response.text();
	if (!text) {
		return undefined as T;
	}

	return JSON.parse(text) as T;
}

const encodePathPart = (value: string | number) =>
	encodeURIComponent(String(value));

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
