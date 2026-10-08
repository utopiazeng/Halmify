package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	companyResearchFeature = "company_research"
	defaultAIProvider      = deepSeekProvider
	defaultAIModel         = defaultDeepSeekModel
	defaultAICurrency      = "USD"
	defaultPriceMicrosPM   = int64(322000)
	defaultMaxResearchURLs = 6
	defaultMaxResearchCost = int64(5000000)
)

type companyResearchCreateRequest struct {
	CompanyID    uint   `json:"company_id"`
	CompanyName  string `json:"company_name"`
	Website      string `json:"website"`
	Region       string `json:"region"`
	IndustryHint string `json:"industry_hint"`
	Note         string `json:"note"`
	Mode         string `json:"mode"`
}

type companyResearchApplyRequest struct {
	Fields            []string `json:"fields"`
	OverwriteExisting bool     `json:"overwrite_existing"`
}

func (c *CRMController) CreateCompanyResearchJob() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "contacts", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req companyResearchCreateRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.Website = strings.TrimSpace(req.Website)
	req.Region = strings.TrimSpace(req.Region)
	req.IndustryHint = strings.TrimSpace(req.IndustryHint)
	req.Note = strings.TrimSpace(req.Note)
	if req.Mode == "" {
		req.Mode = "deep"
	}

	var company models.CRMObject
	if req.CompanyID > 0 {
		if err := crmScopedObjectQuery(auth, "contacts").Where("id = ? AND object_type = ?", req.CompanyID, "company").First(&company).Error; err != nil {
			utils.NotFound(c.Ctx, "公司不存在")
			return
		}
		payload := payloadMap(company.PayloadJSON)
		if req.CompanyName == "" {
			req.CompanyName = company.Name
		}
		if req.Website == "" {
			req.Website = firstPayloadText(payload, "website", "domain")
		}
		if req.IndustryHint == "" {
			req.IndustryHint = firstPayloadText(payload, "industry")
		}
		if req.Note == "" {
			req.Note = firstPayloadText(payload, "note", "description")
		}
	}
	if req.CompanyName == "" && req.Website == "" {
		utils.BadRequest(c.Ctx, "公司名称和官网至少填写一项")
		return
	}
	normalizedWebsite, err := normalizeResearchURL(req.Website)
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	req.Website = normalizedWebsite

	settings := resolveAIResearchSettings()
	if err := ensureTenantAIAvailable(auth, settings.EstimatedMaxTokens, settings.MaxCostMicros); err != nil {
		recordBlockedAIUsage(auth, req.CompanyID, err.Error(), settings)
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}

	inputJSON := toJSONString(map[string]interface{}{
		"company_id":    req.CompanyID,
		"company_name":  req.CompanyName,
		"website":       req.Website,
		"region":        req.Region,
		"industry_hint": req.IndustryHint,
		"note":          req.Note,
		"mode":          req.Mode,
	})
	job := models.AIResearchJob{
		TenantModel:  models.TenantModel{TenantID: auth.TenantID},
		UserID:       auth.UserID,
		ObjectDomain: "contacts",
		ObjectType:   "company",
		ObjectID:     req.CompanyID,
		Status:       "pending",
		InputJSON:    inputJSON,
	}
	if err := utils.DB.Create(&job).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建 AI 调研任务失败")
		return
	}
	go processCompanyResearchJob(auth, job.ID, req, settings)
	utils.Success(c.Ctx, companyResearchJobResponse(job))
}

func (c *CRMController) GetCompanyResearchJob() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "contacts", "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的调研任务")
		return
	}
	var job models.AIResearchJob
	if err := utils.DB.Where("tenant_id = ? AND id = ?", auth.TenantID, id).First(&job).Error; err != nil {
		utils.NotFound(c.Ctx, "调研任务不存在")
		return
	}
	utils.Success(c.Ctx, companyResearchJobResponse(job))
}

func (c *CRMController) ApplyCompanyResearchJob() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "contacts", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的调研任务")
		return
	}
	var req companyResearchApplyRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if len(req.Fields) == 0 {
		utils.BadRequest(c.Ctx, "请选择要应用的字段")
		return
	}
	var job models.AIResearchJob
	if err := utils.DB.Where("tenant_id = ? AND id = ? AND object_domain = ? AND object_type = ?", auth.TenantID, id, "contacts", "company").First(&job).Error; err != nil {
		utils.NotFound(c.Ctx, "调研任务不存在")
		return
	}
	if job.Status != "succeeded" {
		utils.BadRequest(c.Ctx, "只有成功完成的调研结果可以应用")
		return
	}
	if job.ObjectID == 0 {
		utils.BadRequest(c.Ctx, "调研任务未关联公司，请先保存公司后再应用")
		return
	}
	result := mapValueFromJSON(job.ResultJSON)
	formFill := mapValue(firstNonNil(result["form_fill"], result["suggestions"]))
	if len(formFill) == 0 {
		utils.BadRequest(c.Ctx, "调研结果没有可应用字段")
		return
	}
	var company models.CRMObject
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "contacts", "company", job.ObjectID).First(&company).Error; err != nil {
			return err
		}
		payload := payloadMap(company.PayloadJSON)
		applied := []string{}
		for _, field := range req.Fields {
			key := strings.TrimSpace(field)
			if key == "" {
				continue
			}
			if !allowedResearchSuggestionFields()[key] {
				continue
			}
			suggestion := mapValue(formFill[key])
			if len(suggestion) == 0 {
				continue
			}
			value := suggestion["value"]
			if isEmptyValue(value) {
				continue
			}
			if key == "name" {
				if req.OverwriteExisting || strings.TrimSpace(company.Name) == "" {
					company.Name = strings.TrimSpace(fmt.Sprint(value))
					applied = append(applied, key)
				}
				continue
			}
			if key == "tags" {
				tags := stringSlice(value)
				if len(tags) > 0 {
					company.Tags = mergeTagText(company.Tags, tags)
					applied = append(applied, key)
				}
				continue
			}
			if req.OverwriteExisting || isEmptyValue(payload[key]) {
				payload[key] = value
				applied = append(applied, key)
			}
		}
		payload["ai_research"] = map[string]interface{}{
			"job_id":          job.ID,
			"usage_log_id":    job.UsageLogID,
			"confidence":      job.Confidence,
			"updated_at":      time.Now().Format(time.RFC3339),
			"applied":         applied,
			"company_profile": mapValue(firstNonNil(result["company_profile"], result["profile"])),
			"form_fill":       formFill,
			"insights":        mapValue(result["insights"]),
		}
		company.PayloadJSON = toJSONString(payload)
		company.UpdatedBy = auth.UserID
		return tx.Save(&company).Error
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "应用 AI 调研结果失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "APPLY_COMPANY_RESEARCH", fmt.Sprintf("公司[%d]", company.ID), "operation.apply_company_research", map[string]interface{}{"job_id": job.ID, "fields": req.Fields}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"company": company, "job": companyResearchJobResponse(job)})
}

func (c *CRMController) ListCompanyResearchJobs() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "enterprise", "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	companyID := queryUint(c.Ctx, "company_id")
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 10)
	if size > 50 {
		size = 50
	}
	query := utils.DB.Model(&models.AIResearchJob{}).Where("tenant_id = ? AND object_domain = ? AND object_type = ?", auth.TenantID, "contacts", "company")
	if companyID > 0 {
		query = query.Where("object_id = ?", companyID)
	}
	var total int64
	query.Count(&total)
	items := []models.AIResearchJob{}
	if err := query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取调研历史失败")
		return
	}
	responses := []map[string]interface{}{}
	for _, item := range items {
		responses = append(responses, companyResearchJobResponse(item))
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": responses, "total": total})
}

func (c *CRMController) AIUsageSummary() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "enterprise", "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	from := strings.TrimSpace(c.Ctx.Input.Query("from"))
	to := strings.TrimSpace(c.Ctx.Input.Query("to"))
	query := utils.DB.Model(&models.AIUsageLog{}).Where("tenant_id = ?", auth.TenantID)
	if parsed, ok := parseDateBoundary(from, false); ok {
		query = query.Where("created_at >= ?", parsed)
	}
	if parsed, ok := parseDateBoundary(to, true); ok {
		query = query.Where("created_at < ?", parsed)
	}
	var logs []models.AIUsageLog
	if err := query.Order("created_at desc").Limit(500).Find(&logs).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取 AI 用量失败")
		return
	}
	totalTokens := int64(0)
	totalCost := int64(0)
	successCalls := int64(0)
	featureStats := map[string]map[string]interface{}{}
	responses := []map[string]interface{}{}
	for _, log := range logs {
		totalTokens += log.TotalTokens
		totalCost += log.CostMicros
		if log.Status == "succeeded" {
			successCalls++
		}
		key := log.Feature
		if key == "" {
			key = "unknown"
		}
		item := featureStats[key]
		if item == nil {
			item = map[string]interface{}{"feature": key, "calls": int64(0), "tokens": int64(0), "cost_micros": int64(0), "cost_amount": "0.000000"}
			featureStats[key] = item
		}
		item["calls"] = item["calls"].(int64) + 1
		item["tokens"] = item["tokens"].(int64) + log.TotalTokens
		item["cost_micros"] = item["cost_micros"].(int64) + log.CostMicros
		item["cost_amount"] = microsToAmount(item["cost_micros"].(int64))
		responses = append(responses, aiUsageResponse(log))
	}
	features := []map[string]interface{}{}
	for _, item := range featureStats {
		features = append(features, item)
	}
	wallet, _ := ensureTenantAIWallet(auth.TenantID)
	utils.Success(c.Ctx, map[string]interface{}{
		"wallet": map[string]interface{}{
			"enabled":                wallet.Enabled,
			"currency":               wallet.Currency,
			"balance_amount":         microsToAmount(wallet.BalanceMicros),
			"balance_micros":         wallet.BalanceMicros,
			"total_spent_amount":     microsToAmount(wallet.TotalSpentMicros),
			"monthly_token_limit":    wallet.MonthlyTokenLimit,
			"monthly_token_used":     wallet.MonthlyTokenUsed,
			"hard_stop":              wallet.HardStop,
			"last_usage_reset_month": wallet.LastUsageResetMonth,
		},
		"summary": map[string]interface{}{
			"calls":         len(logs),
			"success_calls": successCalls,
			"total_tokens":  totalTokens,
			"cost_amount":   microsToAmount(totalCost),
			"cost_micros":   totalCost,
		},
		"features": features,
		"items":    responses,
	})
}

func (c *CRMController) AIUsageLogs() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "enterprise", "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	from := strings.TrimSpace(c.Ctx.Input.Query("from"))
	to := strings.TrimSpace(c.Ctx.Input.Query("to"))
	status := strings.TrimSpace(c.Ctx.Input.Query("status"))
	feature := strings.TrimSpace(c.Ctx.Input.Query("feature"))
	query := utils.DB.Model(&models.AIUsageLog{}).Where("tenant_id = ?", auth.TenantID)
	if parsed, ok := parseDateBoundary(from, false); ok {
		query = query.Where("created_at >= ?", parsed)
	}
	if parsed, ok := parseDateBoundary(to, true); ok {
		query = query.Where("created_at < ?", parsed)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if feature != "" {
		query = query.Where("feature = ?", feature)
	}
	var total int64
	query.Count(&total)
	var logs []models.AIUsageLog
	if err := query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&logs).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取 AI 消费明细失败")
		return
	}
	items := []map[string]interface{}{}
	for _, log := range logs {
		items = append(items, aiUsageResponse(log))
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": total, "page": page, "size": size})
}

type aiResearchSettings struct {
	Provider           string
	Model              string
	Currency           string
	UnitPriceMicrosPM  int64
	EstimatedMaxTokens int64
	MaxCostMicros      int64
	MaxURLs            int
	RequestTimeout     time.Duration
	APIURL             string
	APIKey             string
	RAGAPIURL          string
	RAGAPIKey          string
}

