package archive

import (
	"comics-galore-web/internal/config"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"io"
	"log/slog"
	"path/filepath"
)

type handler struct {
	svc    Service
	logger *slog.Logger
}

type Handler interface {
	Upload(c fiber.Ctx) error
	Download(c fiber.Ctx) error
	RegisterRoutes(app *fiber.App)
}

func NewHandler(cfg config.Service, svc Service) Handler {
	return &handler{
		svc:    svc,
		logger: cfg.GetLogger().With("component", "archive_handler"),
	}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	group := app.Group("/api/v1/archive")

	group.Post("/upload", h.Upload)
	group.Get("/download/:filename", h.Download)
}

func (h *handler) Upload(c fiber.Ctx) error {
	l := h.logger.With("op", "Upload")

	// 1. Parse the multipart form file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		l.Warn("no file found in request", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file field is required"})
	}

	// 2. Open the file to read bytes
	file, err := fileHeader.Open()
	if err != nil {
		l.Error("failed to open form file", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "file read error"})
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		l.Error("failed to read file data", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "buffer read error"})
	}

	// 3. Delegate to service
	// Note: We use the filename provided in the form, sanitized
	fileName := filepath.Base(fileHeader.Filename)
	if err := h.svc.UploadFile(c.Context(), data, fileName); err != nil {
		l.Error("service upload failed", "filename", fileName, "error", err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "storage upload failed"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "file uploaded successfully",
		"filename": fileName,
		"size":     fileHeader.Size,
	})
}

func (h *handler) Download(c fiber.Ctx) error {
	fileName := c.Params("filename")
	l := h.logger.With("op", "Download", "filename", fileName)

	if fileName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "filename parameter is required"})
	}

	// 1. Get the stream from S3
	// We pass the fiber context which will handle cancellation if the user disconnects
	stream, err := h.svc.DownloadFile(c.Context(), fileName)
	if err != nil {
		l.Error("service download failed", "error", err)
		// We return 404 because usually, an S3 error here means the file isn't there
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found or inaccessible"})
	}

	// 2. Set headers for file download
	c.Set(fiber.HeaderContentType, "application/octet-stream")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", fileName))

	// 3. Stream the file to the client
	// Fiber v3's SendStream will pull from S3 and push to the client chunk-by-chunk
	l.Info("starting file stream to client")
	if err := c.SendStream(stream); err != nil {
		l.Error("streaming interrupted", "error", err)
		_ = stream.Close()
		return err
	}

	// 4. Always close the S3 body when finished
	return stream.Close()
}
