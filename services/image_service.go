package services

import (
	"blog-backend/models"
	"blog-backend/utils/storage"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrImageNotFound      = errors.New("image not found")
	ErrInvalidImageFormat = errors.New("invalid image format")
	ErrImageTooLarge      = errors.New("image size exceeds limit")
)

const (
	MaxImageSize = 50 * 1024 * 1024 // 50MB
)

var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

type ImageService struct {
	imageRepo      ImageRepository
	storageBackend storage.StorageBackend
}

func NewImageService(imageRepo ImageRepository, storageBackend storage.StorageBackend) *ImageService {
	return &ImageService{
		imageRepo:      imageRepo,
		storageBackend: storageBackend,
	}
}

// UploadImage 上传图片
func (s *ImageService) UploadImage(user *models.User, fileHeader *multipart.FileHeader, alt string, ipAddress string) (*models.Image, error) {
	// 权限检查：需要 upload_image 权限
	if !userHasPermission(user, "upload_image") {
		return nil, errors.New("permission denied: upload_image required")
	}

	// 验证文件大小
	if fileHeader.Size > MaxImageSize {
		return nil, ErrImageTooLarge
	}

	// 验证文件格式
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExtensions[ext] {
		return nil, ErrInvalidImageFormat
	}

	// 验证 MIME 类型
	mimeType := fileHeader.Header.Get("Content-Type")
	if !isValidImageMimeType(mimeType) {
		return nil, ErrInvalidImageFormat
	}

	// 打开文件以获取图片尺寸
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 获取图片尺寸
	width, height, err := s.storageBackend.GetImageDimensions(file)
	if err != nil {
		return nil, fmt.Errorf("failed to get image dimensions: %w", err)
	}

	// 保存文件到存储后端
	filename, url, err := s.storageBackend.Save(fileHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 创建数据库记录
	image := &models.Image{
		Filename:       filename,
		OriginalName:   fileHeader.Filename,
		Alt:            alt,
		MimeType:       mimeType,
		Size:           fileHeader.Size,
		Width:          width,
		Height:         height,
		URL:            url,
		UserID:         user.ID,
		StorageBackend: "filesystem", // TODO: 从配置读取
		StoragePath:    filename,
		IPAddress:      ipAddress,
	}

	if err := s.imageRepo.Create(image); err != nil {
		// 如果数据库保存失败，删除已上传的文件
		_ = s.storageBackend.Delete(filename)
		return nil, fmt.Errorf("failed to save image record: %w", err)
	}

	return image, nil
}

// GetImage 获取图片详情
func (s *ImageService) GetImage(id uint) (*models.Image, error) {
	image, err := s.imageRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrImageNotFound
		}
		return nil, err
	}
	return image, nil
}

// ListImages 获取图片列表
func (s *ImageService) ListImages(offset, limit int) ([]models.Image, int64, error) {
	return s.imageRepo.List(offset, limit)
}

// UpdateImage 更新图片信息
func (s *ImageService) UpdateImage(user *models.User, id uint, alt string) (*models.Image, error) {
	image, err := s.imageRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrImageNotFound
		}
		return nil, err
	}

	// 权限检查：必须是图片所有者或拥有 edit_any_image 权限
	if image.UserID != user.ID && !userHasPermission(user, "edit_any_image") {
		return nil, errors.New("permission denied: can only edit your own images")
	}

	image.Alt = alt

	if err := s.imageRepo.Update(image); err != nil {
		return nil, fmt.Errorf("failed to update image: %w", err)
	}

	return image, nil
}

// DeleteImage 删除图片
func (s *ImageService) DeleteImage(user *models.User, id uint) error {
	// 获取图片信息以验证所有权
	image, err := s.imageRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrImageNotFound
		}
		return err
	}

	// 权限检查：必须是图片所有者或拥有 delete_any_image 权限
	if image.UserID != user.ID && !userHasPermission(user, "delete_any_image") {
		return errors.New("permission denied: can only delete your own images")
	}

	// 先删除文件
	if err := s.storageBackend.Delete(image.Filename); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// 再删除数据库记录
	if err := s.imageRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete image record: %w", err)
	}

	return nil
}

// isValidImageMimeType 验证 MIME 类型
func isValidImageMimeType(mimeType string) bool {
	validTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	return validTypes[mimeType]
}
