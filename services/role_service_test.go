package services

import (
	"blog-backend/models"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type roleRepoRenameStub struct {
	role *models.Role
}

func (s roleRepoRenameStub) EnsureRole(name string) (*models.Role, error) {
	return &models.Role{Name: name}, nil
}
func (s roleRepoRenameStub) EnsurePermission(name, description string) (*models.Permission, error) {
	return &models.Permission{Name: name, Description: description}, nil
}
func (s roleRepoRenameStub) ReplacePermissions(role *models.Role, permissions []models.Permission) error {
	role.Permissions = permissions
	return nil
}
func (s roleRepoRenameStub) List(offset, limit int) ([]models.Role, int64, error) { return nil, 0, nil }
func (s roleRepoRenameStub) GetByID(id uint) (*models.Role, error) {
	if s.role == nil {
		return nil, errors.New("not found")
	}
	return s.role, nil
}
func (s roleRepoRenameStub) GetByName(name string) (*models.Role, error) {
	return nil, errors.New("not found")
}
func (s roleRepoRenameStub) GetByIDs(ids []uint) ([]models.Role, error) { return nil, nil }
func (s roleRepoRenameStub) Create(role *models.Role) error             { return nil }
func (s roleRepoRenameStub) Update(role *models.Role) error             { return nil }
func (s roleRepoRenameStub) Delete(id uint) error                       { return nil }

func TestRoleService_UpdateRole_CannotRenameAdmin(t *testing.T) {
	operator := &models.User{Model: gorm.Model{ID: 1}, Roles: []models.Role{{Permissions: []models.Permission{{Name: "assign_roles"}}}}}
	svc := NewRoleService(roleRepoRenameStub{role: &models.Role{Name: adminRoleName}})

	_, err := svc.UpdateRole(operator, 1, "not-admin", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot rename")
}
