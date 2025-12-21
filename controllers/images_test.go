package controllers

import (
	"blog-backend/models"
	"blog-backend/services"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Helper function to create a user with specific permissions
func createTestUserWithPermissions(id uint, permissions ...string) *models.User {
	perms := make([]models.Permission, len(permissions))
	for i, permName := range permissions {
		perms[i] = models.Permission{Name: permName}
	}
	return &models.User{
		Model: gorm.Model{ID: id},
		Roles: []models.Role{
			{
				Permissions: perms,
			},
		},
	}
}

// stubImageRepoForCtrl implements ImageRepository interface for testing
type stubImageRepoForCtrl struct {
	createFunc        func(*models.Image) error
	getByIDFunc       func(uint) (*models.Image, error)
	listFunc          func(int, int) ([]models.Image, int64, error)
	updateFunc        func(*models.Image) error
	deleteFunc        func(uint) error
	getByFilenameFunc func(string) (*models.Image, error)
}

func (s *stubImageRepoForCtrl) Create(image *models.Image) error {
	if s.createFunc != nil {
		return s.createFunc(image)
	}
	image.ID = 1
	return nil
}

func (s *stubImageRepoForCtrl) GetByID(id uint) (*models.Image, error) {
	if s.getByIDFunc != nil {
		return s.getByIDFunc(id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *stubImageRepoForCtrl) List(offset, limit int) ([]models.Image, int64, error) {
	if s.listFunc != nil {
		return s.listFunc(offset, limit)
	}
	return []models.Image{}, 0, nil
}

func (s *stubImageRepoForCtrl) Update(image *models.Image) error {
	if s.updateFunc != nil {
		return s.updateFunc(image)
	}
	return nil
}

func (s *stubImageRepoForCtrl) Delete(id uint) error {
	if s.deleteFunc != nil {
		return s.deleteFunc(id)
	}
	return nil
}

func (s *stubImageRepoForCtrl) GetByFilename(filename string) (*models.Image, error) {
	if s.getByFilenameFunc != nil {
		return s.getByFilenameFunc(filename)
	}
	return nil, gorm.ErrRecordNotFound
}

// stubStorageBackendForCtrl implements StorageBackend interface for testing
type stubStorageBackendForCtrl struct {
	saveFunc               func(*multipart.FileHeader) (string, string, error)
	deleteFunc             func(string) error
	getImageDimensionsFunc func(multipart.File) (int, int, error)
}

func (s *stubStorageBackendForCtrl) Save(fileHeader *multipart.FileHeader) (string, string, error) {
	if s.saveFunc != nil {
		return s.saveFunc(fileHeader)
	}
	return "test-uuid.jpg", "http://localhost/test-uuid.jpg", nil
}

func (s *stubStorageBackendForCtrl) Delete(filename string) error {
	if s.deleteFunc != nil {
		return s.deleteFunc(filename)
	}
	return nil
}

func (s *stubStorageBackendForCtrl) GetImageDimensions(file multipart.File) (int, int, error) {
	if s.getImageDimensionsFunc != nil {
		return s.getImageDimensionsFunc(file)
	}
	return 100, 100, nil
}

func TestImageController_UploadImage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successfully uploads image", func(t *testing.T) {
		imageRepo := &stubImageRepoForCtrl{
			createFunc: func(img *models.Image) error {
				img.ID = 1
				return nil
			},
		}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("currentUser", createTestUserWithPermissions(1, "upload_image"))
			c.Next()
		})
		router.POST("/upload", controller.UploadImage)

		// 创建 multipart 请求，必须设置正确的 Content-Type
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="file"; filename="test.jpg"`)
		h.Set("Content-Type", "image/jpeg")
		part, _ := writer.CreatePart(h)
		part.Write([]byte("test image data"))

		writer.WriteField("alt", "Test alt text")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var result models.Image
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.Equal(t, "test-uuid.jpg", result.Filename)
		assert.Equal(t, "test.jpg", result.OriginalName)
		assert.Equal(t, "Test alt text", result.Alt)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		imageRepo := &stubImageRepoForCtrl{}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.POST("/upload", controller.UploadImage)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.CreateFormFile("file", "test.jpg")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 when file is missing", func(t *testing.T) {
		imageRepo := &stubImageRepoForCtrl{}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("currentUser", createTestUserWithPermissions(1, "upload_image"))
			c.Next()
		})
		router.POST("/upload", controller.UploadImage)

		req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewBuffer([]byte{}))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 when image format is invalid", func(t *testing.T) {
		imageRepo := &stubImageRepoForCtrl{}
		storage := &stubStorageBackendForCtrl{
			saveFunc: func(fh *multipart.FileHeader) (string, string, error) {
				return "", "", services.ErrInvalidImageFormat
			},
		}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("currentUser", createTestUserWithPermissions(1, "upload_image"))
			c.Next()
		})
		router.POST("/upload", controller.UploadImage)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="file"; filename="test.txt"`)
		h.Set("Content-Type", "text/plain")
		part, _ := writer.CreatePart(h)
		part.Write([]byte("not an image"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 when image is too large", func(t *testing.T) {
		imageRepo := &stubImageRepoForCtrl{}
		storage := &stubStorageBackendForCtrl{
			saveFunc: func(fh *multipart.FileHeader) (string, string, error) {
				return "", "", services.ErrImageTooLarge
			},
		}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("currentUser", createTestUserWithPermissions(1, "upload_image"))
			c.Next()
		})
		router.POST("/upload", controller.UploadImage)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="file"; filename="large.jpg"`)
		h.Set("Content-Type", "image/jpeg")
		part, _ := writer.CreatePart(h)
		part.Write([]byte("large image data"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestImageController_ListImages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns image list successfully", func(t *testing.T) {
		images := []models.Image{
			{Filename: "test1.jpg", URL: "http://localhost/test1.jpg"},
			{Filename: "test2.jpg", URL: "http://localhost/test2.jpg"},
		}

		imageRepo := &stubImageRepoForCtrl{
			listFunc: func(offset, limit int) ([]models.Image, int64, error) {
				return images, 2, nil
			},
		}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.GET("/images", controller.ListImages)

		req := httptest.NewRequest(http.MethodGet, "/images?limit=20&offset=0", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var result map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.Equal(t, float64(2), result["total"])
		items := result["items"].([]interface{})
		assert.Len(t, items, 2)
	})
}

func TestImageController_GetImage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns image by ID", func(t *testing.T) {
		expectedImage := &models.Image{
			Filename: "test.jpg",
			URL:      "http://localhost/test.jpg",
		}
		expectedImage.ID = 1

		imageRepo := &stubImageRepoForCtrl{
			getByIDFunc: func(id uint) (*models.Image, error) {
				if id == 1 {
					return expectedImage, nil
				}
				return nil, gorm.ErrRecordNotFound
			},
		}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.GET("/images/:id", controller.GetImage)

		req := httptest.NewRequest(http.MethodGet, "/images/1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var result models.Image
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.Equal(t, "test.jpg", result.Filename)
	})

	t.Run("returns 404 when image not found", func(t *testing.T) {
		imageRepo := &stubImageRepoForCtrl{
			getByIDFunc: func(id uint) (*models.Image, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.GET("/images/:id", controller.GetImage)

		req := httptest.NewRequest(http.MethodGet, "/images/999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 400 for invalid ID", func(t *testing.T) {
		imageRepo := &stubImageRepoForCtrl{}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.GET("/images/:id", controller.GetImage)

		req := httptest.NewRequest(http.MethodGet, "/images/invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestImageController_UpdateImage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("updates image successfully", func(t *testing.T) {
		updatedImage := &models.Image{
			Filename: "test.jpg",
			Alt:      "Updated alt",
			UserID:   1, // 拥有者
		}
		updatedImage.ID = 1

		imageRepo := &stubImageRepoForCtrl{
			getByIDFunc: func(id uint) (*models.Image, error) {
				if id == 1 {
					return updatedImage, nil
				}
				return nil, gorm.ErrRecordNotFound
			},
			updateFunc: func(img *models.Image) error {
				return nil
			},
		}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("currentUser", createTestUserWithPermissions(1)) // 图片拥有者，无需额外权限
			c.Next()
		})
		router.PUT("/images/:id", controller.UpdateImage)

		body, _ := json.Marshal(map[string]string{"alt": "Updated alt"})
		req := httptest.NewRequest(http.MethodPut, "/images/1", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var result models.Image
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.Equal(t, "Updated alt", result.Alt)
	})

	t.Run("returns 404 when image not found", func(t *testing.T) {
		imageRepo := &stubImageRepoForCtrl{
			getByIDFunc: func(id uint) (*models.Image, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("currentUser", createTestUserWithPermissions(1))
			c.Next()
		})
		router.PUT("/images/:id", controller.UpdateImage)

		body, _ := json.Marshal(map[string]string{"alt": "Updated alt"})
		req := httptest.NewRequest(http.MethodPut, "/images/999", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 400 for invalid request body", func(t *testing.T) {
		imageRepo := &stubImageRepoForCtrl{}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("currentUser", createTestUserWithPermissions(1))
			c.Next()
		})
		router.PUT("/images/:id", controller.UpdateImage)

		req := httptest.NewRequest(http.MethodPut, "/images/1", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestImageController_DeleteImage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("deletes image successfully", func(t *testing.T) {
		imageRepo := &stubImageRepoForCtrl{
			getByIDFunc: func(id uint) (*models.Image, error) {
				if id == 1 {
					img := &models.Image{Filename: "test.jpg", UserID: 1}
					img.ID = 1
					return img, nil
				}
				return nil, gorm.ErrRecordNotFound
			},
			deleteFunc: func(id uint) error {
				return nil
			},
		}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("currentUser", createTestUserWithPermissions(1)) // 图片拥有者
			c.Next()
		})
		router.DELETE("/images/:id", controller.DeleteImage)

		req := httptest.NewRequest(http.MethodDelete, "/images/1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns 404 when image not found", func(t *testing.T) {
		imageRepo := &stubImageRepoForCtrl{
			getByIDFunc: func(id uint) (*models.Image, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}
		storage := &stubStorageBackendForCtrl{}
		imageSvc := services.NewImageService(imageRepo, storage)
		controller := NewImageController(imageSvc)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("currentUser", createTestUserWithPermissions(1))
			c.Next()
		})
		router.DELETE("/images/:id", controller.DeleteImage)

		req := httptest.NewRequest(http.MethodDelete, "/images/999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
