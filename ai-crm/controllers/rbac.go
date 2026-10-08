package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"net/http"
	"sort"
	"strings"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RBACController struct {
	web.Controller
}

var fineGrainedAllowedActions = map[string]bool{
	"view":    true,
	"create":  true,
	"update":  true,
	"delete":  true,
	"export":  true,
	"assign":  true,
	"approve": true,
	"manage":  true, // compatibility
}

func (c *RBACController) CreateRole() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		Name        string   `json:"name"`
		Code        string   `json:"code"`
		Description string   `json:"description"`
		Status      string   `json:"status"`
		Permissions []string `json:"permissions"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		utils.BadRequest(c.Ctx, "角色名不能为空")
		return
	}
	role := models.Role{
		TenantModel:     models.TenantModel{TenantID: auth.TenantID},
		Name:            req.Name,
		Code:            strings.TrimSpace(req.Code),
		Description:     req.Description,
		Status:          normalizeStatus(req.Status),
		PermissionsJSON: strings.Join(filterAssignablePermissions(req.Permissions), ","),
	}
	if err := utils.DB.Create(&role).Error; err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_ROLE", "角色管理", "operation.create_role", map[string]interface{}{"role_name": req.Name}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建角色失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_ROLE", "角色管理", "operation.create_role", map[string]interface{}{"role_id": role.ID, "role_name": role.Name}, "success", "")
	utils.Success(c.Ctx, role)
}

func (c *RBACController) ListRoles() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var roles []models.Role
	query := utils.DB.Where("tenant_id = ?", auth.TenantID)
	name := strings.TrimSpace(c.Ctx.Input.Query("name"))
	code := strings.TrimSpace(c.Ctx.Input.Query("code"))
	description := strings.TrimSpace(c.Ctx.Input.Query("description"))
	status := strings.TrimSpace(c.Ctx.Input.Query("status"))
	start := parseTimeQuery(c.Ctx.Input.Query("start_time"))
	end := parseTimeQuery(c.Ctx.Input.Query("end_time"))
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if code != "" {
		query = query.Where("code LIKE ?", "%"+code+"%")
	}
	if description != "" {
		query = query.Where("description LIKE ?", "%"+description+"%")
	}
	if status != "" {
		query = query.Where("status = ?", normalizeStatus(status))
	}
	if start != nil {
		query = query.Where("created_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("created_at <= ?", *end)
	}
	if err := query.Order("is_preset desc, created_at desc").Find(&roles).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取角色失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": roles, "total": len(roles)})
}

func (c *RBACController) UpdateRole() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的角色ID")
		return
	}
	var req struct {
		Name        string   `json:"name"`
		Code        string   `json:"code"`
		Description string   `json:"description"`
		Status      string   `json:"status"`
		Permissions []string `json:"permissions"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var role models.Role
	if err := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).First(&role).Error; err != nil {
		utils.NotFound(c.Ctx, "角色不存在")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		utils.BadRequest(c.Ctx, "角色名不能为空")
		return
	}
	if role.IsPreset && role.Name != req.Name {
		utils.BadRequest(c.Ctx, "预设角色名称不能修改")
		return
	}
	if req.Name != "" {
		role.Name = req.Name
	}
	role.Code = strings.TrimSpace(req.Code)
	role.Description = req.Description
	if req.Status != "" {
		role.Status = normalizeStatus(req.Status)
	}
	role.PermissionsJSON = strings.Join(filterAssignablePermissions(req.Permissions), ",")
	if err := utils.DB.Save(&role).Error; err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_ROLE", "角色管理", "operation.update_role", map[string]interface{}{"role_id": id, "role_name": role.Name}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新角色失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_ROLE", "角色管理", "operation.update_role", map[string]interface{}{"role_id": id, "role_name": role.Name}, "success", "")
	utils.Success(c.Ctx, role)
}

func (c *RBACController) DeleteRole() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的角色ID")
		return
	}
	var role models.Role
	if err := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).First(&role).Error; err != nil {
		utils.NotFound(c.Ctx, "角色不存在")
		return
	}
	var count int64
	utils.DB.Model(&models.User{}).Where("tenant_id = ? AND role_id = ?", auth.TenantID, id).Count(&count)
	if count == 0 {
		utils.DB.Model(&models.UserRole{}).
			Joins("JOIN users ON users.id = user_roles.user_id").
			Where("users.tenant_id = ? AND user_roles.role_id = ?", auth.TenantID, id).
			Count(&count)
	}
	if count > 0 {
		utils.BadRequest(c.Ctx, "该角色仍有用户使用，不能删除")
		return
	}
	if err := utils.DB.Delete(&role).Error; err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "DELETE_ROLE", "角色管理", "operation.delete_role", map[string]interface{}{"role_id": id, "role_name": role.Name}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "删除角色失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "DELETE_ROLE", "角色管理", "operation.delete_role", map[string]interface{}{"role_id": id, "role_name": role.Name}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"deleted_id": id})
}

