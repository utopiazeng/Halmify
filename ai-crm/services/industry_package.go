package services

import (
	"ai-crm/models"
	"ai-crm/utils"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	IndustryPackageManufacturing = "manufacturing"

	IndustryPackageErrorDisabled = "INDUSTRY_PACKAGE_DISABLED"
	IndustryPackageErrorNotFound = "INDUSTRY_PACKAGE_NOT_FOUND"
	IndustryPackageErrorConflict = "INDUSTRY_PACKAGE_CONFLICT"
)

type IndustryPackageInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type IndustryPackageAssignment struct {
	PackageCode    string                 `json:"package_code"`
	PackageVersion string                 `json:"package_version"`
	Primary        bool                   `json:"primary"`
	Enabled        bool                   `json:"enabled"`
	EnabledAt      *time.Time             `json:"enabled_at"`
	DisabledAt     *time.Time             `json:"disabled_at"`
	Config         map[string]interface{} `json:"config"`
}

type IndustryManifest struct {
	Packages         []IndustryPackageInfo               `json:"packages"`
	MenuExtensions   []map[string]interface{}            `json:"menuExtensions"`
	PageTabs         map[string][]map[string]interface{} `json:"pageTabs"`
	ObjectActions    map[string][]map[string]interface{} `json:"objectActions"`
	DashboardWidgets []map[string]interface{}            `json:"dashboardWidgets"`
	Permissions      []string                            `json:"permissions"`
}

type IndustryPackageUpsert struct {
	PackageCode string
	Enabled     bool
	Config      map[string]interface{}
}

func EmptyIndustryManifest() IndustryManifest {
	return IndustryManifest{
		Packages:         []IndustryPackageInfo{},
		MenuExtensions:   []map[string]interface{}{},
		PageTabs:         map[string][]map[string]interface{}{},
		ObjectActions:    map[string][]map[string]interface{}{},
		DashboardWidgets: []map[string]interface{}{},
		Permissions:      []string{},
	}
}

func ManufacturingPackageInfo() IndustryPackageInfo {
	return IndustryPackageInfo{
		Code:        IndustryPackageManufacturing,
		Name:        "工业制造 CRM",
		Version:     "1.0",
		Description: "围绕制造客户、商机、RFQ、报价、订单承诺和售后机会的制造行业 CRM 工作台。",
		Status:      "active",
	}
}

func ManufacturingManifest() IndustryManifest {
	info := ManufacturingPackageInfo()
	menuExtensions := []map[string]interface{}{
		{"key": "manufacturing", "path": "/manufacturing", "titleKey": "industry.manufacturing.menu.title", "descKey": "industry.manufacturing.menu.desc", "icon": "Monitor", "group": "manufacturing", "order": 40, "permission": "mfg:view", "navMode": "workflow", "children": manufacturingMenuChildren()},
	}
	return IndustryManifest{
		Packages:       []IndustryPackageInfo{info},
		MenuExtensions: menuExtensions,
		PageTabs: map[string][]map[string]interface{}{
			"customer.detail": {
				{"key": "mfg_profile", "titleKey": "industry.manufacturing.tabs.profile", "component": "MfgCustomerProfileTab", "permission": "mfg:customer:view"},
				{"key": "mfg_decision_chain", "titleKey": "industry.manufacturing.tabs.decisionChain", "component": "MfgDecisionChainTab", "permission": "mfg:customer:view"},
				{"key": "mfg_rfqs", "titleKey": "industry.manufacturing.tabs.rfqs", "component": "MfgRFQTab", "permission": "mfg:rfq:view"},
				{"key": "mfg_samples", "titleKey": "industry.manufacturing.tabs.samples", "component": "MfgSampleTab", "permission": "mfg:sample:view"},
				{"key": "mfg_assets", "titleKey": "industry.manufacturing.tabs.assets", "component": "MfgAssetTab", "permission": "mfg:asset:view"},
				{"key": "mfg_profit", "titleKey": "industry.manufacturing.tabs.profit", "component": "MfgCustomerProfitTab", "permission": "mfg:metrics:view"},
			},
		},
		ObjectActions: map[string][]map[string]interface{}{
			"opportunity": {
				{"key": "create_mfg_rfq", "titleKey": "industry.manufacturing.action.createRFQ", "permission": "mfg:rfq:manage", "component": "CreateMfgRFQAction"},
			},
		},
		DashboardWidgets: []map[string]interface{}{
			{"key": "mfg_overview", "titleKey": "industry.manufacturing.dashboard.overview", "component": "MfgOverviewWidget", "permission": "mfg:metrics:view"},
		},
		Permissions: []string{
			"mfg:view",
			"mfg:customer:view",
			"mfg:customer:manage",
			"mfg:agreement:view",
			"mfg:agreement:manage",
			"mfg:rfq:view",
			"mfg:rfq:manage",
			"mfg:product:view",
			"mfg:product:manage",
			"mfg:cpq:view",
			"mfg:cpq:manage",
			"mfg:cost:view",
			"mfg:cost:view_sensitive",
			"mfg:cost:manage",
			"mfg:quote:view",
			"mfg:quote:approve",
			"mfg:quote:manage",
			"mfg:sample:view",
			"mfg:sample:manage",
			"mfg:production:view",
			"mfg:production:manage",
			"mfg:ecn:view",
			"mfg:ecn:approve",
			"mfg:ecn:manage",
			"mfg:compensation:view",
			"mfg:compensation:manage",
			"mfg:compensation:approve",
			"mfg:asset:view",
			"mfg:asset:manage",
			"mfg:service:view",
			"mfg:service:manage",
			"mfg:knowledge:view",
			"mfg:knowledge:manage",
			"mfg:metrics:view",
		},
	}
}

