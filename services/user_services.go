package services

import (
	"blog-backend/models"
	"errors"
)

type UserService struct {
	repo     UserRepository
	roleRepo RoleRepository
}

const adminRoleName = "admin"

type permissionSeed struct {
	Name        string
	Description string
}

var adminPermissionSeeds = []permissionSeed{
	{"create_post", "Create new blog posts"},
	{"edit_post", "Edit existing blog posts"},
	{"publish_post", "Publish drafted blog posts"},
	{"delete_post", "Delete blog posts"},
	{"edit_user", "Edit user profiles"},
	{"assign_roles", "Assign roles to users"},
	{"delete_user", "Delete users"},
	{"create_admin_user", "Create additional admin users"},
	{"moderate_comments", "Moderate or hide comments"},
	{"manage_settings", "Manage site-wide settings"},
}

func userHasPermission(user *models.User, permissionName string) bool {
	if user == nil {
		return false
	}

	for _, role := range user.Roles {
		for _, permission := range role.Permissions {
			if permission.Name == permissionName {
				return true
			}
		}
	}

	return false
}

func NewUserService(repo UserRepository, roleRepo RoleRepository) *UserService {
	return &UserService{repo: repo, roleRepo: roleRepo}
}

func (s *UserService) Authenticate(username, password string) (*models.User, error) {
	user, err := s.repo.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if !user.CheckPassword(password) {
		return nil, errors.New("invalid credentials")
	}
	return user, nil
}

func (s *UserService) Register(user *models.User) error {
	if user == nil {
		return errors.New("user is required")
	}
	if _, err := s.repo.GetByUsername(user.Username); err == nil {
		return errors.New("username already taken")
	}
	if _, err := s.repo.GetByEmail(user.Email); err == nil {
		return errors.New("email already registered")
	}
	if len(user.Password) < 12 {
		return errors.New("password must be at least 12 characters long")
	}
	return s.repo.Create(user)
}

func (s *UserService) ListUsers(operator *models.User, offset, limit int) ([]models.User, int64, error) {
	if !userHasPermission(operator, "assign_roles") && !userHasPermission(operator, "edit_user") {
		return nil, 0, errors.New("permission denied")
	}
	return s.repo.List(offset, limit)
}

func (s *UserService) GetUser(operator *models.User, id uint) (*models.User, error) {
	if id == 0 {
		return nil, errors.New("user id is required")
	}
	if operator == nil || (operator.ID != id && !userHasPermission(operator, "edit_user") && !userHasPermission(operator, "assign_roles")) {
		return nil, errors.New("permission denied")
	}
	return s.repo.GetByID(id)
}

func (s *UserService) CreateAdminUser(operator *models.User, user *models.User, isInitialConfiguration bool) error {
	if user == nil {
		return errors.New("user is required")
	}
	adminRole, err := s.ensureAdminRole()
	if err != nil {
		return err
	}
	if isInitialConfiguration {
		hasAdmin, err := s.repo.HasUserWithRole(adminRoleName)
		if err != nil {
			return err
		}
		if hasAdmin {
			return errors.New("admin already exists")
		}
	} else if !userHasPermission(operator, "create_admin_user") {
		return errors.New("permission denied")
	}
	user.Roles = []models.Role{*adminRole}
	return s.Register(user)
}

func (s *UserService) UpdateUser(operator *models.User, updated *models.User) error {
	if updated == nil || updated.ID == 0 {
		return errors.New("user id is required")
	}
	if operator == nil || (operator.ID != updated.ID && !userHasPermission(operator, "edit_user")) {
		return errors.New("permission denied")
	}
	existing, err := s.repo.GetByID(updated.ID)
	if err != nil {
		return err
	}
	if updated.Username != "" {
		existing.Username = updated.Username
	}
	if updated.Email != "" {
		existing.Email = updated.Email
	}
	if updated.Password != "" {
		existing.Password = updated.Password
	}
	return s.repo.Update(existing)
}

func (s *UserService) SetUserRoles(operator *models.User, userID uint, roleIDs []uint) (*models.User, error) {
	if !userHasPermission(operator, "assign_roles") {
		return nil, errors.New("permission denied")
	}
	if userID == 0 {
		return nil, errors.New("user id is required")
	}
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	roles, err := s.roleRepo.GetByIDs(roleIDs)
	if err != nil {
		return nil, err
	}
	if len(roles) != len(roleIDs) {
		return nil, errors.New("one or more roles not found")
	}
	user.Roles = roles
	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) DeleteUser(operator *models.User, userID uint) error {
	if !userHasPermission(operator, "delete_user") {
		return errors.New("permission denied")
	}
	if userID == 0 {
		return errors.New("user id is required")
	}
	if operator != nil && operator.ID == userID {
		return errors.New("cannot delete self")
	}
	return s.repo.Delete(userID)
}

func (s *UserService) ListUserRoles(user *models.User) ([]models.Role, error) {
	return user.Roles, nil
}

func (s *UserService) GetByID(id uint) (*models.User, error) {
	if id == 0 {
		return nil, errors.New("user id is required")
	}
	return s.repo.GetByID(id)
}

func (s *UserService) ensureAdminRole() (*models.Role, error) {
	role, err := s.roleRepo.EnsureRole(adminRoleName)
	if err != nil {
		return nil, err
	}
	perms := make([]models.Permission, 0, len(adminPermissionSeeds))
	for _, seed := range adminPermissionSeeds {
		perm, err := s.roleRepo.EnsurePermission(seed.Name, seed.Description)
		if err != nil {
			return nil, err
		}
		perms = append(perms, *perm)
	}
	if err := s.roleRepo.ReplacePermissions(role, perms); err != nil {
		return nil, err
	}
	return role, nil
}
