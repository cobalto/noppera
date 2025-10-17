package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cobalto/noppera/internal/middleware"
	"github.com/cobalto/noppera/internal/models"
	"github.com/cobalto/noppera/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterPosts sets up post-related routes.
func RegisterPosts(r chi.Router, db *pgxpool.Pool, store storage.Storage) {
	r.Post("/boards/{boardSlug}/threads", createThread(db, store))
	r.Post("/threads/{threadID}/replies", createReply(db, store))
	r.With(middleware.Auth(store.Config())).Delete("/posts/{postID}/user", deletePostUser(db))
	r.With(middleware.Auth(store.Config()), middleware.AdminOnly).Delete("/posts/{postID}/admin", deletePostAdmin(db, store))
}

// createThread handles POST /boards/{boardSlug}/threads, creating a new thread.
// @Summary Create thread
// @Description Create a new thread in a board
// @Tags posts
// @Accept json
// @Produce json
// @Param boardSlug path string true "Board slug"
// @Param thread body object{title=string,content=string,image=string,tags=[]string,metadata=object} true "Thread data"
// @Success 201 {object} models.Post "Thread created successfully"
// @Failure 400 {string} string "Invalid request body"
// @Failure 404 {string} string "Board not found"
// @Failure 403 {string} string "Thread limit reached"
// @Failure 500 {string} string "Failed to create thread"
// @Router /boards/{boardSlug}/threads [post]
func createThread(db *pgxpool.Pool, store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		boardSlug := chi.URLParam(r, "boardSlug")
		cfg := store.Config()

		var input struct {
			Title    string                 `json:"title"`
			Content  string                 `json:"content"`
			Image    string                 `json:"image"`
			Tags     []string               `json:"tags"`
			Metadata map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if input.Content == "" || len(input.Content) > cfg.MaxPostLength {
			http.Error(w, "Content is required and must be within length limits", http.StatusBadRequest)
			return
		}
		if len(input.Tags) > cfg.MaxTags {
			http.Error(w, fmt.Sprintf("Too many tags, maximum is %d", cfg.MaxTags), http.StatusBadRequest)
			return
		}

		// Get board and validate settings
		var board models.Board
		err := db.QueryRow(ctx, "SELECT id, settings FROM boards WHERE slug = $1", boardSlug).Scan(&board.ID, &board.Settings)
		if err != nil {
			http.Error(w, "Board not found", http.StatusNotFound)
			return
		}

		maxThreads := cfg.DefaultMaxThreads
		if val, ok := board.Settings["max_threads"]; ok {
			if mt, ok := val.(float64); ok {
				maxThreads = int(mt)
			}
		}
		var threadCount int
		err = db.QueryRow(ctx, "SELECT COUNT(*) FROM posts WHERE board_id = $1 AND thread_id IS NULL AND archived_at IS NULL", board.ID).Scan(&threadCount)
		if err != nil || threadCount >= maxThreads {
			http.Error(w, "Thread limit reached for this board", http.StatusForbidden)
			return
		}

		var imageURL *string
		if input.Image != "" {
			imgData, err := base64.StdEncoding.DecodeString(input.Image)
			if err != nil {
				http.Error(w, "Invalid image data", http.StatusBadRequest)
				return
			}
			if maxImageSize, ok := board.Settings["max_image_size"].(float64); ok && len(imgData) > int(maxImageSize) {
				http.Error(w, "Image size exceeds board limit", http.StatusBadRequest)
				return
			}
			url, err := store.Upload(ctx, imgData, "jpg")
			if err != nil {
				http.Error(w, "Failed to upload image", http.StatusInternalServerError)
				return
			}
			imageURL = &url
		}

		userID := getUserID(r)
		if input.Metadata == nil {
			input.Metadata = make(map[string]interface{})
		}
		if len(input.Tags) > 0 {
			input.Metadata["tags"] = input.Tags
		}

		post := models.Post{
			BoardID:      board.ID,
			UserID:       userID,
			Title:        &input.Title,
			Content:      input.Content,
			ImageURL:     imageURL,
			Metadata:     input.Metadata,
			CreatedAt:    time.Now(),
			LastBumpedAt: time.Now(),
		}

		if err := models.CreatePost(ctx, db, &post); err != nil {
			http.Error(w, "Failed to create thread", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(post)
	}
}

// createReply handles POST /threads/{threadID}/replies, creating a reply to a thread.
func createReply(db *pgxpool.Pool, store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cfg := store.Config()
		threadID, err := parseInt(chi.URLParam(r, "threadID"))
		if err != nil {
			http.Error(w, "Invalid thread ID", http.StatusBadRequest)
			return
		}

		var input struct {
			Content  string                 `json:"content"`
			Image    string                 `json:"image"`
			Tags     []string               `json:"tags"`
			Metadata map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if input.Content == "" || len(input.Content) > cfg.MaxPostLength {
			http.Error(w, "Content is required and must be within length limits", http.StatusBadRequest)
			return
		}
		if len(input.Tags) > cfg.MaxTags {
			http.Error(w, fmt.Sprintf("Too many tags, maximum is %d", cfg.MaxTags), http.StatusBadRequest)
			return
		}

		// Get thread to verify it exists and get board_id
		thread, err := models.GetPost(ctx, db, threadID)
		if err != nil || thread.ThreadID != nil || thread.ArchivedAt != nil {
			http.Error(w, "Thread not found or archived", http.StatusNotFound)
			return
		}

		// Validate reply count
		var board models.Board
		err = db.QueryRow(ctx, "SELECT settings FROM boards WHERE id = $1", thread.BoardID).Scan(&board.Settings)
		if err != nil {
			http.Error(w, "Board not found", http.StatusNotFound)
			return
		}
		maxReplies := cfg.DefaultMaxReplies
		if val, ok := board.Settings["max_replies"]; ok {
			if mr, ok := val.(float64); ok {
				maxReplies = int(mr)
			}
		}
		var replyCount int
		err = db.QueryRow(ctx, "SELECT COUNT(*) FROM posts WHERE thread_id = $1 AND archived_at IS NULL", threadID).Scan(&replyCount)
		if err != nil || replyCount >= maxReplies {
			http.Error(w, "Reply limit reached for this thread", http.StatusForbidden)
			return
		}

		var imageURL *string
		if input.Image != "" {
			imgData, err := base64.StdEncoding.DecodeString(input.Image)
			if err != nil {
				http.Error(w, "Invalid image data", http.StatusBadRequest)
				return
			}
			if maxImageSize, ok := board.Settings["max_image_size"].(float64); ok && len(imgData) > int(maxImageSize) {
				http.Error(w, "Image size exceeds board limit", http.StatusBadRequest)
				return
			}
			url, err := store.Upload(ctx, imgData, "jpg")
			if err != nil {
				http.Error(w, "Failed to upload image", http.StatusInternalServerError)
				return
			}
			imageURL = &url
		}

		userID := getUserID(r)
		if input.Metadata == nil {
			input.Metadata = make(map[string]interface{})
		}
		if len(input.Tags) > 0 {
			input.Metadata["tags"] = input.Tags
		}

		post := models.Post{
			BoardID:      thread.BoardID,
			ThreadID:     &threadID,
			UserID:       userID,
			Content:      input.Content,
			ImageURL:     imageURL,
			Metadata:     input.Metadata,
			CreatedAt:    time.Now(),
			LastBumpedAt: time.Now(),
		}

		if err := models.CreatePost(ctx, db, &post); err != nil {
			http.Error(w, "Failed to create reply", http.StatusInternalServerError)
			return
		}

		// Bump thread
		if err := models.UpdateThreadBumpTime(ctx, db, threadID, time.Now()); err != nil {
			http.Error(w, "Failed to bump thread", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(post)
	}
}

// deletePostUser handles DELETE /posts/{postID}/user, allowing users to delete their own posts.
func deletePostUser(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		postID, err := parseInt(chi.URLParam(r, "postID"))
		if err != nil {
			http.Error(w, "Invalid post ID", http.StatusBadRequest)
			return
		}

		user, ok := r.Context().Value(middleware.UserContextKey).(*middleware.User)
		if !ok || user.ID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		post, err := models.GetPost(ctx, db, postID)
		if err != nil || post.UserID == nil || *post.UserID != user.ID {
			http.Error(w, "Post not found or not owned by user", http.StatusForbidden)
			return
		}

		if err := models.DeletePost(ctx, db, postID); err != nil {
			http.Error(w, "Failed to delete post", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// deletePostAdmin handles DELETE /posts/{postID}/admin, allowing admins to delete any post.
func deletePostAdmin(db *pgxpool.Pool, store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		postID, err := parseInt(chi.URLParam(r, "postID"))
		if err != nil {
			http.Error(w, "Invalid post ID", http.StatusBadRequest)
			return
		}

		post, err := models.GetPost(ctx, db, postID)
		if err != nil {
			http.Error(w, "Post not found", http.StatusNotFound)
			return
		}

		if post.ImageURL != nil {
			if err := store.Delete(ctx, *post.ImageURL); err != nil {
				http.Error(w, "Failed to delete image", http.StatusInternalServerError)
				return
			}
		}

		if err := models.DeletePost(ctx, db, postID); err != nil {
			http.Error(w, "Failed to delete post", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// getUserID extracts user ID from JWT context, if present.
func getUserID(r *http.Request) *int {
	if user, ok := r.Context().Value(middleware.UserContextKey).(*middleware.User); ok && user.ID != 0 {
		return &user.ID
	}
	return nil
}
