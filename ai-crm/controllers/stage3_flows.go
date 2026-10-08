package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (c *CRMController) ModuleAccessPreview() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	items := make([]map[string]interface{}, 0, len(crmDomains))
	for _, domain := range crmDomains {
		level := "manage"
		if !auth.IsSuperAdmin() && !auth.IsTenantAdmin() {
			level = crmModuleAccessLevel(auth, domain.Key)
			if level == "" {
				level = "edit"
			}
		}
		canRead := level != "none"
		canEdit := level == "edit" || level == "manage"
		items = append(items, map[string]interface{}{
			"domain":      domain.Key,
			"name":        domain.Name,
			"level":       level,
			"level_label": crmModuleAccessLabel(level),
			"can_read":    canRead,
			"can_edit":    canEdit,
		})
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"user_id": auth.UserID,
		"roles":   roleMatrixKeys(auth),
		"items":   items,
	})
}

func (c *CRMController) PendingApprovalSummary() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	now := time.Now()
	overdueHours := queryInt(c.Ctx, "overdue_hours", 48)
	if overdueHours <= 0 {
		overdueHours = 48
	}
	overdueCutoff := now.Add(-time.Duration(overdueHours) * time.Hour)
	var items []models.ApprovalInstance
	if err := utils.DB.Where("tenant_id = ? AND status = ?", auth.TenantID, "pending").
		Order("updated_at asc").
		Limit(500).
		Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取审批汇总失败")
		return
	}
	byBusiness := map[string]int{}
	overdueItems := make([]map[string]interface{}, 0)
	for _, item := range items {
		key := strings.TrimSpace(item.BusinessType)
		if key == "" {
			key = "unknown"
		}
		byBusiness[key]++
		if item.UpdatedAt.Before(overdueCutoff) {
			overdueItems = append(overdueItems, map[string]interface{}{
				"id":            item.ID,
				"title":         item.Title,
				"business_type": key,
				"business_id":   item.BusinessID,
				"updated_at":    item.UpdatedAt,
			})
		}
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"pending_total":  len(items),
		"overdue_total":  len(overdueItems),
		"overdue_hours":  overdueHours,
		"by_business":    byBusiness,
		"overdue_sample": overdueItems,
	})
}

func (c *CRMController) IntegrationHealth() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var settings []models.CRMSetting
	if err := utils.DB.Where("tenant_id = ? AND scope = ?", auth.TenantID, "integration").
		Order("`key` asc").
		Find(&settings).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取集成健康状态失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(settings))
	for _, setting := range settings {
		value := payloadMap(setting.ValueJSON)
		enabled := boolValue(value["enabled"], false)
		integrationType := strings.TrimSpace(strings.ToLower(firstPayloadText(value, "type")))
		endpoint := strings.TrimSpace(firstPayloadText(value, "endpoint", "issuer", "discovery_url"))
		var lastLog models.OperationLog
		lastResult := "unknown"
		lastCheckAt := ""
		if err := utils.DB.Where("tenant_id = ? AND operation IN ? AND (target = ? OR params_json LIKE ?)",
			auth.TenantID,
			[]string{"TEST_INTEGRATION", "EXECUTE_INTEGRATION", "SYNC_HR"},
			setting.Key,
			"%\"type\":\""+integrationType+"\"%",
		).Order("created_at desc").First(&lastLog).Error; err == nil {
			lastResult = lastLog.Result
			lastCheckAt = lastLog.CreatedAt.Format(time.RFC3339)
		}
		health := "warning"
		if !enabled {
			health = "disabled"
		} else if lastResult == "success" {
			health = "healthy"
		} else if lastResult == "failed" {
			health = "error"
		}
		items = append(items, map[string]interface{}{
			"key":           setting.Key,
			"type":          integrationType,
			"enabled":       enabled,
			"endpoint":      endpoint,
			"last_result":   lastResult,
			"last_check_at": lastCheckAt,
			"health":        health,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.Compare(strings.TrimSpace(items[i]["key"].(string)), strings.TrimSpace(items[j]["key"].(string))) < 0
	})
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": len(items)})
}

func (c *CRMController) NextBestActions() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	limit := queryInt(c.Ctx, "limit", 20)
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	now := time.Now()
	actions := make([]map[string]interface{}, 0)

	var opportunities []models.CRMObject
	_ = crmScopedObjectQuery(auth, "opportunities").
		Where("object_type = ? AND status IN ?", "opportunity", []string{"initial_contact", "need_confirmed", "proposal", "negotiation", "on_hold"}).
		Order("updated_at asc").
		Limit(500).
		Find(&opportunities).Error
	for _, opp := range opportunities {
		payload := payloadMap(opp.PayloadJSON)
		stage := strings.TrimSpace(opp.Status)
		lineItems := interfaceList(payload["line_items"])
		expected := strings.TrimSpace(firstPayloadText(payload, "expected_close_date", "close_date"))
		if stage == "need_confirmed" && len(lineItems) == 0 {
			actions = append(actions, map[string]interface{}{"priority": "high", "domain": "opportunities", "object_id": opp.ID, "title": "补充产品明细", "message": "需求确认阶段缺少产品明细，建议先补齐再推进报价。"})
		}
		if stage == "proposal" && expected == "" {
			actions = append(actions, map[string]interface{}{"priority": "medium", "domain": "opportunities", "object_id": opp.ID, "title": "补充预计成交日期", "message": "方案/报价阶段未填写预计成交日期，建议补充预测信息。"})
		}
	}

	var tickets []models.CRMObject
	_ = crmScopedObjectQuery(auth, "tickets").
		Where("object_type = ? AND status IN ?", "ticket", []string{"new", "assigned", "processing", "waiting_customer", "reopened"}).
		Order("updated_at asc").
		Limit(500).
		Find(&tickets).Error
	for _, ticket := range tickets {
		payload := payloadMap(ticket.PayloadJSON)
		if ticket.OwnerID == 0 {
			actions = append(actions, map[string]interface{}{"priority": "high", "domain": "tickets", "object_id": ticket.ID, "title": "工单待分派", "message": "该工单尚未指定负责人，建议立即分派。"})
		}
		dueAt, _ := parseOptionalTime(strings.TrimSpace(firstPayloadText(payload, "sla_due_at")))
		if dueAt != nil && now.After(*dueAt) {
			actions = append(actions, map[string]interface{}{"priority": "high", "domain": "tickets", "object_id": ticket.ID, "title": "工单已超时", "message": "该工单已超过SLA，建议立即升级或转派。"})
		}
	}

	priorityRank := map[string]int{"high": 3, "medium": 2, "low": 1}
	sort.Slice(actions, func(i, j int) bool {
		return priorityRank[strings.TrimSpace(actions[i]["priority"].(string))] > priorityRank[strings.TrimSpace(actions[j]["priority"].(string))]
	})
	if len(actions) > limit {
		actions = actions[:limit]
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"generated_at": now.Format(time.RFC3339),
		"items":        actions,
		"total":        len(actions),
	})
}
