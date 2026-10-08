package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"gorm.io/gorm"
	"net/http"
)

func (c *CRMController) CompleteProject() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的项目ID")
		return
	}
	var project models.CRMObject
	if err := crmScopedObjectQuery(auth, "projects").Where("id = ?", id).First(&project).Error; err != nil {
		utils.NotFound(c.Ctx, "项目不存在")
		return
	}
	project.Status = "completed"
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&project).Error; err != nil {
			return err
		}
		if project.DealID > 0 {
			return tx.Model(&models.CRMObject{}).Where("id = ? AND tenant_id = ? AND domain = ?", project.DealID, auth.TenantID, "opportunities").Updates(map[string]interface{}{"status": "negotiation"}).Error
		}
		return nil
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "项目完成失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "COMPLETE_PROJECT", fmt.Sprintf("项目[%d]", project.ID), "operation.complete_project", map[string]interface{}{"project_id": project.ID, "deal_id": project.DealID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"project": project})
}
