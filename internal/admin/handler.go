package admin

import (
	"comics-galore-web/internal/config"
	"github.com/gofiber/fiber/v3"
	"log/slog"
)

type Handler interface {
	SyncSocialMetrics(c fiber.Ctx) error
	RegisterRoutes(c *fiber.App)
}

type handler struct {
	svc    Service
	logger *slog.Logger
}

func (h *handler) RegisterRoutes(c *fiber.App) {
	group := c.Group("/api/v1/admin")
	group.Post("/sync-stats", h.SyncSocialMetrics) // add authorization check
}

func NewHandler(cfg config.Service, svc Service) Handler {
	return &handler{
		svc:    svc,
		logger: cfg.GetLogger().With("component", "admin_handler"),
	}
}

func (h *handler) SyncSocialMetrics(c fiber.Ctx) error {
	type Payload struct {
		Comments  int32 `json:"comments"`
		Messages  int32 `json:"messages"`
		Reactions int32 `json:"reactions"`
	}

	p := new(Payload)
	if err := c.Bind().Body(&p); err != nil {
		h.logger.Warn("invalid sync payload received", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}

	// Logging before the service call creates a trace of the incoming sync request
	h.logger.Info("processing social metrics sync",
		"comments", p.Comments,
		"messages", p.Messages,
		"reactions", p.Reactions,
	)

	err := h.svc.UpsertSocialMetrics(c.Context(), p.Comments, p.Messages, p.Reactions)
	if err != nil {
		// Log the error returned by the service
		h.logger.Error("failed to upsert social metrics", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "sync failed"})
	}

	return c.SendStatus(fiber.StatusOK)
}
