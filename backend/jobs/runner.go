package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"main/common"
	"main/db"
	"main/services"
)

type RefreshTitlePayload struct {
	SavedTitleID int64 `json:"savedTitleId"`
}

type DownloadChapterPayload struct {
	ChapterID int64 `json:"chapterId"`
}

type JobStore interface {
	CreateJobs(ctx context.Context, jobs []db.Job) error
	UpdateJobStatus(ctx context.Context, status db.JobStatusUpdate) (*db.Job, error)
	UpdateChapter(ctx context.Context, chapter db.Chapter) error
	AddJobLog(ctx context.Context, jobID int64, level, message string) error
}

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
	case RefreshTitle:
		return r.refreshTitle(ctx, job)

	case DownloadChapter:
		return r.downloadChapter(ctx, job)

	default:
		return fmt.Errorf("%w: %q", ErrJobTypeUnknown, job.JobType)
	}
}

func (r *Runner) refreshTitle(ctx context.Context, job *db.Job) error {
	var payload RefreshTitlePayload

	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode refresh title payload: %w", err)
	}

	LogJob(ctx, r.store, job.ID, Info, "refreshing title")
	result, err := r.titleManager.RefreshTitle(ctx, payload.SavedTitleID)
	if err != nil {
		return err
	}

	LogJob(ctx, r.store, job.ID, Info, fmt.Sprintf("creating %d download jobs", len(result.NewChapters)))
	downloadJobs := make([]db.Job, 0, len(result.NewChapters))
	for _, chapter := range result.NewChapters {
		payload, err := json.Marshal(DownloadChapterPayload{
			ChapterID: chapter.ID,
		})
		if err != nil {
			return err
		}
		job := db.Job{
			JobType:   DownloadChapter,
			Status:    JobQueued,
			Payload:   payload,
			CreatedAt: time.Now().UTC(),
		}
		downloadJobs = append(downloadJobs, job)
	}

	if err := r.store.CreateJobs(ctx, downloadJobs); err != nil {
		return fmt.Errorf("create download jobs: %w", err)
	}
	return nil
}

func (r *Runner) downloadChapter(ctx context.Context, job *db.Job) error {
	var payload DownloadChapterPayload

	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("decode download chapter payload: %w", err)
	}

	details, err := r.titleManager.BeginChapterDownload(ctx, payload.ChapterID)
	if err != nil {
		return err
	}
	LogJob(ctx, r.store, job.ID, Info, "starting chapter download")

	chapterDir := filepath.Join(common.DownloadDir, details.DirectoryName, details.Name)
	err = os.MkdirAll(chapterDir, 0o755)
	if err != nil {
		return err
	}
	LogJob(ctx, r.store, job.ID, Info, fmt.Sprintf("download location: %s\n", chapterDir))

	LogJob(ctx, r.store, job.ID, Info, "downloading pages")
	for i, page := range details.Descriptor.Pages {
		err = downloadFile(page.Url, filepath.Join(chapterDir, fmt.Sprint(i, ".png")))
		if err != nil {
			return err
		}

		if (i+1)%5 == 0 {
			job, err = r.store.UpdateJobStatus(ctx, *markJobAsRunning(job.ID, strconv.Itoa(i*100/len(details.Descriptor.Pages)), job.StartedAt))
			if err != nil {
				LogJob(ctx, r.store, job.ID, Error, fmt.Sprintf("error during downloading page %d", i+1))
				return err
			}
		}
	}

	err = r.titleManager.MarkChapterDownloaded(ctx, payload.ChapterID)
	if err != nil {
		return err
	}
	LogJob(ctx, r.store, job.ID, Info, "chapter download finished")

	return nil
}

func downloadFile(url string, filePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unsuccessful download, HTTP status: %s", resp.Status)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
