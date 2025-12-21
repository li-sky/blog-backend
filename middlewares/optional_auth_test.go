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

func TestOptionalAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")

	userSvc := services.NewUserService(stubUserRepo{getByID: func(id uint) (*models.User, error) {
		if id == 9 {
			return &models.User{Model: gorm.Model{ID: id}, Username: "found"}, nil
		}
		return nil, gorm.ErrRecordNotFound
	}}, stubRoleRepo{})

	r := gin.New()
	r.GET("/opt", OptionalAuthMiddleware(userSvc), func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if ok {
			c.JSON(http.StatusOK, gin.H{"user": user.Username})
		} else {
			c.JSON(http.StatusOK, gin.H{"user": "anonymous"})
		}
	})

	// Case 1: No token -> Anonymous
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/opt", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"user": "anonymous"}`, w.Body.String())

	// Case 2: Invalid token -> Anonymous
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/opt", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"user": "anonymous"}`, w.Body.String())

	// Case 3: Valid token, User found -> Authenticated
	token, _ := utils.GenerateToken(9, "found", "admin")
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/opt", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"user": "found"}`, w.Body.String())

	// Case 4: Valid token, User NOT found -> Anonymous
	token2, _ := utils.GenerateToken(99, "missing", "admin")
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/opt", nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"user": "anonymous"}`, w.Body.String())
}
