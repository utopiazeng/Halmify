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

var leadAutomationSchedulerStop chan struct{}

type leadSLARecycleRequest struct {
	Hours  int    `json:"hours"`
	Limit  int    `json:"limit"`
	Reason string `json:"reason"`
	DryRun bool   `json:"dry_run"`
	Force  bool   `json:"force"`
}

type leadNurtureRequest struct {
	TemplateKey string `json:"template_key"`
	Reason      string `json:"reason"`
	Exit        bool   `json:"exit"`
}

type leadAutomationResult struct {
	Scanned     int                      `json:"scanned"`
	Recycled    []uint                   `json:"recycled"`
	Nurtured    []uint                   `json:"nurtured"`
	Created     []models.CRMObject       `json:"created"`
	DryRunItems []models.CRMObject       `json:"dry_run_items,omitempty"`
	Skipped     []map[string]interface{} `json:"skipped,omitempty"`
	Cutoff      time.Time                `json:"cutoff"`
	ExecutedAt  time.Time                `json:"executed_at"`
	NextRunAt   time.Time                `json:"next_run_at,omitempty"`
	Schedule    map[string]interface{}   `json:"schedule,omitempty"`
}

func (c *CRMController) ListLeadViews() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	presets := defaultLeadViews()
	var setting models.CRMSetting
	custom := []map[string]interface{}{}
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "lead_view", "custom").First(&setting).Error; err == nil {
		_ = json.Unmarshal([]byte(setting.ValueJSON), &custom)
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": append(presets, custom...), "tenant_id": auth.TenantID})
}

func (c *CRMController) SaveLeadView() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		Name    string                 `json:"name"`
		Filters map[string]interface{} `json:"filters"`
		Shared  bool                   `json:"shared"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		utils.BadRequest(c.Ctx, "视图名称不能为空")
		return
	}
	if req.Filters == nil {
		req.Filters = map[string]interface{}{}
	}
	var existing models.CRMSetting
	views := []map[string]interface{}{}
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "lead_view", "custom").First(&existing).Error; err == nil {
		_ = json.Unmarshal([]byte(existing.ValueJSON), &views)
	}
	next := map[string]interface{}{"key": fmt.Sprintf("custom_%d", time.Now().UnixNano()), "name": req.Name, "filters": req.Filters, "shared": req.Shared, "created_by": auth.UserID, "created_at": time.Now().Format(time.RFC3339)}
	views = append(views, next)
	setting := models.CRMSetting{TenantModel: models.TenantModel{TenantID: auth.TenantID}, Scope: "lead_view", Key: "custom", ValueJSON: toJSONString(views), UpdatedBy: auth.UserID}
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "lead_view", "custom").Assign(setting).FirstOrCreate(&setting).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存线索视图失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SAVE_LEAD_VIEW", "线索视图", "operation.save_lead_view", map[string]interface{}{"name": req.Name}, "success", "")
	utils.Success(c.Ctx, next)
}

func (c *CRMController) LeadDashboard() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	statuses := []string{"new", "pending_assign", "following", "mql", "sdr_following", "sql", "nurturing", "converted", "invalid"}
	funnel := make([]map[string]interface{}, 0, len(statuses))
	for _, status := range statuses {
		var count int64
		_ = crmScopedObjectQuery(auth, "leads").Where("crm_objects.status = ?", status).Count(&count).Error
		funnel = append(funnel, map[string]interface{}{"status": status, "count": count})
	}
	type sourceRow struct {
		Source string
		Count  int64
	}
	var leads []models.CRMObject
	_ = crmScopedObjectQuery(auth, "leads").Limit(1000).Find(&leads).Error
	sourceCounts := map[string]int64{}
	hot := map[string]int64{}
	mqlBySource := map[string]int64{}
	convertedBySource := map[string]int64{}
	nurturingBySource := map[string]int64{}
	ownerOpen := map[uint]int64{}
	for _, lead := range leads {
		payload := payloadMap(lead.PayloadJSON)
		source := firstPayloadText(payload, "source_channel", "utm_source")
		if source == "" {
			source = "manual"
		}
		sourceCounts[source]++
		switch lead.Status {
		case "mql", "sdr_following", "sql":
			mqlBySource[source]++
		case "converted":
			convertedBySource[source]++
		case "nurturing":
			nurturingBySource[source]++
		}
		if !isTerminalLeadStatus(lead.Status) {
			ownerOpen[lead.OwnerID]++
		}
		temp := leadTemperature(lead)
		hot[temp]++
	}
	sources := make([]map[string]interface{}, 0, len(sourceCounts))
	for source, count := range sourceCounts {
		mqlCount := mqlBySource[source]
		convertedCount := convertedBySource[source]
		rate := 0.0
		if count > 0 {
			rate = float64(convertedCount) / float64(count)
		}
		sources = append(sources, map[string]interface{}{"source": source, "count": count, "mql": mqlCount, "converted": convertedCount, "nurturing": nurturingBySource[source], "conversion_rate": rate})
	}
	var today int64
	start := time.Now().Truncate(24 * time.Hour)
	_ = crmScopedObjectQuery(auth, "leads").Where("crm_objects.created_at >= ?", start).Count(&today).Error
	var highValue int64
	_ = crmScopedObjectQuery(auth, "leads").Where("crm_objects.score >= ?", 75).Count(&highValue).Error
	var overdue int64
	_ = crmScopedObjectQuery(auth, "leads").Where("crm_objects.status IN ? AND crm_objects.updated_at < ?", []string{"pending_assign", "following", "mql", "sdr_following"}, time.Now().Add(-24*time.Hour)).Count(&overdue).Error
	var mqlRejected int64
	if rejectedClause, rejectedArgs := payloadJSONKeyBoolLikeClause(true, "mql_rejected"); rejectedClause != "" {
		_ = crmScopedObjectQuery(auth, "leads").Where("crm_objects.status = ?", "nurturing").Where(rejectedClause, rejectedArgs...).Count(&mqlRejected).Error
	}
	var converted int64
	_ = crmScopedObjectQuery(auth, "leads").Where("crm_objects.status = ?", "converted").Count(&converted).Error
	workload := make([]map[string]interface{}, 0, len(ownerOpen))
	for ownerID, count := range ownerOpen {
		workload = append(workload, map[string]interface{}{"owner_id": ownerID, "name": fmt.Sprintf("负责人#%d", ownerID), "open_leads": count})
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"funnel":      funnel,
		"sources":     sources,
		"temperature": hot,
		"workload":    workload,
		"alerts": map[string]interface{}{
			"mql_rejected": mqlRejected,
			"overdue":      overdue,
		},
		"kpi": map[string]interface{}{
			"total":      len(leads),
			"today":      today,
			"high_value": highValue,
			"overdue":    overdue,
			"converted":  converted,
		},
	})
}

func (c *CRMController) ListLeadNurtureTemplates() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	templates := defaultLeadNurtureTemplates()
	var setting models.CRMSetting
	custom := []map[string]interface{}{}
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "lead_nurture", "templates").First(&setting).Error; err == nil {
		_ = json.Unmarshal([]byte(setting.ValueJSON), &custom)
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": append(templates, custom...), "tenant_id": auth.TenantID})
}

func (c *CRMController) SaveLeadNurtureTemplate() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageLeadOperations(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "仅租户管理员或具备线索管理权限的角色可维护培育模板")
		return
	}
	var req struct {
		Name        string                   `json:"name"`
		Description string                   `json:"description"`
		Trigger     map[string]interface{}   `json:"trigger"`
		Nodes       []map[string]interface{} `json:"nodes"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		utils.BadRequest(c.Ctx, "培育模板名称不能为空")
		return
	}
	if len(req.Nodes) == 0 {
		req.Nodes = []map[string]interface{}{
			{"type": "notice", "title": "创建培育站内提醒", "delay_days": 0},
			{"type": "task", "title": "创建跟进任务", "delay_days": 1},
			{"type": "status", "title": "保持培育状态", "status": "nurturing"},
		}
	}
	var existing models.CRMSetting
	templates := []map[string]interface{}{}
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "lead_nurture", "templates").First(&existing).Error; err == nil {
		_ = json.Unmarshal([]byte(existing.ValueJSON), &templates)
	}
	template := map[string]interface{}{
		"key":         fmt.Sprintf("custom_%d", time.Now().UnixNano()),
		"name":        req.Name,
		"description": req.Description,
		"trigger":     req.Trigger,
		"nodes":       req.Nodes,
		"preset":      false,
		"created_by":  auth.UserID,
		"created_at":  time.Now().Format(time.RFC3339),
	}
	templates = append(templates, template)
	setting := models.CRMSetting{TenantModel: models.TenantModel{TenantID: auth.TenantID}, Scope: "lead_nurture", Key: "templates", ValueJSON: toJSONString(templates), UpdatedBy: auth.UserID}
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "lead_nurture", "templates").Assign(setting).FirstOrCreate(&setting).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存培育模板失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SAVE_LEAD_NURTURE_TEMPLATE", "线索培育模板", "operation.save_lead_nurture_template", map[string]interface{}{"name": req.Name}, "success", "")
	utils.Success(c.Ctx, template)
}

