package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cobalto/noppera/internal/config"
	"github.com/cobalto/noppera/internal/middleware"
	"github.com/cobalto/noppera/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterFlags sets up flag-related routes.
func RegisterFlags(r chi.Router, db *pgxpool.Pool, cfg config.Config) {
	r.Post("/posts/{postID}/flag", flagPost(db, cfg))
	r.With(middleware.Auth(cfg), middleware.AdminOnly).Get("/flags", listFlags(db))
}

// flagPost handles POST /posts/{postID}/flag, allowing users or anonymous to flag a post.
func flagPost(db *pgxpool.Pool, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		postID, err := parseInt(chi.URLParam(r, "postID"))
		if err != nil {
			http.Error(w, "Invalid post ID", http.StatusBadRequest)
			return
		}

		var input struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if input.Reason == "" || len(input.Reason) > cfg.MaxPostLength {
			http.Error(w, "Reason is required and must be within length limits", http.StatusBadRequest)
			return
		}

		// Get user ID from JWT, if present
		var userID *int
		if user, ok := r.Context().Value(middleware.UserContextKey).(*middleware.User); ok && user.ID != 0 {
			userID = &user.ID
		}

		flag := models.Flag{
			PostID:    postID,
			UserID:    userID,
			Reason:    input.Reason,
			CreatedAt: time.Now(),
		}

		if err := models.CreateFlag(ctx, db, &flag); err != nil {
			http.Error(w, "Failed to create flag", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(flag)
	}
}

// listFlags handles GET /flags, returning all flags for admin review.
func listFlags(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		flags, err := models.ListFlags(ctx, db)
		if err != nil {
			http.Error(w, "Failed to list flags", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(flags)
	}
}
