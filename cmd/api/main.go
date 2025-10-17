package main

import (
	"context"
	"log"
	"net/http"

	"github.com/cobalto/noppera/docs"
	"github.com/cobalto/noppera/internal/config"
	"github.com/cobalto/noppera/internal/handlers"
	"github.com/cobalto/noppera/internal/jobs"
	"github.com/cobalto/noppera/internal/middleware"
	"github.com/cobalto/noppera/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Noppera Image Board API
// @version 1.0
// @description A 4chan-inspired image board API built with Go, Chi, PostgreSQL, and JSONB
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Initialize Swagger docs
	docs.SwaggerInfo.Host = "localhost:8080"
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

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
	r.Use(middleware.CORS(cfg))

	// Health check endpoints (no rate limiting)
	handlers.RegisterHealth(r, db)

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

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
