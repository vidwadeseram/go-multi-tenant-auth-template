package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/models"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", strings.ToLower(strings.TrimSpace(email))).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetActiveByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = ?", id, true).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}
