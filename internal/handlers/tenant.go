package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	group.POST("/create", h.CreateTenant)
	group.GET("/list", h.ListMyTenants)
	group.GET("/:tenant_id", h.GetTenant)
	group.PATCH("/:tenant_id", h.UpdateTenant)
	group.DELETE("/:tenant_id", h.DeleteTenant)
	group.GET("/:tenant_id/members", h.ListMembers)
	group.POST("/:tenant_id/invitations", h.InviteMember)
	group.POST("/:tenant_id/invitations/accept", h.AcceptInvitation)
	group.PATCH("/:tenant_id/members/:user_id/role", h.UpdateMemberRole)
	group.DELETE("/:tenant_id/members/:user_id", h.RemoveMember)
}

func (h *TenantHandler) getCurrentUser(c *gin.Context) (*models.User, bool) {
	userValue, ok := c.Get(middleware.ContextUserKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "AUTHENTICATION_REQUIRED", Message: "Authentication required."}})
		return nil, false
	}
	user, ok := userValue.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Unexpected error."}})
		return nil, false
	}
	return user, true
}

func (h *TenantHandler) requireMembership(c *gin.Context, tenantID, userID uuid.UUID) (*models.TenantMember, bool) {
	member, err := h.tenantRepo.GetMembership(c.Request.Context(), tenantID, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "FORBIDDEN", Message: "You are not a member of this tenant."}})
		return nil, false
	}
	return member, true
}

func (h *TenantHandler) requireTenantAdmin(c *gin.Context, tenantID, userID uuid.UUID) bool {
	member, ok := h.requireMembership(c, tenantID, userID)
	if !ok {
		return false
	}
	var role models.Role
	if err := h.tenantRepo.DB().WithContext(c.Request.Context()).First(&role, "id = ?", member.RoleID).Error; err != nil {
		c.JSON(http.StatusForbidden, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "FORBIDDEN", Message: "Only tenant admins can perform this action."}})
		return false
	}
	if role.Name != "tenant_admin" && role.Name != "super_admin" {
		c.JSON(http.StatusForbidden, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "FORBIDDEN", Message: "Only tenant admins can perform this action."}})
		return false
	}
	return true
}

func (h *TenantHandler) CreateTenant(c *gin.Context) {
	user, ok := h.getCurrentUser(c)
	if !ok {
		return
	}
	var input dto.TenantCreateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}
	tenant := &models.Tenant{
		Name:    input.Name,
		Slug:    input.Slug,
		OwnerID: user.ID,
	}
	var tenantAdminRole models.Role
	if err := h.tenantRepo.DB().WithContext(c.Request.Context()).First(&tenantAdminRole, "name = ?", "tenant_admin").Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Tenant admin role not found."}})
		return
	}
	member := &models.TenantMember{
		UserID:   user.ID,
		RoleID:   tenantAdminRole.ID,
		IsActive: true,
	}
	if err := h.tenantService.CreateTenant(c.Request.Context(), tenant, member); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to create tenant."}})
		return
	}
	c.JSON(http.StatusCreated, dto.DataEnvelope{Data: tenant})
}

func (h *TenantHandler) ListMyTenants(c *gin.Context) {
	user, ok := h.getCurrentUser(c)
	if !ok {
		return
	}
	var tenants []models.Tenant
	err := h.tenantRepo.DB().WithContext(c.Request.Context()).
		Joins("JOIN tenant_members ON tenant_members.tenant_id = tenants.id").
		Where("tenant_members.user_id = ? AND tenant_members.is_active = ? AND tenants.is_active = ?", user.ID, true, true).
		Order("tenants.created_at desc").
		Find(&tenants).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to list tenants."}})
		return
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: tenants})
}

func (h *TenantHandler) GetTenant(c *gin.Context) {
	user, ok := h.getCurrentUser(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid tenant ID."}})
		return
	}
	if _, ok := h.requireMembership(c, tenantID, user.ID); !ok {
		return
	}
	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "TENANT_NOT_FOUND", Message: "Tenant not found."}})
		return
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: tenant})
}

func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	user, ok := h.getCurrentUser(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid tenant ID."}})
		return
	}
	if !h.requireTenantAdmin(c, tenantID, user.ID) {
		return
	}
	var input dto.TenantUpdateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}
	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "TENANT_NOT_FOUND", Message: "Tenant not found."}})
		return
	}
	if input.Name != nil {
		tenant.Name = *input.Name
	}
	if input.IsActive != nil {
		tenant.IsActive = *input.IsActive
	}
	if err := h.tenantRepo.DB().WithContext(c.Request.Context()).Save(tenant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to update tenant."}})
		return
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: tenant})
}

func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	user, ok := h.getCurrentUser(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid tenant ID."}})
		return
	}
	if !h.requireTenantAdmin(c, tenantID, user.ID) {
		return
	}
	h.tenantRepo.DB().WithContext(c.Request.Context()).Model(&models.Tenant{}).Where("id = ?", tenantID).Update("is_active", false)
	c.JSON(http.StatusOK, dto.MessageEnvelope{Data: dto.MessageResponse{Message: "Tenant deactivated."}})
}

