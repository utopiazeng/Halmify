package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

func (c *CRMController) CreateKnowledgeFromTicket() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	payload := payloadMap(ticket.PayloadJSON)
	articlePayload := map[string]interface{}{
		"source":    "ticket",
		"ticket_id": ticket.ID,
		"question":  payload["category"],
		"answer":    payload["public_reply"],
	}
	articleJSON, _ := json.Marshal(articlePayload)
	article := cloneCRMObject(auth, "insights", "faq_cluster", "知识库："+ticket.Name, ticket)
	article.Status = "open"
	article.PayloadJSON = string(articleJSON)
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&article).Error; err != nil {
			return err
		}
		relation := newCRMRelation(auth, "tickets", ticket.ID, "insights", article.ID, "generated_knowledge")
		return tx.Create(&relation).Error
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "工单生成知识失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_KNOWLEDGE_FROM_TICKET", fmt.Sprintf("工单[%d]", ticket.ID), "operation.create_knowledge_from_ticket", map[string]interface{}{"ticket_id": ticket.ID, "article_id": article.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ticket": ticket, "article": article})
}

func (c *CRMController) CreateBugTaskFromTicket() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var req struct {
		Name        string `json:"name"`
		Severity    string `json:"severity"`
		Module      string `json:"module"`
		Description string `json:"description"`
		Reason      string `json:"reason"`
		OwnerID     uint   `json:"owner_id"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	if req.OwnerID > 0 && !tenantUserExists(auth.TenantID, req.OwnerID) {
		utils.BadRequest(c.Ctx, "研发负责人不存在或不属于当前租户")
		return
	}
	ticketPayload := payloadMap(ticket.PayloadJSON)
	existingBugID := uintFromValue(ticketPayload["bug_task_id"])
	existingBugStatus := normalizeBugStatus(fmt.Sprint(ticketPayload["bug_status"]))
	if existingBugID > 0 && !bugStatusDone(existingBugStatus) {
		utils.BadRequest(c.Ctx, "该工单已关联进行中的研发Bug任务，请先更新原任务状态")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "研发Bug：" + ticket.Name
	}
	module := strings.TrimSpace(req.Module)
	if module == "" {
		module = strings.TrimSpace(fmt.Sprint(ticketPayload["category"]))
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "工单问题转研发缺陷处理"
	}
	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = strings.TrimSpace(fmt.Sprint(ticketPayload["description"]))
	}
	severity := normalizeBugSeverity(req.Severity, ticketPayload)
	ownerID := req.OwnerID
	if ownerID == 0 {
		ownerID = ticket.OwnerID
	}
	if ownerID == 0 {
		ownerID = tenantAdminUserID(auth.TenantID)
	}
	bug := cloneCRMObject(auth, "projects", "bug_task", name, ticket)
	bug.Status = "open"
	bug.OwnerID = ownerID
	bug.PayloadJSON = toJSONString(mergePayload(ticketPayload, map[string]interface{}{
		"source":           "ticket",
		"source_ticket_id": ticket.ID,
		"source_ticket":    ticket.Name,
		"module":           module,
		"severity":         severity,
		"description":      description,
		"reason":           reason,
		"status":           "open",
		"created_from":     "ticket_flow",
	}))
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&bug).Error; err != nil {
			return err
		}
		ticketPayload["bug_task_id"] = bug.ID
		ticketPayload["bug_status"] = "open"
		ticketPayload["bug_created_at"] = time.Now().Format(time.RFC3339)
		ticketPayload["bug_sync_at"] = time.Now().Format(time.RFC3339)
		ticketPayload["bug_owner_id"] = ownerID
		ticketPayload["bug_severity"] = severity
		ticketPayload["bug_module"] = module
		ticketPayload["bug_reason"] = reason
		ticket.PayloadJSON = toJSONString(ticketPayload)
		if err := tx.Save(&ticket).Error; err != nil {
			return err
		}
		if err := tx.Create(&[]models.CRMRelation{
			newCRMRelation(auth, "tickets", ticket.ID, "projects", bug.ID, "ticket_bug"),
		}).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_bug_created", "工单转研发Bug任务", map[string]interface{}{"reason": reason, "bug_task_id": bug.ID, "bug_name": bug.Name, "bug_status": bug.Status})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "工单转研发Bug失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_BUG_TASK_FROM_TICKET", fmt.Sprintf("工单[%d]", ticket.ID), "operation.create_bug_task_from_ticket", map[string]interface{}{"ticket_id": ticket.ID, "bug_task_id": bug.ID, "owner_id": ownerID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ticket": ticket, "bug_task": bug})
}

func (c *CRMController) SyncTicketBugStatus() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var req struct {
		Status  string `json:"status"`
		Reason  string `json:"reason"`
		SyncBug bool   `json:"sync_bug"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	status := normalizeBugStatus(req.Status)
	if status == "" {
		utils.BadRequest(c.Ctx, "Bug状态不能为空")
		return
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	ticketPayload := payloadMap(ticket.PayloadJSON)
	bugID := uintFromValue(ticketPayload["bug_task_id"])
	if bugID == 0 {
		utils.BadRequest(c.Ctx, "该工单尚未关联研发Bug任务")
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "研发状态同步"
	}
	beforeTicketStatus := ticket.Status
	nextTicketStatus := beforeTicketStatus
	if bugStatusDone(status) && beforeTicketStatus != "closed" {
		if beforeTicketStatus == "new" || beforeTicketStatus == "assigned" || beforeTicketStatus == "processing" || beforeTicketStatus == "reopened" {
			nextTicketStatus = "waiting_customer"
		}
	}
	if status == "reopened" && beforeTicketStatus != "closed" {
		nextTicketStatus = "processing"
	}
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		var bug models.CRMObject
		if err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "projects", "bug_task", bugID).First(&bug).Error; err == nil {
			bugPayload := payloadMap(bug.PayloadJSON)
			bugPayload["status"] = status
			bugPayload["status_reason"] = reason
			bugPayload["updated_from_ticket"] = true
			bugPayload["ticket_sync_at"] = time.Now().Format(time.RFC3339)
			bug.Status = status
			bug.PayloadJSON = toJSONString(bugPayload)
			if req.SyncBug {
				if err := tx.Save(&bug).Error; err != nil {
					return err
				}
			}
		}
		ticketPayload["bug_status"] = status
		ticketPayload["bug_status_reason"] = reason
		ticketPayload["bug_updated_at"] = time.Now().Format(time.RFC3339)
		ticketPayload["bug_sync_at"] = time.Now().Format(time.RFC3339)
		ticket.Status = nextTicketStatus
		ticket.PayloadJSON = toJSONString(ticketPayload)
		if err := tx.Save(&ticket).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_bug_status_sync", "研发Bug状态回写", map[string]interface{}{"reason": reason, "bug_task_id": bugID, "bug_status": status, "ticket_from": beforeTicketStatus, "ticket_to": nextTicketStatus})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "研发Bug状态回写失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SYNC_TICKET_BUG_STATUS", fmt.Sprintf("工单[%d]", ticket.ID), "operation.sync_ticket_bug_status", map[string]interface{}{"ticket_id": ticket.ID, "bug_task_id": bugID, "bug_status": status, "from": beforeTicketStatus, "to": nextTicketStatus}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ticket": ticket, "bug_task_id": bugID, "bug_status": status})
}

