package services

import (
	"blog-backend/models"
	"blog-backend/repositories"
	"errors"
	"strings"
)

// PermissionInput represents the minimal data required to attach a permission to a role.
type PermissionInput struct {
	Name        string
	Description string
}

// RoleService wraps role persistence with permission checks.
type RoleService struct {
	repo *repositories.RoleRepository
}

func NewRoleService(repo *repositories.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) ListRoles(operator *models.User, offset, limit int) ([]models.Role, int64, error) {
	if !userHasPermission(operator, "assign_roles") {
		return nil, 0, errors.New("permission denied")
	}
	return s.repo.List(offset, limit)
}

func (s *RoleService) GetRole(operator *models.User, id uint) (*models.Role, error) {
	if !userHasPermission(operator, "assign_roles") {
		return nil, errors.New("permission denied")
	}
	if id == 0 {
		return nil, errors.New("role id is required")
	}
	return s.repo.GetByID(id)
}

func (s *RoleService) CreateRole(operator *models.User, name string, permissions []PermissionInput) (*models.Role, error) {
	if !userHasPermission(operator, "assign_roles") {
		return nil, errors.New("permission denied")
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("role name is required")
	}
	role := &models.Role{Name: name}
	if err := s.repo.Create(role); err != nil {
		return nil, err
	}
	if err := s.replacePermissions(role, permissions); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RoleService) UpdateRole(operator *models.User, id uint, name string, permissions []PermissionInput) (*models.Role, error) {
	if !userHasPermission(operator, "assign_roles") {
		return nil, errors.New("permission denied")
	}
	if id == 0 {
		return nil, errors.New("role id is required")
	}
	role, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) != "" && role.Name != name {
		if role.Name == adminRoleName && name != adminRoleName {
			return nil, errors.New("cannot rename admin role")
		}
		role.Name = name
	}
	if err := s.repo.Update(role); err != nil {
		return nil, err
	}
	if permissions != nil {
		if err := s.replacePermissions(role, permissions); err != nil {
			return nil, err
		}
	}
	return role, nil
}

func (s *RoleService) DeleteRole(operator *models.User, id uint) error {
	if !userHasPermission(operator, "assign_roles") {
		return errors.New("permission denied")
	}
	if id == 0 {
		return errors.New("role id is required")
	}
	role, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if role.Name == adminRoleName {
		return errors.New("cannot delete admin role")
	}
	return s.repo.Delete(id)
}

func (s *RoleService) replacePermissions(role *models.Role, permissions []PermissionInput) error {
	resolved := make([]models.Permission, 0, len(permissions))
	for _, input := range permissions {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			continue
		}
		perm, err := s.repo.EnsurePermission(name, input.Description)
		if err != nil {
			return err
		}
		resolved = append(resolved, *perm)
	}
	return s.repo.ReplacePermissions(role, resolved)
}