func (c *CRMController) UpdateLeadNurtureTemplate() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageLeadOperations(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "仅租户管理员或具备线索管理权限的角色可维护培育模板")
		return
	}
	key := strings.TrimSpace(c.Ctx.Input.Param(":key"))
	if key == "" {
		utils.BadRequest(c.Ctx, "模板标识不能为空")
		return
	}
	if isPresetLeadNurtureTemplateKey(key) {
		utils.BadRequest(c.Ctx, "系统基础模板不可编辑")
		return
	}
	var req struct {
		Name        string                   `json:"name"`
		Description string                   `json:"description"`
		Trigger     map[string]interface{}   `json:"trigger"`
		Nodes       []map[string]interface{} `json:"nodes"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		utils.BadRequest(c.Ctx, "培育模板名称不能为空")
		return
	}
	templates, setting, err := loadCustomLeadNurtureTemplates(auth.TenantID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "加载培育模板失败")
		return
	}
	updated := map[string]interface{}{}
	found := false
	for index, item := range templates {
		if strings.TrimSpace(fmt.Sprint(item["key"])) != key {
			continue
		}
		item["name"] = req.Name
		item["description"] = req.Description
		item["trigger"] = req.Trigger
		item["nodes"] = req.Nodes
		item["preset"] = false
		item["updated_by"] = auth.UserID
		item["updated_at"] = time.Now().Format(time.RFC3339)
		templates[index] = item
		updated = item
		found = true
		break
	}
	if !found {
		utils.NotFound(c.Ctx, "自定义培育模板不存在")
		return
	}
	if err := saveCustomLeadNurtureTemplates(auth, setting, templates); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存培育模板失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_LEAD_NURTURE_TEMPLATE", "线索培育模板", "operation.update_lead_nurture_template", map[string]interface{}{"key": key, "name": req.Name}, "success", "")
	utils.Success(c.Ctx, updated)
}

func (c *CRMController) DeleteLeadNurtureTemplate() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageLeadOperations(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "仅租户管理员或具备线索管理权限的角色可维护培育模板")
		return
	}
	key := strings.TrimSpace(c.Ctx.Input.Param(":key"))
	if key == "" {
		utils.BadRequest(c.Ctx, "模板标识不能为空")
		return
	}
	if isPresetLeadNurtureTemplateKey(key) {
		utils.BadRequest(c.Ctx, "系统基础模板不可删除")
		return
	}
	templates, setting, err := loadCustomLeadNurtureTemplates(auth.TenantID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "加载培育模板失败")
		return
	}
	next := []map[string]interface{}{}
	found := false
	for _, item := range templates {
		if strings.TrimSpace(fmt.Sprint(item["key"])) == key {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		utils.NotFound(c.Ctx, "自定义培育模板不存在")
		return
	}
	if err := saveCustomLeadNurtureTemplates(auth, setting, next); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "删除培育模板失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "DELETE_LEAD_NURTURE_TEMPLATE", "线索培育模板", "operation.delete_lead_nurture_template", map[string]interface{}{"key": key}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"deleted": true, "key": key})
}

func (c *CRMController) ListLeadAutomationRules() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"rules": leadAutomationRules(auth), "tenant_id": auth.TenantID})
}

func (c *CRMController) SaveLeadAutomationRules() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageLeadOperations(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "仅租户管理员或具备线索管理权限的角色可维护线索自动化规则")
		return
	}
	var req map[string]interface{}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	rules := normalizeLeadAutomationRules(req)
	setting := models.CRMSetting{TenantModel: models.TenantModel{TenantID: auth.TenantID}, Scope: "lead_automation", Key: "rules", ValueJSON: toJSONString(rules), UpdatedBy: auth.UserID}
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "lead_automation", "rules").Assign(setting).FirstOrCreate(&setting).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存线索自动化规则失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SAVE_LEAD_AUTOMATION_RULES", "线索自动化规则", "operation.save_lead_automation_rules", rules, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"rules": rules, "tenant_id": auth.TenantID})
}

func (c *CRMController) RunLeadAutomation() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageLeadOperations(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "仅租户管理员或具备线索管理权限的角色可执行线索自动化")
		return
	}
	var req struct {
		Limit  int    `json:"limit"`
		Reason string `json:"reason"`
		DryRun bool   `json:"dry_run"`
		Force  bool   `json:"force"`
	}
	if len(c.Ctx.Input.RequestBody) > 0 && !bindJSON(c.Ctx, &req) {
		return
	}
	result, err := runLeadAutomation(auth, req.Limit, req.Reason, req.DryRun, req.Force)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "执行线索自动化失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "RUN_LEAD_AUTOMATION", "线索自动化", "operation.run_lead_automation", map[string]interface{}{"recycled": result.Recycled, "nurtured": result.Nurtured, "dry_run": req.DryRun}, "success", "")
	utils.Success(c.Ctx, result)
}

func StartLeadAutomationScheduler() {
	if leadAutomationSchedulerStop != nil {
		return
	}
	leadAutomationSchedulerStop = make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		services.WithSchedulerLock("lead_automation_scheduler", runScheduledLeadAutomationTenants)
		for {
			select {
			case <-ticker.C:
				services.WithSchedulerLock("lead_automation_scheduler", runScheduledLeadAutomationTenants)
			case <-leadAutomationSchedulerStop:
				return
			}
		}
	}()
}

func StopLeadAutomationScheduler() {
	if leadAutomationSchedulerStop == nil {
		return
	}
	close(leadAutomationSchedulerStop)
	leadAutomationSchedulerStop = nil
}

func (c *CRMController) ConvertLead() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的线索ID")
		return
	}
	var lead models.CRMObject
	if err := crmScopedObjectQuery(auth, "leads").Where("id = ?", id).First(&lead).Error; err != nil {
		utils.NotFound(c.Ctx, "线索不存在")
		return
	}
	var req struct {
		CompanyName string                 `json:"company_name"`
		ContactName string                 `json:"contact_name"`
		DealName    string                 `json:"deal_name"`
		Amount      float64                `json:"amount"`
		Currency    string                 `json:"currency"`
		Reason      string                 `json:"reason"`
		Payload     map[string]interface{} `json:"payload"`
	}
	_ = bindJSON(c.Ctx, &req)
	if isTerminalLeadStatus(lead.Status) {
		utils.BadRequest(c.Ctx, "终态线索不能重复转化")
		return
	}
	leadPayload := payloadMap(lead.PayloadJSON)
	req.CompanyName = defaultText(req.CompanyName, firstPayloadText(leadPayload, "company_name", "company"))
	if req.CompanyName == "" {
		utils.BadRequest(c.Ctx, "请先编辑线索补充公司后再转商机")
		return
	}
	req.ContactName = defaultText(req.ContactName, firstPayloadText(leadPayload, "contact_name", "contact", "name"))
	if req.ContactName == "" {
		utils.BadRequest(c.Ctx, "请先编辑线索补充联系人后再转商机")
		return
	}
	if req.DealName == "" {
		req.DealName = lead.Name + " 交易"
	}
	if req.Currency == "" {
		req.Currency = lead.Currency
	}
	if req.Currency == "" {
		req.Currency = tenantBusinessCurrency(auth.TenantID)
	}
	var company, contact, opportunity models.CRMObject
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if companyID := firstPositiveUint(lead.CompanyID, uintFromValue(leadPayload["company_id"])); companyID > 0 {
			_ = tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "contacts", "company", companyID).First(&company).Error
		}
		if company.ID == 0 {
			company = findCRMObjectByPayloadTx(tx, auth, "contacts", "company", req.CompanyName, "", "")
		}
		if company.ID == 0 {
			company = cloneCRMObject(auth, "contacts", "company", req.CompanyName, lead)
			company.PayloadJSON = toJSONString(mergePayload(leadPayload, map[string]interface{}{"company_name": req.CompanyName, "source_channel": firstPayloadText(leadPayload, "source_channel", "utm_source"), "attribution_trace": leadPayload["attribution_trace"]}))
			if err := tx.Create(&company).Error; err != nil {
				return err
			}
		}
		if contactID := firstPositiveUint(lead.ContactID, uintFromValue(leadPayload["contact_id"])); contactID > 0 {
			_ = tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "contacts", "contact", contactID).First(&contact).Error
		}
		if contact.ID == 0 {
			contact = findCRMObjectByPayloadTx(tx, auth, "contacts", "contact", req.ContactName, firstPayloadText(leadPayload, "phone", "mobile"), firstPayloadText(leadPayload, "email"))
		}
		if contact.ID == 0 {
			contact = cloneCRMObject(auth, "contacts", "contact", req.ContactName, lead)
			contact.CompanyID = company.ID
			contact.PayloadJSON = toJSONString(mergePayload(leadPayload, map[string]interface{}{"company": company.Name, "company_id": company.ID, "contact_name": req.ContactName}))
			if err := tx.Create(&contact).Error; err != nil {
				return err
			}
		} else if contact.CompanyID == 0 && company.ID > 0 {
			contact.CompanyID = company.ID
			contactPayload := payloadMap(contact.PayloadJSON)
			contactPayload["company"] = company.Name
			contactPayload["company_name"] = company.Name
			contactPayload["company_id"] = company.ID
			contact.PayloadJSON = toJSONString(contactPayload)
			contact.UpdatedBy = auth.UserID
			if err := tx.Save(&contact).Error; err != nil {
				return err
			}
		}
		opportunity = cloneCRMObject(auth, "opportunities", "opportunity", req.DealName, lead)
		opportunity.Amount = req.Amount
		opportunity.Currency = req.Currency
		opportunity.CompanyID = company.ID
		opportunity.ContactID = contact.ID
		opportunity.Status = "initial_contact"
		attribution := inheritAttributionPayload(leadPayload)
		opportunity.PayloadJSON = toJSONString(mergePayload(leadPayload, mergePayload(req.Payload, mergePayload(attribution, map[string]interface{}{"stage": "initial_contact", "probability": 10, "company_id": company.ID, "contact_id": contact.ID, "source_lead_id": lead.ID, "source_channel": defaultText(firstPayloadText(leadPayload, "source_channel", "utm_source"), "manual"), "attribution_trace": leadPayload["attribution_trace"]}))))
		if err := normalizeCustomerChainObjectTx(tx, auth, &opportunity, lead, company, contact); err != nil {
			return err
		}
		if err := tx.Create(&opportunity).Error; err != nil {
			return err
		}
		lead.Status = "converted"
		lead.CompanyID = company.ID
		lead.ContactID = contact.ID
		lead.PayloadJSON = toJSONString(mergePayload(leadPayload, map[string]interface{}{"converted_at": time.Now().Format(time.RFC3339), "company_id": company.ID, "contact_id": contact.ID, "opportunity_id": opportunity.ID}))
		if err := tx.Save(&lead).Error; err != nil {
			return err
		}
		relations := []models.CRMRelation{
			newCRMRelation(auth, "leads", lead.ID, "contacts", company.ID, "converted_company"),
			newCRMRelation(auth, "leads", lead.ID, "contacts", contact.ID, "converted_to"),
			newCRMRelation(auth, "leads", lead.ID, "opportunities", opportunity.ID, "converted_to"),
			newCRMRelation(auth, "contacts", company.ID, "contacts", contact.ID, "company_contact"),
			newCRMRelation(auth, "contacts", company.ID, "opportunities", opportunity.ID, "linked_to"),
			newCRMRelation(auth, "contacts", contact.ID, "opportunities", opportunity.ID, "linked_to"),
		}
		if err := tx.Create(&relations).Error; err != nil {
			return err
		}
		if err := ensureCustomerChainRelationsTx(tx, auth, opportunity); err != nil {
			return err
		}
		if err := ensureCampaignObjectRelationTx(tx, auth, opportunity, "generated_opportunity", "lead_convert"); err != nil {
			return err
		}
		if err := migrateLeadTimelineRelations(tx, auth, lead.ID, company.ID, contact.ID, opportunity.ID); err != nil {
			return err
		}
		return createLeadTimeline(tx, auth, lead, "convert", "线索转商机", map[string]interface{}{"company_id": company.ID, "contact_id": contact.ID, "opportunity_id": opportunity.ID, "reason": strings.TrimSpace(req.Reason)})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "线索转化失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CONVERT_LEAD", fmt.Sprintf("线索[%d]", lead.ID), "operation.convert_lead", map[string]interface{}{"lead_id": lead.ID, "company_id": company.ID, "contact_id": contact.ID, "opportunity_id": opportunity.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"lead": lead, "company": company, "contact": contact, "opportunity": opportunity})
}

func migrateLeadTimelineRelations(tx *gorm.DB, auth authContext, leadID uint, companyID uint, contactID uint, opportunityID uint) error {
	var leadTimelineRelations []models.CRMRelation
	if err := tx.Where(
		"tenant_id = ? AND source_type = ? AND source_id = ? AND relation_type = ? AND target_type = ?",
		auth.TenantID,
		"leads",
		leadID,
		"timeline",
		"contacts",
	).Find(&leadTimelineRelations).Error; err != nil {
		return err
	}
	for _, relation := range leadTimelineRelations {
		if relation.TargetID == 0 {
			continue
		}
		if contactID > 0 {
			contactRelation := newCRMRelation(auth, "contacts", contactID, "contacts", relation.TargetID, "timeline")
			if err := tx.Where(
				"tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?",
				auth.TenantID, "contacts", contactID, "contacts", relation.TargetID, "timeline",
			).FirstOrCreate(&contactRelation).Error; err != nil {
				return err
			}
		}
		if companyID > 0 {
			companyRelation := newCRMRelation(auth, "contacts", companyID, "contacts", relation.TargetID, "timeline")
			if err := tx.Where(
				"tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?",
				auth.TenantID, "contacts", companyID, "contacts", relation.TargetID, "timeline",
			).FirstOrCreate(&companyRelation).Error; err != nil {
				return err
			}
		}
		if opportunityID > 0 {
			opportunityRelation := newCRMRelation(auth, "opportunities", opportunityID, "contacts", relation.TargetID, "timeline")
			if err := tx.Where(
				"tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?",
				auth.TenantID, "opportunities", opportunityID, "contacts", relation.TargetID, "timeline",
			).FirstOrCreate(&opportunityRelation).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *CRMController) AssignLead() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的线索ID")
		return
	}
	var req struct {
		OwnerID uint   `json:"owner_id"`
		Reason  string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.OwnerID == 0 {
		utils.BadRequest(c.Ctx, "负责人不能为空")
		return
	}
	if !canAssignLead(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "仅租户管理员或具备线索管理权限的角色可分配线索")
		return
	}
	if !tenantUserExists(auth.TenantID, req.OwnerID) {
		utils.BadRequest(c.Ctx, "负责人不存在或不属于当前租户")
		return
	}
	var lead models.CRMObject
	if err := crmScopedObjectQuery(auth, "leads").Where("id = ?", id).First(&lead).Error; err != nil {
		utils.NotFound(c.Ctx, "线索不存在")
		return
	}
	if isTerminalLeadStatus(lead.Status) {
		utils.BadRequest(c.Ctx, "终态线索不能分配")
		return
	}
	payload := payloadMap(lead.PayloadJSON)
	payload["assignment_reason"] = req.Reason
	payload["assigned_at"] = time.Now().Format(time.RFC3339)
	lead.OwnerID = req.OwnerID
	lead.Status = "following"
	lead.PayloadJSON = toJSONString(payload)
	task := cloneCRMObject(auth, "workspace", "task", "跟进线索："+lead.Name, lead)
	task.Status = "open"
	task.OwnerID = req.OwnerID
	task.PayloadJSON = toJSONString(map[string]interface{}{"source": "lead_assignment", "lead_id": lead.ID, "owner_id": req.OwnerID})
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&lead).Error; err != nil {
			return err
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		if err := tx.Create(&[]models.CRMRelation{
			newCRMRelation(auth, "leads", lead.ID, "workspace", task.ID, "created_task"),
		}).Error; err != nil {
			return err
		}
		if err := createLeadTimeline(tx, auth, lead, "assign", "线索分配", map[string]interface{}{"owner_id": req.OwnerID, "reason": req.Reason}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "线索分配失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "ASSIGN_LEAD", fmt.Sprintf("线索[%d]", lead.ID), "operation.assign_lead", map[string]interface{}{"lead_id": lead.ID, "owner_id": req.OwnerID, "task_id": task.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"lead": lead, "task": task})
}

func (c *CRMController) AutoAssignLeads() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canAssignLead(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "仅租户管理员或具备线索管理权限的角色可自动分配线索")
		return
	}
	var req struct {
		OwnerIDs []uint `json:"owner_ids"`
		Limit    int    `json:"limit"`
		Reason   string `json:"reason"`
	}
	if len(c.Ctx.Input.RequestBody) > 0 && !bindJSON(c.Ctx, &req) {
		return
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 200 {
		req.Limit = 200
	}
	if len(req.OwnerIDs) == 0 {
		req.OwnerIDs = activeTenantLeadOwners(auth.TenantID)
	}
	if len(req.OwnerIDs) == 0 {
		utils.BadRequest(c.Ctx, "请先指定可分配负责人")
		return
	}
	validOwners := make([]uint, 0, len(req.OwnerIDs))
	for _, ownerID := range req.OwnerIDs {
		if ownerID > 0 && tenantUserExists(auth.TenantID, ownerID) {
			validOwners = append(validOwners, ownerID)
		}
	}
	if len(validOwners) == 0 {
		utils.BadRequest(c.Ctx, "未找到可用负责人")
		return
	}
	leads := []models.CRMObject{}
	if err := crmScopedObjectQuery(auth, "leads").
		Where("crm_objects.status IN ? AND (crm_objects.owner_id = 0 OR crm_objects.owner_id IS NULL)", []string{"new", "pending_assign"}).
		Order("crm_objects.updated_at asc").
		Limit(req.Limit).
		Find(&leads).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "读取待分配线索失败")
		return
	}
	if len(leads) == 0 {
		utils.Success(c.Ctx, map[string]interface{}{"count": 0, "items": []map[string]interface{}{}, "owners": validOwners})
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "轮询自动分配"
	}
	start := leadAssignmentStartIndex(auth, validOwners)
	items := []map[string]interface{}{}
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		for index, lead := range leads {
			ownerID := validOwners[(start+index)%len(validOwners)]
			payload := payloadMap(lead.PayloadJSON)
			payload["assignment_reason"] = reason
			payload["assigned_at"] = time.Now().Format(time.RFC3339)
			payload["assignment_mode"] = "round_robin"
			lead.OwnerID = ownerID
			lead.Status = "following"
			lead.PayloadJSON = toJSONString(payload)
			lead.UpdatedBy = auth.UserID
			if err := tx.Save(&lead).Error; err != nil {
				return err
			}
			task := cloneCRMObject(auth, "workspace", "task", "跟进线索："+lead.Name, lead)
			task.Status = "open"
			task.OwnerID = ownerID
			task.PayloadJSON = toJSONString(map[string]interface{}{"source": "lead_assignment_auto", "lead_id": lead.ID, "owner_id": ownerID})
			if err := tx.Create(&task).Error; err != nil {
				return err
			}
			if err := tx.Create(&[]models.CRMRelation{
				newCRMRelation(auth, "leads", lead.ID, "workspace", task.ID, "created_task"),
			}).Error; err != nil {
				return err
			}
			if err := createLeadTimeline(tx, auth, lead, "assign_auto", "线索自动分配", map[string]interface{}{"owner_id": ownerID, "reason": reason, "mode": "round_robin"}); err != nil {
				return err
			}
			items = append(items, map[string]interface{}{
				"lead_id":   lead.ID,
				"lead_name": lead.Name,
				"owner_id":  ownerID,
				"task_id":   task.ID,
			})
		}
		return saveLeadAssignmentCursor(tx, auth.TenantID, validOwners, (start+len(leads))%len(validOwners), auth.UserID)
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "自动分配线索失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "AUTO_ASSIGN_LEADS", "线索自动分配", "operation.auto_assign_leads", map[string]interface{}{"count": len(items), "owners": validOwners, "reason": reason}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"count": len(items), "items": items, "owners": validOwners, "reason": reason})
}

func canAssignLead(auth authContext) bool {
	return auth.IsSuperAdmin() || auth.IsTenantAdmin() || authHasPermissionAny(auth, "leads:assign", "leads:update", "leads:manage")
}

func canManageLeadOperations(auth authContext) bool {
	return auth.IsSuperAdmin() || auth.IsTenantAdmin() || authHasPermissionAny(auth, "leads:update", "leads:manage")
}

func tenantUserExists(tenantID uint, userID uint) bool {
	var count int64
	if err := utils.DB.Model(&models.User{}).
		Where("tenant_id = ? AND id = ? AND status = ?", tenantID, userID, "active").
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func activeTenantLeadOwners(tenantID uint) []uint {
	users := []models.User{}
	if err := utils.DB.Where("tenant_id = ? AND status = ?", tenantID, "active").Order("id asc").Find(&users).Error; err != nil {
		return []uint{}
	}
	ownerIDs := make([]uint, 0, len(users))
	for _, user := range users {
		if user.ID > 0 {
			ownerIDs = append(ownerIDs, user.ID)
		}
	}
	return ownerIDs
}

func leadAssignmentStartIndex(auth authContext, owners []uint) int {
	if len(owners) == 0 {
		return 0
	}
	setting := crmSettingValue(auth.TenantID, "lead_assign", "round_robin")
	cursor := int(numericValue(setting["cursor"]))
	if cursor < 0 {
		cursor = 0
	}
	return cursor % len(owners)
}

func saveLeadAssignmentCursor(tx *gorm.DB, tenantID uint, owners []uint, cursor int, userID uint) error {
	if len(owners) == 0 {
		return nil
	}
	value := map[string]interface{}{
		"mode":        "round_robin",
		"owners":      owners,
		"cursor":      cursor,
		"updated_at":  time.Now().Format(time.RFC3339),
		"updated_by":  userID,
		"description": "线索轮询自动分配游标",
	}
	setting := models.CRMSetting{
		TenantModel: models.TenantModel{TenantID: tenantID},
		Scope:       "lead_assign",
		Key:         "round_robin",
		ValueJSON:   toJSONString(value),
		UpdatedBy:   userID,
	}
	return tx.Where("tenant_id = ? AND scope = ? AND `key` = ?", tenantID, setting.Scope, setting.Key).Assign(setting).FirstOrCreate(&setting).Error
}

func (c *CRMController) RecycleLead() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的线索ID")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var lead models.CRMObject
	if err := crmScopedObjectQuery(auth, "leads").Where("id = ?", id).First(&lead).Error; err != nil {
		utils.NotFound(c.Ctx, "线索不存在")
		return
	}
	if isTerminalLeadStatus(lead.Status) {
		utils.BadRequest(c.Ctx, "终态线索不能回收")
		return
	}
	payload := payloadMap(lead.PayloadJSON)
	payload["recycle_reason"] = req.Reason
	payload["recycled_at"] = time.Now().Format(time.RFC3339)
	lead.Status = "pending_assign"
	lead.OwnerID = 0
	lead.PayloadJSON = toJSONString(payload)
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&lead).Error; err != nil {
			return err
		}
		return createLeadTimeline(tx, auth, lead, "recycle", "线索回收", map[string]interface{}{"reason": req.Reason})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "线索回收失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "RECYCLE_LEAD", fmt.Sprintf("线索[%d]", lead.ID), "operation.recycle_lead", map[string]interface{}{"lead_id": lead.ID, "reason": req.Reason}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"lead": lead})
}

func (c *CRMController) DedupeLeads() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		LeadID               uint   `json:"lead_id"`
		CompanyID            uint   `json:"company_id"`
		ContactID            uint   `json:"contact_id"`
		CompanyName          string `json:"company_name"`
		ContactName          string `json:"contact_name"`
		Phone                string `json:"phone"`
		Email                string `json:"email"`
		IntentProductID      uint   `json:"intent_product_id"`
		IntentProduct        string `json:"intent_product"`
		SourceChannel        string `json:"source_channel"`
		UTMSource            string `json:"utm_source"`
		CampaignID           uint   `json:"campaign_id"`
		ExternalSubmissionID string `json:"external_submission_id"`
		ImportBatchID        string `json:"import_batch_id"`
		Note                 string `json:"note"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var base models.CRMObject
	if req.LeadID > 0 {
		_ = crmScopedObjectQuery(auth, "leads").Where("id = ?", req.LeadID).First(&base).Error
		basePayload := payloadMap(base.PayloadJSON)
		if req.CompanyName == "" {
			req.CompanyName = firstPayloadText(basePayload, "company_name", "company")
		}
		if req.ContactName == "" {
			req.ContactName = firstPayloadText(basePayload, "contact_name", "contact")
		}
		if req.Phone == "" {
			req.Phone = firstPayloadText(basePayload, "phone", "mobile")
		}
		if req.Email == "" {
			req.Email = firstPayloadText(basePayload, "email")
		}
		if req.CompanyID == 0 {
			req.CompanyID = base.CompanyID
			if req.CompanyID == 0 {
				req.CompanyID = uintFromValue(basePayload["company_id"])
			}
		}
		if req.ContactID == 0 {
			req.ContactID = base.ContactID
			if req.ContactID == 0 {
				req.ContactID = uintFromValue(basePayload["contact_id"])
			}
		}
		if req.IntentProductID == 0 {
			req.IntentProductID = leadPrimaryIntentProductID(basePayload)
		}
		if req.IntentProduct == "" {
			req.IntentProduct = firstPayloadText(basePayload, "intent_product", "product_name")
		}
		if req.SourceChannel == "" {
			req.SourceChannel = firstPayloadText(basePayload, "source_channel", "source")
		}
		if req.UTMSource == "" {
			req.UTMSource = firstPayloadText(basePayload, "utm_source")
		}
		if req.CampaignID == 0 {
			req.CampaignID = uintFromValue(basePayload["campaign_id"])
		}
		if req.ExternalSubmissionID == "" {
			req.ExternalSubmissionID = firstPayloadText(basePayload, "external_submission_id", "submission_id", "form_submission_id")
		}
		if req.ImportBatchID == "" {
			req.ImportBatchID = firstPayloadText(basePayload, "import_batch_id", "batch_id")
		}
		if req.Note == "" {
			req.Note = firstPayloadText(basePayload, "note", "requirement", "summary")
		}
	}
	possibleDuplicates, sameContactLeads, sameCompanyLeads, matchSummary := findLeadDedupeMatches(auth, leadDedupeInput{
		ExcludeID:            req.LeadID,
		CompanyID:            req.CompanyID,
		ContactID:            req.ContactID,
		CompanyName:          req.CompanyName,
		ContactName:          req.ContactName,
		Phone:                req.Phone,
		Email:                req.Email,
		IntentProductID:      req.IntentProductID,
		IntentProduct:        req.IntentProduct,
		SourceChannel:        req.SourceChannel,
		UTMSource:            req.UTMSource,
		CampaignID:           req.CampaignID,
		ExternalSubmissionID: req.ExternalSubmissionID,
		ImportBatchID:        req.ImportBatchID,
		Note:                 req.Note,
	})
	companies := findCRMObjectsByPayload(auth, "contacts", "company", req.CompanyName, "", "", 10)
	contacts := findCRMObjectsByPayload(auth, "contacts", "contact", req.ContactName, req.Phone, req.Email, 10)
	utils.Success(c.Ctx, map[string]interface{}{
		"leads":               possibleDuplicates,
		"possible_duplicates": possibleDuplicates,
		"same_contact_leads":  sameContactLeads,
		"same_company_leads":  sameCompanyLeads,
		"companies":           companies,
		"contacts":            contacts,
		"match_summary":       matchSummary,
		"rules_version":       "lead-dedupe-v2",
	})
}

