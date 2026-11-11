package repositories

import (
	"blog-backend/models"

	"gorm.io/gorm"
)

type ImageRepo struct {
	DB *gorm.DB
}

func NewImageRepo(db *gorm.DB) *ImageRepo {
	return &ImageRepo{DB: db}
}

func (r *ImageRepo) Create(image *models.Image) error {
	return r.DB.Create(image).Error
}

func (r *ImageRepo) GetByID(id uint) (*models.Image, error) {
	var image models.Image
	err := r.DB.Preload("User").First(&image, id).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

func (r *ImageRepo) List(offset, limit int) ([]models.Image, int64, error) {
	var images []models.Image
	var total int64

	if err := r.DB.Model(&models.Image{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.DB.Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&images).Error

	return images, total, err
}

func (r *ImageRepo) Update(image *models.Image) error {
	return r.DB.Save(image).Error
}

func (r *ImageRepo) Delete(id uint) error {
	return r.DB.Delete(&models.Image{}, id).Error
}

func (r *ImageRepo) GetByFilename(filename string) (*models.Image, error) {
	var image models.Image
	err := r.DB.Where("filename = ?", filename).First(&image).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// ListImageReferencePosts reads both public and editing snapshots without
// exposing article bodies through the image management API.
func (r *ImageRepo) ListImageReferencePosts() ([]models.Post, error) {
	var posts []models.Post
	err := r.DB.Select("id", "title", "summary", "status", "body", "draft", "user_id").Order("id ASC").Find(&posts).Error
	return posts, err
}
