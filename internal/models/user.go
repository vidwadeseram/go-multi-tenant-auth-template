package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Email         string         `json:"email" gorm:"size:255;uniqueIndex;not null"`
	PasswordHash  string         `json:"-" gorm:"size:255;not null"`
	FirstName     string         `json:"first_name" gorm:"size:100;not null"`
	LastName      string         `json:"last_name" gorm:"size:100;not null"`
	IsActive      bool           `json:"is_active" gorm:"not null;default:true"`
	IsVerified    bool           `json:"is_verified" gorm:"not null;default:false"`
	RefreshTokens []RefreshToken `json:"-"`
	TimestampModel
}

func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
