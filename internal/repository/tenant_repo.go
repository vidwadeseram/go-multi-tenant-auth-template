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

func (r *TenantRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Tenant, error) {
	var tenants []models.Tenant
	err := r.db.WithContext(ctx).
		Joins("JOIN tenant_members ON tenant_members.tenant_id = tenants.id").
		Where("tenant_members.user_id = ? AND tenant_members.is_active = ? AND tenants.is_active = ?", userID, true, true).
		Order("tenants.created_at DESC").
		Find(&tenants).Error
	return tenants, err
}

func (r *TenantRepository) UpdateTenant(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.Tenant{}).Where("id = ?", id).Updates(updates).Error
}

func (r *TenantRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.Tenant{}).Where("id = ?", id).Update("is_active", false).Error
}

func (r *TenantRepository) ListMembers(ctx context.Context, tenantID uuid.UUID) ([]models.TenantMember, error) {
	var members []models.TenantMember
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("joined_at DESC").
		Find(&members).Error
	return members, err
}

func (r *TenantRepository) UpdateMemberRole(ctx context.Context, tenantID, userID uuid.UUID, roleID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.TenantMember{}).
		Where("tenant_id = ? AND user_id = ? AND is_active = ?", tenantID, userID, true).
		Update("role_id", roleID).Error
}

func (r *TenantRepository) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.TenantMember{}).
		Where("tenant_id = ? AND user_id = ? AND is_active = ?", tenantID, userID, true).
		Update("is_active", false).Error
}

func (r *TenantRepository) CreateInvitation(ctx context.Context, invitation *models.TenantInvitation) error {
	return r.db.WithContext(ctx).Create(invitation).Error
}

func (r *TenantRepository) GetInvitationByTokenHash(ctx context.Context, tokenHash string) (*models.TenantInvitation, error) {
	var inv models.TenantInvitation
	err := r.db.WithContext(ctx).Where("token_hash = ? AND accepted_at IS NULL", tokenHash).First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *TenantRepository) AcceptInvitation(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.TenantInvitation{}).Where("id = ?", id).Update("accepted_at", gorm.Expr("NOW()")).Error
}

func (r *TenantRepository) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}
