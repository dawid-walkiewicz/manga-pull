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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ok"))
	})

	r.Get("/api/titles", handlers.ListTitlesHandler(dbStore))
	r.Get("/api/titles/{id}", handlers.GetTitleHandler(dbStore))
	r.Post("/api/titles", handlers.CreateTitleHandler(dbStore))
	r.Get("/api/plugins", handlers.ListPluginsHandler(pluginManager))
	r.Post("/api/plugins/scan", handlers.ScanPluginsHandler(pluginManager))
	r.Post("/api/plugins/{id}/enable", handlers.EnablePluginHandler(pluginManager))
	r.Post("/api/plugins/{id}/disable", handlers.DisablePluginHandler(pluginManager))
	r.Get("/api/plugins/{id}/search", handlers.SearchTitleHandler(pluginManager))

	app_port := fmt.Sprintf(":%d", common.AppPort)
	log.Printf("server listening on %s", app_port)
	if err := http.ListenAndServe(app_port, r); err != nil {
		log.Fatal(err)
	}
}
