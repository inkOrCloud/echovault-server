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
	_, err = os.Stat(filepath.Join(tmpDir, "songs"))
	if os.IsNotExist(err) {
		t.Error("songs/ directory not created")
	}
	_, err = os.Stat(filepath.Join(tmpDir, "covers"))
	if os.IsNotExist(err) {
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

	err = s.SaveAudio(ctx, songID, "song.mp3", strings.NewReader(content))
	if err != nil {
		t.Fatalf("SaveAudio() error = %v", err)
	}

	reader, size, err := s.GetAudio(ctx, songID)
	if err != nil {
		t.Fatalf("GetAudio() error = %v", err)
	}
	defer func() {
		cerr := reader.Close()
		if cerr != nil {
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

	err = s.SaveCover(ctx, songID, strings.NewReader(coverContent))
	if err != nil {
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

	err = s.SaveAudio(ctx, songID, "track.mp3", strings.NewReader("audio"))
	if err != nil {
		t.Fatalf("SaveAudio: %v", err)
	}
	err = s.SaveCover(ctx, songID, strings.NewReader("cover"))
	if err != nil {
		t.Fatalf("SaveCover: %v", err)
	}

	err = s.DeleteSongFiles(ctx, songID)
	if err != nil {
		t.Fatalf("DeleteSongFiles() error = %v", err)
	}

	_, _, err = s.GetAudio(ctx, songID)
	if err == nil {
		t.Error("GetAudio() should fail after delete")
	}
}

func TestNewStorage_Local(t *testing.T) { t.Parallel()
	s, err := storage.NewStorage("local", t.TempDir())
	if err != nil { t.Fatalf("err=%v", err) }
	err = s.SaveAudio(context.Background(), "t", "f.mp3", strings.NewReader("d"))
	if err != nil { t.Fatalf("SaveAudio err=%v", err) }
}
func TestNewStorage_Default(t *testing.T) { t.Parallel()
	s, err := storage.NewStorage("unknown", t.TempDir())
	if err != nil { t.Fatalf("err=%v", err) }
	if s == nil { t.Fatal("nil") }
}
func TestNewStorage_S3(t *testing.T) { t.Parallel()
	_, err := storage.NewStorage("s3", "")
	if !errors.Is(err, storage.ErrNotImplemented) { t.Errorf("err=%v", err) }
}
func TestNotFoundError(t *testing.T) { t.Parallel()
	e := &storage.NotFoundError{Type: "audio", ID: "s1"}
	if e.Error() != "audio not found: s1" { t.Errorf("got=%q", e.Error()) }
}
func TestS3_SaveAudio(t *testing.T) { t.Parallel()
	err := (&storage.S3Storage{}).SaveAudio(context.Background(), "id", "n", strings.NewReader("d"))
	if !errors.Is(err, storage.ErrNotImplemented) { t.Errorf("err=%v", err) }
}
func TestS3_GetAudio(t *testing.T) { t.Parallel()
	r, sz, err := (&storage.S3Storage{}).GetAudio(context.Background(), "id")
	if !errors.Is(err, storage.ErrNotImplemented) { t.Errorf("err=%v", err) }
	if r != nil || sz != 0 { t.Error("expected nil/0") }
}
func TestS3_SaveCover(t *testing.T) { t.Parallel()
	err := (&storage.S3Storage{}).SaveCover(context.Background(), "id", strings.NewReader("d"))
	if !errors.Is(err, storage.ErrNotImplemented) { t.Errorf("err=%v", err) }
}
func TestS3_GetCover(t *testing.T) { t.Parallel()
	r, sz, err := (&storage.S3Storage{}).GetCover(context.Background(), "id")
	if !errors.Is(err, storage.ErrNotImplemented) { t.Errorf("err=%v", err) }
	if r != nil || sz != 0 { t.Error("expected nil/0") }
}
func TestS3_Delete(t *testing.T) { t.Parallel()
	err := (&storage.S3Storage{}).DeleteSongFiles(context.Background(), "id")
	if !errors.Is(err, storage.ErrNotImplemented) { t.Errorf("err=%v", err) }
}
func TestDeleteSongFiles_AudioOnly(t *testing.T) { t.Parallel()
	s, _ := storage.NewLocalStorage(t.TempDir()); ctx := context.Background()
	s.SaveAudio(ctx, "d1", "t.mp3", strings.NewReader("a"))
	_, _, err := s.GetAudio(ctx, "d1")
	if err != nil { t.Fatalf("before: %v", err) }
	s.DeleteSongFiles(ctx, "d1")
	_, _, err = s.GetAudio(ctx, "d1")
	if err == nil { t.Error("should error after delete") }
}
func TestGetAudio_EmptyDir(t *testing.T) { t.Parallel()
	s, _ := storage.NewLocalStorage(t.TempDir()); ctx := context.Background()
	_, _, err := s.GetAudio(ctx, "x")
	if err == nil { t.Error("should error") }
}

func TestNewS3Storage(t *testing.T) { t.Parallel()
	s, err := storage.NewS3Storage("", "")
	if s != nil || err == nil { t.Error("expected nil + error") }
}
