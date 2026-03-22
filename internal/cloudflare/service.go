package cloudflare

import (
	"bytes"
	"comics-galore-web/internal/config"
	"comics-galore-web/internal/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v3/client"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

type Images interface {
	UploadFromReader(ctx context.Context, reader io.Reader, filename string, metadata map[string]string, requireSignedUrls bool, shouldBackup bool) (*ImageResponse, error)
	UploadFromURL(ctx context.Context, url string, metadata map[string]string, requireSignedUrls bool) (*ImageResponse, error)
	ListImages(ctx context.Context, page, perPage int) (*ListImagesResponse, error)
	DeleteImage(ctx context.Context, imageID string) (bool, error)
	ImageDetails(ctx context.Context, imageID string) (*ImageResponse, error)
	GetBlob(ctx context.Context, imageID string) (io.ReadCloser, error)
	ListSigningKeys(ctx context.Context) (*SigningKeysResponse, error)
	CreateSigningKey(ctx context.Context, name string) (*SigningKeysResponse, error)
	DeleteSigningKey(ctx context.Context, name string) (*SigningKeysResponse, error)
	GetStats(ctx context.Context) (*StatsResponse, error)
	DeleteVariant(ctx context.Context, variantID string) error
	UpdateVariant(ctx context.Context, variantID string, options VariantOptions) (*VariantResponse, error)
	CreateVariant(ctx context.Context, variant Variant) (*VariantResponse, error)
	VariantDetails(ctx context.Context, variantID string) (*VariantResponse, error)
	ListVariants(ctx context.Context) (*VariantResponse, error)
}

type images struct {
	storage storage.Service
	logger  *slog.Logger
	cfg     config.Service
	client  *client.Client
}

func (s *images) ImageDetails(ctx context.Context, id string) (*ImageResponse, error) {
	// 1. Contextual Logging
	l := s.logger.With("op", "ImageDetails", "image_id", id)

	if id == "" {
		return nil, errors.New("image id is required")
	}

	// 2. Execute GET Request
	// Cloudflare path: /accounts/{account_id}/images/v1/{image_id}
	resp, err := s.client.Get(fmt.Sprintf("/%s", id), client.Config{Ctx: ctx})
	if err != nil {
		l.Error("network request failed", "error", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	result := &ImageResponse{}

	// 3. Always attempt to parse JSON to capture Cloudflare's error array
	if err := resp.JSON(result); err != nil {
		l.Error("failed to decode response", "status", resp.StatusCode(), "error", err)
		return nil, fmt.Errorf("decode error: %w", err)
	}

	// 4. Check Business Logic Success
	if !isSuccess(resp.StatusCode()) || !result.Success {
		l.Error("cloudflare returned error",
			"status", resp.StatusCode(),
			"cf_errors", result.Errors,
		)
		return nil, fmt.Errorf("cloudflare api error (status %d)", resp.StatusCode())
	}

	return result, nil
}

func NewService(cfg config.Service, opts ...ImageOption) Images {
	s := &images{
		cfg:    cfg,
		logger: cfg.GetLogger().With("component", "cloudflare_images"),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Initialize client once
	token := s.cfg.Get().Cloudflare.ImagesAPIKey
	if !strings.HasPrefix(token, "Bearer ") {
		token = "Bearer " + token
	}

	s.client = client.New().
		SetBaseURL(s.cfg.Get().Cloudflare.ImagesURL).
		AddHeaders(map[string][]string{
			"Accept":        {"application/json"},
			"Authorization": {token},
		})

	return s
}

func (s *images) DeleteImage(ctx context.Context, id string) (bool, error) {
	l := s.logger.With("op", "DeleteImage", "id", id)
	type Response struct {
		Success bool  `json:"success"`
		Errors  []any `json:"errors"` // Helpful for debugging failures
	}

	// Ensure the ID is not empty before making the call
	if id == "" {
		l.Error("image id is required")
		return false, fmt.Errorf("image id is required")
	}

	resp, err := s.client.Delete(fmt.Sprintf("/%s", id), client.Config{Ctx: ctx})
	if err != nil {
		l.Error("cloudflare delete request failed", "error", err)
		return false, fmt.Errorf("cloudflare delete request failed: %w", err)
	}

	// Cloudflare returns 4xx for missing images; handle gracefully if needed
	if resp.StatusCode() == 404 {
		l.Error("image not found", "id", id)
		return false, nil // Or handle as "already deleted"
	}

	if !isSuccess(resp.StatusCode()) {
		l.Error("cloudflare delete request failed", "id", id)
		return false, fmt.Errorf("cloudflare returned error status: %d", resp.StatusCode())
	}

	result := &Response{}
	if err := resp.JSON(result); err != nil {
		l.Error("cloudflare delete request failed", "id", id, "error", err)
		return false, fmt.Errorf("failed to decode cloudflare response: %w", err)
	}

	return result.Success, nil
}

func isSuccess(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

type ImageOption func(*images)

func WithBackup(storage storage.Service) ImageOption {
	return func(s *images) {
		s.storage = storage
	}
}

func (s *images) UploadFromURL(ctx context.Context, imageURL string, metadata map[string]string, requireSignedURLs bool) (*ImageResponse, error) {
	l := s.logger.With("op", "UploadFromURL", "url", imageURL)

	metadataBytes, _ := json.Marshal(metadata)

	// Fiber Client uses a map for form data
	formData := map[string]string{
		"url":               imageURL,
		"metadata":          string(metadataBytes),
		"requireSignedURLs": strconv.FormatBool(requireSignedURLs),
	}

	var response ImageResponse
	resp, err := s.client.Post(s.cfg.Get().Cloudflare.ImagesURL, client.Config{
		Ctx:      ctx,
		FormData: formData,
	})

	if err != nil {
		l.Error("http request failed", "error", err)
		return nil, err
	}

	if err := resp.JSON(&response); err != nil {
		l.Error("decode error", "error", err)
		return nil, fmt.Errorf("decode error: %w", err)
	}

	if !isSuccess(resp.StatusCode()) {
		l.Error("cloudflare error", "status", resp.StatusCode(), "details", response.Errors)
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	return &response, nil
}

func (s *images) ListImages(ctx context.Context, page, perPage int) (*ListImagesResponse, error) {
	l := s.logger.With("op", "ListImages", "page", page, "per_page", perPage)

	var response ListImagesResponse
	// Fiber Client handles QueryParams via a map[string]string
	resp, err := s.client.Get(s.cfg.Get().Cloudflare.ImagesURL, client.Config{
		Ctx: ctx,
		Param: map[string]string{
			"page":     strconv.Itoa(page),
			"per_page": strconv.Itoa(perPage),
		},
	})

	if err != nil {
		l.Error("http request failed", "error", err)
		return nil, err
	}

	if err := resp.JSON(&response); err != nil {
		l.Error("failed to decode cloudflare response", "error", err)
		return nil, err
	}

	if !isSuccess(resp.StatusCode()) {
		l.Info("cloudflare error", "status", resp.StatusCode(), "details", response.Errors)
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	return &response, nil
}

func (s *images) UploadFromReader(ctx context.Context, reader io.Reader, fileName string, metadata map[string]string, requireSignedURLs bool, shouldBackup bool) (*ImageResponse, error) {
	l := s.logger.With("op", "UploadFromReader", "filename", fileName)

	// Only close if the underlying reader is actually a Closer
	if rc, ok := reader.(io.ReadCloser); ok {
		defer func(rc io.ReadCloser) {
			err := rc.Close()
			if err != nil {
				l.Error("close reader failed", "error", err)
			}
		}(rc)
	}

	if reader == nil {
		l.Error("reader is nil")
		return nil, errors.New("reader is nil")
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		l.Error("reader failed", "error", err)
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}

	// Get the underlying bytes once
	data := buf.Bytes()

	if shouldBackup && s.storage != nil {
		go func(b []byte, name string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := s.storage.UploadFile(bgCtx, b, name); err != nil {
				l.Error("background backup failed", "error", err)
			}
		}(data, fileName)
	}

	metadataBytes, _ := json.Marshal(metadata)

	var response ImageResponse
	file := client.File{}
	file.SetName(fileName)
	file.SetReader(io.NopCloser(bytes.NewReader(data))) // bytes.Reader is fine here
	file.SetFieldName("file")

	// Use the pre-warmed s.client
	resp, err := s.client.Post("", client.Config{
		Ctx:  ctx,
		File: []*client.File{&file},
		FormData: map[string]string{
			"metadata":          string(metadataBytes),
			"requireSignedURLs": strconv.FormatBool(requireSignedURLs),
		},
	})

	if err != nil {
		l.Error("upload failed", "error", err)
		return nil, err
	}

	if err := resp.JSON(&response); err != nil {
		return nil, err
	}

	if !isSuccess(resp.StatusCode()) {
		l.Error("cloudflare rejected upload", "status", resp.StatusCode(), "details", response.Errors)
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	return &response, nil
}

func (s *images) GetBlob(ctx context.Context, imageID string) (io.ReadCloser, error) {
	l := s.logger.With("op", "GetBlob", "image_id", imageID)

	resp, err := s.client.Get(fmt.Sprintf("/%s/blob", imageID), client.Config{
		Ctx: ctx,
	})

	if err != nil {
		l.Error("network request failed", "error", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if !isSuccess(resp.StatusCode()) {
		l.Error("cloudflare returned error status", "status", resp.StatusCode())
		return nil, fmt.Errorf("cloudflare blob error: %d", resp.StatusCode())
	}

	return io.NopCloser(resp.BodyStream()), nil
}

func (s *images) ListSigningKeys(ctx context.Context) (*SigningKeysResponse, error) {
	l := s.logger.With("op", "ListSigningKeys")
	var res SigningKeysResponse

	resp, err := s.client.Get("/keys", client.Config{Ctx: ctx})
	if err != nil {
		l.Error("request failed", "error", err)
		return nil, fmt.Errorf("list keys request failed: %w", err)
	}

	if err := resp.JSON(&res); err != nil {
		l.Error("failed to decode response", "status", resp.StatusCode(), "error", err)
		return nil, fmt.Errorf("decode error: %w", err)
	}

	if !isSuccess(resp.StatusCode()) || !res.Success {
		l.Error("cloudflare api error", "status", resp.StatusCode(), "cf_errors", res.Errors)
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	l.Debug("signing keys listed", "count", len(res.Result.Keys))
	return &res, nil
}

func (s *images) CreateSigningKey(ctx context.Context, name string) (*SigningKeysResponse, error) {
	l := s.logger.With("op", "CreateSigningKey", "key_name", name)
	var res SigningKeysResponse

	resp, err := s.client.Put(fmt.Sprintf("/keys/%s", name), client.Config{Ctx: ctx})
	if err != nil {
		l.Error("request failed", "error", err)
		return nil, fmt.Errorf("create key request failed: %w", err)
	}

	if err := resp.JSON(&res); err != nil {
		l.Error("failed to decode response", "status", resp.StatusCode(), "error", err)
		return nil, fmt.Errorf("decode error: %w", err)
	}

	if !isSuccess(resp.StatusCode()) || !res.Success {
		l.Error("cloudflare api error", "status", resp.StatusCode(), "cf_errors", res.Errors)
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	l.Info("signing key created")
	return &res, nil
}

func (s *images) DeleteSigningKey(ctx context.Context, name string) (*SigningKeysResponse, error) {
	l := s.logger.With("op", "DeleteSigningKey", "key_name", name)
	var res SigningKeysResponse

	resp, err := s.client.Delete(fmt.Sprintf("/keys/%s", name), client.Config{Ctx: ctx})
	if err != nil {
		l.Error("request failed", "error", err)
		return nil, fmt.Errorf("delete key request failed: %w", err)
	}

	if err := resp.JSON(&res); err != nil {
		l.Error("failed to decode response", "status", resp.StatusCode(), "error", err)
		return nil, fmt.Errorf("decode error: %w", err)
	}

	if !isSuccess(resp.StatusCode()) || !res.Success {
		l.Error("cloudflare api error", "status", resp.StatusCode(), "cf_errors", res.Errors)
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	l.Info("signing key deleted")
	return &res, nil
}

func (s *images) GetStats(ctx context.Context) (*StatsResponse, error) {
	l := s.logger.With("op", "GetStats")
	var res StatsResponse

	resp, err := s.client.Get("/stats", client.Config{Ctx: ctx})
	if err != nil {
		l.Error("request failed", "error", err)
		return nil, fmt.Errorf("get stats request failed: %w", err)
	}

	if err := resp.JSON(&res); err != nil {
		l.Error("failed to decode response", "status", resp.StatusCode(), "error", err)
		return nil, fmt.Errorf("decode error: %w", err)
	}

	if !isSuccess(resp.StatusCode()) || !res.Success {
		l.Error("cloudflare api error", "status", resp.StatusCode(), "cf_errors", res.Errors)
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	l.Debug("stats fetched", "current", res.Result.Count.Current, "allowed", res.Result.Count.Allowed)
	return &res, nil
}

func (s *images) ListVariants(ctx context.Context) (*VariantResponse, error) {
	l := s.logger.With("op", "ListVariants")
	var res VariantResponse

	resp, err := s.client.Get("/variants", client.Config{Ctx: ctx})
	if err != nil {
		l.Error("request failed", "error", err)
		return nil, fmt.Errorf("list variants failed: %w", err)
	}

	if err := resp.JSON(&res); err != nil {
		l.Error("decode failed", "status", resp.StatusCode(), "error", err)
		return nil, err
	}

	if !isSuccess(resp.StatusCode()) || !res.Success {
		l.Error("api error", "status", resp.StatusCode(), "cf_errors", res.Errors)
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	l.Debug("variants listed", "count", len(res.Result.Variants))
	return &res, nil
}

func (s *images) VariantDetails(ctx context.Context, variantID string) (*VariantResponse, error) {
	l := s.logger.With("op", "VariantDetails", "variant_id", variantID)
	var res VariantResponse

	resp, err := s.client.Get(fmt.Sprintf("/variants/%s", variantID), client.Config{Ctx: ctx})
	if err != nil {
		l.Error("request failed", "error", err)
		return nil, err
	}

	if err := resp.JSON(&res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (s *images) CreateVariant(ctx context.Context, variant Variant) (*VariantResponse, error) {
	l := s.logger.With("op", "CreateVariant", "variant_id", variant.ID)
	var res VariantResponse

	// Cloudflare expects JSON body for variant creation
	resp, err := s.client.Post("/variants", client.Config{
		Ctx:  ctx,
		Body: variant,
	})
	if err != nil {
		l.Error("request failed", "error", err)
		return nil, err
	}

	if err := resp.JSON(&res); err != nil {
		l.Error("decode failed", "error", err)
		return nil, err
	}

	if !isSuccess(resp.StatusCode()) || !res.Success {
		l.Error("api error", "status", resp.StatusCode(), "cf_errors", res.Errors)
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	l.Info("variant created")
	return &res, nil
}

func (s *images) UpdateVariant(ctx context.Context, variantID string, options VariantOptions) (*VariantResponse, error) {
	l := s.logger.With("op", "UpdateVariant", "variant_id", variantID)
	var res VariantResponse

	// Update only the options via PATCH
	body := map[string]any{"options": options}

	resp, err := s.client.Patch(fmt.Sprintf("/variants/%s", variantID), client.Config{
		Ctx:  ctx,
		Body: body,
	})
	if err != nil {
		l.Error("request failed", "error", err)
		return nil, err
	}

	if err := resp.JSON(&res); err != nil {
		return nil, err
	}

	l.Info("variant updated (cache purged)")
	return &res, nil
}

func (s *images) DeleteVariant(ctx context.Context, variantID string) error {
	l := s.logger.With("op", "DeleteVariant", "variant_id", variantID)

	resp, err := s.client.Delete(fmt.Sprintf("/variants/%s", variantID), client.Config{Ctx: ctx})
	if err != nil {
		l.Error("request failed", "error", err)
		return err
	}

	if !isSuccess(resp.StatusCode()) {
		l.Error("api error", "status", resp.StatusCode())
		return fmt.Errorf("api error: %d", resp.StatusCode())
	}

	l.Info("variant deleted (cache purged)")
	return nil
}
