package middleware

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	stdctx "context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	bcontext "github.com/beego/beego/v2/server/web/context"
	"github.com/redis/go-redis/v9"
)

const (
	securityOperationRateLimit      = "SECURITY_RATE_LIMIT"
	securityOperationScanBlock      = "SECURITY_SCAN_BLOCK"
	securityOperationExportBlock    = "SECURITY_EXPORT_BLOCK"
	securityOperationSessionRevoked = "SECURITY_SESSION_REVOKED"
)

type securityRiskRule struct {
	Name      string
	Window    time.Duration
	Limit     int
	Operation string
	Severe    bool
}

type securityRiskDecision struct {
	Allowed    bool
	StatusCode int
	Message    string
	Operation  string
	Reason     string
	TraceID    string
	RetryAfter time.Duration
	Severe     bool
	Rule       string
}

type securityRiskSubject struct {
	Dimension string
	Key       string
	Limit     int
}

type securityRiskConfig struct {
	Enabled               bool
	FailOpen              bool
	GeneralReadRule       securityRiskRule
	CRMListRule           securityRiskRule
	CRMSearchRule         securityRiskRule
	CRMDetailRule         securityRiskRule
	ExportRule            securityRiskRule
	DownloadRule          securityRiskRule
	AIWriteRule           securityRiskRule
	CrossModuleRule       securityRiskRule
	LargePageSizeLimit    int
	DeepPageLimit         int
	SequentialIDWindow    time.Duration
	SequentialIDThreshold int
}

var numericPathSegmentPattern = regexp.MustCompile(`/\d+($|/)`)

func SecurityRiskGuard(ctx *bcontext.Context) {
	cfg := currentSecurityRiskConfig()
	if !cfg.Enabled {
		return
	}
	path := ctx.Request.URL.Path
	if !strings.HasPrefix(path, "/api/v1/") || securityRiskSkipPath(path) {
		return
	}
	userID := dataUint(ctx, "user_id")
	tenantID := dataUint(ctx, "tenant_id")
	if userID == 0 {
		return
	}
	decision := evaluateSecurityRisk(ctx, cfg, tenantID, userID)
	if decision.Allowed {
		return
	}
	writeSecurityRiskBlock(ctx, decision)
	logSecurityRiskBlock(ctx, tenantID, userID, decision)
	logSecurityRiskEvent(ctx, tenantID, userID, decision)
	if decision.Severe {
		revokeSecurityRiskSession(ctx, tenantID, userID, decision)
	}
}

