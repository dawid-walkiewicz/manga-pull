package main

import (
	"context"
	"encoding/json"
	"log"
	"main/db"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	ctx := context.Background()
	database, err := db.Open(ctx, "database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	store := db.NewStore(database)

	r := chi.NewRouter()

	r.Use(middleware.Logger)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ok"))
	})

	r.Get("/api/titles", listTitlesHandler(store))
	r.Get("/api/titles/{id}", getTitleHandler(store))
	r.Post("/api/titles", createTitleHandler(store))
	r.Get("/api/plugins", listPluginsHandler(store))

	log.Println("server listening on :8000")
	if err := http.ListenAndServe(":8000", r); err != nil {
		log.Fatal(err)
	}
}

func listTitlesHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		titles, err := store.ListSavedTitles(r.Context())
		if err != nil {
			log.Printf("ListTitles: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(titles); err != nil {
			log.Printf("ListTitles: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

func getTitleHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Printf("GetTitle: %v", err)
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		title, err := store.GetSavedTitle(r.Context(), id)
		if err != nil {
			log.Printf("GetTitle: %v", err)
			http.Error(w, "title not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(title); err != nil {
			log.Printf("GetTitle: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

func createTitleHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateSavedTitleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("CreateTitle: %v", err)
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		title := req.ToModel()
		id, err := store.CreateSavedTitle(r.Context(), title)
		if err != nil {
			log.Printf("CreateTitle: %v", err)
			http.Error(w, "failed to create title", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(map[string]int64{"id": id}); err != nil {
			log.Printf("CreateTitle: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

	}
}

func listPluginsHandler(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plugins, err := store.ListPlugins(r.Context())
		if err != nil {
			log.Printf("ListPlugins: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(plugins); err != nil {
			log.Printf("ListPlugins: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}
