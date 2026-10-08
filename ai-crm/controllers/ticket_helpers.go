package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"time"
)

var ticketSLANoticeSchedulerStop chan struct{}

func normalizeTicketStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	aliases := map[string]string{
		"open":       "new",
		"processing": "processing",
		"done":       "resolved",
		"reopen":     "reopened",
	}
	if value, ok := aliases[status]; ok {
		status = value
	}
	for _, allowed := range []string{"new", "assigned", "processing", "waiting_customer", "resolved", "closed", "reopened"} {
		if status == allowed {
			return status
		}
	}
	return ""
}

func normalizeTicketServiceRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	aliases := map[string]string{
		"l1":              "l1_support",
		"first_line":      "l1_support",
		"first-line":      "l1_support",
		"一线":              "l1_support",
		"l2":              "l2_support",
		"second_line":     "l2_support",
		"second-line":     "l2_support",
		"二线":              "l2_support",
		"pm":              "product_manager",
		"product":         "product_manager",
		"product_manager": "product_manager",
		"产品经理":            "product_manager",
	}
	if value, ok := aliases[role]; ok {
		role = value
	}
	for _, allowed := range []string{"l1_support", "l2_support", "product_manager"} {
		if role == allowed {
			return role
		}
	}
	return ""
}

func ticketServiceRole(payload map[string]interface{}) string {
	if role := normalizeTicketServiceRole(fmt.Sprint(payload["service_role"])); role != "" {
		return role
	}
	if role := normalizeTicketServiceRole(fmt.Sprint(payload["support_queue"])); role != "" {
		return role
	}
	if boolValue(payload["need_pm"], false) {
		return "product_manager"
	}
	if boolValue(payload["need_l2"], false) || boolValue(payload["need_tech"], false) {
		return "l2_support"
	}
	text := strings.ToLower(strings.Join([]string{
		fmt.Sprint(payload["category"]),
		fmt.Sprint(payload["product_line"]),
		fmt.Sprint(payload["module"]),
		fmt.Sprint(payload["description"]),
	}, " "))
	for _, keyword := range []string{"产品", "需求", "功能规划", "roadmap", "feature request", "pricing"} {
		if strings.Contains(text, keyword) {
			return "product_manager"
		}
	}
	for _, keyword := range []string{"bug", "报错", "故障", "接口", "api", "研发", "技术", "系统异常"} {
		if strings.Contains(text, keyword) {
			return "l2_support"
		}
	}
	return "l1_support"
}

func applyTicketServiceRole(payload map[string]interface{}) string {
	role := ticketServiceRole(payload)
	payload["service_role"] = role
	payload["support_queue"] = role
	return role
}

