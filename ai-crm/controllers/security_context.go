package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/server/web/context"
	"gorm.io/gorm"
)

type authContext struct {
	UserID      uint
	TenantID    uint
	Username    string
	Role        string
	RoleID      uint
	AccountType string
	SessionID   string
}

func isPlatformAdminIdentity(accountType, role string, tenantID uint) bool {
	if accountType == "super_admin" || accountType == "platform_super_admin" {
		return true
	}
	if role == "super_admin" || role == "platform_super_admin" {
		return true
	}
	// Backward compatibility: historical "admin" is treated as platform admin
	// only when no tenant context exists.
	if (accountType == "admin" || role == "admin") && tenantID == 0 {
		return true
	}
	return false
}

func currentAuth(ctx *context.Context) authContext {
	return authContext{
		UserID:      dataUint(ctx, "user_id"),
		TenantID:    dataUint(ctx, "tenant_id"),
		Username:    dataString(ctx, "username"),
		Role:        dataString(ctx, "role"),
		RoleID:      dataUint(ctx, "role_id"),
		AccountType: dataString(ctx, "account_type"),
		SessionID:   dataString(ctx, "session_id"),
	}
}

func dataUint(ctx *context.Context, key string) uint {
	value := ctx.Input.GetData(key)
	switch typed := value.(type) {
	case uint:
		return typed
	case int:
		if typed > 0 {
			return uint(typed)
		}
	case int64:
		if typed > 0 {
			return uint(typed)
		}
	case float64:
		if typed > 0 {
			return uint(typed)
		}
	}
	return 0
}

func dataString(ctx *context.Context, key string) string {
	value, _ := ctx.Input.GetData(key).(string)
	return value
}

func (a authContext) IsSuperAdmin() bool {
	return a.IsPlatformAdmin()
}

func (a authContext) IsPlatformAdmin() bool {
	return isPlatformAdminIdentity(a.AccountType, a.Role, a.TenantID)
}

func (a authContext) IsTenantAdmin() bool {
	if a.TenantID == 0 || a.UserID == 0 {
		return false
	}
	accountType := strings.ToLower(strings.TrimSpace(a.AccountType))
	role := strings.ToLower(strings.TrimSpace(a.Role))
	if accountType == "tenant_admin" || role == "tenant_admin" {
		return true
	}
	if utils.DB == nil {
		return false
	}
	var tenant models.Tenant
	if err := utils.DB.Select("admin_user_id").Where("id = ?", a.TenantID).First(&tenant).Error; err == nil && tenant.AdminUserID == a.UserID {
		return true
	}
	var firstUser models.User
	if err := utils.DB.Select("id").
		Where("tenant_id = ? AND (status IS NULL OR status NOT IN ?)", a.TenantID, []string{"deleted"}).
		Order("id asc").
		First(&firstUser).Error; err == nil && firstUser.ID == a.UserID {
		return true
	}
	return false
}

func (a authContext) CanManageTenantUsers() bool {
	return a.IsSuperAdmin() || a.IsTenantAdmin()
}

func requireSuperAdmin(ctx *context.Context) (authContext, bool) {
	auth := currentAuth(ctx)
	if !auth.IsSuperAdmin() {
		utils.Error(ctx, http.StatusForbidden, "仅超管可执行该操作")
		return auth, false
	}
	return auth, true
}

func requireTenantAdmin(ctx *context.Context) (authContext, bool) {
	auth := currentAuth(ctx)
	if !auth.IsTenantAdmin() {
		utils.Error(ctx, http.StatusForbidden, "仅企业管理员可执行该操作")
		return auth, false
	}
	if auth.TenantID == 0 {
		utils.Error(ctx, http.StatusForbidden, "缺少租户上下文")
		return auth, false
	}
	return auth, true
}

