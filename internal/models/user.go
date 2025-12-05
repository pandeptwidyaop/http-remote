package models

import "time"

// User represents a user account.
type User struct {
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	ID           int64     `json:"id"`
	IsAdmin      bool      `json:"is_admin"`
}

// Session represents a user session.
type Session struct {
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
}
