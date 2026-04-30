package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/cobalto/noppera/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterSearch sets up search-related routes.
func RegisterSearch(r chi.Router, db *pgxpool.Pool) {
	r.Get("/posts/search", searchPosts(db))
}

// searchPosts handles GET /posts/search?query={term}&tag={tag}&board_id={id}&page={page}&limit={limit}, searching posts by content or tags.
func searchPosts(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		query := r.URL.Query().Get("query")
		tag := r.URL.Query().Get("tag")
		boardIDStr := r.URL.Query().Get("board_id")
		page := getIntQuery(r, "page", 1)
		limit := getIntQuery(r, "limit", 20)
		if limit > 100 {
			limit = 100
		}
		offset := (page - 1) * limit

		var boardID *int
		if boardIDStr != "" {
			id, err := parseInt(boardIDStr)
			if err != nil {
				http.Error(w, "Invalid board ID", http.StatusBadRequest)
				return
			}
			boardID = &id
		}

		sql := "SELECT id, board_id, thread_id, user_id, title, content, image_url, metadata, created_at, updated_at, last_bumped_at, archived_at " +
			"FROM posts WHERE archived_at IS NULL"
		args := []interface{}{}
		argNum := 1

		if query != "" {
			sanitizedQuery := sanitizeSearchQuery(query)
			if sanitizedQuery == "" {
				http.Error(w, "Invalid search query", http.StatusBadRequest)
				return
			}
			sql += fmt.Sprintf(" AND to_tsvector('english', content) @@ to_tsquery('english', $%d)", argNum)
			args = append(args, sanitizedQuery)
			argNum++
		}
		if tag != "" {
			sql += fmt.Sprintf(" AND metadata->'tags' @> $%d", argNum)
			args = append(args, fmt.Sprintf(`["%s"]`, tag))
			argNum++
		}
		if boardID != nil {
			sql += fmt.Sprintf(" AND board_id = $%d", argNum)
			args = append(args, *boardID)
			argNum++
		}

		sql += fmt.Sprintf(" ORDER BY last_bumped_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
		args = append(args, limit, offset)

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

func getIntQuery(r *http.Request, key string, defaultValue int) int {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(str)
	if err != nil || val < 1 {
		return defaultValue
	}
	return val
}

var searchWordRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

func sanitizeSearchQuery(query string) string {
	words := strings.Fields(query)
	if len(words) == 0 {
		return ""
	}
	var sanitized []string
	for _, word := range words {
		cleaned := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				return r
			}
			return -1
		}, word)
		if len(cleaned) > 0 && len(cleaned) <= 20 {
			sanitized = append(sanitized, cleaned)
		}
	}
	if len(sanitized) == 0 {
		return ""
	}
	result := strings.Join(sanitized, " & ")
	if !searchWordRegex.MatchString(result) {
		result = result + ":*"
	}
	return result
}