func requirePermission(ctx *context.Context, permission string) (authContext, bool) {
	auth := currentAuth(ctx)
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() {
		return auth, true
	}
	if permission == "" {
		utils.Error(ctx, http.StatusForbidden, "无权限访问")
		return auth, false
	}

	if !authHasPermission(auth, permission) {
		utils.Error(ctx, http.StatusForbidden, "无权限访问")
		return auth, false
	}
	return auth, true
}

func authHasPermission(auth authContext, permission string) bool {
	if auth.TenantID == 0 {
		return false
	}
	roleIDs := roleIDsForAuth(auth)
	if len(roleIDs) == 0 {
		return false
	}
	var roles []models.Role
	if err := utils.DB.Where("tenant_id = ? AND id IN ?", auth.TenantID, roleIDs).Find(&roles).Error; err != nil {
		return false
	}
	resource, action := splitPermissionToken(permission)
	for _, role := range roles {
		if permissionInCSV(role.PermissionsJSON, permission) {
			return true
		}
		// Dual-track compatibility: legacy manage keeps working for new action checks.
		if resource != "" && action != "" && action != "view" && action != "manage" {
			if permissionInCSV(role.PermissionsJSON, resource+":manage") {
				return true
			}
		}
	}
	return false
}

func authHasPermissionAny(auth authContext, permissions ...string) bool {
	for _, permission := range permissions {
		if authHasPermission(auth, permission) {
			return true
		}
	}
	return false
}

func authHasExactPermission(auth authContext, permission string) bool {
	if auth.TenantID == 0 {
		return false
	}
	roleIDs := roleIDsForAuth(auth)
	if len(roleIDs) == 0 {
		return false
	}
	var roles []models.Role
	if err := utils.DB.Where("tenant_id = ? AND id IN ?", auth.TenantID, roleIDs).Find(&roles).Error; err != nil {
		return false
	}
	for _, role := range roles {
		if permissionInCSV(role.PermissionsJSON, permission) || permissionInCSV(role.PermissionsJSON, "*") {
			return true
		}
	}
	return false
}

func authCanCancelOrder(auth authContext) bool {
	return auth.IsTenantAdmin() || authHasExactPermission(auth, "orders:manage")
}

func splitPermissionToken(permission string) (resource string, action string) {
	value := strings.TrimSpace(permission)
	idx := strings.LastIndex(value, ":")
	if idx <= 0 || idx >= len(value)-1 {
		return "", ""
	}
	return strings.TrimSpace(value[:idx]), strings.TrimSpace(value[idx+1:])
}

func roleIDsForAuth(auth authContext) []uint {
	seen := map[uint]bool{}
	roleIDs := []uint{}
	if auth.UserID != 0 {
		var userRoles []models.UserRole
		utils.DB.Where("user_id = ?", auth.UserID).Order("id asc").Find(&userRoles)
		for _, item := range userRoles {
			if item.RoleID == 0 || seen[item.RoleID] {
				continue
			}
			seen[item.RoleID] = true
			roleIDs = append(roleIDs, item.RoleID)
		}
	}
	if len(roleIDs) == 0 && auth.RoleID != 0 {
		roleIDs = append(roleIDs, auth.RoleID)
	}
	return roleIDs
}

func permissionInCSV(csvValue, permission string) bool {
	for _, item := range strings.Split(csvValue, ",") {
		if strings.TrimSpace(item) == permission {
			return true
		}
	}
	return false
}

func tenantID(ctx *context.Context) uint {
	return currentAuth(ctx).TenantID
}

func requirePlatformAdmin(ctx *context.Context) (authContext, bool) {
	return requireSuperAdmin(ctx)
}

func tenantDB(ctx *context.Context, model interface{}) *gorm.DB {
	return utils.DB.Model(model).Where("tenant_id = ?", tenantID(ctx))
}

func withTenant(ctx *context.Context, values map[string]interface{}) map[string]interface{} {
	if values == nil {
		values = map[string]interface{}{}
	}
	values["tenant_id"] = tenantID(ctx)
	return values
}
