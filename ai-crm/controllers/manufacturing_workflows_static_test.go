package controllers

import (
	"os"
	"strings"
	"testing"

	"github.com/beego/beego/v2/server/web/context"
)

func TestManufacturingWriteHandlersRecordOperationLog(t *testing.T) {
	raw, err := os.ReadFile("manufacturing_workflows.go")
	if err != nil {
		t.Fatalf("read manufacturing workflow controller: %v", err)
	}
	source := string(raw)
	for _, fn := range []string{
		"Create",
		"Update",
		"Delete",
		"SubmitRFQ",
		"StartRFQCosting",
		"CreateRFQClarification",
		"CreateBOMSnapshotFromConfig",
		"CalculateCostSheet",
		"CompleteCostSheet",
		"CreateQuote",
		"SendQuote",
		"TrackQuoteCustomerView",
		"ApproveDrawingAccess",
		"CreateOrderFromQuote",
		"CreateAssetsFromShipment",
		"updateAssetIoTAuthorization",
		"CreateServiceTicketFromIoTAlarm",
		"CreateOpportunityFromServiceTicket",
		"CreateSparePartQuoteForServiceTicket",
		"RefreshMetricSnapshots",
		"CreateSampleTestRecord",
		"RaiseProductionRisk",
		"changeQuoteApproval",
		"changeECN",
	} {
		body := functionBody(source, fn)
		if !strings.Contains(body, "logMfgOperation") {
			t.Fatalf("%s must record manufacturing operation log", fn)
		}
	}
}

