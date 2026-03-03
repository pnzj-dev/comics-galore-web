package event

import (
	"bufio"
	"log/slog"

	"comics-galore-web/internal/broadcaster"
	"comics-galore-web/internal/config"
	"github.com/gofiber/fiber/v3"
)

type handler struct {
	cfg         config.Service
	broadcaster broadcaster.Service
	logger      *slog.Logger
}

type Handler interface {
	RegisterRoutes(app *fiber.App)
	Listen(c fiber.Ctx) error
}

func NewHandler(cfg config.Service, broadcaster broadcaster.Service) Handler {
	return &handler{
		cfg:         cfg,
		broadcaster: broadcaster,
		logger:      cfg.GetLogger().With("component", "event_handler"),
	}
}

func (h *handler) Listen(c fiber.Ctx) error {
	postID := c.Params("postId")
	if postID == "" {
		h.logger.Warn("invalid request: missing postId")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid postId"})
	}

	h.logger.Info("client connecting to event stream", "postId", postID)

	ctx := c.Context()
	bc := h.broadcaster.Get(postID)
	clientChan := make(chan string, 10)
	bc.Register <- clientChan

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	return c.SendStreamWriter(func(w *bufio.Writer) {
		defer func() {
			bc.Unregister <- clientChan
			close(clientChan)
			h.logger.Info("client disconnected", "postId", postID)
		}()

		for {
			select {
			case msg, ok := <-clientChan:
				if !ok {
					h.logger.Warn("broadcaster channel closed", "postId", postID)
					return
				}

				if _, err := w.WriteString("event: comment\n"); err != nil {
					h.logger.Error("failed to write event name to stream", "error", err, "postId", postID)
					return
				}

				if _, err := w.WriteString("data: " + msg + "\n\n"); err != nil {
					h.logger.Error("failed to write data to stream", "error", err, "postId", postID)
					return
				}

				if err := w.Flush(); err != nil {
					h.logger.Error("failed to flush stream buffer", "error", err, "postId", postID)
					return
				}

			case <-ctx.Done():
				h.logger.Debug("stream context cancelled", "postId", postID)
				return
			}
		}
	})
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	app.Get("/api/v1/events/comment/:postId", h.Listen)
}