func syncPortalTicketTx(tx *gorm.DB, ticket models.CRMObject, reply string, status string) error {
	payload := payloadMap(ticket.PayloadJSON)
	trackingID := strings.TrimSpace(fmt.Sprint(payload["tracking_id"]))
	if trackingID == "" {
		trackingID = strings.TrimSpace(ticket.Code)
	}
	if trackingID == "" {
		return nil
	}
	var portalTicket models.PortalTicket
	if err := tx.Where("tracking_id = ?", trackingID).First(&portalTicket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	reply = strings.TrimSpace(reply)
	if reply != "" {
		if strings.TrimSpace(portalTicket.Replies) == "" {
			portalTicket.Replies = reply
		} else {
			portalTicket.Replies = portalTicket.Replies + "\n" + time.Now().Format("2006-01-02 15:04") + " " + reply
		}
	}
	if strings.TrimSpace(status) != "" {
		portalTicket.Status = strings.TrimSpace(status)
	}
	return tx.Save(&portalTicket).Error
}

func applyTicketSLA(auth authContext, payload map[string]interface{}) {
	setting := crmSettingValue(auth.TenantID, "sla_rule", "tickets")
	defaultHours := uintFromValue(setting["default_hours"])
	if defaultHours == 0 {
		defaultHours = 24
	}
	responseHours := uintFromValue(setting["response_hours"])
	if responseHours == 0 {
		responseHours = defaultHours / 4
		if responseHours == 0 {
			responseHours = 1
		}
	}
	category := strings.TrimSpace(fmt.Sprint(payload["category"]))
	grade := strings.TrimSpace(fmt.Sprint(payload["customer_grade"]))
	priority := strings.TrimSpace(fmt.Sprint(payload["priority"]))
	key := category
	if grade != "" {
		key = category + ":" + grade
	}
	matrix, _ := setting["matrix"].(map[string]interface{})
	if matrixHours := uintFromValue(matrix[key]); matrixHours > 0 {
		defaultHours = matrixHours
	} else if categoryHours := uintFromValue(matrix[category]); categoryHours > 0 {
		defaultHours = categoryHours
	}
	if priority == "high" || priority == "urgent" || grade == "VIP" || grade == "vip" {
		if defaultHours > 4 {
			defaultHours = defaultHours / 2
		}
		if responseHours > 1 {
			responseHours = responseHours / 2
		}
	}
	if strings.TrimSpace(fmt.Sprint(payload["sla_response_due_at"])) == "" {
		payload["sla_response_due_at"] = time.Now().Add(time.Duration(responseHours) * time.Hour).Format(time.RFC3339)
	}
	if strings.TrimSpace(fmt.Sprint(payload["sla_due_at"])) == "" {
		payload["sla_due_at"] = time.Now().Add(time.Duration(defaultHours) * time.Hour).Format(time.RFC3339)
	}
	payload["sla_rule_summary"] = fmt.Sprintf("按客户等级和紧急程度计算：%d小时内响应，%d小时内解决", responseHours, defaultHours)
}

func pickTicketAssignee(auth authContext, ticket models.CRMObject, serviceRole string) (uint, string) {
	payload := payloadMap(ticket.PayloadJSON)
	if normalized := normalizeTicketServiceRole(serviceRole); normalized != "" {
		serviceRole = normalized
	} else {
		serviceRole = ticketServiceRole(payload)
	}
	category := strings.TrimSpace(fmt.Sprint(payload["category"]))
	productLine := strings.TrimSpace(fmt.Sprint(payload["product_line"]))
	if productLine == "" {
		productLine = strings.TrimSpace(fmt.Sprint(payload["product"]))
	}
	grade := strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["customer_grade"])))
	language := strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["language"])))
	if language == "" {
		language = strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["locale"])))
	}
	setting := crmSettingValue(auth.TenantID, "ticket_assignment", "rules")
	languageOwners := mapValue(setting["language_owner_ids"])
	if language != "" {
		if ownerID := uintFromValue(languageOwners[language]); ownerID > 0 && tenantUserExists(auth.TenantID, ownerID) {
			return ownerID, fmt.Sprintf("按服务语言「%s」自动分派", language)
		}
	}
	categoryOwners := mapValue(setting["category_owner_ids"])
	if ownerID := uintFromValue(categoryOwners[category]); ownerID > 0 && tenantUserExists(auth.TenantID, ownerID) {
		return ownerID, fmt.Sprintf("按问题类型「%s」自动分派", category)
	}
	productLineOwners := mapValue(setting["product_line_owner_ids"])
	if ownerID := uintFromValue(productLineOwners[productLine]); ownerID > 0 && tenantUserExists(auth.TenantID, ownerID) {
		return ownerID, fmt.Sprintf("按产品线「%s」自动分派", productLine)
	}
	gradeOwners := mapValue(setting["grade_owner_ids"])
	if ownerID := uintFromValue(gradeOwners[grade]); ownerID > 0 && tenantUserExists(auth.TenantID, ownerID) {
		return ownerID, fmt.Sprintf("按客户等级「%s」自动分派", grade)
	}
	serviceRoleOwners := mapValue(setting["service_role_owner_ids"])
	if ownerID := uintFromValue(serviceRoleOwners[serviceRole]); ownerID > 0 && tenantUserExists(auth.TenantID, ownerID) {
		return ownerID, fmt.Sprintf("按服务角色「%s」自动分派", ticketServiceRoleLabel(serviceRole))
	}
	candidateIDs := numberList(setting["candidate_owner_ids"])
	roleCandidates := mapValue(setting["service_role_candidate_owner_ids"])
	if roleCandidateIDs := numberList(roleCandidates[serviceRole]); len(roleCandidateIDs) > 0 {
		candidateIDs = roleCandidateIDs
	}
	if len(candidateIDs) == 0 {
		var users []models.User
		_ = utils.DB.Where("tenant_id = ? AND status = ?", auth.TenantID, "active").Order("id asc").Limit(50).Find(&users).Error
		for _, user := range users {
			if user.AccountType == "tenant_admin" || user.Role == "admin" {
				continue
			}
			candidateIDs = append(candidateIDs, user.ID)
		}
	}
	var selected uint
	minLoad := int64(1<<62 - 1)
	for _, candidateID := range candidateIDs {
		if !tenantUserExists(auth.TenantID, candidateID) {
			continue
		}
		var load int64
		_ = utils.DB.Model(&models.CRMObject{}).
			Where("tenant_id = ? AND domain = ? AND object_type = ? AND owner_id = ? AND status IN ?", auth.TenantID, "tickets", "ticket", candidateID, []string{"new", "assigned", "processing", "waiting_customer", "reopened"}).
			Count(&load).Error
		if selected == 0 || load < minLoad {
			selected = candidateID
			minLoad = load
		}
	}
	if selected == 0 {
		return 0, ""
	}
	return selected, fmt.Sprintf("按服务角色「%s」和当前待处理工单数量自动分派给负载较低的人", ticketServiceRoleLabel(serviceRole))
}

