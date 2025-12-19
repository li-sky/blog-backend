package services

import (
	"blog-backend/models"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type userRepoStub struct {
	create          func(user *models.User) error
	getByID         func(id uint) (*models.User, error)
	getByUsername   func(username string) (*models.User, error)
	getByEmail      func(email string) (*models.User, error)
	list            func(offset, limit int) ([]models.User, int64, error)
	update          func(user *models.User) error
	delete          func(id uint) error
	hasUserWithRole func(roleName string) (bool, error)
}

func (s userRepoStub) Create(user *models.User) error {
	if s.create != nil {
		return s.create(user)
	}
	return nil
}
func (s userRepoStub) GetByID(id uint) (*models.User, error) {
	if s.getByID != nil {
		return s.getByID(id)
	}
	return nil, errors.New("not found")
}
func (s userRepoStub) GetByUsername(username string) (*models.User, error) {
	if s.getByUsername != nil {
		return s.getByUsername(username)
	}
	return nil, errors.New("not found")
}
func (s userRepoStub) GetByEmail(email string) (*models.User, error) {
	if s.getByEmail != nil {
		return s.getByEmail(email)
	}
	return nil, errors.New("not found")
}
func (s userRepoStub) List(offset, limit int) ([]models.User, int64, error) {
	if s.list != nil {
		return s.list(offset, limit)
	}
	return nil, 0, nil
}
func (s userRepoStub) Update(user *models.User) error {
	if s.update != nil {
		return s.update(user)
	}
	return nil
}
func (s userRepoStub) Delete(id uint) error {
	if s.delete != nil {
		return s.delete(id)
	}
	return nil
}
func (s userRepoStub) HasUserWithRole(roleName string) (bool, error) {
	if s.hasUserWithRole != nil {
		return s.hasUserWithRole(roleName)
	}
	return false, nil
}

type roleRepoStub struct {
	ensureRole         func(name string) (*models.Role, error)
	ensurePermission   func(name, description string) (*models.Permission, error)
	replacePermissions func(role *models.Role, permissions []models.Permission) error
	getByIDs           func(ids []uint) ([]models.Role, error)
}

func (s roleRepoStub) EnsureRole(name string) (*models.Role, error) {
	if s.ensureRole != nil {
		return s.ensureRole(name)
	}
	return &models.Role{Name: name}, nil
}
func (s roleRepoStub) EnsurePermission(name, description string) (*models.Permission, error) {
	if s.ensurePermission != nil {
		return s.ensurePermission(name, description)
	}
	return &models.Permission{Name: name, Description: description}, nil
}
func (s roleRepoStub) ReplacePermissions(role *models.Role, permissions []models.Permission) error {
	if s.replacePermissions != nil {
		return s.replacePermissions(role, permissions)
	}
	role.Permissions = permissions
	return nil
}
func (s roleRepoStub) List(offset, limit int) ([]models.Role, int64, error) { return nil, 0, nil }
func (s roleRepoStub) GetByID(id uint) (*models.Role, error)                { return nil, errors.New("not found") }
func (s roleRepoStub) GetByName(name string) (*models.Role, error) {
	return nil, errors.New("not found")
}
func (s roleRepoStub) GetByIDs(ids []uint) ([]models.Role, error) {
	if s.getByIDs != nil {
		return s.getByIDs(ids)
	}
	return nil, errors.New("not found")
}
func (s roleRepoStub) Create(role *models.Role) error { return nil }
func (s roleRepoStub) Update(role *models.Role) error { return nil }
func (s roleRepoStub) Delete(id uint) error           { return nil }

func TestUserService_Register_PasswordTooShort(t *testing.T) {
	svc := NewUserService(userRepoStub{}, roleRepoStub{})
	err := svc.Register(&models.User{Username: "alice", Email: "a@example.com", Password: "short"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least 12")
}

func TestUserService_Register_UsernameTaken(t *testing.T) {
	svc := NewUserService(userRepoStub{
		getByUsername: func(username string) (*models.User, error) {
			return &models.User{Model: gorm.Model{ID: 1}, Username: username}, nil
		},
	}, roleRepoStub{})

	err := svc.Register(&models.User{Username: "alice", Email: "a@example.com", Password: "veryverysecure"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "username")
}

func TestUserService_Authenticate_InvalidPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	require.NoError(t, err)

	svc := NewUserService(userRepoStub{
		getByUsername: func(username string) (*models.User, error) {
			return &models.User{Model: gorm.Model{ID: 1}, Username: username, Password: string(hash)}, nil
		},
	}, roleRepoStub{})

	user, err := svc.Authenticate("alice", "wrong-password")
	require.Error(t, err)
	require.Nil(t, user)
}

func TestUserService_DeleteUser_CannotDeleteSelf(t *testing.T) {
	operator := &models.User{Model: gorm.Model{ID: 10}, Roles: []models.Role{{Permissions: []models.Permission{{Name: "delete_user"}}}}}
	svc := NewUserService(userRepoStub{}, roleRepoStub{})

	err := svc.DeleteUser(operator, 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot delete self")
}
