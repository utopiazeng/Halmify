package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web/context"
)

type crmFileLinkRequest struct {
	FileID        string `json:"file_id"`
	FileName      string `json:"file_name"`
	FileSize      int64  `json:"file_size"`
	FileType      string `json:"file_type"`
	StorageNodeID string `json:"storage_node_id"`
	DownloadURL   string `json:"download_url"`
	Category      string `json:"category"`
	Version       string `json:"version"`
	Visibility    string `json:"visibility"`
	Remark        string `json:"remark"`
}

func (c *CRMController) ListProjectFiles() {
	c.listObjectFiles("projects", "project", "project_file", "project_file")
}

func (c *CRMController) AddProjectFile() {
	c.addObjectFile("projects", "project", "project_file", "project_file")
}

func (c *CRMController) DeleteProjectFile() {
	c.deleteObjectFile("projects", "project", "project_file", "project_file")
}

func (c *CRMController) ListTicketFiles() {
	c.listObjectFiles("tickets", "ticket", "ticket_file", "ticket_file")
}

func (c *CRMController) AddTicketFile() {
	c.addObjectFile("tickets", "ticket", "ticket_file", "ticket_file")
}

func (c *CRMController) DeleteTicketFile() {
	c.deleteObjectFile("tickets", "ticket", "ticket_file", "ticket_file")
}

func (c *CRMController) ListOrderFiles() {
	c.listObjectFiles("orders", "order", "order_file", "order_file")
}

func (c *CRMController) AddOrderFile() {
	c.addObjectFile("orders", "order", "order_file", "order_file")
}

func (c *CRMController) DeleteOrderFile() {
	c.deleteObjectFile("orders", "order", "order_file", "order_file")
}

func (c *CRMController) ListCampaignFiles() {
	c.listObjectFiles("campaigns", "campaign", "campaign_file", "campaign_file")
}

func (c *CRMController) AddCampaignFile() {
	c.addObjectFile("campaigns", "campaign", "campaign_file", "campaign_file")
}

func (c *CRMController) DeleteCampaignFile() {
	c.deleteObjectFile("campaigns", "campaign", "campaign_file", "campaign_file")
}

func (c *CRMController) listObjectFiles(domain string, parentType string, fileType string, relationType string) {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的对象ID")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	parent, ok := loadCRMParentObject(c.Ctx, auth, domain, parentType, id)
	if !ok {
		return
	}
	if !canViewCRMObject(auth, parent) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权查看该对象资料")
		return
	}
	var files []models.CRMObject
	if err := utils.DB.Model(&models.CRMObject{}).
		Joins("JOIN crm_relations rel ON rel.deleted_at IS NULL AND rel.tenant_id = crm_objects.tenant_id AND rel.target_type = crm_objects.domain AND rel.target_id = crm_objects.id").
		Where("crm_objects.tenant_id = ? AND crm_objects.domain = ? AND crm_objects.object_type = ?", auth.TenantID, domain, fileType).
		Where("rel.source_type = ? AND rel.source_id = ? AND rel.relation_type = ?", domain, id, relationType).
		Order("crm_objects.created_at desc").
		Find(&files).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取资料列表失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": files, "total": len(files)})
}

