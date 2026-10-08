package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
)

type PlatformController struct {
	web.Controller
}

type platformTenantSummary struct {
	TenantID               uint       `json:"tenant_id"`
	Status                 string     `json:"status"`
	PlanCode               string     `json:"plan_code"`
	Plan                   string     `json:"plan"`
	ModuleSet              []string   `json:"module_set"`
	StorageQuota           int64      `json:"storage_quota"`
	StorageUsedBytes       uint64     `json:"storage_used_bytes"`
	StorageUsedMB          float64    `json:"storage_used_mb"`
	UsagePercent           float64    `json:"usage_percent"`
	BaseStorageBytes       int64      `json:"base_storage_bytes"`
	ExpansionStorageGB     int        `json:"expansion_storage_gb"`
	StorageUploadLocked    bool       `json:"storage_upload_locked"`
	StorageLockReason      string     `json:"storage_lock_reason"`
	StorageNextExpireAt    *time.Time `json:"storage_next_expire_at"`
	StorageExpiringSoon    bool       `json:"storage_expiring_soon"`
	StoragePendingCount    int64      `json:"storage_pending_count"`
	MaxUsers               int        `json:"max_users"`
	CurrentUsers           int64      `json:"current_users"`
	ValidTo                *time.Time `json:"valid_to"`
	ExpireAt               *time.Time `json:"expire_at"`
	DaysToExpire           int        `json:"days_to_expire"`
	EffectiveModules       []string   `json:"effective_modules"`
	AdminAccountMasked     string     `json:"admin_account_masked"`
	AIEnabled              bool       `json:"ai_enabled"`
	AIBalanceAmount        string     `json:"ai_balance_amount"`
	AIBalanceMicros        int64      `json:"ai_balance_micros"`
	AIMonthlyTokenLimit    int64      `json:"ai_monthly_token_limit"`
	AIMonthlyTokenUsed     int64      `json:"ai_monthly_token_used"`
	IndustryPackageCode    string     `json:"industry_package_code"`
	IndustryPackageName    string     `json:"industry_package_name"`
	IndustryPackageEnabled bool       `json:"industry_package_enabled"`
}

type moduleOverridePayload struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

type tenantEditRequest struct {
	Status                 string                `json:"status"`
	PlanCode               string                `json:"plan_code"`
	MaxUsers               int                   `json:"max_users"`
	StorageLimit           int64                 `json:"storage_limit"`
	ValidTo                string                `json:"valid_to"`
	ModuleOverrides        moduleOverridePayload `json:"module_overrides"`
	AIEnabled              bool                  `json:"ai_enabled"`
	AIBalanceAmount        float64               `json:"ai_balance_amount"`
	AIMonthlyTokenLimit    int64                 `json:"ai_monthly_token_limit"`
	IndustryPackageCode    string                `json:"industry_package_code"`
	IndustryPackageEnabled *bool                 `json:"industry_package_enabled"`
}

type platformPlanRequest struct {
	PlanCode        string   `json:"plan_code"`
	Name            string   `json:"name"`
	MaxUsersDefault int      `json:"max_users_default"`
	StorageDefault  int64    `json:"storage_default"`
	Modules         []string `json:"modules"`
	Status          string   `json:"status"`
	Remark          string   `json:"remark"`
}

type runtimeNode struct {
	NodeID              string    `json:"node_id"`
	Host                string    `json:"host"`
	Healthy             bool      `json:"healthy"`
	CheckedAt           time.Time `json:"checked_at"`
	UptimeSeconds       int64     `json:"uptime_seconds"`
	Goroutines          int       `json:"goroutines"`
	CPUUsagePercent     float64   `json:"cpu_usage_percent"`
	CPUCount            int       `json:"cpu_count"`
	MemoryTotalBytes    uint64    `json:"memory_total_bytes"`
	MemoryUsedBytes     uint64    `json:"memory_used_bytes"`
	MemoryUsedPercent   float64   `json:"memory_used_percent"`
	DiskTotalBytes      uint64    `json:"disk_total_bytes"`
	DiskUsedBytes       uint64    `json:"disk_used_bytes"`
	DiskFreeBytes       uint64    `json:"disk_free_bytes"`
	DiskUsedPercent     float64   `json:"disk_used_percent"`
	FileserverHealthy   int       `json:"fileserver_healthy"`
	FileserverTotal     int       `json:"fileserver_total"`
	FileserverCandidate int       `json:"fileserver_candidate"`
}

