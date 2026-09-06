package jobs

import "errors"

const (
	JobQueued    string = "queued"
	JobRunning   string = "running"
	JobPaused    string = "paused"
	JobRetrying  string = "retrying"
	JobCompleted string = "completed"
	JobFailed    string = "failed"
	JobCancelled string = "cancelled"
)

const (
	RefreshTitle    string = "refresh_title"
	DownloadChapter string = "download_chapter"
)

var (
	ErrJobTypeUnknown = errors.New("unknown job type")
)

const (
	Info    string = "info"
	Warning string = "warning"
	Error   string = "error"
)
