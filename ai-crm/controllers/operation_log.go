package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
)

type OperationLogController struct {
	web.Controller
}

func (c *OperationLogController) ListLogs() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)
	if size > 100 {
		size = 100
	}
	tenantID := queryUint(c.Ctx, "tenant_id")
	user := strings.TrimSpace(c.Ctx.Input.Query("user"))
	operation := strings.TrimSpace(c.Ctx.Input.Query("operation"))
	result := strings.TrimSpace(c.Ctx.Input.Query("result"))
	start := parseTimeQuery(c.Ctx.Input.Query("start_time"))
	end := parseTimeQuery(c.Ctx.Input.Query("end_time"))

	query := scopedOperationLogQuery(auth, tenantID, user, operation, result, start, end)

	var total int64
	query.Count(&total)
	var logs []models.OperationLog
	if err := query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&logs).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取操作日志失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": logs, "total": total, "page": page, "size": size})
}

func (c *OperationLogController) ExportLogs() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID := queryUint(c.Ctx, "tenant_id")
	user := strings.TrimSpace(c.Ctx.Input.Query("user"))
	operation := strings.TrimSpace(c.Ctx.Input.Query("operation"))
	result := strings.TrimSpace(c.Ctx.Input.Query("result"))
	start := parseTimeQuery(c.Ctx.Input.Query("start_time"))
	end := parseTimeQuery(c.Ctx.Input.Query("end_time"))
	limit := queryInt(c.Ctx, "limit", 5000)
	if limit <= 0 || limit > 20000 {
		limit = 5000
	}

	var logs []models.OperationLog
	if err := scopedOperationLogQuery(auth, tenantID, user, operation, result, start, end).
		Order("created_at desc").
		Limit(limit).
		Find(&logs).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "导出操作日志失败")
		return
	}

	exportCtx := newExportSecurityContext(c.Ctx, auth)
	filename := fmt.Sprintf("operation_logs_%s.zip", exportCtx.TraceID)
	logControllerOperation(c.Ctx, auth, auth.TenantID, "EXPORT_OPERATION_LOGS", "operation_logs", "operation.export_operation_logs", mergeExportParams(exportCtx, map[string]interface{}{"count": len(logs), "operation": operation, "result": result, "limit": limit}), "success", "")
	writeOperationLogsZip(c, logs, filename, "operation_logs.xls", exportCtx)
}

func (c *OperationLogController) ExportLoginLogs() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID := queryUint(c.Ctx, "tenant_id")
	user := strings.TrimSpace(c.Ctx.Input.Query("user"))
	result := strings.TrimSpace(c.Ctx.Input.Query("result"))
	start := parseTimeQuery(c.Ctx.Input.Query("start_time"))
	end := parseTimeQuery(c.Ctx.Input.Query("end_time"))
	limit := queryInt(c.Ctx, "limit", 5000)
	if limit <= 0 || limit > 20000 {
		limit = 5000
	}
	var logs []models.OperationLog
	if err := scopedOperationLogQuery(auth, tenantID, user, "LOGIN", result, start, end).
		Order("created_at desc").
		Limit(limit).
		Find(&logs).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "导出登录日志失败")
		return
	}
	exportCtx := newExportSecurityContext(c.Ctx, auth)
	filename := fmt.Sprintf("login_logs_%s.zip", exportCtx.TraceID)
	logControllerOperation(c.Ctx, auth, auth.TenantID, "EXPORT_LOGIN_LOGS", "operation_logs", "operation.export_login_logs", mergeExportParams(exportCtx, map[string]interface{}{"count": len(logs), "result": result, "limit": limit}), "success", "")
	writeOperationLogsZip(c, logs, filename, "login_logs.xls", exportCtx)
}

