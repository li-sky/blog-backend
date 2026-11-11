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
	payload, _ := json.Marshal(map[string]any{"title": "t", "body": "body", "version": 0})
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
			return &models.Post{
				Model:  gorm.Model{ID: id},
				Status: "draft", Title: "t", Body: "body", PublicationTimeKnown: true,
				UserID: 2,
				User: models.User{
					Model:    gorm.Model{ID: 2},
					Username: "author",
					Email:    "author@example.com",
				},
			}, nil
		},
		getByIDStatus: func(id uint, status string) (*models.Post, error) {
			if status != "published" {
				return nil, gorm.ErrRecordNotFound
			}
			return &models.Post{
				Model:  gorm.Model{ID: id},
				Status: status,
				UserID: 2,
				User: models.User{
					Model:    gorm.Model{ID: 2},
					Username: "author",
					Email:    "author@example.com",
				},
			}, nil
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

	payload, _ := json.Marshal(map[string]any{"title": "t", "body": "body", "version": 0})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/posts/1", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/posts/1/publish", bytes.NewBufferString(`{"version":0}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var detailPayload map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detailPayload))
	userField, ok := detailPayload["user"].(map[string]any)
	require.True(t, ok)
	_, hasEmail := userField["email"]
	require.False(t, hasEmail)
	require.Equal(t, emailSHA256("author@example.com"), userField["emailSha256"])
	require.Equal(t, float64(2), userField["id"])

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts?limit=1&offset=0&sort=created_at&order=desc", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var listPayload map[string]any
	listPayload = map[string]any{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listPayload))
	items, ok := listPayload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	entry, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(1), entry["id"])
	_, hasBody := entry["body"]
	require.False(t, hasBody)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts/manage", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	listPayload = map[string]any{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listPayload))
	items, ok = listPayload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	entry, ok = items[0].(map[string]any)
	require.True(t, ok)
	_, hasBody = entry["body"]
	require.False(t, hasBody)
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

	// An unknown post still returns not found on the original update endpoint
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/posts/1", bytes.NewBufferString(`{"title":"t"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

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
	require.Equal(t, http.StatusNotFound, w.Code)

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

func TestPostController_Get_Permissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	postSvc := services.NewPostService(postRepoCtrlStub{
		getByID: func(id uint) (*models.Post, error) {
			// Simulate different posts based on ID
			if id == 1 {
				return &models.Post{Model: gorm.Model{ID: 1}, Status: "draft", Title: "Draft Post", UserID: 99}, nil
			}
			if id == 2 {
				return &models.Post{Model: gorm.Model{ID: 2}, Status: "published", Title: "Published Post", UserID: 99}, nil
			}
			if id == 3 {
				return &models.Post{Model: gorm.Model{ID: 3}, Status: "draft", Title: "My Draft Post", UserID: 2}, nil
			}
			return nil, errors.New("record not found")
		},
	})

	pc := NewPostController(postSvc)

	// Case 1: User with edit_post permission should get Draft post (ID 1)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("currentUser", controllerUser(1, "edit_post"))
	})
	r.GET("/posts/:postID", pc.Get)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.Equal(t, "Draft Post", resp["title"])

	// Case 2: User without edit_post permission should NOT get Draft post (ID 1)
	r2 := gin.New()
	r2.Use(func(c *gin.Context) {
		c.Set("currentUser", controllerUser(2))
	})
	r2.GET("/posts/:postID", pc.Get)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	r2.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusNotFound, w2.Code)

	// Case 3: User without edit_post permission should get Published Post (ID 2)
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/posts/2", nil)
	r2.ServeHTTP(w3, req3)
	require.Equal(t, http.StatusOK, w3.Code)
	var resp3 map[string]any
	json.Unmarshal(w3.Body.Bytes(), &resp3)
	require.Equal(t, "Published Post", resp3["title"])

	// Case 4: Author should get their own Draft Post (ID 3)
	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/posts/3", nil)
	r2.ServeHTTP(w4, req4)
	require.Equal(t, http.StatusOK, w4.Code)
	var resp4 map[string]any
	json.Unmarshal(w4.Body.Bytes(), &resp4)
	require.Equal(t, "My Draft Post", resp4["title"])

	// Case 5: Anonymous user should get Published Post (ID 2) but not Draft (ID 1)
	r3 := gin.New()
	r3.GET("/posts/:postID", pc.Get)

	w5 := httptest.NewRecorder()
	req5 := httptest.NewRequest(http.MethodGet, "/posts/2", nil)
	r3.ServeHTTP(w5, req5)
	require.Equal(t, http.StatusOK, w5.Code)

	w6 := httptest.NewRecorder()
	req6 := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	r3.ServeHTTP(w6, req6)
	require.Equal(t, http.StatusNotFound, w6.Code)
}
