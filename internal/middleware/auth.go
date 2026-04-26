package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/dto"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/repository"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/services"
	"gorm.io/gorm"
)

const (
	ContextUserKey     = "currentUser"
	ContextClaimsKey   = "currentClaims"
	ContextTenantKey   = "currentTenant"
	ContextTenantDBKey = "tenantDB"
)

type AuthMiddleware struct {
	tokenService *services.TokenService
	userRepo     *repository.UserRepository
}

func NewAuthMiddleware(tokenService *services.TokenService, userRepo *repository.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{tokenService: tokenService, userRepo: userRepo}
}

func (m *AuthMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "AUTHENTICATION_REQUIRED", Message: "Authentication credentials were not provided."}})
			return
		}

		payload, err := m.tokenService.Decode(parts[1], "access")
		if err != nil {
			writeError(c, err)
			return
		}

		user, err := m.userRepo.GetActiveByID(c.Request.Context(), payload.Subject)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "USER_NOT_FOUND", Message: "Authenticated user was not found."}})
				return
			}
			writeError(c, err)
			return
		}

		c.Set(ContextUserKey, user)
		c.Set(ContextClaimsKey, payload)
		c.Next()
	}
}
