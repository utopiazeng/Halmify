package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"gorm.io/gorm"
)

type TenantController struct {
	web.Controller
}

var defaultPermissions = []models.Permission{
	{Resource: "workspace", Action: "view", NameKey: "permission.workspace.view"},
	{Resource: "enterprise", Action: "view", NameKey: "permission.enterprise.view"},
	{Resource: "enterprise", Action: "manage", NameKey: "permission.enterprise.manage"},
	{Resource: "projects", Action: "view", NameKey: "permission.projects.view"},
	{Resource: "projects", Action: "manage", NameKey: "permission.projects.manage"},
	{Resource: "products", Action: "view", NameKey: "permission.products.view"},
	{Resource: "products", Action: "manage", NameKey: "permission.products.manage"},
	{Resource: "campaigns", Action: "view", NameKey: "permission.campaigns.view"},
	{Resource: "campaigns", Action: "manage", NameKey: "permission.campaigns.manage"},
	{Resource: "contacts", Action: "view", NameKey: "permission.contacts.view"},
	{Resource: "contacts", Action: "manage", NameKey: "permission.contacts.manage"},
	{Resource: "leads", Action: "view", NameKey: "permission.leads.view"},
	{Resource: "leads", Action: "manage", NameKey: "permission.leads.manage"},
	{Resource: "opportunities", Action: "view", NameKey: "permission.opportunities.view"},
	{Resource: "opportunities", Action: "manage", NameKey: "permission.opportunities.manage"},
	{Resource: "deals", Action: "view", NameKey: "permission.deals.view"},
	{Resource: "deals", Action: "manage", NameKey: "permission.deals.manage"},
	{Resource: "orders", Action: "view", NameKey: "permission.orders.view"},
	{Resource: "orders", Action: "manage", NameKey: "permission.orders.manage"},
	{Resource: "tickets", Action: "view", NameKey: "permission.tickets.view"},
	{Resource: "tickets", Action: "manage", NameKey: "permission.tickets.manage"},
	{Resource: "dashboards", Action: "view", NameKey: "permission.dashboards.view"},
	{Resource: "teams", Action: "view", NameKey: "permission.teams.view"},
	{Resource: "teams", Action: "manage", NameKey: "permission.teams.manage"},
	{Resource: "insights", Action: "view", NameKey: "permission.insights.view"},
	{Resource: "insights", Action: "manage", NameKey: "permission.insights.manage"},
	{Resource: "storage", Action: "view", NameKey: "permission.storage.view"},
	{Resource: "storage", Action: "manage", NameKey: "permission.storage.manage"},
	{Resource: "iot", Action: "view", NameKey: "permission.iot.view"},
	{Resource: "iot", Action: "manage", NameKey: "permission.iot.manage"},
	{Resource: "accounts", Action: "view", NameKey: "permission.accounts.view"},
	{Resource: "accounts", Action: "manage", NameKey: "permission.accounts.manage"},
	{Resource: "audit_log", Action: "view", NameKey: "permission.audit_log.view"},
	{Resource: "audit_log", Action: "export", NameKey: "permission.audit_log.export"},
	{Resource: "mfg", Action: "view", NameKey: "permission.mfg.view"},
	{Resource: "mfg:rfq", Action: "view", NameKey: "permission.mfg.rfq.view"},
	{Resource: "mfg:rfq", Action: "manage", NameKey: "permission.mfg.rfq.manage"},
	{Resource: "mfg:product", Action: "view", NameKey: "permission.mfg.product.view"},
	{Resource: "mfg:product", Action: "manage", NameKey: "permission.mfg.product.manage"},
	{Resource: "mfg:cost", Action: "view", NameKey: "permission.mfg.cost.view"},
	{Resource: "mfg:quote", Action: "view", NameKey: "permission.mfg.quote.view"},
	{Resource: "mfg:quote", Action: "approve", NameKey: "permission.mfg.quote.approve"},
	{Resource: "mfg:quote", Action: "manage", NameKey: "permission.mfg.quote.manage"},
	{Resource: "mfg:cost", Action: "view_sensitive", NameKey: "permission.mfg.cost.view_sensitive"},
	{Resource: "mfg:cost", Action: "manage", NameKey: "permission.mfg.cost.manage"},
	{Resource: "mfg:sample", Action: "view", NameKey: "permission.mfg.sample.view"},
	{Resource: "mfg:sample", Action: "manage", NameKey: "permission.mfg.sample.manage"},
	{Resource: "mfg:production", Action: "view", NameKey: "permission.mfg.production.view"},
	{Resource: "mfg:production", Action: "manage", NameKey: "permission.mfg.production.manage"},
	{Resource: "mfg:ecn", Action: "view", NameKey: "permission.mfg.ecn.view"},
	{Resource: "mfg:ecn", Action: "approve", NameKey: "permission.mfg.ecn.approve"},
	{Resource: "mfg:ecn", Action: "manage", NameKey: "permission.mfg.ecn.manage"},
	{Resource: "mfg:compensation", Action: "view", NameKey: "permission.mfg.compensation.view"},
	{Resource: "mfg:compensation", Action: "manage", NameKey: "permission.mfg.compensation.manage"},
	{Resource: "mfg:compensation", Action: "approve", NameKey: "permission.mfg.compensation.approve"},
	{Resource: "mfg:metrics", Action: "view", NameKey: "permission.mfg.metrics.view"},
	{Resource: "mfg:metrics", Action: "manage", NameKey: "permission.mfg.metrics.manage"},
}

