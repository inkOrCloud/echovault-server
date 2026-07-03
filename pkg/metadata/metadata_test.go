package metadata_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/metadata"
)

const testMIMEType = "application/octet-stream"

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

func id3v23Frame(frameID string, data []byte) []byte {
	hdr := make([]byte, 10)
	copy(hdr[0:4], frameID)
	binary.BigEndian.PutUint32(hdr[4:8], uint32(len(data))) //nolint:gosec // test data, always small
	out := make([]byte, 0, 10+len(data))
	out = append(out, hdr...)
	out = append(out, data...)
	return out
}

func textFrame(value string) []byte {
	b := make([]byte, 0, 1+len(value)+1)
	b = append(b, 0x00)
	b = append(b, []byte(value)...)
	b = append(b, 0x00)
	return b
}

func buildMinimalID3v2(tags map[string]string) []byte {
	var frames []byte
	for frameID, val := range tags {
		fr := id3v23Frame(frameID, textFrame(val))
		frames = append(frames, fr...)
	}

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

func buildMinimalMP3WithCover(coverData []byte) []byte {
	mimeBytes := []byte("image/jpeg")
	apicPayload := make([]byte, 0, 1+len(mimeBytes)+1+1+1+len(coverData))
	apicPayload = append(apicPayload, 0x00)
	apicPayload = append(apicPayload, mimeBytes...)
	apicPayload = append(apicPayload, 0x00)
	apicPayload = append(apicPayload, 0x03)
	apicPayload = append(apicPayload, 0x00)
	apicPayload = append(apicPayload, coverData...)

	frames := make([]byte, 0, 512)
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

func buildMinimalFLAC(comments map[string]string) []byte {
	vendor := "reference libFLAC"
	vendorLen := len(vendor)

	commentData := make([]byte, 0, 256)
	for k, v := range comments {
		entry := fmt.Sprintf("%s=%s", k, v)
		entryLen := len(entry)
		entryBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(entryBuf, uint32(entryLen)) //nolint:gosec // test data, always small
		commentData = append(commentData, entryBuf...)
		commentData = append(commentData, []byte(entry)...)
	}

	vendorLenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(vendorLenBuf, uint32(vendorLen)) //nolint:gosec // test data
	vorbisPayload := make([]byte, 0, 4+len(vendor)+4+len(commentData))
	vorbisPayload = append(vorbisPayload, vendorLenBuf...)
	vorbisPayload = append(vorbisPayload, []byte(vendor)...)

	numComments := make([]byte, 4)
	binary.LittleEndian.PutUint32(numComments, uint32(len(comments))) //nolint:gosec // test data
	vorbisPayload = append(vorbisPayload, numComments...)
	vorbisPayload = append(vorbisPayload, commentData...)

	vorbisBlockLen := len(vorbisPayload)
	vorbisHdr := make([]byte, 4)
	vorbisHdr[0] = 0x80 | 0x04
	vorbisHdr[1] = byte(vorbisBlockLen >> 16) //nolint:gosec // test data, always small
	vorbisHdr[2] = byte(vorbisBlockLen >> 8)  //nolint:gosec // test data, always small
	vorbisHdr[3] = byte(vorbisBlockLen)       //nolint:gosec // test data, always small

	streamInfo := make([]byte, 34)
	binary.BigEndian.PutUint16(streamInfo[0:2], 4096)
	binary.BigEndian.PutUint16(streamInfo[2:4], 4096)

	sr := uint32(44100)
	ch := uint32(1)
	bps := uint32(15)

	streamInfo[18] = byte(sr >> 12) //nolint:gosec // test data, always small
	streamInfo[19] = byte(sr >> 4)  //nolint:gosec // test data, always small
	streamInfo[20] = byte((sr&0x0F)<<4) | byte(ch<<1) | byte(bps>>4)
	streamInfo[21] = byte(bps << 4)

	siHdr := make([]byte, 4)
	siHdr[0] = 0x00
	siHdr[1] = 0x00
	siHdr[2] = 0x00
	siHdr[3] = 34

	flac := make([]byte, 0, 4+4+34+4+vorbisBlockLen)
	flac = append(flac, []byte("fLaC")...)
	flac = append(flac, siHdr...)
	flac = append(flac, streamInfo...)
	flac = append(flac, vorbisHdr...)
	flac = append(flac, vorbisPayload...)
	return flac
}

func writeTestFile(t *testing.T, data []byte, ext string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test"+ext)
	err := os.WriteFile(path, data, 0o600)
	if err != nil {
		t.Fatalf("write test file: %v", err)
	}
	return path
}

func expectedSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestParseFile_MP3_BasicTags(t *testing.T) {
	t.Parallel()
	data := buildMinimalID3v2(map[string]string{
		"TIT2": "Test Title",
		"TPE1": "Test Artist",
		"TALB": "Test Album",
		"TRCK": "1",
		"TPOS": "2",
		"TYER": "2024",
		"TCON": "Rock",
	})
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
	t.Parallel()
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
	t.Parallel()
	coverData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}
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
	t.Parallel()
	data := []byte("this is not an audio file, just some random text data")
	path := writeTestFile(t, data, ".txt")
	m, err := metadata.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() for unknown format should not return error; got %v", err)
	}
	if m.FileName != "test.txt" {
		t.Errorf("FileName = %q, want %q", m.FileName, "test.txt")
	}
	if m.MIMEType != testMIMEType {
		t.Errorf("MIMEType = %q, want %q", m.MIMEType, testMIMEType)
	}
	wantHash := expectedSHA256(data)
	if m.FileHash != wantHash {
		t.Errorf("FileHash = %q, want %q", m.FileHash, wantHash)
	}
	if m.Title != "" {
		t.Errorf("expected empty Title for unknown format, got %q", m.Title)
	}
}

func TestParseFile_NotFound(t *testing.T) {
	t.Parallel()
	_, err := metadata.ParseFile("/nonexistent/path/file.mp3")
	if err == nil {
		t.Fatal("ParseFile() expected error for non-existent file")
	}
}

func TestMIMETypeByExt_AllFormats(t *testing.T) {
	t.Parallel()
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
		{".unknown", testMIMEType},
		{"", testMIMEType},
		{".txt", testMIMEType},
	}
	for _, c := range cases {
		t.Run(c.ext, func(t *testing.T) {
			t.Parallel()
			got := metadata.MIMETypeByExt(c.ext)
			if got != c.want {
				t.Errorf("MIMETypeByExt(%q) = %q, want %q", c.ext, got, c.want)
			}
		})
	}
}

func TestIsAudioFile(t *testing.T) {
	t.Parallel()
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
