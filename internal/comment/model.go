package comment

import (
	"comics-galore-web/internal/database"
	"github.com/google/uuid"
	"time"
)

type Comment struct {
	ID        uuid.UUID     `json:"id"`
	PostID    uuid.UUID     `json:"post_id"`
	ParentID  uuid.NullUUID `json:"parent_id"`
	Depth     int32         `json:"depth"`
	UserID    string        `json:"user_id"`
	Content   string        `json:"content"`
	CreatedAt time.Time     `json:"created_at"`
	Replies   []*Comment    `json:"replies,omitempty"`
}

func New(comment database.Comment, replies []*Comment) *Comment {
	return &Comment{
		ID:        comment.ID,
		PostID:    comment.PostID,
		ParentID:  comment.ParentID,
		Depth:     comment.Depth,
		UserID:    comment.UserID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt.Time,
		Replies:   replies,
	}
}

type Request struct {
	PostID   uuid.UUID     `json:"post_id"`
	ParentID uuid.NullUUID `json:"parent_id"`
	UserID   string        `json:"user_id"`
	Content  string        `json:"content"`
}