func ticketServiceRoleLabel(role string) string {
	switch normalizeTicketServiceRole(role) {
	case "l1_support":
		return "一线"
	case "l2_support":
		return "二线"
	case "product_manager":
		return "产品经理"
	default:
		return "待指定"
	}
}

func autoSyncTicketFromBugTaskTx(tx *gorm.DB, auth authContext, before models.CRMObject, after models.CRMObject) error {
	if after.Domain != "projects" || after.ObjectType != "bug_task" {
		return nil
	}
	beforePayload := payloadMap(before.PayloadJSON)
	afterPayload := payloadMap(after.PayloadJSON)
	prevBugStatus := normalizeBugStatus(strings.TrimSpace(before.Status))
	if prevBugStatus == "" {
		prevBugStatus = normalizeBugStatus(strings.TrimSpace(fmt.Sprint(beforePayload["status"])))
	}
	nextBugStatus := normalizeBugStatus(strings.TrimSpace(after.Status))
	if nextBugStatus == "" {
		nextBugStatus = normalizeBugStatus(strings.TrimSpace(fmt.Sprint(afterPayload["status"])))
	}
	if nextBugStatus == "" || nextBugStatus == prevBugStatus {
		return nil
	}
	ticketIDSet := map[uint]struct{}{}
	var relations []models.CRMRelation
	if err := tx.Where("tenant_id = ? AND relation_type = ? AND source_type = ? AND target_type = ? AND target_id = ?", auth.TenantID, "ticket_bug", "tickets", "projects", after.ID).Find(&relations).Error; err != nil {
		return err
	}
	for _, relation := range relations {
		if relation.SourceID > 0 {
			ticketIDSet[relation.SourceID] = struct{}{}
		}
	}
	var payloadMatched []models.CRMObject
	bugTaskClause, bugTaskArgs := payloadJSONKeyNumberLikeClause(after.ID, "bug_task_id")
	if bugTaskClause != "" {
		if err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ?", auth.TenantID, "tickets", "ticket").
			Where(bugTaskClause, bugTaskArgs...).
			Find(&payloadMatched).Error; err != nil {
			return err
		}
	}
	if len(payloadMatched) > 0 {
		for _, item := range payloadMatched {
			ticketIDSet[item.ID] = struct{}{}
		}
	}
	if len(ticketIDSet) == 0 {
		return nil
	}
	now := time.Now().Format(time.RFC3339)
	for ticketID := range ticketIDSet {
		var ticket models.CRMObject
		if err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "tickets", "ticket", ticketID).First(&ticket).Error; err != nil {
			continue
		}
		ticketPayload := payloadMap(ticket.PayloadJSON)
		beforeTicketStatus := ticket.Status
		nextTicketStatus := beforeTicketStatus
		if bugStatusDone(nextBugStatus) && beforeTicketStatus != "closed" {
			if beforeTicketStatus == "new" || beforeTicketStatus == "assigned" || beforeTicketStatus == "processing" || beforeTicketStatus == "reopened" {
				nextTicketStatus = "waiting_customer"
			}
		}
		if nextBugStatus == "reopened" && beforeTicketStatus != "closed" {
			nextTicketStatus = "processing"
		}
		ticketPayload["bug_status"] = nextBugStatus
		ticketPayload["bug_previous_status"] = prevBugStatus
		ticketPayload["bug_status_reason"] = "研发任务状态自动回写"
		ticketPayload["bug_updated_at"] = now
		ticketPayload["bug_sync_at"] = now
		ticketPayload["bug_sync_mode"] = "auto"
		ticketPayload["bug_task_status_raw"] = after.Status
		ticket.Status = nextTicketStatus
		ticket.PayloadJSON = toJSONString(ticketPayload)
		ticket.UpdatedBy = auth.UserID
		if err := tx.Save(&ticket).Error; err != nil {
			return err
		}
		if nextTicketStatus != beforeTicketStatus {
			if err := syncPortalTicketTx(tx, ticket, "", nextTicketStatus); err != nil {
				return err
			}
		}
		if err := createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_bug_status_auto_sync", "研发Bug状态自动回写", map[string]interface{}{"reason": "研发任务状态更新后自动回写", "bug_task_id": after.ID, "bug_status_from": prevBugStatus, "bug_status_to": nextBugStatus, "ticket_from": beforeTicketStatus, "ticket_to": nextTicketStatus}); err != nil {
			return err
		}
	}
	return nil
}