type dependencyNode struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Healthy      bool      `json:"healthy"`
	CheckedAt    time.Time `json:"checked_at"`
	LatencyMS    float64   `json:"latency_ms"`
	Address      string    `json:"address"`
	Version      string    `json:"version,omitempty"`
	Database     string    `json:"database,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

var platformBootTime = time.Now()

func (c *PlatformController) TenantSummaries() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	status := strings.TrimSpace(c.Ctx.Input.Query("status"))
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)

	query := utils.DB.Model(&models.Tenant{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	query.Count(&total)

	var tenants []models.Tenant
	if err := query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&tenants).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取租户运营摘要失败")
		return
	}

	items := make([]platformTenantSummary, 0, len(tenants))
	for _, tenant := range tenants {
		item := buildPlatformTenantSummary(tenant)
		items = append(items, item)
	}

	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": total, "page": page, "size": size})
	logControllerOperation(c.Ctx, auth, 0, "LIST_PLATFORM_TENANT_SUMMARY", "platform.tenants.summary", "operation.platform_tenants_summary", map[string]interface{}{"status": status, "page": page, "size": size}, "success", "")
}

func (c *PlatformController) UpdateTenant() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	var req tenantEditRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	req.PlanCode = strings.TrimSpace(req.PlanCode)
	if req.PlanCode == "" {
		req.PlanCode = "basic"
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if req.Status != "active" && req.Status != "disabled" {
		utils.BadRequest(c.Ctx, "状态必须为 active 或 disabled")
		return
	}
	validTo := parseDatePtr(req.ValidTo)
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		var tenant models.Tenant
		if err := tx.Where("id = ?", tenantID).First(&tenant).Error; err != nil {
			return err
		}
		if err := tx.Model(&tenant).Update("status", req.Status).Error; err != nil {
			return err
		}

		var quota models.TenantQuota
		if err := tx.Where("tenant_id = ?", tenantID).First(&quota).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			quota = models.TenantQuota{TenantID: tenantID}
		}

		planModules, planMaxUsers, planStorage, err := resolvePlanDefaultsTx(tx, req.PlanCode)
		if err != nil {
			return err
		}
		if req.MaxUsers > 0 {
			quota.MaxUsers = req.MaxUsers
		} else if quota.MaxUsers <= 0 {
			quota.MaxUsers = planMaxUsers
		}
		if req.StorageLimit > 0 {
			quota.StorageLimit = baseStorageBytes
		} else if quota.StorageLimit <= 0 {
			quota.StorageLimit = baseStorageBytes
		}
		_ = planStorage
		quota.PlanCode = req.PlanCode
		quota.ValidTo = validTo

		// 租户编辑只允许选择套餐，模块集始终由套餐定义。
		quota.ModuleOverridesJSON = ""
		effective := uniqueStrings(planModules)
		quota.Modules = strings.Join(effective, ",")
		features := map[string]interface{}{}
		_ = json.Unmarshal([]byte(quota.FeatureSwitch), &features)
		features["ai"] = map[string]interface{}{"enabled": req.AIEnabled}
		quota.FeatureSwitch = toJSONString(features)

		if quota.ID == 0 {
			if err := tx.Create(&quota).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&quota).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Tenant{}).Where("id = ?", tenantID).Update("quota_json", quotaSummaryJSON(quota)).Error; err != nil {
			return err
		}
		var wallet models.TenantAIWallet
		if err := tx.Where("tenant_id = ?", tenantID).First(&wallet).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			wallet = models.TenantAIWallet{TenantModel: models.TenantModel{TenantID: tenantID}, Currency: "USD", HardStop: true, LastUsageResetMonth: time.Now().Format("2006-01")}
		}
		wallet.Enabled = req.AIEnabled
		wallet.BalanceMicros = int64(req.AIBalanceAmount * 1000000)
		if req.AIMonthlyTokenLimit > 0 {
			wallet.MonthlyTokenLimit = req.AIMonthlyTokenLimit
		}
		if strings.TrimSpace(wallet.Currency) == "" {
			wallet.Currency = "USD"
		}
		if wallet.ID == 0 {
			if err := tx.Create(&wallet).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&wallet).Error; err != nil {
			return err
		}
		if req.IndustryPackageEnabled != nil || strings.TrimSpace(req.IndustryPackageCode) != "" {
			enabled := false
			if req.IndustryPackageEnabled != nil {
				enabled = *req.IndustryPackageEnabled
			}
			_, err = services.UpsertTenantIndustryPackageTx(tx, tenantID, services.IndustryPackageUpsert{
				PackageCode: req.IndustryPackageCode,
				Enabled:     enabled,
			})
			return err
		}
		return nil
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.NotFound(c.Ctx, "租户不存在")
			return
		}
		logControllerOperation(c.Ctx, auth, 0, "UPDATE_PLATFORM_TENANT", fmt.Sprintf("租户[%d]", tenantID), "operation.update_tenant_quota", map[string]interface{}{"tenant_id": tenantID}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新租户失败")
		return
	}

	var tenant models.Tenant
	_ = utils.DB.Where("id = ?", tenantID).First(&tenant).Error
	summary := buildPlatformTenantSummary(tenant)
	logControllerOperation(c.Ctx, auth, 0, "UPDATE_PLATFORM_TENANT", fmt.Sprintf("租户[%d]", tenantID), "operation.update_tenant_quota", map[string]interface{}{"tenant_id": tenantID, "plan_code": req.PlanCode, "status": req.Status}, "success", "")
	utils.Success(c.Ctx, summary)
}

func (c *PlatformController) ListPlans() {
	_, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	status := strings.TrimSpace(c.Ctx.Input.Query("status"))
	query := utils.DB.Model(&models.PlatformPlanTemplate{}).Where("plan_code IN ?", []string{"basic", "plus"})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	var plans []models.PlatformPlanTemplate
	if err := query.Order("plan_code asc").Find(&plans).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取套餐模板失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(plans))
	for _, plan := range plans {
		items = append(items, map[string]interface{}{
			"plan_code":         plan.PlanCode,
			"name":              plan.Name,
			"max_users_default": plan.MaxUsersDefault,
			"storage_default":   plan.StorageDefault,
			"modules":           parseModulesJSON(plan.ModulesJSON),
			"status":            plan.Status,
			"version":           plan.Version,
			"remark":            plan.Remark,
			"updated_at":        plan.UpdatedAt,
		})
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items})
}

func (c *PlatformController) CreatePlan() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	var req platformPlanRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.PlanCode = strings.TrimSpace(req.PlanCode)
	req.Name = strings.TrimSpace(req.Name)
	req.Status = strings.TrimSpace(req.Status)
	if req.PlanCode == "" || req.Name == "" {
		utils.BadRequest(c.Ctx, "plan_code 和 name 不能为空")
		return
	}
	if req.PlanCode != "basic" && req.PlanCode != "plus" {
		utils.BadRequest(c.Ctx, "仅允许维护 basic 或 plus 套餐")
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if req.Status != "active" && req.Status != "disabled" {
		utils.BadRequest(c.Ctx, "status 仅支持 active/disabled")
		return
	}
	if req.MaxUsersDefault <= 0 {
		req.MaxUsersDefault = 20
	}
	if req.StorageDefault <= 0 {
		req.StorageDefault = baseStorageBytes
	}
	plan := models.PlatformPlanTemplate{
		PlanCode:        req.PlanCode,
		Name:            req.Name,
		MaxUsersDefault: req.MaxUsersDefault,
		StorageDefault:  req.StorageDefault,
		ModulesJSON:     toJSONString(uniqueStrings(req.Modules)),
		Status:          req.Status,
		Version:         1,
		Remark:          req.Remark,
	}
	if err := utils.DB.Create(&plan).Error; err != nil {
		logControllerOperation(c.Ctx, auth, 0, "CREATE_PLATFORM_PLAN", req.PlanCode, "operation.platform_settings_draft", nil, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusBadRequest, "创建套餐模板失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"plan_code": plan.PlanCode})
}

func (c *PlatformController) UpdatePlan() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	planCode := strings.TrimSpace(c.Ctx.Input.Param(":code"))
	if planCode == "" {
		utils.BadRequest(c.Ctx, "plan code 不能为空")
		return
	}
	if planCode != "basic" && planCode != "plus" {
		utils.BadRequest(c.Ctx, "仅允许维护 basic 或 plus 套餐")
		return
	}
	var req platformPlanRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Status = strings.TrimSpace(req.Status)
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		var plan models.PlatformPlanTemplate
		if err := tx.Where("plan_code = ?", planCode).First(&plan).Error; err != nil {
			return err
		}
		if req.Name != "" {
			plan.Name = req.Name
		}
		if req.MaxUsersDefault > 0 {
			plan.MaxUsersDefault = req.MaxUsersDefault
		}
		if req.StorageDefault > 0 {
			plan.StorageDefault = req.StorageDefault
		}
		if len(req.Modules) > 0 {
			plan.ModulesJSON = toJSONString(uniqueStrings(req.Modules))
		}
		if req.Status == "active" || req.Status == "disabled" {
			plan.Status = req.Status
		}
		if strings.TrimSpace(req.Remark) != "" {
			plan.Remark = req.Remark
		}
		plan.Version++
		if err := tx.Save(&plan).Error; err != nil {
			return err
		}
		return syncTenantQuotasForPlanTx(tx, planCode, parseModulesJSON(plan.ModulesJSON), plan.MaxUsersDefault, plan.StorageDefault)
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.NotFound(c.Ctx, "套餐不存在")
			return
		}
		logControllerOperation(c.Ctx, auth, 0, "UPDATE_PLATFORM_PLAN", planCode, "operation.platform_settings_publish", nil, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新套餐失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"plan_code": planCode, "updated": true})
}

func syncTenantQuotasForPlanTx(tx *gorm.DB, planCode string, modules []string, maxUsers int, storageLimit int64) error {
	modules = uniqueStrings(modules)
	modulesCSV := strings.Join(modules, ",")
	var quotas []models.TenantQuota
	if err := tx.Where("plan_code = ?", planCode).Find(&quotas).Error; err != nil {
		return err
	}
	for _, quota := range quotas {
		quota.Modules = modulesCSV
		quota.ModuleOverridesJSON = ""
		if quota.MaxUsers <= 0 && maxUsers > 0 {
			quota.MaxUsers = maxUsers
		}
		if quota.StorageLimit <= 0 && storageLimit > 0 {
			quota.StorageLimit = baseStorageBytes
		}
		if err := tx.Save(&quota).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Tenant{}).Where("id = ?", quota.TenantID).Update("quota_json", quotaSummaryJSON(quota)).Error; err != nil {
			return err
		}
	}
	return nil
}

func (c *PlatformController) ListExpiringTenants() {
	_, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	windowDays := queryInt(c.Ctx, "window_days", 30)
	if windowDays <= 0 {
		windowDays = 30
	}
	now := time.Now()
	windowEnd := now.AddDate(0, 0, windowDays)

	type row struct {
		TenantID     uint
		TenantStatus string
		PlanCode     string
		ValidTo      *time.Time
		MaxUsers     int
	}
	rows := make([]row, 0)
	err := utils.DB.Table("tenants t").
		Select("t.id as tenant_id, t.status as tenant_status, q.plan_code, q.valid_to, q.max_users").
		Joins("LEFT JOIN tenant_quota q ON q.tenant_id = t.id").
		Where("q.valid_to IS NOT NULL").
		Where("q.valid_to <= ?", windowEnd).
		Order("q.valid_to asc").
		Scan(&rows).Error
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取到期提醒失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, it := range rows {
		days := 0
		if it.ValidTo != nil {
			days = int(it.ValidTo.Sub(now).Hours() / 24)
		}
		noticeType := "expiring"
		if days < 0 {
			noticeType = "expired"
		}
		items = append(items, map[string]interface{}{
			"tenant_id":      it.TenantID,
			"tenant_status":  it.TenantStatus,
			"plan_code":      it.PlanCode,
			"valid_to":       it.ValidTo,
			"days_to_expire": days,
			"notice_type":    noticeType,
			"max_users":      it.MaxUsers,
		})
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "window_days": windowDays, "total": len(items)})
}

func (c *PlatformController) RuntimeOverview() {
	_, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	node := collectLocalRuntimeNode()
	fs := services.FileserverNodeEvaluationView()
	healthyCount := 0
	candidateCount := 0
	for _, n := range fs.Nodes {
		if n.Healthy {
			healthyCount++
		}
		if n.IsCandidate {
			candidateCount++
		}
	}
	node.FileserverTotal = len(fs.Nodes)
	node.FileserverHealthy = healthyCount
	node.FileserverCandidate = candidateCount
	dependencies := collectDependencyNodes()
	dependencyHealthy := 0
	for _, dep := range dependencies {
		if dep.Healthy {
			dependencyHealthy++
		}
	}

	utils.Success(c.Ctx, map[string]interface{}{
		"summary": map[string]interface{}{
			"backend_nodes":         1,
			"backend_healthy_nodes": 1,
			"fileserver_total":      len(fs.Nodes),
			"fileserver_healthy":    healthyCount,
			"fileserver_candidate":  candidateCount,
			"dependency_total":      len(dependencies),
			"dependency_healthy":    dependencyHealthy,
		},
		"backend_nodes": []runtimeNode{node},
		"fileserver": map[string]interface{}{
			"thresholds":          fs.Thresholds,
			"recommended_node_id": fs.RecommendedNodeID,
			"nodes":               fs.Nodes,
		},
		"dependencies": dependencies,
	})
}

func (c *PlatformController) CreateTenant() {
	tenantController := &TenantController{}
	tenantController.Ctx = c.Ctx
	tenantController.CreateTenant()
}

func (c *PlatformController) ResetTenantAdminPassword() {
	tenantController := &TenantController{}
	tenantController.Ctx = c.Ctx
	tenantController.ResetTenantAdminPassword()
}

func (c *PlatformController) ListSettings() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	group := strings.TrimSpace(c.Ctx.Input.Query("group"))
	key := strings.TrimSpace(c.Ctx.Input.Query("key"))
	status := strings.TrimSpace(c.Ctx.Input.Query("status"))
	if status == "" {
		status = "active"
	}

	query := utils.DB.Model(&models.PlatformSettingVersion{}).Where("scope = ?", "platform")
	if group != "" {
		query = query.Where("`group` = ?", group)
	}
	if key != "" {
		query = query.Where("`key` = ?", key)
	}
	if status != "all" {
		query = query.Where("status = ?", status)
	}
	var items []models.PlatformSettingVersion
	if err := query.Order("`group` asc, `key` asc, version desc").Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取平台配置失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items})
	logControllerOperation(c.Ctx, auth, 0, "LIST_PLATFORM_SETTINGS", "platform.settings", "operation.platform_settings_list", map[string]interface{}{"group": group, "key": key, "status": status}, "success", "")
}

func (c *PlatformController) SaveSettingDraft() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		Group string                 `json:"group"`
		Key   string                 `json:"key"`
		Value map[string]interface{} `json:"value"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Group = strings.TrimSpace(req.Group)
	req.Key = strings.TrimSpace(req.Key)
	if req.Group == "" || req.Key == "" {
		utils.BadRequest(c.Ctx, "group 和 key 不能为空")
		return
	}

	valueJSON, _ := json.Marshal(req.Value)
	var created models.PlatformSettingVersion
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		var currentVersion int
		if err := tx.Model(&models.PlatformSettingVersion{}).
			Where("scope = ? AND `group` = ? AND `key` = ?", "platform", req.Group, req.Key).
			Select("COALESCE(MAX(version),0)").Scan(&currentVersion).Error; err != nil {
			return err
		}
		created = models.PlatformSettingVersion{
			Scope:     "platform",
			Group:     req.Group,
			Key:       req.Key,
			ValueJSON: string(valueJSON),
			Version:   currentVersion + 1,
			Status:    "draft",
			CreatedBy: auth.UserID,
		}
		return tx.Create(&created).Error
	})
	if err != nil {
		logControllerOperation(c.Ctx, auth, 0, "SAVE_PLATFORM_SETTING_DRAFT", fmt.Sprintf("%s.%s", req.Group, req.Key), "operation.platform_settings_draft", req.Value, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存平台配置草稿失败")
		return
	}
	logControllerOperation(c.Ctx, auth, 0, "SAVE_PLATFORM_SETTING_DRAFT", fmt.Sprintf("%s.%s", req.Group, req.Key), "operation.platform_settings_draft", map[string]interface{}{"version": created.Version}, "success", "")
	utils.Success(c.Ctx, created)
}

