package controllers

import (
	"blog-backend/middlewares"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const MaxAttachmentSize = 100 * 1024 * 1024

type FileController struct{ BasePath string }
type attachmentMetadata struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func NewFileController() *FileController {
	root := os.Getenv("STORAGE_FS_BASE_PATH")
	if root == "" {
		root = "./uploads"
	}
	return &FileController{BasePath: filepath.Join(root, "attachments")}
}

// Upload reuses the existing upload_image permission for author attachments.
func (ctrl *FileController) Upload(c *gin.Context) {
	user, ok := middlewares.CurrentUser(c)
	if !ok {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	allowed := false
	for _, role := range user.Roles {
		for _, permission := range role.Permissions {
			if permission.Name == "upload_image" {
				allowed = true
			}
		}
	}
	if !allowed {
		c.JSON(403, gin.H{"error": "permission denied: upload_image required"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxAttachmentSize+1024*1024)
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "file is required; maximum size is 100 MB"})
		return
	}
	if file.Size > MaxAttachmentSize {
		c.JSON(413, gin.H{"error": "file exceeds 100 MB limit"})
		return
	}
	name := filepath.Base(strings.ReplaceAll(file.Filename, "\\", "/"))
	if name == "." || name == "/" || strings.ContainsAny(name, "\r\n\x00") {
		c.JSON(400, gin.H{"error": "invalid filename"})
		return
	}
	id := uuid.NewString()
	dir := filepath.Join(ctrl.BasePath, id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(500, gin.H{"error": "cannot create attachment"})
		return
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(dir)
		}
	}()
	if err := c.SaveUploadedFile(file, filepath.Join(dir, "content")); err != nil {
		c.JSON(500, gin.H{"error": "cannot save attachment"})
		return
	}
	meta := attachmentMetadata{Name: name, Size: file.Size}
	data, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), data, 0644); err != nil {
		c.JSON(500, gin.H{"error": "cannot save metadata"})
		return
	}
	success = true
	c.JSON(201, gin.H{"name": name, "size": file.Size, "url": "/api/v1/files/" + id + "/download"})
}

// Download never serves user-controlled content inline or uses a filename as a path.
func (ctrl *FileController) Download(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Status(404)
		return
	}
	dir := filepath.Join(ctrl.BasePath, id.String())
	data, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	var meta attachmentMetadata
	if err != nil || json.Unmarshal(data, &meta) != nil {
		c.Status(404)
		return
	}
	file, err := os.Open(filepath.Join(dir, "content"))
	if err != nil {
		c.Status(404)
		return
	}
	defer file.Close()
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "sandbox")
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": meta.Name}))
	c.Header("Content-Type", "application/octet-stream")
	c.Status(200)
	_, _ = io.Copy(c.Writer, file)
}