type tenantQuotaRequest struct {
	PlanCode        string   `json:"plan_code"`
	MaxUsers        int      `json:"max_users"`
	StorageLimit    int64    `json:"storage_limit"`
	ValidFrom       string   `json:"valid_from"`
	ValidTo         string   `json:"valid_to"`
	Modules         []string `json:"modules"`
	ModuleOverrides struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	} `json:"module_overrides"`
}

type createTenantRequest struct {
	Name                   string             `json:"name"`
	AdminUsername          string             `json:"admin_username"`
	AdminEmail             string             `json:"admin_email"`
	Quota                  tenantQuotaRequest `json:"quota"`
	Remark                 string             `json:"remark"`
	IndustryPackageCode    string             `json:"industry_package_code"`
	IndustryPackageEnabled bool               `json:"industry_package_enabled"`
}

type tenantSummaryItem struct {
	TenantID           uint       `json:"tenant_id"`
	Status             string     `json:"status"`
	Plan               string     `json:"plan"`
	ModuleSet          []string   `json:"module_set"`
	StorageQuota       int64      `json:"storage_quota"`
	UsagePercent       float64    `json:"usage_percent"`
	ExpireAt           *time.Time `json:"expire_at"`
	AdminAccountMasked string     `json:"admin_account_masked"`
}

func (c *TenantController) ListTenants() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	_ = auth

	status := strings.TrimSpace(c.Ctx.Input.Query("status"))
	keyword := strings.TrimSpace(c.Ctx.Input.Query("keyword"))
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)

	query := utils.DB.Model(&models.Tenant{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)
	var tenants []models.Tenant
	if err := query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&tenants).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取企业租户失败")
		return
	}

	items := make([]tenantSummaryItem, 0, len(tenants))
	for _, tenant := range tenants {
		item := tenantSummaryItem{
			TenantID:     tenant.ID,
			Status:       tenant.Status,
			Plan:         tenantPlanFromRemark(tenant.Remark),
			ModuleSet:    []string{},
			StorageQuota: 0,
			UsagePercent: 0,
			ExpireAt:     nil,
		}
		var quota models.TenantQuota
		if err := utils.DB.Where("tenant_id = ?", tenant.ID).First(&quota).Error; err == nil {
			item.ModuleSet = splitCSV(quota.Modules)
			item.StorageQuota = quota.StorageLimit
			item.ExpireAt = quota.ValidTo
		}
		item.UsagePercent = 0
		var admin models.User
		if err := utils.DB.Select("username").Where("tenant_id = ? AND account_type = ?", tenant.ID, "tenant_admin").Order("id asc").First(&admin).Error; err == nil {
			item.AdminAccountMasked = strings.TrimSpace(admin.Username)
		}
		items = append(items, item)
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": total})
}

