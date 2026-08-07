// Package entity defines the core domain model for comments.
package entity

import "time"

// Comment represents a single comment left on an article.
type Comment struct {
	ID        int64     `json:"id"`
	ArticleID int64     `json:"article_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	Version   int64     `json:"version" gorm:"not null;default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
