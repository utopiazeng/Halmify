package controllers

import (
	"ai-crm/models"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"time"
)

func normalizeApprovalNode(index int, node map[string]interface{}) map[string]interface{} {
	if node == nil {
		node = map[string]interface{}{}
	}
	if _, ok := node["index"]; !ok {
		node["index"] = index
	}
	if _, ok := node["status"]; !ok {
		node["status"] = "pending"
	}
	if _, ok := node["approved_at"]; !ok {
		node["approved_at"] = ""
	}
	return node
}

func normalizeApprovalAction(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	aliases := map[string]string{
		"pass":        "approve",
		"approved":    "approve",
		"agree":       "approve",
		"reject":      "reject",
		"rejected":    "reject",
		"append":      "append",
		"add_sign":    "append",
		"addsign":     "append",
		"countersign": "append",
		"transfer":    "transfer",
		"delegate":    "transfer",
		"withdraw":    "withdraw",
		"recall":      "withdraw",
	}
	return aliases[action]
}

func appendApprovalNodeAfter(nodes []map[string]interface{}, index int, newNode map[string]interface{}) []map[string]interface{} {
	next := make([]map[string]interface{}, 0, len(nodes)+1)
	next = append(next, nodes[:index+1]...)
	next = append(next, newNode)
	next = append(next, nodes[index+1:]...)
	return reindexApprovalNodes(next)
}

func reindexApprovalNodes(nodes []map[string]interface{}) []map[string]interface{} {
	for index := range nodes {
		nodes[index]["index"] = index + 1
	}
	return nodes
}

func approvalObjectStatus(instance models.ApprovalInstance, approvalStatus string) string {
	switch approvalStatus {
	case "approved":
		if instance.BusinessType == "orders" {
			return "confirmed"
		}
		if instance.BusinessType == "projects" {
			return "active"
		}
		if instance.BusinessType == "products" {
			return "on_sale"
		}
		return "active"
	case "rejected":
		return "rejected"
	case "withdrawn":
		return "draft"
	default:
		if instance.BusinessType == "orders" {
			return "pending_review"
		}
		return "approving"
	}
}

func prepareApprovalNodes(auth authContext, object models.CRMObject, nodes []map[string]interface{}) []map[string]interface{} {
	for index := range nodes {
		if !approvalNodeHasApprover(nodes[index]) && object.OwnerID > 0 {
			nodes[index]["approver_id"] = object.OwnerID
		}
		if !approvalNodeHasApprover(nodes[index]) && object.DepartmentID > 0 {
			nodes[index]["approver_role_id"] = object.DepartmentID
		}
		if !approvalNodeHasApprover(nodes[index]) {
			nodes[index]["approver_id"] = auth.UserID
		}
	}
	return nodes
}

func validateObjectApprovalRules(auth authContext, object models.CRMObject) error {
	payload := payloadMap(object.PayloadJSON)
	switch {
	case object.Domain == "orders" && (object.ObjectType == "order" || object.ObjectType == "order_approval"):
		if len(interfaceList(payload["line_items"])) == 0 {
			return fmt.Errorf("订单审核必须包含至少一条订单明细")
		}
		if object.Amount <= 0 {
			return fmt.Errorf("订单审核金额必须大于 0")
		}
		if strings.TrimSpace(fmt.Sprint(payload["payment_status"])) == "paid" {
			return fmt.Errorf("已支付订单不能重新发起订单审核")
		}
	case object.Domain == "products" && object.ObjectType == "pricing_rule":
		discount := numericValue(payload["discount_rate"])
		if discount == 0 {
			discount = numericValue(payload["discount"])
		}
		minMargin := numericValue(payload["min_margin"])
		if discount <= 0 && object.Amount <= 0 {
			return fmt.Errorf("折扣审批必须填写折扣率或折扣金额")
		}
		if discount >= 1 {
			return fmt.Errorf("折扣率必须小于 1")
		}
		if minMargin < 0 {
			return fmt.Errorf("折扣审批的最低毛利率不能为负数")
		}
	case object.Domain == "projects" && object.ObjectType == "deliverable":
		if strings.TrimSpace(fmt.Sprint(payload["project_id"])) == "" && object.ProjectID == 0 {
			return fmt.Errorf("交付物审批必须关联项目")
		}
		if strings.TrimSpace(fmt.Sprint(payload["acceptance_criteria"])) == "" {
			return fmt.Errorf("交付物审批必须填写验收标准")
		}
		if strings.TrimSpace(fmt.Sprint(payload["owner"])) == "" && object.OwnerID == 0 {
			return fmt.Errorf("交付物审批必须指定负责人")
		}
	}
	return nil
}

func applyApprovalResultToBusinessObject(tx *gorm.DB, auth authContext, instance models.ApprovalInstance, nextStatus string) error {
	var object models.CRMObject
	if err := tx.Where("tenant_id = ? AND domain = ? AND id = ?", auth.TenantID, instance.BusinessType, instance.BusinessID).First(&object).Error; err != nil {
		return err
	}
	status := "approving"
	if nextStatus == "approved" || nextStatus == "rejected" || nextStatus == "withdrawn" {
		status = approvalObjectStatus(instance, nextStatus)
	}
	payload := payloadMap(object.PayloadJSON)
	payload["approval_status"] = nextStatus
	payload["approval_instance_id"] = instance.ID
	payload["approval_updated_at"] = time.Now().Format(time.RFC3339)
	payload["approval_updated_by"] = auth.UserID
	if nextStatus == "approved" {
		payload["approved_at"] = time.Now().Format(time.RFC3339)
		payload["approved_by"] = auth.UserID
	}
	if nextStatus == "rejected" {
		payload["rejected_at"] = time.Now().Format(time.RFC3339)
		payload["rejected_by"] = auth.UserID
	}
	if nextStatus == "withdrawn" {
		payload["withdrawn_at"] = time.Now().Format(time.RFC3339)
		payload["withdrawn_by"] = auth.UserID
	}
	object.Status = status
	object.PayloadJSON = toJSONString(payload)
	object.UpdatedBy = auth.UserID
	return tx.Save(&object).Error
}
