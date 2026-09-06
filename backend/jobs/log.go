package jobs

import (
	"context"
	"log"
)

type Logger interface {
	AddJobLog(ctx context.Context, jobID int64, level, message string) error
}

func LogJob(ctx context.Context, store Logger, jobID int64, level, msg string) {
	if err := store.AddJobLog(ctx, jobID, level, msg); err != nil {
		log.Printf("failed to save job log to DB (jobID: %d): %v", jobID, err)
	}
}
