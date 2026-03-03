package archive

import (
	"comics-galore-web/internal/config"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/storage/s3/v2"
)

type Service interface {
	UploadFile(ctx context.Context, data []byte, fileName string) error
	DownloadFile(ctx context.Context, fileName string) (io.ReadCloser, error)
	Ping(ctx context.Context) error
}

type service struct {
	cfg    config.Service
	logger *slog.Logger
}

func NewService(cfg config.Service) Service {
	return &service{
		cfg:    cfg,
		logger: cfg.GetLogger().With("component", "archive_service"),
	}
}

// storage helper ensures hot-reload awareness by fetching fresh config for every call
func (s *service) storage() (*s3.Storage, *s3.Config) {
	s3Cfg := s.cfg.Get().S3Config()
	return s3.New(*s3Cfg), s3Cfg
}

func (s *service) Ping(ctx context.Context) error {
	store, s3Cfg := s.storage()

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := store.Conn().HeadBucket(pingCtx, &s3sdk.HeadBucketInput{
		Bucket: aws.String(s3Cfg.Bucket),
	})

	if err != nil {
		s.logger.Error("S3 health check failed", "bucket", s3Cfg.Bucket, "error", err)
		return fmt.Errorf("s3 storage unreachable: %w", err)
	}

	s.logger.Debug("S3 connection healthy", "bucket", s3Cfg.Bucket)
	return nil
}

func (s *service) UploadFile(ctx context.Context, data []byte, fileName string) error {
	store, _ := s.storage()

	// Adaptive timeout logic
	fileSizeMB := len(data) / (1024 * 1024)
	timeout := time.Duration(fileSizeMB*10+30) * time.Second

	uploadCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	l := s.logger.With("op", "UploadFile", "filename", fileName, "size_mb", fileSizeMB)

	if err := store.SetWithContext(uploadCtx, fileName, data, 0); err != nil {
		l.Error("failed to upload file", "error", err)
		return fmt.Errorf("upload failed: %w", err)
	}

	l.Info("file uploaded successfully")
	return nil
}

func (s *service) DownloadFile(ctx context.Context, fileName string) (io.ReadCloser, error) {
	store, s3Cfg := s.storage()

	l := s.logger.With("op", "DownloadFile", "filename", fileName)

	// We use the underlying AWS SDK client from the storage driver to get a stream.
	// This is better than store.Get() which loads the whole file into RAM.
	output, err := store.Conn().GetObject(ctx, &s3sdk.GetObjectInput{
		Bucket: aws.String(s3Cfg.Bucket),
		Key:    aws.String(fileName),
	})

	if err != nil {
		l.Error("failed to download file", "error", err)
		return nil, fmt.Errorf("download failed: %w", err)
	}

	// Returns io.ReadCloser (the caller must close this)
	return output.Body, nil
}
