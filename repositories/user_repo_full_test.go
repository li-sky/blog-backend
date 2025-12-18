package repositories

import (
	"blog-backend/models"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserRepository_CRUD_AndQueries(t *testing.T) {
	db := openTestDB(t)
	repo := NewUserRepository(db)

	// Seed role + permission and attach to user.
	perm := models.Permission{Name: "assign_roles", Description: "x"}
	require.NoError(t, db.Create(&perm).Error)
	role := models.Role{Name: "admin", Permissions: []models.Permission{perm}}
	require.NoError(t, db.Create(&role).Error)

	u := &models.User{Username: "alice", Email: "a@example.com", Password: "veryverysecure", Roles: []models.Role{role}}
	require.NoError(t, repo.Create(u))
	require.NotZero(t, u.ID)

	byID, err := repo.GetByID(u.ID)
	require.NoError(t, err)
	require.Equal(t, u.Username, byID.Username)
	require.Len(t, byID.Roles, 1)
	require.Len(t, byID.Roles[0].Permissions, 1)

	byUsername, err := repo.GetByUsername("alice")
	require.NoError(t, err)
	require.Equal(t, u.ID, byUsername.ID)

	byEmail, err := repo.GetByEmail("a@example.com")
	require.NoError(t, err)
	require.Equal(t, u.ID, byEmail.ID)

	items, total, err := repo.List(0, 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	require.NotEmpty(t, items)

	// Cover limit <= 0 branch.
	items2, total2, err := repo.List(0, 0)
	require.NoError(t, err)
	require.Equal(t, total, total2)
	require.NotEmpty(t, items2)

	u.Email = "b@example.com"
	require.NoError(t, repo.Update(u))

	updated, err := repo.GetByID(u.ID)
	require.NoError(t, err)
	require.Equal(t, "b@example.com", updated.Email)

	ok, err := repo.HasUserWithRole("admin")
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = repo.HasUserWithRole("missing")
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, repo.Delete(u.ID))
	_, err = repo.GetByID(u.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
