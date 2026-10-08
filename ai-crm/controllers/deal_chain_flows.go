package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

var dealChainRelationTypes = []string{
	"quote",
	"won_as",
	"accepted_as_deal",
	"generated_order",
	"quote_generated_order",
	"generated_invoice",
	"generated_ticket",
	"return_ticket",
	"service_upsell",
	"upsell_order",
	"upsell_deal",
}

func (c *CRMController) GetDealChain() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domain := normalizeDealChainDomain(c.Ctx.Input.Query("object_type"))
	objectID := queryUint(c.Ctx, "object_id")
	if domain == "" || objectID == 0 {
		utils.BadRequest(c.Ctx, "object_type 和 object_id 不能为空")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var object models.CRMObject
	if err := crmScopedObjectQuery(auth, domain).Where("id = ?", objectID).First(&object).Error; err != nil {
		utils.NotFound(c.Ctx, "交易链路对象不存在")
		return
	}
	objects, relations := collectDealChainObjects(auth, object)
	nodes := buildDealChainNodes(objects)
	virtualNodes := buildDealChainVirtualNodes(objects)
	nodes = append(nodes, virtualNodes...)
	sort.SliceStable(nodes, func(i, j int) bool {
		left := intValue(nodes[i]["step_order"])
		right := intValue(nodes[j]["step_order"])
		if left == right {
			return strings.TrimSpace(fmt.Sprint(nodes[i]["created_at"])) < strings.TrimSpace(fmt.Sprint(nodes[j]["created_at"]))
		}
		return left < right
	})
	utils.Success(c.Ctx, map[string]interface{}{
		"object_type": domain,
		"object_id":   objectID,
		"nodes":       nodes,
		"relations":   buildDealChainRelationItems(relations),
		"summary":     buildDealChainSummary(objects),
	})
}

func normalizeDealChainDomain(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "lead", "leads":
		return "leads"
	case "opportunity", "quote", "quotes", "opportunities":
		return "opportunities"
	case "deal", "closed_deal", "deals":
		return "deals"
	case "order", "invoice", "orders":
		return "orders"
	case "ticket", "tickets":
		return "tickets"
	default:
		if validCRMDomain(strings.TrimSpace(value)) {
			return strings.TrimSpace(value)
		}
		return ""
	}
}

func collectDealChainObjects(auth authContext, root models.CRMObject) (map[string]models.CRMObject, []models.CRMRelation) {
	objects := map[string]models.CRMObject{}
	relations := []models.CRMRelation{}
	seenRelations := map[string]bool{}
	addObject := func(item models.CRMObject) bool {
		key := crmObjectKey(item.Domain, item.ID)
		if item.ID == 0 || key == "" {
			return false
		}
		if _, exists := objects[key]; exists {
			return false
		}
		objects[key] = item
		return true
	}
	addObject(root)
	frontier := []models.CRMObject{root}
	for depth := 0; depth < 4 && len(frontier) > 0; depth++ {
		nextFrontier := []models.CRMObject{}
		for _, current := range frontier {
			for _, linked := range dealChainPayloadObjects(auth, current) {
				if addObject(linked) {
					nextFrontier = append(nextFrontier, linked)
				}
			}
			var rels []models.CRMRelation
			_ = utils.DB.Where(
				"tenant_id = ? AND relation_type IN ? AND ((source_type = ? AND source_id = ?) OR (target_type = ? AND target_id = ?))",
				auth.TenantID,
				dealChainRelationTypes,
				current.Domain,
				current.ID,
				current.Domain,
				current.ID,
			).Find(&rels).Error
			for _, relation := range rels {
				relationKey := fmt.Sprintf("%s:%d:%s:%s:%d", relation.SourceType, relation.SourceID, relation.RelationType, relation.TargetType, relation.TargetID)
				if !seenRelations[relationKey] {
					relations = append(relations, relation)
					seenRelations[relationKey] = true
				}
				if source, ok := visibleCRMObject(auth, relation.SourceType, relation.SourceID); ok {
					if addObject(source) {
						nextFrontier = append(nextFrontier, source)
					}
				}
				if target, ok := visibleCRMObject(auth, relation.TargetType, relation.TargetID); ok {
					if addObject(target) {
						nextFrontier = append(nextFrontier, target)
					}
				}
			}
			if current.Domain == "deals" {
				var orders []models.CRMObject
				_ = crmScopedObjectQuery(auth, "orders").Where("deal_id = ?", current.ID).Limit(20).Find(&orders).Error
				for _, order := range orders {
					if addObject(order) {
						nextFrontier = append(nextFrontier, order)
					}
				}
			}
			if current.Domain == "orders" && current.DealID > 0 {
				if deal, ok := visibleCRMObject(auth, "deals", current.DealID); ok {
					if addObject(deal) {
						nextFrontier = append(nextFrontier, deal)
					}
				}
			}
		}
		frontier = nextFrontier
	}
	return objects, relations
}

