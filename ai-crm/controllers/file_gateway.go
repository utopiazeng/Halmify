package controllers

import (
	"ai-crm/services"
	"ai-crm/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	bctx "github.com/beego/beego/v2/server/web/context"
)

type FileGatewayController struct {
	web.Controller
}

type fileCheckRequest struct {
	MD5      string `json:"md5"`
	FileSize uint64 `json:"file_size"`
}

func (c *FileGatewayController) Check() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Error(c.Ctx, http.StatusForbidden, "缺少租户上下文")
		return
	}
	var req fileCheckRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.MD5 = strings.ToLower(strings.TrimSpace(req.MD5))
	if req.MD5 == "" || len(req.MD5) != 32 {
		utils.Error(c.Ctx, http.StatusBadRequest, "md5参数无效")
		return
	}
	body, _ := json.Marshal(req)
	nodes := services.FileserverNodesSnapshot()
	for _, node := range nodes {
		if !node.Healthy {
			continue
		}
		status, payload, err := services.PostInternalJSON(node, "/internal/v1/files/check", auth.TenantID, body)
		if err != nil {
			continue
		}
		if status >= 200 && status < 300 {
			writeRawJSON(c.Ctx, status, payload)
			return
		}
	}
	utils.Error(c.Ctx, http.StatusServiceUnavailable, "文件服务不可用")
}

func (c *FileGatewayController) Upload() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Error(c.Ctx, http.StatusForbidden, "缺少租户上下文")
		return
	}
	maxUploadBytes := currentUploadQuotaConfig().MaxFileBytes
	if maxUploadBytes > 0 && c.Ctx.Request.ContentLength > maxUploadBytes+1024*1024 {
		utils.Error(c.Ctx, http.StatusRequestEntityTooLarge, "文件大小超过上传限制")
		return
	}
	file, header, err := c.GetFile("file")
	if err != nil {
		utils.Error(c.Ctx, http.StatusBadRequest, "缺少上传文件")
		return
	}
	defer file.Close()
	if err := validateUploadSize(auth, header); err != nil {
		if errors.Is(err, errUploadLimitExceeded) {
			utils.Error(c.Ctx, http.StatusRequestEntityTooLarge, "文件大小超过上传限制")
			return
		}
		if errors.Is(err, errTenantStorageInsufficient) {
			utils.Error(c.Ctx, http.StatusForbidden, "租户存储空间不足，请清理资料或升级空间")
			return
		}
		if errors.Is(err, errTenantStorageUploadLocked) {
			utils.Error(c.Ctx, http.StatusForbidden, "存储增购已到期，请续费后继续上传")
			return
		}
		utils.LogWarning("文件上传前置校验失败 tenant=%d user=%d file=%s size=%d err=%v nodes=%s", auth.TenantID, auth.UserID, header.Filename, uploadFileSize(header), err, fileserverNodesDiagnostic())
		utils.Error(c.Ctx, http.StatusServiceUnavailable, err.Error())
		return
	}
	node, err := services.ChooseUploadNodeRealtime()
	if err != nil {
		utils.LogWarning("文件上传无可用文件节点 tenant=%d user=%d file=%s size=%d err=%v nodes=%s", auth.TenantID, auth.UserID, header.Filename, uploadFileSize(header), err, fileserverNodesDiagnostic())
		utils.Error(c.Ctx, http.StatusServiceUnavailable, "暂无可用文件服务节点")
		return
	}
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	contentType := writer.FormDataContentType()
	go func() {
		defer pipeWriter.Close()
		defer writer.Close()
		part, err := writer.CreateFormFile("file", header.Filename)
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
			return
		}
		if err := copyUploadWithLimit(part, file, maxUploadBytes); err != nil {
			_ = pipeWriter.CloseWithError(err)
			return
		}
	}()

	status, payload, err := services.ProxyUpload(*node, auth.TenantID, auth.Username, pipeReader, contentType, -1)
	if err != nil {
		if errors.Is(err, errUploadLimitExceeded) {
			utils.Error(c.Ctx, http.StatusRequestEntityTooLarge, "文件大小超过上传限制")
			return
		}
		utils.LogWarning("文件上传代理失败 tenant=%d user=%d node=%s base=%s file=%s size=%d err=%v", auth.TenantID, auth.UserID, node.ID, node.BaseURL, header.Filename, uploadFileSize(header), err)
		utils.Error(c.Ctx, http.StatusBadGateway, "上传失败: "+err.Error())
		return
	}
	if status < 200 || status >= 300 {
		utils.LogWarning("文件上传文件服务返回异常 tenant=%d user=%d node=%s base=%s file=%s size=%d status=%d payload=%s", auth.TenantID, auth.UserID, node.ID, node.BaseURL, header.Filename, uploadFileSize(header), status, strings.TrimSpace(string(payload)))
	}
	if status >= 200 && status < 300 {
		payload = decorateFilePayload(payload, node.ID)
	}
	writeRawJSON(c.Ctx, status, payload)
}

