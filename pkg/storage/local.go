// Package storage provides audio file storage backends.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const dirPermission = 0o750

const (
	storageTypeAudio = "audio"
	songsSubdir      = "songs"
	coversSubdir     = "covers"
)

// LocalStorage implements Storage using the local filesystem.
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new LocalStorage.
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(filepath.Join(basePath, songsSubdir), dirPermission); err != nil {
		return nil, fmt.Errorf("create songs dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(basePath, coversSubdir), dirPermission); err != nil {
		return nil, fmt.Errorf("create covers dir: %w", err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

// SaveAudio stores an audio file.
func (s *LocalStorage) SaveAudio(_ context.Context, songID, filename string, reader io.Reader) error {
	dir := filepath.Join(s.basePath, songsSubdir, songID)
	if err := os.MkdirAll(dir, dirPermission); err != nil {
		return fmt.Errorf("create song dir: %w", err)
	}
	dst, err := os.Create(filepath.Join(dir, filename)) //nolint:gosec // songID and filename are validated
	if err != nil {
		return fmt.Errorf("create audio file: %w", err)
	}
	defer func() { _ = dst.Close() }()
	if _, err := io.Copy(dst, reader); err != nil {
		return fmt.Errorf("copy audio data: %w", err)
	}
	return nil
}

// GetAudio returns an audio file reader.
func (s *LocalStorage) GetAudio(_ context.Context, songID string) (io.ReadCloser, int64, error) {
	dir := filepath.Join(s.basePath, songsSubdir, songID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, &NotFoundError{Type: storageTypeAudio, ID: songID}
	}
	if len(entries) == 0 {
		return nil, 0, &NotFoundError{Type: storageTypeAudio, ID: songID}
	}
	fpath := filepath.Join(dir, entries[0].Name())
	fi, err := os.Stat(fpath)
	if err != nil {
		return nil, 0, &NotFoundError{Type: storageTypeAudio, ID: songID}
	}
	f, err := os.Open(fpath) //nolint:gosec // path is from validated internal components
	if err != nil {
		return nil, 0, fmt.Errorf("open audio file: %w", err)
	}
	return f, fi.Size(), nil
}

// SaveCover stores a cover image.
func (s *LocalStorage) SaveCover(_ context.Context, songID string, reader io.Reader) error {
	dst, err := os.Create(filepath.Join(s.basePath, coversSubdir, songID+".jpg")) //nolint:gosec // songID is validated
	if err != nil {
		return fmt.Errorf("create cover file: %w", err)
	}
	defer func() { _ = dst.Close() }()
	if _, err := io.Copy(dst, reader); err != nil {
		return fmt.Errorf("copy cover data: %w", err)
	}
	return nil
}

// GetCover returns a cover image reader.
func (s *LocalStorage) GetCover(_ context.Context, songID string) (io.ReadCloser, int64, error) {
	fpath := filepath.Join(s.basePath, coversSubdir, songID+".jpg")
	fi, err := os.Stat(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, &NotFoundError{Type: "cover", ID: songID}
		}
		return nil, 0, fmt.Errorf("stat cover file: %w", err)
	}
	f, err := os.Open(fpath) //nolint:gosec // path is from validated internal components
	if err != nil {
		return nil, 0, fmt.Errorf("open cover file: %w", err)
	}
	return f, fi.Size(), nil
}

// DeleteSongFiles removes all files for a song.
func (s *LocalStorage) DeleteSongFiles(_ context.Context, songID string) error {
	if err := os.RemoveAll(filepath.Join(s.basePath, songsSubdir, songID)); err != nil {
		return fmt.Errorf("remove song files: %w", err)
	}
	if err := os.Remove(filepath.Join(s.basePath, coversSubdir, songID+".jpg")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove cover file: %w", err)
	}
	return nil
}
