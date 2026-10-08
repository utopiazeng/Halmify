package middleware

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web/context"
)

func isPlatformAdminClaims(claims *utils.JWTClaims) bool {
	if claims == nil {
		return false
	}
	if claims.AccountType == "super_admin" || claims.AccountType == "platform_super_admin" {
		return true
	}
	if claims.Role == "super_admin" || claims.Role == "platform_super_admin" {
		return true
	}
	if (claims.AccountType == "admin" || claims.Role == "admin") && claims.TenantID == 0 {
		return true
	}
	return false
}

func isTenantAdminClaims(claims *utils.JWTClaims) bool {
	if claims == nil || claims.TenantID == 0 || claims.UserID == 0 {
		return false
	}
	accountType := strings.ToLower(strings.TrimSpace(claims.AccountType))
	role := strings.ToLower(strings.TrimSpace(claims.Role))
	if accountType == "tenant_admin" || role == "tenant_admin" {
		return true
	}
	if utils.DB == nil {
		return false
	}
	var tenant models.Tenant
	if err := utils.DB.Select("admin_user_id").Where("id = ?", claims.TenantID).First(&tenant).Error; err == nil && tenant.AdminUserID == claims.UserID {
		return true
	}
	var firstUser models.User
	if err := utils.DB.Select("id").
		Where("tenant_id = ? AND (status IS NULL OR status NOT IN ?)", claims.TenantID, []string{"deleted"}).
		Order("id asc").
		First(&firstUser).Error; err == nil && firstUser.ID == claims.UserID {
		return true
	}
	return false
}

// AuthMiddleware 认证中间件
type AuthMiddleware struct{}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{}
}

// CORS 跨域中间件
func CORS(ctx *context.Context) {
	ctx.Output.Header("Access-Control-Allow-Origin", "*")
	ctx.Output.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	ctx.Output.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept-Language")
	ctx.Output.Header("Access-Control-Allow-Credentials", "true")

	if ctx.Request.Method == "OPTIONS" {
		ctx.Output.SetStatus(http.StatusOK)
		return
	}
}

// RequireAuth API鉴权
func RequireAuth(ctx *context.Context) {
	path := ctx.Request.URL.Path
	if path == "/api/v1/auth/register" ||
		path == "/api/v1/auth/login" ||
		path == "/api/v1/auth/activate/password" ||
		path == "/api/v1/auth/refresh" ||
		path == "/api/v1/auth/sso/oidc" ||
		path == "/api/v1/auth/captcha/challenge" ||
		path == "/api/v1/auth/captcha/verify" {
		return
	}
	if strings.HasPrefix(path, "/api/v1/portal/") {
		return
	}
	if strings.HasPrefix(path, "/api/iot/v1/") {
		return
	}

	authHeader := ctx.Request.Header.Get("Authorization")
	if authHeader == "" && strings.HasPrefix(path, "/api/v1/im/ws") {
		queryToken := strings.TrimSpace(ctx.Input.Query("token"))
		if queryToken == "" {
			queryToken = strings.TrimSpace(ctx.Input.Query("access_token"))
		}
		if queryToken != "" {
			authHeader = "Bearer " + queryToken
		}
	}
	if authHeader == "" {
		utils.UnauthorizedKey(ctx, "auth.tokenMissing", nil)
		return
	}

	claims, err := utils.ParseToken(authHeader)
	if err != nil {
		utils.UnauthorizedKey(ctx, "auth.tokenInvalid", nil)
		return
	}
	// Backfill missing account_type from DB for historical tokens.
	if claims.AccountType == "" {
		var user models.User
		if err := utils.DB.Select("id,account_type,role,tenant_id").Where("id = ?", claims.UserID).First(&user).Error; err == nil {
			if user.AccountType != "" {
				claims.AccountType = user.AccountType
			} else if user.Role != "" {
				claims.AccountType = user.Role
			}
			if claims.Role == "" {
				claims.Role = user.Role
			}
			if claims.TenantID == 0 {
				claims.TenantID = user.TenantID
			}
		}
	}

	ctx.Input.SetData("user_id", claims.UserID)
	ctx.Input.SetData("tenant_id", claims.TenantID)
	ctx.Input.SetData("username", claims.Username)
	ctx.Input.SetData("role", claims.Role)
	ctx.Input.SetData("role_id", claims.RoleID)
	ctx.Input.SetData("account_type", claims.AccountType)
	ctx.Input.SetData("session_id", claims.SessionID)

	if !validateSession(ctx, claims) {
		return
	}
	if !authorizeRequest(ctx, claims) {
		return
	}
}

