package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/apperrors"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/models"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/services"
)

type TenantMiddleware struct {
	tenantService *services.TenantService
}

func NewTenantMiddleware(tenantService *services.TenantService) *TenantMiddleware {
	return &TenantMiddleware{tenantService: tenantService}
}

func (m *TenantMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		userValue, ok := c.Get(ContextUserKey)
		if !ok {
			c.Next()
			return
		}
		user, ok := userValue.(*models.User)
		if !ok {
			writeError(c, apperrors.New(500, "TENANT_CONTEXT_ERROR", "Tenant context could not be established."))
			return
		}

		var claimTenant string
		if claimsValue, ok := c.Get(ContextClaimsKey); ok {
			if claims, ok := claimsValue.(*services.TokenPayload); ok && claims.TenantID != nil {
				claimTenant = strings.TrimSpace(*claims.TenantID)
			}
		}
		headerTenant := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))

		var tenantSlug string
		switch {
		case claimTenant != "" && headerTenant != "" && claimTenant != headerTenant:
			writeError(c, apperrors.New(403, "TENANT_MISMATCH", "Tenant header does not match token."))
			return
		case claimTenant != "":
			tenantSlug = claimTenant
		case headerTenant != "":
			tenantSlug = headerTenant
		}
		if tenantSlug == "" {
			c.Next()
			return
		}

		tenantContext, err := m.tenantService.ResolveForUser(c.Request.Context(), user.ID, tenantSlug)
		if err != nil {
			writeError(c, err)
			return
		}
		if tenantContext.Tenant != nil {
			c.Set(ContextTenantKey, tenantContext.Tenant)
		}
		if tenantContext.DB != nil {
			c.Set(ContextTenantDBKey, tenantContext.DB)
		}
		c.Next()
	}
}
