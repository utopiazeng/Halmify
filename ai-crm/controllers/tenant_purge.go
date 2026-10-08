package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"

	"gorm.io/gorm"
)

type tenantDatabaseDeleteSummary struct {
	DeletedRows map[string]int64 `json:"deleted_rows"`
	TotalRows   int64            `json:"total_rows"`
}

type tenantPurgeResult struct {
	DeletedTenantID uint                        `json:"deleted_tenant_id"`
	FileCleanup     tenantFileDeleteSummary     `json:"file_cleanup"`
	DBCleanup       tenantDatabaseDeleteSummary `json:"db_cleanup"`
}

type tenantScopedModelDelete struct {
	Name  string
	Model interface{}
	Table string
}

var tenantScopedModelDeletes = []tenantScopedModelDelete{
	{Name: "im_session_settings", Model: &models.IMSessionSetting{}},
	{Name: "im_user_settings", Model: &models.IMUserSetting{}},
	{Name: "im_user_presences", Model: &models.IMUserPresence{}},
	{Name: "im_message_receipts", Model: &models.IMMessageReceipt{}},
	{Name: "im_messages", Model: &models.IMMessage{}},
	{Name: "im_conversations", Model: &models.IMConversation{}},
	{Name: "iot_firmware_targets", Model: &models.IoTFirmwareTarget{}},
	{Name: "iot_devices", Model: &models.IoTDevice{}},
	{Name: "tenant_admin_activation_tokens", Model: &models.TenantAdminActivationToken{}},
	{Name: "operation_logs", Model: &models.OperationLog{}},
	{Name: "tenant_ai_applications", Model: &models.TenantAIApplication{}},
	{Name: "ai_token_ledger_entries", Model: &models.AITokenLedgerEntry{}},
	{Name: "ai_research_jobs", Model: &models.AIResearchJob{}},
	{Name: "ai_usage_logs", Model: &models.AIUsageLog{}},
	{Name: "tenant_ai_wallets", Model: &models.TenantAIWallet{}},
	{Name: "crm_object_acl_entries", Model: &models.CRMObjectACLEntry{}},
	{Name: "crm_relations", Model: &models.CRMRelation{}},
	{Name: "approval_action_logs", Model: &models.ApprovalActionLog{}},
	{Name: "approval_instances", Model: &models.ApprovalInstance{}},
	{Name: "approval_templates", Model: &models.ApprovalTemplate{}},
	{Name: "crm_settings", Model: &models.CRMSetting{}},
	{Name: "crm_objects", Model: &models.CRMObject{}},
	{Name: "user_sessions", Model: &models.UserSession{}},
	{Name: "system_menus", Model: &models.SystemMenu{}},
	{Name: "system_posts", Model: &models.SystemPost{}},
	{Name: "system_departments", Model: &models.SystemDepartment{}},
	{Name: "tenant_seat_expansions", Model: &models.TenantSeatExpansion{}},
	{Name: "tenant_storage_expansions", Model: &models.TenantStorageExpansion{}},
	{Name: "tenant_expiry_notices", Model: &models.TenantExpiryNotice{}},
	{Name: "tenant_quota", Model: &models.TenantQuota{}, Table: "tenant_quota"},
}

func purgeTenantCompletely(auth authContext, tenantID uint) (tenantPurgeResult, error) {
	result := tenantPurgeResult{DeletedTenantID: tenantID}
	if tenantID == 0 {
		return result, fmt.Errorf("无效的租户ID")
	}
	var tenant models.Tenant
	if err := utils.DB.First(&tenant, tenantID).Error; err != nil {
		return result, err
	}

	fileCleanup, err := cleanupTenantPhysicalFiles(auth, tenantID)
	result.FileCleanup = fileCleanup
	if err != nil {
		return result, err
	}

	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		dbCleanup, err := purgeTenantDatabaseTx(tx, tenantID)
		result.DBCleanup = dbCleanup
		if err != nil {
			return err
		}
		deleted, err := hardDeleteTenantModel(tx, "tenants", &models.Tenant{}, "id = ?", tenantID)
		if err != nil {
			return err
		}
		result.DBCleanup.DeletedRows["tenants"] = deleted
		result.DBCleanup.TotalRows += deleted
		return nil
	})
	return result, err
}

func purgeTenantDatabaseTx(tx *gorm.DB, tenantID uint) (tenantDatabaseDeleteSummary, error) {
	summary := tenantDatabaseDeleteSummary{DeletedRows: map[string]int64{}}

	var userIDs []uint
	if err := tx.Unscoped().Model(&models.User{}).Where("tenant_id = ?", tenantID).Pluck("id", &userIDs).Error; err != nil {
		return summary, err
	}
	var roleIDs []uint
	if err := tx.Unscoped().Model(&models.Role{}).Where("tenant_id = ?", tenantID).Pluck("id", &roleIDs).Error; err != nil {
		return summary, err
	}
	var crmObjectIDs []uint
	if err := tx.Unscoped().Model(&models.CRMObject{}).Where("tenant_id = ?", tenantID).Pluck("id", &crmObjectIDs).Error; err != nil {
		return summary, err
	}

	if len(userIDs) > 0 {
		if err := addTenantDeleteCount(tx.Unscoped().Where("user_id IN ?", userIDs).Delete(&models.UserRole{}), summary.DeletedRows, "user_roles"); err != nil {
			return summary, err
		}
	}
	if len(roleIDs) > 0 {
		if err := addTenantDeleteCount(tx.Unscoped().Where("role_id IN ?", roleIDs).Delete(&models.RolePermission{}), summary.DeletedRows, "role_permissions"); err != nil {
			return summary, err
		}
	}
	if len(crmObjectIDs) > 0 {
		if err := addTenantDeleteCount(tx.Unscoped().Where("ticket_id IN ?", crmObjectIDs).Delete(&models.PortalTicket{}), summary.DeletedRows, "portal_tickets"); err != nil {
			return summary, err
		}
	}

	for _, item := range tenantScopedModelDeletes {
		deleteTx := tx
		if item.Table != "" {
			deleteTx = deleteTx.Table(item.Table)
		}
		deleted, err := hardDeleteTenantModel(deleteTx, item.Name, item.Model, "tenant_id = ?", tenantID)
		if err != nil {
			return summary, err
		}
		summary.DeletedRows[item.Name] += deleted
	}

	deletedUsers, err := hardDeleteTenantModel(tx, "users", &models.User{}, "tenant_id = ?", tenantID)
	if err != nil {
		return summary, err
	}
	summary.DeletedRows["users"] += deletedUsers

	deletedRoles, err := hardDeleteTenantModel(tx, "roles", &models.Role{}, "tenant_id = ?", tenantID)
	if err != nil {
		return summary, err
	}
	summary.DeletedRows["roles"] += deletedRoles

	for _, count := range summary.DeletedRows {
		summary.TotalRows += count
	}
	return summary, nil
}

func hardDeleteTenantModel(tx *gorm.DB, tableName string, model interface{}, query string, args ...interface{}) (int64, error) {
	result := tx.Unscoped().Where(query, args...).Delete(model)
	if result.Error != nil {
		return 0, fmt.Errorf("删除 %s 失败: %w", tableName, result.Error)
	}
	return result.RowsAffected, nil
}

func addTenantDeleteCount(result *gorm.DB, deletedRows map[string]int64, tableName string) error {
	if result.Error != nil {
		return fmt.Errorf("删除 %s 失败: %w", tableName, result.Error)
	}
	deletedRows[tableName] += result.RowsAffected
	return nil
}
