package database

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/config"
)

func Connect(settings *config.Settings) (*gorm.DB, *TenantResolverManager, error) {
	db, err := gorm.Open(postgres.Open(settings.DatabaseDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	resolverManager, err := ConfigureTenantResolver(db, settings)
	if err != nil {
		return nil, nil, err
	}

	return db, resolverManager, nil
}
