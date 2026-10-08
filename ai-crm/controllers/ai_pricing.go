package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
)

const (
	deepSeekProvider           = "deepseek"
	defaultDeepSeekModel       = "deepseek-v4-flash"
	deepSeekPricingSourceURL   = "https://api-docs.deepseek.com/quick_start/pricing/"
	deepSeekPricingSyncWindow  = 24 * time.Hour
	aiPricingMarginRate        = 0.15
	aiPricingMarginNumerator   = int64(115)
	aiPricingMarginDenominator = int64(100)
)

var deepSeekFallbackPrices = map[string]map[string]int64{
	"deepseek-v4-flash": {
		"input_cache_hit":  amountFloatToMicros(0.0028),
		"input_cache_miss": amountFloatToMicros(0.14),
		"output":           amountFloatToMicros(0.28),
	},
	"deepseek-v4-pro": {
		"input_cache_hit":  amountFloatToMicros(0.003625),
		"input_cache_miss": amountFloatToMicros(0.435),
		"output":           amountFloatToMicros(0.87),
	},
}

type aiPricingRate struct {
	Metric                        string `json:"metric"`
	Label                         string `json:"label"`
	Currency                      string `json:"currency"`
	OfficialPriceMicrosPerMillion int64  `json:"official_price_micros_per_million"`
	TenantPriceMicrosPerMillion   int64  `json:"tenant_price_micros_per_million"`
	OfficialPriceAmount           string `json:"official_price_amount"`
	TenantPriceAmount             string `json:"tenant_price_amount"`
}

type aiPricingModel struct {
	Provider      string          `json:"provider"`
	Model         string          `json:"model"`
	DisplayName   string          `json:"display_name"`
	ContextLength string          `json:"context_length"`
	MaxOutput     string          `json:"max_output"`
	Rates         []aiPricingRate `json:"rates"`
}

type aiPricingSnapshot struct {
	Provider      string            `json:"provider"`
	Currency      string            `json:"currency"`
	MarginRate    float64           `json:"margin_rate"`
	SourceURL     string            `json:"source_url"`
	SourceStatus  string            `json:"source_status"`
	SourceMessage string            `json:"source_message"`
	SyncedAt      string            `json:"synced_at"`
	NextSyncAt    string            `json:"next_sync_at"`
	Aliases       map[string]string `json:"aliases"`
	Models        []aiPricingModel  `json:"models"`
}

func (c *CRMController) AIPricing() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "insights", "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	snapshot := ensureAIPricingSnapshot(0)
	wallet, _ := ensureTenantAIWallet(auth.TenantID)
	utils.Success(c.Ctx, map[string]interface{}{
		"pricing": snapshot,
		"wallet": map[string]interface{}{
			"enabled":             wallet.Enabled,
			"currency":            wallet.Currency,
			"balance_amount":      microsToAmount(wallet.BalanceMicros),
			"balance_micros":      wallet.BalanceMicros,
			"monthly_token_limit": wallet.MonthlyTokenLimit,
			"monthly_token_used":  wallet.MonthlyTokenUsed,
			"total_spent_amount":  microsToAmount(wallet.TotalSpentMicros),
		},
	})
}

func (c *PlatformController) SyncAIPricing() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	snapshot, err := syncDeepSeekPricingSnapshot()
	if err != nil {
		logControllerOperation(c.Ctx, auth, 0, "SYNC_AI_PRICING", "deepseek.pricing", "operation.platform_settings_publish", nil, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusBadGateway, "同步 DeepSeek 官方报价失败")
		return
	}
	if err := saveActiveAIPricingSnapshot(snapshot, auth.UserID); err != nil {
		logControllerOperation(c.Ctx, auth, 0, "SYNC_AI_PRICING", "deepseek.pricing", "operation.platform_settings_publish", nil, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存 AI 报价失败")
		return
	}
	logControllerOperation(c.Ctx, auth, 0, "SYNC_AI_PRICING", "deepseek.pricing", "operation.platform_settings_publish", map[string]interface{}{"source": snapshot.SourceURL}, "success", "")
	utils.Success(c.Ctx, snapshot)
}

func StartAIPricingSyncScheduler() {
	run := func() {
		snapshot, err := syncDeepSeekPricingSnapshot()
		if err != nil {
			utils.LogWarning("DeepSeek 报价定时同步失败: %v", err)
			return
		}
		if err := saveActiveAIPricingSnapshot(snapshot, 0); err != nil {
			utils.LogWarning("DeepSeek 报价定时保存失败: %v", err)
			return
		}
	}
	go func() {
		services.WithSchedulerLock("ai_pricing_sync_scheduler", run)
		ticker := time.NewTicker(deepSeekPricingSyncWindow)
		defer ticker.Stop()
		for range ticker.C {
			services.WithSchedulerLock("ai_pricing_sync_scheduler", run)
		}
	}()
}

