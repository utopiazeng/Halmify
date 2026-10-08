package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"encoding/json"
	"strings"
)

func getTenantQuota(tenantID uint) (*models.TenantQuota, error) {
	if tenantID == 0 {
		return nil, nil
	}
	var quota models.TenantQuota
	if err := utils.DB.Where("tenant_id = ?", tenantID).First(&quota).Error; err != nil {
		return nil, err
	}
	return &quota, nil
}

var supportedTenantModules = map[string]bool{
	"workspace": true, "enterprise": true, "projects": true, "products": true, "campaigns": true,
	"contacts": true, "leads": true, "opportunities": true, "deals": true, "orders": true,
	"tickets": true, "dashboards": true, "teams": true, "insights": true, "accounts": true,
	"im": true, "iot": true, "audit_log": true,
}

func tenantEffectiveModules(quota *models.TenantQuota) []string {
	if quota == nil {
		return []string{}
	}
	modules := normalizeTenantModules(splitCSV(quota.Modules))
	if len(modules) > 0 {
		return modules
	}
	planModules := modulesByPlanCode(quota.PlanCode)
	if len(planModules) > 0 {
		return planModules
	}
	return []string{}
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
