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

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")

	userSvc := services.NewUserService(stubUserRepo{getByID: func(id uint) (*models.User, error) {
		return &models.User{Model: gorm.Model{ID: id}}, nil
	}}, stubRoleRepo{})

	r := gin.New()
	r.GET("/p", AuthMiddleware(userSvc), func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_UserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")

	token, err := utils.GenerateToken(9, "x", "admin")
	require.NoError(t, err)

	userSvc := services.NewUserService(stubUserRepo{getByID: func(id uint) (*models.User, error) {
		return nil, gorm.ErrRecordNotFound
	}}, stubRoleRepo{})

	r := gin.New()
	r.GET("/p", AuthMiddleware(userSvc), func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_TokenFromQueryAndCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")

	token, err := utils.GenerateToken(9, "x", "admin")
	require.NoError(t, err)

	userSvc := services.NewUserService(stubUserRepo{getByID: func(id uint) (*models.User, error) {
		return &models.User{Model: gorm.Model{ID: id}}, nil
	}}, stubRoleRepo{})

	r := gin.New()
	r.GET("/q", AuthMiddleware(userSvc), func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/c", AuthMiddleware(userSvc), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/q?token="+token, nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/c", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
