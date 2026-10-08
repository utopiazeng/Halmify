package controllers

import (
	"strings"
	"testing"
)

func TestTenantAdminIdentityDoesNotUseOrdinaryAdminRole(t *testing.T) {
	auth := authContext{TenantID: 1, UserID: 11, AccountType: "sub_user", Role: "admin"}
	if auth.IsTenantAdmin() {
		t.Fatalf("ordinary role=admin must not be treated as tenant admin")
	}
}

func TestOrganizationPermissionsAreNotAssignableToRoles(t *testing.T) {
	source := []string{
		"accounts:manage",
		"enterprise:view",
		"storage:view",
		"iot:manage",
		"audit_log:view",
		"system:department:view",
		"system:post:update",
		"campaigns:view",
		"mfg:quote:approve",
	}
	got := filterAssignablePermissions(source)
	want := []string{"campaigns:view", "mfg:quote:approve"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("filterAssignablePermissions() = %#v, want %#v", got, want)
	}
}
