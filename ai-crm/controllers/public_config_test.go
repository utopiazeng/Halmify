package controllers

import "testing"

func TestResolveMaintenanceNoticeText(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		locale string
		want   string
	}{
		{name: "empty", raw: "", locale: "zh-CN", want: ""},
		{name: "plain text", raw: "Scheduled maintenance", locale: "en-US", want: "Scheduled maintenance"},
		{name: "exact locale", raw: `{"default":"Default","zh-CN":"中文公告","en-US":"English notice"}`, locale: "zh-CN", want: "中文公告"},
		{name: "language fallback", raw: `{"default":"Default","en":"English base"}`, locale: "en-US", want: "English base"},
		{name: "default fallback", raw: `{"default":"Default"}`, locale: "ja-JP", want: "Default"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveMaintenanceNoticeText(tc.raw, tc.locale); got != tc.want {
				t.Fatalf("resolveMaintenanceNoticeText() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeMaintenanceNoticeLocale(t *testing.T) {
	if got := normalizeMaintenanceNoticeLocale("en"); got != "en-US" {
		t.Fatalf("normalizeMaintenanceNoticeLocale(en) = %q", got)
	}
	if got := normalizeMaintenanceNoticeLocale("ar-sa"); got != "ar-SA" {
		t.Fatalf("normalizeMaintenanceNoticeLocale(ar-sa) = %q", got)
	}
	if got := normalizeMaintenanceNoticeLocale("unknown"); got != "zh-CN" {
		t.Fatalf("normalizeMaintenanceNoticeLocale(unknown) = %q", got)
	}
}

func TestReadMaintenanceNoticeConfigPrefersEnvironment(t *testing.T) {
	t.Setenv("CRM_MAINTENANCE_NOTICE", "env notice")
	if got := readMaintenanceNoticeConfig(); got != "env notice" {
		t.Fatalf("readMaintenanceNoticeConfig() = %q, want env notice", got)
	}
}
