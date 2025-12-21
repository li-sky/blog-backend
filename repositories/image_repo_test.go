package repositories

import (
	"blog-backend/models"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupImageTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Image{})
	require.NoError(t, err)

	return db
}

func TestImageRepo_Create(t *testing.T) {
	db := setupImageTestDB(t)
	repo := NewImageRepo(db)

	// 创建测试用户
	user := &models.User{Username: "testuser", Email: "test@example.com", Password: "password"}
	db.Create(user)

	image := &models.Image{
		Filename:       "test.jpg",
		OriginalName:   "original.jpg",
		Alt:            "Test image",
		MimeType:       "image/jpeg",
		Size:           1024,
		Width:          800,
		Height:         600,
		URL:            "http://localhost/test.jpg",
		UserID:         user.ID,
		StorageBackend: "filesystem",
		StoragePath:    "uploads/test.jpg",
		IPAddress:      "127.0.0.1",
	}

	err := repo.Create(image)

	require.NoError(t, err)
	assert.NotZero(t, image.ID)
}

func TestImageRepo_GetByID(t *testing.T) {
	db := setupImageTestDB(t)
	repo := NewImageRepo(db)

	user := &models.User{Username: "testuser", Email: "test@example.com", Password: "password"}
	db.Create(user)

	original := &models.Image{
		Filename:       "test.jpg",
		OriginalName:   "original.jpg",
		MimeType:       "image/jpeg",
		Size:           1024,
		URL:            "http://localhost/test.jpg",
		UserID:         user.ID,
		StorageBackend: "filesystem",
		StoragePath:    "uploads/test.jpg",
	}
	db.Create(original)

	result, err := repo.GetByID(original.ID)

	require.NoError(t, err)
	assert.Equal(t, original.ID, result.ID)
	assert.Equal(t, original.Filename, result.Filename)
	assert.NotNil(t, result.User)
	assert.Equal(t, user.Username, result.User.Username)
}

func TestImageRepo_GetByID_NotFound(t *testing.T) {
	db := setupImageTestDB(t)
	repo := NewImageRepo(db)

	result, err := repo.GetByID(9999)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestImageRepo_List(t *testing.T) {
	db := setupImageTestDB(t)
	repo := NewImageRepo(db)

	user := &models.User{Username: "testuser", Email: "test@example.com", Password: "password"}
	db.Create(user)

	// 创建多个图片
	for i := 1; i <= 5; i++ {
		img := &models.Image{
			Filename:       "test" + string(rune(i)) + ".jpg",
			OriginalName:   "original.jpg",
			MimeType:       "image/jpeg",
			Size:           1024,
			URL:            "http://localhost/test.jpg",
			UserID:         user.ID,
			StorageBackend: "filesystem",
			StoragePath:    "uploads/test.jpg",
		}
		db.Create(img)
	}

	images, total, err := repo.List(0, 10)

	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, images, 5)
}

func TestImageRepo_List_Pagination(t *testing.T) {
	db := setupImageTestDB(t)
	repo := NewImageRepo(db)

	user := &models.User{Username: "testuser", Email: "test@example.com", Password: "password"}
	db.Create(user)

	for i := 1; i <= 10; i++ {
		img := &models.Image{
			Filename:       "test" + string(rune(i)) + ".jpg",
			OriginalName:   "original.jpg",
			MimeType:       "image/jpeg",
			Size:           1024,
			URL:            "http://localhost/test.jpg",
			UserID:         user.ID,
			StorageBackend: "filesystem",
			StoragePath:    "uploads/test.jpg",
		}
		db.Create(img)
	}

	images, total, err := repo.List(5, 3)

	require.NoError(t, err)
	assert.Equal(t, int64(10), total)
	assert.Len(t, images, 3)
}

func TestImageRepo_Update(t *testing.T) {
	db := setupImageTestDB(t)
	repo := NewImageRepo(db)

	user := &models.User{Username: "testuser", Email: "test@example.com", Password: "password"}
	db.Create(user)

	image := &models.Image{
		Filename:       "test.jpg",
		OriginalName:   "original.jpg",
		Alt:            "Old alt",
		MimeType:       "image/jpeg",
		Size:           1024,
		URL:            "http://localhost/test.jpg",
		UserID:         user.ID,
		StorageBackend: "filesystem",
		StoragePath:    "uploads/test.jpg",
	}
	db.Create(image)

	image.Alt = "New alt"
	err := repo.Update(image)

	require.NoError(t, err)

	updated, _ := repo.GetByID(image.ID)
	assert.Equal(t, "New alt", updated.Alt)
}

func TestImageRepo_Delete(t *testing.T) {
	db := setupImageTestDB(t)
	repo := NewImageRepo(db)

	user := &models.User{Username: "testuser", Email: "test@example.com", Password: "password"}
	db.Create(user)

	image := &models.Image{
		Filename:       "test.jpg",
		OriginalName:   "original.jpg",
		MimeType:       "image/jpeg",
		Size:           1024,
		URL:            "http://localhost/test.jpg",
		UserID:         user.ID,
		StorageBackend: "filesystem",
		StoragePath:    "uploads/test.jpg",
	}
	db.Create(image)

	err := repo.Delete(image.ID)

	require.NoError(t, err)

	_, err = repo.GetByID(image.ID)
	assert.Error(t, err)
}

func TestImageRepo_GetByFilename(t *testing.T) {
	db := setupImageTestDB(t)
	repo := NewImageRepo(db)

	user := &models.User{Username: "testuser", Email: "test@example.com", Password: "password"}
	db.Create(user)

	image := &models.Image{
		Filename:       "unique-test.jpg",
		OriginalName:   "original.jpg",
		MimeType:       "image/jpeg",
		Size:           1024,
		URL:            "http://localhost/test.jpg",
		UserID:         user.ID,
		StorageBackend: "filesystem",
		StoragePath:    "uploads/test.jpg",
	}
	db.Create(image)

	result, err := repo.GetByFilename("unique-test.jpg")

	require.NoError(t, err)
	assert.Equal(t, image.ID, result.ID)
	assert.Equal(t, "unique-test.jpg", result.Filename)
}

func TestImageRepo_GetByFilename_NotFound(t *testing.T) {
	db := setupImageTestDB(t)
	repo := NewImageRepo(db)

	result, err := repo.GetByFilename("nonexistent.jpg")

	assert.Error(t, err)
	assert.Nil(t, result)
}
