package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"main/common"
	"main/db"
)

type JobProcessor interface {
	ClaimNextJob(ctx context.Context) (*db.Job, error)
	AddJobLog(ctx context.Context, jobID int64, level, message string) error
	MarkJobAsPaused(ctx context.Context, id int64) (*db.Job, error)
	MarkJobAsRetrying(ctx context.Context, id int64, retries int) (*db.Job, error)
	MarkJobAsCompleted(ctx context.Context, id int64, progress string, errMessage *string, finishedAt time.Time) (*db.Job, error)
	MarkJobAsFailed(ctx context.Context, id int64, errMessage string) (*db.Job, error)
	MarkJobAsCancelled(ctx context.Context, id int64) (*db.Job, error)
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
	store         JobProcessor
	executor      Executor
	configManager ConfigGetter

	mu      sync.Mutex
	running map[int64]context.CancelCauseFunc
}

func NewWorker(store JobProcessor, executor Executor, configManager ConfigGetter) *Worker {
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

	LogJob(ctx, w.store, id, Info, "cancelling job")
	_, err := w.store.MarkJobAsCancelled(ctx, id)
	return err
}

func (w *Worker) PauseJob(ctx context.Context, id int64) error {
	if w.pauseRunning(id) {
		return nil
	}

	LogJob(ctx, w.store, id, Info, "pausing job")
	_, err := w.store.MarkJobAsPaused(ctx, id)
	return err
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	log.Println("starting worker")

	for {
		select {
		case <-ctx.Done():
			log.Println("stopping worker")
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
	for {
		lastExecErr = w.executor.Execute(jobCtx, job)
		if lastExecErr == nil {
			LogJob(jobCtx, w.store, job.ID, Info, "job executed successfully")
			_, err = w.store.MarkJobAsCompleted(jobCtx, job.ID, "100", nil, time.Now().UTC())
			return err
		}

		if errors.Is(lastExecErr, ErrJobTypeUnknown) {
			LogJob(jobCtx, w.store, job.ID, Error, lastExecErr.Error())
			break
		}

		LogJob(jobCtx, w.store, job.ID, Error, lastExecErr.Error())

		if errors.Is(lastExecErr, context.Canceled) {
			err = context.Cause(jobCtx)

			safeCtx, cancel := context.WithTimeout(context.WithoutCancel(jobCtx), 5*time.Second)
			defer cancel()

			switch err {
			case ErrJobPaused:
				LogJob(safeCtx, w.store, job.ID, Info, "pausing job")
				_, err = w.store.MarkJobAsPaused(safeCtx, job.ID)
				return err
			case ErrJobCancelled:
				LogJob(safeCtx, w.store, job.ID, Info, "cancelling job")
				_, err = w.store.MarkJobAsCancelled(safeCtx, job.ID)
				return err
			default:
				return err
			}
		}

		if job.Retries >= config.MaxRetries {
			LogJob(jobCtx, w.store, job.ID, Warning, "maximum number of retries reached")
			break
		}

		LogJob(jobCtx, w.store, job.ID, Error, "job execution failed, marking as retrying")
		job, err = w.store.MarkJobAsRetrying(jobCtx, job.ID, job.Retries+1)
		if err != nil {
			return err
		}
		delay := time.Duration(10*job.Retries) * time.Second
		LogJob(jobCtx, w.store, job.ID, Info, fmt.Sprintf("retrying after %s", delay))

		select {
		case <-time.After(delay):
		case <-jobCtx.Done():
			safeCtx, cancel := context.WithTimeout(context.WithoutCancel(jobCtx), 5*time.Second)

			switch context.Cause(jobCtx) {
			case ErrJobPaused:
				LogJob(safeCtx, w.store, job.ID, Info, "pausing job")
				_, err = w.store.MarkJobAsPaused(safeCtx, job.ID)
			case ErrJobCancelled:
				LogJob(safeCtx, w.store, job.ID, Info, "cancelling job")
				_, err = w.store.MarkJobAsCancelled(safeCtx, job.ID)
			default:
				err = jobCtx.Err()
			}
			cancel()
			return err
		}
	}

	safeCtx, safeCancel := context.WithTimeout(context.WithoutCancel(jobCtx), 5*time.Second)
	defer safeCancel()
	LogJob(safeCtx, w.store, job.ID, Info, "marking job as failed")
	_, err = w.store.MarkJobAsFailed(safeCtx, job.ID, lastExecErr.Error())
	return err
}

func ptr[T any](v T) *T {
	return &v
}