func currentSecurityRiskConfig() securityRiskConfig {
	return securityRiskConfig{
		Enabled:  appConfigBool("security_risk_enabled", "SECURITY_RISK_ENABLED", true),
		FailOpen: appConfigBool("security_risk_fail_open", "SECURITY_RISK_FAIL_OPEN", false),
		GeneralReadRule: securityRiskRule{
			Name:      "general_read_rate",
			Window:    appConfigDurationSeconds("security_risk_general_read_window_sec", "SECURITY_RISK_GENERAL_READ_WINDOW_SEC", 60),
			Limit:     appConfigInt("security_risk_general_read_limit", "SECURITY_RISK_GENERAL_READ_LIMIT", 240),
			Operation: securityOperationRateLimit,
		},
		CRMListRule: securityRiskRule{
			Name:      "crm_list_rate",
			Window:    appConfigDurationSeconds("security_risk_crm_list_window_sec", "SECURITY_RISK_CRM_LIST_WINDOW_SEC", 60),
			Limit:     appConfigInt("security_risk_crm_list_limit", "SECURITY_RISK_CRM_LIST_LIMIT", 80),
			Operation: securityOperationScanBlock,
		},
		CRMSearchRule: securityRiskRule{
			Name:      "crm_search_rate",
			Window:    appConfigDurationSeconds("security_risk_crm_search_window_sec", "SECURITY_RISK_CRM_SEARCH_WINDOW_SEC", 60),
			Limit:     appConfigInt("security_risk_crm_search_limit", "SECURITY_RISK_CRM_SEARCH_LIMIT", 40),
			Operation: securityOperationScanBlock,
		},
		CRMDetailRule: securityRiskRule{
			Name:      "crm_detail_rate",
			Window:    appConfigDurationSeconds("security_risk_crm_detail_window_sec", "SECURITY_RISK_CRM_DETAIL_WINDOW_SEC", 60),
			Limit:     appConfigInt("security_risk_crm_detail_limit", "SECURITY_RISK_CRM_DETAIL_LIMIT", 120),
			Operation: securityOperationScanBlock,
		},
		ExportRule: securityRiskRule{
			Name:      "export_rate",
			Window:    appConfigDurationSeconds("security_risk_export_window_sec", "SECURITY_RISK_EXPORT_WINDOW_SEC", 86400),
			Limit:     appConfigInt("security_risk_export_limit", "SECURITY_RISK_EXPORT_LIMIT", 20),
			Operation: securityOperationExportBlock,
			Severe:    true,
		},
		DownloadRule: securityRiskRule{
			Name:      "download_rate",
			Window:    appConfigDurationSeconds("security_risk_download_window_sec", "SECURITY_RISK_DOWNLOAD_WINDOW_SEC", 60),
			Limit:     appConfigInt("security_risk_download_limit", "SECURITY_RISK_DOWNLOAD_LIMIT", 80),
			Operation: securityOperationRateLimit,
		},
		AIWriteRule: securityRiskRule{
			Name:      "ai_write_rate",
			Window:    appConfigDurationSeconds("security_risk_ai_write_window_sec", "SECURITY_RISK_AI_WRITE_WINDOW_SEC", 300),
			Limit:     appConfigInt("security_risk_ai_write_limit", "SECURITY_RISK_AI_WRITE_LIMIT", 12),
			Operation: securityOperationRateLimit,
		},
		CrossModuleRule: securityRiskRule{
			Name:      "cross_module_scan",
			Window:    appConfigDurationSeconds("security_risk_cross_module_window_sec", "SECURITY_RISK_CROSS_MODULE_WINDOW_SEC", 120),
			Limit:     appConfigInt("security_risk_cross_module_limit", "SECURITY_RISK_CROSS_MODULE_LIMIT", 16),
			Operation: securityOperationScanBlock,
		},
		LargePageSizeLimit:    appConfigInt("security_risk_page_size_limit", "SECURITY_RISK_PAGE_SIZE_LIMIT", 100),
		DeepPageLimit:         appConfigInt("security_risk_deep_page_limit", "SECURITY_RISK_DEEP_PAGE_LIMIT", 200),
		SequentialIDWindow:    appConfigDurationSeconds("security_risk_sequential_id_window_sec", "SECURITY_RISK_SEQUENTIAL_ID_WINDOW_SEC", 60),
		SequentialIDThreshold: appConfigInt("security_risk_sequential_id_limit", "SECURITY_RISK_SEQUENTIAL_ID_LIMIT", 60),
	}
}

func evaluateSecurityRisk(ctx *bcontext.Context, cfg securityRiskConfig, tenantID uint, userID uint) securityRiskDecision {
	path := ctx.Request.URL.Path
	method := ctx.Request.Method
	traceID := utils.NewSessionID()[:16]
	if isSuspiciousUserAgent(ctx.Request.UserAgent()) {
		return denySecurityRisk(traceID, securityOperationScanBlock, "异常 User-Agent 或自动化访问特征", "suspicious_user_agent", true, 0)
	}
	if isCRMListPath(path) {
		if size := queryInt(ctx, "size", 20); size > cfg.LargePageSizeLimit {
			return denySecurityRisk(traceID, securityOperationScanBlock, fmt.Sprintf("分页大小超过限制: %d", size), "large_page_size", false, 0)
		}
		if page := queryInt(ctx, "page", 1); page > cfg.DeepPageLimit {
			return denySecurityRisk(traceID, securityOperationScanBlock, fmt.Sprintf("异常深分页访问: page=%d", page), "deep_page_scan", true, 0)
		}
	}
	if redisUnavailable() {
		if isHighRiskPath(method, path) && !cfg.FailOpen {
			return denySecurityRisk(traceID, securityOperationRateLimit, "风控服务不可用，高风险接口已阻断", "redis_unavailable", false, 0)
		}
		return securityRiskDecision{Allowed: true}
	}
	checks := rulesForRequest(cfg, method, path, ctx)
	for _, rule := range checks {
		decision := applySecurityRiskRule(ctx, tenantID, userID, traceID, rule, cfg.FailOpen)
		if !decision.Allowed {
			return decision
		}
	}
	if pathModule := riskModuleByPath(path); pathModule != "" && method == http.MethodGet && isRiskReadPath(path) {
		decision := applyCrossModuleRule(ctx, cfg, tenantID, userID, traceID, pathModule)
		if !decision.Allowed {
			return decision
		}
	}
	if isCRMDetailPath(path) && method == http.MethodGet {
		decision := applySequentialIDRule(ctx, cfg, tenantID, userID, traceID, path)
		if !decision.Allowed {
			return decision
		}
	}
	return securityRiskDecision{Allowed: true}
}

