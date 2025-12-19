package controllers

import (
	"blog-backend/models"
	"blog-backend/services"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type authUserRepoStub struct {
	userByUsername map[string]*models.User
	userByEmail    map[string]*models.User
	created        []*models.User
}

func (s *authUserRepoStub) Create(user *models.User) error {
	s.created = append(s.created, user)
	return nil
}
func (s *authUserRepoStub) GetByID(id uint) (*models.User, error) {
	return nil, errors.New("not found")
}
func (s *authUserRepoStub) GetByUsername(username string) (*models.User, error) {
	if u, ok := s.userByUsername[username]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}
func (s *authUserRepoStub) GetByEmail(email string) (*models.User, error) {
	if u, ok := s.userByEmail[email]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}
func (s *authUserRepoStub) List(offset, limit int) ([]models.User, int64, error) { return nil, 0, nil }
func (s *authUserRepoStub) Update(user *models.User) error                       { return nil }
func (s *authUserRepoStub) Delete(id uint) error                                 { return nil }
func (s *authUserRepoStub) HasUserWithRole(roleName string) (bool, error)        { return false, nil }

type authRoleRepoStub struct{}

func (authRoleRepoStub) EnsureRole(name string) (*models.Role, error) {
	return &models.Role{Name: name}, nil
}
func (authRoleRepoStub) EnsurePermission(name, description string) (*models.Permission, error) {
	return &models.Permission{Name: name, Description: description}, nil
}
func (authRoleRepoStub) ReplacePermissions(role *models.Role, permissions []models.Permission) error {
	role.Permissions = permissions
	return nil
}
func (authRoleRepoStub) List(offset, limit int) ([]models.Role, int64, error) { return nil, 0, nil }
func (authRoleRepoStub) GetByID(id uint) (*models.Role, error)                { return nil, errors.New("not found") }
func (authRoleRepoStub) GetByName(name string) (*models.Role, error) {
	return nil, errors.New("not found")
}
func (authRoleRepoStub) GetByIDs(ids []uint) ([]models.Role, error) {
	return nil, errors.New("not found")
}
func (authRoleRepoStub) Create(role *models.Role) error { return nil }
func (authRoleRepoStub) Update(role *models.Role) error { return nil }
func (authRoleRepoStub) Delete(id uint) error           { return nil }

func TestAuthController_Login_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	require.NoError(t, err)

	repo := &authUserRepoStub{userByUsername: map[string]*models.User{}}
	repo.userByUsername["alice"] = &models.User{Model: gorm.Model{ID: 1}, Username: "alice", Email: "a@example.com", Password: string(hash), Roles: []models.Role{{Name: "admin"}}}

	userSvc := services.NewUserService(repo, authRoleRepoStub{})
	controller := NewAuthController(userSvc)

	r := gin.New()
	r.POST("/login", controller.Login)

	payload := LoginRequest{Username: "alice", Password: "correct-password"}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "\"token\"")
	require.Contains(t, w.Body.String(), "\"username\":\"alice\"")
}

func TestAuthController_Register_Created(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")

	repo := &authUserRepoStub{userByUsername: map[string]*models.User{}, userByEmail: map[string]*models.User{}}
	userSvc := services.NewUserService(repo, authRoleRepoStub{})
	controller := NewAuthController(userSvc)

	r := gin.New()
	r.POST("/register", controller.Register)

	payload := RegisterRequest{Username: "alice", Email: "a@example.com", Password: "veryverysecure"}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "\"token\"")
}
