package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

var activeOpportunityStages = []string{"initial_contact", "need_confirmed", "proposal", "negotiation", "on_hold"}

type opportunityFollowUpRequest struct {
	Content      string `json:"content"`
	NextFollowAt string `json:"next_follow_at"`
	NextAction   string `json:"next_action"`
}

type opportunityReminderRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	RemindAt string `json:"remind_at"`
	OwnerIDs []uint `json:"owner_ids"`
}

func defaultOpportunityFollowUpRules() map[string]int {
	return map[string]int{
		"initial_contact": 1,
		"need_confirmed":  2,
		"proposal":        2,
		"negotiation":     1,
		"on_hold":         7,
	}
}

func opportunityFollowUpRules(auth authContext) map[string]int {
	rules := defaultOpportunityFollowUpRules()
	setting := crmSettingValue(auth.TenantID, "opportunity_follow_up", "rules")
	if len(setting) == 0 {
		return rules
	}
	candidates := []map[string]interface{}{setting}
	if nested, ok := setting["rules"].(map[string]interface{}); ok {
		candidates = append(candidates, nested)
	}
	if nested, ok := setting["stages"].(map[string]interface{}); ok {
		candidates = append(candidates, nested)
	}
	for _, source := range candidates {
		for stage := range rules {
			value, ok := source[stage]
			if !ok {
				continue
			}
			days := int(numericValue(value))
			if days <= 0 {
				if item, ok := value.(map[string]interface{}); ok {
					for _, key := range []string{"days", "max_idle_days", "remind_after_days", "follow_up_days"} {
						if parsed := int(numericValue(item[key])); parsed > 0 {
							days = parsed
							break
						}
					}
				}
			}
			if days > 0 {
				rules[stage] = days
			}
		}
	}
	return rules
}

func opportunityFollowUpStatus(opportunity models.CRMObject, now time.Time, rules map[string]int) string {
	payload := payloadMap(opportunity.PayloadJSON)
	stage := normalizeOpportunityStage(opportunity.Status)
	if stage == "" {
		stage = normalizeOpportunityStage(fmt.Sprint(payload["stage"]))
	}
	if stage == "won" || stage == "lost" {
		return "normal"
	}
	if nextFollowAt, _ := parseOptionalTime(firstPayloadText(payload, "next_follow_at")); nextFollowAt != nil {
		if nextFollowAt.Before(now) {
			return "overdue"
		}
		if nextFollowAt.Before(now.Add(24 * time.Hour)) {
			return "due_soon"
		}
		return "normal"
	}
	reference := opportunity.UpdatedAt
	if stageEnteredAt, _ := parseOptionalTime(firstPayloadText(payload, "stage_entered_at", "stage_changed_at")); stageEnteredAt != nil {
		reference = *stageEnteredAt
	}
	if lastFollowAt, _ := parseOptionalTime(firstPayloadText(payload, "last_follow_up_at", "last_follow_at")); lastFollowAt != nil && lastFollowAt.After(reference) {
		reference = *lastFollowAt
	}
	days := rules[stage]
	if days <= 0 {
		days = 2
	}
	if reference.Add(time.Duration(days) * 24 * time.Hour).Before(now) {
		return "stale"
	}
	return "normal"
}

func applyOpportunityFollowState(opportunity *models.CRMObject, payload map[string]interface{}, now time.Time, rules map[string]int) {
	if opportunity == nil || payload == nil {
		return
	}
	opportunity.PayloadJSON = toJSONString(payload)
	status := opportunityFollowUpStatus(*opportunity, now, rules)
	payload["follow_up_status"] = status
	opportunity.PayloadJSON = toJSONString(payload)
}

