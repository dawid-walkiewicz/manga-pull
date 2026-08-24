package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"main/db"
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
			writeError(w, http.StatusBadGateway, "failed to fetch titles from plugin")
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
			writeError(w, http.StatusBadGateway, "failed to fetch titles from plugin")
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
			writeError(w, http.StatusBadGateway, "failed to fetch title from plugin")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(title); err != nil {
			log.Printf("GetPluginTitle: %v", err)
		}
	}
}

func SavePluginTitleHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		titleID := chi.URLParam(r, "titleId")

		createdID, err := service.SaveTitle(r.Context(), id, titleID)
		if err != nil {
			switch {
			case errors.Is(err, services.ErrPluginNotFound):
				writeError(w, http.StatusNotFound, "plugin not found")
			case errors.Is(err, services.ErrPluginDisabled):
				writeError(w, http.StatusConflict, "plugin is disabled")
			case errors.Is(err, services.ErrPluginRuntime):
				writeError(w, http.StatusInternalServerError, "plugin runtime unavailable")
			case errors.Is(err, services.ErrPluginFailedFetch):
				log.Println(err)
				writeError(w, http.StatusBadGateway, "failed to fetch title from plugin")
			default:
				writeError(w, http.StatusInternalServerError, "error during saving title")
			}

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

func RefreshPluginTitleHandler(service *services.PluginManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		titleID := chi.URLParam(r, "titleId")

		refreshedTitle, err := service.RefreshTitle(r.Context(), id, titleID)
		if err != nil {
			log.Printf("RefreshPluginTitle: %v", err)
			switch {
			case errors.Is(err, services.ErrPluginNotFound):
				writeError(w, http.StatusNotFound, "plugin not found")
			case errors.Is(err, services.ErrPluginDisabled):
				writeError(w, http.StatusConflict, "plugin is disabled")
			case errors.Is(err, services.ErrPluginRuntime):
				writeError(w, http.StatusInternalServerError, "plugin runtime unavailable")
			case errors.Is(err, services.ErrPluginFailedFetch):
				writeError(w, http.StatusBadGateway, "failed to fetch title from plugin")
			case errors.Is(err, sql.ErrNoRows), errors.Is(err, db.ErrNotFound):
				writeError(w, http.StatusNotFound, "title not found")
			default:
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
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
			writeError(w, http.StatusNotFound, "plugin not found")
		case errors.Is(err, services.ErrPluginDisabled):
			writeError(w, http.StatusConflict, "plugin is disabled")
		default:
			writeError(w, http.StatusInternalServerError, "plugin runtime unavailable")
		}
		return nil, false
	}

	return runtime, true
}