func rulesForRequest(cfg securityRiskConfig, method string, path string, ctx *bcontext.Context) []securityRiskRule {
	rules := []securityRiskRule{}
	switch {
	case isCRMListPath(path):
		rules = append(rules, cfg.CRMListRule)
		if hasSearchQuery(ctx) {
			rules = append(rules, cfg.CRMSearchRule)
		}
	case isCRMDetailPath(path):
		rules = append(rules, cfg.CRMDetailRule)
	case isExportPath(path):
		rules = append(rules, cfg.ExportRule)
	case isDownloadPath(path):
		rules = append(rules, cfg.DownloadRule)
	case isAIWritePath(method, path):
		rules = append(rules, cfg.AIWriteRule)
	}
	return rules
}

func applySecurityRiskRule(ctx *bcontext.Context, tenantID uint, userID uint, traceID string, rule securityRiskRule, failOpen bool) securityRiskDecision {
	if rule.Limit <= 0 || rule.Window <= 0 {
		return securityRiskDecision{Allowed: true}
	}
	subjects := securityRiskSubjects(rule, tenantID, userID, dataString(ctx, "session_id"), ctx.Input.IP())
	for _, subject := range subjects {
		if subject.Key == "" || subject.Limit <= 0 {
			continue
		}
		count, ttl, err := incrementRiskWindow(subject.Key, rule.Window)
		if err != nil {
			utils.LogWarning("安全风控计数失败 key=%s err=%v", subject.Key, err)
			if !failOpen {
				return denySecurityRisk(traceID, securityOperationRateLimit, "风控计数服务不可用，高风险接口已阻断", "risk_counter_unavailable", rule.Severe, rule.Window)
			}
			return securityRiskDecision{Allowed: true}
		}
		if count > int64(subject.Limit) {
			return denySecurityRisk(traceID, rule.Operation, fmt.Sprintf("命中风控规则 %s[%s]: %d/%d", rule.Name, subject.Dimension, count, subject.Limit), rule.Name, rule.Severe, ttl)
		}
	}
	return securityRiskDecision{Allowed: true}
}

func securityRiskSubjects(rule securityRiskRule, tenantID uint, userID uint, sessionID string, ip string) []securityRiskSubject {
	tenantLimit := rule.Limit * 10
	ipLimit := rule.Limit * 5
	if rule.Severe {
		tenantLimit = rule.Limit * 3
		ipLimit = rule.Limit * 2
	}
	return []securityRiskSubject{
		{
			Dimension: "user",
			Key:       securityRiskKey(rule.Name, "user", strconv.FormatUint(uint64(userID), 10)),
			Limit:     rule.Limit,
		},
		{
			Dimension: "session",
			Key:       securityRiskKey(rule.Name, "session", sessionID),
			Limit:     rule.Limit,
		},
		{
			Dimension: "ip",
			Key:       securityRiskKey(rule.Name, "ip", ip),
			Limit:     ipLimit,
		},
		{
			Dimension: "tenant",
			Key:       securityRiskKey(rule.Name, "tenant", strconv.FormatUint(uint64(tenantID), 10)),
			Limit:     tenantLimit,
		},
	}
}

