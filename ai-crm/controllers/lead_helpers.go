package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"time"
)

func calculateLeadScore(payload map[string]interface{}) (int, map[string]interface{}) {
	score := 20
	breakdown := map[string]interface{}{}
	email := strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["email"])))
	if strings.Contains(email, "@") {
		score += 10
		breakdown["email"] = 10
		if !strings.Contains(email, "@gmail.") && !strings.Contains(email, "@qq.") && !strings.Contains(email, "@163.") {
			score += 15
			breakdown["business_domain"] = 15
		}
	}
	title := strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["title"])))
	if title == "" {
		title = strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["position"])))
	}
	if strings.Contains(title, "总监") || strings.Contains(title, "负责人") || strings.Contains(title, "vp") || strings.Contains(title, "director") || strings.Contains(title, "manager") {
		score += 20
		breakdown["title_level"] = 20
	}
	visits := int(uintFromValue(payload["product_page_visits"]))
	if visits == 0 {
		visits = int(uintFromValue(payload["visits"]))
	}
	visitScore := visits * 4
	if visitScore > 20 {
		visitScore = 20
	}
	score += visitScore
	breakdown["product_page_visits"] = visitScore
	completeness := int(uintFromValue(payload["form_completeness"]))
	if completeness == 0 {
		completeness = formCompleteness(payload)
	}
	completenessScore := completeness / 5
	if completenessScore > 20 {
		completenessScore = 20
	}
	score += completenessScore
	breakdown["form_completeness"] = completenessScore
	if score > 100 {
		score = 100
	}
	breakdown["base"] = 20
	return score, breakdown
}

func defaultLeadViews() []map[string]interface{} {
	return []map[string]interface{}{
		{"key": "today", "name": "今日待联系", "preset": true, "filters": map[string]interface{}{"status": "following"}},
		{"key": "high_value", "name": "高价值未处理", "preset": true, "filters": map[string]interface{}{"min_score": 75, "exclude_status": []string{"converted", "invalid"}}},
		{"key": "overdue", "name": "即将超时", "preset": true, "filters": map[string]interface{}{"overdue_hours": 24, "status": []string{"pending_assign", "following", "mql", "sdr_following"}}},
		{"key": "pool", "name": "公海池", "preset": true, "filters": map[string]interface{}{"status": "pending_assign", "owner_id": 0}},
		{"key": "mql", "name": "MQL", "preset": true, "filters": map[string]interface{}{"status": "mql"}},
		{"key": "sdr", "name": "SDR Following", "preset": true, "filters": map[string]interface{}{"status": "sdr_following"}},
	}
}

func defaultLeadNurtureTemplates() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"key":         "standard_education",
			"name":        "标准产品教育（新/低分线索）",
			"description": "给刚进来或评分偏低的线索做基础教育：先准备产品价值资料，再安排短期跟进确认兴趣。",
			"preset":      true,
			"trigger":     map[string]interface{}{"status": []string{"new", "following", "mql", "sdr_following"}, "score_lt": 80},
			"nodes": []map[string]interface{}{
				{"type": "notice", "title": "准备产品价值资料和适用场景说明", "delay_days": 0},
				{"type": "task", "title": "两天后确认客户是否有初步兴趣", "delay_days": 2},
				{"type": "status", "title": "保持培育状态，继续观察行为信号", "status": "nurturing", "delay_days": 3},
			},
		},
		{
			"key":         "mql_rejected_recycle",
			"name":        "拒绝后再激活（SDR/MQL退回）",
			"description": "给已被SDR拒绝、预算不清或时机不合适的线索做再激活：记录退回原因，隔一段时间再评估。",
			"preset":      true,
			"trigger":     map[string]interface{}{"mql_rejected": true},
			"nodes": []map[string]interface{}{
				{"type": "notice", "title": "查看拒绝原因，准备行业案例或ROI说明", "delay_days": 0},
				{"type": "task", "title": "七天后重新评估是否值得再次跟进", "delay_days": 7},
				{"type": "status", "title": "保持培育状态，等待重新激活", "status": "nurturing", "delay_days": 7},
			},
		},
	}
}

func isPresetLeadNurtureTemplateKey(key string) bool {
	for _, item := range defaultLeadNurtureTemplates() {
		if strings.TrimSpace(fmt.Sprint(item["key"])) == key {
			return true
		}
	}
	return false
}

