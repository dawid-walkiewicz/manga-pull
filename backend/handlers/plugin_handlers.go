package handlers

import (
	"encoding/json"
	"log"
	"main/db"
	"main/plugins"
	"net/http"
)

func ListPluginsHandler(store *db.Store) http.HandlerFunc {
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

func convertPlugins(plugins []plugins.Plugin) []db.Plugin {
	conv_plugins := make([]db.Plugin, 0)
	for _, p := range plugins {
		conv_plugins = append(conv_plugins, db.Plugin{
			ID:         p.ID,
			Name:       p.Name,
			Version:    p.Version,
			APIVersion: p.APIVersion,
			Enabled:    false,
			Path:       p.Path,
		})
	}
	return conv_plugins
}

func ScanPlugins(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paths, err := plugins.FindPlugins("../plugins")
		if err != nil {
			log.Printf("ScanPlugins: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		manifests := plugins.LoadPlugins(paths)
		plugins := convertPlugins(manifests)

		err = store.CreatePlugins(r.Context(), plugins)
		if err != nil {
			log.Printf("ScanPlugins: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
