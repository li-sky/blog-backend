package repositories

import (
	"errors"

	"blog-backend/models"

	"gorm.io/gorm"
)

// RoleRepository manages role and permission persistence.
type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) EnsureRole(name string) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions").Where("name = ?", name).First(&role).Error
	if err == nil {
		return &role, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		role = models.Role{Name: name}
		if err := r.db.Create(&role).Error; err != nil {
			return nil, err
		}
		return &role, nil
	}
	return nil, err
}

func (r *RoleRepository) List(offset, limit int) ([]models.Role, int64, error) {
	var (
		roles []models.Role
		total int64
	)
	if err := r.db.Model(&models.Role{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query := r.db.Model(&models.Role{}).Preload("Permissions").Order("created_at DESC")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	if err := query.Find(&roles).Error; err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

func (r *RoleRepository) GetByID(id uint) (*models.Role, error) {
	var role models.Role
	if err := r.db.Preload("Permissions").First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) GetByName(name string) (*models.Role, error) {
	var role models.Role
	if err := r.db.Preload("Permissions").Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) GetByIDs(ids []uint) ([]models.Role, error) {
	var roles []models.Role
	if len(ids) == 0 {
		return roles, nil
	}
	if err := r.db.Preload("Permissions").Where("id IN ?", ids).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *RoleRepository) Create(role *models.Role) error {
	return r.db.Create(role).Error
}

func (r *RoleRepository) Update(role *models.Role) error {
	return r.db.Save(role).Error
}

func (r *RoleRepository) Delete(id uint) error {
	return r.db.Delete(&models.Role{}, id).Error
}

func (r *RoleRepository) EnsurePermission(name, description string) (*models.Permission, error) {
	var perm models.Permission
	err := r.db.Where("name = ?", name).First(&perm).Error
	if err == nil {
		if description != "" && perm.Description != description {
			perm.Description = description
			if err := r.db.Save(&perm).Error; err != nil {
				return nil, err
			}
		}
		return &perm, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		perm = models.Permission{Name: name, Description: description}
		if err := r.db.Create(&perm).Error; err != nil {
			return nil, err
		}
		return &perm, nil
	}
	return nil, err
}

func (r *RoleRepository) ReplacePermissions(role *models.Role, permissions []models.Permission) error {
	permPtrs := make([]*models.Permission, len(permissions))
	for i := range permissions {
		permPtrs[i] = &permissions[i]
	}
	if err := r.db.Model(role).Association("Permissions").Replace(permPtrs); err != nil {
		return err
	}
	role.Permissions = permissions
	return nil
}