func TestManufacturingClosedLoopRoutesCoverPursueGoalJourneys(t *testing.T) {
	controllerSource := mustReadTestFile(t, "manufacturing_workflows.go")
	serviceSource := mustReadTestFile(t, "../services/manufacturing_workflows.go")
	routerSource := mustReadTestFile(t, "../routers/router.go")

	expectedRoutes := []string{
		"/api/v1/mfg/customers/:customer_id/360",
		"/api/v1/mfg/rfqs/:id/quote-workspace",
		"/api/v1/mfg/rfq-clarifications/:id/answer-histories",
		"/api/v1/mfg/rfqs/:id/start-costing",
		"/api/v1/mfg/cpq/configure",
		"/api/v1/mfg/bom-snapshots/from-config",
		"/api/v1/mfg/cost-sheets/calculate",
		"/api/v1/mfg/cost-sheets/:id/complete",
		"/api/v1/mfg/quotes/:id/send",
		"/api/v1/mfg/quotes/order",
		"/api/v1/mfg/sample-projects/:id/test-records",
		"/api/v1/mfg/production-progress/:id/raise-risk",
		"/api/v1/mfg/ecns/:id/effective",
		"/api/v1/mfg/assets/from-shipment",
		"/api/v1/mfg/assets/:id/customer-view",
		"/api/v1/mfg/assets/:id/iot-authorize",
		"/api/v1/mfg/assets/:id/iot-revoke",
		"/api/v1/mfg/service-tickets/from-iot-alarm",
		"/api/v1/mfg/service-tickets/:id/create-opportunity",
		"/api/v1/mfg/dashboard/refresh-metrics",
		"/api/v1/mfg/dashboard/drilldown",
		"/api/v1/mfg/integration-outbox/:id/retry",
		"/api/v1/mfg/integration-outbox/:id/dead-letter",
	}
	for _, route := range expectedRoutes {
		if !strings.Contains(routerSource, route) {
			t.Fatalf("expected manufacturing closed-loop route %s", route)
		}
	}
	for _, route := range []string{
		`"/api/v1/mfg/:resource", "get:List;post:Create"`,
		`"/api/v1/mfg/:resource/:id", "get:Get;put:Update;delete:Delete"`,
	} {
		if !strings.Contains(routerSource, route) {
			t.Fatalf("expected manufacturing CRUD route %s", route)
		}
	}
	if strings.Contains(routerSource, `"/api/v1/mfg/*"`) {
		t.Fatalf("manufacturing wildcard placeholder route must not shadow real CRUD routes")
	}

	expectedControllerFns := []string{
		"RFQQuoteWorkspace",
		"RFQClarificationAnswerHistories",
		"StartRFQCosting",
		"ConfigureCPQ",
		"CreateBOMSnapshotFromConfig",
		"CalculateCostSheet",
		"CompleteCostSheet",
		"SendQuote",
		"CreateOrderFromQuote",
		"CreateSampleTestRecord",
		"RaiseProductionRisk",
		"EffectiveECN",
		"CreateAssetsFromShipment",
		"AssetCustomerView",
		"AuthorizeAssetIoT",
		"RevokeAssetIoT",
		"CreateServiceTicketFromIoTAlarm",
		"CreateOpportunityFromServiceTicket",
		"RefreshMetricSnapshots",
		"MetricDrilldown",
		"RetryIntegrationOutbox",
		"DeadLetterIntegrationOutbox",
	}
	for _, fn := range expectedControllerFns {
		if functionBody(controllerSource, fn) == "" {
			t.Fatalf("expected controller function %s for manufacturing closed-loop journey", fn)
		}
	}

	expectedServiceFns := []string{
		"MfgRFQQuoteWorkspace",
		"StartMfgRFQCosting",
		"ConfigureMfgCPQ",
		"CreateMfgBOMSnapshotFromConfig",
		"CalculateMfgCostSheet",
		"CompleteMfgCostSheet",
		"SendMfgQuote",
		"CreateMfgOrderFromQuote",
		"CreateMfgSampleTestRecord",
		"RaiseMfgProductionRisk",
		"ChangeMfgECNStatus",
		"CreateMfgAssetsFromShipment",
		"MfgAssetCustomerView",
		"UpdateMfgAssetIoTAuthorization",
		"CreateMfgServiceTicketFromIoTAlarm",
		"CreateMfgOpportunityFromServiceTicket",
		"RefreshMfgMetricSnapshots",
		"MfgMetricDrilldown",
		"RetryMfgIntegrationOutbox",
		"DeadLetterMfgIntegrationOutbox",
		"mfgDashboardMetricGroups",
	}
	for _, fn := range expectedServiceFns {
		if !strings.Contains(serviceSource, "func "+fn) {
			t.Fatalf("expected service function %s for manufacturing closed-loop journey", fn)
		}
	}
	for _, group := range []string{"growth", "efficiency", "quality", "delivery", "profit", "cash", "lifecycle"} {
		if !strings.Contains(serviceSource, `"key":            "`+group+`"`) {
			t.Fatalf("expected dashboard metric group %s", group)
		}
	}
}

func TestManufacturingResourcePermissions(t *testing.T) {
	cases := []struct {
		resource string
		action   string
		want     string
	}{
		{resource: "rfqs", action: "view", want: "mfg:rfq:view"},
		{resource: "rfq-clarifications", action: "manage", want: "mfg:rfq:manage"},
		{resource: "bom-templates", action: "manage", want: "mfg:product:manage"},
		{resource: "bom-snapshots", action: "manage", want: "mfg:product:manage"},
		{resource: "cost-sheets", action: "manage", want: "mfg:cost:manage"},
		{resource: "quotes", action: "manage", want: "mfg:quote:manage"},
		{resource: "production-progress", action: "view", want: "mfg:production:view"},
		{resource: "ecns", action: "manage", want: "mfg:ecn:manage"},
		{resource: "compensation-records", action: "manage", want: "mfg:compensation:manage"},
	}
	for _, tc := range cases {
		t.Run(tc.resource+"_"+tc.action, func(t *testing.T) {
			ctx := &context.Context{}
			ctx.Input = &context.BeegoInput{}
			ctx.Input.SetParam(":resource", tc.resource)
			controller := &ManufacturingWorkflowController{}
			controller.Ctx = ctx
			if got := controller.permission(tc.action); got != tc.want {
				t.Fatalf("permission(%q, %q) = %q, want %q", tc.resource, tc.action, got, tc.want)
			}
		})
	}
}