func (c *PlatformController) PublishSetting() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		ID uint `json:"id"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.ID == 0 {
		utils.BadRequest(c.Ctx, "id 不能为空")
		return
	}
	now := time.Now()
	var updated models.PlatformSettingVersion
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND scope = ?", req.ID, "platform").First(&updated).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.PlatformSettingVersion{}).
			Where("scope = ? AND `group` = ? AND `key` = ? AND status = ?", "platform", updated.Group, updated.Key, "active").
			Updates(map[string]interface{}{"status": "rollback"}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.PlatformSettingVersion{}).Where("id = ?", updated.ID).Updates(map[string]interface{}{
			"status":       "active",
			"effective_at": &now,
			"published_by": auth.UserID,
			"published_at": &now,
		}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", updated.ID).First(&updated).Error
	})
	if err != nil {
		logControllerOperation(c.Ctx, auth, 0, "PUBLISH_PLATFORM_SETTING", "platform.settings", "operation.platform_settings_publish", map[string]interface{}{"id": req.ID}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "发布平台配置失败")
		return
	}
	logControllerOperation(c.Ctx, auth, 0, "PUBLISH_PLATFORM_SETTING", fmt.Sprintf("%s.%s", updated.Group, updated.Key), "operation.platform_settings_publish", map[string]interface{}{"id": updated.ID, "version": updated.Version}, "success", "")
	utils.Success(c.Ctx, updated)
}

func (c *PlatformController) RollbackSetting() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		Group string `json:"group"`
		Key   string `json:"key"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Group = strings.TrimSpace(req.Group)
	req.Key = strings.TrimSpace(req.Key)
	if req.Group == "" || req.Key == "" {
		utils.BadRequest(c.Ctx, "group 和 key 不能为空")
		return
	}
	var active models.PlatformSettingVersion
	var target models.PlatformSettingVersion
	now := time.Now()
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("scope = ? AND `group` = ? AND `key` = ? AND status = ?", "platform", req.Group, req.Key, "active").Order("version desc").First(&active).Error; err != nil {
			return err
		}
		if err := tx.Where("scope = ? AND `group` = ? AND `key` = ? AND id <> ?", "platform", req.Group, req.Key, active.ID).
			Order("version desc").First(&target).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.PlatformSettingVersion{}).Where("id = ?", active.ID).Update("status", "rollback").Error; err != nil {
			return err
		}
		if err := tx.Model(&models.PlatformSettingVersion{}).Where("id = ?", target.ID).Updates(map[string]interface{}{
			"status":              "active",
			"effective_at":        &now,
			"published_by":        auth.UserID,
			"published_at":        &now,
			"rolled_back_from_id": active.ID,
		}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", target.ID).First(&target).Error
	})
	if err != nil {
		logControllerOperation(c.Ctx, auth, 0, "ROLLBACK_PLATFORM_SETTING", fmt.Sprintf("%s.%s", req.Group, req.Key), "operation.platform_settings_rollback", map[string]interface{}{"group": req.Group, "key": req.Key}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "回滚平台配置失败")
		return
	}
	logControllerOperation(c.Ctx, auth, 0, "ROLLBACK_PLATFORM_SETTING", fmt.Sprintf("%s.%s", req.Group, req.Key), "operation.platform_settings_rollback", map[string]interface{}{"id": target.ID, "version": target.Version}, "success", "")
	utils.Success(c.Ctx, target)
}