func (c *CRMController) ChangeLeadStatus() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的线索ID")
		return
	}
	var req struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	status := normalizeLeadStatus(req.Status)
	if status == "" {
		utils.BadRequest(c.Ctx, "无效的线索状态")
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		utils.BadRequest(c.Ctx, "状态变更原因不能为空")
		return
	}
	var lead models.CRMObject
	if err := crmScopedObjectQuery(auth, "leads").Where("id = ?", id).First(&lead).Error; err != nil {
		utils.NotFound(c.Ctx, "线索不存在")
		return
	}
	if isTerminalLeadStatus(lead.Status) && !auth.IsSuperAdmin() && !auth.IsTenantAdmin() {
		utils.BadRequest(c.Ctx, "终态线索不可变更")
		return
	}
	before := lead.Status
	payload := payloadMap(lead.PayloadJSON)
	payload["status_reason"] = strings.TrimSpace(req.Reason)
	if status == "invalid" {
		payload["invalid_reason"] = strings.TrimSpace(req.Reason)
	}
	if status == "nurturing" && isMQLRejectedReason(req.Reason) {
		payload["mql_rejected"] = true
		payload["mql_rejected_reason"] = strings.TrimSpace(req.Reason)
		payload["mql_rejected_at"] = time.Now().Format(time.RFC3339)
		payload["mql_rejected_by"] = auth.UserID
	}
	payload["status_changed_at"] = time.Now().Format(time.RFC3339)
	payload["status_changed_by"] = auth.UserID
	lead.Status = status
	lead.PayloadJSON = toJSONString(payload)
	lead.UpdatedBy = auth.UserID
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&lead).Error; err != nil {
			return err
		}
		return createLeadTimeline(tx, auth, lead, "status_change", fmt.Sprintf("%s -> %s", before, status), map[string]interface{}{"from": before, "to": status, "reason": req.Reason})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "线索状态变更失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CHANGE_LEAD_STATUS", fmt.Sprintf("线索[%d]", lead.ID), "operation.change_lead_status", map[string]interface{}{"from": before, "to": status, "reason": req.Reason}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"lead": lead})
}

