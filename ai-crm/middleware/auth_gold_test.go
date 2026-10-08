package middleware

import (
	"ai-crm/utils"
	"net/http"
	"reflect"
	"testing"
)

func TestGoldPermissionByPathCoreMappings(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{name: "prd leads list", method: http.MethodGet, path: "/api/v1/prd/leads/objects", want: "leads:view"},
		{name: "prd leads create", method: http.MethodPost, path: "/api/v1/prd/leads/objects", want: "leads:create"},
		{name: "prd leads update", method: http.MethodPut, path: "/api/v1/prd/leads/objects/12", want: "leads:update"},
		{name: "prd leads delete", method: http.MethodDelete, path: "/api/v1/prd/leads/objects/12", want: "leads:delete"},
		{name: "lead export", method: http.MethodGet, path: "/api/v1/prd/leads/objects/export", want: "leads:export"},
		{name: "lead import", method: http.MethodPost, path: "/api/v1/prd/leads/objects/import", want: "leads:create"},
		{name: "generic object export", method: http.MethodGet, path: "/api/v1/prd/orders/objects/export", want: "orders:export"},
		{name: "generic object import", method: http.MethodPost, path: "/api/v1/prd/orders/objects/import", want: "orders:create"},
		{name: "files read", method: http.MethodGet, path: "/api/v1/files/file_1", want: "projects:view"},
		{name: "files upload", method: http.MethodPost, path: "/api/v1/files/upload", want: "projects:create"},
		{name: "files delete", method: http.MethodDelete, path: "/api/v1/files/file_1", want: "projects:delete"},
		{name: "disk space read", method: http.MethodGet, path: "/api/v1/disk/space", want: "projects:view"},
		{name: "im websocket", method: http.MethodGet, path: "/api/v1/im/ws", want: "im:view"},
		{name: "im send", method: http.MethodPost, path: "/api/v1/im/conversations", want: "im:create"},
		{name: "iot read", method: http.MethodGet, path: "/api/v1/iot/devices", want: "iot:view"},
		{name: "iot manage", method: http.MethodPost, path: "/api/v1/iot/devices", want: "iot:manage"},
		{name: "mfg rfq clarification answer", method: http.MethodPut, path: "/api/v1/mfg/rfq-clarifications/8", want: "mfg:rfq:view"},
		{name: "mfg rfq clarification delete", method: http.MethodDelete, path: "/api/v1/mfg/rfq-clarifications/8", want: "mfg:rfq:manage"},
		{name: "mfg rfq clarification history", method: http.MethodGet, path: "/api/v1/mfg/rfq-clarifications/8/answer-histories", want: "mfg:rfq:view"},
		{name: "mfg rfq creator bom snapshot", method: http.MethodPost, path: "/api/v1/mfg/bom-snapshots/from-config", want: "mfg:product:view"},
		{name: "mfg rfq creator calculate cost", method: http.MethodPost, path: "/api/v1/mfg/cost-sheets/calculate", want: "mfg:cost:view"},
		{name: "mfg rfq creator complete cost", method: http.MethodPost, path: "/api/v1/mfg/cost-sheets/8/complete", want: "mfg:cost:view"},
		{name: "mfg rfq creator create quote", method: http.MethodPost, path: "/api/v1/mfg/quotes", want: "mfg:quote:view"},
		{name: "mfg rfq creator submit quote", method: http.MethodPost, path: "/api/v1/mfg/quotes/8/submit-approval", want: "mfg:quote:view"},
		{name: "mfg quote approve", method: http.MethodPost, path: "/api/v1/mfg/quotes/8/approve", want: "mfg:quote:approve"},
		{name: "mfg quote reject", method: http.MethodPost, path: "/api/v1/mfg/quotes/8/reject", want: "mfg:quote:approve"},
		{name: "mfg rfq creator send quote", method: http.MethodPost, path: "/api/v1/mfg/quotes/8/send", want: "mfg:quote:view"},
		{name: "mfg rfq creator quote revision", method: http.MethodPost, path: "/api/v1/mfg/quotes/8/revisions", want: "mfg:quote:view"},
		{name: "mfg quote manage fallback still required", method: http.MethodPost, path: "/api/v1/mfg/quotes/8/unknown-action", want: "mfg:quote:manage"},
		{name: "mfg refresh metrics manage", method: http.MethodPost, path: "/api/v1/mfg/dashboard/refresh-metrics", want: "mfg:metrics:manage"},
		{name: "mfg cost write", method: http.MethodPost, path: "/api/v1/mfg/cost-sheets", want: "mfg:cost:manage"},
		{name: "operation log list", method: http.MethodGet, path: "/api/v1/operation-logs", want: "audit_log:view"},
		{name: "operation log export", method: http.MethodGet, path: "/api/v1/operation-logs/export", want: "audit_log:export"},
		{name: "account self language", method: http.MethodPut, path: "/api/v1/accounts/me/language", want: ""},
		{name: "account user picker", method: http.MethodGet, path: "/api/v1/accounts/users", want: ""},
		{name: "account user create", method: http.MethodPost, path: "/api/v1/accounts/users", want: "accounts:create"},
		{name: "industry package manifest", method: http.MethodGet, path: "/api/v1/prd/extensions/manifest", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := permissionByPath(tt.method, tt.path); got != tt.want {
				t.Fatalf("permissionByPath(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestGoldPermissionByFlowPathMappings(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{name: "lead assign", method: http.MethodPost, path: "/api/v1/prd/flows/leads/42/assign", want: "leads:assign"},
		{name: "lead auto assign", method: http.MethodPost, path: "/api/v1/prd/flows/leads/auto-assign", want: "leads:assign"},
		{name: "deal chain view", method: http.MethodGet, path: "/api/v1/prd/flows/deal-chain/summary", want: "deals:view"},
		{name: "opportunity review update", method: http.MethodPatch, path: "/api/v1/prd/flows/opportunity-reviews/8", want: "opportunities:update"},
		{name: "quote send update", method: http.MethodPost, path: "/api/v1/prd/flows/quotes/8/send", want: "opportunities:update"},
		{name: "quote order update", method: http.MethodPost, path: "/api/v1/prd/flows/quotes/8/order", want: "opportunities:update"},
		{name: "approval action", method: http.MethodPost, path: "/api/v1/prd/flows/approvals/9/action", want: "workspace:approve"},
		{name: "tenant ai export", method: http.MethodGet, path: "/api/v1/prd/flows/tenant-ai-analysis/reports/export.pdf", want: "insights:export"},
		{name: "tenant ai usage", method: http.MethodGet, path: "/api/v1/prd/flows/tenant-ai-analysis/usage/daily", want: "ai_usage:view"},
		{name: "sales target view", method: http.MethodGet, path: "/api/v1/prd/flows/sales-targets", want: "dashboards:view"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := permissionByPath(tt.method, tt.path); got != tt.want {
				t.Fatalf("permissionByPath(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestGoldModuleByPathMappings(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/api/v1/prd/cockpit", want: "dashboards"},
		{path: "/api/v1/prd/ai/reports", want: "insights"},
		{path: "/api/v1/prd/next-actions", want: "workspace"},
		{path: "/api/v1/prd/settings", want: "enterprise"},
		{path: "/api/v1/prd/flows/deal-chain", want: "deals"},
		{path: "/api/v1/prd/flows/opportunity-reviews", want: "opportunities"},
		{path: "/api/v1/prd/flows/quotes/8/send", want: "opportunities"},
		{path: "/api/v1/prd/flows/sales-targets", want: "dashboards"},
		{path: "/api/v1/im/conversations", want: "im"},
		{path: "/api/v1/iot/devices", want: "iot"},
		{path: "/api/v1/files/upload", want: "projects"},
		{path: "/api/v1/system/users", want: "system"},
		{path: "/api/v1/accounts/users", want: "accounts"},
		{path: "/api/v1/operation-logs", want: "audit_log"},
		{path: "/api/v1/tenants/1", want: "enterprise"},
		{path: "/api/v1/unknown", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := moduleByPath(tt.path); got != tt.want {
				t.Fatalf("moduleByPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestGoldPermissionGrantedWithManageFallback(t *testing.T) {
	permissionSet := map[string]bool{
		"leads:manage": true,
	}
	allowed := []string{
		"leads:view",
		"leads:create",
		"leads:update",
		"leads:delete",
		"leads:export",
		"leads:assign",
		"leads:approve",
	}
	for _, required := range allowed {
		t.Run("allow "+required, func(t *testing.T) {
			if !permissionGranted(permissionSet, required) {
				t.Fatalf("expected %q to be granted by leads:manage", required)
			}
		})
	}

	denied := []string{"contacts:view", "leads:archive", "broken"}
	for _, required := range denied {
		t.Run("deny "+required, func(t *testing.T) {
			if permissionGranted(permissionSet, required) {
				t.Fatalf("expected %q to be denied by permission set %#v", required, permissionSet)
			}
		})
	}

	if !permissionGranted(nil, "") {
		t.Fatal("empty required permission should be allowed")
	}
}

func TestGoldPermissionCandidates(t *testing.T) {
	tests := []struct {
		required string
		want     []string
	}{
		{required: "", want: []string{}},
		{required: "leads:view", want: []string{"leads:view", "leads:manage"}},
		{required: "leads:create", want: []string{"leads:create", "leads:manage"}},
		{required: "leads:update", want: []string{"leads:update", "leads:manage"}},
		{required: "leads:delete", want: []string{"leads:delete", "leads:manage"}},
		{required: "leads:export", want: []string{"leads:export", "leads:manage"}},
		{required: "leads:assign", want: []string{"leads:assign", "leads:manage"}},
		{required: "leads:approve", want: []string{"leads:approve", "leads:manage"}},
		{required: "leads:manage", want: []string{"leads:manage"}},
		{required: "invalid", want: []string{"invalid"}},
		{required: ":", want: []string{":"}},
	}

	for _, tt := range tests {
		t.Run(tt.required, func(t *testing.T) {
			if got := permissionCandidates(tt.required); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("permissionCandidates(%q) = %#v, want %#v", tt.required, got, tt.want)
			}
		})
	}
}

func TestGoldManufacturingReadPermission(t *testing.T) {
	allowed := []string{"mfg:view", "mfg:rfq:view", "mfg:product:view", "mfg:cost:view"}
	for _, permission := range allowed {
		t.Run("read "+permission, func(t *testing.T) {
			if !isManufacturingReadPermission(permission) {
				t.Fatalf("expected %q to be treated as manufacturing read permission", permission)
			}
		})
	}
	denied := []string{"mfg:rfq:manage", "mfg:quote:approve", "mfg:cost:view_sensitive", "mfg:metrics:manage", "rfq:view"}
	for _, permission := range denied {
		t.Run("write "+permission, func(t *testing.T) {
			if isManufacturingReadPermission(permission) {
				t.Fatalf("expected %q to require explicit role permission", permission)
			}
		})
	}
}

func TestGoldPlatformAdminClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims *utils.JWTClaims
		want   bool
	}{
		{name: "nil claims", claims: nil, want: false},
		{name: "account super admin", claims: &utils.JWTClaims{AccountType: "super_admin", TenantID: 99}, want: true},
		{name: "account platform super admin", claims: &utils.JWTClaims{AccountType: "platform_super_admin", TenantID: 99}, want: true},
		{name: "role super admin", claims: &utils.JWTClaims{Role: "super_admin", TenantID: 99}, want: true},
		{name: "role platform super admin", claims: &utils.JWTClaims{Role: "platform_super_admin", TenantID: 99}, want: true},
		{name: "admin without tenant", claims: &utils.JWTClaims{AccountType: "admin", TenantID: 0}, want: true},
		{name: "admin with tenant", claims: &utils.JWTClaims{AccountType: "admin", TenantID: 1}, want: false},
		{name: "tenant admin", claims: &utils.JWTClaims{AccountType: "tenant_admin", Role: "tenant_admin", TenantID: 1}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPlatformAdminClaims(tt.claims); got != tt.want {
				t.Fatalf("isPlatformAdminClaims(%#v) = %v, want %v", tt.claims, got, tt.want)
			}
		})
	}
}
