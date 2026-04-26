package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Description string    `gorm:"size:255;not null" json:"description"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (p *Permission) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type RolePermission struct {
	RoleID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"role_id"`
	PermissionID uuid.UUID `gorm:"type:uuid;primaryKey" json:"permission_id"`
}

type UserRole struct {
	UserID uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	RoleID uuid.UUID `gorm:"type:uuid;primaryKey" json:"role_id"`
}
