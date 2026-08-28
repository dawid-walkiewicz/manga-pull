package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"main/db"
	"main/plugins"
	"main/services"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(errorResponse{Error: message}); err != nil {
		log.Printf("write error response: %v", err)
	}
}

func writePluginServiceError(w http.ResponseWriter, err error, fetchMessage string) {
	switch {
	case errors.Is(err, plugins.ErrPluginNotFound):
		WriteError(w, http.StatusNotFound, "plugin not found")
	case errors.Is(err, plugins.ErrPluginDisabled):
		WriteError(w, http.StatusConflict, "plugin is disabled")
	case errors.Is(err, plugins.ErrPluginRuntime):
		WriteError(w, http.StatusInternalServerError, "plugin runtime unavailable")
	case errors.Is(err, plugins.ErrPluginFailedFetch):
		WriteError(w, http.StatusBadGateway, fetchMessage)
	case errors.Is(err, services.ErrTitleAlreadySaved):
		WriteError(w, http.StatusConflict, "title already saved")
	case errors.Is(err, db.ErrNotFound):
		WriteError(w, http.StatusNotFound, "title not found")
	default:
		WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