func (c *CRMController) AddTicketNote() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var req struct {
		Visibility string `json:"visibility"`
		Content    string `json:"content"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Visibility = strings.TrimSpace(req.Visibility)
	if req.Visibility == "" {
		req.Visibility = "internal"
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		utils.BadRequest(c.Ctx, "处理内容不能为空")
		return
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	note := cloneCRMObject(auth, "tickets", "ticket_note", "处理记录："+ticket.Name, ticket)
	note.Status = "active"
	note.PayloadJSON = toJSONString(map[string]interface{}{"visibility": req.Visibility, "content": req.Content, "agent_id": auth.UserID, "created_at": time.Now().Format(time.RFC3339)})
	payload := payloadMap(ticket.PayloadJSON)
	if req.Visibility == "public" {
		payload["public_reply"] = req.Content
	} else {
		payload["last_internal_note"] = req.Content
	}
	ticket.PayloadJSON = toJSONString(payload)
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&ticket).Error; err != nil {
			return err
		}
		if err := tx.Create(&note).Error; err != nil {
			return err
		}
		if err := tx.Create(&[]models.CRMRelation{
			newCRMRelation(auth, "tickets", ticket.ID, "tickets", note.ID, "has_note"),
		}).Error; err != nil {
			return err
		}
		if req.Visibility == "public" {
			if err := syncPortalTicketTx(tx, ticket, req.Content, ticket.Status); err != nil {
				return err
			}
		}
		return createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_note", "工单处理记录", map[string]interface{}{"reason": req.Content, "visibility": req.Visibility, "note_id": note.ID})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "工单处理记录保存失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "ADD_TICKET_NOTE", fmt.Sprintf("工单[%d]", ticket.ID), "operation.add_ticket_note", map[string]interface{}{"ticket_id": ticket.ID, "note_id": note.ID, "visibility": req.Visibility}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ticket": ticket, "note": note})
}

func (c *CRMController) ChangeTicketStatus() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var req struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	status := normalizeTicketStatus(req.Status)
	if status == "" {
		utils.BadRequest(c.Ctx, "工单状态无效")
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		utils.BadRequest(c.Ctx, "请填写处理内容")
		return
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	if ticket.Status == "closed" && status != "reopened" {
		utils.BadRequest(c.Ctx, "已关闭工单只能重新打开")
		return
	}
	before := ticket.Status
	payload := payloadMap(ticket.PayloadJSON)
	payload["status_reason"] = reason
	payload["status_changed_at"] = time.Now().Format(time.RFC3339)
	payload["status_changed_by"] = auth.UserID
	wasResolved := before != "resolved" && status == "resolved"
	if wasResolved {
		payload["resolved_at"] = time.Now().Format(time.RFC3339)
	}
	ticket.Status = status
	ticket.PayloadJSON = toJSONString(payload)
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&ticket).Error; err != nil {
			return err
		}
		if wasResolved {
			if err := createTicketSatisfactionNotification(tx, auth, &ticket); err != nil {
				return err
			}
		}
		if err := syncPortalTicketTx(tx, ticket, reason, status); err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_status", fmt.Sprintf("%s -> %s", before, status), map[string]interface{}{"reason": reason, "from": before, "to": status})
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "工单状态变更失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CHANGE_TICKET_STATUS", fmt.Sprintf("工单[%d]", ticket.ID), "operation.change_ticket_status", map[string]interface{}{"ticket_id": ticket.ID, "from": before, "to": status}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ticket": ticket})
}

func (c *CRMController) AssignTicket() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var req struct {
		OwnerID     uint   `json:"owner_id"`
		ServiceRole string `json:"service_role"`
		Reason      string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.OwnerID == 0 {
		utils.BadRequest(c.Ctx, "处理人不能为空")
		return
	}
	if !tenantUserExists(auth.TenantID, req.OwnerID) {
		utils.BadRequest(c.Ctx, "处理人不存在或不属于当前租户")
		return
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	payload := payloadMap(ticket.PayloadJSON)
	serviceRole := normalizeTicketServiceRole(req.ServiceRole)
	if serviceRole == "" {
		serviceRole = ticketServiceRole(payload)
	}
	payload["service_role"] = serviceRole
	payload["support_queue"] = serviceRole
	payload["assigned_at"] = time.Now().Format(time.RFC3339)
	payload["assignment_reason"] = strings.TrimSpace(req.Reason)
	ticket.OwnerID = req.OwnerID
	ticket.Status = "assigned"
	ticket.PayloadJSON = toJSONString(payload)
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&ticket).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_assign", "工单分派", map[string]interface{}{"reason": req.Reason, "owner_id": req.OwnerID, "service_role": serviceRole})
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "工单分派失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "ASSIGN_TICKET", fmt.Sprintf("工单[%d]", ticket.ID), "operation.assign_ticket", map[string]interface{}{"ticket_id": ticket.ID, "owner_id": req.OwnerID, "service_role": serviceRole}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ticket": ticket})
}

func (c *CRMController) TransferTicket() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var req struct {
		OwnerID     uint   `json:"owner_id"`
		ServiceRole string `json:"service_role"`
		Reason      string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.OwnerID == 0 {
		utils.BadRequest(c.Ctx, "转派目标不能为空")
		return
	}
	if !tenantUserExists(auth.TenantID, req.OwnerID) {
		utils.BadRequest(c.Ctx, "转派目标不存在或不属于当前租户")
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		utils.BadRequest(c.Ctx, "请填写转派原因")
		return
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	fromOwner := ticket.OwnerID
	if fromOwner == req.OwnerID {
		utils.BadRequest(c.Ctx, "当前负责人与转派目标一致")
		return
	}
	payload := payloadMap(ticket.PayloadJSON)
	serviceRole := normalizeTicketServiceRole(req.ServiceRole)
	if serviceRole == "" {
		serviceRole = ticketServiceRole(payload)
	}
	payload["transfer_at"] = time.Now().Format(time.RFC3339)
	payload["transfer_by"] = auth.UserID
	payload["transfer_reason"] = reason
	payload["from_owner_id"] = fromOwner
	payload["service_role"] = serviceRole
	payload["support_queue"] = serviceRole
	ticket.OwnerID = req.OwnerID
	if ticket.Status == "new" {
		ticket.Status = "assigned"
	}
	ticket.PayloadJSON = toJSONString(payload)
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&ticket).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_transfer", "工单内部转派", map[string]interface{}{"reason": reason, "from_owner_id": fromOwner, "to_owner_id": req.OwnerID, "service_role": serviceRole})
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "工单转派失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "TRANSFER_TICKET", fmt.Sprintf("工单[%d]", ticket.ID), "operation.transfer_ticket", map[string]interface{}{"ticket_id": ticket.ID, "from_owner_id": fromOwner, "to_owner_id": req.OwnerID, "service_role": serviceRole}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ticket": ticket, "from_owner_id": fromOwner, "to_owner_id": req.OwnerID, "service_role": serviceRole})
}

func (c *CRMController) AutoAssignTicket() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var req struct {
		ServiceRole string `json:"service_role"`
		Reason      string `json:"reason"`
	}
	if len(c.Ctx.Input.RequestBody) > 0 {
		_ = bindJSON(c.Ctx, &req)
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	payload := payloadMap(ticket.PayloadJSON)
	serviceRole := normalizeTicketServiceRole(req.ServiceRole)
	if serviceRole == "" {
		serviceRole = ticketServiceRole(payload)
	}
	ownerID, reason := pickTicketAssignee(auth, ticket, serviceRole)
	if ownerID == 0 {
		utils.BadRequest(c.Ctx, "当前没有可分派的客服账号")
		return
	}
	if strings.TrimSpace(req.Reason) != "" {
		reason = strings.TrimSpace(req.Reason)
	}
	payload["service_role"] = serviceRole
	payload["support_queue"] = serviceRole
	payload["assigned_at"] = time.Now().Format(time.RFC3339)
	payload["assignment_reason"] = reason
	payload["assignment_mode"] = "auto"
	ticket.OwnerID = ownerID
	ticket.Status = "assigned"
	ticket.PayloadJSON = toJSONString(payload)
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&ticket).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_auto_assign", "自动分派工单", map[string]interface{}{"reason": reason, "owner_id": ownerID, "service_role": serviceRole})
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "工单自动分派失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "AUTO_ASSIGN_TICKET", fmt.Sprintf("工单[%d]", ticket.ID), "operation.auto_assign_ticket", map[string]interface{}{"ticket_id": ticket.ID, "owner_id": ownerID, "service_role": serviceRole}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ticket": ticket, "owner_id": ownerID, "reason": reason, "service_role": serviceRole})
}

func (c *CRMController) EscalateTicketSLA() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var req struct {
		Reason string `json:"reason"`
		Force  bool   `json:"force"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	if ticket.Status == "closed" || ticket.Status == "resolved" {
		utils.BadRequest(c.Ctx, "已关闭或已解决工单不能升级")
		return
	}
	payload := payloadMap(ticket.PayloadJSON)
	dueAt, dueOK := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["sla_due_at"])))
	if dueOK != nil {
		utils.BadRequest(c.Ctx, "SLA截止时间格式无效")
		return
	}
	priority := strings.TrimSpace(fmt.Sprint(payload["priority"]))
	overdue := dueAt != nil && time.Now().After(*dueAt)
	if !req.Force && !overdue && priority != "urgent" && priority != "high" {
		utils.BadRequest(c.Ctx, "工单尚未触发SLA升级条件")
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		if overdue {
			reason = "SLA已超时"
		} else {
			reason = "高优先级工单升级"
		}
	}
	payload["sla_escalated"] = true
	payload["sla_escalated_at"] = time.Now().Format(time.RFC3339)
	payload["sla_escalated_by"] = auth.UserID
	payload["sla_escalation_reason"] = reason
	payload["sla_overdue"] = overdue
	ticket.Status = "processing"
	ticket.PayloadJSON = toJSONString(payload)
	task := cloneCRMObject(auth, "workspace", "task", "SLA升级处理："+ticket.Name, ticket)
	task.Status = "open"
	task.PayloadJSON = toJSONString(map[string]interface{}{
		"source":     "ticket_sla_escalation",
		"ticket_id":  ticket.ID,
		"priority":   "high",
		"reason":     reason,
		"sla_due_at": payload["sla_due_at"],
		"overdue":    overdue,
	})
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&ticket).Error; err != nil {
			return err
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		return tx.Create(&[]models.CRMRelation{
			newCRMRelation(auth, "tickets", ticket.ID, "workspace", task.ID, "sla_escalation_task"),
		}).Error
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "工单SLA升级失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "ESCALATE_TICKET_SLA", fmt.Sprintf("工单[%d]", ticket.ID), "operation.escalate_ticket_sla", map[string]interface{}{"ticket_id": ticket.ID, "task_id": task.ID, "reason": reason, "overdue": overdue}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ticket": ticket, "task": task, "overdue": overdue})
}