func (c *TenantController) CreateTenant() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	var req createTenantRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.AdminUsername = strings.TrimSpace(req.AdminUsername)
	if req.Name == "" || req.AdminUsername == "" {
		utils.BadRequest(c.Ctx, "企业名称、管理员用户名不能为空")
		return
	}
	var createdTenant models.Tenant
	var createdAdmin models.User
	activationLink := ""
	activationExpiresAt := time.Time{}
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		tenant := models.Tenant{Name: req.Name, Status: "active", Remark: req.Remark}
		if err := tx.Create(&tenant).Error; err != nil {
			return err
		}
		if err := ensurePresetRoles(tx, tenant.ID); err != nil {
			return err
		}
		adminRoleID, err := roleIDByName(tx, tenant.ID, "管理员")
		if err != nil {
			return err
		}
		admin := models.User{
			TenantModel:        models.TenantModel{TenantID: tenant.ID},
			Username:           req.AdminUsername,
			Email:              req.AdminEmail,
			Role:               "tenant_admin",
			RoleID:             adminRoleID,
			AccountType:        "tenant_admin",
			Status:             "active",
			MustChangePassword: true,
			LanguagePref:       "zh-CN",
		}
		if err := admin.SetPassword(generateBootstrapPassword(20)); err != nil {
			return err
		}
		if err := tx.Create(&admin).Error; err != nil {
			return err
		}
		tenant.AdminUserID = admin.ID
		quota := quotaFromRequest(req.Quota)
		if modules, maxUsers, storage, err := resolvePlanDefaultsTx(tx, quota.PlanCode); err == nil {
			quota.Modules = strings.Join(uniqueStrings(modules), ",")
			if quota.MaxUsers <= 0 {
				quota.MaxUsers = maxUsers
			}
			if quota.StorageLimit <= 0 {
				quota.StorageLimit = baseStorageBytes
			}
			_ = storage
		}
		quota.ModuleOverridesJSON = ""
		quota.TenantID = tenant.ID
		tenant.QuotaJSON = quotaSummaryJSON(quota)
		if err := tx.Create(&quota).Error; err != nil {
			return err
		}
		if err := tx.Save(&tenant).Error; err != nil {
			return err
		}
		if req.IndustryPackageEnabled || services.NormalizeIndustryPackageCode(req.IndustryPackageCode) != "" {
			if _, err := services.UpsertTenantIndustryPackageTx(tx, tenant.ID, services.IndustryPackageUpsert{
				PackageCode: req.IndustryPackageCode,
				Enabled:     req.IndustryPackageEnabled,
			}); err != nil {
				return err
			}
		}
		_, link, expiresAt, err := createTenantAdminActivationToken(tx, tenant.ID, admin.ID, auth.UserID, c.Ctx.Input.IP())
		if err != nil {
			return err
		}
		activationLink = link
		activationExpiresAt = expiresAt
		createdTenant = tenant
		createdAdmin = admin
		return nil
	})
	if err != nil {
		logControllerOperation(c.Ctx, auth, 0, "CREATE_TENANT", "租户[0]", "operation.create_tenant", map[string]interface{}{"tenant_name": req.Name}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建企业租户失败: "+err.Error())
		return
	}

	logControllerOperation(c.Ctx, auth, 0, "CREATE_TENANT", fmt.Sprintf("租户[%d]", createdTenant.ID), "operation.create_tenant", map[string]interface{}{"tenant_id": createdTenant.ID, "tenant_name": createdTenant.Name}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{
		"tenant_id":                  createdTenant.ID,
		"status":                     createdTenant.Status,
		"admin_user_id":              createdAdmin.ID,
		"admin_username":             createdAdmin.Username,
		"initial_password_once":      "",
		"activation_link_once":       activationLink,
		"activation_expires_at":      activationExpiresAt,
		"must_change_password_first": true,
	})
}

