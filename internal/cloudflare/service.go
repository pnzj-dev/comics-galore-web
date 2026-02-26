package cloudflare

import (
	"bytes"
	"comics-galore-web/internal/archive"
	"comics-galore-web/internal/config"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"resty.dev/v3"
	"strconv"
	"time"
)

type Images interface {
	UploadFromReader(ctx context.Context, reader io.Reader, filename string, metadata map[string]string, requireSignedUrls bool, shouldBackup bool) (*ImageResponse, error)
	UploadFromURL(ctx context.Context, url string, metadata map[string]string, requireSignedUrls bool) (*ImageResponse, error)
	ListImages(ctx context.Context, page, perPage int) (*ListImagesResponse, error)
}

type images struct {
	storage archive.Service
	logger  *slog.Logger
	client  *resty.Client
	cfg     config.Service
}

type ImageOption func(*images)

func WithBackup(s3 archive.Service) ImageOption {
	return func(s *images) {
		s.storage = s3
	}
}

func NewService(cfg config.Service, logger *slog.Logger, opts ...ImageOption) Images {
	client := resty.New().
		SetTimeout(2*time.Minute).
		SetHeader("Accept", "application/json")

	s := &images{
		cfg:    cfg,
		client: client,
		logger: logger,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *images) UploadFromURL(ctx context.Context, imageURL string, metadata map[string]string, requireSignedURLs bool) (*ImageResponse, error) {
	l := s.logger.With("op", "UploadFromURL", "url", imageURL)
	l.Debug("starting cloudflare url upload")

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		l.Error("metadata marshaling failed", "error", err)
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	var response ImageResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetAuthToken(s.cfg.Get().Cloudflare.ImagesAPIKey).
		SetFormData(map[string]string{
			"url":               imageURL,
			"metadata":          string(metadataBytes),
			"requireSignedURLs": strconv.FormatBool(requireSignedURLs),
		}).
		SetResult(&response).
		SetError(&response).
		Post(s.cfg.Get().Cloudflare.ImagesURL)

	if err != nil {
		l.Error("http request failed", "error", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.IsError() {
		l.Error("cloudflare api returned error", "status", resp.StatusCode(), "details", response.Errors)
		return nil, fmt.Errorf("cloudflare error: %d", resp.StatusCode())
	}

	l.Info("url upload successful", "image_id", response.Result.ID)
	return &response, nil
}

func (s *images) ListImages(ctx context.Context, page, perPage int) (*ListImagesResponse, error) {
	l := s.logger.With("op", "ListImages", "page", page, "per_page", perPage)

	var response ListImagesResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetAuthToken(s.cfg.Get().Cloudflare.ImagesAPIKey).
		SetQueryParams(map[string]string{
			"page":     strconv.Itoa(page),
			"per_page": strconv.Itoa(perPage),
		}).
		SetResult(&response).
		SetError(&response).
		Get(s.cfg.Get().Cloudflare.ImagesURL)

	if err != nil {
		l.Error("request execution failed", "error", err)
		return nil, fmt.Errorf("fetch images: %w", err)
	}

	if resp.IsError() {
		l.Error("cloudflare api error", "status", resp.StatusCode())
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	return &response, nil
}

func (s *images) UploadFromReader(ctx context.Context, reader io.Reader, fileName string, metadata map[string]string, requireSignedURLs bool, shouldBackup bool) (*ImageResponse, error) {
	l := s.logger.With("op", "UploadFromReader", "filename", fileName)

	if reader == nil {
		l.Error("nil reader provided")
		return nil, errors.New("reader is nil")
	}

	var uploadReader = reader
	if shouldBackup && s.storage != nil {
		buf := new(bytes.Buffer)
		uploadReader = io.TeeReader(reader, buf)

		// Improved background backup with structured logging
		defer func() {
			go func(data []byte, name string) {
				bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()

				if err := s.storage.UploadFile(bgCtx, data, name); err != nil {
					l.Error("secondary backup failed", "error", err, "filename", name)
				} else {
					l.Debug("secondary backup successful", "filename", name)
				}
			}(buf.Bytes(), fileName)
		}()
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		l.Error("metadata marshaling failed", "error", err)
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	var response ImageResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetAuthToken(s.cfg.Get().Cloudflare.ImagesAPIKey).
		SetMultipartFields(
			&resty.MultipartField{
				Name:        "file",
				FileName:    fileName,
				ContentType: "application/octet-stream",
				Reader:      uploadReader,
			},
			&resty.MultipartField{
				Name:        "metadata",
				ContentType: "application/json",
				Values:      []string{string(metadataBytes)},
			},
			&resty.MultipartField{
				Name:   "requireSignedURLs",
				Values: []string{strconv.FormatBool(requireSignedURLs)},
			},
		).
		SetResult(&response).
		SetError(&response).
		Post(s.cfg.Get().Cloudflare.ImagesURL)

	if err != nil {
		l.Error("multipart upload failed", "error", err)
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	if resp.IsError() {
		l.Error("cloudflare rejected upload", "status", resp.StatusCode(), "details", response.Errors)
		return nil, fmt.Errorf("api error: %d", resp.StatusCode())
	}

	l.Info("file upload successful", "image_id", response.Result.ID)
	return &response, nil
}
