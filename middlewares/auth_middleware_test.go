package middlewares

import (
	"blog-backend/models"
	"blog-backend/services"
	"blog-backend/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type stubUserRepo struct {
	getByID func(id uint) (*models.User, error)
}

func (s stubUserRepo) Create(user *models.User) error                                { return nil }
func (s stubUserRepo) GetByID(id uint) (*models.User, error)                          { return s.getByID(id) }
func (s stubUserRepo) GetByUsername(username string) (*models.User, error)            { return nil, nil }
func (s stubUserRepo) GetByEmail(email string) (*models.User, error)                  { return nil, nil }
func (s stubUserRepo) List(offset, limit int) ([]models.User, int64, error)           { return nil, 0, nil }
func (s stubUserRepo) Update(user *models.User) error                                 { return nil }
func (s stubUserRepo) Delete(id uint) error                                            { return nil }
func (s stubUserRepo) HasUserWithRole(roleName string) (bool, error)                   { return false, nil }

type stubRoleRepo struct{}

func (stubRoleRepo) EnsureRole(name string) (*models.Role, error)                                   { return &models.Role{Name: name}, nil }
func (stubRoleRepo) EnsurePermission(name, description string) (*models.Permission, error)          { return &models.Permission{Name: name, Description: description}, nil }
func (stubRoleRepo) ReplacePermissions(role *models.Role, permissions []models.Permission) error    { role.Permissions = permissions; return nil }
func (stubRoleRepo) List(offset, limit int) ([]models.Role, int64, error)                           { return nil, 0, nil }
func (stubRoleRepo) GetByID(id uint) (*models.Role, error)                                          { return nil, nil }
func (stubRoleRepo) GetByName(name string) (*models.Role, error)                                    { return nil, nil }
func (stubRoleRepo) GetByIDs(ids []uint) ([]models.Role, error)                                     { return nil, nil }
func (stubRoleRepo) Create(role *models.Role) error                                                 { return nil }
func (stubRoleRepo) Update(role *models.Role) error                                                 { return nil }
func (stubRoleRepo) Delete(id uint) error                                                            { return nil }

func TestAuthMiddleware_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userSvc := services.NewUserService(stubUserRepo{getByID: func(id uint) (*models.User, error) {
		return &models.User{Username: "x"}, nil
	}}, stubRoleRepo{})

	r := gin.New()
	r.GET("/protected", AuthMiddleware(userSvc), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_SetsContextUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Ensure JWT secret is stable for tests.
	t.Setenv("JWT_SECRET", "test-secret")

	userSvc := services.NewUserService(stubUserRepo{getByID: func(id uint) (*models.User, error) {
		require.Equal(t, uint(99), id)
		return &models.User{Model: gorm.Model{ID: 99}, Username: "alice"}, nil
	}}, stubRoleRepo{})

	token, err := utils.GenerateToken(99, "alice", "admin")
	require.NoError(t, err)

	r := gin.New()
	r.GET("/protected", AuthMiddleware(userSvc), func(c *gin.Context) {
		user, ok := CurrentUser(c)
		require.True(t, ok)
		require.Equal(t, "alice", user.Username)

		claims, ok := ClaimsFromContext(c)
		require.True(t, ok)
		require.Equal(t, uint(99), claims.UserID)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestExtractToken_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/t", func(c *gin.Context) {
		c.String(http.StatusOK, extractToken(c))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/t?token=querytoken", nil)
	req.Header.Set("Authorization", "Bearer headertoken")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "headertoken", w.Body.String())
}