func manufacturingMenuChildren() []map[string]interface{} {
	return []map[string]interface{}{
		{"key": "mfg_quote_to_order", "path": "/manufacturing/quote-to-order", "titleKey": "industry.manufacturing.workflow.quoteToOrder", "descKey": "industry.manufacturing.workflow.quoteToOrderDesc", "permission": "mfg:rfq:view", "order": 2, "navMode": "workflow", "workflowGroup": "quote_to_order", "beginnerVisible": true, "legacyModules": []string{"rfqs", "clarifications", "drawings", "bom-snapshots", "cost-sheets", "quotes"}},
		{"key": "mfg_sample_to_mass", "path": "/manufacturing/sample-to-mass", "titleKey": "industry.manufacturing.workflow.sampleToMass", "descKey": "industry.manufacturing.workflow.sampleToMassDesc", "permission": "mfg:sample:view", "order": 3, "navMode": "workflow", "workflowGroup": "sample_to_mass", "beginnerVisible": true, "legacyModules": []string{"samples", "production"}},
		{"key": "mfg_delivery_change", "path": "/manufacturing/delivery-change", "titleKey": "industry.manufacturing.workflow.deliveryChange", "descKey": "industry.manufacturing.workflow.deliveryChangeDesc", "permission": "mfg:production:view", "order": 4, "navMode": "workflow", "workflowGroup": "delivery_change", "beginnerVisible": true, "legacyModules": []string{"production", "ecns"}},
		{"key": "mfg_settings", "path": "/manufacturing/settings", "titleKey": "industry.manufacturing.workflow.settings", "descKey": "industry.manufacturing.workflow.settingsDesc", "permission": "mfg:product:view", "order": 5, "navMode": "workflow", "workflowGroup": "settings", "beginnerVisible": true, "legacyModules": []string{"product-families", "bom-templates"}},
	}
}

func SeedIndustryPackages(tx *gorm.DB) error {
	manifestBytes, err := json.Marshal(ManufacturingManifest())
	if err != nil {
		return err
	}
	info := ManufacturingPackageInfo()
	registry := models.IndustryPackageRegistry{
		Code:         info.Code,
		Name:         info.Name,
		Version:      info.Version,
		Description:  info.Description,
		ManifestJSON: string(manifestBytes),
		Status:       info.Status,
	}
	if err := tx.Where("code = ?", registry.Code).Assign(registry).FirstOrCreate(&registry).Error; err != nil {
		return err
	}
	return seedManufacturingExtensions(tx)
}

