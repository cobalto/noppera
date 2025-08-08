package storage

import (
	"context"

	"github.com/cobalto/noppera/internal/config"
)

// supportedImageExtensions lists allowed image file extensions for uploads.
var supportedImageExtensions = map[string]string{
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
}

// Storage defines the interface for file storage operations.
type Storage interface {
	Upload(ctx context.Context, data []byte, ext string) (string, error) // Uploads a file and returns its URL
	Delete(ctx context.Context, url string) error                        // Deletes a file by URL
	Exists(ctx context.Context, url string) (bool, error)                // Checks if a file exists (optional)
	Config() config.Config                                               // Returns the configuration
}

// NewStorage creates a storage implementation based on config.
func NewStorage(cfg config.Config) (Storage, error) {
	if cfg.StorageType == "s3" {
		return NewS3Storage(cfg)
	}
	return NewLocalStorage(cfg)
}
