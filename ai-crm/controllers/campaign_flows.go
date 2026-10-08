package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

type campaignFollowUpRequest struct {
	Type         string `json:"type"`
	Content      string `json:"content"`
	NextAction   string `json:"next_action"`
	NextFollowAt string `json:"next_follow_at"`
}

func (c *CRMController) GenerateCampaignAttribution() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "campaigns", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的活动ID")
		return
	}
	var campaign models.CRMObject
	if err := crmScopedObjectQuery(auth, "campaigns").Where("id = ?", id).First(&campaign).Error; err != nil {
		utils.NotFound(c.Ctx, "活动不存在")
		return
	}
	leads := campaignRelatedLeads(auth, campaign)
	touches, wonAmount, attributionSummary := campaignMultiTouchAttribution(auth, campaign, leads)
	payload := map[string]interface{}{
		"campaign_id":       campaign.ID,
		"campaign_name":     campaign.Name,
		"generated_at":      time.Now().Format(time.RFC3339),
		"lead_count":        len(leads),
		"estimated_revenue": wonAmount,
		"model":             attributionSummary["model"],
		"touches":           touches,
		"summary":           attributionSummary,
	}
	attribution := cloneCRMObject(auth, "campaigns", "attribution", "活动归因："+campaign.Name, campaign)
	attribution.Status = "active"
	attribution.Amount = wonAmount
	attribution.PayloadJSON = toJSONString(payload)
	insight := cloneCRMObject(auth, "insights", "touch_attribution", "多触点归因："+campaign.Name, campaign)
	insight.Status = "open"
	insight.Amount = wonAmount
	insight.PayloadJSON = toJSONString(payload)
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&attribution).Error; err != nil {
			return err
		}
		if err := tx.Create(&insight).Error; err != nil {
			return err
		}
		return tx.Create(&[]models.CRMRelation{
			newCRMRelation(auth, "campaigns", campaign.ID, "campaigns", attribution.ID, "generated_attribution"),
			newCRMRelation(auth, "campaigns", campaign.ID, "insights", insight.ID, "touch_attribution"),
		}).Error
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "生成活动归因失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "GENERATE_CAMPAIGN_ATTRIBUTION", fmt.Sprintf("活动[%d]", campaign.ID), "operation.generate_campaign_attribution", map[string]interface{}{"campaign_id": campaign.ID, "lead_count": len(leads), "amount": wonAmount}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"campaign": campaign, "attribution": attribution, "insight": insight})
}

func (c *CRMController) CampaignPerformance() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "campaigns", "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的活动ID")
		return
	}
	var campaign models.CRMObject
	if err := crmScopedObjectQuery(auth, "campaigns").Where("id = ?", id).First(&campaign).Error; err != nil {
		utils.NotFound(c.Ctx, "活动不存在")
		return
	}
	performance := buildCampaignPerformance(auth, campaign)
	utils.Success(c.Ctx, performance)
}