func TestRFQClarificationAnswerPermissionIsReadScoped(t *testing.T) {
	controllerSource := mustReadTestFile(t, "manufacturing_workflows.go")
	updateBody := functionBody(controllerSource, "requireUpdatePermission")
	if !strings.Contains(updateBody, `resource == "rfq-clarifications"`) || !strings.Contains(updateBody, `"mfg:rfq:view"`) {
		t.Fatalf("rfq clarification answer updates should be allowed for manufacturing RFQ viewers")
	}
	deleteBody := functionBody(controllerSource, "requireDeletePermission")
	if !strings.Contains(deleteBody, `resource == "rfq-clarifications"`) || !strings.Contains(deleteBody, `c.permission("manage")`) {
		t.Fatalf("rfq clarification deletes must remain manage-scoped")
	}
	if !strings.Contains(controllerSource, "answerOnlyMfgRFQClarificationInput") {
		t.Fatalf("non-manager clarification updates should be filtered to answer/status fields")
	}
	if !strings.Contains(functionBody(controllerSource, "Update"), "c.requireUpdatePermission(resource, id)") {
		t.Fatalf("Update must use update permission checks")
	}
	if !strings.Contains(functionBody(controllerSource, "Delete"), "c.requireDeletePermission(resource, id)") {
		t.Fatalf("Delete must use delete permission checks")
	}
}

func TestRFQCreatorCanAdvanceOwnQuoteActions(t *testing.T) {
	controllerSource := mustReadTestFile(t, "manufacturing_workflows.go")
	for _, fn := range []string{
		"StartRFQCosting",
		"CreateQuoteRevision",
		"SendQuote",
		"ConfirmQuoteByCustomer",
		"CreateOrderFromQuote",
	} {
		body := functionBody(controllerSource, fn)
		if !strings.Contains(body, "canCreateCostForRFQ") &&
			!strings.Contains(body, "canManageQuoteByID") {
			t.Fatalf("%s must use RFQ row-level capability checks", fn)
		}
		if strings.Contains(body, `requireManufacturingPermission(c.Ctx, "mfg:quote:manage")`) ||
			strings.Contains(body, `requireManufacturingPermission(c.Ctx, "mfg:cost:manage")`) {
			t.Fatalf("%s must not be hard-gated by global manufacturing manage permission", fn)
		}
		if !strings.Contains(body, `requireManufacturingPermission(c.Ctx, "mfg:`) {
			t.Fatalf("%s must still verify manufacturing package/read access before row-level checks", fn)
		}
	}
	approvalBody := functionBody(controllerSource, "changeQuoteApproval")
	if !strings.Contains(approvalBody, `requireManufacturingPermission(c.Ctx, "mfg:quote:approve")`) ||
		!strings.Contains(approvalBody, "authCanApproveMfgQuote") {
		t.Fatalf("quote approve/reject must remain approval-permission scoped")
	}
	if !strings.Contains(approvalBody, "canManageQuoteByID") {
		t.Fatalf("quote submit approval must use RFQ row-level quote capability")
	}
}

func mustReadTestFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func functionBody(source string, name string) string {
	marker := "func (c *ManufacturingWorkflowController) " + name
	start := strings.Index(source, marker)
	if start < 0 {
		marker = "func (c *ManufacturingWorkflowController) " + name
		start = strings.Index(source, marker)
	}
	if start < 0 {
		return ""
	}
	open := strings.Index(source[start:], "{")
	if open < 0 {
		return ""
	}
	pos := start + open
	depth := 0
	for i := pos; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[pos : i+1]
			}
		}
	}
	return ""
}
