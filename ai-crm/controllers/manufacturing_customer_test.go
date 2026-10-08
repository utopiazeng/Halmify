package controllers

import (
	"ai-crm/models"
	"testing"
)

func TestIsManufacturingReadPermission(t *testing.T) {
	cases := []struct {
		name       string
		permission string
		want       bool
	}{
		{name: "root view", permission: "mfg:view", want: true},
		{name: "resource view", permission: "mfg:rfq:view", want: true},
		{name: "metrics view", permission: "mfg:metrics:view", want: true},
		{name: "manage is not read", permission: "mfg:rfq:manage", want: false},
		{name: "approval is not read", permission: "mfg:quote:approve", want: false},
		{name: "sensitive view is not package read", permission: "mfg:cost:view_sensitive", want: false},
		{name: "empty", permission: "", want: false},
		{name: "non manufacturing", permission: "orders:view", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isManufacturingReadPermission(tc.permission); got != tc.want {
				t.Fatalf("isManufacturingReadPermission(%q) = %v, want %v", tc.permission, got, tc.want)
			}
		})
	}
}

func TestManufacturingRFQPermissionHelpers(t *testing.T) {
	creator := authContext{TenantID: 1, UserID: 11}
	rfq := models.MfgRFQ{TenantID: 1, CreatedBy: 11}

	if !canManageManufacturingRFQ(creator, rfq) {
		t.Fatalf("RFQ creator should manage own RFQ")
	}
}
