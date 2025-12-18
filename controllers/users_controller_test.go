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
	"gorm.io/gorm"
)

type userRepoCtrlStub struct {
	list         func(offset, limit int) ([]models.User, int64, error)
	getByID      func(id uint) (*models.User, error)
	getByUsername func(username string) (*models.User, error)
	getByEmail   func(email string) (*models.User, error)
	create       func(user *models.User) error
	update       func(user *models.User) error
	delete       func(id uint) error
	hasUserWithRole func(roleName string) (bool, error)
}

func (s userRepoCtrlStub) Create(user *models.User) error {
	if s.create != nil {
		return s.create(user)
	}
	user.Model = gorm.Model{ID: 1}
	return nil
}
func (s userRepoCtrlStub) GetByID(id uint) (*models.User, error) {
	if s.getByID != nil {
		return s.getByID(id)
	}
	return &models.User{Model: gorm.Model{ID: id}, Username: "u", Email: "e@example.com"}, nil
}
func (s userRepoCtrlStub) GetByUsername(username string) (*models.User, error) {
	if s.getByUsername != nil {
		return s.getByUsername(username)
	}
	return nil, gorm.ErrRecordNotFound
}
func (s userRepoCtrlStub) GetByEmail(email string) (*models.User, error) {
	if s.getByEmail != nil {
		return s.getByEmail(email)
	}
	return nil, gorm.ErrRecordNotFound
}
func (s userRepoCtrlStub) List(offset, limit int) ([]models.User, int64, error) {
	if s.list != nil {
		return s.list(offset, limit)
	}
	return []models.User{{Model: gorm.Model{ID: 1}, Username: "a", Email: "a@example.com"}}, 1, nil
}
func (s userRepoCtrlStub) Update(user *models.User) error {
	if s.update != nil {
		return s.update(user)
	}
	return nil
}
func (s userRepoCtrlStub) Delete(id uint) error {
	if s.delete != nil {
		return s.delete(id)
	}
	return nil
}
func (s userRepoCtrlStub) HasUserWithRole(roleName string) (bool, error) {
	if s.hasUserWithRole != nil {
		return s.hasUserWithRole(roleName)
	}
	return false, nil
}

