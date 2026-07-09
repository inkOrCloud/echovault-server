// Package rest provides REST API handlers for file upload/download operations.
package rest

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/metadata"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/storage"
)

const errField = "error"

const (
	coverCacheMaxAge = 7 * 24 * time.Hour
	copyBufSize      = 32 * 1024 // 32 KB copy buffer
	rangePartsCount  = 2
	rangeValParts    = 2
	headerPartsCount = 2
)

// Handler handles REST file requests.
type Handler struct {
	Storage     storage.Storage
	SongUpdater SongUpdater
	router      *gin.Engine
}

// NewHandler creates a REST Handler.
func NewHandler(s storage.Storage, songUpdater SongUpdater) *Handler {
	gin.SetMode(gin.ReleaseMode)
	h := &Handler{Storage: s, SongUpdater: songUpdater}
	router := gin.New()
	router.Use(gin.Recovery())

	api := router.Group("/api/v1/files")
	{
		api.POST("/upload", h.handleUpload)
		api.GET("/download/audio/:songID", h.handleDownloadAudio)
		api.GET("/download/cover/:songID", h.handleDownloadCover)
		api.DELETE("/:type/:songID", h.handleDelete)
	}

	router.GET("/api/v1/health", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		_, _ = c.Writer.WriteString(`{"status":"ok"}`)
	})

	h.router = router
	return h
}

// ServeHTTP dispatches to the gin router.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) handleUpload(c *gin.Context) {
	fileType := c.Query("type")
	songID := c.Query("song_id")
	if fileType == "" || songID == "" {
		c.JSON(http.StatusBadRequest, gin.H{errField: "type and song_id required"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errField: err.Error()})
		return
	}
	defer func() { _ = file.Close() }()

	switch fileType {
	case "audio":
		err = h.handleAudioUpload(c, songID, file, header.Filename, header.Size)
		if err != nil {
			return
		}
	case "cover":
		err = h.Storage.SaveCover(c.Request.Context(), songID, file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{errField: "save cover: " + err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{errField: "invalid type: " + fileType})
		return
	}

	c.JSON(http.StatusOK, gin.H{errField: "ok"})
}

func (h *Handler) handleAudioUpload(c *gin.Context, songID string, file io.Reader, fileName string, fileSize int64) error {
	tmpFile, err := os.CreateTemp("", "echovault-upload-*"+filepath.Ext(fileName))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errField: "create temp file: " + err.Error()})
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	_, copyErr := io.Copy(tmpFile, file)
	_ = tmpFile.Close()
	if copyErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errField: "save temp file: " + copyErr.Error()})
		return fmt.Errorf("save temp file: %w", copyErr)
	}

	meta, _ := metadata.ParseFile(tmpPath)

	tmpForRead, lerr := os.Open(tmpPath) //nolint:gosec // tmpPath is from os.CreateTemp
	if lerr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errField: "open temp file: " + lerr.Error()})
		return fmt.Errorf("open temp file: %w", lerr)
	}
	defer func() { _ = tmpForRead.Close() }()

	err = h.Storage.SaveAudio(c.Request.Context(), songID, fileName, tmpForRead)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errField: "save audio: " + err.Error()})
		return fmt.Errorf("save audio: %w", err)
	}

	if meta != nil && meta.Picture != nil {
		coverReader := &pictureReader{data: meta.Picture.Data}
		_ = h.Storage.SaveCover(c.Request.Context(), songID, coverReader)
	}

	if h.SongUpdater != nil && meta != nil {
		_ = h.SongUpdater.UpdateFromScan(songID, meta, fileSize)
	}

	return nil
}

// pictureReader wraps []byte as io.Reader.
type pictureReader struct {
	data   []byte
	offset int
}

func (r *pictureReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}

var (
	errInvalidRangeFormat  = errors.New("invalid range format")
	errInvalidRangeStart   = errors.New("invalid range start")
	errStartGreaterThanEnd = errors.New("range start > end")
)

