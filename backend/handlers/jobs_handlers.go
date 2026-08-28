package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"main/db"
	"main/jobs"
	"net/http"
	"strconv"
	"time"

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
		idStr := chi.URLParam(r, "jobId")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Printf("CancelJob: %v", err)
			WriteError(w, http.StatusBadRequest, "invalid id")
			return
		}

		job, err := store.GetJob(r.Context(), id)
		if err != nil {
			log.Printf("CancelJob: %v", err)
			if errors.Is(err, db.ErrNotFound) {
				WriteError(w, http.StatusNotFound, "job not found")
				return
			}
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		switch job.Status {
		case string(jobs.JobCompleted), string(jobs.JobFailed):
			WriteError(w, http.StatusConflict, "job is already finished")
			return
		case string(jobs.JobCancelled):
			w.WriteHeader(http.StatusNoContent)
			return
		}

		err = worker.CancelJob(r.Context(), id)
		if err != nil {
			log.Printf("CancelJob: %v", err)
			WriteError(w, http.StatusConflict, "job cannot be cancelled")
		}

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

		refreshJob := db.Job{
			JobType:      string(jobs.JobRefreshTitle),
			Status:       string(jobs.JobQueued),
			SavedTitleID: &id,
			CreatedAt:    time.Now().UTC(),
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
