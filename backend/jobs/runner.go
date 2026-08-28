package jobs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"main/db"
	"main/services"
	"time"
)

type Runner struct {
	store        JobStore
	titleManager *services.TitleManager
}

func NewRunner(store JobStore, titleManager *services.TitleManager) *Runner {
	return &Runner{
		store:        store,
		titleManager: titleManager,
	}
}

func (r *Runner) Execute(ctx context.Context, job *db.Job) error {
	switch job.JobType {
	case string(JobRefreshTitle):
		if job.SavedTitleID == nil {
			return errors.New("refresh_title job missing saved_title_id")
		}

		title, err := r.store.GetSavedTitle(ctx, *job.SavedTitleID)
		if err != nil {
			return err
		}

		result, err := r.titleManager.RefreshTitle(ctx, title.PluginID, title.RemoteID)
		if err != nil {
			return err
		}

		downloadJobs := make([]db.Job, 0, len(result.NewChapters))
		for _, chapter := range result.NewChapters {
			job := db.Job{
				JobType:      string(JobDownloadChapter),
				Status:       string(JobQueued),
				SavedTitleID: &title.ID,
				ChapterID:    &chapter.ID,
				CreatedAt:    time.Now().UTC(),
			}
			downloadJobs = append(downloadJobs, job)

			log.Printf("[worker] %+v", job)
		}
		return nil

	case string(JobDownloadChapter):
		return errors.New("download_chapter not implemented")

	default:
		return fmt.Errorf("%w: %q", ErrJobTypeUnknown, job.JobType)
	}
}
