package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"sort"
	"strings"
	"time"
)

func (c *CRMController) AnalyticsSummary() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	year := queryYear(c.Ctx.Input.Query("year"))
	response := map[string]interface{}{
		"funnel":          salesFunnel(auth),
		"channel_roi":     channelROI(auth),
		"forecast":        salesForecast(auth),
		"loss_analysis":   lossAnalysis(auth),
		"service_quality": serviceQuality(auth),
		"customer_health": customerHealth(auth),
		"alerts":          businessAlerts(auth),
	}
	for key, value := range salesAnalyticsDimensions(auth, year) {
		response[key] = value
	}
	utils.Success(c.Ctx, response)
}

func channelROI(auth authContext) []map[string]interface{} {
	var campaigns []models.CRMObject
	_ = crmScopedObjectQuery(auth, "campaigns").Where("object_type = ?", "campaign").Limit(500).Find(&campaigns).Error
	items := make([]map[string]interface{}, 0, len(campaigns))
	for _, campaign := range campaigns {
		payload := payloadMap(campaign.PayloadJSON)
		channel := firstPayloadText(payload, "source_channel", "utm_source", "type")
		if channel == "" {
			channel = "未标注"
		}
		cost := numericValue(payload["budget"])
		if cost == 0 {
			cost = numericValue(payload["cost"])
		}
		revenue := campaign.Amount
		if revenue == 0 {
			revenue = numericValue(payload["estimated_revenue"])
		}
		roi := 0.0
		if cost > 0 {
			roi = (revenue - cost) / cost
		}
		items = append(items, map[string]interface{}{"campaign_id": campaign.ID, "name": campaign.Name, "channel": channel, "cost": cost, "revenue": revenue, "roi": roi})
	}
	sort.Slice(items, func(i, j int) bool { return numericValue(items[i]["revenue"]) > numericValue(items[j]["revenue"]) })
	if len(items) > 10 {
		return items[:10]
	}
	return items
}

func salesForecast(auth authContext) []map[string]interface{} {
	var opportunities []models.CRMObject
	_ = crmScopedObjectQuery(auth, "opportunities").Where("object_type = ? AND status NOT IN ?", "opportunity", []string{"won", "lost"}).Limit(1000).Find(&opportunities).Error
	buckets := map[string]float64{}
	for _, opp := range opportunities {
		payload := payloadMap(opp.PayloadJSON)
		month := strings.TrimSpace(firstPayloadText(payload, "expected_close_date", "close_date"))
		if len(month) >= 7 {
			month = month[:7]
		} else {
			month = "未填写"
		}
		probability := numericValue(payload["probability"])
		if probability == 0 {
			probability = float64(opportunityProbability(opp.Status))
		}
		buckets[month] += opp.Amount * probability / 100
	}
	return sortedAmountBuckets(buckets, "month")
}

func lossAnalysis(auth authContext) []map[string]interface{} {
	var opportunities []models.CRMObject
	_ = crmScopedObjectQuery(auth, "opportunities").Where("object_type = ? AND status = ?", "opportunity", "lost").Limit(1000).Find(&opportunities).Error
	buckets := map[string]float64{}
	for _, opp := range opportunities {
		payload := payloadMap(opp.PayloadJSON)
		reason := firstPayloadText(payload, "loss_reason", "lost_reason", "stage_comment")
		if reason == "" {
			reason = "未填写"
		}
		buckets[reason]++
	}
	return sortedAmountBuckets(buckets, "reason")
}

