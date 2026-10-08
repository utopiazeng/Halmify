package utils

import (
	"ai-crm/models"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const migrationsTableName = "schema_migrations"

// SchemaMigration 记录已执行的迁移版本
type SchemaMigration struct {
	Version   string    `gorm:"primaryKey;size:64" json:"version"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	AppliedAt time.Time `gorm:"not null;autoCreateTime" json:"applied_at"`
}

type migration struct {
	Version string
	Name    string
	Up      func(tx *gorm.DB) error
}

// RunMigrations 在服务启动时执行增量迁移
func RunMigrations() error {
	if err := ensureMigrationsTable(); err != nil {
		return err
	}

	migrations := []migration{
		{
			Version: "2026051201",
			Name:    "init_all_tables",
			Up: func(tx *gorm.DB) error {
				return autoMigrateWithDB(tx)
			},
		},
		{
			Version: "2026051202",
			Name:    "users_last_login_nullable",
			Up: func(tx *gorm.DB) error {
				return tx.Exec("ALTER TABLE users MODIFY COLUMN last_login datetime(3) NULL").Error
			},
		},
		{
			Version: "2026051203",
			Name:    "p0_business_tables",
			Up: func(tx *gorm.DB) error {
				return autoMigrateWithDB(tx)
			},
		},
		{
			Version: "2026051204",
			Name:    "p1_business_tables",
			Up: func(tx *gorm.DB) error {
				return autoMigrateWithDB(tx)
			},
		},
		{
			Version: "2026051205",
			Name:    "p2_intelligence_and_open_platform_tables",
			Up: func(tx *gorm.DB) error {
				return autoMigrateWithDB(tx)
			},
		},
		{
			Version: "2026051301",
			Name:    "saas_tenant_rbac_operation_log",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := seedDefaultPermissions(tx); err != nil {
					return err
				}
				if err := normalizeExistingUsers(tx); err != nil {
					return err
				}
				return createSaaSIndexes(tx)
			},
		},
		{
			Version: "2026051302",
			Name:    "new_prd_unified_crm_platform",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := seedNewPRDPermissions(tx); err != nil {
					return err
				}
				return createNewPRDIndexes(tx)
			},
		},
		{
			Version: "2026051303",
			Name:    "user_multi_roles",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return backfillUserRoles(tx)
			},
		},
		{
			Version: "2026051304",
			Name:    "operation_log_user_agent",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return createOperationLogExportIndexes(tx)
			},
		},
		{
			Version: "2026051305",
			Name:    "new_prd_only_schema_cleanup",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := dropLegacyDemoTables(tx); err != nil {
					return err
				}
				if err := seedNewPRDPermissions(tx); err != nil {
					return err
				}
				if err := ensureNewPRDDefaultTenant(tx); err != nil {
					return err
				}
				if err := seedNewPRDSampleData(tx); err != nil {
					return err
				}
				return createNewPRDIndexes(tx)
			},
		},
		{
			Version: "2026051306",
			Name:    "new_prd_final_schema_prune",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := dropLegacyDemoTables(tx); err != nil {
					return err
				}
				if err := pruneLegacyPermissions(tx); err != nil {
					return err
				}
				if err := ensureNewPRDDefaultTenant(tx); err != nil {
					return err
				}
				if err := seedNewPRDPermissions(tx); err != nil {
					return err
				}
				return createNewPRDIndexes(tx)
			},
		},
		{
			Version: "2026051307",
			Name:    "new_prd_complete_capability_seed",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := dropLegacyDemoTables(tx); err != nil {
					return err
				}
				if err := pruneLegacyPermissions(tx); err != nil {
					return err
				}
				if err := ensureNewPRDDefaultTenant(tx); err != nil {
					return err
				}
				if err := seedNewPRDPermissions(tx); err != nil {
					return err
				}
				if err := seedNewPRDSettings(tx); err != nil {
					return err
				}
				if err := seedNewPRDCompleteSamples(tx); err != nil {
					return err
				}
				return createNewPRDIndexes(tx)
			},
		},
		{
			Version: "2026051401",
			Name:    "system_management_module",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := seedSystemPermissions(tx); err != nil {
					return err
				}
				if err := seedSystemManagementData(tx); err != nil {
					return err
				}
				return createSystemManagementIndexes(tx)
			},
		},
		{
			Version: "2026051501",
			Name:    "lead_owner_data_scope",
			Up: func(tx *gorm.DB) error {
				return seedLeadOwnerDataScope(tx)
			},
		},
		{
			Version: "2026051701",
			Name:    "platform_boundary_and_settings_center",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return createPlatformIndexes(tx)
			},
		},
		{
			Version: "2026051702",
			Name:    "tenant_admin_activation_tokens",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return createTenantAdminActivationIndexes(tx)
			},
		},
		{
			Version: "2026051703",
			Name:    "performance_indexes_for_crm_objects",
			Up: func(tx *gorm.DB) error {
				return createPerformanceIndexes(tx)
			},
		},
		{
			Version: "2026051704",
			Name:    "operation_log_user_query_index",
			Up: func(tx *gorm.DB) error {
				return createOperationLogUserIndexes(tx)
			},
		},
		{
			Version: "2026051705",
			Name:    "normalize_platform_super_admin_accounts",
			Up: func(tx *gorm.DB) error {
				return normalizePlatformSuperAdmins(tx)
			},
		},
		{
			Version: "2026051706",
			Name:    "im_single_chat_mvp",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := seedIMPermissions(tx); err != nil {
					return err
				}
				return createIMIndexes(tx)
			},
		},
		{
			Version: "2026051707",
			Name:    "platform_tenant_plan_expiry_runtime",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := seedPlatformPlanTemplates(tx); err != nil {
					return err
				}
				return createPlatformGovernanceIndexes(tx)
			},
		},
		{
			Version: "2026051708",
			Name:    "shrink_plans_to_basic_plus",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return shrinkPlansToBasicPlus(tx)
			},
		},
		{
			Version: "2026051709",
			Name:    "fine_grained_permissions_v1",
			Up: func(tx *gorm.DB) error {
				if err := seedFineGrainedPermissions(tx); err != nil {
					return err
				}
				return backfillFineGrainedRolePermissions(tx)
			},
		},
		{
			Version: "2026051801",
			Name:    "crm_object_acl_project_ticket_files",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return createCRMObjectACLIndexes(tx)
			},
		},
		{
			Version: "2026051802",
			Name:    "company_ai_research_wallet_usage",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := seedTenantAIWallets(tx); err != nil {
					return err
				}
				return createAIResearchIndexes(tx)
			},
		},
		{
			Version: "2026051901",
			Name:    "im_support_conversation_metadata",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return ensureIMSupportConversationColumns(tx)
			},
		},
		{
			Version: "2026052401",
			Name:    "user_avatar_url_longtext",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return tx.Exec("ALTER TABLE users MODIFY COLUMN avatar_url LONGTEXT").Error
			},
		},
		{
			Version: "2026052402",
			Name:    "tenant_ai_analysis_billing",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := seedAIAnalysisPackages(tx); err != nil {
					return err
				}
				if err := seedFineGrainedPermissions(tx); err != nil {
					return err
				}
				return createAIAnalysisIndexes(tx)
			},
		},
		{
			Version: "2026052501",
			Name:    "tenant_storage_expansion",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := seedStoragePermissions(tx); err != nil {
					return err
				}
				return createTenantStorageExpansionIndexes(tx)
			},
		},
		{
			Version: "2026052701",
			Name:    "tenant_entitlement_expansion_billing",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := createTenantStorageExpansionIndexes(tx); err != nil {
					return err
				}
				if err := createTenantSeatExpansionIndexes(tx); err != nil {
					return err
				}
				return backfillTenantSeatExpansions(tx)
			},
		},
		{
			Version: "2026052702",
			Name:    "tenant_entitlement_expansion_utf8mb4",
			Up: func(tx *gorm.DB) error {
				return convertTenantEntitlementExpansionTablesToUTF8MB4(tx)
			},
		},
		{
			Version: "2026052801",
			Name:    "iot_sales_dashboard_devices",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := seedIoTPermissions(tx); err != nil {
					return err
				}
				return createIoTDeviceIndexes(tx)
			},
		},
		{
			Version: "2026052802",
			Name:    "iot_devices_table_name_fix",
			Up: func(tx *gorm.DB) error {
				if err := ensureIoTDevicesTable(tx); err != nil {
					return err
				}
				if err := seedIoTPermissions(tx); err != nil {
					return err
				}
				return createIoTDeviceIndexes(tx)
			},
		},
		{
			Version: "2026052803",
			Name:    "iot_device_auto_bootstrap",
			Up: func(tx *gorm.DB) error {
				if err := ensureIoTDevicesTable(tx); err != nil {
					return err
				}
				return createIoTDeviceIndexes(tx)
			},
		},
		{
			Version: "2026053001",
			Name:    "iot_device_ota_releases",
			Up: func(tx *gorm.DB) error {
				if err := ensureIoTDevicesTable(tx); err != nil {
					return err
				}
				if err := createIoTDeviceIndexes(tx); err != nil {
					return err
				}
				return createIoTFirmwareIndexes(tx)
			},
		},
		{
			Version: "2026053101",
			Name:    "iot_firmware_targets",
			Up: func(tx *gorm.DB) error {
				if err := ensureIoTDevicesTable(tx); err != nil {
					return err
				}
				if err := createIoTFirmwareIndexes(tx); err != nil {
					return err
				}
				return backfillIoTFirmwareTargets(tx)
			},
		},
		{
			Version: "2026062701",
			Name:    "industry_package_base",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return seedManufacturingIndustryPackage(tx)
			},
		},
		{
			Version: "2026062702",
			Name:    "manufacturing_customer_360",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return seedManufacturingIndustryPackage(tx)
			},
		},
		{
			Version: "2026062703",
			Name:    "manufacturing_business_workflows",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return seedManufacturingIndustryPackage(tx)
			},
		},
		{
			Version: "2026062901",
			Name:    "mfg_rfq_clarification_sources",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return ensureMfgRFQClarificationSourceColumns(tx)
			},
		},
		{
			Version: "2026062902",
			Name:    "mfg_rfq_clarification_answer_history",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return ensureMfgRFQClarificationAnswerAudit(tx)
			},
		},
		{
			Version: "2026062903",
			Name:    "mfg_crm_delivery_change_fields",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := ensureMfgDeliveryChangeFields(tx); err != nil {
					return err
				}
				return disableHiddenManufacturingMenus(tx)
			},
		},
		{
			Version: "2026062904",
			Name:    "mfg_quote_approval_price_permissions",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				if err := seedMfgQuoteApprovalPermissions(tx); err != nil {
					return err
				}
				return backfillMfgQuoteApprovalRolePermissions(tx)
			},
		},
		{
			Version: "2026070101",
			Name:    "mfg_clarification_answerer_name",
			Up: func(tx *gorm.DB) error {
				return ensureMfgRFQClarificationAnswererName(tx)
			},
		},
		{
			Version: "2026070301",
			Name:    "tenant_admin_only_organization_permissions",
			Up: func(tx *gorm.DB) error {
				return stripOrganizationPermissionsFromRoles(tx)
			},
		},
		{
			Version: "2026070302",
			Name:    "mfg_quote_revision_backfill_by_rfq",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return backfillMfgQuoteRevisionsByRFQ(tx)
			},
		},
		{
			Version: "2026070303",
			Name:    "mfg_quote_approval_history_from_operation_logs",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return backfillMfgQuoteApprovalHistoriesFromOperationLogs(tx)
			},
		},
		{
			Version: "2026070401",
			Name:    "mfg_rfq_bom_business_names",
			Up: func(tx *gorm.DB) error {
				if err := autoMigrateWithDB(tx); err != nil {
					return err
				}
				return ensureMfgBusinessNameColumns(tx)
			},
		},
		{
			Version: "2026070402",
			Name:    "mfg_sample_progress_histories",
			Up: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.MfgSampleProgressHistory{})
			},
		},
		{
			Version: "2026070403",
			Name:    "mfg_production_progress_status_history",
			Up: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&models.MfgProductionProgress{}, &models.MfgProductionProgressHistory{}); err != nil {
					return err
				}
				return ensureMfgProductionProgressStatusColumn(tx)
			},
		},
		{
			Version: "2026070404",
			Name:    "mfg_compensation_approval_flow",
			Up: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&models.MfgCompensationRecord{}); err != nil {
					return err
				}
				return ensureMfgCompensationApprovalColumns(tx)
			},
		},
		{
			Version: "2026070405",
			Name:    "mfg_approval_histories",
			Up: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.MfgApprovalHistory{})
			},
		},
	}

	for _, m := range migrations {
		applied, err := isMigrationApplied(m.Version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := DB.Transaction(func(tx *gorm.DB) error {
			if err := m.Up(tx); err != nil {
				return err
			}
			return tx.Table(migrationsTableName).Create(&SchemaMigration{
				Version: m.Version,
				Name:    m.Name,
			}).Error
		}); err != nil {
			return fmt.Errorf("迁移 %s(%s) 执行失败: %w", m.Version, m.Name, err)
		}

		LogInfo("迁移已执行: version=%s name=%s", m.Version, m.Name)
	}

	return nil
}

func seedManufacturingIndustryPackage(tx *gorm.DB) error {
	manifest := manufacturingSeedManifest()
	raw, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	registry := models.IndustryPackageRegistry{
		Code:         "manufacturing",
		Name:         "工业制造 CRM",
		Version:      "1.0",
		Description:  "围绕制造客户、商机、RFQ、报价、订单承诺和售后机会的制造行业 CRM 工作台。",
		ManifestJSON: string(raw),
		Status:       "active",
	}
	if err := tx.Where("code = ?", registry.Code).Assign(registry).FirstOrCreate(&registry).Error; err != nil {
		return err
	}
	return seedManufacturingIndustryExtensions(tx, manifest)
}

func manufacturingSeedManifest() map[string]interface{} {
	return map[string]interface{}{
		"packages": []map[string]interface{}{
			{"code": "manufacturing", "name": "工业制造 CRM", "version": "1.0", "description": "围绕制造客户、商机、RFQ、报价、订单承诺和售后机会的制造行业 CRM 工作台。", "status": "active"},
		},
		"menuExtensions": []map[string]interface{}{
			{"key": "manufacturing", "path": "/manufacturing", "titleKey": "industry.manufacturing.menu.title", "descKey": "industry.manufacturing.menu.desc", "icon": "Monitor", "group": "manufacturing", "order": 40, "permission": "mfg:view", "navMode": "workflow", "children": manufacturingSeedMenuChildren()},
		},
		"pageTabs": map[string]interface{}{
			"customer.detail": []map[string]interface{}{
				{"key": "mfg_profile", "titleKey": "industry.manufacturing.tabs.profile", "component": "MfgCustomerProfileTab", "permission": "mfg:customer:view"},
				{"key": "mfg_decision_chain", "titleKey": "industry.manufacturing.tabs.decisionChain", "component": "MfgDecisionChainTab", "permission": "mfg:customer:view"},
				{"key": "mfg_rfqs", "titleKey": "industry.manufacturing.tabs.rfqs", "component": "MfgRFQTab", "permission": "mfg:rfq:view"},
				{"key": "mfg_samples", "titleKey": "industry.manufacturing.tabs.samples", "component": "MfgSampleTab", "permission": "mfg:sample:view"},
				{"key": "mfg_assets", "titleKey": "industry.manufacturing.tabs.assets", "component": "MfgAssetTab", "permission": "mfg:asset:view"},
				{"key": "mfg_profit", "titleKey": "industry.manufacturing.tabs.profit", "component": "MfgCustomerProfitTab", "permission": "mfg:metrics:view"},
			},
		},
		"objectActions": map[string]interface{}{
			"opportunity": []map[string]interface{}{
				{"key": "create_mfg_rfq", "titleKey": "industry.manufacturing.action.createRFQ", "permission": "mfg:rfq:manage", "component": "CreateMfgRFQAction"},
			},
		},
		"dashboardWidgets": []map[string]interface{}{
			{"key": "mfg_overview", "titleKey": "industry.manufacturing.dashboard.overview", "component": "MfgOverviewWidget", "permission": "mfg:metrics:view"},
		},
		"permissions": []string{"mfg:view", "mfg:customer:view", "mfg:customer:manage", "mfg:agreement:view", "mfg:agreement:manage", "mfg:rfq:view", "mfg:rfq:manage", "mfg:product:view", "mfg:product:manage", "mfg:cpq:view", "mfg:cpq:manage", "mfg:cost:view", "mfg:cost:view_sensitive", "mfg:cost:manage", "mfg:quote:view", "mfg:quote:approve", "mfg:quote:manage", "mfg:sample:view", "mfg:sample:manage", "mfg:production:view", "mfg:production:manage", "mfg:ecn:view", "mfg:ecn:manage", "mfg:asset:view", "mfg:asset:manage", "mfg:service:view", "mfg:service:manage", "mfg:knowledge:view", "mfg:knowledge:manage", "mfg:metrics:view", "mfg:metrics:manage"},
	}
}

func backfillMfgQuoteRevisionsByRFQ(tx *gorm.DB) error {
	type quoteGroup struct {
		TenantID uint
		RFQID    uint
	}
	var groups []quoteGroup
	if err := tx.Model(&models.MfgQuoteRevision{}).
		Select("tenant_id, rfq_id").
		Where("rfq_id > 0").
		Group("tenant_id, rfq_id").
		Having("COUNT(*) > 1").
		Find(&groups).Error; err != nil {
		return err
	}
	for _, group := range groups {
		var quotes []models.MfgQuoteRevision
		if err := tx.Where("tenant_id = ? AND rfq_id = ?", group.TenantID, group.RFQID).
			Order("created_at asc, id asc").
			Find(&quotes).Error; err != nil {
			return err
		}
		for index, quote := range quotes {
			revision := index + 1
			if quote.Revision == revision {
				continue
			}
			if err := tx.Model(&models.MfgQuoteRevision{}).
				Where("id = ?", quote.ID).
				Update("revision", revision).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.MfgQuoteApprovalHistory{}).
				Where("tenant_id = ? AND quote_id = ?", quote.TenantID, quote.ID).
				Update("revision", revision).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func backfillMfgQuoteApprovalHistoriesFromOperationLogs(tx *gorm.DB) error {
	operations := []string{"SUBMIT_MFG_QUOTE_APPROVAL", "APPROVE_MFG_QUOTE", "REJECT_MFG_QUOTE"}
	var logs []models.OperationLog
	if err := tx.Where("operation IN ? AND result = ?", operations, "success").
		Order("created_at asc, id asc").
		Find(&logs).Error; err != nil {
		return err
	}
	for _, log := range logs {
		quoteID := mfgQuoteIDFromOperationTarget(log.Target)
		if quoteID == 0 {
			continue
		}
		action, oldStatus, newStatus := mfgQuoteApprovalHistoryStateFromOperation(log.Operation)
		if action == "" {
			continue
		}
		var quote models.MfgQuoteRevision
		if err := tx.Where("tenant_id = ? AND id = ?", log.TenantID, quoteID).First(&quote).Error; err != nil {
			continue
		}
		params := map[string]interface{}{}
		if strings.TrimSpace(log.ParamsJSON) != "" {
			_ = json.Unmarshal([]byte(log.ParamsJSON), &params)
		}
		if value := strings.TrimSpace(fmt.Sprint(params["approval_status"])); value != "" && value != "<nil>" {
			newStatus = value
		}
		remark := ""
		if value, ok := params["remark"]; ok {
			remark = strings.TrimSpace(fmt.Sprint(value))
		}
		var existing int64
		if err := tx.Model(&models.MfgQuoteApprovalHistory{}).
			Where("tenant_id = ? AND quote_id = ? AND action = ? AND operated_at = ?", log.TenantID, quote.ID, action, log.CreatedAt).
			Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			continue
		}
		history := models.MfgQuoteApprovalHistory{
			TenantID:          quote.TenantID,
			RFQID:             quote.RFQID,
			QuoteID:           quote.ID,
			QuoteNo:           quote.QuoteNo,
			Revision:          quote.Revision,
			Action:            action,
			OldApprovalStatus: oldStatus,
			NewApprovalStatus: newStatus,
			Remark:            remark,
			OperatorID:        log.UserID,
			OperatorName:      strings.TrimSpace(log.Username),
			OperatedAt:        log.CreatedAt,
		}
		if err := tx.Create(&history).Error; err != nil {
			return err
		}
	}
	return nil
}

func mfgQuoteIDFromOperationTarget(target string) uint {
	var id uint
	if _, err := fmt.Sscanf(strings.TrimSpace(target), "mfg/quotes[%d]", &id); err != nil {
		return 0
	}
	return id
}

func mfgQuoteApprovalHistoryStateFromOperation(operation string) (string, string, string) {
	switch strings.ToUpper(strings.TrimSpace(operation)) {
	case "SUBMIT_MFG_QUOTE_APPROVAL":
		return "submit", "draft", "pending"
	case "APPROVE_MFG_QUOTE":
		return "approve", "pending", "approved"
	case "REJECT_MFG_QUOTE":
		return "reject", "pending", "rejected"
	default:
		return "", "", ""
	}
}

func manufacturingSeedMenuChildren() []map[string]interface{} {
	return []map[string]interface{}{
		{"key": "mfg_quote_to_order", "path": "/manufacturing/quote-to-order", "titleKey": "industry.manufacturing.workflow.quoteToOrder", "descKey": "industry.manufacturing.workflow.quoteToOrderDesc", "permission": "mfg:rfq:view", "order": 2, "navMode": "workflow", "workflowGroup": "quote_to_order", "beginnerVisible": true, "legacyModules": []string{"rfqs", "clarifications", "drawings", "config-models", "bom-snapshots", "cost-sheets", "quotes"}},
		{"key": "mfg_sample_to_mass", "path": "/manufacturing/sample-to-mass", "titleKey": "industry.manufacturing.workflow.sampleToMass", "descKey": "industry.manufacturing.workflow.sampleToMassDesc", "permission": "mfg:sample:view", "order": 3, "navMode": "workflow", "workflowGroup": "sample_to_mass", "beginnerVisible": true, "legacyModules": []string{"samples", "production"}},
		{"key": "mfg_delivery_change", "path": "/manufacturing/delivery-change", "titleKey": "industry.manufacturing.workflow.deliveryChange", "descKey": "industry.manufacturing.workflow.deliveryChangeDesc", "permission": "mfg:production:view", "order": 4, "navMode": "workflow", "workflowGroup": "delivery_change", "beginnerVisible": true, "legacyModules": []string{"ecns", "compensation-records"}},
		{"key": "mfg_settings", "path": "/manufacturing/settings", "titleKey": "industry.manufacturing.workflow.settings", "descKey": "industry.manufacturing.workflow.settingsDesc", "permission": "mfg:product:view", "order": 7, "navMode": "workflow", "workflowGroup": "settings", "beginnerVisible": true, "legacyModules": []string{"product-families", "bom-templates"}},
	}
}

func seedManufacturingIndustryExtensions(tx *gorm.DB, manifest map[string]interface{}) error {
	menus, _ := manifest["menuExtensions"].([]map[string]interface{})
	for _, menu := range menus {
		if err := upsertIndustryExtension(tx, "manufacturing", "menu", "global.nav", fmt.Sprint(menu["key"]), menu, intMapValue(menu, "order")); err != nil {
			return err
		}
	}
	pageTabs, _ := manifest["pageTabs"].(map[string]interface{})
	for target, value := range pageTabs {
		tabs, _ := value.([]map[string]interface{})
		for index, tab := range tabs {
			if err := upsertIndustryExtension(tx, "manufacturing", "page_tab", target, fmt.Sprint(tab["key"]), tab, index+1); err != nil {
				return err
			}
		}
	}
	actionsByTarget, _ := manifest["objectActions"].(map[string]interface{})
	for target, value := range actionsByTarget {
		actions, _ := value.([]map[string]interface{})
		for index, action := range actions {
			if err := upsertIndustryExtension(tx, "manufacturing", "object_action", target, fmt.Sprint(action["key"]), action, index+1); err != nil {
				return err
			}
		}
	}
	widgets, _ := manifest["dashboardWidgets"].([]map[string]interface{})
	for index, widget := range widgets {
		if err := upsertIndustryExtension(tx, "manufacturing", "dashboard_widget", "dashboard", fmt.Sprint(widget["key"]), widget, index+1); err != nil {
			return err
		}
	}
	return nil
}

func upsertIndustryExtension(tx *gorm.DB, packageCode, extensionType, targetKey, extensionKey string, config map[string]interface{}, sort int) error {
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	extension := models.IndustryExtensionRegistry{
		PackageCode:   packageCode,
		ExtensionType: extensionType,
		TargetKey:     targetKey,
		ExtensionKey:  extensionKey,
		ConfigJSON:    string(raw),
		Sort:          sort,
		Enabled:       true,
	}
	return tx.Where("package_code = ? AND extension_type = ? AND target_key = ? AND extension_key = ?", packageCode, extensionType, targetKey, extensionKey).
		Assign(extension).
		FirstOrCreate(&extension).Error
}

func intMapValue(values map[string]interface{}, key string) int {
	switch typed := values[key].(type) {
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

func ensureNewPRDDefaultTenant(tx *gorm.DB) error {
	tenant := models.Tenant{
		Name:      "Utopia CRM 演示企业",
		Status:    "active",
		QuotaJSON: mustJSON(map[string]interface{}{"max_users": 50, "modules": []string{"workspace", "enterprise", "projects", "products", "campaigns", "contacts", "leads", "opportunities", "deals", "orders", "tickets", "dashboards", "teams", "insights", "accounts"}}),
		Remark:    "newprd.md 默认业务租户",
	}
	if err := tx.Where("name = ?", tenant.Name).FirstOrCreate(&tenant).Error; err != nil {
		return err
	}
	quota := models.TenantQuota{
		TenantID:      tenant.ID,
		MaxUsers:      50,
		StorageLimit:  1024 * 1024 * 1024,
		Modules:       "workspace,enterprise,projects,products,campaigns,contacts,leads,opportunities,deals,orders,tickets,dashboards,teams,insights,accounts",
		FeatureSwitch: mustJSON(map[string]interface{}{"new_prd_only": true}),
	}
	if err := tx.Where("tenant_id = ?", tenant.ID).Assign(quota).FirstOrCreate(&quota).Error; err != nil {
		return err
	}
	adminRole := models.Role{
		TenantModel:     models.TenantModel{TenantID: tenant.ID},
		Name:            "管理员",
		Description:     "newprd.md 全模块管理员",
		IsPreset:        true,
		PermissionsJSON: "workspace:view,workspace:create,workspace:update,workspace:delete,workspace:export,workspace:approve,workspace:manage,enterprise:view,enterprise:create,enterprise:update,enterprise:delete,enterprise:export,enterprise:manage,projects:view,projects:create,projects:update,projects:delete,projects:export,projects:manage,products:view,products:create,products:update,products:delete,products:export,products:manage,campaigns:view,campaigns:create,campaigns:update,campaigns:delete,campaigns:export,campaigns:manage,contacts:view,contacts:create,contacts:update,contacts:delete,contacts:export,contacts:manage,leads:view,leads:create,leads:update,leads:delete,leads:export,leads:assign,leads:manage,opportunities:view,opportunities:create,opportunities:update,opportunities:delete,opportunities:export,opportunities:manage,deals:view,deals:create,deals:update,deals:delete,deals:export,deals:manage,orders:view,orders:create,orders:update,orders:delete,orders:export,orders:manage,tickets:view,tickets:create,tickets:update,tickets:delete,tickets:export,tickets:assign,tickets:manage,dashboards:view,dashboards:create,dashboards:update,dashboards:delete,dashboards:export,teams:view,teams:create,teams:update,teams:delete,teams:export,teams:manage,insights:view,insights:create,insights:update,insights:delete,insights:export,insights:manage,accounts:view,accounts:create,accounts:update,accounts:delete,accounts:export,accounts:manage,audit_log:view,audit_log:export",
	}
	if err := tx.Where("tenant_id = ? AND name = ?", tenant.ID, adminRole.Name).Assign(adminRole).FirstOrCreate(&adminRole).Error; err != nil {
		return err
	}
	return nil
}

func seedNewPRDSampleData(tx *gorm.DB) error {
	var tenant models.Tenant
	if err := tx.Where("status = ?", "active").Order("id asc").First(&tenant).Error; err != nil {
		return err
	}
	var count int64
	if err := tx.Model(&models.CRMObject{}).Where("tenant_id = ?", tenant.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now()
	product := crmSeedObject(tenant.ID, "products", "product", "Revenue Intelligence 套件", "SKU-RI-001", "on_sale", 128000, "CNY", 82, "ai,subscription", mustJSON(map[string]interface{}{"sku": "RI-001", "category": "Halmify CRM/Revenue/Intelligence", "unit": "套", "stock": 120, "stock_threshold": 10, "tax_rate": 0.06, "standard_price": 128000}))
	seed := map[string]*models.CRMObject{
		"company_config": {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "enterprise", ObjectType: "company_config", Name: "Utopia CRM 演示企业", Code: "ENT-001", Status: "active", Currency: "CNY", Tags: "system,config", PayloadJSON: mustJSON(map[string]interface{}{"full_name": "Utopia CRM 演示企业", "short_name": "Utopia CRM", "timezone": "Asia/Shanghai", "currency": "CNY", "date_format": "YYYY-MM-DD", "language": "zh-CN"})},
		"department":     {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "enterprise", ObjectType: "department", Name: "销售一部", Code: "DEPT-SALES-01", Status: "active", Currency: "CNY", Tags: "sales,team", PayloadJSON: mustJSON(map[string]interface{}{"department_code": "SALES-01", "manager": "李经理", "hr_sync": "manual"})},
		"product":        &product,
		"campaign":       {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "campaigns", ObjectType: "campaign", Name: "制造业增长直播", Code: "CAM-2026-01", Status: "running", Amount: 35000, Currency: "CNY", Score: 74, Tags: "webinar,manufacturing", PayloadJSON: mustJSON(map[string]interface{}{"type": "webinar", "budget_items": []map[string]interface{}{{"name": "投放", "amount": 18000}, {"name": "设计", "amount": 7000}}, "utm_source": "webinar_mfg", "owner": "市场部"})},
		"company":        {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "contacts", ObjectType: "company", Name: "星河制造集团", Code: "COM-XH-001", Status: "active", Currency: "CNY", Score: 91, Tags: "VIP,manufacturing", PayloadJSON: mustJSON(map[string]interface{}{"industry": "制造业", "source": "HubSpot", "open_deal_amount": 860000, "recent_interaction": "2026-05-12", "grade": "VIP"})},
		"contact":        {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "contacts", ObjectType: "contact", Name: "周明", Code: "CT-001", Status: "active", Currency: "CNY", Score: 88, Tags: "decision-maker,VIP", PayloadJSON: mustJSON(map[string]interface{}{"title": "数字化负责人", "phone": "13800001111", "email": "zhouming@example.com", "company": "星河制造集团", "source": "manual", "content_preference": "ROI案例"})},
		"lead":           {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "leads", ObjectType: "lead", Name: "华东制造业白皮书线索", Code: "LEAD-001", Status: "qualified", Currency: "CNY", Score: 86, Tags: "utm,hot", PayloadJSON: mustJSON(map[string]interface{}{"email": "lead@example.com", "utm_source": "whitepaper", "intent": "high", "product_page_visits": 7, "next_action": "安排顾问演示"})},
		"opportunity":    {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "opportunities", ObjectType: "opportunity", Name: "星河制造集团数字化升级", Code: "OPP-001", Status: "proposal", Amount: 860000, Currency: "CNY", Score: 78, Tags: "commit,risk-watch", PayloadJSON: mustJSON(map[string]interface{}{"stage": "proposal", "probability": 0.65, "forecast_category": "commit", "next_activity_date": "2026-05-16", "proposal_url": "https://example.com/proposal/xh", "competitors": []string{"竞品A"}})},
		"deal":           {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "deals", ObjectType: "closed_deal", Name: "南方零售集团首年订阅", Code: "DEAL-001", Status: "closed_won", Amount: 420000, Currency: "CNY", Score: 95, Tags: "won,subscription", PayloadJSON: mustJSON(map[string]interface{}{"contract_no": "CN-2026-0501", "invoice_no": "INV-2026-0501", "signed_at": "2026-05-01", "owner_split": []map[string]interface{}{{"owner": "王销售", "ratio": 70}, {"owner": "李顾问", "ratio": 30}}})},
		"order":          {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "orders", ObjectType: "order", Name: "南方零售集团实施订单", Code: "ORD-001", Status: "paid", Amount: 420000, Currency: "CNY", Score: 80, Tags: "implementation", PayloadJSON: mustJSON(map[string]interface{}{"payment_status": "paid", "shipment_no": "ONLINE-DELIVERY", "invoice_type": "special_vat", "line_items": []map[string]interface{}{{"quantity": 1, "unit_price": 420000}}})},
		"project":        {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "projects", ObjectType: "project", Name: "南方零售集团上线项目", Code: "PRJ-001", Status: "active", Amount: 420000, Currency: "CNY", Score: 68, Tags: "delivery,customer-visible", PayloadJSON: mustJSON(map[string]interface{}{"manager": "赵项目", "progress": 62, "customer_visible": true, "milestones": []map[string]interface{}{{"name": "需求确认", "status": "done"}, {"name": "试点上线", "status": "doing"}}})},
		"ticket":         {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "tickets", ObjectType: "ticket", Name: "VIP客户接口延迟", Code: "TKT-001", Status: "processing", Currency: "CNY", Score: 45, Tags: "VIP,SLA", PayloadJSON: mustJSON(map[string]interface{}{"category": "complaint", "priority": "high", "customer_grade": "VIP", "sla_due_at": now.Add(4 * time.Hour).Format(time.RFC3339), "public_reply": "已定位为上游接口波动，正在加固重试策略。"})},
		"risk":           {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "insights", ObjectType: "deal_risk", Name: "星河制造集团交易风险", Code: "RISK-001", Status: "watching", Amount: 860000, Currency: "CNY", Score: 70, Tags: "risk,manager-alert", PayloadJSON: mustJSON(map[string]interface{}{"reason": "最后互动超过14天，方案阶段未更新", "suggestion": "由销售经理介入并安排下一次方案复盘", "risk_score": 70})},
		"faq":            {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "insights", ObjectType: "faq_cluster", Name: "接口延迟高频问题", Code: "FAQ-001", Status: "open", Currency: "CNY", Score: 63, Tags: "knowledge,ticket", PayloadJSON: mustJSON(map[string]interface{}{"question": "接口延迟如何处理", "answer": "请先查看状态页，确认调用重试策略，并在工单中附上 request_id 便于排查。", "source": "ticket_cluster"})},
		"task":           {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "workspace", ObjectType: "task", Name: "跟进星河制造集团风险交易", Code: "TASK-001", Status: "open", Currency: "CNY", Score: 90, Tags: "today,manager", PayloadJSON: mustJSON(map[string]interface{}{"source": "deal_risk", "due_at": now.Add(24 * time.Hour).Format(time.RFC3339), "priority": "high"})},
		"dashboard":      {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "dashboards", ObjectType: "dashboard", Name: "销售经营总览", Code: "DASH-001", Status: "active", Currency: "CNY", Tags: "sales,executive", PayloadJSON: mustJSON(map[string]interface{}{"widgets": []string{"revenue", "funnel", "target", "risk"}, "refresh_seconds": 30})},
		"team_member":    {TenantModel: models.TenantModel{TenantID: tenant.ID}, Domain: "teams", ObjectType: "team_member", Name: "王销售", Code: "TM-001", Status: "active", Currency: "CNY", Tags: "sales,enterprise", PayloadJSON: mustJSON(map[string]interface{}{"email": "wang.sales@example.com", "skills": []string{"大客户", "制造业"}, "max_load": 25, "current_load": 14})},
	}
	orderKeys := []string{"company_config", "department", "product", "campaign", "company", "contact", "lead", "opportunity", "deal", "order", "project", "ticket", "risk", "faq", "task", "dashboard", "team_member"}
	for _, key := range orderKeys {
		if err := tx.Create(seed[key]).Error; err != nil {
			return err
		}
	}
	orderPayload := map[string]interface{}{"payment_status": "paid", "shipment_no": "ONLINE-DELIVERY", "invoice_type": "special_vat", "line_items": []map[string]interface{}{{"product_id": seed["product"].ID, "quantity": 1, "unit_price": 420000}}}
	if err := tx.Model(seed["order"]).Update("payload_json", mustJSON(orderPayload)).Error; err != nil {
		return err
	}
	riskPayload := map[string]interface{}{"opportunity_id": seed["opportunity"].ID, "reason": "最后互动超过14天，方案阶段未更新", "suggestion": "由销售经理介入并安排下一次方案复盘", "risk_score": 70}
	if err := tx.Model(seed["risk"]).Update("payload_json", mustJSON(riskPayload)).Error; err != nil {
		return err
	}

	relations := []models.CRMRelation{
		crmSeedRelation(tenant.ID, "contacts", seed["company"].ID, "contacts", seed["contact"].ID, "has_contact"),
		crmSeedRelation(tenant.ID, "campaigns", seed["campaign"].ID, "leads", seed["lead"].ID, "generated_lead"),
		crmSeedRelation(tenant.ID, "leads", seed["lead"].ID, "opportunities", seed["opportunity"].ID, "converted_to"),
		crmSeedRelation(tenant.ID, "opportunities", seed["opportunity"].ID, "insights", seed["risk"].ID, "risk_alert"),
		crmSeedRelation(tenant.ID, "deals", seed["deal"].ID, "orders", seed["order"].ID, "generated_order"),
		crmSeedRelation(tenant.ID, "orders", seed["order"].ID, "projects", seed["project"].ID, "generated_project"),
		crmSeedRelation(tenant.ID, "tickets", seed["ticket"].ID, "insights", seed["faq"].ID, "generated_knowledge"),
	}
	return tx.Create(&relations).Error
}

func seedNewPRDSettings(tx *gorm.DB) error {
	var tenant models.Tenant
	if err := tx.Where("status = ?", "active").Order("id asc").First(&tenant).Error; err != nil {
		return err
	}
	settings := []models.CRMSetting{
		{TenantModel: models.TenantModel{TenantID: tenant.ID}, Scope: "company", Key: "enterprise", ValueJSON: mustJSON(map[string]interface{}{"full_name": "Utopia CRM 演示企业", "short_name": "Utopia CRM", "timezone": "Asia/Shanghai", "currency": "CNY", "date_format": "YYYY-MM-DD", "language": "zh-CN"})},
		{TenantModel: models.TenantModel{TenantID: tenant.ID}, Scope: "sales_process", Key: "opportunities", ValueJSON: mustJSON(map[string]interface{}{"stages": []string{"pipeline", "proposal", "negotiation", "won", "lost"}, "stage_required_fields": map[string]interface{}{"proposal": []string{"proposal_url"}, "negotiation": []string{"quote_amount"}}})},
		{TenantModel: models.TenantModel{TenantID: tenant.ID}, Scope: "data_scope", Key: "leads", ValueJSON: mustJSON(map[string]interface{}{"mode": "owner", "piercing_user_ids": []uint{}})},
		{TenantModel: models.TenantModel{TenantID: tenant.ID}, Scope: "data_scope", Key: "contacts", ValueJSON: mustJSON(map[string]interface{}{"mode": "shared", "piercing_user_ids": []uint{}})},
		{TenantModel: models.TenantModel{TenantID: tenant.ID}, Scope: "field_acl", Key: "contacts", ValueJSON: mustJSON(map[string]interface{}{"hidden": []string{"cost_price"}, "readonly": []string{"email"}, "bypass_user_ids": []uint{}})},
		{TenantModel: models.TenantModel{TenantID: tenant.ID}, Scope: "sla_rule", Key: "tickets", ValueJSON: mustJSON(map[string]interface{}{"default_hours": 24, "matrix": map[string]interface{}{"complaint:VIP": 4, "complaint": 8, "repair": 24}})},
		{TenantModel: models.TenantModel{TenantID: tenant.ID}, Scope: "mfa_policy", Key: "accounts", ValueJSON: mustJSON(map[string]interface{}{"required": false, "methods": []string{"totp", "sms"}})},
	}
	for _, setting := range settings {
		if err := tx.Where("tenant_id = ? AND scope = ? AND `key` = ?", tenant.ID, setting.Scope, setting.Key).Assign(setting).FirstOrCreate(&setting).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedNewPRDCompleteSamples(tx *gorm.DB) error {
	var tenant models.Tenant
	if err := tx.Where("status = ?", "active").Order("id asc").First(&tenant).Error; err != nil {
		return err
	}
	samples := []models.CRMObject{
		crmSeedObject(tenant.ID, "enterprise", "sales_process", "标准销售流程", "PROC-STD", "active", 0, "CNY", 0, "process", mustJSON(map[string]interface{}{"stages": []string{"pipeline", "proposal", "negotiation", "won"}, "stage_required_fields": map[string]interface{}{"proposal": []string{"proposal_url"}, "negotiation": []string{"quote_amount"}}})),
		crmSeedObject(tenant.ID, "enterprise", "field_permission", "联系人字段权限", "ACL-CONTACT", "active", 0, "CNY", 0, "acl", mustJSON(map[string]interface{}{"domain": "contacts", "hidden": []string{"cost_price"}, "readonly": []string{"email"}})),
		crmSeedObject(tenant.ID, "enterprise", "data_scope_rule", "销售本人数据隔离", "SCOPE-OWNER", "active", 0, "CNY", 0, "scope", mustJSON(map[string]interface{}{"mode": "owner", "piercing_user_ids": []uint{}})),
		crmSeedObject(tenant.ID, "projects", "milestone", "上线里程碑", "MS-GO-LIVE", "planning", 0, "CNY", 0, "milestone", mustJSON(map[string]interface{}{"planned_date": "2026-06-01", "completion_rule": "manual", "warning_days": 3})),
		crmSeedObject(tenant.ID, "projects", "project_view", "交付甘特视图", "VIEW-GANTT", "active", 0, "CNY", 0, "view", mustJSON(map[string]interface{}{"view_type": "gantt", "template_scope": "team"})),
		crmSeedObject(tenant.ID, "projects", "deliverable", "上线方案 V1.0", "DELIV-001", "active", 0, "CNY", 0, "deliverable", mustJSON(map[string]interface{}{"file_name": "上线方案.pdf", "version": "V1.0", "customer_downloadable": true})),
		crmSeedObject(tenant.ID, "projects", "customer_space", "星河客户协同空间", "SPACE-XH", "active", 0, "CNY", 0, "portal", mustJSON(map[string]interface{}{"share_url": "/portal/xh", "visible_sections": []string{"milestones", "deliverables"}})),
		crmSeedObject(tenant.ID, "projects", "project_deal_rule", "上线推动续约谈判", "PDR-001", "active", 0, "CNY", 0, "automation", mustJSON(map[string]interface{}{"milestone": "上线", "target_stage": "negotiation"})),
		crmSeedObject(tenant.ID, "products", "category", "Halmify CRM", "CAT-AI", "active", 0, "CNY", 0, "catalog", mustJSON(map[string]interface{}{"level": 1, "parent": ""})),
		crmSeedObject(tenant.ID, "products", "service_package", "企业实施套餐", "PKG-ENT", "on_sale", 98000, "CNY", 0, "package", mustJSON(map[string]interface{}{"billing_type": "one_time", "base_items": []string{"需求调研", "上线培训"}, "addons": []string{"高级支持"}})),
		crmSeedObject(tenant.ID, "products", "pricing_rule", "年度订阅阶梯价", "PRICE-TIER", "active", 0, "CNY", 0, "pricing", mustJSON(map[string]interface{}{"price_type": "tier", "standard_price": 128000, "tiers": []map[string]interface{}{{"min": 1, "max": 5, "price": 128000}, {"min": 6, "price": 108000}}})),
		crmSeedObject(tenant.ID, "products", "variant", "Revenue Intelligence 100席", "VAR-RI-100", "on_sale", 188000, "CNY", 0, "variant", mustJSON(map[string]interface{}{"parent_sku": "RI-001", "seats": 100, "stock": 30})),
		crmSeedObject(tenant.ID, "products", "inventory", "上海仓 RI 库存", "INV-SH-RI", "active", 0, "CNY", 0, "inventory", mustJSON(map[string]interface{}{"warehouse": "上海仓", "sku": "RI-001", "stock": 120, "stock_threshold": 10})),
		crmSeedObject(tenant.ID, "products", "order_line", "订单产品行模板", "OL-RI", "active", 128000, "CNY", 0, "order-line", mustJSON(map[string]interface{}{"sku": "RI-001", "quantity": 1, "unit_price": 128000, "tax_rate": 0.06})),
		crmSeedObject(tenant.ID, "campaigns", "campaign_template", "直播活动模板", "TPL-WEBINAR", "active", 0, "CNY", 0, "template", mustJSON(map[string]interface{}{"layout": "webinar", "email_template": "报名确认邮件", "variables": []string{"活动名称", "日期"}})),
		crmSeedObject(tenant.ID, "campaigns", "campaign_budget", "直播预算 ROI", "ROI-WEBINAR", "active", 35000, "CNY", 0, "roi", mustJSON(map[string]interface{}{"cost": 35000, "deal_amount": 180000, "roi": 4.14, "cpl": 210})),
		crmSeedObject(tenant.ID, "campaigns", "campaign_task", "活动前发布推文", "CAM-TASK-001", "scheduled", 0, "CNY", 0, "task", mustJSON(map[string]interface{}{"owner": "市场运营", "due_at": "2026-05-20", "template_generated": true})),
		crmSeedObject(tenant.ID, "campaigns", "participant", "直播报名：陈总", "PART-001", "active", 0, "CNY", 62, "participant", mustJSON(map[string]interface{}{"email": "chen@example.com", "checked_in": true, "questions": []string{"价格体系"}})),
		crmSeedObject(tenant.ID, "campaigns", "attribution", "直播首次归因", "ATTR-WEBINAR", "active", 180000, "CNY", 0, "attribution", mustJSON(map[string]interface{}{"model": "first_touch", "campaign_id": 1, "won_amount": 180000})),
		crmSeedObject(tenant.ID, "contacts", "interaction", "报价邮件已打开", "INT-001", "active", 0, "CNY", 0, "timeline", mustJSON(map[string]interface{}{"interaction_type": "email", "summary": "客户打开报价邮件", "happened_at": "2026-05-12T10:00:00+08:00"})),
		crmSeedObject(tenant.ID, "contacts", "customer_grade", "VIP 分级规则", "GRADE-VIP", "active", 0, "CNY", 0, "grade", mustJSON(map[string]interface{}{"rule": "累计交易额>100000", "level": "VIP", "sla": "1h/4h"})),
		crmSeedObject(tenant.ID, "contacts", "profile", "星河客户画像", "PROFILE-XH", "active", 0, "CNY", 88, "profile", mustJSON(map[string]interface{}{"active_time": "10:00-12:00", "content_preference": "ROI案例", "predicted_nps": 8})),
		crmSeedObject(tenant.ID, "contacts", "anniversary", "客户成立日提醒", "ANN-001", "active", 0, "CNY", 0, "reminder", mustJSON(map[string]interface{}{"type": "公司成立日", "date": "06-18", "notify_days": 7})),
		crmSeedObject(tenant.ID, "contacts", "import_export", "联系人导入任务", "IMP-CONTACT", "active", 0, "CNY", 0, "import", mustJSON(map[string]interface{}{"dedupe_by": "email", "error_rows": 0, "retention_days": 30})),
		crmSeedObject(tenant.ID, "contacts", "dedupe_job", "邮箱重复检测", "DEDUPE-001", "active", 0, "CNY", 0, "dedupe", mustJSON(map[string]interface{}{"strategy": "email_phone", "undo_supported": true})),
		crmSeedObject(tenant.ID, "deals", "revenue_line", "一次性实施收入", "REV-001", "closed_won", 98000, "CNY", 0, "revenue", mustJSON(map[string]interface{}{"revenue_type": "one_time", "base_currency_amount": 98000})),
		crmSeedObject(tenant.ID, "deals", "deal_date", "合同关键日期", "DATE-001", "active", 0, "CNY", 0, "date", mustJSON(map[string]interface{}{"signed_at": "2026-05-01", "effective_at": "2026-05-10", "paid_at": "2026-05-12"})),
		crmSeedObject(tenant.ID, "deals", "owner_split", "业绩分成", "SPLIT-001", "active", 0, "CNY", 0, "split", mustJSON(map[string]interface{}{"owners": []map[string]interface{}{{"owner": "王销售", "ratio": 70}, {"owner": "李顾问", "ratio": 30}}})),
		crmSeedObject(tenant.ID, "deals", "contract_invoice", "合同发票关联", "CI-001", "active", 0, "CNY", 0, "contract", mustJSON(map[string]interface{}{"contract_no": "CN-2026-0501", "invoice_no": "INV-2026-0501"})),
		crmSeedObject(tenant.ID, "deals", "deal_timeline", "成交阶段时间线", "DTL-001", "active", 0, "CNY", 0, "timeline", mustJSON(map[string]interface{}{"stages": []map[string]interface{}{{"stage": "lead", "at": "2026-04-01"}, {"stage": "won", "at": "2026-05-01"}}})),
		crmSeedObject(tenant.ID, "deals", "loss_reason", "输单原因样例", "LOSS-001", "closed_lost", 0, "CNY", 0, "loss", mustJSON(map[string]interface{}{"primary": "价格", "secondary": "预算不足", "competitor": "竞品A"})),
		crmSeedObject(tenant.ID, "leads", "assignment_rule", "制造业轮询分配", "ASSIGN-001", "active", 0, "CNY", 0, "rule", mustJSON(map[string]interface{}{"strategy": "load_balance", "industry": "制造业"})),
		crmSeedObject(tenant.ID, "leads", "recycle_rule", "48小时未跟进回收", "RECYCLE-001", "active", 0, "CNY", 0, "rule", mustJSON(map[string]interface{}{"inactive_hours": 48, "notify_manager": true})),
		crmSeedObject(tenant.ID, "leads", "lead_score", "AI评分规则", "SCORE-001", "active", 0, "CNY", 0, "score", mustJSON(map[string]interface{}{"inputs": []string{"email_domain", "title", "visits", "form_completeness"}, "threshold": 60})),
		crmSeedObject(tenant.ID, "leads", "follow_sequence", "制造业跟进序列", "SEQ-MFG", "active", 0, "CNY", 0, "sequence", mustJSON(map[string]interface{}{"steps": []map[string]interface{}{{"action": "email", "delay_days": 0}, {"action": "call", "delay_days": 2}}})),
		crmSeedObject(tenant.ID, "leads", "nurture_rule", "白皮书培育规则", "NURTURE-001", "active", 0, "CNY", 0, "nurture", mustJSON(map[string]interface{}{"trigger": "download_whitepaper", "delay_days": 3, "action": "send_case"})),
		crmSeedObject(tenant.ID, "leads", "conversion", "转化配置", "CONVERT-001", "active", 0, "CNY", 0, "conversion", mustJSON(map[string]interface{}{"create_contact": true, "create_opportunity": true, "copy_interactions": true})),
		crmSeedObject(tenant.ID, "dashboards", "leaderboard", "团队业绩排行", "LB-001", "active", 0, "CNY", 0, "rank", mustJSON(map[string]interface{}{"metric": "won_amount", "scope": "team"})),
		crmSeedObject(tenant.ID, "dashboards", "funnel", "销售漏斗", "FUNNEL-001", "active", 0, "CNY", 0, "funnel", mustJSON(map[string]interface{}{"stages": []map[string]interface{}{{"stage": "pipeline", "count": 12}, {"stage": "proposal", "count": 5}}})),
		crmSeedObject(tenant.ID, "dashboards", "target_progress", "本月目标进度", "TARGET-001", "active", 0, "CNY", 0, "target", mustJSON(map[string]interface{}{"target": 1000000, "actual": 420000, "progress": 0.42})),
		crmSeedObject(tenant.ID, "dashboards", "trend_chart", "成交趋势", "TREND-001", "active", 0, "CNY", 0, "trend", mustJSON(map[string]interface{}{"chart_type": "line", "dimension": "date"})),
		crmSeedObject(tenant.ID, "dashboards", "big_screen", "销售大屏轮播", "BIG-001", "active", 0, "CNY", 0, "screen", mustJSON(map[string]interface{}{"rotation_seconds": 30, "pages": []string{"sales", "service"}})),
		crmSeedObject(tenant.ID, "accounts", "user_account", "租户管理员账号策略", "ACC-USER", "active", 0, "CNY", 0, "account", mustJSON(map[string]interface{}{"sso": "OIDC", "ad_sync": true})),
		crmSeedObject(tenant.ID, "accounts", "role", "销售经理角色模板", "ROLE-SALES-MGR", "active", 0, "CNY", 0, "role", mustJSON(map[string]interface{}{"permissions": []string{"contacts:view", "opportunities:manage"}})),
		crmSeedObject(tenant.ID, "accounts", "user_role", "成员角色矩阵", "UR-001", "active", 0, "CNY", 0, "role", mustJSON(map[string]interface{}{"user": "王销售", "roles": []string{"销售"}})),
		crmSeedObject(tenant.ID, "accounts", "field_acl", "手机号只读策略", "FACL-001", "active", 0, "CNY", 0, "acl", mustJSON(map[string]interface{}{"domain": "contacts", "readonly": []string{"phone"}})),
		crmSeedObject(tenant.ID, "accounts", "mfa_policy", "MFA策略", "MFA-001", "active", 0, "CNY", 0, "security", mustJSON(map[string]interface{}{"required": false, "methods": []string{"totp", "sms"}})),
		crmSeedObject(tenant.ID, "accounts", "login_log", "登录日志保留策略", "LOGINLOG-001", "active", 0, "CNY", 0, "audit", mustJSON(map[string]interface{}{"retention_days": 180, "exportable": true})),
		crmSeedObject(tenant.ID, "accounts", "audit_log_policy", "操作审计策略", "AUDIT-001", "active", 0, "CNY", 0, "audit", mustJSON(map[string]interface{}{"retention_days": 180, "immutable": true})),
		crmSeedObject(tenant.ID, "teams", "reporting_line", "销售一部汇报线", "RL-001", "active", 0, "CNY", 0, "org", mustJSON(map[string]interface{}{"manager": "李经理", "members": []string{"王销售"}})),
		crmSeedObject(tenant.ID, "teams", "role_matrix", "成员角色矩阵", "RM-001", "active", 0, "CNY", 0, "role", mustJSON(map[string]interface{}{"rows": []map[string]interface{}{{"member": "王销售", "roles": []string{"销售代表"}}}})),
		crmSeedObject(tenant.ID, "teams", "sales_goal", "Q2销售目标", "GOAL-Q2", "active", 1000000, "CNY", 0, "goal", mustJSON(map[string]interface{}{"period": "2026Q2", "target": 1000000, "weight": 1.0})),
		crmSeedObject(tenant.ID, "teams", "workload", "销售负载视图", "LOAD-001", "active", 0, "CNY", 0, "load", mustJSON(map[string]interface{}{"member": "王销售", "deals": 8, "tasks": 12, "lead_following": 14, "max_load": 25})),
		crmSeedObject(tenant.ID, "insights", "hot_customer", "热点客户排行", "HOT-CUST", "watching", 0, "CNY", 86, "hot", mustJSON(map[string]interface{}{"customer": "星河制造集团", "interactions_30d": 18, "task_created": true})),
		crmSeedObject(tenant.ID, "insights", "hot_product", "热点产品排行", "HOT-PROD", "watching", 0, "CNY", 78, "hot", mustJSON(map[string]interface{}{"sku": "RI-001", "downloads": 56})),
		crmSeedObject(tenant.ID, "insights", "behavior_heatmap", "销售行为热力图", "HEAT-001", "active", 0, "CNY", 0, "heatmap", mustJSON(map[string]interface{}{"best_hour": "10:00", "matrix": [][]int{{1, 3, 8}, {2, 5, 9}}})),
		crmSeedObject(tenant.ID, "insights", "content_hotspot", "白皮书内容热度", "CONTENT-HOT", "active", 0, "CNY", 0, "content", mustJSON(map[string]interface{}{"asset": "ROI白皮书", "downloads": 126, "tag": "内容活跃"})),
		crmSeedObject(tenant.ID, "insights", "touch_attribution", "多触点归因", "ATTR-MULTI", "active", 420000, "CNY", 0, "attribution", mustJSON(map[string]interface{}{"model": "linear", "touches": []string{"活动", "邮件", "销售电话"}})),
		crmSeedObject(tenant.ID, "opportunities", "pipeline", "标准销售管道", "PIPE-001", "active", 0, "CNY", 0, "pipeline", mustJSON(map[string]interface{}{"stages": []string{"pipeline", "proposal", "negotiation", "won", "lost"}})),
		crmSeedObject(tenant.ID, "opportunities", "stage_gate", "报价阶段门禁", "GATE-001", "active", 0, "CNY", 0, "gate", mustJSON(map[string]interface{}{"stage": "proposal", "required_fields": []string{"proposal_url"}})),
		crmSeedObject(tenant.ID, "opportunities", "forecast", "承诺预测", "FC-001", "active", 860000, "CNY", 0, "forecast", mustJSON(map[string]interface{}{"category": "commit", "weighted_amount": 559000, "expected_close": "2026-06-20"})),
		crmSeedObject(tenant.ID, "opportunities", "competition_note", "竞品备注", "COMP-001", "active", 0, "CNY", 0, "competition", mustJSON(map[string]interface{}{"competitor": "竞品A", "advantage": "价格低", "strategy": "强调ROI"})),
		crmSeedObject(tenant.ID, "opportunities", "deal_activity", "客户电话记录", "ACT-OPP", "active", 0, "CNY", 0, "activity", mustJSON(map[string]interface{}{"activity_type": "call", "summary": "讨论预算和上线时间"})),
		crmSeedObject(tenant.ID, "opportunities", "revenue_intelligence", "Revenue Intelligence 建议", "RI-INSIGHT", "active", 0, "CNY", 0, "ai", mustJSON(map[string]interface{}{"suggestion": "客户频繁提到预算，建议发送 ROI 白皮书", "consent": true})),
		crmSeedObject(tenant.ID, "opportunities", "risk_alert", "交易停滞预警", "RA-001", "watching", 0, "CNY", 70, "risk", mustJSON(map[string]interface{}{"reason": "互动频率下降", "score": 70})),
		crmSeedObject(tenant.ID, "tickets", "ticket_category", "工单分类矩阵", "TCAT-001", "active", 0, "CNY", 0, "category", mustJSON(map[string]interface{}{"level1": "报修", "level2": []string{"软件", "硬件", "网络"}, "default_sla": "24h"})),
		crmSeedObject(tenant.ID, "tickets", "ticket_assignment", "技能组分配", "TASSIGN-001", "active", 0, "CNY", 0, "assignment", mustJSON(map[string]interface{}{"strategy": "skill_load", "groups": []string{"软件支持", "VIP支持"}})),
		crmSeedObject(tenant.ID, "tickets", "sla_rule", "VIP投诉SLA", "SLA-001", "active", 0, "CNY", 0, "sla", mustJSON(map[string]interface{}{"category": "complaint", "grade": "VIP", "response_hours": 1, "resolve_hours": 4})),
		crmSeedObject(tenant.ID, "tickets", "ticket_note", "工单公开回复", "TNOTE-001", "active", 0, "CNY", 0, "note", mustJSON(map[string]interface{}{"visibility": "public", "content": "已安排工程师处理"})),
		crmSeedObject(tenant.ID, "tickets", "satisfaction", "满意度评价", "SAT-001", "active", 0, "CNY", 0, "csat", mustJSON(map[string]interface{}{"csat": 5, "nps": 9, "comment": "响应及时"})),
		crmSeedObject(tenant.ID, "tickets", "ticket_report", "工单报表", "TREPORT-001", "active", 0, "CNY", 0, "report", mustJSON(map[string]interface{}{"trend": "up", "sla_rate": 0.96, "avg_resolution_hours": 6})),
		crmSeedObject(tenant.ID, "orders", "order_approval", "订单审核模板", "OAPP-001", "active", 0, "CNY", 0, "approval", mustJSON(map[string]interface{}{"thresholds": []map[string]interface{}{{"max": 100000, "approver": "经理"}, {"min": 100000, "approver": "总监"}}})),
		crmSeedObject(tenant.ID, "orders", "payment", "支付流水样例", "PAY-001", "paid", 420000, "CNY", 0, "payment", mustJSON(map[string]interface{}{"payment_no": "PAY-20260501", "method": "bank", "paid_at": "2026-05-12"})),
		crmSeedObject(tenant.ID, "orders", "shipment", "发货记录样例", "SHIP-001", "shipped", 0, "CNY", 0, "shipment", mustJSON(map[string]interface{}{"shipment_no": "SF-001", "carrier": "顺丰", "signed": false})),
		crmSeedObject(tenant.ID, "orders", "invoice", "增值税专用发票", "INV-SAMPLE", "active", 0, "CNY", 0, "invoice", mustJSON(map[string]interface{}{"invoice_no": "INV-20260501", "invoice_type": "special_vat"})),
		crmSeedObject(tenant.ID, "orders", "order_history", "客户订单历史", "OH-001", "active", 0, "CNY", 0, "history", mustJSON(map[string]interface{}{"filterable": true, "exportable": true})),
		crmSeedObject(tenant.ID, "orders", "return_request", "退换货申请", "RET-001", "active", 0, "CNY", 0, "after-sales", mustJSON(map[string]interface{}{"type": "return", "reason": "客户申请换货"})),
		crmSeedObject(tenant.ID, "workspace", "customer_activity", "客户打开报价邮件", "WA-001", "open", 0, "CNY", 0, "activity", mustJSON(map[string]interface{}{"source": "email", "related_domain": "opportunities"})),
		crmSeedObject(tenant.ID, "workspace", "mail_notification", "审批通知", "MAIL-001", "open", 0, "CNY", 0, "notification", mustJSON(map[string]interface{}{"category": "approval", "content": "有一笔订单待审批"})),
		crmSeedObject(tenant.ID, "workspace", "shortcut_group", "快捷入口", "SHORTCUT-001", "active", 0, "CNY", 0, "shortcut", mustJSON(map[string]interface{}{"items": []string{"新建联系人", "快速添加交易", "写邮件"}})),
		crmSeedObject(tenant.ID, "workspace", "widget_layout", "经理工作台布局", "WIDGET-001", "active", 0, "CNY", 0, "widget", mustJSON(map[string]interface{}{"widgets": []string{"funnel", "target", "hot_customer"}, "layout": "manager"})),
		crmSeedObject(tenant.ID, "workspace", "kpi_card", "本月成交额", "KPI-WON", "active", 420000, "CNY", 0, "kpi", mustJSON(map[string]interface{}{"metric": "won_amount", "drilldown_domain": "deals"})),
	}
	for _, sample := range samples {
		var existing int64
		if err := tx.Model(&models.CRMObject{}).Where("tenant_id = ? AND domain = ? AND object_type = ? AND code = ?", tenant.ID, sample.Domain, sample.ObjectType, sample.Code).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			continue
		}
		if err := tx.Create(&sample).Error; err != nil {
			return err
		}
	}
	return nil
}

func crmSeedObject(tenantID uint, domain, objectType, name, code, status string, amount float64, currency string, score int, tags string, payload string) models.CRMObject {
	return models.CRMObject{
		TenantModel: models.TenantModel{TenantID: tenantID},
		Domain:      domain,
		ObjectType:  objectType,
		Name:        name,
		Code:        code,
		Status:      status,
		Amount:      amount,
		Currency:    currency,
		Score:       score,
		Tags:        tags,
		PayloadJSON: payload,
	}
}

func crmSeedRelation(tenantID uint, sourceType string, sourceID uint, targetType string, targetID uint, relationType string) models.CRMRelation {
	return models.CRMRelation{
		TenantModel:  models.TenantModel{TenantID: tenantID},
		SourceType:   sourceType,
		SourceID:     sourceID,
		TargetType:   targetType,
		TargetID:     targetID,
		RelationType: relationType,
		PayloadJSON:  "{}",
	}
}

func mustJSON(value interface{}) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func dropLegacyDemoTables(tx *gorm.DB) error {
	legacyTables := []string{
		"agent_runs",
		"agent_tasks",
		"agent_definitions",
		"ai_call_logs",
		"analytics_reports",
		"api_keys",
		"audit_logs",
		"automation_runs",
		"automation_rules",
		"campaigns",
		"chat_messages",
		"chat_sessions",
		"contract_review_records",
		"customers",
		"interactions",
		"knowledge_bases",
		"lead_assignment_logs",
		"lead_assignment_rules",
		"leads",
		"model_configs",
		"multimodal_tasks",
		"nurture_records",
		"nurture_sequence_templates",
		"notification_preferences",
		"notifications",
		"opportunities",
		"prompt_templates",
		"quality_check_records",
		"sales_targets",
		"sentiment_records",
		"ticket_histories",
		"tickets",
		"webhook_deliveries",
		"webhook_subscriptions",
	}
	for _, table := range legacyTables {
		if err := tx.Exec("DROP TABLE IF EXISTS `" + table + "`").Error; err != nil {
			return fmt.Errorf("删除旧表 %s 失败: %w", table, err)
		}
	}
	return nil
}

func pruneLegacyPermissions(tx *gorm.DB) error {
	legacyResources := []string{
		"dashboard",
		"marketing",
		"sales",
		"customer_service",
		"data_insight",
		"system",
	}
	if err := tx.Exec("DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE resource IN ?)", legacyResources).Error; err != nil {
		return err
	}
	return tx.Exec("DELETE FROM permissions WHERE resource IN ?", legacyResources).Error
}

func backfillUserRoles(tx *gorm.DB) error {
	return tx.Exec(`
		INSERT INTO user_roles (created_at, updated_at, user_id, role_id)
		SELECT NOW(3), NOW(3), users.id, users.role_id
		FROM users
		WHERE users.role_id > 0
		  AND NOT EXISTS (
			SELECT 1 FROM user_roles
			WHERE user_roles.user_id = users.id
			  AND user_roles.role_id = users.role_id
			  AND user_roles.deleted_at IS NULL
		  )
	`).Error
}

func createOperationLogExportIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_operation_logs_tenant_operation_created ON operation_logs (tenant_id, operation, created_at)",
		"CREATE INDEX idx_operation_logs_tenant_result_created ON operation_logs (tenant_id, result, created_at)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func seedNewPRDPermissions(tx *gorm.DB) error {
	items := []struct {
		Resource string
		Action   string
		NameKey  string
	}{
		{"enterprise", "view", "permission.enterprise.view"},
		{"enterprise", "manage", "permission.enterprise.manage"},
		{"projects", "view", "permission.projects.view"},
		{"projects", "manage", "permission.projects.manage"},
		{"products", "view", "permission.products.view"},
		{"products", "manage", "permission.products.manage"},
		{"campaigns", "view", "permission.campaigns.view"},
		{"campaigns", "manage", "permission.campaigns.manage"},
		{"contacts", "view", "permission.contacts.view"},
		{"contacts", "manage", "permission.contacts.manage"},
		{"deals", "view", "permission.deals.view"},
		{"deals", "manage", "permission.deals.manage"},
		{"leads", "view", "permission.leads.view"},
		{"leads", "manage", "permission.leads.manage"},
		{"opportunities", "view", "permission.opportunities.view"},
		{"opportunities", "manage", "permission.opportunities.manage"},
		{"dashboards", "view", "permission.dashboards.view"},
		{"storage", "view", "permission.storage.view"},
		{"storage", "manage", "permission.storage.manage"},
		{"accounts", "manage", "permission.accounts.manage"},
		{"accounts", "view", "permission.accounts.view"},
		{"system:department", "view", "permission.system.department.view"},
		{"system:department", "create", "permission.system.department.create"},
		{"system:department", "update", "permission.system.department.update"},
		{"system:department", "delete", "permission.system.department.delete"},
		{"system:post", "view", "permission.system.post.view"},
		{"system:post", "create", "permission.system.post.create"},
		{"system:post", "update", "permission.system.post.update"},
		{"system:post", "delete", "permission.system.post.delete"},
		{"teams", "view", "permission.teams.view"},
		{"teams", "manage", "permission.teams.manage"},
		{"insights", "view", "permission.insights.view"},
		{"insights", "manage", "permission.insights.manage"},
		{"tickets", "view", "permission.tickets.view"},
		{"tickets", "manage", "permission.tickets.manage"},
		{"orders", "view", "permission.orders.view"},
		{"orders", "manage", "permission.orders.manage"},
		{"workspace", "view", "permission.workspace.view"},
		{"workspace", "manage", "permission.workspace.manage"},
		{"iot", "view", "permission.iot.view"},
		{"iot", "manage", "permission.iot.manage"},
		{"audit_log", "view", "permission.audit_log.view"},
		{"audit_log", "export", "permission.audit_log.export"},
		{"mfg", "view", "permission.mfg.view"},
		{"mfg:rfq", "view", "permission.mfg.rfq.view"},
		{"mfg:rfq", "manage", "permission.mfg.rfq.manage"},
		{"mfg:product", "view", "permission.mfg.product.view"},
		{"mfg:product", "manage", "permission.mfg.product.manage"},
		{"mfg:cost", "view", "permission.mfg.cost.view"},
		{"mfg:quote", "view", "permission.mfg.quote.view"},
		{"mfg:quote", "approve", "permission.mfg.quote.approve"},
		{"mfg:quote", "manage", "permission.mfg.quote.manage"},
		{"mfg:cost", "view_sensitive", "permission.mfg.cost.view_sensitive"},
		{"mfg:cost", "manage", "permission.mfg.cost.manage"},
		{"mfg:sample", "view", "permission.mfg.sample.view"},
		{"mfg:sample", "manage", "permission.mfg.sample.manage"},
		{"mfg:production", "view", "permission.mfg.production.view"},
		{"mfg:production", "manage", "permission.mfg.production.manage"},
		{"mfg:ecn", "view", "permission.mfg.ecn.view"},
		{"mfg:ecn", "manage", "permission.mfg.ecn.manage"},
		{"mfg:metrics", "view", "permission.mfg.metrics.view"},
		{"mfg:metrics", "manage", "permission.mfg.metrics.manage"},
	}
	for _, item := range items {
		if err := tx.Exec(
			"INSERT INTO permissions (created_at, updated_at, resource, action, name_key) VALUES (NOW(3), NOW(3), ?, ?, ?) ON DUPLICATE KEY UPDATE name_key = VALUES(name_key), updated_at = NOW(3)",
			item.Resource, item.Action, item.NameKey,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedMfgQuoteApprovalPermissions(tx *gorm.DB) error {
	items := []models.Permission{
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
		{Resource: "mfg:ecn", Action: "manage", NameKey: "permission.mfg.ecn.manage"},
		{Resource: "mfg:metrics", Action: "view", NameKey: "permission.mfg.metrics.view"},
		{Resource: "mfg:metrics", Action: "manage", NameKey: "permission.mfg.metrics.manage"},
	}
	for _, item := range items {
		if err := tx.Exec(
			"INSERT INTO permissions (created_at, updated_at, resource, action, name_key) VALUES (NOW(3), NOW(3), ?, ?, ?) ON DUPLICATE KEY UPDATE name_key = VALUES(name_key), updated_at = NOW(3)",
			item.Resource, item.Action, item.NameKey,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillMfgQuoteApprovalRolePermissions(tx *gorm.DB) error {
	adminPermissions := []string{
		"mfg:view", "mfg:rfq:view", "mfg:rfq:manage", "mfg:product:view", "mfg:product:manage",
		"mfg:cost:view", "mfg:cost:view_sensitive", "mfg:cost:manage", "mfg:quote:view", "mfg:quote:approve", "mfg:quote:manage",
		"mfg:sample:view", "mfg:sample:manage", "mfg:production:view", "mfg:production:manage", "mfg:ecn:view", "mfg:ecn:manage", "mfg:metrics:view", "mfg:metrics:manage",
	}
	supervisorPermissions := []string{
		"mfg:view", "mfg:rfq:view", "mfg:cost:view", "mfg:cost:view_sensitive", "mfg:quote:view", "mfg:quote:approve",
	}
	var roles []models.Role
	if err := tx.Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		name := strings.TrimSpace(role.Name)
		code := strings.ToUpper(strings.TrimSpace(role.Code))
		additions := []string{}
		if name == "管理员" || code == "ADMIN" || code == "TENANT_ADMIN" {
			additions = adminPermissions
		}
		if name == "主管" || name == "SDR主管" || code == "R_SUPERVISOR" || code == "R_SDR_SUPERVISOR" {
			additions = supervisorPermissions
		}
		if len(additions) == 0 {
			continue
		}
		merged := uniqueStrings(append(splitPermissionCSV(role.PermissionsJSON), additions...))
		if err := tx.Model(&models.Role{}).Where("id = ?", role.ID).Update("permissions_json", strings.Join(merged, ",")).Error; err != nil {
			return err
		}
	}
	return nil
}

func stripOrganizationPermissionsFromRoles(tx *gorm.DB) error {
	var roles []models.Role
	if err := tx.Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		source := splitPermissionCSV(role.PermissionsJSON)
		next := make([]string, 0, len(source))
		for _, permission := range source {
			resource, _ := splitPermissionParts(permission)
			if isOrganizationPermissionResource(resource) {
				continue
			}
			next = append(next, permission)
		}
		next = uniqueStrings(next)
		if strings.Join(source, ",") != strings.Join(next, ",") {
			if err := tx.Model(&models.Role{}).Where("id = ?", role.ID).Update("permissions_json", strings.Join(next, ",")).Error; err != nil {
				return err
			}
		}
	}
	var permissions []models.Permission
	if err := tx.Where("resource IN ?", organizationPermissionResources()).Find(&permissions).Error; err != nil {
		return err
	}
	if len(permissions) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(permissions))
	for _, permission := range permissions {
		ids = append(ids, permission.ID)
	}
	return tx.Unscoped().Where("permission_id IN ?", ids).Delete(&models.RolePermission{}).Error
}

func organizationPermissionResources() []string {
	return []string{"enterprise", "accounts", "system:department", "system:post", "system:menu", "storage", "iot", "audit_log"}
}

func isOrganizationPermissionResource(resource string) bool {
	resource = strings.TrimSpace(resource)
	for _, item := range organizationPermissionResources() {
		if resource == item {
			return true
		}
	}
	return false
}

func seedFineGrainedPermissions(tx *gorm.DB) error {
	resources := []string{
		"workspace", "enterprise", "projects", "products", "campaigns", "contacts",
		"leads", "opportunities", "deals", "orders", "tickets", "dashboards", "teams",
		"insights", "ai_usage", "accounts", "im", "audit_log", "system:department", "system:post",
	}
	defaultActions := []string{"view", "create", "update", "delete", "export"}
	extraActions := map[string][]string{
		"leads":             {"assign"},
		"tickets":           {"assign"},
		"workspace":         {"approve"},
		"audit_log":         {"export"},
		"system:department": {"create", "update", "delete"},
		"system:post":       {"create", "update", "delete"},
		"im":                {"create", "update", "delete"},
	}
	for _, resource := range resources {
		actions := append([]string{}, defaultActions...)
		switch resource {
		case "system:department", "system:post":
			actions = []string{"view"}
		case "im":
			actions = []string{"view"}
		case "audit_log":
			actions = []string{"view"}
		}
		actions = uniqueStrings(append(actions, extraActions[resource]...))
		for _, action := range actions {
			nameKey := fmt.Sprintf("permission.%s.%s", strings.ReplaceAll(resource, ":", "."), action)
			if err := tx.Exec(
				"INSERT INTO permissions (created_at, updated_at, resource, action, name_key) VALUES (NOW(3), NOW(3), ?, ?, ?) ON DUPLICATE KEY UPDATE name_key = VALUES(name_key), updated_at = NOW(3)",
				resource, action, nameKey,
			).Error; err != nil {
				return err
			}
		}
		// Compatibility permission remains.
		nameKey := fmt.Sprintf("permission.%s.manage", strings.ReplaceAll(resource, ":", "."))
		if err := tx.Exec(
			"INSERT INTO permissions (created_at, updated_at, resource, action, name_key) VALUES (NOW(3), NOW(3), ?, ?, ?) ON DUPLICATE KEY UPDATE name_key = VALUES(name_key), updated_at = NOW(3)",
			resource, "manage", nameKey,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillFineGrainedRolePermissions(tx *gorm.DB) error {
	var roles []models.Role
	if err := tx.Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		source := splitPermissionCSV(role.PermissionsJSON)
		next := append([]string{}, source...)
		for _, permission := range source {
			resource, action := splitPermissionParts(permission)
			if resource == "" {
				continue
			}
			if action == "manage" {
				next = append(next,
					resource+":create",
					resource+":update",
					resource+":delete",
					resource+":export",
				)
				if resource == "leads" || resource == "tickets" {
					next = append(next, resource+":assign")
				}
				if resource == "workspace" {
					next = append(next, resource+":approve")
				}
			}
		}
		next = uniqueStrings(next)
		if strings.Join(next, ",") == strings.Join(source, ",") {
			continue
		}
		if err := tx.Model(&models.Role{}).Where("id = ?", role.ID).Update("permissions_json", strings.Join(next, ",")).Error; err != nil {
			return err
		}
	}
	return nil
}

func splitPermissionParts(permission string) (resource string, action string) {
	value := strings.TrimSpace(permission)
	idx := strings.LastIndex(value, ":")
	if idx <= 0 || idx >= len(value)-1 {
		return "", ""
	}
	return strings.TrimSpace(value[:idx]), strings.TrimSpace(value[idx+1:])
}

func seedSystemPermissions(tx *gorm.DB) error {
	if err := seedNewPRDPermissions(tx); err != nil {
		return err
	}
	items := []struct {
		Resource string
		Action   string
		NameKey  string
	}{
		{"system:department", "view", "permission.system.department.view"},
		{"system:department", "manage", "permission.system.department.manage"},
		{"system:post", "view", "permission.system.post.view"},
		{"system:post", "manage", "permission.system.post.manage"},
		{"system:menu", "view", "permission.system.menu.view"},
		{"system:menu", "manage", "permission.system.menu.manage"},
	}
	for _, item := range items {
		if err := tx.Exec(
			"INSERT INTO permissions (created_at, updated_at, resource, action, name_key) VALUES (NOW(3), NOW(3), ?, ?, ?) ON DUPLICATE KEY UPDATE name_key = VALUES(name_key), updated_at = NOW(3)",
			item.Resource, item.Action, item.NameKey,
		).Error; err != nil {
			return err
		}
	}
	systemPermissions := []string{
		"system:department:view", "system:department:manage",
		"system:post:view", "system:post:manage",
		"system:menu:view", "system:menu:manage",
		"accounts:manage", "audit_log:view",
	}
	var roles []models.Role
	if err := tx.Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		if !role.IsPreset && !role.IsSystem {
			continue
		}
		merged := append(splitPermissionCSV(role.PermissionsJSON), systemPermissions...)
		if err := tx.Model(&models.Role{}).Where("id = ?", role.ID).Updates(map[string]interface{}{
			"permissions_json": strings.Join(uniqueStrings(merged), ","),
			"status":           "active",
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedSystemManagementData(tx *gorm.DB) error {
	var tenants []models.Tenant
	if err := tx.Find(&tenants).Error; err != nil {
		return err
	}
	for _, tenant := range tenants {
		if err := seedSystemTenantData(tx, tenant.ID); err != nil {
			return err
		}
	}
	return nil
}

func seedSystemTenantData(tx *gorm.DB, tenantID uint) error {
	departments := []models.SystemDepartment{
		{TenantModel: models.TenantModel{TenantID: tenantID}, Name: "研发部", Code: "RD", Sort: 1, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, Name: "产品部", Code: "PD", Sort: 2, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, Name: "运营中心", Code: "OP", Sort: 3, Status: "active"},
	}
	for _, item := range departments {
		if err := tx.Where("tenant_id = ? AND code = ?", tenantID, item.Code).FirstOrCreate(&item).Error; err != nil {
			return err
		}
	}
	var rd models.SystemDepartment
	if err := tx.Where("tenant_id = ? AND code = ?", tenantID, "RD").First(&rd).Error; err == nil {
		children := []models.SystemDepartment{
			{TenantModel: models.TenantModel{TenantID: tenantID}, ParentID: rd.ID, Name: "后端开发组", Code: "RD-BE", Sort: 1, Status: "active"},
			{TenantModel: models.TenantModel{TenantID: tenantID}, ParentID: rd.ID, Name: "前端开发组", Code: "RD-FE", Sort: 2, Status: "active"},
			{TenantModel: models.TenantModel{TenantID: tenantID}, ParentID: rd.ID, Name: "测试组", Code: "RD-QA", Sort: 3, Status: "active"},
		}
		for _, item := range children {
			if err := tx.Where("tenant_id = ? AND code = ?", tenantID, item.Code).Assign(map[string]interface{}{"parent_id": item.ParentID}).FirstOrCreate(&item).Error; err != nil {
				return err
			}
		}
	}
	posts := []models.SystemPost{
		{TenantModel: models.TenantModel{TenantID: tenantID}, Name: "运维工程师", Code: "OPS-ENG", Sort: 1, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, Name: "测试工程师", Code: "QA-ENG", Sort: 1, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, Name: "前端开发工程师", Code: "DEV-FE", Sort: 1, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, Name: "后端开发工程师", Code: "DEV-BE", Sort: 1, Status: "active"},
	}
	for _, item := range posts {
		if err := tx.Where("tenant_id = ? AND code = ?", tenantID, item.Code).FirstOrCreate(&item).Error; err != nil {
			return err
		}
	}
	root := models.SystemMenu{TenantModel: models.TenantModel{TenantID: tenantID}, Type: "catalog", Name: "系统管理", Path: "/system", Permission: "system:menu:view", Icon: "ri:settings-3-line", Sort: 90, Status: "active"}
	if err := tx.Where("tenant_id = ? AND path = ?", tenantID, root.Path).FirstOrCreate(&root).Error; err != nil {
		return err
	}
	menus := []models.SystemMenu{
		{TenantModel: models.TenantModel{TenantID: tenantID}, ParentID: root.ID, Type: "menu", Name: "用户管理", Path: "/system/user", Permission: "accounts:manage", Icon: "ri:user-line", Sort: 1, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, ParentID: root.ID, Type: "menu", Name: "角色管理", Path: "/system/role", Permission: "accounts:manage", Icon: "ri:user-settings-line", Sort: 2, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, ParentID: root.ID, Type: "menu", Name: "部门管理", Path: "/system/department", Permission: "system:department:view", Icon: "ri:building-2-line", Sort: 3, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, ParentID: root.ID, Type: "menu", Name: "岗位管理", Path: "/system/post", Permission: "system:post:view", Icon: "ri:briefcase-line", Sort: 4, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, ParentID: root.ID, Type: "menu", Name: "菜单管理", Path: "/system/menu", Permission: "system:menu:view", Icon: "ri:menu-line", Sort: 5, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, ParentID: root.ID, Type: "menu", Name: "操作日志", Path: "/system/operation-log", Permission: "audit_log:view", Icon: "ri:file-list-3-line", Sort: 6, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, ParentID: root.ID, Type: "menu", Name: "登录日志", Path: "/system/login-log", Permission: "audit_log:view", Icon: "ri:shield-user-line", Sort: 7, Status: "active"},
		{TenantModel: models.TenantModel{TenantID: tenantID}, ParentID: root.ID, Type: "menu", Name: "个人中心", Path: "/system/user-center", Permission: "", Icon: "ri:account-circle-line", Sort: 8, Status: "active"},
	}
	for _, item := range menus {
		if err := tx.Where("tenant_id = ? AND path = ?", tenantID, item.Path).Assign(map[string]interface{}{"parent_id": root.ID}).FirstOrCreate(&item).Error; err != nil {
			return err
		}
	}
	return tx.Exec("UPDATE roles SET code = CONCAT('R_', UPPER(REPLACE(name, ' ', '_'))) WHERE (code IS NULL OR code = '')").Error
}

func createSystemManagementIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_system_departments_tenant_parent_sort ON system_departments (tenant_id, parent_id, sort)",
		"CREATE INDEX idx_system_posts_tenant_sort ON system_posts (tenant_id, sort)",
		"CREATE INDEX idx_system_menus_tenant_parent_sort ON system_menus (tenant_id, parent_id, sort)",
		"CREATE INDEX idx_user_sessions_user_active ON user_sessions (user_id, revoked, expires_at)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func splitPermissionCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

func createNewPRDIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_crm_objects_tenant_domain_created ON crm_objects (tenant_id, domain, created_at)",
		"CREATE INDEX idx_crm_objects_tenant_status ON crm_objects (tenant_id, status)",
		"CREATE INDEX idx_crm_objects_owner ON crm_objects (tenant_id, owner_id)",
		"CREATE INDEX idx_crm_relations_source ON crm_relations (tenant_id, source_type, source_id)",
		"CREATE INDEX idx_crm_relations_target ON crm_relations (tenant_id, target_type, target_id)",
		"CREATE UNIQUE INDEX idx_crm_settings_tenant_scope_key ON crm_settings (tenant_id, scope, `key`)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func seedLeadOwnerDataScope(tx *gorm.DB) error {
	var tenants []models.Tenant
	if err := tx.Find(&tenants).Error; err != nil {
		return err
	}
	for _, tenant := range tenants {
		setting := models.CRMSetting{
			TenantModel: models.TenantModel{TenantID: tenant.ID},
			Scope:       "data_scope",
			Key:         "leads",
			ValueJSON:   mustJSON(map[string]interface{}{"mode": "owner", "piercing_user_ids": []uint{}}),
		}
		if err := tx.Where("tenant_id = ? AND scope = ? AND `key` = ?", tenant.ID, "data_scope", "leads").
			Assign(setting).
			FirstOrCreate(&setting).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeExistingUsers(tx *gorm.DB) error {
	return tx.Exec("UPDATE users SET account_type = 'super_admin', role = 'super_admin', tenant_id = 0 WHERE (account_type IS NULL OR account_type = '' OR account_type = 'sub_user') AND tenant_id = 0").Error
}

func seedDefaultPermissions(tx *gorm.DB) error {
	return seedNewPRDPermissions(tx)
}

func createSaaSIndexes(tx *gorm.DB) error {
	dropStatements := []string{
		"DROP INDEX idx_users_email ON users",
		"DROP INDEX idx_roles_name ON roles",
	}
	for _, statement := range dropStatements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("旧索引删除跳过或失败: %s err=%v", statement, err)
		}
	}
	statements := []string{
		"CREATE INDEX idx_operation_logs_tenant_created ON operation_logs (tenant_id, created_at)",
		"CREATE INDEX idx_operation_logs_operation_created ON operation_logs (operation, created_at)",
		"CREATE INDEX idx_users_tenant_status ON users (tenant_id, status)",
		"CREATE INDEX idx_roles_tenant ON roles (tenant_id)",
		"CREATE UNIQUE INDEX idx_roles_tenant_name ON roles (tenant_id, name)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func createPlatformIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_platform_settings_scope_status_group_key ON platform_setting_versions (scope, status, `group`, `key`)",
		"CREATE INDEX idx_platform_settings_group_key_version ON platform_setting_versions (`group`, `key`, version)",
		"CREATE INDEX idx_users_account_type_status_tenant ON users (account_type, status, tenant_id)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func createTenantAdminActivationIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_tenant_admin_activation_user_status_exp ON tenant_admin_activation_tokens (user_id, status, expires_at)",
		"CREATE INDEX idx_tenant_admin_activation_tenant_status ON tenant_admin_activation_tokens (tenant_id, status)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func seedStoragePermissions(tx *gorm.DB) error {
	items := []struct {
		Resource string
		Action   string
		NameKey  string
	}{
		{"storage", "view", "permission.storage.view"},
		{"storage", "manage", "permission.storage.manage"},
	}
	for _, item := range items {
		if err := tx.Exec(
			"INSERT INTO permissions (created_at, updated_at, resource, action, name_key) VALUES (NOW(3), NOW(3), ?, ?, ?) ON DUPLICATE KEY UPDATE name_key = VALUES(name_key), updated_at = NOW(3)",
			item.Resource, item.Action, item.NameKey,
		).Error; err != nil {
			return err
		}
	}
	var roles []models.Role
	if err := tx.Where("is_preset = ? OR is_system = ?", true, true).Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		merged := uniqueStrings(append(splitPermissionCSV(role.PermissionsJSON), "storage:view", "storage:manage"))
		if err := tx.Model(&models.Role{}).Where("id = ?", role.ID).
			Update("permissions_json", strings.Join(merged, ",")).
			Error; err != nil {
			return err
		}
	}
	return nil
}

func seedIoTPermissions(tx *gorm.DB) error {
	items := []struct {
		Resource string
		Action   string
		NameKey  string
	}{
		{"iot", "view", "permission.iot.view"},
		{"iot", "manage", "permission.iot.manage"},
	}
	for _, item := range items {
		if err := tx.Exec(
			"INSERT INTO permissions (created_at, updated_at, resource, action, name_key) VALUES (NOW(3), NOW(3), ?, ?, ?) ON DUPLICATE KEY UPDATE name_key = VALUES(name_key), updated_at = NOW(3)",
			item.Resource, item.Action, item.NameKey,
		).Error; err != nil {
			return err
		}
	}
	var roles []models.Role
	if err := tx.Where("is_preset = ? OR is_system = ?", true, true).Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		merged := uniqueStrings(append(splitPermissionCSV(role.PermissionsJSON), "iot:view", "iot:manage"))
		if err := tx.Model(&models.Role{}).Where("id = ?", role.ID).
			Update("permissions_json", strings.Join(merged, ",")).
			Error; err != nil {
			return err
		}
	}
	if err := appendIoTToTenantQuotaModules(tx); err != nil {
		return err
	}
	if err := appendIoTToPlanModules(tx); err != nil {
		return err
	}
	return nil
}

func createTenantStorageExpansionIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_tenant_storage_exp_tenant_status_valid ON tenant_storage_expansions (tenant_id, status, valid_to)",
		"CREATE INDEX idx_tenant_storage_exp_valid_to ON tenant_storage_expansions (valid_to)",
		"CREATE INDEX idx_tenant_storage_exp_tenant_source_valid ON tenant_storage_expansions (tenant_id, source, valid_to)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func createIoTDeviceIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_iot_devices_tenant_status ON iot_devices (tenant_id, status)",
		"CREATE INDEX idx_iot_devices_tenant_seen ON iot_devices (tenant_id, last_seen_at)",
		"CREATE INDEX idx_iot_devices_public_key_fingerprint ON iot_devices (public_key_fingerprint)",
		"CREATE INDEX idx_iot_devices_bound_at ON iot_devices (bound_at)",
		"CREATE INDEX idx_iot_devices_last_bootstrap_at ON iot_devices (last_bootstrap_at)",
		"CREATE INDEX idx_iot_devices_ota_status ON iot_devices (ota_status)",
		"CREATE INDEX idx_iot_devices_last_ota_check ON iot_devices (last_ota_check_at)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func createIoTFirmwareIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_iot_fw_status_model ON iot_firmware_releases (status, model, created_at)",
		"CREATE INDEX idx_iot_fw_sha256 ON iot_firmware_releases (sha256)",
		"CREATE INDEX idx_iot_fw_target_release_status ON iot_firmware_targets (release_id, status)",
		"CREATE INDEX idx_iot_fw_target_device_status ON iot_firmware_targets (device_id, status, assigned_at)",
		"CREATE INDEX idx_iot_fw_target_sn_version ON iot_firmware_targets (sn, version)",
		"CREATE INDEX idx_iot_fw_target_tenant_assigned ON iot_firmware_targets (tenant_id, assigned_at)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func backfillIoTFirmwareTargets(tx *gorm.DB) error {
	var devices []models.IoTDevice
	if err := tx.Where("target_firmware_version <> ''").Find(&devices).Error; err != nil {
		return err
	}
	for _, device := range devices {
		version := strings.TrimSpace(device.TargetFirmwareVersion)
		if version == "" {
			continue
		}
		var release models.IoTFirmwareRelease
		if err := tx.Where("model = ? AND version = ?", device.Model, version).Order("id desc").First(&release).Error; err != nil {
			continue
		}
		assignedAt := time.Now()
		if device.LastOTACheckAt != nil {
			assignedAt = *device.LastOTACheckAt
		}
		status := strings.TrimSpace(device.OTAStatus)
		if status == "" {
			status = "assigned"
		}
		target := models.IoTFirmwareTarget{
			ReleaseID:     release.ID,
			DeviceID:      device.ID,
			TenantID:      device.TenantID,
			SN:            device.SN,
			Model:         device.Model,
			Version:       release.Version,
			Status:        status,
			Force:         release.Force,
			FailureReason: device.OTAFailureReason,
			AssignedAt:    assignedAt,
			CheckedAt:     device.LastOTACheckAt,
			DownloadedAt:  device.LastOTADownloadAt,
			ConfirmedAt:   device.LastOTAConfirmAt,
			LastSeenAt:    device.LastSeenAt,
		}
		if err := tx.Where("release_id = ? AND device_id = ?", release.ID, device.ID).Assign(target).FirstOrCreate(&target).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureIoTDevicesTable(tx *gorm.DB) error {
	if tx.Migrator().HasTable("iot_devices") {
		return autoMigrateWithDB(tx)
	}
	if tx.Migrator().HasTable("io_t_devices") {
		if err := tx.Migrator().RenameTable("io_t_devices", "iot_devices"); err != nil {
			return err
		}
	}
	return autoMigrateWithDB(tx)
}

func appendIoTToTenantQuotaModules(tx *gorm.DB) error {
	var quotas []models.TenantQuota
	if err := tx.Find(&quotas).Error; err != nil {
		return err
	}
	for _, quota := range quotas {
		modules := splitPermissionCSV(quota.Modules)
		if len(modules) == 0 {
			continue
		}
		merged := uniqueStrings(append(modules, "iot"))
		if strings.Join(merged, ",") == quota.Modules {
			continue
		}
		if err := tx.Model(&models.TenantQuota{}).Where("id = ?", quota.ID).Update("modules", strings.Join(merged, ",")).Error; err != nil {
			return err
		}
	}
	return nil
}

func appendIoTToPlanModules(tx *gorm.DB) error {
	var plans []models.PlatformPlanTemplate
	if err := tx.Find(&plans).Error; err != nil {
		return err
	}
	for _, plan := range plans {
		var modules []string
		if strings.TrimSpace(plan.ModulesJSON) != "" {
			if err := json.Unmarshal([]byte(plan.ModulesJSON), &modules); err != nil {
				continue
			}
		}
		merged := uniqueStrings(append(modules, "iot"))
		data, err := json.Marshal(merged)
		if err != nil {
			return err
		}
		if string(data) == plan.ModulesJSON {
			continue
		}
		if err := tx.Model(&models.PlatformPlanTemplate{}).Where("id = ?", plan.ID).Update("modules_json", string(data)).Error; err != nil {
			return err
		}
	}
	return nil
}

func createTenantSeatExpansionIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_tenant_seat_exp_tenant_status_valid ON tenant_seat_expansions (tenant_id, status, valid_to)",
		"CREATE INDEX idx_tenant_seat_exp_valid_to ON tenant_seat_expansions (valid_to)",
		"CREATE INDEX idx_tenant_seat_exp_tenant_source_valid ON tenant_seat_expansions (tenant_id, source, valid_to)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func backfillTenantSeatExpansions(tx *gorm.DB) error {
	var quotas []models.TenantQuota
	if err := tx.Where("max_users > 0").Find(&quotas).Error; err != nil {
		return err
	}
	now := time.Now()
	for _, quota := range quotas {
		var count int64
		if err := tx.Model(&models.TenantSeatExpansion{}).
			Where("tenant_id = ? AND source = ?", quota.TenantID, "legacy").
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		validFrom := quota.CreatedAt
		if quota.ValidFrom != nil {
			validFrom = *quota.ValidFrom
		}
		if validFrom.IsZero() {
			validFrom = now
		}
		validTo := validFrom
		status := "expired"
		if quota.ValidTo != nil {
			validTo = *quota.ValidTo
			if validTo.After(now) {
				status = "active"
			}
		}
		if !validTo.After(validFrom) {
			validTo = validFrom.AddDate(1, 0, 0)
			status = "expired"
		}
		record := models.TenantSeatExpansion{
			TenantModel:  models.TenantModel{TenantID: quota.TenantID},
			SeatCount:    quota.MaxUsers,
			Years:        1,
			PricePerSeat: 10,
			Currency:     "USD",
			Source:       "legacy",
			Status:       status,
			ValidFrom:    validFrom,
			ValidTo:      validTo,
			Remark:       "Legacy seat quota migration",
			RequestJSON:  `{"migration":"tenant_entitlement_expansion_billing"}`,
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
	}
	return nil
}

func convertTenantEntitlementExpansionTablesToUTF8MB4(tx *gorm.DB) error {
	statements := []string{
		"ALTER TABLE tenant_storage_expansions CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		"ALTER TABLE tenant_seat_expansions CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func createPerformanceIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_crm_objects_tenant_domain_obj_status_updated ON crm_objects (tenant_id, domain, object_type, status, updated_at)",
		"CREATE INDEX idx_crm_objects_tenant_domain_owner_updated ON crm_objects (tenant_id, domain, owner_id, updated_at)",
		"CREATE INDEX idx_crm_objects_tenant_domain_company_updated ON crm_objects (tenant_id, domain, company_id, updated_at)",
		"CREATE INDEX idx_crm_objects_tenant_domain_contact_updated ON crm_objects (tenant_id, domain, contact_id, updated_at)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func createOperationLogUserIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_operation_logs_tenant_user_created ON operation_logs (tenant_id, user_id, created_at)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func seedTenantAIWallets(tx *gorm.DB) error {
	var tenants []models.Tenant
	if err := tx.Where("status = ?", "active").Find(&tenants).Error; err != nil {
		return err
	}
	for _, tenant := range tenants {
		wallet := models.TenantAIWallet{
			TenantModel:         models.TenantModel{TenantID: tenant.ID},
			Currency:            "USD",
			BalanceMicros:       100000000,
			MonthlyTokenLimit:   1000000,
			MonthlyTokenUsed:    0,
			HardStop:            true,
			Enabled:             true,
			LastUsageResetMonth: time.Now().Format("2006-01"),
		}
		if err := tx.Where("tenant_id = ?", tenant.ID).FirstOrCreate(&wallet).Error; err != nil {
			return err
		}
		var quota models.TenantQuota
		if err := tx.Where("tenant_id = ?", tenant.ID).First(&quota).Error; err == nil {
			features := map[string]interface{}{}
			_ = json.Unmarshal([]byte(quota.FeatureSwitch), &features)
			if _, exists := features["ai"]; !exists {
				features["ai"] = map[string]interface{}{"enabled": true}
				quota.FeatureSwitch = mustJSON(features)
				if err := tx.Save(&quota).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func createAIResearchIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE UNIQUE INDEX idx_tenant_ai_wallets_tenant ON tenant_ai_wallets (tenant_id)",
		"CREATE INDEX idx_ai_usage_logs_tenant_feature_created ON ai_usage_logs (tenant_id, feature, created_at)",
		"CREATE INDEX idx_ai_usage_logs_tenant_object ON ai_usage_logs (tenant_id, object_domain, object_type, object_id)",
		"CREATE INDEX idx_ai_research_jobs_tenant_object_created ON ai_research_jobs (tenant_id, object_domain, object_type, object_id, created_at)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func seedAIAnalysisPackages(tx *gorm.DB) error {
	packages := []models.AIAnalysisPackage{
		{Code: "trial", Name: "AI 分析体验包", Currency: "USD", PriceMicros: 9900000, BalanceMicros: 9900000, MonthlyTokenLimit: 1000000, ValidityDays: 30, Description: "适合体验 AI 经营分析，包含基础 token 额度。", Status: "active", Sort: 10},
		{Code: "standard", Name: "AI 分析标准包", Currency: "USD", PriceMicros: 49900000, BalanceMicros: 49900000, MonthlyTokenLimit: 8000000, ValidityDays: 90, Description: "适合中小团队按月生成经营分析报告。", Status: "active", Sort: 20},
		{Code: "enterprise", Name: "AI 分析企业包", Currency: "USD", PriceMicros: 199900000, BalanceMicros: 199900000, MonthlyTokenLimit: 50000000, ValidityDays: 365, Description: "适合管理层高频查看经营报告和历史账务。", Status: "active", Sort: 30},
	}
	for _, item := range packages {
		if err := tx.Exec(`
INSERT INTO ai_analysis_packages
  (created_at, updated_at, code, name, currency, price_micros, balance_micros, monthly_token_limit, validity_days, description, status, sort)
VALUES
  (NOW(3), NOW(3), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  currency = VALUES(currency),
  price_micros = VALUES(price_micros),
  balance_micros = VALUES(balance_micros),
  monthly_token_limit = VALUES(monthly_token_limit),
  validity_days = VALUES(validity_days),
  description = VALUES(description),
  status = VALUES(status),
  sort = VALUES(sort),
  updated_at = NOW(3)`,
			item.Code, item.Name, item.Currency, item.PriceMicros, item.BalanceMicros, item.MonthlyTokenLimit, item.ValidityDays, item.Description, item.Status, item.Sort,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func createAIAnalysisIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_ai_token_ledger_tenant_created ON ai_token_ledger_entries (tenant_id, created_at)",
		"CREATE INDEX idx_ai_token_ledger_tenant_job ON ai_token_ledger_entries (tenant_id, job_id, entry_type)",
		"CREATE INDEX idx_ai_token_ledger_trace ON ai_token_ledger_entries (trace_id)",
		"CREATE INDEX idx_tenant_ai_applications_tenant_status ON tenant_ai_applications (tenant_id, status, created_at)",
		"CREATE INDEX idx_ai_analysis_packages_status_sort ON ai_analysis_packages (status, sort)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func normalizePlatformSuperAdmins(tx *gorm.DB) error {
	return tx.Exec(`
UPDATE users
SET account_type = 'super_admin',
    role = CASE
      WHEN role IS NULL OR role = '' OR role = 'admin' THEN 'super_admin'
      ELSE role
    END
WHERE tenant_id = 0
  AND status = 'active'
  AND (
    account_type IS NULL OR account_type = '' OR account_type = 'sub_user' OR account_type = 'admin' OR account_type = 'platform_super_admin'
  )
`).Error
}

func seedIMPermissions(tx *gorm.DB) error {
	items := []struct {
		Resource string
		Action   string
		NameKey  string
	}{
		{"im", "view", "permission.im.view"},
		{"im", "manage", "permission.im.manage"},
	}
	for _, item := range items {
		if err := tx.Exec(
			"INSERT INTO permissions (created_at, updated_at, resource, action, name_key) VALUES (NOW(3), NOW(3), ?, ?, ?) ON DUPLICATE KEY UPDATE name_key = VALUES(name_key), updated_at = NOW(3)",
			item.Resource, item.Action, item.NameKey,
		).Error; err != nil {
			return err
		}
	}

	var roles []models.Role
	if err := tx.Where("is_preset = ? OR is_system = ?", true, true).Find(&roles).Error; err != nil {
		return err
	}
	for _, role := range roles {
		merged := uniqueStrings(append(splitPermissionCSV(role.PermissionsJSON), "im:view", "im:manage"))
		if err := tx.Model(&models.Role{}).Where("id = ?", role.ID).
			Update("permissions_json", strings.Join(merged, ",")).
			Error; err != nil {
			return err
		}
	}
	return nil
}

func createIMIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_im_messages_tenant_conv_id_desc ON im_messages (tenant_id, conversation_id, id DESC)",
		"CREATE INDEX idx_im_messages_tenant_recipient_created ON im_messages (tenant_id, recipient_id, created_at)",
		"CREATE INDEX idx_im_receipts_tenant_recipient_read ON im_message_receipts (tenant_id, recipient_id, read_at)",
		"CREATE INDEX idx_im_presence_tenant_user ON im_user_presences (tenant_id, user_id)",
		"CREATE UNIQUE INDEX idx_im_conversations_tenant_users ON im_conversations (tenant_id, user_low, user_high)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func ensureIMSupportConversationColumns(tx *gorm.DB) error {
	if !tx.Migrator().HasColumn(&models.IMConversation{}, "ConversationType") {
		if err := tx.Migrator().AddColumn(&models.IMConversation{}, "ConversationType"); err != nil {
			return err
		}
	}
	if !tx.Migrator().HasColumn(&models.IMConversation{}, "Category") {
		if err := tx.Migrator().AddColumn(&models.IMConversation{}, "Category"); err != nil {
			return err
		}
	}
	if !tx.Migrator().HasColumn(&models.IMConversation{}, "Priority") {
		if err := tx.Migrator().AddColumn(&models.IMConversation{}, "Priority"); err != nil {
			return err
		}
	}
	if !tx.Migrator().HasColumn(&models.IMConversation{}, "Title") {
		if err := tx.Migrator().AddColumn(&models.IMConversation{}, "Title"); err != nil {
			return err
		}
	}
	if !tx.Migrator().HasColumn(&models.IMConversation{}, "PayloadJSON") {
		if err := tx.Migrator().AddColumn(&models.IMConversation{}, "PayloadJSON"); err != nil {
			return err
		}
	}
	if !tx.Migrator().HasColumn(&models.IMMessage{}, "PayloadJSON") {
		if err := tx.Migrator().AddColumn(&models.IMMessage{}, "PayloadJSON"); err != nil {
			return err
		}
	}
	return tx.Model(&models.IMConversation{}).
		Where("conversation_type IS NULL OR conversation_type = ''").
		Update("conversation_type", "direct").Error
}

func ensureMfgRFQClarificationSourceColumns(tx *gorm.DB) error {
	columns := []string{"SourceType", "SourceKey", "SourceLabel", "SortOrder"}
	for _, column := range columns {
		if !tx.Migrator().HasColumn(&models.MfgRFQClarification{}, column) {
			if err := tx.Migrator().AddColumn(&models.MfgRFQClarification{}, column); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureMfgRFQClarificationAnswerAudit(tx *gorm.DB) error {
	columns := []string{"LastAnsweredBy", "LastAnsweredAt"}
	for _, column := range columns {
		if !tx.Migrator().HasColumn(&models.MfgRFQClarification{}, column) {
			if err := tx.Migrator().AddColumn(&models.MfgRFQClarification{}, column); err != nil {
				return err
			}
		}
	}
	return tx.AutoMigrate(&models.MfgRFQClarificationAnswerHistory{})
}

func ensureMfgRFQClarificationAnswererName(tx *gorm.DB) error {
	if !tx.Migrator().HasColumn(&models.MfgRFQClarificationAnswerHistory{}, "AnsweredByName") {
		if err := tx.Migrator().AddColumn(&models.MfgRFQClarificationAnswerHistory{}, "AnsweredByName"); err != nil {
			return err
		}
	}
	return nil
}

func ensureMfgBusinessNameColumns(tx *gorm.DB) error {
	if !tx.Migrator().HasColumn(&models.MfgRFQ{}, "BusinessName") {
		if err := tx.Migrator().AddColumn(&models.MfgRFQ{}, "BusinessName"); err != nil {
			return err
		}
	}
	if !tx.Migrator().HasColumn(&models.MfgBOMSnapshot{}, "BusinessName") {
		if err := tx.Migrator().AddColumn(&models.MfgBOMSnapshot{}, "BusinessName"); err != nil {
			return err
		}
	}
	return nil
}

func ensureMfgProductionProgressStatusColumn(tx *gorm.DB) error {
	if !tx.Migrator().HasColumn(&models.MfgProductionProgress{}, "ProgressStatus") {
		if err := tx.Migrator().AddColumn(&models.MfgProductionProgress{}, "ProgressStatus"); err != nil {
			return err
		}
	}
	return nil
}

func ensureMfgCompensationApprovalColumns(tx *gorm.DB) error {
	columns := []string{"ApprovalStatus", "ApprovedBy", "ApprovedAt", "ApprovalRemark"}
	for _, column := range columns {
		if !tx.Migrator().HasColumn(&models.MfgCompensationRecord{}, column) {
			if err := tx.Migrator().AddColumn(&models.MfgCompensationRecord{}, column); err != nil {
				return err
			}
		}
	}
	return tx.Model(&models.MfgCompensationRecord{}).
		Where("approval_status = ? OR approval_status IS NULL", "").
		Update("approval_status", "draft").Error
}

func ensureMfgDeliveryChangeFields(tx *gorm.DB) error {
	productionColumns := []string{"CustomerID", "SalesOwnerID"}
	for _, column := range productionColumns {
		if !tx.Migrator().HasColumn(&models.MfgProductionProgress{}, column) {
			if err := tx.Migrator().AddColumn(&models.MfgProductionProgress{}, column); err != nil {
				return err
			}
		}
	}
	ecnColumns := []string{"ImpactLevel", "CostImpactCurrency", "ImpactScope"}
	for _, column := range ecnColumns {
		if !tx.Migrator().HasColumn(&models.MfgECN{}, column) {
			if err := tx.Migrator().AddColumn(&models.MfgECN{}, column); err != nil {
				return err
			}
		}
	}
	return tx.AutoMigrate(&models.MfgCompensationRecord{})
}

func disableHiddenManufacturingMenus(tx *gorm.DB) error {
	return tx.Model(&models.IndustryExtensionRegistry{}).
		Where("package_code = ? AND extension_type = ? AND target_key = ? AND extension_key IN ?", "manufacturing", "menu", "global.nav", []string{"mfg_asset_service", "mfg_analytics"}).
		Update("enabled", false).Error
}

func seedPlatformPlanTemplates(tx *gorm.DB) error {
	plans := []models.PlatformPlanTemplate{
		{
			PlanCode:        "basic",
			Name:            "基础版",
			MaxUsersDefault: 20,
			StorageDefault:  1024 * 1024 * 1024,
			ModulesJSON:     mustJSON([]string{"workspace", "contacts", "leads", "im"}),
			Status:          "active",
			Version:         1,
			Remark:          "适合小团队起步",
		},
		{
			PlanCode:        "plus",
			Name:            "Plus版",
			MaxUsersDefault: 100,
			StorageDefault:  1024 * 1024 * 1024,
			ModulesJSON:     mustJSON([]string{"workspace", "enterprise", "projects", "products", "campaigns", "contacts", "leads", "opportunities", "deals", "orders", "tickets", "dashboards", "teams", "insights", "accounts", "im"}),
			Status:          "active",
			Version:         1,
			Remark:          "覆盖主流 CRM 全模块场景",
		},
	}

	for _, item := range plans {
		plan := item
		if err := tx.Where("plan_code = ?", plan.PlanCode).Assign(plan).FirstOrCreate(&plan).Error; err != nil {
			return err
		}
	}
	return nil
}

func shrinkPlansToBasicPlus(tx *gorm.DB) error {
	if tx.Migrator().HasTable("tenant_quota") {
		if err := tx.Exec("UPDATE tenant_quota SET plan_code = 'plus' WHERE plan_code IN ('standard','enterprise')").Error; err != nil {
			return err
		}
	}
	if tx.Migrator().HasTable("platform_plan_templates") {
		if err := tx.Exec("DELETE FROM platform_plan_templates WHERE plan_code IN ('standard','enterprise')").Error; err != nil {
			return err
		}
	}
	return seedPlatformPlanTemplates(tx)
}

func createPlatformGovernanceIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_tenant_quota_plan_valid_status ON tenant_quota (plan_code, valid_to)",
		"CREATE UNIQUE INDEX idx_plan_templates_code ON platform_plan_templates (plan_code)",
		"CREATE INDEX idx_plan_templates_status ON platform_plan_templates (status)",
		"CREATE INDEX idx_expiry_notices_type_date ON tenant_expiry_notices (notice_type, notice_date)",
		"CREATE INDEX idx_expiry_notices_days_left ON tenant_expiry_notices (days_left, status)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func createCRMObjectACLIndexes(tx *gorm.DB) error {
	statements := []string{
		"CREATE INDEX idx_crm_object_acl_user_level ON crm_object_acl_entries (tenant_id, user_id, access_level)",
		"CREATE INDEX idx_crm_object_acl_object_level ON crm_object_acl_entries (tenant_id, domain, object_id, access_level)",
		"CREATE INDEX idx_crm_files_parent ON crm_objects (tenant_id, domain, object_type, project_id, created_at)",
		"CREATE INDEX idx_crm_rel_file_link ON crm_relations (tenant_id, source_type, source_id, relation_type, target_type, target_id)",
	}
	for _, statement := range statements {
		if err := tx.Exec(statement).Error; err != nil {
			LogWarning("索引创建跳过或失败: %s err=%v", statement, err)
		}
	}
	return nil
}

func ensureMigrationsTable() error {
	return DB.Table(migrationsTableName).AutoMigrate(&SchemaMigration{})
}

func isMigrationApplied(version string) (bool, error) {
	var count int64
	err := DB.Table(migrationsTableName).
		Where("version = ?", version).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