func (c *RBACController) ListPermissions() {
	var permissions []models.Permission
	if err := utils.DB.Where("resource NOT IN ?", reservedOrganizationPermissionResources()).Order("resource asc, action asc").Find(&permissions).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取权限资源失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": permissions})
}

func (c *RBACController) PermissionCatalog() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Error(c.Ctx, http.StatusForbidden, "缺少租户上下文")
		return
	}
	var quota models.TenantQuota
	if err := utils.DB.Where("tenant_id = ?", auth.TenantID).First(&quota).Error; err != nil {
		quota.PlanCode = "basic"
	}
	modules := normalizeTenantModules(splitCSV(quota.Modules))
	if len(modules) == 0 {
		modules = modulesByPlanCode(quota.PlanCode)
	}
	moduleSet := map[string]bool{}
	for _, item := range modules {
		moduleSet[item] = true
	}
	if services.IsIndustryPackageEnabled(auth.TenantID, services.IndustryPackageManufacturing) {
		moduleSet["manufacturing"] = true
	}

	catalog := permissionCatalogBase()
	items := make([]map[string]interface{}, 0, len(catalog))
	for _, item := range catalog {
		resource := item["resource"].(string)
		if isReservedOrganizationPermissionResource(resource) {
			continue
		}
		module := item["module"].(string)
		if module != "" && !moduleSet[module] {
			continue
		}
		items = append(items, item)
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items})
}

