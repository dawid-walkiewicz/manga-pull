package main

import (
	"context"
	"fmt"
	"log"
	"main/common"
	"main/db"
	"main/handlers"
	"main/services"
	"net/http"
	"path/filepath"

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

	pluginManager := services.NewPluginManager(dbStore, common.PluginsDir)

	if err := pluginManager.Start(context.Background()); err != nil {
		log.Printf("plugin manager start: %v", err)
	}

	r := chi.NewRouter()

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
	r.Post("/api/plugins/{id}/enable", handlers.EnablePluginHandler(pluginManager))
	r.Post("/api/plugins/{id}/disable", handlers.DisablePluginHandler(pluginManager))
	r.Get("/api/plugins/{id}/search", handlers.SearchPluginTitleHandler(pluginManager))
	r.Get("/api/plugins/{id}/titles", handlers.BrowsePluginTitlesHandler(pluginManager))
	r.Get("/api/plugins/{id}/titles/{titleId}", handlers.GetPluginTitleHandler(pluginManager))
	r.Post("/api/plugins/{id}/titles/{titleId}/save", handlers.SavePluginTitleHandler(pluginManager))
	r.Post("/api/plugins/{id}/titles/{titleId}/refresh", handlers.RefreshPluginTitleHandler(pluginManager))

	app_port := fmt.Sprintf(":%d", common.AppPort)
	log.Printf("server listening on %s", app_port)
	if err := http.ListenAndServe(app_port, r); err != nil {
		log.Fatal(err)
	}
}
