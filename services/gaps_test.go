package services

import (
	"blog-backend/models"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type postRepoGapStub struct {
	getByID       func(id uint) (*models.Post, error)
	getByIDStatus func(id uint, status string) (*models.Post, error)
	list          func(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error)
	listByStatus  func(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error)
	create        func(post *models.Post) error
	update        func(post *models.Post) error
	delete        func(id uint) error
}

func (s postRepoGapStub) Create(post *models.Post) error {
	if s.create != nil {
		return s.create(post)
	}
	post.Model = gorm.Model{ID: 1}
	return nil
}
func (s postRepoGapStub) GetByID(id uint) (*models.Post, error) {
	if s.getByID != nil {
		return s.getByID(id)
	}
	return &models.Post{Model: gorm.Model{ID: id}}, nil
}
func (s postRepoGapStub) GetByIDAndStatus(id uint, status string) (*models.Post, error) {
	if s.getByIDStatus != nil {
		return s.getByIDStatus(id, status)
	}
	return &models.Post{Model: gorm.Model{ID: id}, Status: status}, nil
}
func (s postRepoGapStub) List(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	if s.list != nil {
		return s.list(offset, limit, sortBy, desc)
	}
	return nil, 0, nil
}
func (s postRepoGapStub) ListByStatus(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	if s.listByStatus != nil {
		return s.listByStatus(status, offset, limit, sortBy, desc)
	}
	return nil, 0, nil
}
func (s postRepoGapStub) Update(post *models.Post) error {
	if s.update != nil {
		return s.update(post)
	}
	return nil
}
func (s postRepoGapStub) Delete(id uint) error {
	if s.delete != nil {
		return s.delete(id)
	}
	return nil
}

type commentRepoGapStub struct {
	create     func(comment *models.Comment) error
	listByPost func(postID uint, offset, limit int) ([]models.Comment, int64, error)
	getByID    func(id uint) (*models.Comment, error)
	delete     func(id uint) error
	update     func(comment *models.Comment) error
}

func (s commentRepoGapStub) Create(comment *models.Comment) error {
	if s.create != nil {
		return s.create(comment)
	}
	comment.Model = gorm.Model{ID: 1}
	return nil
}
func (s commentRepoGapStub) ListByPost(postID uint, offset, limit int) ([]models.Comment, int64, error) {
	if s.listByPost != nil {
		return s.listByPost(postID, offset, limit)
	}
	return nil, 0, nil
}
func (s commentRepoGapStub) GetByID(id uint) (*models.Comment, error) {
	if s.getByID != nil {
		return s.getByID(id)
	}
	return nil, gorm.ErrRecordNotFound
}
func (s commentRepoGapStub) Delete(id uint) error {
	if s.delete != nil {
		return s.delete(id)
	}
	return nil
}
func (s commentRepoGapStub) Update(comment *models.Comment) error {
	if s.update != nil {
		return s.update(comment)
	}
	return nil
}

type settingRepoGapStub struct {
	list   func() ([]models.Setting, error)
	get    func(key string) (*models.Setting, error)
	upsert func(key, value, desc string) (*models.Setting, error)
}

func (s settingRepoGapStub) List() ([]models.Setting, error) {
	if s.list != nil {
		return s.list()
	}
	return nil, nil
}
func (s settingRepoGapStub) GetByKey(key string) (*models.Setting, error) {
	if s.get != nil {
		return s.get(key)
	}
	return &models.Setting{Key: key, Value: "v"}, nil
}
func (s settingRepoGapStub) Upsert(key, value, desc string) (*models.Setting, error) {
	if s.upsert != nil {
		return s.upsert(key, value, desc)
	}
	return &models.Setting{Key: key, Value: value, Desc: desc}, nil
}

func TestSettingService_ListSettings_And_GetByKey(t *testing.T) {
	svc := NewSettingService(settingRepoGapStub{list: func() ([]models.Setting, error) { return []models.Setting{{Key: "k"}}, nil }})
	items, err := svc.ListSettings()
	require.NoError(t, err)
	require.Len(t, items, 1)

	_, err = svc.GetByKey("  ")
	require.Error(t, err)

	got, err := svc.GetByKey(" k ")
	require.NoError(t, err)
	require.Equal(t, "k", got.Key)
}

func TestPostService_ListPublishedPosts_And_DeletePost_Errors(t *testing.T) {
	called := false
	svc := NewPostService(postRepoGapStub{listByStatus: func(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
		called = true
		return []models.Post{{Model: gorm.Model{ID: 1}, Status: status}}, 1, nil
	}})
	posts, total, err := svc.ListPublishedPosts(0, 10, "created_at", true)
	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, int64(1), total)
	require.Len(t, posts, 1)

	err = svc.DeletePost(nil, 1)
	require.Error(t, err)
	err = svc.DeletePost(&models.User{}, 0)
	require.Error(t, err)
	de := NewPostService(postRepoGapStub{delete: func(id uint) error { return errors.New("boom") }})
	err = de.DeletePost(controllerUserForService("delete_post"), 1)
	require.Error(t, err)
}