func applyCrossModuleRule(ctx *bcontext.Context, cfg securityRiskConfig, tenantID uint, userID uint, traceID string, module string) securityRiskDecision {
	rule := cfg.CrossModuleRule
	key := securityRiskKey(rule.Name, "user", strconv.FormatUint(uint64(userID), 10))
	member := module
	redisCtx := stdctx.Background()
	client := utils.RedisClient
	if client == nil || rule.Limit <= 0 {
		return securityRiskDecision{Allowed: true}
	}
	zkey := key + ":modules"
	now := time.Now().Unix()
	pipe := client.TxPipeline()
	pipe.ZAdd(redisCtx, zkey, redis.Z{Score: float64(now), Member: member})
	pipe.ZRemRangeByScore(redisCtx, zkey, "0", strconv.FormatInt(now-int64(rule.Window.Seconds()), 10))
	countCmd := pipe.ZCard(redisCtx, zkey)
	pipe.Expire(redisCtx, zkey, rule.Window)
	if _, err := pipe.Exec(redisCtx); err != nil {
		utils.LogWarning("跨模块风控计数失败 tenant=%d user=%d ip=%s err=%v", tenantID, userID, ctx.Input.IP(), err)
		if !cfg.FailOpen {
			return denySecurityRisk(traceID, securityOperationRateLimit, "风控计数服务不可用，高风险接口已阻断", "risk_counter_unavailable", rule.Severe, rule.Window)
		}
		return securityRiskDecision{Allowed: true}
	}
	if countCmd.Val() > int64(rule.Limit) {
		return denySecurityRisk(traceID, rule.Operation, fmt.Sprintf("短时间访问不同模块过多: %d/%d", countCmd.Val(), rule.Limit), rule.Name, rule.Severe, rule.Window)
	}
	return securityRiskDecision{Allowed: true}
}

func applySequentialIDRule(ctx *bcontext.Context, cfg securityRiskConfig, tenantID uint, userID uint, traceID string, path string) securityRiskDecision {
	if cfg.SequentialIDThreshold <= 0 || cfg.SequentialIDWindow <= 0 {
		return securityRiskDecision{Allowed: true}
	}
	id := lastNumericPathSegment(path)
	if id == "" {
		return securityRiskDecision{Allowed: true}
	}
	key := securityRiskKey("sequential_id_scan", "user", strconv.FormatUint(uint64(userID), 10)) + ":" + pathFamily(path)
	redisCtx := stdctx.Background()
	client := utils.RedisClient
	if client == nil {
		return securityRiskDecision{Allowed: true}
	}
	now := time.Now().Unix()
	pipe := client.TxPipeline()
	pipe.ZAdd(redisCtx, key, redis.Z{Score: float64(now), Member: id})
	pipe.ZRemRangeByScore(redisCtx, key, "0", strconv.FormatInt(now-int64(cfg.SequentialIDWindow.Seconds()), 10))
	countCmd := pipe.ZCard(redisCtx, key)
	idsCmd := pipe.ZRange(redisCtx, key, 0, -1)
	pipe.Expire(redisCtx, key, cfg.SequentialIDWindow)
	if _, err := pipe.Exec(redisCtx); err != nil {
		utils.LogWarning("顺序枚举风控计数失败 tenant=%d user=%d ip=%s err=%v", tenantID, userID, ctx.Input.IP(), err)
		if !cfg.FailOpen {
			return denySecurityRisk(traceID, securityOperationRateLimit, "风控计数服务不可用，高风险接口已阻断", "risk_counter_unavailable", false, cfg.SequentialIDWindow)
		}
		return securityRiskDecision{Allowed: true}
	}
	if countCmd.Val() > int64(cfg.SequentialIDThreshold) && hasSequentialIDStreak(idsCmd.Val(), cfg.SequentialIDThreshold) {
		return denySecurityRisk(traceID, securityOperationScanBlock, fmt.Sprintf("疑似顺序枚举对象ID: %d/%d", countCmd.Val(), cfg.SequentialIDThreshold), "sequential_id_scan", false, cfg.SequentialIDWindow)
	}
	return securityRiskDecision{Allowed: true}
}

func denySecurityRisk(traceID string, operation string, reason string, rule string, severe bool, retryAfter time.Duration) securityRiskDecision {
	status := http.StatusTooManyRequests
	if severe {
		status = http.StatusForbidden
	}
	return securityRiskDecision{
		Allowed:    false,
		StatusCode: status,
		Message:    "访问行为触发安全风控，已阻断",
		Operation:  operation,
		Reason:     reason,
		TraceID:    traceID,
		RetryAfter: retryAfter,
		Severe:     severe,
		Rule:       rule,
	}
}

func writeSecurityRiskBlock(ctx *bcontext.Context, decision securityRiskDecision) {
	if decision.RetryAfter > 0 && decision.StatusCode == http.StatusTooManyRequests {
		ctx.Output.Header("Retry-After", strconv.FormatInt(int64(decision.RetryAfter.Seconds()), 10))
	}
	utils.ErrorKeyWithData(ctx, decision.StatusCode, "security.riskBlocked", nil, map[string]interface{}{
		"error_code": "SECURITY_RISK_BLOCKED",
		"reason":     decision.Reason,
		"trace_id":   decision.TraceID,
		"rule":       decision.Rule,
	})
}

