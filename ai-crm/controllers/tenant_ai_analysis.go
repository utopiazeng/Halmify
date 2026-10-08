package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	tenantBusinessAnalysisFeature  = "tenant_business_analysis"
	tenantAnalysisObjectDomain     = "tenant"
	tenantAnalysisObjectType       = "business_analysis"
	tenantAnalysisReserveFactor    = 12
	tenantAnalysisReserveDivisor   = 10
	tenantAnalysisSnapshotTimeout  = 8 * time.Second
	tenantAnalysisCountTimeout     = 1200 * time.Millisecond
	tenantAnalysisSnapshotCacheTTL = 5 * time.Minute
	tenantAnalysisSampleLimit      = 200
	tenantAnalysisEvidenceLimit    = 120
	tenantAnalysisTopLimit         = 10
	tenantAnalysisOpportunityLimit = 80
	tenantAnalysisTicketLimit      = 200
)

var tenantAnalysisSectionSet = map[string]bool{
	"overview":          true,
	"customers":         true,
	"sales_pipeline":    true,
	"revenue_orders":    true,
	"products_projects": true,
	"service_delivery":  true,
	"team_execution":    true,
}

type tenantAnalysisRequest struct {
	PeriodType     string   `json:"period_type"`
	StartDate      string   `json:"start_date"`
	EndDate        string   `json:"end_date"`
	Sections       []string `json:"sections"`
	Depth          string   `json:"depth"`
	CompareEnabled bool     `json:"compare_enabled"`
}

type tenantAIApplicationRequest struct {
	ApplicationType string `json:"application_type"`
	PackageCode     string `json:"package_code"`
	ExpectedUsage   string `json:"expected_usage"`
	ContactName     string `json:"contact_name"`
	ContactEmail    string `json:"contact_email"`
	ContactPhone    string `json:"contact_phone"`
	Note            string `json:"note"`
}

