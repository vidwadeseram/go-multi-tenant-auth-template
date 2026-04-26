package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/models"
	"gorm.io/gorm"
)

type RBACRepository struct {
	db *gorm.DB
}

func NewRBACRepository(db *gorm.DB) *RBACRepository {
	return &RBACRepository{db: db}
}

func (r *RBACRepository) DB() *gorm.DB {
	return r.db
}

func (r *RBACRepository) UserHasPermission(ctx context.Context, userID uuid.UUID, permissionName string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ? AND permissions.name = ?", userID, permissionName).
		Count(&count).Error
	return count > 0, err
}

func (r *RBACRepository) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	var perms []models.Permission
	err := r.db.WithContext(ctx).Order("name").Find(&perms).Error
	return perms, err
}

func (r *RBACRepository) ListRoles(ctx context.Context) ([]models.Role, error) {
	var roles []models.Role
	err := r.db.WithContext(ctx).Order("name").Find(&roles).Error
	return roles, err
}

func (r *RBACRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.Permission, error) {
	var perms []models.Permission
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&perms).Error
	return perms, err
}

func (r *RBACRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error) {
	var perms []models.Permission
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Find(&perms).Error
	return perms, err
}

func (r *RBACRepository) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO user_roles (user_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		userID, roleID,
	).Error
}

func (r *RBACRepository) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		roleID, permissionID,
	).Error
}
