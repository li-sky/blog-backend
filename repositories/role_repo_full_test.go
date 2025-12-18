package repositories

import (
	"blog-backend/models"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRoleRepository_EnsureRole_CreateAndReuse(t *testing.T) {
	db := openTestDB(t)
	repo := NewRoleRepository(db)

	role, err := repo.EnsureRole("admin")
	require.NoError(t, err)
	require.NotZero(t, role.ID)

	again, err := repo.EnsureRole("admin")
	require.NoError(t, err)
	require.Equal(t, role.ID, again.ID)
}

func TestRoleRepository_EnsurePermission_CreateAndUpdateDescription(t *testing.T) {
	db := openTestDB(t)
	repo := NewRoleRepository(db)

	perm, err := repo.EnsurePermission("p1", "d1")
	require.NoError(t, err)
	require.NotZero(t, perm.ID)
	require.Equal(t, "d1", perm.Description)

	perm2, err := repo.EnsurePermission("p1", "d2")
	require.NoError(t, err)
	require.Equal(t, perm.ID, perm2.ID)
	require.Equal(t, "d2", perm2.Description)
}

func TestRoleRepository_CRUD_List_GetBy(t *testing.T) {
	db := openTestDB(t)
	repo := NewRoleRepository(db)

	role := &models.Role{Name: "editor"}
	require.NoError(t, repo.Create(role))

	got, err := repo.GetByID(role.ID)
	require.NoError(t, err)
	require.Equal(t, "editor", got.Name)

	gotByName, err := repo.GetByName("editor")
	require.NoError(t, err)
	require.Equal(t, role.ID, gotByName.ID)

	roles, err := repo.GetByIDs([]uint{role.ID})
	require.NoError(t, err)
	require.Len(t, roles, 1)

	rolesEmpty, err := repo.GetByIDs(nil)
	require.NoError(t, err)
	require.Len(t, rolesEmpty, 0)

	list, total, err := repo.List(0, 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	require.NotEmpty(t, list)

	list2, _, err := repo.List(0, 0)
	require.NoError(t, err)
	require.NotEmpty(t, list2)

	role.Name = "editor2"
	require.NoError(t, repo.Update(role))

	updated, err := repo.GetByID(role.ID)
	require.NoError(t, err)
	require.Equal(t, "editor2", updated.Name)

	require.NoError(t, repo.Delete(role.ID))
	_, err = repo.GetByID(role.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRoleRepository_ReplacePermissions(t *testing.T) {
	db := openTestDB(t)
	repo := NewRoleRepository(db)

	role := &models.Role{Name: "editor"}
	require.NoError(t, repo.Create(role))

	p1, err := repo.EnsurePermission("a", "")
	require.NoError(t, err)
	p2, err := repo.EnsurePermission("b", "")
	require.NoError(t, err)

	require.NoError(t, repo.ReplacePermissions(role, []models.Permission{*p1, *p2}))

	reloaded, err := repo.GetByID(role.ID)
	require.NoError(t, err)
	require.Len(t, reloaded.Permissions, 2)
}
