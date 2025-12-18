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

	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAuthController_Me_UnauthorizedAndOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ac := NewAuthController(nil)

	r := gin.New()
	r.GET("/me", ac.Me)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1)) })
	r2.GET("/me", ac.Me)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHelpers_Responders_AndBuildAuthResponseNil(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	respondValidationError(c, errors.New("bad"))
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	respondServiceError(c, errors.New("already exists"))
	require.Equal(t, http.StatusConflict, w.Code)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	respondServiceError(c, errors.New("nope"))
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	respondServiceError(c, nil)
	require.Equal(t, http.StatusBadRequest, w.Code)

	_, err := buildAuthResponse(nil)
	require.Error(t, err)

	require.Equal(t, "", derivePrimaryRole(nil))
}

func TestAuthController_InitialAdminRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")

	userRepo := userRepoCtrlStub{
		hasUserWithRole: func(roleName string) (bool, error) { return false, nil },
		getByUsername:   func(username string) (*models.User, error) { return nil, gorm.ErrRecordNotFound },
		getByEmail:      func(email string) (*models.User, error) { return nil, gorm.ErrRecordNotFound },
		create:          func(user *models.User) error { user.Model = gorm.Model{ID: 1}; return nil },
	}
	svc := services.NewUserService(userRepo, roleRepoCtrlStub{})
	ac := NewAuthController(svc)

	r := gin.New()
	r.POST("/init", ac.InitialAdminRegister)

	// validation error
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/init", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// success
	w = httptest.NewRecorder()
	body, _ := json.Marshal(RegisterRequest{Username: " admin ", Email: "admin@example.com", Password: "123456789012"})
	req = httptest.NewRequest(http.MethodPost, "/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// conflict when admin exists
	userRepo2 := userRepoCtrlStub{
		hasUserWithRole: func(roleName string) (bool, error) { return true, nil },
		getByUsername:   func(username string) (*models.User, error) { return nil, gorm.ErrRecordNotFound },
		getByEmail:      func(email string) (*models.User, error) { return nil, gorm.ErrRecordNotFound },
	}
	ac2 := NewAuthController(services.NewUserService(userRepo2, roleRepoCtrlStub{}))
	r2 := gin.New()
	r2.POST("/init", ac2.InitialAdminRegister)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestAuthController_Register_Validation_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret")

	// conflict via username already taken
	userRepo := userRepoCtrlStub{getByUsername: func(username string) (*models.User, error) {
		return &models.User{Model: gorm.Model{ID: 1}}, nil
	}}
	svc := services.NewUserService(userRepo, roleRepoCtrlStub{})
	ac := NewAuthController(svc)

	r := gin.New()
	r.POST("/reg", ac.Register)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/reg", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	body, _ := json.Marshal(RegisterRequest{Username: "u", Email: "u@example.com", Password: "123456789012"})
	req = httptest.NewRequest(http.MethodPost, "/reg", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestAuthController_Login_BindAndUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// repo returns a user, but password check fails (not a bcrypt hash)
	userRepo := userRepoCtrlStub{getByUsername: func(username string) (*models.User, error) {
		return &models.User{Model: gorm.Model{ID: 1}, Username: username, Password: "not-hash"}, nil
	}}
	svc := services.NewUserService(userRepo, roleRepoCtrlStub{})
	ac := NewAuthController(svc)

	r := gin.New()
	r.POST("/login", ac.Login)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	body, _ := json.Marshal(LoginRequest{Username: "u", Password: "wrong"})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthController_TokenGenerationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prev := generateToken
	generateToken = func(userID uint, username, role string) (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { generateToken = prev })

	userRepo := userRepoCtrlStub{
		getByUsername: func(username string) (*models.User, error) {
			// Authenticate will succeed only if CheckPassword passes; use bcrypt hash from models hook? keep it simple by calling Register flow.
			return nil, gorm.ErrRecordNotFound
		},
		getByEmail: func(email string) (*models.User, error) { return nil, gorm.ErrRecordNotFound },
		create:     func(user *models.User) error { user.Model = gorm.Model{ID: 1}; return nil },
	}
	svc := services.NewUserService(userRepo, roleRepoCtrlStub{})
	ac := NewAuthController(svc)

	r := gin.New()
	r.POST("/reg", ac.Register)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(RegisterRequest{Username: "u", Email: "u@example.com", Password: "123456789012"})
	req := httptest.NewRequest(http.MethodPost, "/reg", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthController_InitialAdminRegister_TokenFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prev := generateToken
	generateToken = func(userID uint, username, role string) (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { generateToken = prev })

	userRepo := userRepoCtrlStub{
		hasUserWithRole: func(roleName string) (bool, error) { return false, nil },
		getByUsername:   func(username string) (*models.User, error) { return nil, gorm.ErrRecordNotFound },
		getByEmail:      func(email string) (*models.User, error) { return nil, gorm.ErrRecordNotFound },
		create:          func(user *models.User) error { user.Model = gorm.Model{ID: 1}; return nil },
	}
	ac := NewAuthController(services.NewUserService(userRepo, roleRepoCtrlStub{}))

	r := gin.New()
	r.POST("/init", ac.InitialAdminRegister)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(RegisterRequest{Username: "admin", Email: "admin@example.com", Password: "123456789012"})
	req := httptest.NewRequest(http.MethodPost, "/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthController_Login_TokenFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prev := generateToken
	generateToken = func(userID uint, username, role string) (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { generateToken = prev })

	hash, err := bcrypt.GenerateFromPassword([]byte("123456789012"), bcrypt.DefaultCost)
	require.NoError(t, err)
	userRepo := userRepoCtrlStub{getByUsername: func(username string) (*models.User, error) {
		return &models.User{Model: gorm.Model{ID: 1}, Username: username, Password: string(hash)}, nil
	}}
	ac := NewAuthController(services.NewUserService(userRepo, roleRepoCtrlStub{}))

	r := gin.New()
	r.POST("/login", ac.Login)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(LoginRequest{Username: "u", Password: "123456789012"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