func loadCustomLeadNurtureTemplates(tenantID uint) ([]map[string]interface{}, models.CRMSetting, error) {
	var setting models.CRMSetting
	templates := []map[string]interface{}{}
	err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", tenantID, "lead_nurture", "templates").First(&setting).Error
	if err == nil {
		_ = json.Unmarshal([]byte(setting.ValueJSON), &templates)
		return templates, setting, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return templates, setting, nil
	}
	return templates, setting, err
}

func saveCustomLeadNurtureTemplates(auth authContext, setting models.CRMSetting, templates []map[string]interface{}) error {
	setting.TenantID = auth.TenantID
	setting.Scope = "lead_nurture"
	setting.Key = "templates"
	setting.ValueJSON = toJSONString(templates)
	setting.UpdatedBy = auth.UserID
	return utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "lead_nurture", "templates").Assign(setting).FirstOrCreate(&setting).Error
}

func selectLeadNurtureTemplate(auth authContext, key string) map[string]interface{} {
	key = strings.TrimSpace(key)
	for _, template := range defaultLeadNurtureTemplates() {
		if key == "" || fmt.Sprint(template["key"]) == key {
			return template
		}
	}
	var setting models.CRMSetting
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "lead_nurture", "templates").First(&setting).Error; err == nil {
		custom := []map[string]interface{}{}
		if json.Unmarshal([]byte(setting.ValueJSON), &custom) == nil {
			for _, template := range custom {
				if key == "" || fmt.Sprint(template["key"]) == key {
					return template
				}
			}
		}
	}
	return defaultLeadNurtureTemplates()[0]
}

func defaultLeadAutomationRules() map[string]interface{} {
	return map[string]interface{}{
		"sla": map[string]interface{}{
			"enabled":       true,
			"recycle_hours": 24,
			"limit":         100,
			"statuses":      []string{"new", "following", "mql", "sdr_following"},
			"reason":        "SLA超时自动回收",
		},
		"nurture": map[string]interface{}{
			"enabled":               true,
			"template_key":          "standard_education",
			"statuses":              []string{"new", "mql", "sdr_following"},
			"score_lt":              80,
			"limit":                 100,
			"cooldown_hours":        24,
			"only_without_template": true,
			"reason":                "自动进入培育",
		},
		"schedule": map[string]interface{}{
			"enabled":          false,
			"interval_minutes": 30,
		},
	}
}

func leadAutomationRules(auth authContext) map[string]interface{} {
	return normalizeLeadAutomationRules(crmSettingValue(auth.TenantID, "lead_automation", "rules"))
}

func normalizeLeadAutomationRules(raw map[string]interface{}) map[string]interface{} {
	defaults := defaultLeadAutomationRules()
	for key, defaultValue := range defaults {
		defaultSection, _ := defaultValue.(map[string]interface{})
		rawSection, _ := raw[key].(map[string]interface{})
		if rawSection == nil {
			rawSection = map[string]interface{}{}
		}
		merged := map[string]interface{}{}
		for field, value := range defaultSection {
			merged[field] = value
		}
		for field, value := range rawSection {
			merged[field] = value
		}
		defaults[key] = merged
	}
	return defaults
}

