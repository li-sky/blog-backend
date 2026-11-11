package models

import (
	"time"

	"gorm.io/gorm"
)

type PostDraft struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Body    string `json:"body"`
}

type Post struct {
	gorm.Model

	Title   string `json:"title" gorm:"type:varchar(255)"`
	Summary string `json:"summary" gorm:"type:text"`
	Body    string `json:"body" gorm:"type:text"`

	UserID uint `json:"userId" gorm:"not null;uniqueIndex:idx_post_create_request"`
	User   User `json:"user" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	Status               string     `json:"status" gorm:"type:varchar(50);default:'draft';not null"`
	PublishedAt          *time.Time `json:"publishedAt"`
	PublicationTimeKnown bool       `json:"publicationTimeKnown" gorm:"not null;default:false"`
	PublishedUpdatedAt   *time.Time `json:"publishedUpdatedAt,omitempty"`
	Draft                *PostDraft `json:"-" gorm:"serializer:json;type:jsonb"`
	DraftVersion         uint64     `json:"-" gorm:"not null;default:0"`
	DraftUpdatedAt       *time.Time `json:"-"`
	ClientRequestID      *string    `json:"-" gorm:"size:100;uniqueIndex:idx_post_create_request"`

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

	IPAddress string `json:"-" gorm:"type:varchar(45)"`
	UserAgent string `json:"userAgent" gorm:"type:varchar(255)"`
	Country   string `json:"country" gorm:"type:varchar(100)"`
}
