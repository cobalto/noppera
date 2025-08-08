package models

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Flag represents a post flag for moderation.
type Flag struct {
	ID        int       `json:"id"`
	PostID    int       `json:"post_id"`
	UserID    *int      `json:"user_id"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateFlag creates a new flag for a post.
func CreateFlag(ctx context.Context, db *pgxpool.Pool, flag *Flag) error {
	err := db.QueryRow(ctx,
		"INSERT INTO flags (post_id, user_id, reason, created_at) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
		flag.PostID, flag.UserID, flag.Reason, flag.CreatedAt,
	).Scan(&flag.ID, &flag.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create flag: %w", err)
	}
	return nil
}

// ListFlags retrieves all flags for admin review.
func ListFlags(ctx context.Context, db *pgxpool.Pool) ([]Flag, error) {
	rows, err := db.Query(ctx, "SELECT id, post_id, user_id, reason, created_at FROM flags ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to list flags: %w", err)
	}
	defer rows.Close()

	var flags []Flag
	for rows.Next() {
		var f Flag
		if err := rows.Scan(&f.ID, &f.PostID, &f.UserID, &f.Reason, &f.CreatedAt); err != nil {
			return nil, err
		}
		flags = append(flags, f)
	}
	return flags, nil
}