func validateSession(ctx *context.Context, claims *utils.JWTClaims) bool {
	if claims.SessionID == "" {
		utils.UnauthorizedKey(ctx, "auth.sessionExpired", nil)
		return false
	}
	var session models.UserSession
	if err := utils.DB.Where("session_id = ? AND user_id = ?", claims.SessionID, claims.UserID).First(&session).Error; err != nil {
		utils.UnauthorizedKey(ctx, "auth.sessionNotFound", nil)
		return false
	}
	if session.Revoked || time.Now().After(session.ExpiresAt) {
		utils.UnauthorizedKey(ctx, "auth.sessionExpired", nil)
		return false
	}
	utils.DB.Model(&models.UserSession{}).Where("id = ?", session.ID).Update("last_active", time.Now())
	return true
}

func authorizeRequest(ctx *context.Context, claims *utils.JWTClaims) bool {
	path := ctx.Request.URL.Path
	if !enforceMustChangePassword(ctx, claims) {
		return false
	}
	isPlatformAdmin := isPlatformAdminClaims(claims)
	if strings.HasPrefix(path, "/api/v1/platform/") {
		if isPlatformAdmin {
			return true
		}
		writeForbidden(ctx, "仅平台超管可访问平台域接口")
		return false
	}

	if isPlatformAdmin {
		if isPlatformAdminAllowedPath(path) {
			return true
		}
		if isTenantBusinessPath(path) {
			writeForbidden(ctx, "平台超管不可直接访问租户业务域")
			return false
		}
		return true
	}

	if strings.HasPrefix(path, "/api/v1/platform/") {
		writeForbidden(ctx, "无权限访问平台域接口")
		return false
	}
	if claims.TenantID == 0 {
		writeForbidden(ctx, "缺少租户上下文")
		return false
	}
	if reason, blocked := tenantBlockedByPolicy(claims.TenantID, time.Now()); blocked {
		writeForbidden(ctx, reason)
		return false
	}
	if !tenantModuleEnabled(claims.TenantID, moduleByPath(ctx.Request.URL.Path)) {
		writeForbidden(ctx, "当前租户未开通该功能模块")
		return false
	}
	if strings.HasPrefix(path, "/api/v1/system/menus") && claims.TenantID != 0 {
		// Tenant side should consume platform-assigned menu visibility only.
		// Keep sidebar loading endpoint available, block menu management/listing operations.
		visible := strings.TrimSpace(ctx.Input.Query("visible"))
		if ctx.Request.Method != http.MethodGet || visible != "1" {
			writeForbidden(ctx, "租户不允许管理菜单")
			return false
		}
		return true
	}
	if isTenantAdminClaims(claims) {
		return true
	}
	if isTenantAdminOnlyPath(ctx.Request.Method, path, ctx.Input.Query("visible")) {
		writeForbidden(ctx, "仅企业管理员可执行该操作")
		return false
	}

	permission := permissionByPath(ctx.Request.Method, ctx.Request.URL.Path)
	if permission == "" {
		return true
	}
	if manufacturingReadPermissionGrantedByPackage(claims.TenantID, permission) {
		return true
	}
	permissions := userPermissionSet(claims.UserID, claims.TenantID, claims.RoleID)
	if len(permissions) == 0 {
		writeForbidden(ctx, "无权限访问")
		return false
	}
	if !permissionGranted(permissions, permission) {
		writeForbidden(ctx, "无权限访问")
		return false
	}
	return true
}

