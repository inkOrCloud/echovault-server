package storage

import (
	"context"
	"fmt"
	"io"
)

type Storage interface {
	SaveAudio(ctx context.Context, songID, filename string, reader io.Reader) error
	GetAudio(ctx context.Context, songID string) (io.ReadCloser, int64, error)
	SaveCover(ctx context.Context, songID string, reader io.Reader) error
	GetCover(ctx context.Context, songID string) (io.ReadCloser, int64, error)
	DeleteSongFiles(ctx context.Context, songID string) error
}

func NewStorage(storageType, storagePath string) (Storage, error) {
	switch storageType {
	case "local":
		return NewLocalStorage(storagePath)
	case "s3":
		return NewS3Storage(storagePath, "")
	default:
		return NewLocalStorage(storagePath)
	}
}

type StorageNotFoundError struct {
	Type string
	ID   string
}

func (e *StorageNotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Type, e.ID)
}
