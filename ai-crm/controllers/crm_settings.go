package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (c *CRMController) ListSettings() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	scope := strings.TrimSpace(c.Ctx.Input.Query("scope"))
	key := strings.TrimSpace(c.Ctx.Input.Query("key"))
	if !canReadCRMSetting(auth, scope, key) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	query := utils.DB.Where("tenant_id = ?", auth.TenantID)
	if scope != "" {
		query = query.Where("scope = ?", scope)
	}
	if key != "" {
		query = query.Where("`key` = ?", key)
	}
	var items []models.CRMSetting
	if err := query.Order("scope asc, `key` asc").Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取配置失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items})
}

func (c *CRMController) SaveSetting() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		Scope string                 `json:"scope"`
		Key   string                 `json:"key"`
		Value map[string]interface{} `json:"value"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Scope = strings.TrimSpace(req.Scope)
	req.Key = strings.TrimSpace(req.Key)
	if req.Scope == "" || req.Key == "" {
		utils.BadRequest(c.Ctx, "配置范围和键不能为空")
		return
	}
	if !canManageCRMSetting(auth, req.Scope, req.Key) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	valueJSON, _ := json.Marshal(req.Value)
	setting := models.CRMSetting{
		TenantModel: models.TenantModel{TenantID: auth.TenantID},
		Scope:       req.Scope,
		Key:         req.Key,
		ValueJSON:   string(valueJSON),
		UpdatedBy:   auth.UserID,
	}
	err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, req.Scope, req.Key).Assign(setting).FirstOrCreate(&setting).Error
	if err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_CRM_SETTING", fmt.Sprintf("%s.%s", req.Scope, req.Key), "operation.update_crm_setting", map[string]interface{}{"scope": req.Scope, "key": req.Key}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存配置失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_CRM_SETTING", fmt.Sprintf("%s.%s", req.Scope, req.Key), "operation.update_crm_setting", map[string]interface{}{"scope": req.Scope, "key": req.Key}, "success", "")
	utils.Success(c.Ctx, setting)
}

func canReadCRMSetting(auth authContext, scope string, key string) bool {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() {
		return true
	}
	return false
}

func canManageCRMSetting(auth authContext, scope string, key string) bool {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() {
		return true
	}
	return false
}

func isEnterpriseCompanySetting(scope string, key string) bool {
	return strings.TrimSpace(scope) == "company" && strings.TrimSpace(key) == "enterprise"
}

func (c *CRMController) SyncExchangeRates() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "orders", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		BaseCurrency string   `json:"base_currency"`
		Currencies   []string `json:"currencies"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	base := normalizeCurrencyCode(req.BaseCurrency)
	if base == "" {
		base = tenantBusinessCurrency(auth.TenantID)
	}
	currencies := normalizeCurrencyCodes(req.Currencies)
	if len(currencies) == 0 {
		currencies = []string{"CNY", "USD", "EUR", "GBP", "JPY", "KRW", "HKD", "TWD", "SGD", "AUD", "CAD", "AED"}
	}
	if !containsCurrencyCode(currencies, base) {
		currencies = append([]string{base}, currencies...)
	}
	rates, err := syncExchangeRatesFromProviders(base, currencies)
	if err != nil {
		utils.Error(c.Ctx, http.StatusBadGateway, "同步汇率失败，请稍后重试")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SYNC_EXCHANGE_RATES", "order_finance.default", "operation.sync_exchange_rates", map[string]interface{}{"base_currency": base, "currencies": currencies}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{
		"base_currency":   base,
		"exchange_rates":  rates,
		"synced_at":       time.Now().Format(time.RFC3339),
		"requested_codes": currencies,
	})
}

func syncExchangeRatesFromProviders(base string, currencies []string) (map[string]float64, error) {
	rates, err := fetchFrankfurterRates(base, currencies)
	if err == nil && hasAllRates(rates, currencies) {
		return rates, nil
	}
	fallbackRates, fallbackErr := fetchOpenExchangeRates(base, currencies)
	if fallbackErr == nil && len(fallbackRates) > 1 {
		return fallbackRates, nil
	}
	if err != nil {
		return nil, err
	}
	return rates, nil
}

func fetchFrankfurterRates(base string, currencies []string) (map[string]float64, error) {
	targets := make([]string, 0, len(currencies))
	for _, currency := range currencies {
		if currency != base {
			targets = append(targets, currency)
		}
	}
	rates := map[string]float64{base: 1}
	if len(targets) == 0 {
		return rates, nil
	}
	endpoint := "https://api.frankfurter.app/latest?from=" + url.QueryEscape(base) + "&to=" + url.QueryEscape(strings.Join(targets, ","))
	providerRates, err := fetchProviderRates(endpoint)
	if err != nil {
		return nil, err
	}
	for _, currency := range targets {
		rate := providerRates[currency]
		if rate > 0 {
			rates[currency] = roundExchangeRate(1 / rate)
		}
	}
	return rates, nil
}

func fetchOpenExchangeRates(base string, currencies []string) (map[string]float64, error) {
	endpoint := "https://open.er-api.com/v6/latest/" + url.PathEscape(base)
	providerRates, err := fetchProviderRates(endpoint)
	if err != nil {
		return nil, err
	}
	rates := map[string]float64{base: 1}
	for _, currency := range currencies {
		if currency == base {
			continue
		}
		rate := providerRates[currency]
		if rate > 0 {
			rates[currency] = roundExchangeRate(1 / rate)
		}
	}
	return rates, nil
}

func fetchProviderRates(endpoint string) (map[string]float64, error) {
	client := http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("exchange provider returned status %d", resp.StatusCode)
	}
	var payload struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Rates) == 0 {
		return nil, fmt.Errorf("exchange provider returned empty rates")
	}
	normalized := make(map[string]float64, len(payload.Rates))
	for key, value := range payload.Rates {
		normalized[normalizeCurrencyCode(key)] = value
	}
	return normalized, nil
}

func normalizeCurrencyCodes(values []string) []string {
	seen := map[string]bool{}
	items := make([]string, 0, len(values))
	for _, value := range values {
		code := normalizeCurrencyCode(value)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		items = append(items, code)
	}
	return items
}

func normalizeCurrencyCode(value string) string {
	code := strings.ToUpper(strings.TrimSpace(value))
	if len(code) != 3 {
		return ""
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return code
}

func hasAllRates(rates map[string]float64, currencies []string) bool {
	for _, currency := range currencies {
		if rates[currency] <= 0 {
			return false
		}
	}
	return true
}

func containsCurrencyCode(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func roundExchangeRate(value float64) float64 {
	return float64(int(value*10000+0.5)) / 10000
}
