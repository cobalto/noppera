package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cobalto/noppera/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterSearch sets up search-related routes.
func RegisterSearch(r chi.Router, db *pgxpool.Pool) {
	r.Get("/posts/search", searchPosts(db))
}

// searchPosts handles GET /posts/search?query={term}&tag={tag}&board_id={id}, searching posts by content or tags.
// @Summary Search posts
// @Description Search posts by content, tags, or board
// @Tags search
// @Produce json
// @Param query query string false "Search query"
// @Param tag query string false "Tag to filter by"
// @Param board_id query int false "Board ID to filter by"
// @Success 200 {array} models.Post "Search results"
// @Failure 400 {string} string "Invalid board ID"
// @Failure 500 {string} string "Failed to search posts"
// @Router /posts/search [get]
func searchPosts(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query().Get("query")
		tag := r.URL.Query().Get("tag")
		boardIDStr := r.URL.Query().Get("board_id")

		var boardID *int
		if boardIDStr != "" {
			id, err := parseInt(boardIDStr)
			if err != nil {
				http.Error(w, "Invalid board ID", http.StatusBadRequest)
				return
			}
			boardID = &id
		}

		// Build SQL query with full-text search
		sql := "SELECT id, board_id, thread_id, user_id, title, content, image_url, metadata, created_at, updated_at, last_bumped_at, archived_at " +
			"FROM posts WHERE archived_at IS NULL"
		args := []interface{}{}
		var conditions []string

		if query != "" {
			conditions = append(conditions, "to_tsvector('english', content) @@ to_tsquery('english', $1)")
			args = append(args, strings.ReplaceAll(query, " ", " & "))
		}
		if tag != "" {
			conditions = append(conditions, "metadata->'tags' @> $2")
			args = append(args, fmt.Sprintf(`["%s"]`, tag))
		}
		if boardID != nil {
			conditions = append(conditions, "board_id = $3")
			args = append(args, *boardID)
		}

		if len(conditions) > 0 {
			sql += " AND " + strings.Join(conditions, " AND ")
		}
		sql += " ORDER BY last_bumped_at DESC LIMIT 100"

		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			http.Error(w, "Failed to search posts", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var posts []models.Post
		for rows.Next() {
			var p models.Post
			if err := rows.Scan(&p.ID, &p.BoardID, &p.ThreadID, &p.UserID, &p.Title, &p.Content, &p.ImageURL, &p.Metadata,
				&p.CreatedAt, &p.UpdatedAt, &p.LastBumpedAt, &p.ArchivedAt); err != nil {
				http.Error(w, "Failed to scan posts", http.StatusInternalServerError)
				return
			}
			posts = append(posts, p)
		}

		json.NewEncoder(w).Encode(posts)
	}
}
