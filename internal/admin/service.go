package admin

import (
	"comics-galore-web/internal/database"
	"context"
	"fmt"
	"log/slog"
)

type Service interface {
	UpsertSocialMetrics(ctx context.Context, comments, messages, reactions int32) error
}

type service struct {
	queries database.Querier
	logger  *slog.Logger
}

func (s *service) UpsertSocialMetrics(ctx context.Context, comments, messages, reactions int32) error {
	// 1. Contextual Logging: Capture input params to trace drift from DO sync
	l := s.logger.With(
		"op", "UpsertSocialMetrics",
		"comments", comments,
		"messages", messages,
		"reactions", reactions,
	)

	params := database.SocialEngagementUpdateParams{
		Comments:  comments,
		Messages:  messages,
		Reactions: reactions,
	}

	// 2. Perform DB operation
	if err := s.queries.SocialEngagementUpdate(ctx, params); err != nil {
		// 3. Structured Error Context: Include the error as a field
		l.Error("failed to upsert social engagement metrics", "error", err)
		return fmt.Errorf("social engagement upsert: %w", err)
	}

	l.Debug("social metrics synced successfully")
	return nil
}

// NewService now accepts the logger and querier directly,
// making it easier to unit test without mocking the whole config.
func NewService(queries database.Querier, logger *slog.Logger) Service {
	return &service{
		queries: queries,
		logger:  logger.With("component", "admin_service"),
	}
}
