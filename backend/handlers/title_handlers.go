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

type CreateSavedTitleRequest struct {
	PluginID            string   `json:"pluginId"`
	RemoteID            string   `json:"remoteId"`
	Title               string   `json:"title"`
	AlternativeTitles   []string `json:"alternativeTitles"`
	Author              *string  `json:"author"`
	Artist              *string  `json:"artist"`
	Status              *string  `json:"status"`
	Description         *string  `json:"description"`
	Cover               *string  `json:"cover"`
	DirectoryName       string   `json:"directoryName"`
	ChapterNameTemplate string   `json:"chapterNameTemplate"`
}

func (r CreateSavedTitleRequest) ToModel() db.SavedTitle {
	return db.SavedTitle{
		PluginID:            r.PluginID,
		RemoteID:            r.RemoteID,
		Title:               r.Title,
		AlternativeTitles:   db.StringList(r.AlternativeTitles),
		Author:              r.Author,
		Artist:              r.Artist,
		Status:              r.Status,
		Description:         r.Description,
		Cover:               r.Cover,
		DirectoryName:       r.DirectoryName,
		ChapterNameTemplate: r.ChapterNameTemplate,
	}
}

func CreateTitleHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateSavedTitleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("CreateTitle: %v", err)
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}

		title := req.ToModel()
		id, err := store.CreateSavedTitle(r.Context(), title)
		if err != nil {
			log.Printf("CreateTitle: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(map[string]int64{"id": id}); err != nil {
			log.Printf("CreateTitle: %v", err)
		}
	}
}
