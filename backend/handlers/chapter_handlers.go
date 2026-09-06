package handlers

import (
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

func DownloadChapterHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chapterIdStr := chi.URLParam(r, "chapterId")

		id, err := strconv.ParseInt(chapterIdStr, 10, 64)
		if err != nil {
			log.Printf("DownloadChapter: %v", err)
			WriteError(w, http.StatusBadRequest, "invalid id")
			return
		}

		_, err = store.GetChapter(r.Context(), id)
		if err != nil {
			log.Printf("DownloadChapter: %v", err)
			if errors.Is(db.ErrNotFound, err) {
				WriteError(w, http.StatusNotFound, "chapter not found")
				return
			}
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		payload, err := json.Marshal(jobs.DownloadChapterPayload{
			ChapterID: id,
		})
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		now := time.Now().UTC()
		_, err = store.CreateJob(r.Context(), db.Job{
			JobType:   jobs.DownloadChapter,
			Status:    jobs.JobQueued,
			Payload:   payload,
			CreatedAt: now,
		})
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "cannot create job")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
