package repositories

import (
	"blog-backend/models"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(t.Name())
	dsn := "file:" + name + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Setting{}, &models.User{}, &models.Role{}, &models.Permission{}, &models.Post{}, &models.Comment{}))
	return db
}

func TestSettingRepository_Upsert_CreateThenUpdate(t *testing.T) {
	db := openTestDB(t)
	repo := NewSettingRepository(db)

	created, err := repo.Upsert("site_title", "Hello", "desc")
	require.NoError(t, err)
	require.Equal(t, "site_title", created.Key)
	require.Equal(t, "Hello", created.Value)
	require.Equal(t, "desc", created.Desc)

	updated, err := repo.Upsert("site_title", "World", "")
	require.NoError(t, err)
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, "World", updated.Value)
	require.Equal(t, "desc", updated.Desc)
}
