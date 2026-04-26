package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/models"
)

type TokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *TokenRepository) CreateWithTx(ctx context.Context, tx *gorm.DB, token *models.RefreshToken) error {
	return tx.WithContext(ctx).Create(token).Error
}

func (r *TokenRepository) FindActiveByHash(ctx context.Context, hash string, userID uuid.UUID) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND user_id = ? AND revoked_at IS NULL", hash, userID).
		First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *TokenRepository) Revoke(ctx context.Context, token *models.RefreshToken) error {
	now := time.Now().UTC()
	token.RevokedAt = &now
	return r.db.WithContext(ctx).Save(token).Error
}

func (r *TokenRepository) RevokeWithTx(ctx context.Context, tx *gorm.DB, token *models.RefreshToken) error {
	now := time.Now().UTC()
	token.RevokedAt = &now
	return tx.WithContext(ctx).Save(token).Error
}
