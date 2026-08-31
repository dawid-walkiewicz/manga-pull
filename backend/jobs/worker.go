package jobs

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"main/common"
	"main/db"
	"sync"
	"time"
)

type JobStore interface {
	ClaimNextJob(ctx context.Context) (*db.Job, error)
	UpdateJobStatus(ctx context.Context, status db.JobStatusUpdate) (*db.Job, error)
	GetSavedTitle(ctx context.Context, id int64) (db.SavedTitle, error)
	CreateJob(ctx context.Context, job db.Job) (int64, error)
}

type Executor interface {
	Execute(ctx context.Context, job *db.Job) error
}

type ConfigGetter interface {
	Get() common.ConfigData
}

var (
	ErrJobPaused    = errors.New("job paused")
	ErrJobCancelled = errors.New("job cancelled")
)

type Worker struct {
	store         JobStore
	executor      Executor
	configManager ConfigGetter

	mu      sync.Mutex
	running map[int64]context.CancelCauseFunc
}

func NewWorker(store JobStore, executor Executor, configManager ConfigGetter) *Worker {
	return &Worker{
		store:         store,
		executor:      executor,
		configManager: configManager,
		running:       make(map[int64]context.CancelCauseFunc),
	}
}

func (w *Worker) registerRunning(id int64, cancel context.CancelCauseFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.running[id] = cancel
}

func (w *Worker) unregisterRunning(id int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.running, id)
}

func (w *Worker) getCancel(id int64) (context.CancelCauseFunc, bool) {
	w.mu.Lock()
	cancel, ok := w.running[id]
	w.mu.Unlock()

	if !ok {
		return nil, ok
	}
	return cancel, ok
}

func (w *Worker) cancelRunning(id int64) bool {
	cancel, ok := w.getCancel(id)
	if !ok {
		return false
	}

	cancel(ErrJobCancelled)
	return true
}

func (w *Worker) pauseRunning(id int64) bool {
	cancel, ok := w.getCancel(id)
	if !ok {
		return false
	}

	cancel(ErrJobPaused)
	return true
}

func (w *Worker) CancelJob(ctx context.Context, id int64) error {
	if w.cancelRunning(id) {
		return nil
	}

	_, err := w.store.UpdateJobStatus(ctx, *markJobAsCancelled(id))
	return err
}

func (w *Worker) PauseJob(ctx context.Context, job *db.Job) error {
	if w.pauseRunning(job.ID) {
		return nil
	}
	_, err := w.store.UpdateJobStatus(ctx, *markJobAsPaused(job))
	return err
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			err := w.processNext(ctx)

			if err != nil {
				log.Printf("job worker: %v", err)
			}
		}
	}
}

func (w *Worker) processNext(ctx context.Context) error {
	job, err := w.store.ClaimNextJob(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	jobCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(ErrJobCancelled)
	w.registerRunning(job.ID, cancel)
	defer w.unregisterRunning(job.ID)

	config := w.configManager.Get()

	var lastExecErr error
	for job.Attempt < (config.MaxRetries + 1) {
		lastExecErr = w.executor.Execute(jobCtx, job)
		if lastExecErr == nil {
			_, err = w.store.UpdateJobStatus(jobCtx, *markJobAsCompleted(job.ID))
			return err
		}

		if errors.Is(lastExecErr, ErrJobTypeUnknown) {
			break
		}

		if errors.Is(lastExecErr, context.Canceled) {
			err = context.Cause(jobCtx)

			switch err {
			case ErrJobPaused:
				_, err = w.store.UpdateJobStatus(context.Background(), *markJobAsPaused(job))
				return err
			case ErrJobCancelled:
				_, err = w.store.UpdateJobStatus(context.Background(), *markJobAsCancelled(job.ID))
				return err
			default:
				return err
			}
		}

		if job.Attempt+1 >= (config.MaxRetries + 1) {
			break
		}

		job, err = w.store.UpdateJobStatus(jobCtx, *markJobAsRetrying(job))
		if err != nil {
			return err
		}
		delay := time.Duration(10*job.Attempt) * time.Second

		select {
		case <-time.After(delay):
		case <-jobCtx.Done():
			switch context.Cause(jobCtx) {
			case ErrJobPaused:
				_, err = w.store.UpdateJobStatus(context.Background(), *markJobAsPaused(job))
			case ErrJobCancelled:
				_, err = w.store.UpdateJobStatus(context.Background(), *markJobAsCancelled(job.ID))
			default:
				err = jobCtx.Err()
			}
			return err
		}
	}

	_, err = w.store.UpdateJobStatus(ctx, *markJobAsFailed(job.ID, lastExecErr))
	return err
}

func markJobAsRunning(jobId int64) *db.JobStatusUpdate {
	now := time.Now().UTC()
	return &db.JobStatusUpdate{
		ID:        jobId,
		Status:    JobRunning,
		Progress:  ptr("0"),
		StartedAt: &now,
	}
}

func markJobAsCompleted(jobId int64) *db.JobStatusUpdate {
	now := time.Now().UTC()
	return &db.JobStatusUpdate{
		ID:           jobId,
		Status:       JobCompleted,
		Progress:     ptr("100"),
		ErrorMessage: nil,
		FinishedAt:   &now,
	}
}

func markJobAsFailed(jobId int64, err error) *db.JobStatusUpdate {
	return &db.JobStatusUpdate{
		ID:           jobId,
		Status:       JobFailed,
		ErrorMessage: ptr(err.Error()),
	}
}

func markJobAsRetrying(job *db.Job) *db.JobStatusUpdate {
	return &db.JobStatusUpdate{
		ID:      job.ID,
		Status:  JobRetrying,
		Attempt: ptr(job.Attempt + 1),
	}
}

func markJobAsCancelled(jobId int64) *db.JobStatusUpdate {
	return &db.JobStatusUpdate{
		ID:     jobId,
		Status: JobCancelled,
	}
}

func markJobAsPaused(job *db.Job) *db.JobStatusUpdate {
	return &db.JobStatusUpdate{
		ID:       job.ID,
		Status:   JobPaused,
		Progress: &job.Progress,
	}
}

func ptr[T any](v T) *T {
	return &v
}
