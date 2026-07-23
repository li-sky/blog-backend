package services

import (
	"blog-backend/models"
	"blog-backend/utils/storage"
	"errors"
	"fmt"
	stdimage "image"
	"math"
	"mime/multipart"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrImageNotFound      = errors.New("image not found")
	ErrInvalidImageFormat = errors.New("invalid image format")
	ErrImageTooLarge      = errors.New("image size exceeds limit")
	ErrInvalidVariant     = errors.New("invalid image variant")
	ErrVariantUnsupported = errors.New("storage backend does not support image variants")
)

const (
	MaxImageSize   = 50 * 1024 * 1024 // 50MB
	MaxVariantSize = 50 * 1024 * 1024
)

type ImageVariantFiles struct {
	WebP600  *multipart.FileHeader
	WebP1200 *multipart.FileHeader
	WebPFull *multipart.FileHeader
}

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
	return s.uploadImage(user, fileHeader, alt, ipAddress, ImageVariantFiles{})
}

// UploadImageWithVariants stores an original plus its browser-generated WebP
// derivatives as one logical image.
func (s *ImageService) UploadImageWithVariants(user *models.User, fileHeader *multipart.FileHeader, alt string, ipAddress string, variants ImageVariantFiles) (*models.Image, error) {
	return s.uploadImage(user, fileHeader, alt, ipAddress, variants)
}