func ticketEscalationReason(payload map[string]interface{}, overdue bool, responseOverdue bool) string {
	if overdue {
		return "解决SLA已超时"
	}
	if responseOverdue {
		return "首次响应SLA已超时"
	}
	priority := strings.TrimSpace(fmt.Sprint(payload["priority"]))
	if priority == "urgent" || priority == "high" {
		return "高优先级工单需要主管关注"
	}
	return "需要关注"
}

func ticketKnowledgeKeywords(ticket models.CRMObject, payload map[string]interface{}) []string {
	candidates := []string{
		strings.TrimSpace(fmt.Sprint(payload["category"])),
		strings.TrimSpace(fmt.Sprint(payload["description"])),
		strings.TrimSpace(ticket.Name),
	}
	keywords := []string{}
	for _, value := range candidates {
		for _, part := range strings.FieldsFunc(value, func(r rune) bool {
			return r == ' ' || r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' || r == '\t'
		}) {
			part = strings.TrimSpace(part)
			if len([]rune(part)) >= 2 && !stringInSlice(part, keywords) {
				keywords = append(keywords, part)
			}
			if len(keywords) >= 3 {
				return keywords
			}
		}
	}
	if len(keywords) == 0 {
		keywords = append(keywords, ticket.Name)
	}
	return keywords
}

type ticketSLANoticeResult struct {
	Scanned      int
	Notified     []uint
	Skipped      []map[string]interface{}
	Now          time.Time
	WindowMinute int
}