func ensureAIPricingSnapshot(userID uint) aiPricingSnapshot {
	if snapshot, ok := activeAIPricingSnapshot(); ok && !aiPricingSnapshotStale(snapshot) {
		return normalizeDeepSeekPricingSnapshot(snapshot)
	}
	synced, err := syncDeepSeekPricingSnapshot()
	if err == nil {
		synced = normalizeDeepSeekPricingSnapshot(synced)
		_ = saveActiveAIPricingSnapshot(synced, userID)
		return synced
	}
	if snapshot, ok := activeAIPricingSnapshot(); ok {
		snapshot.SourceStatus = "stale"
		snapshot.SourceMessage = err.Error()
		return normalizeDeepSeekPricingSnapshot(snapshot)
	}
	return normalizeDeepSeekPricingSnapshot(fallbackDeepSeekPricingSnapshot(err.Error()))
}

func activeAIPricingSnapshot() (aiPricingSnapshot, bool) {
	var active models.PlatformSettingVersion
	if err := utils.DB.Where("scope = ? AND `group` = ? AND `key` = ? AND status = ?", "platform", "ai", "deepseek_pricing", "active").Order("version desc").First(&active).Error; err != nil {
		return aiPricingSnapshot{}, false
	}
	var snapshot aiPricingSnapshot
	if err := json.Unmarshal([]byte(active.ValueJSON), &snapshot); err != nil || len(snapshot.Models) == 0 {
		return aiPricingSnapshot{}, false
	}
	return snapshot, true
}

func aiPricingSnapshotStale(snapshot aiPricingSnapshot) bool {
	if snapshot.MarginRate != aiPricingMarginRate {
		return true
	}
	if !deepSeekSnapshotHasConfiguredModel(snapshot) {
		return true
	}
	if strings.TrimSpace(snapshot.SyncedAt) == "" {
		return true
	}
	syncedAt, err := time.Parse(time.RFC3339, snapshot.SyncedAt)
	if err != nil {
		return true
	}
	return time.Since(syncedAt) > deepSeekPricingSyncWindow
}

func deepSeekSnapshotHasConfiguredModel(snapshot aiPricingSnapshot) bool {
	modelName := currentDeepSeekModel()
	for _, model := range snapshot.Models {
		if normalizeDeepSeekConfiguredModel(model.Model) == modelName {
			return true
		}
	}
	for alias, target := range snapshot.Aliases {
		if normalizeDeepSeekConfiguredModel(alias) == modelName || normalizeDeepSeekConfiguredModel(target) == modelName {
			return true
		}
	}
	return false
}

func saveActiveAIPricingSnapshot(snapshot aiPricingSnapshot, userID uint) error {
	snapshot = normalizeDeepSeekPricingSnapshot(snapshot)
	valueJSON, _ := json.Marshal(snapshot)
	now := time.Now()
	return utils.DB.Transaction(func(tx *gorm.DB) error {
		var currentVersion int
		if err := tx.Model(&models.PlatformSettingVersion{}).
			Where("scope = ? AND `group` = ? AND `key` = ?", "platform", "ai", "deepseek_pricing").
			Select("COALESCE(MAX(version),0)").Scan(&currentVersion).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.PlatformSettingVersion{}).
			Where("scope = ? AND `group` = ? AND `key` = ? AND status = ?", "platform", "ai", "deepseek_pricing", "active").
			Update("status", "rollback").Error; err != nil {
			return err
		}
		item := models.PlatformSettingVersion{
			Scope:       "platform",
			Group:       "ai",
			Key:         "deepseek_pricing",
			ValueJSON:   string(valueJSON),
			Version:     currentVersion + 1,
			Status:      "active",
			EffectiveAt: &now,
			CreatedBy:   userID,
			PublishedBy: userID,
			PublishedAt: &now,
		}
		return tx.Create(&item).Error
	})
}