func (c *CRMController) TicketEscalationQueue() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	limit := queryInt(c.Ctx, "limit", 50)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var tickets []models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").
		Where("object_type = ? AND status NOT IN ?", "ticket", []string{"resolved", "closed"}).
		Order("updated_at desc").
		Limit(500).
		Find(&tickets).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取SLA升级队列失败")
		return
	}
	items := make([]map[string]interface{}, 0)
	now := time.Now()
	for _, ticket := range tickets {
		payload := payloadMap(ticket.PayloadJSON)
		dueAt, _ := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["sla_due_at"])))
		responseDueAt, _ := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["sla_response_due_at"])))
		priority := strings.TrimSpace(fmt.Sprint(payload["priority"]))
		overdue := dueAt != nil && now.After(*dueAt)
		responseOverdue := responseDueAt != nil && now.After(*responseDueAt) && strings.TrimSpace(fmt.Sprint(payload["first_response_at"])) == ""
		if !overdue && !responseOverdue && priority != "urgent" && priority != "high" {
			continue
		}
		items = append(items, map[string]interface{}{"ticket": ticket, "overdue": overdue, "response_overdue": responseOverdue, "priority": priority, "sla_due_at": payload["sla_due_at"], "reason": ticketEscalationReason(payload, overdue, responseOverdue)})
		if len(items) >= limit {
			break
		}
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": len(items)})
}

