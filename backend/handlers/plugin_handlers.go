package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"main/models"
	"main/plugins"
	"main/services"

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
			WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func GetPluginIconHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "pluginId")
		plugin, ok := service.Plugin(id)
		if !ok {
			WriteError(w, http.StatusNotFound, "plugin not found")
			return
		}
		icon, contentType, err := plugins.ReadIcon(plugin)

		if err != nil {
			log.Printf("GetPluginIcon: %v", err)
			if errors.Is(err, plugins.ErrPluginIconNotFound) {
				WriteError(w, http.StatusNotFound, "plugin icon not found")
				return
			}

			WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(icon)
	}
}

func EnablePluginHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "pluginId")
		err := service.Enable(r.Context(), id)
		if err != nil {
			log.Printf("EnablePlugin: %v", err)
			if errors.Is(err, services.ErrPluginNotFound) {
				WriteError(w, http.StatusNotFound, "plugin not found")
				return
			}

			WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func DisablePluginHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "pluginId")
		err := service.Disable(r.Context(), id)
		if err != nil {
			log.Printf("DisablePlugin: %v", err)
			if errors.Is(err, services.ErrPluginNotFound) {
				WriteError(w, http.StatusNotFound, "plugin not found")
				return
			}

			WriteError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func SearchPluginTitleHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "pluginId")
		plugin, ok := pluginRuntime(service, id, w)
		if !ok {
			return
		}

		title := r.URL.Query().Get("title")

		titles, err := plugin.Search(title)
		if err != nil {
			log.Printf("SearchTitle: %v", err)
			WriteError(w, http.StatusBadGateway, "failed to fetch titles from plugin")
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
		id := chi.URLParam(r, "pluginId")
		plugin, ok := pluginRuntime(service, id, w)
		if !ok {
			return
		}

		titles, err := plugin.Browse()
		if err != nil {
			log.Printf("BrowsePluginTitles: %v", err)
			WriteError(w, http.StatusBadGateway, "failed to fetch titles from plugin")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(titles); err != nil {
			log.Printf("BrowsePluginTitles: %v", err)
		}
	}
}

func GetPluginTitleHandler(service *services.TitleManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "pluginId")
		titleId := chi.URLParam(r, "titleId")
		title, err := service.GetPluginTitle(r.Context(), id, titleId)

		if err != nil {
			log.Printf("GetPluginTitle: %v", err)
			writePluginServiceError(w, err, "failed to fetch title from plugin")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(title); err != nil {
			log.Printf("GetPluginTitle: %v", err)
		}
	}
}

func SavePluginTitleHandler(service *services.TitleManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "pluginId")
		titleID := chi.URLParam(r, "titleId")

		createdID, err := service.SaveTitle(r.Context(), id, titleID)
		if err != nil {
			log.Printf("SavePluginTitle: %v", err)
			writePluginServiceError(w, err, "failed to fetch title from plugin")
			return
		}

		response := models.SaveTitleResponse{ID: createdID}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("SavePluginTitle: %v", err)
		}
	}
}

func RefreshPluginTitleHandler(service *services.TitleManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "pluginId")
		titleID := chi.URLParam(r, "titleId")

		refreshedTitle, err := service.RefreshTitle(r.Context(), id, titleID)
		if err != nil {
			log.Printf("RefreshPluginTitle: %v", err)
			writePluginServiceError(w, err, "failed to fetch title from plugin")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(refreshedTitle); err != nil {
			log.Printf("RefreshPluginTitle: %v", err)
		}
	}
}

func pluginRuntime(
	service *services.PluginManager,
	id string,
	w http.ResponseWriter,
) (*plugins.PluginRuntime, bool) {
	runtime, err := service.Runtime(id)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrPluginNotFound):
			WriteError(w, http.StatusNotFound, "plugin not found")
		case errors.Is(err, services.ErrPluginDisabled):
			WriteError(w, http.StatusConflict, "plugin is disabled")
		default:
			WriteError(w, http.StatusInternalServerError, "plugin runtime unavailable")
		}
		return nil, false
	}

	return runtime, true
}