func runLeadSLARecycle(auth authContext, req leadSLARecycleRequest) (leadAutomationResult, error) {
	result := leadAutomationResult{ExecutedAt: time.Now()}
	rules := mapValue(leadAutomationRules(auth)["sla"])
	if req.Limit <= 0 {
		req.Limit = int(numericValue(rules["limit"]))
	}
	if req.Limit <= 0 || req.Limit > 200 {
		req.Limit = 100
	}
	if strings.TrimSpace(req.Reason) == "" {
		req.Reason = defaultText(fmt.Sprint(rules["reason"]), "SLA超时自动回收")
	}
	hours := req.Hours
	if hours <= 0 && !req.Force {
		hours = int(numericValue(rules["recycle_hours"]))
		if hours <= 0 {
			hours = int(numericValue(rules["default_hours"]))
		}
		if hours <= 0 {
			setting := crmSettingValue(auth.TenantID, "lead_sla", "rules")
			hours = int(numericValue(setting["recycle_hours"]))
			if hours <= 0 {
				hours = int(numericValue(setting["default_hours"]))
			}
		}
		if hours <= 0 {
			hours = 24
		}
	}
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	if req.Force {
		cutoff = time.Now().Add(time.Second)
	}
	result.Cutoff = cutoff
	statuses := stringListValue(rules["statuses"], []string{"new", "following", "mql", "sdr_following"})
	var leads []models.CRMObject
	if err := crmScopedObjectQuery(auth, "leads").
		Where("object_type = ? AND status IN ? AND updated_at < ?", "lead", statuses, cutoff).
		Order("updated_at asc").
		Limit(req.Limit).
		Find(&leads).Error; err != nil {
		return result, err
	}
	result.Scanned = len(leads)
	if req.DryRun {
		result.DryRunItems = leads
		return result, nil
	}
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		for _, lead := range leads {
			if isTerminalLeadStatus(lead.Status) {
				continue
			}
			payload := payloadMap(lead.PayloadJSON)
			payload["sla_recycled_at"] = time.Now().Format(time.RFC3339)
			payload["sla_recycle_reason"] = req.Reason
			payload["previous_owner_id"] = lead.OwnerID
			lead.OwnerID = 0
			lead.Status = "pending_assign"
			lead.PayloadJSON = toJSONString(payload)
			lead.UpdatedBy = auth.UserID
			if err := tx.Save(&lead).Error; err != nil {
				return err
			}
			if err := createLeadTimeline(tx, auth, lead, "sla_recycle", "SLA自动回收", map[string]interface{}{"reason": req.Reason, "cutoff": cutoff.Format(time.RFC3339)}); err != nil {
				return err
			}
			result.Recycled = append(result.Recycled, lead.ID)
		}
		return nil
	})
	return result, err
}

func runLeadAutomation(auth authContext, limit int, reason string, dryRun bool, force bool) (leadAutomationResult, error) {
	rules := leadAutomationRules(auth)
	result := leadAutomationResult{ExecutedAt: time.Now(), Schedule: mapValue(rules["schedule"])}
	schedule := mapValue(rules["schedule"])
	if boolValue(schedule["enabled"], false) && !force {
		interval := int(numericValue(schedule["interval_minutes"]))
		if interval <= 0 {
			interval = 30
		}
		if lastRun, ok := parseTimeValue(schedule["last_run_at"]); ok {
			next := lastRun.Add(time.Duration(interval) * time.Minute)
			result.NextRunAt = next
			if time.Now().Before(next) {
				return result, nil
			}
		}
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if strings.TrimSpace(reason) == "" {
		reason = "线索自动化规则触发"
	}
	sla := mapValue(rules["sla"])
	if boolValue(sla["enabled"], true) {
		slaResult, err := runLeadSLARecycle(auth, leadSLARecycleRequest{
			Hours:  int(numericValue(sla["recycle_hours"])),
			Limit:  limit,
			Reason: defaultText(fmt.Sprint(sla["reason"]), reason),
			DryRun: dryRun,
		})
		if err != nil {
			return result, err
		}
		result.Scanned += slaResult.Scanned
		result.Recycled = append(result.Recycled, slaResult.Recycled...)
		result.DryRunItems = append(result.DryRunItems, slaResult.DryRunItems...)
		result.Cutoff = slaResult.Cutoff
	}
	nurture := mapValue(rules["nurture"])
	if boolValue(nurture["enabled"], true) {
		nurtureResult, err := runLeadNurtureAutomation(auth, nurture, limit, reason, dryRun)
		if err != nil {
			return result, err
		}
		result.Scanned += nurtureResult.Scanned
		result.Nurtured = append(result.Nurtured, nurtureResult.Nurtured...)
		result.Created = append(result.Created, nurtureResult.Created...)
		result.DryRunItems = append(result.DryRunItems, nurtureResult.DryRunItems...)
		result.Skipped = append(result.Skipped, nurtureResult.Skipped...)
	}
	if boolValue(schedule["enabled"], false) && !dryRun {
		interval := int(numericValue(schedule["interval_minutes"]))
		if interval <= 0 {
			interval = 30
		}
		schedule["last_run_at"] = result.ExecutedAt.Format(time.RFC3339)
		schedule["next_run_at"] = result.ExecutedAt.Add(time.Duration(interval) * time.Minute).Format(time.RFC3339)
		result.NextRunAt = result.ExecutedAt.Add(time.Duration(interval) * time.Minute)
		rules["schedule"] = schedule
		_ = utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "lead_automation", "rules").
			Assign(models.CRMSetting{TenantModel: models.TenantModel{TenantID: auth.TenantID}, Scope: "lead_automation", Key: "rules", ValueJSON: toJSONString(rules), UpdatedBy: auth.UserID}).
			FirstOrCreate(&models.CRMSetting{}).Error
	}
	return result, nil
}

