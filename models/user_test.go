package models

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openModelDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(t.Name())
	dsn := "file:" + name + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Role{}, &Permission{}))
	return db
}

func TestUser_BeforeSave_HashesPassword(t *testing.T) {
	db := openModelDB(t)
	u := &User{Username: "alice_hash", Email: "a_hash@example.com", Password: "veryverysecure"}
	require.NoError(t, db.Create(u).Error)
	require.NotEmpty(t, u.Password)
	require.NotEqual(t, "veryverysecure", u.Password)
	require.True(t, u.CheckPassword("veryverysecure"))
}

func TestUser_BeforeSave_SkipsEmpty(t *testing.T) {
	db := openModelDB(t)
	u := &User{Username: "alice_empty", Email: "a_empty@example.com", Password: ""}
	require.NoError(t, db.Create(u).Error)
	require.Equal(t, "", u.Password)
}

func TestUser_BeforeSave_SkipsAlreadyHashed(t *testing.T) {
	db := openModelDB(t)
	// bcrypt hash prefix "$2a$" and length >= 60 triggers skip.
	u := &User{Username: "alice_hashed", Email: "a_hashed@example.com", Password: "$2a$10$0123456789012345678901234567890123456789012345678901234"}
	require.NoError(t, db.Create(u).Error)
	require.Equal(t, "$2a$10$0123456789012345678901234567890123456789012345678901234", u.Password)
}

func TestUser_HasPermission_TrueFalse(t *testing.T) {
	db := openModelDB(t)

	perm := Permission{Name: "edit_post", Description: "x"}
	require.NoError(t, db.Create(&perm).Error)
	role := Role{Name: "editor", Permissions: []Permission{perm}}
	require.NoError(t, db.Create(&role).Error)

	user := &User{Username: "alice_perm", Email: "a_perm@example.com", Password: "veryverysecure", Roles: []Role{role}}
	require.NoError(t, db.Create(user).Error)

	require.True(t, user.HasPermission("edit_post", db))
	require.False(t, user.HasPermission("nope", db))
	require.False(t, (*User)(nil).HasPermission("edit_post", db))
	require.False(t, (&User{}).HasPermission("edit_post", db))
}
