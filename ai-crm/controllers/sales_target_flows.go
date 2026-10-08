package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	salesGoalDomain     = "teams"
	salesGoalObjectType = "sales_goal"
)

var salesRoleNames = map[string]bool{
	"销售员":  true,
	"销售代表": true,
}

var salesRoleCodes = map[string]bool{
	"sales":         true,
	"salesperson":   true,
	"sales_rep":     true,
	"r_sales":       true,
	"r_salesperson": true,
	"r_sales_rep":   true,
}

type salesGoalInput struct {
	OwnerID           uint
	Year              int
	PeriodType        string
	Period            string
	Target            float64
	Currency          string
	MetricBasis       string
	AmountUnit        string
	ParentGoalID      uint
	PlannedOrderCount float64
	FocusCustomers    []string
	FocusProducts     []string
	ActionPlan        string
	AllocatedBy       uint
	PlannedBy         uint
	Name              string
}

type ownerSalesMetrics struct {
	OrderAmount float64
	PaidAmount  float64
	OrderCount  int64
}

type unassignedSalesMetrics struct {
	OrderAmount float64 `json:"order_amount"`
	PaidAmount  float64 `json:"paid_amount"`
	OrderCount  int64   `json:"order_count"`
}

type salesDimensionMetric struct {
	Key         string
	Name        string
	Amount      float64
	PaidAmount  float64
	Quantity    float64
	Count       int64
	ProductID   uint
	CompanyID   uint
	CountryCode string
}

type salesOrderAggregate struct {
	OrderAmount float64
	PaidAmount  float64
	OrderCount  int64
	Owners      map[uint]*ownerSalesMetrics
	Unassigned  unassignedSalesMetrics
	Countries   map[string]*salesDimensionMetric
	Customers   map[string]*salesDimensionMetric
	Products    map[string]*salesDimensionMetric
}

type salesComparisonRange struct {
	PeriodType string
	Period     string
	Start      time.Time
	End        time.Time
	LastStart  time.Time
	LastEnd    time.Time
}

func (c *CRMController) SalesTargetSummary() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	year := queryYear(c.Ctx.Input.Query("year"))
	utils.Success(c.Ctx, buildSalesTargetSummaryWithOptions(auth, year, salesTargetSummaryOptions{Month: c.Ctx.Input.Query("month")}))
}

func (c *CRMController) MySalesTargets() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	year := queryYear(c.Ctx.Input.Query("year"))
	summary := buildSalesTargetSummary(auth, year)
	utils.Success(c.Ctx, map[string]interface{}{
		"year":            year,
		"metric_basis":    summary["metric_basis"],
		"company_goal":    summary["company_goal"],
		"my_goal":         summary["my_goal"],
		"my_plans":        summary["my_plans"],
		"my_progress":     summary["my_progress"],
		"current_month":   summary["current_month"],
		"current_quarter": summary["current_quarter"],
	})
}

func (c *CRMController) SaveCompanySalesTarget() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageSalesTargets(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "仅租户管理员或大屏/团队管理权限用户可设置公司销售目标")
		return
	}
	var req struct {
		Year        int     `json:"year"`
		Target      float64 `json:"target"`
		Amount      float64 `json:"amount"`
		MetricBasis string  `json:"metric_basis"`
		AmountUnit  string  `json:"amount_unit"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.Year <= 0 {
		req.Year = time.Now().Year()
	}
	target := req.Target
	if target <= 0 {
		target = req.Amount
	}
	if target <= 0 {
		utils.BadRequest(c.Ctx, "年度目标必须大于0")
		return
	}
	metricBasis := normalizeSalesMetricBasis(req.MetricBasis)
	currency := salesTargetCurrency(auth.TenantID)
	amountUnit := salesTargetSettingAmountUnit(auth.TenantID)
	if strings.TrimSpace(req.AmountUnit) != "" {
		amountUnit = normalizeSalesAmountUnit(req.AmountUnit)
	}
	goal, err := upsertSalesGoal(auth, salesGoalInput{
		OwnerID:     0,
		Year:        req.Year,
		PeriodType:  "year",
		Period:      fmt.Sprintf("%d", req.Year),
		Target:      target,
		Currency:    currency,
		MetricBasis: metricBasis,
		AmountUnit:  amountUnit,
		AllocatedBy: auth.UserID,
		Name:        fmt.Sprintf("%d 公司年度销售目标", req.Year),
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存公司销售目标失败")
		return
	}
	saveSalesTargetDashboardSetting(auth, target, metricBasis, amountUnit, currency)
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SAVE_COMPANY_SALES_TARGET", fmt.Sprintf("%d年度销售目标", req.Year), "operation.save_company_sales_target", map[string]interface{}{"year": req.Year, "target": target, "currency": currency, "metric_basis": metricBasis, "amount_unit": amountUnit}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"goal": salesGoalView(goal), "summary": buildSalesTargetSummary(auth, req.Year)})
}

func (c *CRMController) AllocateSalesTargets() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageSalesTargets(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "仅租户管理员或大屏/团队管理权限用户可分配销售目标")
		return
	}
	var req struct {
		Year         int  `json:"year"`
		ParentGoalID uint `json:"parent_goal_id"`
		Allocations  []struct {
			OwnerID           uint    `json:"owner_id"`
			Target            float64 `json:"target"`
			PlannedOrderCount float64 `json:"planned_order_count"`
		} `json:"allocations"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.Year <= 0 {
		req.Year = time.Now().Year()
	}
	companyGoal := salesCompanyGoal(auth, req.Year)
	amountUnit := salesGoalAmountUnit(companyGoal)
	if amountUnit == "" {
		amountUnit = salesTargetSettingAmountUnit(auth.TenantID)
	}
	if req.ParentGoalID == 0 && companyGoal.ID > 0 {
		req.ParentGoalID = companyGoal.ID
	}
	currency := salesTargetCurrency(auth.TenantID)
	saved := make([]map[string]interface{}, 0, len(req.Allocations))
	for _, allocation := range req.Allocations {
		if allocation.OwnerID == 0 {
			continue
		}
		if !tenantUserExists(auth.TenantID, allocation.OwnerID) {
			utils.BadRequest(c.Ctx, fmt.Sprintf("销售人员[%d]不存在", allocation.OwnerID))
			return
		}
		goal, err := upsertSalesGoal(auth, salesGoalInput{
			OwnerID:           allocation.OwnerID,
			Year:              req.Year,
			PeriodType:        "year",
			Period:            fmt.Sprintf("%d", req.Year),
			Target:            allocation.Target,
			Currency:          currency,
			MetricBasis:       salesGoalMetricBasis(companyGoal),
			AmountUnit:        amountUnit,
			ParentGoalID:      req.ParentGoalID,
			PlannedOrderCount: allocation.PlannedOrderCount,
			AllocatedBy:       auth.UserID,
			Name:              fmt.Sprintf("%d 个人年度销售目标", req.Year),
		})
		if err != nil {
			utils.Error(c.Ctx, http.StatusInternalServerError, "分配销售目标失败")
			return
		}
		saved = append(saved, salesGoalView(goal))
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "ALLOCATE_SALES_TARGETS", fmt.Sprintf("%d年度目标分配", req.Year), "operation.allocate_sales_targets", map[string]interface{}{"year": req.Year, "count": len(saved)}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"items": saved, "summary": buildSalesTargetSummary(auth, req.Year)})
}

