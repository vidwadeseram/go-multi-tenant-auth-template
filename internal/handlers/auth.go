package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/dto"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/middleware"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/models"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, tenantMiddleware gin.HandlerFunc) {
	group := router.Group("/auth")
	group.POST("/register", h.register)
	group.POST("/login", h.login)
	group.POST("/logout", h.logout)
	group.POST("/refresh", h.refresh)
	group.GET("/me", authMiddleware, tenantMiddleware, h.me)
}

func (h *AuthHandler) register(c *gin.Context) {
	var payload dto.RegisterRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), payload)
	if err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.AuthUserEnvelope{Data: dto.RegisterResponse{User: dto.NewUserData(*user), Message: "Registration successful. Verification email sent."}})
}

func (h *AuthHandler) login(c *gin.Context) {
	var payload dto.LoginRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}
	tenantSlug := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))
	tokens, err := h.authService.Login(c.Request.Context(), payload, tenantSlug)
	if err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.TokenEnvelope{Data: *tokens})
}

func (h *AuthHandler) logout(c *gin.Context) {
	var payload dto.LogoutRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}
	if err := h.authService.Logout(c.Request.Context(), payload.RefreshToken); err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessageEnvelope{Data: dto.MessageResponse{Message: "Logout successful."}})
}

func (h *AuthHandler) refresh(c *gin.Context) {
	var payload dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}
	tokens, err := h.authService.Refresh(c.Request.Context(), payload.RefreshToken)
	if err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.TokenEnvelope{Data: *tokens})
}

func (h *AuthHandler) me(c *gin.Context) {
	userValue, ok := c.Get(middleware.ContextUserKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "AUTHENTICATION_REQUIRED", Message: "Authentication credentials were not provided."}})
		return
	}
	user, ok := userValue.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "An unexpected error occurred."}})
		return
	}
	fullUser, err := h.authService.Me(c.Request.Context(), user.ID)
	if err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.UserEnvelope{Data: dto.UserResponse{User: dto.NewUserData(*fullUser)}})
}
