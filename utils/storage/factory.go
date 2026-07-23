package storage

import (
	"fmt"
	"mime/multipart"
	"os"
)

// StorageBackend 定义存储后端的通用接口
type StorageBackend interface {
	// Save 保存文件并返回文件名和完整 URL
	Save(file *multipart.FileHeader) (filename string, url string, err error)
	// Delete 删除文件
	Delete(filename string) error
	// GetImageDimensions 获取图片尺寸
	GetImageDimensions(file multipart.File) (width, height int, err error)
}

// VariantStorageBackend is implemented by storage backends that can save a
// deterministic derivative next to an original image.
type VariantStorageBackend interface {
	StorageBackend
	SaveVariant(originalFilename, label string, file *multipart.FileHeader) (filename string, url string, err error)
}

// NewStorageFromEnv 从环境变量创建存储后端实例
func NewStorageFromEnv() (StorageBackend, error) {
	backend := getEnv("STORAGE_BACKEND", "filesystem")

	switch backend {
	case "filesystem":
		basePath := getEnv("STORAGE_FS_BASE_PATH", "./uploads")
		baseURL := getEnv("STORAGE_FS_BASE_URL", "http://localhost:8080/uploads")
		return NewFilesystemStorage(basePath, baseURL)
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", backend)
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
