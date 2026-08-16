package main

import (
	"context"
	"log"
	"main/db"
	"main/handlers"
	"net/http"

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

	r.Get("/api/titles", handlers.ListTitlesHandler(store))
	r.Get("/api/titles/{id}", handlers.GetTitleHandler(store))
	r.Post("/api/titles", handlers.CreateTitleHandler(store))
	r.Get("/api/plugins", handlers.ListPluginsHandler(store))
	r.Post("/api/plugins/scan", handlers.ScanPlugins(store))

	log.Println("server listening on :8000")
	if err := http.ListenAndServe(":8000", r); err != nil {
		log.Fatal(err)
	}
}
