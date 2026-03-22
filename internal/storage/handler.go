package storage

import (
	"comics-galore-web/internal/auth"
	"comics-galore-web/internal/config"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type Handler interface {
	RegisterRoutes(app *fiber.App)
	GetPresignedURL(c fiber.Ctx) error
	Upload(c fiber.Ctx) error
	Download(c fiber.Ctx) error
}

type handler struct {
	cfg    config.Service
	svc    Service
	logger *slog.Logger
}

func NewHandler(cfg config.Service, svc Service) Handler {
	return &handler{
		svc:    svc,
		cfg:    cfg,
		logger: cfg.GetLogger().With("component", "storage_handler"),
	}
}

func (h *handler) RegisterRoutes(app *fiber.App) {
	// Grouping helps with middleware attachment later (e.g., Auth)
	group := app.Group("/api/v1/storage")
	group.Post("/upload", h.Upload)
	group.Get("/download/:filename", h.Download)
	group.Get("/presigned", auth.JWTProtected(h.cfg), auth.HasRole("writer", "admin"), h.GetPresignedURL)
}

func (h *handler) GetPresignedURL(c fiber.Ctx) error {
	// 1. Extract params from Query.
	// Content-Type must be a query param because GET requests don't have bodies.
	filename := c.Query("filename")
	contentType := c.Query("content_type", "application/octet-stream")

	if filename == "" {
		return fiber.NewError(http.StatusBadRequest, "filename is required")
	}

	// 2. Security: Generate a unique, server-controlled key
	// This prevents users from overwriting existing files via the 'key' param.
	ext := filepath.Ext(filename)
	key := fmt.Sprintf("uploads/%d-%s%s", time.Now().Unix(), uuid.NewString(), ext)

	l := h.logger.With(
		"op", "GetPresignedURL",
		"key", key,
		"content_type", contentType,
	)

	// 3. Request the URL from S3 Service
	url, err := h.svc.GetPresignedUploadURL(c.Context(), key, contentType)
	if err != nil {
		l.Error("failed to generate presigned url", "error", err)
		return fiber.NewError(http.StatusInternalServerError, "Storage service unavailable")
	}

	// 4. Return both the URL (for the PUT) and the Key (to save in your DB)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"url": url,
		"key": key,
	})
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
