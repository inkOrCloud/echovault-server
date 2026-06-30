package storage_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/storage"
)

func TestNewLocalStorage_CreatesDirs(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	if s == nil {
		t.Fatal("NewLocalStorage() = nil, want non-nil")
	}
	// 验证目录被创建
	if _, err := os.Stat(filepath.Join(tmpDir, "songs")); os.IsNotExist(err) {
		t.Error("songs/ directory not created")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "covers")); os.IsNotExist(err) {
		t.Error("covers/ directory not created")
	}
}

func TestSaveAndGetAudio(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	ctx := context.Background()
	songID := "test-song-123"
	content := "fake audio binary content"

	// 保存
	err = s.SaveAudio(ctx, songID, "song.mp3", strings.NewReader(content))
	if err != nil {
		t.Fatalf("SaveAudio() error = %v", err)
	}

	// 读取
	reader, size, err := s.GetAudio(ctx, songID)
	if err != nil {
		t.Fatalf("GetAudio() error = %v", err)
	}
	defer reader.Close()

	if size != int64(len(content)) {
		t.Errorf("GetAudio() size = %d, want %d", size, len(content))
	}

	data, _ := io.ReadAll(reader)
	if string(data) != content {
		t.Errorf("GetAudio() content = %q, want %q", string(data), content)
	}
}

func TestGetAudio_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	_, _, err = s.GetAudio(context.Background(), "nonexistent-song")
	if err == nil {
		t.Fatal("GetAudio() expected error for nonexistent song")
	}
	// 验证错误类型
	var notFound *storage.StorageNotFoundError
	if !asNotFoundError(err, &notFound) {
		t.Errorf("GetAudio() error type = %T, want *StorageNotFoundError", err)
	}
}

func TestSaveAndGetCover(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	ctx := context.Background()
	songID := "test-cover-456"

	err = s.SaveCover(ctx, songID, strings.NewReader("cover image bytes"))
	if err != nil {
		t.Fatalf("SaveCover() error = %v", err)
	}

	reader, size, err := s.GetCover(ctx, songID)
	if err != nil {
		t.Fatalf("GetCover() error = %v", err)
	}
	defer reader.Close()

	if size != 17 {
		t.Errorf("GetCover() size = %d, want 17", size)
	}
}

func TestGetCover_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	_, _, err = s.GetCover(context.Background(), "nonexistent-cover")
	if err == nil {
		t.Fatal("GetCover() expected error for nonexistent cover")
	}
}

func TestDeleteSongFiles(t *testing.T) {
	tmpDir := t.TempDir()
	s, err := storage.NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	ctx := context.Background()
	songID := "test-delete-789"

	// 先保存
	s.SaveAudio(ctx, songID, "track.flac", strings.NewReader("audio"))
	s.SaveCover(ctx, songID, strings.NewReader("cover"))

	// 删除
	if err := s.DeleteSongFiles(ctx, songID); err != nil {
		t.Fatalf("DeleteSongFiles() error = %v", err)
	}

	// 验证已删除
	_, _, err = s.GetAudio(ctx, songID)
	if err == nil {
		t.Error("GetAudio() should fail after DeleteSongFiles")
	}
	_, _, err = s.GetCover(ctx, songID)
	if err == nil {
		t.Error("GetCover() should fail after DeleteSongFiles")
	}
}

// 辅助函数：类型断言
func asNotFoundError(err error, target **storage.StorageNotFoundError) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*storage.StorageNotFoundError)
	if !ok {
		return false
	}
	*target = e
	return true
}
