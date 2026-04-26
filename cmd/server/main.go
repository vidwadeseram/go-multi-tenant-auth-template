package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/config"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/database"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/handlers"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/mailer"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/middleware"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/repository"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/services"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	settings, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, resolverManager, err := database.Connect(settings)
	if err != nil {
		logger.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("failed to get sql db handle", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	tenantRepo := repository.NewTenantRepository(db)

	tenantService := services.NewTenantService(db, resolverManager, tenantRepo, settings)
	tokenService := services.NewTokenService(settings)
	authService := services.NewAuthService(userRepo, tokenRepo, tenantService, tokenService, mailer.New(settings), db, logger)

	authMiddleware := middleware.NewAuthMiddleware(tokenService, userRepo)
	tenantMiddleware := middleware.NewTenantMiddleware(tenantService)
	authHandler := handlers.NewAuthHandler(authService)
	healthHandler := handlers.NewHealthHandler(sqlDB)
	rbacRepo := repository.NewRBACRepository(db)
	adminHandler := handlers.NewAdminHandler(rbacRepo, authService, tokenService)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/health", healthHandler.Handle)

	router.StaticFile("/openapi.json", "./static/openapi.json")
	router.StaticFile("/docs", "./static/swagger.html")

	api := router.Group("/api/v1")
	authHandler.RegisterRoutes(api, authMiddleware.Handle(), tenantMiddleware.Handle())
	adminHandler.RegisterRoutes(api, authMiddleware.Handle(), tenantMiddleware.Handle())

	addr := fmt.Sprintf(":%d", settings.AppPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		logger.Info("starting server", "addr", addr, "multi_tenant_mode", settings.MultiTenantMode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}
	logger.Info("server exited")
}