func isTenantAdminOnlyPath(method, path, visible string) bool {
	if path == "/api/v1/accounts/me/language" || strings.HasPrefix(path, "/api/v1/accounts/me/") {
		return false
	}
	if path == "/api/v1/accounts/users" && method == http.MethodGet {
		return false
	}
	if strings.HasPrefix(path, "/api/v1/system/menus") && method == http.MethodGet && strings.TrimSpace(visible) == "1" {
		return false
	}
	if strings.HasPrefix(path, "/api/v1/rbac/") ||
		strings.HasPrefix(path, "/api/v1/system/departments") ||
		strings.HasPrefix(path, "/api/v1/system/posts") ||
		strings.HasPrefix(path, "/api/v1/system/menus") ||
		strings.HasPrefix(path, "/api/v1/operation-logs") ||
		strings.HasPrefix(path, "/api/v1/iot/") ||
		strings.HasPrefix(path, "/api/v1/storage/") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/accounts/users") {
		return true
	}
	return false
}

func enforceMustChangePassword(ctx *context.Context, claims *utils.JWTClaims) bool {
	var user models.User
	if err := utils.DB.Select("id,must_change_password").Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		utils.UnauthorizedKey(ctx, "auth.sessionInvalid", nil)
		return false
	}
	if !user.MustChangePassword {
		return true
	}
	path := ctx.Request.URL.Path
	if path == "/api/v1/accounts/me/password" || path == "/api/v1/auth/logout" || path == "/api/v1/auth/me" {
		return true
	}
	utils.ErrorKeyWithData(ctx, http.StatusForbidden, "auth.passwordChangeRequired", nil, map[string]interface{}{"error_code": "PASSWORD_CHANGE_REQUIRED"})
	return false
}

func writeForbidden(ctx *context.Context, message string) {
	data := interface{}(nil)
	key := ""
	if message == "TENANT_DISABLED" || message == "TENANT_EXPIRED" {
		data = map[string]interface{}{"error_code": message}
		if message == "TENANT_DISABLED" {
			key = "auth.tenantDisabled"
		} else {
			key = "auth.tenantExpired"
		}
	}
	if key == "" {
		key, _ = utils.ResolveErrorMessageKey(message, http.StatusForbidden)
	}
	utils.ErrorKeyWithLegacy(ctx, http.StatusForbidden, key, nil, message, data)
}

func permissionInCSV(csvValue, permission string) bool {
	for _, item := range strings.Split(csvValue, ",") {
		if strings.TrimSpace(item) == permission {
			return true
		}
	}
	return false
}

func userPermissionSet(userID, tenantID, fallbackRoleID uint) map[string]bool {
	roleIDs := roleIDsForUser(userID, fallbackRoleID)
	permissions := map[string]bool{}
	if tenantID == 0 || len(roleIDs) == 0 {
		return permissions
	}
	var roles []models.Role
	if err := utils.DB.Where("tenant_id = ? AND id IN ?", tenantID, roleIDs).Find(&roles).Error; err != nil {
		return permissions
	}
	for _, role := range roles {
		for _, item := range strings.Split(role.PermissionsJSON, ",") {
			permission := strings.TrimSpace(item)
			if permission != "" {
				permissions[permission] = true
			}
		}
	}
	return permissions
}

func roleIDsForUser(userID, fallbackRoleID uint) []uint {
	seen := map[uint]bool{}
	roleIDs := []uint{}
	if userID != 0 {
		var userRoles []models.UserRole
		utils.DB.Where("user_id = ?", userID).Order("id asc").Find(&userRoles)
		for _, item := range userRoles {
			if item.RoleID == 0 || seen[item.RoleID] {
				continue
			}
			seen[item.RoleID] = true
			roleIDs = append(roleIDs, item.RoleID)
		}
	}
	if len(roleIDs) == 0 && fallbackRoleID != 0 {
		roleIDs = append(roleIDs, fallbackRoleID)
	}
	return roleIDs
}

func moduleByPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/v1/prd/"):
		return prdModuleByPath(path)
	case strings.HasPrefix(path, "/api/v1/im/"):
		return "im"
	case strings.HasPrefix(path, "/api/v1/iot/"):
		return "iot"
	case strings.HasPrefix(path, "/api/v1/files/"), strings.HasPrefix(path, "/api/v1/disk/"):
		return "projects"
	case strings.HasPrefix(path, "/api/v1/system/"):
		return "system"
	case strings.HasPrefix(path, "/api/v1/rbac/"), strings.HasPrefix(path, "/api/v1/accounts/"):
		return "accounts"
	case strings.HasPrefix(path, "/api/v1/operation-logs"):
		return "audit_log"
	case strings.HasPrefix(path, "/api/v1/tenants"):
		return "enterprise"
	default:
		return ""
	}
}

func prdModuleByPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/prd/"), "/")
	if len(parts) == 0 {
		return "workspace"
	}
	switch parts[0] {
	case "cockpit", "analytics":
		return "dashboards"
	case "ai":
		return "insights"
	case "next-actions":
		return "workspace"
	case "domains", "summary", "relations", "flows", "extensions":
		if parts[0] == "flows" && len(parts) > 1 {
			switch parts[1] {
			case "deal-chain":
				return "deals"
			case "opportunity-reviews":
				return "opportunities"
			case "quotes":
				return "opportunities"
			case "sales-targets":
				return "dashboards"
			case "tenant-ai-analysis":
				return "insights"
			}
			return parts[1]
		}
		return "workspace"
	case "settings":
		return "enterprise"
	default:
		return parts[0]
	}
}

func permissionByPath(method, path string) string {
	if path == "/api/v1/accounts/me/language" {
		return ""
	}
	if strings.HasPrefix(path, "/api/v1/accounts/me/") {
		return ""
	}
	if path == "/api/v1/accounts/users" && method == http.MethodGet {
		return ""
	}
	if path == "/api/v1/prd/extensions/manifest" && method == http.MethodGet {
		return ""
	}
	if strings.HasPrefix(path, "/api/v1/system/departments") {
		if method == http.MethodGet {
			return "system:department:view"
		}
		if method == http.MethodPost {
			return "system:department:create"
		}
		if method == http.MethodDelete {
			return "system:department:delete"
		}
		return "system:department:update"
	}
	if strings.HasPrefix(path, "/api/v1/system/posts") {
		if method == http.MethodGet {
			return "system:post:view"
		}
		if method == http.MethodPost {
			return "system:post:create"
		}
		if method == http.MethodDelete {
			return "system:post:delete"
		}
		return "system:post:update"
	}
	if strings.HasPrefix(path, "/api/v1/system/menus") {
		if method == http.MethodGet {
			return "system:menu:view"
		}
		if method == http.MethodPost {
			return "system:menu:create"
		}
		if method == http.MethodDelete {
			return "system:menu:delete"
		}
		return "system:menu:update"
	}
	if strings.HasPrefix(path, "/api/v1/disk/space") {
		return "projects:view"
	}
	if strings.HasPrefix(path, "/api/v1/im/ws") {
		return "im:view"
	}
	if strings.HasPrefix(path, "/api/v1/im/") {
		if method == http.MethodGet {
			return "im:view"
		}
		if method == http.MethodPost {
			return "im:create"
		}
		if method == http.MethodDelete {
			return "im:delete"
		}
		return "im:update"
	}
	if strings.HasPrefix(path, "/api/v1/iot/") {
		if method == http.MethodGet {
			return "iot:view"
		}
		return "iot:manage"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/") {
		return manufacturingPermissionByPath(method, path)
	}
	if strings.HasPrefix(path, "/api/v1/files/check") || strings.HasPrefix(path, "/api/v1/files/upload") || strings.HasPrefix(path, "/api/v1/files/") {
		if method == http.MethodGet {
			return "projects:view"
		}
		if method == http.MethodPost {
			return "projects:create"
		}
		if method == http.MethodDelete {
			return "projects:delete"
		}
		return "projects:update"
	}
	if path == "/api/v1/operation-logs/export" || path == "/api/v1/operation-logs/login/export" {
		return "audit_log:export"
	}
	if strings.HasPrefix(path, "/api/v1/prd/leads/objects/export") {
		return "leads:export"
	}
	if strings.HasPrefix(path, "/api/v1/prd/leads/objects/import") {
		return "leads:create"
	}
	if strings.Contains(path, "/api/v1/prd/flows/leads/") && strings.HasSuffix(path, "/assign") {
		return "leads:assign"
	}
	if strings.HasPrefix(path, "/api/v1/prd/flows/leads/auto-assign") {
		return "leads:assign"
	}
	if strings.HasPrefix(path, "/api/v1/prd/approvals/") && strings.HasSuffix(path, "/action") {
		return "workspace:approve"
	}
	if strings.HasPrefix(path, "/api/v1/prd/flows/") {
		flowPermission := permissionByFlowPath(method, path)
		if flowPermission != "" {
			return flowPermission
		}
	}
	if strings.HasSuffix(path, "/objects/export") && strings.HasPrefix(path, "/api/v1/prd/") {
		domain := domainFromObjectPath(path)
		if domain != "" {
			return domain + ":export"
		}
	}
	if strings.HasSuffix(path, "/objects/import") && strings.HasPrefix(path, "/api/v1/prd/") {
		domain := domainFromObjectPath(path)
		if domain != "" {
			return domain + ":create"
		}
	}
	module := moduleByPath(path)
	if module == "" {
		return ""
	}
	if strings.HasPrefix(path, "/api/v1/operation-logs") {
		return "audit_log:view"
	}
	action := actionByMethod(method)
	return module + ":" + action
}