func runLeadNurtureAutomation(auth authContext, rules map[string]interface{}, limit int, reason string, dryRun bool) (leadAutomationResult, error) {
	result := leadAutomationResult{ExecutedAt: time.Now()}
	statuses := stringListValue(rules["statuses"], []string{"new", "mql", "sdr_following"})
	if limit <= 0 || limit > 200 {
		limit = int(numericValue(rules["limit"]))
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	templateKey := defaultText(fmt.Sprint(rules["template_key"]), "standard_education")
	template := selectLeadNurtureTemplate(auth, templateKey)
	if strings.TrimSpace(reason) == "" {
		reason = defaultText(fmt.Sprint(rules["reason"]), "自动进入培育")
	}
	var candidates []models.CRMObject
	if err := crmScopedObjectQuery(auth, "leads").
		Where("object_type = ? AND status IN ?", "lead", statuses).
		Order("score desc, updated_at asc").
		Limit(limit * 2).
		Find(&candidates).Error; err != nil {
		return result, err
	}
	scoreLT := numericValue(rules["score_lt"])
	scoreGTE := numericValue(rules["score_gte"])
	cooldownHours := int(numericValue(rules["cooldown_hours"]))
	onlyWithoutTemplate := boolValue(rules["only_without_template"], true)
	for _, lead := range candidates {
		if len(result.Nurtured)+len(result.DryRunItems) >= limit {
			break
		}
		result.Scanned++
		if isTerminalLeadStatus(lead.Status) {
			continue
		}
		payload := payloadMap(lead.PayloadJSON)
		if onlyWithoutTemplate && strings.TrimSpace(fmt.Sprint(payload["nurture_template_key"])) != "" {
			result.Skipped = append(result.Skipped, map[string]interface{}{"lead_id": lead.ID, "reason": "already_nurtured"})
			continue
		}
		if cooldownHours > 0 {
			if startedAt, ok := parseTimeValue(payload["nurture_started_at"]); ok && startedAt.Add(time.Duration(cooldownHours)*time.Hour).After(time.Now()) {
				result.Skipped = append(result.Skipped, map[string]interface{}{"lead_id": lead.ID, "reason": "cooldown"})
				continue
			}
		}
		if scoreLT > 0 && float64(lead.Score) >= scoreLT {
			result.Skipped = append(result.Skipped, map[string]interface{}{"lead_id": lead.ID, "reason": "score_too_high"})
			continue
		}
		if scoreGTE > 0 && float64(lead.Score) < scoreGTE {
			result.Skipped = append(result.Skipped, map[string]interface{}{"lead_id": lead.ID, "reason": "score_too_low"})
			continue
		}
		if dryRun {
			result.DryRunItems = append(result.DryRunItems, lead)
			continue
		}
		created, err := applyLeadNurture(auth, &lead, leadNurtureRequest{TemplateKey: templateKey, Reason: reason}, template, nil)
		if err != nil {
			return result, err
		}
		result.Nurtured = append(result.Nurtured, lead.ID)
		result.Created = append(result.Created, created...)
	}
	return result, nil
}

func applyLeadNurture(auth authContext, lead *models.CRMObject, req leadNurtureRequest, template map[string]interface{}, tx *gorm.DB) ([]models.CRMObject, error) {
	if isTerminalLeadStatus(lead.Status) {
		return nil, fmt.Errorf("终态线索不能进入培育")
	}
	if strings.TrimSpace(req.Reason) == "" {
		if req.Exit {
			req.Reason = "退出培育"
		} else {
			req.Reason = "进入培育"
		}
	}
	if len(template) == 0 {
		template = selectLeadNurtureTemplate(auth, req.TemplateKey)
	}
	created := []models.CRMObject{}
	run := func(inner *gorm.DB) error {
		payload := payloadMap(lead.PayloadJSON)
		if req.Exit {
			payload["nurture_exited_at"] = time.Now().Format(time.RFC3339)
			payload["nurture_exit_reason"] = req.Reason
			lead.Status = "following"
			lead.PayloadJSON = toJSONString(payload)
			lead.UpdatedBy = auth.UserID
			if err := inner.Save(lead).Error; err != nil {
				return err
			}
			return createLeadTimeline(inner, auth, *lead, "nurture_exit", "退出培育", map[string]interface{}{"reason": req.Reason})
		}
		nodes := templateNodes(template)
		payload["nurture_template_key"] = template["key"]
		payload["nurture_template_name"] = template["name"]
		payload["nurture_started_at"] = time.Now().Format(time.RFC3339)
		payload["nurture_reason"] = req.Reason
		if len(nodes) > 0 {
			payload["nurture_node"] = defaultText(fmt.Sprint(nodes[0]["title"]), fmt.Sprint(nodes[0]["type"]))
		}
		if isMQLRejectedReason(req.Reason) {
			payload["mql_rejected"] = true
			payload["mql_rejected_reason"] = req.Reason
			payload["mql_rejected_at"] = time.Now().Format(time.RFC3339)
			payload["mql_rejected_by"] = auth.UserID
		} else {
			delete(payload, "mql_rejected")
			delete(payload, "mql_rejected_reason")
			delete(payload, "mql_rejected_at")
			delete(payload, "mql_rejected_by")
		}
		lead.Status = "nurturing"
		lead.PayloadJSON = toJSONString(payload)
		lead.UpdatedBy = auth.UserID
		if err := inner.Save(lead).Error; err != nil {
			return err
		}
		for _, node := range nodes {
			nodeType := strings.ToLower(strings.TrimSpace(fmt.Sprint(node["type"])))
			title := defaultText(fmt.Sprint(node["title"]), "培育节点")
			delayDays := int(numericValue(node["delay_days"]))
			switch nodeType {
			case "task":
				task := cloneCRMObject(auth, "workspace", "task", "培育任务："+lead.Name+"："+title, *lead)
				task.Status = "open"
				task.PayloadJSON = toJSONString(map[string]interface{}{"source": "lead_nurture", "lead_id": lead.ID, "node": node, "due_at": time.Now().AddDate(0, 0, delayDays).Format(time.RFC3339)})
				if err := inner.Create(&task).Error; err != nil {
					return err
				}
				if err := inner.Create(&[]models.CRMRelation{newCRMRelation(auth, "leads", lead.ID, "workspace", task.ID, "nurture_task")}).Error; err != nil {
					return err
				}
				created = append(created, task)
			case "notice", "email":
				notice := cloneCRMObject(auth, "workspace", "customer_activity", "培育提醒："+lead.Name+"："+title, *lead)
				notice.Status = "open"
				notice.PayloadJSON = toJSONString(map[string]interface{}{"source": "lead_nurture", "lead_id": lead.ID, "node": node, "category": "nurture_notice", "content": title, "due_at": time.Now().AddDate(0, 0, delayDays).Format(time.RFC3339)})
				if err := inner.Create(&notice).Error; err != nil {
					return err
				}
				publishSystemNotificationUnread(auth, notice.OwnerID)
				if err := inner.Create(&[]models.CRMRelation{newCRMRelation(auth, "leads", lead.ID, "workspace", notice.ID, "nurture_notice")}).Error; err != nil {
					return err
				}
				created = append(created, notice)
			default:
				interaction := cloneCRMObject(auth, "contacts", "interaction", "培育记录："+lead.Name+"："+title, *lead)
				interaction.Status = "active"
				interaction.PayloadJSON = toJSONString(map[string]interface{}{"interaction_type": defaultText(nodeType, "nurture"), "summary": title, "source_module": "leads", "related_object_id": lead.ID, "happened_at": time.Now().Format(time.RFC3339)})
				if err := inner.Create(&interaction).Error; err != nil {
					return err
				}
				if err := inner.Create(&[]models.CRMRelation{newCRMRelation(auth, "leads", lead.ID, "contacts", interaction.ID, "timeline")}).Error; err != nil {
					return err
				}
				created = append(created, interaction)
			}
		}
		return createLeadTimeline(inner, auth, *lead, "nurture_start", "进入培育", map[string]interface{}{"template": template["name"], "reason": req.Reason, "nodes": nodes})
	}
	if tx != nil {
		return created, run(tx)
	}
	err := utils.DB.Transaction(run)
	return created, err
}

func normalizeLeadStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	aliases := map[string]string{
		"assigned":  "following",
		"qualified": "mql",
		"recycled":  "pending_assign",
		"invalid":   "invalid",
	}
	if value, ok := aliases[status]; ok {
		status = value
	}
	for _, allowed := range []string{"new", "pending_assign", "following", "mql", "sdr_following", "sql", "nurturing", "converted", "invalid"} {
		if status == allowed {
			return status
		}
	}
	return ""
}

