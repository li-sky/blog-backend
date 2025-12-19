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

type postRepoCtrlStub struct {
	create        func(post *models.Post) error
	getByID       func(id uint) (*models.Post, error)
	getByIDStatus func(id uint, status string) (*models.Post, error)
	list          func(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error)
	listByStatus  func(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error)
	update        func(post *models.Post) error
	delete        func(id uint) error
}

func (s postRepoCtrlStub) Create(post *models.Post) error {
	if s.create != nil {
		return s.create(post)
	}
	return nil
}
func (s postRepoCtrlStub) GetByID(id uint) (*models.Post, error) {
	if s.getByID != nil {
		return s.getByID(id)
	}
	return nil, gorm.ErrRecordNotFound
}
func (s postRepoCtrlStub) GetByIDAndStatus(id uint, status string) (*models.Post, error) {
	if s.getByIDStatus != nil {
		return s.getByIDStatus(id, status)
	}
	return nil, gorm.ErrRecordNotFound
}
func (s postRepoCtrlStub) List(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	if s.list != nil {
		return s.list(offset, limit, sortBy, desc)
	}
	return nil, 0, nil
}
func (s postRepoCtrlStub) ListByStatus(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	if s.listByStatus != nil {
		return s.listByStatus(status, offset, limit, sortBy, desc)
	}
	return nil, 0, nil
}
func (s postRepoCtrlStub) Update(post *models.Post) error {
	if s.update != nil {
		return s.update(post)
	}
	return nil
}
func (s postRepoCtrlStub) Delete(id uint) error {
	if s.delete != nil {
		return s.delete(id)
	}
	return nil
}

func controllerUser(id uint, perms ...string) *models.User {
	ps := make([]models.Permission, 0, len(perms))
	for _, p := range perms {
		ps = append(ps, models.Permission{Name: p})
	}
	return &models.User{Model: gorm.Model{ID: id}, Username: "u", Roles: []models.Role{{Permissions: ps}}}
}

func TestNewPostController(t *testing.T) {
	pc := NewPostController(services.NewPostService(postRepoCtrlStub{}))
	require.NotNil(t, pc)
}

func TestPostController_Create_Unauthorized_BadJSON_ServiceError_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Service that will succeed for create_post
	postSvc := services.NewPostService(postRepoCtrlStub{create: func(post *models.Post) error { return nil }})
	pc := NewPostController(postSvc)

	r := gin.New()
	r.POST("/posts", func(c *gin.Context) {
		pc.Create(c)
	})

	// unauthorized
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString(`{"title":"t"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// bad json
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r = gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "create_post")) })
	r.POST("/posts", pc.Create)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// service error via permission denied
	w = httptest.NewRecorder()
	payload, _ := json.Marshal(models.Post{Title: "t"})
	req = httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r = gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1 /* no perms */)) })
	r.POST("/posts", pc.Create)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// success
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/posts", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r = gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "create_post")) })
	r.POST("/posts", pc.Create)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestPostController_Update_Publish_Delete_Get_List_ListManage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	postSvc := services.NewPostService(postRepoCtrlStub{
		getByID: func(id uint) (*models.Post, error) {
			return &models.Post{Model: gorm.Model{ID: id}, Status: "draft"}, nil
		},
		getByIDStatus: func(id uint, status string) (*models.Post, error) {
			if status != "published" {
				return nil, gorm.ErrRecordNotFound
			}
			return &models.Post{Model: gorm.Model{ID: id}, Status: status}, nil
		},
		list: func(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
			return []models.Post{{Model: gorm.Model{ID: 1}}}, 1, nil
		},
		listByStatus: func(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
			return []models.Post{{Model: gorm.Model{ID: 1}, Status: status}}, 1, nil
		},
		update: func(post *models.Post) error { return nil },
		delete: func(id uint) error { return nil },
	})

	pc := NewPostController(postSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("currentUser", controllerUser(1, "edit_post", "publish_post", "delete_post"))
	})
	r.PUT("/posts/:postID", pc.Update)
	r.POST("/posts/:postID/publish", pc.Publish)
	r.DELETE("/posts/:postID", pc.Delete)
	r.GET("/posts/:postID", pc.Get)
	r.GET("/posts", pc.List)
	r.GET("/posts/manage", pc.ListManage)

	payload, _ := json.Marshal(models.Post{Title: "t"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/posts/1", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/posts/1/publish", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts?limit=1&offset=0&sort=created_at&order=desc", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts/manage", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestPostController_ErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	postSvc := services.NewPostService(postRepoCtrlStub{
		getByID:       func(id uint) (*models.Post, error) { return nil, gorm.ErrRecordNotFound },
		getByIDStatus: func(id uint, status string) (*models.Post, error) { return nil, gorm.ErrRecordNotFound },
		listByStatus: func(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
			return nil, 0, errors.New("boom")
		},
		delete: func(id uint) error { return errors.New("boom") },
		update: func(post *models.Post) error { return errors.New("boom") },
	})
	pc := NewPostController(postSvc)

	// unauthorized Update
	r := gin.New()
	r.PUT("/posts/:postID", pc.Update)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/posts/1", bytes.NewBufferString(`{"title":"t"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Update bind error
	r = gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "edit_post")) })
	r.PUT("/posts/:postID", pc.Update)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/posts/1", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// bad postID
	r = gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "edit_post")) })
	r.PUT("/posts/:postID", pc.Update)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/posts/nope", bytes.NewBufferString(`{"title":"t"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// service error on Update (repo get returns not found)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/posts/1", bytes.NewBufferString(`{"title":"t"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Publish bad id + service error
	r = gin.New()
	r.POST("/posts/:postID/publish", pc.Publish)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/posts/1/publish", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	r = gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "publish_post")) })
	r.POST("/posts/:postID/publish", pc.Publish)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/posts/nope/publish", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/posts/1/publish", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Delete bad id + service error
	r = gin.New()
	r.DELETE("/posts/:postID", pc.Delete)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	r = gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "delete_post")) })
	r.DELETE("/posts/:postID", pc.Delete)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/posts/nope", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Get bad id + not found
	r = gin.New()
	r.GET("/posts/:postID", pc.Get)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts/nope", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	// List internal error
	r = gin.New()
	r.GET("/posts", pc.List)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	// ListManage unauthorized + forbidden
	r = gin.New()
	r.GET("/posts/manage", pc.ListManage)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts/manage", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	forbiddenSvc := services.NewPostService(postRepoCtrlStub{})
	pc2 := NewPostController(forbiddenSvc)
	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1 /* no edit/publish perms */)) })
	r2.GET("/posts/manage", pc2.ListManage)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts/manage", nil)
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}
