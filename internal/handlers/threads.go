package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cobalto/noppera/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterThreads sets up thread-related routes.
func RegisterThreads(r chi.Router, db *pgxpool.Pool) {
	r.Get("/threads/{threadID}", getThread(db))
}

// ThreadResponse represents a thread with its replies.
type ThreadResponse struct {
	Thread  models.Post   `json:"thread"`
	Replies []models.Post `json:"replies"`
}

// getThread handles GET /threads/{threadID}, retrieving a thread and its replies.
func getThread(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		threadID, err := parseInt(chi.URLParam(r, "threadID"))
		if err != nil {
			http.Error(w, "Invalid thread ID", http.StatusBadRequest)
			return
		}

		// Get thread (original post)
		thread, err := models.GetPost(ctx, db, threadID)
		if err != nil || thread.ThreadID != nil || thread.ArchivedAt != nil {
			http.Error(w, "Thread not found or archived", http.StatusNotFound)
			return
		}

		// Get replies
		rows, err := db.Query(ctx,
			"SELECT id, board_id, thread_id, user_id, title, content, image_url, metadata, created_at, updated_at, last_bumped_at, archived_at "+
				"FROM posts WHERE thread_id = $1 AND archived_at IS NULL ORDER BY created_at ASC",
			threadID,
		)
		if err != nil {
			http.Error(w, "Failed to fetch replies", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var replies []models.Post
		for rows.Next() {
			var p models.Post
			if err := rows.Scan(&p.ID, &p.BoardID, &p.ThreadID, &p.UserID, &p.Title, &p.Content, &p.ImageURL, &p.Metadata,
				&p.CreatedAt, &p.UpdatedAt, &p.LastBumpedAt, &p.ArchivedAt); err != nil {
				http.Error(w, "Failed to scan replies", http.StatusInternalServerError)
				return
			}
			replies = append(replies, p)
		}

		response := ThreadResponse{
			Thread:  *thread,
			Replies: replies,
		}
		json.NewEncoder(w).Encode(response)
	}
}
