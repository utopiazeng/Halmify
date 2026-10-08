package controllers

import (
	"ai-crm/models"
	"strings"
	"testing"
)

func TestCanOrderStatusTransition(t *testing.T) {
	cases := []struct {
		from string
		to   string
		ok   bool
	}{
		{from: "draft", to: "pending_review", ok: true},
		{from: "draft", to: "confirmed", ok: false},
		{from: "pending_review", to: "draft", ok: true},
		{from: "pending_review", to: "partial_shipped", ok: false},
		{from: "confirmed", to: "shipped", ok: true},
		{from: "confirmed", to: "completed", ok: false},
		{from: "partial_shipped", to: "completed", ok: true},
		{from: "shipped", to: "completed", ok: true},
		{from: "shipped", to: "canceled", ok: false},
		{from: "completed", to: "canceled", ok: false},
		{from: "", to: "draft", ok: false},
		{from: "draft", to: "", ok: false},
		{from: "draft", to: "draft", ok: false},
	}
	for _, tc := range cases {
		if got := canOrderStatusTransition(tc.from, tc.to); got != tc.ok {
			t.Fatalf("transition %s -> %s expected %v got %v", tc.from, tc.to, tc.ok, got)
		}
	}
}

func TestValidateOrderReadyForStatus(t *testing.T) {
	order := models.CRMObject{
		Amount:      100,
		PayloadJSON: toJSONString(map[string]interface{}{"line_items": []interface{}{map[string]interface{}{"name": "服务", "quantity": 1, "unit_price": 100}}, "payment_terms": "Net 30", "shipping_address": "上海"}),
	}
	if err := validateOrderReadyForStatus(order, "confirmed"); err != nil {
		t.Fatalf("expected order ready for confirmation, got %v", err)
	}
	order.PayloadJSON = toJSONString(map[string]interface{}{"payment_terms": "Net 30", "shipping_address": "上海"})
	if err := validateOrderReadyForStatus(order, "confirmed"); err == nil || !strings.Contains(err.Error(), "产品明细") {
		t.Fatalf("expected missing line items error, got %v", err)
	}
}

func TestValidateOrderReadyForCompleted(t *testing.T) {
	basePayload := map[string]interface{}{
		"line_items":         []interface{}{map[string]interface{}{"name": "服务", "quantity": 1, "unit_price": 100}},
		"payment_terms":      "Net 30",
		"shipping_address":   "上海",
		"paid_amount":        100,
		"receivable_balance": 0,
		"invoice_no":         "INV-001",
	}
	order := models.CRMObject{Amount: 100, Status: "shipped", PayloadJSON: toJSONString(basePayload)}
	if err := validateOrderReadyForStatus(order, "completed"); err != nil {
		t.Fatalf("expected completed order ready, got %v", err)
	}
	order.Status = "confirmed"
	if err := validateOrderReadyForStatus(order, "completed"); err == nil || !strings.Contains(err.Error(), "发货") {
		t.Fatalf("expected shipment gate error, got %v", err)
	}
	order.Status = "shipped"
	payload := map[string]interface{}{}
	for key, value := range basePayload {
		payload[key] = value
	}
	payload["receivable_balance"] = 10
	order.PayloadJSON = toJSONString(payload)
	if err := validateOrderReadyForStatus(order, "completed"); err == nil || !strings.Contains(err.Error(), "回款") {
		t.Fatalf("expected receivable gate error, got %v", err)
	}
	payload["receivable_balance"] = 0
	delete(payload, "invoice_no")
	order.PayloadJSON = toJSONString(payload)
	if err := validateOrderReadyForStatus(order, "completed"); err == nil || !strings.Contains(err.Error(), "发票") {
		t.Fatalf("expected invoice gate error, got %v", err)
	}
	payload["invoice_exemption_reason"] = "客户无需开票"
	order.PayloadJSON = toJSONString(payload)
	if err := validateOrderReadyForStatus(order, "completed"); err != nil {
		t.Fatalf("expected exemption reason to satisfy invoice gate, got %v", err)
	}
}

func TestCurrentReceivable(t *testing.T) {
	order := models.CRMObject{Amount: 100}
	payload := map[string]interface{}{"paid_amount": 20}
	if got := currentReceivable(order, payload); got != 80 {
		t.Fatalf("expected receivable 80 got %v", got)
	}
	payload["receivable_balance"] = 12
	if got := currentReceivable(order, payload); got != 12 {
		t.Fatalf("expected explicit receivable 12 got %v", got)
	}
}

func TestNormalizePaymentPlanWithCurrencyCleansEmptyNames(t *testing.T) {
	plan := normalizePaymentPlanWithCurrency([]map[string]interface{}{
		{"name": nil, "amount": 30, "currency": "", "due_date": nil},
		{"name": "<nil>", "amount": 70, "currency": "usd", "due_date": "2026-06-01"},
	}, "CNY")

	if len(plan) != 2 {
		t.Fatalf("expected two payment plan entries, got %d", len(plan))
	}
	if plan[0]["name"] != "第1期" || plan[0]["currency"] != "CNY" || plan[0]["due_date"] != "" {
		t.Fatalf("expected first entry to be normalized, got %#v", plan[0])
	}
	if plan[1]["name"] != "第2期" || plan[1]["currency"] != "USD" || plan[1]["due_date"] != "2026-06-01" {
		t.Fatalf("expected second entry to be normalized, got %#v", plan[1])
	}
}

func TestValidateOrderPaymentPlanRequiresAmountAndDueDate(t *testing.T) {
	if err := validateOrderPaymentPlan([]map[string]interface{}{{"name": "第1期", "amount": 10, "due_date": "2026-06-01"}}); err != nil {
		t.Fatalf("expected valid payment plan, got %v", err)
	}
	if err := validateOrderPaymentPlan([]map[string]interface{}{{"name": "第1期", "amount": 0, "due_date": "2026-06-01"}}); err == nil || !strings.Contains(err.Error(), "付款金额") {
		t.Fatalf("expected amount validation error, got %v", err)
	}
	if err := validateOrderPaymentPlan([]map[string]interface{}{{"name": "第1期", "amount": 10, "due_date": ""}}); err == nil || !strings.Contains(err.Error(), "到期日期") {
		t.Fatalf("expected due date validation error, got %v", err)
	}
}

func TestNormalizePaymentPlanFromInterfacePreservesMapSlice(t *testing.T) {
	plan := normalizePaymentPlanFromInterfaceWithCurrency([]map[string]interface{}{
		{"name": "第1期", "amount": 100, "currency": "CNY", "due_date": "2026-06-01"},
	}, "CNY")

	if len(plan) != 1 {
		t.Fatalf("expected one payment plan entry, got %#v", plan)
	}
	if plan[0]["amount"] != float64(100) || plan[0]["due_date"] != "2026-06-01" {
		t.Fatalf("expected payment plan to survive map slice normalization, got %#v", plan[0])
	}
}
