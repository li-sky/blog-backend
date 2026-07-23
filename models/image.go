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

	// 上传时预生成的固定 WebP 变体。文件名由原图文件名确定，
	// 这样文章 Markdown 只需保存原图 URL。
	WebP600Filename  string `json:"-" gorm:"type:varchar(255)"`
	WebP600URL       string `json:"webp600Url,omitempty" gorm:"type:varchar(500)"`
	WebP1200Filename string `json:"-" gorm:"type:varchar(255)"`
	WebP1200URL      string `json:"webp1200Url,omitempty" gorm:"type:varchar(500)"`

	// 只用于 API 响应，不持久化。
	VariantsComplete bool `json:"variantsComplete" gorm:"-"`

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