func isTerminalLeadStatus(status string) bool {
	status = normalizeLeadStatus(status)
	return status == "converted" || status == "invalid"
}

func leadTemperature(lead models.CRMObject) string {
	payload := payloadMap(lead.PayloadJSON)
	if text := strings.TrimSpace(fmt.Sprint(payload["temperature"])); text != "" {
		return text
	}
	return scoreToTemperature(lead.Score)
}

func createLeadTimeline(tx *gorm.DB, auth authContext, lead models.CRMObject, eventType string, summary string, payload map[string]interface{}) error {
	if payload == nil {
		payload = map[string]interface{}{}
	}
	payload["event_type"] = eventType
	payload["summary"] = summary
	payload["source_module"] = "leads"
	payload["related_object_id"] = lead.ID
	operatedAt := time.Now().Format(time.RFC3339)
	payload["happened_at"] = operatedAt
	payload["operated_at"] = operatedAt
	payload["operator_id"] = auth.UserID
	payload["operator_name"] = defaultText(auth.Username, fmt.Sprintf("用户#%d", auth.UserID))
	item := cloneCRMObject(auth, "contacts", "interaction", summary+"："+lead.Name, lead)
	item.Status = "active"
	item.PayloadJSON = toJSONString(payload)
	if err := tx.Create(&item).Error; err != nil {
		return err
	}
	relation := newCRMRelation(auth, "leads", lead.ID, "contacts", item.ID, "timeline")
	return tx.Create(&relation).Error
}