func (c *CRMController) MergeLeads() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的线索ID")
		return
	}
	var req struct {
		DuplicateIDs []uint `json:"duplicate_ids"`
		Reason       string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if len(req.DuplicateIDs) == 0 {
		utils.BadRequest(c.Ctx, "请选择要合并的重复线索")
		return
	}
	var primary models.CRMObject
	if err := crmScopedObjectQuery(auth, "leads").Where("id = ?", id).First(&primary).Error; err != nil {
		utils.NotFound(c.Ctx, "主线索不存在")
		return
	}
	merged := make([]uint, 0, len(req.DuplicateIDs))
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		primaryPayload := payloadMap(primary.PayloadJSON)
		mergedSources := interfaceList(primaryPayload["merged_sources"])
		for _, duplicateID := range req.DuplicateIDs {
			if duplicateID == 0 || duplicateID == primary.ID {
				continue
			}
			var duplicate models.CRMObject
			if err := crmScopedObjectQuery(auth, "leads").Where("id = ?", duplicateID).First(&duplicate).Error; err != nil {
				return fmt.Errorf("重复线索[%d]不存在", duplicateID)
			}
			duplicatePayload := payloadMap(duplicate.PayloadJSON)
			for key, value := range duplicatePayload {
				if strings.TrimSpace(fmt.Sprint(primaryPayload[key])) == "" {
					primaryPayload[key] = value
				}
			}
			mergedSources = append(mergedSources, map[string]interface{}{"lead_id": duplicate.ID, "name": duplicate.Name, "payload": duplicatePayload})
			if duplicate.Score > primary.Score {
				primary.Score = duplicate.Score
			}
			if duplicate.Amount > primary.Amount {
				primary.Amount = duplicate.Amount
			}
			if err := tx.Model(&models.CRMRelation{}).Where("tenant_id = ? AND source_type = ? AND source_id = ?", auth.TenantID, "leads", duplicate.ID).Update("source_id", primary.ID).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.CRMRelation{}).Where("tenant_id = ? AND target_type = ? AND target_id = ?", auth.TenantID, "leads", duplicate.ID).Update("target_id", primary.ID).Error; err != nil {
				return err
			}
			duplicate.Status = "invalid"
			duplicate.UpdatedBy = auth.UserID
			duplicate.PayloadJSON = toJSONString(map[string]interface{}{"merged_to": primary.ID, "invalid_reason": "重复线索合并", "merge_reason": req.Reason})
			if err := tx.Save(&duplicate).Error; err != nil {
				return err
			}
			merged = append(merged, duplicate.ID)
		}
		primaryPayload["merged_sources"] = mergedSources
		primaryPayload["merged_at"] = time.Now().Format(time.RFC3339)
		primaryPayload["merge_reason"] = req.Reason
		primary.PayloadJSON = toJSONString(primaryPayload)
		primary.UpdatedBy = auth.UserID
		if err := tx.Save(&primary).Error; err != nil {
			return err
		}
		return createLeadTimeline(tx, auth, primary, "merge", "合并重复线索", map[string]interface{}{"merged_ids": merged, "reason": req.Reason})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, err.Error())
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "MERGE_LEADS", fmt.Sprintf("线索[%d]", primary.ID), "operation.merge_leads", map[string]interface{}{"primary_id": primary.ID, "merged_ids": merged}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"primary": primary, "merged_ids": merged})
}

