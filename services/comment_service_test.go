package services

import (
	"blog-backend/models"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type commentRepoStub struct {
	create func(comment *models.Comment) error
	getByID func(id uint) (*models.Comment, error)
	update func(comment *models.Comment) error
	delete func(id uint) error
}

func (s commentRepoStub) Create(comment *models.Comment) error {
	if s.create != nil {
		return s.create(comment)
	}
	return nil
}
func (s commentRepoStub) ListByPost(postID uint, offset, limit int) ([]models.Comment, int64, error) {
	return nil, 0, nil
}
func (s commentRepoStub) GetByID(id uint) (*models.Comment, error) {
	if s.getByID != nil {
		return s.getByID(id)
	}
	return nil, errors.New("not found")
}
func (s commentRepoStub) Delete(id uint) error {
	if s.delete != nil {
		return s.delete(id)
	}
	return nil
}
func (s commentRepoStub) Update(comment *models.Comment) error {
	if s.update != nil {
		return s.update(comment)
	}
	return nil
}

func TestCommentService_AddComment_RequiresPost(t *testing.T) {
	author := &models.User{Model: gorm.Model{ID: 1}}
	svc := NewCommentService(commentRepoStub{}, postRepoStub{
		getByID: func(id uint) (*models.Post, error) {
			return nil, errors.New("missing post")
		},
	})

	c, err := svc.AddComment(author, 123, "hello", "", "", "")
	require.Error(t, err)
	require.Nil(t, c)
}

func TestCommentService_DeleteComment_PermissionDenied(t *testing.T) {
	requestor := &models.User{Model: gorm.Model{ID: 2}}
	
	svc := NewCommentService(commentRepoStub{
		getByID: func(id uint) (*models.Comment, error) {
			return &models.Comment{Model: gorm.Model{ID: id}, UserID: 999}, nil
		},
	}, postRepoStub{})

	err := svc.DeleteComment(requestor, 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}

func TestCommentService_SetVisibility_PermissionDenied(t *testing.T) {
	operator := &models.User{Model: gorm.Model{ID: 1}}
	svc := NewCommentService(commentRepoStub{}, postRepoStub{})

	err := svc.SetVisibility(operator, 10, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}
