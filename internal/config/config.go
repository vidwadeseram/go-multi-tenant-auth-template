package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Settings struct {
	DBHost                 string
	DBPort                 int
	DBUser                 string
	DBPassword             string
	DBName                 string
	DBSSLMode              string
	JWTSecret              string
	JWTAccessExpireMinutes int
	JWTRefreshExpireDays   int
	JWTAlgorithm           string
	SMTPHost               string
	SMTPPort               int
	SMTPUser               string
	SMTPPassword           string
	SMTPSender             string
	AppPort                int
	MultiTenantMode        string
}

func Load() (*Settings, error) {
	v := viper.New()
	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for _, file := range []string{".env", ".env.example"} {
		v.SetConfigFile(file)
		if err := v.MergeInConfig(); err == nil {
			break
		}
	}

	v.SetDefault("DB_HOST", "db")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_PASSWORD", "postgres")
	v.SetDefault("DB_NAME", "authdb")
	v.SetDefault("DB_SSLMODE", "disable")
	v.SetDefault("JWT_ACCESS_EXPIRE_MINUTES", 15)
	v.SetDefault("JWT_REFRESH_EXPIRE_DAYS", 7)
	v.SetDefault("JWT_ALGORITHM", "HS256")
	v.SetDefault("SMTP_HOST", "mailhog")
	v.SetDefault("SMTP_PORT", 1025)
	v.SetDefault("SMTP_SENDER", "no-reply@example.com")
	v.SetDefault("APP_PORT", 8000)
	v.SetDefault("MULTI_TENANT_MODE", "row")

	settings := &Settings{
		DBHost:                 v.GetString("DB_HOST"),
		DBPort:                 v.GetInt("DB_PORT"),
		DBUser:                 v.GetString("DB_USER"),
		DBPassword:             v.GetString("DB_PASSWORD"),
		DBName:                 v.GetString("DB_NAME"),
		DBSSLMode:              v.GetString("DB_SSLMODE"),
		JWTSecret:              v.GetString("JWT_SECRET"),
		JWTAccessExpireMinutes: v.GetInt("JWT_ACCESS_EXPIRE_MINUTES"),
		JWTRefreshExpireDays:   v.GetInt("JWT_REFRESH_EXPIRE_DAYS"),
		JWTAlgorithm:           v.GetString("JWT_ALGORITHM"),
		SMTPHost:               v.GetString("SMTP_HOST"),
		SMTPPort:               v.GetInt("SMTP_PORT"),
		SMTPUser:               v.GetString("SMTP_USER"),
		SMTPPassword:           v.GetString("SMTP_PASSWORD"),
		SMTPSender:             v.GetString("SMTP_SENDER"),
		AppPort:                v.GetInt("APP_PORT"),
		MultiTenantMode:        strings.ToLower(v.GetString("MULTI_TENANT_MODE")),
	}

	if settings.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if settings.MultiTenantMode != "row" && settings.MultiTenantMode != "schema" {
		return nil, fmt.Errorf("MULTI_TENANT_MODE must be row or schema")
	}

	return settings, nil
}

func (s *Settings) DatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		s.DBHost,
		s.DBPort,
		s.DBUser,
		s.DBPassword,
		s.DBName,
		s.DBSSLMode,
	)
}
