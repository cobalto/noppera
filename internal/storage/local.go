package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cobalto/noppera/internal/config"
)

// LocalStorage implements file storage on the local filesystem.
type LocalStorage struct {
	cfg config.Config
}

// NewLocalStorage creates a new LocalStorage instance.
func NewLocalStorage(cfg config.Config) (Storage, error) {
	if cfg.UploadDir == "" {
		return nil, fmt.Errorf("missing required upload directory configuration")
	}
	return &LocalStorage{cfg: cfg}, nil
}

// Upload saves an image file to the local filesystem and returns its URL.
func (s *LocalStorage) Upload(ctx context.Context, data []byte, ext string) (string, error) {
	// Validate file extension
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	if _, ok := supportedImageExtensions[ext]; !ok {
		return "", fmt.Errorf("unsupported file extension: %s (allowed: %v)", ext, supportedImageExtensions)
	}

	// Validate file size
	if len(data) > s.cfg.DefaultMaxImageSize {
		return "", fmt.Errorf("file size %d bytes exceeds maximum allowed %d bytes", len(data), s.cfg.DefaultMaxImageSize)
	}

	// Set timeout for file operation
	ctx, cancel := context.WithTimeout(ctx, s.cfg.StorageTimeout)
	defer cancel()

	// Create uploads directory
	uploadsDir := s.cfg.UploadDir
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory %s: %w", uploadsDir, err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d.%s", time.Now().UnixNano(), ext)
	path := filepath.Join(uploadsDir, filename)

	// Write file with context cancellation
	errChan := make(chan error, 1)
	go func() {
		errChan <- os.WriteFile(path, data, 0644)
	}()
	select {
	case err := <-errChan:
		if err != nil {
			return "", fmt.Errorf("failed to write file %s: %w", path, err)
		}
	case <-ctx.Done():
		return "", fmt.Errorf("upload to %s cancelled: %w", path, ctx.Err())
	}

	// Construct URL using configurable prefix
	url := fmt.Sprintf("http://%s:%s%s/%s", s.cfg.APIHost, s.cfg.APIPort, s.cfg.UploadURLPrefix, filename)
	return url, nil
}

// Delete removes a file from the local filesystem.
func (s *LocalStorage) Delete(ctx context.Context, url string) error {
	// Set timeout for file operation
	ctx, cancel := context.WithTimeout(ctx, s.cfg.StorageTimeout)
	defer cancel()

	filename := filepath.Base(url)
	path := filepath.Join(s.cfg.UploadDir, filename)

	// Delete file with context cancellation
	errChan := make(chan error, 1)
	go func() {
		errChan <- os.Remove(path)
	}()
	select {
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("failed to delete file %s: %w", path, err)
		}
	case <-ctx.Done():
		return fmt.Errorf("deletion of %s cancelled: %w", path, ctx.Err())
	}
	return nil
}

// Exists checks if a file exists in the local filesystem.
func (s *LocalStorage) Exists(ctx context.Context, url string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.StorageTimeout)
	defer cancel()

	filename := filepath.Base(url)
	path := filepath.Join(s.cfg.UploadDir, filename)

	errChan := make(chan error, 1)
	go func() {
		_, err := os.Stat(path)
		errChan <- err
	}()
	select {
	case err := <-errChan:
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, fmt.Errorf("failed to check existence of %s: %w", path, err)
		}
		return true, nil
	case <-ctx.Done():
		return false, fmt.Errorf("existence check for %s cancelled: %w", path, ctx.Err())
	}
}

// Config returns the storage configuration.
func (s *LocalStorage) Config() config.Config {
	return s.cfg
}
