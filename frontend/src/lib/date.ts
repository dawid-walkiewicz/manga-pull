export function formatTime(time: string) {
	return new Date(time).toLocaleString(undefined, {
		dateStyle: "medium",
		timeStyle: "short",
	});
}
