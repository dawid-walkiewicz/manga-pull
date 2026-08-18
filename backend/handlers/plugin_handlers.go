package handlers

import (
	"encoding/json"
	"log"
	"main/services"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func ListPluginsHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plugins := service.Plugins()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(plugins); err != nil {
			log.Printf("ListPlugins: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

func ScanPluginsHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := service.Scan(r.Context()); err != nil {
			log.Printf("ScanPlugins: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func EnablePluginHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := service.Enable(r.Context(), id)
		if err != nil {
			log.Printf("EnablePlugin: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func DisablePluginHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := service.Disable(r.Context(), id)
		if err != nil {
			log.Printf("DisablePlugin: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func SearchTitleHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		plugin, ok := service.Runtime(id)
		if !ok {
			log.Printf("SearchTitle: runtime not found")
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		title := r.URL.Query().Get("title")

		titles, err := plugin.Search(title)
		if err != nil {
			log.Printf("SearchTitle: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(titles); err != nil {
			log.Printf("SearchTitle: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}