func (h *Handler) handleDownloadAudio(c *gin.Context) {
	songID := c.Param("songID")
	reader, size, err := h.Storage.GetAudio(c.Request.Context(), songID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{errField: "audio not found"})
		return
	}
	defer func() { _ = reader.Close() }()

	contentType := "audio/mpeg"
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		h.serveRangeRequest(c, reader, size, rangeHeader, contentType)
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, reader)
}

type parsedRange struct {
	start, end int64
}

func parseRangeHeader(rangeVal string, totalSize int64) (*parsedRange, error) {
	// Handle suffix range: -500 (last 500 bytes)
	prefix, hasPrefix := strings.CutPrefix(rangeVal, "-")
	if hasPrefix {
		suffix, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil || suffix <= 0 {
			return nil, errInvalidRangeFormat
		}
		start := totalSize - suffix
		start = max(start, 0)
		return &parsedRange{start: start, end: totalSize - 1}, nil
	}

	parts := strings.SplitN(rangeVal, "-", rangeValParts)
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 {
		return nil, errInvalidRangeStart
	}
	if start >= totalSize {
		return nil, errInvalidRangeStart
	}

	if len(parts) != rangeValParts || parts[1] == "" {
		return &parsedRange{start: start, end: totalSize - 1}, nil
	}

	end, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || end >= totalSize {
		end = totalSize - 1
	}

	if start > end {
		return nil, errStartGreaterThanEnd
	}

	return &parsedRange{start: start, end: end}, nil
}

func (h *Handler) serveRangeRequest(c *gin.Context, reader io.ReadCloser, totalSize int64, rangeHeader, contentType string) {
	rangeParts := strings.SplitN(rangeHeader, "=", headerPartsCount)
	if len(rangeParts) != headerPartsCount || rangeParts[0] != "bytes" {
		c.Header("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	pr, err := parseRangeHeader(strings.TrimSpace(rangeParts[1]), totalSize)
	if err != nil {
		c.Header("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLength := pr.end - pr.start + 1
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", pr.start, pr.end, totalSize))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Status(http.StatusPartialContent)

	if seeker, ok := reader.(io.Seeker); ok {
		_, _ = seeker.Seek(pr.start, io.SeekStart)
		_, _ = io.CopyN(c.Writer, reader, contentLength)
		return
	}

	// Fallback for non-seekable readers: skip to start then copy range
	buf := make([]byte, copyBufSize)
	for remaining := pr.start; remaining > 0; {
		readSize := int64(len(buf))
		readSize = min(remaining, readSize)
		n, readErr := reader.Read(buf[:readSize])
		remaining -= int64(n)
		if readErr != nil {
			return
		}
	}

	for written := contentLength; written > 0; {
		readSize := int64(len(buf))
		readSize = min(written, readSize)
		n, readErr := reader.Read(buf[:readSize])
		if n > 0 {
			_, writeErr := c.Writer.Write(buf[:n])
			if writeErr != nil {
				return
			}
			written -= int64(n)
		}
		if readErr != nil {
			return
		}
	}
}

func (h *Handler) handleDownloadCover(c *gin.Context) {
	songID := c.Param("songID")

	etag := fmt.Sprintf("\"%x\"", sha256.Sum256([]byte("cover:"+songID)))

	if match := c.GetHeader("If-None-Match"); match == etag {
		c.Status(http.StatusNotModified)
		return
	}

	reader, size, err := h.Storage.GetCover(c.Request.Context(), songID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{errField: "cover not found"})
		return
	}
	defer func() { _ = reader.Close() }()

	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("Cache-Control", "public, max-age="+strconv.Itoa(int(coverCacheMaxAge.Seconds())))
	c.Header("ETag", etag)
	c.Header("Expires", time.Now().Add(coverCacheMaxAge).Format(http.TimeFormat))
	c.DataFromReader(http.StatusOK, size, "image/jpeg", reader, nil)
}

func (h *Handler) handleDelete(c *gin.Context) {
	songID := c.Param("songID")
	err := h.Storage.DeleteSongFiles(c.Request.Context(), songID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errField: fmt.Sprintf("delete song: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{errField: "deleted"})
}