func buildPlatformTenantSummary(tenant models.Tenant) platformTenantSummary {
	item := platformTenantSummary{
		TenantID:     tenant.ID,
		Status:       tenant.Status,
		PlanCode:     "basic",
		Plan:         tenantPlanFromRemark(tenant.Remark),
		ModuleSet:    []string{},
		StorageQuota: 0,
		UsagePercent: 0,
		MaxUsers:     0,
		CurrentUsers: 0,
		ExpireAt:     nil,
		ValidTo:      nil,
	}
	var quota models.TenantQuota
	if err := utils.DB.Where("tenant_id = ?", tenant.ID).First(&quota).Error; err == nil {
		if strings.TrimSpace(quota.PlanCode) != "" {
			item.PlanCode = quota.PlanCode
			item.Plan = quota.PlanCode
		}
		item.ModuleSet = splitCSV(quota.Modules)
		item.EffectiveModules = splitCSV(quota.Modules)
		item.StorageQuota = quota.StorageLimit
		item.MaxUsers = quota.MaxUsers
		item.ExpireAt = quota.ValidTo
		item.ValidTo = quota.ValidTo
		item.DaysToExpire = daysToExpire(quota.ValidTo)
		features := map[string]interface{}{}
		_ = json.Unmarshal([]byte(quota.FeatureSwitch), &features)
		item.AIEnabled = boolValue(mapValue(features["ai"])["enabled"], false)
	}
	var wallet models.TenantAIWallet
	if err := utils.DB.Where("tenant_id = ?", tenant.ID).First(&wallet).Error; err == nil {
		item.AIEnabled = wallet.Enabled
		item.AIBalanceMicros = wallet.BalanceMicros
		item.AIBalanceAmount = microsToAmount(wallet.BalanceMicros)
		item.AIMonthlyTokenLimit = wallet.MonthlyTokenLimit
		item.AIMonthlyTokenUsed = wallet.MonthlyTokenUsed
	}
	if usage, err := storageUsageForPlatformSummary(tenant.ID); err == nil {
		item.StorageQuota = usage.TotalBytes
		item.StorageUsedBytes = usage.UsedBytes
		item.StorageUsedMB = roundFloat(float64(usage.UsedBytes) / float64(1024*1024))
		item.UsagePercent = usage.UsagePercent
		item.BaseStorageBytes = usage.BaseBytes
		item.ExpansionStorageGB = usage.ExpansionGB
		item.StorageUploadLocked = usage.UploadLocked
		item.StorageLockReason = usage.LockReason
		item.StorageNextExpireAt = usage.NextExpireAt
		item.StorageExpiringSoon = usage.ExpiringSoon
		item.StoragePendingCount = storageExpansionCountByStatus(tenant.ID, "pending")
	}
	var activeUsers int64
	_ = utils.DB.Model(&models.User{}).Where("tenant_id = ? AND status = ?", tenant.ID, "active").Count(&activeUsers).Error
	item.CurrentUsers = activeUsers
	var admin models.User
	if err := utils.DB.Select("username").Where("tenant_id = ? AND account_type = ?", tenant.ID, "tenant_admin").Order("id asc").First(&admin).Error; err == nil {
		item.AdminAccountMasked = strings.TrimSpace(admin.Username)
	}
	if assignment, exists, err := services.TenantIndustryPackage(tenant.ID); err == nil && exists {
		item.IndustryPackageCode = assignment.PackageCode
		item.IndustryPackageEnabled = assignment.Enabled
		if assignment.PackageCode == services.IndustryPackageManufacturing {
			item.IndustryPackageName = services.ManufacturingPackageInfo().Name
		}
	}
	return item
}

func parseModulesJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		return uniqueStrings(arr)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err == nil {
		if modules, ok := payload["modules"].([]interface{}); ok {
			items := make([]string, 0, len(modules))
			for _, m := range modules {
				items = append(items, strings.TrimSpace(fmt.Sprint(m)))
			}
			return uniqueStrings(items)
		}
	}
	return splitCSV(raw)
}

func resolvePlanDefaultsTx(tx *gorm.DB, planCode string) ([]string, int, int64, error) {
	var plan models.PlatformPlanTemplate
	if err := tx.Where("plan_code = ?", planCode).First(&plan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			planCode = "basic"
			if err2 := tx.Where("plan_code = ?", planCode).First(&plan).Error; err2 != nil {
				if err2 == gorm.ErrRecordNotFound {
					return []string{}, 20, baseStorageBytes, nil
				}
				return nil, 0, 0, err2
			}
		} else {
			return nil, 0, 0, err
		}
	}
	modules := parseModulesJSON(plan.ModulesJSON)
	return modules, plan.MaxUsersDefault, plan.StorageDefault, nil
}

func effectiveModules(planModules []string, explicit []string, overrideJSON string) []string {
	base := uniqueStrings(planModules)
	if len(explicit) > 0 {
		base = uniqueStrings(explicit)
	}
	if strings.TrimSpace(overrideJSON) == "" {
		return base
	}
	var payload moduleOverridePayload
	if err := json.Unmarshal([]byte(overrideJSON), &payload); err != nil {
		return base
	}
	set := map[string]bool{}
	for _, item := range base {
		set[strings.TrimSpace(item)] = true
	}
	for _, add := range payload.Add {
		add = strings.TrimSpace(add)
		if add != "" {
			set[add] = true
		}
	}
	for _, del := range payload.Remove {
		del = strings.TrimSpace(del)
		if del != "" {
			delete(set, del)
		}
	}
	result := make([]string, 0, len(set))
	for item := range set {
		if item != "" {
			result = append(result, item)
		}
	}
	sort.Strings(result)
	return result
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		normalized := strings.TrimSpace(item)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		result = append(result, normalized)
	}
	return result
}

