package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"net/http"
	"strings"
)

func (c *CRMController) ListRelations() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	objectType := strings.TrimSpace(c.Ctx.Input.Query("object_type"))
	objectID := queryUint(c.Ctx, "object_id")
	relationType := strings.TrimSpace(c.Ctx.Input.Query("relation_type"))
	query := utils.DB.Where("tenant_id = ?", auth.TenantID)
	if objectType != "" && objectID > 0 {
		if err := ensureCRMObjectVisible(auth, objectType, objectID); err != nil {
			utils.Success(c.Ctx, map[string]interface{}{"items": []models.CRMRelation{}})
			return
		}
		query = query.Where("(source_type = ? AND source_id = ?) OR (target_type = ? AND target_id = ?)", objectType, objectID, objectType, objectID)
	}
	if relationType != "" {
		query = query.Where("relation_type = ?", relationType)
	}
	var items []models.CRMRelation
	if err := query.Order("created_at desc").Limit(100).Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取关联关系失败")
		return
	}
	items = filterVisibleRelations(auth, items)
	utils.Success(c.Ctx, map[string]interface{}{"items": items})
}

func (c *CRMController) CreateRelation() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req crmRelationRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	relation, err := crmRelationFromRequest(auth, req)
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	if err := ensureCRMObjectVisible(auth, req.SourceType, req.SourceID); err != nil {
		utils.BadRequest(c.Ctx, "来源对象不存在")
		return
	}
	if err := ensureCRMObjectVisible(auth, req.TargetType, req.TargetID); err != nil {
		utils.BadRequest(c.Ctx, "目标对象不存在")
		return
	}
	if err := utils.DB.Create(&relation).Error; err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_CRM_RELATION", fmt.Sprintf("%s[%d]->%s[%d]", relation.SourceType, relation.SourceID, relation.TargetType, relation.TargetID), "operation.create_crm_relation", map[string]interface{}{"relation_type": relation.RelationType}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建关联关系失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_CRM_RELATION", fmt.Sprintf("%s[%d]->%s[%d]", relation.SourceType, relation.SourceID, relation.TargetType, relation.TargetID), "operation.create_crm_relation", map[string]interface{}{"relation_id": relation.ID, "relation_type": relation.RelationType}, "success", "")
	utils.Success(c.Ctx, relation)
}

func (c *CRMController) DeleteRelation() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的关联关系ID")
		return
	}
	var relation models.CRMRelation
	if err := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).First(&relation).Error; err != nil {
		utils.NotFound(c.Ctx, "关联关系不存在")
		return
	}
	if err := ensureCRMObjectVisible(auth, relation.SourceType, relation.SourceID); err != nil {
		utils.NotFound(c.Ctx, "关联关系不存在")
		return
	}
	if err := ensureCRMObjectVisible(auth, relation.TargetType, relation.TargetID); err != nil {
		utils.NotFound(c.Ctx, "关联关系不存在")
		return
	}
	result := utils.DB.Delete(&relation)
	if result.Error != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "删除关联关系失败")
		return
	}
	if result.RowsAffected == 0 {
		utils.NotFound(c.Ctx, "关联关系不存在")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "DELETE_CRM_RELATION", fmt.Sprintf("关联[%d]", id), "operation.delete_crm_relation", map[string]interface{}{"relation_id": id}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"deleted_id": id})
}