func serviceQuality(auth authContext) map[string]interface{} {
	var tickets []models.CRMObject
	_ = crmScopedObjectQuery(auth, "tickets").Where("object_type = ?", "ticket").Limit(1000).Find(&tickets).Error
	total := len(tickets)
	closed := 0
	overdue := 0
	var csatTotal float64
	csatCount := 0
	statuses := map[string]float64{}
	for _, ticket := range tickets {
		statuses[ticket.Status]++
		payload := payloadMap(ticket.PayloadJSON)
		if ticket.Status == "closed" || ticket.Status == "resolved" {
			closed++
		}
		dueAt, _ := parseOptionalTime(firstPayloadText(payload, "sla_due_at"))
		if dueAt != nil && ticket.Status != "closed" && ticket.Status != "resolved" && dueAt.Before(time.Now()) {
			overdue++
		}
		if csat := numericValue(payload["csat"]); csat > 0 {
			csatTotal += csat
			csatCount++
		}
	}
	avgCSAT := 0.0
	if csatCount > 0 {
		avgCSAT = csatTotal / float64(csatCount)
	}
	return map[string]interface{}{"total": total, "closed": closed, "overdue": overdue, "avg_csat": avgCSAT, "status_buckets": sortedAmountBuckets(statuses, "status")}
}

func customerHealth(auth authContext) []map[string]interface{} {
	var companies []models.CRMObject
	_ = crmScopedObjectQuery(auth, "contacts").Where("object_type = ?", "company").Order("updated_at desc").Limit(100).Find(&companies).Error

	if len(companies) == 0 {
		return []map[string]interface{}{}
	}

	companyIDs := make([]uint, 0, len(companies))
	for _, company := range companies {
		companyIDs = append(companyIDs, company.ID)
	}

	type orderAggRow struct {
		CompanyID uint
		Orders    int64
		Amount    float64
	}
	orderAggMap := map[uint]orderAggRow{}
	var orderRows []orderAggRow
	_ = crmScopedObjectQuery(auth, "orders").
		Select("company_id, COUNT(*) AS orders, COALESCE(SUM(amount), 0) AS amount").
		Where("company_id IN ?", companyIDs).
		Group("company_id").
		Find(&orderRows).Error
	for _, row := range orderRows {
		orderAggMap[row.CompanyID] = row
	}

	type ticketAggRow struct {
		CompanyID uint
		Tickets   int64
	}
	ticketAggMap := map[uint]ticketAggRow{}
	var ticketRows []ticketAggRow
	_ = crmScopedObjectQuery(auth, "tickets").
		Select("company_id, COUNT(*) AS tickets").
		Where("company_id IN ?", companyIDs).
		Group("company_id").
		Find(&ticketRows).Error
	for _, row := range ticketRows {
		ticketAggMap[row.CompanyID] = row
	}

	items := make([]map[string]interface{}, 0, len(companies))
	for _, company := range companies {
		orderAgg := orderAggMap[company.ID]
		ticketAgg := ticketAggMap[company.ID]
		orders := orderAgg.Orders
		amount := orderAgg.Amount
		tickets := ticketAgg.Tickets
		score := 60 + int(amount/10000) - int(tickets)*8
		if orders > 0 {
			score += 10
		}
		if score > 100 {
			score = 100
		}
		if score < 0 {
			score = 0
		}
		items = append(items, map[string]interface{}{"company_id": company.ID, "name": company.Name, "orders": orders, "tickets": tickets, "amount": amount, "health_score": score})
	}
	sort.Slice(items, func(i, j int) bool {
		return numericValue(items[i]["health_score"]) < numericValue(items[j]["health_score"])
	})
	if len(items) > 10 {
		return items[:10]
	}
	return items
}