func daysToExpire(validTo *time.Time) int {
	if validTo == nil {
		return 0
	}
	return int(validTo.Sub(time.Now()).Hours() / 24)
}

func collectLocalRuntimeNode() runtimeNode {
	host, _ := os.Hostname()
	if host == "" {
		host = "localhost"
	}
	now := time.Now()
	memTotal, memUsed := hostMemoryStats()
	diskTotal, diskUsed, diskFree := rootDiskStats()
	cpuPercent := sampledCPUUsagePercent()

	memoryPercent := 0.0
	if memTotal > 0 {
		memoryPercent = (float64(memUsed) / float64(memTotal)) * 100
	}
	diskPercent := 0.0
	if diskTotal > 0 {
		diskPercent = (float64(diskUsed) / float64(diskTotal)) * 100
	}
	return runtimeNode{
		NodeID:            "local",
		Host:              host,
		Healthy:           true,
		CheckedAt:         now,
		UptimeSeconds:     int64(now.Sub(platformBootTime).Seconds()),
		Goroutines:        runtime.NumGoroutine(),
		CPUUsagePercent:   roundFloat(cpuPercent),
		CPUCount:          runtime.NumCPU(),
		MemoryTotalBytes:  memTotal,
		MemoryUsedBytes:   memUsed,
		MemoryUsedPercent: roundFloat(memoryPercent),
		DiskTotalBytes:    diskTotal,
		DiskUsedBytes:     diskUsed,
		DiskFreeBytes:     diskFree,
		DiskUsedPercent:   roundFloat(diskPercent),
	}
}

