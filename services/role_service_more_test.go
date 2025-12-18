package services

import (
	"blog-backend/models"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRoleService_PermissionDeniedPaths(t *testing.T) {
	svc := NewRoleService(roleRepoRenameStub{role: &models.Role{Model: gorm.Model{ID: 1}, Name: "x"}})

	_, _, err := svc.ListRoles(permUser(1), 0, 10)
	require.Error(t, err)

	_, err = svc.GetRole(permUser(1), 1)
	require.Error(t, err)

	_, err = svc.CreateRole(permUser(1), "x", nil)
	require.Error(t, err)

	_, err = svc.UpdateRole(permUser(1), 1, "x", nil)
	require.Error(t, err)

	err = svc.DeleteRole(permUser(1), 1)
	require.Error(t, err)
}

func TestRoleService_CRUD_SuccessAndValidation(t *testing.T) {
	operator := permUser(1, "assign_roles")

	created := &models.Role{}
	repo := roleRepoRenameStub{role: &models.Role{Model: gorm.Model{ID: 1}, Name: "editor"}}
	// Override methods via a full stub implementation.
	repo2 := struct{ roleRepoRenameStub }{repo}
	_ = repo2

	stub := roleRepoStub{
		ensurePermission: func(name, description string) (*models.Permission, error) {
			return &models.Permission{Name: name, Description: description}, nil
		},
		replacePermissions: func(role *models.Role, permissions []models.Permission) error {
			role.Permissions = permissions
			return nil
		},
	}

	// Create
	svcCreate := NewRoleService(stub)
	role, err := svcCreate.CreateRole(operator, "editor", []PermissionInput{{Name: "p1"}})
	require.NoError(t, err)
	created = role
	require.Equal(t, "editor", created.Name)

	// Update validation: id required
	_, err = svcCreate.UpdateRole(operator, 0, "x", nil)
	require.Error(t, err)

	// Delete validation: id required
	err = svcCreate.DeleteRole(operator, 0)
	require.Error(t, err)

	// replacePermissions skip empty
	stub2 := roleRepoStub{ensurePermission: func(name, description string) (*models.Permission, error) {
		return nil, errors.New("boom")
	}}
	svcErr := NewRoleService(stub2)
	_, err = svcErr.CreateRole(operator, "x", []PermissionInput{{Name: "p"}})
	require.Error(t, err)
}
