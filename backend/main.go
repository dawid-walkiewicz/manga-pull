package main

import (
	"context"
	"fmt"
	"log"
	"main/common"
	"main/db"
	"main/handlers"
	"main/jobs"
	"main/plugins"
	"main/services"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	common.InitEnv()

	ctx := context.Background()
	database, err := db.Open(ctx, filepath.Join(common.DataDir, "database.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	dbStore := db.NewStore(database)

	pluginManager := plugins.NewPluginManager(dbStore, common.PluginsDir)

	if err := pluginManager.Start(context.Background()); err != nil {
		log.Printf("plugin manager start: %v", err)
	}

	titleManager := services.NewTitleManager(dbStore, pluginManager)

	jobRunner := jobs.NewRunner(dbStore, titleManager)
	jobWorker := jobs.NewWorker(dbStore, jobRunner)
	go jobWorker.Run(ctx)

	r := chi.NewRouter()
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			handlers.WriteError(w, http.StatusNotFound, "not found")
			return
		}

		http.NotFound(w, r)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			handlers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		handlers.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
		},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ok"))
	})

	r.Get("/api/titles", handlers.ListTitlesHandler(dbStore))
	r.Get("/api/titles/{id}", handlers.GetTitleHandler(dbStore))
	r.Delete("/api/titles/{id}", handlers.DeleteTitleHandler(dbStore))

	r.Get("/api/plugins", handlers.ListPluginsHandler(pluginManager))
	r.Post("/api/plugins/scan", handlers.ScanPluginsHandler(pluginManager))
	r.Get("/api/plugins/{pluginId}/icon", handlers.GetPluginIconHandler(pluginManager))
	r.Post("/api/plugins/{pluginId}/enable", handlers.EnablePluginHandler(pluginManager))
	r.Post("/api/plugins/{pluginId}/disable", handlers.DisablePluginHandler(pluginManager))
	r.Get("/api/plugins/{pluginId}/search", handlers.SearchPluginTitleHandler(pluginManager))
	r.Get("/api/plugins/{pluginId}/titles", handlers.BrowsePluginTitlesHandler(pluginManager))
	r.Get("/api/plugins/{pluginId}/titles/{titleId}", handlers.GetPluginTitleHandler(titleManager))
	r.Post("/api/plugins/{pluginId}/titles/{titleId}/save", handlers.SavePluginTitleHandler(titleManager))
	r.Post("/api/plugins/{pluginId}/titles/{titleId}/refresh", handlers.RefreshPluginTitleHandler(titleManager))

	r.Get("/api/jobs", handlers.ListJobsHandler(dbStore))
	// r.Get("/api/jobs/watch", handlers.WatchJobsHandler(dbStore))
	r.Post("/api/jobs/{jobId}/cancel", handlers.CancelJobHandler(dbStore, jobWorker))
	// r.Post("/api/jobs/{jobId}/pause", handlers.PauseJobHandler(jobWorker))
	r.Post("/api/jobs/refresh/{titleId}", handlers.RefreshPluginTitleJobHandler(dbStore))

	app_port := fmt.Sprintf(":%d", common.AppPort)
	log.Printf("server listening on %s", app_port)
	if err := http.ListenAndServe(app_port, r); err != nil {
		log.Fatal(err)
	}
}

func isAPIPath(path string) bool {
	return path == "/api" || strings.HasPrefix(path, "/api/")
}
