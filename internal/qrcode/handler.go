package qrcode

import (
	"comics-galore-web/internal/config"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v3"
)

type handler struct {
	svc    Service
	logger *slog.Logger
}

type Handler interface {
	GetQRCode(c fiber.Ctx) error
	RegisterRoutes(app *fiber.App)
}

func NewHandler(cfg config.Service, service Service) Handler {
	return &handler{
		svc:    service,
		logger: cfg.GetLogger().With("component", "qrcode_handler"),
	}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	group := app.Group("/api/v1/qrcode")
	group.Get("/image", h.GetQRCode)
}

func (h *handler) GetQRCode(c fiber.Ctx) error {
	// 1. Identify the data type
	dataType := c.Query("type", "text")

	// 2. Extract parameters
	// BUG FIX: We should exclude 'type' from the params map so it doesn't
	// interfere with specific formatters (like vCard) that might loop over the map.
	params := c.Queries()
	delete(params, "type")

	l := h.logger.With(
		"op", "GetQRCode",
		"data_type", dataType,
		"param_count", len(params),
	)

	// 3. Call service
	pngBytes, err := h.svc.GeneratePNG(dataType, params)
	if err != nil {
		l.Warn("request failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to generate %s QR code", dataType),
		})
	}

	// 4. Set headers
	c.Set(fiber.HeaderContentType, "image/png")

	// Cache Control: Crypto and Wifi QRs are usually static, so we cache them.
	// 3600 seconds = 1 hour.
	c.Set(fiber.HeaderCacheControl, "public, max-age=3600")

	// 5. Return the binary data
	// Using c.Send(pngBytes) is correct for []byte.
	// Since pngBytes is a slice and not an io.ReadCloser, we don't need SendStream.
	return c.Send(pngBytes)
}
