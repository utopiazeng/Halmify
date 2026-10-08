package controllers

import (
	"ai-crm/models"
	"testing"
	"time"
)

func TestSalesComparisonPeriodCurrentYearToDate(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, 5, 23, 15, 30, 0, 0, loc)

	got := salesComparisonPeriod(now, 2026, "year", "2026")

	wantStart := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	wantLastStart := time.Date(2025, 1, 1, 0, 0, 0, 0, loc)
	wantLastEnd := time.Date(2025, 5, 23, 15, 30, 0, 0, loc)
	if !got.Start.Equal(wantStart) || !got.End.Equal(now) {
		t.Fatalf("current range = %s -> %s, want %s -> %s", got.Start, got.End, wantStart, now)
	}
	if !got.LastStart.Equal(wantLastStart) || !got.LastEnd.Equal(wantLastEnd) {
		t.Fatalf("last range = %s -> %s, want %s -> %s", got.LastStart, got.LastEnd, wantLastStart, wantLastEnd)
	}
}

func TestSalesComparisonPeriodFinishedMonthUsesFullMonth(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, 5, 23, 15, 30, 0, 0, loc)

	got := salesComparisonPeriod(now, 2026, "month", "2026-04")

	wantStart := time.Date(2026, 4, 1, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, loc)
	wantLastStart := time.Date(2025, 4, 1, 0, 0, 0, 0, loc)
	wantLastEnd := time.Date(2025, 5, 1, 0, 0, 0, 0, loc)
	if !got.Start.Equal(wantStart) || !got.End.Equal(wantEnd) {
		t.Fatalf("current range = %s -> %s, want %s -> %s", got.Start, got.End, wantStart, wantEnd)
	}
	if !got.LastStart.Equal(wantLastStart) || !got.LastEnd.Equal(wantLastEnd) {
		t.Fatalf("last range = %s -> %s, want %s -> %s", got.LastStart, got.LastEnd, wantLastStart, wantLastEnd)
	}
}

func TestSalesYoYWithoutLastYearData(t *testing.T) {
	got := salesYoY(120, 0)
	if got["has_last_year"] != false {
		t.Fatalf("has_last_year = %v, want false", got["has_last_year"])
	}
	if numericValue(got["yoy_rate"]) != 0 {
		t.Fatalf("yoy_rate = %v, want 0", got["yoy_rate"])
	}
	if numericValue(got["current_amount"]) != 120 {
		t.Fatalf("current_amount = %v, want 120", got["current_amount"])
	}
}

func TestNormalizeSalesMetricBasis(t *testing.T) {
	if normalizeSalesMetricBasis("order_amount") != "order_amount" {
		t.Fatalf("order_amount should be kept")
	}
	if normalizeSalesMetricBasis("") != "payment" {
		t.Fatalf("empty basis should default to payment")
	}
}

func TestIsSalesRole(t *testing.T) {
	if !isSalesRole(models.Role{Name: "销售员"}) {
		t.Fatalf("销售员 should be treated as sales role")
	}
	if !isSalesRole(models.Role{Code: "R_SALES"}) {
		t.Fatalf("R_SALES should be treated as sales role")
	}
	if !isSalesRole(models.Role{Name: "E2E销售员", Code: "E2E_R_SALES"}) {
		t.Fatalf("custom sales role should be treated as sales role")
	}
	if isSalesRole(models.Role{Name: "管理员", Code: "tenant_admin"}) {
		t.Fatalf("tenant admin should not be treated as sales role")
	}
}

func TestSalesOwnerIDFromPayload(t *testing.T) {
	payload := map[string]interface{}{"sales_owner_id": float64(42)}
	if got := salesOwnerIDFromPayload(payload); got != 42 {
		t.Fatalf("salesOwnerIDFromPayload = %d, want 42", got)
	}
}

func TestSalesOrderAttributionOwnerPrecedenceWithoutDB(t *testing.T) {
	order := models.CRMObject{OwnerID: 7, CreatedBy: 8}
	payload := map[string]interface{}{"sales_owner_id": 9}
	if got := salesOrderAttributionOwnerID(0, order, payload); got != 7 {
		t.Fatalf("owner attribution = %d, want order owner 7", got)
	}
	if got := salesOrderAttributionOwnerID(0, models.CRMObject{}, payload); got != 9 {
		t.Fatalf("owner attribution = %d, want payload owner 9", got)
	}
	if got := salesOrderAttributionOwnerID(0, models.CRMObject{CreatedBy: 8}, map[string]interface{}{}); got != 0 {
		t.Fatalf("owner attribution = %d, want unassigned without sales-role creator", got)
	}
}

func TestSalesOrderAmountStatusExcludesDraftReviewAndCanceled(t *testing.T) {
	for _, status := range []string{"draft", "active", "pending_review", "canceled", "cancelled"} {
		if salesOrderAmountStatus(status) {
			t.Fatalf("status %s should not count as order amount", status)
		}
	}
	for _, status := range []string{"confirmed", "partial_shipped", "shipped", "completed", "approved"} {
		if !salesOrderAmountStatus(status) {
			t.Fatalf("status %s should count as order amount", status)
		}
	}
}

func TestSalesMemberPeriodTargetFallback(t *testing.T) {
	ownerGoals := map[uint]models.CRMObject{11: {Amount: 1200}}
	periodGoals := map[uint]map[string]models.CRMObject{
		11: {
			"month:2026-05":   {Amount: 180},
			"quarter:2026-Q3": {Amount: 450},
		},
	}
	if got := salesMemberPeriodTarget(11, ownerGoals, periodGoals, "month", "2026-05"); got != 180 {
		t.Fatalf("month target = %v, want explicit 180", got)
	}
	if got := salesMemberPeriodTarget(11, ownerGoals, periodGoals, "month", "2026-06"); got != 100 {
		t.Fatalf("month fallback = %v, want annual/12", got)
	}
	if got := salesMemberPeriodTarget(11, ownerGoals, periodGoals, "quarter", "2026-Q3"); got != 450 {
		t.Fatalf("quarter target = %v, want explicit 450", got)
	}
	if got := salesMemberPeriodTarget(11, ownerGoals, periodGoals, "quarter", "2026-Q1"); got != 300 {
		t.Fatalf("quarter fallback = %v, want annual/4", got)
	}
}

func TestCompanyTargetDoesNotFallbackAcrossYears(t *testing.T) {
	if got := salesCompanyTargetAmount(models.CRMObject{}, map[string]interface{}{"target": 10000}); got != 0 {
		t.Fatalf("missing yearly company goal target = %v, want 0", got)
	}
	if got := salesCompanyTargetAmount(models.CRMObject{BaseModel: models.BaseModel{ID: 1}, Amount: 12000}, map[string]interface{}{"target": 10000}); got != 12000 {
		t.Fatalf("explicit yearly company goal target = %v, want 12000", got)
	}
}