func (c *FileGatewayController) Download() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Error(c.Ctx, http.StatusForbidden, "缺少租户上下文")
		return
	}
	fileID := strings.TrimSpace(c.Ctx.Input.Param(":file_id"))
	if fileID == "" {
		utils.Error(c.Ctx, http.StatusBadRequest, "缺少file_id")
		return
	}
	if !projectFileDownloadAuthorized(auth, fileID) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权下载该业务资料")
		return
	}
	limit := applyDownloadLimit(auth, c.Ctx.Input.IP())
	if !limit.Allowed {
		traceID := strings.TrimSpace(utils.NewSessionID())
		if len(traceID) > 16 {
			traceID = traceID[:16]
		}
		if limit.RetryAfter > 0 {
			c.Ctx.Output.Header("Retry-After", strconv.FormatInt(int64(limit.RetryAfter.Seconds()), 10))
		}
		logControllerOperation(c.Ctx, auth, auth.TenantID, "SECURITY_RATE_LIMIT", "files["+fileID+"]", "operation.security_rate_limit", map[string]interface{}{
			"file_id":     fileID,
			"path":        c.Ctx.Request.URL.Path,
			"trace_id":    traceID,
			"retry_after": int(limit.RetryAfter.Seconds()),
		}, "failed", "文件下载触发限流")
		utils.Error(c.Ctx, http.StatusTooManyRequests, "下载过于频繁，请稍后再试")
		return
	}
	if limit.ReleaseFunc != nil {
		defer limit.ReleaseFunc()
	}
	nodes := orderedFileserverNodes(storageNodeIDForFile(auth, fileID))
	query := c.Ctx.Request.URL.RawQuery
	rangeHeader := c.Ctx.Request.Header.Get("Range")
	path := fmt.Sprintf("/internal/v1/files/%s/download", fileID)
	for _, node := range nodes {
		if !node.Healthy {
			continue
		}
		resp, err := services.GetInternal(node, path, query, auth.TenantID, rangeHeader)
		if err != nil {
			continue
		}
		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			continue
		}
		defer resp.Body.Close()
		for key, values := range resp.Header {
			for _, v := range values {
				c.Ctx.Output.Header(key, v)
			}
		}
		c.Ctx.Output.SetStatus(resp.StatusCode)
		_, _ = io.Copy(c.Ctx.ResponseWriter, resp.Body)
		return
	}
	utils.Error(c.Ctx, http.StatusNotFound, "文件不存在")
}