func (c *CRMController) TicketTransferQueue() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	limit := queryInt(c.Ctx, "limit", 50)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	groupBy := strings.ToLower(strings.TrimSpace(c.Ctx.Input.Query("group_by")))
	if groupBy == "" {
		groupBy = "queue"
	}
	includeClosed := strings.TrimSpace(c.Ctx.Input.Query("include_closed")) == "1"
	query := crmScopedObjectQuery(auth, "tickets").
		Where("object_type = ?", "ticket").
		Order("updated_at desc").
		Limit(500)
	if !includeClosed {
		query = query.Where("status NOT IN ?", []string{"resolved", "closed"})
	}
	var tickets []models.CRMObject
	if err := query.Find(&tickets).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取转派队列失败")
		return
	}
	now := time.Now()
	items := make([]map[string]interface{}, 0)
	queueStats := map[string]int{}
	for _, ticket := range tickets {
		payload := payloadMap(ticket.PayloadJSON)
		queue := ticketServiceRole(payload)
		priority := strings.TrimSpace(fmt.Sprint(payload["priority"]))
		dueAt, _ := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["sla_due_at"])))
		isOverdue := dueAt != nil && now.After(*dueAt) && ticket.Status != "closed" && ticket.Status != "resolved"
		assignedAt, _ := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["assigned_at"])))
		transferAt, _ := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["transfer_at"])))
		holdMinutes := 0
		if transferAt != nil {
			holdMinutes = int(now.Sub(*transferAt).Minutes())
		} else if assignedAt != nil {
			holdMinutes = int(now.Sub(*assignedAt).Minutes())
		} else {
			holdMinutes = int(now.Sub(ticket.UpdatedAt).Minutes())
		}
		queueStats[queue]++
		items = append(items, map[string]interface{}{
			"ticket":        ticket,
			"queue":         queue,
			"priority":      priority,
			"is_overdue":    isOverdue,
			"hold_minutes":  holdMinutes,
			"sla_due_at":    payload["sla_due_at"],
			"transfer_at":   payload["transfer_at"],
			"transfer_by":   payload["transfer_by"],
			"transfer_from": payload["from_owner_id"],
			"transfer_to":   ticket.OwnerID,
			"group_key":     queue,
		})
		if len(items) >= limit {
			break
		}
	}
	if groupBy == "owner" {
		for i := range items {
			t := mapValue(items[i]["ticket"])
			items[i]["group_key"] = fmt.Sprintf("owner:%d", uintFromValue(t["owner_id"]))
		}
	}
	stats := map[string]interface{}{
		"total":           len(items),
		"queue_counts":    queueStats,
		"l1_support":      queueStats["l1_support"],
		"l2_support":      queueStats["l2_support"],
		"product_manager": queueStats["product_manager"],
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "stats": stats})
}

