package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/beego/beego/v2/server/web/context"
)

func testContextWithLanguage(language string) *context.Context {
	req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
	if language != "" {
		req.Header.Set("Accept-Language", language)
	}
	return &context.Context{Request: req}
}

func testResponseContextWithLanguage(language string) (*context.Context, *httptest.ResponseRecorder) {
	req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
	if language != "" {
		req.Header.Set("Accept-Language", language)
	}
	recorder := httptest.NewRecorder()
	ctx := context.NewContext()
	ctx.Reset(recorder, req)
	return ctx, recorder
}

func TestLocalizedErrorMessageUsesAcceptLanguage(t *testing.T) {
	ctx := testContextWithLanguage("en-US")
	got := LocalizedErrorMessage(ctx, "auth.invalidCredentials", nil, "用户名或密码错误", http.StatusUnauthorized)
	if got != "Incorrect username or password" {
		t.Fatalf("expected English message, got %q", got)
	}
}

func TestLocalizedErrorMessageUsesChinese(t *testing.T) {
	ctx := testContextWithLanguage("zh-CN")
	got := LocalizedErrorMessage(ctx, "auth.invalidCredentials", nil, "用户名或密码错误", http.StatusUnauthorized)
	if got != "用户名或密码错误" {
		t.Fatalf("expected Chinese message, got %q", got)
	}
}

func TestRequestLocaleFallsBackToChinese(t *testing.T) {
	ctx := testContextWithLanguage("xx-YY")
	got := RequestLocale(ctx)
	if got != "zh-CN" {
		t.Fatalf("expected zh-CN fallback, got %q", got)
	}
}

func TestLocalizedErrorMessageInterpolatesParams(t *testing.T) {
	ctx := testContextWithLanguage("en-US")
	got := LocalizedErrorMessage(ctx, "file.serviceStatusFailed", map[string]interface{}{"status": 503}, "", http.StatusBadGateway)
	if got != "File service returned 503" {
		t.Fatalf("expected interpolated message, got %q", got)
	}
}

func TestResolveErrorMessageKeyMapsKnownChineseText(t *testing.T) {
	key, _ := ResolveErrorMessageKey("用户名或密码错误", http.StatusUnauthorized)
	if key != "auth.invalidCredentials" {
		t.Fatalf("expected auth.invalidCredentials, got %q", key)
	}
}

func TestErrorKeyResponseIncludesProtocolFields(t *testing.T) {
	ctx, recorder := testResponseContextWithLanguage("en-US")
	ErrorKey(ctx, http.StatusUnauthorized, "auth.invalidCredentials", nil)

	var body Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if body.Message != "Incorrect username or password" {
		t.Fatalf("expected localized message, got %q", body.Message)
	}
	if body.ErrorCode != "AUTH_INVALID_CREDENTIALS" {
		t.Fatalf("expected error code, got %q", body.ErrorCode)
	}
	if body.MessageKey != "auth.invalidCredentials" {
		t.Fatalf("expected message key, got %q", body.MessageKey)
	}
}

func TestUnauthorizedKeyResponseForMissingToken(t *testing.T) {
	ctx, recorder := testResponseContextWithLanguage("en-US")
	UnauthorizedKey(ctx, "auth.tokenMissing", nil)

	var body Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Message != "Authentication token is missing" {
		t.Fatalf("expected token message, got %q", body.Message)
	}
	if body.ErrorCode != "AUTH_TOKEN_MISSING" || body.MessageKey != "auth.tokenMissing" {
		t.Fatalf("expected token protocol fields, got code=%q key=%q", body.ErrorCode, body.MessageKey)
	}
}

func TestForbiddenLegacyTenantDisabledKeepsErrorCode(t *testing.T) {
	ctx, recorder := testResponseContextWithLanguage("en-US")
	ErrorWithData(ctx, http.StatusForbidden, "TENANT_DISABLED", map[string]interface{}{"error_code": "TENANT_DISABLED"})

	var body Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Message != "This tenant has been disabled" {
		t.Fatalf("expected tenant disabled message, got %q", body.Message)
	}
	if body.ErrorCode != "TENANT_DISABLED" {
		t.Fatalf("expected legacy error code, got %q", body.ErrorCode)
	}
	if body.MessageKey != "auth.tenantDisabled" {
		t.Fatalf("expected tenant message key, got %q", body.MessageKey)
	}
}
