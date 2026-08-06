// Package entity defines the core domain types for the article module.
package entity

import "time"

// Article represents a blog-style article owned by a user.
type Article struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
