package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

func (c *CRMController) ListApprovalTemplates() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	templateType := strings.TrimSpace(c.Ctx.Input.Query("type"))
	query := utils.DB.Where("tenant_id = ?", auth.TenantID)
	if templateType != "" {
		query = query.Where("type = ?", templateType)
	}
	var items []models.ApprovalTemplate
	if err := query.Order("updated_at desc").Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取审批模板失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items})
}

func (c *CRMController) CreateApprovalTemplate() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		Name       string                 `json:"name"`
		Type       string                 `json:"type"`
		Nodes      interface{}            `json:"nodes"`
		Conditions map[string]interface{} `json:"conditions"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.TrimSpace(req.Type)
	if req.Name == "" || req.Type == "" {
		utils.BadRequest(c.Ctx, "审批模板名称和类型不能为空")
		return
	}
	nodesJSON := toJSONString(req.Nodes)
	if _, err := approvalNodesFromJSON(nodesJSON); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	conditionsJSON := toJSONString(req.Conditions)
	template := models.ApprovalTemplate{
		TenantModel: models.TenantModel{TenantID: auth.TenantID},
		Name:        req.Name,
		Type:        req.Type,
		Nodes:       nodesJSON,
		Conditions:  conditionsJSON,
	}
	err := utils.DB.Where("tenant_id = ? AND type = ? AND name = ?", auth.TenantID, req.Type, req.Name).Assign(template).FirstOrCreate(&template).Error
	if err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "UPSERT_APPROVAL_TEMPLATE", req.Name, "operation.upsert_approval_template", map[string]interface{}{"type": req.Type, "name": req.Name}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存审批模板失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "UPSERT_APPROVAL_TEMPLATE", req.Name, "operation.upsert_approval_template", map[string]interface{}{"template_id": template.ID, "type": req.Type, "name": req.Name}, "success", "")
	utils.Success(c.Ctx, template)
}

func (c *CRMController) ListApprovalInstances() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	status := strings.TrimSpace(c.Ctx.Input.Query("status"))
	businessType := strings.TrimSpace(c.Ctx.Input.Query("business_type"))
	query := utils.DB.Where("tenant_id = ?", auth.TenantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if businessType != "" {
		query = query.Where("business_type = ?", businessType)
	}
	var items []models.ApprovalInstance
	if err := query.Order("updated_at desc").Limit(100).Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取审批实例失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items})
}

func (c *CRMController) ActApprovalInstance() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的审批ID")
		return
	}
	var req struct {
		Action       string                 `json:"action"`
		Comment      string                 `json:"comment"`
		Name         string                 `json:"name"`
		TargetUserID uint                   `json:"target_user_id"`
		TargetRoleID uint                   `json:"target_role_id"`
		TargetRole   string                 `json:"target_role"`
		Node         map[string]interface{} `json:"node"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Action = normalizeApprovalAction(req.Action)
	if req.Action == "" {
		utils.BadRequest(c.Ctx, "审批动作必须为 approve、reject、append、transfer 或 withdraw")
		return
	}
	var instance models.ApprovalInstance
	if err := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).First(&instance).Error; err != nil {
		utils.NotFound(c.Ctx, "审批实例不存在")
		return
	}
	if instance.Status != "pending" {
		utils.BadRequest(c.Ctx, "审批实例已处理")
		return
	}
	nodes, err := approvalNodesFromJSON(instance.NodesStatus)
	if err != nil {
		utils.BadRequest(c.Ctx, "审批节点状态无效")
		return
	}
	nodeIndex := instance.CurrentNode - 1
	if nodeIndex < 0 {
		nodeIndex = 0
	}
	if nodeIndex >= len(nodes) {
		utils.BadRequest(c.Ctx, "当前审批节点不存在")
		return
	}
	if !approvalActorMatches(auth, nodes[nodeIndex]) {
		utils.Error(c.Ctx, http.StatusForbidden, "当前用户不是该节点审批人")
		return
	}
	now := time.Now().Format(time.RFC3339)
	nextStatus := instance.Status
	switch req.Action {
	case "approve":
		nodes[nodeIndex]["status"] = "approved"
		nodes[nodeIndex]["approved_at"] = now
		nodes[nodeIndex]["operator_id"] = auth.UserID
		nodes[nodeIndex]["operator"] = auth.Username
		nodes[nodeIndex]["comment"] = req.Comment
		if nodeIndex+1 < len(nodes) {
			instance.CurrentNode = nodeIndex + 2
			nodes[nodeIndex+1]["status"] = "pending"
			nextStatus = "pending"
		} else {
			nextStatus = "approved"
		}
	case "reject":
		nodes[nodeIndex]["status"] = "rejected"
		nodes[nodeIndex]["rejected_at"] = now
		nodes[nodeIndex]["operator_id"] = auth.UserID
		nodes[nodeIndex]["operator"] = auth.Username
		nodes[nodeIndex]["comment"] = req.Comment
		nextStatus = "rejected"
	case "append":
		newNode := normalizeApprovalNode(nodeIndex+2, req.Node)
		if text := strings.TrimSpace(req.Name); text != "" {
			newNode["name"] = text
		}
		if req.TargetUserID > 0 {
			newNode["approver_id"] = req.TargetUserID
		}
		if req.TargetRoleID > 0 {
			newNode["approver_role_id"] = req.TargetRoleID
		}
		if text := strings.TrimSpace(req.TargetRole); text != "" {
			newNode["approver_role"] = text
		}
		if !approvalNodeHasApprover(newNode) {
			utils.BadRequest(c.Ctx, "加签节点必须指定审批人或审批角色")
			return
		}
		newNode["status"] = "pending"
		newNode["appended_by"] = auth.UserID
		newNode["appended_at"] = now
		nodes = appendApprovalNodeAfter(nodes, nodeIndex, newNode)
		nextStatus = "pending"
	case "transfer":
		if req.TargetUserID == 0 && req.TargetRoleID == 0 && strings.TrimSpace(req.TargetRole) == "" && !approvalNodeHasApprover(req.Node) {
			utils.BadRequest(c.Ctx, "转交必须指定目标审批人或目标角色")
			return
		}
		nodes[nodeIndex]["transferred_from_user_id"] = auth.UserID
		nodes[nodeIndex]["transferred_from"] = auth.Username
		nodes[nodeIndex]["transferred_at"] = now
		if req.TargetUserID > 0 {
			nodes[nodeIndex]["approver_id"] = req.TargetUserID
		}
		if req.TargetRoleID > 0 {
			nodes[nodeIndex]["approver_role_id"] = req.TargetRoleID
		}
		if text := strings.TrimSpace(req.TargetRole); text != "" {
			nodes[nodeIndex]["approver_role"] = text
		}
		for key, value := range req.Node {
			if strings.HasPrefix(key, "approver") || key == "role" || key == "role_id" || key == "user_id" || key == "user_ids" {
				nodes[nodeIndex][key] = value
			}
		}
		nodes[nodeIndex]["status"] = "pending"
		nextStatus = "pending"
	case "withdraw":
		nodes[nodeIndex]["status"] = "withdrawn"
		nodes[nodeIndex]["withdrawn_at"] = now
		nodes[nodeIndex]["operator_id"] = auth.UserID
		nodes[nodeIndex]["operator"] = auth.Username
		nodes[nodeIndex]["comment"] = req.Comment
		nextStatus = "withdrawn"
	}
	instance.Status = nextStatus
	instance.NodesStatus = toJSONString(reindexApprovalNodes(nodes))
	actionLog := models.ApprovalActionLog{
		TenantModel: models.TenantModel{TenantID: auth.TenantID},
		InstanceID:  instance.ID,
		Action:      req.Action,
		Comment:     req.Comment,
		Operator:    auth.Username,
	}
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&instance).Error; err != nil {
			return err
		}
		if err := tx.Create(&actionLog).Error; err != nil {
			return err
		}
		if instance.BusinessType != "" && instance.BusinessID > 0 {
			return applyApprovalResultToBusinessObject(tx, auth, instance, nextStatus)
		}
		return nil
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "处理审批失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "ACT_APPROVAL", fmt.Sprintf("审批[%d]", instance.ID), "operation.act_approval", map[string]interface{}{"action": req.Action, "status": nextStatus}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"approval": instance, "action_log": actionLog})
}

