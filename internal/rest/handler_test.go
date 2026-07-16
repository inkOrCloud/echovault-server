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

func buildTestMP3(t *testing.T, dir, title, artist string, coverData []byte) string {
	t.Helper()
	id3 := []byte("ID3\x04\x00\x00")
	frames := []byte{}

	tBytes := append([]byte{0x03}, []byte(title)...)
	frames = append(frames, "TIT2"...)
	fSize := len(tBytes)
	frames = append(frames, byte(uint32(fSize)>>24), byte(uint32(fSize)>>16), byte(uint32(fSize)>>8), byte(uint32(fSize))) //nolint:gosec // test data, always small
	frames = append(frames, 0, 0)
	frames = append(frames, tBytes...)

	aBytes := append([]byte{0x03}, []byte(artist)...)
	frames = append(frames, "TPE1"...)
	fSize = len(aBytes)
	frames = append(frames, byte(uint32(fSize)>>24), byte(uint32(fSize)>>16), byte(uint32(fSize)>>8), byte(uint32(fSize))) //nolint:gosec // test data, always small
	frames = append(frames, 0, 0)
	frames = append(frames, aBytes...)

	if coverData != nil {
		apicBody := []byte{0x03}
		apicBody = append(apicBody, "image/jpeg"...)
		apicBody = append(apicBody, 0)
		apicBody = append(apicBody, 0x03)
		apicBody = append(apicBody, 0)
		apicBody = append(apicBody, coverData...)
		frames = append(frames, "APIC"...)
		fSize = len(apicBody)
		frames = append(frames, byte(uint32(fSize)>>24), byte(uint32(fSize)>>16), byte(uint32(fSize)>>8), byte(uint32(fSize))) //nolint:gosec // test data, always small
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
	err := os.WriteFile(path, id3, 0o600)
	if err != nil {
		t.Fatalf("write mp3: %v", err)
	}
	return path
}

func TestUploadAudio_Basic(t *testing.T) {
	t.Parallel()
	h := newTestHandler(t, nil)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, wErr := w.CreateFormFile("file", "test.mp3")
	if wErr != nil {
		t.Fatalf("CreateFormFile: %v", wErr)
	}
	_, writeErr := part.Write([]byte("fake audio content"))
	if writeErr != nil {
		t.Fatalf("Write: %v", writeErr)
	}
	closeErr := w.Close()
	if closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/files/upload?type=audio&song_id=s1", &buf)
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
	defer func() { _ = reader.Close() }()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if size != int64(len(data)) {
		t.Errorf("saved size = %d, want %d", size, len(data))
	}
}

func TestDownloadAudio(t *testing.T) {
	t.Parallel()
	h := newTestHandler(t, nil)
	ctx := context.Background()
	err := h.Storage.SaveAudio(ctx, "s1", "track.mp3", strings.NewReader("audio data"))
	if err != nil {
		t.Fatalf("SaveAudio: %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/files/download/audio/s1", nil)
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
	t.Parallel()
	h := newTestHandler(t, nil)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", "cover.jpg")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	_, err = part.Write([]byte("cover bytes"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = w.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/files/upload?type=cover&song_id=s1", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if resp.Code != 200 {
		t.Errorf("status = %d, want 200", resp.Code)
	}
}

func TestUpload_MissingParams(t *testing.T) {
	t.Parallel()
	h := newTestHandler(t, nil)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/files/upload", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	if resp.Code != 400 {
		t.Errorf("status = %d, want 400 for missing params", resp.Code)
	}
}

func TestUploadAudio_CallsUpdater(t *testing.T) {
	t.Parallel()
	updater := &mockSongUpdater{}
	h := newTestHandler(t, updater)

	mp3Dir := t.TempDir()
	mp3Path := buildTestMP3(t, mp3Dir, "Auto Title", "Auto Artist", nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, err := writer.CreateFormFile("file", "song.mp3")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	mp3Data, err := os.ReadFile(mp3Path) //nolint:gosec // test file path from t.TempDir
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	_, writeErr := fw.Write(mp3Data)
	if writeErr != nil {
		t.Fatalf("Write: %v", writeErr)
	}
	closeErr := writer.Close()
	if closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/files/upload?type=audio&song_id=test-song-001", body)
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
	t.Parallel()
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
	fw, err := writer.CreateFormFile("file", "cover-song.mp3")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	mp3Data, err := os.ReadFile(mp3Path) //nolint:gosec // test file path from t.TempDir
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	_, writeErr := fw.Write(mp3Data)
	if writeErr != nil {
		t.Fatalf("Write: %v", writeErr)
	}
	closeErr := writer.Close()
	if closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/files/upload?type=audio&song_id=test-cover-song", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	coverReader, coverSize, err := s.GetCover(req.Context(), "test-cover-song")
	if err != nil {
		t.Fatalf("GetCover() error = %v (cover was not saved)", err)
	}
	defer func() { _ = coverReader.Close() }()

	if coverSize != int64(len(coverData)) {
		t.Errorf("cover size = %d, want %d", coverSize, len(coverData))
	}
	readData, err := io.ReadAll(coverReader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(readData) != string(coverData) {
		t.Errorf("cover data mismatch")
	}
}

func TestUploadAudio_NoMetadata_StillSucceeds(t *testing.T) {
	t.Parallel()
	h := newTestHandler(t, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, err := writer.CreateFormFile("file", "noise.bin")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	_, err = fw.Write([]byte("not an audio file"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/files/upload?type=audio&song_id=test-no-meta", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestUploadCover_DoesNotCallUpdater(t *testing.T) {
	t.Parallel()
	updater := &mockSongUpdater{}
	h := newTestHandler(t, updater)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, err := writer.CreateFormFile("file", "cover.jpg")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	_, err = fw.Write([]byte("fake jpeg bytes"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/files/upload?type=cover&song_id=test-cover", body)
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

func TestUpload_InvalidType(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "t.mp3"); part.Write([]byte("d")); w.Close()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/files/upload?type=invalid&song_id=s1", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 400 { t.Errorf("status=%d", resp.Code) }
}
func TestDownloadAudio_NotFound(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/files/download/audio/x", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 404 { t.Errorf("status=%d", resp.Code) }
}
func TestDownloadAudioWithRange(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil); ctx := context.Background()
	h.Storage.SaveAudio(ctx, "r1", "t.mp3", strings.NewReader("0123456789"))
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/files/download/audio/r1", nil)
	req.Header.Set("Range", "bytes=2-5")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 206 { t.Errorf("status=%d", resp.Code) }
	if resp.Body.String() != "2345" { t.Errorf("body=%q", resp.Body.String()) }
}
func TestDownloadAudio_RangeSuffix(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil); ctx := context.Background()
	h.Storage.SaveAudio(ctx, "r2", "t.mp3", strings.NewReader("0123456789"))
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/files/download/audio/r2", nil)
	req.Header.Set("Range", "bytes=-3")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 206 { t.Errorf("status=%d", resp.Code) }
	if resp.Body.String() != "789" { t.Errorf("body=%q", resp.Body.String()) }
}
func TestDownloadAudio_RangeToEnd(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil); ctx := context.Background()
	h.Storage.SaveAudio(ctx, "r3", "t.mp3", strings.NewReader("0123456789"))
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/files/download/audio/r3", nil)
	req.Header.Set("Range", "bytes=3-")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 206 { t.Errorf("status=%d", resp.Code) }
	if resp.Body.String() != "3456789" { t.Errorf("body=%q", resp.Body.String()) }
}
func TestDownloadCover_NotFound(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/files/download/cover/x", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 404 { t.Errorf("status=%d", resp.Code) }
}
func TestDownloadCover_ETagMatch(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil); ctx := context.Background()
	h.Storage.SaveCover(ctx, "e1", strings.NewReader("data"))
	req1 := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/files/download/cover/e1", nil)
	resp1 := httptest.NewRecorder()
	h.ServeHTTP(resp1, req1)
	etag := resp1.Header().Get("ETag")
	req2 := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/files/download/cover/e1", nil)
	req2.Header.Set("If-None-Match", etag)
	resp2 := httptest.NewRecorder()
	h.ServeHTTP(resp2, req2)
	if resp2.Code != 304 { t.Errorf("status=%d", resp2.Code) }
}
func TestDeleteSong(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil); ctx := context.Background()
	h.Storage.SaveAudio(ctx, "d1", "t.mp3", strings.NewReader("a"))
	req := httptest.NewRequestWithContext(ctx, "DELETE", "/api/v1/files/audio/d1", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 200 { t.Errorf("status=%d", resp.Code) }
}
func TestHealthEndpoint(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/health", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 200 || resp.Body.String() != `{"status":"ok"}` { t.Errorf("body=%q", resp.Body.String()) }
}
func TestParseRange_InvalidFormat(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil); ctx := context.Background()
	h.Storage.SaveAudio(ctx, "inv", "t.mp3", strings.NewReader("0123456789"))
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/files/download/audio/inv", nil)
	req.Header.Set("Range", "invalid")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 416 { t.Errorf("status=%d", resp.Code) }
}
type errStore struct { storage.Storage }
func (e *errStore) GetAudio(context.Context, string) (io.ReadCloser, int64, error) { return nil, 0, io.ErrUnexpectedEOF }
func TestDownloadAudio_StorageError(t *testing.T) { t.Parallel()
	ls, _ := storage.NewLocalStorage(t.TempDir())
	h := rest.NewHandler(&errStore{Storage: ls}, nil)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/files/download/audio/x", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 404 { t.Errorf("status=%d", resp.Code) }
}
func TestUpload_NoFile(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/files/upload?type=audio&song_id=s1", strings.NewReader("x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 400 { t.Errorf("status=%d", resp.Code) }
}

func TestDownloadCover_Success(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil); ctx := context.Background()
	h.Storage.SaveCover(ctx, "c1", strings.NewReader("cover-data"))
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/files/download/cover/c1", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 200 { t.Errorf("status=%d", resp.Code) }
	if resp.Body.String() != "cover-data" { t.Errorf("body=%q", resp.Body.String()) }
	if ct := resp.Header().Get("Content-Type"); ct != "image/jpeg" { t.Errorf("Content-Type=%q", ct) }
}

type nonSeekerReader struct { *strings.Reader }
func (r *nonSeekerReader) Close() error { return nil }
type nonSeekableWrapper struct{ storage.Storage }
func (w *nonSeekableWrapper) GetAudio(ctx context.Context, id string) (io.ReadCloser, int64, error) {
	rc, sz, err := w.Storage.GetAudio(ctx, id)
	if err != nil { return nil, 0, err }
	d, _ := io.ReadAll(rc); rc.Close()
	return &nonSeekerReader{Reader: strings.NewReader(string(d))}, sz, nil
}
func TestServeRange_NonSeekable(t *testing.T) { t.Parallel()
	ls, _ := storage.NewLocalStorage(t.TempDir())
	h := rest.NewHandler(&nonSeekableWrapper{ls}, nil); ctx := context.Background()
	ls.SaveAudio(ctx, "ns1", "t.mp3", strings.NewReader("0123456789"))
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/files/download/audio/ns1", nil)
	req.Header.Set("Range", "bytes=2-5")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 206 { t.Errorf("status=%d", resp.Code) }
	if resp.Body.String() != "2345" { t.Errorf("body=%q", resp.Body.String()) }
}

func TestServeRange_StartBeyondFile(t *testing.T) { t.Parallel()
	h := newTestHandler(t, nil); ctx := context.Background()
	h.Storage.SaveAudio(ctx, "big", "t.mp3", strings.NewReader("0123456789"))
	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/files/download/audio/big", nil)
	req.Header.Set("Range", "bytes=100-110")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != 416 { t.Errorf("status=%d", resp.Code) }
}
