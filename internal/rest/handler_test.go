package rest_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/rest"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/metadata"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/storage"
)

// mockSongUpdater 实现 rest.SongUpdater 接口，记录调用。
type mockSongUpdater struct {
	LastSongID string
	LastHash   string
	LastSize   int64
	CallCount  int
}

func (m *mockSongUpdater) UpdateFromScan(songID string, meta *metadata.AudioMetadata, fileSize int64) error {
	m.CallCount++
	m.LastSongID = songID
	m.LastHash = meta.FileHash
	m.LastSize = fileSize
	return nil
}

func newTestHandler(t *testing.T, updater rest.SongUpdater) *rest.Handler {
	t.Helper()
	s, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	return rest.NewHandler(s, updater)
}

// buildTestMP3 构造带 ID3v2 标签 + 可选 APIC 封面的 MP3 文件。
func buildTestMP3(t *testing.T, dir, title, artist string, coverData []byte) string {
	t.Helper()
	id3 := []byte("ID3\x04\x00\x00")
	frames := []byte{}

	// TIT2 frame
	tBytes := append([]byte{0x03}, []byte(title)...)
	frames = append(frames, "TIT2"...)
	fSize := len(tBytes)
	frames = append(frames, byte(fSize>>24), byte(fSize>>16), byte(fSize>>8), byte(fSize))
	frames = append(frames, 0, 0)
	frames = append(frames, tBytes...)

	// TPE1 frame
	aBytes := append([]byte{0x03}, []byte(artist)...)
	frames = append(frames, "TPE1"...)
	fSize = len(aBytes)
	frames = append(frames, byte(fSize>>24), byte(fSize>>16), byte(fSize>>8), byte(fSize))
	frames = append(frames, 0, 0)
	frames = append(frames, aBytes...)

	// APIC frame (cover art)
	if coverData != nil {
		apicBody := []byte{0x03}
		apicBody = append(apicBody, "image/jpeg"...)
		apicBody = append(apicBody, 0)
		apicBody = append(apicBody, 0x03) // cover front
		apicBody = append(apicBody, 0)
		apicBody = append(apicBody, coverData...)
		frames = append(frames, "APIC"...)
		fSize = len(apicBody)
		frames = append(frames, byte(fSize>>24), byte(fSize>>16), byte(fSize>>8), byte(fSize))
		frames = append(frames, 0, 0)
		frames = append(frames, apicBody...)
	}

	tagSize := len(frames)
	id3 = append(id3,
		byte((tagSize>>21)&0x7F),
		byte((tagSize>>14)&0x7F),
		byte((tagSize>>7)&0x7F),
		byte(tagSize&0x7F))
	id3 = append(id3, frames...)

	path := filepath.Join(dir, "test.mp3")
	if err := os.WriteFile(path, id3, 0644); err != nil {
		t.Fatalf("write mp3: %v", err)
	}
	return path
}

func TestUploadAudio_Basic(t *testing.T) {
	h := newTestHandler(t, nil)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "test.mp3")
	part.Write([]byte("fake audio content"))
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload?type=audio&song_id=s1", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if resp.Code != 200 {
		t.Errorf("status = %d, want 200; body=%s", resp.Code, resp.Body.String())
	}

	reader, size, err := h.Storage.GetAudio(context.Background(), "s1")
	if err != nil {
		t.Fatalf("GetAudio() error = %v", err)
	}
	defer reader.Close()
	data, _ := io.ReadAll(reader)
	if size != int64(len(data)) {
		t.Errorf("saved size = %d, want %d", size, len(data))
	}
}

func TestDownloadAudio(t *testing.T) {
	h := newTestHandler(t, nil)
	ctx := context.Background()
	h.Storage.SaveAudio(ctx, "s1", "track.mp3", strings.NewReader("audio data"))

	req := httptest.NewRequest("GET", "/api/v1/files/download/audio/s1", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if resp.Code != 200 {
		t.Errorf("status = %d, want 200", resp.Code)
	}
	if resp.Body.String() != "audio data" {
		t.Errorf("body = %q, want %q", resp.Body.String(), "audio data")
	}
}

func TestUploadCover(t *testing.T) {
	h := newTestHandler(t, nil)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "cover.jpg")
	part.Write([]byte("cover bytes"))
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload?type=cover&song_id=s1", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if resp.Code != 200 {
		t.Errorf("status = %d, want 200", resp.Code)
	}
}

func TestUpload_MissingParams(t *testing.T) {
	h := newTestHandler(t, nil)
	req := httptest.NewRequest("POST", "/api/v1/files/upload", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if resp.Code != 400 {
		t.Errorf("status = %d, want 400 for missing params", resp.Code)
	}
}

func TestUploadAudio_CallsUpdater(t *testing.T) {
	updater := &mockSongUpdater{}
	h := newTestHandler(t, updater)

	mp3Dir := t.TempDir()
	mp3Path := buildTestMP3(t, mp3Dir, "Auto Title", "Auto Artist", nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, _ := writer.CreateFormFile("file", "song.mp3")
	mp3Data, _ := os.ReadFile(mp3Path)
	fw.Write(mp3Data)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload?type=audio&song_id=test-song-001", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if updater.CallCount != 1 {
		t.Errorf("CallCount = %d, want 1", updater.CallCount)
	}
	if updater.LastSongID != "test-song-001" {
		t.Errorf("SongID = %q, want %q", updater.LastSongID, "test-song-001")
	}
}

func TestUploadAudio_WithCover_SavesCover(t *testing.T) {
	s, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	h := rest.NewHandler(s, nil)

	coverData := []byte("fake-cover-image-bytes")
	mp3Dir := t.TempDir()
	mp3Path := buildTestMP3(t, mp3Dir, "Cover Song", "Cover Artist", coverData)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, _ := writer.CreateFormFile("file", "cover-song.mp3")
	mp3Data, _ := os.ReadFile(mp3Path)
	fw.Write(mp3Data)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload?type=audio&song_id=test-cover-song", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	// 验证封面已保存
	coverReader, coverSize, err := s.GetCover(req.Context(), "test-cover-song")
	if err != nil {
		t.Fatalf("GetCover() error = %v (cover was not saved)", err)
	}
	defer coverReader.Close()

	if coverSize != int64(len(coverData)) {
		t.Errorf("cover size = %d, want %d", coverSize, len(coverData))
	}
	readData, _ := io.ReadAll(coverReader)
	if string(readData) != string(coverData) {
		t.Errorf("cover data mismatch")
	}
}

func TestUploadAudio_NoMetadata_StillSucceeds(t *testing.T) {
	h := newTestHandler(t, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, _ := writer.CreateFormFile("file", "noise.bin")
	fw.Write([]byte("not an audio file"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload?type=audio&song_id=test-no-meta", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestUploadCover_DoesNotCallUpdater(t *testing.T) {
	updater := &mockSongUpdater{}
	h := newTestHandler(t, updater)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, _ := writer.CreateFormFile("file", "cover.jpg")
	fw.Write([]byte("fake jpeg bytes"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/files/upload?type=cover&song_id=test-cover", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if updater.CallCount != 0 {
		t.Errorf("CallCount = %d, want 0 for cover upload", updater.CallCount)
	}
}
