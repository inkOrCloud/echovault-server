// Package metadata provides audio file metadata parsing.
package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
)

// Picture represents embedded cover art.
type Picture struct {
	Data     []byte
	MIMEType string
	Ext      string
}

// AudioMetadata holds parsed metadata for an audio file.
type AudioMetadata struct {
	Title       string
	Artist      string
	Album       string
	Genre       string
	TrackNumber int
	DiscNumber  int
	Year        int
	FileName    string
	FileHash    string // SHA256 hex
	MIMEType    string
	Picture     *Picture // embedded cover art, may be nil
}

var extToMIME = map[string]string{
	".mp3":  "audio/mpeg",
	".flac": "audio/flac",
	".ogg":  "audio/ogg",
	".m4a":  "audio/mp4",
	".mp4":  "audio/mp4",
	".wav":  "audio/wav",
	".aac":  "audio/aac",
	".wma":  "audio/x-ms-wma",
	".opus": "audio/opus",
}

// ParseFile opens the file at path and returns parsed audio metadata.
func ParseFile(path string) (*AudioMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("metadata: open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	meta, err := ParseReader(f, path)
	if err != nil {
		return nil, fmt.Errorf("metadata: parse reader: %w", err)
	}
	return meta, nil
}

// ParseReader reads audio metadata from an io.ReadSeeker.
func ParseReader(r io.ReadSeeker, filePath string) (*AudioMetadata, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("metadata: read data: %w", err)
	}

	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	mime := MIMETypeByExt(filePath)
	fileName := filepath.Base(filePath)

	meta := &AudioMetadata{
		FileName: fileName,
		FileHash: fileHash,
		MIMEType: mime,
	}

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return meta, nil //nolint:nilerr // return basic info, don't crash on seek error
	}

	m, err := tag.ReadFrom(r)
	if err != nil {
		return meta, nil //nolint:nilerr // return basic info on tag parse failure
	}

	meta.Title = m.Title()
	meta.Artist = m.Artist()
	meta.Album = m.Album()
	meta.Genre = m.Genre()
	meta.Year = m.Year()

	if track, _ := m.Track(); track > 0 {
		meta.TrackNumber = track
	}
	if disc, _ := m.Disc(); disc > 0 {
		meta.DiscNumber = disc
	}

	if pic := m.Picture(); pic != nil {
		meta.Picture = &Picture{
			Data:     pic.Data,
			MIMEType: pic.MIMEType,
			Ext:      pic.Ext,
		}
	}

	return meta, nil
}

// MIMETypeByExt returns the MIME type for a given file path.
func MIMETypeByExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if mime, ok := extToMIME[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// AudioFileExts returns a set of supported audio file extensions.
func AudioFileExts() map[string]bool {
	exts := make(map[string]bool, len(extToMIME))
	for ext := range extToMIME {
		exts[ext] = true
	}
	return exts
}

// IsAudioFile checks whether the given file path has a supported audio extension.
func IsAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := extToMIME[ext]
	return ok
}