func (c *TenantController) UpdateTenantStatus() {
	auth, ok := requireSuperAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.Status != "active" && req.Status != "disabled" {
		utils.BadRequest(c.Ctx, "状态必须为 active 或 disabled")
		return
	}
	if err := utils.DB.Model(&models.Tenant{}).Where("id = ?", id).Update("status", req.Status).Error; err != nil {
		logControllerOperation(c.Ctx, auth, 0, "UPDATE_TENANT_STATUS", fmt.Sprintf("租户[%d]", id), "operation.update_tenant_status", map[string]interface{}{"tenant_id": id, "status": req.Status}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新租户状态失败")
		return
	}
	logControllerOperation(c.Ctx, auth, 0, "UPDATE_TENANT_STATUS", fmt.Sprintf("租户[%d]", id), "operation.update_tenant_status", map[string]interface{}{"tenant_id": id, "status": req.Status}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"id": id, "status": req.Status})
}

func (c *TenantController) UpdateTenantQuota() {
	auth, ok := requireSuperAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	var req tenantQuotaRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	quota := quotaFromRequest(req)
	quota.TenantID = id
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.TenantQuota
		if err := tx.Where("tenant_id = ?", id).First(&existing).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			return tx.Create(&quota).Error
		}
		existing.MaxUsers = quota.MaxUsers
		existing.StorageLimit = quota.StorageLimit
		existing.ValidFrom = quota.ValidFrom
		existing.ValidTo = quota.ValidTo
		existing.Modules = quota.Modules
		existing.FeatureSwitch = quota.FeatureSwitch
		if err := tx.Save(&existing).Error; err != nil {
			return err
		}
		return tx.Model(&models.Tenant{}).Where("id = ?", id).Update("quota_json", quotaSummaryJSON(existing)).Error
	})
	if err != nil {
		logControllerOperation(c.Ctx, auth, 0, "UPDATE_TENANT_QUOTA", fmt.Sprintf("租户[%d]", id), "operation.update_tenant_quota", map[string]interface{}{"tenant_id": id}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新租户配额失败")
		return
	}
	logControllerOperation(c.Ctx, auth, 0, "UPDATE_TENANT_QUOTA", fmt.Sprintf("租户[%d]", id), "operation.update_tenant_quota", map[string]interface{}{"tenant_id": id}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"tenant_id": id})
}

