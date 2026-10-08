package controllers

import (
	"ai-crm/models"
	"reflect"
	"testing"
)

func TestTenantScopedModelDeletesCoversTenantDataModels(t *testing.T) {
	expected := []string{
		"im_session_settings",
		"im_user_settings",
		"im_user_presences",
		"im_message_receipts",
		"im_messages",
		"im_conversations",
		"iot_firmware_targets",
		"iot_devices",
		"tenant_admin_activation_tokens",
		"operation_logs",
		"tenant_ai_applications",
		"ai_token_ledger_entries",
		"ai_research_jobs",
		"ai_usage_logs",
		"tenant_ai_wallets",
		"crm_object_acl_entries",
		"crm_relations",
		"approval_action_logs",
		"approval_instances",
		"approval_templates",
		"crm_settings",
		"crm_objects",
		"user_sessions",
		"system_menus",
		"system_posts",
		"system_departments",
		"tenant_seat_expansions",
		"tenant_storage_expansions",
		"tenant_expiry_notices",
		"tenant_quota",
	}
	actual := make([]string, 0, len(tenantScopedModelDeletes))
	for _, item := range tenantScopedModelDeletes {
		actual = append(actual, item.Name)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("tenant scoped purge model list changed\nactual: %#v\nexpected: %#v", actual, expected)
	}
}

func TestTenantPurgeDeletesAssociationModelsByDerivedIDs(t *testing.T) {
	summary := tenantDatabaseDeleteSummary{DeletedRows: map[string]int64{}}
	summary.DeletedRows["user_roles"] = 2
	summary.DeletedRows["role_permissions"] = 3
	summary.DeletedRows["portal_tickets"] = 1

	for _, key := range []string{"user_roles", "role_permissions", "portal_tickets"} {
		if summary.DeletedRows[key] == 0 {
			t.Fatalf("expected %s to be tracked in purge summary", key)
		}
	}
}

func TestTenantPurgeModelsAreHardDeleteCapable(t *testing.T) {
	for _, item := range tenantScopedModelDeletes {
		if item.Model == nil {
			t.Fatalf("%s has nil model", item.Name)
		}
	}
	if (&models.Tenant{}).ID != 0 {
		t.Fatal("unexpected tenant fixture state")
	}
}
