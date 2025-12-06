package models

type Setting struct {
	ID uint `json:"id" gorm:"primaryKey"`

	Key string `json:"key" gorm:"uniqueIndex;type:varchar(50);not null"`

	Value string `json:"value" gorm:"type:text"`

	Desc string `json:"desc" gorm:"type:varchar(255)"`
}
