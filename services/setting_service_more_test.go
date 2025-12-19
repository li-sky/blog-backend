package services

import (
	"blog-backend/models"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type settingRepoRecorder struct {
	upserts int
}

func (s *settingRepoRecorder) List() ([]models.Setting, error) { return nil, nil }
func (s *settingRepoRecorder) GetByKey(key string) (*models.Setting, error) {
	return &models.Setting{Key: key, Value: ""}, nil
}
func (s *settingRepoRecorder) Upsert(key, value, description string) (*models.Setting, error) {
	s.upserts++
	return &models.Setting{ID: 1, Key: key, Value: value, Desc: description}, nil
}

func TestSettingService_CreateOrUpdate_ValidationAndSuccess(t *testing.T) {
	repo := &settingRepoRecorder{}
	svc := NewSettingService(repo)

	_, err := svc.CreateOrUpdate("", "x")
	require.Error(t, err)

	setting, err := svc.CreateOrUpdate(" site_title ", "Hello")
	require.NoError(t, err)
	require.Equal(t, "site_title", setting.Key)
	require.Equal(t, "Hello", setting.Value)
	require.Equal(t, 1, repo.upserts)
}

func TestSettingService_UpdateSetting_Success(t *testing.T) {
	repo := &settingRepoRecorder{}
	svc := NewSettingService(repo)

	operator := &models.User{Model: gorm.Model{ID: 1}, Roles: []models.Role{{Permissions: []models.Permission{{Name: "manage_settings"}}}}}
	setting, err := svc.UpdateSetting(operator, "site_title", "Hello", "d")
	require.NoError(t, err)
	require.Equal(t, "site_title", setting.Key)
	require.Equal(t, "Hello", setting.Value)
}