func (c *CRMController) SaveMySalesPlan() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		OwnerID           uint     `json:"owner_id"`
		Year              int      `json:"year"`
		PeriodType        string   `json:"period_type"`
		Period            string   `json:"period"`
		Target            float64  `json:"target"`
		PlannedOrderCount float64  `json:"planned_order_count"`
		FocusCustomers    []string `json:"focus_customers"`
		FocusProducts     []string `json:"focus_products"`
		ActionPlan        string   `json:"action_plan"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.Year <= 0 {
		req.Year = time.Now().Year()
	}
	ownerID := req.OwnerID
	if ownerID == 0 {
		ownerID = auth.UserID
	}
	if ownerID != auth.UserID && !canManageSalesTargets(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "只能编辑自己的销售计划")
		return
	}
	if !tenantUserExists(auth.TenantID, ownerID) {
		utils.BadRequest(c.Ctx, "销售人员不存在")
		return
	}
	periodType := normalizeSalesPeriodType(req.PeriodType)
	if periodType == "" {
		utils.BadRequest(c.Ctx, "计划周期必须为年度、季度或月度")
		return
	}
	period := normalizeSalesPeriod(req.Year, periodType, req.Period)
	if period == "" {
		utils.BadRequest(c.Ctx, "计划周期格式不正确")
		return
	}
	parentGoal := salesOwnerYearGoal(auth, req.Year, ownerID)
	parentGoalID := parentGoal.ID
	companyGoal := salesCompanyGoal(auth, req.Year)
	amountUnit := salesGoalAmountUnit(companyGoal)
	if amountUnit == "" {
		amountUnit = salesTargetSettingAmountUnit(auth.TenantID)
	}
	currency := salesTargetCurrency(auth.TenantID)
	goal, err := upsertSalesGoal(auth, salesGoalInput{
		OwnerID:           ownerID,
		Year:              req.Year,
		PeriodType:        periodType,
		Period:            period,
		Target:            req.Target,
		Currency:          currency,
		MetricBasis:       salesGoalMetricBasis(companyGoal),
		AmountUnit:        amountUnit,
		ParentGoalID:      parentGoalID,
		PlannedOrderCount: req.PlannedOrderCount,
		FocusCustomers:    compactStringList(req.FocusCustomers),
		FocusProducts:     compactStringList(req.FocusProducts),
		ActionPlan:        strings.TrimSpace(req.ActionPlan),
		PlannedBy:         auth.UserID,
		Name:              fmt.Sprintf("%s 销售行动计划", period),
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存个人销售计划失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SAVE_MY_SALES_PLAN", fmt.Sprintf("%s销售计划", period), "operation.save_my_sales_plan", map[string]interface{}{"owner_id": ownerID, "year": req.Year, "period": period}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"goal": salesGoalView(goal), "summary": buildSalesTargetSummary(auth, req.Year)})
}

type salesTargetSummaryOptions struct {
	Month string
}

func buildSalesTargetSummary(auth authContext, year int) map[string]interface{} {
	return buildSalesTargetSummaryWithOptions(auth, year, salesTargetSummaryOptions{})
}

func buildSalesTargetSummaryWithOptions(auth authContext, year int, options salesTargetSummaryOptions) map[string]interface{} {
	if year <= 0 {
		year = time.Now().Year()
	}
	companyGoal := salesCompanyGoal(auth, year)
	setting := crmSettingValue(auth.TenantID, "sales_target", "dashboard")
	metricBasis := salesGoalMetricBasis(companyGoal)
	if metricBasis == "" {
		metricBasis = normalizeSalesMetricBasis(fmt.Sprint(setting["metric_basis"]))
	}
	amountUnit := salesGoalAmountUnit(companyGoal)
	if amountUnit == "" {
		amountUnit = normalizeSalesAmountUnit(fmt.Sprint(setting["amount_unit"]))
	}
	companyTarget := salesCompanyTargetAmount(companyGoal, setting)
	period := salesComparisonPeriod(time.Now(), year, "year", fmt.Sprintf("%d", year))
	currentAgg := salesOrderAggregateForRange(auth.TenantID, period.Start, period.End)
	lastAgg := salesOrderAggregateForRange(auth.TenantID, period.LastStart, period.LastEnd)
	actual := aggregateMetricAmount(currentAgg, metricBasis)
	companyProgress := progressRatio(actual, companyTarget)
	goals := salesGoalsByYear(auth, year)
	userMap := salesTenantUserMap(auth.TenantID, goals, currentAgg, lastAgg)
	ownerGoals := map[uint]models.CRMObject{}
	ownerPlans := map[uint][]models.CRMObject{}
	periodGoals := map[uint]map[string]models.CRMObject{}
	allocatedTarget := 0.0
	for _, goal := range goals {
		payload := payloadMap(goal.PayloadJSON)
		periodType := normalizeSalesPeriodType(fmt.Sprint(payload["period_type"]))
		if goal.OwnerID == 0 || periodType == "" {
			continue
		}
		periodKey := salesGoalPeriodKey(periodType, fmt.Sprint(payload["period"]))
		if periodKey != "" {
			if periodGoals[goal.OwnerID] == nil {
				periodGoals[goal.OwnerID] = map[string]models.CRMObject{}
			}
			if _, exists := periodGoals[goal.OwnerID][periodKey]; !exists {
				periodGoals[goal.OwnerID][periodKey] = goal
			}
		}
		if periodType == "year" {
			if _, exists := ownerGoals[goal.OwnerID]; !exists {
				ownerGoals[goal.OwnerID] = goal
				allocatedTarget += goal.Amount
			}
		}
		ownerPlans[goal.OwnerID] = append(ownerPlans[goal.OwnerID], goal)
	}
	teamRows := make([]map[string]interface{}, 0, len(userMap))
	for ownerID, user := range userMap {
		target := 0.0
		if goal, exists := ownerGoals[ownerID]; exists {
			target = goal.Amount
		}
		currentOwner := currentAgg.Owners[ownerID]
		lastOwner := lastAgg.Owners[ownerID]
		if currentOwner == nil {
			currentOwner = &ownerSalesMetrics{}
		}
		if lastOwner == nil {
			lastOwner = &ownerSalesMetrics{}
		}
		ownerActual := ownerMetricAmount(currentOwner, metricBasis)
		lastActual := ownerMetricAmount(lastOwner, metricBasis)
		teamRows = append(teamRows, map[string]interface{}{
			"owner_id":         ownerID,
			"name":             user["name"],
			"username":         user["username"],
			"target":           target,
			"actual":           ownerActual,
			"order_amount":     currentOwner.OrderAmount,
			"paid_amount":      currentOwner.PaidAmount,
			"order_count":      currentOwner.OrderCount,
			"progress":         progressRatio(ownerActual, target),
			"metric_basis":     metricBasis,
			"yoy":              salesYoY(ownerActual, lastActual),
			"last_year_amount": lastActual,
		})
	}
	sort.Slice(teamRows, func(i, j int) bool {
		leftProgress := numericValue(teamRows[i]["progress"])
		rightProgress := numericValue(teamRows[j]["progress"])
		if leftProgress == rightProgress {
			return numericValue(teamRows[i]["actual"]) > numericValue(teamRows[j]["actual"])
		}
		return leftProgress > rightProgress
	})
	users := make([]map[string]interface{}, 0, len(userMap))
	for ownerID, user := range userMap {
		row := map[string]interface{}{"id": ownerID, "name": user["name"], "username": user["username"], "status": user["status"]}
		if goal, exists := ownerGoals[ownerID]; exists {
			row["target"] = goal.Amount
		} else {
			row["target"] = 0
		}
		users = append(users, row)
	}
	sort.Slice(users, func(i, j int) bool { return fmt.Sprint(users[i]["name"]) < fmt.Sprint(users[j]["name"]) })
	allocatableUsers := salesAllocatableUsers(auth.TenantID, users)
	myGoal := map[string]interface{}{}
	if goal, exists := ownerGoals[auth.UserID]; exists {
		myGoal = salesGoalView(goal)
	}
	myPlans := make([]map[string]interface{}, 0)
	for _, goal := range ownerPlans[auth.UserID] {
		myPlans = append(myPlans, salesGoalView(goal))
	}
	sort.Slice(myPlans, func(i, j int) bool { return fmt.Sprint(myPlans[i]["period"]) < fmt.Sprint(myPlans[j]["period"]) })
	quarterPeriod := normalizeSalesPeriod(year, "quarter", "")
	monthPeriod := normalizeSalesPeriod(year, "month", options.Month)
	quarterRange := salesComparisonPeriod(time.Now(), year, "quarter", quarterPeriod)
	monthRange := salesComparisonPeriod(time.Now(), year, "month", monthPeriod)
	quarterAgg := salesOrderAggregateForRange(auth.TenantID, quarterRange.Start, quarterRange.End)
	quarterLastAgg := salesOrderAggregateForRange(auth.TenantID, quarterRange.LastStart, quarterRange.LastEnd)
	monthAgg := salesOrderAggregateForRange(auth.TenantID, monthRange.Start, monthRange.End)
	monthLastAgg := salesOrderAggregateForRange(auth.TenantID, monthRange.LastStart, monthRange.LastEnd)
	ensureAggregateUsers(userMap, quarterAgg, quarterLastAgg, monthAgg, monthLastAgg)
	memberMonthRows := salesMemberPeriodRows(userMap, ownerGoals, periodGoals, metricBasis, "month", monthPeriod, monthAgg, monthLastAgg)
	memberQuarterRows := salesMemberQuarterRows(auth.TenantID, userMap, ownerGoals, periodGoals, metricBasis, year)
	myCurrent := currentAgg.Owners[auth.UserID]
	if myCurrent == nil {
		myCurrent = &ownerSalesMetrics{}
	}
	myLast := lastAgg.Owners[auth.UserID]
	if myLast == nil {
		myLast = &ownerSalesMetrics{}
	}
	return map[string]interface{}{
		"year":                year,
		"currency":            salesTargetCurrency(auth.TenantID),
		"metric_basis":        metricBasis,
		"metric_basis_label":  salesMetricBasisLabel(metricBasis),
		"amount_unit":         amountUnit,
		"amount_unit_label":   salesAmountUnitLabel(amountUnit),
		"company_goal":        salesGoalViewWithFallback(companyGoal, auth.TenantID, year, companyTarget, metricBasis, amountUnit),
		"company_target":      companyTarget,
		"current_amount":      actual,
		"target_progress":     companyProgress,
		"allocated_target":    allocatedTarget,
		"unallocated_target":  companyTarget - allocatedTarget,
		"overallocated":       companyTarget > 0 && allocatedTarget > companyTarget,
		"order_amount":        currentAgg.OrderAmount,
		"paid_amount":         currentAgg.PaidAmount,
		"order_count":         currentAgg.OrderCount,
		"yoy":                 salesYoY(actual, aggregateMetricAmount(lastAgg, metricBasis)),
		"order_amount_yoy":    salesYoY(currentAgg.OrderAmount, lastAgg.OrderAmount),
		"paid_amount_yoy":     salesYoY(currentAgg.PaidAmount, lastAgg.PaidAmount),
		"order_count_yoy":     salesYoY(float64(currentAgg.OrderCount), float64(lastAgg.OrderCount)),
		"team_rows":           teamRows,
		"selected_month":      monthPeriod,
		"member_month_rows":   memberMonthRows,
		"member_quarter_rows": memberQuarterRows,
		"unassigned_sales":    salesUnassignedSummary(currentAgg, lastAgg, metricBasis),
		"users":               users,
		"allocatable_users":   allocatableUsers,
		"my_goal":             myGoal,
		"my_plans":            myPlans,
		"my_progress": map[string]interface{}{
			"target":       numericValue(myGoal["target"]),
			"actual":       ownerMetricAmount(myCurrent, metricBasis),
			"order_amount": myCurrent.OrderAmount,
			"paid_amount":  myCurrent.PaidAmount,
			"order_count":  myCurrent.OrderCount,
			"progress":     progressRatio(ownerMetricAmount(myCurrent, metricBasis), numericValue(myGoal["target"])),
			"yoy":          salesYoY(ownerMetricAmount(myCurrent, metricBasis), ownerMetricAmount(myLast, metricBasis)),
		},
		"current_quarter": salesPeriodSummary(auth.UserID, metricBasis, quarterPeriod, quarterAgg, quarterLastAgg),
		"current_month":   salesPeriodSummary(auth.UserID, metricBasis, monthPeriod, monthAgg, monthLastAgg),
	}
}

func salesPeriodSummary(ownerID uint, metricBasis, period string, current salesOrderAggregate, last salesOrderAggregate) map[string]interface{} {
	currentOwner := current.Owners[ownerID]
	lastOwner := last.Owners[ownerID]
	if currentOwner == nil {
		currentOwner = &ownerSalesMetrics{}
	}
	if lastOwner == nil {
		lastOwner = &ownerSalesMetrics{}
	}
	currentAmount := ownerMetricAmount(currentOwner, metricBasis)
	lastAmount := ownerMetricAmount(lastOwner, metricBasis)
	return map[string]interface{}{
		"period":       period,
		"actual":       currentAmount,
		"order_amount": currentOwner.OrderAmount,
		"paid_amount":  currentOwner.PaidAmount,
		"order_count":  currentOwner.OrderCount,
		"yoy":          salesYoY(currentAmount, lastAmount),
	}
}

func salesMemberPeriodRows(userMap map[uint]map[string]interface{}, ownerGoals map[uint]models.CRMObject, periodGoals map[uint]map[string]models.CRMObject, metricBasis, periodType, period string, current salesOrderAggregate, last salesOrderAggregate) []map[string]interface{} {
	rows := make([]map[string]interface{}, 0, len(userMap))
	for ownerID, user := range userMap {
		target := salesMemberPeriodTarget(ownerID, ownerGoals, periodGoals, periodType, period)
		currentOwner := current.Owners[ownerID]
		lastOwner := last.Owners[ownerID]
		if currentOwner == nil {
			currentOwner = &ownerSalesMetrics{}
		}
		if lastOwner == nil {
			lastOwner = &ownerSalesMetrics{}
		}
		actual := ownerMetricAmount(currentOwner, metricBasis)
		lastActual := ownerMetricAmount(lastOwner, metricBasis)
		rows = append(rows, map[string]interface{}{
			"owner_id":     ownerID,
			"name":         user["name"],
			"username":     user["username"],
			"target":       target,
			"actual":       actual,
			"order_amount": currentOwner.OrderAmount,
			"paid_amount":  currentOwner.PaidAmount,
			"order_count":  currentOwner.OrderCount,
			"progress":     progressRatio(actual, target),
			"metric_basis": metricBasis,
			"yoy":          salesYoY(actual, lastActual),
			"period":       period,
		})
	}
	sortSalesMemberRows(rows)
	return rows
}

func salesMemberQuarterRows(tenantID uint, userMap map[uint]map[string]interface{}, ownerGoals map[uint]models.CRMObject, periodGoals map[uint]map[string]models.CRMObject, metricBasis string, year int) []map[string]interface{} {
	rows := make([]map[string]interface{}, 0, 4)
	for quarter := 1; quarter <= 4; quarter++ {
		period := fmt.Sprintf("%d-Q%d", year, quarter)
		rng := salesComparisonPeriod(time.Now(), year, "quarter", period)
		current := salesOrderAggregateForRange(tenantID, rng.Start, rng.End)
		last := salesOrderAggregateForRange(tenantID, rng.LastStart, rng.LastEnd)
		ensureAggregateUsers(userMap, current, last)
		rows = append(rows, map[string]interface{}{
			"period":           period,
			"rows":             salesMemberPeriodRows(userMap, ownerGoals, periodGoals, metricBasis, "quarter", period, current, last),
			"unassigned_sales": salesUnassignedSummary(current, last, metricBasis),
		})
	}
	return rows
}

func salesMemberPeriodTarget(ownerID uint, ownerGoals map[uint]models.CRMObject, periodGoals map[uint]map[string]models.CRMObject, periodType, period string) float64 {
	key := salesGoalPeriodKey(periodType, period)
	if key != "" {
		if goal, exists := periodGoals[ownerID][key]; exists && goal.Amount > 0 {
			return goal.Amount
		}
	}
	yearTarget := 0.0
	if goal, exists := ownerGoals[ownerID]; exists {
		yearTarget = goal.Amount
	}
	if yearTarget <= 0 {
		return 0
	}
	switch normalizeSalesPeriodType(periodType) {
	case "month":
		return yearTarget / 12
	case "quarter":
		return yearTarget / 4
	default:
		return yearTarget
	}
}

func salesGoalPeriodKey(periodType, period string) string {
	periodType = normalizeSalesPeriodType(periodType)
	period = strings.ToUpper(strings.TrimSpace(period))
	if periodType == "" || period == "" {
		return ""
	}
	return periodType + ":" + period
}

func sortSalesMemberRows(rows []map[string]interface{}) {
	sort.Slice(rows, func(i, j int) bool {
		leftProgress := numericValue(rows[i]["progress"])
		rightProgress := numericValue(rows[j]["progress"])
		if leftProgress == rightProgress {
			leftActual := numericValue(rows[i]["actual"])
			rightActual := numericValue(rows[j]["actual"])
			if leftActual == rightActual {
				return fmt.Sprint(rows[i]["name"]) < fmt.Sprint(rows[j]["name"])
			}
			return leftActual > rightActual
		}
		return leftProgress > rightProgress
	})
}

func salesUnassignedSummary(current, last salesOrderAggregate, metricBasis string) map[string]interface{} {
	currentAmount := unassignedMetricAmount(current.Unassigned, metricBasis)
	lastAmount := unassignedMetricAmount(last.Unassigned, metricBasis)
	return map[string]interface{}{
		"actual":       currentAmount,
		"order_amount": current.Unassigned.OrderAmount,
		"paid_amount":  current.Unassigned.PaidAmount,
		"order_count":  current.Unassigned.OrderCount,
		"metric_basis": normalizeSalesMetricBasis(metricBasis),
		"yoy":          salesYoY(currentAmount, lastAmount),
	}
}

func salesCompanyTargetAmount(companyGoal models.CRMObject, _ map[string]interface{}) float64 {
	// 年度总目标必须来自对应年份的公司年度目标，不能从全局大屏设置跨年继承。
	if companyGoal.ID == 0 {
		return 0
	}
	return companyGoal.Amount
}

func unassignedMetricAmount(metrics unassignedSalesMetrics, metricBasis string) float64 {
	if normalizeSalesMetricBasis(metricBasis) == "order_amount" {
		return metrics.OrderAmount
	}
	return metrics.PaidAmount
}

func ensureAggregateUsers(userMap map[uint]map[string]interface{}, aggregates ...salesOrderAggregate) {
	for _, aggregate := range aggregates {
		for ownerID := range aggregate.Owners {
			ensureSalesUser(userMap, ownerID)
		}
	}
}

func salesAnalyticsDimensions(auth authContext, year int) map[string]interface{} {
	if year <= 0 {
		year = time.Now().Year()
	}
	period := salesComparisonPeriod(time.Now(), year, "year", fmt.Sprintf("%d", year))
	current := salesOrderAggregateForRange(auth.TenantID, period.Start, period.End)
	last := salesOrderAggregateForRange(auth.TenantID, period.LastStart, period.LastEnd)
	return map[string]interface{}{
		"country_sales":        salesDimensionYoYRows(current.Countries, last.Countries, "country", false, 10),
		"customer_sales":       salesDimensionYoYRows(current.Customers, last.Customers, "customer", false, 10),
		"product_sales_top":    salesDimensionYoYRows(current.Products, last.Products, "product", false, 10),
		"product_sales_bottom": salesDimensionYoYRows(current.Products, last.Products, "product", true, 10),
		"payment_summary": map[string]interface{}{
			"order_amount":     current.OrderAmount,
			"paid_amount":      current.PaidAmount,
			"order_count":      current.OrderCount,
			"order_amount_yoy": salesYoY(current.OrderAmount, last.OrderAmount),
			"paid_amount_yoy":  salesYoY(current.PaidAmount, last.PaidAmount),
			"order_count_yoy":  salesYoY(float64(current.OrderCount), float64(last.OrderCount)),
		},
	}
}

func salesDimensionYoYRows(current, last map[string]*salesDimensionMetric, labelKey string, ascending bool, limit int) []map[string]interface{} {
	rows := make([]map[string]interface{}, 0, len(current))
	for key, metric := range current {
		if metric == nil {
			continue
		}
		lastMetric := last[key]
		lastAmount := 0.0
		if lastMetric != nil {
			lastAmount = lastMetric.Amount
		}
		row := map[string]interface{}{
			labelKey:           metric.Name,
			"key":              key,
			"amount":           metric.Amount,
			"paid_amount":      metric.PaidAmount,
			"quantity":         metric.Quantity,
			"count":            metric.Count,
			"current_amount":   metric.Amount,
			"last_year_amount": lastAmount,
			"yoy":              salesYoY(metric.Amount, lastAmount),
			"yoy_delta":        metric.Amount - lastAmount,
			"yoy_rate":         0.0,
		}
		if lastAmount > 0 {
			row["yoy_rate"] = (metric.Amount - lastAmount) / lastAmount
		}
		if metric.ProductID > 0 {
			row["product_id"] = metric.ProductID
		}
		if metric.CompanyID > 0 {
			row["company_id"] = metric.CompanyID
		}
		if metric.CountryCode != "" {
			row["country_code"] = metric.CountryCode
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if ascending {
			return numericValue(rows[i]["amount"]) < numericValue(rows[j]["amount"])
		}
		return numericValue(rows[i]["amount"]) > numericValue(rows[j]["amount"])
	})
	if limit > 0 && len(rows) > limit {
		return rows[:limit]
	}
	return rows
}

func salesOrderAggregateForRange(tenantID uint, start, end time.Time) salesOrderAggregate {
	agg := salesOrderAggregate{
		Owners:    map[uint]*ownerSalesMetrics{},
		Countries: map[string]*salesDimensionMetric{},
		Customers: map[string]*salesDimensionMetric{},
		Products:  map[string]*salesDimensionMetric{},
	}
	if tenantID == 0 || start.IsZero() || end.IsZero() || !end.After(start) {
		return agg
	}
	companyNames := crmObjectNameMap(tenantID, "contacts", "company")
	companyPayloads := crmObjectPayloadMap(tenantID, "contacts", "company")
	productNames := crmObjectNameMap(tenantID, "products", "product")
	var orders []models.CRMObject
	_ = utils.DB.Where("tenant_id = ? AND domain = ? AND object_type = ? AND status IN ?", tenantID, "orders", "order", salesOrderAmountStatuses()).Find(&orders).Error
	for _, order := range orders {
		payload := payloadMap(order.PayloadJSON)
		companyID := firstPositiveUint(order.CompanyID, uintFromValue(payload["company_id"]))
		chainPayloads := []map[string]interface{}{}
		if companyID == 0 || strings.TrimSpace(companyNames[companyID]) == "" || customerCountryFromPayload(companyPayloads[companyID]) == "" {
			for _, object := range salesOrderRelatedOwners(tenantID, order) {
				objectPayload := payloadMap(object.PayloadJSON)
				chainPayloads = append(chainPayloads, objectPayload)
				companyID = firstPositiveUint(companyID, object.CompanyID, uintFromValue(objectPayload["company_id"]))
			}
		}
		ownerID := salesOrderAttributionOwnerID(tenantID, order, payload)
		createdInRange := timeInRange(order.CreatedAt, start, end)
		paidInRange := orderPaidAmountInRange(order, payload, start, end, createdInRange)
		if paidInRange > 0 {
			agg.PaidAmount += paidInRange
			if ownerID > 0 {
				owner := ensureOwnerSalesMetrics(agg.Owners, ownerID)
				owner.PaidAmount += paidInRange
			} else {
				agg.Unassigned.PaidAmount += paidInRange
			}
		}
		if !createdInRange {
			continue
		}
		agg.OrderAmount += order.Amount
		agg.OrderCount++
		if ownerID > 0 {
			owner := ensureOwnerSalesMetrics(agg.Owners, ownerID)
			owner.OrderAmount += order.Amount
			owner.OrderCount++
		} else {
			agg.Unassigned.OrderAmount += order.Amount
			agg.Unassigned.OrderCount++
		}
		country := salesOrderCountry(payload, companyPayloads[companyID], chainPayloads...)
		countryBucket := ensureDimensionMetric(agg.Countries, normalizeDimensionKey(country), country)
		countryBucket.Amount += order.Amount
		countryBucket.PaidAmount += paidInRange
		countryBucket.Count++
		customerName := salesOrderCustomerName(order, payload, companyNames, chainPayloads...)
		customerKey := fmt.Sprintf("company:%d", companyID)
		if companyID == 0 {
			customerKey = normalizeDimensionKey(customerName)
		}
		customerBucket := ensureDimensionMetric(agg.Customers, customerKey, customerName)
		customerBucket.CompanyID = companyID
		customerBucket.Amount += order.Amount
		customerBucket.PaidAmount += paidInRange
		customerBucket.Count++
		for _, raw := range interfaceList(payload["line_items"]) {
			line, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			productID := uintFromValue(line["product_id"])
			productName := strings.TrimSpace(firstPayloadText(line, "product_name", "name", "sku"))
			if productName == "" && productID > 0 {
				productName = productNames[productID]
			}
			if productName == "" {
				productName = "未命名产品"
			}
			quantity := numericValue(line["quantity"])
			if quantity <= 0 {
				quantity = 1
			}
			amount := numericValue(line["amount"])
			if amount <= 0 {
				price := numericValue(line["unit_price"])
				if price <= 0 {
					price = numericValue(line["price"])
				}
				amount = quantity * price
			}
			if amount <= 0 {
				continue
			}
			productKey := fmt.Sprintf("product:%d", productID)
			if productID == 0 {
				productKey = normalizeDimensionKey(productName)
			}
			productBucket := ensureDimensionMetric(agg.Products, productKey, productName)
			productBucket.ProductID = productID
			productBucket.Amount += amount
			productBucket.Quantity += quantity
			productBucket.Count++
		}
	}
	return agg
}

func salesOrderAmountStatuses() []string {
	return []string{"confirmed", "partial_shipped", "shipped", "completed"}
}

func salesOrderAmountStatus(status string) bool {
	normalized := normalizeOrderStatus(status)
	for _, allowed := range salesOrderAmountStatuses() {
		if normalized == allowed {
			return true
		}
	}
	return false
}

func orderPaidAmountInRange(order models.CRMObject, payload map[string]interface{}, start, end time.Time, orderCreatedInRange bool) float64 {
	payments := interfaceList(payload["payments"])
	total := 0.0
	for _, raw := range payments {
		payment, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		paidAmount := numericValue(payment["paid_amount"])
		if paidAmount <= 0 {
			paidAmount = numericValue(payment["amount"])
		}
		if paidAmount <= 0 {
			continue
		}
		paidAt, _ := parseOptionalTime(firstPayloadText(payment, "paid_at", "payment_at", "created_at"))
		if paidAt == nil {
			if orderCreatedInRange {
				total += paidAmount
			}
			continue
		}
		if timeInRange(*paidAt, start, end) {
			total += paidAmount
		}
	}
	if len(payments) > 0 {
		return total
	}
	paidAmount := numericValue(payload["paid_amount"])
	if paidAmount <= 0 {
		return 0
	}
	paidAt, _ := parseOptionalTime(firstPayloadText(payload, "paid_at", "payment_at"))
	if paidAt != nil {
		if timeInRange(*paidAt, start, end) {
			return paidAmount
		}
		return 0
	}
	if orderCreatedInRange {
		return paidAmount
	}
	return 0
}

func salesOrderAttributionOwnerID(tenantID uint, order models.CRMObject, payload map[string]interface{}) uint {
	if ownerID := order.OwnerID; ownerID > 0 {
		return ownerID
	}
	if ownerID := salesOwnerIDFromPayload(payload); ownerID > 0 {
		return ownerID
	}
	if ownerID := salesOwnerIDFromOrderChain(tenantID, order); ownerID > 0 {
		return ownerID
	}
	if order.CreatedBy > 0 && salesUserID(tenantID, order.CreatedBy) {
		return order.CreatedBy
	}
	return 0
}

func salesOwnerIDFromPayload(payload map[string]interface{}) uint {
	for _, key := range []string{"sales_owner_id", "sales_user_id", "salesperson_id", "sales_rep_id", "owner_id", "account_owner_id"} {
		if ownerID := uintFromValue(payload[key]); ownerID > 0 {
			return ownerID
		}
	}
	return 0
}

func salesOwnerIDFromOrderChain(tenantID uint, order models.CRMObject) uint {
	if tenantID == 0 || order.ID == 0 || utils.DB == nil {
		return 0
	}
	for _, object := range salesOrderRelatedOwners(tenantID, order) {
		if object.OwnerID > 0 {
			return object.OwnerID
		}
		payload := payloadMap(object.PayloadJSON)
		if ownerID := salesOwnerIDFromPayload(payload); ownerID > 0 {
			return ownerID
		}
		if object.CreatedBy > 0 && salesUserID(tenantID, object.CreatedBy) {
			return object.CreatedBy
		}
	}
	return 0
}

func salesOrderRelatedOwners(tenantID uint, order models.CRMObject) []models.CRMObject {
	if tenantID == 0 || order.ID == 0 || utils.DB == nil {
		return []models.CRMObject{}
	}
	idsByDomain := map[string]map[uint]bool{
		"deals":         {},
		"opportunities": {},
	}
	if order.DealID > 0 {
		idsByDomain["deals"][order.DealID] = true
	}
	var relations []models.CRMRelation
	_ = utils.DB.Where("tenant_id = ? AND ((target_type = ? AND target_id = ?) OR (source_type = ? AND source_id = ?))", tenantID, "orders", order.ID, "orders", order.ID).Find(&relations).Error
	for _, relation := range relations {
		if relation.TargetType == "orders" && relation.TargetID == order.ID {
			addSalesOrderRelatedID(idsByDomain, relation.SourceType, relation.SourceID)
			continue
		}
		if relation.SourceType == "orders" && relation.SourceID == order.ID {
			addSalesOrderRelatedID(idsByDomain, relation.TargetType, relation.TargetID)
		}
	}
	items := make([]models.CRMObject, 0)
	for _, domain := range []string{"deals", "opportunities"} {
		ids := make([]uint, 0, len(idsByDomain[domain]))
		for id := range idsByDomain[domain] {
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			continue
		}
		var objects []models.CRMObject
		_ = utils.DB.Where("tenant_id = ? AND domain = ? AND id IN ?", tenantID, domain, ids).Order("id asc").Find(&objects).Error
		items = append(items, objects...)
	}
	return items
}

func addSalesOrderRelatedID(idsByDomain map[string]map[uint]bool, domain string, id uint) {
	if id == 0 {
		return
	}
	if _, ok := idsByDomain[domain]; ok {
		idsByDomain[domain][id] = true
	}
}

func salesUserID(tenantID uint, userID uint) bool {
	if tenantID == 0 || userID == 0 {
		return false
	}
	return salesRoleUserIDs(tenantID)[userID]
}

func salesOrderCountry(payload map[string]interface{}, companyPayload map[string]interface{}, chainPayloads ...map[string]interface{}) string {
	country := customerCountryFromPayload(payload)
	if country == "" {
		country = customerCountryFromPayload(companyPayload)
	}
	if country == "" {
		country = customerCountryFromPayloads(chainPayloads)
	}
	if country == "" {
		country = "未标注国家"
	}
	return country
}

func salesOrderCustomerName(order models.CRMObject, payload map[string]interface{}, companyNames map[uint]string, chainPayloads ...map[string]interface{}) string {
	companyID := firstPositiveUint(order.CompanyID, uintFromValue(payload["company_id"]))
	for _, chainPayload := range chainPayloads {
		companyID = firstPositiveUint(companyID, uintFromValue(chainPayload["company_id"]))
	}
	if companyID > 0 && strings.TrimSpace(companyNames[companyID]) != "" {
		return companyNames[companyID]
	}
	name := firstPayloadText(payload, "company", "company_name", "customer", "customer_name", "account_name")
	if name == "" {
		name = firstPayloadTextFromPayloads(chainPayloads, "company", "company_name", "customer", "customer_name", "account_name")
	}
	if name == "" {
		name = order.Name
	}
	if name == "" {
		name = "未标注客户"
	}
	return name
}

func crmObjectPayloadMap(tenantID uint, domain, objectType string) map[uint]map[string]interface{} {
	items := map[uint]map[string]interface{}{}
	var objects []models.CRMObject
	_ = utils.DB.Select("id, payload_json").Where("tenant_id = ? AND domain = ? AND object_type = ?", tenantID, domain, objectType).Find(&objects).Error
	for _, object := range objects {
		items[object.ID] = payloadMap(object.PayloadJSON)
	}
	return items
}

func crmObjectNameMap(tenantID uint, domain, objectType string) map[uint]string {
	items := map[uint]string{}
	var objects []models.CRMObject
	_ = utils.DB.Select("id, name").Where("tenant_id = ? AND domain = ? AND object_type = ?", tenantID, domain, objectType).Find(&objects).Error
	for _, object := range objects {
		items[object.ID] = object.Name
	}
	return items
}

func ensureOwnerSalesMetrics(items map[uint]*ownerSalesMetrics, ownerID uint) *ownerSalesMetrics {
	if existing, ok := items[ownerID]; ok {
		return existing
	}
	items[ownerID] = &ownerSalesMetrics{}
	return items[ownerID]
}

func ensureDimensionMetric(items map[string]*salesDimensionMetric, key, name string) *salesDimensionMetric {
	key = normalizeDimensionKey(key)
	if key == "" {
		key = "unknown"
	}
	if existing, ok := items[key]; ok {
		return existing
	}
	if strings.TrimSpace(name) == "" {
		name = "未标注"
	}
	items[key] = &salesDimensionMetric{Key: key, Name: name}
	return items[key]
}

func normalizeDimensionKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "<nil>" {
		return "unknown"
	}
	return value
}

func timeInRange(value, start, end time.Time) bool {
	return !value.Before(start) && value.Before(end)
}

func salesComparisonPeriod(now time.Time, year int, periodType, period string) salesComparisonRange {
	if now.IsZero() {
		now = time.Now()
	}
	loc := now.Location()
	periodType = normalizeSalesPeriodType(periodType)
	if periodType == "" {
		periodType = "year"
	}
	period = normalizeSalesPeriod(year, periodType, period)
	start, end := salesPeriodBounds(loc, year, periodType, period)
	currentEnd := end
	if !now.Before(start) && now.Before(end) {
		currentEnd = now
	}
	lastStart := start.AddDate(-1, 0, 0)
	lastEnd := end.AddDate(-1, 0, 0)
	if currentEnd.Before(end) {
		lastEnd = currentEnd.AddDate(-1, 0, 0)
	}
	return salesComparisonRange{PeriodType: periodType, Period: period, Start: start, End: currentEnd, LastStart: lastStart, LastEnd: lastEnd}
}

func salesPeriodBounds(loc *time.Location, year int, periodType, period string) (time.Time, time.Time) {
	if loc == nil {
		loc = time.Local
	}
	switch periodType {
	case "month":
		month := time.Now().Month()
		if len(period) >= 7 {
			if parsed, err := strconv.Atoi(period[5:7]); err == nil && parsed >= 1 && parsed <= 12 {
				month = time.Month(parsed)
			}
		}
		start := time.Date(year, month, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 1, 0)
	case "quarter":
		quarter := currentQuarter(time.Now())
		if idx := strings.Index(period, "Q"); idx >= 0 && idx < len(period)-1 {
			if parsed, err := strconv.Atoi(period[idx+1:]); err == nil && parsed >= 1 && parsed <= 4 {
				quarter = parsed
			}
		}
		startMonth := time.Month((quarter-1)*3 + 1)
		start := time.Date(year, startMonth, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 3, 0)
	default:
		start := time.Date(year, 1, 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(1, 0, 0)
	}
}

func normalizeSalesPeriodType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "year", "annual", "yearly", "年度":
		return "year"
	case "quarter", "quarterly", "季度":
		return "quarter"
	case "month", "monthly", "月度":
		return "month"
	default:
		return ""
	}
}

func normalizeSalesPeriod(year int, periodType, period string) string {
	if year <= 0 {
		year = time.Now().Year()
	}
	period = strings.ToUpper(strings.TrimSpace(period))
	switch periodType {
	case "year":
		return fmt.Sprintf("%d", year)
	case "quarter":
		quarter := currentQuarter(time.Now())
		if strings.Contains(period, "Q") {
			parts := strings.Split(period, "Q")
			if len(parts) == 2 {
				if parsed, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && parsed >= 1 && parsed <= 4 {
					quarter = parsed
				}
			}
		} else if parsed, err := strconv.Atoi(period); err == nil && parsed >= 1 && parsed <= 4 {
			quarter = parsed
		}
		return fmt.Sprintf("%d-Q%d", year, quarter)
	case "month":
		month := int(time.Now().Month())
		if len(period) >= 7 {
			if parsed, err := strconv.Atoi(period[5:7]); err == nil && parsed >= 1 && parsed <= 12 {
				month = parsed
			}
		} else if parsed, err := strconv.Atoi(period); err == nil && parsed >= 1 && parsed <= 12 {
			month = parsed
		}
		return fmt.Sprintf("%d-%02d", year, month)
	default:
		return ""
	}
}

func currentQuarter(now time.Time) int {
	return int(now.Month()-1)/3 + 1
}

func queryYear(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().Year()
	}
	year, err := strconv.Atoi(value)
	if err != nil || year < 2000 || year > 2100 {
		return time.Now().Year()
	}
	return year
}

func normalizeSalesMetricBasis(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "order", "orders", "order_amount", "amount", "下单金额":
		return "order_amount"
	default:
		return "payment"
	}
}

func salesMetricBasisLabel(value string) string {
	if normalizeSalesMetricBasis(value) == "order_amount" {
		return "下单金额"
	}
	return "回款金额"
}

func normalizeSalesAmountUnit(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "wan", "ten_thousand", "ten-thousand", "10000", "万元", "万":
		return "wan"
	case "yi", "hundred_million", "hundred-million", "100000000", "亿元", "亿":
		return "yi"
	default:
		return "yuan"
	}
}

func normalizeSalesCurrency(value string) string {
	text := strings.ToUpper(strings.TrimSpace(value))
	if len(text) == 3 {
		valid := true
		for _, char := range text {
			if char < 'A' || char > 'Z' {
				valid = false
				break
			}
		}
		if valid {
			return text
		}
	}
	return "USD"
}

func salesCurrencyFromValue(value interface{}) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return ""
	}
	return normalizeSalesCurrency(text)
}

func salesAmountUnitFromValue(value interface{}) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return ""
	}
	return normalizeSalesAmountUnit(text)
}

func salesAmountUnitLabel(value string) string {
	switch normalizeSalesAmountUnit(value) {
	case "wan":
		return "万元"
	case "yi":
		return "亿元"
	default:
		return "元"
	}
}

func aggregateMetricAmount(agg salesOrderAggregate, metricBasis string) float64 {
	if normalizeSalesMetricBasis(metricBasis) == "order_amount" {
		return agg.OrderAmount
	}
	return agg.PaidAmount
}

func ownerMetricAmount(metrics *ownerSalesMetrics, metricBasis string) float64 {
	if metrics == nil {
		return 0
	}
	if normalizeSalesMetricBasis(metricBasis) == "order_amount" {
		return metrics.OrderAmount
	}
	return metrics.PaidAmount
}

func progressRatio(actual, target float64) float64 {
	if target <= 0 {
		return 0
	}
	return actual / target
}

func salesYoY(current, last float64) map[string]interface{} {
	result := map[string]interface{}{
		"current_amount":   current,
		"last_year_amount": last,
		"yoy_delta":        current - last,
		"yoy_rate":         0.0,
		"has_last_year":    last > 0,
	}
	if last > 0 {
		result["yoy_rate"] = (current - last) / last
	}
	return result
}

func salesCompanyGoal(auth authContext, year int) models.CRMObject {
	goals := salesGoalsByYear(auth, year)
	for _, goal := range goals {
		payload := payloadMap(goal.PayloadJSON)
		if goal.OwnerID == 0 && normalizeSalesPeriodType(fmt.Sprint(payload["period_type"])) == "year" {
			return goal
		}
	}
	return models.CRMObject{}
}

func salesOwnerYearGoal(auth authContext, year int, ownerID uint) models.CRMObject {
	goals := salesGoalsByYear(auth, year)
	for _, goal := range goals {
		payload := payloadMap(goal.PayloadJSON)
		if goal.OwnerID == ownerID && normalizeSalesPeriodType(fmt.Sprint(payload["period_type"])) == "year" {
			return goal
		}
	}
	return models.CRMObject{}
}

func salesGoalsByYear(auth authContext, year int) []models.CRMObject {
	var goals []models.CRMObject
	_ = utils.DB.Where("tenant_id = ? AND domain = ? AND object_type = ? AND status <> ?", auth.TenantID, salesGoalDomain, salesGoalObjectType, "archived").Order("owner_id asc, id asc").Find(&goals).Error
	filtered := make([]models.CRMObject, 0, len(goals))
	for _, goal := range goals {
		payload := payloadMap(goal.PayloadJSON)
		if int(numericValue(payload["year"])) == year {
			filtered = append(filtered, goal)
		}
	}
	return filtered
}

func findSalesGoal(auth authContext, ownerID uint, year int, periodType, period string) (models.CRMObject, bool, error) {
	var goals []models.CRMObject
	err := utils.DB.Where("tenant_id = ? AND domain = ? AND object_type = ? AND owner_id = ? AND status <> ?", auth.TenantID, salesGoalDomain, salesGoalObjectType, ownerID, "archived").Order("id asc").Find(&goals).Error
	if err != nil {
		return models.CRMObject{}, false, err
	}
	for _, goal := range goals {
		payload := payloadMap(goal.PayloadJSON)
		if int(numericValue(payload["year"])) == year &&
			normalizeSalesPeriodType(fmt.Sprint(payload["period_type"])) == periodType &&
			strings.EqualFold(strings.TrimSpace(fmt.Sprint(payload["period"])), period) {
			return goal, true, nil
		}
	}
	return models.CRMObject{}, false, nil
}

func upsertSalesGoal(auth authContext, input salesGoalInput) (models.CRMObject, error) {
	input.PeriodType = normalizeSalesPeriodType(input.PeriodType)
	if input.PeriodType == "" {
		input.PeriodType = "year"
	}
	input.Period = normalizeSalesPeriod(input.Year, input.PeriodType, input.Period)
	input.MetricBasis = normalizeSalesMetricBasis(input.MetricBasis)
	input.Currency = normalizeSalesCurrency(input.Currency)
	start, end := salesPeriodBounds(time.Now().Location(), input.Year, input.PeriodType, input.Period)
	existing, found, err := findSalesGoal(auth, input.OwnerID, input.Year, input.PeriodType, input.Period)
	if err != nil {
		return models.CRMObject{}, err
	}
	payload := map[string]interface{}{}
	if found {
		payload = payloadMap(existing.PayloadJSON)
	}
	payload["year"] = input.Year
	payload["period_type"] = input.PeriodType
	payload["period"] = input.Period
	payload["target"] = input.Target
	payload["metric_basis"] = input.MetricBasis
	payload["amount_unit"] = normalizeSalesAmountUnit(input.AmountUnit)
	payload["parent_goal_id"] = input.ParentGoalID
	if input.PlannedOrderCount > 0 {
		payload["planned_order_count"] = input.PlannedOrderCount
	}
	if len(input.FocusCustomers) > 0 {
		payload["focus_customers"] = input.FocusCustomers
	}
	if len(input.FocusProducts) > 0 {
		payload["focus_products"] = input.FocusProducts
	}
	if strings.TrimSpace(input.ActionPlan) != "" {
		payload["action_plan"] = strings.TrimSpace(input.ActionPlan)
	}
	if input.AllocatedBy > 0 {
		payload["allocated_by"] = input.AllocatedBy
	}
	if input.PlannedBy > 0 {
		payload["planned_by"] = input.PlannedBy
		payload["planned_at"] = time.Now().Format(time.RFC3339)
	}
	if input.Name == "" {
		input.Name = fmt.Sprintf("%s 销售目标", input.Period)
	}
	codeOwner := "company"
	if input.OwnerID > 0 {
		codeOwner = fmt.Sprintf("owner-%d", input.OwnerID)
	}
	code := fmt.Sprintf("SALES-GOAL-%s-%s", input.Period, codeOwner)
	if found {
		existing.Name = input.Name
		existing.Code = code
		existing.Status = "active"
		existing.Amount = input.Target
		existing.Currency = input.Currency
		existing.StartAt = &start
		existing.EndAt = &end
		existing.PayloadJSON = toJSONString(payload)
		existing.UpdatedBy = auth.UserID
		return existing, utils.DB.Save(&existing).Error
	}
	goal := models.CRMObject{
		TenantModel: models.TenantModel{TenantID: auth.TenantID},
		Domain:      salesGoalDomain,
		ObjectType:  salesGoalObjectType,
		Name:        input.Name,
		Code:        code,
		Status:      "active",
		OwnerID:     input.OwnerID,
		Amount:      input.Target,
		Currency:    input.Currency,
		StartAt:     &start,
		EndAt:       &end,
		PayloadJSON: toJSONString(payload),
		CreatedBy:   auth.UserID,
		UpdatedBy:   auth.UserID,
	}
	return goal, utils.DB.Create(&goal).Error
}

func saveSalesTargetDashboardSetting(auth authContext, target float64, metricBasis string, amountUnit string, currency string) {
	setting := models.CRMSetting{
		TenantModel: models.TenantModel{TenantID: auth.TenantID},
		Scope:       "sales_target",
		Key:         "dashboard",
		ValueJSON:   toJSONString(map[string]interface{}{"target": target, "period": "year", "currency": normalizeSalesCurrency(currency), "metric_basis": normalizeSalesMetricBasis(metricBasis), "amount_unit": normalizeSalesAmountUnit(amountUnit)}),
		UpdatedBy:   auth.UserID,
	}
	_ = utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "sales_target", "dashboard").Assign(setting).FirstOrCreate(&setting).Error
}

func salesGoalMetricBasis(goal models.CRMObject) string {
	payload := payloadMap(goal.PayloadJSON)
	return normalizeSalesMetricBasis(fmt.Sprint(payload["metric_basis"]))
}

func salesGoalAmountUnit(goal models.CRMObject) string {
	payload := payloadMap(goal.PayloadJSON)
	return salesAmountUnitFromValue(payload["amount_unit"])
}

func salesTargetSettingMetricBasis(tenantID uint) string {
	setting := crmSettingValue(tenantID, "sales_target", "dashboard")
	return normalizeSalesMetricBasis(fmt.Sprint(setting["metric_basis"]))
}

func salesTargetSettingAmountUnit(tenantID uint) string {
	setting := crmSettingValue(tenantID, "sales_target", "dashboard")
	if unit := salesAmountUnitFromValue(setting["amount_unit"]); unit != "" {
		return unit
	}
	return "yuan"
}

func salesTargetCurrency(tenantID uint) string {
	enterprise := crmSettingValue(tenantID, "company", "enterprise")
	if currency := salesCurrencyFromValue(enterprise["currency"]); currency != "" {
		return currency
	}
	setting := crmSettingValue(tenantID, "sales_target", "dashboard")
	if currency := salesCurrencyFromValue(setting["currency"]); currency != "" {
		return currency
	}
	return "USD"
}

func salesGoalView(goal models.CRMObject) map[string]interface{} {
	if goal.ID == 0 {
		return map[string]interface{}{}
	}
	payload := payloadMap(goal.PayloadJSON)
	amountUnit := salesAmountUnitFromValue(payload["amount_unit"])
	if amountUnit == "" {
		amountUnit = "yuan"
	}
	view := map[string]interface{}{
		"id":                  goal.ID,
		"name":                goal.Name,
		"owner_id":            goal.OwnerID,
		"target":              goal.Amount,
		"amount":              goal.Amount,
		"currency":            goal.Currency,
		"status":              goal.Status,
		"year":                int(numericValue(payload["year"])),
		"period_type":         normalizeSalesPeriodType(fmt.Sprint(payload["period_type"])),
		"period":              fmt.Sprint(payload["period"]),
		"metric_basis":        normalizeSalesMetricBasis(fmt.Sprint(payload["metric_basis"])),
		"metric_basis_label":  salesMetricBasisLabel(fmt.Sprint(payload["metric_basis"])),
		"amount_unit":         amountUnit,
		"amount_unit_label":   salesAmountUnitLabel(amountUnit),
		"parent_goal_id":      uintFromValue(payload["parent_goal_id"]),
		"planned_order_count": numericValue(payload["planned_order_count"]),
		"focus_customers":     payload["focus_customers"],
		"focus_products":      payload["focus_products"],
		"action_plan":         strings.TrimSpace(fmt.Sprint(payload["action_plan"])),
		"allocated_by":        uintFromValue(payload["allocated_by"]),
		"planned_by":          uintFromValue(payload["planned_by"]),
		"planned_at":          strings.TrimSpace(fmt.Sprint(payload["planned_at"])),
		"created_at":          goal.CreatedAt,
		"updated_at":          goal.UpdatedAt,
	}
	return view
}

func salesGoalViewWithFallback(goal models.CRMObject, tenantID uint, year int, target float64, metricBasis string, amountUnit string) map[string]interface{} {
	view := salesGoalView(goal)
	if len(view) == 0 {
		currency := salesTargetCurrency(tenantID)
		return map[string]interface{}{
			"id":                 0,
			"name":               fmt.Sprintf("%d 公司年度销售目标", year),
			"owner_id":           0,
			"target":             target,
			"amount":             target,
			"currency":           currency,
			"year":               year,
			"period_type":        "year",
			"period":             fmt.Sprintf("%d", year),
			"metric_basis":       normalizeSalesMetricBasis(metricBasis),
			"metric_basis_label": salesMetricBasisLabel(metricBasis),
			"amount_unit":        normalizeSalesAmountUnit(amountUnit),
			"amount_unit_label":  salesAmountUnitLabel(amountUnit),
		}
	}
	if numericValue(view["target"]) <= 0 && target > 0 {
		view["target"] = target
		view["amount"] = target
	}
	view["metric_basis"] = normalizeSalesMetricBasis(metricBasis)
	view["metric_basis_label"] = salesMetricBasisLabel(metricBasis)
	if salesGoalAmountUnit(goal) == "" {
		view["amount_unit"] = normalizeSalesAmountUnit(amountUnit)
		view["amount_unit_label"] = salesAmountUnitLabel(amountUnit)
	}
	return view
}

func salesTenantUserMap(tenantID uint, goals []models.CRMObject, currentAgg, lastAgg salesOrderAggregate) map[uint]map[string]interface{} {
	items := map[uint]map[string]interface{}{}
	var users []models.User
	_ = utils.DB.Where("tenant_id = ? AND account_type <> ? AND (status IS NULL OR status NOT IN ?)", tenantID, "super_admin", []string{"disabled", "deleted"}).Order("name asc, username asc").Find(&users).Error
	for _, user := range users {
		name := strings.TrimSpace(user.Name)
		if name == "" {
			name = user.Username
		}
		if name == "" {
			name = fmt.Sprintf("销售#%d", user.ID)
		}
		items[user.ID] = map[string]interface{}{"id": user.ID, "name": name, "username": user.Username, "status": user.Status}
	}
	for _, goal := range goals {
		if goal.OwnerID > 0 {
			ensureSalesUser(items, goal.OwnerID)
		}
	}
	for ownerID := range currentAgg.Owners {
		ensureSalesUser(items, ownerID)
	}
	for ownerID := range lastAgg.Owners {
		ensureSalesUser(items, ownerID)
	}
	return items
}

func salesAllocatableUsers(tenantID uint, fallback []map[string]interface{}) []map[string]interface{} {
	items := make([]map[string]interface{}, 0)
	seen := map[uint]bool{}
	salesUserIDs := salesRoleUserIDs(tenantID)
	add := func(id uint, name, username, status, source string) {
		if id == 0 || seen[id] {
			return
		}
		name = strings.TrimSpace(name)
		username = strings.TrimSpace(username)
		if name == "" {
			name = username
		}
		if name == "" {
			name = fmt.Sprintf("销售#%d", id)
		}
		seen[id] = true
		items = append(items, map[string]interface{}{"id": id, "name": name, "username": username, "status": status, "source": source})
	}
	var users []models.User
	_ = utils.DB.Where("tenant_id = ? AND account_type <> ? AND (status IS NULL OR status NOT IN ?)", tenantID, "super_admin", []string{"disabled", "deleted"}).Order("name asc, username asc").Find(&users).Error
	for _, user := range users {
		if !salesUserIDs[user.ID] {
			continue
		}
		add(user.ID, user.Name, user.Username, user.Status, "account")
	}
	var teamMembers []models.CRMObject
	_ = utils.DB.Where("tenant_id = ? AND domain = ? AND object_type = ? AND status <> ?", tenantID, "teams", "team_member", "inactive").Order("name asc").Find(&teamMembers).Error
	for _, member := range teamMembers {
		payload := payloadMap(member.PayloadJSON)
		ownerID := firstPositiveUint(member.OwnerID, uintFromValue(payload["user_id"]), uintFromValue(payload["owner_id"]))
		if !salesUserIDs[ownerID] {
			continue
		}
		add(ownerID, member.Name, firstPayloadText(payload, "username", "email"), member.Status, "team_member")
	}
	for _, row := range fallback {
		ownerID := uintFromValue(row["id"])
		if !salesUserIDs[ownerID] {
			continue
		}
		add(ownerID, fmt.Sprint(row["name"]), fmt.Sprint(row["username"]), fmt.Sprint(row["status"]), "summary")
	}
	sort.Slice(items, func(i, j int) bool { return fmt.Sprint(items[i]["name"]) < fmt.Sprint(items[j]["name"]) })
	return items
}

func salesRoleUserIDs(tenantID uint) map[uint]bool {
	roleIDs := salesRoleIDs(tenantID)
	userIDs := map[uint]bool{}
	if len(roleIDs) == 0 {
		return userIDs
	}
	var users []models.User
	_ = utils.DB.Where("tenant_id = ? AND (role_id IN ? OR id IN (SELECT user_id FROM user_roles WHERE role_id IN ? AND deleted_at IS NULL)) AND (status IS NULL OR status NOT IN ?)", tenantID, roleIDs, roleIDs, []string{"disabled", "deleted"}).Find(&users).Error
	for _, user := range users {
		userIDs[user.ID] = true
	}
	return userIDs
}

func salesRoleIDs(tenantID uint) []uint {
	var roles []models.Role
	_ = utils.DB.Where("tenant_id = ? AND status <> ?", tenantID, "disabled").Find(&roles).Error
	ids := make([]uint, 0)
	for _, role := range roles {
		if isSalesRole(role) {
			ids = append(ids, role.ID)
		}
	}
	return ids
}

func isSalesRole(role models.Role) bool {
	name := strings.TrimSpace(role.Name)
	code := strings.ToLower(strings.TrimSpace(role.Code))
	code = strings.ReplaceAll(code, "-", "_")
	return salesRoleNames[name] ||
		strings.Contains(name, "销售员") ||
		strings.Contains(name, "销售代表") ||
		salesRoleCodes[code] ||
		strings.Contains(code, "sales")
}

func ensureSalesUser(items map[uint]map[string]interface{}, ownerID uint) {
	if ownerID == 0 {
		return
	}
	if _, exists := items[ownerID]; exists {
		return
	}
	items[ownerID] = map[string]interface{}{"id": ownerID, "name": fmt.Sprintf("销售#%d", ownerID), "username": "", "status": ""}
}

func compactStringList(values []string) []string {
	items := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		items = append(items, value)
	}
	return items
}

func canManageSalesTargets(auth authContext) bool {
	return auth.IsSuperAdmin() || auth.IsTenantAdmin() ||
		authHasPermissionAny(auth, "dashboards:update", "dashboards:manage", "teams:update", "teams:manage") ||
		isCRMModuleManager(auth, "dashboards") || isCRMModuleManager(auth, "teams")
}

func salesTargetProgress(auth authContext, actualAmount float64) map[string]interface{} {
	year := time.Now().Year()
	summary := buildSalesTargetSummary(auth, year)
	target := numericValue(summary["company_target"])
	actual := numericValue(summary["current_amount"])
	if actual <= 0 && actualAmount > 0 {
		actual = actualAmount
	}
	return map[string]interface{}{
		"target":       target,
		"actual":       actual,
		"progress":     progressRatio(actual, target),
		"metric_basis": summary["metric_basis"],
		"yoy":          summary["yoy"],
	}
}

func salesTargetLeaderboard(auth authContext, year int) []map[string]interface{} {
	summary := buildSalesTargetSummary(auth, year)
	rows, _ := summary["team_rows"].([]map[string]interface{})
	if len(rows) > 8 {
		return rows[:8]
	}
	return rows
}

func salesTargetSummaryForCockpit(auth authContext, year int) map[string]interface{} {
	return buildSalesTargetSummary(auth, year)
}