func resolveAIResearchSettings() aiResearchSettings {
	apiURL := strings.TrimSpace(firstNonEmpty(os.Getenv("AI_API_URL"), web.AppConfig.DefaultString("ai_api_url", ""), os.Getenv("DEEPSEEK_API_URL"), web.AppConfig.DefaultString("deepseek_api_url", ""), "https://api.deepseek.com"))
	model := strings.TrimSpace(firstNonEmpty(os.Getenv("AI_MODEL"), web.AppConfig.DefaultString("ai_model", ""), os.Getenv("DEEPSEEK_MODEL"), web.AppConfig.DefaultString("deepseek_model", ""), defaultAIModel))
	settings := aiResearchSettings{
		Provider:           defaultAIProvider,
		Model:              model,
		Currency:           defaultAICurrency,
		UnitPriceMicrosPM:  defaultPriceMicrosPM,
		EstimatedMaxTokens: 25000,
		MaxCostMicros:      defaultMaxResearchCost,
		MaxURLs:            defaultMaxResearchURLs,
		RequestTimeout:     8 * time.Second,
		APIURL:             apiURL,
		APIKey:             strings.TrimSpace(firstNonEmpty(os.Getenv("AI_API_KEY"), web.AppConfig.DefaultString("ai_api_key", ""), os.Getenv("DEEPSEEK_API_KEY"), web.AppConfig.DefaultString("deepseek_api_key", ""))),
		RAGAPIURL:          strings.TrimSpace(firstNonEmpty(os.Getenv("RAG_API_URL"), web.AppConfig.DefaultString("rag_api_url", ""))),
		RAGAPIKey:          strings.TrimSpace(firstNonEmpty(os.Getenv("RAG_API_KEY"), web.AppConfig.DefaultString("rag_api_key", ""))),
	}
	settings.Provider = inferAIProvider(settings.APIURL, settings.Model)
	var active models.PlatformSettingVersion
	if err := utils.DB.Where("scope = ? AND `group` = ? AND `key` = ? AND status = ?", "platform", "ai", "company_research", "active").Order("version desc").First(&active).Error; err == nil {
		payload := mapValueFromJSON(active.ValueJSON)
		if value := strings.TrimSpace(fmt.Sprint(payload["provider"])); value != "" && value != "<nil>" {
			settings.Provider = value
		}
		if value := strings.TrimSpace(fmt.Sprint(payload["model"])); value != "" && value != "<nil>" {
			settings.Model = value
		}
		if value := strings.TrimSpace(fmt.Sprint(payload["currency"])); value != "" && value != "<nil>" {
			settings.Currency = value
		}
		if price := int64(numericValue(payload["unit_price_micros_per_million"])); price > 0 {
			settings.UnitPriceMicrosPM = price
		}
		if price := int64(numericValue(payload["price_per_million_micros"])); price > 0 {
			settings.UnitPriceMicrosPM = price
		}
		if tokens := int64(numericValue(payload["estimated_max_tokens"])); tokens > 0 {
			settings.EstimatedMaxTokens = tokens
		}
		if cost := int64(numericValue(payload["max_cost_micros"])); cost > 0 {
			settings.MaxCostMicros = cost
		}
		if maxURLs := int(numericValue(payload["max_urls"])); maxURLs > 0 {
			settings.MaxURLs = maxURLs
		}
		if value := strings.TrimSpace(fmt.Sprint(payload["api_url"])); value != "" && value != "<nil>" {
			settings.APIURL = value
		}
		if value := strings.TrimSpace(fmt.Sprint(payload["api_key"])); value != "" && value != "<nil>" {
			settings.APIKey = value
		}
	}
	if value := strings.TrimSpace(firstNonEmpty(os.Getenv("AI_API_URL"), web.AppConfig.DefaultString("ai_api_url", ""))); value != "" {
		settings.APIURL = value
	}
	if value := strings.TrimSpace(firstNonEmpty(os.Getenv("AI_API_KEY"), web.AppConfig.DefaultString("ai_api_key", ""))); value != "" {
		settings.APIKey = value
	}
	if value := strings.TrimSpace(firstNonEmpty(os.Getenv("AI_MODEL"), web.AppConfig.DefaultString("ai_model", ""))); value != "" {
		settings.Model = value
	}
	settings.Model = effectiveDeepSeekModel(settings.Model)
	settings.Provider = inferAIProvider(settings.APIURL, settings.Model)
	return settings
}

func inferAIProvider(apiURL string, model string) string {
	lower := strings.ToLower(apiURL + " " + model)
	if strings.Contains(lower, "deepseek") {
		return deepSeekProvider
	}
	return "openai-compatible"
}

func aiChatCompletionsEndpoint(settings aiResearchSettings) string {
	endpoint := strings.TrimRight(settings.APIURL, "/")
	if endpoint == "" {
		endpoint = "https://api.deepseek.com"
	}
	if strings.HasSuffix(endpoint, "/chat/completions") {
		return endpoint
	}
	if strings.HasSuffix(endpoint, "/v1") {
		return endpoint + "/chat/completions"
	}
	if settings.Provider == deepSeekProvider {
		return endpoint + "/chat/completions"
	}
	return endpoint + "/v1/chat/completions"
}

func ensureTenantAIAvailable(auth authContext, estimatedTokens int64, maxCostMicros int64) error {
	if auth.TenantID == 0 {
		return fmt.Errorf("缺少租户上下文")
	}
	if !tenantFeatureAIEnabled(auth.TenantID) {
		return fmt.Errorf("当前租户未开通 AI 功能")
	}
	wallet, err := ensureTenantAIWallet(auth.TenantID)
	if err != nil {
		return fmt.Errorf("读取 AI 钱包失败")
	}
	resetWalletMonthIfNeeded(&wallet)
	if !wallet.Enabled {
		return fmt.Errorf("当前租户未开通 AI 功能")
	}
	if wallet.MonthlyTokenLimit > 0 && wallet.MonthlyTokenUsed+estimatedTokens > wallet.MonthlyTokenLimit {
		return fmt.Errorf("本月 AI token 额度不足")
	}
	estimatedCost := maxCostMicros
	if estimatedCost <= 0 {
		estimatedCost = calculateAICostMicros(estimatedTokens, defaultPriceMicrosPM)
	}
	if wallet.HardStop && wallet.BalanceMicros < estimatedCost {
		return fmt.Errorf("AI 账户余额不足")
	}
	return nil
}

func tenantFeatureAIEnabled(tenantID uint) bool {
	var quota models.TenantQuota
	if err := utils.DB.Where("tenant_id = ?", tenantID).First(&quota).Error; err != nil {
		return false
	}
	features := mapValueFromJSON(quota.FeatureSwitch)
	ai := mapValue(features["ai"])
	if len(ai) > 0 {
		if enabled, ok := ai["enabled"]; ok {
			return boolValue(enabled, false)
		}
	}
	return boolValue(features["ai_enabled"], false)
}

func ensureTenantAIWallet(tenantID uint) (models.TenantAIWallet, error) {
	wallet := models.TenantAIWallet{}
	err := utils.DB.Where("tenant_id = ?", tenantID).First(&wallet).Error
	if err == nil {
		return wallet, nil
	}
	if err != gorm.ErrRecordNotFound {
		return wallet, err
	}
	wallet = models.TenantAIWallet{
		TenantModel:         models.TenantModel{TenantID: tenantID},
		Currency:            defaultAICurrency,
		HardStop:            true,
		Enabled:             false,
		LastUsageResetMonth: time.Now().Format("2006-01"),
	}
	return wallet, utils.DB.Create(&wallet).Error
}

func resetWalletMonthIfNeeded(wallet *models.TenantAIWallet) {
	current := time.Now().Format("2006-01")
	if wallet.LastUsageResetMonth == current {
		return
	}
	wallet.MonthlyTokenUsed = 0
	wallet.LastUsageResetMonth = current
	_ = utils.DB.Save(wallet).Error
}

func parseDateBoundary(value string, end bool) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			if end && layout == "2006-01-02" {
				parsed = parsed.AddDate(0, 0, 1)
			}
			return parsed, true
		}
	}
	return time.Time{}, false
}

func recordBlockedAIUsage(auth authContext, objectID uint, reason string, settings aiResearchSettings) {
	log := models.AIUsageLog{
		TenantModel:               models.TenantModel{TenantID: auth.TenantID},
		UserID:                    auth.UserID,
		Feature:                   companyResearchFeature,
		ObjectDomain:              "contacts",
		ObjectType:                "company",
		ObjectID:                  objectID,
		Provider:                  settings.Provider,
		Model:                     settings.Model,
		UnitPriceMicrosPerMillion: settings.UnitPriceMicrosPM,
		Currency:                  settings.Currency,
		Status:                    "blocked",
		TraceID:                   newResearchTraceID(objectID),
		ErrorMessage:              reason,
	}
	_ = utils.DB.Create(&log).Error
}

func processCompanyResearchJob(auth authContext, jobID uint, req companyResearchCreateRequest, settings aiResearchSettings) {
	started := time.Now()
	_ = utils.DB.Model(&models.AIResearchJob{}).Where("tenant_id = ? AND id = ?", auth.TenantID, jobID).Updates(map[string]interface{}{"status": "running"}).Error
	result, sources, promptTokens, completionTokens, err := runBuiltinCompanyResearch(req, settings)
	status := "succeeded"
	errorMessage := ""
	if err != nil {
		status = "failed"
		errorMessage = err.Error()
		result = map[string]interface{}{"suggestions": map[string]interface{}{}, "insights": map[string]interface{}{"summary": errorMessage}}
	}
	totalTokens := promptTokens + completionTokens
	pricingSnapshot := ensureAIPricingSnapshot(0)
	priceModel := pricingModel(pricingSnapshot, settings.Model)
	inputRate := pricingRate(priceModel, "input_cache_miss")
	outputRate := pricingRate(priceModel, "output")
	if settings.Provider != deepSeekProvider {
		priceModel = aiPricingModel{Provider: settings.Provider, Model: settings.Model, DisplayName: settings.Model}
		inputRate = aiPricingRate{
			Metric:                      "input_output",
			Label:                       "输入/输出 token",
			Currency:                    settings.Currency,
			TenantPriceMicrosPerMillion: settings.UnitPriceMicrosPM,
			TenantPriceAmount:           microsToAmount(settings.UnitPriceMicrosPM),
		}
		outputRate = inputRate
		pricingSnapshot.Provider = settings.Provider
		pricingSnapshot.Currency = settings.Currency
		pricingSnapshot.SourceURL = ""
		pricingSnapshot.MarginRate = 0
	}
	if inputRate.TenantPriceMicrosPerMillion <= 0 {
		inputRate.TenantPriceMicrosPerMillion = settings.UnitPriceMicrosPM
		inputRate.TenantPriceAmount = microsToAmount(settings.UnitPriceMicrosPM)
	}
	if outputRate.TenantPriceMicrosPerMillion <= 0 {
		outputRate.TenantPriceMicrosPerMillion = settings.UnitPriceMicrosPM
		outputRate.TenantPriceAmount = microsToAmount(settings.UnitPriceMicrosPM)
	}
	costMicros := calculateAICostMicros(promptTokens, inputRate.TenantPriceMicrosPerMillion) + calculateAICostMicros(completionTokens, outputRate.TenantPriceMicrosPerMillion)
	effectivePrice := settings.UnitPriceMicrosPM
	if totalTokens > 0 && costMicros > 0 {
		effectivePrice = int64(math.Ceil(float64(costMicros) * 1000000.0 / float64(totalTokens)))
	}
	traceID := newResearchTraceID(jobID)
	usageLogID := uint(0)
	if txErr := utils.DB.Transaction(func(tx *gorm.DB) error {
		wallet := models.TenantAIWallet{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ?", auth.TenantID).First(&wallet).Error; err != nil {
			return err
		}
		currentMonth := time.Now().Format("2006-01")
		if wallet.LastUsageResetMonth != currentMonth {
			wallet.MonthlyTokenUsed = 0
			wallet.LastUsageResetMonth = currentMonth
		}
		usage := models.AIUsageLog{
			TenantModel:               models.TenantModel{TenantID: auth.TenantID},
			UserID:                    auth.UserID,
			Feature:                   companyResearchFeature,
			ObjectDomain:              "contacts",
			ObjectType:                "company",
			ObjectID:                  req.CompanyID,
			Provider:                  settings.Provider,
			Model:                     settings.Model,
			PromptTokens:              promptTokens,
			CompletionTokens:          completionTokens,
			TotalTokens:               totalTokens,
			UnitPriceMicrosPerMillion: effectivePrice,
			CostMicros:                costMicros,
			Currency:                  pricingSnapshot.Currency,
			Status:                    status,
			TraceID:                   traceID,
			LatencyMS:                 time.Since(started).Milliseconds(),
			ErrorMessage:              errorMessage,
			MetadataJSON: toJSONString(map[string]interface{}{
				"source_count": len(sources),
				"mode":         req.Mode,
				"pricing": map[string]interface{}{
					"provider":                          pricingSnapshot.Provider,
					"model":                             settings.Model,
					"pricing_model":                     priceModel.Model,
					"source_url":                        pricingSnapshot.SourceURL,
					"margin_rate":                       pricingSnapshot.MarginRate,
					"input_metric":                      inputRate.Metric,
					"input_price_per_million":           inputRate.TenantPriceAmount,
					"input_price_micros_per_million":    inputRate.TenantPriceMicrosPerMillion,
					"output_metric":                     outputRate.Metric,
					"output_price_per_million":          outputRate.TenantPriceAmount,
					"output_price_micros_per_million":   outputRate.TenantPriceMicrosPerMillion,
					"official_input_price_per_million":  inputRate.OfficialPriceAmount,
					"official_output_price_per_million": outputRate.OfficialPriceAmount,
				},
			}),
		}
		if err := tx.Create(&usage).Error; err != nil {
			return err
		}
		usageLogID = usage.ID
		if costMicros > 0 {
			wallet.BalanceMicros -= costMicros
			wallet.TotalSpentMicros += costMicros
		}
		wallet.MonthlyTokenUsed += totalTokens
		if err := tx.Save(&wallet).Error; err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&models.AIResearchJob{}).Where("tenant_id = ? AND id = ?", auth.TenantID, jobID).Updates(map[string]interface{}{
			"status":        status,
			"result_json":   toJSONString(result),
			"source_json":   toJSONString(sources),
			"confidence":    numericValue(result["confidence"]),
			"usage_log_id":  usage.ID,
			"error_message": errorMessage,
			"finished_at":   &now,
		}).Error
	}); txErr != nil {
		now := time.Now()
		_ = utils.DB.Model(&models.AIResearchJob{}).Where("tenant_id = ? AND id = ?", auth.TenantID, jobID).Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": txErr.Error(),
			"finished_at":   &now,
		}).Error
		return
	}
	if status == "succeeded" && req.CompanyID > 0 {
		updateCompanyResearchSnapshot(auth, req.CompanyID, jobID, usageLogID, result, sources)
	}
}