type leadDedupeInput struct {
	ExcludeID            uint
	CompanyID            uint
	ContactID            uint
	CompanyName          string
	ContactName          string
	Phone                string
	Email                string
	IntentProductID      uint
	IntentProduct        string
	SourceChannel        string
	UTMSource            string
	CampaignID           uint
	ExternalSubmissionID string
	ImportBatchID        string
	Note                 string
}

func findLeadDuplicates(auth authContext, excludeID uint, companyName string, contactName string, phone string, email string) []models.CRMObject {
	possibleDuplicates, _, _, _ := findLeadDedupeMatches(auth, leadDedupeInput{
		ExcludeID:   excludeID,
		CompanyName: companyName,
		ContactName: contactName,
		Phone:       phone,
		Email:       email,
	})
	return possibleDuplicates
}

func validateLeadRequiredFields(lead models.CRMObject) error {
	if lead.Domain != "leads" || lead.ObjectType != "lead" {
		return nil
	}
	payload := payloadMap(lead.PayloadJSON)
	missing := []string{}
	if strings.TrimSpace(firstPayloadText(payload, "company_name", "company")) == "" {
		missing = append(missing, "公司")
	}
	if strings.TrimSpace(firstPayloadText(payload, "contact_name", "contact", "name")) == "" {
		missing = append(missing, "联系人")
	}
	if strings.TrimSpace(firstPayloadText(payload, "phone", "mobile")) == "" && strings.TrimSpace(firstPayloadText(payload, "email")) == "" {
		missing = append(missing, "手机或邮箱")
	}
	if strings.TrimSpace(firstPayloadText(payload, "source_channel", "source", "utm_source")) == "" {
		missing = append(missing, "来源")
	}
	if leadPrimaryIntentProductID(payload) == 0 && strings.TrimSpace(firstPayloadText(payload, "intent_product", "product_name")) == "" && len(leadIntentProductNames(payload)) == 0 {
		missing = append(missing, "意向产品")
	}
	if lead.OwnerID == 0 {
		missing = append(missing, "负责人")
	}
	if len(missing) > 0 {
		return fmt.Errorf("线索缺少必填信息：%s", strings.Join(missing, "、"))
	}
	return nil
}