func syncDeepSeekPricingSnapshot() (aiPricingSnapshot, error) {
	modelName := currentDeepSeekModel()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(deepSeekPricingSourceURL)
	if err != nil {
		return fallbackDeepSeekPricingSnapshot(err.Error()), err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("DeepSeek 报价页返回状态 %d", resp.StatusCode)
		return fallbackDeepSeekPricingSnapshot(err.Error()), err
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return fallbackDeepSeekPricingSnapshot(err.Error()), err
	}
	html := string(body)
	prices, err := parseDeepSeekPricingHTML(html, modelName)
	if err != nil {
		return fallbackDeepSeekPricingSnapshot(err.Error()), err
	}
	return buildDeepSeekPricingSnapshot(prices, "synced", "已从 DeepSeek 官方报价页同步"), nil
}

func parseDeepSeekPricingHTML(html string, modelName string) (map[string]map[string]int64, error) {
	modelName = normalizeDeepSeekConfiguredModel(modelName)
	labels := map[string]string{
		"input_cache_hit":  "1M INPUT TOKENS (CACHE HIT)",
		"input_cache_miss": "1M INPUT TOKENS (CACHE MISS)",
		"output":           "1M OUTPUT TOKENS",
	}
	modelIndex := deepSeekPricingModelIndex(html, modelName)
	if modelIndex < 0 {
		return nil, fmt.Errorf("DeepSeek 官方页面未找到模型 %s 报价列", modelName)
	}
	prices := map[string]int64{}
	tableRows := regexp.MustCompile(`(?is)<tr>.*?</tr>`).FindAllString(html, -1)
	for metric, label := range labels {
		row := ""
		for _, candidate := range tableRows {
			if strings.Contains(stripHTML(candidate), label) {
				row = candidate
				break
			}
		}
		if row == "" {
			return nil, fmt.Errorf("未能从 DeepSeek 官方页面解析 %s 报价", metric)
		}
		values := dollarValues(row)
		if len(values) <= modelIndex {
			return nil, fmt.Errorf("DeepSeek 官方页面 %s 报价格式不可识别", metric)
		}
		prices[metric] = values[modelIndex]
	}
	return map[string]map[string]int64{
		modelName: prices,
	}, nil
}

func deepSeekPricingModelIndex(html string, modelName string) int {
	modelName = normalizeDeepSeekConfiguredModel(modelName)
	tableRows := regexp.MustCompile(`(?is)<tr>.*?</tr>`).FindAllString(html, -1)
	for _, row := range tableRows {
		cells := tableCells(row)
		modelColumnIndex := 0
		seenModelHeader := false
		for _, cell := range cells {
			cell = strings.TrimSpace(cell)
			if strings.EqualFold(cell, "MODEL") {
				seenModelHeader = true
				continue
			}
			if !seenModelHeader || cell == "" {
				continue
			}
			if strings.EqualFold(normalizeDeepSeekConfiguredModel(cell), modelName) {
				return modelColumnIndex
			}
			modelColumnIndex++
		}
	}
	return -1
}

func tableCells(row string) []string {
	matches := regexp.MustCompile(`(?is)<t[dh][^>]*>.*?</t[dh]>`).FindAllString(row, -1)
	cells := make([]string, 0, len(matches))
	for _, match := range matches {
		colspan := tableCellSpan(match, "colspan")
		value := stripHTML(match)
		value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")
		value = strings.TrimSpace(value)
		if value != "" {
			for i := 0; i < colspan; i++ {
				cells = append(cells, value)
			}
		}
	}
	return cells
}

func tableCellSpan(cell string, attr string) int {
	re := regexp.MustCompile(`(?i)` + attr + `\s*=\s*["']?([0-9]+)`)
	match := re.FindStringSubmatch(cell)
	if len(match) < 2 {
		return 1
	}
	parsed, err := strconv.Atoi(match[1])
	if err != nil || parsed <= 0 {
		return 1
	}
	return parsed
}

func dollarValues(text string) []int64 {
	matches := regexp.MustCompile(`\$([0-9]+(?:\.[0-9]+)?)`).FindAllStringSubmatch(text, -1)
	values := []int64{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		parsed, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			continue
		}
		values = append(values, amountFloatToMicros(parsed))
	}
	return values
}

func fallbackDeepSeekPricingSnapshot(message string) aiPricingSnapshot {
	modelName := currentDeepSeekModel()
	prices := deepSeekFallbackPrices[modelName]
	if len(prices) == 0 {
		prices = deepSeekFallbackPrices[defaultDeepSeekModel]
		message = strings.TrimSpace(message)
		if message == "" {
			message = fmt.Sprintf("missing fallback pricing for %s", modelName)
		}
	}
	return buildDeepSeekPricingSnapshot(map[string]map[string]int64{modelName: prices}, "fallback", message)
}

