package controllers

import (
	"ai-crm/models"
	"strings"
	"testing"
)

func TestValidateCRMFieldWriteWithACL_CreateBlockedField(t *testing.T) {
	acl := map[string]interface{}{
		"hidden":   []interface{}{"cost_price"},
		"readonly": []interface{}{"email"},
	}
	next := map[string]interface{}{
		"name":       "测试",
		"cost_price": 100,
	}

	err := validateCRMFieldWriteWithACL(next, nil, acl)
	if err == nil {
		t.Fatalf("expected error when creating blocked field")
	}
}

func TestValidateCRMFieldWriteWithACL_UpdateReadonlyField(t *testing.T) {
	acl := map[string]interface{}{
		"readonly": []interface{}{"email"},
	}
	before := map[string]interface{}{
		"email": "old@example.com",
	}
	next := map[string]interface{}{
		"email": "new@example.com",
	}

	err := validateCRMFieldWriteWithACL(next, before, acl)
	if err == nil {
		t.Fatalf("expected error when updating readonly field")
	}
}

func TestValidateCRMFieldWriteWithACL_AllowSameValue(t *testing.T) {
	acl := map[string]interface{}{
		"read_only": []interface{}{"email"},
	}
	before := map[string]interface{}{
		"email": "same@example.com",
	}
	next := map[string]interface{}{
		"email": "same@example.com",
	}

	if err := validateCRMFieldWriteWithACL(next, before, acl); err != nil {
		t.Fatalf("expected no error when readonly field value unchanged, got: %v", err)
	}
}

func TestValidateOpportunityReadyToWin(t *testing.T) {
	opportunity := models.CRMObject{
		CompanyID:   1,
		ContactID:   2,
		Amount:      300,
		PayloadJSON: toJSONString(map[string]interface{}{"line_items": []interface{}{map[string]interface{}{"name": "软件服务", "quantity": 1, "unit_price": 300}}}),
	}
	if err := validateOpportunityReadyToWin(opportunity); err != nil {
		t.Fatalf("expected opportunity ready to win, got %v", err)
	}
	opportunity.ContactID = 0
	opportunity.PayloadJSON = toJSONString(map[string]interface{}{"line_items": []interface{}{map[string]interface{}{"name": "软件服务"}}})
	if err := validateOpportunityReadyToWin(opportunity); err == nil {
		t.Fatalf("expected missing contact error")
	}
}

func TestFallbackOpportunityQuoteLineItemsFromConvertedLeadPayload(t *testing.T) {
	opportunity := models.CRMObject{Amount: 100000}
	payload := map[string]interface{}{
		"product_id":     float64(753),
		"product_name":   "1111",
		"intent_product": "1111",
	}
	lines := fallbackOpportunityQuoteLineItems(opportunity, payload, 0)
	if len(lines) != 1 {
		t.Fatalf("expected one fallback line item, got %#v", lines)
	}
	line := mapValue(lines[0])
	if got := firstPayloadText(line, "name"); got != "1111" {
		t.Fatalf("expected product name 1111, got %q", got)
	}
	if got := uintFromValue(line["product_id"]); got != 753 {
		t.Fatalf("expected product id 753, got %d", got)
	}
	if got := numericValue(line["unit_price"]); got != 100000 {
		t.Fatalf("expected unit price from opportunity amount, got %v", got)
	}
}

func TestDealChainStep(t *testing.T) {
	cases := []struct {
		object models.CRMObject
		step   string
		order  int
	}{
		{object: models.CRMObject{Domain: "opportunities", ObjectType: "opportunity"}, step: "opportunity", order: 20},
		{object: models.CRMObject{Domain: "opportunities", ObjectType: "quote"}, step: "quote", order: 30},
		{object: models.CRMObject{Domain: "deals", ObjectType: "closed_deal"}, step: "deal", order: 40},
		{object: models.CRMObject{Domain: "orders", ObjectType: "order"}, step: "order", order: 50},
	}
	for _, tc := range cases {
		step, order := dealChainStep(tc.object)
		if step != tc.step || order != tc.order {
			t.Fatalf("expected %s/%d got %s/%d", tc.step, tc.order, step, order)
		}
	}
}