func (c *FileGatewayController) Info() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Error(c.Ctx, http.StatusForbidden, "缺少租户上下文")
		return
	}
	fileID := strings.TrimSpace(c.Ctx.Input.Param(":file_id"))
	if fileID == "" {
		utils.Error(c.Ctx, http.StatusBadRequest, "缺少file_id")
		return
	}
	nodes := orderedFileserverNodes(strings.TrimSpace(c.Ctx.Input.Query("storage_node_id")))
	path := fmt.Sprintf("/internal/v1/files/%s/info", fileID)
	for _, node := range nodes {
		if !node.Healthy {
			continue
		}
		resp, err := services.GetInternal(node, path, "", auth.TenantID, "")
		if err != nil {
			continue
		}
		payload, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			continue
		}
		writeRawJSON(c.Ctx, resp.StatusCode, payload)
		return
	}
	utils.Error(c.Ctx, http.StatusNotFound, "文件不存在")
}

func (c *FileGatewayController) Delete() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Error(c.Ctx, http.StatusForbidden, "缺少租户上下文")
		return
	}
	fileID := strings.TrimSpace(c.Ctx.Input.Param(":file_id"))
	if fileID == "" {
		utils.Error(c.Ctx, http.StatusBadRequest, "缺少file_id")
		return
	}
	status, payload, err := deleteRemoteFileByID(auth, fileID, strings.TrimSpace(c.Ctx.Input.Query("storage_node_id")))
	if err != nil {
		utils.Error(c.Ctx, http.StatusBadGateway, "删除文件失败: "+err.Error())
		return
	}
	if status == http.StatusNotFound {
		utils.Error(c.Ctx, http.StatusNotFound, "文件不存在")
		return
	}
	writeRawJSON(c.Ctx, status, payload)
}

func (c *FileGatewayController) DiskSpace() {
	nodes := services.FileserverNodesSnapshot()
	var total uint64
	var free uint64
	var used uint64
	healthy := 0
	for _, node := range nodes {
		if !node.Healthy {
			continue
		}
		healthy++
		total += node.TotalBytes
		free += node.FreeBytes
		used += node.UsedBytes
	}
	if healthy == 0 {
		utils.Error(c.Ctx, http.StatusServiceUnavailable, "暂无可用文件服务节点")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"total_bytes": total,
		"free_bytes":  free,
		"used_bytes":  used,
		"total_gb":    formatGB(total),
		"free_gb":     formatGB(free),
		"used_gb":     formatGB(used),
		"node_count":  healthy,
	})
}

func (c *FileGatewayController) ListNodes() {
	snapshot := services.FileserverNodeEvaluationView()
	totalNodes := len(snapshot.Nodes)
	healthyNodes := 0
	candidateNodes := 0
	for _, node := range snapshot.Nodes {
		if node.Healthy {
			healthyNodes++
		}
		if node.IsCandidate {
			candidateNodes++
		}
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"summary": map[string]interface{}{
			"total_nodes":         totalNodes,
			"healthy_nodes":       healthyNodes,
			"candidate_nodes":     candidateNodes,
			"recommended_node_id": snapshot.RecommendedNodeID,
		},
		"thresholds": snapshot.Thresholds,
		"nodes":      snapshot.Nodes,
	})
}

func writeRawJSON(ctx *bctx.Context, status int, payload []byte) {
	if status <= 0 {
		status = http.StatusOK
	}
	ctx.Output.SetStatus(status)
	ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
	_ = ctx.Output.Body(payload)
}

func orderedFileserverNodes(preferredNodeID string) []services.FileserverNode {
	nodes := services.FileserverNodesSnapshot()
	preferredNodeID = strings.TrimSpace(preferredNodeID)
	if preferredNodeID == "" {
		return nodes
	}
	result := make([]services.FileserverNode, 0, len(nodes))
	for _, node := range nodes {
		if node.ID == preferredNodeID {
			result = append(result, node)
			break
		}
	}
	for _, node := range nodes {
		if node.ID != preferredNodeID {
			result = append(result, node)
		}
	}
	return result
}