func seedManufacturingExtensions(tx *gorm.DB) error {
	manifest := ManufacturingManifest()
	extensions := []models.IndustryExtensionRegistry{}
	for _, menu := range manifest.MenuExtensions {
		extensions = append(extensions, industryExtension("menu", "global.nav", stringValue(menu["key"]), menu, intValue(menu["order"])))
	}
	for target, tabs := range manifest.PageTabs {
		for idx, tab := range tabs {
			extensions = append(extensions, industryExtension("page_tab", target, stringValue(tab["key"]), tab, idx+1))
		}
	}
	for target, actions := range manifest.ObjectActions {
		for idx, action := range actions {
			extensions = append(extensions, industryExtension("object_action", target, stringValue(action["key"]), action, idx+1))
		}
	}
	for idx, widget := range manifest.DashboardWidgets {
		extensions = append(extensions, industryExtension("dashboard_widget", "dashboard", stringValue(widget["key"]), widget, idx+1))
	}
	for _, item := range extensions {
		ext := item
		if err := tx.Where("package_code = ? AND extension_type = ? AND target_key = ? AND extension_key = ?", ext.PackageCode, ext.ExtensionType, ext.TargetKey, ext.ExtensionKey).
			Assign(ext).
			FirstOrCreate(&ext).Error; err != nil {
			return err
		}
	}
	return nil
}

func industryExtension(extensionType, targetKey, extensionKey string, config map[string]interface{}, sort int) models.IndustryExtensionRegistry {
	raw, _ := json.Marshal(config)
	return models.IndustryExtensionRegistry{
		PackageCode:   IndustryPackageManufacturing,
		ExtensionType: extensionType,
		TargetKey:     targetKey,
		ExtensionKey:  extensionKey,
		ConfigJSON:    string(raw),
		Sort:          sort,
		Enabled:       true,
	}
}

func ListIndustryPackages() ([]IndustryPackageInfo, error) {
	var registries []models.IndustryPackageRegistry
	if err := utils.DB.Where("status <> ?", "deleted").Order("code asc").Find(&registries).Error; err != nil {
		return nil, err
	}
	if len(registries) == 0 {
		return []IndustryPackageInfo{ManufacturingPackageInfo()}, nil
	}
	items := make([]IndustryPackageInfo, 0, len(registries))
	for _, item := range registries {
		items = append(items, IndustryPackageInfo{
			Code:        item.Code,
			Name:        item.Name,
			Version:     item.Version,
			Description: item.Description,
			Status:      item.Status,
		})
	}
	return items, nil
}

func TenantIndustryPackage(tenantID uint) (IndustryPackageAssignment, bool, error) {
	var item models.TenantIndustryPackage
	err := utils.DB.Where("tenant_id = ? AND `primary` = ?", tenantID, true).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return IndustryPackageAssignment{Config: map[string]interface{}{}}, false, nil
		}
		return IndustryPackageAssignment{}, false, err
	}
	return assignmentFromModel(item), true, nil
}

func IsIndustryPackageEnabled(tenantID uint, packageCode string) bool {
	packageCode = NormalizeIndustryPackageCode(packageCode)
	if tenantID == 0 || packageCode == "" {
		return false
	}
	var count int64
	if err := utils.DB.Model(&models.TenantIndustryPackage{}).
		Where("tenant_id = ? AND package_code = ? AND enabled = ?", tenantID, packageCode, true).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func UpsertTenantIndustryPackageTx(tx *gorm.DB, tenantID uint, input IndustryPackageUpsert) (IndustryPackageAssignment, error) {
	code := NormalizeIndustryPackageCode(input.PackageCode)
	if code == "" || !input.Enabled {
		return disableTenantIndustryPackageTx(tx, tenantID)
	}
	if code != IndustryPackageManufacturing {
		return IndustryPackageAssignment{}, errors.New(IndustryPackageErrorNotFound)
	}
	var registry models.IndustryPackageRegistry
	if err := tx.Where("code = ? AND status = ?", code, "active").First(&registry).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return IndustryPackageAssignment{}, errors.New(IndustryPackageErrorNotFound)
		}
		return IndustryPackageAssignment{}, err
	}
	now := time.Now()
	configJSON := "{}"
	if input.Config != nil {
		raw, err := json.Marshal(input.Config)
		if err != nil {
			return IndustryPackageAssignment{}, err
		}
		configJSON = string(raw)
	}
	var existing models.TenantIndustryPackage
	err := tx.Where("tenant_id = ? AND `primary` = ?", tenantID, true).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return IndustryPackageAssignment{}, err
	}
	if existing.ID > 0 && existing.PackageCode != code && existing.Enabled {
		return IndustryPackageAssignment{}, errors.New(IndustryPackageErrorConflict)
	}
	existing.TenantID = tenantID
	existing.PackageCode = code
	existing.PackageVersion = registry.Version
	existing.Primary = true
	existing.Enabled = true
	existing.EnabledAt = &now
	existing.DisabledAt = nil
	existing.ConfigJSON = configJSON
	if existing.ID == 0 {
		if err := tx.Create(&existing).Error; err != nil {
			return IndustryPackageAssignment{}, err
		}
	} else if err := tx.Save(&existing).Error; err != nil {
		return IndustryPackageAssignment{}, err
	}
	return assignmentFromModel(existing), nil
}

