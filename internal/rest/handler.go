package rest

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/storage"
)

type Handler struct {
	Storage storage.Storage
	router  *gin.Engine
}

func NewHandler(s storage.Storage) *Handler {
	gin.SetMode(gin.ReleaseMode)
	h := &Handler{Storage: s}
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

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) handleUpload(c *gin.Context) {
	fileType := c.Query("type")
	songID := c.Query("song_id")
	if fileType == "" || songID == "" {
		c.JSON(400, gin.H{"error": "type and song_id required"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	switch fileType {
	case "audio":
		err = h.Storage.SaveAudio(c.Request.Context(), songID, header.Filename, file)
	case "cover":
		err = h.Storage.SaveCover(c.Request.Context(), songID, file)
	default:
		c.JSON(400, gin.H{"error": "invalid type: " + fileType})
		return
	}

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *Handler) handleDownloadAudio(c *gin.Context) {
	songID := c.Param("songID")
	reader, size, err := h.Storage.GetAudio(c.Request.Context(), songID)
	if err != nil {
		c.JSON(404, gin.H{"error": "audio not found"})
		return
	}
	defer reader.Close()

	c.Header("Content-Type", "audio/mpeg")
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("Accept-Ranges", "bytes")
	c.DataFromReader(200, size, "audio/mpeg", reader, nil)
}

func (h *Handler) handleDownloadCover(c *gin.Context) {
	songID := c.Param("songID")
	reader, size, err := h.Storage.GetCover(c.Request.Context(), songID)
	if err != nil {
		c.JSON(404, gin.H{"error": "cover not found"})
		return
	}
	defer reader.Close()

	c.DataFromReader(200, size, "image/jpeg", reader, nil)
}

func (h *Handler) handleDelete(c *gin.Context) {
	songID := c.Param("songID")
	if err := h.Storage.DeleteSongFiles(c.Request.Context(), songID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}