func (c *CRMController) addObjectFile(domain string, parentType string, fileType string, relationType string) {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的对象ID")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	parent, ok := loadCRMParentObject(c.Ctx, auth, domain, parentType, id)
	if !ok {
		return
	}
	if !canFollowCRMObject(auth, parent) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权为该对象上传资料")
		return
	}
	var req crmFileLinkRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.FileID = strings.TrimSpace(req.FileID)
	req.FileName = strings.TrimSpace(req.FileName)
	req.StorageNodeID = strings.TrimSpace(req.StorageNodeID)
	req.DownloadURL = strings.TrimSpace(req.DownloadURL)
	if req.FileID == "" || req.FileName == "" {
		utils.BadRequest(c.Ctx, "file_id 和 file_name 不能为空")
		return
	}
	if req.DownloadURL == "" {
		req.DownloadURL = "/api/v1/files/" + req.FileID + "/download"
	}
	category := normalizeCRMFileCategory(req.Category)
	visibility := normalizeCRMFileVisibility(req.Visibility)
	payload := map[string]interface{}{
		"file_id":         req.FileID,
		"file_name":       req.FileName,
		"file_size":       req.FileSize,
		"file_type":       strings.TrimSpace(req.FileType),
		"storage_node_id": req.StorageNodeID,
		"download_url":    req.DownloadURL,
		"category":        category,
		"version":         defaultText(req.Version, "v1"),
		"visibility":      visibility,
		"remark":          strings.TrimSpace(req.Remark),
		"uploaded_by":     auth.UserID,
		"uploaded_at":     time.Now().Format(time.RFC3339),
	}
	if domain == "tickets" {
		payload["ticket_id"] = parent.ID
	} else if domain == "orders" {
		payload["order_id"] = parent.ID
	} else if domain == "campaigns" {
		payload["campaign_id"] = parent.ID
	} else {
		payload["project_id"] = parent.ID
	}
	fileObject := models.CRMObject{
		TenantModel: models.TenantModel{TenantID: auth.TenantID},
		Domain:      domain,
		ObjectType:  fileType,
		Name:        req.FileName,
		Status:      "active",
		OwnerID:     auth.UserID,
		CompanyID:   parent.CompanyID,
		ContactID:   parent.ContactID,
		DealID:      parent.DealID,
		PayloadJSON: toJSONString(payload),
		CreatedBy:   auth.UserID,
		UpdatedBy:   auth.UserID,
	}
	if domain == "projects" {
		fileObject.ProjectID = parent.ID
	} else if domain == "tickets" {
		fileObject.ProjectID = parent.ProjectID
	} else if domain == "orders" {
		fileObject.ProjectID = parent.ProjectID
	}
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&fileObject).Error; err != nil {
			return err
		}
		parentPayload := payloadMap(parent.PayloadJSON)
		parentPayload["file_count"] = int(numericValue(parentPayload["file_count"])) + 1
		parent.PayloadJSON = toJSONString(parentPayload)
		parent.UpdatedBy = auth.UserID
		if err := tx.Save(&parent).Error; err != nil {
			return err
		}
		relation := newCRMRelation(auth, domain, parent.ID, domain, fileObject.ID, relationType)
		relation.PayloadJSON = toJSONString(map[string]interface{}{"file_id": req.FileID, "storage_node_id": req.StorageNodeID, "download_url": req.DownloadURL, "category": category, "visibility": visibility})
		if err := tx.Create(&relation).Error; err != nil {
			return err
		}
		eventType := strings.TrimSuffix(fileType, "_file") + "_file_upload"
		return createBusinessTimeline(tx, auth, domain, parent.ID, eventType, "上传资料", map[string]interface{}{"file_object_id": fileObject.ID, "file_id": req.FileID, "file_name": req.FileName, "category": category, "visibility": visibility, "reason": strings.TrimSpace(req.Remark)})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存资料记录失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"file": fileObject})
}

func (c *CRMController) deleteObjectFile(domain string, parentType string, fileType string, relationType string) {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	parentID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的对象ID")
		return
	}
	fileObjectID, valid := pathUint(c.Ctx, ":file_object_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的资料ID")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	parent, ok := loadCRMParentObject(c.Ctx, auth, domain, parentType, parentID)
	if !ok {
		return
	}
	var fileObject models.CRMObject
	if err := utils.DB.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, domain, fileType, fileObjectID).First(&fileObject).Error; err != nil {
		utils.NotFound(c.Ctx, "资料不存在")
		return
	}
	if !canDeleteCRMObjectAttachment(auth, parent, fileObject) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权删除该资料")
		return
	}
	payload := payloadMap(fileObject.PayloadJSON)
	fileID, _ := payload["file_id"].(string)
	storageNodeID, _ := payload["storage_node_id"].(string)
	status, _, err := deleteRemoteFileByID(auth, fileID, storageNodeID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusBadGateway, "删除文件服务资料失败: "+err.Error())
		return
	}
	if status != http.StatusNotFound && (status < 200 || status >= 300) {
		utils.Error(c.Ctx, http.StatusBadGateway, "删除文件服务资料失败")
		return
	}
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&fileObject).Error; err != nil {
			return err
		}
		parentPayload := payloadMap(parent.PayloadJSON)
		count := int(numericValue(parentPayload["file_count"]))
		if count > 0 {
			parentPayload["file_count"] = count - 1
			parent.PayloadJSON = toJSONString(parentPayload)
			parent.UpdatedBy = auth.UserID
			if err := tx.Save(&parent).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?", auth.TenantID, domain, parent.ID, domain, fileObject.ID, relationType).Delete(&models.CRMRelation{}).Error; err != nil {
			return err
		}
		eventType := strings.TrimSuffix(fileType, "_file") + "_file_delete"
		return createBusinessTimeline(tx, auth, domain, parent.ID, eventType, "删除资料", map[string]interface{}{"file_object_id": fileObject.ID, "file_id": payload["file_id"], "file_name": fileObject.Name})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "删除资料失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"deleted_id": fileObjectID})
}