func manufacturingPermissionByPath(method, path string) string {
	if strings.HasPrefix(path, "/api/v1/mfg/rfq-clarifications/") {
		if method == http.MethodPut || method == http.MethodPatch || strings.HasSuffix(path, "/answer-histories") {
			return "mfg:rfq:view"
		}
		return "mfg:rfq:manage"
	}
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return manufacturingViewPermissionByPath(path)
	}
	if strings.HasPrefix(path, "/api/v1/mfg/rfqs") {
		return "mfg:rfq:view"
	}
	if path == "/api/v1/mfg/bom-snapshots/from-config" {
		return "mfg:product:view"
	}
	if path == "/api/v1/mfg/cost-sheets/calculate" || strings.HasSuffix(path, "/complete") && strings.HasPrefix(path, "/api/v1/mfg/cost-sheets/") {
		return "mfg:cost:view"
	}
	if path == "/api/v1/mfg/quotes" ||
		path == "/api/v1/mfg/quotes/order" ||
		strings.HasPrefix(path, "/api/v1/mfg/quotes/") && (strings.HasSuffix(path, "/submit-approval") ||
			strings.HasSuffix(path, "/send") ||
			strings.HasSuffix(path, "/customer-confirm") ||
			strings.HasSuffix(path, "/customer-view") ||
			strings.HasSuffix(path, "/revisions")) {
		return "mfg:quote:view"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/quotes/") && (strings.HasSuffix(path, "/approve") || strings.HasSuffix(path, "/reject")) {
		return "mfg:quote:approve"
	}
	if path == "/api/v1/mfg/dashboard/refresh-metrics" {
		return "mfg:metrics:manage"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/bom-") || strings.HasPrefix(path, "/api/v1/mfg/cpq/") || strings.HasPrefix(path, "/api/v1/mfg/product-families") {
		return "mfg:product:manage"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/cost-sheets") {
		return "mfg:cost:manage"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/quotes") {
		return "mfg:quote:manage"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/sample-projects") {
		return "mfg:sample:manage"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/production-progress") {
		return "mfg:production:manage"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/ecns") || strings.HasPrefix(path, "/api/v1/mfg/compensation-records") {
		return "mfg:ecn:manage"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/dashboard") {
		return "mfg:metrics:view"
	}
	return "mfg:view"
}

func manufacturingViewPermissionByPath(path string) string {
	if strings.HasPrefix(path, "/api/v1/mfg/dashboard") {
		return "mfg:metrics:view"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/rfqs") || strings.HasPrefix(path, "/api/v1/mfg/rfq-clarifications") {
		return "mfg:rfq:view"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/bom-") || strings.HasPrefix(path, "/api/v1/mfg/product-families") {
		return "mfg:product:view"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/cost-sheets") {
		return "mfg:cost:view"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/quotes") {
		return "mfg:quote:view"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/sample-projects") {
		return "mfg:sample:view"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/production-progress") {
		return "mfg:production:view"
	}
	if strings.HasPrefix(path, "/api/v1/mfg/ecns") || strings.HasPrefix(path, "/api/v1/mfg/compensation-records") {
		return "mfg:ecn:view"
	}
	return "mfg:view"
}

