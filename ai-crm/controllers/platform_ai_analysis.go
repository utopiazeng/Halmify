package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type aiAnalysisPackageRequest struct {
	Code              string `json:"code"`
	Name              string `json:"name"`
	Currency          string `json:"currency"`
	PriceAmount       string `json:"price_amount"`
	PriceMicros       int64  `json:"price_micros"`
	BalanceAmount     string `json:"balance_amount"`
	BalanceMicros     int64  `json:"balance_micros"`
	MonthlyTokenLimit int64  `json:"monthly_token_limit"`
	ValidityDays      int    `json:"validity_days"`
	Description       string `json:"description"`
	Status            string `json:"status"`
	Sort              int    `json:"sort"`
}

type aiApplicationDecisionRequest struct {
	Reason                  string `json:"reason"`
	BalanceMicros           int64  `json:"balance_micros"`
	MonthlyTokenLimit       int64  `json:"monthly_token_limit"`
	AdditionalBalanceMicros int64  `json:"additional_balance_micros"`
}

func (c *PlatformController) ListAIAnalysisApplications() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	_ = auth
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)
	if size > 100 {
		size = 100
	}
	status := strings.TrimSpace(c.Ctx.Input.Query("status"))
	tenantID := queryUint(c.Ctx, "tenant_id")
	query := utils.DB.Model(&models.TenantAIApplication{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	var total int64
	_ = query.Count(&total).Error
	var rows []models.TenantAIApplication
	_ = query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		item := tenantAIApplicationResponse(row)
		item["tenant_name"] = tenantName(row.TenantID)
		items = append(items, item)
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": total})
}

func (c *PlatformController) ApproveAIAnalysisApplication() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的 AI 分析申请")
		return
	}
	var req aiApplicationDecisionRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var app models.TenantAIApplication
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&app).Error; err != nil {
			return err
		}
		if app.Status != "pending" {
			return fmt.Errorf("只有待审批申请可以通过")
		}
		var pkg models.AIAnalysisPackage
		if app.PackageCode != "" {
			_ = tx.Where("code = ?", app.PackageCode).First(&pkg).Error
		}
		balance := firstPositiveInt64(req.BalanceMicros, req.AdditionalBalanceMicros, pkg.BalanceMicros)
		tokenLimit := firstPositiveInt64(req.MonthlyTokenLimit, pkg.MonthlyTokenLimit)
		var quota models.TenantQuota
		if err := tx.Where("tenant_id = ?", app.TenantID).First(&quota).Error; err == nil {
			features := mapValueFromJSON(quota.FeatureSwitch)
			features["ai"] = map[string]interface{}{"enabled": true}
			quota.FeatureSwitch = toJSONString(features)
			if err := tx.Save(&quota).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Tenant{}).Where("id = ?", app.TenantID).Update("quota_json", quotaSummaryJSON(quota)).Error; err != nil {
				return err
			}
		}
		var wallet models.TenantAIWallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ?", app.TenantID).First(&wallet).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			wallet = models.TenantAIWallet{TenantModel: models.TenantModel{TenantID: app.TenantID}, Currency: firstNonEmpty(pkg.Currency, defaultAICurrency), HardStop: true, LastUsageResetMonth: time.Now().Format("2006-01")}
		}
		beforeBalance := wallet.BalanceMicros
		beforeReserved := wallet.ReservedMicros
		wallet.Enabled = true
		if strings.TrimSpace(wallet.Currency) == "" {
			wallet.Currency = firstNonEmpty(pkg.Currency, defaultAICurrency)
		}
		if balance > 0 {
			wallet.BalanceMicros += balance
		}
		if tokenLimit > 0 {
			wallet.MonthlyTokenLimit = tokenLimit
		}
		if wallet.ID == 0 {
			if err := tx.Create(&wallet).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&wallet).Error; err != nil {
			return err
		}
		now := time.Now()
		app.Status = "approved"
		app.ApprovedBy = auth.UserID
		app.ApprovedAt = &now
		if err := tx.Save(&app).Error; err != nil {
			return err
		}
		if balance > 0 {
			ledger := models.AITokenLedgerEntry{
				TenantModel:          models.TenantModel{TenantID: app.TenantID},
				UserID:               app.UserID,
				JobID:                0,
				EntryType:            "purchase",
				Direction:            "credit",
				AmountMicros:         balance,
				PriceSnapshotJSON:    toJSONString(map[string]interface{}{"package_code": app.PackageCode, "approved_by": auth.UserID}),
				BalanceBeforeMicros:  beforeBalance,
				BalanceAfterMicros:   wallet.BalanceMicros,
				ReservedBeforeMicros: beforeReserved,
				ReservedAfterMicros:  wallet.ReservedMicros,
				Status:               "succeeded",
				TraceID:              newTenantAnalysisTraceID(app.TenantID, auth.UserID),
				Description:          "AI 分析套餐开通充值",
			}
			if err := tx.Create(&ledger).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		utils.Error(c.Ctx, http.StatusBadRequest, err.Error())
		return
	}
	logControllerOperation(c.Ctx, auth, app.TenantID, "APPROVE_AI_ANALYSIS_APPLICATION", fmt.Sprintf("AI分析申请[%d]", app.ID), "operation.approve_ai_analysis_application", map[string]interface{}{"application_id": app.ID, "tenant_id": app.TenantID}, "success", "")
	utils.Success(c.Ctx, tenantAIApplicationResponse(app))
}