func (c *CRMController) TicketTransferStats() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	windowDays := queryInt(c.Ctx, "days", 30)
	if windowDays <= 0 || windowDays > 180 {
		windowDays = 30
	}
	since := time.Now().Add(-time.Duration(windowDays) * 24 * time.Hour)
	var tickets []models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").
		Where("object_type = ? AND updated_at >= ?", "ticket", since).
		Order("updated_at desc").
		Limit(2000).
		Find(&tickets).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取转派统计失败")
		return
	}
	transferCount := 0
	reopenAfterTransfer := 0
	overdueAfterTransfer := 0
	byQueue := map[string]int{}
	totalHandleHours := 0.0
	handleCount := 0
	for _, ticket := range tickets {
		payload := payloadMap(ticket.PayloadJSON)
		transferAtText := strings.TrimSpace(fmt.Sprint(payload["transfer_at"]))
		if transferAtText == "" {
			continue
		}
		transferCount++
		queue := ticketServiceRole(payload)
		byQueue[queue]++
		if ticket.Status == "reopened" {
			reopenAfterTransfer++
		}
		dueAt, _ := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["sla_due_at"])))
		if dueAt != nil && (ticket.Status != "closed" && ticket.Status != "resolved") && time.Now().After(*dueAt) {
			overdueAfterTransfer++
		}
		transferAt, _ := parseOptionalTime(transferAtText)
		if transferAt == nil {
			continue
		}
		endAt := ticket.UpdatedAt
		if ticket.Status == "resolved" || ticket.Status == "closed" {
			if resolvedAt, _ := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["resolved_at"]))); resolvedAt != nil {
				endAt = *resolvedAt
			}
			if closedAt, _ := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["closed_at"]))); closedAt != nil {
				endAt = *closedAt
			}
		}
		if endAt.After(*transferAt) {
			totalHandleHours += endAt.Sub(*transferAt).Hours()
			handleCount++
		}
	}
	avgHandleHours := 0.0
	if handleCount > 0 {
		avgHandleHours = totalHandleHours / float64(handleCount)
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"window_days":               windowDays,
		"transfer_count":            transferCount,
		"reopen_after_transfer":     reopenAfterTransfer,
		"overdue_after_transfer":    overdueAfterTransfer,
		"queue_counts":              byQueue,
		"avg_handle_hours_transfer": avgHandleHours,
	})
}