func (c *TenantController) ResetTenantAdminPassword() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		NewPassword string `json:"new_password"`
	}
	if len(c.Ctx.Input.RequestBody) > 0 {
		if !bindJSON(c.Ctx, &req) {
			return
		}
	}
	req.NewPassword = strings.TrimSpace(req.NewPassword)
	if req.NewPassword != "" && len(req.NewPassword) < 8 {
		utils.BadRequest(c.Ctx, "新密码至少需要8位")
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	var tenant models.Tenant
	if err := utils.DB.First(&tenant, id).Error; err != nil {
		utils.NotFound(c.Ctx, "租户不存在")
		return
	}
	var admin models.User
	query := utils.DB.Where("tenant_id = ? AND account_type = ?", id, "tenant_admin")
	if tenant.AdminUserID > 0 {
		query = utils.DB.Where("id = ? AND tenant_id = ?", tenant.AdminUserID, id)
	}
	if err := query.First(&admin).Error; err != nil {
		logControllerOperation(c.Ctx, auth, 0, "RESET_TENANT_ADMIN_PASSWORD", fmt.Sprintf("租户[%d]", id), "operation.reset_tenant_admin_password", map[string]interface{}{"tenant_id": id}, "failed", err.Error())
		utils.NotFound(c.Ctx, "企业管理员账号不存在")
		return
	}
	if admin.AccountType != "tenant_admin" {
		logControllerOperation(c.Ctx, auth, 0, "RESET_TENANT_ADMIN_PASSWORD", fmt.Sprintf("用户[%d]", admin.ID), "operation.reset_tenant_admin_password", map[string]interface{}{"tenant_id": id, "user_id": admin.ID}, "failed", "目标账号不是企业管理员")
		utils.BadRequest(c.Ctx, "目标账号不是企业管理员")
		return
	}
	if isImmutableDemoAccount(admin) {
		utils.ForbiddenKey(c.Ctx, "auth.demoAccountPasswordLocked", nil)
		return
	}
	initialPassword := ""
	passwordSource := "manual"
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		plain := req.NewPassword
		if plain == "" {
			plain = generateBootstrapPassword(20)
			passwordSource = "generated"
		}
		if err := admin.SetPassword(plain); err != nil {
			return err
		}
		initialPassword = plain
		admin.MustChangePassword = true
		if err := tx.Save(&admin).Error; err != nil {
			return err
		}
		if err := revokeTenantAdminSessions(tx, admin.ID); err != nil {
			return err
		}
		if err := revokePendingActivationTokens(tx, admin.ID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		logControllerOperation(c.Ctx, auth, 0, "RESET_TENANT_ADMIN_PASSWORD", fmt.Sprintf("用户[%d]", admin.ID), "operation.reset_tenant_admin_password", map[string]interface{}{"tenant_id": id, "user_id": admin.ID}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "重置企业管理员密码失败")
		return
	}
	logControllerOperation(c.Ctx, auth, 0, "RESET_TENANT_ADMIN_PASSWORD", fmt.Sprintf("用户[%d]", admin.ID), "operation.reset_tenant_admin_password", map[string]interface{}{"tenant_id": id, "user_id": admin.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{
		"tenant_id":                  id,
		"admin_user_id":              admin.ID,
		"initial_password_once":      initialPassword,
		"password_source":            passwordSource,
		"must_change_password_first": true,
		"sessions_revoked":           true,
	})
}

func (c *TenantController) DeleteTenant() {
	auth, ok := requireSuperAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	result, err := purgeTenantCompletely(auth, id)
	if err != nil {
		logControllerOperation(c.Ctx, auth, 0, "DELETE_TENANT", fmt.Sprintf("租户[%d]", id), "operation.delete_tenant", map[string]interface{}{"tenant_id": id, "file_cleanup": result.FileCleanup}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusBadGateway, "删除租户失败: "+err.Error())
		return
	}
	logControllerOperation(c.Ctx, auth, 0, "DELETE_TENANT", fmt.Sprintf("租户[%d]", id), "operation.delete_tenant", map[string]interface{}{"tenant_id": id, "deleted_file_bytes": result.FileCleanup.DeletedBytes, "deleted_file_mb": result.FileCleanup.DeletedMB, "deleted_db_rows": result.DBCleanup.TotalRows}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"deleted_id": id, "deleted_tenant_id": id, "file_cleanup": result.FileCleanup, "db_cleanup": result.DBCleanup})
}