func TestPostService_CreateUpdatePublish_Branches(t *testing.T) {
	svc := NewPostService(postRepoGapStub{})

	err := svc.CreatePost(nil, &models.Post{})
	require.Error(t, err)
	err = svc.CreatePost(controllerUserForService("create_post"), nil)
	require.Error(t, err)
	err = svc.CreatePost(controllerUserForService("create_post"), &models.Post{Status: "nope"})
	require.Error(t, err)

	p := &models.Post{Title: "t", Summary: "s", Body: "b", Status: "published"}
	err = svc.CreatePost(controllerUserForService("create_post", "publish_post"), p)
	require.NoError(t, err)
	require.NotNil(t, p.PublishedAt)

	// UpdatePost invalid args
	err = svc.UpdatePost(nil, &models.Post{Model: gorm.Model{ID: 1}})
	require.Error(t, err)
	err = svc.UpdatePost(controllerUserForService("edit_post"), &models.Post{})
	require.Error(t, err)
	// invalid status
	err = svc.UpdatePost(controllerUserForService("edit_post"), &models.Post{Model: gorm.Model{ID: 1}, Status: "nope"})
	require.Error(t, err)

	existing := &models.Post{Model: gorm.Model{ID: 2}, Status: "draft"}
	svc2 := NewPostService(postRepoGapStub{getByID: func(id uint) (*models.Post, error) { return existing, nil }, update: func(post *models.Post) error { return nil }})
	err = svc2.UpdatePost(controllerUserForService("edit_post"), &models.Post{Model: gorm.Model{ID: 2}, Title: "x", Status: "published"})
	require.NoError(t, err)
	require.Equal(t, "draft", existing.Status)
	require.Nil(t, existing.PublishedAt)
	require.Equal(t, "x", existing.Draft.Title)

	// PublishPost branches
	err = svc2.PublishPost(nil, 1)
	require.Error(t, err)
	err = svc2.PublishPost(controllerUserForService("publish_post"), 0)
	require.Error(t, err)
	svc3 := NewPostService(postRepoGapStub{getByID: func(id uint) (*models.Post, error) { return nil, errors.New("boom") }})
	err = svc3.PublishPost(controllerUserForService("publish_post"), 1)
	require.Error(t, err)

	already := &models.Post{Model: gorm.Model{ID: 3}, Status: "published", PublishedAt: ptrTime(time.Now())}
	svc4 := NewPostService(postRepoGapStub{getByID: func(id uint) (*models.Post, error) { return already, nil }})
	err = svc4.PublishPost(controllerUserForService("publish_post"), 3)
	require.NoError(t, err)
}

