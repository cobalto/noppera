package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/cobalto/noppera/internal/middleware"
	"github.com/cobalto/noppera/internal/models"
	"github.com/cobalto/noppera/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterBoards sets up board-related routes.
func RegisterBoards(r chi.Router, db *pgxpool.Pool, store storage.Storage) {
	r.Get("/boards", listBoards(db))
	r.With(middleware.Auth(store.Config()), middleware.AdminOnly).Post("/boards", createBoard(db))
}

// listBoards handles GET /boards, listing all boards.
func listBoards(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		boards, err := models.ListBoards(r.Context(), db)
		if err != nil {
			http.Error(w, "Failed to list boards", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(boards)
	}
}

// createBoard handles POST /boards, creating a new board (admin only).
func createBoard(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var board models.Board
		if err := json.NewDecoder(r.Body).Decode(&board); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if board.Name == "" || board.Slug == "" {
			http.Error(w, "Name and slug are required", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		if err := models.CreateBoard(ctx, db, &board); err != nil {
			http.Error(w, "Failed to create board", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(board)
	}
}

// parseInt converts a string to an integer or returns an error.
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