func dealChainPayloadObjects(auth authContext, object models.CRMObject) []models.CRMObject {
	payload := payloadMap(object.PayloadJSON)
	candidates := []struct {
		domain string
		id     uint
	}{
		{domain: "leads", id: uintFromValue(payload["source_lead_id"])},
		{domain: "opportunities", id: uintFromValue(payload["source_opportunity_id"])},
		{domain: "opportunities", id: uintFromValue(payload["quote_id"])},
		{domain: "opportunities", id: uintFromValue(payload["source_quote_id"])},
		{domain: "deals", id: uintFromValue(payload["source_deal_id"])},
		{domain: "deals", id: uintFromValue(payload["deal_id"])},
		{domain: "orders", id: uintFromValue(payload["order_id"])},
		{domain: "orders", id: uintFromValue(payload["source_order_id"])},
		{domain: "tickets", id: uintFromValue(payload["source_ticket_id"])},
		{domain: "tickets", id: uintFromValue(payload["ticket_id"])},
	}
	if object.Domain == "orders" && object.DealID > 0 {
		candidates = append(candidates, struct {
			domain string
			id     uint
		}{domain: "deals", id: object.DealID})
	}
	items := []models.CRMObject{}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate.id == 0 {
			continue
		}
		key := crmObjectKey(candidate.domain, candidate.id)
		if seen[key] {
			continue
		}
		if object, ok := visibleCRMObject(auth, candidate.domain, candidate.id); ok {
			items = append(items, object)
			seen[key] = true
		}
	}
	return items
}

func visibleCRMObject(auth authContext, domain string, id uint) (models.CRMObject, bool) {
	if id == 0 || !validCRMDomain(domain) {
		return models.CRMObject{}, false
	}
	var object models.CRMObject
	if err := crmScopedObjectQuery(auth, domain).Where("id = ?", id).First(&object).Error; err != nil {
		return models.CRMObject{}, false
	}
	return object, true
}

func buildDealChainNodes(objects map[string]models.CRMObject) []map[string]interface{} {
	nodes := make([]map[string]interface{}, 0, len(objects))
	for _, object := range objects {
		payload := payloadMap(object.PayloadJSON)
		step, stepOrder := dealChainStep(object)
		nodes = append(nodes, map[string]interface{}{
			"key":          crmObjectKey(object.Domain, object.ID),
			"domain":       object.Domain,
			"object_type":  object.ObjectType,
			"id":           object.ID,
			"name":         object.Name,
			"code":         object.Code,
			"status":       object.Status,
			"step":         step,
			"step_order":   stepOrder,
			"amount":       object.Amount,
			"currency":     object.Currency,
			"owner_id":     object.OwnerID,
			"created_by":   object.CreatedBy,
			"company_id":   object.CompanyID,
			"contact_id":   object.ContactID,
			"deal_id":      object.DealID,
			"created_at":   object.CreatedAt,
			"updated_at":   object.UpdatedAt,
			"payment":      firstPayloadText(payload, "payment_status"),
			"receivable":   numericValue(payload["receivable_balance"]),
			"source_label": firstPayloadText(payload, "source_channel", "growth_source", "utm_source"),
		})
	}
	return nodes
}

