package services

import (
	"blog-backend/models"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type settingRepoStub struct{}

func (settingRepoStub) List() ([]models.Setting, error) { return nil, nil }
func (settingRepoStub) GetByKey(key string) (*models.Setting, error) {
	return nil, errors.New("not found")
}
func (settingRepoStub) Upsert(key, value, description string) (*models.Setting, error) {
	return &models.Setting{Key: key, Value: value, Desc: description}, nil
}

func TestSettingService_UpdateSetting_PermissionDenied(t *testing.T) {
	operator := &models.User{Model: gorm.Model{ID: 1}}
	svc := NewSettingService(settingRepoStub{})

	_, err := svc.UpdateSetting(operator, "site_title", "x", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}