func runTicketSLANoticeSchedule() {
	var settings []models.CRMSetting
	if err := utils.DB.Where("scope = ? AND `key` = ?", "sla_rule", "tickets").Find(&settings).Error; err != nil {
		utils.LogWarning("SLA提醒调度读取规则失败: %v", err)
		return
	}
	for _, setting := range settings {
		rules := payloadMap(setting.ValueJSON)
		notify := mapValue(rules["notify"])
		if !boolValue(notify["enabled"], true) {
			continue
		}
		tenantID := setting.TenantID
		if tenantID == 0 {
			tenantID = defaultTenantID()
		}
		auth := authContext{TenantID: tenantID, UserID: setting.UpdatedBy, Username: "system", Role: "tenant_admin", AccountType: "tenant_admin"}
		result, err := runTicketSLANotice(auth)
		if err != nil {
			utils.LogWarning("SLA提醒调度失败 tenant=%d: %v", tenantID, err)
			services.LogOperation(services.OperationLogEvent{
				TenantID:       tenantID,
				UserID:         auth.UserID,
				Username:       "system",
				Operation:      "RUN_TICKET_SLA_NOTICE_SCHEDULE",
				Target:         "工单SLA提醒",
				DescriptionKey: "operation.run_ticket_sla_notice_schedule",
				Result:         "failed",
				Reason:         err.Error(),
				Source:         "scheduler",
			})
			continue
		}
		if len(result.Notified) == 0 {
			continue
		}
		services.LogOperation(services.OperationLogEvent{
			TenantID:       tenantID,
			UserID:         auth.UserID,
			Username:       "system",
			Operation:      "RUN_TICKET_SLA_NOTICE_SCHEDULE",
			Target:         "工单SLA提醒",
			DescriptionKey: "operation.run_ticket_sla_notice_schedule",
			Params: map[string]interface{}{
				"scanned":  result.Scanned,
				"notified": result.Notified,
				"window":   result.WindowMinute,
			},
			Result: "success",
			Source: "scheduler",
		})
	}
}

func StartTicketSLANoticeScheduler() {
	if ticketSLANoticeSchedulerStop != nil {
		return
	}
	ticketSLANoticeSchedulerStop = make(chan struct{})
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		services.WithSchedulerLock("ticket_sla_notice_scheduler", runTicketSLANoticeSchedule)
		for {
			select {
			case <-ticker.C:
				services.WithSchedulerLock("ticket_sla_notice_scheduler", runTicketSLANoticeSchedule)
			case <-ticketSLANoticeSchedulerStop:
				ticker.Stop()
				return
			}
		}
	}()
}

func StopTicketSLANoticeScheduler() {
	if ticketSLANoticeSchedulerStop == nil {
		return
	}
	close(ticketSLANoticeSchedulerStop)
	ticketSLANoticeSchedulerStop = nil
}

