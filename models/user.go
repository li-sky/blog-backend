package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Permission struct {
	gorm.Model
	Name        string `json:"name" gorm:"type:varchar(100);uniqueIndex;not null"`
	Description string `json:"description" gorm:"type:text;not null"`
}

type Role struct {
	gorm.Model

	Name        string       `json:"name" gorm:"type:varchar(100);uniqueIndex;not null"`
	Permissions []Permission `gorm:"many2many:role_permissions;"`
}

type User struct {
	gorm.Model

	Username string `json:"username" gorm:"type:varchar(100);uniqueIndex;not null"`
	Email    string `json:"email" gorm:"type:varchar(100);uniqueIndex;not null"`
	Password string `json:"-" gorm:"type:varchar(255);not null"`

	Roles []Role `json:"roles" gorm:"many2many:user_roles;"`
}

func (u *User) HasPermission(permissionName string, db *gorm.DB) bool {
	if u == nil || u.ID == 0 {
		return false
	}

	var count int64
	if err := db.Model(&User{}).
		Joins("JOIN user_roles ON user_roles.user_id = users.id").
		Joins("JOIN role_permissions ON role_permissions.role_id = user_roles.role_id").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("users.id = ? AND permissions.name = ?", u.ID, permissionName).
		Count(&count).Error; err != nil {
		return false
	}

	return count > 0
}

func (u *User) BeforeSave(tx *gorm.DB) (err error) {
	if u.Password == "" {
		return nil
	}

	if len(u.Password) >= 60 && u.Password[:4] == "$2a$" {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
