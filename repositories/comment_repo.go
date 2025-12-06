package repositories

import (
	"blog-backend/models"

	"gorm.io/gorm"
)

// CommentRepository persists comment entities.
type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(comment *models.Comment) error {
	return r.db.Create(comment).Error
}

func (r *CommentRepository) ListByPost(postID uint, offset, limit int) ([]models.Comment, int64, error) {
	var (
		comments []models.Comment
		total    int64
	)

	query := r.db.Model(&models.Comment{}).Where("post_id = ?", postID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Preload("User").Order("created_at ASC")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	if err := query.Find(&comments).Error; err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

func (r *CommentRepository) GetByID(id uint) (*models.Comment, error) {
	var comment models.Comment
	if err := r.db.Preload("User").First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *CommentRepository) Delete(id uint) error {
	return r.db.Delete(&models.Comment{}, id).Error
}

func (r *CommentRepository) Update(comment *models.Comment) error {
	return r.db.Save(comment).Error
}