func (c *CRMController) StartObjectApproval() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domain := strings.TrimSpace(c.Ctx.Input.Query(":domain"))
	id, valid := pathUint(c.Ctx, ":id")
	if !valid || !validCRMDomain(domain) {
		utils.BadRequest(c.Ctx, "无效的业务对象")
		return
	}
	var object models.CRMObject
	if err := crmScopedObjectQuery(auth, domain).Where("id = ?", id).First(&object).Error; err != nil {
		utils.NotFound(c.Ctx, "业务对象不存在")
		return
	}
	var req struct {
		TemplateID uint   `json:"template_id"`
		Title      string `json:"title"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.TemplateID == 0 {
		utils.BadRequest(c.Ctx, "审批模板不能为空")
		return
	}
	var template models.ApprovalTemplate
	if err := utils.DB.Where("id = ? AND tenant_id = ?", req.TemplateID, auth.TenantID).First(&template).Error; err != nil {
		utils.NotFound(c.Ctx, "审批模板不存在")
		return
	}
	nodes, err := approvalNodesFromJSON(template.Nodes)
	if err != nil {
		utils.BadRequest(c.Ctx, "审批模板节点无效")
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = object.Name + " 审批"
	}
	if err := validateObjectApprovalRules(auth, object); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	nodes = prepareApprovalNodes(auth, object, nodes)
	instance := models.ApprovalInstance{
		TenantModel:  models.TenantModel{TenantID: auth.TenantID},
		TemplateID:   req.TemplateID,
		Title:        title,
		BusinessType: domain,
		BusinessID:   object.ID,
		Status:       "pending",
		CurrentNode:  1,
		NodesStatus:  toJSONString(nodes),
	}
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&instance).Error; err != nil {
			return err
		}
		object.Status = "approving"
		if domain == "orders" {
			object.Status = "pending_review"
		}
		if err := tx.Save(&object).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "发起业务审批失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "START_OBJECT_APPROVAL", fmt.Sprintf("%s[%d]", domain, object.ID), "operation.start_object_approval", map[string]interface{}{"domain": domain, "object_id": object.ID, "instance_id": instance.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"object": object, "approval": instance})
}