func TestUserController_AllHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	roleRepo := roleRepoCtrlStub{}
	userRepo := userRepoCtrlStub{
		getByID: func(id uint) (*models.User, error) {
			return &models.User{Model: gorm.Model{ID: id}, Username: "u", Email: "u@example.com"}, nil
		},
		update: func(user *models.User) error { return nil },
		delete: func(id uint) error { return nil },
		create: func(user *models.User) error { user.Model = gorm.Model{ID: 99}; return nil },
	}
	svc := services.NewUserService(userRepo, roleRepo)
	uc := NewUserController(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles", "edit_user", "delete_user", "create_admin_user")) })
	r.GET("/users", uc.List)
	r.GET("/users/:id", uc.Get)
	r.PUT("/users/:id", uc.Update)
	r.PUT("/users/:id/roles", uc.SetRoles)
	r.DELETE("/users/:id", uc.Delete)
	r.POST("/users/admin", uc.CreateAdmin)

	// List ok
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users?limit=10&offset=0", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Get bad id
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/users/nope", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Get ok
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/users/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Update bind error
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Update ok
	w = httptest.NewRecorder()
	ub, _ := json.Marshal(updateUserRequest{Username: "x", Email: "x@example.com"})
	req = httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewReader(ub))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// SetRoles ok
	w = httptest.NewRecorder()
	rb, _ := json.Marshal(setRolesRequest{RoleIDs: []uint{1}})
	req = httptest.NewRequest(http.MethodPut, "/users/1/roles", bytes.NewReader(rb))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Delete ok
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/users/2", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	// CreateAdmin validation error
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/users/admin", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// CreateAdmin ok
	w = httptest.NewRecorder()
	ab, _ := json.Marshal(RegisterRequest{Username: " admin ", Email: "admin@example.com", Password: "123456789012"})
	req = httptest.NewRequest(http.MethodPost, "/users/admin", bytes.NewReader(ab))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestUserController_Unauthorized_And_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewUserService(userRepoCtrlStub{list: func(offset, limit int) ([]models.User, int64, error) {
		return nil, 0, errors.New("boom")
	}}, roleRepoCtrlStub{})
	uc := NewUserController(svc)

	r := gin.New()
	r.GET("/users", uc.List)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles")) })
	r2.GET("/users", uc.List)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/users", nil)
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserController_ParseAndBindErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewUserService(userRepoCtrlStub{}, roleRepoCtrlStub{})
	uc := NewUserController(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles", "edit_user", "delete_user")) })
	r.GET("/users/:id", uc.Get)
	r.PUT("/users/:id", uc.Update)
	r.PUT("/users/:id/roles", uc.SetRoles)
	r.DELETE("/users/:id", uc.Delete)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/nope", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/users/nope", bytes.NewBufferString(`{"username":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/users/1/roles", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/users/nope", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserController_PermissionDeniedBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewUserService(userRepoCtrlStub{getByID: func(id uint) (*models.User, error) {
		return &models.User{Model: gorm.Model{ID: id}, Username: "u"}, nil
	}}, roleRepoCtrlStub{})
	uc := NewUserController(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1 /* no perms */)) })
	r.GET("/users/:id", uc.Get)
	r.PUT("/users/:id", uc.Update)
	r.PUT("/users/:id/roles", uc.SetRoles)
	r.DELETE("/users/:id", uc.Delete)

	// Get permission denied (requesting other user)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/2", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Update permission denied
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/users/2", bytes.NewBufferString(`{"username":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// SetRoles permission denied
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/users/2/roles", bytes.NewBufferString(`{"roleIds":[1]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Delete permission denied
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/users/2", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserController_Delete_CannotDeleteSelf(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewUserService(userRepoCtrlStub{delete: func(id uint) error { return nil }}, roleRepoCtrlStub{})
	uc := NewUserController(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "delete_user")) })
	r.DELETE("/users/:id", uc.Delete)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserController_SetRoles_BadID_And_RoleNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// bad id
	uc := NewUserController(services.NewUserService(userRepoCtrlStub{}, roleRepoCtrlStub{}))
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles")) })
	r.PUT("/users/:id/roles", uc.SetRoles)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/users/nope/roles", bytes.NewBufferString(`{"roleIds":[1]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// role not found
	roleRepo := roleRepoCtrlStub{getByIDs: func(ids []uint) ([]models.Role, error) { return []models.Role{}, nil }}
	svc := services.NewUserService(userRepoCtrlStub{getByID: func(id uint) (*models.User, error) { return &models.User{Model: gorm.Model{ID: id}}, nil }}, roleRepo)
	uc2 := NewUserController(svc)
	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles")) })
	r2.PUT("/users/:id/roles", uc2.SetRoles)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/users/1/roles", bytes.NewBufferString(`{"roleIds":[1]}`))
	req.Header.Set("Content-Type", "application/json")
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserController_Unauthorized_OtherHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uc := NewUserController(services.NewUserService(userRepoCtrlStub{}, roleRepoCtrlStub{}))
	r := gin.New()
	r.GET("/users/:id", uc.Get)
	r.PUT("/users/:id", uc.Update)
	r.PUT("/users/:id/roles", uc.SetRoles)
	r.DELETE("/users/:id", uc.Delete)
	r.POST("/users/admin", uc.CreateAdmin)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewBufferString(`{"username":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/users/1/roles", bytes.NewBufferString(`{"roleIds":[1]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/users/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/users/admin", bytes.NewBufferString(`{"username":"u","email":"u@example.com","password":"123456789012"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserController_CreateAdmin_PermissionDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uc := NewUserController(services.NewUserService(userRepoCtrlStub{getByUsername: func(username string) (*models.User, error) { return nil, gorm.ErrRecordNotFound }, getByEmail: func(email string) (*models.User, error) { return nil, gorm.ErrRecordNotFound }}, roleRepoCtrlStub{}))
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1 /* no create_admin_user */)) })
	r.POST("/users/admin", uc.CreateAdmin)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(RegisterRequest{Username: "u", Email: "u@example.com", Password: "123456789012"})
	req := httptest.NewRequest(http.MethodPost, "/users/admin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserController_CreateAdmin_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewUserService(userRepoCtrlStub{
		getByUsername: func(username string) (*models.User, error) { return &models.User{Model: gorm.Model{ID: 1}}, nil },
	}, roleRepoCtrlStub{})
	uc := NewUserController(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "create_admin_user")) })
	r.POST("/users/admin", uc.CreateAdmin)

	w := httptest.NewRecorder()
	ab, _ := json.Marshal(RegisterRequest{Username: "admin", Email: "admin@example.com", Password: "123456789012"})
	req := httptest.NewRequest(http.MethodPost, "/users/admin", bytes.NewReader(ab))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}