func SetTenantIndustryPackage(tenantID uint, input IndustryPackageUpsert) (IndustryPackageAssignment, error) {
	var assignment IndustryPackageAssignment
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		next, err := UpsertTenantIndustryPackageTx(tx, tenantID, input)
		if err != nil {
			return err
		}
		assignment = next
		return nil
	})
	return assignment, err
}

func TenantIndustryManifest(tenantID uint) (IndustryManifest, error) {
	if !IsIndustryPackageEnabled(tenantID, IndustryPackageManufacturing) {
		return EmptyIndustryManifest(), nil
	}
	var registry models.IndustryPackageRegistry
	if err := utils.DB.Where("code = ? AND status = ?", IndustryPackageManufacturing, "active").First(&registry).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ManufacturingManifest(), nil
		}
		return EmptyIndustryManifest(), err
	}
	var manifest IndustryManifest
	if strings.TrimSpace(registry.ManifestJSON) != "" {
		if err := json.Unmarshal([]byte(registry.ManifestJSON), &manifest); err == nil {
			normalizeManifest(&manifest)
			return manifest, nil
		}
	}
	return ManufacturingManifest(), nil
}

func NormalizeIndustryPackageCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "none", "generic", "core":
		return ""
	case IndustryPackageManufacturing, "mfg", "industrial_manufacturing":
		return IndustryPackageManufacturing
	default:
		return value
	}
}

func disableTenantIndustryPackageTx(tx *gorm.DB, tenantID uint) (IndustryPackageAssignment, error) {
	var existing models.TenantIndustryPackage
	err := tx.Where("tenant_id = ? AND `primary` = ?", tenantID, true).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return IndustryPackageAssignment{Config: map[string]interface{}{}}, nil
		}
		return IndustryPackageAssignment{}, err
	}
	now := time.Now()
	existing.Enabled = false
	existing.DisabledAt = &now
	if err := tx.Save(&existing).Error; err != nil {
		return IndustryPackageAssignment{}, err
	}
	return assignmentFromModel(existing), nil
}

func assignmentFromModel(item models.TenantIndustryPackage) IndustryPackageAssignment {
	config := map[string]interface{}{}
	if strings.TrimSpace(item.ConfigJSON) != "" {
		_ = json.Unmarshal([]byte(item.ConfigJSON), &config)
	}
	return IndustryPackageAssignment{
		PackageCode:    item.PackageCode,
		PackageVersion: item.PackageVersion,
		Primary:        item.Primary,
		Enabled:        item.Enabled,
		EnabledAt:      item.EnabledAt,
		DisabledAt:     item.DisabledAt,
		Config:         config,
	}
}

func normalizeManifest(manifest *IndustryManifest) {
	if manifest.Packages == nil {
		manifest.Packages = []IndustryPackageInfo{}
	}
	if manifest.MenuExtensions == nil {
		manifest.MenuExtensions = []map[string]interface{}{}
	}
	if manifest.PageTabs == nil {
		manifest.PageTabs = map[string][]map[string]interface{}{}
	}
	if manifest.ObjectActions == nil {
		manifest.ObjectActions = map[string][]map[string]interface{}{}
	}
	if manifest.DashboardWidgets == nil {
		manifest.DashboardWidgets = []map[string]interface{}{}
	}
	if manifest.Permissions == nil {
		manifest.Permissions = []string{}
	}
}

func stringValue(value interface{}) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func intValue(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	raw, _ := json.Marshal(value)
	return string(raw)
}