func (c *CRMController) DetectOpportunityConflicts() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	companyID := queryUint(c.Ctx, "company_id")
	excludeID := queryUint(c.Ctx, "exclude_id")
	company := strings.TrimSpace(c.Ctx.Input.Query("company"))
	name := strings.TrimSpace(c.Ctx.Input.Query("name"))
	limit := queryInt(c.Ctx, "limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if companyID == 0 && company == "" {
		utils.BadRequest(c.Ctx, "company_id 或 company 不能为空")
		return
	}
	query := crmScopedObjectQuery(auth, "opportunities").
		Where("object_type = ? AND status IN ?", "opportunity", activeOpportunityStages)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if company != "" {
		sub := []string{"name LIKE ?"}
		subArgs := []interface{}{"%" + company + "%"}
		if clause, clauseArgs := payloadJSONKeyLikeClause(company, "company", "company_name"); clause != "" {
			sub = append(sub, clause)
			subArgs = append(subArgs, clauseArgs...)
		}
		query = query.Where("("+strings.Join(sub, " OR ")+")", subArgs...)
	}
	var candidates []models.CRMObject
	if err := query.Order("updated_at desc").Limit(limit).Find(&candidates).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "查询冲突商机失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(candidates))
	needle := normalizeSearchKeyword(name)
	for _, item := range candidates {
		score := 40
		payload := payloadMap(item.PayloadJSON)
		if companyID > 0 && item.CompanyID == companyID {
			score += 40
		}
		itemCompany := strings.TrimSpace(firstPayloadText(payload, "company", "company_name"))
		if company != "" && strings.Contains(strings.ToLower(itemCompany), strings.ToLower(company)) {
			score += 20
		}
		if needle != "" {
			target := normalizeSearchKeyword(item.Name)
			if target != "" && (strings.Contains(target, needle) || strings.Contains(needle, target)) {
				score += 20
			}
			if score < 60 {
				continue
			}
		}
		items = append(items, map[string]interface{}{
			"id":          item.ID,
			"name":        item.Name,
			"status":      item.Status,
			"amount":      item.Amount,
			"currency":    item.Currency,
			"owner_id":    item.OwnerID,
			"company_id":  item.CompanyID,
			"updated_at":  item.UpdatedAt,
			"company":     itemCompany,
			"contact":     firstPayloadText(payload, "contact_name", "contact"),
			"confidence":  score,
			"warning_tip": "同一客户存在进行中商机，请确认是否重复跟进",
		})
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"count":      len(items),
		"items":      items,
		"company_id": companyID,
		"company":    company,
		"name":       name,
		"exclude_id": excludeID,
		"stages":     activeOpportunityStages,
		"checked_at": time.Now().Format(time.RFC3339),
	})
}

