package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/cobalto/noppera/internal/config"
)

// S3Storage implements file storage using AWS S3.
type S3Storage struct {
	cfg    config.Config
	client *s3.Client
}

// NewS3Storage creates a new S3Storage instance with AWS S3 configuration.
func NewS3Storage(cfg config.Config) (Storage, error) {
	if cfg.S3Bucket == "" || cfg.S3Region == "" || cfg.S3AccessKeyID == "" || cfg.S3SecretAccessKey == "" {
		return nil, fmt.Errorf("missing required S3 configuration: bucket, region, access key, or secret key")
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &S3Storage{
		cfg:    cfg,
		client: s3.NewFromConfig(awsCfg),
	}, nil
}

// Upload saves an image file to S3 and returns its URL.
func (s *S3Storage) Upload(ctx context.Context, data []byte, ext string) (string, error) {
	// Validate file extension
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	mimeType, ok := supportedImageExtensions[ext]
	if !ok {
		return "", fmt.Errorf("unsupported file extension: %s (allowed: %v)", ext, supportedImageExtensions)
	}

	// Validate file size
	if len(data) > s.cfg.DefaultMaxImageSize {
		return "", fmt.Errorf("file size %d bytes exceeds maximum allowed %d bytes", len(data), s.cfg.DefaultMaxImageSize)
	}

	// Set timeout for S3 operation
	ctx, cancel := context.WithTimeout(ctx, s.cfg.StorageTimeout)
	defer cancel()

	// Generate unique filename
	filename := fmt.Sprintf("%d.%s", time.Now().UnixNano(), ext)
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.S3Bucket),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload %s to S3 bucket %s: %w", filename, s.cfg.S3Bucket, err)
	}

	// Construct URL
	url := fmt.Sprintf("%s/%s", s.cfg.S3BaseURL, filename)
	return url, nil
}

// Delete removes a file from S3.
func (s *S3Storage) Delete(ctx context.Context, url string) error {
	// Set timeout for S3 operation
	ctx, cancel := context.WithTimeout(ctx, s.cfg.StorageTimeout)
	defer cancel()

	filename := filepath.Base(url)
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.S3Bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		return fmt.Errorf("failed to delete %s from S3 bucket %s: %w", filename, s.cfg.S3Bucket, err)
	}
	return nil
}

// Exists checks if a file exists in S3.
func (s *S3Storage) Exists(ctx context.Context, url string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.StorageTimeout)
	defer cancel()

	filename := filepath.Base(url)
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.cfg.S3Bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		var smithyErr smithy.APIError
		if errors.As(err, &smithyErr) && (smithyErr.ErrorCode() == "NotFound" || smithyErr.ErrorCode() == "404") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence of %s in S3 bucket %s: %w", filename, s.cfg.S3Bucket, err)
	}
	return true, nil
}

// Config returns the storage configuration.
func (s *S3Storage) Config() config.Config {
	return s.cfg
}