func logSecurityRiskBlock(ctx *bcontext.Context, tenantID uint, userID uint, decision securityRiskDecision) {
	utils.LogWarning(
		"安全风控阻断 trace_id=%s status=%d severe=%t rule=%s operation=%s reason=%s tenant=%d user=%d username=%s session=%s ip=%s method=%s path=%s query=%s ua=%s retry_after_sec=%d",
		decision.TraceID,
		decision.StatusCode,
		decision.Severe,
		decision.Rule,
		decision.Operation,
		decision.Reason,
		tenantID,
		userID,
		dataString(ctx, "username"),
		dataString(ctx, "session_id"),
		ctx.Input.IP(),
		ctx.Request.Method,
		ctx.Request.URL.Path,
		ctx.Request.URL.RawQuery,
		ctx.Request.UserAgent(),
		int64(decision.RetryAfter.Seconds()),
	)
}

func logSecurityRiskEvent(ctx *bcontext.Context, tenantID uint, userID uint, decision securityRiskDecision) {
	services.LogOperation(services.OperationLogEvent{
		TenantID:       tenantID,
		UserID:         userID,
		Username:       dataString(ctx, "username"),
		Operation:      decision.Operation,
		Target:         ctx.Request.URL.Path,
		DescriptionKey: "operation." + strings.ToLower(decision.Operation),
		Params: map[string]interface{}{
			"method":     ctx.Request.Method,
			"path":       ctx.Request.URL.Path,
			"query":      ctx.Request.URL.RawQuery,
			"rule":       decision.Rule,
			"trace_id":   decision.TraceID,
			"session_id": dataString(ctx, "session_id"),
		},
		Result:    "failed",
		Reason:    decision.Reason,
		IP:        ctx.Input.IP(),
		UserAgent: ctx.Request.UserAgent(),
		Source:    "security",
	})
}

func revokeSecurityRiskSession(ctx *bcontext.Context, tenantID uint, userID uint, decision securityRiskDecision) {
	sessionID := dataString(ctx, "session_id")
	if sessionID == "" {
		return
	}
	now := time.Now()
	if err := utils.DB.Model(&models.UserSession{}).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Updates(map[string]interface{}{"revoked": true, "revoked_at": &now}).Error; err != nil {
		utils.LogWarning("安全风控撤销会话失败 tenant=%d user=%d session=%s err=%v", tenantID, userID, sessionID, err)
		return
	}
	services.LogOperation(services.OperationLogEvent{
		TenantID:       tenantID,
		UserID:         userID,
		Username:       dataString(ctx, "username"),
		Operation:      securityOperationSessionRevoked,
		Target:         "session[" + sessionID + "]",
		DescriptionKey: "operation.security_session_revoked",
		Params: map[string]interface{}{
			"path":     ctx.Request.URL.Path,
			"rule":     decision.Rule,
			"trace_id": decision.TraceID,
		},
		Result:    "failed",
		Reason:    decision.Reason,
		IP:        ctx.Input.IP(),
		UserAgent: ctx.Request.UserAgent(),
		Source:    "security",
	})
}

func incrementRiskWindow(key string, window time.Duration) (int64, time.Duration, error) {
	client := utils.RedisClient
	if client == nil {
		return 0, 0, fmt.Errorf("redis client is nil")
	}
	redisCtx := stdctx.Background()
	count, err := client.Incr(redisCtx, key).Result()
	if err != nil {
		return 0, 0, err
	}
	if count == 1 {
		if err := client.Expire(redisCtx, key, window).Err(); err != nil {
			return 0, 0, err
		}
		return count, window, nil
	}
	ttl, err := client.TTL(redisCtx, key).Result()
	if err != nil {
		return 0, 0, err
	}
	if ttl <= 0 {
		ttl = window
		_ = client.Expire(redisCtx, key, window).Err()
	}
	return count, ttl, nil
}

func redisUnavailable() bool {
	return utils.RedisClient == nil
}

func securityRiskSkipPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/auth/") ||
		strings.HasPrefix(path, "/api/v1/portal/")
}

func isHighRiskPath(method string, path string) bool {
	return isExportPath(path) || isDownloadPath(path) || isAIWritePath(method, path)
}

func isRiskReadPath(path string) bool {
	return isCRMListPath(path) || isCRMDetailPath(path) || isDownloadPath(path) || isExportPath(path)
}

func isCRMListPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/prd/") && strings.HasSuffix(path, "/objects")
}

func isCRMDetailPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/prd/") && strings.Contains(path, "/objects/") && numericPathSegmentPattern.MatchString(path)
}

func isExportPath(path string) bool {
	return strings.HasSuffix(path, "/objects/export") ||
		strings.HasPrefix(path, "/api/v1/operation-logs/export") ||
		strings.HasPrefix(path, "/api/v1/operation-logs/login/export") ||
		strings.HasSuffix(path, "/backup") ||
		strings.HasSuffix(path, "/export.pdf")
}

func isDownloadPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/files/") && strings.HasSuffix(path, "/download")
}

func isAIWritePath(method string, path string) bool {
	if method == http.MethodGet || method == http.MethodOptions {
		return false
	}
	return strings.Contains(path, "/tenant-ai-analysis/") ||
		strings.Contains(path, "/company-research/") ||
		strings.HasSuffix(path, "/hotspots") ||
		strings.HasSuffix(path, "/risk") ||
		strings.HasSuffix(path, "/score") ||
		strings.HasSuffix(path, "/attribution")
}

func hasSearchQuery(ctx *bcontext.Context) bool {
	keys := []string{"keyword", "phone", "email", "company_name", "contact_name", "source_channel", "priority", "importance", "reliability"}
	for _, key := range keys {
		if strings.TrimSpace(ctx.Input.Query(key)) != "" {
			return true
		}
	}
	return false
}

func isSuspiciousUserAgent(ua string) bool {
	value := strings.ToLower(strings.TrimSpace(ua))
	if value == "" {
		return true
	}
	needles := []string{"python-requests", "curl/", "wget/", "scrapy", "headlesschrome", "phantomjs", "playwright", "selenium", "puppeteer", "httpclient", "okhttp"}
	for _, item := range needles {
		if strings.Contains(value, item) {
			return true
		}
	}
	return false
}

func riskModuleByPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/v1/prd/"):
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/prd/"), "/")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	case strings.HasPrefix(path, "/api/v1/files/"):
		return "files"
	case strings.HasPrefix(path, "/api/v1/operation-logs"):
		return "audit_log"
	case strings.HasPrefix(path, "/api/v1/accounts/"):
		return "accounts"
	}
	return ""
}

func lastNumericPathSegment(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if _, err := strconv.ParseUint(parts[i], 10, 64); err == nil {
			return parts[i]
		}
	}
	return ""
}

func pathFamily(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if _, err := strconv.ParseUint(part, 10, 64); err == nil {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}

func hasSequentialIDStreak(values []string, threshold int) bool {
	if threshold <= 0 {
		return false
	}
	ids := make([]int64, 0, len(values))
	for _, value := range values {
		id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) <= threshold {
		return false
	}
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		seen[id] = struct{}{}
	}
	requiredStreak := threshold / 2
	if requiredStreak < 10 {
		requiredStreak = 10
	}
	for _, id := range ids {
		if _, ok := seen[id-1]; ok {
			continue
		}
		streak := 1
		for next := id + 1; ; next++ {
			if _, ok := seen[next]; !ok {
				break
			}
			streak++
			if streak >= requiredStreak {
				return true
			}
		}
	}
	return false
}

func securityRiskKey(rule string, dimension string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "security:risk:" + strings.TrimSpace(rule) + ":" + strings.TrimSpace(dimension) + ":" + value
}

func queryInt(ctx *bcontext.Context, key string, fallback int) int {
	value := strings.TrimSpace(ctx.Input.Query(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func appConfigInt(key string, envKey string, fallback int) int {
	if env := strings.TrimSpace(os.Getenv(envKey)); env != "" {
		if parsed, err := strconv.Atoi(env); err == nil {
			return parsed
		}
	}
	return web.AppConfig.DefaultInt(key, fallback)
}

func appConfigBool(key string, envKey string, fallback bool) bool {
	if env := strings.TrimSpace(os.Getenv(envKey)); env != "" {
		if parsed, err := strconv.ParseBool(env); err == nil {
			return parsed
		}
	}
	return web.AppConfig.DefaultBool(key, fallback)
}

func appConfigDurationSeconds(key string, envKey string, fallback int) time.Duration {
	value := appConfigInt(key, envKey, fallback)
	if value <= 0 {
		value = fallback
	}
	return time.Duration(value) * time.Second
}