func (s *ImageService) uploadImage(user *models.User, fileHeader *multipart.FileHeader, alt string, ipAddress string, variants ImageVariantFiles) (*models.Image, error) {
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
	if !strings.EqualFold(mimeType, "image/gif") &&
		((variants.WebP600 == nil) != (variants.WebP1200 == nil)) {
		return nil, fmt.Errorf("%w: webp600 and webp1200 must be uploaded together", ErrInvalidVariant)
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

	storedFiles := []string{filename}
	cleanup := func() {
		for _, storedFilename := range storedFiles {
			_ = s.storageBackend.Delete(storedFilename)
		}
	}

	if !isVariantExempt(image) && (variants.WebP600 != nil || variants.WebP1200 != nil || variants.WebPFull != nil) {
		variantStorage, ok := s.storageBackend.(storage.VariantStorageBackend)
		if !ok {
			cleanup()
			return nil, ErrVariantUnsupported
		}

		if variants.WebP600 != nil {
			variantFilename, variantURL, err := s.saveVariant(variantStorage, image, variants.WebP600, 600)
			if err != nil {
				cleanup()
				return nil, err
			}
			image.WebP600Filename = variantFilename
			image.WebP600URL = variantURL
			storedFiles = append(storedFiles, variantFilename)
		}
		if variants.WebP1200 != nil {
			variantFilename, variantURL, err := s.saveVariant(variantStorage, image, variants.WebP1200, 1200)
			if err != nil {
				cleanup()
				return nil, err
			}
			image.WebP1200Filename = variantFilename
			image.WebP1200URL = variantURL
			storedFiles = append(storedFiles, variantFilename)
		}
		if variants.WebPFull != nil {
			variantFilename, variantURL, err := s.saveVariant(variantStorage, image, variants.WebPFull, 0)
			if err != nil {
				cleanup()
				return nil, err
			}
			image.WebPFullFilename = variantFilename
			image.WebPFullURL = variantURL
			storedFiles = append(storedFiles, variantFilename)
		}
	}
	setVariantsComplete(image)

	if err := s.imageRepo.Create(image); err != nil {
		cleanup()
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
	setVariantsComplete(image)
	return image, nil
}

// ListImages 获取图片列表
func (s *ImageService) ListImages(offset, limit int) ([]models.Image, int64, error) {
	images, total, err := s.imageRepo.List(offset, limit)
	if err != nil {
		return nil, 0, err
	}
	for i := range images {
		setVariantsComplete(&images[i])
	}
	return images, total, nil
}

// UpdateImageVariants attaches browser-generated derivatives to a legacy image.
func (s *ImageService) UpdateImageVariants(user *models.User, id uint, variants ImageVariantFiles) (*models.Image, error) {
	image, err := s.imageRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrImageNotFound
		}
		return nil, err
	}

	if image.UserID != user.ID && !userHasPermission(user, "edit_any_image") {
		return nil, errors.New("permission denied: can only upgrade your own images")
	}
	if isVariantExempt(image) {
		setVariantsComplete(image)
		return image, nil
	}

	variantStorage, ok := s.storageBackend.(storage.VariantStorageBackend)
	if !ok {
		return nil, ErrVariantUnsupported
	}

	storedFiles := make([]string, 0, 3)
	cleanup := func() {
		for _, filename := range storedFiles {
			_ = s.storageBackend.Delete(filename)
		}
	}

	if image.WebP600URL == "" {
		if variants.WebP600 == nil {
			return nil, fmt.Errorf("%w: webp600 is required", ErrInvalidVariant)
		}
		filename, url, err := s.saveVariant(variantStorage, image, variants.WebP600, 600)
		if err != nil {
			return nil, err
		}
		image.WebP600Filename = filename
		image.WebP600URL = url
		storedFiles = append(storedFiles, filename)
	}

	if image.WebP1200URL == "" {
		if variants.WebP1200 == nil {
			cleanup()
			return nil, fmt.Errorf("%w: webp1200 is required", ErrInvalidVariant)
		}
		filename, url, err := s.saveVariant(variantStorage, image, variants.WebP1200, 1200)
		if err != nil {
			cleanup()
			return nil, err
		}
		image.WebP1200Filename = filename
		image.WebP1200URL = url
		storedFiles = append(storedFiles, filename)
	}

	if image.WebPFullURL == "" {
		if variants.WebPFull == nil {
			cleanup()
			return nil, fmt.Errorf("%w: webpFull is required", ErrInvalidVariant)
		}
		filename, url, err := s.saveVariant(variantStorage, image, variants.WebPFull, 0)
		if err != nil {
			cleanup()
			return nil, err
		}
		image.WebPFullFilename = filename
		image.WebPFullURL = url
		storedFiles = append(storedFiles, filename)
	}

	setVariantsComplete(image)
	if err := s.imageRepo.Update(image); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to save image variants: %w", err)
	}
	return image, nil
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
	for _, variantFilename := range []string{image.WebP600Filename, image.WebP1200Filename, image.WebPFullFilename} {
		if variantFilename != "" {
			if err := s.storageBackend.Delete(variantFilename); err != nil {
				return fmt.Errorf("failed to delete image variant: %w", err)
			}
		}
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

func (s *ImageService) saveVariant(backend storage.VariantStorageBackend, image *models.Image, fileHeader *multipart.FileHeader, targetWidth int) (string, string, error) {
	if fileHeader.Size <= 0 || fileHeader.Size > MaxVariantSize {
		return "", "", fmt.Errorf("%w: variant exceeds size limit", ErrInvalidVariant)
	}
	if strings.ToLower(filepath.Ext(fileHeader.Filename)) != ".webp" ||
		fileHeader.Header.Get("Content-Type") != "image/webp" {
		return "", "", fmt.Errorf("%w: variant must be WebP", ErrInvalidVariant)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", "", fmt.Errorf("%w: failed to open variant", ErrInvalidVariant)
	}
	config, format, dimensionsErr := stdimage.DecodeConfig(file)
	_ = file.Close()
	if dimensionsErr != nil {
		return "", "", fmt.Errorf("%w: cannot decode WebP: %v", ErrInvalidVariant, dimensionsErr)
	}
	if format != "webp" {
		return "", "", fmt.Errorf("%w: encoded content is not WebP", ErrInvalidVariant)
	}
	width, height := config.Width, config.Height

	if image.Width <= 0 || image.Height <= 0 {
		return "", "", fmt.Errorf("%w: original dimensions are missing", ErrInvalidVariant)
	}
	matchesStoredOrientation := matchesVariantDimensions(
		width, height, image.Width, image.Height, targetWidth,
	)
	// Go's image.DecodeConfig reports stored JPEG dimensions while browsers
	// normally apply EXIF orientation. Accept the swapped ratio for 90°/270°
	// camera images so correctly oriented browser output is not rejected.
	matchesRotatedOrientation := matchesVariantDimensions(
		width, height, image.Height, image.Width, targetWidth,
	)
	if !matchesStoredOrientation && !matchesRotatedOrientation {
		return "", "", fmt.Errorf(
			"%w: dimensions %dx%d do not match original %dx%d",
			ErrInvalidVariant,
			width,
			height,
			image.Width,
			image.Height,
		)
	}

	label := "full"
	if targetWidth > 0 {
		label = fmt.Sprintf("w%d", targetWidth)
	}
	return backend.SaveVariant(image.Filename, label, fileHeader)
}

func matchesVariantDimensions(width, height, originalWidth, originalHeight, targetWidth int) bool {
	expectedWidth := targetWidth
	if expectedWidth <= 0 || originalWidth < expectedWidth {
		expectedWidth = originalWidth
	}
	expectedHeight := int(math.Round(float64(originalHeight) * float64(expectedWidth) / float64(originalWidth)))
	return width == expectedWidth && int(math.Abs(float64(height-expectedHeight))) <= 1
}

func isVariantExempt(image *models.Image) bool {
	return image.MimeType == "image/gif"
}

func setVariantsComplete(image *models.Image) {
	image.VariantsComplete = isVariantExempt(image) ||
		(image.WebP600URL != "" && image.WebP1200URL != "" && image.WebPFullURL != "")
}