func (c *RBACController) AssignPermissions() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		RoleID      uint `json:"role_id"`
		Permissions []struct {
			Resource string   `json:"resource"`
			Actions  []string `json:"actions"`
		} `json:"permissions"`
		DataScope string `json:"data_scope"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if !roleBelongsToTenant(req.RoleID, auth.TenantID) {
		utils.BadRequest(c.Ctx, "角色不存在或不属于当前租户")
		return
	}
	permissions := make([]string, 0)
	permissionSet := map[string]bool{}
	permissionKeys := make([]string, 0)
	seenPermissionKeys := map[string]bool{}
	for _, item := range req.Permissions {
		resource := strings.TrimSpace(item.Resource)
		if resource == "" || resource == "system:menu" || isReservedOrganizationPermissionResource(resource) {
			continue
		}
		for _, action := range item.Actions {
			action = strings.TrimSpace(action)
			if !fineGrainedAllowedActions[action] {
				continue
			}
			key := resource + ":" + action
			if !seenPermissionKeys[key] {
				permissionKeys = append(permissionKeys, key)
				seenPermissionKeys[key] = true
			}
			permissionSet[key] = true
			// Dual track compatibility: write old manage when write-class actions are selected.
			if action == "create" || action == "update" || action == "delete" || action == "assign" || action == "approve" {
				permissionSet[resource+":manage"] = true
			}
		}
	}
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	permissionsJSON := strings.Join(permissions, ",")
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Role{}).Where("id = ? AND tenant_id = ?", req.RoleID, auth.TenantID).Update("permissions_json", permissionsJSON).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("role_id = ?", req.RoleID).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		if len(permissionKeys) == 0 {
			return nil
		}
		var permissionRows []models.Permission
		for _, key := range permissionKeys {
			resource, action, valid := splitPermissionKey(key)
			if !valid {
				continue
			}
			permissionRows = append(permissionRows, models.Permission{
				Resource: resource,
				Action:   action,
				NameKey:  "permission." + strings.ReplaceAll(resource, ":", ".") + "." + action,
			})
		}
		if len(permissionRows) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "resource"}, {Name: "action"}},
				DoUpdates: clause.AssignmentColumns([]string{"name_key", "updated_at"}),
			}).Create(&permissionRows).Error; err != nil {
				return err
			}
		}
		var existing []models.Permission
		condition, args := permissionLookupCondition(permissionKeys)
		if err := tx.Where(condition, args...).Find(&existing).Error; err != nil {
			return err
		}
		rolePermissions := make([]models.RolePermission, 0, len(existing))
		for _, permission := range existing {
			rolePermissions = append(rolePermissions, models.RolePermission{
				RoleID:       req.RoleID,
				PermissionID: permission.ID,
				DataScope:    req.DataScope,
			})
		}
		if len(rolePermissions) > 0 {
			if err := tx.CreateInBatches(rolePermissions, 200).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存角色权限失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_ROLE_PERMISSION", "角色管理", "operation.update_role_permission", map[string]interface{}{"role_id": req.RoleID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"role_id": req.RoleID, "permissions": permissions, "permissions_json": permissionsJSON})
}

func splitPermissionKey(key string) (string, string, bool) {
	index := strings.LastIndex(strings.TrimSpace(key), ":")
	if index <= 0 || index >= len(key)-1 {
		return "", "", false
	}
	resource := strings.TrimSpace(key[:index])
	action := strings.TrimSpace(key[index+1:])
	return resource, action, resource != "" && action != ""
}

func reservedOrganizationPermissionResources() []string {
	return []string{"enterprise", "accounts", "system:department", "system:post", "system:menu", "storage", "iot", "audit_log"}
}

func isReservedOrganizationPermissionResource(resource string) bool {
	resource = strings.TrimSpace(resource)
	for _, item := range reservedOrganizationPermissionResources() {
		if resource == item {
			return true
		}
	}
	return false
}

func filterAssignablePermissions(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		permission := strings.TrimSpace(item)
		if permission == "" {
			continue
		}
		resource, _, valid := splitPermissionKey(permission)
		if !valid || isReservedOrganizationPermissionResource(resource) {
			continue
		}
		result = append(result, permission)
	}
	sort.Strings(result)
	return result
}

func permissionLookupCondition(keys []string) (string, []interface{}) {
	parts := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys)*2)
	for _, key := range keys {
		resource, action, valid := splitPermissionKey(key)
		if !valid {
			continue
		}
		parts = append(parts, "(resource = ? AND action = ?)")
		args = append(args, resource, action)
	}
	if len(parts) == 0 {
		return "1 = 0", nil
	}
	return strings.Join(parts, " OR "), args
}

func permissionCatalogBase() []map[string]interface{} {
	catalog := []map[string]interface{}{
		{"resource": "workspace", "module": "workspace", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "enterprise", "module": "enterprise", "actions": []string{"view", "create", "update", "delete", "export", "manage"}, "is_extension_action": false},
		{"resource": "projects", "module": "projects", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "products", "module": "products", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "campaigns", "module": "campaigns", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "contacts", "module": "contacts", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "leads", "module": "leads", "actions": []string{"view", "create", "update", "delete", "export", "assign"}, "is_extension_action": true},
		{"resource": "opportunities", "module": "opportunities", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "deals", "module": "deals", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "orders", "module": "orders", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "tickets", "module": "tickets", "actions": []string{"view", "create", "update", "delete", "export", "assign"}, "is_extension_action": true},
		{"resource": "dashboards", "module": "dashboards", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "teams", "module": "teams", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "insights", "module": "insights", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "storage", "module": "", "actions": []string{"view", "manage"}, "is_extension_action": false},
		{"resource": "iot", "module": "iot", "actions": []string{"view", "manage"}, "is_extension_action": false},
		{"resource": "im", "module": "im", "actions": []string{"view", "create", "update", "delete"}, "is_extension_action": false},
		{"resource": "mfg", "module": "manufacturing", "actions": []string{"view"}, "is_extension_action": true},
		{"resource": "mfg:rfq", "module": "manufacturing", "actions": []string{"view", "manage"}, "is_extension_action": true},
		{"resource": "mfg:product", "module": "manufacturing", "actions": []string{"view", "manage"}, "is_extension_action": true},
		{"resource": "mfg:cost", "module": "manufacturing", "actions": []string{"view", "view_sensitive", "manage"}, "is_extension_action": true},
		{"resource": "mfg:quote", "module": "manufacturing", "actions": []string{"view", "approve", "manage"}, "is_extension_action": true},
		{"resource": "mfg:sample", "module": "manufacturing", "actions": []string{"view", "manage"}, "is_extension_action": true},
		{"resource": "mfg:production", "module": "manufacturing", "actions": []string{"view", "manage"}, "is_extension_action": true},
		{"resource": "mfg:ecn", "module": "manufacturing", "actions": []string{"view", "approve", "manage"}, "is_extension_action": true},
		{"resource": "mfg:compensation", "module": "manufacturing", "actions": []string{"view", "manage", "approve"}, "is_extension_action": true},
		{"resource": "mfg:metrics", "module": "manufacturing", "actions": []string{"view", "manage"}, "is_extension_action": true},
		{"resource": "accounts", "module": "accounts", "actions": []string{"view", "create", "update", "delete", "export"}, "is_extension_action": false},
		{"resource": "system:department", "module": "accounts", "actions": []string{"view", "create", "update", "delete"}, "is_extension_action": false},
		{"resource": "system:post", "module": "accounts", "actions": []string{"view", "create", "update", "delete"}, "is_extension_action": false},
		{"resource": "audit_log", "module": "audit_log", "actions": []string{"view", "export"}, "is_extension_action": true},
	}
	for _, item := range catalog {
		resource := item["resource"].(string)
		labels := map[string]string{}
		for _, action := range item["actions"].([]string) {
			labels[action] = "permission." + resource + "." + action
		}
		item["labels"] = labels
	}
	return catalog
}