func (c *OperationLogController) CleanupLogs() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		RetentionDays int    `json:"retention_days"`
		Clear         bool   `json:"clear"`
		Operation     string `json:"operation"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.Clear {
		query := utils.DB.Unscoped().Where("tenant_id = ?", auth.TenantID)
		if strings.TrimSpace(req.Operation) != "" {
			query = query.Where("operation = ?", strings.TrimSpace(req.Operation))
		}
		if err := query.Delete(&models.OperationLog{}).Error; err != nil {
			utils.Error(c.Ctx, http.StatusInternalServerError, "清空日志失败")
			return
		}
		logControllerOperation(c.Ctx, auth, auth.TenantID, "CLEANUP_OPERATION_LOGS", "operation_logs", "operation.cleanup_operation_logs", map[string]interface{}{"clear": true, "operation": req.Operation}, "success", "")
		utils.Success(c.Ctx, map[string]interface{}{"cleared": true})
		return
	}
	if req.RetentionDays <= 0 {
		req.RetentionDays = 180
	}
	if req.RetentionDays < 30 {
		utils.BadRequest(c.Ctx, "审计日志至少保留30天")
		return
	}
	retention := time.Duration(req.RetentionDays) * 24 * time.Hour
	services.CleanupOperationLogs(retention)
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CLEANUP_OPERATION_LOGS", "operation_logs", "operation.cleanup_operation_logs", map[string]interface{}{"retention_days": req.RetentionDays}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"retention_days": req.RetentionDays})
}

func writeOperationLogsZip(c *OperationLogController, logs []models.OperationLog, filename, xlsName string, exportCtx exportSecurityContext) {
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	rows := make([][]interface{}, 0, len(logs))
	for _, item := range logs {
		rows = append(rows, []interface{}{
			item.ID,
			item.TenantID,
			item.UserID,
			item.Username,
			item.CreatedAt,
			item.IP,
			item.UserAgent,
			operationLogActionLabel(item.Operation),
			operationLogTargetLabel(item),
			operationLogDescription(item),
			item.ParamsJSON,
			item.Result,
			item.Reason,
			item.Source,
		})
	}
	if err := writeExcelHTMLZipFile(zipWriter, xlsName, exportCtx.WatermarkTable(), excelTable{
		Title:   "操作日志",
		Headers: []string{"日志ID", "租户ID", "用户ID", "用户名", "时间", "IP", "设备", "操作类型", "操作对象", "描述Key", "参数JSON", "结果", "失败原因", "来源"},
		Rows:    rows,
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "写入日志文件失败")
		return
	}
	if err := zipWriter.Close(); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "生成日志压缩包失败")
		return
	}
	c.Ctx.Output.Header("Content-Type", "application/zip")
	applyExportTraceHeader(c.Ctx, exportCtx)
	c.Ctx.Output.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	_, _ = c.Ctx.ResponseWriter.Write(buffer.Bytes())
}

func scopedOperationLogQuery(auth authContext, tenantID uint, user, operation, result string, start, end *time.Time) *gorm.DB {
	query := utils.DB.Model(&models.OperationLog{})
	if auth.IsSuperAdmin() {
		query = query.Where("tenant_id = ?", 0)
	} else {
		query = query.Where("tenant_id = ?", auth.TenantID)
	}
	if user != "" {
		if userID, err := strconv.ParseUint(user, 10, 64); err == nil {
			query = query.Where("username LIKE ? OR user_id = ?", "%"+user+"%", uint(userID))
		} else {
			query = query.Where("username LIKE ?", "%"+user+"%")
		}
	}
	if operation != "" {
		if strings.HasSuffix(operation, "*") {
			prefix := strings.TrimSuffix(operation, "*")
			query = query.Where("operation LIKE ?", prefix+"%")
		} else {
			query = query.Where("operation = ?", operation)
		}
	}
	if result != "" {
		query = query.Where("result = ?", result)
	}
	if start != nil {
		query = query.Where("created_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("created_at <= ?", *end)
	}
	return query
}

func parseTimeQuery(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func operationLogActionLabel(operation string) string {
	value := strings.TrimSpace(operation)
	upper := strings.ToUpper(value)
	labels := map[string]string{
		"ADD_TICKET_NOTE":               "添加工单记录",
		"SYNC_TICKET_BUG_STATUS":        "同步缺陷状态",
		"SYNC_EXCHANGE_RATES":           "同步汇率",
		"ADD_CAMPAIGN_FOLLOW_UP":        "添加活动跟进",
		"ADD_LEAD_FOLLOW_UP":            "添加线索跟进",
		"ADD_OPPORTUNITY_FOLLOW_UP":     "添加商机跟进",
		"APPLY_COMPANY_RESEARCH":        "应用公司画像",
		"GENERATE_HOT_INSIGHTS":         "生成运营洞察",
		"CLEANUP_OPERATION_LOGS":        "清理操作日志",
		"SAVE_COMPANY_SALES_TARGET":     "保存公司销售目标",
		"ALLOCATE_SALES_TARGETS":        "分配销售目标",
		"SAVE_MY_SALES_PLAN":            "保存我的销售计划",
		"SAVE_ORDER_PAYMENT_PLAN":       "保存订单付款计划",
		"INVOICE_ORDER":                 "订单开票",
		"CLOSE_TICKET":                  "关闭工单",
		"CHANGE_TICKET_STATUS":          "变更工单状态",
		"ASSIGN_TICKET":                 "分派工单",
		"TRANSFER_TICKET":               "转派工单",
		"AUTO_ASSIGN_TICKET":            "自动分派工单",
		"CREATE_TICKET_FROM_ORDER":      "订单创建售后工单",
		"MARK_ORDER_PAID":               "登记订单回款",
		"SHIP_ORDER":                    "订单发货",
		"RETURN_ORDER":                  "订单退换货",
		"CHANGE_ORDER_STATUS":           "变更订单状态",
		"FULFILL_ORDER":                 "订单履约",
		"CREATE_ORDER_FROM_DEAL":        "成交生成订单",
		"CREATE_ORDER_FROM_QUOTE":       "报价生成订单",
		"CREATE_CRM_OBJECT":             "新增业务记录",
		"UPDATE_CRM_OBJECT":             "编辑业务记录",
		"DELETE_CRM_OBJECT":             "删除业务记录",
		"CREATE_CRM_RELATION":           "新增关联关系",
		"DELETE_CRM_RELATION":           "删除关联关系",
		"SYNC_AI_PRICING":               "同步 AI 价格",
		"BROADCAST_SYSTEM_NOTIFICATION": "发布系统通知",
		"SECURITY_RATE_LIMIT":           "安全限流阻断",
		"SECURITY_SCAN_BLOCK":           "扫描行为阻断",
		"SECURITY_EXPORT_BLOCK":         "导出行为阻断",
		"SECURITY_SESSION_REVOKED":      "风控撤销会话",
		"EXPORT_OPERATION_LOGS":         "导出操作日志",
		"EXPORT_LOGIN_LOGS":             "导出登录日志",
		"EXPORT_ORDER_BACKUP":           "导出订单备份",
	}
	if label, ok := labels[upper]; ok {
		return label
	}
	switch {
	case strings.Contains(upper, "LOGIN"):
		return "登录成功"
	case strings.Contains(upper, "SYNC"):
		return "同步"
	case strings.Contains(upper, "ADD"):
		return "添加"
	case strings.Contains(upper, "CREATE"):
		return "新增"
	case strings.Contains(upper, "UPDATE"):
		return "编辑"
	case strings.Contains(upper, "DELETE"):
		return "删除"
	}
	return formatOperationLogKey(value)
}

func operationLogTargetLabel(log models.OperationLog) string {
	target := strings.TrimSpace(log.Target)
	params := map[string]interface{}{}
	if strings.TrimSpace(log.ParamsJSON) != "" {
		_ = json.Unmarshal([]byte(log.ParamsJSON), &params)
	}
	upper := strings.ToUpper(log.Operation)
	if strings.Contains(upper, "EXCHANGE_RATE") || target == "order_finance.default" {
		return "默认汇率配置"
	}
	if strings.Contains(upper, "SALES_TARGET") || strings.Contains(upper, "SALES_PLAN") {
		if target != "" {
			return target
		}
		return "销售目标"
	}
	if id := fmt.Sprint(params["order_id"]); id != "" && id != "<nil>" {
		return "订单 #" + id
	}
	if id := fmt.Sprint(params["ticket_id"]); id != "" && id != "<nil>" {
		return "服务工单 #" + id
	}
	if id := fmt.Sprint(params["lead_id"]); id != "" && id != "<nil>" {
		return "线索 #" + id
	}
	if id := fmt.Sprint(params["opportunity_id"]); id != "" && id != "<nil>" {
		return "商机 #" + id
	}
	if domain := fmt.Sprint(params["domain"]); domain != "" && domain != "<nil>" {
		return operationLogModuleLabel(domain) + "记录"
	}
	if target == "" {
		return ""
	}
	if idx := strings.Index(target, "["); idx > 0 && strings.HasSuffix(target, "]") {
		return operationLogModuleLabel(target[:idx]) + " #" + strings.TrimSuffix(target[idx+1:], "]")
	}
	return operationLogModuleLabel(target)
}

func operationLogDescription(log models.OperationLog) string {
	if strings.TrimSpace(log.Reason) != "" {
		return log.Reason
	}
	target := operationLogTargetLabel(log)
	action := operationLogActionLabel(log.Operation)
	if target == "" {
		return action
	}
	return action + "：" + target
}

func operationLogModuleLabel(value string) string {
	text := strings.TrimSpace(value)
	labels := map[string]string{
		"order_finance":         "订单财税设置",
		"order_finance.default": "订单财税设置",
		"operation_logs":        "操作日志",
		"security":              "安全风控",
		"dashboards":            "销售大屏",
		"sales_target":          "销售目标",
		"sales_targets":         "销售目标",
		"tickets":               "服务工单",
		"ticket":                "服务工单",
		"工单":                    "服务工单",
		"leads":                 "线索",
		"lead":                  "线索",
		"opportunities":         "商机",
		"opportunity":           "商机",
		"campaigns":             "市场活动",
		"campaign":              "市场活动",
		"products":              "产品服务",
		"product":               "产品服务",
		"orders":                "订单",
		"order":                 "订单",
		"projects":              "客户项目",
		"project":               "客户项目",
	}
	if label, ok := labels[text]; ok {
		return label
	}
	first := strings.Split(text, ".")[0]
	if label, ok := labels[first]; ok {
		return label
	}
	return formatOperationLogKey(text)
}

func formatOperationLogKey(value string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(value), func(r rune) bool {
		return r == '_' || r == '.' || r == '-'
	})
	words := map[string]string{
		"ADD": "添加", "SYNC": "同步", "EXCHANGE": "汇率", "RATES": "汇率",
		"TICKET": "工单", "NOTE": "记录", "ORDER": "订单", "FINANCE": "财税",
		"DEFAULT": "默认", "CRM": "业务", "OBJECT": "记录", "RELATION": "关系",
		"CREATE": "新增", "UPDATE": "编辑", "DELETE": "删除", "CLEANUP": "清理", "LOGS": "日志",
	}
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		if label, ok := words[strings.ToUpper(part)]; ok {
			labels = append(labels, label)
		} else if strings.TrimSpace(part) != "" {
			labels = append(labels, part)
		}
	}
	if len(labels) == 0 {
		return value
	}
	return strings.Join(labels, "")
}