type tenantAISettingsRequest struct {
	Provider string `json:"provider"`
	APIURL   string `json:"api_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

type tenantAnalysisUsage struct {
	PromptTokens        int64 `json:"prompt_tokens"`
	CompletionTokens    int64 `json:"completion_tokens"`
	TotalTokens         int64 `json:"total_tokens"`
	CachedInputTokens   int64 `json:"cached_input_tokens"`
	UncachedInputTokens int64 `json:"uncached_input_tokens"`
	OutputTokens        int64 `json:"output_tokens"`
}

type tenantAnalysisPricing struct {
	Snapshot      aiPricingSnapshot `json:"snapshot"`
	Model         aiPricingModel    `json:"model"`
	InputHitRate  aiPricingRate     `json:"input_hit_rate"`
	InputMissRate aiPricingRate     `json:"input_miss_rate"`
	OutputRate    aiPricingRate     `json:"output_rate"`
	Provider      string            `json:"provider"`
	ModelName     string            `json:"model_name"`
	Currency      string            `json:"currency"`
}

const (
	tenantAISettingsScope   = "ai_analysis"
	tenantAISettingsKey     = "provider_settings"
	tenantAIModelsKey       = "models"
	openAIProvider          = "openai"
	tenantAIModelSyncWindow = 48 * time.Hour
)

var fallbackDeepSeekModels = []string{currentDeepSeekModel(), "deepseek-v4-pro", "deepseek-chat", "deepseek-reasoner"}
var fallbackOpenAIModels = []string{"gpt-5.5", "gpt-5.5-pro", "gpt-5.4", "gpt-5.4-pro", "gpt-5.4-mini", "gpt-5.4-nano", "gpt-5-mini", "gpt-5-nano", "gpt-5", "gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano", "gpt-5-chat-latest"}

func (c *CRMController) TenantAIAnalysisStatus() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "insights", "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	wallet, _ := ensureTenantAIWallet(auth.TenantID)
	resetWalletMonthIfNeeded(&wallet)
	settings := tenantAISettingsForTenant(auth.TenantID)
	var latest models.TenantAIApplication
	latestPayload := map[string]interface{}{}
	if err := utils.DB.Where("tenant_id = ?", auth.TenantID).Order("created_at desc").First(&latest).Error; err == nil {
		latestPayload = tenantAIApplicationResponse(latest)
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"enabled":               true,
		"configured":            tenantAISettingsConfigured(settings),
		"ai_settings":           tenantAISettingsResponse(settings),
		"provider_options":      tenantAIProviderOptions(auth.TenantID),
		"wallet":                aiWalletResponse(wallet),
		"tenant_identity":       tenantAIIdentityPayload(auth),
		"pricing":               ensureAIPricingSnapshot(0),
		"packages":              activeAIAnalysisPackages(),
		"latest_application":    latestPayload,
		"can_apply":             auth.TenantID > 0,
		"can_manage_settings":   auth.IsTenantAdmin() || authHasPermissionAny(auth, "insights:manage"),
		"can_view_tenant_usage": canViewTenantAIUsage(auth),
		"can_create_report":     auth.IsTenantAdmin() || authHasPermissionAny(auth, "insights:create", "insights:manage"),
		"can_export_report":     auth.IsTenantAdmin() || authHasPermissionAny(auth, "insights:export", "insights:manage"),
	})
}

func (c *CRMController) GetTenantAIAnalysisSettings() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "insights", "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	settings := tenantAISettingsForTenant(auth.TenantID)
	utils.Success(c.Ctx, map[string]interface{}{
		"settings":         tenantAISettingsResponse(settings),
		"provider_options": tenantAIProviderOptions(auth.TenantID),
	})
}

func (c *CRMController) SaveTenantAIAnalysisSettings() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var req tenantAISettingsRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	current := tenantAISettingsForTenant(auth.TenantID)
	settings, err := normalizeTenantAISettingsRequest(req, current)
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	payload := map[string]interface{}{
		"provider": settings.Provider,
		"api_url":  settings.APIURL,
		"model":    settings.Model,
		"api_key":  settings.APIKey,
	}
	if err := saveCRMSetting(auth.TenantID, tenantAISettingsScope, tenantAISettingsKey, payload, auth.UserID); err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_TENANT_AI_SETTINGS", "tenant_ai_analysis.settings", "operation.update_crm_setting", map[string]interface{}{"provider": settings.Provider, "model": settings.Model}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存 AI 配置失败")
		return
	}
	go syncTenantAIModels(auth.TenantID, settings, auth.UserID)
	logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_TENANT_AI_SETTINGS", "tenant_ai_analysis.settings", "operation.update_crm_setting", map[string]interface{}{"provider": settings.Provider, "model": settings.Model}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"settings": tenantAISettingsResponse(settings)})
}

func tenantAIIdentityPayload(auth authContext) map[string]interface{} {
	payload := map[string]interface{}{
		"tenant_id": auth.TenantID,
		"user_id":   auth.UserID,
		"username":  auth.Username,
	}
	var tenant models.Tenant
	if err := utils.DB.Select("id, name, admin_user_id").Where("id = ?", auth.TenantID).First(&tenant).Error; err == nil {
		payload["tenant_name"] = tenant.Name
		payload["tenant_admin_user_id"] = tenant.AdminUserID
	}
	var user models.User
	if err := utils.DB.Select("id, username, name, email, phone, account_type").Where("id = ?", auth.UserID).First(&user).Error; err == nil {
		payload["username"] = user.Username
		payload["name"] = user.Name
		payload["email"] = user.Email
		payload["phone"] = user.Phone
		payload["account_type"] = user.AccountType
	}
	return payload
}

func tenantAISettingsForTenant(tenantID uint) aiResearchSettings {
	payload := crmSettingValue(tenantID, tenantAISettingsScope, tenantAISettingsKey)
	provider := normalizeAIProviderName(fmt.Sprint(payload["provider"]))
	if provider == "" || provider == "<nil>" {
		provider = deepSeekProvider
	}
	if provider == "chatgpt" || provider == "openai-compatible" {
		provider = openAIProvider
	}
	apiURL := strings.TrimSpace(fmt.Sprint(payload["api_url"]))
	apiKey := strings.TrimSpace(fmt.Sprint(payload["api_key"]))
	model := strings.TrimSpace(fmt.Sprint(payload["model"]))
	if apiURL == "<nil>" {
		apiURL = ""
	}
	if apiKey == "<nil>" {
		apiKey = ""
	}
	if model == "<nil>" {
		model = ""
	}
	defaults := tenantAIProviderDefault(provider)
	if apiURL == "" {
		apiURL = defaults.APIURL
	}
	if model == "" {
		model = defaults.Model
	}
	if provider == deepSeekProvider {
		model = effectiveDeepSeekModel(model)
	}
	return aiResearchSettings{
		Provider:           provider,
		Model:              model,
		Currency:           defaultAICurrency,
		UnitPriceMicrosPM:  defaultPriceMicrosPM,
		EstimatedMaxTokens: 25000,
		MaxCostMicros:      defaultMaxResearchCost,
		MaxURLs:            defaultMaxResearchURLs,
		RequestTimeout:     90 * time.Second,
		APIURL:             apiURL,
		APIKey:             apiKey,
	}
}

func tenantAIProviderDefault(provider string) aiResearchSettings {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case openAIProvider:
		return aiResearchSettings{Provider: openAIProvider, APIURL: "https://api.openai.com/v1", Model: "gpt-5.5", Currency: defaultAICurrency}
	default:
		return aiResearchSettings{Provider: deepSeekProvider, APIURL: "https://api.deepseek.com", Model: currentDeepSeekModel(), Currency: defaultAICurrency}
	}
}

func tenantAISettingsConfigured(settings aiResearchSettings) bool {
	return strings.TrimSpace(settings.APIURL) != "" && tenantAIAPIKeyConfigured(settings.APIKey) && strings.TrimSpace(settings.Model) != ""
}

func tenantAISettingsResponse(settings aiResearchSettings) map[string]interface{} {
	return map[string]interface{}{
		"provider":       settings.Provider,
		"api_url":        settings.APIURL,
		"model":          settings.Model,
		"has_api_key":    tenantAIAPIKeyConfigured(settings.APIKey),
		"masked_api_key": maskAPIKey(settings.APIKey),
		"configured":     tenantAISettingsConfigured(settings),
	}
}

func tenantAIProviderOptions(tenantID uint) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"value":         deepSeekProvider,
			"label":         "DeepSeek",
			"default_url":   "https://api.deepseek.com",
			"default_model": currentDeepSeekModel(),
			"models":        tenantAIModelOptions(tenantID, deepSeekProvider),
			"help_url":      "https://platform.deepseek.com/api_keys",
		},
		{
			"value":         openAIProvider,
			"label":         "ChatGPT / OpenAI",
			"default_url":   "https://api.openai.com/v1",
			"default_model": "gpt-5.5",
			"models":        tenantAIModelOptions(tenantID, openAIProvider),
			"help_url":      "https://platform.openai.com/api-keys",
		},
	}
}

func normalizeTenantAISettingsRequest(req tenantAISettingsRequest, current aiResearchSettings) (aiResearchSettings, error) {
	provider := normalizeAIProviderName(req.Provider)
	if provider == "" {
		provider = current.Provider
	}
	if provider != deepSeekProvider && provider != openAIProvider {
		return aiResearchSettings{}, fmt.Errorf("暂只支持 DeepSeek 和 ChatGPT/OpenAI")
	}
	defaults := tenantAIProviderDefault(provider)
	apiURL := strings.TrimSpace(req.APIURL)
	if apiURL == "" {
		apiURL = defaults.APIURL
	}
	if _, err := url.ParseRequestURI(apiURL); err != nil {
		return aiResearchSettings{}, fmt.Errorf("AI API 地址格式不正确")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = defaults.Model
	}
	if provider == deepSeekProvider {
		model = effectiveDeepSeekModel(model)
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if isMaskedAPIKey(apiKey) {
		apiKey = current.APIKey
	}
	if apiKey == "" {
		apiKey = current.APIKey
	}
	if !tenantAIAPIKeyConfigured(apiKey) {
		return aiResearchSettings{}, fmt.Errorf("请填写 AI API Key")
	}
	return aiResearchSettings{
		Provider:           provider,
		Model:              model,
		Currency:           defaultAICurrency,
		UnitPriceMicrosPM:  defaultPriceMicrosPM,
		EstimatedMaxTokens: 25000,
		MaxCostMicros:      defaultMaxResearchCost,
		MaxURLs:            defaultMaxResearchURLs,
		RequestTimeout:     90 * time.Second,
		APIURL:             apiURL,
		APIKey:             apiKey,
	}, nil
}

func maskAPIKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

func isMaskedAPIKey(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && strings.Contains(value, "****")
}

func tenantAIAPIKeyConfigured(value string) bool {
	key := strings.TrimSpace(value)
	if key == "" || isMaskedAPIKey(key) {
		return false
	}
	lower := strings.ToLower(key)
	invalids := []string{"local-fallback", "fallback", "test", "demo", "your-api-key", "your_api_key", "replace-me", "changeme", "sk-xxxx", "sk-****"}
	for _, item := range invalids {
		if lower == item || strings.Contains(lower, item) {
			return false
		}
	}
	if len(key) < 16 {
		return false
	}
	return true
}

func tenantAIModelOptions(tenantID uint, provider string) []string {
	fallback := fallbackAIModelsForProvider(provider)
	if tenantID > 0 {
		payload := crmSettingValue(tenantID, tenantAISettingsScope, tenantAIModelsCacheKey(provider))
		models := stringSlice(payload["models"])
		if len(models) > 0 {
			return mergeModelOptions(models, fallback)
		}
	}
	return append([]string{}, fallback...)
}

func fallbackAIModelsForProvider(provider string) []string {
	if normalizeAIProviderName(provider) == openAIProvider {
		return fallbackOpenAIModels
	}
	return fallbackDeepSeekModels
}

func tenantAIModelsCacheKey(provider string) string {
	provider = normalizeAIProviderName(provider)
	if provider == "" {
		provider = deepSeekProvider
	}
	return tenantAIModelsKey + "_" + provider
}

func mergeModelOptions(primary []string, fallback []string) []string {
	seen := map[string]bool{}
	out := []string{}
	add := func(items []string) {
		for _, item := range items {
			model := strings.TrimSpace(item)
			if model == "" || seen[model] {
				continue
			}
			seen[model] = true
			out = append(out, model)
		}
	}
	add(primary)
	add(fallback)
	return out
}

func syncTenantAIModels(tenantID uint, settings aiResearchSettings, userID uint) {
	if tenantID == 0 || strings.TrimSpace(settings.APIKey) == "" {
		return
	}
	models, err := fetchTenantAIModels(settings)
	if err != nil {
		saveTenantAIModelSyncStatus(tenantID, settings.Provider, nil, "failed", err.Error(), userID)
		utils.LogWarning("租户 %d %s 模型同步失败: %v", tenantID, settings.Provider, err)
		return
	}
	saveTenantAIModelSyncStatus(tenantID, settings.Provider, models, "synced", "", userID)
}

func fetchTenantAIModels(settings aiResearchSettings) ([]string, error) {
	endpoint := strings.TrimRight(settings.APIURL, "/")
	if endpoint == "" {
		endpoint = tenantAIProviderDefault(settings.Provider).APIURL
	}
	if strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint = strings.TrimSuffix(endpoint, "/chat/completions")
	}
	if !strings.HasSuffix(endpoint, "/v1") {
		endpoint += "/v1"
	}
	req, err := http.NewRequest(http.MethodGet, endpoint+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(settings.APIKey))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s 模型列表返回状态 %d: %s", settings.Provider, resp.StatusCode, trimRunes(string(body), 160))
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	models := []string{}
	for _, item := range parsed.Data {
		id := strings.TrimSpace(item.ID)
		if isAIChatModelCandidate(settings.Provider, id) {
			models = append(models, id)
		}
	}
	sort.SliceStable(models, func(i, j int) bool {
		return aiModelRank(settings.Provider, models[i]) < aiModelRank(settings.Provider, models[j])
	})
	models = mergeModelOptions(models, nil)
	if len(models) == 0 {
		return nil, fmt.Errorf("%s 模型列表未返回可用于经营分析的模型", settings.Provider)
	}
	if len(models) > 80 {
		models = models[:80]
	}
	return models, nil
}

func isAIChatModelCandidate(provider string, model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return false
	}
	if strings.Contains(model, "embedding") || strings.Contains(model, "audio") || strings.Contains(model, "image") || strings.Contains(model, "tts") || strings.Contains(model, "whisper") || strings.Contains(model, "moderation") || strings.Contains(model, "realtime") || strings.Contains(model, "transcribe") {
		return false
	}
	if normalizeAIProviderName(provider) == deepSeekProvider {
		return strings.HasPrefix(model, "deepseek")
	}
	return strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "chatgpt-") || strings.HasPrefix(model, "o")
}

func aiModelRank(provider string, model string) int {
	lower := strings.ToLower(model)
	if normalizeAIProviderName(provider) == deepSeekProvider {
		switch {
		case lower == currentDeepSeekModel():
			return 10
		case strings.Contains(lower, "flash"):
			return 20
		case strings.Contains(lower, "chat"):
			return 30
		case strings.Contains(lower, "reasoner"):
			return 40
		default:
			return 100
		}
	}
	switch {
	case strings.HasPrefix(lower, "gpt-5.5"):
		return 10
	case strings.HasPrefix(lower, "gpt-5.4"):
		return 20
	case strings.HasPrefix(lower, "gpt-5"):
		return 30
	case strings.HasPrefix(lower, "gpt-4.1"):
		return 40
	case strings.HasPrefix(lower, "gpt-4o"):
		return 50
	case strings.HasPrefix(lower, "o"):
		return 60
	default:
		return 100
	}
}

func saveTenantAIModelSyncStatus(tenantID uint, provider string, models []string, status string, message string, userID uint) {
	provider = normalizeAIProviderName(provider)
	current := crmSettingValue(tenantID, tenantAISettingsScope, tenantAIModelsCacheKey(provider))
	if len(models) == 0 {
		models = stringSlice(current["models"])
	}
	payload := map[string]interface{}{
		"provider":       provider,
		"models":         models,
		"status":         status,
		"message":        message,
		"last_synced_at": time.Now().Format(time.RFC3339),
		"next_sync_at":   time.Now().Add(tenantAIModelSyncWindow).Format(time.RFC3339),
	}
	_ = saveCRMSetting(tenantID, tenantAISettingsScope, tenantAIModelsCacheKey(provider), payload, userID)
}

func StartTenantAIModelSyncScheduler() {
	run := func() {
		syncConfiguredTenantAIModelCaches()
	}
	go func() {
		services.WithSchedulerLock("tenant_ai_model_sync_scheduler", run)
		ticker := time.NewTicker(tenantAIModelSyncWindow)
		defer ticker.Stop()
		for range ticker.C {
			services.WithSchedulerLock("tenant_ai_model_sync_scheduler", run)
		}
	}()
}

func syncConfiguredTenantAIModelCaches() {
	var settingsRows []models.CRMSetting
	if err := utils.DB.Where("scope = ? AND `key` = ?", tenantAISettingsScope, tenantAISettingsKey).Find(&settingsRows).Error; err != nil {
		utils.LogWarning("读取租户 AI 模型同步配置失败: %v", err)
		return
	}
	for _, row := range settingsRows {
		settings := tenantAISettingsFromPayload(mapValueFromJSON(row.ValueJSON))
		if !tenantAISettingsConfigured(settings) {
			continue
		}
		cache := crmSettingValue(row.TenantID, tenantAISettingsScope, tenantAIModelsCacheKey(settings.Provider))
		if syncedAt := strings.TrimSpace(fmt.Sprint(cache["last_synced_at"])); syncedAt != "" && syncedAt != "<nil>" {
			if parsed, err := time.Parse(time.RFC3339, syncedAt); err == nil && time.Since(parsed) < tenantAIModelSyncWindow {
				continue
			}
		}
		syncTenantAIModels(row.TenantID, settings, row.UpdatedBy)
	}
}

func tenantAISettingsFromPayload(payload map[string]interface{}) aiResearchSettings {
	provider := normalizeAIProviderName(fmt.Sprint(payload["provider"]))
	if provider == "" || provider == "<nil>" {
		provider = deepSeekProvider
	}
	apiURL := strings.TrimSpace(fmt.Sprint(payload["api_url"]))
	apiKey := strings.TrimSpace(fmt.Sprint(payload["api_key"]))
	model := strings.TrimSpace(fmt.Sprint(payload["model"]))
	if apiURL == "<nil>" {
		apiURL = ""
	}
	if apiKey == "<nil>" {
		apiKey = ""
	}
	if model == "<nil>" {
		model = ""
	}
	defaults := tenantAIProviderDefault(provider)
	if apiURL == "" {
		apiURL = defaults.APIURL
	}
	if model == "" {
		model = defaults.Model
	}
	return aiResearchSettings{Provider: provider, APIURL: apiURL, APIKey: apiKey, Model: model, Currency: defaultAICurrency, RequestTimeout: 90 * time.Second}
}

func normalizeAIProviderName(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "chatgpt" || provider == "openai-compatible" {
		return openAIProvider
	}
	return provider
}

func (c *CRMController) CreateTenantAIApplication() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var req tenantAIApplicationRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.PackageCode = strings.TrimSpace(req.PackageCode)
	if req.PackageCode == "" {
		utils.BadRequest(c.Ctx, "请选择 AI 分析套餐")
		return
	}
	var pkg models.AIAnalysisPackage
	if err := utils.DB.Where("code = ? AND status = ?", req.PackageCode, "active").First(&pkg).Error; err != nil {
		utils.BadRequest(c.Ctx, "AI 分析套餐不存在或已停用")
		return
	}
	app := models.TenantAIApplication{
		TenantModel:     models.TenantModel{TenantID: auth.TenantID},
		UserID:          auth.UserID,
		ApplicationType: firstNonEmpty(strings.TrimSpace(req.ApplicationType), "enable"),
		PackageCode:     req.PackageCode,
		ExpectedUsage:   strings.TrimSpace(req.ExpectedUsage),
		ContactName:     strings.TrimSpace(req.ContactName),
		ContactEmail:    strings.TrimSpace(req.ContactEmail),
		ContactPhone:    strings.TrimSpace(req.ContactPhone),
		Note:            strings.TrimSpace(req.Note),
		Status:          "pending",
	}
	if err := utils.DB.Create(&app).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "提交 AI 分析开通申请失败")
		return
	}
	supportMessage := ""
	if convID, msg, _, err := imSvc.OpenSupportConversation(auth.TenantID, auth.UserID, services.IMSupportInput{
		Category: "billing_ai",
		Priority: "normal",
		Title:    "申请开通 AI 经营分析",
		Content: fmt.Sprintf("租户管理员申请开通 AI 经营分析。\n套餐：%s\n联系人：%s\n邮箱：%s\n使用频率：%s\n备注：%s",
			req.PackageCode, app.ContactName, app.ContactEmail, app.ExpectedUsage, app.Note),
	}); err == nil {
		if platformAdminIDs, receiptErr := imSvc.EnsureSupportReceiptsForPlatformAdmins(auth.TenantID, convID, msg.ID); receiptErr == nil && len(platformAdminIDs) > 0 {
			for _, platformAdminID := range platformAdminIDs {
				services.PublishIMEvent("im.message.new", 0, platformAdminID, msg)
				if summary, sumErr := imSvc.UnreadSummary(0, platformAdminID); sumErr == nil {
					services.PublishIMEvent("im.conversation.unread", 0, platformAdminID, summary)
				}
			}
		}
		supportMessage = "sent"
	} else {
		supportMessage = err.Error()
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_AI_ANALYSIS_APPLICATION", fmt.Sprintf("AI分析申请[%d]", app.ID), "operation.create_ai_analysis_application", map[string]interface{}{"package_code": req.PackageCode}, "success", "")
	resp := tenantAIApplicationResponse(app)
	resp["support_message_status"] = supportMessage
	utils.Success(c.Ctx, resp)
}

func (c *CRMController) ListTenantAIApplications() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !auth.IsTenantAdmin() && !authHasPermissionAny(auth, "insights:view", "insights:manage") {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	var apps []models.TenantAIApplication
	_ = utils.DB.Where("tenant_id = ?", auth.TenantID).Order("created_at desc").Limit(50).Find(&apps).Error
	items := make([]map[string]interface{}, 0, len(apps))
	for _, app := range apps {
		items = append(items, tenantAIApplicationResponse(app))
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": len(items)})
}

func (c *CRMController) EstimateTenantAIAnalysis() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := requireTenantAnalysisPermission(c.Ctx, auth, "create"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	if !allowTenantAnalysisRate(auth.TenantID, auth.UserID, c.Ctx.Input.IP(), "estimate", 5, 8) {
		recordBlockedTenantAnalysisUsage(auth, 0, "AI 经营分析预估过于频繁", "")
		utils.Error(c.Ctx, http.StatusTooManyRequests, "AI 经营分析预估过于频繁")
		return
	}
	req, period, ok := c.bindTenantAnalysisRequest()
	if !ok {
		return
	}
	settings := tenantAISettingsForTenant(auth.TenantID)
	if !tenantAISettingsConfigured(settings) {
		utils.Error(c.Ctx, http.StatusForbidden, "请先配置企业自己的 AI API Key 后再生成经营分析")
		return
	}
	snapshot := buildTenantAnalysisSnapshot(auth, req, period)
	prompt := buildTenantAnalysisPrompt(snapshot, req)
	usage := tenantAnalysisUsage{PromptTokens: estimateTokens(prompt), CompletionTokens: estimatedTenantAnalysisCompletionTokens(req.Depth)}
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	usage.UncachedInputTokens = usage.PromptTokens
	usage.OutputTokens = usage.CompletionTokens
	pricing := tenantAnalysisPricingForSettings(settings)
	cost := calculateTenantAnalysisCostMicros(usage, pricing)
	reserveCost := int64(0)
	canGenerate := true
	blockReason := ""
	utils.Success(c.Ctx, map[string]interface{}{
		"period":                  period,
		"sections":                req.Sections,
		"depth":                   req.Depth,
		"estimated_prompt_tokens": usage.PromptTokens,
		"estimated_output_tokens": usage.OutputTokens,
		"estimated_total_tokens":  usage.TotalTokens,
		"estimated_cost_micros":   cost,
		"estimated_cost_amount":   microsToAmount(cost),
		"reserve_cost_micros":     reserveCost,
		"reserve_cost_amount":     microsToAmount(reserveCost),
		"available_micros":        0,
		"available_amount":        microsToAmount(0),
		"can_generate":            canGenerate,
		"block_reason":            blockReason,
		"currency":                pricing.Currency,
		"provider":                settings.Provider,
		"model":                   settings.Model,
		"pricing":                 tenantAnalysisPriceSnapshot(pricing),
	})
}

func (c *CRMController) CreateTenantAIAnalysisJob() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := requireTenantAnalysisPermission(c.Ctx, auth, "create"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	idempotencyKey := strings.TrimSpace(c.Ctx.Input.Header("Idempotency-Key"))
	if idempotencyKey == "" {
		utils.BadRequest(c.Ctx, "缺少 Idempotency-Key")
		return
	}
	if len(idempotencyKey) > 128 {
		utils.BadRequest(c.Ctx, "Idempotency-Key 过长")
		return
	}
	if !allowTenantAnalysisRate(auth.TenantID, auth.UserID, c.Ctx.Input.IP(), "create", 2, 3) {
		recordBlockedTenantAnalysisUsage(auth, 0, "AI 经营分析创建过于频繁", idempotencyKey)
		utils.Error(c.Ctx, http.StatusTooManyRequests, "AI 经营分析创建过于频繁")
		return
	}
	req, period, ok := c.bindTenantAnalysisRequest()
	if !ok {
		return
	}
	settings := tenantAISettingsForTenant(auth.TenantID)
	if !tenantAISettingsConfigured(settings) {
		recordBlockedTenantAnalysisUsage(auth, 0, "请先配置企业自己的 AI API Key 后再生成经营分析", idempotencyKey)
		utils.Error(c.Ctx, http.StatusForbidden, "请先配置企业自己的 AI API Key 后再生成经营分析")
		return
	}
	idempotencyHash := tenantAnalysisIdempotencyHash(auth.TenantID, auth.UserID, idempotencyKey)
	var existing models.AIResearchJob
	if err := utils.DB.Where("tenant_id = ? AND user_id = ? AND object_domain = ? AND object_type = ? AND input_json LIKE ? AND created_at >= ?", auth.TenantID, auth.UserID, tenantAnalysisObjectDomain, tenantAnalysisObjectType, "%"+idempotencyHash+"%", time.Now().Add(-24*time.Hour)).Order("id desc").First(&existing).Error; err == nil {
		utils.Success(c.Ctx, tenantAnalysisJobResponse(existing))
		return
	}
	if hasRunningTenantAnalysis(auth.TenantID, 0) || hasRunningTenantAnalysis(auth.TenantID, auth.UserID) {
		recordBlockedTenantAnalysisUsage(auth, 0, "已有 AI 经营分析任务正在运行", idempotencyKey)
		utils.Error(c.Ctx, http.StatusConflict, "已有 AI 经营分析任务正在运行")
		return
	}
	snapshot := buildTenantAnalysisSnapshot(auth, req, period)
	prompt := buildTenantAnalysisPrompt(snapshot, req)
	estimated := tenantAnalysisUsage{PromptTokens: estimateTokens(prompt), CompletionTokens: estimatedTenantAnalysisCompletionTokens(req.Depth)}
	estimated.TotalTokens = estimated.PromptTokens + estimated.CompletionTokens
	estimated.UncachedInputTokens = estimated.PromptTokens
	estimated.OutputTokens = estimated.CompletionTokens
	pricing := tenantAnalysisPricingForSettings(settings)
	reserveCost := int64(0)
	traceID := newTenantAnalysisTraceID(auth.TenantID, auth.UserID)
	inputJSON := toJSONString(map[string]interface{}{
		"idempotency_hash": idempotencyHash,
		"request":          req,
		"period":           period,
		"estimated_usage":  estimated,
		"reserved_micros":  reserveCost,
		"trace_id":         traceID,
	})
	var job models.AIResearchJob
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		job = models.AIResearchJob{
			TenantModel:  models.TenantModel{TenantID: auth.TenantID},
			UserID:       auth.UserID,
			ObjectDomain: tenantAnalysisObjectDomain,
			ObjectType:   tenantAnalysisObjectType,
			ObjectID:     0,
			Status:       "pending",
			InputJSON:    inputJSON,
			SourceJSON:   toJSONString(snapshot),
		}
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
		ledger := models.AITokenLedgerEntry{
			TenantModel:          models.TenantModel{TenantID: auth.TenantID},
			UserID:               auth.UserID,
			JobID:                job.ID,
			EntryType:            "reserve",
			Direction:            "debit",
			AmountMicros:         reserveCost,
			PromptTokens:         estimated.PromptTokens,
			CompletionTokens:     estimated.CompletionTokens,
			UncachedInputTokens:  estimated.UncachedInputTokens,
			OutputTokens:         estimated.OutputTokens,
			TotalTokens:          estimated.TotalTokens,
			PriceSnapshotJSON:    toJSONString(tenantAnalysisPriceSnapshot(pricing)),
			BalanceBeforeMicros:  0,
			BalanceAfterMicros:   0,
			ReservedBeforeMicros: 0,
			ReservedAfterMicros:  0,
			Status:               "reserved",
			TraceID:              traceID,
			IdempotencyKey:       idempotencyHash,
			Description:          "租户自有 API Key 模式费用预估记录",
		}
		return tx.Create(&ledger).Error
	}); err != nil {
		recordBlockedTenantAnalysisUsage(auth, 0, err.Error(), idempotencyKey)
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_TENANT_AI_ANALYSIS", fmt.Sprintf("AI经营分析[%d]", job.ID), "operation.create_tenant_ai_analysis", map[string]interface{}{"job_id": job.ID, "trace_id": traceID}, "success", "")
	go processTenantAIAnalysisJob(auth, job.ID, req, snapshot, prompt, pricing, reserveCost, traceID, idempotencyHash, settings)
	utils.Success(c.Ctx, tenantAnalysisJobResponse(job))
}

func (c *CRMController) ListTenantAIAnalysisJobs() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := requireTenantAnalysisPermission(c.Ctx, auth, "view"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)
	if size > 100 {
		size = 100
	}
	query := utils.DB.Model(&models.AIResearchJob{}).Where("tenant_id = ? AND object_domain = ? AND object_type = ?", auth.TenantID, tenantAnalysisObjectDomain, tenantAnalysisObjectType)
	if !auth.IsTenantAdmin() && !authHasPermissionAny(auth, "insights:manage", "ai_usage:view") {
		query = query.Where("user_id = ?", auth.UserID)
	}
	var total int64
	_ = query.Count(&total).Error
	var jobs []models.AIResearchJob
	_ = query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&jobs).Error
	items := make([]map[string]interface{}, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, tenantAnalysisJobResponse(job))
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": total})
}

func (c *CRMController) GetTenantAIAnalysisJob() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := requireTenantAnalysisPermission(c.Ctx, auth, "view"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的 AI 经营分析任务")
		return
	}
	var job models.AIResearchJob
	query := utils.DB.Where("tenant_id = ? AND id = ? AND object_domain = ? AND object_type = ?", auth.TenantID, id, tenantAnalysisObjectDomain, tenantAnalysisObjectType)
	if !auth.IsTenantAdmin() && !authHasPermissionAny(auth, "insights:manage", "ai_usage:view") {
		query = query.Where("user_id = ?", auth.UserID)
	}
	if err := query.First(&job).Error; err != nil {
		utils.NotFound(c.Ctx, "AI 经营分析任务不存在")
		return
	}
	utils.Success(c.Ctx, tenantAnalysisJobResponse(job))
}

func (c *CRMController) ExportTenantAIAnalysisPDF() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := requireTenantAnalysisPermission(c.Ctx, auth, "export"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的 AI 经营分析任务")
		return
	}
	var job models.AIResearchJob
	if err := utils.DB.Where("tenant_id = ? AND id = ? AND object_domain = ? AND object_type = ?", auth.TenantID, id, tenantAnalysisObjectDomain, tenantAnalysisObjectType).First(&job).Error; err != nil {
		utils.NotFound(c.Ctx, "AI 经营分析任务不存在")
		return
	}
	if job.Status != "succeeded" {
		utils.BadRequest(c.Ctx, "只有成功生成的报告可以导出")
		return
	}
	exportCtx := newExportSecurityContext(c.Ctx, auth)
	payload := tenantAnalysisJobResponse(job)
	payload["export_watermark"] = exportCtx.Params()
	logControllerOperation(c.Ctx, auth, auth.TenantID, "EXPORT_TENANT_AI_ANALYSIS", fmt.Sprintf("AI经营分析[%d]", job.ID), "operation.export_tenant_ai_analysis", mergeExportParams(exportCtx, map[string]interface{}{"job_id": job.ID}), "success", "")
	c.Ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
	applyExportTraceHeader(c.Ctx, exportCtx)
	c.Ctx.Output.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"ai-analysis-%d-%s.json\"", job.ID, exportCtx.TraceID))
	_ = c.Ctx.Output.Body([]byte(toJSONString(payload)))
}

func (c *CRMController) TenantAIUsageSummary() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canViewTenantAIUsage(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	wallet, _ := ensureTenantAIWallet(auth.TenantID)
	resetWalletMonthIfNeeded(&wallet)
	var totalCost, totalTokens int64
	query := utils.DB.Model(&models.AIUsageLog{}).Where("tenant_id = ? AND feature = ?", auth.TenantID, tenantBusinessAnalysisFeature)
	_ = query.Select("COALESCE(SUM(cost_micros), 0)").Scan(&totalCost).Error
	_ = query.Select("COALESCE(SUM(total_tokens), 0)").Scan(&totalTokens).Error
	utils.Success(c.Ctx, map[string]interface{}{
		"wallet":            aiWalletResponse(wallet),
		"total_cost_micros": totalCost,
		"total_cost_amount": microsToAmount(totalCost),
		"total_tokens":      totalTokens,
		"currency":          wallet.Currency,
	})
}

func (c *CRMController) TenantAIUsageLedger() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canViewTenantAIUsage(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)
	if size > 100 {
		size = 100
	}
	query := utils.DB.Model(&models.AITokenLedgerEntry{}).Where("tenant_id = ?", auth.TenantID)
	var total int64
	_ = query.Count(&total).Error
	var rows []models.AITokenLedgerEntry
	_ = query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, aiLedgerResponse(row))
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": total})
}

func (c *CRMController) bindTenantAnalysisRequest() (tenantAnalysisRequest, map[string]interface{}, bool) {
	req := tenantAnalysisRequest{PeriodType: "month", Depth: "executive", CompareEnabled: true}
	if !bindJSON(c.Ctx, &req) {
		return req, nil, false
	}
	normalized, err := normalizeTenantAnalysisRequest(req)
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return req, nil, false
	}
	period, err := tenantAnalysisPeriod(normalized)
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return normalized, nil, false
	}
	return normalized, period, true
}

func processTenantAIAnalysisJob(auth authContext, jobID uint, req tenantAnalysisRequest, snapshot map[string]interface{}, prompt string, pricing tenantAnalysisPricing, reservedMicros int64, traceID string, idempotencyHash string, settings aiResearchSettings) {
	started := time.Now()
	_ = utils.DB.Model(&models.AIResearchJob{}).Where("tenant_id = ? AND id = ?", auth.TenantID, jobID).Update("status", "running").Error
	result, usage, err := runTenantBusinessAnalysisAI(prompt, snapshot, req, settings)
	status := "succeeded"
	errorMessage := ""
	if err != nil {
		status = "failed"
		errorMessage = err.Error()
		if usage.TotalTokens == 0 {
			result = fallbackTenantAnalysisReport(snapshot, errorMessage)
		}
	}
	if usage.TotalTokens == 0 {
		usage.PromptTokens = estimateTokens(prompt)
		usage.CompletionTokens = estimateTokens(toJSONString(result))
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		usage.UncachedInputTokens = usage.PromptTokens
		usage.OutputTokens = usage.CompletionTokens
	}
	if usage.OutputTokens == 0 {
		usage.OutputTokens = usage.CompletionTokens
	}
	if usage.UncachedInputTokens == 0 && usage.CachedInputTokens == 0 {
		usage.UncachedInputTokens = usage.PromptTokens
	}
	actualCost := calculateTenantAnalysisCostMicros(usage, pricing)
	effectivePrice := defaultPriceMicrosPM
	if usage.TotalTokens > 0 && actualCost > 0 {
		effectivePrice = int64(math.Ceil(float64(actualCost) * 1000000.0 / float64(usage.TotalTokens)))
	}
	if err := settleTenantAnalysisBilling(auth, jobID, status, errorMessage, result, snapshot, usage, pricing, actualCost, reservedMicros, effectivePrice, started, traceID, idempotencyHash); err != nil {
		now := time.Now()
		_ = utils.DB.Model(&models.AIResearchJob{}).Where("tenant_id = ? AND id = ?", auth.TenantID, jobID).Updates(map[string]interface{}{
			"status":        "billing_failed",
			"error_message": err.Error(),
			"finished_at":   &now,
		}).Error
	}
}

func runTenantBusinessAnalysisAI(prompt string, snapshot map[string]interface{}, req tenantAnalysisRequest, settings aiResearchSettings) (map[string]interface{}, tenantAnalysisUsage, error) {
	if strings.TrimSpace(settings.APIKey) == "" {
		return nil, tenantAnalysisUsage{}, fmt.Errorf("AI API Key 未配置")
	}
	systemPrompt := `你是企业经营分析顾问。只能基于用户提供的CRM结构化快照输出JSON，不允许编造快照外客户、订单、国家、产品或金额。每个关键结论必须引用 evidence_ids。`
	userPrompt := prompt
	modelName := firstNonEmpty(strings.TrimSpace(settings.Model), defaultAIModel)
	if settings.Provider == deepSeekProvider {
		modelName = effectiveDeepSeekModel(modelName)
	}
	body := deepSeekChatRequest{
		Model:       modelName,
		Temperature: 0.2,
		Messages: []deepSeekChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: map[string]string{"type": "json_object"},
	}
	bodyBytes, _ := json.Marshal(body)
	httpReq, err := http.NewRequest(http.MethodPost, aiChatCompletionsEndpoint(settings), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, tenantAnalysisUsage{PromptTokens: estimateTokens(systemPrompt + userPrompt)}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+settings.APIKey)
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(httpReq)
	promptEstimate := estimateTokens(systemPrompt + userPrompt)
	if err != nil {
		return nil, tenantAnalysisUsage{PromptTokens: promptEstimate}, fmt.Errorf("AI 经营分析调用失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, tenantAnalysisUsage{PromptTokens: promptEstimate}, err
	}
	var parsed deepSeekChatResponse
	_ = json.Unmarshal(respBody, &parsed)
	usage := tenantAnalysisUsage{
		PromptTokens:        nonZeroInt64(parsed.Usage.PromptTokens, promptEstimate),
		CompletionTokens:    parsed.Usage.CompletionTokens,
		TotalTokens:         parsed.Usage.TotalTokens,
		CachedInputTokens:   parsed.Usage.PromptCacheHitTokens,
		UncachedInputTokens: parsed.Usage.PromptCacheMissTokens,
		OutputTokens:        parsed.Usage.CompletionTokens,
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	if usage.CachedInputTokens == 0 && usage.UncachedInputTokens == 0 {
		usage.UncachedInputTokens = usage.PromptTokens
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(parsed.Error.Message)
		if message == "" {
			message = string(respBody)
		}
		return nil, usage, fmt.Errorf("AI 经营分析返回错误: %s", trimRunes(message, 200))
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return nil, usage, fmt.Errorf("AI 经营分析未返回有效内容")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if usage.CompletionTokens == 0 {
		usage.CompletionTokens = estimateTokens(content)
		usage.OutputTokens = usage.CompletionTokens
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	result := map[string]interface{}{}
	if err := json.Unmarshal([]byte(extractJSONObject(content)), &result); err != nil {
		return nil, usage, fmt.Errorf("AI 经营分析JSON解析失败: %w", err)
	}
	return normalizeTenantAnalysisReport(result, snapshot), usage, nil
}

func settleTenantAnalysisBilling(auth authContext, jobID uint, status string, errorMessage string, result map[string]interface{}, snapshot map[string]interface{}, usage tenantAnalysisUsage, pricing tenantAnalysisPricing, actualCost int64, reservedMicros int64, effectivePrice int64, started time.Time, traceID string, idempotencyHash string) error {
	return utils.DB.Transaction(func(tx *gorm.DB) error {
		var wallet models.TenantAIWallet
		hasWallet := false
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ?", auth.TenantID).First(&wallet).Error; err == nil {
			hasWallet = true
			currentMonth := time.Now().Format("2006-01")
			if wallet.LastUsageResetMonth != currentMonth {
				wallet.MonthlyTokenUsed = 0
				wallet.LastUsageResetMonth = currentMonth
			}
			wallet.MonthlyTokenUsed += usage.TotalTokens
			if err := tx.Save(&wallet).Error; err != nil {
				return err
			}
		}
		beforeBalance := int64(0)
		beforeReserved := int64(0)
		afterBalance := int64(0)
		afterReserved := int64(0)
		if hasWallet {
			beforeBalance = wallet.BalanceMicros
			beforeReserved = wallet.ReservedMicros
			afterBalance = wallet.BalanceMicros
			afterReserved = wallet.ReservedMicros
		}
		release := reservedMicros - actualCost
		if release < 0 {
			release = 0
		}
		extra := actualCost - reservedMicros
		if extra < 0 {
			extra = 0
		}
		usageLog := models.AIUsageLog{
			TenantModel:               models.TenantModel{TenantID: auth.TenantID},
			UserID:                    auth.UserID,
			Feature:                   tenantBusinessAnalysisFeature,
			ObjectDomain:              tenantAnalysisObjectDomain,
			ObjectType:                tenantAnalysisObjectType,
			ObjectID:                  jobID,
			Provider:                  pricing.Provider,
			Model:                     pricing.ModelName,
			PromptTokens:              usage.PromptTokens,
			CompletionTokens:          usage.CompletionTokens,
			TotalTokens:               usage.TotalTokens,
			UnitPriceMicrosPerMillion: effectivePrice,
			CostMicros:                actualCost,
			Currency:                  pricing.Currency,
			Status:                    status,
			TraceID:                   traceID,
			LatencyMS:                 time.Since(started).Milliseconds(),
			ErrorMessage:              errorMessage,
			MetadataJSON: toJSONString(map[string]interface{}{
				"cached_input_tokens":   usage.CachedInputTokens,
				"uncached_input_tokens": usage.UncachedInputTokens,
				"output_tokens":         usage.OutputTokens,
				"pricing":               tenantAnalysisPriceSnapshot(pricing),
				"reserved_micros":       reservedMicros,
				"released_micros":       release,
				"extra_micros":          extra,
				"billing_mode":          "tenant_own_api_key",
			}),
		}
		if err := tx.Create(&usageLog).Error; err != nil {
			return err
		}
		consume := models.AITokenLedgerEntry{
			TenantModel:          models.TenantModel{TenantID: auth.TenantID},
			UserID:               auth.UserID,
			UsageLogID:           usageLog.ID,
			JobID:                jobID,
			EntryType:            "consume",
			Direction:            "debit",
			AmountMicros:         actualCost,
			PromptTokens:         usage.PromptTokens,
			CompletionTokens:     usage.CompletionTokens,
			CachedInputTokens:    usage.CachedInputTokens,
			UncachedInputTokens:  usage.UncachedInputTokens,
			OutputTokens:         usage.OutputTokens,
			TotalTokens:          usage.TotalTokens,
			PriceSnapshotJSON:    toJSONString(tenantAnalysisPriceSnapshot(pricing)),
			BalanceBeforeMicros:  beforeBalance,
			BalanceAfterMicros:   afterBalance,
			ReservedBeforeMicros: beforeReserved,
			ReservedAfterMicros:  afterReserved,
			Status:               status,
			TraceID:              traceID,
			IdempotencyKey:       idempotencyHash,
			Description:          "AI 经营分析实际消费",
		}
		if err := tx.Create(&consume).Error; err != nil {
			return err
		}
		if release > 0 {
			entry := consume
			entry.ID = 0
			entry.CreatedAt = time.Time{}
			entry.UpdatedAt = time.Time{}
			entry.EntryType = "release"
			entry.Direction = "credit"
			entry.AmountMicros = release
			entry.Description = "AI 经营分析释放预授权差额"
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}
		now := time.Now()
		return tx.Model(&models.AIResearchJob{}).Where("tenant_id = ? AND id = ?", auth.TenantID, jobID).Updates(map[string]interface{}{
			"status":        status,
			"result_json":   toJSONString(result),
			"source_json":   toJSONString(snapshot),
			"confidence":    numericValue(result["confidence"]),
			"usage_log_id":  usageLog.ID,
			"error_message": errorMessage,
			"finished_at":   &now,
		}).Error
	})
}

func normalizeTenantAnalysisRequest(req tenantAnalysisRequest) (tenantAnalysisRequest, error) {
	req.PeriodType = strings.ToLower(strings.TrimSpace(req.PeriodType))
	if req.PeriodType == "" {
		req.PeriodType = "month"
	}
	if req.PeriodType != "month" && req.PeriodType != "quarter" && req.PeriodType != "year" && req.PeriodType != "custom" {
		return req, fmt.Errorf("无效的分析周期")
	}
	req.Depth = strings.ToLower(strings.TrimSpace(req.Depth))
	if req.Depth == "" {
		req.Depth = "executive"
	}
	if req.Depth != "executive" && req.Depth != "deep" {
		return req, fmt.Errorf("无效的分析深度")
	}
	if len(req.Sections) == 0 {
		req.Sections = []string{"overview", "customers", "sales_pipeline", "revenue_orders", "products_projects", "service_delivery", "team_execution"}
	}
	sections := []string{}
	seen := map[string]bool{}
	for _, section := range req.Sections {
		key := strings.ToLower(strings.TrimSpace(section))
		if !tenantAnalysisSectionSet[key] {
			return req, fmt.Errorf("无效的分析模块: %s", section)
		}
		if !seen[key] {
			seen[key] = true
			sections = append(sections, key)
		}
	}
	req.Sections = sections
	return req, nil
}

func tenantAnalysisPeriod(req tenantAnalysisRequest) (map[string]interface{}, error) {
	now := time.Now()
	var start, end time.Time
	switch req.PeriodType {
	case "month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
	case "quarter":
		month := time.Month(((int(now.Month())-1)/3)*3 + 1)
		start = time.Date(now.Year(), month, 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 3, 0)
	case "year":
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(1, 0, 0)
	case "custom":
		var ok bool
		start, ok = parseDateBoundary(req.StartDate, false)
		if !ok {
			return nil, fmt.Errorf("自定义周期需要开始日期")
		}
		end, ok = parseDateBoundary(req.EndDate, true)
		if !ok {
			return nil, fmt.Errorf("自定义周期需要结束日期")
		}
	}
	if req.StartDate != "" && req.PeriodType != "custom" {
		if parsed, ok := parseDateBoundary(req.StartDate, false); ok {
			start = parsed
		}
	}
	if req.EndDate != "" && req.PeriodType != "custom" {
		if parsed, ok := parseDateBoundary(req.EndDate, true); ok {
			end = parsed
		}
	}
	if !end.After(start) {
		return nil, fmt.Errorf("结束日期必须晚于开始日期")
	}
	if end.Sub(start) > 366*24*time.Hour {
		return nil, fmt.Errorf("分析周期不能超过 1 年")
	}
	return map[string]interface{}{
		"period_type": req.PeriodType,
		"start":       start.Format(time.RFC3339),
		"end":         end.Format(time.RFC3339),
		"label":       start.Format("2006-01-02") + " - " + end.Add(-time.Second).Format("2006-01-02"),
	}, nil
}

func buildTenantAnalysisSnapshot(auth authContext, req tenantAnalysisRequest, period map[string]interface{}) map[string]interface{} {
	if cached, ok := cachedTenantAnalysisSnapshot(auth, req, period); ok {
		return cached
	}
	start, _ := time.Parse(time.RFC3339, fmt.Sprint(period["start"]))
	end, _ := time.Parse(time.RFC3339, fmt.Sprint(period["end"]))
	ctx, cancel := context.WithTimeout(context.Background(), tenantAnalysisSnapshotTimeout)
	defer cancel()
	snapshot := map[string]interface{}{
		"tenant_id":    auth.TenantID,
		"generated_at": time.Now().Format(time.RFC3339),
		"period":       period,
		"sections":     req.Sections,
		"evidence":     []map[string]interface{}{},
		"snapshot_policy": map[string]interface{}{
			"mode":           "bounded_aggregate",
			"sample_limit":   tenantAnalysisSampleLimit,
			"evidence_limit": tenantAnalysisEvidenceLimit,
			"timeout_ms":     tenantAnalysisSnapshotTimeout.Milliseconds(),
			"notes":          []string{"经营分析使用聚合指标和限量样本，避免一次性读取全租户明细数据。"},
		},
	}
	evidence := []map[string]interface{}{}
	addEvidence := func(domain string, objectType string, id uint, name string, metric string) string {
		if id == 0 && strings.TrimSpace(name) == "" {
			return ""
		}
		if len(evidence) >= tenantAnalysisEvidenceLimit {
			return ""
		}
		key := fmt.Sprintf("ev_%s_%d_%d", domain, id, len(evidence)+1)
		evidence = append(evidence, map[string]interface{}{"id": key, "domain": domain, "object_type": objectType, "object_id": id, "name": name, "metric": metric})
		return key
	}
	domains := []string{"contacts", "leads", "opportunities", "deals", "orders", "products", "projects", "tickets", "campaigns"}
	counts := map[string]int64{}
	truncated := map[string]interface{}{}
	for _, domain := range domains {
		var count int64
		countCtx, countCancel := context.WithTimeout(ctx, tenantAnalysisCountTimeout)
		_ = crmScopedObjectQuery(auth, domain).WithContext(countCtx).Where("created_at >= ? AND created_at < ?", start, end).Count(&count).Error
		countCancel()
		counts[domain] = count
		if count > tenantAnalysisSampleLimit {
			truncated[domain] = map[string]interface{}{"total": count, "sample_limit": tenantAnalysisSampleLimit}
		}
	}
	agg := boundedSalesOrderAggregateForRange(ctx, auth.TenantID, start, end, tenantAnalysisSampleLimit)
	if agg.Sampled || agg.TotalOrders > agg.ProcessedOrders {
		truncated["orders"] = map[string]interface{}{"total": agg.TotalOrders, "processed": agg.ProcessedOrders, "sample_limit": tenantAnalysisSampleLimit}
	}
	topCountries := topDimensionRows(agg.Countries, tenantAnalysisTopLimit, addEvidence, "orders", "order", "国家销售贡献")
	topCustomers := topDimensionRows(agg.Customers, tenantAnalysisTopLimit, addEvidence, "contacts", "company", "客户销售贡献")
	topProducts := topDimensionRows(agg.Products, tenantAnalysisTopLimit, addEvidence, "products", "product", "产品销售贡献")
	var opportunities []models.CRMObject
	_ = crmScopedObjectQuery(auth, "opportunities").WithContext(ctx).
		Select("id, tenant_id, domain, object_type, name, status, amount, payload_json, created_at").
		Where("object_type = ? AND created_at >= ? AND created_at < ?", "opportunity", start, end).
		Order("amount desc").
		Limit(tenantAnalysisOpportunityLimit).
		Find(&opportunities).Error
	pipelineAmount := 0.0
	totalOpportunityAmount := 0.0
	openOpps := 0
	stageSummary := map[string]map[string]interface{}{}
	for _, opp := range opportunities {
		totalOpportunityAmount += opp.Amount
		stage := strings.TrimSpace(opp.Status)
		if stage == "" {
			stage = "unknown"
		}
		row := stageSummary[stage]
		if row == nil {
			row = map[string]interface{}{"stage": stage, "count": int64(0), "amount": float64(0)}
			stageSummary[stage] = row
		}
		row["count"] = int64(numericValue(row["count"])) + 1
		row["amount"] = numericValue(row["amount"]) + opp.Amount
		if opp.Status != "won" && opp.Status != "lost" {
			openOpps++
			pipelineAmount += opp.Amount
		}
		addEvidence("opportunities", "opportunity", opp.ID, opp.Name, "商机管道")
	}
	stageRows := opportunityStageRows(stageSummary)
	var tickets []models.CRMObject
	_ = crmScopedObjectQuery(auth, "tickets").WithContext(ctx).
		Select("id, tenant_id, domain, object_type, name, status, payload_json, created_at").
		Where("object_type = ? AND created_at >= ? AND created_at < ?", "ticket", start, end).
		Order("updated_at desc").
		Limit(tenantAnalysisTicketLimit).
		Find(&tickets).Error
	overdueTickets := 0
	openTickets := 0
	for _, ticket := range tickets {
		payload := payloadMap(ticket.PayloadJSON)
		if ticket.Status != "closed" && ticket.Status != "resolved" {
			openTickets++
		}
		dueAt, _ := parseOptionalTime(firstPayloadText(payload, "sla_due_at"))
		if dueAt != nil && dueAt.Before(time.Now()) && ticket.Status != "closed" && ticket.Status != "resolved" {
			overdueTickets++
			addEvidence("tickets", "ticket", ticket.ID, ticket.Name, "SLA超期")
		}
	}
	if int(counts["opportunities"]) > tenantAnalysisOpportunityLimit {
		truncated["opportunities"] = map[string]interface{}{"total": counts["opportunities"], "sample_limit": tenantAnalysisOpportunityLimit}
	}
	if int(counts["tickets"]) > tenantAnalysisTicketLimit {
		truncated["tickets"] = map[string]interface{}{"total": counts["tickets"], "sample_limit": tenantAnalysisTicketLimit}
	}
	snapshot["overview"] = map[string]interface{}{
		"counts":                   counts,
		"order_amount":             agg.OrderAmount,
		"paid_amount":              agg.PaidAmount,
		"order_count":              agg.OrderCount,
		"pipeline_amount":          pipelineAmount,
		"open_pipeline_amount":     pipelineAmount,
		"total_opportunity_amount": totalOpportunityAmount,
		"open_opportunities":       openOpps,
		"open_tickets":             openTickets,
		"overdue_tickets":          overdueTickets,
	}
	snapshot["customers"] = map[string]interface{}{"top_customers": topCustomers, "top_countries": topCountries, "health": customerHealth(auth)}
	snapshot["sales_pipeline"] = map[string]interface{}{
		"funnel":                   stageRows,
		"all_time_funnel":          salesFunnel(auth),
		"forecast":                 salesForecast(auth),
		"loss_analysis":            lossAnalysis(auth),
		"pipeline_amount":          pipelineAmount,
		"open_pipeline_amount":     pipelineAmount,
		"total_opportunity_amount": totalOpportunityAmount,
		"amount_definition":        "open_pipeline_amount只包含当前报告周期内未赢单且未丢单的开放商机金额；total_opportunity_amount才是当前报告周期内全部商机金额；won/lost阶段金额不得加回open_pipeline_amount。",
	}
	snapshot["revenue_orders"] = map[string]interface{}{"order_amount": agg.OrderAmount, "paid_amount": agg.PaidAmount, "order_count": agg.OrderCount, "top_countries": topCountries, "top_customers": topCustomers}
	snapshot["products_projects"] = map[string]interface{}{"top_products": topProducts, "project_count": counts["projects"]}
	snapshot["service_delivery"] = map[string]interface{}{"service_quality": serviceQuality(auth), "open_tickets": openTickets, "overdue_tickets": overdueTickets}
	snapshot["team_execution"] = map[string]interface{}{"sales_targets": salesTargetSummaryPayload(auth, time.Now().Year())}
	snapshot["truncated"] = truncated
	snapshot["evidence"] = evidence
	cacheTenantAnalysisSnapshot(auth, req, period, snapshot)
	return snapshot
}

func tenantAnalysisSnapshotCacheKey(auth authContext, req tenantAnalysisRequest, period map[string]interface{}) string {
	payload := toJSONString(map[string]interface{}{
		"tenant_id":    auth.TenantID,
		"user_id":      auth.UserID,
		"role":         auth.Role,
		"account_type": auth.AccountType,
		"period":       period,
		"sections":     req.Sections,
		"depth":        req.Depth,
		"model":        currentDeepSeekModel(),
		"policy":       "bounded_v1",
		"sample":       tenantAnalysisSampleLimit,
		"top":          tenantAnalysisTopLimit,
		"evidence":     tenantAnalysisEvidenceLimit,
	})
	sum := sha1.Sum([]byte(payload))
	return fmt.Sprintf("ai_crm:tenant_ai_analysis:snapshot:%d:%x", auth.TenantID, sum[:])
}

func cachedTenantAnalysisSnapshot(auth authContext, req tenantAnalysisRequest, period map[string]interface{}) (map[string]interface{}, bool) {
	if utils.RedisClient == nil {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	raw, err := utils.RedisClient.Get(ctx, tenantAnalysisSnapshotCacheKey(auth, req, period)).Result()
	if err != nil {
		if err != redis.Nil {
			utils.LogWarning("AI经营分析快照缓存读取失败: %v", err)
		}
		return nil, false
	}
	snapshot := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return nil, false
	}
	policy := mapValue(snapshot["snapshot_policy"])
	policy["cache_hit"] = true
	policy["cache_ttl_seconds"] = int64(tenantAnalysisSnapshotCacheTTL.Seconds())
	snapshot["snapshot_policy"] = policy
	return snapshot, true
}

func cacheTenantAnalysisSnapshot(auth authContext, req tenantAnalysisRequest, period map[string]interface{}, snapshot map[string]interface{}) {
	if utils.RedisClient == nil || len(snapshot) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	policy := mapValue(snapshot["snapshot_policy"])
	policy["cache_hit"] = false
	policy["cache_ttl_seconds"] = int64(tenantAnalysisSnapshotCacheTTL.Seconds())
	snapshot["snapshot_policy"] = policy
	raw := toJSONString(snapshot)
	if err := utils.RedisClient.Set(ctx, tenantAnalysisSnapshotCacheKey(auth, req, period), raw, tenantAnalysisSnapshotCacheTTL).Err(); err != nil {
		utils.LogWarning("AI经营分析快照缓存写入失败: %v", err)
	}
}

func buildTenantAnalysisPrompt(snapshot map[string]interface{}, req tenantAnalysisRequest) string {
	payload := map[string]interface{}{
		"snapshot": snapshot,
		"schema": map[string]interface{}{
			"executive_summary":    "string",
			"key_metrics":          "array of {name,value,comment,evidence_ids[]}",
			"customer_insights":    "array of {title,summary,evidence_ids[]}",
			"sales_pipeline":       "array of {title,summary,evidence_ids[]}",
			"revenue_and_orders":   "array of {title,summary,evidence_ids[]}",
			"service_and_delivery": "array of {title,summary,evidence_ids[]}",
			"team_execution":       "array of {title,summary,evidence_ids[]}",
			"risks":                "array of {level,title,impact,evidence_ids[]}",
			"opportunities":        "array of {title,summary,evidence_ids[]}",
			"recommended_actions":  "array of {priority,action,owner_suggestion,due,evidence_ids[]}",
			"follow_up_questions":  "array of strings",
			"confidence":           "number 0-1",
		},
		"instructions": []string{
			"只输出JSON，不要Markdown。",
			"所有金额和数量必须来自snapshot。",
			"销售管道金额口径必须严格区分：open_pipeline_amount是不含won/lost的开放商机金额，total_opportunity_amount是全部商机金额，不能把open_pipeline_amount再加won/lost阶段金额当成管道总额。",
			"描述销售漏斗阶段金额时，阶段金额合计必须等于同一口径的total_opportunity_amount；描述开放管道时只能使用open_pipeline_amount。",
			"关键结论必须引用snapshot.evidence中的id。",
			"没有证据时要明确写数据不足，不得编造。",
		},
		"depth": req.Depth,
	}
	return "请基于以下CRM快照生成企业经营分析报告JSON：\n" + toJSONString(payload)
}

func fallbackTenantAnalysisReport(snapshot map[string]interface{}, reason string) map[string]interface{} {
	overview := mapValue(snapshot["overview"])
	return map[string]interface{}{
		"executive_summary": fmt.Sprintf("已基于当前 CRM 结构化数据生成经营概览。AI 深度总结未完成：%s", reason),
		"key_metrics": []map[string]interface{}{
			{"name": "订单金额", "value": overview["order_amount"], "comment": "统计周期内有效订单金额"},
			{"name": "回款金额", "value": overview["paid_amount"], "comment": "统计周期内已回款金额"},
			{"name": "订单数", "value": overview["order_count"], "comment": "统计周期内有效订单数量"},
		},
		"risks":               []map[string]interface{}{},
		"opportunities":       []map[string]interface{}{},
		"recommended_actions": []map[string]interface{}{},
		"follow_up_questions": []string{"建议补充客户、订单、服务和团队目标数据后重新生成深度报告。"},
		"confidence":          0.55,
	}
}

func normalizeTenantAnalysisReport(result map[string]interface{}, snapshot map[string]interface{}) map[string]interface{} {
	if strings.TrimSpace(fmt.Sprint(result["executive_summary"])) == "" {
		result["executive_summary"] = "AI 已生成经营分析报告。"
	}
	result["sales_pipeline"] = prependTenantPipelineSummary(result["sales_pipeline"], snapshot)
	if numericValue(result["confidence"]) <= 0 {
		result["confidence"] = 0.75
	}
	result["evidence"] = snapshot["evidence"]
	return result
}

func prependTenantPipelineSummary(value interface{}, snapshot map[string]interface{}) []map[string]interface{} {
	sales := mapValue(snapshot["sales_pipeline"])
	openAmount := numericValue(firstNonNil(sales["open_pipeline_amount"], sales["pipeline_amount"]))
	totalAmount := numericValue(sales["total_opportunity_amount"])
	stageRows := mapsFromInterface(sales["funnel"])
	stageText := []string{}
	for _, row := range stageRows {
		count := int64(numericValue(row["count"]))
		amount := numericValue(row["amount"])
		if count == 0 && amount == 0 {
			continue
		}
		stageText = append(stageText, fmt.Sprintf("%s %s（%d个商机）", fmt.Sprint(row["stage"]), formatFloatAmount(amount), count))
	}
	summary := map[string]interface{}{
		"title":   "销售管道口径校验",
		"summary": fmt.Sprintf("当前报告周期开放管道金额为%s，只包含未赢单且未丢单的开放商机；全部商机金额为%s。阶段分布：%s。", formatFloatAmount(openAmount), formatFloatAmount(totalAmount), strings.Join(stageText, "，")),
	}
	items := []map[string]interface{}{summary}
	for _, item := range mapsFromInterface(value) {
		items = append(items, item)
	}
	return items
}

func mapsFromInterface(value interface{}) []map[string]interface{} {
	items := []map[string]interface{}{}
	switch typed := value.(type) {
	case []map[string]interface{}:
		return typed
	case []interface{}:
		for _, item := range typed {
			if mapped := mapValue(item); len(mapped) > 0 {
				items = append(items, mapped)
			}
		}
	}
	return items
}

func formatFloatAmount(value float64) string {
	return fmt.Sprintf("%.0f", value)
}

func tenantAnalysisPricingForModel(modelName string) tenantAnalysisPricing {
	snapshot := ensureAIPricingSnapshot(0)
	model := pricingModel(snapshot, modelName)
	hit := pricingRate(model, "input_cache_hit")
	miss := pricingRate(model, "input_cache_miss")
	output := pricingRate(model, "output")
	if miss.TenantPriceMicrosPerMillion <= 0 {
		miss.TenantPriceMicrosPerMillion = defaultPriceMicrosPM
		miss.TenantPriceAmount = microsToAmount(defaultPriceMicrosPM)
	}
	if hit.TenantPriceMicrosPerMillion <= 0 {
		hit = miss
		hit.Metric = "input_cache_hit"
	}
	if output.TenantPriceMicrosPerMillion <= 0 {
		output = miss
		output.Metric = "output"
	}
	return tenantAnalysisPricing{Snapshot: snapshot, Model: model, InputHitRate: hit, InputMissRate: miss, OutputRate: output, Provider: snapshot.Provider, ModelName: model.Model, Currency: snapshot.Currency}
}

func calculateTenantAnalysisCostMicros(usage tenantAnalysisUsage, pricing tenantAnalysisPricing) int64 {
	cached := usage.CachedInputTokens
	uncached := usage.UncachedInputTokens
	if cached == 0 && uncached == 0 {
		uncached = usage.PromptTokens
	}
	output := usage.OutputTokens
	if output == 0 {
		output = usage.CompletionTokens
	}
	return calculateAICostMicros(cached, pricing.InputHitRate.TenantPriceMicrosPerMillion) +
		calculateAICostMicros(uncached, pricing.InputMissRate.TenantPriceMicrosPerMillion) +
		calculateAICostMicros(output, pricing.OutputRate.TenantPriceMicrosPerMillion)
}

func tenantAnalysisReserveCost(estimatedCost int64, totalTokens int64) int64 {
	reserveCost := int64(math.Ceil(float64(estimatedCost*tenantAnalysisReserveFactor) / float64(tenantAnalysisReserveDivisor)))
	if reserveCost <= 0 {
		reserveCost = calculateAICostMicros(totalTokens, defaultPriceMicrosPM)
	}
	return reserveCost
}

func tenantAnalysisPriceSnapshot(pricing tenantAnalysisPricing) map[string]interface{} {
	return map[string]interface{}{
		"provider":                      pricing.Provider,
		"model":                         pricing.ModelName,
		"pricing_model":                 pricing.Model.Model,
		"currency":                      pricing.Currency,
		"source_url":                    pricing.Snapshot.SourceURL,
		"source_status":                 pricing.Snapshot.SourceStatus,
		"synced_at":                     pricing.Snapshot.SyncedAt,
		"margin_rate":                   pricing.Snapshot.MarginRate,
		"cached_input_price_micros":     pricing.InputHitRate.TenantPriceMicrosPerMillion,
		"uncached_input_price_micros":   pricing.InputMissRate.TenantPriceMicrosPerMillion,
		"output_price_micros":           pricing.OutputRate.TenantPriceMicrosPerMillion,
		"official_cached_input_price":   pricing.InputHitRate.OfficialPriceAmount,
		"official_uncached_input_price": pricing.InputMissRate.OfficialPriceAmount,
		"official_output_price":         pricing.OutputRate.OfficialPriceAmount,
		"tenant_cached_input_price":     pricing.InputHitRate.TenantPriceAmount,
		"tenant_uncached_input_price":   pricing.InputMissRate.TenantPriceAmount,
		"tenant_output_price":           pricing.OutputRate.TenantPriceAmount,
	}
}

func tenantAnalysisPricingForSettings(settings aiResearchSettings) tenantAnalysisPricing {
	if settings.Provider == deepSeekProvider {
		return tenantAnalysisPricingForModel(settings.Model)
	}
	rate := aiPricingRate{
		Metric:                      "input_output",
		Label:                       "输入/输出 token",
		Currency:                    defaultAICurrency,
		TenantPriceMicrosPerMillion: 0,
		TenantPriceAmount:           microsToAmount(0),
	}
	model := aiPricingModel{Provider: settings.Provider, Model: settings.Model, DisplayName: settings.Model, Rates: []aiPricingRate{rate}}
	return tenantAnalysisPricing{
		Snapshot: aiPricingSnapshot{
			Provider:     settings.Provider,
			Currency:     defaultAICurrency,
			SourceStatus: "tenant_configured",
			SyncedAt:     time.Now().Format(time.RFC3339),
			Models:       []aiPricingModel{model},
		},
		Model:         model,
		InputHitRate:  rate,
		InputMissRate: rate,
		OutputRate:    rate,
		Provider:      settings.Provider,
		ModelName:     settings.Model,
		Currency:      defaultAICurrency,
	}
}

type boundedSalesOrderAggregate struct {
	salesOrderAggregate
	TotalOrders      int64 `json:"total_orders"`
	ProcessedOrders  int64 `json:"processed_orders"`
	Sampled          bool  `json:"sampled"`
	ProcessingFailed bool  `json:"processing_failed"`
}

func boundedSalesOrderAggregateForRange(ctx context.Context, tenantID uint, start, end time.Time, limit int) boundedSalesOrderAggregate {
	agg := boundedSalesOrderAggregate{
		salesOrderAggregate: salesOrderAggregate{
			Owners:    map[uint]*ownerSalesMetrics{},
			Countries: map[string]*salesDimensionMetric{},
			Customers: map[string]*salesDimensionMetric{},
			Products:  map[string]*salesDimensionMetric{},
		},
	}
	if tenantID == 0 || start.IsZero() || end.IsZero() || !end.After(start) {
		return agg
	}
	if limit <= 0 {
		limit = tenantAnalysisSampleLimit
	}
	base := utils.DB.WithContext(ctx).Model(&models.CRMObject{}).
		Where("tenant_id = ? AND domain = ? AND object_type = ? AND status IN ? AND created_at >= ? AND created_at < ?", tenantID, "orders", "order", salesOrderAmountStatuses(), start, end)
	if err := base.Count(&agg.TotalOrders).Error; err != nil {
		agg.ProcessingFailed = true
		return agg
	}
	var orders []models.CRMObject
	if err := base.
		Select("id, tenant_id, domain, object_type, name, status, owner_id, company_id, contact_id, deal_id, amount, payload_json, created_at, created_by").
		Order("amount desc, created_at desc").
		Limit(limit).
		Find(&orders).Error; err != nil {
		agg.ProcessingFailed = true
		return agg
	}
	agg.ProcessedOrders = int64(len(orders))
	agg.Sampled = agg.TotalOrders > agg.ProcessedOrders
	companyNames, companyPayloads, productNames := scopedOrderLookupMaps(ctx, tenantID, orders)
	for _, order := range orders {
		payload := payloadMap(order.PayloadJSON)
		companyID := firstPositiveUint(order.CompanyID, uintFromValue(payload["company_id"]))
		chainPayloads := []map[string]interface{}{}
		if companyID == 0 || strings.TrimSpace(companyNames[companyID]) == "" || customerCountryFromPayload(companyPayloads[companyID]) == "" {
			for _, object := range salesOrderRelatedOwners(tenantID, order) {
				objectPayload := payloadMap(object.PayloadJSON)
				chainPayloads = append(chainPayloads, objectPayload)
				companyID = firstPositiveUint(companyID, object.CompanyID, uintFromValue(objectPayload["company_id"]))
			}
		}
		ownerID := salesOrderAttributionOwnerID(tenantID, order, payload)
		paidInRange := orderPaidAmountInRange(order, payload, start, end, true)
		agg.PaidAmount += paidInRange
		agg.OrderAmount += order.Amount
		agg.OrderCount++
		if ownerID > 0 {
			owner := ensureOwnerSalesMetrics(agg.Owners, ownerID)
			owner.OrderAmount += order.Amount
			owner.PaidAmount += paidInRange
			owner.OrderCount++
		} else {
			agg.Unassigned.OrderAmount += order.Amount
			agg.Unassigned.PaidAmount += paidInRange
			agg.Unassigned.OrderCount++
		}
		country := salesOrderCountry(payload, companyPayloads[companyID], chainPayloads...)
		countryBucket := ensureDimensionMetric(agg.Countries, normalizeDimensionKey(country), country)
		countryBucket.Amount += order.Amount
		countryBucket.PaidAmount += paidInRange
		countryBucket.Count++

		customerName := salesOrderCustomerName(order, payload, companyNames, chainPayloads...)
		customerKey := fmt.Sprintf("company:%d", companyID)
		if companyID == 0 {
			customerKey = normalizeDimensionKey(customerName)
		}
		customerBucket := ensureDimensionMetric(agg.Customers, customerKey, customerName)
		customerBucket.CompanyID = companyID
		customerBucket.Amount += order.Amount
		customerBucket.PaidAmount += paidInRange
		customerBucket.Count++

		for _, raw := range interfaceList(payload["line_items"]) {
			line, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			productID := uintFromValue(line["product_id"])
			productName := strings.TrimSpace(firstPayloadText(line, "product_name", "name", "sku"))
			if productName == "" && productID > 0 {
				productName = productNames[productID]
			}
			if productName == "" {
				productName = "未命名产品"
			}
			quantity := numericValue(line["quantity"])
			if quantity <= 0 {
				quantity = 1
			}
			amount := numericValue(line["amount"])
			if amount <= 0 {
				price := numericValue(line["unit_price"])
				if price <= 0 {
					price = numericValue(line["price"])
				}
				amount = quantity * price
			}
			if amount <= 0 {
				continue
			}
			productKey := fmt.Sprintf("product:%d", productID)
			if productID == 0 {
				productKey = normalizeDimensionKey(productName)
			}
			productBucket := ensureDimensionMetric(agg.Products, productKey, productName)
			productBucket.ProductID = productID
			productBucket.Amount += amount
			productBucket.Quantity += quantity
			productBucket.Count++
		}
	}
	return agg
}

func scopedOrderLookupMaps(ctx context.Context, tenantID uint, orders []models.CRMObject) (map[uint]string, map[uint]map[string]interface{}, map[uint]string) {
	companyIDs := map[uint]bool{}
	productIDs := map[uint]bool{}
	for _, order := range orders {
		payload := payloadMap(order.PayloadJSON)
		companyID := firstPositiveUint(order.CompanyID, uintFromValue(payload["company_id"]))
		if companyID > 0 {
			companyIDs[companyID] = true
		}
		for _, raw := range interfaceList(payload["line_items"]) {
			line, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			if productID := uintFromValue(line["product_id"]); productID > 0 {
				productIDs[productID] = true
			}
		}
	}
	companyNames := map[uint]string{}
	companyPayloads := map[uint]map[string]interface{}{}
	productNames := map[uint]string{}
	if len(companyIDs) > 0 {
		ids := make([]uint, 0, len(companyIDs))
		for id := range companyIDs {
			ids = append(ids, id)
		}
		var companies []models.CRMObject
		_ = utils.DB.WithContext(ctx).
			Select("id, name, payload_json").
			Where("tenant_id = ? AND domain = ? AND object_type = ? AND id IN ?", tenantID, "contacts", "company", ids).
			Limit(len(ids)).
			Find(&companies).Error
		for _, company := range companies {
			companyNames[company.ID] = company.Name
			companyPayloads[company.ID] = payloadMap(company.PayloadJSON)
		}
	}
	if len(productIDs) > 0 {
		ids := make([]uint, 0, len(productIDs))
		for id := range productIDs {
			ids = append(ids, id)
		}
		var products []models.CRMObject
		_ = utils.DB.WithContext(ctx).
			Select("id, name").
			Where("tenant_id = ? AND domain = ? AND object_type = ? AND id IN ?", tenantID, "products", "product", ids).
			Limit(len(ids)).
			Find(&products).Error
		for _, product := range products {
			productNames[product.ID] = product.Name
		}
	}
	return companyNames, companyPayloads, productNames
}

func estimatedTenantAnalysisCompletionTokens(depth string) int64 {
	if depth == "deep" {
		return 6000
	}
	return 3000
}

func opportunityStageRows(stageSummary map[string]map[string]interface{}) []map[string]interface{} {
	order := []string{"initial_contact", "need_confirmed", "proposal", "negotiation", "won", "lost", "on_hold", "unknown"}
	rows := make([]map[string]interface{}, 0, len(stageSummary))
	seen := map[string]bool{}
	for _, stage := range order {
		if row := stageSummary[stage]; row != nil {
			rows = append(rows, row)
			seen[stage] = true
		}
	}
	extra := make([]string, 0)
	for stage := range stageSummary {
		if !seen[stage] {
			extra = append(extra, stage)
		}
	}
	sort.Strings(extra)
	for _, stage := range extra {
		rows = append(rows, stageSummary[stage])
	}
	return rows
}

func requireTenantAnalysisPermission(_ interface{}, auth authContext, action string) error {
	if auth.IsTenantAdmin() {
		return nil
	}
	switch action {
	case "create":
		if authHasPermissionAny(auth, "insights:create", "insights:manage") {
			return nil
		}
	case "view":
		if authHasPermissionAny(auth, "insights:view", "insights:manage") {
			return nil
		}
	case "export":
		if authHasPermissionAny(auth, "insights:export", "insights:manage") {
			return nil
		}
	}
	return fmt.Errorf("无权限访问")
}

func canViewTenantAIUsage(auth authContext) bool {
	return auth.IsTenantAdmin() || authHasPermissionAny(auth, "insights:manage", "ai_usage:view", "ai_usage:manage")
}

func hasRunningTenantAnalysis(tenantID uint, userID uint) bool {
	query := utils.DB.Model(&models.AIResearchJob{}).Where("tenant_id = ? AND object_domain = ? AND object_type = ? AND status IN ?", tenantID, tenantAnalysisObjectDomain, tenantAnalysisObjectType, []string{"pending", "running"})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	var count int64
	_ = query.Count(&count).Error
	return count > 0
}

var tenantAnalysisRateBuckets = struct {
	items map[string][]time.Time
}{items: map[string][]time.Time{}}

func allowTenantAnalysisRate(tenantID uint, userID uint, ip string, action string, tenantLimit int, userLimit int) bool {
	now := time.Now()
	window := now.Add(-time.Hour)
	tenantKey := fmt.Sprintf("tenant:%d:%s", tenantID, action)
	userKey := fmt.Sprintf("user:%d:%d:%s", tenantID, userID, action)
	ipKey := fmt.Sprintf("ip:%s:%s", ip, action)
	return allowRateKey(tenantKey, tenantLimit, window, now) && allowRateKey(userKey, userLimit, window, now) && allowRateKey(ipKey, userLimit*3, window, now)
}

func allowRateKey(key string, limit int, window time.Time, now time.Time) bool {
	if limit <= 0 {
		return true
	}
	values := tenantAnalysisRateBuckets.items[key]
	next := []time.Time{}
	for _, value := range values {
		if value.After(window) {
			next = append(next, value)
		}
	}
	if len(next) >= limit {
		tenantAnalysisRateBuckets.items[key] = next
		return false
	}
	next = append(next, now)
	tenantAnalysisRateBuckets.items[key] = next
	return true
}

func tenantAnalysisIdempotencyHash(tenantID uint, userID uint, key string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%d:%d:%s", tenantID, userID, key)))
	return hex.EncodeToString(sum[:])
}

func newTenantAnalysisTraceID(tenantID uint, userID uint) string {
	return fmt.Sprintf("tenant_analysis_%d_%d_%d", tenantID, userID, time.Now().UnixNano())
}

func recordBlockedTenantAnalysisUsage(auth authContext, jobID uint, reason string, idempotencyKey string) {
	settings := resolveAIResearchSettings()
	log := models.AIUsageLog{
		TenantModel:  models.TenantModel{TenantID: auth.TenantID},
		UserID:       auth.UserID,
		Feature:      tenantBusinessAnalysisFeature,
		ObjectDomain: tenantAnalysisObjectDomain,
		ObjectType:   tenantAnalysisObjectType,
		ObjectID:     jobID,
		Provider:     settings.Provider,
		Model:        settings.Model,
		Currency:     settings.Currency,
		Status:       "blocked",
		TraceID:      newTenantAnalysisTraceID(auth.TenantID, auth.UserID),
		ErrorMessage: reason,
		MetadataJSON: toJSONString(map[string]interface{}{"idempotency_key": idempotencyKey}),
	}
	_ = utils.DB.Create(&log).Error
}

func tenantAnalysisJobResponse(job models.AIResearchJob) map[string]interface{} {
	usagePayload := map[string]interface{}{}
	if job.UsageLogID > 0 {
		var usage models.AIUsageLog
		if err := utils.DB.Where("tenant_id = ? AND id = ?", job.TenantID, job.UsageLogID).First(&usage).Error; err == nil {
			usagePayload = aiUsageResponse(usage)
		}
	}
	return map[string]interface{}{
		"id":            job.ID,
		"tenant_id":     job.TenantID,
		"user_id":       job.UserID,
		"status":        job.Status,
		"input":         mapValueFromJSON(job.InputJSON),
		"result":        mapValueFromJSON(job.ResultJSON),
		"sources":       mapValueFromJSON(job.SourceJSON),
		"confidence":    job.Confidence,
		"usage_log_id":  job.UsageLogID,
		"usage":         usagePayload,
		"error_message": job.ErrorMessage,
		"created_at":    job.CreatedAt,
		"updated_at":    job.UpdatedAt,
		"finished_at":   job.FinishedAt,
	}
}

func aiWalletResponse(wallet models.TenantAIWallet) map[string]interface{} {
	return map[string]interface{}{
		"enabled":                wallet.Enabled,
		"currency":               wallet.Currency,
		"balance_micros":         wallet.BalanceMicros,
		"balance_amount":         microsToAmount(wallet.BalanceMicros),
		"reserved_micros":        wallet.ReservedMicros,
		"reserved_amount":        microsToAmount(wallet.ReservedMicros),
		"available_micros":       wallet.BalanceMicros - wallet.ReservedMicros,
		"available_amount":       microsToAmount(wallet.BalanceMicros - wallet.ReservedMicros),
		"total_spent_micros":     wallet.TotalSpentMicros,
		"total_spent_amount":     microsToAmount(wallet.TotalSpentMicros),
		"monthly_token_limit":    wallet.MonthlyTokenLimit,
		"monthly_token_used":     wallet.MonthlyTokenUsed,
		"hard_stop":              wallet.HardStop,
		"last_usage_reset_month": wallet.LastUsageResetMonth,
	}
}

func tenantAIApplicationResponse(app models.TenantAIApplication) map[string]interface{} {
	return map[string]interface{}{
		"id":               app.ID,
		"tenant_id":        app.TenantID,
		"user_id":          app.UserID,
		"application_type": app.ApplicationType,
		"package_code":     app.PackageCode,
		"expected_usage":   app.ExpectedUsage,
		"contact_name":     app.ContactName,
		"contact_email":    app.ContactEmail,
		"contact_phone":    app.ContactPhone,
		"note":             app.Note,
		"status":           app.Status,
		"approved_by":      app.ApprovedBy,
		"approved_at":      app.ApprovedAt,
		"rejected_by":      app.RejectedBy,
		"rejected_at":      app.RejectedAt,
		"reject_reason":    app.RejectReason,
		"created_at":       app.CreatedAt,
	}
}

func aiLedgerResponse(row models.AITokenLedgerEntry) map[string]interface{} {
	return map[string]interface{}{
		"id":                     row.ID,
		"tenant_id":              row.TenantID,
		"user_id":                row.UserID,
		"usage_log_id":           row.UsageLogID,
		"job_id":                 row.JobID,
		"entry_type":             row.EntryType,
		"direction":              row.Direction,
		"amount_micros":          row.AmountMicros,
		"amount":                 microsToAmount(row.AmountMicros),
		"prompt_tokens":          row.PromptTokens,
		"completion_tokens":      row.CompletionTokens,
		"cached_input_tokens":    row.CachedInputTokens,
		"uncached_input_tokens":  row.UncachedInputTokens,
		"output_tokens":          row.OutputTokens,
		"total_tokens":           row.TotalTokens,
		"price_snapshot":         mapValueFromJSON(row.PriceSnapshotJSON),
		"balance_before_micros":  row.BalanceBeforeMicros,
		"balance_after_micros":   row.BalanceAfterMicros,
		"reserved_before_micros": row.ReservedBeforeMicros,
		"reserved_after_micros":  row.ReservedAfterMicros,
		"status":                 row.Status,
		"trace_id":               row.TraceID,
		"description":            row.Description,
		"created_at":             row.CreatedAt,
	}
}

func activeAIAnalysisPackages() []map[string]interface{} {
	var rows []models.AIAnalysisPackage
	_ = utils.DB.Where("status = ?", "active").Order("sort asc, id asc").Find(&rows).Error
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, aiAnalysisPackageResponse(row))
	}
	return items
}

func aiAnalysisPackageResponse(row models.AIAnalysisPackage) map[string]interface{} {
	return map[string]interface{}{
		"id":                  row.ID,
		"code":                row.Code,
		"name":                row.Name,
		"currency":            row.Currency,
		"price_micros":        row.PriceMicros,
		"price_amount":        microsToAmount(row.PriceMicros),
		"balance_micros":      row.BalanceMicros,
		"balance_amount":      microsToAmount(row.BalanceMicros),
		"monthly_token_limit": row.MonthlyTokenLimit,
		"validity_days":       row.ValidityDays,
		"description":         row.Description,
		"status":              row.Status,
		"sort":                row.Sort,
	}
}

func topDimensionRows(values map[string]*salesDimensionMetric, limit int, addEvidence func(string, string, uint, string, string) string, domain string, objectType string, metric string) []map[string]interface{} {
	rows := make([]*salesDimensionMetric, 0, len(values))
	for _, item := range values {
		rows = append(rows, item)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Amount > rows[j].Amount })
	if len(rows) > limit {
		rows = rows[:limit]
	}
	result := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		objectID := row.CompanyID
		if domain == "products" {
			objectID = row.ProductID
		}
		evidenceID := addEvidence(domain, objectType, objectID, row.Name, metric)
		result = append(result, map[string]interface{}{
			"key":         row.Key,
			"name":        row.Name,
			"amount":      row.Amount,
			"paid_amount": row.PaidAmount,
			"quantity":    row.Quantity,
			"count":       row.Count,
			"evidence_id": evidenceID,
		})
	}
	return result
}

func salesTargetSummaryPayload(auth authContext, year int) map[string]interface{} {
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(1, 0, 0)
	current := salesOrderAggregateForRange(auth.TenantID, start, end)
	return map[string]interface{}{
		"year":         year,
		"order_amount": current.OrderAmount,
		"paid_amount":  current.PaidAmount,
		"order_count":  current.OrderCount,
		"unassigned":   current.Unassigned,
	}
}