type memberOut struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	UserID    uuid.UUID `json:"user_id"`
	RoleID    uuid.UUID `json:"role_id"`
	IsActive  bool      `json:"is_active"`
	JoinedAt  time.Time `json:"joined_at"`
	UserEmail string    `json:"user_email"`
	RoleName  string    `json:"role_name"`
}

func (h *TenantHandler) ListMembers(c *gin.Context) {
	user, ok := h.getCurrentUser(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid tenant ID."}})
		return
	}
	if _, ok := h.requireMembership(c, tenantID, user.ID); !ok {
		return
	}
	var members []models.TenantMember
	h.tenantRepo.DB().WithContext(c.Request.Context()).
		Preload("User").Preload("Role").
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("joined_at desc").Find(&members)
	out := make([]memberOut, len(members))
	for i, m := range members {
		out[i] = memberOut{ID: m.ID, TenantID: m.TenantID, UserID: m.UserID, RoleID: m.RoleID, IsActive: m.IsActive, JoinedAt: m.JoinedAt}
		if m.User.ID != uuid.Nil {
			out[i].UserEmail = m.User.Email
		}
		if m.Role.ID != uuid.Nil {
			out[i].RoleName = m.Role.Name
		}
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: out})
}

func (h *TenantHandler) InviteMember(c *gin.Context) {
	user, ok := h.getCurrentUser(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid tenant ID."}})
		return
	}
	if !h.requireTenantAdmin(c, tenantID, user.ID) {
		return
	}
	var input dto.TenantInviteRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}
	rawToken := uuid.New().String()
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])
	invitation := &models.TenantInvitation{
		TenantID:  tenantID,
		Email:     input.Email,
		RoleID:    input.RoleID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := h.tenantRepo.DB().WithContext(c.Request.Context()).Create(invitation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to create invitation."}})
		return
	}
	c.JSON(http.StatusCreated, dto.DataEnvelope{Data: gin.H{
		"id":          invitation.ID,
		"tenant_id":   tenantID,
		"email":       input.Email,
		"role_id":     input.RoleID,
		"expires_at":  invitation.ExpiresAt,
		"token":       rawToken,
	}})
}

func (h *TenantHandler) AcceptInvitation(c *gin.Context) {
	user, ok := h.getCurrentUser(c)
	if !ok {
		return
	}
	var input dto.TenantInviteAcceptRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}
	hash := sha256.Sum256([]byte(input.Token))
	tokenHash := hex.EncodeToString(hash[:])
	var invitation models.TenantInvitation
	if err := h.tenantRepo.DB().WithContext(c.Request.Context()).
		Where("token_hash = ? AND accepted_at IS NULL", tokenHash).First(&invitation).Error; err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVITATION_NOT_FOUND", Message: "Invitation not found or already accepted."}})
		return
	}
	if invitation.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INVITATION_EXPIRED", Message: "Invitation has expired."}})
		return
	}
	now := time.Now()
	h.tenantRepo.DB().WithContext(c.Request.Context()).Model(&invitation).Update("accepted_at", now)
	var count int64
	h.tenantRepo.DB().WithContext(c.Request.Context()).Model(&models.TenantMember{}).
		Where("tenant_id = ? AND user_id = ? AND is_active = ?", invitation.TenantID, user.ID, true).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "ALREADY_MEMBER", Message: "User is already a member of this tenant."}})
		return
	}
	member := &models.TenantMember{
		TenantID: invitation.TenantID,
		UserID:   user.ID,
		RoleID:   invitation.RoleID,
		IsActive: true,
	}
	if err := h.tenantRepo.DB().WithContext(c.Request.Context()).Create(member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to accept invitation."}})
		return
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: gin.H{"message": "Invitation accepted.", "member_id": member.ID}})
}

func (h *TenantHandler) UpdateMemberRole(c *gin.Context) {
	user, ok := h.getCurrentUser(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid tenant ID."}})
		return
	}
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid user ID."}})
		return
	}
	if !h.requireTenantAdmin(c, tenantID, user.ID) {
		return
	}
	var input dto.TenantMemberRoleUpdateRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}
	result := h.tenantRepo.DB().WithContext(c.Request.Context()).
		Model(&models.TenantMember{}).
		Where("tenant_id = ? AND user_id = ? AND is_active = ?", tenantID, userID, true).
		Update("role_id", input.RoleID)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "MEMBER_NOT_FOUND", Message: "Member not found."}})
		return
	}
	c.JSON(http.StatusOK, dto.MessageEnvelope{Data: dto.MessageResponse{Message: "Member role updated."}})
}

func (h *TenantHandler) RemoveMember(c *gin.Context) {
	user, ok := h.getCurrentUser(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid tenant ID."}})
		return
	}
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid user ID."}})
		return
	}
	if !h.requireTenantAdmin(c, tenantID, user.ID) {
		return
	}
	result := h.tenantRepo.DB().WithContext(c.Request.Context()).
		Model(&models.TenantMember{}).
		Where("tenant_id = ? AND user_id = ? AND is_active = ?", tenantID, userID, true).
		Update("is_active", false)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "MEMBER_NOT_FOUND", Message: "Member not found."}})
		return
	}
	c.JSON(http.StatusOK, dto.MessageEnvelope{Data: dto.MessageResponse{Message: "Member removed from tenant."}})
}
