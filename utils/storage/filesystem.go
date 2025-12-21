package storage

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
)

// FilesystemStorage 文件系统存储后端
type FilesystemStorage struct {
	BasePath string // 文件存储基础路径
	BaseURL  string // 文件访问完整 URL 前缀
}

// NewFilesystemStorage 创建文件系统存储后端实例
func NewFilesystemStorage(basePath, baseURL string) (*FilesystemStorage, error) {
	// 确保目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 移除 BaseURL 末尾的斜杠
	baseURL = strings.TrimRight(baseURL, "/")

	return &FilesystemStorage{
		BasePath: basePath,
		BaseURL:  baseURL,
	}, nil
}

// Save 保存文件到文件系统
func (fs *FilesystemStorage) Save(fileHeader *multipart.FileHeader) (string, string, error) {
	// 生成唯一文件名
	ext := filepath.Ext(fileHeader.Filename)
	filename := uuid.New().String() + ext

	// 构造完整路径
	fullPath := filepath.Join(fs.BasePath, filename)

	// 打开上传的文件
	src, err := fileHeader.Open()
	if err != nil {
		return "", "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// 创建目标文件
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// 复制文件内容
	if _, err := io.Copy(dst, src); err != nil {
		return "", "", fmt.Errorf("failed to save file: %w", err)
	}

	// 构造完整 URL
	url := fs.BaseURL + "/" + filename

	return filename, url, nil
}

// Delete 从文件系统删除文件
func (fs *FilesystemStorage) Delete(filename string) error {
	fullPath := filepath.Join(fs.BasePath, filename)

	if err := os.Remove(fullPath); err != nil {
		// 如果文件不存在，不视为错误
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetImageDimensions 获取图片尺寸
func (fs *FilesystemStorage) GetImageDimensions(file multipart.File) (int, int, error) {
	// 重置文件指针到开头
	if _, err := file.Seek(0, 0); err != nil {
		return 0, 0, fmt.Errorf("failed to seek file: %w", err)
	}

	// 解码图片以获取尺寸
	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}

	// 重置文件指针到开头（为后续操作准备）
	if _, err := file.Seek(0, 0); err != nil {
		return 0, 0, fmt.Errorf("failed to seek file: %w", err)
	}

	return img.Width, img.Height, nil
}