func buildDeepSeekPricingSnapshot(prices map[string]map[string]int64, status string, message string) aiPricingSnapshot {
	now := time.Now()
	modelName := currentDeepSeekModel()
	if _, ok := prices[modelName]; !ok {
		for name := range prices {
			modelName = normalizeDeepSeekConfiguredModel(name)
			break
		}
	}
	models := ensureDeepSeekPricingModels([]aiPricingModel{
		deepSeekPricingModel(modelName, deepSeekModelDisplayName(modelName), prices[modelName]),
	})
	return aiPricingSnapshot{
		Provider:      deepSeekProvider,
		Currency:      defaultAICurrency,
		MarginRate:    aiPricingMarginRate,
		SourceURL:     deepSeekPricingSourceURL,
		SourceStatus:  status,
		SourceMessage: message,
		SyncedAt:      now.Format(time.RFC3339),
		NextSyncAt:    now.Add(deepSeekPricingSyncWindow).Format(time.RFC3339),
		Aliases: map[string]string{
			"deepseek-chat":     modelName,
			"deepseek-reasoner": modelName,
		},
		Models: models,
	}
}

func normalizeDeepSeekPricingSnapshot(snapshot aiPricingSnapshot) aiPricingSnapshot {
	modelName := currentDeepSeekModel()
	if strings.TrimSpace(snapshot.Provider) == "" {
		snapshot.Provider = deepSeekProvider
	}
	if strings.TrimSpace(snapshot.Currency) == "" {
		snapshot.Currency = defaultAICurrency
	}
	if strings.TrimSpace(snapshot.SourceURL) == "" {
		snapshot.SourceURL = deepSeekPricingSourceURL
	}
	if snapshot.MarginRate != aiPricingMarginRate {
		snapshot.MarginRate = aiPricingMarginRate
	}
	snapshot.Aliases = map[string]string{
		"deepseek-chat":     modelName,
		"deepseek-reasoner": modelName,
	}
	snapshot.Models = ensureDeepSeekPricingModels(snapshot.Models)
	for i := range snapshot.Models {
		snapshot.Models[i].Provider = deepSeekProvider
		snapshot.Models[i].Model = modelName
		snapshot.Models[i].DisplayName = deepSeekModelDisplayName(modelName)
		for j := range snapshot.Models[i].Rates {
			rate := &snapshot.Models[i].Rates[j]
			if strings.TrimSpace(rate.Currency) == "" {
				rate.Currency = snapshot.Currency
			}
			if rate.OfficialPriceMicrosPerMillion <= 0 && rate.TenantPriceMicrosPerMillion > 0 {
				rate.OfficialPriceMicrosPerMillion = reverseAIPricingMargin(rate.TenantPriceMicrosPerMillion)
			}
			if rate.OfficialPriceMicrosPerMillion > 0 {
				rate.TenantPriceMicrosPerMillion = applyAIPricingMargin(rate.OfficialPriceMicrosPerMillion)
			}
			rate.OfficialPriceAmount = microsToAmount(rate.OfficialPriceMicrosPerMillion)
			rate.TenantPriceAmount = microsToAmount(rate.TenantPriceMicrosPerMillion)
		}
	}
	return snapshot
}

func deepSeekPricingModel(model string, displayName string, prices map[string]int64) aiPricingModel {
	prices = completeDeepSeekPricing(model, prices)
	return aiPricingModel{
		Provider:      deepSeekProvider,
		Model:         model,
		DisplayName:   displayName,
		ContextLength: "1M",
		MaxOutput:     "384K",
		Rates: []aiPricingRate{
			newAIPricingRate("input_cache_hit", "输入 token（缓存命中）", prices["input_cache_hit"]),
			newAIPricingRate("input_cache_miss", "输入 token（缓存未命中）", prices["input_cache_miss"]),
			newAIPricingRate("output", "输出 token", prices["output"]),
		},
	}
}

func ensureDeepSeekPricingModels(models []aiPricingModel) []aiPricingModel {
	modelName := currentDeepSeekModel()
	modelPrices := map[string]int64{}
	for _, model := range models {
		itemModel := effectiveDeepSeekModel(model.Model)
		if itemModel != modelName {
			continue
		}
		for _, rate := range model.Rates {
			if rate.OfficialPriceMicrosPerMillion > 0 {
				modelPrices[rate.Metric] = rate.OfficialPriceMicrosPerMillion
				continue
			}
			if rate.TenantPriceMicrosPerMillion > 0 {
				modelPrices[rate.Metric] = reverseAIPricingMargin(rate.TenantPriceMicrosPerMillion)
			}
		}
	}
	return []aiPricingModel{
		deepSeekPricingModel(modelName, deepSeekModelDisplayName(modelName), modelPrices),
	}
}

