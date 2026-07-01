package rest

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/metadata"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/storage"
)

// Handler 是 REST 文件服务处理器。
type Handler struct {
	Storage     storage.Storage
	SongUpdater SongUpdater
	router      *gin.Engine
}

// NewHandler 创建 REST Handler。
// songUpdater 可为 nil（此时不更新 Song 记录）。
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
		// 保存到临时文件以便解析元数据
		tmpFile, err := os.CreateTemp("", "echovault-upload-*"+filepath.Ext(header.Filename))
		if err != nil {
			c.JSON(500, gin.H{"error": "create temp file: " + err.Error()})
			return
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		if _, err := io.Copy(tmpFile, file); err != nil {
			c.JSON(500, gin.H{"error": "save temp file: " + err.Error()})
			return
		}
		tmpFile.Close()

		// 解析元数据（失败时 meta 为 nil，不阻止上传）
		meta, _ := metadata.ParseFile(tmpPath)

		// 保存音频文件到持久存储
		tmpForRead, err := os.Open(tmpPath)
		if err != nil {
			c.JSON(500, gin.H{"error": "open temp file: " + err.Error()})
			return
		}
		defer tmpForRead.Close()

		if err := h.Storage.SaveAudio(c.Request.Context(), songID, header.Filename, tmpForRead); err != nil {
			c.JSON(500, gin.H{"error": "save audio: " + err.Error()})
			return
		}

		// 如果有嵌入封面，保存到 Storage
		if meta != nil && meta.Picture != nil {
			coverReader := &pictureReader{data: meta.Picture.Data}
			_ = h.Storage.SaveCover(c.Request.Context(), songID, coverReader)
		}

		// 补充 Song 记录中用户未填的字段
		if h.SongUpdater != nil && meta != nil {
			_ = h.SongUpdater.UpdateFromScan(songID, meta, header.Size)
		}

	case "cover":
		if err := h.Storage.SaveCover(c.Request.Context(), songID, file); err != nil {
			c.JSON(500, gin.H{"error": "save cover: " + err.Error()})
			return
		}
	default:
		c.JSON(400, gin.H{"error": "invalid type: " + fileType})
		return
	}

	c.JSON(200, gin.H{"status": "ok"})
}

// pictureReader 将 []byte 包装为 io.Reader，用于 Storage.SaveCover。
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