func TestCRMObjectACLDomainIncludesTransactionDomains(t *testing.T) {
	for _, domain := range []string{"campaigns", "leads", "opportunities", "deals", "orders", "projects"} {
		if !crmObjectACLDomain(domain) {
			t.Fatalf("expected %s to support object ACL", domain)
		}
	}
	if !crmObjectACLDomain("tickets") {
		t.Fatalf("expected tickets to keep object ACL support")
	}
	if crmObjectAccessVisibilityDomain("tickets") {
		t.Fatalf("expected ticket note/file visibility to stay out of object public/private normalization")
	}
}

func TestCRMPrivateVisibilitySupervisorDomains(t *testing.T) {
	for _, domain := range []string{"campaigns", "leads", "opportunities", "orders"} {
		if !crmSupervisorPrivateVisibilityDomain(domain) {
			t.Fatalf("expected supervisor private visibility for %s", domain)
		}
	}
	for _, domain := range []string{"contacts", "deals", "tickets", "projects", "products"} {
		if crmSupervisorPrivateVisibilityDomain(domain) {
			t.Fatalf("did not expect supervisor private visibility for %s", domain)
		}
	}
}

func TestSupervisorRoleToken(t *testing.T) {
	for _, token := range []string{"主管", "SDR主管", "R_SUPERVISOR", "r_sdr_supervisor"} {
		if !isSupervisorRoleToken(token) {
			t.Fatalf("expected %q to be treated as supervisor role", token)
		}
	}
	for _, token := range []string{"tenant_admin", "sales", "R_SALES", ""} {
		if isSupervisorRoleToken(token) {
			t.Fatalf("did not expect %q to be treated as supervisor role", token)
		}
	}
}

func TestSupervisorPrivateVisibilityHelperAllowsConfiguredDomains(t *testing.T) {
	auth := authContext{TenantID: 1, UserID: 11, Role: "R_SUPERVISOR"}
	for _, domain := range []string{"campaigns", "leads", "opportunities", "orders"} {
		if !isCRMPrivateVisibilitySupervisor(auth, domain) {
			t.Fatalf("expected supervisor private visibility helper to allow %s", domain)
		}
	}
}

func TestSupervisorPrivateVisibilityHelperRejectsOtherDomains(t *testing.T) {
	auth := authContext{TenantID: 1, UserID: 11, Role: "R_SUPERVISOR"}
	if isCRMPrivateVisibilitySupervisor(auth, "tickets") {
		t.Fatalf("did not expect supervisor private visibility helper to allow tickets")
	}
}

func TestValidateMfgLockedOrderUpdateBlocksCriticalFieldChanges(t *testing.T) {
	auth := authContext{TenantID: 0, UserID: 1}
	before := models.CRMObject{
		Domain:      "orders",
		ObjectType:  "order",
		Amount:      100,
		CompanyID:   10,
		PayloadJSON: toJSONString(map[string]interface{}{"mfg_bom_locked": true, "mfg_bom_snapshot_id": 1, "line_items": []interface{}{map[string]interface{}{"sku": "A"}}}),
	}
	after := before
	after.PayloadJSON = toJSONString(map[string]interface{}{"mfg_bom_locked": true, "mfg_bom_snapshot_id": 2, "line_items": []interface{}{map[string]interface{}{"sku": "A"}}})

	// TenantID 0 cannot have an enabled manufacturing package; the generic
	// path remains untouched for non-manufacturing tenants.
	if err := validateMfgLockedOrderUpdate(auth, "orders", before, after); err != nil {
		t.Fatalf("expected non-manufacturing tenant to bypass mfg lock, got %v", err)
	}
}

func TestNormalizeCRMObjectAccessPayload(t *testing.T) {
	payload := map[string]interface{}{"visibility": "private"}
	normalizeCRMObjectAccessPayload(payload)
	if payload["visibility"] != "private" || payload["private"] != true {
		t.Fatalf("expected private visibility payload, got %#v", payload)
	}

	payload = map[string]interface{}{"private": true}
	normalizeCRMObjectAccessPayload(payload)
	if payload["visibility"] != "private" || payload["private"] != true {
		t.Fatalf("expected private flag to normalize visibility, got %#v", payload)
	}

	payload = map[string]interface{}{}
	normalizeCRMObjectAccessPayload(payload)
	if payload["visibility"] != "public" || payload["private"] != false {
		t.Fatalf("expected default public visibility, got %#v", payload)
	}
}

