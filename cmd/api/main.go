package main

import (
	"context"
	"log"
	"net/http"

	"github.com/cobalto/noppera/internal/config"
	"github.com/cobalto/noppera/internal/handlers"
	"github.com/cobalto/noppera/internal/jobs"
	"github.com/cobalto/noppera/internal/middleware"
	"github.com/cobalto/noppera/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	store, err := storage.NewStorage(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	archiver := jobs.NewArchiver(db, store, cfg)
	archiver.Start()
	defer archiver.Stop()

	r := chi.NewRouter()
	r.Use(middleware.Logging(cfg))
	r.Group(func(r chi.Router) {
		r.Use(middleware.RateLimitPublic(cfg))
		handlers.RegisterBoards(r, db, store)
		handlers.RegisterPosts(r, db, store)
		handlers.RegisterSearch(r, db)
		handlers.RegisterFlags(r, db, cfg)
		handlers.RegisterThreads(r, db)
	})
	handlers.RegisterAuth(r, db, cfg)

	log.Printf("Starting server on %s:%s", cfg.APIHost, cfg.APIPort)
	if err := http.ListenAndServe(cfg.APIHost+":"+cfg.APIPort, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
