package storage

import (
	"context"
	"fmt"
	"io"
)

type S3Storage struct{}

func NewS3Storage(bucket, region string) (*S3Storage, error) {
	return nil, fmt.Errorf("S3 storage not yet implemented")
}

func (s *S3Storage) SaveAudio(_ context.Context, _, _ string, _ io.Reader) error {
	return fmt.Errorf("S3 storage not yet implemented")
}
func (s *S3Storage) GetAudio(_ context.Context, _ string) (io.ReadCloser, int64, error) {
	return nil, 0, fmt.Errorf("S3 storage not yet implemented")
}
func (s *S3Storage) SaveCover(_ context.Context, _ string, _ io.Reader) error {
	return fmt.Errorf("S3 storage not yet implemented")
}
func (s *S3Storage) GetCover(_ context.Context, _ string) (io.ReadCloser, int64, error) {
	return nil, 0, fmt.Errorf("S3 storage not yet implemented")
}
func (s *S3Storage) DeleteSongFiles(_ context.Context, _ string) error {
	return fmt.Errorf("S3 storage not yet implemented")
}
