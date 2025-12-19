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

type roleRepoCtrlStub struct {
	list               func(offset, limit int) ([]models.Role, int64, error)
	get                func(id uint) (*models.Role, error)
	getByIDs           func(ids []uint) ([]models.Role, error)
	create             func(role *models.Role) error
	update             func(role *models.Role) error
	delete             func(id uint) error
	ensurePermission   func(name, description string) (*models.Permission, error)
	replacePermissions func(role *models.Role, permissions []models.Permission) error
}

func (s roleRepoCtrlStub) EnsureRole(name string) (*models.Role, error) {
	return &models.Role{Name: name}, nil
}
func (s roleRepoCtrlStub) EnsurePermission(name, description string) (*models.Permission, error) {
	if s.ensurePermission != nil {
		return s.ensurePermission(name, description)
	}
	return &models.Permission{Name: name, Description: description}, nil
}
func (s roleRepoCtrlStub) ReplacePermissions(role *models.Role, permissions []models.Permission) error {
	if s.replacePermissions != nil {
		return s.replacePermissions(role, permissions)
	}
	role.Permissions = permissions
	return nil
}
func (s roleRepoCtrlStub) List(offset, limit int) ([]models.Role, int64, error) {
	if s.list != nil {
		return s.list(offset, limit)
	}
	return nil, 0, nil
}
func (s roleRepoCtrlStub) GetByID(id uint) (*models.Role, error) {
	if s.get != nil {
		return s.get(id)
	}
	return nil, gorm.ErrRecordNotFound
}
func (s roleRepoCtrlStub) GetByName(name string) (*models.Role, error) {
	return nil, gorm.ErrRecordNotFound
}
func (s roleRepoCtrlStub) GetByIDs(ids []uint) ([]models.Role, error) {
	if s.getByIDs != nil {
		return s.getByIDs(ids)
	}
	roles := make([]models.Role, 0, len(ids))
	for _, id := range ids {
		roles = append(roles, models.Role{Model: gorm.Model{ID: id}, Name: "r"})
	}
	return roles, nil
}
func (s roleRepoCtrlStub) Create(role *models.Role) error {
	if s.create != nil {
		return s.create(role)
	}
	return nil
}
func (s roleRepoCtrlStub) Update(role *models.Role) error {
	if s.update != nil {
		return s.update(role)
	}
	return nil
}
func (s roleRepoCtrlStub) Delete(id uint) error {
	if s.delete != nil {
		return s.delete(id)
	}
	return nil
}

func TestRoleController_CRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := roleRepoCtrlStub{
		list: func(offset, limit int) ([]models.Role, int64, error) {
			return []models.Role{{Model: gorm.Model{ID: 1}, Name: "r"}}, 1, nil
		},
		get:    func(id uint) (*models.Role, error) { return &models.Role{Model: gorm.Model{ID: id}, Name: "r"}, nil },
		delete: func(id uint) error { return nil },
		create: func(role *models.Role) error { role.Model = gorm.Model{ID: 2}; return nil },
		update: func(role *models.Role) error { return nil },
	}

	svc := services.NewRoleService(repo)
	rc := NewRoleController(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles")) })
	r.GET("/roles", rc.List)
	r.GET("/roles/:id", rc.Get)
	r.POST("/roles", rc.Create)
	r.PUT("/roles/:id", rc.Update)
	r.DELETE("/roles/:id", rc.Delete)

	// List
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/roles", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Get bad param
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/roles/nope", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Get ok
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/roles/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Create validation error
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/roles", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Create ok
	w = httptest.NewRecorder()
	body, _ := json.Marshal(roleCreateRequest{Name: "x", Permissions: []permissionPayload{{Name: "p"}}})
	req = httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Update ok
	w = httptest.NewRecorder()
	ub, _ := json.Marshal(roleUpdateRequest{Name: "x"})
	req = httptest.NewRequest(http.MethodPut, "/roles/1", bytes.NewReader(ub))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Delete ok
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/roles/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	// Cover toPermissionInputs
	inputs := toPermissionInputs([]permissionPayload{{Name: "a", Description: "b"}})
	require.Len(t, inputs, 1)
}

func TestRoleController_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewRoleService(roleRepoCtrlStub{list: func(offset, limit int) ([]models.Role, int64, error) {
		return nil, 0, errors.New("already exists")
	}})
	rc := NewRoleController(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles")) })
	r.GET("/roles", rc.List)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/roles", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestRoleController_Unauthorized_And_DeleteError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewRoleService(roleRepoCtrlStub{delete: func(id uint) error { return errors.New("boom") }})
	rc := NewRoleController(svc)

	r := gin.New()
	r.GET("/roles", rc.List)
	r.DELETE("/roles/:id", rc.Delete)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/roles", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/roles/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles")) })
	r2.DELETE("/roles/:id", rc.Delete)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/roles/1", nil)
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleController_Get_Create_Update_ErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewRoleService(roleRepoCtrlStub{get: func(id uint) (*models.Role, error) {
		return nil, errors.New("boom")
	}})
	rc := NewRoleController(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles")) })
	r.GET("/roles/:id", rc.Get)
	r.POST("/roles", rc.Create)
	r.PUT("/roles/:id", rc.Update)

	// Get bad id
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/roles/nope", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Get service error
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/roles/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Create validation error
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/roles", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Create service error (permission denied)
	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1 /* no assign_roles */)) })
	r2.POST("/roles", rc.Create)
	w = httptest.NewRecorder()
	body, _ := json.Marshal(roleCreateRequest{Name: "x", Permissions: []permissionPayload{{Name: "p"}}})
	req = httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Update validation error
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/roles/1", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleController_Unauthorized_GetCreateUpdateAnd_DeleteBadID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rc := NewRoleController(services.NewRoleService(roleRepoCtrlStub{}))
	r := gin.New()
	r.GET("/roles/:id", rc.Get)
	r.POST("/roles", rc.Create)
	r.PUT("/roles/:id", rc.Update)
	r.DELETE("/roles/:id", rc.Delete)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/roles/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/roles", bytes.NewBufferString(`{"name":"x","permissions":[]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/roles/1", bytes.NewBufferString(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/roles/nope", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles")) })
	r2.DELETE("/roles/:id", rc.Delete)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/roles/nope", nil)
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleController_Update_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewRoleService(roleRepoCtrlStub{
		get:    func(id uint) (*models.Role, error) { return &models.Role{Model: gorm.Model{ID: id}, Name: "r"}, nil },
		update: func(role *models.Role) error { return errors.New("boom") },
	})
	rc := NewRoleController(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles")) })
	r.PUT("/roles/:id", rc.Update)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(roleUpdateRequest{Name: "x"})
	req := httptest.NewRequest(http.MethodPut, "/roles/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleController_Update_BadID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rc := NewRoleController(services.NewRoleService(roleRepoCtrlStub{}))
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "assign_roles")) })
	r.PUT("/roles/:id", rc.Update)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/roles/nope", bytes.NewBufferString(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