func (c *CRMController) AddCampaignFollowUp() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "campaigns", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的活动ID")
		return
	}
	var campaign models.CRMObject
	if err := crmScopedObjectQuery(auth, "campaigns").Where("id = ? AND object_type = ?", id, "campaign").First(&campaign).Error; err != nil {
		utils.NotFound(c.Ctx, "活动不存在")
		return
	}
	if !canFollowCRMObject(auth, campaign) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权记录该活动动态")
		return
	}
	var req campaignFollowUpRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		utils.BadRequest(c.Ctx, "动态内容不能为空")
		return
	}
	nextFollowAt, err := parseOptionalTime(strings.TrimSpace(req.NextFollowAt))
	if err != nil {
		utils.BadRequest(c.Ctx, "下次提醒时间格式不正确")
		return
	}
	followType := normalizeCampaignFollowUpType(req.Type)
	now := time.Now().Format(time.RFC3339)
	payload := payloadMap(campaign.PayloadJSON)
	payload["last_follow_up"] = content
	payload["last_follow_up_at"] = now
	payload["last_follow_up_type"] = followType
	nextAction := strings.TrimSpace(req.NextAction)
	if nextAction != "" {
		payload["next_action"] = nextAction
	}
	if nextFollowAt != nil {
		payload["next_follow_at"] = nextFollowAt.Format(time.RFC3339)
	}
	campaign.PayloadJSON = toJSONString(payload)
	campaign.UpdatedBy = auth.UserID
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&campaign).Error; err != nil {
			return err
		}
		eventPayload := map[string]interface{}{
			"interaction_type": followType,
			"content":          content,
			"reason":           content,
			"next_action":      nextAction,
		}
		if nextFollowAt != nil {
			eventPayload["next_follow_at"] = nextFollowAt.Format(time.RFC3339)
		}
		if err := createBusinessTimeline(tx, auth, "campaigns", campaign.ID, "campaign_follow_up", "活动动态："+campaign.Name, eventPayload); err != nil {
			return err
		}
		if nextFollowAt != nil {
			title := "活动下次跟进：" + campaign.Name
			reminderContent := defaultText(nextAction, content)
			if err := createWorkspaceReminder(tx, auth, campaign, title, reminderContent, nextFollowAt, nil, "campaign_follow_reminder", "campaign_follow"); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "ADD_CAMPAIGN_FOLLOW_UP", fmt.Sprintf("活动[%d]", campaign.ID), "operation.add_campaign_follow_up", map[string]interface{}{"campaign_id": campaign.ID}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存活动动态失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "ADD_CAMPAIGN_FOLLOW_UP", fmt.Sprintf("活动[%d]", campaign.ID), "operation.add_campaign_follow_up", map[string]interface{}{"campaign_id": campaign.ID, "next_follow_at": payload["next_follow_at"]}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"campaign": campaign})
}

func buildCampaignPerformance(auth authContext, campaign models.CRMObject) map[string]interface{} {
	leads := campaignRelatedLeads(auth, campaign)
	opportunities := campaignRelatedObjects(auth, campaign, leads, "opportunities")
	deals := campaignRelatedObjects(auth, campaign, opportunities, "deals")
	orders := campaignRelatedObjects(auth, campaign, deals, "orders")
	orders = mergeCRMObjects(orders, campaignRelatedObjects(auth, campaign, opportunities, "orders"))

	campaignPayload := payloadMap(campaign.PayloadJSON)
	cost := campaignCost(campaign, campaignPayload)
	dealAmount := sumObjectAmount(deals)
	orderAmount := sumObjectAmount(orders)
	paidAmount := 0.0
	receivableAmount := 0.0
	for _, order := range orders {
		payload := payloadMap(order.PayloadJSON)
		paidAmount += numericValue(payload["paid_amount"])
		receivableAmount += currentReceivable(order, payload)
	}
	revenue := dealAmount
	if orderAmount > revenue {
		revenue = orderAmount
	}
	roi := 0.0
	if cost > 0 {
		roi = (revenue - cost) / cost
	}
	cpl := 0.0
	if cost > 0 && len(leads) > 0 {
		cpl = cost / float64(len(leads))
	}
	cac := 0.0
	if cost > 0 && len(deals) > 0 {
		cac = cost / float64(len(deals))
	}
	return map[string]interface{}{
		"campaign":      campaign,
		"leads":         leads,
		"opportunities": opportunities,
		"deals":         deals,
		"orders":        orders,
		"timeline":      campaignTimelineObjects(auth, campaign.ID),
		"files":         campaignFileObjects(auth, campaign.ID),
		"summary": map[string]interface{}{
			"lead_count":        len(leads),
			"opportunity_count": len(opportunities),
			"deal_count":        len(deals),
			"order_count":       len(orders),
			"campaign_cost":     cost,
			"deal_amount":       dealAmount,
			"order_amount":      orderAmount,
			"revenue_amount":    revenue,
			"paid_amount":       paidAmount,
			"receivable_amount": receivableAmount,
			"roi":               roi,
			"cpl":               cpl,
			"cac":               cac,
		},
		"chain": []map[string]interface{}{
			{"key": "campaign", "label": "活动", "count": 1, "amount": cost},
			{"key": "leads", "label": "线索", "count": len(leads), "amount": sumObjectAmount(leads)},
			{"key": "opportunities", "label": "商机", "count": len(opportunities), "amount": sumObjectAmount(opportunities)},
			{"key": "deals", "label": "成交记录", "count": len(deals), "amount": dealAmount},
			{"key": "orders", "label": "订单", "count": len(orders), "amount": orderAmount},
			{"key": "payments", "label": "回款", "count": len(orders), "amount": paidAmount},
		},
	}
}

