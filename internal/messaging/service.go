package messaging

import (
	"comics-galore-web/internal/database"
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type service struct {
	querier *database.Queries
	logger  *slog.Logger
}

type Service interface {
	MarkAsRead(ctx context.Context, userID string, conversationID uuid.UUID, messageID uuid.UUID) error
	CreateGroup(ctx context.Context, name string) (string, error)
	InsertMessage(ctx context.Context, conversationID uuid.UUID, senderID string, content string) (database.Message, error)
	AddParticipant(ctx context.Context, conversationID uuid.UUID, userID string) error
	DeleteConversation(ctx context.Context, conversationID uuid.UUID) error
	ListConversations(ctx context.Context, userID string) ([]database.ListUserConversationsRow, error)
	GetConversationHistory(ctx context.Context, conversationID uuid.UUID, cursor uuid.NullUUID, limit int32) ([]database.GetMessagesBeforeRow, error)
	GetOrCreateDirectConversation(ctx context.Context, userID1, userID2 string) (int64, error)
}

func NewService(pool *pgxpool.Pool, logger *slog.Logger) Service {
	return &service{
		querier: database.New(pool),
		logger:  logger.With("component", "messaging_service"),
	}
}

func (s *service) ListConversations(ctx context.Context, userID string) ([]database.ListUserConversationsRow, error) {
	l := s.logger.With("op", "ListConversations", "user_id", userID)

	rows, err := s.querier.ListUserConversations(ctx, userID)
	if err != nil {
		l.Error("failed to fetch user conversations", "error", err)
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	l.Debug("conversations retrieved", "count", len(rows))
	return rows, nil
}

func (s *service) GetConversationHistory(ctx context.Context, conversationID uuid.UUID, cursor uuid.NullUUID, limit int32) ([]database.GetMessagesBeforeRow, error) {
	l := s.logger.With("op", "GetConversationHistory", "conversation_id", conversationID, "limit", limit)

	rows, err := s.querier.GetMessagesBefore(ctx, database.GetMessagesBeforeParams{
		ConversationID: conversationID,
		Cursor:         cursor,
		Limit:          limit,
	})
	if err != nil {
		l.Error("database error retrieving history", "error", err)
		return nil, fmt.Errorf("get history: %w", err)
	}

	return rows, nil
}

func (s *service) GetOrCreateDirectConversation(ctx context.Context, userID1, userID2 string) (int64, error) {
	l := s.logger.With("op", "GetOrCreateDirect", "user_1", userID1, "user_2", userID2)

	idRaw, err := s.querier.GetOrCreateDirectConversation(ctx, database.GetOrCreateDirectConversationParams{
		GetOrCreateDirectConversation:   userID1,
		GetOrCreateDirectConversation_2: userID2,
	})
	if err != nil {
		l.Error("direct conversation lookup failed", "error", err)
		return 0, fmt.Errorf("get/create direct: %w", err)
	}

	id, ok := idRaw.(int64)
	if !ok {
		l.Error("unexpected database return type", "type", fmt.Sprintf("%T", idRaw))
		return 0, fmt.Errorf("invalid ID type returned from database")
	}

	return id, nil
}

func (s *service) AddParticipant(ctx context.Context, conversationID uuid.UUID, userID string) error {
	l := s.logger.With("op", "AddParticipant", "conversation_id", conversationID, "user_id", userID)

	err := s.querier.AddParticipant(ctx, database.AddParticipantParams{
		ConversationID: conversationID,
		UserID:         userID,
	})
	if err != nil {
		l.Error("failed to add participant", "error", err)
		return fmt.Errorf("add participant: %w", err)
	}

	l.Info("participant added to conversation")
	return nil
}

func (s *service) CreateGroup(ctx context.Context, name string) (string, error) {
	l := s.logger.With("op", "CreateGroup", "group_name", name)

	id, err := s.querier.CreateConversation(ctx, database.CreateConversationParams{
		Name:    &name,
		IsGroup: true,
	})
	if err != nil {
		l.Error("group creation failed", "error", err)
		return "", fmt.Errorf("create group: %w", err)
	}

	l.Info("new group conversation created", "conversation_id", id)
	return id.String(), nil
}

func (s *service) InsertMessage(ctx context.Context, conversationID uuid.UUID, senderID string, content string) (database.Message, error) {
	l := s.logger.With("op", "InsertMessage", "conversation_id", conversationID, "sender_id", senderID)

	msg, err := s.querier.InsertMessage(ctx, database.InsertMessageParams{
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
	})
	if err != nil {
		// Note: We don't log 'content' for privacy and log size reasons
		l.Error("failed to persist message", "error", err)
		return database.Message{}, fmt.Errorf("insert message: %w", err)
	}

	l.Debug("message saved")
	return msg, nil
}

func (s *service) MarkAsRead(ctx context.Context, userID string, conversationID uuid.UUID, messageID uuid.UUID) error {
	l := s.logger.With("op", "MarkAsRead", "user_id", userID, "conversation_id", conversationID)

	err := s.querier.UpdateLastRead(ctx, database.UpdateLastReadParams{
		UserID:            userID,
		ConversationID:    conversationID,
		LastReadMessageID: uuid.NullUUID{UUID: messageID, Valid: true},
	})
	if err != nil {
		l.Error("failed to update read status", "message_id", messageID, "error", err)
		return fmt.Errorf("mark as read: %w", err)
	}

	return nil
}

func (s *service) DeleteConversation(ctx context.Context, conversationID uuid.UUID) error {
	l := s.logger.With("op", "DeleteConversation", "conversation_id", conversationID)

	err := s.querier.DeleteConversation(ctx, conversationID)
	if err != nil {
		l.Error("failed to delete conversation", "error", err)
		return fmt.Errorf("delete conversation: %w", err)
	}

	l.Warn("conversation permanently removed")
	return nil
}