func fallbackDeepSeekPricingModels() []aiPricingModel {
	modelName := currentDeepSeekModel()
	prices := deepSeekFallbackPrices[modelName]
	if len(prices) == 0 {
		prices = deepSeekFallbackPrices[defaultDeepSeekModel]
	}
	return []aiPricingModel{
		deepSeekPricingModel(modelName, deepSeekModelDisplayName(modelName), prices),
	}
}

func completeDeepSeekPricing(model string, prices map[string]int64) map[string]int64 {
	if prices == nil {
		prices = map[string]int64{}
	}
	model = effectiveDeepSeekModel(model)
	fallback := deepSeekFallbackPrices[model]
	if len(fallback) == 0 {
		fallback = deepSeekFallbackPrices[defaultDeepSeekModel]
	}
	for key, value := range fallback {
		if prices[key] <= 0 {
			prices[key] = value
		}
	}
	return prices
}

func newAIPricingRate(metric string, label string, officialMicros int64) aiPricingRate {
	tenantMicros := applyAIPricingMargin(officialMicros)
	return aiPricingRate{
		Metric:                        metric,
		Label:                         label,
		Currency:                      defaultAICurrency,
		OfficialPriceMicrosPerMillion: officialMicros,
		TenantPriceMicrosPerMillion:   tenantMicros,
		OfficialPriceAmount:           microsToAmount(officialMicros),
		TenantPriceAmount:             microsToAmount(tenantMicros),
	}
}

func applyAIPricingMargin(officialMicros int64) int64 {
	if officialMicros <= 0 {
		return 0
	}
	return (officialMicros*aiPricingMarginNumerator + aiPricingMarginDenominator - 1) / aiPricingMarginDenominator
}

func reverseAIPricingMargin(tenantMicros int64) int64 {
	if tenantMicros <= 0 {
		return 0
	}
	return (tenantMicros*aiPricingMarginDenominator + aiPricingMarginNumerator - 1) / aiPricingMarginNumerator
}

func amountFloatToMicros(value float64) int64 {
	return int64(math.Round(value * 1000000))
}

func pricingModel(snapshot aiPricingSnapshot, modelName string) aiPricingModel {
	snapshot = normalizeDeepSeekPricingSnapshot(snapshot)
	modelName = effectiveDeepSeekModel(modelName)
	if alias, ok := snapshot.Aliases[modelName]; ok {
		modelName = alias
	}
	for _, item := range snapshot.Models {
		if item.Model == modelName {
			return item
		}
	}
	if len(snapshot.Models) > 0 {
		return snapshot.Models[0]
	}
	return fallbackDeepSeekPricingSnapshot("missing pricing").Models[0]
}

func effectiveDeepSeekModel(model string) string {
	if strings.TrimSpace(model) == "" {
		return currentDeepSeekModel()
	}
	model = normalizeDeepSeekConfiguredModel(model)
	if model == "" || model == "deepseek-chat" || model == "deepseek-reasoner" {
		return currentDeepSeekModel()
	}
	return model
}

func currentDeepSeekModel() string {
	return normalizeDeepSeekConfiguredModel(firstNonEmpty(
		os.Getenv("AI_MODEL"),
		web.AppConfig.DefaultString("ai_model", ""),
		os.Getenv("DEEPSEEK_MODEL"),
		web.AppConfig.DefaultString("deepseek_model", ""),
		defaultDeepSeekModel,
	))
}

func normalizeDeepSeekConfiguredModel(model string) string {
	model = strings.TrimSpace(stripHTML(model))
	model = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(model, "")
	model = regexp.MustCompile(`\s+`).ReplaceAllString(model, " ")
	fields := strings.Fields(model)
	if len(fields) > 0 {
		model = fields[0]
	}
	model = strings.Trim(model, " \t\r\n*：:")
	if model == "" {
		return defaultDeepSeekModel
	}
	return strings.ToLower(model)
}

func deepSeekModelDisplayName(model string) string {
	model = effectiveDeepSeekModel(model)
	parts := strings.Split(model, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		if part == "v4" {
			parts[i] = "V4"
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func pricingRate(model aiPricingModel, metric string) aiPricingRate {
	for _, item := range model.Rates {
		if item.Metric == metric {
			return item
		}
	}
	return aiPricingRate{}
}
