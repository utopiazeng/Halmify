package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type crmFileReference struct {
	FileID        string
	StorageNodeID string
	Source        string
}

func cleanupCRMObjectFilesForDelete(auth authContext, object models.CRMObject) error {
	refs := crmFileReferencesForObjectDelete(auth, object)
	return deleteCRMFileReferences(auth, refs)
}

func cleanupCRMObjectReplacedFiles(auth authContext, before models.CRMObject, after models.CRMObject) error {
	refs := removedCRMFileReferences(fileRefsFromPayload(payloadMap(before.PayloadJSON), before.Domain, before.ID), fileRefsFromPayload(payloadMap(after.PayloadJSON), after.Domain, after.ID))
	return deleteCRMFileReferences(auth, refs)
}

func crmFileReferencesForObjectDelete(auth authContext, object models.CRMObject) []crmFileReference {
	refs := fileRefsFromPayload(payloadMap(object.PayloadJSON), object.Domain, object.ID)
	refs = append(refs, childFileObjectReferences(auth, object)...)
	return dedupeCRMFileReferences(refs)
}

func childFileObjectReferences(auth authContext, object models.CRMObject) []crmFileReference {
	fileTypes := childFileTypesForObject(object)
	if len(fileTypes) == 0 {
		return nil
	}
	var files []models.CRMObject
	_ = utils.DB.Model(&models.CRMObject{}).
		Joins("JOIN crm_relations rel ON rel.deleted_at IS NULL AND rel.tenant_id = crm_objects.tenant_id AND rel.target_type = crm_objects.domain AND rel.target_id = crm_objects.id").
		Where("crm_objects.tenant_id = ? AND crm_objects.domain = ? AND crm_objects.object_type IN ?", auth.TenantID, object.Domain, fileTypes).
		Where("rel.source_type = ? AND rel.source_id = ?", object.Domain, object.ID).
		Find(&files).Error
	refs := make([]crmFileReference, 0, len(files))
	for _, file := range files {
		refs = append(refs, fileRefsFromPayload(payloadMap(file.PayloadJSON), file.Domain, file.ID)...)
	}
	return refs
}

func childFileTypesForObject(object models.CRMObject) []string {
	switch object.Domain {
	case "projects":
		if object.ObjectType == "project" {
			return []string{"project_file"}
		}
	case "tickets":
		if object.ObjectType == "ticket" {
			return []string{"ticket_file"}
		}
	case "orders":
		if object.ObjectType == "order" {
			return []string{"order_file"}
		}
	case "campaigns":
		if object.ObjectType == "campaign" {
			return []string{"campaign_file"}
		}
	}
	return nil
}

func fileRefsFromPayload(payload map[string]interface{}, source string, objectID uint) []crmFileReference {
	refs := make([]crmFileReference, 0)
	collectFileRefsFromValue(payload, "", fmt.Sprintf("%s:%d", source, objectID), &refs)
	return dedupeCRMFileReferences(refs)
}

func collectFileRefsFromValue(value interface{}, inheritedNodeID string, source string, refs *[]crmFileReference) {
	switch typed := value.(type) {
	case map[string]interface{}:
		nodeID := inheritedNodeID
		if currentNodeID := firstFileText(typed, "storage_node_id", "node_id"); currentNodeID != "" {
			nodeID = currentNodeID
		}
		for _, key := range []string{"file_id", "image_file_id", "avatar_file_id", "cover_file_id", "attachment_file_id"} {
			if fileID := strings.TrimSpace(fmt.Sprint(typed[key])); fileID != "" && fileID != "<nil>" {
				*refs = append(*refs, crmFileReference{FileID: fileID, StorageNodeID: nodeID, Source: source})
			}
		}
		for _, child := range typed {
			collectFileRefsFromValue(child, nodeID, source, refs)
		}
	case []interface{}:
		for _, child := range typed {
			collectFileRefsFromValue(child, inheritedNodeID, source, refs)
		}
	case []map[string]interface{}:
		for _, child := range typed {
			collectFileRefsFromValue(child, inheritedNodeID, source, refs)
		}
	}
}