func businessAlerts(auth authContext) []map[string]interface{} {
	alerts := make([]map[string]interface{}, 0)

	// 1) 回款异常：已完成订单且应收余额>0
	var completedOrders []models.CRMObject
	_ = crmScopedObjectQuery(auth, "orders").
		Where("object_type = ? AND status = ?", "order", "completed").
		Order("updated_at desc").
		Limit(500).
		Find(&completedOrders).Error
	receivableRisk := 0
	for _, order := range completedOrders {
		payload := payloadMap(order.PayloadJSON)
		balance := numericValue(payload["receivable_balance"])
		if balance <= 0 {
			continue
		}
		receivableRisk++
		if len(alerts) < 8 {
			alerts = append(alerts, map[string]interface{}{
				"type":      "receivable_overdue",
				"level":     "high",
				"title":     "回款异常",
				"message":   "订单已完成但应收未清，请及时跟进回款。",
				"domain":    "orders",
				"object_id": order.ID,
				"name":      order.Name,
				"value":     balance,
			})
		}
	}

	// 2) 丢单异常：最近30天输单率高
	var recentWon int64
	var recentLost int64
	windowStart := time.Now().Add(-30 * 24 * time.Hour)
	_ = crmScopedObjectQuery(auth, "opportunities").
		Where("object_type = ? AND status = ? AND updated_at >= ?", "opportunity", "won", windowStart).
		Count(&recentWon).Error
	_ = crmScopedObjectQuery(auth, "opportunities").
		Where("object_type = ? AND status = ? AND updated_at >= ?", "opportunity", "lost", windowStart).
		Count(&recentLost).Error
	totalClosed := recentWon + recentLost
	if totalClosed >= 5 {
		lossRate := float64(recentLost) / float64(totalClosed)
		if lossRate >= 0.6 {
			alerts = append(alerts, map[string]interface{}{
				"type":    "loss_spike",
				"level":   "high",
				"title":   "丢单异常",
				"message": "近30天输单率偏高，请排查价格策略和竞品压力。",
				"domain":  "opportunities",
				"value":   lossRate,
			})
		}
	}

	// 3) 服务异常：超时工单多
	var openTickets []models.CRMObject
	_ = crmScopedObjectQuery(auth, "tickets").
		Where("object_type = ? AND status NOT IN ?", "ticket", []string{"closed", "resolved"}).
		Order("updated_at desc").
		Limit(1000).
		Find(&openTickets).Error
	overdueTickets := int64(0)
	now := time.Now()
	for _, ticket := range openTickets {
		payload := payloadMap(ticket.PayloadJSON)
		dueAt, _ := parseOptionalTime(firstPayloadText(payload, "sla_due_at"))
		if dueAt != nil && now.After(*dueAt) {
			overdueTickets++
		}
	}
	if overdueTickets >= 5 {
		alerts = append(alerts, map[string]interface{}{
			"type":    "sla_overdue",
			"level":   "medium",
			"title":   "服务SLA异常",
			"message": "超时工单较多，请优先清理客服队列。",
			"domain":  "tickets",
			"value":   overdueTickets,
		})
	}

	// 4) 渠道质量异常：高拒绝线索偏多
	var invalidLeads int64
	var totalLeads int64
	_ = crmScopedObjectQuery(auth, "leads").Where("object_type = ? AND status = ? AND updated_at >= ?", "lead", "invalid", windowStart).Count(&invalidLeads).Error
	_ = crmScopedObjectQuery(auth, "leads").Where("object_type = ? AND updated_at >= ?", "lead", windowStart).Count(&totalLeads).Error
	if totalLeads >= 10 {
		invalidRate := float64(invalidLeads) / float64(totalLeads)
		if invalidRate >= 0.45 {
			alerts = append(alerts, map[string]interface{}{
				"type":    "channel_quality",
				"level":   "medium",
				"title":   "渠道质量异常",
				"message": "近期无效线索占比偏高，建议复盘渠道投放质量。",
				"domain":  "leads",
				"value":   invalidRate,
			})
		}
	}

	// 摘要
	alerts = append([]map[string]interface{}{{
		"type":    "summary",
		"level":   "info",
		"title":   "异常预警汇总",
		"message": "用于聚焦当前最需要处理的经营风险。",
		"domain":  "dashboards",
		"value": map[string]interface{}{
			"receivable_risk_orders": receivableRisk,
			"total_alerts":           len(alerts),
		},
	}}, alerts...)
	return alerts
}

func sortedAmountBuckets(buckets map[string]float64, labelKey string) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(buckets))
	for key, value := range buckets {
		items = append(items, map[string]interface{}{labelKey: key, "amount": value, "count": value})
	}
	sort.Slice(items, func(i, j int) bool { return numericValue(items[i]["amount"]) > numericValue(items[j]["amount"]) })
	return items
}
