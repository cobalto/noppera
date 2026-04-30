package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cobalto/noppera/docs"
	"github.com/cobalto/noppera/internal/config"
	"github.com/cobalto/noppera/internal/handlers"
	"github.com/cobalto/noppera/internal/jobs"
	"github.com/cobalto/noppera/internal/middleware"
	"github.com/cobalto/noppera/internal/models"
	"github.com/cobalto/noppera/internal/storage"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func validateConfig(cfg config.Config) error {
	if len(cfg.JWTSecret) < 16 {
		return errors.New("JWT_SECRET must be at least 16 characters")
	}
	return nil
}

func checkAndCreateAdmin(db *pgxpool.Pool, cfg config.Config) error {
	ctx := context.Background()
	count, err := models.CountUsers(ctx, db)
	if err != nil {
		return err
	}
	if count == 0 {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cfg.JWTSecret), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user := models.User{
			Username: "admin",
			Password: string(hashedPassword),
			IsAdmin:  true,
		}
		if err := models.CreateUser(ctx, db, &user); err != nil {
			return err
		}
		log.Println("Auto-created admin user")
	}
	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func main() {
	// Initialize Swagger docs
	docs.SwaggerInfo.Host = "localhost:8080"
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	cfg := config.Load()
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}
	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	store, err := storage.NewStorage(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	if err := checkAndCreateAdmin(db, cfg); err != nil {
		log.Printf("Warning: Could not create admin user: %v", err)
	}

	archiver := jobs.NewArchiver(db, store, cfg)
	archiver.Start()
	defer archiver.Stop()

	r := chi.NewRouter()
	r.Use(middleware.Logging(cfg))
	r.Use(middleware.JSONContentType)
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)

	if cfg.StorageType == "local" {
		uploadDir := cfg.UploadDir
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				log.Fatalf("Failed to create upload directory: %v", err)
			}
		}
		r.Handle("/uploads/*", http.StripPrefix("/uploads", http.FileServer(http.Dir(uploadDir))))
	}

	r.Get("/health", healthHandler)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RateLimitPublic(cfg))
		r.Use(middleware.BodyLimit(cfg.MaxBodySize))
		handlers.RegisterBoards(r, db, store)
		handlers.RegisterPosts(r, db, store)
		handlers.RegisterSearch(r, db)
		handlers.RegisterFlags(r, db, cfg)
		handlers.RegisterThreads(r, db)
	})
	handlers.RegisterAuth(r, db, cfg)

	srv := &http.Server{
		Addr:    cfg.APIHost + ":" + cfg.APIPort,
		Handler: r,
	}

	go func() {
		log.Printf("Starting server on %s:%s", cfg.APIHost, cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