func runTicketSLANotice(auth authContext) (ticketSLANoticeResult, error) {
	result := ticketSLANoticeResult{
		Now:          time.Now(),
		WindowMinute: 15,
		Notified:     []uint{},
		Skipped:      []map[string]interface{}{},
	}
	rules := crmSettingValue(auth.TenantID, "sla_rule", "tickets")
	notify := mapValue(rules["notify"])
	if value := int(numericValue(notify["before_minutes"])); value > 0 && value <= 120 {
		result.WindowMinute = value
	}
	windowStart := result.Now
	windowEnd := result.Now.Add(time.Duration(result.WindowMinute) * time.Minute)
	var tickets []models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").
		Where("object_type = ? AND status IN ?", "ticket", []string{"new", "assigned", "processing", "waiting_customer", "reopened"}).
		Order("updated_at asc").
		Limit(500).
		Find(&tickets).Error; err != nil {
		return result, err
	}
	result.Scanned = len(tickets)
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		for _, ticket := range tickets {
			payload := payloadMap(ticket.PayloadJSON)
			dueAt, err := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["sla_due_at"])))
			if err != nil || dueAt == nil {
				continue
			}
			if dueAt.Before(windowStart) || dueAt.After(windowEnd) {
				continue
			}
			lastAt, _ := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["sla_notice_15m_at"])))
			if lastAt != nil && result.Now.Sub(*lastAt) < 10*time.Minute {
				continue
			}
			payload["sla_notice_15m_at"] = result.Now.Format(time.RFC3339)
			payload["sla_notice_15m_owner_id"] = ticket.OwnerID
			payload["sla_notice_15m_due_at"] = dueAt.Format(time.RFC3339)
			ticket.PayloadJSON = toJSONString(payload)
			ticket.UpdatedBy = auth.UserID
			if err := tx.Save(&ticket).Error; err != nil {
				return err
			}
			taskName := fmt.Sprintf("SLA即将超时：%s", ticket.Name)
			task := cloneCRMObject(auth, "workspace", "mail_notification", taskName, ticket)
			task.Status = "open"
			task.OwnerID = ticket.OwnerID
			task.PayloadJSON = toJSONString(map[string]interface{}{
				"category":     "sla_warning",
				"source":       "ticket_sla_notice",
				"ticket_id":    ticket.ID,
				"content":      fmt.Sprintf("工单将在%d分钟内到达SLA截止，请优先处理。", result.WindowMinute),
				"sla_due_at":   dueAt.Format(time.RFC3339),
				"ticket_owner": ticket.OwnerID,
				"to_role":      "owner",
			})
			if err := tx.Create(&task).Error; err != nil {
				return err
			}
			publishSystemNotificationUnread(auth, task.OwnerID)
			if err := tx.Create(&models.CRMRelation{
				TenantModel:  models.TenantModel{TenantID: auth.TenantID},
				SourceType:   "tickets",
				SourceID:     ticket.ID,
				TargetType:   "workspace",
				TargetID:     task.ID,
				RelationType: "sla_notice_task",
			}).Error; err != nil {
				return err
			}
			if managerID := tenantAdminUserID(auth.TenantID); managerID > 0 && managerID != ticket.OwnerID {
				managerTask := cloneCRMObject(auth, "workspace", "mail_notification", "[主管提醒]"+taskName, ticket)
				managerTask.Status = "open"
				managerTask.OwnerID = managerID
				managerTask.PayloadJSON = toJSONString(map[string]interface{}{
					"category":       "sla_warning",
					"source":         "ticket_sla_notice",
					"ticket_id":      ticket.ID,
					"content":        "团队工单即将超时，请关注处理进度。",
					"sla_due_at":     dueAt.Format(time.RFC3339),
					"ticket_owner":   ticket.OwnerID,
					"manager_notice": true,
					"to_role":        "manager",
				})
				if err := tx.Create(&managerTask).Error; err != nil {
					return err
				}
				publishSystemNotificationUnread(auth, managerTask.OwnerID)
				if err := tx.Create(&models.CRMRelation{
					TenantModel:  models.TenantModel{TenantID: auth.TenantID},
					SourceType:   "tickets",
					SourceID:     ticket.ID,
					TargetType:   "workspace",
					TargetID:     managerTask.ID,
					RelationType: "sla_notice_task",
				}).Error; err != nil {
					return err
				}
			}
			if err := createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_sla_notice_15m", "SLA 15分钟预警通知", map[string]interface{}{"reason": "SLA即将超时通知负责人和主管", "sla_due_at": dueAt.Format(time.RFC3339), "owner_id": ticket.OwnerID}); err != nil {
				return err
			}
			result.Notified = append(result.Notified, ticket.ID)
		}
		return nil
	})
	return result, err
}

func tenantAdminUserID(tenantID uint) uint {
	if tenantID == 0 {
		return 0
	}
	var admin models.User
	if err := utils.DB.Where("tenant_id = ? AND account_type = ? AND status = ?", tenantID, "tenant_admin", "active").Order("id asc").First(&admin).Error; err == nil && admin.ID > 0 {
		return admin.ID
	}
	if err := utils.DB.Where("tenant_id = ? AND role = ? AND status = ?", tenantID, "admin", "active").Order("id asc").First(&admin).Error; err == nil && admin.ID > 0 {
		return admin.ID
	}
	return 0
}