func (c *CRMController) TicketKnowledgeSuggestions() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	ticketPayload := payloadMap(ticket.PayloadJSON)
	keywords := ticketKnowledgeKeywords(ticket, ticketPayload)
	query := crmScopedObjectQuery(auth, "insights").Where("object_type = ?", "faq_cluster")
	for _, keyword := range keywords {
		query = query.Where("name LIKE ? OR tags LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var articles []models.CRMObject
	if err := query.Order("updated_at desc").Limit(10).Find(&articles).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取知识推荐失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": articles, "keywords": keywords})
}

func (c *CRMController) CloseTicket() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的工单ID")
		return
	}
	var req struct {
		CSAT    int    `json:"csat"`
		NPS     int    `json:"nps"`
		Comment string `json:"comment"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var ticket models.CRMObject
	if err := crmScopedObjectQuery(auth, "tickets").Where("id = ?", id).First(&ticket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	payload := payloadMap(ticket.PayloadJSON)
	if strings.TrimSpace(req.Comment) == "" && strings.TrimSpace(fmt.Sprint(payload["public_reply"])) == "" && strings.TrimSpace(fmt.Sprint(payload["last_internal_note"])) == "" {
		utils.BadRequest(c.Ctx, "关闭工单前请先填写处理内容")
		return
	}
	payload["closed_at"] = time.Now().Format(time.RFC3339)
	payload["satisfaction"] = map[string]interface{}{"csat": req.CSAT, "nps": req.NPS, "comment": req.Comment}
	ticket.Status = "closed"
	ticket.PayloadJSON = toJSONString(payload)
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&ticket).Error; err != nil {
			return err
		}
		if req.CSAT > 0 && req.CSAT <= 2 {
			task := cloneCRMObject(auth, "workspace", "task", "低满意度回访："+ticket.Name, ticket)
			task.Status = "open"
			task.PayloadJSON = toJSONString(map[string]interface{}{"source": "ticket_satisfaction", "ticket_id": ticket.ID, "csat": req.CSAT})
			if err := tx.Create(&task).Error; err != nil {
				return err
			}
			if err := tx.Create(&[]models.CRMRelation{
				newCRMRelation(auth, "tickets", ticket.ID, "workspace", task.ID, "created_callback_task"),
			}).Error; err != nil {
				return err
			}
		}
		if err := syncPortalTicketTx(tx, ticket, req.Comment, "closed"); err != nil {
			return err
		}
		if err := backwriteTicketSatisfactionToContact(tx, auth, &ticket, req.CSAT, req.NPS, req.Comment); err != nil {
			return err
		}
		if err := createContinuousLowSatisfactionAlert(tx, auth, &ticket, req.CSAT, req.NPS); err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_close", "关闭工单", map[string]interface{}{"reason": req.Comment, "csat": req.CSAT, "nps": req.NPS})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "工单关闭失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CLOSE_TICKET", fmt.Sprintf("工单[%d]", ticket.ID), "operation.close_ticket", map[string]interface{}{"ticket_id": ticket.ID, "csat": req.CSAT, "nps": req.NPS}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ticket": ticket})
}

func createTicketSatisfactionNotification(tx *gorm.DB, auth authContext, ticket *models.CRMObject) error {
	if ticket == nil {
		return nil
	}
	payload := payloadMap(ticket.PayloadJSON)
	if boolValue(payload["satisfaction_notice_sent"], false) {
		return nil
	}
	ownerID := ticket.OwnerID
	if ownerID == 0 {
		ownerID = tenantAdminUserID(auth.TenantID)
	}
	notice := cloneCRMObject(auth, "workspace", "mail_notification", "满意度调查："+ticket.Name, *ticket)
	notice.Status = "open"
	notice.OwnerID = ownerID
	notice.PayloadJSON = toJSONString(map[string]interface{}{
		"source":      "ticket_satisfaction_survey",
		"category":    "satisfaction",
		"ticket_id":   ticket.ID,
		"content":     "工单已解决，请发送NPS/满意度调查并记录客户反馈。",
		"ticket_name": ticket.Name,
		"to_role":     "owner",
	})
	if err := tx.Create(&notice).Error; err != nil {
		return err
	}
	publishSystemNotificationUnread(auth, ownerID)
	if err := tx.Create(&models.CRMRelation{
		TenantModel:  models.TenantModel{TenantID: auth.TenantID},
		SourceType:   "tickets",
		SourceID:     ticket.ID,
		TargetType:   "workspace",
		TargetID:     notice.ID,
		RelationType: "satisfaction_notice",
	}).Error; err != nil {
		return err
	}
	payload["satisfaction_notice_sent"] = true
	payload["satisfaction_notice_sent_at"] = time.Now().Format(time.RFC3339)
	payload["satisfaction_notice_id"] = notice.ID
	ticket.PayloadJSON = toJSONString(payload)
	ticket.UpdatedBy = auth.UserID
	return tx.Save(ticket).Error
}

func backwriteTicketSatisfactionToContact(tx *gorm.DB, auth authContext, ticket *models.CRMObject, csat int, nps int, comment string) error {
	if ticket == nil {
		return nil
	}
	if csat <= 0 && nps <= 0 {
		return nil
	}
	targetID := ticket.ContactID
	if targetID == 0 {
		targetID = ticket.CompanyID
	}
	if targetID == 0 {
		return nil
	}
	var contact models.CRMObject
	if err := crmScopedObjectQuery(auth, "contacts").Where("id = ?", targetID).First(&contact).Error; err != nil {
		return nil
	}
	payload := payloadMap(contact.PayloadJSON)
	history := make([]interface{}, 0)
	if raw, ok := payload["satisfaction_history"].([]interface{}); ok {
		history = append(history, raw...)
	}
	record := map[string]interface{}{
		"ticket_id":   ticket.ID,
		"ticket_name": ticket.Name,
		"csat":        csat,
		"nps":         nps,
		"comment":     strings.TrimSpace(comment),
		"at":          time.Now().Format(time.RFC3339),
	}
	history = append(history, record)
	if len(history) > 20 {
		history = history[len(history)-20:]
	}
	payload["satisfaction_history"] = history
	payload["last_satisfaction"] = record
	var sumCSAT float64
	var countCSAT float64
	for _, row := range history {
		item := mapValue(row)
		value := numericValue(item["csat"])
		if value > 0 {
			sumCSAT += value
			countCSAT++
		}
	}
	if countCSAT > 0 {
		payload["avg_csat"] = sumCSAT / countCSAT
	}
	contact.PayloadJSON = toJSONString(payload)
	contact.UpdatedBy = auth.UserID
	return tx.Save(&contact).Error
}

func createContinuousLowSatisfactionAlert(tx *gorm.DB, auth authContext, ticket *models.CRMObject, csat int, nps int) error {
	if ticket == nil {
		return nil
	}
	low := (csat > 0 && csat <= 2) || (nps >= 0 && nps <= 6)
	if !low {
		return nil
	}
	var recent []models.CRMObject
	query := crmScopedObjectQuery(auth, "tickets").Where("object_type = ? AND status = ? AND (company_id = ? OR contact_id = ?)", "ticket", "closed", ticket.CompanyID, ticket.ContactID).Order("updated_at desc").Limit(10)
	if err := query.Find(&recent).Error; err != nil {
		return nil
	}
	lowCount := 0
	for _, row := range recent {
		p := payloadMap(row.PayloadJSON)
		sat := mapValue(p["satisfaction"])
		rcsat := int(numericValue(sat["csat"]))
		rnps := int(numericValue(sat["nps"]))
		if (rcsat > 0 && rcsat <= 2) || (rnps >= 0 && rnps <= 6) {
			lowCount++
		}
		if lowCount >= 2 {
			break
		}
	}
	if lowCount < 2 {
		return nil
	}
	managerID := tenantAdminUserID(auth.TenantID)
	if managerID == 0 {
		return nil
	}
	task := cloneCRMObject(auth, "workspace", "task", "客户成功介入："+ticket.Name, *ticket)
	task.Status = "open"
	task.OwnerID = managerID
	task.PayloadJSON = toJSONString(map[string]interface{}{
		"source":        "ticket_satisfaction_alert",
		"ticket_id":     ticket.ID,
		"company_id":    ticket.CompanyID,
		"contact_id":    ticket.ContactID,
		"low_count":     lowCount,
		"priority":      "high",
		"alert_message": "连续低满意度，请客户成功经理介入回访。",
	})
	if err := tx.Create(&task).Error; err != nil {
		return err
	}
	if err := tx.Create(&models.CRMRelation{
		TenantModel:  models.TenantModel{TenantID: auth.TenantID},
		SourceType:   "tickets",
		SourceID:     ticket.ID,
		TargetType:   "workspace",
		TargetID:     task.ID,
		RelationType: "satisfaction_alert_task",
	}).Error; err != nil {
		return err
	}
	return createBusinessTimeline(tx, auth, "tickets", ticket.ID, "ticket_satisfaction_alert", "连续低满意度预警", map[string]interface{}{"low_count": lowCount, "task_id": task.ID})
}

func normalizeBugSeverity(raw string, ticketPayload map[string]interface{}) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "critical", "high", "medium", "low":
		return value
	}
	priority := strings.ToLower(strings.TrimSpace(fmt.Sprint(ticketPayload["priority"])))
	switch priority {
	case "urgent":
		return "critical"
	case "high":
		return "high"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func normalizeBugStatus(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	aliases := map[string]string{
		"todo":         "open",
		"new":          "open",
		"opened":       "open",
		"inprogress":   "in_progress",
		"in-progress":  "in_progress",
		"processing":   "in_progress",
		"solved":       "fixed",
		"resolved":     "fixed",
		"done":         "fixed",
		"verified":     "closed",
		"closed":       "closed",
		"reopen":       "reopened",
		"regression":   "reopened",
		"cannot_repro": "wontfix",
	}
	if mapped, ok := aliases[value]; ok {
		value = mapped
	}
	for _, item := range []string{"open", "in_progress", "fixed", "closed", "reopened", "wontfix"} {
		if value == item {
			return value
		}
	}
	return ""
}

func bugStatusDone(status string) bool {
	return status == "fixed" || status == "closed" || status == "wontfix"
}
