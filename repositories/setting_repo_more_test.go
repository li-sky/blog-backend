package repositories

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSettingRepository_ListAndGetByKey(t *testing.T) {
	db := openTestDB(t)
	repo := NewSettingRepository(db)

	_, err := repo.GetByKey("missing")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	created, err := repo.Upsert("k", "v", "d")
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	got, err := repo.GetByKey("k")
	require.NoError(t, err)
	require.Equal(t, "v", got.Value)

	items, err := repo.List()
	require.NoError(t, err)
	require.NotEmpty(t, items)
}