func loadCRMParentObject(ctx *context.Context, auth authContext, domain string, parentType string, id uint) (models.CRMObject, bool) {
	var parent models.CRMObject
	if err := crmScopedObjectQuery(auth, domain).Where("id = ? AND object_type = ?", id, parentType).First(&parent).Error; err != nil {
		utils.NotFound(ctx, "业务对象不存在")
		return parent, false
	}
	return parent, true
}

func normalizeCRMFileCategory(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	allowed := map[string]bool{"contract": true, "proposal": true, "meeting": true, "delivery": true, "acceptance": true, "screenshot": true, "log": true, "feedback": true, "resolution": true, "brochure": true, "poster": true, "whitepaper": true, "video": true, "quote": true, "landing": true, "creative": true, "other": true}
	if !allowed[value] {
		return "other"
	}
	return value
}

func normalizeCRMFileVisibility(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "internal" || value == "customer_visible" || value == "private" {
		return value
	}
	return "internal"
}

func createWorkspaceReminder(tx *gorm.DB, auth authContext, source models.CRMObject, title string, content string, remindAt *time.Time, ownerIDs []uint, relationType string, category string) error {
	return createWorkspaceReminderWithPayload(tx, auth, source, title, content, remindAt, ownerIDs, relationType, category, nil)
}

func createWorkspaceReminderWithPayload(tx *gorm.DB, auth authContext, source models.CRMObject, title string, content string, remindAt *time.Time, ownerIDs []uint, relationType string, category string, payloadPatch map[string]interface{}) error {
	if len(ownerIDs) == 0 {
		if source.OwnerID > 0 {
			ownerIDs = append(ownerIDs, source.OwnerID)
		} else if source.CreatedBy > 0 {
			ownerIDs = append(ownerIDs, source.CreatedBy)
		}
	}
	seen := map[uint]bool{}
	for _, ownerID := range ownerIDs {
		if ownerID == 0 || seen[ownerID] {
			continue
		}
		seen[ownerID] = true
		payload := map[string]interface{}{
			"category":  category,
			"source":    relationType,
			"domain":    source.Domain,
			"object_id": source.ID,
			"content":   content,
		}
		for key, value := range payloadPatch {
			payload[key] = value
		}
		if remindAt != nil {
			payload["remind_at"] = remindAt.Format(time.RFC3339)
		}
		notice := cloneCRMObject(auth, "workspace", "mail_notification", title, source)
		notice.Status = "open"
		notice.OwnerID = ownerID
		notice.CreatedBy = 0
		notice.UpdatedBy = 0
		payload["sender_type"] = "system"
		payload["sender_name"] = "系统"
		if _, ok := payload["sender_i18n_key"]; !ok {
			payload["sender_i18n_key"] = "notification.sender.system"
		}
		payload["sender_id"] = 0
		notice.PayloadJSON = toJSONString(payload)
		if err := tx.Create(&notice).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.CRMRelation{TenantModel: models.TenantModel{TenantID: auth.TenantID}, SourceType: source.Domain, SourceID: source.ID, TargetType: "workspace", TargetID: notice.ID, RelationType: relationType}).Error; err != nil {
			return err
		}
		publishSystemNotificationUnread(auth, ownerID)
	}
	return nil
}