func manufacturingReadPermissionGrantedByPackage(tenantID uint, permission string) bool {
	if tenantID == 0 || !isManufacturingReadPermission(permission) {
		return false
	}
	return services.IsIndustryPackageEnabled(tenantID, services.IndustryPackageManufacturing)
}

func isManufacturingReadPermission(permission string) bool {
	permission = strings.TrimSpace(permission)
	return permission == "mfg:view" || strings.HasPrefix(permission, "mfg:") && strings.HasSuffix(permission, ":view")
}

func permissionByFlowPath(method, path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/prd/flows/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return ""
	}
	domain := strings.TrimSpace(parts[0])
	switch domain {
	case "deal-chain":
		domain = "deals"
	case "opportunity-reviews":
		domain = "opportunities"
	case "quotes":
		domain = "opportunities"
	case "tenant-ai-analysis":
		if strings.Contains(path, "/export.pdf") {
			return "insights:export"
		}
		if strings.Contains(path, "/usage/") {
			return "ai_usage:view"
		}
		if strings.HasSuffix(path, "/estimate") || strings.HasSuffix(path, "/jobs") && method == http.MethodPost {
			return "insights:create"
		}
		return "insights:view"
	case "sales-targets":
		return ""
	}
	if domain == "approvals" {
		if strings.HasSuffix(path, "/action") {
			return "workspace:approve"
		}
		if method == http.MethodGet {
			return "workspace:view"
		}
		return "workspace:update"
	}
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return domain + ":view"
	}
	if strings.HasSuffix(path, "/assign") ||
		strings.Contains(path, "/auto-assign") ||
		strings.HasSuffix(path, "/transfer") {
		return domain + ":assign"
	}
	if method == http.MethodDelete {
		return domain + ":delete"
	}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		return domain + ":update"
	}
	return domain + ":" + actionByMethod(method)
}

func domainFromObjectPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/prd/"), "/")
	if len(parts) < 3 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func actionByMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return "view"
	case http.MethodPost:
		return "create"
	case http.MethodDelete:
		return "delete"
	case http.MethodPut, http.MethodPatch:
		return "update"
	default:
		return "update"
	}
}

func permissionGranted(permissionSet map[string]bool, required string) bool {
	if required == "" {
		return true
	}
	candidates := permissionCandidates(required)
	for _, item := range candidates {
		if permissionSet[item] {
			return true
		}
	}
	return false
}

func permissionCandidates(required string) []string {
	required = strings.TrimSpace(required)
	if required == "" {
		return []string{}
	}
	resource, action, ok := splitPermission(required)
	if !ok {
		return []string{required}
	}
	candidates := []string{required}
	switch action {
	case "view":
		candidates = append(candidates, resource+":manage")
	case "create", "update", "delete", "export", "assign", "approve":
		candidates = append(candidates, resource+":manage")
	}
	return uniquePermissionCandidates(candidates)
}

func splitPermission(value string) (resource string, action string, ok bool) {
	idx := strings.LastIndex(strings.TrimSpace(value), ":")
	if idx <= 0 || idx >= len(value)-1 {
		return "", "", false
	}
	resource = strings.TrimSpace(value[:idx])
	action = strings.TrimSpace(value[idx+1:])
	if resource == "" || action == "" {
		return "", "", false
	}
	return resource, action, true
}

