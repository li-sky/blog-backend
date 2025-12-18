package repositories

import (
	"blog-backend/models"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPostRepository_CRUD_AndLists(t *testing.T) {
	db := openTestDB(t)
	repo := NewPostRepository(db)

	user := &models.User{Username: "alice", Email: "a@example.com", Password: "veryverysecure"}
	require.NoError(t, db.Create(user).Error)

	now := time.Now().UTC()
	post := &models.Post{Title: "t", Summary: "s", Body: "b", UserID: user.ID, Status: "published", PublishedAt: &now}
	require.NoError(t, repo.Create(post))

	got, err := repo.GetByID(post.ID)
	require.NoError(t, err)
	require.Equal(t, post.Title, got.Title)
	require.Equal(t, user.ID, got.User.ID)

	got2, err := repo.GetByIDAndStatus(post.ID, "published")
	require.NoError(t, err)
	require.Equal(t, post.ID, got2.ID)

	_, err = repo.GetByIDAndStatus(post.ID, "draft")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	items, total, err := repo.List(0, 10, "created_at", true)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	require.NotEmpty(t, items)

	// Cover limit <= 0 branch
	items2, _, err := repo.List(0, 0, "created_at", true)
	require.NoError(t, err)
	require.NotEmpty(t, items2)

	pubItems, pubTotal, err := repo.ListByStatus("published", 0, 10, "published_at", true)
	require.NoError(t, err)
	require.GreaterOrEqual(t, pubTotal, int64(1))
	require.NotEmpty(t, pubItems)

	post.Title = "t2"
	require.NoError(t, repo.Update(post))

	updated, err := repo.GetByID(post.ID)
	require.NoError(t, err)
	require.Equal(t, "t2", updated.Title)

	require.NoError(t, repo.Delete(post.ID))
	_, err = repo.GetByID(post.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
