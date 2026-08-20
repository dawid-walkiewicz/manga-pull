package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"main/db"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func ListTitlesHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		titles, err := store.ListSavedTitles(r.Context())
		if err != nil {
			log.Printf("ListTitles: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(titles); err != nil {
			log.Printf("ListTitles: %v", err)
		}
	}
}

func GetTitleHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Printf("GetTitle: %v", err)
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		title, err := store.GetSavedTitle(r.Context(), id)
		if err != nil {
			log.Printf("GetTitle: %v", err)
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "title not found")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(title); err != nil {
			log.Printf("GetTitle: %v", err)
		}
	}
}