func uniquePermissionCandidates(items []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func isBusinessPath(path string) bool {
	switch moduleByPath(path) {
	case "workspace", "enterprise", "projects", "products", "campaigns", "contacts", "leads", "deals", "orders", "tickets", "dashboards", "teams", "insights", "accounts", "opportunities", "im":
		return !strings.HasPrefix(path, "/api/v1/tenants") && !strings.HasPrefix(path, "/api/v1/operation-logs")
	default:
		return false
	}
}

func isTenantBusinessPath(path string) bool {
	if path == "/api/v1/accounts/me/language" || strings.HasPrefix(path, "/api/v1/accounts/me/") {
		return false
	}
	switch {
	case strings.HasPrefix(path, "/api/v1/prd/"),
		strings.HasPrefix(path, "/api/v1/im/"),
		strings.HasPrefix(path, "/api/v1/rbac/"),
		strings.HasPrefix(path, "/api/v1/system/"),
		strings.HasPrefix(path, "/api/v1/accounts/"),
		strings.HasPrefix(path, "/api/v1/files/"),
		strings.HasPrefix(path, "/api/v1/disk/"):
		return true
	default:
		return false
	}
}

func isPlatformAdminAllowedPath(path string) bool {
	if path == "/api/v1/im/ws" ||
		path == "/api/v1/im/conversations" ||
		path == "/api/v1/im/support/conversations" ||
		path == "/api/v1/im/unread/summary" {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/im/system-notifications") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/im/conversations/") {
		return true
	}
	return false
}

func tenantModuleEnabled(tenantID uint, module string) bool {
	// Tenant-level audit trails are mandatory capabilities.
	// Login/operation logs must stay accessible regardless of plan/module toggles.
	if module == "" || module == "system" || module == "audit_log" {
		return true
	}
	if strings.HasPrefix(module, "system:") {
		return true
	}
	var quota models.TenantQuota
	if err := utils.DB.Where("tenant_id = ?", tenantID).First(&quota).Error; err != nil {
		return true
	}
	modules := normalizeTenantModules(splitCSV(quota.Modules))
	if len(modules) == 0 {
		modules = modulesByPlanCode(quota.PlanCode)
	}
	if len(modules) == 0 {
		return true
	}
	for _, item := range modules {
		if item == module {
			return true
		}
	}
	return false
}

var supportedTenantModules = map[string]bool{
	"workspace": true, "enterprise": true, "projects": true, "products": true, "campaigns": true,
	"contacts": true, "leads": true, "opportunities": true, "deals": true, "orders": true,
	"tickets": true, "dashboards": true, "teams": true, "insights": true, "accounts": true,
	"im": true, "iot": true, "audit_log": true,
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func normalizeTenantModules(modules []string) []string {
	items := make([]string, 0, len(modules))
	seen := map[string]bool{}
	for _, module := range modules {
		code := strings.TrimSpace(module)
		if code == "" || !supportedTenantModules[code] || seen[code] {
			continue
		}
		seen[code] = true
		items = append(items, code)
	}
	return items
}

func modulesByPlanCode(planCode string) []string {
	code := strings.TrimSpace(planCode)
	if code == "" {
		code = "basic"
	}
	var plan models.PlatformPlanTemplate
	if err := utils.DB.Select("modules_json").Where("plan_code = ?", code).First(&plan).Error; err == nil {
		var modules []string
		if err := json.Unmarshal([]byte(plan.ModulesJSON), &modules); err == nil {
			return normalizeTenantModules(modules)
		}
	}
	if code == "plus" {
		return normalizeTenantModules([]string{"workspace", "enterprise", "projects", "products", "campaigns", "contacts", "leads", "opportunities", "deals", "orders", "tickets", "dashboards", "teams", "insights", "accounts", "im", "iot"})
	}
	return normalizeTenantModules([]string{"workspace", "contacts", "leads", "im", "iot"})
}

func tenantBlockedByPolicy(tenantID uint, now time.Time) (string, bool) {
	var tenant models.Tenant
	if err := utils.DB.Select("id,status").Where("id = ?", tenantID).First(&tenant).Error; err == nil {
		if tenant.Status != "active" {
			return "TENANT_DISABLED", true
		}
	}
	var quota models.TenantQuota
	if err := utils.DB.Select("valid_to").Where("tenant_id = ?", tenantID).First(&quota).Error; err == nil {
		if quota.ValidTo != nil && now.After(*quota.ValidTo) {
			return "TENANT_EXPIRED", true
		}
	}
	return "", false
}
