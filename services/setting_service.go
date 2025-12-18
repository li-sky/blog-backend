package services

import (
	"blog-backend/models"
	"errors"
	"strings"
)

// SettingService coordinates configuration management.
type SettingService struct {
	repo SettingRepository
}

func NewSettingService(repo SettingRepository) *SettingService {
	return &SettingService{repo: repo}
}

func (s *SettingService) ListSettings() ([]models.Setting, error) {
	return s.repo.List()
}

func (s *SettingService) GetByKey(key string) (*models.Setting, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("setting key is required")
	}
	return s.repo.GetByKey(key)
}

func (s *SettingService) UpdateSetting(operator *models.User, key, value, description string) (*models.Setting, error) {
	if !userHasPermission(operator, "manage_settings") {
		return nil, errors.New("permission denied")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("setting key is required")
	}
	return s.repo.Upsert(key, value, description)
}

func (s *SettingService) CreateOrUpdate(key, value string) (*models.Setting, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("setting key is required")
	}
	return s.repo.Upsert(key, value, "")
}