func (c *CRMController) WinOpportunity() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的商机ID")
		return
	}
	var opportunity models.CRMObject
	if err := crmScopedObjectQuery(auth, "opportunities").Where("id = ?", id).First(&opportunity).Error; err != nil {
		utils.NotFound(c.Ctx, "商机不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, opportunity, "商机"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if len(c.Ctx.Input.RequestBody) > 0 {
		_ = bindJSON(c.Ctx, &req)
	}
	var deal models.CRMObject
	var linkedTicketID uint
	reused := false
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		existingDeal, hasDeal, err := findExistingDealForOpportunityTx(tx, auth, opportunity.ID)
		if err != nil {
			return err
		}
		if hasDeal {
			deal = existingDeal
			reused = true
		}
		if err := validateOpportunityReadyToWin(opportunity); err != nil {
			return err
		}
		opportunity.Status = "won"
		oppPayload := payloadMap(opportunity.PayloadJSON)
		now := time.Now()
		oppPayload["stage"] = "won"
		oppPayload["probability"] = 100
		oppPayload["stage_changed_at"] = now.Format(time.RFC3339)
		oppPayload["stage_entered_at"] = now.Format(time.RFC3339)
		oppPayload["follow_up_status"] = "normal"
		oppPayload["won_at"] = now.Format(time.RFC3339)
		oppPayload["won_reason"] = strings.TrimSpace(req.Reason)
		linkedTicketID = uintFromValue(oppPayload["source_ticket_id"])
		opportunity.PayloadJSON = toJSONString(oppPayload)
		opportunity.UpdatedBy = auth.UserID
		if err := tx.Save(&opportunity).Error; err != nil {
			return err
		}
		if hasDeal {
			deal = existingDeal
			if err := ensureCRMRelationTx(tx, auth, "opportunities", opportunity.ID, "deals", deal.ID, "won_as"); err != nil {
				return err
			}
		} else {
			createdDeal, _, createErr := ensureClosedDealForOpportunityTx(tx, auth, opportunity, nil, req.Reason)
			if createErr != nil {
				return createErr
			}
			deal = createdDeal
		}
		if linkedTicketID > 0 {
			var ticket models.CRMObject
			if err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "tickets", "ticket", linkedTicketID).First(&ticket).Error; err == nil {
				ticketPayload := payloadMap(ticket.PayloadJSON)
				ticketPayload["upsell_opportunity_id"] = opportunity.ID
				ticketPayload["upsell_deal_id"] = deal.ID
				ticketPayload["upsell_won_at"] = time.Now().Format(time.RFC3339)
				ticket.PayloadJSON = toJSONString(ticketPayload)
				ticket.UpdatedBy = auth.UserID
				if err := tx.Save(&ticket).Error; err != nil {
					return err
				}
				if err := ensureCRMRelationTx(tx, auth, "tickets", ticket.ID, "deals", deal.ID, "upsell_deal"); err != nil {
					return err
				}
				if err := createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_upsell_won", "增购商机赢单并生成成交记录", map[string]interface{}{"reason": strings.TrimSpace(req.Reason), "opportunity_id": opportunity.ID, "deal_id": deal.ID}); err != nil {
					return err
				}
			}
		}
		return createBusinessTimeline(tx, auth, "opportunities", opportunity.ID, "opportunity_win", "商机赢单", map[string]interface{}{"reason": strings.TrimSpace(req.Reason), "deal_id": deal.ID})
	})
	if err != nil {
		if strings.Contains(err.Error(), "必须") {
			utils.BadRequest(c.Ctx, err.Error())
			return
		}
		utils.Error(c.Ctx, http.StatusInternalServerError, "商机赢单失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "WIN_OPPORTUNITY", fmt.Sprintf("商机[%d]", opportunity.ID), "operation.win_opportunity", map[string]interface{}{"opportunity_id": opportunity.ID, "deal_id": deal.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"opportunity": opportunity, "deal": deal, "order": nil, "reused": reused})
}

func (c *CRMController) AdvanceOpportunityStage() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的商机ID")
		return
	}
	var req struct {
		Stage        string                 `json:"stage"`
		Comment      string                 `json:"comment"`
		PayloadPatch map[string]interface{} `json:"payload_patch"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Stage = normalizeOpportunityStage(req.Stage)
	if req.Stage == "" {
		utils.BadRequest(c.Ctx, "目标阶段不能为空")
		return
	}
	var opportunity models.CRMObject
	if err := crmScopedObjectQuery(auth, "opportunities").Where("id = ?", id).First(&opportunity).Error; err != nil {
		utils.NotFound(c.Ctx, "商机不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, opportunity, "商机"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	beforeStage := opportunity.Status
	opportunity.Status = req.Stage
	payload := payloadMap(opportunity.PayloadJSON)
	for key, value := range req.PayloadPatch {
		payload[key] = value
	}
	now := time.Now()
	payload["stage"] = req.Stage
	payload["probability"] = opportunityProbability(req.Stage)
	payload["stage_changed_at"] = now.Format(time.RFC3339)
	if beforeStage != req.Stage || strings.TrimSpace(fmt.Sprint(payload["stage_entered_at"])) == "" {
		payload["stage_entered_at"] = now.Format(time.RFC3339)
	}
	payload["stage_comment"] = req.Comment
	applyOpportunityFollowState(&opportunity, payload, now, opportunityFollowUpRules(auth))
	opportunity.UpdatedBy = auth.UserID
	if err := validateStageGate(auth, "opportunities", opportunity); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	activity := cloneCRMObject(auth, "opportunities", "deal_activity", "阶段推进："+opportunity.Name, opportunity)
	activity.Status = "active"
	activity.PayloadJSON = toJSONString(map[string]interface{}{"activity_type": "stage_change", "from": beforeStage, "to": req.Stage, "summary": req.Comment, "happened_at": now.Format(time.RFC3339)})
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&opportunity).Error; err != nil {
			return err
		}
		if err := tx.Create(&activity).Error; err != nil {
			return err
		}
		if err := tx.Create(&[]models.CRMRelation{
			newCRMRelation(auth, "opportunities", opportunity.ID, "opportunities", activity.ID, "stage_timeline"),
		}).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "opportunities", opportunity.ID, "opportunity_stage", fmt.Sprintf("%s -> %s", beforeStage, req.Stage), map[string]interface{}{"reason": req.Comment, "from": beforeStage, "to": req.Stage})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "商机阶段推进失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "ADVANCE_OPPORTUNITY_STAGE", fmt.Sprintf("商机[%d]", opportunity.ID), "operation.advance_opportunity_stage", map[string]interface{}{"opportunity_id": opportunity.ID, "from": beforeStage, "to": req.Stage, "activity_id": activity.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"opportunity": opportunity, "activity": activity})
}

func (c *CRMController) AddOpportunityFollowUp() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的商机ID")
		return
	}
	if err := ensureCRMModuleAccess(auth, "opportunities", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req opportunityFollowUpRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	req.NextAction = strings.TrimSpace(req.NextAction)
	if req.Content == "" {
		utils.BadRequest(c.Ctx, "跟进内容不能为空")
		return
	}
	var opportunity models.CRMObject
	if err := crmScopedObjectQuery(auth, "opportunities").Where("id = ? AND object_type = ?", id, "opportunity").First(&opportunity).Error; err != nil {
		utils.NotFound(c.Ctx, "商机不存在")
		return
	}
	if !canManageCRMObject(auth, opportunity) && !canFollowCRMObject(auth, opportunity) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权跟进该商机")
		return
	}
	nextFollowAt, err := parseOptionalTime(req.NextFollowAt)
	if err != nil {
		utils.BadRequest(c.Ctx, "下次跟进时间格式无效")
		return
	}
	now := time.Now()
	payload := payloadMap(opportunity.PayloadJSON)
	payload["last_follow_up_at"] = now.Format(time.RFC3339)
	payload["last_follow_up"] = req.Content
	payload["last_follow_by"] = auth.UserID
	if nextFollowAt != nil {
		payload["next_follow_at"] = nextFollowAt.Format(time.RFC3339)
	}
	if req.NextAction != "" {
		payload["next_action"] = req.NextAction
	}
	applyOpportunityFollowState(&opportunity, payload, now, opportunityFollowUpRules(auth))
	opportunity.UpdatedBy = auth.UserID
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&opportunity).Error; err != nil {
			return err
		}
		if nextFollowAt != nil {
			if err := createWorkspaceReminder(tx, auth, opportunity, "商机下次跟进："+opportunity.Name, req.Content, nextFollowAt, nil, "opportunity_follow_reminder", "opportunity_follow"); err != nil {
				return err
			}
		}
		return createBusinessTimeline(tx, auth, "opportunities", opportunity.ID, "opportunity_follow_up", "添加商机跟进", map[string]interface{}{"reason": req.Content, "content": req.Content, "next_follow_at": payload["next_follow_at"], "next_action": req.NextAction})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "添加商机跟进失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "ADD_OPPORTUNITY_FOLLOW_UP", fmt.Sprintf("商机[%d]", opportunity.ID), "operation.add_opportunity_follow_up", map[string]interface{}{"opportunity_id": opportunity.ID, "next_follow_at": payload["next_follow_at"]}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"opportunity": opportunity})
}

func (c *CRMController) CreateOpportunityReminder() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的商机ID")
		return
	}
	if err := ensureCRMModuleAccess(auth, "opportunities", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req opportunityReminderRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var opportunity models.CRMObject
	if err := crmScopedObjectQuery(auth, "opportunities").Where("id = ? AND object_type = ?", id, "opportunity").First(&opportunity).Error; err != nil {
		utils.NotFound(c.Ctx, "商机不存在")
		return
	}
	if !canManageCRMObject(auth, opportunity) && !canFollowCRMObject(auth, opportunity) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权设置该商机提醒")
		return
	}
	remindAt, err := parseOptionalTime(req.RemindAt)
	if err != nil {
		utils.BadRequest(c.Ctx, "提醒时间格式无效")
		return
	}
	title := defaultText(req.Title, "商机提醒："+opportunity.Name)
	content := defaultText(req.Content, "请及时跟进商机。")
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := createWorkspaceReminder(tx, auth, opportunity, title, content, remindAt, req.OwnerIDs, "opportunity_manual_reminder", "opportunity_reminder"); err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "opportunities", opportunity.ID, "opportunity_reminder", "创建商机提醒", map[string]interface{}{"reason": content, "title": title, "remind_at": req.RemindAt, "owner_ids": req.OwnerIDs})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建商机提醒失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_OPPORTUNITY_REMINDER", fmt.Sprintf("商机[%d]", opportunity.ID), "operation.create_opportunity_reminder", map[string]interface{}{"opportunity_id": opportunity.ID, "remind_at": req.RemindAt, "owner_ids": req.OwnerIDs}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"created": true})
}

func (c *CRMController) CreateQuoteFromOpportunity() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的商机ID")
		return
	}
	var req struct {
		QuoteAmount float64                  `json:"quote_amount"`
		ValidUntil  string                   `json:"valid_until"`
		Reason      string                   `json:"reason"`
		LineItems   []map[string]interface{} `json:"line_items"`
	}
	if len(c.Ctx.Input.RequestBody) > 0 {
		_ = bindJSON(c.Ctx, &req)
	}
	var opportunity models.CRMObject
	if err := crmScopedObjectQuery(auth, "opportunities").Where("id = ? AND object_type = ?", id, "opportunity").First(&opportunity).Error; err != nil {
		utils.NotFound(c.Ctx, "商机不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, opportunity, "商机"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	oppPayload := payloadMap(opportunity.PayloadJSON)
	lineItems := mapsToInterfaces(req.LineItems)
	if len(lineItems) == 0 {
		lineItems = interfaceList(oppPayload["line_items"])
	}
	if len(lineItems) == 0 {
		lineItems = fallbackOpportunityQuoteLineItems(opportunity, oppPayload, req.QuoteAmount)
	}
	if len(lineItems) == 0 {
		utils.BadRequest(c.Ctx, "生成报价单前，请先在商机里补产品或服务明细")
		return
	}
	quoteAmount := req.QuoteAmount
	if quoteAmount <= 0 {
		quoteAmount = opportunity.Amount
	}
	if quoteAmount <= 0 {
		quoteAmount = sumLineItems(lineItems)
	}
	quotePayload := mergePayload(oppPayload, inheritAttributionPayload(oppPayload))
	quotePayload["source_opportunity_id"] = opportunity.ID
	quotePayload["quote_status"] = "draft"
	quotePayload["quote_no"] = fmt.Sprintf("Q-%d", time.Now().UnixNano()%1000000000)
	quotePayload["quote_amount"] = quoteAmount
	quotePayload["line_items"] = lineItems
	quotePayload["valid_until"] = strings.TrimSpace(req.ValidUntil)
	quotePayload["created_reason"] = strings.TrimSpace(req.Reason)
	quote := cloneCRMObject(auth, "opportunities", "quote", opportunity.Name+" 报价单", opportunity)
	quote.Code = fmt.Sprint(quotePayload["quote_no"])
	quote.Status = "draft"
	quote.Amount = quoteAmount
	quote.PayloadJSON = toJSONString(quotePayload)
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&quote).Error; err != nil {
			return err
		}
		if err := tx.Create(&[]models.CRMRelation{
			newCRMRelation(auth, "opportunities", opportunity.ID, "opportunities", quote.ID, "quote"),
		}).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "opportunities", opportunity.ID, "quote_created", "生成报价单", map[string]interface{}{"reason": strings.TrimSpace(req.Reason), "quote_id": quote.ID, "quote_no": quote.Code, "quote_amount": quoteAmount})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "生成报价单失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_QUOTE_FROM_OPPORTUNITY", fmt.Sprintf("商机[%d]", opportunity.ID), "operation.create_quote_from_opportunity", map[string]interface{}{"opportunity_id": opportunity.ID, "quote_id": quote.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"opportunity": opportunity, "quote": quote})
}

func fallbackOpportunityQuoteLineItems(opportunity models.CRMObject, payload map[string]interface{}, quoteAmount float64) []interface{} {
	productName := firstPayloadText(payload, "product_name", "intent_product", "product", "sku")
	productID := uintFromValue(firstNonNil(payload["product_id"], payload["intent_product_id"]))
	if productName == "" {
		for _, raw := range interfaceList(firstNonNil(payload["product_names"], payload["intent_products"])) {
			if text := strings.TrimSpace(fmt.Sprint(raw)); text != "" {
				productName = text
				break
			}
		}
	}
	if productID == 0 {
		for _, raw := range interfaceList(firstNonNil(payload["product_ids"], payload["intent_product_ids"])) {
			if id := uintFromValue(raw); id > 0 {
				productID = id
				break
			}
		}
	}
	amount := quoteAmount
	if amount <= 0 {
		amount = opportunity.Amount
	}
	if amount <= 0 {
		amount = numericValue(firstNonNil(payload["quote_amount"], payload["expected_amount"], payload["amount"]))
	}
	if productName == "" && productID == 0 && amount <= 0 {
		return []interface{}{}
	}
	if productName == "" {
		productName = "未命名产品"
	}
	line := map[string]interface{}{
		"product_id": productID,
		"name":       productName,
		"quantity":   1,
		"unit_price": amount,
		"discount":   0,
		"tax_rate":   0,
		"source":     "opportunity_payload",
	}
	return []interface{}{line}
}

func (c *CRMController) SendQuote() {
	c.changeQuoteStatus("sent")
}

func (c *CRMController) AcceptQuote() {
	c.changeQuoteStatus("accepted")
}

func (c *CRMController) changeQuoteStatus(status string) {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的报价单ID")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if len(c.Ctx.Input.RequestBody) > 0 {
		_ = bindJSON(c.Ctx, &req)
	}
	if strings.TrimSpace(req.Reason) == "" {
		utils.BadRequest(c.Ctx, "请填写处理内容")
		return
	}
	var quote models.CRMObject
	if err := utils.DB.Where("tenant_id = ? AND domain = ? AND id = ? AND object_type = ?", auth.TenantID, "opportunities", id, "quote").First(&quote).Error; err != nil {
		utils.NotFound(c.Ctx, "报价单不存在")
		return
	}
	opportunity, err := quoteOpportunity(auth, quote.ID)
	if err != nil {
		utils.BadRequest(c.Ctx, "报价单没有关联商机")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, opportunity, "商机"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	now := time.Now().Format(time.RFC3339)
	summary := "发送报价单"
	eventType := "quote_sent"
	if status == "accepted" {
		summary = "客户接受报价"
		eventType = "quote_accepted"
	}
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		quote.Status = status
		quotePayload := payloadMap(quote.PayloadJSON)
		quotePayload["quote_status"] = status
		quotePayload[status+"_at"] = now
		quotePayload[status+"_reason"] = strings.TrimSpace(req.Reason)
		quote.PayloadJSON = toJSONString(quotePayload)
		quote.UpdatedBy = auth.UserID
		if err := tx.Save(&quote).Error; err != nil {
			return err
		}
		oppPayload := payloadMap(opportunity.PayloadJSON)
		oppPayload["quote_status"] = status
		if status == "accepted" {
			oppPayload["quote_status"] = "accepted"
		}
		oppPayload["quote_id"] = quote.ID
		oppPayload["quote_no"] = quote.Code
		oppPayload["quote_amount"] = quote.Amount
		oppPayload[status+"_quote_at"] = now
		opportunity.PayloadJSON = toJSONString(oppPayload)
		opportunity.UpdatedBy = auth.UserID
		if err := tx.Save(&opportunity).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "opportunities", opportunity.ID, eventType, summary, map[string]interface{}{"reason": strings.TrimSpace(req.Reason), "quote_id": quote.ID, "quote_no": quote.Code, "quote_status": status})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "报价单处理失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, strings.ToUpper(eventType), fmt.Sprintf("报价单[%d]", quote.ID), "operation.change_quote_status", map[string]interface{}{"quote_id": quote.ID, "opportunity_id": opportunity.ID, "status": status}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"quote": quote, "opportunity": opportunity})
}

func (c *CRMController) CreateOrderFromQuote() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的报价单ID")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if len(c.Ctx.Input.RequestBody) > 0 {
		_ = bindJSON(c.Ctx, &req)
	}
	var quote models.CRMObject
	if err := utils.DB.Where("tenant_id = ? AND domain = ? AND id = ? AND object_type = ?", auth.TenantID, "opportunities", id, "quote").First(&quote).Error; err != nil {
		utils.NotFound(c.Ctx, "报价单不存在")
		return
	}
	opportunity, err := quoteOpportunity(auth, quote.ID)
	if err != nil {
		utils.BadRequest(c.Ctx, "报价单没有关联商机")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, opportunity, "商机"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	if quote.Status != "accepted" {
		utils.BadRequest(c.Ctx, "客户接受报价后，才能从报价单生成订单")
		return
	}
	var deal, order models.CRMObject
	reused := false
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		deal, _, err = ensureClosedDealForOpportunityTx(tx, auth, opportunity, &quote, req.Reason)
		if err != nil {
			return err
		}
		if existingOrder, hasOrder, err := findExistingOrderForQuoteTx(tx, auth, quote.ID); err != nil {
			return err
		} else if hasOrder {
			order = existingOrder
			reused = true
			order.DealID = deal.ID
			payload := payloadMap(order.PayloadJSON)
			payload["source_deal_id"] = deal.ID
			payload["source_quote_id"] = quote.ID
			payload["source_quote_no"] = quote.Code
			payload["source_opportunity_id"] = opportunity.ID
			order.PayloadJSON = toJSONString(payload)
			if err := normalizeCustomerChainObjectTx(tx, auth, &order, deal, quote, opportunity); err != nil {
				return err
			}
			order.UpdatedBy = auth.UserID
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
			if err := ensureCRMRelationTx(tx, auth, "deals", deal.ID, "orders", order.ID, "generated_order"); err != nil {
				return err
			}
			if err := ensureCRMRelationTx(tx, auth, "opportunities", quote.ID, "orders", order.ID, "generated_order"); err != nil {
				return err
			}
			if err := ensureCustomerChainRelationsTx(tx, auth, order); err != nil {
				return err
			}
			return ensureCRMRelationTx(tx, auth, "opportunities", opportunity.ID, "orders", order.ID, "quote_generated_order")
		}
		if existingOrder, hasOrder, err := findExistingOrderForDealTx(tx, auth, deal.ID); err != nil {
			return err
		} else if hasOrder {
			order = existingOrder
			reused = true
			order.DealID = deal.ID
			payload := payloadMap(order.PayloadJSON)
			payload["source_deal_id"] = deal.ID
			payload["source_quote_id"] = quote.ID
			payload["source_quote_no"] = quote.Code
			payload["source_opportunity_id"] = opportunity.ID
			order.PayloadJSON = toJSONString(payload)
			if err := normalizeCustomerChainObjectTx(tx, auth, &order, deal, quote, opportunity); err != nil {
				return err
			}
			order.UpdatedBy = auth.UserID
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
			if err := ensureCRMRelationTx(tx, auth, "opportunities", quote.ID, "orders", order.ID, "generated_order"); err != nil {
				return err
			}
			if err := ensureCustomerChainRelationsTx(tx, auth, order); err != nil {
				return err
			}
			return ensureCRMRelationTx(tx, auth, "opportunities", opportunity.ID, "orders", order.ID, "quote_generated_order")
		}
		order, err = createDraftOrderForDealTx(tx, auth, deal, quote, map[string]interface{}{"source_quote_id": quote.ID, "source_quote_no": quote.Code, "source_opportunity_id": opportunity.ID})
		if err != nil {
			return err
		}
		if err := ensureCRMRelationTx(tx, auth, "opportunities", quote.ID, "orders", order.ID, "generated_order"); err != nil {
			return err
		}
		if err := ensureCRMRelationTx(tx, auth, "opportunities", opportunity.ID, "orders", order.ID, "quote_generated_order"); err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "opportunities", opportunity.ID, "quote_order", "报价生成订单草稿", map[string]interface{}{"reason": strings.TrimSpace(req.Reason), "quote_id": quote.ID, "quote_no": quote.Code, "order_id": order.ID})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "报价生成订单失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_ORDER_FROM_QUOTE", fmt.Sprintf("报价单[%d]", quote.ID), "operation.create_order_from_quote", map[string]interface{}{"quote_id": quote.ID, "opportunity_id": opportunity.ID, "order_id": order.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"quote": quote, "opportunity": opportunity, "deal": deal, "order": order, "reused": reused})
}

func (c *CRMController) CreateOpportunityFromTicket() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var req struct {
		Name   string  `json:"name"`
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	payload := payloadMap(ticket.PayloadJSON)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "服务增购：" + ticket.Name
	}
	opportunity := cloneCRMObject(auth, "opportunities", "opportunity", name, ticket)
	opportunity.Status = "initial_contact"
	opportunity.Amount = req.Amount
	now := time.Now()
	opportunity.PayloadJSON = toJSONString(mergePayload(payload, mergePayload(inheritAttributionPayload(payload), map[string]interface{}{"stage": "initial_contact", "probability": 10, "stage_entered_at": now.Format(time.RFC3339), "follow_up_status": "normal", "growth_source": "service_upsell", "source_ticket_id": ticket.ID, "service_upsell_reason": strings.TrimSpace(req.Reason)})))
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&opportunity).Error; err != nil {
			return err
		}
		if err := tx.Create(&[]models.CRMRelation{
			newCRMRelation(auth, "tickets", ticket.ID, "opportunities", opportunity.ID, "service_upsell"),
		}).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_upsell", "工单创建增购商机", map[string]interface{}{"reason": req.Reason, "opportunity_id": opportunity.ID})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "工单创建增购商机失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_OPPORTUNITY_FROM_TICKET", fmt.Sprintf("工单[%d]", ticket.ID), "operation.create_opportunity_from_ticket", map[string]interface{}{"ticket_id": ticket.ID, "opportunity_id": opportunity.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ticket": ticket, "opportunity": opportunity})
}

func (c *CRMController) GenerateOpportunityRisk() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的商机ID")
		return
	}
	var opportunity models.CRMObject
	if err := crmScopedObjectQuery(auth, "opportunities").Where("id = ?", id).First(&opportunity).Error; err != nil {
		utils.NotFound(c.Ctx, "商机不存在")
		return
	}
	payload := payloadMap(opportunity.PayloadJSON)
	reason := "阶段推进停滞或缺少近期互动"
	if value, ok := payload["risk_reason"].(string); ok && strings.TrimSpace(value) != "" {
		reason = value
	}
	riskPayload := map[string]interface{}{
		"opportunity_id": opportunity.ID,
		"reason":         reason,
		"suggestion":     "建议安排一次客户会议并更新下一步行动",
		"risk_score":     70,
	}
	riskJSON, _ := json.Marshal(riskPayload)
	risk := cloneCRMObject(auth, "insights", "deal_risk", "交易风险："+opportunity.Name, opportunity)
	risk.Status = "watching"
	risk.Score = 70
	risk.PayloadJSON = string(riskJSON)
	task := cloneCRMObject(auth, "workspace", "task", "处理交易风险："+opportunity.Name, opportunity)
	task.Status = "open"
	task.PayloadJSON = toJSONString(map[string]interface{}{"source": "deal_risk", "opportunity_id": opportunity.ID, "risk_score": 70})
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&risk).Error; err != nil {
			return err
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		relations := []models.CRMRelation{
			newCRMRelation(auth, "opportunities", opportunity.ID, "insights", risk.ID, "risk_alert"),
			newCRMRelation(auth, "insights", risk.ID, "workspace", task.ID, "created_task"),
		}
		return tx.Create(&relations).Error
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "生成交易风险失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "GENERATE_OPPORTUNITY_RISK", fmt.Sprintf("商机[%d]", opportunity.ID), "operation.generate_opportunity_risk", map[string]interface{}{"opportunity_id": opportunity.ID, "risk_id": risk.ID, "task_id": task.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"opportunity": opportunity, "risk": risk, "task": task})
}

var opportunityFollowUpSchedulerStop chan struct{}

func StartOpportunityFollowUpScheduler() {
	if opportunityFollowUpSchedulerStop != nil {
		return
	}
	opportunityFollowUpSchedulerStop = make(chan struct{})
	ticker := time.NewTicker(6 * time.Hour)
	go func() {
		services.WithSchedulerLock("opportunity_follow_up_scheduler", runOpportunityFollowUpSchedule)
		for {
			select {
			case <-ticker.C:
				services.WithSchedulerLock("opportunity_follow_up_scheduler", runOpportunityFollowUpSchedule)
			case <-opportunityFollowUpSchedulerStop:
				ticker.Stop()
				return
			}
		}
	}()
}

func runOpportunityFollowUpSchedule() {
	var tenants []models.Tenant
	if err := utils.DB.Where("status = ?", "active").Find(&tenants).Error; err != nil {
		return
	}
	for _, tenant := range tenants {
		auth := authContext{TenantID: tenant.ID, UserID: tenantAdminUserID(tenant.ID), AccountType: "tenant_admin", Username: "system"}
		_ = runOpportunityFollowUpTenant(auth)
	}
}

func runOpportunityFollowUpTenant(auth authContext) error {
	now := time.Now()
	rules := opportunityFollowUpRules(auth)
	var opportunities []models.CRMObject
	if err := utils.DB.Where("tenant_id = ? AND domain = ? AND object_type = ? AND status NOT IN ?", auth.TenantID, "opportunities", "opportunity", []string{"won", "lost"}).Limit(1000).Find(&opportunities).Error; err != nil {
		return err
	}
	today := now.Format("2006-01-02")
	return utils.DB.Transaction(func(tx *gorm.DB) error {
		for _, opportunity := range opportunities {
			payload := payloadMap(opportunity.PayloadJSON)
			status := opportunityFollowUpStatus(opportunity, now, rules)
			changed := strings.TrimSpace(fmt.Sprint(payload["follow_up_status"])) != status
			payload["follow_up_status"] = status
			ownerID := opportunity.OwnerID
			if ownerID == 0 {
				ownerID = opportunity.CreatedBy
			}
			needsNotice := (status == "overdue" || status == "stale") && ownerID > 0 && strings.TrimSpace(fmt.Sprint(payload["opportunity_follow_notice_date"])) != today
			if !changed && !needsNotice {
				continue
			}
			if needsNotice {
				payload["opportunity_follow_notice_date"] = today
			}
			opportunity.PayloadJSON = toJSONString(payload)
			opportunity.UpdatedBy = auth.UserID
			if err := tx.Save(&opportunity).Error; err != nil {
				return err
			}
			if needsNotice {
				title := "商机跟进逾期：" + opportunity.Name
				content := "当前商机在“" + opportunityStageName(opportunity.Status) + "”阶段需要跟进，请更新客户反馈和下一步动作。"
				titleKey := "notification.opportunity.followOverdue.title"
				contentKey := "notification.opportunity.followOverdue.content"
				if status == "stale" {
					title = "商机阶段停留过久：" + opportunity.Name
					content = "当前商机在“" + opportunityStageName(opportunity.Status) + "”阶段停留过久，请确认是否推进、搁置或关闭。"
					titleKey = "notification.opportunity.stageStale.title"
					contentKey = "notification.opportunity.stageStale.content"
				}
				noticePayload := map[string]interface{}{
					"title_i18n_key":    titleKey,
					"title_i18n_params": map[string]interface{}{"opportunity": opportunity.Name},
					"i18n_key":          contentKey,
					"i18n_params":       map[string]interface{}{"stage": opportunityStageName(opportunity.Status), "opportunity": opportunity.Name},
					"fallback_title":    title,
					"fallback_content":  content,
				}
				if err := createWorkspaceReminderWithPayload(tx, auth, opportunity, title, content, nil, []uint{ownerID}, "opportunity_follow_overdue_notice", "opportunity_follow", noticePayload); err != nil {
					return err
				}
				if err := createBusinessTimeline(tx, auth, "opportunities", opportunity.ID, "opportunity_follow_overdue_notice", title, map[string]interface{}{"reason": content, "follow_up_status": status}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func opportunityStageName(stage string) string {
	switch normalizeOpportunityStage(stage) {
	case "initial_contact":
		return "初步接触"
	case "need_confirmed":
		return "需求确认"
	case "proposal":
		return "方案/报价"
	case "negotiation":
		return "谈判/评审"
	case "on_hold":
		return "搁置"
	case "won":
		return "赢单"
	case "lost":
		return "输单"
	default:
		return defaultText(stage, "未分阶段")
	}
}
