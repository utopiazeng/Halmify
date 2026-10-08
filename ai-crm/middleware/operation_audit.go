package middleware

import (
	"ai-crm/services"
	"fmt"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/server/web/context"
)

func OperationAudit(ctx *context.Context) {
	path := ctx.Request.URL.Path
	if !strings.HasPrefix(path, "/api/v1/") || shouldSkipAudit(path) {
		return
	}
	if ctx.Request.Method == http.MethodGet || ctx.Request.Method == http.MethodOptions {
		return
	}

	userID := dataUint(ctx, "user_id")
	if userID == 0 {
		return
	}
	operation := operationByRequest(ctx.Request.Method, path)
	if operation == "" {
		return
	}
	status := ctx.Output.Status
	result := "success"
	reason := ""
	if status >= http.StatusBadRequest {
		result = "failed"
		reason = http.StatusText(status)
	}
	services.LogOperation(services.OperationLogEvent{
		TenantID:       dataUint(ctx, "tenant_id"),
		UserID:         userID,
		Username:       dataString(ctx, "username"),
		Operation:      operation,
		Target:         auditTarget(path),
		DescriptionKey: "operation." + strings.ToLower(operation),
		Params: map[string]interface{}{
			"method": ctx.Request.Method,
			"path":   path,
			"query":  ctx.Request.URL.RawQuery,
		},
		Result:    result,
		Reason:    reason,
		IP:        ctx.Input.IP(),
		UserAgent: ctx.Request.UserAgent(),
		Source:    "web",
	})
}

func shouldSkipAudit(path string) bool {
	return strings.HasPrefix(path, "/api/v1/auth/") ||
		strings.HasPrefix(path, "/api/v1/operation-logs") ||
		strings.HasPrefix(path, "/api/v1/portal/")
}

func operationByRequest(method, path string) string {
	module := moduleByPath(path)
	if module == "" {
		module = "system"
	}
	action := "UPDATE"
	switch method {
	case http.MethodPost:
		action = "CREATE"
	case http.MethodPut, http.MethodPatch:
		action = "UPDATE"
	case http.MethodDelete:
		action = "DELETE"
	default:
		return ""
	}
	return action + "_" + strings.ToUpper(module)
}

func auditTarget(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return path
	}
	last := parts[len(parts)-1]
	if last == "" {
		return path
	}
	return fmt.Sprintf("%s[%s]", parts[len(parts)-2], last)
}

func dataUint(ctx *context.Context, key string) uint {
	value := ctx.Input.GetData(key)
	switch typed := value.(type) {
	case uint:
		return typed
	case int:
		if typed > 0 {
			return uint(typed)
		}
	case int64:
		if typed > 0 {
			return uint(typed)
		}
	}
	return 0
}

func dataString(ctx *context.Context, key string) string {
	value, _ := ctx.Input.GetData(key).(string)
	return value
}