func (c *PlatformController) RejectAIAnalysisApplication() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的 AI 分析申请")
		return
	}
	var req aiApplicationDecisionRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var app models.TenantAIApplication
	if err := utils.DB.Where("id = ?", id).First(&app).Error; err != nil {
		utils.NotFound(c.Ctx, "AI 分析申请不存在")
		return
	}
	if app.Status != "pending" {
		utils.BadRequest(c.Ctx, "只有待审批申请可以驳回")
		return
	}
	now := time.Now()
	app.Status = "rejected"
	app.RejectedBy = auth.UserID
	app.RejectedAt = &now
	app.RejectReason = strings.TrimSpace(req.Reason)
	if err := utils.DB.Save(&app).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "驳回 AI 分析申请失败")
		return
	}
	logControllerOperation(c.Ctx, auth, app.TenantID, "REJECT_AI_ANALYSIS_APPLICATION", fmt.Sprintf("AI分析申请[%d]", app.ID), "operation.reject_ai_analysis_application", map[string]interface{}{"application_id": app.ID, "tenant_id": app.TenantID}, "success", "")
	utils.Success(c.Ctx, tenantAIApplicationResponse(app))
}

func (c *PlatformController) ListAIAnalysisPackages() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	_ = auth
	var rows []models.AIAnalysisPackage
	_ = utils.DB.Order("sort asc, id asc").Find(&rows).Error
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, aiAnalysisPackageResponse(row))
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": len(items)})
}

func (c *PlatformController) SaveAIAnalysisPackage() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	var req aiAnalysisPackageRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	if req.Code == "" || req.Name == "" {
		utils.BadRequest(c.Ctx, "套餐编码和名称不能为空")
		return
	}
	if req.PriceMicros <= 0 {
		req.PriceMicros = amountStringToMicros(req.PriceAmount)
	}
	if req.BalanceMicros <= 0 {
		req.BalanceMicros = amountStringToMicros(req.BalanceAmount)
	}
	if strings.TrimSpace(req.Currency) == "" {
		req.Currency = defaultAICurrency
	}
	if strings.TrimSpace(req.Status) == "" {
		req.Status = "active"
	}
	var pkg models.AIAnalysisPackage
	err := utils.DB.Where("code = ?", req.Code).First(&pkg).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		utils.Error(c.Ctx, http.StatusInternalServerError, "读取 AI 分析套餐失败")
		return
	}
	pkg.Code = req.Code
	pkg.Name = req.Name
	pkg.Currency = strings.TrimSpace(req.Currency)
	pkg.PriceMicros = req.PriceMicros
	pkg.BalanceMicros = req.BalanceMicros
	pkg.MonthlyTokenLimit = req.MonthlyTokenLimit
	pkg.ValidityDays = req.ValidityDays
	pkg.Description = strings.TrimSpace(req.Description)
	pkg.Status = strings.TrimSpace(req.Status)
	pkg.Sort = req.Sort
	if pkg.ID == 0 {
		err = utils.DB.Create(&pkg).Error
	} else {
		err = utils.DB.Save(&pkg).Error
	}
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存 AI 分析套餐失败")
		return
	}
	logControllerOperation(c.Ctx, auth, 0, "SAVE_AI_ANALYSIS_PACKAGE", fmt.Sprintf("AI套餐[%s]", pkg.Code), "operation.save_ai_analysis_package", map[string]interface{}{"package_code": pkg.Code}, "success", "")
	utils.Success(c.Ctx, aiAnalysisPackageResponse(pkg))
}

func (c *PlatformController) AIAnalysisUsageSummary() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	_ = auth
	var wallets []models.TenantAIWallet
	_ = utils.DB.Order("tenant_id asc").Find(&wallets).Error
	items := make([]map[string]interface{}, 0, len(wallets))
	for _, wallet := range wallets {
		item := aiWalletResponse(wallet)
		item["tenant_id"] = wallet.TenantID
		item["tenant_name"] = tenantName(wallet.TenantID)
		items = append(items, item)
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": len(items)})
}

func (c *PlatformController) AIAnalysisLedger() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	_ = auth
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)
	if size > 100 {
		size = 100
	}
	tenantID := queryUint(c.Ctx, "tenant_id")
	query := utils.DB.Model(&models.AITokenLedgerEntry{})
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	var total int64
	_ = query.Count(&total).Error
	var rows []models.AITokenLedgerEntry
	_ = query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		item := aiLedgerResponse(row)
		item["tenant_name"] = tenantName(row.TenantID)
		items = append(items, item)
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": total})
}

func (c *PlatformController) AIAnalysisTenantWallet() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	_ = auth
	tenantID, valid := pathUint(c.Ctx, ":tenant_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户")
		return
	}
	wallet, err := ensureTenantAIWallet(tenantID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "读取 AI 钱包失败")
		return
	}
	payload := aiWalletResponse(wallet)
	payload["tenant_id"] = tenantID
	payload["tenant_name"] = tenantName(tenantID)
	utils.Success(c.Ctx, payload)
}

func tenantName(tenantID uint) string {
	if tenantID == 0 {
		return ""
	}
	var tenant models.Tenant
	if err := utils.DB.Select("name").Where("id = ?", tenantID).First(&tenant).Error; err == nil {
		return tenant.Name
	}
	return ""
}

func amountStringToMicros(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	var parsed float64
	if _, err := fmt.Sscanf(value, "%f", &parsed); err != nil {
		return 0
	}
	return amountFloatToMicros(parsed)
}

func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
