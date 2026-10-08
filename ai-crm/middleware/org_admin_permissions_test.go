package middleware

import (
	"net/http"
	"testing"

	"ai-crm/utils"
)

func TestTenantAdminClaimsDoesNotUseOrdinaryAdminRole(t *testing.T) {
	claims := &utils.JWTClaims{TenantID: 1, UserID: 11, AccountType: "sub_user", Role: "admin"}
	if isTenantAdminClaims(claims) {
		t.Fatalf("ordinary role=admin must not be treated as tenant admin")
	}
}

func TestTenantAdminOnlyPath(t *testing.T) {
	adminPaths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/rbac/permission-catalog"},
		{http.MethodGet, "/api/v1/system/departments"},
		{http.MethodGet, "/api/v1/system/posts"},
		{http.MethodGet, "/api/v1/operation-logs"},
		{http.MethodGet, "/api/v1/iot/devices"},
		{http.MethodGet, "/api/v1/storage/usage"},
		{http.MethodPost, "/api/v1/accounts/users"},
	}
	for _, item := range adminPaths {
		if !isTenantAdminOnlyPath(item.method, item.path, "") {
			t.Fatalf("expected %s %s to be tenant-admin-only", item.method, item.path)
		}
	}
	if isTenantAdminOnlyPath(http.MethodGet, "/api/v1/accounts/users", "") {
		t.Fatalf("employee picker must stay available to tenant users")
	}
	if isTenantAdminOnlyPath(http.MethodGet, "/api/v1/system/menus", "1") {
		t.Fatalf("visible menu endpoint must stay available to tenant users")
	}
}
