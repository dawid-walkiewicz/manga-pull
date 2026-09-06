package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"main/db"
	"main/jobs"

	"github.com/go-chi/chi/v5"
)

func ListJobsHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobs, err := store.ListJobs(r.Context())
		if err != nil {
			log.Printf("ListJobs: %v", err)
			WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(jobs); err != nil {
			log.Printf("ListTitles: %v", err)
		}
	}
}

func CancelJobHandler(store *db.Store, worker *jobs.Worker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		job, ok := jobFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		switch job.Status {
		case jobs.JobCompleted, jobs.JobFailed:
			WriteError(w, http.StatusConflict, "job is already finished")
			return
		case jobs.JobCancelled:
			w.WriteHeader(http.StatusNoContent)
			return
		}

		err := worker.CancelJob(r.Context(), job.ID)
		if err != nil {
			log.Printf("CancelJob: %v", err)
			WriteError(w, http.StatusConflict, "job could not be cancelled")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func PauseJobHandler(store *db.Store, worker *jobs.Worker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		job, ok := jobFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		switch job.Status {
		case jobs.JobCompleted, jobs.JobFailed, jobs.JobCancelled:
			WriteError(w, http.StatusConflict, "job is already finished")
			return
		case jobs.JobPaused:
			w.WriteHeader(http.StatusNoContent)
			return
		}

		err := worker.PauseJob(r.Context(), job)
		if err != nil {
			log.Printf("PauseJob: %v", err)
			WriteError(w, http.StatusConflict, "job could not be paused")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func ResumeJobHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		job, ok := jobFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if job.Status != jobs.JobPaused {
			WriteError(w, http.StatusConflict, "job is not paused")
			return
		}

		update := db.JobStatusUpdate{
			ID:     job.ID,
			Status: jobs.JobQueued,
		}
		_, err := store.UpdateJobStatus(r.Context(), update)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		jobs.LogJob(r.Context(), store, job.ID, jobs.Info, "resuming job")

		w.WriteHeader(http.StatusNoContent)
	}
}

func RetryJobHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		job, ok := jobFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		switch job.Status {
		case jobs.JobCompleted:
			WriteError(w, http.StatusConflict, "job is already completed")
			return
		case jobs.JobPaused, jobs.JobRetrying, jobs.JobRunning:
			WriteError(w, http.StatusConflict, "job is still running")
			return
		}

		progress := "0"
		errorMessage := ""
		update := db.JobStatusUpdate{
			ID:           job.ID,
			Status:       jobs.JobQueued,
			Progress:     &progress,
			ErrorMessage: &errorMessage,
			FinishedAt:   nil,
		}
		_, err := store.RetryJob(r.Context(), update)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		jobs.LogJob(r.Context(), store, job.ID, jobs.Info, "retrying job")

		w.WriteHeader(http.StatusNoContent)
	}
}

func RefreshPluginTitleJobHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		titleID := chi.URLParam(r, "titleId")

		id, err := strconv.ParseInt(titleID, 10, 64)
		if err != nil {
			log.Printf("RefreshPluginTitleJob: %v", err)
			WriteError(w, http.StatusBadRequest, "invalid id")
			return
		}

		_, err = store.GetSavedTitle(r.Context(), id)
		if err != nil {
			log.Printf("RefreshPluginTitleJob: %v", err)
			if errors.Is(err, db.ErrNotFound) {
				WriteError(w, http.StatusNotFound, "title not found")
				return
			}
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		payload, err := json.Marshal(jobs.RefreshTitlePayload{
			SavedTitleID: id,
		})
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		refreshJob := db.Job{
			JobType:   jobs.RefreshTitle,
			Status:    jobs.JobQueued,
			Payload:   payload,
			CreatedAt: time.Now().UTC(),
		}

		_, err = store.CreateJob(r.Context(), refreshJob)
		if err != nil {
			log.Printf("RefreshPluginTitleJob: %v", err)
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type jobContextKey struct{}

func withJob(ctx context.Context, job *db.Job) context.Context {
	return context.WithValue(ctx, jobContextKey{}, job)
}

func jobFromContext(ctx context.Context) (*db.Job, bool) {
	job, ok := ctx.Value(jobContextKey{}).(*db.Job)
	return job, ok
}

func JobMiddleware(store *db.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "jobId")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				WriteError(w, http.StatusBadRequest, "invalid id")
				return
			}

			job, err := store.GetJob(r.Context(), id)
			if err != nil {
				if errors.Is(err, db.ErrNotFound) {
					WriteError(w, http.StatusNotFound, "job not found")
					return
				}
				WriteError(w, http.StatusInternalServerError, "internal error")
				return
			}

			ctx := withJob(r.Context(), &job)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
