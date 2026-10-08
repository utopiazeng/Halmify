package controllers

import (
	"os"
	"testing"
)

func TestBuildActivationLinkUsesEnv(t *testing.T) {
	_ = os.Setenv("TENANT_ADMIN_ACTIVATION_BASE_URL", "https://crm.example.com")
	defer os.Unsetenv("TENANT_ADMIN_ACTIVATION_BASE_URL")

	link := buildActivationLink("tok123")
	expected := "https://crm.example.com/activate-account?token=tok123"
	if link != expected {
		t.Fatalf("expected %s, got %s", expected, link)
	}
}

func TestActivationTokenTTLUsesEnv(t *testing.T) {
	_ = os.Setenv("TENANT_ADMIN_ACTIVATION_TTL_HOURS", "48")
	defer os.Unsetenv("TENANT_ADMIN_ACTIVATION_TTL_HOURS")

	if got := activationTokenTTL().Hours(); got != 48 {
		t.Fatalf("expected ttl 48h, got %.0fh", got)
	}
}
