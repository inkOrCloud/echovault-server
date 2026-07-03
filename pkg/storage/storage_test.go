package storage_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/storage"
)

func TestNewLocalStorage_CreatesDirs(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	if s == nil {
		t.Fatal("NewLocalStorage() = nil, want non-nil")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "songs")); os.IsNotExist(err) {
		t.Error("songs/ directory not created")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "covers")); os.IsNotExist(err) {
		t.Error("covers/ directory not created")
	}
}

func TestSaveAndGetAudio(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	ctx := context.Background()
	songID := "test-song-123"
	content := "fake audio binary content"

	if err := s.SaveAudio(ctx, songID, "song.mp3", strings.NewReader(content)); err != nil {
		t.Fatalf("SaveAudio() error = %v", err)
	}

	reader, size, err := s.GetAudio(ctx, songID)
	if err != nil {
		t.Fatalf("GetAudio() error = %v", err)
	}
	defer func() {
		if cerr := reader.Close(); cerr != nil {
			t.Errorf("reader.Close() error = %v", cerr)
		}
	}()

	if size != int64(len(content)) {
		t.Errorf("size = %d, want %d", size, len(content))
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != content {
		t.Errorf("content mismatch: got %q, want %q", string(data), content)
	}
}

func TestGetAudio_NotFound(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	_, _, err = s.GetAudio(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("GetAudio() expected error for nonexistent song")
	}
	var notFoundErr *storage.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestSaveAndGetCover(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	ctx := context.Background()
	songID := "cover-song-1"
	coverContent := "fake jpeg bytes"

	if err := s.SaveCover(ctx, songID, strings.NewReader(coverContent)); err != nil {
		t.Fatalf("SaveCover() error = %v", err)
	}

	reader, size, err := s.GetCover(ctx, songID)
	if err != nil {
		t.Fatalf("GetCover() error = %v", err)
	}
	defer func() { _ = reader.Close() }()

	if size != int64(len(coverContent)) {
		t.Errorf("size = %d, want %d", size, len(coverContent))
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != coverContent {
		t.Errorf("content mismatch")
	}
}

func TestGetCover_NotFound(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	_, _, err = s.GetCover(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("GetCover() expected error for nonexistent song")
	}
}

func TestDeleteSongFiles(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	ctx := context.Background()
	songID := "delete-song"

	if err := s.SaveAudio(ctx, songID, "track.mp3", strings.NewReader("audio")); err != nil {
		t.Fatalf("SaveAudio: %v", err)
	}
	if err := s.SaveCover(ctx, songID, strings.NewReader("cover")); err != nil {
		t.Fatalf("SaveCover: %v", err)
	}

	if err := s.DeleteSongFiles(ctx, songID); err != nil {
		t.Fatalf("DeleteSongFiles() error = %v", err)
	}

	_, _, err = s.GetAudio(ctx, songID)
	if err == nil {
		t.Error("GetAudio() should fail after delete")
	}
}
