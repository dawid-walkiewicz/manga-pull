package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"main/db"
	"main/models"
	"main/services"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func ListTitlesHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		titles, err := store.ListSavedTitles(r.Context())
		if err != nil {
			log.Printf("ListTitles: %v", err)
			WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		summaries := make([]models.SavedTitleSummary, 0, len(titles))
		for _, t := range titles {
			summaries = append(summaries, models.ConvertSavedTitleToSummary(t))
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summaries); err != nil {
			log.Printf("ListTitles: %v", err)
		}
	}
}

func GetTitleHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "titleId")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Printf("GetTitle: %v", err)
			WriteError(w, http.StatusBadRequest, "invalid id")
			return
		}

		title, err := store.GetSavedTitle(r.Context(), id)
		if err != nil {
			log.Printf("GetTitle: %v", err)
			if errors.Is(err, db.ErrNotFound) {
				WriteError(w, http.StatusNotFound, "title not found")
				return
			}

			WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		chapters, err := store.ListChapters(r.Context(), id)
		if err != nil {
			log.Printf("GetTitle: %v", err)

			WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		fullTitle := models.ConvertSavedTitle(title, chapters)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(fullTitle); err != nil {
			log.Printf("GetTitle: %v", err)
		}
	}
}

func DeleteTitleHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "titleId")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Printf("DeleteTitle: %v", err)
			WriteError(w, http.StatusBadRequest, "invalid id")
			return
		}

		err = store.DeleteSavedTitle(r.Context(), id)
		if err != nil {
			log.Printf("DeleteTitle: %v", err)
			if errors.Is(err, db.ErrNotFound) {
				WriteError(w, http.StatusNotFound, "title not found")
				return
			}

			WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func RefreshTitleHandler(service *services.TitleManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "titleId")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Printf("RefreshTitle: %v", err)
			WriteError(w, http.StatusBadRequest, "invalid id")
			return
		}

		refreshedTitle, err := service.RefreshTitle(r.Context(), id)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				WriteError(w, http.StatusNotFound, "title not found")
				return
			}
			log.Printf("RefreshTitle: %v", err)
			writePluginServiceError(w, err, "failed to fetch title from plugin")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(refreshedTitle.Title); err != nil {
			log.Printf("RefreshTitle: %v", err)
		}
	}
}