func (c *CRMController) ReleaseLead() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的线索ID")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		utils.BadRequest(c.Ctx, "释放原因不能为空")
		return
	}
	var lead models.CRMObject
	if err := crmScopedObjectQuery(auth, "leads").Where("id = ?", id).First(&lead).Error; err != nil {
		utils.NotFound(c.Ctx, "线索不存在")
		return
	}
	if isTerminalLeadStatus(lead.Status) {
		utils.BadRequest(c.Ctx, "终态线索不能释放")
		return
	}
	payload := payloadMap(lead.PayloadJSON)
	payload["release_reason"] = req.Reason
	payload["released_at"] = time.Now().Format(time.RFC3339)
	lead.OwnerID = 0
	lead.Status = "pending_assign"
	lead.PayloadJSON = toJSONString(payload)
	lead.UpdatedBy = auth.UserID
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&lead).Error; err != nil {
			return err
		}
		return createLeadTimeline(tx, auth, lead, "release", "释放到公海", map[string]interface{}{"reason": req.Reason})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "线索释放失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "RELEASE_LEAD", fmt.Sprintf("线索[%d]", lead.ID), "operation.release_lead", map[string]interface{}{"lead_id": lead.ID, "reason": req.Reason}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"lead": lead})
}

func (c *CRMController) RunLeadSLARecycle() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageLeadOperations(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "仅租户管理员或具备线索管理权限的角色可执行SLA回收")
		return
	}
	var req leadSLARecycleRequest
	if len(c.Ctx.Input.RequestBody) > 0 && !bindJSON(c.Ctx, &req) {
		return
	}
	result, err := runLeadSLARecycle(auth, req)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "SLA回收失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "RUN_LEAD_SLA_RECYCLE", "线索SLA回收", "operation.run_lead_sla_recycle", map[string]interface{}{"count": len(result.Recycled), "lead_ids": result.Recycled, "dry_run": req.DryRun}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"count": len(result.Recycled), "lead_ids": result.Recycled, "items": result.DryRunItems, "cutoff": result.Cutoff})
}

