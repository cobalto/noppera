package models

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Board represents a board entity.
type Board struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Description string                 `json:"description"`
	Settings    map[string]interface{} `json:"settings"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ListBoards retrieves all boards.
func ListBoards(ctx context.Context, db *pgxpool.Pool) ([]Board, error) {
	rows, err := db.Query(ctx, "SELECT id, name, slug, description, settings, created_at FROM boards")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boards []Board
	for rows.Next() {
		var b Board
		if err := rows.Scan(&b.ID, &b.Name, &b.Slug, &b.Description, &b.Settings, &b.CreatedAt); err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	return boards, nil
}

// CreateBoard creates a new board.
func CreateBoard(ctx context.Context, db *pgxpool.Pool, board *Board) error {
	return db.QueryRow(ctx,
		"INSERT INTO boards (name, slug, description, settings) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
		board.Name, board.Slug, board.Description, board.Settings,
	).Scan(&board.ID, &board.CreatedAt)
}
