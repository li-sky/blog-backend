package services

import (
	"blog-backend/models"
	"bytes"
	"errors"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 创建测试用户（带权限）
func createTestUserWithPermission(id uint, permissionName string) *models.User {
	return &models.User{
		Model: gorm.Model{ID: id},
		Roles: []models.Role{
			{
				Permissions: []models.Permission{
					{Name: permissionName},
				},
			},
		},
	}
}

// 创建没有权限的测试用户
func createTestUserWithoutPermission(id uint) *models.User {
	return &models.User{
		Model: gorm.Model{ID: id},
		Roles: []models.Role{},
	}
}

// Stub ImageRepository
type stubImageRepo struct {
	createFunc        func(*models.Image) error
	getByIDFunc       func(uint) (*models.Image, error)
	listFunc          func(int, int) ([]models.Image, int64, error)
	updateFunc        func(*models.Image) error
	deleteFunc        func(uint) error
	getByFilenameFunc func(string) (*models.Image, error)
}

func (s *stubImageRepo) Create(image *models.Image) error {
	if s.createFunc != nil {
		return s.createFunc(image)
	}
	return nil
}

func (s *stubImageRepo) GetByID(id uint) (*models.Image, error) {
	if s.getByIDFunc != nil {
		return s.getByIDFunc(id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *stubImageRepo) List(offset, limit int) ([]models.Image, int64, error) {
	if s.listFunc != nil {
		return s.listFunc(offset, limit)
	}
	return []models.Image{}, 0, nil
}

func (s *stubImageRepo) Update(image *models.Image) error {
	if s.updateFunc != nil {
		return s.updateFunc(image)
	}
	return nil
}

func (s *stubImageRepo) Delete(id uint) error {
	if s.deleteFunc != nil {
		return s.deleteFunc(id)
	}
	return nil
}

func (s *stubImageRepo) GetByFilename(filename string) (*models.Image, error) {
	if s.getByFilenameFunc != nil {
		return s.getByFilenameFunc(filename)
	}
	return nil, gorm.ErrRecordNotFound
}

// Stub StorageBackend
type stubStorageBackend struct {
	saveFunc               func(*multipart.FileHeader) (string, string, error)
	deleteFunc             func(string) error
	getImageDimensionsFunc func(multipart.File) (int, int, error)
}

func (s *stubStorageBackend) Save(file *multipart.FileHeader) (string, string, error) {
	if s.saveFunc != nil {
		return s.saveFunc(file)
	}
	return "test-uuid.jpg", "http://localhost/test-uuid.jpg", nil
}

func (s *stubStorageBackend) Delete(filename string) error {
	if s.deleteFunc != nil {
		return s.deleteFunc(filename)
	}
	return nil
}

func (s *stubStorageBackend) GetImageDimensions(file multipart.File) (int, int, error) {
	if s.getImageDimensionsFunc != nil {
		return s.getImageDimensionsFunc(file)
	}
	return 100, 100, nil
}

func TestImageService_UploadImage(t *testing.T) {
	t.Run("successfully uploads image", func(t *testing.T) {
		imageRepo := &stubImageRepo{
			createFunc: func(img *models.Image) error {
				img.ID = 1
				return nil
			},
		}
		storageBackend := &stubStorageBackend{}
		service := NewImageService(imageRepo, storageBackend)

		user := createTestUserWithPermission(1, "upload_image")
		fileHeader := createTestFileHeader(t, "test.jpg", "image/jpeg", []byte("test data"))

		result, err := service.UploadImage(user, fileHeader, "Test alt", "127.0.0.1")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-uuid.jpg", result.Filename)
		assert.Equal(t, "test.jpg", result.OriginalName)
		assert.Equal(t, "Test alt", result.Alt)
		assert.Equal(t, "image/jpeg", result.MimeType)
		assert.Equal(t, int64(9), result.Size)
		assert.Equal(t, uint(1), result.UserID)
	})

	t.Run("rejects file that is too large", func(t *testing.T) {
		service := NewImageService(&stubImageRepo{}, &stubStorageBackend{})

		user := createTestUserWithPermission(1, "upload_image")
		// 模拟一个大于 50MB 的文件
		fileHeader := createTestFileHeader(t, "large.jpg", "image/jpeg", []byte("small data"))
		fileHeader.Size = MaxImageSize + 1

		result, err := service.UploadImage(user, fileHeader, "", "127.0.0.1")

		assert.Error(t, err)
		assert.Equal(t, ErrImageTooLarge, err)
		assert.Nil(t, result)
	})

	t.Run("rejects invalid image format by extension", func(t *testing.T) {
		service := NewImageService(&stubImageRepo{}, &stubStorageBackend{})

		user := createTestUserWithPermission(1, "upload_image")
		fileHeader := createTestFileHeader(t, "test.txt", "text/plain", []byte("test"))

		result, err := service.UploadImage(user, fileHeader, "", "127.0.0.1")

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidImageFormat, err)
		assert.Nil(t, result)
	})

	t.Run("rejects invalid MIME type", func(t *testing.T) {
		service := NewImageService(&stubImageRepo{}, &stubStorageBackend{})

		user := createTestUserWithPermission(1, "upload_image")
		fileHeader := createTestFileHeader(t, "test.jpg", "application/pdf", []byte("test"))

		result, err := service.UploadImage(user, fileHeader, "", "127.0.0.1")

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidImageFormat, err)
		assert.Nil(t, result)
	})

	t.Run("deletes file if database save fails", func(t *testing.T) {
		deleteWasCalled := false
		imageRepo := &stubImageRepo{
			createFunc: func(*models.Image) error {
				return errors.New("database error")
			},
		}
		storageBackend := &stubStorageBackend{
			deleteFunc: func(filename string) error {
				deleteWasCalled = true
				return nil
			},
		}
		service := NewImageService(imageRepo, storageBackend)

		user := createTestUserWithPermission(1, "upload_image")
		fileHeader := createTestFileHeader(t, "test.jpg", "image/jpeg", []byte("test"))

		result, err := service.UploadImage(user, fileHeader, "", "127.0.0.1")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, deleteWasCalled, "Delete should be called to cleanup")
	})
}