func (c *CRMController) AddLeadFollowUp() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的线索ID")
		return
	}
	var req struct {
		Type       string `json:"type"`
		Content    string `json:"content"`
		NextAction string `json:"next_action"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		utils.BadRequest(c.Ctx, "跟进内容不能为空")
		return
	}
	var lead models.CRMObject
	if err := crmScopedObjectQuery(auth, "leads").Where("id = ?", id).First(&lead).Error; err != nil {
		utils.NotFound(c.Ctx, "线索不存在")
		return
	}
	payload := payloadMap(lead.PayloadJSON)
	payload["last_follow_up_at"] = time.Now().Format(time.RFC3339)
	payload["last_follow_up"] = req.Content
	if strings.TrimSpace(req.NextAction) != "" {
		payload["next_action"] = req.NextAction
	}
	if lead.Status == "new" || lead.Status == "pending_assign" || lead.Status == "mql" {
		lead.Status = "following"
	}
	lead.PayloadJSON = toJSONString(payload)
	lead.UpdatedBy = auth.UserID
	var interaction models.CRMObject
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&lead).Error; err != nil {
			return err
		}
		interaction = cloneCRMObject(auth, "contacts", "interaction", "线索跟进："+lead.Name, lead)
		interaction.Status = "active"
		interaction.PayloadJSON = toJSONString(map[string]interface{}{"interaction_type": defaultText(req.Type, "note"), "summary": req.Content, "source_module": "leads", "related_object_id": lead.ID, "happened_at": time.Now().Format(time.RFC3339)})
		if err := tx.Create(&interaction).Error; err != nil {
			return err
		}
		if err := tx.Create(&[]models.CRMRelation{
			newCRMRelation(auth, "leads", lead.ID, "contacts", interaction.ID, "timeline"),
		}).Error; err != nil {
			return err
		}
		return createLeadTimeline(tx, auth, lead, "follow_up", req.Content, map[string]interface{}{"type": req.Type, "next_action": req.NextAction})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "添加跟进失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "ADD_LEAD_FOLLOW_UP", fmt.Sprintf("线索[%d]", lead.ID), "operation.add_lead_follow_up", map[string]interface{}{"lead_id": lead.ID, "interaction_id": interaction.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"lead": lead, "interaction": interaction})
}

func (c *CRMController) StartLeadNurture() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的线索ID")
		return
	}
	var req leadNurtureRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var lead models.CRMObject
	if err := crmScopedObjectQuery(auth, "leads").Where("id = ?", id).First(&lead).Error; err != nil {
		utils.NotFound(c.Ctx, "线索不存在")
		return
	}
	template := selectLeadNurtureTemplate(auth, req.TemplateKey)
	created, err := applyLeadNurture(auth, &lead, req, template, nil)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "线索培育执行失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "START_LEAD_NURTURE", fmt.Sprintf("线索[%d]", lead.ID), "operation.start_lead_nurture", map[string]interface{}{"lead_id": lead.ID, "template": template["key"], "created": len(created), "exit": req.Exit}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"lead": lead, "template": template, "created": created})
}

func (c *CRMController) ScoreLead() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的线索ID")
		return
	}
	var lead models.CRMObject
	if err := crmScopedObjectQuery(auth, "leads").Where("id = ?", id).First(&lead).Error; err != nil {
		utils.NotFound(c.Ctx, "线索不存在")
		return
	}
	if isTerminalLeadStatus(lead.Status) {
		utils.BadRequest(c.Ctx, "终态线索不能重新评分")
		return
	}
	payload := payloadMap(lead.PayloadJSON)
	score, breakdown := calculateLeadScore(payload)
	intent := "low"
	switch {
	case score >= 75:
		intent = "high"
	case score >= 45:
		intent = "medium"
	}
	payload["ai_score"] = score
	payload["score_breakdown"] = breakdown
	payload["intent"] = intent
	payload["temperature"] = scoreToTemperature(score)
	payload["scored_at"] = time.Now().Format(time.RFC3339)
	lead.Score = score
	threshold := int(numericValue(crmSettingValue(auth.TenantID, "lead_score", "model")["mql_threshold"]))
	if threshold <= 0 {
		threshold = 60
	}
	if (lead.Status == "new" || lead.Status == "pending_assign" || lead.Status == "nurturing") && score >= threshold {
		lead.Status = "mql"
	}
	lead.PayloadJSON = toJSONString(payload)
	var task models.CRMObject
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&lead).Error; err != nil {
			return err
		}
		if score >= 60 {
			task = cloneCRMObject(auth, "workspace", "task", "跟进高分线索："+lead.Name, lead)
			task.Status = "open"
			task.PayloadJSON = toJSONString(map[string]interface{}{"source": "lead_score", "lead_id": lead.ID, "score": score, "priority": "high"})
			if err := tx.Create(&task).Error; err != nil {
				return err
			}
			if err := tx.Create(&[]models.CRMRelation{
				newCRMRelation(auth, "leads", lead.ID, "workspace", task.ID, "created_task"),
			}).Error; err != nil {
				return err
			}
		}
		return createLeadTimeline(tx, auth, lead, "score", "线索评分", map[string]interface{}{"score": score, "intent": intent, "breakdown": breakdown})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "线索评分失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SCORE_LEAD", fmt.Sprintf("线索[%d]", lead.ID), "operation.score_lead", map[string]interface{}{"lead_id": lead.ID, "score": score, "intent": intent, "task_id": task.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"lead": lead, "score": score, "intent": intent, "breakdown": breakdown, "task": task})
}

func (c *CRMController) CaptureCampaignLead() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "campaigns", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	if err := ensureCRMModuleAccess(auth, "leads", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的活动ID")
		return
	}
	var campaign models.CRMObject
	if err := crmScopedObjectQuery(auth, "campaigns").Where("id = ?", id).First(&campaign).Error; err != nil {
		utils.NotFound(c.Ctx, "活动不存在")
		return
	}
	var req struct {
		Name    string                 `json:"name"`
		Email   string                 `json:"email"`
		Phone   string                 `json:"phone"`
		Company string                 `json:"company"`
		Payload map[string]interface{} `json:"payload"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		utils.BadRequest(c.Ctx, "线索名称不能为空")
		return
	}
	campaignPayload := payloadMap(campaign.PayloadJSON)
	sourceChannel := defaultText(firstPayloadText(campaignPayload, "source_channel", "utm_source", "source"), "campaign")
	utmSource := defaultText(firstPayloadText(campaignPayload, "utm_source"), sourceChannel)
	leadPayload := map[string]interface{}{
		"email":          strings.TrimSpace(req.Email),
		"phone":          strings.TrimSpace(req.Phone),
		"company":        strings.TrimSpace(req.Company),
		"company_name":   strings.TrimSpace(req.Company),
		"contact_name":   req.Name,
		"source":         "campaign",
		"source_channel": sourceChannel,
		"campaign_id":    campaign.ID,
		"campaign":       campaign.Name,
		"campaign_name":  campaign.Name,
		"utm_source":     utmSource,
	}
	for key, value := range req.Payload {
		leadPayload[key] = value
	}
	campaignVisibility := crmObjectPayloadVisibility(campaign)
	leadPayload["visibility"] = campaignVisibility
	leadPayload["private"] = campaignVisibility == "private"
	for _, key := range []string{"utm_medium", "utm_campaign"} {
		if value, ok := campaignPayload[key]; ok && strings.TrimSpace(fmt.Sprint(value)) != "" {
			leadPayload[key] = value
		}
	}
	touches := normalizeAttributionTrace(leadPayload["attribution_trace"])
	touches = append(touches, map[string]interface{}{"touch": campaign.Name, "channel": sourceChannel, "stage": "lead_capture", "source": "campaign_capture", "occurred_at": time.Now().Format(time.RFC3339)})
	leadPayload["attribution_trace"] = touches
	if firstPayloadText(leadPayload, "source_channel") == "" {
		leadPayload["source_channel"] = sourceChannel
	}
	if firstPayloadText(leadPayload, "utm_source") == "" {
		leadPayload["utm_source"] = utmSource
	}
	leadPayloadJSON, _ := json.Marshal(leadPayload)
	lead := cloneCRMObject(auth, "leads", "lead", req.Name, campaign)
	lead.Status = "new"
	lead.PayloadJSON = string(leadPayloadJSON)
	var activity models.CRMObject
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&lead).Error; err != nil {
			return err
		}
		activity = cloneCRMObject(auth, "workspace", "task", "跟进活动线索："+req.Name, lead)
		activity.Status = "open"
		activity.PayloadJSON = toJSONString(map[string]interface{}{"source": "campaign_lead", "lead_id": lead.ID, "campaign_id": campaign.ID})
		if err := tx.Create(&activity).Error; err != nil {
			return err
		}
		relations := []models.CRMRelation{
			newCRMRelation(auth, "campaigns", campaign.ID, "leads", lead.ID, "generated_lead"),
			newCRMRelation(auth, "leads", lead.ID, "workspace", activity.ID, "created_task"),
		}
		return tx.Create(&relations).Error
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "活动线索捕获失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CAPTURE_CAMPAIGN_LEAD", fmt.Sprintf("活动[%d]", campaign.ID), "operation.capture_campaign_lead", map[string]interface{}{"campaign_id": campaign.ID, "lead_id": lead.ID, "task_id": activity.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"campaign": campaign, "lead": lead, "task": activity})
}

