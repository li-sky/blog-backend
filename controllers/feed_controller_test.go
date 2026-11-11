package controllers

import (
	"blog-backend/models"
	"blog-backend/services"
	"crypto/tls"
	"encoding/xml"
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

func TestFeedPublicationDatesIgnoreDraftSaves(t *testing.T) {
	gin.SetMode(gin.TestMode)
	first := time.Date(2025, 12, 1, 8, 30, 0, 0, time.UTC)
	updated := first.Add(24 * time.Hour)
	databaseSave := updated.Add(200 * time.Hour)
	posts := []models.Post{{Model: gorm.Model{ID: 7, CreatedAt: first.Add(-72 * time.Hour), UpdatedAt: databaseSave}, Status: "published", Title: "Live title", Summary: "Live summary", Body: "Live body", PublishedAt: &first, PublishedUpdatedAt: &updated, PublicationTimeKnown: true}}
	controller := NewFeedController(nil, services.NewSettingService(settingRepoForFeedStub{getByKey: func(key string) (*models.Setting, error) {
		return &models.Setting{Key: key, Value: map[string]string{"site_title": "Blog", "site_url": "https://example.com", "site_description": "Description"}[key]}, nil
	}}))
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "https://example.com/feed/atom", nil)
	atomBefore := controller.buildAtomFeed(context, posts)
	rssBefore := controller.buildRSSFeed(context, posts)
	require.Len(t, atomBefore.Entries, 1)
	require.Equal(t, "Blog", atomBefore.Author.Name)
	require.Equal(t, first.Format(time.RFC3339), atomBefore.Entries[0].Published)
	require.Equal(t, updated.Format(time.RFC3339), atomBefore.Entries[0].Updated)
	require.Equal(t, updated.Format(time.RFC3339), atomBefore.Updated)
	require.Equal(t, first.Format(time.RFC1123Z), rssBefore.Channel.Items[0].PubDate)
	require.Equal(t, updated.Format(time.RFC3339), rssBefore.Channel.Items[0].Updated)
	require.Equal(t, first.Format(time.RFC1123Z), rssBefore.Channel.PubDate)
	require.Equal(t, updated.Format(time.RFC1123Z), rssBefore.Channel.LastBuildDate)
	// Database UpdatedAt changes during draft autosave; public XML stays identical.
	posts[0].UpdatedAt = databaseSave.Add(24 * time.Hour)
	require.Equal(t, atomBefore, controller.buildAtomFeed(context, posts))
	require.Equal(t, rssBefore, controller.buildRSSFeed(context, posts))
	atomXML, err := xml.Marshal(atomBefore)
	require.NoError(t, err)
	rssXML, err := xml.Marshal(rssBefore)
	require.NoError(t, err)
	require.NotContains(t, string(atomXML), databaseSave.Format(time.RFC3339))
	require.Contains(t, string(rssXML), "<atom:updated>"+updated.Format(time.RFC3339)+"</atom:updated>")
}

func TestFeedFirstPublicationAndUnknownHistory(t *testing.T) {
	first := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	posts := []models.Post{
		{Model: gorm.Model{ID: 1, CreatedAt: first.Add(-time.Hour), UpdatedAt: first.Add(time.Hour)}, Status: "published", Title: "First", PublishedAt: &first, PublicationTimeKnown: true},
		{Model: gorm.Model{ID: 2, CreatedAt: first.Add(-time.Hour), UpdatedAt: first.Add(time.Hour)}, Status: "published", Title: "Unknown history"},
		{Model: gorm.Model{ID: 4}, Status: "published", Title: "Recorded publication, unknown first", PublishedAt: &first},
		{Model: gorm.Model{ID: 3}, Status: "draft", Title: "Secret draft", Body: "Private body", PublishedAt: &first},
	}
	controller := NewFeedController(nil, services.NewSettingService(settingRepoForFeedStub{getByKey: func(key string) (*models.Setting, error) {
		return &models.Setting{Key: key, Value: "https://example.com"}, nil
	}}))
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "https://example.com/feed/atom", nil)
	atom := controller.buildAtomFeed(context, posts)
	require.Len(t, atom.Entries, 2) // A recorded public event can satisfy updated without claiming it is the first publication.
	require.Empty(t, atom.Entries[1].Published)
	require.Equal(t, first.Format(time.RFC3339), atom.Entries[1].Updated)
	require.Equal(t, first.Format(time.RFC3339), atom.Entries[0].Updated)
	require.Equal(t, first.Format(time.RFC3339), atom.Entries[0].Published)
	rss := controller.buildRSSFeed(context, posts)
	require.Len(t, rss.Channel.Items, 3) // RSS may retain the article without optional pubDate.
	require.Empty(t, rss.Channel.Items[0].Updated)
	require.Empty(t, rss.Channel.Items[1].PubDate)
	require.Empty(t, rss.Channel.Items[1].Updated)
	require.Empty(t, rss.Channel.Items[2].PubDate)
	atomXML, _ := xml.Marshal(atom)
	rssXML, _ := xml.Marshal(rss)
	require.NotContains(t, string(atomXML), "Secret draft")
	require.NotContains(t, string(rssXML), "Secret draft")
	require.NotContains(t, string(rssXML), "Private body")
}

func TestFeedControllersQueryOnlyPublishedByPublicationDate(t *testing.T) {
	for _, kind := range []string{"atom", "rss"} {
		t.Run(kind, func(t *testing.T) {
			called := false
			postService := services.NewPostService(postRepoForFeedStub{listByStatus: func(status string, offset, limit int, sortBy string, descending bool) ([]models.Post, int64, error) {
				called = true
				require.Equal(t, "published", status)
				require.Equal(t, "published_at", sortBy)
				require.True(t, descending)
				require.Equal(t, 20, limit)
				return nil, 0, nil
			}})
			controller := NewFeedController(postService, services.NewSettingService(settingRepoForFeedStub{}))
			router := gin.New()
			router.GET("/", func(c *gin.Context) {
				if kind == "atom" {
					controller.AtomFeed(c)
				} else {
					controller.RSSFeed(c)
				}
			})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest("GET", "https://example.com/", nil))
			require.True(t, called)
			require.Equal(t, http.StatusOK, response.Code)
		})
	}
}
