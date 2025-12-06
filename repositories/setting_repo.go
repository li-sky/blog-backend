package repositories

import (
	"errors"

	"blog-backend/models"

	"gorm.io/gorm"
)

// SettingRepository persists key/value configuration entries.
type SettingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

func (r *SettingRepository) List() ([]models.Setting, error) {
	var settings []models.Setting
	if err := r.db.Order("key ASC").Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *SettingRepository) GetByKey(key string) (*models.Setting, error) {
	var setting models.Setting
	if err := r.db.Where("key = ?", key).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *SettingRepository) Upsert(key, value, description string) (*models.Setting, error) {
	var setting models.Setting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = models.Setting{Key: key, Value: value, Desc: description}
		if err := r.db.Create(&setting).Error; err != nil {
			return nil, err
		}
		return &setting, nil
	}
	if err != nil {
		return nil, err
	}
	setting.Value = value
	if description != "" {
		setting.Desc = description
	}
	if err := r.db.Save(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}
