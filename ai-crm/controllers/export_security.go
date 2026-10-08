package controllers

import (
	"ai-crm/utils"
	"strings"
	"time"

	bcontext "github.com/beego/beego/v2/server/web/context"
)

type exportSecurityContext struct {
	TraceID    string
	TenantID   uint
	UserID     uint
	Username   string
	IP         string
	ExportedAt time.Time
}

func newExportSecurityContext(ctx *bcontext.Context, auth authContext) exportSecurityContext {
	traceID := strings.TrimSpace(utils.NewSessionID())
	if len(traceID) > 16 {
		traceID = traceID[:16]
	}
	if traceID == "" {
		traceID = time.Now().Format("20060102150405")
	}
	return exportSecurityContext{
		TraceID:    traceID,
		TenantID:   auth.TenantID,
		UserID:     auth.UserID,
		Username:   auth.Username,
		IP:         ctx.Input.IP(),
		ExportedAt: time.Now(),
	}
}

func (e exportSecurityContext) WatermarkTable() excelTable {
	return excelTable{
		Title:   "导出追踪水印",
		Headers: []string{"字段", "值"},
		Rows: [][]interface{}{
			{"trace_id", e.TraceID},
			{"tenant_id", e.TenantID},
			{"user_id", e.UserID},
			{"username", e.Username},
			{"ip", e.IP},
			{"exported_at", e.ExportedAt},
		},
	}
}

func (e exportSecurityContext) Params() map[string]interface{} {
	return map[string]interface{}{
		"trace_id":    e.TraceID,
		"tenant_id":   e.TenantID,
		"user_id":     e.UserID,
		"username":    e.Username,
		"ip":          e.IP,
		"exported_at": e.ExportedAt.Format(time.RFC3339),
	}
}

func applyExportTraceHeader(ctx *bcontext.Context, exportCtx exportSecurityContext) {
	ctx.Output.Header("X-Export-Trace-ID", exportCtx.TraceID)
}

func mergeExportParams(exportCtx exportSecurityContext, extra map[string]interface{}) map[string]interface{} {
	params := exportCtx.Params()
	for key, value := range extra {
		params[key] = value
	}
	return params
}
