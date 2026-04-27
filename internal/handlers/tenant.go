package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/apperrors"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/dto"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/middleware"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/models"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/repository"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/services"
)

type TenantHandler struct {
	tenantRepo    *repository.TenantRepository
	tenantService *services.TenantService
}

func NewTenantHandler(tenantRepo *repository.TenantRepository, tenantService *services.TenantService) *TenantHandler {
	return &TenantHandler{tenantRepo: tenantRepo, tenantService: tenantService}
}

func (h *TenantHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, _ gin.HandlerFunc) {
	group := router.Group("/tenants")
	group.Use(authMiddleware)
	group.POST("/create", h.createTenant)
	group.GET("/list", h.listTenants)
	group.GET("/:tenant_id", h.getTenant)
	group.PATCH("/:tenant_id", h.updateTenant)
	group.DELETE("/:tenant_id", h.deleteTenant)
	group.GET("/:tenant_id/members", h.listMembers)
	group.POST("/:tenant_id/invitations", h.inviteMember)
	group.POST("/:tenant_id/invitations/accept", h.acceptInvitation)
	group.PATCH("/:tenant_id/members/:user_id/role", h.updateMemberRole)
	group.DELETE("/:tenant_id/members/:user_id", h.removeMember)
}

func (h *TenantHandler) getUserID(c *gin.Context) (uuid.UUID, bool) {
	userValue, ok := c.Get(middleware.ContextUserKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "AUTHENTICATION_REQUIRED", Message: "Authentication required."}})
		return uuid.Nil, false
	}
	user, ok := userValue.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Unexpected error."}})
		return uuid.Nil, false
	}
	return user.ID, true
}

func (h *TenantHandler) requireTenantAdmin(c *gin.Context, tenantID, userID uuid.UUID) bool {
	member, err := h.tenantRepo.GetMembership(c.Request.Context(), tenantID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "FORBIDDEN", Message: "Only tenant admins can perform this action."}})
		return false
	}
	adminRole, err := h.tenantRepo.GetRoleByName(c.Request.Context(), "tenant_admin")
	if err == nil && member.RoleID == adminRole.ID {
		return true
	}
	superAdminRole, err := h.tenantRepo.GetRoleByName(c.Request.Context(), "super_admin")
	if err == nil && member.RoleID == superAdminRole.ID {
		return true
	}
	c.JSON(http.StatusForbidden, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "FORBIDDEN", Message: "Only tenant admins can perform this action."}})
	return false
}

func (h *TenantHandler) requireMembership(c *gin.Context, tenantID, userID uuid.UUID) bool {
	_, err := h.tenantRepo.GetMembership(c.Request.Context(), tenantID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "FORBIDDEN", Message: "You are not a member of this tenant."}})
		return false
	}
	return true
}

type createTenantRequest struct {
	Name string `json:"name" binding:"required"`
	Slug string `json:"slug" binding:"required"`
}

func (h *TenantHandler) createTenant(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var payload createTenantRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	adminRole, err := h.tenantRepo.GetRoleByName(c.Request.Context(), "tenant_admin")
	if err != nil {
		middleware.WriteErrorShim(c, apperrors.New(500, "INTERNAL_SERVER_ERROR", "Failed to resolve tenant admin role."))
		return
	}

	tenant := &models.Tenant{
		Name:     payload.Name,
		Slug:     payload.Slug,
		OwnerID:  userID,
		IsActive: true,
	}
	member := &models.TenantMember{
		TenantID: tenant.ID,
		UserID:   userID,
		RoleID:   adminRole.ID,
		IsActive: true,
	}

	if err := h.tenantService.CreateTenant(c.Request.Context(), tenant, member); err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.DataEnvelope{Data: tenant})
}

func (h *TenantHandler) listTenants(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	tenants, err := h.tenantRepo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	if tenants == nil {
		tenants = []models.Tenant{}
	}

	c.JSON(http.StatusOK, dto.DataEnvelope{Data: tenants})
}

func (h *TenantHandler) getTenant(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVALID_ID", Message: "Invalid tenant ID."}})
		return
	}
	if !h.requireMembership(c, tenantID, userID) {
		return
	}

	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: tenant})
}

type updateTenantRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

func (h *TenantHandler) updateTenant(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVALID_ID", Message: "Invalid tenant ID."}})
		return
	}
	if !h.requireTenantAdmin(c, tenantID, userID) {
		return
	}

	var payload updateTenantRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	updates := map[string]interface{}{}
	if payload.Name != nil {
		updates["name"] = *payload.Name
	}
	if payload.IsActive != nil {
		updates["is_active"] = *payload.IsActive
	}
	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := h.tenantRepo.UpdateTenant(c.Request.Context(), tenantID, updates); err != nil {
			middleware.WriteErrorShim(c, err)
			return
		}
	}

	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: tenant})
}