func runBuiltinCompanyResearch(req companyResearchCreateRequest, settings aiResearchSettings) (map[string]interface{}, []map[string]interface{}, int64, int64, error) {
	sources := collectCompanyResearchSources(req, settings)
	usableSources := usableResearchSources(sources)
	if len(usableSources) == 0 {
		sources = append(sources, collectPublicKnowledgeSources(req, settings)...)
		usableSources = usableResearchSources(sources)
	}
	if len(usableSources) == 0 && strings.TrimSpace(settings.APIKey) != "" {
		result, promptTokens, completionTokens, err := callAICompanyResearch(req, []map[string]interface{}{}, settings)
		if err == nil {
			return normalizeAIResearchResult(req, result, []map[string]interface{}{}), sources, promptTokens, completionTokens, nil
		}
		if isKnownCompanyName(req.CompanyName) {
			fallback := buildPublicKnowledgeResearchResult(req, usableSources, err.Error())
			return fallback, sources, promptTokens, completionTokens, nil
		}
	}
	if len(usableSources) == 0 {
		if isKnownCompanyName(req.CompanyName) {
			return buildPublicKnowledgeResearchResult(req, usableSources, "公开网页抓取不足，使用模型通用知识降级生成"), sources, 0, 0, nil
		}
		return lowQualityResearchResult(req, sources, "可用公开资料不足，需人工核实"), sources, 0, 0, nil
	}
	if strings.TrimSpace(settings.APIKey) == "" {
		return nil, sources, 0, 0, fmt.Errorf("AI API Key 未配置，无法完成 AI 调研")
	}
	result, promptTokens, completionTokens, err := callAICompanyResearch(req, usableSources, settings)
	if err != nil {
		fallback := buildPublicKnowledgeResearchResult(req, usableSources, err.Error())
		return fallback, sources, promptTokens, completionTokens, nil
	}
	result = normalizeAIResearchResult(req, result, usableSources)
	return result, sources, promptTokens, completionTokens, nil
}

func collectCompanyResearchSources(req companyResearchCreateRequest, settings aiResearchSettings) []map[string]interface{} {
	items := []map[string]interface{}{}
	seen := map[string]bool{}
	addURL := func(raw string) {
		if len(items) >= settings.MaxURLs {
			return
		}
		normalized, err := normalizeResearchURL(raw)
		if err != nil || normalized == "" || seen[normalized] {
			return
		}
		seen[normalized] = true
		title, text, err := fetchResearchPage(normalized, settings.RequestTimeout)
		if err != nil || strings.TrimSpace(text) == "" {
			items = append(items, map[string]interface{}{"title": titleOrURL(title, normalized), "url": normalized, "fetched_at": time.Now().Format(time.RFC3339), "summary": "页面暂时无法访问或无可抽取正文", "status": "unavailable", "relevance_score": 0})
			return
		}
		score := researchSourceRelevance(req, title, text, normalized)
		status := "ok"
		if score < 0.35 {
			status = "low_quality"
		}
		items = append(items, map[string]interface{}{"title": titleOrURL(title, normalized), "url": normalized, "fetched_at": time.Now().Format(time.RFC3339), "summary": text, "status": status, "relevance_score": score})
	}
	if req.Website != "" {
		addURL(req.Website)
		base, _ := url.Parse(req.Website)
		if base != nil {
			root := base.Scheme + "://" + base.Host
			for _, path := range []string{"/about", "/about-us", "/company", "/products", "/services", "/solutions", "/customers", "/case-studies", "/news", "/press", "/careers", "/jobs", "/contact"} {
				addURL(root + path)
			}
		}
	}
	items = append(items, collectRAGResearchSources(req, settings)...)
	items = append(items, collectPublicKnowledgeSources(req, settings)...)
	if len(items) == 0 {
		items = append(items, map[string]interface{}{"title": "用户输入", "url": "", "fetched_at": time.Now().Format(time.RFC3339), "summary": strings.TrimSpace(req.CompanyName + " " + req.IndustryHint + " " + req.Note), "status": "input", "relevance_score": 0.2})
	}
	return items
}

func collectRAGResearchSources(req companyResearchCreateRequest, settings aiResearchSettings) []map[string]interface{} {
	apiURL := strings.TrimSpace(settings.RAGAPIURL)
	apiKey := strings.TrimSpace(settings.RAGAPIKey)
	if apiURL == "" || apiKey == "" || strings.Contains(apiKey, "your-rag-api-key") {
		return []map[string]interface{}{}
	}
	queryParts := []string{req.CompanyName, req.Website, "公司 业务 产品 客户 新闻"}
	if parsed, err := url.Parse(req.Website); err == nil && parsed.Hostname() != "" {
		queryParts = append(queryParts, parsed.Hostname())
	}
	payload := map[string]interface{}{
		"query": strings.Join(compactStrings(queryParts), " "),
		"top_k": settings.MaxURLs,
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return []map[string]interface{}{}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := (&http.Client{Timeout: settings.RequestTimeout}).Do(httpReq)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return []map[string]interface{}{}
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return []map[string]interface{}{}
	}
	parsed := map[string]interface{}{}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return []map[string]interface{}{}
	}
	rawItems := firstNonNil(parsed["items"], parsed["results"], parsed["data"], parsed["documents"])
	items := []map[string]interface{}{}
	for _, raw := range interfaceList(rawItems) {
		item := mapValue(raw)
		title := firstNonEmpty(strings.TrimSpace(fmt.Sprint(item["title"])), strings.TrimSpace(fmt.Sprint(item["name"])))
		rawURL := firstNonEmpty(strings.TrimSpace(fmt.Sprint(item["url"])), strings.TrimSpace(fmt.Sprint(item["link"])))
		summary := firstNonEmpty(strings.TrimSpace(fmt.Sprint(item["summary"])), strings.TrimSpace(fmt.Sprint(item["snippet"])), strings.TrimSpace(fmt.Sprint(item["content"])), strings.TrimSpace(fmt.Sprint(item["text"])))
		if summary == "" || summary == "<nil>" {
			continue
		}
		score := researchSourceRelevance(req, title, summary, rawURL)
		if score < 0.35 {
			score = 0.45
		}
		items = append(items, map[string]interface{}{
			"title":           titleOrURL(title, rawURL),
			"url":             rawURL,
			"fetched_at":      time.Now().Format(time.RFC3339),
			"summary":         trimRunes(normalizeResearchText(summary), 1800),
			"status":          "ok",
			"relevance_score": score,
			"source_type":     "rag",
		})
		if len(items) >= settings.MaxURLs {
			break
		}
	}
	return items
}

func collectPublicKnowledgeSources(req companyResearchCreateRequest, settings aiResearchSettings) []map[string]interface{} {
	name := strings.TrimSpace(req.CompanyName)
	if name == "" {
		return []map[string]interface{}{}
	}
	items := []map[string]interface{}{}
	seen := map[string]bool{}
	add := func(title string, rawURL string) {
		if len(items) >= 4 || seen[rawURL] {
			return
		}
		seen[rawURL] = true
		pageTitle, text, err := fetchResearchPage(rawURL, settings.RequestTimeout)
		if err != nil || strings.TrimSpace(text) == "" {
			return
		}
		score := researchSourceRelevance(req, firstNonEmpty(pageTitle, title), text, rawURL)
		if score < 0.35 {
			score = 0.55
		}
		items = append(items, map[string]interface{}{
			"title":           titleOrURL(firstNonEmpty(pageTitle, title), rawURL),
			"url":             rawURL,
			"fetched_at":      time.Now().Format(time.RFC3339),
			"summary":         text,
			"status":          "ok",
			"relevance_score": score,
			"source_type":     "public_knowledge",
		})
	}
	escaped := url.PathEscape(name)
	add("Wikipedia: "+name, "https://en.wikipedia.org/wiki/"+escaped)
	add("Wikidata: "+name, "https://www.wikidata.org/wiki/Special:Search?search="+url.QueryEscape(name))
	add("百度百科: "+name, "https://baike.baidu.com/item/"+escaped)
	for _, alias := range publicKnowledgeAliases(name) {
		escapedAlias := url.PathEscape(alias)
		add("Wikipedia: "+alias, "https://en.wikipedia.org/wiki/"+escapedAlias)
		add("百度百科: "+alias, "https://baike.baidu.com/item/"+escapedAlias)
	}
	return items
}

func fetchResearchPage(rawURL string, timeout time.Duration) (string, string, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "UtopiaCRM-CompanyResearch/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("页面返回状态 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", "", err
	}
	html := string(body)
	title := extractHTMLTitle(html)
	text := normalizeResearchText(stripHTML(html))
	if len(text) > 2500 {
		text = text[:2500]
	}
	return title, text, nil
}

func usableResearchSources(sources []map[string]interface{}) []map[string]interface{} {
	items := []map[string]interface{}{}
	for _, source := range sources {
		if strings.TrimSpace(fmt.Sprint(source["status"])) != "ok" {
			continue
		}
		if numericValue(source["relevance_score"]) < 0.35 {
			continue
		}
		summary := strings.TrimSpace(fmt.Sprint(source["summary"]))
		if len([]rune(summary)) < 80 || isNoisyResearchText(summary) {
			continue
		}
		clean := map[string]interface{}{}
		for key, value := range source {
			clean[key] = value
		}
		if len([]rune(summary)) > 1800 {
			summary = string([]rune(summary)[:1800])
		}
		clean["summary"] = summary
		items = append(items, clean)
	}
	return items
}

func researchSourceRelevance(req companyResearchCreateRequest, title string, text string, rawURL string) float64 {
	lower := strings.ToLower(strings.TrimSpace(title + " " + text + " " + rawURL))
	score := 0.2
	host := ""
	if parsed, err := url.Parse(rawURL); err == nil {
		host = strings.TrimPrefix(parsed.Hostname(), "www.")
	}
	if req.Website != "" {
		if parsed, err := url.Parse(req.Website); err == nil && strings.TrimPrefix(parsed.Hostname(), "www.") == host {
			score += 0.25
		}
	}
	name := strings.ToLower(strings.TrimSpace(req.CompanyName))
	if name != "" && strings.Contains(lower, name) {
		score += 0.25
	}
	for _, keyword := range []string{"about", "company", "product", "service", "solution", "customer", "case", "news", "career", "contact", "关于", "公司", "产品", "服务", "解决方案", "客户", "案例", "新闻", "招聘", "联系"} {
		if strings.Contains(lower, keyword) {
			score += 0.08
		}
	}
	for _, keyword := range []string{"sign in", "login", "gmail", "images", "advanced search", "advertising", "merchant center", "help", "silent looping video", "cookie", "privacy terms", "登录", "图片", "帮助中心", "广告"} {
		if strings.Contains(lower, keyword) {
			score -= 0.12
		}
	}
	if isNoisyResearchText(lower) {
		score -= 0.25
	}
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return math.Round(score*100) / 100
}

