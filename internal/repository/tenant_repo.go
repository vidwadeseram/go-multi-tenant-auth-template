package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/models"
)

type TenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) DB() *gorm.DB {
	return r.db
}

func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := r.db.WithContext(ctx).Where("slug = ? AND is_active = ?", slug, true).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	err := r.db.WithContext(ctx).Where("id = ? AND is_active = ?", id, true).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepository) GetMembership(ctx context.Context, tenantID, userID uuid.UUID) (*models.TenantMember, error) {
	var member models.TenantMember
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND is_active = ?", tenantID, userID, true).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *TenantRepository) CreateTenant(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

func (r *TenantRepository) AddMember(ctx context.Context, member *models.TenantMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}
