package metadata_test

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/metadata"
)

// ---------------------------------------------------------------------------
// helpers — build minimal ID3v2.3 binary data
// ---------------------------------------------------------------------------

// synchsafe encodes n into 4 bytes per ID3 synchsafe integer.
func synchsafe(n int) []byte {
	b := make([]byte, 4)
	b[3] = byte(n & 0x7f)
	n >>= 7
	b[2] = byte(n & 0x7f)
	n >>= 7
	b[1] = byte(n & 0x7f)
	n >>= 7
	b[0] = byte(n & 0x7f)
	return b
}

// id3v23Frame builds a single ID3v2.3 frame body.
func id3v23Frame(frameID string, data []byte) []byte {
	hdr := make([]byte, 10)
	copy(hdr[0:4], frameID)
	binary.BigEndian.PutUint32(hdr[4:8], uint32(len(data)))
	// flags = 0x0000
	out := make([]byte, 0, 10+len(data))
	out = append(out, hdr...)
	out = append(out, data...)
	return out
}

// textFrame returns an ISO-8859-1 encoded ID3v2 text frame payload.
func textFrame(value string) []byte {
	b := make([]byte, 0, 1+len(value)+1)
	b = append(b, 0x00) // encoding = ISO-8859-1
	b = append(b, []byte(value)...)
	b = append(b, 0x00) // null terminator
	return b
}

// buildMinimalID3v2 constructs a complete ID3v2.3 header + frames.
func buildMinimalID3v2(tags map[string]string) []byte {
	var frames []byte
	for frameID, val := range tags {
		fr := id3v23Frame(frameID, textFrame(val))
		frames = append(frames, fr...)
	}

	tagSize := len(frames)
	hdr := make([]byte, 10)
	copy(hdr[0:3], "ID3")
	hdr[3] = 0x03 // major
	hdr[4] = 0x00 // minor
	hdr[5] = 0x00 // flags
	copy(hdr[6:10], synchsafe(tagSize))

	out := make([]byte, 0, 10+tagSize)
	out = append(out, hdr...)
	out = append(out, frames...)
	return out
}

// buildMinimalMP3WithCover constructs ID3v2.3 with an APIC frame.
func buildMinimalMP3WithCover(coverData []byte) []byte {
	// APIC frame payload:
	//   encoding (1) + MIME (null-term) + picType (1) + desc (null-term) + data
	apicPayload := []byte{0x00} // ISO-8859-1
	apicPayload = append(apicPayload, []byte("image/jpeg")...)
	apicPayload = append(apicPayload, 0x00) // null-term MIME
	apicPayload = append(apicPayload, 0x03) // front cover
	apicPayload = append(apicPayload, 0x00) // empty description
	apicPayload = append(apicPayload, coverData...)

	var frames []byte
	frames = append(frames, id3v23Frame("TIT2", textFrame("Cover Song"))...)
	frames = append(frames, id3v23Frame("TPE1", textFrame("Cover Artist"))...)
	frames = append(frames, id3v23Frame("APIC", apicPayload)...)

	tagSize := len(frames)
	hdr := make([]byte, 10)
	copy(hdr[0:3], "ID3")
	hdr[3] = 0x03
	hdr[4] = 0x00
	hdr[5] = 0x00
	copy(hdr[6:10], synchsafe(tagSize))

	out := make([]byte, 0, 10+tagSize)
	out = append(out, hdr...)
	out = append(out, frames...)
	return out
}