func collectDependencyNodes() []dependencyNode {
	return []dependencyNode{
		collectMySQLDependencyNode(),
		collectRedisDependencyNode(),
	}
}

func collectMySQLDependencyNode() dependencyNode {
	now := time.Now()
	node := dependencyNode{
		Name:      "MySQL",
		Type:      "mysql",
		CheckedAt: now,
		Database:  dbConfigValue("DB_NAME", "db_name", ""),
		Address:   mysqlMonitorAddress(),
	}
	start := time.Now()
	sqlDB, err := utils.DB.DB()
	if err != nil {
		node.ErrorMessage = err.Error()
		return node
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var version string
	err = sqlDB.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	node.LatencyMS = roundFloat(float64(time.Since(start).Microseconds()) / 1000)
	if err != nil {
		node.ErrorMessage = err.Error()
		return node
	}
	stats := sqlDB.Stats()
	node.Healthy = true
	node.Version = version
	node.ErrorMessage = fmt.Sprintf("open=%d idle=%d in_use=%d", stats.OpenConnections, stats.Idle, stats.InUse)
	return node
}

func collectRedisDependencyNode() dependencyNode {
	now := time.Now()
	node := dependencyNode{
		Name:      "Redis",
		Type:      "redis",
		CheckedAt: now,
		Address:   redisMonitorAddress(),
	}
	if utils.RedisClient == nil {
		node.ErrorMessage = "redis client not initialized"
		return node
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := utils.RedisClient.Ping(ctx).Err(); err != nil {
		node.LatencyMS = roundFloat(float64(time.Since(start).Microseconds()) / 1000)
		node.ErrorMessage = err.Error()
		return node
	}
	node.LatencyMS = roundFloat(float64(time.Since(start).Microseconds()) / 1000)
	if info, err := utils.RedisClient.Info(ctx, "server").Result(); err == nil {
		node.Version = parseRedisVersion(info)
	}
	if db, err := strconv.Atoi(dbConfigValue("REDIS_DB", "redis_db", "0")); err == nil {
		node.Database = fmt.Sprintf("db%d", db)
	}
	stats := utils.RedisClient.PoolStats()
	node.Healthy = true
	node.ErrorMessage = fmt.Sprintf("hits=%d misses=%d timeouts=%d total_conns=%d idle_conns=%d", stats.Hits, stats.Misses, stats.Timeouts, stats.TotalConns, stats.IdleConns)
	return node
}

func mysqlMonitorAddress() string {
	host := dbConfigValue("DB_HOST", "db_host", "127.0.0.1")
	port := dbConfigValue("DB_PORT", "db_port", "3306")
	if host == "127.0.0.1" || host == "localhost" {
		socket := strings.TrimSpace(os.Getenv("DB_SOCKET"))
		if socket == "" {
			socket = "/var/run/mysqld/mysqld.sock"
		}
		return "unix:" + socket
	}
	return host + ":" + port
}

func redisMonitorAddress() string {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr != "" {
		return addr
	}
	host := dbConfigValue("", "redis_host", "127.0.0.1")
	port := dbConfigValue("", "redis_port", "6379")
	return host + ":" + port
}

func dbConfigValue(envKey, confKey, fallback string) string {
	if envKey != "" {
		if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
			return value
		}
	}
	if confKey != "" {
		if value, err := web.AppConfig.String(confKey); err == nil && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return fallback
}

func parseRedisVersion(info string) string {
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "redis_version:") {
			return strings.TrimPrefix(line, "redis_version:")
		}
	}
	return ""
}

