package storage

import (
	"context"
	"fmt"
	"io"
)

// Storage defines the interface for audio file storage backends.
type Storage interface {
	SaveAudio(ctx context.Context, songID, filename string, reader io.Reader) error
	GetAudio(ctx context.Context, songID string) (io.ReadCloser, int64, error)
	SaveCover(ctx context.Context, songID string, reader io.Reader) error
	GetCover(ctx context.Context, songID string) (io.ReadCloser, int64, error)
	DeleteSongFiles(ctx context.Context, songID string) error
}

// NewStorage creates a new Storage backend based on the storage type.
func NewStorage(storageType, storagePath string) (Storage, error) {
	switch storageType {
	case "local":
		s, err := NewLocalStorage(storagePath)
		if err != nil {
			return nil, fmt.Errorf("new local storage: %w", err)
		}
		return s, nil
	case "s3":
		s, err := NewS3Storage(storagePath, "")
		if err != nil {
			return nil, fmt.Errorf("new s3 storage: %w", err)
		}
		return s, nil
	default:
		s, err := NewLocalStorage(storagePath)
		if err != nil {
			return nil, fmt.Errorf("new local storage: %w", err)
		}
		return s, nil
	}
}

// NotFoundError is returned when a requested file is not found.
type NotFoundError struct {
	Type string
	ID   string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Type, e.ID)
}
