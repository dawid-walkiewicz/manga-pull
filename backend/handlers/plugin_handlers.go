package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"main/plugins"
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
		}
	}
}

func ScanPluginsHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := service.Scan(r.Context()); err != nil {
			log.Printf("ScanPlugins: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
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
			if errors.Is(err, services.ErrPluginNotFound) {
				writeError(w, http.StatusNotFound, "plugin not found")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
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
			if errors.Is(err, services.ErrPluginNotFound) {
				writeError(w, http.StatusNotFound, "plugin not found")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func SearchPluginTitleHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		plugin, ok := pluginRuntime(service, id, w)
		if !ok {
			return
		}

		title := r.URL.Query().Get("title")

		titles, err := plugin.Search(title)
		if err != nil {
			log.Printf("SearchTitle: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(titles); err != nil {
			log.Printf("SearchTitle: %v", err)
		}
	}
}

func BrowsePluginTitlesHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		plugin, ok := pluginRuntime(service, id, w)
		if !ok {
			return
		}

		titles, err := plugin.Browse()
		if err != nil {
			log.Printf("BrowsePluginTitles: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(titles); err != nil {
			log.Printf("BrowsePluginTitles: %v", err)
		}
	}
}

func GetPluginTitleHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		plugin, ok := pluginRuntime(service, id, w)
		if !ok {
			return
		}

		titleId := chi.URLParam(r, "titleId")
		title, err := plugin.GetTitle(titleId)

		if err != nil {
			log.Printf("GetPluginTitle: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(title); err != nil {
			log.Printf("GetPluginTitle: %v", err)
		}
	}
}

func pluginRuntime(
	service *services.PluginManager,
	id string,
	w http.ResponseWriter,
) (*plugins.PluginRuntime, bool) {
	runtime, ok := service.Runtime(id)
	if ok {
		return runtime, true
	}

	plugin, ok := service.Plugin(id)
	if !ok {
		writeError(w, http.StatusNotFound, "plugin not found")
		return nil, false
	}

	if !plugin.Enabled {
		writeError(w, http.StatusConflict, "plugin is disabled")
		return nil, false
	}

	writeError(w, http.StatusConflict, "plugin runtime unavailable")
	return nil, false
}
