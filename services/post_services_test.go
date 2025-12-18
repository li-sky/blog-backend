package services

import (
	"blog-backend/models"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type postRepoStub struct {
	create        func(post *models.Post) error
	getByID       func(id uint) (*models.Post, error)
	getByIDStatus func(id uint, status string) (*models.Post, error)
	list          func(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error)
	update        func(post *models.Post) error
	delete        func(id uint) error
}

func (s postRepoStub) Create(post *models.Post) error {
	if s.create != nil {
		return s.create(post)
	}
	return nil
}
func (s postRepoStub) GetByID(id uint) (*models.Post, error) {
	if s.getByID != nil {
		return s.getByID(id)
	}
	return nil, errors.New("not found")
}
func (s postRepoStub) GetByIDAndStatus(id uint, status string) (*models.Post, error) {
	if s.getByIDStatus != nil {
		return s.getByIDStatus(id, status)
	}
	return nil, errors.New("not found")
}
func (s postRepoStub) List(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	if s.list != nil {
		return s.list(offset, limit, sortBy, desc)
	}
	return nil, 0, nil
}
func (s postRepoStub) ListByStatus(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	return nil, 0, nil
}
func (s postRepoStub) Update(post *models.Post) error {
	if s.update != nil {
		return s.update(post)
	}
	return nil
}
func (s postRepoStub) Delete(id uint) error {
	if s.delete != nil {
		return s.delete(id)
	}
	return nil
}

func TestPostService_CreatePost_SetsDefaultsAndPublishedAt(t *testing.T) {
	author := &models.User{Model: gorm.Model{ID: 7}, Roles: []models.Role{{Permissions: []models.Permission{{Name: "create_post"}}}}}

	var captured *models.Post
	svc := NewPostService(postRepoStub{
		create: func(post *models.Post) error {
			captured = post
			return nil
		},
	})

	post := &models.Post{Title: "t", Summary: "s", Body: "b", Status: "published"}
	before := time.Now().UTC()
	err := svc.CreatePost(author, post)
	after := time.Now().UTC()

	require.NoError(t, err)
	require.NotNil(t, captured)
	require.Equal(t, uint(7), captured.UserID)
	require.Equal(t, "published", captured.Status)
	require.NotNil(t, captured.PublishedAt)
	require.True(t, captured.PublishedAt.After(before) || captured.PublishedAt.Equal(before))
	require.True(t, captured.PublishedAt.Before(after) || captured.PublishedAt.Equal(after))
}

func TestPostService_CreatePost_PermissionDenied(t *testing.T) {
	author := &models.User{Model: gorm.Model{ID: 7}}
	svc := NewPostService(postRepoStub{})
	
	err := svc.CreatePost(author, &models.Post{Title: "t"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}