func buildDealChainVirtualNodes(objects map[string]models.CRMObject) []map[string]interface{} {
	nodes := []map[string]interface{}{}
	for _, object := range objects {
		if object.Domain != "orders" {
			continue
		}
		payload := payloadMap(object.PayloadJSON)
		for index, payment := range normalizeOrderPayments(payload["payments"]) {
			nodes = append(nodes, map[string]interface{}{
				"key":         fmt.Sprintf("payment:%d:%d", object.ID, index),
				"domain":      "payments",
				"object_type": "payment",
				"id":          fmt.Sprintf("%d-%d", object.ID, index),
				"name":        defaultText(strings.TrimSpace(fmt.Sprint(payment["payment_no"])), "回款记录"),
				"status":      "paid",
				"step":        "payment",
				"step_order":  60,
				"amount":      numericValue(payment["paid_amount"]),
				"currency":    object.Currency,
				"created_at":  defaultText(strings.TrimSpace(fmt.Sprint(payment["paid_at"])), object.CreatedAt.Format(time.RFC3339)),
				"created_by":  object.CreatedBy,
				"source_id":   object.ID,
			})
		}
		if shipmentNo := strings.TrimSpace(fmt.Sprint(payload["shipment_no"])); shipmentNo != "" && shipmentNo != "<nil>" {
			nodes = append(nodes, map[string]interface{}{
				"key":         fmt.Sprintf("shipment:%d", object.ID),
				"domain":      "shipments",
				"object_type": "shipment",
				"id":          fmt.Sprintf("%d-shipment", object.ID),
				"name":        "发货：" + shipmentNo,
				"status":      object.Status,
				"step":        "shipment",
				"step_order":  70,
				"amount":      0,
				"currency":    object.Currency,
				"created_at":  defaultText(strings.TrimSpace(fmt.Sprint(payload["shipped_at"])), object.UpdatedAt.Format(time.RFC3339)),
				"created_by":  object.CreatedBy,
				"source_id":   object.ID,
			})
		}
		if invoiceNo := strings.TrimSpace(fmt.Sprint(payload["invoice_no"])); invoiceNo != "" && invoiceNo != "<nil>" {
			nodes = append(nodes, map[string]interface{}{
				"key":         fmt.Sprintf("invoice:%d", object.ID),
				"domain":      "invoices",
				"object_type": "invoice",
				"id":          fmt.Sprintf("%d-invoice", object.ID),
				"name":        "发票：" + invoiceNo,
				"status":      "active",
				"step":        "invoice",
				"step_order":  80,
				"amount":      object.Amount,
				"currency":    object.Currency,
				"created_at":  defaultText(strings.TrimSpace(fmt.Sprint(payload["invoiced_at"])), object.UpdatedAt.Format(time.RFC3339)),
				"created_by":  object.CreatedBy,
				"source_id":   object.ID,
			})
		}
	}
	return nodes
}

func buildDealChainRelationItems(relations []models.CRMRelation) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(relations))
	for _, relation := range relations {
		items = append(items, map[string]interface{}{
			"source_key":    crmObjectKey(relation.SourceType, relation.SourceID),
			"source_type":   relation.SourceType,
			"source_id":     relation.SourceID,
			"target_key":    crmObjectKey(relation.TargetType, relation.TargetID),
			"target_type":   relation.TargetType,
			"target_id":     relation.TargetID,
			"relation_type": relation.RelationType,
			"created_at":    relation.CreatedAt,
		})
	}
	return items
}

