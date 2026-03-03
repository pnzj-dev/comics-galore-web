package messaging

import (
	"comics-galore-web/internal/config"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type handler struct {
	svc    Service
	logger *slog.Logger
}

type Handler interface {
	RegisterRoutes(app *fiber.App)
	List(c fiber.Ctx) error
	Delete(c fiber.Ctx) error
	MarkRead(c fiber.Ctx) error
	GetHistory(c fiber.Ctx) error
	SendMessage(c fiber.Ctx) error
	StartDirectChat(c fiber.Ctx) error
}

func NewHandler(cfg config.Service, svc Service) Handler {
	return &handler{
		svc:    svc,
		logger: cfg.GetLogger().With("component", "messaging_handler"),
	}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1/messaging")
	api.Get("/conversations", h.List)
	api.Delete("/conversations/:id", h.Delete)
	api.Get("/conversations/:id", h.GetHistory)
	api.Post("/conversations/:id/messages", h.SendMessage)
	api.Post("/conversations/:id/read/:messageId", h.MarkRead)
	api.Post("/conversations/direct/:otherUserId", h.StartDirectChat)
}

func (h *handler) List(c fiber.Ctx) error {
	userID, ok := c.Locals("userId").(string)
	l := h.logger.With("op", "List", "user_id", userID)

	if !ok || userID == "" {
		l.Warn("unauthorized access attempt to inbox")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	conversations, err := h.svc.ListConversations(c.Context(), userID)
	if err != nil {
		l.Error("failed to load user inbox", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load inbox"})
	}

	l.Debug("inbox retrieved", "count", len(conversations))
	return c.JSON(conversations)
}

func (h *handler) GetHistory(c fiber.Ctx) error {
	convID := c.Params("id")
	cursor := c.Query("cursor")
	limitStr := c.Query("limit", "20")

	l := h.logger.With("op", "GetHistory", "conversation_id", convID, "limit", limitStr)

	limitInt, err := strconv.Atoi(limitStr)
	if err != nil {
		l.Warn("invalid limit parameter", "value", limitStr)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid limit"})
	}

	convIdUUID, err := uuid.Parse(convID)
	if err != nil {
		l.Warn("invalid conversation uuid")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid Id"})
	}

	var cursorUuid uuid.NullUUID
	if cursor != "" && cursor != "0" {
		parsed, err := uuid.Parse(cursor)
		if err != nil {
			l.Warn("invalid cursor uuid", "cursor", cursor)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid cursor"})
		}
		cursorUuid = uuid.NullUUID{UUID: parsed, Valid: true}
	}

	messages, err := h.svc.GetConversationHistory(c.Context(), convIdUUID, cursorUuid, int32(limitInt))
	if err != nil {
		l.Error("service failed to fetch history", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to retrieve history"})
	}

	return c.JSON(messages)
}

func (h *handler) SendMessage(c fiber.Ctx) error {
	l := h.logger.With("op", "SendMessage")

	payload := new(Request)
	if err := c.Bind().Body(payload); err != nil {
		l.Warn("payload binding failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Enrich logger with payload context
	l = l.With("conversation_id", payload.ConversationID, "sender_id", payload.SenderID)

	_, err := h.svc.InsertMessage(c.Context(), payload.ConversationID, payload.SenderID, payload.Content)
	if err != nil {
		l.Error("service failed to insert message", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not send message"})
	}

	l.Info("message processed")
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "sent"})
}

func (h *handler) MarkRead(c fiber.Ctx) error {
	convID := c.Params("id")
	messageID := c.Params("messageId")
	userID, _ := c.Locals("userId").(string)

	l := h.logger.With(
		"op", "MarkRead",
		"user_id", userID,
		"conversation_id", convID,
		"message_id", messageID,
	)

	convIdUUID, err := uuid.Parse(convID)
	if err != nil {
		l.Warn("invalid conversation uuid")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid conversation id"})
	}

	messageUUID, err := uuid.Parse(messageID)
	if err != nil {
		l.Warn("invalid message uuid")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid message id"})
	}

	err = h.svc.MarkAsRead(c.Context(), userID, convIdUUID, messageUUID)
	if err != nil {
		l.Error("service failed to update read status", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *handler) Delete(c fiber.Ctx) error {
	convID := c.Params("id")
	l := h.logger.With("op", "Delete", "conversation_id", convID)

	convIdUUID, err := uuid.Parse(convID)
	if err != nil {
		l.Warn("invalid conversation uuid")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid conversation id"})
	}

	if err := h.svc.DeleteConversation(c.Context(), convIdUUID); err != nil {
		l.Error("service failed to delete conversation", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "delete failed"})
	}

	l.Info("conversation deleted successfully")
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *handler) StartDirectChat(c fiber.Ctx) error {
	userID, _ := c.Locals("userId").(string)
	otherUserID := c.Params("otherUserId")

	l := h.logger.With("op", "StartDirectChat", "user_id", userID, "target_user_id", otherUserID)

	if userID == otherUserID {
		l.Warn("attempted self-chat")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "self-chat not allowed"})
	}

	convID, err := h.svc.GetOrCreateDirectConversation(c.Context(), userID, otherUserID)
	if err != nil {
		l.Error("service failed to start direct chat", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to start chat"})
	}

	l.Info("direct chat initialized", "conversation_id", convID)
	return c.JSON(fiber.Map{"conversation_id": convID})
}