func firstFileText(payload map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(fmt.Sprint(payload[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func removedCRMFileReferences(before []crmFileReference, after []crmFileReference) []crmFileReference {
	afterSet := map[string]bool{}
	for _, ref := range after {
		afterSet[ref.FileID] = true
	}
	removed := make([]crmFileReference, 0)
	for _, ref := range before {
		if ref.FileID != "" && !afterSet[ref.FileID] {
			removed = append(removed, ref)
		}
	}
	return dedupeCRMFileReferences(removed)
}

func dedupeCRMFileReferences(refs []crmFileReference) []crmFileReference {
	seen := map[string]bool{}
	items := make([]crmFileReference, 0, len(refs))
	for _, ref := range refs {
		ref.FileID = strings.TrimSpace(ref.FileID)
		ref.StorageNodeID = strings.TrimSpace(ref.StorageNodeID)
		if ref.FileID == "" || seen[ref.FileID] {
			continue
		}
		seen[ref.FileID] = true
		items = append(items, ref)
	}
	return items
}

func deleteCRMFileReferences(auth authContext, refs []crmFileReference) error {
	for _, ref := range dedupeCRMFileReferences(refs) {
		status, _, err := deleteRemoteFileByID(auth, ref.FileID, ref.StorageNodeID)
		if err != nil {
			return fmt.Errorf("删除文件资源 %s 失败: %w", ref.FileID, err)
		}
		if status != http.StatusNotFound && (status < 200 || status >= 300) {
			return fmt.Errorf("删除文件资源 %s 失败，文件服务返回 %d", ref.FileID, status)
		}
	}
	return nil
}

type tenantFileUsageSummary struct {
	UsedBytes     uint64  `json:"used_bytes"`
	UsedMB        float64 `json:"used_mb"`
	PhysicalCount int64   `json:"physical_count"`
	RecordCount   int64   `json:"record_count"`
}

type tenantFileDeleteSummary struct {
	DeletedBytes         uint64  `json:"deleted_bytes"`
	DeletedMB            float64 `json:"deleted_mb"`
	DeletedPhysicalCount int64   `json:"deleted_physical_count"`
	DeletedRecordCount   int64   `json:"deleted_record_count"`
}

func tenantFileUsage(tenantID uint) (tenantFileUsageSummary, error) {
	var summary tenantFileUsageSummary
	for _, node := range services.FileserverNodesSnapshot() {
		status, payload, err := services.GetInternalJSON(node, "/internal/v1/files/tenant/usage", "", tenantID)
		if err != nil {
			return summary, fmt.Errorf("读取文件节点 %s 租户用量失败: %w", node.ID, err)
		}
		if status < 200 || status >= 300 {
			return summary, fmt.Errorf("读取文件节点 %s 租户用量失败，文件服务返回 %d", node.ID, status)
		}
		var resp struct {
			Data struct {
				UsedBytes     uint64 `json:"used_bytes"`
				PhysicalCount int64  `json:"physical_count"`
				RecordCount   int64  `json:"record_count"`
			} `json:"data"`
		}
		if err := json.Unmarshal(payload, &resp); err != nil {
			return summary, fmt.Errorf("读取文件节点 %s 租户用量返回异常: %w", node.ID, err)
		}
		summary.UsedBytes += resp.Data.UsedBytes
		summary.PhysicalCount += resp.Data.PhysicalCount
		summary.RecordCount += resp.Data.RecordCount
	}
	summary.UsedMB = bytesToMB(summary.UsedBytes)
	return summary, nil
}

func cleanupTenantPhysicalFiles(auth authContext, tenantID uint) (tenantFileDeleteSummary, error) {
	var summary tenantFileDeleteSummary
	if tenantID == 0 {
		return summary, nil
	}
	tenantAuth := auth
	tenantAuth.TenantID = tenantID
	nodes := services.FileserverNodesSnapshot()
	if len(nodes) == 0 {
		return summary, fmt.Errorf("未配置文件服务节点，无法确认租户文件已物理删除")
	}
	for _, node := range nodes {
		status, payload, err := services.DeleteInternal(node, "/internal/v1/files/tenant", tenantAuth.TenantID)
		if err != nil {
			return summary, fmt.Errorf("删除文件节点 %s 的租户文件失败: %w", node.ID, err)
		}
		if status < 200 || status >= 300 {
			return summary, fmt.Errorf("删除文件节点 %s 的租户文件失败，文件服务返回 %d", node.ID, status)
		}
		var resp struct {
			Data struct {
				DeletedBytes         uint64 `json:"deleted_bytes"`
				DeletedPhysicalCount int64  `json:"deleted_physical_count"`
				DeletedRecordCount   int64  `json:"deleted_record_count"`
			} `json:"data"`
		}
		if err := json.Unmarshal(payload, &resp); err != nil {
			return summary, fmt.Errorf("删除文件节点 %s 的租户文件返回异常: %w", node.ID, err)
		}
		summary.DeletedBytes += resp.Data.DeletedBytes
		summary.DeletedPhysicalCount += resp.Data.DeletedPhysicalCount
		summary.DeletedRecordCount += resp.Data.DeletedRecordCount
	}
	summary.DeletedMB = bytesToMB(summary.DeletedBytes)
	return summary, nil
}

func bytesToMB(value uint64) float64 {
	return float64(value) / 1024 / 1024
}
