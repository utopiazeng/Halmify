package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"gorm.io/gorm"
	"strings"
)

func normalizeOrderRequest(req *crmObjectRequest, payload map[string]interface{}) {
	status := normalizeOrderStatus(req.Status)
	if status == "" {
		status = "draft"
	}
	req.Status = status
	if _, ok := payload["line_items"]; !ok {
		payload["line_items"] = []interface{}{}
	}
	if strings.TrimSpace(fmt.Sprint(payload["payment_status"])) == "" {
		payload["payment_status"] = "pending"
	}
	order := models.CRMObject{Amount: req.Amount}
	recalculateOrderFinancials(&order, payload)
	req.Amount = order.Amount
}

func preserveCompletedOrderLockedFields(before models.CRMObject, next *models.CRMObject) {
	if before.Domain != "orders" || before.ObjectType != "order" || before.Status != "completed" || next == nil {
		return
	}
	beforePayload := payloadMap(before.PayloadJSON)
	nextPayload := payloadMap(next.PayloadJSON)
	for _, key := range []string{"line_items", "paid_amount", "payments", "payment_no", "payment_method", "paid_at", "payment_status", "receivable_balance", "invoice_no", "invoice_type", "invoice", "invoice_exemption_reason", "invoice_exempted_at", "shipment_no", "carrier", "shipped_at"} {
		if value, ok := beforePayload[key]; ok {
			nextPayload[key] = value
		} else {
			delete(nextPayload, key)
		}
	}
	next.Amount = before.Amount
	next.Currency = before.Currency
	next.Status = before.Status
	recalculateOrderFinancials(next, nextPayload)
	next.PayloadJSON = toJSONString(nextPayload)
}

func ensureOrderNotCompleted(order models.CRMObject, action string) error {
	if order.Domain == "orders" && order.ObjectType == "order" && order.Status == "completed" {
		return fmt.Errorf("已完成订单不能%s", action)
	}
	return nil
}

func validateOrderPaymentPlan(plan []map[string]interface{}) error {
	if len(plan) == 0 {
		return fmt.Errorf("付款计划不能为空")
	}
	for index, item := range plan {
		if numericValue(item["amount"]) <= 0 {
			return fmt.Errorf("第%d期付款金额必须大于0", index+1)
		}
		if _, ok := parseDateOnly(fmt.Sprint(item["due_date"])); !ok {
			return fmt.Errorf("第%d期到期日期不能为空或格式不正确", index+1)
		}
	}
	return nil
}

