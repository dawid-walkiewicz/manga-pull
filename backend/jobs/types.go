package jobs

import "errors"

type JobStatus string

const (
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobPaused    JobStatus = "paused"
	JobRetrying  JobStatus = "retrying"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
	JobCancelled JobStatus = "cancelled"
)

type JobType string

const (
	JobRefreshTitle    JobType = "refresh_title"
	JobDownloadChapter JobType = "download_chapter"
)

var (
	ErrJobTypeUnknown = errors.New("unknown job type")
)