func runScheduledLeadAutomationTenants() {
	var settings []models.CRMSetting
	if err := utils.DB.Where("scope = ? AND `key` = ?", "lead_automation", "rules").Find(&settings).Error; err != nil {
		utils.LogWarning("线索自动化调度读取规则失败: %v", err)
		return
	}
	for _, setting := range settings {
		rules := normalizeLeadAutomationRules(payloadMap(setting.ValueJSON))
		schedule := mapValue(rules["schedule"])
		if !boolValue(schedule["enabled"], false) {
			continue
		}
		tenantID := setting.TenantID
		if tenantID == 0 {
			tenantID = defaultTenantID()
		}
		auth := authContext{TenantID: tenantID, UserID: setting.UpdatedBy, Username: "system", Role: "tenant_admin", AccountType: "tenant_admin"}
		result, err := runLeadAutomation(auth, 0, "后台调度执行线索自动化", false, false)
		if err != nil {
			utils.LogWarning("线索自动化调度失败 tenant=%d: %v", tenantID, err)
			services.LogOperation(services.OperationLogEvent{
				TenantID:       tenantID,
				UserID:         setting.UpdatedBy,
				Username:       "system",
				Operation:      "RUN_LEAD_AUTOMATION_SCHEDULE",
				Target:         "线索自动化",
				DescriptionKey: "operation.run_lead_automation_schedule",
				Result:         "failed",
				Reason:         err.Error(),
				Source:         "scheduler",
			})
			continue
		}
		if result.Scanned == 0 && len(result.Recycled) == 0 && len(result.Nurtured) == 0 {
			continue
		}
		services.LogOperation(services.OperationLogEvent{
			TenantID:       tenantID,
			UserID:         setting.UpdatedBy,
			Username:       "system",
			Operation:      "RUN_LEAD_AUTOMATION_SCHEDULE",
			Target:         "线索自动化",
			DescriptionKey: "operation.run_lead_automation_schedule",
			Params: map[string]interface{}{
				"scanned":     result.Scanned,
				"recycled":    result.Recycled,
				"nurtured":    result.Nurtured,
				"created":     len(result.Created),
				"next_run_at": result.NextRunAt,
			},
			Result: "success",
			Source: "scheduler",
		})
	}
}

