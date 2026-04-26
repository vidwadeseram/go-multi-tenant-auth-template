package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantInvitation struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID   uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Email      string     `json:"email" gorm:"size:255;not null;index"`
	RoleID     uuid.UUID  `json:"role_id" gorm:"type:uuid;not null"`
	TokenHash  string     `json:"-" gorm:"size:255;not null;uniqueIndex"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"not null"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at" gorm:"not null;default:now()"`
	Tenant     Tenant     `json:"-" gorm:"foreignKey:TenantID"`
	Role       Role       `json:"-" gorm:"foreignKey:RoleID"`
}

func (i *TenantInvitation) BeforeCreate(_ *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}
