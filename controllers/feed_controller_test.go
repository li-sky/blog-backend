package controllers

import (
	"blog-backend/models"
	"blog-backend/services"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type postRepoForFeedStub struct {
	listByStatus func(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error)
}

func (s postRepoForFeedStub) Create(post *models.Post) error { return nil }
func (s postRepoForFeedStub) GetByID(id uint) (*models.Post, error) {
	return nil, gorm.ErrRecordNotFound
}
func (s postRepoForFeedStub) GetByIDAndStatus(id uint, status string) (*models.Post, error) {
	return nil, gorm.ErrRecordNotFound
}
func (s postRepoForFeedStub) List(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	return nil, 0, nil
}
func (s postRepoForFeedStub) ListByStatus(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	if s.listByStatus != nil {
		return s.listByStatus(status, offset, limit, sortBy, desc)
	}
	return nil, 0, nil
}
func (s postRepoForFeedStub) Update(post *models.Post) error { return nil }
func (s postRepoForFeedStub) Delete(id uint) error           { return nil }

type settingRepoForFeedStub struct {
	getByKey func(key string) (*models.Setting, error)
	upsert   func(key, value, desc string) (*models.Setting, error)
}

func (s settingRepoForFeedStub) List() ([]models.Setting, error) { return nil, nil }
func (s settingRepoForFeedStub) GetByKey(key string) (*models.Setting, error) {
	if s.getByKey != nil {
		return s.getByKey(key)
	}
	return &models.Setting{Key: key, Value: ""}, nil
}
func (s settingRepoForFeedStub) Upsert(key, value, desc string) (*models.Setting, error) {
	if s.upsert != nil {
		return s.upsert(key, value, desc)
	}
	return &models.Setting{Key: key, Value: value, Desc: desc}, nil
}

func TestFeedController_AtomAndRSS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	posts := []models.Post{
		{Model: gorm.Model{ID: 1, UpdatedAt: time.Now()}, Title: "t", Summary: "s", Body: "b"},
		{Model: gorm.Model{ID: 2, UpdatedAt: time.Now()}, Title: "t2", Summary: "s2", Body: "b2", PublishedAt: ptrTime(time.Now().Add(-time.Hour))},
	}

	postSvcOK := services.NewPostService(postRepoForFeedStub{listByStatus: func(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
		return posts, int64(len(posts)), nil
	}})
	settingSvc := services.NewSettingService(settingRepoForFeedStub{
		getByKey: func(key string) (*models.Setting, error) {
			// force fallback path for one key
			if key == "site_title" {
				return &models.Setting{Key: key, Value: ""}, nil
			}
			return &models.Setting{Key: key, Value: "V"}, nil
		},
		upsert: func(key, value, desc string) (*models.Setting, error) {
			return &models.Setting{Key: key, Value: value}, nil
		},
	})

	fc := NewFeedController(postSvcOK, settingSvc)
	require.NotNil(t, fc)

	r := gin.New()
	r.GET("/feed/atom", func(c *gin.Context) {
		c.Request.Host = "example.com"
		fc.AtomFeed(c)
	})
	r.GET("/feed/rss", func(c *gin.Context) {
		c.Request.Host = "example.com"
		c.Request.TLS = &tls.ConnectionState{}
		fc.RSSFeed(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feed/atom", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "<feed")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/feed/rss", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "<rss")

	// cover postURL helper directly
	require.Contains(t, fc.postURL("https://x", 9), "/post/9")
}

func TestFeedController_ErrorAndGetSettingFallbackError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	postSvcErr := services.NewPostService(postRepoForFeedStub{listByStatus: func(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
		return nil, 0, errors.New("boom")
	}})
	settingSvc := services.NewSettingService(settingRepoForFeedStub{upsert: func(key, value, desc string) (*models.Setting, error) {
		return nil, errors.New("boom")
	}})
	fc := NewFeedController(postSvcErr, settingSvc)

	r := gin.New()
	r.GET("/feed/atom", func(c *gin.Context) {
		c.Request.Host = "example.com"
		fc.AtomFeed(c)
	})
	r.GET("/feed/rss", func(c *gin.Context) {
		c.Request.Host = "example.com"
		fc.RSSFeed(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feed/atom", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/feed/rss", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	// cover getSetting fallback returning default when upsert fails
	w2 := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w2)
	c.Request = httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	c.Request.Host = "example.com"
	require.Equal(t, "D", fc.getSetting("k", "D"))

	// cover getSetting returning default when GetByKey errors
	fc2 := NewFeedController(postSvcErr, services.NewSettingService(settingRepoForFeedStub{getByKey: func(key string) (*models.Setting, error) {
		return &models.Setting{Key: key, Value: "X"}, errors.New("boom")
	}}))
	require.Equal(t, "D", fc2.getSetting("k", "D"))
}

func ptrTime(t time.Time) *time.Time { return &t }
