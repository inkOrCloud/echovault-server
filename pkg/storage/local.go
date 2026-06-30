package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(filepath.Join(basePath, "songs"), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(basePath, "covers"), 0755); err != nil {
		return nil, err
	}
	return &LocalStorage{basePath: basePath}, nil
}

func (s *LocalStorage) SaveAudio(_ context.Context, songID, filename string, reader io.Reader) error {
	dir := filepath.Join(s.basePath, "songs", songID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	dst, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, reader)
	return err
}

func (s *LocalStorage) GetAudio(_ context.Context, songID string) (io.ReadCloser, int64, error) {
	dir := filepath.Join(s.basePath, "songs", songID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, &StorageNotFoundError{Type: "audio", ID: songID}
	}
	if len(entries) == 0 {
		return nil, 0, &StorageNotFoundError{Type: "audio", ID: songID}
	}
	fpath := filepath.Join(dir, entries[0].Name())
	fi, err := os.Stat(fpath)
	if err != nil {
		return nil, 0, &StorageNotFoundError{Type: "audio", ID: songID}
	}
	f, err := os.Open(fpath)
	if err != nil {
		return nil, 0, err
	}
	return f, fi.Size(), nil
}

func (s *LocalStorage) SaveCover(_ context.Context, songID string, reader io.Reader) error {
	dst, err := os.Create(filepath.Join(s.basePath, "covers", songID+".jpg"))
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, reader)
	return err
}

func (s *LocalStorage) GetCover(_ context.Context, songID string) (io.ReadCloser, int64, error) {
	fpath := filepath.Join(s.basePath, "covers", songID+".jpg")
	fi, err := os.Stat(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, &StorageNotFoundError{Type: "cover", ID: songID}
		}
		return nil, 0, err
	}
	f, err := os.Open(fpath)
	if err != nil {
		return nil, 0, err
	}
	return f, fi.Size(), nil
}

func (s *LocalStorage) DeleteSongFiles(_ context.Context, songID string) error {
	os.RemoveAll(filepath.Join(s.basePath, "songs", songID))
	os.Remove(filepath.Join(s.basePath, "covers", songID+".jpg"))
	return nil
}
