package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Name        string    `json:"name" gorm:"size:50;uniqueIndex;not null"`
	Description string    `json:"description" gorm:"size:255;not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (r *Role) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