// buildMinimalFLAC constructs a minimal FLAC file with Vorbis comments.
func buildMinimalFLAC(comments map[string]string) []byte {
	// Vendor string
	vendor := "reference libFLAC"
	vendorLen := len(vendor)

	// Build comment entries
	var commentData []byte
	for k, v := range comments {
		entry := fmt.Sprintf("%s=%s", k, v)
		entryLen := len(entry)
		entryBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(entryBuf, uint32(entryLen))
		commentData = append(commentData, entryBuf...)
		commentData = append(commentData, []byte(entry)...)
	}

	// Vorbis comment payload
	var vorbisPayload []byte
	vendorLenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(vendorLenBuf, uint32(vendorLen))
	vorbisPayload = append(vorbisPayload, vendorLenBuf...)
	vorbisPayload = append(vorbisPayload, []byte(vendor)...)

	numComments := make([]byte, 4)
	binary.LittleEndian.PutUint32(numComments, uint32(len(comments)))
	vorbisPayload = append(vorbisPayload, numComments...)
	vorbisPayload = append(vorbisPayload, commentData...)

	// Vorbis comment block header (last = true, type = 4)
	vorbisBlockLen := len(vorbisPayload)
	vorbisHdr := make([]byte, 4)
	vorbisHdr[0] = 0x80 | 0x04
	vorbisHdr[1] = byte(vorbisBlockLen >> 16)
	vorbisHdr[2] = byte(vorbisBlockLen >> 8)
	vorbisHdr[3] = byte(vorbisBlockLen)

	// STREAMINFO block (34 bytes, mandatory)
	streamInfo := make([]byte, 34)
	binary.BigEndian.PutUint16(streamInfo[0:2], 4096) // min block size
	binary.BigEndian.PutUint16(streamInfo[2:4], 4096) // max block size

	// 8 packed bytes starting at offset 18:
	//   bits 0-19  : sample rate (44100)
	//   bits 20-22 : channels - 1 (1)
	//   bits 23-27 : bits per sample - 1 (15)
	//   bits 28-63 : total samples (0)
	sr := uint32(44100)
	ch := uint32(1)
	bps := uint32(15)

	streamInfo[18] = byte(sr >> 12)
	streamInfo[19] = byte(sr >> 4)
	streamInfo[20] = byte((sr&0x0F)<<4) | byte(ch<<1) | byte(bps>>4)
	streamInfo[21] = byte(bps << 4)

	// STREAMINFO block header (not last)
	siHdr := make([]byte, 4)
	siHdr[0] = 0x00
	siHdr[1] = 0x00
	siHdr[2] = 0x00
	siHdr[3] = 34

	// Build final FLAC
	var flac []byte
	flac = append(flac, []byte("fLaC")...)
	flac = append(flac, siHdr...)
	flac = append(flac, streamInfo...)
	flac = append(flac, vorbisHdr...)
	flac = append(flac, vorbisPayload...)
	return flac
}

// writeTestFile writes data into a temp file and returns the path.
func writeTestFile(t *testing.T, data []byte, ext string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test"+ext)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("writeTestFile: %v", err)
	}
	return path
}

// expectedSHA256 returns the hex SHA256 of data.
func expectedSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestParseFile_MP3_BasicTags(t *testing.T) {
	data := buildMinimalID3v2(map[string]string{
		"TIT2": "Test Title",
		"TPE1": "Test Artist",
		"TALB": "Test Album",
		"TRCK": "1",
		"TPOS": "2",
		"TYER": "2024",
		"TCON": "Rock",
	})
	// Append a bit of silence so the file is slightly larger than just the tag
	data = append(data, make([]byte, 256)...) // dummy audio frame

	path := writeTestFile(t, data, ".mp3")
	m, err := metadata.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	if m.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", m.Title, "Test Title")
	}
	if m.Artist != "Test Artist" {
		t.Errorf("Artist = %q, want %q", m.Artist, "Test Artist")
	}
	if m.Album != "Test Album" {
		t.Errorf("Album = %q, want %q", m.Album, "Test Album")
	}
	if m.TrackNumber != 1 {
		t.Errorf("TrackNumber = %d, want %d", m.TrackNumber, 1)
	}
	if m.DiscNumber != 2 {
		t.Errorf("DiscNumber = %d, want %d", m.DiscNumber, 2)
	}
	if m.Year != 2024 {
		t.Errorf("Year = %d, want %d", m.Year, 2024)
	}
	if m.Genre != "Rock" {
		t.Errorf("Genre = %q, want %q", m.Genre, "Rock")
	}
	if m.MIMEType != "audio/mpeg" {
		t.Errorf("MIMEType = %q, want %q", m.MIMEType, "audio/mpeg")
	}
	wantHash := expectedSHA256(data)
	if m.FileHash != wantHash {
		t.Errorf("FileHash = %q, want %q", m.FileHash, wantHash)
	}
	if m.FileName != "test.mp3" {
		t.Errorf("FileName = %q, want %q", m.FileName, "test.mp3")
	}
}