func TestImageService_GetImage(t *testing.T) {
	t.Run("returns image by ID", func(t *testing.T) {
		expectedImage := &models.Image{
			Filename: "test.jpg",
		}
		expectedImage.ID = 1

		imageRepo := &stubImageRepo{
			getByIDFunc: func(id uint) (*models.Image, error) {
				if id == 1 {
					return expectedImage, nil
				}
				return nil, gorm.ErrRecordNotFound
			},
		}
		service := NewImageService(imageRepo, &stubStorageBackend{})

		result, err := service.GetImage(1)

		require.NoError(t, err)
		assert.Equal(t, expectedImage, result)
	})

	t.Run("returns error when image not found", func(t *testing.T) {
		imageRepo := &stubImageRepo{
			getByIDFunc: func(id uint) (*models.Image, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}
		service := NewImageService(imageRepo, &stubStorageBackend{})

		result, err := service.GetImage(999)

		assert.Error(t, err)
		assert.Equal(t, ErrImageNotFound, err)
		assert.Nil(t, result)
	})
}

func TestImageService_ListImages(t *testing.T) {
	t.Run("returns paginated image list", func(t *testing.T) {
		images := []models.Image{
			{Filename: "test1.jpg"},
			{Filename: "test2.jpg"},
		}

		imageRepo := &stubImageRepo{
			listFunc: func(offset, limit int) ([]models.Image, int64, error) {
				return images, 10, nil
			},
		}
		service := NewImageService(imageRepo, &stubStorageBackend{})

		result, total, err := service.ListImages(0, 20)

		require.NoError(t, err)
		assert.Equal(t, images, result)
		assert.Equal(t, int64(10), total)
	})
}

func TestImageService_UpdateImage(t *testing.T) {
	t.Run("updates image alt text", func(t *testing.T) {
		existingImage := &models.Image{
			Filename: "test.jpg",
			Alt:      "Old alt",
			UserID:   1,
		}
		existingImage.ID = 1

		imageRepo := &stubImageRepo{
			getByIDFunc: func(id uint) (*models.Image, error) {
				if id == 1 {
					return existingImage, nil
				}
				return nil, gorm.ErrRecordNotFound
			},
			updateFunc: func(img *models.Image) error {
				return nil
			},
		}
		service := NewImageService(imageRepo, &stubStorageBackend{})

		// 图片拥有者可以更新
		user := createTestUserWithoutPermission(1)
		result, err := service.UpdateImage(user, 1, "New alt")

		require.NoError(t, err)
		assert.Equal(t, "New alt", result.Alt)
	})

	t.Run("returns error when image not found", func(t *testing.T) {
		imageRepo := &stubImageRepo{
			getByIDFunc: func(id uint) (*models.Image, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}
		service := NewImageService(imageRepo, &stubStorageBackend{})

		user := createTestUserWithoutPermission(1)
		result, err := service.UpdateImage(user, 999, "New alt")

		assert.Error(t, err)
		assert.Equal(t, ErrImageNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("rejects update by non-owner without permission", func(t *testing.T) {
		existingImage := &models.Image{
			Filename: "test.jpg",
			UserID:   2, // 不同的拥有者
		}
		existingImage.ID = 1

		imageRepo := &stubImageRepo{
			getByIDFunc: func(id uint) (*models.Image, error) {
				return existingImage, nil
			},
		}
		service := NewImageService(imageRepo, &stubStorageBackend{})

		// 用户 1 试图更新用户 2 的图片
		user := createTestUserWithoutPermission(1)
		result, err := service.UpdateImage(user, 1, "New alt")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
		assert.Nil(t, result)
	})

	t.Run("allows update with edit_any_image permission", func(t *testing.T) {
		existingImage := &models.Image{
			Filename: "test.jpg",
			UserID:   2,
		}
		existingImage.ID = 1

		imageRepo := &stubImageRepo{
			getByIDFunc: func(id uint) (*models.Image, error) {
				return existingImage, nil
			},
			updateFunc: func(img *models.Image) error {
				return nil
			},
		}
		service := NewImageService(imageRepo, &stubStorageBackend{})

		// 管理员可以编辑任何图片
		admin := createTestUserWithPermission(1, "edit_any_image")
		result, err := service.UpdateImage(admin, 1, "New alt")

		require.NoError(t, err)
		assert.Equal(t, "New alt", result.Alt)
	})
}

func TestImageService_DeleteImage(t *testing.T) {
	t.Run("deletes image and file", func(t *testing.T) {
		existingImage := &models.Image{
			Filename: "test.jpg",
			UserID:   1,
		}
		existingImage.ID = 1

		deleteFileCalled := false
		deleteRecordCalled := false

		imageRepo := &stubImageRepo{
			getByIDFunc: func(id uint) (*models.Image, error) {
				if id == 1 {
					return existingImage, nil
				}
				return nil, gorm.ErrRecordNotFound
			},
			deleteFunc: func(id uint) error {
				deleteRecordCalled = true
				return nil
			},
		}
		storageBackend := &stubStorageBackend{
			deleteFunc: func(filename string) error {
				deleteFileCalled = true
				assert.Equal(t, "test.jpg", filename)
				return nil
			},
		}
		service := NewImageService(imageRepo, storageBackend)

		// 图片拥有者可以删除
		user := createTestUserWithoutPermission(1)
		err := service.DeleteImage(user, 1)

		require.NoError(t, err)
		assert.True(t, deleteFileCalled)
		assert.True(t, deleteRecordCalled)
	})

	t.Run("returns error when image not found", func(t *testing.T) {
		imageRepo := &stubImageRepo{
			getByIDFunc: func(id uint) (*models.Image, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}
		service := NewImageService(imageRepo, &stubStorageBackend{})

		user := createTestUserWithoutPermission(1)
		err := service.DeleteImage(user, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrImageNotFound, err)
	})

	t.Run("rejects delete by non-owner without permission", func(t *testing.T) {
		existingImage := &models.Image{
			Filename: "test.jpg",
			UserID:   2,
		}
		existingImage.ID = 1

		imageRepo := &stubImageRepo{
			getByIDFunc: func(id uint) (*models.Image, error) {
				return existingImage, nil
			},
		}
		service := NewImageService(imageRepo, &stubStorageBackend{})

		user := createTestUserWithoutPermission(1)
		err := service.DeleteImage(user, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
	})

	t.Run("allows delete with delete_any_image permission", func(t *testing.T) {
		existingImage := &models.Image{
			Filename: "test.jpg",
			UserID:   2,
		}
		existingImage.ID = 1

		imageRepo := &stubImageRepo{
			getByIDFunc: func(id uint) (*models.Image, error) {
				return existingImage, nil
			},
			deleteFunc: func(id uint) error {
				return nil
			},
		}
		storageBackend := &stubStorageBackend{
			deleteFunc: func(filename string) error {
				return nil
			},
		}
		service := NewImageService(imageRepo, storageBackend)

		admin := createTestUserWithPermission(1, "delete_any_image")
		err := service.DeleteImage(admin, 1)

		require.NoError(t, err)
	})
}

func TestMatchesVariantDimensions(t *testing.T) {
	assert.True(t, matchesVariantDimensions(600, 338, 1600, 900, 600))
	assert.True(t, matchesVariantDimensions(500, 281, 500, 281, 600))
	assert.False(t, matchesVariantDimensions(600, 600, 1600, 900, 600))
	assert.True(t, matchesVariantDimensions(1600, 900, 1600, 900, 0))
	assert.False(t, matchesVariantDimensions(1200, 675, 1600, 900, 0))

	// A browser-applied 90-degree EXIF orientation is validated by passing
	// swapped original dimensions.
	assert.True(t, matchesVariantDimensions(600, 1067, 900, 1600, 600))
}

func TestSetVariantsCompleteRequiresFullVariant(t *testing.T) {
	image := &models.Image{
		WebP600URL:  "https://example.com/image_w600.webp",
		WebP1200URL: "https://example.com/image_w1200.webp",
	}

	setVariantsComplete(image)
	assert.False(t, image.VariantsComplete)

	image.WebPFullURL = "https://example.com/image_full.webp"
	setVariantsComplete(image)
	assert.True(t, image.VariantsComplete)
}

// Helper functions

func createTestFileHeader(t *testing.T, filename, contentType string, data []byte) *multipart.FileHeader {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)

	part, err := writer.CreatePart(h)
	require.NoError(t, err)

	// 如果是图片格式，创建真实的图片数据
	if contentType == "image/jpeg" || contentType == "image/png" {
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		if contentType == "image/png" {
			err = png.Encode(part, img)
		} else {
			// For JPEG, just write dummy data
			_, err = part.Write(data)
		}
		require.NoError(t, err)
	} else {
		_, err = part.Write(data)
		require.NoError(t, err)
	}

	err = writer.Close()
	require.NoError(t, err)

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(20 << 20) // 20MB
	require.NoError(t, err)

	files := form.File["file"]
	require.Len(t, files, 1)

	return files[0]
}

type mockMultipartFile struct {
	*bytes.Reader
}

func (m *mockMultipartFile) Close() error {
	return nil
}

func (m *mockMultipartFile) Seek(offset int64, whence int) (int64, error) {
	return m.Reader.Seek(offset, whence)
}

var _ multipart.File = (*mockMultipartFile)(nil)
var _ io.ReadSeeker = (*mockMultipartFile)(nil)
