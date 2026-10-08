package controllers

import (
	"ai-crm/models"
	"testing"
)

func TestNormalizeCustomerChainObjectInheritsCompanyContactAndCountry(t *testing.T) {
	company := models.CRMObject{
		Domain:      "contacts",
		ObjectType:  "company",
		Name:        "Acme Japan",
		PayloadJSON: toJSONString(map[string]interface{}{"country": "日本"}),
	}
	company.ID = 23
	contact := models.CRMObject{Domain: "contacts", ObjectType: "contact", Name: "Yuki"}
	contact.ID = 31
	opportunity := models.CRMObject{
		Domain:      "opportunities",
		ObjectType:  "opportunity",
		PayloadJSON: toJSONString(map[string]interface{}{"company_id": float64(23), "contact_id": float64(31)}),
	}
	order := models.CRMObject{
		Domain:      "orders",
		ObjectType:  "order",
		PayloadJSON: toJSONString(map[string]interface{}{"source_opportunity_id": float64(9)}),
	}

	if err := normalizeCustomerChainObjectTx(nil, authContext{}, &order, opportunity, company, contact); err != nil {
		t.Fatalf("normalizeCustomerChainObjectTx returned error: %v", err)
	}

	payload := payloadMap(order.PayloadJSON)
	if order.CompanyID != 23 || uintFromValue(payload["company_id"]) != 23 {
		t.Fatalf("company chain not inherited: company_id=%d payload=%#v", order.CompanyID, payload["company_id"])
	}
	if order.ContactID != 31 || uintFromValue(payload["contact_id"]) != 31 {
		t.Fatalf("contact chain not inherited: contact_id=%d payload=%#v", order.ContactID, payload["contact_id"])
	}
	if payload["company_name"] != "Acme Japan" || payload["customer_name"] != "Acme Japan" {
		t.Fatalf("company names not normalized: %#v", payload)
	}
	if payload["contact_name"] != "Yuki" {
		t.Fatalf("contact name not normalized: %#v", payload)
	}
	if payload["country"] != "日本" || payload["customer_country"] != "日本" {
		t.Fatalf("country not normalized from company payload: %#v", payload)
	}
}

func TestSalesOrderDimensionsUsePayloadCompanyIDAndCompanyCountry(t *testing.T) {
	order := models.CRMObject{Name: "ORD-1", CompanyID: 0, PayloadJSON: toJSONString(map[string]interface{}{"company_id": float64(45)})}
	payload := payloadMap(order.PayloadJSON)
	companyNames := map[uint]string{45: "Global Buyer"}
	companyPayload := map[string]interface{}{"country": "德国"}

	if got := salesOrderCustomerName(order, payload, companyNames); got != "Global Buyer" {
		t.Fatalf("salesOrderCustomerName = %q, want company name", got)
	}
	if got := salesOrderCountry(payload, companyPayload); got != "德国" {
		t.Fatalf("salesOrderCountry = %q, want company country", got)
	}
}

func TestSalesOrderDimensionsUseChainPayloadFallback(t *testing.T) {
	order := models.CRMObject{Name: "ORD-2"}
	payload := map[string]interface{}{}
	chainPayload := map[string]interface{}{"company_name": "Linked Customer", "country": "法国"}

	if got := salesOrderCustomerName(order, payload, nil, chainPayload); got != "Linked Customer" {
		t.Fatalf("salesOrderCustomerName = %q, want chain customer", got)
	}
	if got := salesOrderCountry(payload, nil, chainPayload); got != "法国" {
		t.Fatalf("salesOrderCountry = %q, want chain country", got)
	}
}
