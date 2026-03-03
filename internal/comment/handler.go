package comment

import (
	"comics-galore-web/internal/config"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type handler struct {
	svc    Service
	logger *slog.Logger
}

type Handler interface {
	RegisterRoutes(app *fiber.App)
	Get(c fiber.Ctx) error
	Create(c fiber.Ctx) error
	Update(c fiber.Ctx) error
	Delete(c fiber.Ctx) error
}

func NewHandler(cfg config.Service, svc Service) Handler {
	return &handler{
		svc:    svc,
		logger: cfg.GetLogger().With("component", "comment_handler"),
	}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	group := app.Group("/api/v1/comment")
	group.Get("/list/:postId", h.Get)
	group.Put("/:postId", h.Update)
	group.Post("/:postId", h.Create)
	group.Delete("/:postId", h.Delete)
}

func (h *handler) Get(c fiber.Ctx) error {
	postIDStr := c.Params("postId")
	l := h.logger.With(
		"op", "Get",
		"post_id_raw", postIDStr,
		"method", c.Method(),
		"path", c.Path(),
	)

	id, err := uuid.Parse(postIDStr)
	if err != nil {
		l.Warn("invalid post uuid provided")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid post identifier",
		})
	}

	comments, err := h.svc.GetComments(c.Context(), id)
	if err != nil {
		l.Error("service failed to retrieve comments", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	l.Debug("successfully served comments list", "count", len(comments))
	return c.JSON(comments)
}

func (h *handler) Create(c fiber.Ctx) error {
	l := h.logger.With("op", "Create", "ip", c.IP())

	payload := new(Request)
	if err := c.Bind().Body(payload); err != nil {
		l.Warn("payload binding failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot parse request body",
		})
	}

	// Update logger with specific IDs from the payload
	l = l.With("post_id", payload.PostID, "user_id", payload.UserID)

	comment, err := h.svc.CreateComment(c.Context(), *payload)
	if err != nil {
		l.Error("comment creation failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create comment",
		})
	}

	l.Info("new comment published", "comment_id", comment.ID)
	return c.Status(fiber.StatusCreated).JSON(comment)
}

func (h *handler) Update(c fiber.Ctx) error {
	commentID := c.Params("postId") // Adjusted from "id" to match RegisterRoutes path
	l := h.logger.With("op", "Update", "comment_id", commentID)

	l.Info("update request received (not implemented)")
	return c.SendStatus(fiber.StatusNotImplemented)
}

func (h *handler) Delete(c fiber.Ctx) error {
	commentID := c.Params("postId") // Adjusted from "id" to match RegisterRoutes path
	l := h.logger.With("op", "Delete", "comment_id", commentID)

	l.Warn("delete request received (not implemented)")
	return c.SendStatus(fiber.StatusNotImplemented)
}
