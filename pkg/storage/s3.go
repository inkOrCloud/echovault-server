package storage

import (
	"context"
	"errors"
	"io"
)

// ErrNotImplemented indicates the S3 storage backend is not yet implemented.
var ErrNotImplemented = errors.New("S3 storage not yet implemented")

// S3Storage implements Storage using Amazon S3.
type S3Storage struct{}

// NewS3Storage creates a new S3Storage.
func NewS3Storage(_ string, _ string) (*S3Storage, error) {
	return nil, ErrNotImplemented
}

// SaveAudio stores an audio file in S3.
func (s *S3Storage) SaveAudio(_ context.Context, _, _ string, _ io.Reader) error {
	return ErrNotImplemented
}

// GetAudio returns an audio file from S3.
func (s *S3Storage) GetAudio(_ context.Context, _ string) (io.ReadCloser, int64, error) {
	return nil, 0, ErrNotImplemented
}

// SaveCover stores a cover image in S3.
func (s *S3Storage) SaveCover(_ context.Context, _ string, _ io.Reader) error {
	return ErrNotImplemented
}

// GetCover returns a cover image from S3.
func (s *S3Storage) GetCover(_ context.Context, _ string) (io.ReadCloser, int64, error) {
	return nil, 0, ErrNotImplemented
}

// DeleteSongFiles removes all files for a song from S3.
func (s *S3Storage) DeleteSongFiles(_ context.Context, _ string) error {
	return ErrNotImplemented
}