func ensurePresetRoles(tx *gorm.DB, tenantID uint) error {
	presets := []struct {
		Name        string
		Description string
		Permissions string
	}{
		{"管理员", "企业管理员预设角色", "workspace:view,workspace:create,workspace:update,workspace:delete,workspace:export,workspace:manage,enterprise:view,enterprise:create,enterprise:update,enterprise:delete,enterprise:export,enterprise:manage,projects:view,projects:create,projects:update,projects:delete,projects:export,projects:manage,products:view,products:create,products:update,products:delete,products:export,products:manage,campaigns:view,campaigns:create,campaigns:update,campaigns:delete,campaigns:export,campaigns:manage,contacts:view,contacts:create,contacts:update,contacts:delete,contacts:export,contacts:manage,leads:view,leads:create,leads:update,leads:delete,leads:export,leads:assign,leads:manage,opportunities:view,opportunities:create,opportunities:update,opportunities:delete,opportunities:export,opportunities:manage,deals:view,deals:create,deals:update,deals:delete,deals:export,deals:manage,orders:view,orders:create,orders:update,orders:delete,orders:export,orders:manage,tickets:view,tickets:create,tickets:update,tickets:delete,tickets:export,tickets:assign,tickets:manage,dashboards:view,dashboards:create,dashboards:update,dashboards:delete,dashboards:export,teams:view,teams:create,teams:update,teams:delete,teams:export,teams:manage,insights:view,insights:create,insights:update,insights:delete,insights:export,insights:manage,storage:view,storage:manage,iot:view,iot:manage,accounts:view,accounts:create,accounts:update,accounts:delete,accounts:export,accounts:manage,audit_log:view,audit_log:export"},
		{"编辑者", "可编辑业务数据的预设角色", "workspace:view,workspace:update,workspace:create,workspace:manage,projects:view,projects:create,projects:update,projects:manage,products:view,products:create,products:update,products:manage,campaigns:view,campaigns:create,campaigns:update,campaigns:manage,contacts:view,contacts:create,contacts:update,contacts:manage,leads:view,leads:create,leads:update,leads:assign,leads:manage,opportunities:view,opportunities:create,opportunities:update,opportunities:manage,deals:view,deals:create,deals:update,deals:manage,orders:view,orders:create,orders:update,orders:manage,tickets:view,tickets:create,tickets:update,tickets:assign,tickets:manage,dashboards:view,teams:view,insights:view,storage:view,iot:view"},
		{"只读用户", "只读访问业务模块的预设角色", "workspace:view,projects:view,products:view,campaigns:view,contacts:view,leads:view,opportunities:view,deals:view,orders:view,tickets:view,dashboards:view,teams:view,insights:view,storage:view,iot:view"},
	}
	for _, permission := range defaultPermissions {
		p := permission
		if err := tx.Where("resource = ? AND action = ?", p.Resource, p.Action).FirstOrCreate(&p).Error; err != nil {
			return err
		}
	}
	for _, preset := range presets {
		permissions := preset.Permissions
		if preset.Name == "管理员" {
			permissions = strings.Join(uniqueCSVPermissions(permissions+",mfg:view,mfg:rfq:view,mfg:rfq:manage,mfg:product:view,mfg:product:manage,mfg:cost:view,mfg:cost:view_sensitive,mfg:cost:manage,mfg:quote:view,mfg:quote:approve,mfg:quote:manage,mfg:sample:view,mfg:sample:manage,mfg:production:view,mfg:production:manage,mfg:ecn:view,mfg:ecn:approve,mfg:ecn:manage,mfg:compensation:view,mfg:compensation:manage,mfg:compensation:approve,mfg:metrics:view,mfg:metrics:manage"), ",")
		}
		permissions = strings.Join(filterOrganizationPermissionsCSV(permissions), ",")
		role := models.Role{
			TenantModel:     models.TenantModel{TenantID: tenantID},
			Name:            preset.Name,
			Description:     preset.Description,
			IsPreset:        true,
			PermissionsJSON: permissions,
		}
		if err := tx.Where("tenant_id = ? AND name = ?", tenantID, preset.Name).FirstOrCreate(&role).Error; err != nil {
			return err
		}
	}
	return nil
}

func uniqueCSVPermissions(value string) []string {
	seen := map[string]bool{}
	items := []string{}
	for _, item := range strings.Split(value, ",") {
		permission := strings.TrimSpace(item)
		if permission == "" || seen[permission] {
			continue
		}
		seen[permission] = true
		items = append(items, permission)
	}
	return items
}

func filterOrganizationPermissionsCSV(value string) []string {
	result := []string{}
	for _, permission := range uniqueCSVPermissions(value) {
		resource, _, ok := splitTenantPermission(permission)
		if !ok || isTenantOrganizationPermissionResource(resource) {
			continue
		}
		result = append(result, permission)
	}
	return result
}

func splitTenantPermission(permission string) (string, string, bool) {
	idx := strings.LastIndex(strings.TrimSpace(permission), ":")
	if idx <= 0 || idx >= len(permission)-1 {
		return "", "", false
	}
	return strings.TrimSpace(permission[:idx]), strings.TrimSpace(permission[idx+1:]), true
}

