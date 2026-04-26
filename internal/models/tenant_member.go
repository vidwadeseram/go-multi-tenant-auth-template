package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantMember struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID   uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	RoleID   uuid.UUID `json:"role_id" gorm:"type:uuid;not null"`
	IsActive bool      `json:"is_active" gorm:"not null;default:true"`
	JoinedAt time.Time `json:"joined_at" gorm:"not null;default:now()"`
	Tenant   Tenant    `json:"-" gorm:"foreignKey:TenantID"`
	User     User      `json:"-" gorm:"foreignKey:UserID"`
	Role     Role      `json:"-" gorm:"foreignKey:RoleID"`
}

func (m *TenantMember) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