func findLeadDedupeMatches(auth authContext, input leadDedupeInput) ([]models.CRMObject, []models.CRMObject, []models.CRMObject, map[string]interface{}) {
	candidates := findLeadHistoryCandidates(auth, input, 100)
	possible := make([]models.CRMObject, 0)
	sameContact := make([]models.CRMObject, 0)
	sameCompany := make([]models.CRMObject, 0)
	possibleIDs := map[uint]bool{}
	contactIDs := map[uint]bool{}
	companyIDs := map[uint]bool{}
	duplicateReasons := map[uint][]string{}
	for _, item := range candidates {
		payload := payloadMap(item.PayloadJSON)
		contactMatched := leadContactMatched(input, item, payload)
		companyMatched := leadCompanyMatched(input, item, payload)
		reasons := leadDuplicateReasons(input, item, payload, contactMatched)
		if len(reasons) > 0 && !possibleIDs[item.ID] {
			possible = append(possible, item)
			possibleIDs[item.ID] = true
			duplicateReasons[item.ID] = reasons
		}
		if contactMatched && !contactIDs[item.ID] {
			sameContact = append(sameContact, item)
			contactIDs[item.ID] = true
		}
		if companyMatched && !companyIDs[item.ID] {
			sameCompany = append(sameCompany, item)
			companyIDs[item.ID] = true
		}
	}
	summary := map[string]interface{}{
		"rules_version":             "lead-dedupe-v2",
		"duplicate_window_days":     7,
		"possible_duplicate_count":  len(possible),
		"same_contact_lead_count":   len(sameContact),
		"same_company_lead_count":   len(sameCompany),
		"contact_identity_is_hint":  true,
		"company_identity_is_hint":  true,
		"duplicate_requires_intent": true,
	}
	if len(duplicateReasons) > 0 {
		summary["duplicate_reasons"] = duplicateReasons
	}
	return possible, sameContact, sameCompany, summary
}

func findLeadHistoryCandidates(auth authContext, input leadDedupeInput, limit int) []models.CRMObject {
	query := crmScopedObjectQuery(auth, "leads").Where("object_type = ?", "lead")
	if input.ExcludeID > 0 {
		query = query.Where("id <> ?", input.ExcludeID)
	}
	conditions := []string{}
	args := []interface{}{}
	if input.CompanyID > 0 {
		conditions = append(conditions, "company_id = ?")
		args = append(args, input.CompanyID)
		if clause, clauseArgs := payloadJSONKeyNumberLikeClause(input.CompanyID, "company_id"); clause != "" {
			conditions = append(conditions, clause)
			args = append(args, clauseArgs...)
		}
	}
	if input.ContactID > 0 {
		conditions = append(conditions, "contact_id = ?")
		args = append(args, input.ContactID)
		if clause, clauseArgs := payloadJSONKeyNumberLikeClause(input.ContactID, "contact_id"); clause != "" {
			conditions = append(conditions, clause)
			args = append(args, clauseArgs...)
		}
	}
	if clause, clauseArgs := payloadJSONKeyLikeClause(input.CompanyName, "company_name", "company"); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if clause, clauseArgs := payloadJSONKeyLikeClause(input.ContactName, "contact_name", "contact"); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if clause, clauseArgs := payloadJSONKeyLikeClause(input.Phone, "phone", "mobile"); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if clause, clauseArgs := payloadJSONKeyLikeClause(input.Email, "email"); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if clause, clauseArgs := payloadJSONKeyLikeClause(input.ExternalSubmissionID, "external_submission_id", "submission_id", "form_submission_id"); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if len(conditions) == 0 {
		return []models.CRMObject{}
	}
	var items []models.CRMObject
	if limit <= 0 {
		limit = 50
	}
	_ = query.Where(strings.Join(conditions, " OR "), args...).Order("created_at desc").Limit(limit).Find(&items).Error
	return items
}

func leadDuplicateReasons(input leadDedupeInput, lead models.CRMObject, payload map[string]interface{}, contactMatched bool) []string {
	reasons := []string{}
	if sameLeadExternalSubmission(input, payload) {
		return append(reasons, "external_submission_id")
	}
	if !contactMatched {
		return reasons
	}
	if leadCreatedOutsideDuplicateWindow(lead, 7) {
		return reasons
	}
	intentMatched := sameLeadIntent(input, payload)
	sourceMatched := sameLeadSource(input, payload)
	batchMatched := sameLeadImportBatch(input, payload)
	noteMatched := similarLeadNote(input.Note, firstPayloadText(payload, "note", "requirement", "summary", "description"))
	if intentMatched && sourceMatched {
		reasons = append(reasons, "same_contact_same_intent_same_source_7d")
	}
	if batchMatched && noteMatched {
		reasons = append(reasons, "same_contact_same_batch_similar_note_7d")
	}
	return reasons
}

