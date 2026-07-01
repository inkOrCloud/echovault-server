package rest_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/rest"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/storage"
)

func newTestHandler(t *testing.T) *rest.Handler {
	t.Helper()
	s, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}
	return rest.NewHandler(s)
}

func TestUploadAudio(t *testing.T) {
	h := newTestHandler(t)

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
	h := newTestHandler(t)
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
	h := newTestHandler(t)

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
	h := newTestHandler(t)
	req := httptest.NewRequest("POST", "/api/v1/files/upload", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if resp.Code != 400 {
		t.Errorf("status = %d, want 400 for missing params", resp.Code)
	}
}
