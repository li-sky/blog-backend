package controllers

import (
	"blog-backend/models"
	"blog-backend/services"
	"blog-backend/utils"
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

type commentRepoCtrlStub struct {
	create     func(comment *models.Comment) error
	listByPost func(postID uint, offset, limit int) ([]models.Comment, int64, error)
	getByID    func(id uint) (*models.Comment, error)
	delete     func(id uint) error
	update     func(comment *models.Comment) error
}

func (s commentRepoCtrlStub) Create(comment *models.Comment) error {
	if s.create != nil {
		return s.create(comment)
	}
	return nil
}
func (s commentRepoCtrlStub) ListByPost(postID uint, offset, limit int) ([]models.Comment, int64, error) {
	if s.listByPost != nil {
		return s.listByPost(postID, offset, limit)
	}
	return nil, 0, nil
}
func (s commentRepoCtrlStub) GetByID(id uint) (*models.Comment, error) {
	if s.getByID != nil {
		return s.getByID(id)
	}
	return nil, gorm.ErrRecordNotFound
}
func (s commentRepoCtrlStub) Delete(id uint) error {
	if s.delete != nil {
		return s.delete(id)
	}
	return nil
}
func (s commentRepoCtrlStub) Update(comment *models.Comment) error {
	if s.update != nil {
		return s.update(comment)
	}
	return nil
}

type postRepoForCommentsStub struct {
	getByID func(id uint) (*models.Post, error)
}

func (s postRepoForCommentsStub) Create(post *models.Post) error                                                 { return nil }
func (s postRepoForCommentsStub) GetByID(id uint) (*models.Post, error) {
	if s.getByID != nil {
		return s.getByID(id)
	}
	return &models.Post{Model: gorm.Model{ID: id}}, nil
}
func (s postRepoForCommentsStub) GetByIDAndStatus(id uint, status string) (*models.Post, error)                   { return nil, gorm.ErrRecordNotFound }
func (s postRepoForCommentsStub) List(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error)  { return nil, 0, nil }
func (s postRepoForCommentsStub) ListByStatus(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	return nil, 0, nil
}
func (s postRepoForCommentsStub) Update(post *models.Post) error { return nil }
func (s postRepoForCommentsStub) Delete(id uint) error           { return nil }

func TestNewCommentController_andToCommentResponse(t *testing.T) {
	cc := NewCommentController(services.NewCommentService(commentRepoCtrlStub{}, postRepoForCommentsStub{}))
	require.NotNil(t, cc)

	r := toCommentResponse(models.Comment{Model: gorm.Model{ID: 1}, User: models.User{}})
	require.Nil(t, r.User)

	r2 := toCommentResponse(models.Comment{Model: gorm.Model{ID: 1}, User: models.User{Model: gorm.Model{ID: 2}}})
	require.NotNil(t, r2.User)
}

func TestCommentController_List_BadParam_Error_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svcErr := services.NewCommentService(commentRepoCtrlStub{listByPost: func(postID uint, offset, limit int) ([]models.Comment, int64, error) {
		return nil, 0, errors.New("boom")
	}}, postRepoForCommentsStub{})
	ccErr := NewCommentController(svcErr)

	r := gin.New()
	r.GET("/posts/:postID/comments", ccErr.List)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/posts/nope/comments", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts/1/comments", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	svcOK := services.NewCommentService(commentRepoCtrlStub{listByPost: func(postID uint, offset, limit int) ([]models.Comment, int64, error) {
		return []models.Comment{{Model: gorm.Model{ID: 1}, PostID: postID, UserID: 2, Body: "hi", Visible: true}}, 1, nil
	}}, postRepoForCommentsStub{})
	ccOK := NewCommentController(svcOK)
		r = gin.New()
	r.GET("/posts/:postID/comments", ccOK.List)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/posts/1/comments?limit=1&offset=0", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestCommentController_Create_Delete_SetVisibility(t *testing.T) {
	gin.SetMode(gin.TestMode)
	utils.SetGeoIPBaseURLForTest("http://127.0.0.1")
	utils.SetGeoIPHTTPClientForTest(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		// Return a minimal JSON OK to avoid real network.
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"country":"CN"}`))
		return w.Result(), nil
	})})

	commentID := uint(10)

	svc := services.NewCommentService(commentRepoCtrlStub{
		create: func(comment *models.Comment) error { comment.Model = gorm.Model{ID: 1}; return nil },
		getByID: func(id uint) (*models.Comment, error) {
			return &models.Comment{Model: gorm.Model{ID: id}, UserID: 1, Visible: true}, nil
		},
		delete: func(id uint) error { return nil },
		update: func(comment *models.Comment) error { return nil },
	}, postRepoForCommentsStub{getByID: func(id uint) (*models.Post, error) {
		return &models.Post{Model: gorm.Model{ID: id}}, nil
	}})

	cc := NewCommentController(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "moderate_comments")) })
	r.POST("/posts/:postID/comments", cc.Create)
	r.DELETE("/posts/:postID/comments/:commentID", cc.Delete)
	r.PUT("/posts/:postID/comments/:commentID/visibility", cc.SetVisibility)

	// Create bind error
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Create success
	w = httptest.NewRecorder()
	body, _ := json.Marshal(createCommentRequest{Body: "hello"})
	req = httptest.NewRequest(http.MethodPost, "/posts/1/comments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Create bad post id
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/posts/nope/comments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Create service error (post not found)
	svcErr := services.NewCommentService(commentRepoCtrlStub{}, postRepoForCommentsStub{getByID: func(id uint) (*models.Post, error) {
		return nil, gorm.ErrRecordNotFound
	}})
	ccErr := NewCommentController(svcErr)
	rErr := gin.New()
	rErr.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "moderate_comments")) })
	rErr.POST("/posts/:postID/comments", ccErr.Create)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/posts/1/comments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rErr.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Delete bad param
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/posts/1/comments/nope", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Delete success
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/posts/1/comments/10", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	// SetVisibility validation error (covers respondValidationError)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/posts/1/comments/10/visibility", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// SetVisibility success
	w = httptest.NewRecorder()
	vb, _ := json.Marshal(visibilityRequest{Visible: false})
	req = httptest.NewRequest(http.MethodPut, "/posts/1/comments/10/visibility", bytes.NewReader(vb))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	_ = commentID
}

func TestCommentController_Unauthorized_And_ServiceErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewCommentService(commentRepoCtrlStub{getByID: func(id uint) (*models.Comment, error) {
		return nil, errors.New("boom")
	}}, postRepoForCommentsStub{})
	cc := NewCommentController(svc)

	r := gin.New()
	r.POST("/posts/:postID/comments", cc.Create)
	r.DELETE("/posts/:postID/comments/:commentID", cc.Delete)
	r.PUT("/posts/:postID/comments/:commentID/visibility", cc.SetVisibility)

	// Create unauthorized
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/posts/1/comments", bytes.NewBufferString(`{"body":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Delete unauthorized
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/posts/1/comments/1", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// SetVisibility unauthorized
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/posts/1/comments/1/visibility", bytes.NewBufferString(`{"visible":true}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// Delete service error
	r2 := gin.New()
	r2.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "delete_post")) })
	r2.DELETE("/posts/:postID/comments/:commentID", cc.Delete)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/posts/1/comments/1", nil)
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// SetVisibility permission denied branch
	r3 := gin.New()
	r3.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1 /* no moderate_comments */)) })
	r3.PUT("/posts/:postID/comments/:commentID/visibility", cc.SetVisibility)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/posts/1/comments/1/visibility", bytes.NewBufferString(`{"visible":true}`))
	req.Header.Set("Content-Type", "application/json")
	r3.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// SetVisibility bad commentID
	r4 := gin.New()
	r4.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "moderate_comments")) })
	r4.PUT("/posts/:postID/comments/:commentID/visibility", cc.SetVisibility)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/posts/1/comments/nope/visibility", bytes.NewBufferString(`{"visible":true}`))
	req.Header.Set("Content-Type", "application/json")
	r4.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