func TestCommentService_ListComments_And_SetVisibility(t *testing.T) {
	svc := NewCommentService(commentRepoGapStub{listByPost: func(postID uint, offset, limit int) ([]models.Comment, int64, error) {
		return []models.Comment{{Model: gorm.Model{ID: 1}, PostID: postID}}, 1, nil
	}}, postRepoGapStub{})

	_, _, err := svc.ListComments(0, 0, 10)
	require.Error(t, err)
	items, total, err := svc.ListComments(1, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	// SetVisibility permission denied
	err = svc.SetVisibility(&models.User{}, 1, false)
	require.Error(t, err)

	// SetVisibility comment id required
	err = svc.SetVisibility(controllerUserForService("moderate_comments"), 0, false)
	require.Error(t, err)

	// SetVisibility repo error and success
	svc2 := NewCommentService(commentRepoGapStub{getByID: func(id uint) (*models.Comment, error) { return nil, errors.New("boom") }}, postRepoGapStub{})
	err = svc2.SetVisibility(controllerUserForService("moderate_comments"), 1, false)
	require.Error(t, err)

	comment := &models.Comment{Model: gorm.Model{ID: 2}, Visible: true}
	svc3 := NewCommentService(commentRepoGapStub{getByID: func(id uint) (*models.Comment, error) { return comment, nil }, update: func(c *models.Comment) error { return nil }}, postRepoGapStub{})
	err = svc3.SetVisibility(controllerUserForService("moderate_comments"), 2, false)
	require.NoError(t, err)
	require.False(t, comment.Visible)
}

func TestCommentService_AddAndDelete_Branches(t *testing.T) {
	svc := NewCommentService(commentRepoGapStub{}, postRepoGapStub{})

	_, err := svc.AddComment(nil, 1, "x", "", "", "")
	require.Error(t, err)
	_, err = svc.AddComment(&models.User{}, 1, "x", "", "", "")
	require.Error(t, err)
	_, err = svc.AddComment(&models.User{Model: gorm.Model{ID: 1}}, 0, "x", "", "", "")
	require.Error(t, err)
	_, err = svc.AddComment(&models.User{Model: gorm.Model{ID: 1}}, 1, "", "", "", "")
	require.Error(t, err)

	postErr := NewCommentService(commentRepoGapStub{}, postRepoGapStub{getByID: func(id uint) (*models.Post, error) { return nil, errors.New("boom") }})
	_, err = postErr.AddComment(&models.User{Model: gorm.Model{ID: 1}}, 1, "x", "", "", "")
	require.Error(t, err)

	createErr := NewCommentService(commentRepoGapStub{create: func(comment *models.Comment) error { return errors.New("boom") }}, postRepoGapStub{})
	_, err = createErr.AddComment(&models.User{Model: gorm.Model{ID: 1}}, 1, "x", "", "", "")
	require.Error(t, err)

	ok := NewCommentService(commentRepoGapStub{}, postRepoGapStub{})
	c, err := ok.AddComment(&models.User{Model: gorm.Model{ID: 1}}, 1, "x", "ip", "ua", "CN")
	require.NoError(t, err)
	require.Equal(t, uint(1), c.PostID)
	require.Equal(t, "CN", c.Country)

	// DeleteComment
	err = ok.DeleteComment(nil, 1)
	require.Error(t, err)
	err = ok.DeleteComment(&models.User{Model: gorm.Model{ID: 1}}, 0)
	require.Error(t, err)

	getErr := NewCommentService(commentRepoGapStub{getByID: func(id uint) (*models.Comment, error) { return nil, errors.New("boom") }}, postRepoGapStub{})
	err = getErr.DeleteComment(&models.User{Model: gorm.Model{ID: 1}}, 1)
	require.Error(t, err)

	permDenied := NewCommentService(commentRepoGapStub{getByID: func(id uint) (*models.Comment, error) {
		return &models.Comment{Model: gorm.Model{ID: 1}, UserID: 2}, nil
	}}, postRepoGapStub{})
	err = permDenied.DeleteComment(&models.User{Model: gorm.Model{ID: 1}}, 1)
	require.Error(t, err)

	deleteErr := NewCommentService(commentRepoGapStub{getByID: func(id uint) (*models.Comment, error) {
		return &models.Comment{Model: gorm.Model{ID: 1}, UserID: 1}, nil
	}, delete: func(id uint) error { return errors.New("boom") }}, postRepoGapStub{})
	err = deleteErr.DeleteComment(&models.User{Model: gorm.Model{ID: 1}}, 1)
	require.Error(t, err)
}

func controllerUserForService(perms ...string) *models.User {
	role := models.Role{Name: "r"}
	for _, p := range perms {
		role.Permissions = append(role.Permissions, models.Permission{Name: p})
	}
	return &models.User{Model: gorm.Model{ID: 1}, Roles: []models.Role{role}}
}

func ptrTime(t time.Time) *time.Time { return &t }
