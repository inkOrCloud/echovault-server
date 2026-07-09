// Package rest provides REST API handlers for file upload/download operations.
package rest

import (
	"crypto/sha256"
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

// cache duration for cover images (7 days)
const coverCacheMaxAge = 7 * 24 * time.Hour

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

	// Health check endpoint
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
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
		err := h.handleAudioUpload(c, songID, file, header.Filename, header.Size)
		if err != nil {
			return // response already written
		}
	case "cover":
		err := h.Storage.SaveCover(c.Request.Context(), songID, file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{errField: "save cover: " + err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{errField: "invalid type: " + fileType})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

	tmpForRead, err := os.Open(tmpPath) //nolint:gosec // tmpPath is from os.CreateTemp
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errField: "open temp file: " + err.Error()})
		return fmt.Errorf("open temp file: %w", err)
	}
	defer func() { _ = tmpForRead.Close() }()

	saveErr := h.Storage.SaveAudio(c.Request.Context(), songID, fileName, tmpForRead)
	if saveErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errField: "save audio: " + saveErr.Error()})
		return fmt.Errorf("save audio: %w", saveErr)
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

func (h *Handler) handleDownloadAudio(c *gin.Context) {
	songID := c.Param("songID")
	reader, size, err := h.Storage.GetAudio(c.Request.Context(), songID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{errField: "audio not found"})
		return
	}
	defer func() { _ = reader.Close() }()

	contentType := "audio/mpeg"
	// Check if client sent a Range header (for seeking support)
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		// Parse range header and serve partial content
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

func (h *Handler) serveRangeRequest(c *gin.Context, reader io.ReadCloser, totalSize int64, rangeHeader string, contentType string) {
	// Parse "bytes=start-end"
	rangeParts := strings.Split(rangeHeader, "=")
	if len(rangeParts) != 2 || rangeParts[0] != "bytes" {
		c.Header("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	rangeVal := strings.TrimSpace(rangeParts[1])
	var start, end int64

	if strings.HasPrefix(rangeVal, "-") {
		// Suffix range: -500 (last 500 bytes)
		suffix, err := strconv.ParseInt(strings.TrimPrefix(rangeVal, "-"), 10, 64)
		if err != nil || suffix <= 0 {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		start = totalSize - suffix
		if start < 0 {
			start = 0
		}
		end = totalSize - 1
	} else {
		parts := strings.SplitN(rangeVal, "-", 2)
		var err error
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || start < 0 || start >= totalSize {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		if len(parts) == 2 && parts[1] != "" {
			end, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil || end >= totalSize {
				end = totalSize - 1
			}
		} else {
			// Default: read to end
			end = totalSize - 1
		}
	}

	if start > end || start < 0 {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLength := end - start + 1
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Status(http.StatusPartialContent)

	// Seek and copy the requested range
	if seeker, ok := reader.(io.Seeker); ok {
		_, _ = seeker.Seek(start, io.SeekStart)
		_, _ = io.CopyN(c.Writer, reader, contentLength)
	} else {
		// Fallback: read and discard
		buf := make([]byte, 32*1024)
		written := int64(0)
		for written < contentLength {
			remaining := contentLength - written
			readSize := int64(len(buf))
			if remaining < readSize {
				readSize = remaining
			}
			n, readErr := reader.Read(buf[:readSize])
			if n > 0 {
				_, writeErr := c.Writer.Write(buf[:n])
				if writeErr != nil {
					return
				}
				written += int64(n)
			}
			if readErr != nil {
				return
			}
		}
	}
}

func (h *Handler) handleDownloadCover(c *gin.Context) {
	songID := c.Param("songID")

	// Generate ETag based on songID for caching
	etag := fmt.Sprintf("\"%x\"", sha256.Sum256([]byte("cover:"+songID)))

	// Check If-None-Match (client cache validation)
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
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
