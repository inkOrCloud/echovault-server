// Package rest provides REST API handlers for file upload/download operations.
package rest

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/metadata"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/storage"
)

const errField = "error"

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

	c.Header("Content-Type", "audio/mpeg")
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("Accept-Ranges", "bytes")
	c.DataFromReader(http.StatusOK, size, "audio/mpeg", reader, nil)
}

func (h *Handler) handleDownloadCover(c *gin.Context) {
	songID := c.Param("songID")
	reader, size, err := h.Storage.GetCover(c.Request.Context(), songID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{errField: "cover not found"})
		return
	}
	defer func() { _ = reader.Close() }()

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
