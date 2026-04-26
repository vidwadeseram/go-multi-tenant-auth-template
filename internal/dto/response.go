package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/models"
)

type ErrorEnvelope struct {
	Error ErrorResponse `json:"error"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MessageEnvelope struct {
	Data MessageResponse `json:"data"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type UserEnvelope struct {
	Data UserResponse `json:"data"`
}

type AuthUserEnvelope struct {
	Data RegisterResponse `json:"data"`
}

type RegisterResponse struct {
	User    UserData `json:"user"`
	Message string   `json:"message"`
}

type TokenEnvelope struct {
	Data TokenResponse `json:"data"`
}

type TokenResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	TokenType    string  `json:"token_type"`
	ExpiresIn    int     `json:"expires_in"`
	TenantID     *string `json:"tenant_id,omitempty"`
}

type UserResponse struct {
	User UserData `json:"user"`
}

type UserData struct {
	ID         uuid.UUID `json:"id"`
	Email      string    `json:"email"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	IsActive   bool      `json:"is_active"`
	IsVerified bool      `json:"is_verified"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewUserData(user models.User) UserData {
	return UserData{
		ID:         user.ID,
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		IsActive:   user.IsActive,
		IsVerified: user.IsVerified,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}
}

type DataEnvelope struct {
	Data interface{} `json:"data"`
}
