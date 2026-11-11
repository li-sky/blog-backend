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

func TestPublishedPostOrderUsesPublicationEventWithoutPromotingDraftSaves(t *testing.T) {
	db := openTestDB(t)
	repo := NewPostRepository(db)
	user := &models.User{Username: "publication-order", Email: "publication-order@example.com"}
	require.NoError(t, db.Create(user).Error)
	base := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	firstOld, firstRecent := base.Add(time.Hour), base.Add(2*time.Hour)
	fallbackUpdate, newerUpdate := base.Add(3*time.Hour), base.Add(4*time.Hour)
	posts := []models.Post{
		{Title: "old first with newer republication", Status: "published", PublishedAt: &firstOld, PublishedUpdatedAt: &newerUpdate},
		{Title: "recent first", Status: "published", PublishedAt: &firstRecent},
		{Title: "missing first but actual republication", Status: "published", PublishedUpdatedAt: &fallbackUpdate},
		{Title: "unknown publication A", Status: "published"},
		{Title: "unknown publication B", Status: "published"},
		{Title: "private draft", Status: "draft", PublishedAt: &newerUpdate},
	}
	for i := range posts {
		posts[i].UserID = user.ID
		require.NoError(t, repo.Create(&posts[i]))
	}
	assertOrder := func(desc bool, expected []uint) {
		t.Helper()
		items, total, err := repo.ListByStatus("published", 0, 10, "published_at", desc)
		require.NoError(t, err)
		require.Equal(t, int64(5), total)
		ids := make([]uint, len(items))
		for i, item := range items {
			ids[i] = item.ID
			require.Equal(t, "published", item.Status)
		}
		require.Equal(t, expected, ids)
	}
	descending := []uint{posts[2].ID, posts[1].ID, posts[0].ID, posts[4].ID, posts[3].ID}
	assertOrder(true, descending)
	assertOrder(false, []uint{posts[0].ID, posts[1].ID, posts[2].ID, posts[3].ID, posts[4].ID})
	// Saving an editing snapshot changes raw database timestamps, never its public order.
	savedAt := base.Add(24 * time.Hour)
	require.NoError(t, db.Model(&models.Post{}).Where("id = ?", posts[0].ID).
		Select("Draft", "DraftUpdatedAt", "UpdatedAt").Updates(&models.Post{Draft: &models.PostDraft{Title: "private edits"}, DraftUpdatedAt: &savedAt, Model: gorm.Model{UpdatedAt: savedAt}}).Error)
	assertOrder(true, descending)
	items, total, err := repo.ListByStatus("published", 0, 1, "published_at", true)
	require.NoError(t, err)
	require.Equal(t, int64(5), total)
	require.Equal(t, posts[2].ID, items[0].ID, "the recent actual publish must be visible on the first page")
	require.Nil(t, items[0].PublishedAt, "sorting must not invent a first publication time")
	require.True(t, fallbackUpdate.Equal(*items[0].PublishedUpdatedAt))
	old, err := repo.GetByID(posts[0].ID)
	require.NoError(t, err)
	require.True(t, firstOld.Equal(*old.PublishedAt))
	require.True(t, newerUpdate.Equal(*old.PublishedUpdatedAt))
}