func buildDealChainSummary(objects map[string]models.CRMObject) map[string]interface{} {
	summary := map[string]interface{}{
		"active_pipeline_amount": 0.0,
		"closed_revenue":         0.0,
		"order_amount":           0.0,
		"open_receivable":        0.0,
		"active_opportunities":   0,
		"won_deals":              0,
		"open_orders":            0,
	}
	for _, object := range objects {
		payload := payloadMap(object.PayloadJSON)
		switch object.Domain {
		case "opportunities":
			if object.ObjectType == "opportunity" && containsString(activeOpportunityStages, object.Status) {
				summary["active_pipeline_amount"] = numericValue(summary["active_pipeline_amount"]) + object.Amount
				summary["active_opportunities"] = intValue(summary["active_opportunities"]) + 1
			}
		case "deals":
			if object.Status == "closed_won" || object.ObjectType == "closed_deal" || object.ObjectType == "deal" {
				summary["closed_revenue"] = numericValue(summary["closed_revenue"]) + object.Amount
				summary["won_deals"] = intValue(summary["won_deals"]) + 1
			}
		case "orders":
			summary["order_amount"] = numericValue(summary["order_amount"]) + object.Amount
			if object.Status != "completed" && object.Status != "canceled" {
				summary["open_orders"] = intValue(summary["open_orders"]) + 1
			}
			summary["open_receivable"] = numericValue(summary["open_receivable"]) + currentReceivable(object, payload)
		}
	}
	return summary
}

func dealChainStep(object models.CRMObject) (string, int) {
	switch object.Domain {
	case "leads":
		return "lead", 10
	case "opportunities":
		if object.ObjectType == "quote" {
			return "quote", 30
		}
		return "opportunity", 20
	case "deals":
		return "deal", 40
	case "orders":
		if object.ObjectType == "invoice" {
			return "invoice", 80
		}
		return "order", 50
	case "tickets":
		return "ticket", 90
	default:
		return object.Domain, 100
	}
}

func crmObjectKey(domain string, id uint) string {
	if domain == "" || id == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", domain, id)
}

func intValue(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	}
	return 0
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func ensureCRMRelationTx(tx *gorm.DB, auth authContext, sourceType string, sourceID uint, targetType string, targetID uint, relationType string) error {
	relation := newCRMRelation(auth, sourceType, sourceID, targetType, targetID, relationType)
	return tx.Where(
		"tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?",
		auth.TenantID,
		sourceType,
		sourceID,
		targetType,
		targetID,
		relationType,
	).FirstOrCreate(&relation).Error
}

func firstRelatedCRMObjectTx(tx *gorm.DB, auth authContext, sourceType string, sourceID uint, targetType string, relationTypes []string) (models.CRMObject, bool, error) {
	var relation models.CRMRelation
	query := tx.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ?", auth.TenantID, sourceType, sourceID, targetType)
	if len(relationTypes) > 0 {
		query = query.Where("relation_type IN ?", relationTypes)
	}
	if err := query.Order("created_at asc").First(&relation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return models.CRMObject{}, false, nil
		}
		return models.CRMObject{}, false, err
	}
	var object models.CRMObject
	if err := tx.Where("tenant_id = ? AND domain = ? AND id = ?", auth.TenantID, targetType, relation.TargetID).First(&object).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return models.CRMObject{}, false, nil
		}
		return models.CRMObject{}, false, err
	}
	return object, true, nil
}

func findExistingDealForOpportunityTx(tx *gorm.DB, auth authContext, opportunityID uint) (models.CRMObject, bool, error) {
	if deal, ok, err := firstRelatedCRMObjectTx(tx, auth, "opportunities", opportunityID, "deals", []string{"won_as"}); ok || err != nil {
		return deal, ok, err
	}
	clause, args := payloadJSONKeyNumberLikeClause(opportunityID, "source_opportunity_id")
	if clause == "" {
		return models.CRMObject{}, false, nil
	}
	var deal models.CRMObject
	err := tx.Where("tenant_id = ? AND domain = ? AND object_type IN ?", auth.TenantID, "deals", []string{"closed_deal", "deal"}).
		Where(clause, args...).
		Order("created_at asc").
		First(&deal).Error
	if err == gorm.ErrRecordNotFound {
		return models.CRMObject{}, false, nil
	}
	return deal, err == nil, err
}