func normalizeCampaignFollowUpType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	allowed := map[string]bool{"execution": true, "call": true, "email": true, "meeting": true, "material_update": true, "review": true, "note": true}
	if !allowed[value] {
		return "note"
	}
	return value
}

func campaignTimelineObjects(auth authContext, campaignID uint) []models.CRMObject {
	var items []models.CRMObject
	_ = utils.DB.Model(&models.CRMObject{}).
		Joins("JOIN crm_relations rel ON rel.deleted_at IS NULL AND rel.tenant_id = crm_objects.tenant_id AND rel.target_type = crm_objects.domain AND rel.target_id = crm_objects.id").
		Where("crm_objects.tenant_id = ? AND crm_objects.domain = ? AND crm_objects.object_type = ?", auth.TenantID, "contacts", "interaction").
		Where("rel.source_type = ? AND rel.source_id = ? AND rel.relation_type = ?", "campaigns", campaignID, "timeline").
		Order("crm_objects.created_at desc").
		Limit(100).
		Find(&items).Error
	return items
}

func campaignRelatedLeads(auth authContext, campaign models.CRMObject) []models.CRMObject {
	leads := relatedObjects(auth, "campaigns", campaign.ID, "leads", "generated_lead")
	if clause, args := payloadJSONKeyNumberEqualsClause(campaign.ID, "campaign_id"); clause != "" {
		var matched []models.CRMObject
		_ = crmScopedObjectQuery(auth, "leads").Where(clause, args...).Limit(500).Find(&matched).Error
		leads = mergeCRMObjects(leads, matched)
	}
	return leads
}

func campaignRelatedObjects(auth authContext, campaign models.CRMObject, upstream []models.CRMObject, targetDomain string) []models.CRMObject {
	relations := map[string][]string{
		"opportunities": {"converted_to", "generated_opportunity"},
		"deals":         {"won_as"},
		"orders":        {"generated_order", "quote_generated_order"},
	}
	objects := []models.CRMObject{}
	for _, source := range upstream {
		for _, relationType := range relations[targetDomain] {
			objects = mergeCRMObjects(objects, relatedObjects(auth, source.Domain, source.ID, targetDomain, relationType))
		}
	}
	if clause, args := payloadJSONKeyNumberEqualsClause(campaign.ID, "campaign_id"); clause != "" {
		var matched []models.CRMObject
		_ = crmScopedObjectQuery(auth, targetDomain).Where(clause, args...).Limit(500).Find(&matched).Error
		objects = mergeCRMObjects(objects, matched)
	}
	return objects
}

func mergeCRMObjects(base []models.CRMObject, extra []models.CRMObject) []models.CRMObject {
	seen := map[uint]bool{}
	next := make([]models.CRMObject, 0, len(base)+len(extra))
	for _, item := range append(base, extra...) {
		if item.ID == 0 || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		next = append(next, item)
	}
	return next
}

func sumObjectAmount(items []models.CRMObject) float64 {
	total := 0.0
	for _, item := range items {
		total += item.Amount
	}
	return total
}

func campaignCost(campaign models.CRMObject, payload map[string]interface{}) float64 {
	cost := numericValue(payload["budget"])
	if cost <= 0 {
		cost = numericValue(payload["cost"])
	}
	if cost <= 0 {
		cost = campaign.Amount
	}
	return cost
}