func leadContactMatched(input leadDedupeInput, lead models.CRMObject, payload map[string]interface{}) bool {
	if input.ContactID > 0 && (lead.ContactID == input.ContactID || uintFromValue(payload["contact_id"]) == input.ContactID) {
		return true
	}
	if normalizedDedupeText(input.Email) != "" && normalizedDedupeText(input.Email) == normalizedDedupeText(firstPayloadText(payload, "email")) {
		return true
	}
	if normalizedDedupeText(input.Phone) != "" && normalizedDedupeText(input.Phone) == normalizedDedupeText(firstPayloadText(payload, "phone", "mobile")) {
		return true
	}
	if normalizedDedupeText(input.ContactName) != "" && normalizedDedupeText(input.ContactName) == normalizedDedupeText(firstPayloadText(payload, "contact_name", "contact")) {
		return true
	}
	return false
}

func leadCompanyMatched(input leadDedupeInput, lead models.CRMObject, payload map[string]interface{}) bool {
	if input.CompanyID > 0 && (lead.CompanyID == input.CompanyID || uintFromValue(payload["company_id"]) == input.CompanyID) {
		return true
	}
	if normalizedDedupeText(input.CompanyName) != "" && normalizedDedupeText(input.CompanyName) == normalizedDedupeText(firstPayloadText(payload, "company_name", "company")) {
		return true
	}
	return false
}

func sameLeadExternalSubmission(input leadDedupeInput, payload map[string]interface{}) bool {
	value := normalizedDedupeText(input.ExternalSubmissionID)
	if value == "" {
		return false
	}
	other := normalizedDedupeText(firstPayloadText(payload, "external_submission_id", "submission_id", "form_submission_id"))
	return other != "" && other == value
}

func sameLeadIntent(input leadDedupeInput, payload map[string]interface{}) bool {
	if input.IntentProductID > 0 {
		for _, item := range leadIntentProductIDs(payload) {
			if item == input.IntentProductID {
				return true
			}
		}
	}
	value := normalizedDedupeText(input.IntentProduct)
	if value == "" {
		return false
	}
	for _, name := range leadIntentProductNames(payload) {
		if normalizedDedupeText(name) == value {
			return true
		}
	}
	other := normalizedDedupeText(firstPayloadText(payload, "intent_product", "product_name"))
	return other != "" && other == value
}

func leadPrimaryIntentProductID(payload map[string]interface{}) uint {
	ids := leadIntentProductIDs(payload)
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

func leadIntentProductIDs(payload map[string]interface{}) []uint {
	ids := []uint{}
	if id := uintFromValue(firstNonNil(payload["intent_product_id"], payload["product_id"])); id > 0 {
		ids = append(ids, id)
	}
	for _, item := range interfaceList(firstNonNil(payload["intent_product_ids"], payload["product_ids"])) {
		if id := uintFromValue(item); id > 0 && !uintInList(id, ids) {
			ids = append(ids, id)
		}
	}
	return ids
}

func leadIntentProductNames(payload map[string]interface{}) []string {
	names := []string{}
	for _, item := range interfaceList(firstNonNil(payload["intent_products"], payload["product_names"])) {
		text := strings.TrimSpace(fmt.Sprint(item))
		if text != "" && text != "<nil>" {
			names = append(names, text)
		}
	}
	return names
}

func sameLeadSource(input leadDedupeInput, payload map[string]interface{}) bool {
	if input.CampaignID > 0 {
		otherID := uintFromValue(payload["campaign_id"])
		if otherID > 0 {
			return otherID == input.CampaignID
		}
	}
	for _, pair := range [][2]string{
		{input.UTMSource, firstPayloadText(payload, "utm_source")},
		{input.SourceChannel, firstPayloadText(payload, "source_channel", "source")},
	} {
		value := normalizedDedupeText(pair[0])
		other := normalizedDedupeText(pair[1])
		if value != "" && other != "" && value == other {
			return true
		}
	}
	return false
}

func sameLeadImportBatch(input leadDedupeInput, payload map[string]interface{}) bool {
	value := normalizedDedupeText(input.ImportBatchID)
	if value == "" {
		return false
	}
	other := normalizedDedupeText(firstPayloadText(payload, "import_batch_id", "batch_id"))
	return other != "" && other == value
}

func similarLeadNote(current string, other string) bool {
	current = normalizedDedupeText(current)
	other = normalizedDedupeText(other)
	if len(current) < 8 || len(other) < 8 {
		return false
	}
	return current == other || strings.Contains(current, other) || strings.Contains(other, current)
}

func leadCreatedOutsideDuplicateWindow(lead models.CRMObject, days int) bool {
	if days <= 0 || lead.CreatedAt.IsZero() {
		return false
	}
	return lead.CreatedAt.Before(time.Now().AddDate(0, 0, -days))
}

func normalizedDedupeText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "　", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	return value
}
