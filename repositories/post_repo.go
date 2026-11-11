package repositories

import (
	"blog-backend/services"
	"fmt"

	"blog-backend/models"

	"gorm.io/gorm"
)

// PostRepository handles persistence for blog posts.
type PostRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) Create(post *models.Post) error {
	return r.db.Create(post).Error
}

func (r *PostRepository) GetByID(id uint) (*models.Post, error) {
	var post models.Post
	if err := r.db.Preload("User").First(&post, id).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) GetByIDAndStatus(id uint, status string) (*models.Post, error) {
	var post models.Post
	if err := r.db.Preload("User").Where("id = ? AND status = ?", id, status).First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) List(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	return r.listWithQuery(r.db.Model(&models.Post{}), offset, limit, sortBy, desc)
}

func (r *PostRepository) ListByStatus(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	base := r.db.Model(&models.Post{}).Where("status = ?", status)
	return r.listWithQuery(base, offset, limit, sortBy, desc)
}

func (r *PostRepository) Update(post *models.Post) error {
	return r.db.Save(post).Error
}

func (r *PostRepository) Delete(id uint) error {
	return r.db.Delete(&models.Post{}, id).Error
}

func (r *PostRepository) listWithQuery(base *gorm.DB, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	var (
		posts []models.Post
		total int64
	)

	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderExpr := buildPostOrder(sortBy, desc)
	query := base.Preload("User").Order(orderExpr)
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	if err := query.Find(&posts).Error; err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func buildPostOrder(sortBy string, desc bool) string {
	allowed := map[string]string{
		"title":        "title",
		"created_at":   "created_at",
		"updated_at":   "updated_at",
		"published_at": "published_at",
	}

	column, ok := allowed[sortBy]
	if !ok {
		column = "created_at"
	}

	direction := "ASC"
	if desc {
		direction = "DESC"
	}

	if column == "published_at" {
		return fmt.Sprintf("COALESCE(published_at, published_updated_at) %s NULLS LAST, id %s", direction, direction)
	}
	return fmt.Sprintf("%s %s", column, direction)
}

// UpdateVersioned atomically changes one author revision. A publish and an autosave
// cannot overwrite each other: both must compare the same persisted version.
func (r *PostRepository) UpdateVersioned(post *models.Post, expected uint64) error {
	result := r.db.Model(&models.Post{}).Where("id = ? AND draft_version = ?", post.ID, expected).
		Select("Title", "Summary", "Body", "Status", "PublishedAt", "PublishedUpdatedAt", "Draft", "DraftVersion", "DraftUpdatedAt", "UpdatedAt").Updates(post)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return services.ErrDraftConflict
	}
	return nil
}

func (r *PostRepository) GetByCreateRequest(userID uint, key string) (*models.Post, error) {
	var post models.Post
	if err := r.db.Where("user_id = ? AND client_request_id = ?", userID, key).First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}