func findExistingOrderForDealTx(tx *gorm.DB, auth authContext, dealID uint) (models.CRMObject, bool, error) {
	if order, ok, err := firstRelatedCRMObjectTx(tx, auth, "deals", dealID, "orders", []string{"generated_order"}); ok || err != nil {
		return order, ok, err
	}
	var order models.CRMObject
	err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND deal_id = ?", auth.TenantID, "orders", "order", dealID).
		Order("created_at asc").
		First(&order).Error
	if err == nil {
		return order, true, nil
	}
	if err != gorm.ErrRecordNotFound {
		return models.CRMObject{}, false, err
	}
	clause, args := payloadJSONKeyNumberLikeClause(dealID, "source_deal_id", "deal_id")
	if clause == "" {
		return models.CRMObject{}, false, nil
	}
	err = tx.Where("tenant_id = ? AND domain = ? AND object_type = ?", auth.TenantID, "orders", "order").
		Where(clause, args...).
		Order("created_at asc").
		First(&order).Error
	if err == gorm.ErrRecordNotFound {
		return models.CRMObject{}, false, nil
	}
	return order, err == nil, err
}

func findExistingOrderForQuoteTx(tx *gorm.DB, auth authContext, quoteID uint) (models.CRMObject, bool, error) {
	if order, ok, err := firstRelatedCRMObjectTx(tx, auth, "opportunities", quoteID, "orders", []string{"generated_order"}); ok || err != nil {
		return order, ok, err
	}
	clause, args := payloadJSONKeyNumberLikeClause(quoteID, "source_quote_id", "quote_id")
	if clause == "" {
		return models.CRMObject{}, false, nil
	}
	var order models.CRMObject
	err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ?", auth.TenantID, "orders", "order").
		Where(clause, args...).
		Order("created_at asc").
		First(&order).Error
	if err == gorm.ErrRecordNotFound {
		return models.CRMObject{}, false, nil
	}
	return order, err == nil, err
}

func ensureClosedDealForOpportunityTx(tx *gorm.DB, auth authContext, opportunity models.CRMObject, quote *models.CRMObject, reason string) (models.CRMObject, bool, error) {
	deal, existing, err := findExistingDealForOpportunityTx(tx, auth, opportunity.ID)
	if err != nil {
		return models.CRMObject{}, false, err
	}
	if existing {
		if err := normalizeCustomerChainObjectTx(tx, auth, &deal, opportunity); err != nil {
			return models.CRMObject{}, false, err
		}
		if deal.ObjectType != "closed_deal" {
			deal.ObjectType = "closed_deal"
		}
		deal.UpdatedBy = auth.UserID
		if err := tx.Save(&deal).Error; err != nil {
			return models.CRMObject{}, false, err
		}
		if err := ensureCRMRelationTx(tx, auth, "opportunities", opportunity.ID, "deals", deal.ID, "won_as"); err != nil {
			return models.CRMObject{}, false, err
		}
		if err := ensureCustomerChainRelationsTx(tx, auth, deal); err != nil {
			return models.CRMObject{}, false, err
		}
		return deal, true, nil
	}
	source := opportunity
	if quote != nil {
		source = *quote
	}
	deal = cloneCRMObject(auth, "deals", "closed_deal", opportunity.Name+" 成交", source)
	deal.Status = "closed_won"
	deal.CompanyID = opportunity.CompanyID
	deal.ContactID = opportunity.ContactID
	deal.DealID = 0
	payload := mergePayload(payloadMap(opportunity.PayloadJSON), payloadMap(source.PayloadJSON))
	payload = mergePayload(payload, inheritAttributionPayload(payload))
	payload["source_opportunity_id"] = opportunity.ID
	payload["closed_date"] = time.Now().Format("2006-01-02")
	payload["closed_at"] = time.Now().Format(time.RFC3339)
	payload["won_reason"] = strings.TrimSpace(reason)
	if quote != nil {
		payload["source_quote_id"] = quote.ID
		payload["source_quote_no"] = quote.Code
		deal.Amount = quote.Amount
		deal.Currency = quote.Currency
	}
	deal.PayloadJSON = toJSONString(payload)
	if quote != nil {
		if err := normalizeCustomerChainObjectTx(tx, auth, &deal, opportunity, *quote); err != nil {
			return models.CRMObject{}, false, err
		}
	} else if err := normalizeCustomerChainObjectTx(tx, auth, &deal, opportunity); err != nil {
		return models.CRMObject{}, false, err
	}
	if err := tx.Create(&deal).Error; err != nil {
		return models.CRMObject{}, false, err
	}
	if err := ensureCRMRelationTx(tx, auth, "opportunities", opportunity.ID, "deals", deal.ID, "won_as"); err != nil {
		return models.CRMObject{}, false, err
	}
	if quote != nil {
		if err := ensureCRMRelationTx(tx, auth, "opportunities", quote.ID, "deals", deal.ID, "accepted_as_deal"); err != nil {
			return models.CRMObject{}, false, err
		}
	}
	if err := ensureCustomerChainRelationsTx(tx, auth, deal); err != nil {
		return models.CRMObject{}, false, err
	}
	return deal, false, nil
}