func fileserverNodesDiagnostic() string {
	snapshot := services.FileserverNodeEvaluationView()
	parts := make([]string, 0, len(snapshot.Nodes))
	for _, node := range snapshot.Nodes {
		parts = append(parts, fmt.Sprintf(
			"id=%s base=%s healthy=%t candidate=%t reasons=%s last_error=%q free=%d used=%d total=%d storage_percent=%.2f checked=%s",
			node.ID,
			node.BaseURL,
			node.Healthy,
			node.IsCandidate,
			strings.Join(node.Reasons, ","),
			node.LastError,
			node.FreeBytes,
			node.UsedBytes,
			node.TotalBytes,
			node.StoragePercent,
			node.LastCheckedAt.Format(time.RFC3339),
		))
	}
	return fmt.Sprintf("thresholds={min_free=%d high_watermark=%.2f cache_ttl=%d} nodes=[%s]",
		snapshot.Thresholds.MinFreeBytes,
		snapshot.Thresholds.HighWatermarkPercent,
		snapshot.Thresholds.CacheTTLSec,
		strings.Join(parts, " | "),
	)
}

func storageNodeIDForFile(auth authContext, fileID string) string {
	var fileObject struct {
		PayloadJSON string
	}
	if err := utils.DB.Table("crm_objects").
		Select("payload_json").
		Where("tenant_id = ? AND object_type IN ? AND payload_json LIKE ?", auth.TenantID, []string{"project_file", "ticket_file", "order_file", "campaign_file"}, fmt.Sprintf("%%\"file_id\":\"%s\"%%", fileID)).
		Order("id desc").
		Limit(1).
		Scan(&fileObject).Error; err != nil {
		return ""
	}
	payload := payloadMap(fileObject.PayloadJSON)
	if value, ok := payload["storage_node_id"].(string); ok {
		return strings.TrimSpace(value)
	}
	var rfq struct {
		DrawingStorageNodeID string
	}
	if err := utils.DB.Table("mfg_rfqs").
		Select("drawing_storage_node_id").
		Where("tenant_id = ? AND drawing_file_id = ?", auth.TenantID, fileID).
		Order("id desc").
		Limit(1).
		Scan(&rfq).Error; err == nil && strings.TrimSpace(rfq.DrawingStorageNodeID) != "" {
		return strings.TrimSpace(rfq.DrawingStorageNodeID)
	}
	return ""
}

func deleteRemoteFileByID(auth authContext, fileID string, storageNodeID string) (int, []byte, error) {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return http.StatusBadRequest, nil, errors.New("missing file_id")
	}
	path := fmt.Sprintf("/internal/v1/files/%s", fileID)
	foundNotFound := false
	for _, node := range orderedFileserverNodes(storageNodeID) {
		if !node.Healthy {
			continue
		}
		status, payload, err := services.DeleteInternal(node, path, auth.TenantID)
		if err != nil {
			continue
		}
		if status == http.StatusNotFound {
			foundNotFound = true
			if strings.TrimSpace(storageNodeID) != "" && node.ID == strings.TrimSpace(storageNodeID) {
				return status, payload, nil
			}
			continue
		}
		return status, payload, nil
	}
	if foundNotFound {
		return http.StatusNotFound, []byte(`{"code":404,"message":"file not found","data":null}`), nil
	}
	return 0, nil, errors.New("no reachable fileserver node")
}

func decorateFilePayload(payload []byte, storageNodeID string) []byte {
	var envelope struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.Data == nil {
		return payload
	}
	fileID, _ := envelope.Data["file_id"].(string)
	if strings.TrimSpace(fileID) == "" {
		return payload
	}
	envelope.Data["storage_node_id"] = storageNodeID
	envelope.Data["download_url"] = "/api/v1/files/" + strings.TrimSpace(fileID) + "/download"
	next, err := json.Marshal(envelope)
	if err != nil {
		return payload
	}
	return next
}

func formatGB(value uint64) string {
	gb := float64(value) / (1024 * 1024 * 1024)
	return strconv.FormatFloat(gb, 'f', 2, 64)
}