func projectFileDownloadAuthorized(auth authContext, fileID string) bool {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" || auth.TenantID == 0 {
		return false
	}
	var fileObject models.CRMObject
	if err := utils.DB.Where("tenant_id = ? AND object_type IN ? AND payload_json LIKE ?", auth.TenantID, []string{"project_file", "ticket_file", "order_file", "campaign_file"}, fmt.Sprintf("%%\"file_id\":\"%s\"%%", fileID)).Order("id desc").First(&fileObject).Error; err != nil {
		return profileFileDownloadAuthorized(auth, fileID) || crmObjectInlineFileDownloadAuthorized(auth, fileID) || mfgRFQFileDownloadAuthorized(auth, fileID)
	}
	parentDomain := fileObject.Domain
	relationType := "project_file"
	if fileObject.ObjectType == "ticket_file" {
		relationType = "ticket_file"
	} else if fileObject.ObjectType == "order_file" {
		relationType = "order_file"
	} else if fileObject.ObjectType == "campaign_file" {
		relationType = "campaign_file"
	}
	var rel models.CRMRelation
	if err := utils.DB.Where("tenant_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?", auth.TenantID, fileObject.Domain, fileObject.ID, relationType).Order("id desc").First(&rel).Error; err != nil {
		return false
	}
	var parent models.CRMObject
	if err := crmScopedObjectQuery(auth, parentDomain).Where("id = ?", rel.SourceID).First(&parent).Error; err != nil {
		return false
	}
	return canViewCRMObject(auth, parent)
}

func mfgRFQFileDownloadAuthorized(auth authContext, fileID string) bool {
	var count int64
	if err := utils.DB.Table("mfg_rfqs").
		Where("tenant_id = ? AND drawing_file_id = ?", auth.TenantID, fileID).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func crmObjectInlineFileDownloadAuthorized(auth authContext, fileID string) bool {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" || auth.TenantID == 0 {
		return false
	}
	var objects []models.CRMObject
	if err := utils.DB.
		Where("tenant_id = ? AND payload_json LIKE ?", auth.TenantID, fmt.Sprintf("%%\"%s\"%%", fileID)).
		Order("id desc").
		Limit(20).
		Find(&objects).Error; err != nil {
		return false
	}
	for _, object := range objects {
		refs := fileRefsFromPayload(payloadMap(object.PayloadJSON), object.Domain, object.ID)
		for _, ref := range refs {
			if ref.FileID == fileID && canViewCRMObject(auth, object) {
				return true
			}
		}
	}
	return false
}

func profileFileDownloadAuthorized(auth authContext, fileID string) bool {
	downloadURL := "/api/v1/files/" + strings.TrimSpace(fileID) + "/download"
	var count int64
	if err := utils.DB.Model(&models.User{}).
		Where("tenant_id = ? AND (avatar_url = ? OR cover_url = ?)", auth.TenantID, downloadURL, downloadURL).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func orderFileObjects(auth authContext, orderID uint) []models.CRMObject {
	var files []models.CRMObject
	_ = utils.DB.Model(&models.CRMObject{}).
		Joins("JOIN crm_relations rel ON rel.deleted_at IS NULL AND rel.tenant_id = crm_objects.tenant_id AND rel.target_type = crm_objects.domain AND rel.target_id = crm_objects.id").
		Where("crm_objects.tenant_id = ? AND crm_objects.domain = ? AND crm_objects.object_type = ?", auth.TenantID, "orders", "order_file").
		Where("rel.source_type = ? AND rel.source_id = ? AND rel.relation_type = ?", "orders", orderID, "order_file").
		Order("crm_objects.created_at asc").
		Find(&files).Error
	return files
}

func campaignFileObjects(auth authContext, campaignID uint) []models.CRMObject {
	var files []models.CRMObject
	_ = utils.DB.Model(&models.CRMObject{}).
		Joins("JOIN crm_relations rel ON rel.deleted_at IS NULL AND rel.tenant_id = crm_objects.tenant_id AND rel.target_type = crm_objects.domain AND rel.target_id = crm_objects.id").
		Where("crm_objects.tenant_id = ? AND crm_objects.domain = ? AND crm_objects.object_type = ?", auth.TenantID, "campaigns", "campaign_file").
		Where("rel.source_type = ? AND rel.source_id = ? AND rel.relation_type = ?", "campaigns", campaignID, "campaign_file").
		Order("crm_objects.created_at desc").
		Find(&files).Error
	return files
}
