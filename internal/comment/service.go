package comment

import (
	"comics-galore-web/internal/broadcaster"
	"comics-galore-web/internal/config"
	"comics-galore-web/internal/database"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service interface {
	CreateComment(ctx context.Context, req Request) (*Comment, error)
	GetComments(ctx context.Context, postID uuid.UUID) ([]*Comment, error)
}

type service struct {
	logger      *slog.Logger
	querier     *database.Queries
	cfg         config.Service
	broadcaster broadcaster.Service
}

func NewService(cfg config.Service, pool *pgxpool.Pool, broadcaster broadcaster.Service, logger *slog.Logger) Service {
	return &service{
		cfg:         cfg,
		querier:     database.New(pool),
		broadcaster: broadcaster,
		logger:      logger.With("service", "comment"),
	}
}

func (s *service) CreateComment(ctx context.Context, req Request) (*Comment, error) {
	l := s.logger.With(
		"op", "CreateComment",
		"post_id", req.PostID,
		"user_id", req.UserID,
	)

	// 1. Validation Logic
	if req.ParentID.Valid {
		parentDepth, err := s.querier.GetCommentDepth(ctx, req.ParentID.UUID)
		if err != nil {
			l.Warn("parent comment lookup failed", "parent_id", req.ParentID.UUID, "error", err)
			return nil, fmt.Errorf("parent comment not found: %w", err)
		}

		if int(parentDepth)+1 > s.cfg.Get().MaxCommentNesting {
			l.Warn("nesting limit reached",
				"attempted_depth", parentDepth+1,
				"limit", s.cfg.Get().MaxCommentNesting)
			return nil, fmt.Errorf("max nesting level of %d exceeded", s.cfg.Get().MaxCommentNesting)
		}
	}

	// 2. Persistence
	created, err := s.querier.CreateComment(ctx, database.CreateCommentParams{
		PostID:   req.PostID,
		UserID:   req.UserID,
		Content:  req.Content,
		ParentID: req.ParentID,
	})
	if err != nil {
		l.Error("failed to save comment to database", "error", err)
		return nil, fmt.Errorf("failed to save comment: %w", err)
	}

	newComment := New(created, []*Comment{})

	// 3. Async Broadcasting with dedicated context
	go func(c *Comment) {
		// Using a timeout for the broadcast to prevent hanging goroutines
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		msg, err := json.Marshal(c)
		if err != nil {
			l.Error("failed to marshal comment for broadcast", "comment_id", c.ID, "error", err)
			return
		}

		room := s.broadcaster.Get(c.PostID.String())
		// The broadcaster.Get we refactored earlier already logs its own creation

		select {
		case room.Messages <- string(msg):
			l.Debug("comment broadcasted", "comment_id", c.ID)
		case <-bgCtx.Done():
			l.Error("broadcast timed out", "comment_id", c.ID)
		}
	}(newComment)

	l.Info("comment created successfully", "comment_id", newComment.ID)
	return newComment, nil
}

func (s *service) GetComments(ctx context.Context, postID uuid.UUID) ([]*Comment, error) {
	l := s.logger.With("op", "GetComments", "post_id", postID)

	raw, err := s.querier.GetCommentsByPostID(ctx, postID)
	if err != nil {
		l.Error("failed to fetch comments from database", "error", err)
		return nil, fmt.Errorf("fetch comments: %w", err)
	}

	l.Debug("raw comments retrieved", "count", len(raw))
	return s.buildCommentTree(l, raw), nil
}

func (s *service) buildCommentTree(l *slog.Logger, commentList []database.GetCommentsByPostIDRow) []*Comment {
	if len(commentList) == 0 {
		return []*Comment{}
	}

	roots := make([]*Comment, 0)
	commentMap := make(map[uuid.UUID]*Comment, len(commentList))

	// Map conversion
	for _, c := range commentList {
		commentMap[c.ID] = &Comment{
			ID:        c.ID,
			PostID:    c.PostID,
			ParentID:  c.ParentID,
			Depth:     c.Depth,
			UserID:    c.UserID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt.Time,
			Replies:   []*Comment{},
		}
	}

	// Linkage
	for _, c := range commentList {
		comment := commentMap[c.ID]
		if !c.ParentID.Valid {
			roots = append(roots, comment)
		} else if parent, ok := commentMap[c.ParentID.UUID]; ok {
			parent.Replies = append(parent.Replies, comment)
		} else {
			// Structured warning for data integrity issues
			l.Warn("orphan comment detected during tree build",
				"comment_id", c.ID,
				"expected_parent", c.ParentID.UUID)
		}
	}

	// Modern Sort Logic (Go 1.21+)
	slices.SortFunc(roots, func(a, b *Comment) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	for _, r := range roots {
		sortReplies(r)
	}

	return roots
}

func sortReplies(c *Comment) {
	if len(c.Replies) == 0 {
		return
	}
	slices.SortFunc(c.Replies, func(a, b *Comment) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	for i := range c.Replies {
		sortReplies(c.Replies[i])
	}
}
