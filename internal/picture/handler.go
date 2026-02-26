package picture

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
)

type handler struct {
	svc    Service
	logger *slog.Logger
}

type Handler interface {
	GetPicture(c fiber.Ctx) error
	RegisterRoutes(app *fiber.App)
}

func NewHandler(service Service, logger *slog.Logger) Handler {
	return &handler{
		svc:    service,
		logger: logger.With("component", "picture_handler"),
	}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	// Hooks are global to the app; ensure this is called once per service lifecycle
	app.Hooks().OnPostShutdown(func(err error) error {
		h.logger.Info("triggering service shutdown hook")
		return h.svc.Shutdown(10 * time.Second)
	})

	group := app.Group("/api/v1/image")
	// Changed route to match your param usage or query usage
	group.Get("/process", h.GetPicture)
}

func (h *handler) GetPicture(c fiber.Ctx) error {
	originalKey := c.Params("key")

	widthStr := c.Query("width", "1024")
	qualityStr := c.Query("quality", "80")

	l := h.logger.With(
		"op", "GetPicture",
		"key", originalKey,
		"req_width", widthStr,
		"req_quality", qualityStr,
	)

	if originalKey == "" {
		l.Warn("request rejected: missing key")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing ?key="})
	}

	width, _ := strconv.Atoi(widthStr)
	quality, _ := strconv.Atoi(qualityStr)

	// Sanitization (Log if we had to override extreme values)
	if width < 10 || width > 4096 {
		l.Debug("width out of bounds, resetting to default", "original", width)
		width = 1024
	}
	if quality < 1 || quality > 100 {
		quality = 80
	}

	// 2. Process Image
	processedReader, err := h.svc.ProcessAndCacheFromS3(originalKey, width, quality)
	if err != nil {
		l.Error("processing service failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "processing failed"})
	}

	// 3. Set Headers
	c.Set(fiber.HeaderContentType, "image/jpeg")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`inline; filename="processed_%s.jpg"`, originalKey))
	c.Set(fiber.HeaderCacheControl, "public, max-age=86400")

	// 4. Fiber v3 SendStream Handling
	// BUG FIX: fiber.Ctx.SendStream(io.Reader) handles the streaming.
	// However, if the service returns an io.ReadCloser, we must ensure it is closed
	// AFTER the stream is fully written to the client to avoid leaking memory/S3 connections.
	if err := c.SendStream(processedReader); err != nil {
		l.Error("failed to stream image to client", "error", err)
		_ = processedReader.Close()
		return err
	}

	// Successfully handed off to stream; close the reader.
	return processedReader.Close()
}