func (h *TenantHandler) deleteTenant(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVALID_ID", Message: "Invalid tenant ID."}})
		return
	}
	if !h.requireTenantAdmin(c, tenantID, userID) {
		return
	}

	if err := h.tenantRepo.SoftDelete(c.Request.Context(), tenantID); err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessageEnvelope{Data: dto.MessageResponse{Message: "Tenant deactivated."}})
}

func (h *TenantHandler) listMembers(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVALID_ID", Message: "Invalid tenant ID."}})
		return
	}
	if !h.requireMembership(c, tenantID, userID) {
		return
	}

	members, err := h.tenantRepo.ListMembers(c.Request.Context(), tenantID)
	if err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	if members == nil {
		members = []models.TenantMember{}
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: members})
}

type inviteRequest struct {
	Email  string `json:"email" binding:"required,email"`
	RoleID string `json:"role_id" binding:"required"`
}

func (h *TenantHandler) inviteMember(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVALID_ID", Message: "Invalid tenant ID."}})
		return
	}
	if !h.requireTenantAdmin(c, tenantID, userID) {
		return
	}

	var payload inviteRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	roleID, err := uuid.Parse(payload.RoleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "Invalid role_id."}})
		return
	}

	rawToken := uuid.New().String()
	tokenHash := hashToken(rawToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	invitation := &models.TenantInvitation{
		TenantID:  tenantID,
		Email:     payload.Email,
		RoleID:    roleID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	if err := h.tenantRepo.CreateInvitation(c.Request.Context(), invitation); err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.DataEnvelope{Data: gin.H{
		"id":         invitation.ID.String(),
		"tenant_id":  tenantID.String(),
		"email":      payload.Email,
		"role_id":    roleID.String(),
		"expires_at": expiresAt.Format(time.RFC3339),
		"token":      rawToken,
	}})
}

type acceptInvitationRequest struct {
	Token string `json:"token" binding:"required"`
}

func (h *TenantHandler) acceptInvitation(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVALID_ID", Message: "Invalid tenant ID."}})
		return
	}

	var payload acceptInvitationRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	tokenHash := hashToken(payload.Token)
	invitation, err := h.tenantRepo.GetInvitationByTokenHash(c.Request.Context(), tokenHash)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "NOT_FOUND", Message: "Invitation not found or already accepted."}})
		return
	}
	if invitation.TenantID != tenantID {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVALID_INVITATION", Message: "Invitation does not belong to this tenant."}})
		return
	}
	if invitation.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVITATION_EXPIRED", Message: "Invitation has expired."}})
		return
	}

	_ = h.tenantRepo.AcceptInvitation(c.Request.Context(), invitation.ID)

	member := &models.TenantMember{
		TenantID: tenantID,
		UserID:   userID,
		RoleID:   invitation.RoleID,
		IsActive: true,
	}
	if err := h.tenantRepo.AddMember(c.Request.Context(), member); err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MessageEnvelope{Data: dto.MessageResponse{Message: "Invitation accepted."}})
}

type updateRoleRequest struct {
	RoleID string `json:"role_id" binding:"required"`
}

func (h *TenantHandler) updateMemberRole(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVALID_ID", Message: "Invalid tenant ID."}})
		return
	}
	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVALID_ID", Message: "Invalid user ID."}})
		return
	}
	if !h.requireTenantAdmin(c, tenantID, userID) {
		return
	}

	var payload updateRoleRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}
	roleID, err := uuid.Parse(payload.RoleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: "Invalid role_id."}})
		return
	}

	if err := h.tenantRepo.UpdateMemberRole(c.Request.Context(), tenantID, targetUserID, roleID); err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessageEnvelope{Data: dto.MessageResponse{Message: "Member role updated."}})
}

func (h *TenantHandler) removeMember(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVALID_ID", Message: "Invalid tenant ID."}})
		return
	}
	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVALID_ID", Message: "Invalid user ID."}})
		return
	}
	if !h.requireTenantAdmin(c, tenantID, userID) {
		return
	}

	if err := h.tenantRepo.RemoveMember(c.Request.Context(), tenantID, targetUserID); err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessageEnvelope{Data: dto.MessageResponse{Message: "Member removed from tenant."}})
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
