package models

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User represents a user account.
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUser creates a new user in the database.
func CreateUser(ctx context.Context, db *pgxpool.Pool, user *User) error {
	err := db.QueryRow(ctx,
		"INSERT INTO users (username, password, is_admin) VALUES ($1, $2, $3) RETURNING id, created_at",
		user.Username, user.Password, user.IsAdmin,
	).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUserByUsername retrieves a user by username.
func GetUserByUsername(ctx context.Context, db *pgxpool.Pool, username string) (*User, error) {
	var u User
	err := db.QueryRow(ctx,
		"SELECT id, username, password, is_admin, created_at FROM users WHERE username = $1",
		username,
	).Scan(&u.ID, &u.Username, &u.Password, &u.IsAdmin, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}
