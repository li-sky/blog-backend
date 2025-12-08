package models

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	gorm.Model

	Title   string `json:"title" gorm:"type:varchar(255)"`
	Summary string `json:"summary" gorm:"type:text"`
	Body    string `json:"body" gorm:"type:text"`

	UserID uint `json:"userId" gorm:"not null"`
	User   User `json:"user" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	Status      string     `json:"status" gorm:"type:varchar(50);default:'draft';not null"`
	PublishedAt *time.Time `json:"publishedAt"`

	IPAddress string `json:"ipAddress" gorm:"type:varchar(45)"`
}

type Comment struct {
	gorm.Model

	PostID uint `json:"postId" gorm:"not null"`
	Post   Post `json:"-" gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	UserID uint `json:"userId" gorm:"not null"`
	User   User `json:"user" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	Body string `json:"body" gorm:"type:text;not null"`

	Visible bool `json:"visible" gorm:"default:true;not null"`

	IPAddress string `json:"ipAddress" gorm:"type:varchar(45)"`
}