func roundFloat(value float64) float64 {
	parsed, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return parsed
}

func rootDiskStats() (uint64, uint64, uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0, 0, 0
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free
	return total, used, free
}

func hostMemoryStats() (uint64, uint64) {
	content, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	lines := strings.Split(string(content), "\n")
	memTotal := uint64(0)
	memAvailable := uint64(0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "MemTotal:") {
			memTotal = parseMeminfoKB(line) * 1024
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			memAvailable = parseMeminfoKB(line) * 1024
		}
	}
	if memTotal == 0 {
		return 0, 0
	}
	if memAvailable > memTotal {
		memAvailable = 0
	}
	return memTotal, memTotal - memAvailable
}

type cpuStatSnapshot struct {
	idle  uint64
	total uint64
}

func sampledCPUUsagePercent() float64 {
	first, ok := readCPUStatSnapshot()
	if !ok {
		return 0
	}
	time.Sleep(1 * time.Second)
	second, ok := readCPUStatSnapshot()
	if !ok || second.total <= first.total {
		return 0
	}
	totalDelta := second.total - first.total
	idleDelta := uint64(0)
	if second.idle > first.idle {
		idleDelta = second.idle - first.idle
	}
	if idleDelta >= totalDelta {
		return 0
	}
	return (float64(totalDelta-idleDelta) / float64(totalDelta)) * 100
}

func readCPUStatSnapshot() (cpuStatSnapshot, bool) {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuStatSnapshot{}, false
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return cpuStatSnapshot{}, false
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuStatSnapshot{}, false
	}
	values := make([]uint64, 0, len(fields)-1)
	total := uint64(0)
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuStatSnapshot{}, false
		}
		values = append(values, value)
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return cpuStatSnapshot{idle: idle, total: total}, total > 0
}

func parseMeminfoKB(line string) uint64 {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}
	value, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func StartTenantExpiryNoticeScheduler() {
	run := func() {
		if err := generateTenantExpiryNotices(); err != nil {
			utils.LogWarning("租户到期提醒生成失败: %v", err)
		}
	}
	services.WithSchedulerLock("tenant_expiry_notice_scheduler", run)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			services.WithSchedulerLock("tenant_expiry_notice_scheduler", run)
		}
	}()
}

func generateTenantExpiryNotices() error {
	now := time.Now()
	windowEnd := now.AddDate(0, 0, 30)
	type row struct {
		TenantID uint
		ValidTo  *time.Time
		Email    string
	}
	rows := make([]row, 0)
	err := utils.DB.Table("tenants t").
		Select("t.id as tenant_id, q.valid_to, u.email").
		Joins("LEFT JOIN tenant_quota q ON q.tenant_id = t.id").
		Joins("LEFT JOIN users u ON u.id = t.admin_user_id").
		Where("q.valid_to IS NOT NULL").
		Where("q.valid_to <= ?", windowEnd).
		Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.ValidTo == nil {
			continue
		}
		days := int(row.ValidTo.Sub(now).Hours() / 24)
		noticeType := "expiring"
		if days < 0 {
			noticeType = "expired"
		}
		noticeDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		notice := models.TenantExpiryNotice{
			TenantID:   row.TenantID,
			NoticeType: noticeType,
			NoticeDate: noticeDate,
			SentTo:     strings.TrimSpace(row.Email),
			Status:     "generated",
			DaysLeft:   days,
		}
		if err := utils.DB.Where("tenant_id = ? AND notice_type = ? AND notice_date = ?", notice.TenantID, notice.NoticeType, notice.NoticeDate).
			Assign(notice).FirstOrCreate(&notice).Error; err != nil {
			return err
		}
	}
	return nil
}