func isTenantOrganizationPermissionResource(resource string) bool {
	switch strings.TrimSpace(resource) {
	case "enterprise", "accounts", "system:department", "system:post", "system:menu", "storage", "iot", "audit_log":
		return true
	default:
		return false
	}
}

func roleIDByName(tx *gorm.DB, tenantID uint, name string) (uint, error) {
	var role models.Role
	if err := tx.Where("tenant_id = ? AND name = ?", tenantID, name).First(&role).Error; err != nil {
		return 0, err
	}
	return role.ID, nil
}

func quotaFromRequest(req tenantQuotaRequest) models.TenantQuota {
	maxUsers := req.MaxUsers
	if maxUsers <= 0 {
		maxUsers = 20
	}
	storageLimit := req.StorageLimit
	if storageLimit <= 0 {
		storageLimit = baseStorageBytes
	}
	storageLimit = baseStorageBytes
	planCode := strings.TrimSpace(req.PlanCode)
	if planCode == "" {
		planCode = "basic"
	}
	if planCode != "basic" && planCode != "plus" {
		planCode = "basic"
	}
	return models.TenantQuota{
		PlanCode:            planCode,
		MaxUsers:            maxUsers,
		StorageLimit:        storageLimit,
		ValidFrom:           parseDatePtr(req.ValidFrom),
		ValidTo:             parseDatePtr(req.ValidTo),
		Modules:             strings.Join(req.Modules, ","),
		ModuleOverridesJSON: "",
	}
}

func quotaSummaryJSON(quota models.TenantQuota) string {
	data, _ := json.Marshal(map[string]interface{}{
		"plan_code":             quota.PlanCode,
		"max_users":             quota.MaxUsers,
		"storage_limit":         quota.StorageLimit,
		"valid_from":            quota.ValidFrom,
		"valid_to":              quota.ValidTo,
		"modules":               splitCSV(quota.Modules),
		"module_overrides_json": quota.ModuleOverridesJSON,
	})
	return string(data)
}

func parseDatePtr(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func tenantPlanFromRemark(remark string) string {
	remark = strings.TrimSpace(remark)
	if remark == "" {
		return "standard"
	}
	lower := strings.ToLower(remark)
	switch {
	case strings.Contains(lower, "enterprise"), strings.Contains(remark, "企业版"):
		return "enterprise"
	case strings.Contains(lower, "pro"), strings.Contains(remark, "专业版"):
		return "pro"
	default:
		return "standard"
	}
}

func maskUsername(username string) string {
	username = strings.TrimSpace(username)
	runes := []rune(username)
	if len(runes) <= 2 {
		return "**"
	}
	if len(runes) == 3 {
		return string(runes[0]) + "*" + string(runes[2])
	}
	return string(runes[0:2]) + strings.Repeat("*", len(runes)-3) + string(runes[len(runes)-1])
}

func generateBootstrapPassword(length int) string {
	if length < 10 {
		length = 10
	}
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%*"
	bytes := make([]byte, length)
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return "Temp#2026" + fmt.Sprint(time.Now().Unix()%100000)
	}
	for i := range bytes {
		bytes[i] = charset[int(random[i])%len(charset)]
	}
	return string(bytes)
}

func logControllerOperation(ctx *context.Context, auth authContext, tenantID uint, operation, target, descriptionKey string, params map[string]interface{}, result, reason string) {
	if tenantID == 0 && !auth.IsSuperAdmin() {
		tenantID = auth.TenantID
	}
	services.LogOperation(services.OperationLogEvent{
		TenantID:       tenantID,
		UserID:         auth.UserID,
		Username:       auth.Username,
		Operation:      operation,
		Target:         target,
		DescriptionKey: descriptionKey,
		Params:         params,
		Result:         result,
		Reason:         reason,
		IP:             ctx.Input.IP(),
		UserAgent:      ctx.Request.UserAgent(),
		Source:         "web",
	})
}