func salesLeaderboard(auth authContext) []map[string]interface{} {
	type row struct {
		OwnerID uint
		Amount  float64
		Count   int64
	}
	var rows []row
	_ = crmScopedObjectQuery(auth, "deals").
		Select("crm_objects.owner_id, COALESCE(SUM(crm_objects.amount), 0) AS amount, COUNT(*) AS count").
		Where("crm_objects.status = ?", "closed_won").
		Group("crm_objects.owner_id").
		Order("amount desc").
		Limit(8).
		Scan(&rows).Error
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, map[string]interface{}{"owner_id": row.OwnerID, "name": fmt.Sprintf("负责人#%d", row.OwnerID), "amount": row.Amount, "count": row.Count})
	}
	return items
}

func campaignMultiTouchAttribution(auth authContext, campaign models.CRMObject, leads []models.CRMObject) ([]map[string]interface{}, float64, map[string]interface{}) {
	touches := []map[string]interface{}{}
	totalRevenue := 0.0
	channelRevenue := map[string]float64{}
	channelTouches := map[string]int{}
	model := crmSettingValue(auth.TenantID, "attribution", "multi_touch")
	modelName := defaultText(fmt.Sprint(model["model"]), "position_time_decay")
	halfLifeDays := numericValue(model["half_life_days"])
	if halfLifeDays <= 0 {
		halfLifeDays = 14
	}
	campaignPayload := payloadMap(campaign.PayloadJSON)
	campaignCost := numericValue(campaignPayload["budget"])
	if campaignCost <= 0 {
		campaignCost = numericValue(campaignPayload["cost"])
	}
	for _, lead := range leads {
		leadRevenue, convertedAt := leadWonRevenueAndDate(auth, lead)
		if leadRevenue == 0 {
			continue
		}
		totalRevenue += leadRevenue
		leadTouches := buildLeadAttributionTouches(auth, campaign, lead, convertedAt)
		weighted := weightAttributionTouches(leadTouches, convertedAt, halfLifeDays)
		for _, touch := range weighted {
			weight := numericValue(touch["weight"])
			credit := leadRevenue * weight
			channel := defaultText(fmt.Sprint(touch["channel"]), "unknown")
			channelRevenue[channel] += credit
			channelTouches[channel]++
			touches = append(touches, map[string]interface{}{
				"campaign_id":  campaign.ID,
				"lead_id":      lead.ID,
				"touch":        defaultText(fmt.Sprint(touch["touch"]), channel),
				"channel":      channel,
				"stage":        defaultText(fmt.Sprint(touch["stage"]), "unknown"),
				"weight":       weight,
				"revenue":      credit,
				"occurred_at":  touch["occurred_at"],
				"decay_factor": touch["decay_factor"],
				"source":       touch["source"],
			})
		}
	}
	if totalRevenue == 0 {
		channel := defaultText(firstPayloadText(campaignPayload, "utm_source", "source_channel", "source"), "campaign")
		channelRevenue[channel] = 0
		channelTouches[channel] = 1
		touches = append(touches, map[string]interface{}{
			"campaign_id": campaign.ID,
			"touch":       campaign.Name,
			"channel":     channel,
			"stage":       "campaign_touch",
			"weight":      1,
			"revenue":     0,
			"source":      "campaign",
		})
	}
	channels := []map[string]interface{}{}
	for channel, revenue := range channelRevenue {
		channels = append(channels, map[string]interface{}{"channel": channel, "revenue": revenue, "touches": channelTouches[channel]})
	}
	roi := 0.0
	if campaignCost > 0 {
		roi = (totalRevenue - campaignCost) / campaignCost
	}
	return touches, totalRevenue, map[string]interface{}{
		"model":          modelName,
		"half_life_days": halfLifeDays,
		"campaign_cost":  campaignCost,
		"roi":            roi,
		"channels":       channels,
		"touch_count":    len(touches),
	}
}

func leadWonRevenueAndDate(auth authContext, lead models.CRMObject) (float64, time.Time) {
	total := 0.0
	convertedAt := lead.UpdatedAt
	leadPayload := payloadMap(lead.PayloadJSON)
	if parsed, ok := parseTimeValue(leadPayload["converted_at"]); ok {
		convertedAt = parsed
	}
	opportunities := relatedObjects(auth, "leads", lead.ID, "opportunities", "converted_to")
	for _, opportunity := range opportunities {
		if opportunity.Amount > 0 && (opportunity.Status == "won" || opportunity.Status == "closed_won") {
			total += opportunity.Amount
			if opportunity.UpdatedAt.After(convertedAt) {
				convertedAt = opportunity.UpdatedAt
			}
		}
		deals := relatedObjects(auth, "opportunities", opportunity.ID, "deals", "won_as")
		for _, deal := range deals {
			if deal.Status == "closed_won" || deal.Status == "active" {
				total += deal.Amount
				if deal.UpdatedAt.After(convertedAt) {
					convertedAt = deal.UpdatedAt
				}
			}
		}
	}
	if total == 0 && lead.Amount > 0 && lead.Status == "converted" {
		total = lead.Amount
	}
	return total, convertedAt
}

func buildLeadAttributionTouches(auth authContext, campaign models.CRMObject, lead models.CRMObject, convertedAt time.Time) []map[string]interface{} {
	leadPayload := payloadMap(lead.PayloadJSON)
	touches := normalizeAttributionTrace(leadPayload["attribution_trace"])
	campaignPayload := payloadMap(campaign.PayloadJSON)
	if len(touches) == 0 {
		source := defaultText(firstPayloadText(leadPayload, "utm_source", "source_channel", "source"), firstPayloadText(campaignPayload, "utm_source", "source_channel", "source"))
		if source == "" {
			source = "campaign"
		}
		touches = append(touches, map[string]interface{}{"touch": source, "channel": source, "stage": "first_touch", "source": "utm", "occurred_at": lead.CreatedAt.Format(time.RFC3339)})
	}
	touches = append(touches, map[string]interface{}{"touch": campaign.Name, "channel": defaultText(firstPayloadText(campaignPayload, "utm_source", "source_channel", "source"), "campaign"), "stage": "lead_capture", "source": "campaign_capture", "occurred_at": lead.CreatedAt.Format(time.RFC3339)})
	if strings.TrimSpace(fmt.Sprint(leadPayload["nurture_template_key"])) != "" || lead.Status == "nurturing" {
		touches = append(touches, map[string]interface{}{"touch": defaultText(fmt.Sprint(leadPayload["nurture_template_name"]), "lead_nurture"), "channel": "nurture", "stage": "nurture", "source": "nurture", "occurred_at": defaultText(fmt.Sprint(leadPayload["nurture_started_at"]), lead.UpdatedAt.Format(time.RFC3339))})
	}
	opportunities := relatedObjects(auth, "leads", lead.ID, "opportunities", "converted_to")
	for _, opportunity := range opportunities {
		touches = append(touches, map[string]interface{}{"touch": opportunity.Name, "channel": "sales", "stage": "opportunity", "source": "opportunity", "occurred_at": opportunity.CreatedAt.Format(time.RFC3339)})
		deals := relatedObjects(auth, "opportunities", opportunity.ID, "deals", "won_as")
		for _, deal := range deals {
			touches = append(touches, map[string]interface{}{"touch": deal.Name, "channel": "sales", "stage": "closed_won", "source": "deal", "occurred_at": deal.CreatedAt.Format(time.RFC3339)})
		}
	}
	if len(touches) == 1 {
		touches = append(touches, map[string]interface{}{"touch": "conversion", "channel": "sales", "stage": "last_touch", "source": "conversion", "occurred_at": convertedAt.Format(time.RFC3339)})
	}
	return touches
}