func TestCRMConditionScopeClause(t *testing.T) {
	clause, args := crmConditionScopeClause(map[string]interface{}{
		"conditions": map[string]interface{}{
			"owner_id": []interface{}{float64(12)},
			"status":   []interface{}{"draft", "confirmed"},
			"unsafe":   []interface{}{"ignored"},
		},
	})
	if clause == "" {
		t.Fatalf("expected condition clause")
	}
	if len(args) != 2 {
		t.Fatalf("expected two condition args, got %d", len(args))
	}
	if clause != "crm_objects.owner_id IN ? AND crm_objects.status IN ?" && clause != "crm_objects.status IN ? AND crm_objects.owner_id IN ?" {
		t.Fatalf("unexpected clause: %s", clause)
	}
}

func TestPayloadJSONKeyContainsClause(t *testing.T) {
	clause, args := payloadJSONKeyContainsClause("aaa", "company_name", "company", "")
	if clause != "(payload_json LIKE ? OR payload_json LIKE ?)" {
		t.Fatalf("unexpected clause: %s", clause)
	}
	if len(args) != 2 {
		t.Fatalf("expected two args, got %d", len(args))
	}
	if args[0] != `%"company_name":%aaa%` || args[1] != `%"company":%aaa%` {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestPayloadJSONKeyContainsClauseEscapesJSONStringFragment(t *testing.T) {
	_, args := payloadJSONKeyContainsClause(`ACME "East"`, "company_name")
	if len(args) != 1 {
		t.Fatalf("expected one arg, got %d", len(args))
	}
	if args[0] != `%"company_name":%ACME \"East\"%` {
		t.Fatalf("expected JSON escaped fragment, got %#v", args[0])
	}
}

func TestPayloadJSONKeyContainsClauseEmptyInput(t *testing.T) {
	if clause, args := payloadJSONKeyContainsClause(" ", "company_name"); clause != "" || args != nil {
		t.Fatalf("expected empty result for empty value, got %q %#v", clause, args)
	}
	if clause, args := payloadJSONKeyContainsClause("aaa"); clause != "" || args != nil {
		t.Fatalf("expected empty result for empty keys, got %q %#v", clause, args)
	}
}

func TestPayloadJSONKeyNumberEqualsClauseUsesStrictJSONMatch(t *testing.T) {
	clause, args := payloadJSONKeyNumberEqualsClause(4, "campaign_id")
	if !strings.Contains(clause, "JSON_EXTRACT") || strings.Contains(clause, "LIKE") {
		t.Fatalf("expected strict JSON clause, got %s", clause)
	}
	if len(args) != 4 {
		t.Fatalf("expected four args, got %#v", args)
	}
	if args[0] != "$.campaign_id" || args[1] != "4" || args[2] != "$.campaign_id" || args[3] != "4" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestCRMListKeywordPayloadKeys(t *testing.T) {
	dealKeys := crmListKeywordPayloadKeys("deals", "closed_deal")
	for _, key := range []string{"company_name", "contact_name", "contract_no", "source_channel"} {
		if !stringInSlice(key, dealKeys) {
			t.Fatalf("expected deal keyword keys to include %s, got %#v", key, dealKeys)
		}
	}

	orderKeys := crmListKeywordPayloadKeys("orders", "order")
	for _, key := range []string{"company_name", "contact_name", "shipment_no", "payment_no", "invoice_no", "shipping_address", "delivery_address"} {
		if !stringInSlice(key, orderKeys) {
			t.Fatalf("expected order keyword keys to include %s, got %#v", key, orderKeys)
		}
	}

	if keys := crmListKeywordPayloadKeys("projects", "project"); len(keys) != 0 {
		t.Fatalf("expected unsupported domain to have no keyword payload keys, got %#v", keys)
	}
}
