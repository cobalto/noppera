package models

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Post represents a thread or reply.
type Post struct {
	ID           int                    `json:"id"`
	BoardID      int                    `json:"board_id"`
	ThreadID     *int                   `json:"thread_id"`
	UserID       *int                   `json:"user_id"`
	Title        *string                `json:"title"`
	Content      string                 `json:"content"`
	ImageURL     *string                `json:"image_url"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    *time.Time             `json:"updated_at"`
	LastBumpedAt time.Time              `json:"last_bumped_at"`
	ArchivedAt   *time.Time             `json:"archived_at"`
}

// ListThreads retrieves all active threads for a board.
func ListThreads(ctx context.Context, db *pgxpool.Pool, boardID string) ([]Post, error) {
	rows, err := db.Query(ctx,
		"SELECT id, board_id, user_id, title, content, image_url, metadata, created_at, updated_at, last_bumped_at, archived_at "+
			"FROM posts WHERE board_id = $1 AND thread_id IS NULL AND archived_at IS NULL ORDER BY last_bumped_at DESC",
		boardID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.BoardID, &p.UserID, &p.Title, &p.Content, &p.ImageURL, &p.Metadata,
			&p.CreatedAt, &p.UpdatedAt, &p.LastBumpedAt, &p.ArchivedAt); err != nil {
			return nil, err
		}
		threads = append(threads, p)
	}
	return threads, nil
}

// CreatePost creates a new post (thread or reply).
func CreatePost(ctx context.Context, db *pgxpool.Pool, post *Post) error {
	return db.QueryRow(ctx,
		"INSERT INTO posts (board_id, user_id, title, content, image_url, metadata, created_at, last_bumped_at) "+
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at, last_bumped_at",
		post.BoardID, post.UserID, post.Title, post.Content, post.ImageURL, post.Metadata, post.CreatedAt, post.LastBumpedAt,
	).Scan(&post.ID, &post.CreatedAt, &post.LastBumpedAt)
}

// UpdateThreadBumpTime updates the thread's last_bumped_at.
func UpdateThreadBumpTime(ctx context.Context, db *pgxpool.Pool, threadID int, bumpTime time.Time) error {
	_, err := db.Exec(ctx, "UPDATE posts SET last_bumped_at = $1 WHERE id = $2 AND thread_id IS NULL", bumpTime, threadID)
	return err
}

// GetPost retrieves a post by ID.
func GetPost(ctx context.Context, db *pgxpool.Pool, postID int) (*Post, error) {
	var p Post
	err := db.QueryRow(ctx,
		"SELECT id, board_id, thread_id, user_id, title, content, image_url, metadata, created_at, updated_at, last_bumped_at, archived_at "+
			"FROM posts WHERE id = $1", postID,
	).Scan(&p.ID, &p.BoardID, &p.ThreadID, &p.UserID, &p.Title, &p.Content, &p.ImageURL, &p.Metadata,
		&p.CreatedAt, &p.UpdatedAt, &p.LastBumpedAt, &p.ArchivedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("post not found")
	}
	return &p, err
}

// DeletePost deletes a post by ID.
func DeletePost(ctx context.Context, db *pgxpool.Pool, postID int) error {
	_, err := db.Exec(ctx, "DELETE FROM posts WHERE id = $1", postID)
	return err
}
