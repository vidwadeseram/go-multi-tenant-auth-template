package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/apperrors"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/config"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/database"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/models"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/repository"
)

type TenantContext struct {
	Tenant *models.Tenant
	Member *models.TenantMember
	DB     *gorm.DB
}

type TenantService struct {
	db         *gorm.DB
	manager    *database.TenantResolverManager
	tenantRepo *repository.TenantRepository
	settings   *config.Settings
}

func NewTenantService(db *gorm.DB, manager *database.TenantResolverManager, tenantRepo *repository.TenantRepository, settings *config.Settings) *TenantService {
	return &TenantService{db: db, manager: manager, tenantRepo: tenantRepo, settings: settings}
}

func (s *TenantService) ResolveForUser(ctx context.Context, userID uuid.UUID, tenantSlug string) (*TenantContext, error) {
	tenantIdentifier := strings.TrimSpace(tenantSlug)
	if tenantIdentifier == "" {
		return &TenantContext{DB: s.db}, nil
	}

	var (
		tenant *models.Tenant
		err    error
	)
	if tenantID, parseErr := uuid.Parse(tenantIdentifier); parseErr == nil {
		tenant, err = s.tenantRepo.GetByID(ctx, tenantID)
	} else {
		tenant, err = s.tenantRepo.GetBySlug(ctx, tenantIdentifier)
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(404, "TENANT_NOT_FOUND", "Tenant was not found.")
		}
		return nil, err
	}
	member, err := s.tenantRepo.GetMembership(ctx, tenant.ID, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.New(403, "TENANT_ACCESS_DENIED", "Tenant membership is required.")
		}
		return nil, err
	}
	tenantDB, err := s.manager.WithTenant(tenant.ID, tenant.Slug)
	if err != nil {
		return nil, err
	}

	return &TenantContext{Tenant: tenant, Member: member, DB: tenantDB}, nil
}

func (s *TenantService) EnsureTenantSchema(ctx context.Context, tenant *models.Tenant) error {
	return s.ensureTenantSchema(s.db.WithContext(ctx), tenant)
}

func (s *TenantService) ensureTenantSchema(db *gorm.DB, tenant *models.Tenant) error {
	if s.settings.MultiTenantMode != "schema" {
		return nil
	}
	if !isSafeSlug(tenant.Slug) {
		return apperrors.New(400, "INVALID_TENANT_SLUG", "Tenant slug contains unsafe characters.")
	}
	schema := fmt.Sprintf("tenant_%s", strings.ReplaceAll(tenant.Slug, "-", "_"))
	if !isSafeIdentifier(schema) {
		return apperrors.New(400, "INVALID_TENANT_SLUG", "Tenant slug produces unsafe identifier.")
	}
	return db.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schema)).Error
}

func isSafeSlug(s string) bool {
	if s == "" || len(s) > 63 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func isSafeIdentifier(s string) bool {
	if s == "" || len(s) > 63 {
		return false
	}
	for i, r := range s {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' {
			return false
		}
		if i == 0 && (r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func (s *TenantService) CreateTenant(ctx context.Context, tenant *models.Tenant, member *models.TenantMember) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(tenant).Error; err != nil {
			return err
		}
		if err := s.ensureTenantSchema(tx, tenant); err != nil {
			return err
		}
		return tx.Create(member).Error
	})
}