func reserveOrderInventory(tx *gorm.DB, auth authContext, order models.CRMObject) error {
	payload := payloadMap(order.PayloadJSON)
	lines := interfaceList(payload["line_items"])
	if len(lines) == 0 {
		return nil
	}
	for _, raw := range lines {
		line, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		productID := uintFromValue(line["product_id"])
		quantity := int(uintFromValue(line["quantity"]))
		if productID == 0 || quantity <= 0 {
			continue
		}
		var product models.CRMObject
		if err := tx.Where("id = ? AND tenant_id = ? AND domain = ?", productID, auth.TenantID, "products").First(&product).Error; err != nil {
			return fmt.Errorf("订单产品[%d]不存在", productID)
		}
		productPayload := payloadMap(product.PayloadJSON)
		stock := int(uintFromValue(productPayload["stock"]))
		if stock < quantity {
			return fmt.Errorf("产品[%d]库存不足", productID)
		}
		productPayload["stock"] = stock - quantity
		product.PayloadJSON = toJSONString(productPayload)
		product.UpdatedBy = auth.UserID
		if err := tx.Save(&product).Error; err != nil {
			return err
		}
		relation := newCRMRelation(auth, "orders", order.ID, "products", product.ID, "deducted_inventory")
		relation.PayloadJSON = toJSONString(map[string]interface{}{"quantity": quantity})
		if err := tx.Create(&relation).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeOrderStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	aliases := map[string]string{
		"active":          "draft",
		"approving":       "pending_review",
		"approved":        "confirmed",
		"pending_payment": "confirmed",
		"paid":            "confirmed",
		"refunded":        "canceled",
		"cancelled":       "canceled",
	}
	if value, ok := aliases[status]; ok {
		status = value
	}
	for _, allowed := range []string{"draft", "pending_review", "confirmed", "partial_shipped", "shipped", "completed", "canceled"} {
		if status == allowed {
			return status
		}
	}
	return ""
}

func canOrderStatusTransition(from string, to string) bool {
	if from == "" || to == "" || from == to {
		return false
	}
	allowed := map[string]map[string]bool{
		"draft": {
			"pending_review": true,
			"canceled":       true,
		},
		"pending_review": {
			"confirmed": true,
			"draft":     true,
			"canceled":  true,
		},
		"confirmed": {
			"partial_shipped": true,
			"shipped":         true,
			"canceled":        true,
		},
		"partial_shipped": {
			"shipped":   true,
			"completed": true,
		},
		"shipped": {
			"completed": true,
		},
	}
	next, ok := allowed[from]
	if !ok {
		return false
	}
	return next[to]
}

func quoteOpportunity(auth authContext, quoteID uint) (models.CRMObject, error) {
	var relation models.CRMRelation
	err := utils.DB.Where("tenant_id = ? AND source_type = ? AND target_type = ? AND target_id = ? AND relation_type = ?", auth.TenantID, "opportunities", "opportunities", quoteID, "quote").First(&relation).Error
	if err != nil {
		return models.CRMObject{}, err
	}
	var opportunity models.CRMObject
	if err := crmScopedObjectQuery(auth, "opportunities").Where("id = ? AND object_type = ?", relation.SourceID, "opportunity").First(&opportunity).Error; err != nil {
		return models.CRMObject{}, err
	}
	return opportunity, nil
}

func recalculateOrderFinancials(order *models.CRMObject, payload map[string]interface{}) {
	auth := authContext{TenantID: order.TenantID}
	lines := interfaceList(payload["line_items"])
	applyTaxCodesToLineItems(auth, lines)
	subtotal := sumLineItems(lines)
	taxTotal := sumLineItemTax(lines)
	if len(lines) > 0 {
		payload["line_items"] = lines
	}
	if subtotal > 0 {
		payload["subtotal_amount"] = subtotal
		payload["tax_amount"] = taxTotal
		if strings.TrimSpace(fmt.Sprint(payload["amount_mode"])) != "manual" {
			order.Amount = subtotal + taxTotal
		}
	}
	if order.Amount < 0 {
		order.Amount = 0
	}
	paid := numericValue(payload["paid_amount"])
	receivable := order.Amount - paid
	if receivable < 0 {
		receivable = 0
	}
	payload["receivable_balance"] = receivable
	if paid <= 0 {
		payload["payment_status"] = "pending"
	} else if receivable > 0 {
		payload["payment_status"] = "partial_paid"
	} else {
		payload["payment_status"] = "paid"
	}
	plan := normalizePaymentPlanFromInterfaceWithCurrency(payload["payment_plan"], order.Currency)
	if len(plan) > 0 {
		payload["payment_plan"] = plan
		payload["overdue_amount"] = overduePaymentAmount(plan, paid)
	}
	applyOrderExchangeRate(auth, order, payload)
}

func orderFinanceSetting(auth authContext) map[string]interface{} {
	setting := crmSettingValue(auth.TenantID, "order_finance", "default")
	if len(setting) == 0 {
		baseCurrency := tenantBusinessCurrency(auth.TenantID)
		setting = map[string]interface{}{
			"base_currency": baseCurrency,
			"exchange_rates": map[string]interface{}{
				baseCurrency: 1,
				"USD":        1,
				"CNY":        0.14,
				"EUR":        1.08,
				"GBP":        1.26,
				"JPY":        0.0069,
				"KRW":        0.00072,
				"HKD":        0.13,
				"TWD":        0.031,
				"SGD":        0.74,
				"AUD":        0.65,
				"CAD":        0.72,
				"AED":        0.27,
			},
			"tax_codes": []interface{}{
				map[string]interface{}{"code": "VAT13", "name": "增值税13%", "rate": 13},
				map[string]interface{}{"code": "VAT6", "name": "服务税6%", "rate": 6},
				map[string]interface{}{"code": "ZERO", "name": "零税率", "rate": 0},
			},
		}
	}
	return setting
}

func validateOrderReadyForStatus(order models.CRMObject, nextStatus string) error {
	if nextStatus != "confirmed" && nextStatus != "partial_shipped" && nextStatus != "shipped" && nextStatus != "completed" {
		return nil
	}
	payload := payloadMap(order.PayloadJSON)
	if len(interfaceList(payload["line_items"])) == 0 {
		return fmt.Errorf("订单确认前必须填写产品明细")
	}
	if strings.TrimSpace(firstPayloadText(payload, "payment_terms")) == "" {
		return fmt.Errorf("订单确认前必须填写付款条款")
	}
	if strings.TrimSpace(firstPayloadText(payload, "shipping_address", "delivery_address", "delivery_note")) == "" {
		return fmt.Errorf("订单确认前必须填写收货或交付信息")
	}
	if order.Amount <= 0 {
		return fmt.Errorf("订单确认前必须填写有效金额")
	}
	if nextStatus != "completed" {
		return nil
	}
	if order.Status != "shipped" && order.Status != "partial_shipped" {
		return fmt.Errorf("订单完成前必须先登记发货")
	}
	if currentReceivable(order, payload) > 0 {
		return fmt.Errorf("订单完成前必须收齐回款")
	}
	if strings.TrimSpace(firstPayloadText(payload, "invoice_no")) == "" && strings.TrimSpace(firstPayloadText(payload, "invoice_exemption_reason")) == "" {
		return fmt.Errorf("订单完成前必须登记发票或免票原因")
	}
	return nil
}

func currentReceivable(order models.CRMObject, payload map[string]interface{}) float64 {
	receivable := numericValue(payload["receivable_balance"])
	if receivable <= 0 {
		receivable = order.Amount - numericValue(payload["paid_amount"])
	}
	if receivable < 0 {
		return 0
	}
	return receivable
}

func applyOrderExchangeRate(auth authContext, order *models.CRMObject, payload map[string]interface{}) {
	setting := orderFinanceSetting(auth)
	baseCurrency := strings.TrimSpace(fmt.Sprint(setting["base_currency"]))
	if baseCurrency == "" {
		baseCurrency = tenantBusinessCurrency(auth.TenantID)
	}
	currency := strings.TrimSpace(order.Currency)
	if currency == "" {
		currency = baseCurrency
		order.Currency = currency
	}
	rate := exchangeRateForCurrency(setting, currency)
	payload["base_currency"] = baseCurrency
	payload["exchange_rate"] = rate
	payload["base_amount"] = order.Amount * rate
	payload["base_tax_amount"] = numericValue(payload["tax_amount"]) * rate
	payload["base_receivable_balance"] = numericValue(payload["receivable_balance"]) * rate
	payload["base_overdue_amount"] = numericValue(payload["overdue_amount"]) * rate
}
