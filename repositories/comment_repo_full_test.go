package repositories

import (
	"blog-backend/models"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCommentRepository_CRUD_List(t *testing.T) {
	db := openTestDB(t)
	repo := NewCommentRepository(db)

	user := &models.User{Username: "alice", Email: "a@example.com", Password: "veryverysecure"}
	require.NoError(t, db.Create(user).Error)
	post := &models.Post{Title: "t", UserID: user.ID, Status: "published"}
	require.NoError(t, db.Create(post).Error)

	comment := &models.Comment{PostID: post.ID, UserID: user.ID, Body: "hello", Visible: true}
	require.NoError(t, repo.Create(comment))

	list, total, err := repo.ListByPost(post.ID, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	require.Equal(t, user.ID, list[0].User.ID)

	list2, _, err := repo.ListByPost(post.ID, 0, 0)
	require.NoError(t, err)
	require.Len(t, list2, 1)

	got, err := repo.GetByID(comment.ID)
	require.NoError(t, err)
	require.Equal(t, comment.Body, got.Body)

	got.Visible = false
	require.NoError(t, repo.Update(got))

	reloaded, err := repo.GetByID(comment.ID)
	require.NoError(t, err)
	require.False(t, reloaded.Visible)

	require.NoError(t, repo.Delete(comment.ID))
	_, err = repo.GetByID(comment.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
