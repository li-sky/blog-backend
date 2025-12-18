package services

import "blog-backend/models"

// Repository interfaces live in services so we can unit-test services with stubs/mocks
// without pulling in a real database.

type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uint) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	List(offset, limit int) ([]models.User, int64, error)
	Update(user *models.User) error
	Delete(id uint) error
	HasUserWithRole(roleName string) (bool, error)
}

type RoleRepository interface {
	EnsureRole(name string) (*models.Role, error)
	EnsurePermission(name, description string) (*models.Permission, error)
	ReplacePermissions(role *models.Role, permissions []models.Permission) error
	List(offset, limit int) ([]models.Role, int64, error)
	GetByID(id uint) (*models.Role, error)
	GetByName(name string) (*models.Role, error)
	GetByIDs(ids []uint) ([]models.Role, error)
	Create(role *models.Role) error
	Update(role *models.Role) error
	Delete(id uint) error
}

type PostRepository interface {
	Create(post *models.Post) error
	GetByID(id uint) (*models.Post, error)
	GetByIDAndStatus(id uint, status string) (*models.Post, error)
	List(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error)
	ListByStatus(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error)
	Update(post *models.Post) error
	Delete(id uint) error
}

type CommentRepository interface {
	Create(comment *models.Comment) error
	ListByPost(postID uint, offset, limit int) ([]models.Comment, int64, error)
	GetByID(id uint) (*models.Comment, error)
	Delete(id uint) error
	Update(comment *models.Comment) error
}

type SettingRepository interface {
	List() ([]models.Setting, error)
	GetByKey(key string) (*models.Setting, error)
	Upsert(key, value, description string) (*models.Setting, error)
}
