package services

import (
	"blog-backend/models"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPostService_UpdatePost_InvalidStatus(t *testing.T) {
	editor := permUser(1, "edit_post")
	svc := NewPostService(postRepoStub{})

	err := svc.UpdatePost(editor, &models.Post{Model: gorm.Model{ID: 1}, Status: "nope"})
	require.Error(t, err)
}

func TestPostService_UpdatePost_SetsPublishedAtAndUnsets(t *testing.T) {
	editor := permUser(1, "edit_post")

	existing := &models.Post{Model: gorm.Model{ID: 1}, Status: "draft"}
	svc := NewPostService(postRepoStub{
		getByID: func(id uint) (*models.Post, error) { return existing, nil },
		update: func(post *models.Post) error {
			require.Equal(t, "published", post.Status)
			require.NotNil(t, post.PublishedAt)
			return nil
		},
	})

	err := svc.UpdatePost(editor, &models.Post{Model: gorm.Model{ID: 1}, Title: "t", Summary: "s", Body: "b", Status: "published"})
	require.NoError(t, err)

	svc2 := NewPostService(postRepoStub{
		getByID: func(id uint) (*models.Post, error) { return &models.Post{Model: gorm.Model{ID: 1}, Status: "published"}, nil },
		update: func(post *models.Post) error {
			require.Equal(t, "archived", post.Status)
			require.Nil(t, post.PublishedAt)
			return nil
		},
	})
	err = svc2.UpdatePost(editor, &models.Post{Model: gorm.Model{ID: 1}, Status: "archived"})
	require.NoError(t, err)
}

func TestPostService_PublishPost_NoOpAndUpdate(t *testing.T) {
	editor := permUser(1, "publish_post")

	svcNoOp := NewPostService(postRepoStub{getByID: func(id uint) (*models.Post, error) {
		return &models.Post{Model: gorm.Model{ID: id}, Status: "published"}, nil
	}})
	require.NoError(t, svcNoOp.PublishPost(editor, 1))

	svcUpdate := NewPostService(postRepoStub{
		getByID: func(id uint) (*models.Post, error) { return &models.Post{Model: gorm.Model{ID: id}, Status: "draft"}, nil },
		update: func(post *models.Post) error {
			require.Equal(t, "published", post.Status)
			require.NotNil(t, post.PublishedAt)
			return nil
		},
	})
	require.NoError(t, svcUpdate.PublishPost(editor, 1))
}

func TestPostService_GetAndList_Validation(t *testing.T) {
	svc := NewPostService(postRepoStub{})

	_, err := svc.GetPost(0)
	require.Error(t, err)
	_, err = svc.GetPublishedPost(0)
	require.Error(t, err)

	svc2 := NewPostService(postRepoStub{
		getByID: func(id uint) (*models.Post, error) { return &models.Post{Model: gorm.Model{ID: id}}, nil },
		getByIDStatus: func(id uint, status string) (*models.Post, error) { return &models.Post{Model: gorm.Model{ID: id}, Status: status}, nil },
	})
	p, err := svc2.GetPost(1)
	require.NoError(t, err)
	require.Equal(t, uint(1), p.ID)
	pp, err := svc2.GetPublishedPost(1)
	require.NoError(t, err)
	require.Equal(t, "published", pp.Status)
}

func TestPostService_ListAllPosts_PermissionDeniedAndAllowed(t *testing.T) {
	svc := NewPostService(postRepoStub{list: func(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
		return []models.Post{{Model: gorm.Model{ID: 1}}}, 1, nil
	}})

	_, _, err := svc.ListAllPosts(permUser(1), 0, 10, "created_at", true)
	require.Error(t, err)

	posts, total, err := svc.ListAllPosts(permUser(1, "edit_post"), 0, 10, "created_at", true)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, posts, 1)
}

func TestPostService_CreatePost_InvalidStatus(t *testing.T) {
	author := permUser(1, "create_post")
	svc := NewPostService(postRepoStub{})
	
	err := svc.CreatePost(author, &models.Post{Status: "nope"})
	require.Error(t, err)
}

func TestPostService_DeletePost_PermissionDenied(t *testing.T) {
	svc := NewPostService(postRepoStub{})
	err := svc.DeletePost(permUser(1), 1)
	require.Error(t, err)
}
