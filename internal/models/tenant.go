package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Tenant struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Name     string    `json:"name" gorm:"size:255;not null"`
	Slug     string    `json:"slug" gorm:"size:100;uniqueIndex;not null"`
	OwnerID  uuid.UUID `json:"owner_id" gorm:"type:uuid;not null"`
	IsActive bool      `json:"is_active" gorm:"not null;default:true"`
	Owner    User      `json:"-" gorm:"foreignKey:OwnerID"`
	TimestampModel
}

func (t *Tenant) BeforeCreate(_ *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
