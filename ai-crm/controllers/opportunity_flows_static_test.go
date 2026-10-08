package controllers

import (
	"os"
	"strings"
	"testing"
)

func TestQuoteActionsAuthorizeThroughSourceOpportunity(t *testing.T) {
	source, err := os.ReadFile("opportunity_flows.go")
	if err != nil {
		t.Fatalf("read opportunity_flows.go: %v", err)
	}
	code := string(source)
	start := strings.Index(code, "func (c *CRMController) changeQuoteStatus")
	end := strings.Index(code, "func (c *CRMController) CreateOrderFromQuote")
	if start < 0 || end <= start {
		t.Fatalf("changeQuoteStatus function not found")
	}
	changeQuoteStatus := code[start:end]
	if strings.Contains(changeQuoteStatus, "ensureCRMObjectEditAccess(auth, quote") {
		t.Fatalf("quote status changes should authorize through the source opportunity, not quote object ACL")
	}
	if !strings.Contains(changeQuoteStatus, `utils.DB.Where("tenant_id = ? AND domain = ? AND id = ? AND object_type = ?"`) {
		t.Fatalf("quote status changes should load the quote by tenant before checking source opportunity access")
	}

	start = strings.Index(code, "func (c *CRMController) CreateOrderFromQuote")
	end = strings.Index(code, "func (c *CRMController) CreateOpportunityFromTicket")
	if start < 0 || end <= start {
		t.Fatalf("CreateOrderFromQuote function not found")
	}
	createOrder := code[start:end]
	if strings.Contains(createOrder, "ensureCRMObjectEditAccess(auth, quote") {
		t.Fatalf("quote order creation should authorize through the source opportunity, not quote object ACL")
	}
	if !strings.Contains(createOrder, `utils.DB.Where("tenant_id = ? AND domain = ? AND id = ? AND object_type = ?"`) {
		t.Fatalf("quote order creation should load the quote by tenant before checking source opportunity access")
	}
}