func TestParseFile_FLAC(t *testing.T) {
	data := buildMinimalFLAC(map[string]string{
		"TITLE":       "Test FLAC Title",
		"ARTIST":      "Test FLAC Artist",
		"ALBUM":       "Test FLAC Album",
		"TRACKNUMBER": "3",
		"DISCNUMBER":  "1",
		"DATE":        "2023",
		"GENRE":       "Classical",
	})
	path := writeTestFile(t, data, ".flac")
	m, err := metadata.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	if m.Title != "Test FLAC Title" {
		t.Errorf("Title = %q, want %q", m.Title, "Test FLAC Title")
	}
	if m.Artist != "Test FLAC Artist" {
		t.Errorf("Artist = %q, want %q", m.Artist, "Test FLAC Artist")
	}
	if m.Album != "Test FLAC Album" {
		t.Errorf("Album = %q, want %q", m.Album, "Test FLAC Album")
	}
	if m.TrackNumber != 3 {
		t.Errorf("TrackNumber = %d, want %d", m.TrackNumber, 3)
	}
	if m.DiscNumber != 1 {
		t.Errorf("DiscNumber = %d, want %d", m.DiscNumber, 1)
	}
	if m.Year != 2023 {
		t.Errorf("Year = %d, want %d", m.Year, 2023)
	}
	if m.Genre != "Classical" {
		t.Errorf("Genre = %q, want %q", m.Genre, "Classical")
	}
	if m.MIMEType != "audio/flac" {
		t.Errorf("MIMEType = %q, want %q", m.MIMEType, "audio/flac")
	}
	wantHash := expectedSHA256(data)
	if m.FileHash != wantHash {
		t.Errorf("FileHash = %q, want %q", m.FileHash, wantHash)
	}
}

func TestParseFile_WithCoverArt(t *testing.T) {
	coverData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01} // minimal JPEG marker
	data := buildMinimalMP3WithCover(coverData)
	data = append(data, make([]byte, 256)...)

	path := writeTestFile(t, data, ".mp3")
	m, err := metadata.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	if m.Picture == nil {
		t.Fatal("Picture = nil, expected non-nil")
	}
	if len(m.Picture.Data) == 0 {
		t.Fatal("Picture.Data is empty")
	}
	if m.Picture.MIMEType != "image/jpeg" {
		t.Errorf("Picture.MIMEType = %q, want %q", m.Picture.MIMEType, "image/jpeg")
	}
	if m.Picture.Ext != "jpg" {
		t.Errorf("Picture.Ext = %q, want %q", m.Picture.Ext, "jpg")
	}
	if m.Title != "Cover Song" {
		t.Errorf("Title = %q, want %q", m.Title, "Cover Song")
	}
	if m.Artist != "Cover Artist" {
		t.Errorf("Artist = %q, want %q", m.Artist, "Cover Artist")
	}
}

func TestParseFile_UnknownFormat_ReturnsBasicInfo(t *testing.T) {
	data := []byte("this is not an audio file, just some random text data")
	path := writeTestFile(t, data, ".txt")
	m, err := metadata.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() for unknown format should not return error; got %v", err)
	}
	if m.FileName != "test.txt" {
		t.Errorf("FileName = %q, want %q", m.FileName, "test.txt")
	}
	if m.MIMEType != "application/octet-stream" {
		t.Errorf("MIMEType = %q, want %q", m.MIMEType, "application/octet-stream")
	}
	wantHash := expectedSHA256(data)
	if m.FileHash != wantHash {
		t.Errorf("FileHash = %q, want %q", m.FileHash, wantHash)
	}
	// Title/Artist should be empty (not crash)
	if m.Title != "" {
		t.Errorf("expected empty Title for unknown format, got %q", m.Title)
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := metadata.ParseFile("/nonexistent/path/file.mp3")
	if err == nil {
		t.Fatal("ParseFile() expected error for non-existent file")
	}
}

func TestMIMETypeByExt_AllFormats(t *testing.T) {
	cases := []struct {
		ext  string
		want string
	}{
		{".mp3", "audio/mpeg"},
		{".flac", "audio/flac"},
		{".ogg", "audio/ogg"},
		{".m4a", "audio/mp4"},
		{".mp4", "audio/mp4"},
		{".wav", "audio/wav"},
		{".aac", "audio/aac"},
		{".wma", "audio/x-ms-wma"},
		{".opus", "audio/opus"},
		{".unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
		{".txt", "application/octet-stream"},
	}
	for _, c := range cases {
		t.Run(c.ext, func(t *testing.T) {
			got := metadata.MIMETypeByExt(c.ext)
			if got != c.want {
				t.Errorf("MIMETypeByExt(%q) = %q, want %q", c.ext, got, c.want)
			}
		})
	}
}

func TestIsAudioFile(t *testing.T) {
	audioExts := []string{".mp3", ".flac", ".ogg", ".m4a", ".mp4", ".wav", ".aac", ".wma", ".opus"}
	nonAudioExts := []string{".txt", ".go", ".jpg", ".pdf", ".zip", ""}

	for _, ext := range audioExts {
		if !metadata.IsAudioFile("file" + ext) {
			t.Errorf("IsAudioFile(%q) = false, want true", "file"+ext)
		}
	}
	for _, ext := range nonAudioExts {
		if metadata.IsAudioFile("file" + ext) {
			t.Errorf("IsAudioFile(%q) = true, want false", "file"+ext)
		}
	}
}