func createDraftOrderForDealTx(tx *gorm.DB, auth authContext, deal models.CRMObject, source models.CRMObject, patch map[string]interface{}) (models.CRMObject, error) {
	order := cloneCRMObject(auth, "orders", "order", source.Name+" 订单", source)
	order.Status = "draft"
	order.DealID = deal.ID
	order.CompanyID = firstNonZero(source.CompanyID, deal.CompanyID)
	order.ContactID = firstNonZero(source.ContactID, deal.ContactID)
	payload := mergePayload(payloadMap(deal.PayloadJSON), payloadMap(source.PayloadJSON))
	payload = mergePayload(payload, inheritAttributionPayload(payload))
	payload = mergePayload(payload, patch)
	payload["source_deal_id"] = deal.ID
	if _, ok := payload["payment_status"]; !ok {
		payload["payment_status"] = "pending"
	}
	if _, ok := payload["paid_amount"]; !ok {
		payload["paid_amount"] = 0
	}
	payload["receivable_balance"] = order.Amount
	recalculateOrderFinancials(&order, payload)
	order.PayloadJSON = toJSONString(payload)
	if err := normalizeCustomerChainObjectTx(tx, auth, &order, deal, source); err != nil {
		return models.CRMObject{}, err
	}
	if err := tx.Create(&order).Error; err != nil {
		return models.CRMObject{}, err
	}
	if err := ensureCRMRelationTx(tx, auth, "deals", deal.ID, "orders", order.ID, "generated_order"); err != nil {
		return models.CRMObject{}, err
	}
	if err := updateDealGeneratedOrderTx(tx, auth, deal, order); err != nil {
		return models.CRMObject{}, err
	}
	if err := ensureCustomerChainRelationsTx(tx, auth, order); err != nil {
		return models.CRMObject{}, err
	}
	return order, nil
}

func updateDealGeneratedOrderTx(tx *gorm.DB, auth authContext, deal models.CRMObject, order models.CRMObject) error {
	payload := payloadMap(deal.PayloadJSON)
	payload["order_id"] = order.ID
	payload["source_order_id"] = order.ID
	payload["order_status"] = order.Status
	deal.PayloadJSON = toJSONString(payload)
	deal.UpdatedBy = auth.UserID
	return tx.Save(&deal).Error
}

func validateOpportunityReadyToWin(opportunity models.CRMObject) error {
	payload := payloadMap(opportunity.PayloadJSON)
	if opportunity.CompanyID == 0 && firstPayloadText(payload, "company", "company_name") == "" {
		return fmt.Errorf("商机赢单前必须填写客户公司")
	}
	if opportunity.ContactID == 0 && firstPayloadText(payload, "contact_name", "contact") == "" {
		return fmt.Errorf("商机赢单前必须填写联系人")
	}
	if opportunity.Amount <= 0 {
		return fmt.Errorf("商机赢单前必须填写有效金额")
	}
	if len(interfaceList(payload["line_items"])) == 0 {
		return fmt.Errorf("商机赢单前必须填写产品明细")
	}
	return nil
}

func firstNonZero(values ...uint) uint {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
