package database

import (
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/config"
)

type TenantResolverManager struct {
	db              *gorm.DB
	settings        *config.Settings
	mu              sync.Mutex
	registeredNames map[string]struct{}
}

func ConfigureTenantResolver(db *gorm.DB, settings *config.Settings) (*TenantResolverManager, error) {
	plugin := dbresolver.Register(dbresolver.Config{
		Sources:           []gorm.Dialector{postgres.Open(settings.DatabaseDSN())},
		Replicas:          []gorm.Dialector{postgres.Open(settings.DatabaseDSN())},
		TraceResolverMode: true,
	})

	if err := db.Use(plugin); err != nil {
		return nil, err
	}

	return &TenantResolverManager{
		db:              db,
		settings:        settings,
		registeredNames: map[string]struct{}{"default": {}},
	}, nil
}

func (m *TenantResolverManager) WithTenant(tenantID uuid.UUID, tenantSlug string) (*gorm.DB, error) {
	if m.settings.MultiTenantMode == "row" || tenantID == uuid.Nil {
		return m.db.Scopes(TenantScope(tenantID)), nil
	}

	if !isSafeResolverSlug(tenantSlug) {
		return nil, fmt.Errorf("invalid tenant slug")
	}
	resolverName := resolverNameForSlug(tenantSlug)
	if err := m.ensureSchemaResolver(resolverName); err != nil {
		return nil, err
	}

	return m.db.Clauses(dbresolver.Use(resolverName)), nil
}

func isSafeResolverSlug(s string) bool {
	cleaned := strings.ToLower(strings.TrimSpace(s))
	if cleaned == "" || len(cleaned) > 63 {
		return false
	}
	for _, r := range cleaned {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func (m *TenantResolverManager) ensureSchemaResolver(resolverName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.registeredNames[resolverName]; ok {
		return nil
	}

	dsn := schemaDSN(m.settings, resolverName)
	plugin := dbresolver.Register(dbresolver.Config{
		Sources:           []gorm.Dialector{postgres.Open(dsn)},
		Replicas:          []gorm.Dialector{postgres.Open(dsn)},
		TraceResolverMode: true,
	}, resolverName)

	if err := m.db.Use(plugin); err != nil {
		return err
	}

	m.registeredNames[resolverName] = struct{}{}
	return nil
}

func TenantScope(tenantID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if tenantID == uuid.Nil {
			return tx
		}
		return tx.Where("tenant_id = ?", tenantID)
	}
}

func resolverNameForSlug(slug string) string {
	cleaned := strings.ToLower(strings.TrimSpace(slug))
	cleaned = strings.ReplaceAll(cleaned, "-", "_")
	return fmt.Sprintf("tenant_%s", cleaned)
}

func schemaDSN(settings *config.Settings, schema string) string {
	searchPath := fmt.Sprintf("%s,public", schema)
	return fmt.Sprintf("%s search_path=%s", settings.DatabaseDSN(), searchPath)
}
