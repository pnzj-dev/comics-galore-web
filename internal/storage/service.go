package storage

import (
	"bytes"
	"comics-galore-web/internal/config"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"log/slog"
	"time"
)

type service struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	logger        *slog.Logger
}

type Service interface {
	GetPresignedUploadURL(ctx context.Context, key string, contentType string) (string, error)
	UploadFile(ctx context.Context, data []byte, fileName string) error
	DownloadFile(ctx context.Context, fileName string) (io.ReadCloser, error)
	Ping(ctx context.Context) error
}

func NewService(cfg config.Service) Service {
	ctx := context.Background()
	s3Params := cfg.Get().S3Config() // Your existing function

	// 1. Map credentials and region to AWS Config
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithRegion(s3Params.Region),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			s3Params.Credentials.AccessKey,
			s3Params.Credentials.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		// In a constructor, you might want to panic or return (Service, error)
		panic(fmt.Sprintf("unable to load AWS SDK config: %v", err))
	}

	// 2. Initialize the Official S3 Client with your Endpoint
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if s3Params.Endpoint != "" {
			o.BaseEndpoint = aws.String(s3Params.Endpoint)
		}
		// Most non-AWS S3 providers (R2, Minio) require PathStyle
		o.UsePathStyle = true
	})

	return &service{
		logger:        cfg.GetLogger().With("component", "archive_service"),
		client:        s3Client,
		presignClient: s3.NewPresignClient(s3Client),
		bucket:        s3Params.Bucket,
	}
}

func (s *service) GetPresignedUploadURL(ctx context.Context, key string, contentType string) (string, error) {
	l := s.logger.With(
		"op", "GetPresignedUploadURL",
		"key", key,
		"content_type", contentType,
	)

	// Define the expiration (15 minutes)
	lifetimeSecs := int64(900)

	request, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(lifetimeSecs) * time.Second
	})

	if err != nil {
		l.Error("failed to generate presigned url", "error", err)
		return "", fmt.Errorf("presign failed: %w", err)
	}

	l.Debug("presigned url generated successfully", "expires_in", "15m")

	return request.URL, nil
}

func (s *service) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Using the official client directly
	_, err := s.client.HeadBucket(pingCtx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})

	if err != nil {
		s.logger.Error("S3 health check failed", "bucket", s.bucket, "error", err)
		return fmt.Errorf("s3 storage unreachable: %w", err)
	}

	s.logger.Debug("S3 connection healthy", "bucket", s.bucket)
	return nil
}

func (s *service) UploadFile(ctx context.Context, data []byte, fileName string) error {
	// 1. Improved timeout math
	timeout := 30 * time.Second
	if len(data) > 0 {
		// Add 10 seconds per MB, using float math for accuracy
		extra := time.Duration(float64(len(data))/(1024*1024)*10) * time.Second
		timeout += extra
	}

	uploadCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	l := s.logger.With("op", "UploadFile", "filename", fileName, "size_bytes", len(data))

	// 2. Use official client PutObject (standard for S3)
	_, err := s.client.PutObject(uploadCtx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileName),
		Body:   bytes.NewReader(data),
	})

	if err != nil {
		l.Error("failed to upload file", "error", err)
		return fmt.Errorf("upload failed: %w", err)
	}

	l.Info("file uploaded successfully")
	return nil
}

func (s *service) DownloadFile(ctx context.Context, fileName string) (io.ReadCloser, error) {
	l := s.logger.With("op", "DownloadFile", "filename", fileName)

	// 3. Use official client GetObject
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileName),
	})

	if err != nil {
		l.Error("failed to download file", "error", err)
		return nil, fmt.Errorf("download failed: %w", err)
	}

	return output.Body, nil
}
