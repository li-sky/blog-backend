package models

import (
	"gorm.io/gorm"
)

type Image struct {
	gorm.Model

	// 存储的文件名（UUID + 扩展名）
	Filename string `json:"filename" gorm:"type:varchar(255);not null;uniqueIndex"`

	// 原始文件名
	OriginalName string `json:"originalName" gorm:"type:varchar(255);not null"`

	// 图片描述文本
	Alt string `json:"alt" gorm:"type:varchar(500)"`

	// MIME 类型
	MimeType string `json:"mimeType" gorm:"type:varchar(100);not null"`

	// 文件大小（字节）
	Size int64 `json:"size" gorm:"not null"`

	// 图片宽度
	Width int `json:"width"`

	// 图片高度
	Height int `json:"height"`

	// 完整的访问 URL
	URL string `json:"url" gorm:"type:varchar(500);not null"`

	// 上传者 ID
	UserID uint `json:"userId" gorm:"not null;index"`
	User   User `json:"user" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	// 存储后端类型（filesystem, s3, oss 等）
	StorageBackend string `json:"storageBackend" gorm:"type:varchar(50);not null"`

	// 存储后端的实际路径
	StoragePath string `json:"storagePath" gorm:"type:varchar(500);not null"`

	// 上传时的 IP 地址
	IPAddress string `json:"-" gorm:"type:varchar(45)"`
}
