package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"net/http"
	"time"
)

func (c *CRMController) CompleteWorkspaceTask() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的待办ID")
		return
	}
	var task models.CRMObject
	if err := crmScopedObjectQuery(auth, "workspace").Where("id = ?", id).First(&task).Error; err != nil {
		utils.NotFound(c.Ctx, "待办不存在")
		return
	}
	payload := payloadMap(task.PayloadJSON)
	payload["completed_at"] = time.Now().Format(time.RFC3339)
	payload["completed_by"] = auth.UserID
	task.Status = "done"
	task.PayloadJSON = toJSONString(payload)
	if err := utils.DB.Save(&task).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "待办完成失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "COMPLETE_WORKSPACE_TASK", fmt.Sprintf("待办[%d]", task.ID), "operation.complete_workspace_task", map[string]interface{}{"task_id": task.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"task": task})
}

func (c *CRMController) MarkWorkspaceNotificationRead() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 && !auth.IsPlatformAdmin() {
		utils.Error(c.Ctx, http.StatusForbidden, "缺少租户上下文")
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的通知ID")
		return
	}
	var notification models.CRMObject
	query := crmScopedObjectQuery(auth, "workspace")
	if auth.TenantID == 0 && auth.IsPlatformAdmin() {
		query = personalSystemNotificationBaseQuery(auth)
	}
	if err := query.Where("id = ?", id).First(&notification).Error; err != nil {
		utils.NotFound(c.Ctx, "通知不存在")
		return
	}
	if notification.ObjectType != "mail_notification" && notification.ObjectType != "customer_activity" {
		utils.BadRequest(c.Ctx, "该记录不是可标记已读的通知")
		return
	}
	if !auth.IsSuperAdmin() && !auth.IsTenantAdmin() && notification.OwnerID != auth.UserID {
		utils.Error(c.Ctx, http.StatusForbidden, "无权处理该通知")
		return
	}
	payload := payloadMap(notification.PayloadJSON)
	payload["read_at"] = time.Now().Format(time.RFC3339)
	payload["read_by"] = auth.UserID
	notification.Status = "read"
	notification.PayloadJSON = toJSONString(payload)
	if err := utils.DB.Save(&notification).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "通知标记已读失败")
		return
	}
	publishSystemNotificationUnread(auth, notification.OwnerID)
	logControllerOperation(c.Ctx, auth, auth.TenantID, "READ_WORKSPACE_NOTIFICATION", fmt.Sprintf("通知[%d]", notification.ID), "operation.read_workspace_notification", map[string]interface{}{"notification_id": notification.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"notification": notification})
}
