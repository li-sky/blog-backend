package controllers

import (
	"blog-backend/models"
	"blog-backend/repositories"
	"blog-backend/services"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func draftTestApp(t *testing.T) (*gin.Engine, *gorm.DB, *services.PostService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "drafts.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Permission{}, &models.Role{}, &models.User{}, &models.Post{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.Create(&models.User{Model: gorm.Model{ID: 1}, Username: "author", Email: "author@example.com"}).Error)
	svc := services.NewPostService(repositories.NewPostRepository(db))
	ctrl := NewPostController(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if c.GetHeader("X-Test-Auth") == "yes" {
			c.Set("currentUser", controllerUser(1, "create_post", "edit_post", "publish_post", "delete_post"))
		}
	})
	r.POST("/posts", ctrl.Create)
	r.GET("/posts", ctrl.List)
	r.GET("/posts/manage", ctrl.ListManage)
	r.GET("/posts/:postID", ctrl.Get)
	r.PUT("/posts/:postID", ctrl.Update)
	r.GET("/posts/:postID/draft", ctrl.GetDraft)
	r.PUT("/posts/:postID/draft", ctrl.SaveDraft)
	r.DELETE("/posts/:postID/draft", ctrl.DiscardDraft)
	r.POST("/posts/:postID/publish", ctrl.Publish)
	return r, db, svc
}
func draftRequest(r *gin.Engine, method, path string, payload any, auth bool) *httptest.ResponseRecorder {
	var body bytes.Buffer
	if payload != nil {
		_ = json.NewEncoder(&body).Encode(payload)
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", "application/json")
	if auth {
		req.Header.Set("X-Test-Auth", "yes")
	}
	result := httptest.NewRecorder()
	r.ServeHTTP(result, req)
	return result
}
func decodeDraft(t *testing.T, response *httptest.ResponseRecorder) services.DraftResponse {
	t.Helper()
	require.Equal(t, 200, response.Code, response.Body.String())
	var draft services.DraftResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &draft))
	return draft
}
func TestDraftHTTP_PublicIsolationPublishDatesConflictDiscardAndIdempotency(t *testing.T) {
	r, db, _ := draftTestApp(t)
	create := map[string]any{"title": "First title", "body": "Live body", "summary": "Live summary", "status": "draft", "clientRequestId": "create-one"}
	first := draftRequest(r, "POST", "/posts", create, true)
	require.Equal(t, 201, first.Code, first.Body.String())
	var post struct {
		ID           uint   `json:"id"`
		DraftVersion uint64 `json:"draftVersion"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &post))
	require.NotZero(t, post.ID)
	require.Equal(t, uint64(1), post.DraftVersion)
	retry := draftRequest(r, "POST", "/posts", create, true)
	require.Equal(t, 201, retry.Code)
	var repeated struct {
		ID uint `json:"id"`
	}
	require.NoError(t, json.Unmarshal(retry.Body.Bytes(), &repeated))
	require.Equal(t, post.ID, repeated.ID)
	var count int64
	require.NoError(t, db.Model(&models.Post{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
	path := fmt.Sprintf("/posts/%d", post.ID)
	require.Equal(t, 404, draftRequest(r, "GET", path, nil, false).Code)
	require.Equal(t, 401, draftRequest(r, "GET", path+"/draft", nil, false).Code)
	draft := decodeDraft(t, draftRequest(r, "GET", path+"/draft", nil, true))
	require.Nil(t, draft.PublishedAt)
	published := decodeDraft(t, draftRequest(r, "POST", path+"/publish", map[string]any{"version": draft.Version}, true))
	require.True(t, published.PublicationTimeKnown)
	require.NotNil(t, published.PublishedAt)
	require.Nil(t, published.PublishedUpdatedAt)
	public := draftRequest(r, "GET", path, nil, false)
	require.Equal(t, 200, public.Code)
	require.Contains(t, public.Body.String(), "Live body")
	require.NotContains(t, public.Body.String(), "draftVersion")
	saved := decodeDraft(t, draftRequest(r, "PUT", path+"/draft", map[string]any{"version": published.Version, "title": "SECRET draft", "body": "SECRET unsent content"}, true))
	require.Equal(t, "published", saved.Status)
	require.True(t, saved.HasUnpublishedChanges)
	require.Equal(t, *published.PublishedAt, *saved.PublishedAt)
	require.Nil(t, saved.PublishedUpdatedAt)
	for _, endpoint := range []string{path, "/posts"} {
		response := draftRequest(r, "GET", endpoint, nil, false)
		require.Equal(t, 200, response.Code)
		require.NotContains(t, response.Body.String(), "SECRET")
		require.NotContains(t, response.Body.String(), "draftVersion")
	}
	conflict := draftRequest(r, "PUT", path+"/draft", map[string]any{"version": published.Version, "title": "stale", "body": "stale"}, true)
	require.Equal(t, 409, conflict.Code)
	require.Contains(t, conflict.Body.String(), "draft_conflict")
	require.Equal(t, "private, no-store", conflict.Header().Get("Cache-Control"))
	require.Equal(t, 409, draftRequest(r, "POST", path+"/publish", nil, true).Code)
	require.Equal(t, 409, draftRequest(r, "PUT", path, map[string]any{"title": "old client"}, true).Code)
	republished := decodeDraft(t, draftRequest(r, "POST", path+"/publish", map[string]any{"version": saved.Version}, true))
	require.Equal(t, *published.PublishedAt, *republished.PublishedAt)
	require.NotNil(t, republished.PublishedUpdatedAt)
	again := decodeDraft(t, draftRequest(r, "POST", path+"/publish", map[string]any{"version": republished.Version}, true))
	require.Equal(t, *republished.PublishedUpdatedAt, *again.PublishedUpdatedAt)
	saved = decodeDraft(t, draftRequest(r, "PUT", path+"/draft", map[string]any{"version": again.Version, "title": "discard me", "body": ""}, true))
	discarded := decodeDraft(t, draftRequest(r, "DELETE", fmt.Sprintf("%s/draft?version=%d", path, saved.Version), nil, true))
	require.Equal(t, "SECRET draft", discarded.Title)
	require.False(t, discarded.HasUnpublishedChanges)
	require.Equal(t, 409, draftRequest(r, "DELETE", path+"/draft", nil, true).Code)
}
func TestDraftCAS_OneConcurrentWriterAndLegacyDatesRemainUnknown(t *testing.T) {
	_, db, svc := draftTestApp(t)
	user := controllerUser(1, "edit_post", "publish_post")
	past := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	post := models.Post{Title: "Original", Body: "Original", UserID: 1, Status: "published", PublishedAt: &past}
	require.NoError(t, db.Create(&post).Error)
	var wait sync.WaitGroup
	results := make(chan error, 10)
	for index := 0; index < 10; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			_, err := svc.SaveDraft(user, post.ID, models.PostDraft{Title: fmt.Sprintf("Draft %d", index), Body: "Changed"}, 0)
			results <- err
		}(index)
	}
	wait.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else {
			require.ErrorIs(t, err, services.ErrDraftConflict)
		}
	}
	require.Equal(t, 1, success)
	draft, err := svc.GetDraft(user, post.ID)
	require.NoError(t, err)
	require.False(t, draft.PublicationTimeKnown)
	updated, err := svc.PublishDraft(user, post.ID, draft.Version)
	require.NoError(t, err)
	require.False(t, updated.PublicationTimeKnown)
	require.Equal(t, past, *updated.PublishedAt)
	require.NotNil(t, updated.PublishedUpdatedAt)
	public, err := svc.GetPublishedPost(post.ID)
	require.NoError(t, err)
	data, err := json.Marshal(public)
	require.NoError(t, err)
	require.False(t, strings.Contains(string(data), "DraftVersion"))
	require.False(t, strings.Contains(string(data), "draftUpdatedAt"))
}

func TestLegacyDraftWithoutHistoricalTimeDoesNotInventFirstPublication(t *testing.T) {
	_, db, svc := draftTestApp(t)
	post := models.Post{Title: "Legacy", Body: "Possibly once published", Status: "draft", UserID: 1}
	require.NoError(t, db.Create(&post).Error)
	result, err := svc.PublishDraft(controllerUser(1, "publish_post"), post.ID, 0)
	require.NoError(t, err)
	require.False(t, result.PublicationTimeKnown)
	require.Nil(t, result.PublishedAt)
	require.NotNil(t, result.PublishedUpdatedAt)
}
