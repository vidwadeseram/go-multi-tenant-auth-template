package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/dto"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/middleware"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/models"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/repository"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/services"
)

type AdminHandler struct {
	rbacRepo     *repository.RBACRepository
	authService  *services.AuthService
	tokenService *services.TokenService
}

func NewAdminHandler(rbacRepo *repository.RBACRepository, authService *services.AuthService, tokenService *services.TokenService) *AdminHandler {
	return &AdminHandler{rbacRepo: rbacRepo, authService: authService, tokenService: tokenService}
}

func (h *AdminHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, _ gin.HandlerFunc) {
	group := router.Group("/admin")
	group.Use(authMiddleware)
	group.GET("/roles", h.ListRoles)
	group.GET("/permissions", h.ListPermissions)
	group.GET("/roles/:role_id/permissions", h.GetRolePermissions)
	group.POST("/roles/permissions", h.AssignPermissionToRole)
	group.GET("/users", h.ListUsers)
	group.GET("/users/:user_id", h.GetUser)
	group.DELETE("/users/:user_id", h.DeleteUser)
	group.GET("/users/:user_id/permissions", h.GetUserPermissions)
	group.POST("/users/roles", h.AssignRoleToUser)
}

func (h *AdminHandler) requirePermission(c *gin.Context, permissionName string) (uuid.UUID, bool) {
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
	has, err := h.rbacRepo.UserHasPermission(c.Request.Context(), user.ID, permissionName)
	if err != nil || !has {
		c.JSON(http.StatusForbidden, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "FORBIDDEN", Message: "Permission '" + permissionName + "' is required."}})
		return uuid.Nil, false
	}
	return user.ID, true
}

func (h *AdminHandler) ListRoles(c *gin.Context) {
	if _, ok := h.requirePermission(c, "roles.manage"); !ok {
		return
	}
	roles, err := h.rbacRepo.ListRoles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to list roles."}})
		return
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: roles})
}

func (h *AdminHandler) ListPermissions(c *gin.Context) {
	if _, ok := h.requirePermission(c, "roles.manage"); !ok {
		return
	}
	perms, err := h.rbacRepo.ListPermissions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to list permissions."}})
		return
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: perms})
}

func (h *AdminHandler) GetRolePermissions(c *gin.Context) {
	if _, ok := h.requirePermission(c, "roles.manage"); !ok {
		return
	}
	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid role ID."}})
		return
	}
	perms, err := h.rbacRepo.GetRolePermissions(c.Request.Context(), roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to get role permissions."}})
		return
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: perms})
}

func (h *AdminHandler) AssignPermissionToRole(c *gin.Context) {
	if _, ok := h.requirePermission(c, "roles.manage"); !ok {
		return
	}
	var input dto.RolePermissionRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}
	if err := h.rbacRepo.AssignPermissionToRole(c.Request.Context(), input.RoleID, input.PermissionID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to assign permission."}})
		return
	}
	c.JSON(http.StatusOK, dto.MessageEnvelope{Data: dto.MessageResponse{Message: "Permission assigned to role."}})
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	if _, ok := h.requirePermission(c, "users.read"); !ok {
		return
	}
	var users []models.User
	if err := h.rbacRepo.DB().WithContext(c.Request.Context()).Order("created_at desc").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to list users."}})
		return
	}
	out := make([]dto.UserData, len(users))
	for i, u := range users {
		out[i] = dto.NewUserData(u)
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: out})
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	if _, ok := h.requirePermission(c, "users.read"); !ok {
		return
	}
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid user ID."}})
		return
	}
	user, err := h.authService.Me(c.Request.Context(), userID)
	if err != nil {
		middleware.WriteErrorShim(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.UserEnvelope{Data: dto.UserResponse{User: dto.NewUserData(*user)}})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	if _, ok := h.requirePermission(c, "users.delete"); !ok {
		return
	}
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid user ID."}})
		return
	}
	if err := h.rbacRepo.DB().WithContext(c.Request.Context()).Delete(&models.User{}, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to delete user."}})
		return
	}
	c.JSON(http.StatusOK, dto.MessageEnvelope{Data: dto.MessageResponse{Message: "User deleted."}})
}

func (h *AdminHandler) GetUserPermissions(c *gin.Context) {
	if _, ok := h.requirePermission(c, "users.read"); !ok {
		return
	}
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "BAD_REQUEST", Message: "Invalid user ID."}})
		return
	}
	perms, err := h.rbacRepo.GetUserPermissions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to get user permissions."}})
		return
	}
	c.JSON(http.StatusOK, dto.DataEnvelope{Data: perms})
}

func (h *AdminHandler) AssignRoleToUser(c *gin.Context) {
	if _, ok := h.requirePermission(c, "roles.manage"); !ok {
		return
	}
	var input dto.UserRoleRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}
	if err := h.rbacRepo.AssignRoleToUser(c.Request.Context(), input.UserID, input.RoleID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorEnvelope{Error: dto.ErrorResponse{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to assign role."}})
		return
	}
	c.JSON(http.StatusOK, dto.MessageEnvelope{Data: dto.MessageResponse{Message: "Role assigned to user."}})
}
