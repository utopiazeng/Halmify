package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (c *CRMController) ListContactViews() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	objectType := strings.TrimSpace(c.Ctx.Input.Query("object_type"))
	if objectType != "contact" && objectType != "company" {
		objectType = "contact"
	}
	items := []map[string]interface{}{
		{
			"key":         "default_all",
			"name":        "全部记录",
			"object_type": objectType,
			"shared":      true,
			"preset":      true,
			"filters": map[string]interface{}{
				"status": "",
			},
		},
		{
			"key":         "default_active",
			"name":        "仅启用",
			"object_type": objectType,
			"shared":      true,
			"preset":      true,
			"filters": map[string]interface{}{
				"status": "active",
			},
		},
	}
	var setting models.CRMSetting
	custom := []map[string]interface{}{}
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "contact_view", "custom").First(&setting).Error; err == nil {
		_ = json.Unmarshal([]byte(setting.ValueJSON), &custom)
	}
	for _, item := range custom {
		rowType := strings.TrimSpace(fmt.Sprint(item["object_type"]))
		if rowType == "" || rowType == objectType {
			shared := boolValue(item["shared"], false)
			createdBy := uintFromValue(item["created_by"])
			if shared || createdBy == auth.UserID || auth.IsTenantAdmin() || auth.IsSuperAdmin() {
				items = append(items, item)
			}
		}
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "object_type": objectType})
}

func (c *CRMController) SaveContactView() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		Name       string                 `json:"name"`
		ObjectType string                 `json:"object_type"`
		Filters    map[string]interface{} `json:"filters"`
		Shared     bool                   `json:"shared"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.ObjectType = strings.TrimSpace(req.ObjectType)
	if req.Name == "" {
		utils.BadRequest(c.Ctx, "视图名称不能为空")
		return
	}
	if req.ObjectType != "contact" && req.ObjectType != "company" {
		req.ObjectType = "contact"
	}
	if req.Filters == nil {
		req.Filters = map[string]interface{}{}
	}
	var setting models.CRMSetting
	custom := []map[string]interface{}{}
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "contact_view", "custom").First(&setting).Error; err == nil {
		_ = json.Unmarshal([]byte(setting.ValueJSON), &custom)
	}
	item := map[string]interface{}{
		"key":         fmt.Sprintf("custom_%d", time.Now().UnixNano()),
		"name":        req.Name,
		"object_type": req.ObjectType,
		"shared":      req.Shared,
		"filters":     req.Filters,
		"preset":      false,
		"created_by":  auth.UserID,
		"created_at":  time.Now().Format(time.RFC3339),
	}
	custom = append(custom, item)
	next := models.CRMSetting{
		TenantModel: models.TenantModel{TenantID: auth.TenantID},
		Scope:       "contact_view",
		Key:         "custom",
		ValueJSON:   toJSONString(custom),
		UpdatedBy:   auth.UserID,
	}
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", auth.TenantID, "contact_view", "custom").Assign(next).FirstOrCreate(&next).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存视图失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SAVE_CONTACT_VIEW", "客户联系人视图", "operation.save_contact_view", map[string]interface{}{"name": req.Name, "object_type": req.ObjectType, "shared": req.Shared}, "success", "")
	utils.Success(c.Ctx, item)
}