func isNoisyResearchText(text string) bool {
	lower := strings.ToLower(text)
	noiseHits := 0
	for _, keyword := range []string{"gmail", "images", "sign in", "advanced search", "advertising", "merchant center", "silent looping video", "pause silent looping", "cookie", "privacy terms"} {
		if strings.Contains(lower, keyword) {
			noiseHits++
		}
	}
	return noiseHits >= 3
}

type deepSeekChatRequest struct {
	Model          string                `json:"model"`
	Messages       []deepSeekChatMessage `json:"messages"`
	Temperature    float64               `json:"temperature"`
	ResponseFormat map[string]string     `json:"response_format,omitempty"`
}

type deepSeekChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepSeekChatResponse struct {
	Choices []struct {
		Message deepSeekChatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens          int64 `json:"prompt_tokens"`
		CompletionTokens      int64 `json:"completion_tokens"`
		TotalTokens           int64 `json:"total_tokens"`
		PromptCacheHitTokens  int64 `json:"prompt_cache_hit_tokens"`
		PromptCacheMissTokens int64 `json:"prompt_cache_miss_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func callAICompanyResearch(req companyResearchCreateRequest, sources []map[string]interface{}, settings aiResearchSettings) (map[string]interface{}, int64, int64, error) {
	systemPrompt := `你是资深B2B公司研究员，不是网页摘要工具。你要做的是把公开资料和模型通用知识转化为CRM可用的结构化公司情报和可回填表单字段。
工作要求：
1. 先判断公司主体是否匹配，区分可验证事实、合理推断、待核实事项。
2. 输出固定JSON结构，顶层只能包含 company_profile、form_fill、insights、sources_used、confidence；不允许Markdown。
3. company_profile 是主要结果，必须是结构化对象；form_fill 是唯一用于自动回填的字段集合。
4. 每个 form_fill 字段必须包含 value、confidence、evidence、source_urls、basis、apply_default，其中 basis 只能是 source、model_knowledge、inference、user_input。
5. 有来源支撑的事实优先；知名公司可以用模型通用知识补全，但 evidence 必须写“模型通用知识推断，需人工核实”，confidence 不得超过0.72。
6. 无法确认就返回空数组、空字符串或“需人工核实”，不要编造具体数字。`
	userPayload := map[string]interface{}{
		"company_name": req.CompanyName,
		"website":      req.Website,
		"sources":      sources,
		"schema": map[string]interface{}{
			"company_profile": map[string]interface{}{
				"identity":       "{name, legal_name, website, headquarters, founded_year, employee_count_range, ownership, public_status}",
				"classification": "{industry, sub_industry, customer_type, grade, tags}",
				"business":       "{description, products_services[], business_model, revenue_model[], markets[]}",
				"customers":      "{target_customers[], typical_buyers[], decision_makers[]}",
				"go_to_market":   "{channels[], partnerships[], positioning}",
				"signals":        "{buying_signals[], growth_signals[], recent_updates[]}",
				"risks":          "{risk_signals[], unknowns[]}",
				"sales_strategy": "{fit_score, potential_needs[], recommended_next_actions[], opening_talk_tracks[]}",
				"verification":   "{source_quality, confidence, quality_flags[]}",
				"field_evidence": "object: field -> {evidence, source_urls[], basis, confidence}",
			},
			"form_fill": "object: CRM字段 -> {value, confidence, evidence, source_urls, basis, apply_default}",
			"insights":  "从company_profile提炼的销售视图，字段固定为 summary, target_customers[], buying_signals[], risk_signals[], recommended_next_actions[], fit_score, source_quality, quality_flags[]",
		},
		"instructions": []string{
			"company_profile 必须比 form_fill 更完整，体现业务结构、客户结构、销售切入点、风险和待核实项。",
			"summary 不超过160字，note 要是销售可执行摘要，不要复制网页正文。",
			"products_services、target_customers、markets、buying_signals、risk_signals、recommended_next_actions 必须是数组。",
			"form_fill 只包含：name, website, industry, size, customer_type, grade, description, note, headquarters, founded_year, employee_count_range, products_services, target_customers, business_model, markets, buying_signals, risk_signals, tech_stack, tags。",
			"apply_default 只有在字段高置信、有证据、适合自动回填时才为 true。",
		},
	}
	userPrompt := "请根据以下资料生成公司调研JSON，不要输出Markdown：\n" + toJSONString(userPayload)
	model := effectiveDeepSeekModel(settings.Model)
	body := deepSeekChatRequest{
		Model:       model,
		Temperature: 0.2,
		Messages: []deepSeekChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	if !strings.Contains(model, "reasoner") {
		body.ResponseFormat = map[string]string{"type": "json_object"}
		bodyBytes, _ = json.Marshal(body)
	}
	endpoint := aiChatCompletionsEndpoint(settings)
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, estimateTokens(systemPrompt + userPrompt), 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+settings.APIKey)
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(httpReq)
	promptEstimate := estimateTokens(systemPrompt + userPrompt)
	if err != nil {
		return nil, promptEstimate, 0, fmt.Errorf("AI 调研调用失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, promptEstimate, 0, err
	}
	var parsed deepSeekChatResponse
	_ = json.Unmarshal(respBody, &parsed)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(parsed.Error.Message)
		if message == "" {
			message = string(respBody)
		}
		return nil, nonZeroInt64(parsed.Usage.PromptTokens, promptEstimate), parsed.Usage.CompletionTokens, fmt.Errorf("AI 调研返回错误: %s", trimRunes(message, 200))
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		retryResult, retryPrompt, retryCompletion, retryErr := callAICompanyResearchFallback(req, sources, settings)
		return retryResult, nonZeroInt64(parsed.Usage.PromptTokens, promptEstimate) + retryPrompt, parsed.Usage.CompletionTokens + retryCompletion, retryErr
	}
	result := map[string]interface{}{}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(extractJSONObject(content)), &result); err != nil {
		retryResult, retryPrompt, retryCompletion, retryErr := callAICompanyResearchFallback(req, sources, settings)
		return retryResult, nonZeroInt64(parsed.Usage.PromptTokens, promptEstimate) + retryPrompt, nonZeroInt64(parsed.Usage.CompletionTokens, estimateTokens(content)) + retryCompletion, retryErr
	}
	return result, nonZeroInt64(parsed.Usage.PromptTokens, promptEstimate), nonZeroInt64(parsed.Usage.CompletionTokens, estimateTokens(content)), nil
}

func callAICompanyResearchFallback(req companyResearchCreateRequest, sources []map[string]interface{}, settings aiResearchSettings) (map[string]interface{}, int64, int64, error) {
	model := effectiveDeepSeekModel(settings.Model)
	systemPrompt := `你是资深B2B公司研究员。请重新输出严格JSON，顶层只能包含 company_profile、form_fill、insights、sources_used、confidence。company_profile 必须是结构化公司情报，form_fill 必须严格对应CRM公司表单字段。即使网页来源不足，只要公司知名，也要基于模型通用知识给出有价值但标注需核实的资料。不要输出Markdown。`
	userPayload := map[string]interface{}{
		"company_name": req.CompanyName,
		"website":      req.Website,
		"sources":      sources,
		"output_schema": map[string]interface{}{
			"company_profile": "{identity, classification, business, customers, go_to_market, signals, risks, sales_strategy, verification, field_evidence}",
			"form_fill":       "CRM field -> {value, confidence, evidence, source_urls, basis, apply_default}",
			"insights":        "{summary, target_customers, buying_signals, risk_signals, recommended_next_actions, fit_score, quality_flags, source_quality}",
		},
	}
	userPrompt := "请深度思考并返回公司调研JSON：" + toJSONString(userPayload)
	body := deepSeekChatRequest{
		Model:       model,
		Temperature: 0.1,
		Messages: []deepSeekChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	endpoint := aiChatCompletionsEndpoint(settings)
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	promptEstimate := estimateTokens(systemPrompt + userPrompt)
	if err != nil {
		return nil, promptEstimate, 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+settings.APIKey)
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(httpReq)
	if err != nil {
		return nil, promptEstimate, 0, fmt.Errorf("AI 深度思考调用失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, promptEstimate, 0, err
	}
	var parsed deepSeekChatResponse
	_ = json.Unmarshal(respBody, &parsed)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(parsed.Error.Message)
		if message == "" {
			message = string(respBody)
		}
		return nil, nonZeroInt64(parsed.Usage.PromptTokens, promptEstimate), parsed.Usage.CompletionTokens, fmt.Errorf("AI 深度思考返回错误: %s", trimRunes(message, 200))
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return nil, nonZeroInt64(parsed.Usage.PromptTokens, promptEstimate), parsed.Usage.CompletionTokens, fmt.Errorf("AI 深度思考未返回有效内容")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	result := map[string]interface{}{}
	if err := json.Unmarshal([]byte(extractJSONObject(content)), &result); err != nil {
		return nil, nonZeroInt64(parsed.Usage.PromptTokens, promptEstimate), nonZeroInt64(parsed.Usage.CompletionTokens, estimateTokens(content)), fmt.Errorf("AI 深度思考JSON解析失败: %w", err)
	}
	return result, nonZeroInt64(parsed.Usage.PromptTokens, promptEstimate), nonZeroInt64(parsed.Usage.CompletionTokens, estimateTokens(content)), nil
}

func normalizeAIResearchResult(req companyResearchCreateRequest, result map[string]interface{}, sources []map[string]interface{}) map[string]interface{} {
	sourceURLSet := map[string]bool{}
	for _, source := range sources {
		raw := strings.TrimSpace(fmt.Sprint(source["url"]))
		if raw != "" && raw != "<nil>" {
			sourceURLSet[raw] = true
		}
	}
	profile := normalizeCompanyResearchProfile(req, result, sources)
	formFill := normalizeCompanyResearchFormFill(req, result, profile, sources, sourceURLSet)
	suggestions := mapValue(result["suggestions"])
	if len(suggestions) == 0 {
		suggestions = formFill
	}
	cleanSuggestions := map[string]interface{}{}
	allowed := allowedResearchSuggestionFields()
	for key, raw := range suggestions {
		if !allowed[key] {
			continue
		}
		item := mapValue(raw)
		value := normalizeResearchFieldValue(key, item["value"])
		sourceURLs := stringSlice(firstNonNil(item["source_urls"], item["sources"]))
		evidence := strings.TrimSpace(fmt.Sprint(item["evidence"]))
		basis := strings.TrimSpace(fmt.Sprint(item["basis"]))
		confidence := numericValue(item["confidence"])
		if isEmptyValue(value) || evidence == "" || confidence < 0.6 {
			continue
		}
		filteredURLs := []string{}
		for _, sourceURL := range sourceURLs {
			if sourceURLSet[sourceURL] {
				filteredURLs = append(filteredURLs, sourceURL)
			}
		}
		if len(filteredURLs) == 0 && confidence > 0.72 {
			confidence = 0.72
		}
		if len(filteredURLs) == 0 && !strings.Contains(evidence, "通用知识") && !strings.Contains(evidence, "需人工核实") {
			continue
		}
		if basis == "" {
			if len(filteredURLs) > 0 {
				basis = "source"
			} else {
				basis = "model_knowledge"
			}
		}
		cleanSuggestions[key] = map[string]interface{}{
			"value":       value,
			"confidence":  confidence,
			"evidence":    trimRunes(evidence, 220),
			"source_urls": filteredURLs,
			"sources":     filteredURLs,
			"basis":       basis,
			"updated_at":  time.Now().Format(time.RFC3339),
		}
	}
	if req.Website != "" {
		cleanSuggestions["website"] = researchSuggestionWithEvidence(req.Website, 0.95, []string{req.Website}, "用户提供官网")
	}
	if req.CompanyName != "" {
		if _, ok := cleanSuggestions["name"]; !ok {
			cleanSuggestions["name"] = researchSuggestionWithEvidence(req.CompanyName, 0.75, sourceURLs(sources), "用户提供公司名称，来源内容需进一步核实")
		}
	}
	insights := normalizeCompanyResearchInsights(result, profile)
	confidence := averageSuggestionConfidence(formFill)
	if confidence == 0 {
		confidence = averageSuggestionConfidence(cleanSuggestions)
	}
	if confidence == 0 {
		confidence = 0.45
	}
	return map[string]interface{}{"company_profile": profile, "profile": profile, "form_fill": formFill, "suggestions": formFill, "legacy_suggestions": cleanSuggestions, "insights": insights, "sources_used": sourceURLs(sources), "confidence": confidence}
}

func defaultCompanyResearchProfile(req companyResearchCreateRequest, sources []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"identity": map[string]interface{}{
			"name":                 req.CompanyName,
			"legal_name":           "",
			"website":              req.Website,
			"headquarters":         "",
			"founded_year":         "",
			"employee_count_range": "",
			"ownership":            "",
			"public_status":        "",
		},
		"classification": map[string]interface{}{
			"industry":      "",
			"sub_industry":  "",
			"customer_type": "",
			"grade":         "",
			"tags":          []string{},
		},
		"business": map[string]interface{}{
			"description":       "",
			"products_services": []string{},
			"business_model":    "",
			"revenue_model":     []string{},
			"markets":           []string{},
		},
		"customers": map[string]interface{}{
			"target_customers": []string{},
			"typical_buyers":   []string{},
			"decision_makers":  []string{},
		},
		"go_to_market": map[string]interface{}{
			"channels":     []string{},
			"partnerships": []string{},
			"positioning":  "",
		},
		"signals": map[string]interface{}{
			"buying_signals": []string{},
			"growth_signals": []string{},
			"recent_updates": []string{},
		},
		"risks": map[string]interface{}{
			"risk_signals": []string{},
			"unknowns":     []string{},
		},
		"sales_strategy": map[string]interface{}{
			"fit_score":                0,
			"potential_needs":          []string{},
			"recommended_next_actions": []string{},
			"opening_talk_tracks":      []string{},
		},
		"verification": map[string]interface{}{
			"source_quality": "low",
			"confidence":     0.35,
			"quality_flags":  []string{},
			"sources_used":   sourceURLs(sources),
		},
		"field_evidence": map[string]interface{}{},
	}
}

func normalizeCompanyResearchProfile(req companyResearchCreateRequest, result map[string]interface{}, sources []map[string]interface{}) map[string]interface{} {
	profile := defaultCompanyResearchProfile(req, sources)
	rawProfile := mapValue(firstNonNil(result["company_profile"], result["profile"]))
	copyProfileSection(profile, rawProfile, "identity", []string{"name", "legal_name", "website", "headquarters", "founded_year", "employee_count_range", "ownership", "public_status"}, nil)
	copyProfileSection(profile, rawProfile, "classification", []string{"industry", "sub_industry", "customer_type", "grade"}, []string{"tags"})
	copyProfileSection(profile, rawProfile, "business", []string{"description", "business_model"}, []string{"products_services", "revenue_model", "markets"})
	copyProfileSection(profile, rawProfile, "customers", nil, []string{"target_customers", "typical_buyers", "decision_makers"})
	copyProfileSection(profile, rawProfile, "go_to_market", []string{"positioning"}, []string{"channels", "partnerships"})
	copyProfileSection(profile, rawProfile, "signals", nil, []string{"buying_signals", "growth_signals", "recent_updates"})
	copyProfileSection(profile, rawProfile, "risks", nil, []string{"risk_signals", "unknowns"})
	copyProfileSection(profile, rawProfile, "sales_strategy", nil, []string{"potential_needs", "recommended_next_actions", "opening_talk_tracks"})
	copyProfileSection(profile, rawProfile, "verification", []string{"source_quality"}, []string{"quality_flags"})

	if score := numericValue(mapValue(rawProfile["sales_strategy"])["fit_score"]); score > 0 {
		mapValue(profile["sales_strategy"])["fit_score"] = int(math.Round(score))
	}
	verification := mapValue(profile["verification"])
	if confidence := numericValue(mapValue(rawProfile["verification"])["confidence"]); confidence > 0 {
		if confidence > 1 {
			confidence = confidence / 100
		}
		verification["confidence"] = math.Round(confidence*100) / 100
	}
	verification["sources_used"] = sourceURLs(sources)

	fieldEvidence := mapValue(rawProfile["field_evidence"])
	cleanEvidence := map[string]interface{}{}
	for key, value := range fieldEvidence {
		if !allowedResearchSuggestionFields()[key] {
			continue
		}
		item := mapValue(value)
		cleanEvidence[key] = map[string]interface{}{
			"evidence":    trimRunes(strings.TrimSpace(fmt.Sprint(item["evidence"])), 220),
			"source_urls": stringSlice(item["source_urls"]),
			"basis":       strings.TrimSpace(fmt.Sprint(item["basis"])),
			"confidence":  numericValue(item["confidence"]),
		}
	}
	profile["field_evidence"] = cleanEvidence
	return profile
}

func normalizeCompanyResearchFormFill(req companyResearchCreateRequest, result map[string]interface{}, profile map[string]interface{}, sources []map[string]interface{}, sourceURLSet map[string]bool) map[string]interface{} {
	rawFormFill := mapValue(result["form_fill"])
	if len(rawFormFill) == 0 {
		rawFormFill = mapValue(result["suggestions"])
	}
	if len(rawFormFill) == 0 {
		rawFormFill = suggestionsFromResearchProfile(req, profile, sources)
	}
	formFill := map[string]interface{}{}
	allowed := allowedResearchSuggestionFields()
	for key, raw := range rawFormFill {
		if !allowed[key] {
			continue
		}
		item := mapValue(raw)
		value := normalizeResearchFieldValue(key, item["value"])
		if isEmptyValue(value) {
			continue
		}
		evidence := strings.TrimSpace(fmt.Sprint(item["evidence"]))
		if evidence == "" || evidence == "<nil>" {
			evidence = fallbackFormFillEvidence(key)
		}
		sourceURLs := stringSlice(firstNonNil(item["source_urls"], item["sources"]))
		filteredURLs := []string{}
		for _, sourceURL := range sourceURLs {
			if sourceURLSet[sourceURL] {
				filteredURLs = append(filteredURLs, sourceURL)
			}
		}
		confidence := numericValue(item["confidence"])
		if confidence > 1 {
			confidence = confidence / 100
		}
		if confidence == 0 {
			confidence = 0.68
		}
		basis := strings.TrimSpace(fmt.Sprint(item["basis"]))
		if basis == "" || basis == "<nil>" {
			if len(filteredURLs) > 0 {
				basis = "source"
			} else {
				basis = "inference"
			}
		}
		if len(filteredURLs) == 0 && basis == "source" {
			basis = "inference"
		}
		if basis == "model_knowledge" && !strings.Contains(evidence, "需人工核实") {
			evidence = strings.TrimSpace(evidence + "；模型通用知识推断，需人工核实")
		}
		if len(filteredURLs) == 0 && confidence > 0.72 {
			confidence = 0.72
		}
		applyDefault := boolValue(item["apply_default"], false)
		if !applyDefault {
			applyDefault = confidence >= 0.75 && len(filteredURLs) > 0 && evidence != ""
		}
		formFill[key] = map[string]interface{}{
			"value":         value,
			"confidence":    math.Round(confidence*100) / 100,
			"evidence":      trimRunes(evidence, 220),
			"source_urls":   filteredURLs,
			"sources":       filteredURLs,
			"basis":         basis,
			"apply_default": applyDefault,
			"updated_at":    time.Now().Format(time.RFC3339),
		}
	}
	return formFill
}

func fallbackFormFillEvidence(key string) string {
	labels := map[string]string{
		"name":                 "公司主体识别结果",
		"website":              "官网字段",
		"industry":             "行业分类判断",
		"size":                 "公司规模判断",
		"customer_type":        "客户类型判断",
		"grade":                "客户价值分级判断",
		"description":          "公司画像摘要",
		"note":                 "销售可执行摘要",
		"headquarters":         "总部/主要地区",
		"founded_year":         "成立年份",
		"employee_count_range": "员工规模区间",
		"products_services":    "主营产品/服务",
		"target_customers":     "目标客户结构",
		"business_model":       "商业模式判断",
		"markets":              "覆盖市场/区域",
		"buying_signals":       "采购/增长信号",
		"risk_signals":         "风险信号",
		"tech_stack":           "技术栈线索",
		"tags":                 "结构化标签",
	}
	return firstNonEmpty(labels[key], "AI结构化调研字段")
}

func mapFromStrings(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result[value] = true
		}
	}
	return result
}

func copyProfileSection(profile map[string]interface{}, rawProfile map[string]interface{}, section string, scalarKeys []string, listKeys []string) {
	target := mapValue(profile[section])
	raw := mapValue(rawProfile[section])
	for _, key := range scalarKeys {
		text := trimRunes(strings.TrimSpace(fmt.Sprint(raw[key])), 600)
		if text != "" && text != "<nil>" {
			target[key] = text
		}
	}
	for _, key := range listKeys {
		target[key] = trimStringSlice(stringSlice(raw[key]), 8, 120)
	}
}

func trimStringSlice(values []string, limit int, itemLimit int) []string {
	items := []string{}
	for _, value := range values {
		text := trimRunes(value, itemLimit)
		if text != "" && text != "<nil>" {
			items = append(items, text)
		}
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return uniqueStrings(items)
}

func suggestionsFromResearchProfile(req companyResearchCreateRequest, profile map[string]interface{}, sources []map[string]interface{}) map[string]interface{} {
	identity := mapValue(profile["identity"])
	classification := mapValue(profile["classification"])
	business := mapValue(profile["business"])
	customers := mapValue(profile["customers"])
	signals := mapValue(profile["signals"])
	risks := mapValue(profile["risks"])
	evidence := mapValue(profile["field_evidence"])
	sourceURLs := sourceURLs(sources)
	makeSuggestion := func(key string, value interface{}, fallbackEvidence string) map[string]interface{} {
		item := mapValue(evidence[key])
		text := strings.TrimSpace(fmt.Sprint(item["evidence"]))
		if text == "" || text == "<nil>" {
			text = fallbackEvidence
		}
		urls := stringSlice(item["source_urls"])
		if len(urls) == 0 {
			urls = sourceURLs
		}
		confidence := numericValue(item["confidence"])
		if confidence == 0 {
			confidence = 0.68
		}
		basis := strings.TrimSpace(fmt.Sprint(item["basis"]))
		if basis == "" || basis == "<nil>" {
			basis = "inference"
		}
		suggestion := researchSuggestionWithEvidence(value, confidence, urls, text)
		suggestion["basis"] = basis
		return suggestion
	}
	return map[string]interface{}{
		"name":                 makeSuggestion("name", firstNonEmpty(fmt.Sprint(identity["name"]), req.CompanyName), "公司主体识别结果"),
		"website":              makeSuggestion("website", firstNonEmpty(fmt.Sprint(identity["website"]), req.Website), "用户提供官网"),
		"industry":             makeSuggestion("industry", classification["industry"], "公司行业分类"),
		"customer_type":        makeSuggestion("customer_type", classification["customer_type"], "客户类型判断"),
		"grade":                makeSuggestion("grade", classification["grade"], "客户价值分级判断"),
		"description":          makeSuggestion("description", business["description"], "公司画像摘要"),
		"note":                 makeSuggestion("note", buildStructuredSalesNote(profile), "基于公司画像生成销售摘要"),
		"headquarters":         makeSuggestion("headquarters", identity["headquarters"], "总部/主要地区"),
		"founded_year":         makeSuggestion("founded_year", identity["founded_year"], "成立年份"),
		"employee_count_range": makeSuggestion("employee_count_range", identity["employee_count_range"], "员工规模区间"),
		"size":                 makeSuggestion("size", identity["employee_count_range"], "员工规模区间"),
		"products_services":    makeSuggestion("products_services", business["products_services"], "主营产品/服务"),
		"target_customers":     makeSuggestion("target_customers", customers["target_customers"], "目标客户结构"),
		"business_model":       makeSuggestion("business_model", business["business_model"], "商业模式判断"),
		"markets":              makeSuggestion("markets", business["markets"], "覆盖市场/区域"),
		"buying_signals":       makeSuggestion("buying_signals", signals["buying_signals"], "采购/增长信号"),
		"risk_signals":         makeSuggestion("risk_signals", risks["risk_signals"], "风险信号"),
		"tags":                 makeSuggestion("tags", classification["tags"], "结构化标签"),
	}
}

func normalizeCompanyResearchInsights(result map[string]interface{}, profile map[string]interface{}) map[string]interface{} {
	raw := mapValue(result["insights"])
	business := mapValue(profile["business"])
	customers := mapValue(profile["customers"])
	signals := mapValue(profile["signals"])
	risks := mapValue(profile["risks"])
	strategy := mapValue(profile["sales_strategy"])
	verification := mapValue(profile["verification"])
	summary := strings.TrimSpace(fmt.Sprint(raw["summary"]))
	if summary == "" || summary == "<nil>" {
		summary = trimRunes(strings.TrimSpace(fmt.Sprint(business["description"])), 180)
	}
	if summary == "" || summary == "<nil>" {
		summary = "AI 已完成结构化调研，请优先查看公司画像、销售策略和字段证据。"
	}
	nextActions := trimStringSlice(stringSlice(firstNonNil(raw["recommended_next_actions"], strategy["recommended_next_actions"])), 6, 120)
	if len(nextActions) == 0 {
		nextActions = []string{"核对高置信度字段后保存公司资料", "基于目标客户和采购信号安排下一步跟进"}
	}
	return map[string]interface{}{
		"summary":                  summary,
		"products_services":        trimStringSlice(stringSlice(firstNonNil(raw["products_services"], business["products_services"])), 8, 120),
		"target_customers":         trimStringSlice(stringSlice(firstNonNil(raw["target_customers"], customers["target_customers"])), 8, 120),
		"buying_signals":           trimStringSlice(stringSlice(firstNonNil(raw["buying_signals"], signals["buying_signals"])), 8, 140),
		"risk_signals":             trimStringSlice(stringSlice(firstNonNil(raw["risk_signals"], risks["risk_signals"])), 8, 140),
		"recommended_next_actions": nextActions,
		"fit_score":                int(math.Round(numericValue(firstNonNil(raw["fit_score"], strategy["fit_score"])))),
		"source_quality":           firstNonEmpty(strings.TrimSpace(fmt.Sprint(raw["source_quality"])), strings.TrimSpace(fmt.Sprint(verification["source_quality"])), "medium"),
		"quality_flags":            trimStringSlice(stringSlice(firstNonNil(raw["quality_flags"], verification["quality_flags"])), 8, 140),
	}
}

func allowedResearchSuggestionFields() map[string]bool {
	fields := map[string]bool{}
	for _, key := range []string{"name", "website", "industry", "size", "customer_type", "grade", "description", "note", "headquarters", "founded_year", "employee_count_range", "products_services", "target_customers", "business_model", "markets", "buying_signals", "risk_signals", "tech_stack", "tags"} {
		fields[key] = true
	}
	return fields
}

func normalizeResearchFieldValue(key string, value interface{}) interface{} {
	switch key {
	case "products_services", "target_customers", "markets", "buying_signals", "risk_signals", "tech_stack", "tags":
		return trimStringSlice(stringSlice(value), 8, 120)
	case "description", "note":
		return trimRunes(strings.TrimSpace(fmt.Sprint(value)), 900)
	default:
		return trimRunes(strings.TrimSpace(fmt.Sprint(value)), 240)
	}
}

func buildStructuredSalesNote(profile map[string]interface{}) string {
	classification := mapValue(profile["classification"])
	business := mapValue(profile["business"])
	customers := mapValue(profile["customers"])
	signals := mapValue(profile["signals"])
	risks := mapValue(profile["risks"])
	strategy := mapValue(profile["sales_strategy"])
	parts := []string{}
	if industry := strings.TrimSpace(fmt.Sprint(classification["industry"])); industry != "" && industry != "<nil>" {
		parts = append(parts, "定位："+industry)
	}
	if model := strings.TrimSpace(fmt.Sprint(business["business_model"])); model != "" && model != "<nil>" {
		parts = append(parts, "商业模式："+model)
	}
	if targets := trimStringSlice(stringSlice(customers["target_customers"]), 3, 80); len(targets) > 0 {
		parts = append(parts, "目标客户："+strings.Join(targets, "、"))
	}
	if values := trimStringSlice(stringSlice(signals["buying_signals"]), 3, 90); len(values) > 0 {
		parts = append(parts, "关注信号："+strings.Join(values, "；"))
	}
	if values := trimStringSlice(stringSlice(risks["risk_signals"]), 2, 90); len(values) > 0 {
		parts = append(parts, "需核实："+strings.Join(values, "；"))
	}
	if values := trimStringSlice(stringSlice(strategy["recommended_next_actions"]), 2, 90); len(values) > 0 {
		parts = append(parts, "下一步："+strings.Join(values, "；"))
	}
	return strings.Join(parts, "\n")
}

func lowQualityResearchResult(req companyResearchCreateRequest, sources []map[string]interface{}, reason string) map[string]interface{} {
	suggestions := map[string]interface{}{}
	if req.Website != "" {
		suggestions["website"] = researchSuggestionWithEvidence(req.Website, 0.95, []string{req.Website}, "用户提供官网")
	}
	if req.CompanyName != "" {
		suggestions["name"] = researchSuggestionWithEvidence(req.CompanyName, 0.7, sourceURLs(sources), "用户提供公司名称")
	}
	profile := defaultCompanyResearchProfile(req, sources)
	mapValue(profile["verification"])["quality_flags"] = []string{reason}
	mapValue(profile["verification"])["source_quality"] = "low"
	formFill := suggestionsFromResearchProfile(req, profile, sources)
	return map[string]interface{}{
		"company_profile":    profile,
		"profile":            profile,
		"form_fill":          formFill,
		"suggestions":        formFill,
		"legacy_suggestions": suggestions,
		"insights": map[string]interface{}{
			"summary":                  reason,
			"recommended_next_actions": []string{"补充正确官网或可验证资料后重新调研", "由销售人工确认行业、规模、主营产品和决策链"},
			"quality_flags":            []string{reason},
			"source_quality":           "low",
		},
		"sources_used": sourceURLs(sources),
		"confidence":   0.35,
	}
}

func buildPublicKnowledgeResearchResult(req companyResearchCreateRequest, sources []map[string]interface{}, fallbackReason string) map[string]interface{} {
	sourceURLs := sourceURLs(sources)
	sourceText := sourceSummariesText(sources)
	companyName := firstNonEmpty(req.CompanyName, inferCompanyName(req.Website, sources))
	known := isKnownCompanyName(companyName)
	industry := inferIndustry(sourceText)
	if industry == "待确认行业" && known {
		industry = knownCompanyIndustry(companyName)
	}
	products := inferListByKeywords(sourceText, []string{"产品", "服务", "平台", "搜索", "广告", "云", "AI", "地图", "百科", "product", "service", "cloud", "search", "advertising"}, 5)
	if len(products) == 0 && known {
		products = knownCompanyProducts(companyName)
	}
	targetCustomers := inferTargetCustomers(industry)
	if known {
		targetCustomers = []string{"个人互联网用户", "广告主/品牌客户", "企业与开发者客户"}
	}
	buyingSignals := knownCompanyBuyingSignals(companyName)
	if len(buyingSignals) == 0 {
		buyingSignals = []string{"知名公司公开资料较多，可围绕AI、云服务、广告增长、企业合作场景切入"}
	}
	riskSignals := knownCompanyRiskSignals(companyName)
	if len(riskSignals) == 0 {
		riskSignals = []string{"需人工核实当前组织架构、预算和采购窗口"}
	}
	businessModel := inferBusinessModel(sourceText)
	if known {
		businessModel = knownCompanyBusinessModel(companyName)
	}
	markets := inferMarkets(req.Region, sourceText)
	if known {
		markets = knownCompanyMarkets(companyName)
	}
	note := buildSalesNote(companyName, industry, buyingSignals, riskSignals)
	description := buildCompanyDescription(companyName, industry, products)
	suggestions := map[string]interface{}{
		"name":                 researchSuggestionWithEvidence(companyName, 0.78, sourceURLs, "公开知识源和用户输入共同确认公司主体"),
		"website":              researchSuggestionWithEvidence(req.Website, 0.95, []string{req.Website}, "用户提供官网"),
		"industry":             researchSuggestionWithEvidence(industry, 0.72, sourceURLs, "公开知识源与模型通用知识推断，需人工核实"),
		"description":          researchSuggestionWithEvidence(description, 0.7, sourceURLs, "基于公开知识源摘要生成公司画像"),
		"note":                 researchSuggestionWithEvidence(note, 0.7, sourceURLs, "基于公开知识源生成销售摘要"),
		"customer_type":        researchSuggestionWithEvidence("target", 0.68, sourceURLs, "模型通用知识推断，需人工核实"),
		"grade":                researchSuggestionWithEvidence("important", 0.66, sourceURLs, "知名公司通常具备较高客户价值，需结合实际业务核实"),
		"headquarters":         researchSuggestionWithEvidence(knownCompanyHeadquarters(companyName), 0.66, sourceURLs, "模型通用知识推断，需人工核实"),
		"size":                 researchSuggestionWithEvidence(knownCompanySize(companyName), 0.64, sourceURLs, "模型通用知识推断，需人工核实"),
		"employee_count_range": researchSuggestionWithEvidence(knownCompanySize(companyName), 0.64, sourceURLs, "模型通用知识推断，需人工核实"),
		"products_services":    researchSuggestionWithEvidence(products, 0.7, sourceURLs, "公开知识源与模型通用知识推断，需人工核实"),
		"target_customers":     researchSuggestionWithEvidence(targetCustomers, 0.68, sourceURLs, "基于业务类型推断，需人工核实"),
		"business_model":       researchSuggestionWithEvidence(businessModel, 0.66, sourceURLs, "基于公开资料关键词与模型通用知识推断，需人工核实"),
		"markets":              researchSuggestionWithEvidence(markets, 0.64, sourceURLs, "模型通用知识推断，需人工核实"),
		"buying_signals":       researchSuggestionWithEvidence(buyingSignals, 0.64, sourceURLs, "模型通用知识推断，需人工核实"),
		"risk_signals":         researchSuggestionWithEvidence(riskSignals, 0.62, sourceURLs, "模型通用知识推断，需人工核实"),
		"tags":                 researchSuggestionWithEvidence(compactStrings([]string{industry, "AI调研", "公开知识源"}), 0.7, sourceURLs, "基于公开知识源生成标签"),
	}
	profile := map[string]interface{}{
		"identity": map[string]interface{}{
			"name":                 companyName,
			"legal_name":           "",
			"website":              req.Website,
			"headquarters":         knownCompanyHeadquarters(companyName),
			"founded_year":         "",
			"employee_count_range": knownCompanySize(companyName),
			"ownership":            "",
			"public_status":        "需人工核实",
		},
		"classification": map[string]interface{}{
			"industry":      industry,
			"sub_industry":  "",
			"customer_type": "target",
			"grade":         "important",
			"tags":          compactStrings([]string{industry, "AI调研", "公开知识源"}),
		},
		"business": map[string]interface{}{
			"description":       description,
			"products_services": products,
			"business_model":    businessModel,
			"revenue_model":     []string{},
			"markets":           markets,
		},
		"customers": map[string]interface{}{
			"target_customers": targetCustomers,
			"typical_buyers":   []string{},
			"decision_makers":  []string{},
		},
		"go_to_market": map[string]interface{}{
			"channels":     []string{},
			"partnerships": []string{},
			"positioning":  "",
		},
		"signals": map[string]interface{}{
			"buying_signals": buyingSignals,
			"growth_signals": []string{},
			"recent_updates": []string{},
		},
		"risks": map[string]interface{}{
			"risk_signals": riskSignals,
			"unknowns":     []string{"关键数据需以官网、财报或客户访谈核实"},
		},
		"sales_strategy": map[string]interface{}{
			"fit_score":                70,
			"potential_needs":          targetCustomers,
			"recommended_next_actions": []string{"核对公司官网与公开知识源", "确认实际采购部门、预算窗口和合作切入点"},
			"opening_talk_tracks":      []string{},
		},
		"verification": map[string]interface{}{
			"source_quality": "medium",
			"confidence":     0.7,
			"quality_flags":  []string{"AI结构化输出失败: " + trimRunes(fallbackReason, 100), "已使用公开知识源降级生成"},
		},
		"field_evidence": map[string]interface{}{},
	}
	formFill := normalizeCompanyResearchFormFill(req, map[string]interface{}{"form_fill": suggestions}, profile, sources, mapFromStrings(sourceURLs))
	return map[string]interface{}{
		"company_profile":    profile,
		"profile":            profile,
		"form_fill":          formFill,
		"suggestions":        formFill,
		"legacy_suggestions": suggestions,
		"insights": map[string]interface{}{
			"summary":                  description,
			"products_services":        products,
			"target_customers":         targetCustomers,
			"buying_signals":           buyingSignals,
			"risk_signals":             riskSignals,
			"recommended_next_actions": []string{"核对公司官网与公开知识源", "确认实际采购部门、预算窗口和合作切入点"},
			"fit_score":                70,
			"quality_flags":            []string{"AI结构化输出失败: " + trimRunes(fallbackReason, 100), "已使用公开知识源降级生成"},
			"source_quality":           "medium",
		},
		"sources_used": sourceURLs,
		"confidence":   0.7,
	}
}

func isKnownCompanyName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, keyword := range []string{"baidu", "百度", "google", "谷歌", "microsoft", "微软", "apple", "苹果", "amazon", "亚马逊", "alibaba", "阿里巴巴", "tencent", "腾讯"} {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func publicKnowledgeAliases(name string) []string {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(lower, "baidu") || strings.Contains(name, "百度"):
		return []string{"Baidu", "百度", "百度公司"}
	case strings.Contains(lower, "google") || strings.Contains(name, "谷歌"):
		return []string{"Google", "谷歌", "Alphabet Inc."}
	case strings.Contains(lower, "microsoft") || strings.Contains(name, "微软"):
		return []string{"Microsoft", "微软"}
	case strings.Contains(lower, "apple") || strings.Contains(name, "苹果"):
		return []string{"Apple Inc.", "苹果公司"}
	case strings.Contains(lower, "amazon") || strings.Contains(name, "亚马逊"):
		return []string{"Amazon", "Amazon.com", "亚马逊公司"}
	case strings.Contains(lower, "alibaba") || strings.Contains(name, "阿里巴巴"):
		return []string{"Alibaba Group", "阿里巴巴集团"}
	case strings.Contains(lower, "tencent") || strings.Contains(name, "腾讯"):
		return []string{"Tencent", "腾讯"}
	default:
		return []string{}
	}
}

func knownCompanyIndustry(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "baidu") || strings.Contains(name, "百度"):
		return "互联网/搜索引擎/人工智能"
	case strings.Contains(lower, "google") || strings.Contains(name, "谷歌"):
		return "互联网/搜索引擎/广告/云计算"
	case strings.Contains(lower, "microsoft") || strings.Contains(name, "微软"):
		return "企业软件/云计算/人工智能"
	case strings.Contains(lower, "apple") || strings.Contains(name, "苹果"):
		return "消费电子/互联网服务"
	default:
		return "互联网/科技"
	}
}

func knownCompanyProducts(name string) []string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "baidu") || strings.Contains(name, "百度"):
		return []string{"搜索引擎", "在线广告", "百度智能云", "AI大模型与智能驾驶相关业务", "地图与内容生态"}
	case strings.Contains(lower, "google") || strings.Contains(name, "谷歌"):
		return []string{"搜索引擎", "广告平台", "Google Cloud", "Android", "Workspace与AI产品"}
	case strings.Contains(lower, "microsoft") || strings.Contains(name, "微软"):
		return []string{"Microsoft 365", "Azure", "Windows", "Dynamics", "AI与开发者工具"}
	default:
		return []string{"核心产品/服务需进一步核实"}
	}
}

func knownCompanyHeadquarters(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "baidu") || strings.Contains(name, "百度"):
		return "中国北京"
	case strings.Contains(lower, "google") || strings.Contains(name, "谷歌"):
		return "美国加利福尼亚州"
	case strings.Contains(lower, "microsoft") || strings.Contains(name, "微软"):
		return "美国华盛顿州雷德蒙德"
	case strings.Contains(lower, "apple") || strings.Contains(name, "苹果"):
		return "美国加利福尼亚州库比蒂诺"
	case strings.Contains(lower, "amazon") || strings.Contains(name, "亚马逊"):
		return "美国华盛顿州西雅图"
	case strings.Contains(lower, "alibaba") || strings.Contains(name, "阿里巴巴"):
		return "中国杭州"
	case strings.Contains(lower, "tencent") || strings.Contains(name, "腾讯"):
		return "中国深圳"
	default:
		return "总部/地区需人工核实"
	}
}

func knownCompanySize(name string) string {
	if isKnownCompanyName(name) {
		return "10000人以上"
	}
	return "规模需人工核实"
}

func knownCompanyBusinessModel(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "baidu") || strings.Contains(name, "百度"):
		return "广告变现 + 云服务 + AI产品/平台服务"
	case strings.Contains(lower, "google") || strings.Contains(name, "谷歌"):
		return "广告变现 + 云服务 + 软硬件与订阅服务"
	case strings.Contains(lower, "microsoft") || strings.Contains(name, "微软"):
		return "企业软件订阅 + 云服务 + 生态授权"
	case strings.Contains(lower, "apple") || strings.Contains(name, "苹果"):
		return "硬件销售 + 服务订阅 + 应用生态分成"
	case strings.Contains(lower, "amazon") || strings.Contains(name, "亚马逊"):
		return "电商平台 + 云服务 + 会员订阅与广告"
	case strings.Contains(lower, "alibaba") || strings.Contains(name, "阿里巴巴"):
		return "电商平台 + 云服务 + 本地生活/国际化业务"
	case strings.Contains(lower, "tencent") || strings.Contains(name, "腾讯"):
		return "社交生态 + 游戏娱乐 + 金融科技与云服务"
	default:
		return "商业模式需人工核实"
	}
}

func knownCompanyMarkets(name string) []string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "baidu") || strings.Contains(name, "百度"):
		return []string{"中国", "亚太", "全球开发者/企业客户"}
	case strings.Contains(lower, "alibaba") || strings.Contains(name, "阿里巴巴"):
		return []string{"中国", "亚太", "全球跨境贸易市场"}
	case strings.Contains(lower, "tencent") || strings.Contains(name, "腾讯"):
		return []string{"中国", "亚太", "全球数字娱乐市场"}
	default:
		if isKnownCompanyName(name) {
			return []string{"全球"}
		}
		return []string{"市场区域需人工核实"}
	}
}

func knownCompanyBuyingSignals(name string) []string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "baidu") || strings.Contains(name, "百度"):
		return []string{"AI大模型、智能云、智能驾驶和企业AI应用是可关注业务方向", "适合围绕行业AI落地、数据治理、生态合作寻找切入点"}
	case strings.Contains(lower, "google") || strings.Contains(name, "谷歌"):
		return []string{"AI、云计算、广告技术和开发者生态持续投入", "适合围绕企业云、AI应用和生态合作寻找切入点"}
	case strings.Contains(lower, "microsoft") || strings.Contains(name, "微软"):
		return []string{"企业AI、云计算、办公协同和开发者工具持续投入", "适合围绕企业数字化、AI Copilot生态和行业解决方案切入"}
	default:
		if isKnownCompanyName(name) {
			return []string{"公司业务线多，建议围绕AI、云服务、广告增长或生态合作场景切入"}
		}
		return []string{}
	}
}

func knownCompanyRiskSignals(name string) []string {
	if isKnownCompanyName(name) {
		return []string{"组织层级复杂，需要确认真实采购部门、预算归属和决策链", "公开资料可能滞后，关键数据需以官网、财报或客户访谈核实"}
	}
	return []string{}
}

func buildCompanyResearchResult(req companyResearchCreateRequest, sources []map[string]interface{}) map[string]interface{} {
	combined := strings.TrimSpace(req.CompanyName + " " + req.IndustryHint + " " + req.Note + " " + sourceSummariesText(sources))
	companyName := strings.TrimSpace(req.CompanyName)
	if companyName == "" {
		companyName = inferCompanyName(req.Website, sources)
	}
	industry := firstNonEmpty(req.IndustryHint, inferIndustry(combined))
	products := inferListByKeywords(combined, []string{"产品", "服务", "平台", "解决方案", "product", "service", "platform", "solution"}, 4)
	if len(products) == 0 && industry != "" {
		products = []string{industry + "相关产品/服务"}
	}
	buyingSignals := inferListByKeywords(combined, []string{"招聘", "扩张", "融资", "增长", "发布", "采购", "招标", "hiring", "growth", "funding", "launch"}, 4)
	riskSignals := inferListByKeywords(combined, []string{"风险", "下滑", "裁员", "诉讼", "合规", "安全", "risk", "layoff", "lawsuit", "compliance"}, 4)
	if len(riskSignals) == 0 {
		riskSignals = []string{"公开资料未发现明确风险信号，需销售进一步核实"}
	}
	size := inferCompanySize(combined)
	confidence := 0.62
	if len(sources) >= 2 {
		confidence = 0.78
	}
	if req.Website != "" && companyName != "" {
		confidence += 0.08
	}
	if confidence > 0.92 {
		confidence = 0.92
	}
	sourceURLs := sourceURLs(sources)
	suggestions := map[string]interface{}{
		"name":                 researchSuggestion(companyName, confidence, sourceURLs),
		"website":              researchSuggestion(req.Website, 0.95, sourceURLs),
		"industry":             researchSuggestion(industry, confidence, sourceURLs),
		"size":                 researchSuggestion(size, 0.68, sourceURLs),
		"customer_type":        researchSuggestion("target", 0.72, sourceURLs),
		"grade":                researchSuggestion(inferGrade(confidence, buyingSignals), 0.7, sourceURLs),
		"description":          researchSuggestion(buildCompanyDescription(companyName, industry, products), confidence, sourceURLs),
		"note":                 researchSuggestion(buildSalesNote(companyName, industry, buyingSignals, riskSignals), confidence, sourceURLs),
		"headquarters":         researchSuggestion(inferRegion(req.Region, combined), 0.58, sourceURLs),
		"employee_count_range": researchSuggestion(size, 0.68, sourceURLs),
		"products_services":    researchSuggestion(products, confidence, sourceURLs),
		"target_customers":     researchSuggestion(inferTargetCustomers(industry), 0.62, sourceURLs),
		"business_model":       researchSuggestion(inferBusinessModel(combined), 0.62, sourceURLs),
		"markets":              researchSuggestion(inferMarkets(req.Region, combined), 0.56, sourceURLs),
		"buying_signals":       researchSuggestion(buyingSignals, 0.66, sourceURLs),
		"risk_signals":         researchSuggestion(riskSignals, 0.58, sourceURLs),
		"tech_stack":           researchSuggestion(inferTechStack(combined), 0.55, sourceURLs),
		"tags":                 researchSuggestion(compactStrings([]string{industry, inferRegion(req.Region, combined), "AI调研"}), confidence, sourceURLs),
	}
	insights := map[string]interface{}{
		"summary":                  buildCompanyDescription(companyName, industry, products),
		"products_services":        products,
		"target_customers":         inferTargetCustomers(industry),
		"buying_signals":           buyingSignals,
		"risk_signals":             riskSignals,
		"recommended_next_actions": recommendedNextActions(industry, buyingSignals),
		"fit_score":                int(math.Round(confidence * 100)),
		"recent_updates":           inferListByKeywords(combined, []string{"新闻", "发布", "公告", "news", "press", "release"}, 3),
	}
	return map[string]interface{}{"suggestions": suggestions, "insights": insights, "confidence": confidence}
}

func researchSuggestion(value interface{}, confidence float64, sources []string) map[string]interface{} {
	return map[string]interface{}{"value": value, "confidence": confidence, "sources": sources, "updated_at": time.Now().Format(time.RFC3339)}
}

func researchSuggestionWithEvidence(value interface{}, confidence float64, sources []string, evidence string) map[string]interface{} {
	return map[string]interface{}{"value": value, "confidence": confidence, "sources": sources, "source_urls": sources, "evidence": evidence, "updated_at": time.Now().Format(time.RFC3339)}
}

func updateCompanyResearchSnapshot(auth authContext, companyID uint, jobID uint, usageLogID uint, result map[string]interface{}, sources []map[string]interface{}) {
	var company models.CRMObject
	if err := utils.DB.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "contacts", "company", companyID).First(&company).Error; err != nil {
		return
	}
	payload := payloadMap(company.PayloadJSON)
	payload["ai_research"] = map[string]interface{}{
		"job_id":          jobID,
		"usage_log_id":    usageLogID,
		"confidence":      result["confidence"],
		"status":          "succeeded",
		"updated_at":      time.Now().Format(time.RFC3339),
		"sources":         sources,
		"company_profile": mapValue(firstNonNil(result["company_profile"], result["profile"])),
		"form_fill":       mapValue(firstNonNil(result["form_fill"], result["suggestions"])),
		"insights":        mapValue(result["insights"]),
	}
	_ = utils.DB.Model(&company).Updates(map[string]interface{}{"payload_json": toJSONString(payload), "updated_by": auth.UserID}).Error
}

func companyResearchJobResponse(job models.AIResearchJob) map[string]interface{} {
	result := mapValueFromJSON(job.ResultJSON)
	sources := []interface{}{}
	_ = json.Unmarshal([]byte(job.SourceJSON), &sources)
	var usage models.AIUsageLog
	usagePayload := map[string]interface{}{}
	if job.UsageLogID > 0 {
		if err := utils.DB.Where("id = ? AND tenant_id = ?", job.UsageLogID, job.TenantID).First(&usage).Error; err == nil {
			usagePayload = aiUsageResponse(usage)
		}
	}
	return map[string]interface{}{
		"id":        job.ID,
		"job_id":    fmt.Sprintf("research_%d", job.ID),
		"status":    job.Status,
		"object_id": job.ObjectID,
		"input":     mapValueFromJSON(job.InputJSON),
		"company_profile": mapValue(firstNonNil(
			result["company_profile"],
			result["profile"],
		)),
		"form_fill":     mapValue(firstNonNil(result["form_fill"], result["suggestions"])),
		"suggestions":   mapValue(result["suggestions"]),
		"insights":      mapValue(result["insights"]),
		"confidence":    job.Confidence,
		"sources":       sources,
		"usage":         usagePayload,
		"error_message": job.ErrorMessage,
		"created_at":    job.CreatedAt,
		"updated_at":    job.UpdatedAt,
		"finished_at":   job.FinishedAt,
	}
}

func aiUsageResponse(usage models.AIUsageLog) map[string]interface{} {
	userName := ""
	if usage.UserID > 0 {
		var user models.User
		if err := utils.DB.Select("id, username, name, email").Where("tenant_id = ? AND id = ?", usage.TenantID, usage.UserID).First(&user).Error; err == nil {
			userName = firstNonEmpty(user.Name, user.Username, user.Email)
		}
	}
	return map[string]interface{}{
		"id":                            usage.ID,
		"user_id":                       usage.UserID,
		"user_name":                     userName,
		"feature":                       usage.Feature,
		"object_domain":                 usage.ObjectDomain,
		"object_type":                   usage.ObjectType,
		"object_id":                     usage.ObjectID,
		"provider":                      usage.Provider,
		"model":                         usage.Model,
		"prompt_tokens":                 usage.PromptTokens,
		"completion_tokens":             usage.CompletionTokens,
		"total_tokens":                  usage.TotalTokens,
		"unit_price_per_million":        microsToAmount(usage.UnitPriceMicrosPerMillion),
		"unit_price_micros_per_million": usage.UnitPriceMicrosPerMillion,
		"cost_amount":                   microsToAmount(usage.CostMicros),
		"cost_micros":                   usage.CostMicros,
		"currency":                      usage.Currency,
		"status":                        usage.Status,
		"trace_id":                      usage.TraceID,
		"latency_ms":                    usage.LatencyMS,
		"error_message":                 usage.ErrorMessage,
		"metadata":                      mapValueFromJSON(usage.MetadataJSON),
		"created_at":                    usage.CreatedAt,
	}
}

func normalizeResearchURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("官网地址格式无效")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("官网地址仅支持 http 或 https")
	}
	parsed.Fragment = ""
	return parsed.String(), nil
}

func calculateAICostMicros(tokens int64, unitPriceMicrosPerMillion int64) int64 {
	if tokens <= 0 || unitPriceMicrosPerMillion <= 0 {
		return 0
	}
	return int64(math.Ceil(float64(tokens) * float64(unitPriceMicrosPerMillion) / 1000000.0))
}

func microsToAmount(value int64) string {
	return fmt.Sprintf("%.6f", float64(value)/1000000.0)
}

func estimateTokens(text string) int64 {
	runes := []rune(text)
	if len(runes) == 0 {
		return 0
	}
	return int64(math.Ceil(float64(len(runes)) / 3.2))
}

func mapValueFromJSON(raw string) map[string]interface{} {
	payload := map[string]interface{}{}
	_ = json.Unmarshal([]byte(raw), &payload)
	return payload
}

func stringSlice(value interface{}) []string {
	items := []string{}
	for _, item := range interfaceList(value) {
		text := strings.TrimSpace(fmt.Sprint(item))
		if text != "" && text != "<nil>" {
			items = append(items, text)
		}
	}
	if len(items) == 0 {
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" && text != "<nil>" {
			items = splitCSV(text)
		}
	}
	return items
}

func isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		return text == "" || text == "<nil>"
	case []interface{}:
		return len(typed) == 0
	case []string:
		return len(typed) == 0
	default:
		return strings.TrimSpace(fmt.Sprint(value)) == "" || strings.TrimSpace(fmt.Sprint(value)) == "<nil>"
	}
}

func firstNonNil(values ...interface{}) interface{} {
	for _, value := range values {
		if !isEmptyValue(value) {
			return value
		}
	}
	return nil
}

func nonZeroInt64(value int64, fallback int64) int64 {
	if value > 0 {
		return value
	}
	return fallback
}

func extractJSONObject(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		return content[start : end+1]
	}
	return content
}

func averageSuggestionConfidence(suggestions map[string]interface{}) float64 {
	if len(suggestions) == 0 {
		return 0
	}
	total := 0.0
	count := 0.0
	for _, raw := range suggestions {
		confidence := numericValue(mapValue(raw)["confidence"])
		if confidence > 0 {
			total += confidence
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return math.Round((total/count)*100) / 100
}

func newResearchTraceID(id uint) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%d-%d", id, time.Now().UnixNano())))
	return "ai_" + hex.EncodeToString(sum[:])[:16]
}

func extractHTMLTitle(html string) string {
	re := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	match := re.FindStringSubmatch(html)
	if len(match) < 2 {
		return ""
	}
	return normalizeResearchText(match[1])
}

func stripHTML(html string) string {
	reScript := regexp.MustCompile(`(?is)<(script|style|noscript)[^>]*>.*?</(script|style|noscript)>`)
	html = reScript.ReplaceAllString(html, " ")
	reTags := regexp.MustCompile(`(?s)<[^>]+>`)
	text := reTags.ReplaceAllString(html, " ")
	reEntities := strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&#39;", "'", "&quot;", "\"")
	return reEntities.Replace(text)
}

func normalizeResearchText(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	return strings.Join(fields, " ")
}

func titleOrURL(title string, rawURL string) string {
	title = strings.TrimSpace(title)
	if title != "" {
		return title
	}
	return rawURL
}

func sourceSummariesText(sources []map[string]interface{}) string {
	parts := []string{}
	for _, source := range sources {
		parts = append(parts, strings.TrimSpace(fmt.Sprint(source["title"])), strings.TrimSpace(fmt.Sprint(source["summary"])))
	}
	return strings.Join(parts, " ")
}

func sourceURLs(sources []map[string]interface{}) []string {
	urls := []string{}
	for _, source := range sources {
		raw := strings.TrimSpace(fmt.Sprint(source["url"]))
		if raw != "" && raw != "<nil>" {
			urls = append(urls, raw)
		}
	}
	return urls
}

func inferCompanyName(website string, sources []map[string]interface{}) string {
	for _, source := range sources {
		title := strings.TrimSpace(fmt.Sprint(source["title"]))
		if title != "" {
			title = strings.Split(title, "|")[0]
			title = strings.Split(title, "-")[0]
			if len([]rune(title)) <= 80 {
				return strings.TrimSpace(title)
			}
		}
	}
	if parsed, err := url.Parse(website); err == nil {
		host := strings.TrimPrefix(parsed.Hostname(), "www.")
		if host != "" {
			return host
		}
	}
	return ""
}

func inferIndustry(text string) string {
	lower := strings.ToLower(text)
	candidates := []struct{ key, label string }{
		{"saas", "SaaS/企业软件"},
		{"crm", "企业服务/CRM"},
		{"manufactur", "制造业"},
		{"工业", "制造业"},
		{"医疗", "医疗健康"},
		{"health", "医疗健康"},
		{"教育", "教育科技"},
		{"finance", "金融科技"},
		{"金融", "金融科技"},
		{"retail", "零售电商"},
		{"电商", "零售电商"},
		{"logistics", "物流供应链"},
		{"供应链", "物流供应链"},
		{"security", "网络安全"},
		{"安全", "网络安全"},
		{"ai", "人工智能"},
	}
	for _, item := range candidates {
		if strings.Contains(lower, item.key) {
			return item.label
		}
	}
	return "待确认行业"
}

func inferCompanySize(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "10000+") || strings.Contains(lower, "万名员工"):
		return "10000人以上"
	case strings.Contains(lower, "1000") || strings.Contains(lower, "千名员工"):
		return "1001-10000人"
	case strings.Contains(lower, "500"):
		return "501-1000人"
	case strings.Contains(lower, "200") || strings.Contains(lower, "large"):
		return "201-500人"
	case strings.Contains(lower, "50") || strings.Contains(lower, "team"):
		return "51-200人"
	default:
		return "规模待确认"
	}
}

func inferListByKeywords(text string, keys []string, limit int) []string {
	sentences := splitSentences(text)
	items := []string{}
	for _, sentence := range sentences {
		lower := strings.ToLower(sentence)
		for _, key := range keys {
			if strings.Contains(lower, strings.ToLower(key)) {
				items = append(items, trimRunes(sentence, 90))
				break
			}
		}
		if len(items) >= limit {
			break
		}
	}
	return uniqueStrings(items)
}

func splitSentences(text string) []string {
	re := regexp.MustCompile(`[。！？.!?\n]+`)
	parts := re.Split(text, -1)
	items := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

func trimRunes(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}

func inferGrade(confidence float64, buyingSignals []string) string {
	if confidence >= 0.82 || len(buyingSignals) >= 2 {
		return "important"
	}
	if confidence >= 0.7 {
		return "potential"
	}
	return "normal"
}

func buildCompanyDescription(name string, industry string, products []string) string {
	parts := []string{}
	if name != "" {
		parts = append(parts, name)
	}
	if industry != "" {
		parts = append(parts, "主要处于"+industry+"领域")
	}
	if len(products) > 0 {
		parts = append(parts, "公开资料显示其重点业务包括"+strings.Join(products, "、"))
	}
	if len(parts) == 0 {
		return "公开资料不足，建议继续补充公司官网和行业线索。"
	}
	return strings.Join(parts, "，") + "。"
}

func buildSalesNote(name string, industry string, buyingSignals []string, riskSignals []string) string {
	items := []string{}
	if name != "" || industry != "" {
		items = append(items, fmt.Sprintf("建议围绕%s的%s场景确认预算、痛点和决策链。", name, industry))
	}
	if len(buyingSignals) > 0 {
		items = append(items, "可关注信号："+strings.Join(buyingSignals, "；"))
	}
	if len(riskSignals) > 0 {
		items = append(items, "需核实风险："+strings.Join(riskSignals, "；"))
	}
	return strings.Join(items, "\n")
}

func inferRegion(region string, text string) string {
	if strings.TrimSpace(region) != "" {
		return strings.TrimSpace(region)
	}
	for _, candidate := range []string{"中国", "上海", "北京", "深圳", "杭州", "广州", "美国", "新加坡", "欧洲", "日本"} {
		if strings.Contains(text, candidate) {
			return candidate
		}
	}
	return "地区待确认"
}

func inferTargetCustomers(industry string) []string {
	if strings.Contains(industry, "企业") || strings.Contains(industry, "SaaS") {
		return []string{"中大型企业客户", "业务部门负责人", "IT/数字化团队"}
	}
	if strings.Contains(industry, "制造") {
		return []string{"制造企业", "供应链团队", "生产运营负责人"}
	}
	return []string{"目标客户待进一步确认"}
}

func inferBusinessModel(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "subscription") || strings.Contains(lower, "订阅"):
		return "订阅制"
	case strings.Contains(lower, "marketplace") || strings.Contains(lower, "平台"):
		return "平台型业务"
	case strings.Contains(lower, "service") || strings.Contains(lower, "服务"):
		return "产品+服务"
	default:
		return "商业模式待确认"
	}
}

func inferMarkets(region string, text string) []string {
	items := []string{}
	for _, candidate := range []string{"中国", "亚太", "北美", "欧洲", "全球", "上海", "北京", "深圳", "香港", "新加坡"} {
		if strings.Contains(text, candidate) {
			items = append(items, candidate)
		}
	}
	if strings.TrimSpace(region) != "" {
		items = append(items, strings.TrimSpace(region))
	}
	if len(items) == 0 {
		items = append(items, "市场区域待确认")
	}
	return uniqueStrings(items)
}

func inferTechStack(text string) []string {
	items := []string{}
	lower := strings.ToLower(text)
	for _, candidate := range []string{"salesforce", "hubspot", "sap", "oracle", "aws", "azure", "kubernetes", "ai", "machine learning", "api", "云"} {
		if strings.Contains(lower, strings.ToLower(candidate)) {
			items = append(items, candidate)
		}
	}
	if len(items) == 0 {
		items = append(items, "公开资料未明确技术栈")
	}
	return uniqueStrings(items)
}

func recommendedNextActions(industry string, buyingSignals []string) []string {
	items := []string{"确认公司当前业务重点、预算周期和决策链", "基于官网业务线准备行业案例和价值假设"}
	if len(buyingSignals) > 0 {
		items = append(items, "围绕采购/增长信号设计首轮触达话术")
	}
	if strings.Contains(industry, "待确认") {
		items = append(items, "先通过联系人访谈确认行业归类和核心场景")
	}
	return items
}

func compactStrings(values []string) []string {
	items := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && value != "<nil>" && !strings.Contains(value, "待确认") {
			items = append(items, value)
		}
	}
	return uniqueStrings(items)
}
