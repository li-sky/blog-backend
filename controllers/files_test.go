package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-backend/models"
	"github.com/gin-gonic/gin"
)

func TestAttachmentUploadAndDownload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := &FileController{BasePath: t.TempDir()}
	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
		if c.GetHeader("X-Test-Auth") != "" {
			user := &models.User{}
			if c.GetHeader("X-Test-Auth") == "allowed" {
				user.Roles = []models.Role{{Permissions: []models.Permission{{Name: "upload_image"}}}}
			}
			c.Set("currentUser", user)
		}
		ctrl.Upload(c)
	})
	r.GET("/api/v1/files/:id/download", ctrl.Download)
	for _, tc := range []struct {
		auth   string
		status int
	}{{"", 401}, {"denied", 403}, {"allowed", 201}} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, _ := writer.CreateFormFile("file", "测试文档.html")
		_, _ = io.WriteString(part, "<script>alert('attachment')</script>")
		_ = writer.Close()
		req := httptest.NewRequest(http.MethodPost, "/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-Test-Auth", tc.auth)
		response := httptest.NewRecorder()
		r.ServeHTTP(response, req)
		if response.Code != tc.status {
			t.Fatalf("auth %q: got %d: %s", tc.auth, response.Code, response.Body.String())
		}
		if response.Code != 201 {
			continue
		}
		var result struct {
			URL  string `json:"url"`
			Name string `json:"name"`
			Size int64  `json:"size"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Name != "测试文档.html" || result.Size == 0 {
			t.Fatalf("bad metadata: %+v", result)
		}
		downloaded := httptest.NewRecorder()
		r.ServeHTTP(downloaded, httptest.NewRequest("GET", result.URL, nil))
		if downloaded.Code != 200 || downloaded.Body.String() != "<script>alert('attachment')</script>" {
			t.Fatalf("download mismatch: %s", downloaded.Body.String())
		}
		if !strings.HasPrefix(downloaded.Header().Get("Content-Disposition"), "attachment;") || downloaded.Header().Get("Content-Type") != "application/octet-stream" || downloaded.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal("unsafe download headers")
		}
	}
	for _, id := range []string{"invalid", "00000000-0000-0000-0000-000000000000"} {
		response := httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/files/"+id+"/download", nil))
		if response.Code != 404 {
			t.Fatalf("unknown id returned %d", response.Code)
		}
	}
}
