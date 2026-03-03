package picture

import (
	"bytes"
	"comics-galore-web/internal/config"
	"context"
	"fmt"
	_ "golang.org/x/image/webp" // RegisterRoutes WebP decoder
	"image"
	"image/jpeg"
	_ "image/png" // RegisterRoutes PNG decoder
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/storage/s3/v2"
	"golang.org/x/image/draw"
)

type service struct {
	cfg             config.Service
	logger          *slog.Logger
	cachePrefix     string
	cacheExpiration time.Duration
	dryRun          bool

	// Graceful shutdown tools
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

type Service interface {
	ProcessAndCacheFromS3(originalKey string, targetWidth, jpegQuality int) (io.ReadCloser, error)
	Shutdown(timeout time.Duration) error
}

func NewService(cfg config.Service, dryRun bool) Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &service{
		cfg:             cfg,
		logger:          cfg.GetLogger().With("component", "picture_service"),
		cachePrefix:     "processed/",
		cacheExpiration: 24 * time.Hour,
		dryRun:          dryRun,
		ctx:             ctx,
		cancel:          cancel,
	}
}

func (s *service) storage() *s3.Storage {
	return s3.New(*s.cfg.Get().S3Config())
}

func (s *service) getCacheKey(originalKey string, width, quality int) string {
	return fmt.Sprintf("%s%s_w%d_q%d.jpg", s.cachePrefix, originalKey, width, quality)
}

func (s *service) ProcessAndCacheFromS3(originalKey string, targetWidth, jpegQuality int) (io.ReadCloser, error) {
	// 1. Check if service is shutting down
	select {
	case <-s.ctx.Done():
		return nil, fmt.Errorf("service is shutting down")
	default:
	}

	// Sanitization
	if targetWidth <= 0 || targetWidth > 4096 {
		targetWidth = 1024
	}
	if jpegQuality < 1 || jpegQuality > 100 {
		jpegQuality = 80
	}

	l := s.logger.With(
		"op", "ProcessAndCacheFromS3",
		"original_key", originalKey,
		"target_width", targetWidth,
	)

	cacheKey := s.getCacheKey(originalKey, targetWidth, jpegQuality)
	store := s.storage()

	// 2. Cache Hit Logic
	cachedData, err := store.Get(cacheKey)
	if err == nil && len(cachedData) > 0 {
		l.Debug("image cache hit")
		return io.NopCloser(bytes.NewReader(cachedData)), nil
	}

	// 3. Fetch and Validate
	start := time.Now()
	originalData, err := store.Get(originalKey)
	if err != nil {
		l.Error("failed to get original from s3", "error", err)
		return nil, fmt.Errorf("s3 source error: %w", err)
	}

	contentType := http.DetectContentType(originalData)
	if !strings.HasPrefix(contentType, "image/") {
		l.Warn("skipped processing: unsupported file type", "detected_type", contentType)
		return nil, fmt.Errorf("invalid file type: %s", contentType)
	}

	// 4. Decode and Resize
	srcImg, format, err := image.Decode(bytes.NewReader(originalData))
	if err != nil {
		l.Error("image decode failed", "error", err)
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	bounds := srcImg.Bounds()
	if bounds.Dx() <= targetWidth {
		targetWidth = bounds.Dx()
	}
	ratio := float64(bounds.Dy()) / float64(bounds.Dx())
	targetHeight := int(float64(targetWidth) * ratio)

	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.BiLinear.Scale(dst, dst.Bounds(), srcImg, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		l.Error("jpeg encode failed", "error", err)
		return nil, fmt.Errorf("encode failed: %w", err)
	}

	processedBytes := buf.Bytes()

	// 5. Background Caching with Shutdown Awareness
	if s.dryRun {
		l.Info("dry run: skipping cache upload")
	} else {
		s.wg.Add(1)
		go func(st *s3.Storage, key string, data []byte) {
			defer s.wg.Done()
			if err := st.Set(key, data, s.cacheExpiration); err != nil {
				l.Warn("failed to save processed image to cache", "error", err)
			}
		}(store, cacheKey, processedBytes)
	}

	l.Info("image processed",
		"duration_ms", time.Since(start).Milliseconds(),
		"source_format", format,
		"size_kb", len(processedBytes)/1024,
	)

	return io.NopCloser(bytes.NewReader(processedBytes)), nil
}

// Shutdown waits for background uploads to finish or times out
func (s *service) Shutdown(timeout time.Duration) error {
	s.logger.Info("shutting down picture service...")

	// 1. Stop accepting new processing requests
	s.cancel()

	// 2. Wait for background tasks with a timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("picture service shut down gracefully")
		return nil
	case <-time.After(timeout):
		s.logger.Warn("picture service shutdown timed out; some uploads may have been lost")
		return fmt.Errorf("shutdown timed out")
	}
}
