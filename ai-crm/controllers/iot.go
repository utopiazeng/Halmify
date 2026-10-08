package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"archive/zip"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
)

const (
	iotDefaultWidth             = 200
	iotDefaultHeight            = 200
	iotRefreshInterval          = 600
	iotDashboardCacheTTL        = 5 * time.Minute
	iotOnlineWindow             = 25 * time.Minute
	iotTimestampSkew            = 5 * time.Minute
	iotNonceTTL                 = 10 * time.Minute
	iotDeviceUpdateInterval     = time.Minute
	iotDeviceRateWindow         = 30 * time.Second
	iotTenantRateWindow         = time.Minute
	iotTenantRateLimit          = 120
	iotPreAuthIPRateWindow      = time.Minute
	iotPreAuthIPRateLimit       = 60
	iotFailedRateWindow         = 5 * time.Minute
	iotFailedSNRateLimit        = 20
	iotFailedIPRateLimit        = 80
	iotMaxSNLength              = 128
	iotMaxNonceLength           = 80
	iotMaxTimestampLength       = 16
	iotMaxKeyVersionLength      = 16
	iotHMACSignatureHexLength   = 64
	iotDeviceSignatureHexLength = 128
	iotMaxUserAgentLength       = 512
	iotDashboardCacheVersion    = "v1"
	iotKeyDerivationContext     = "iot-signing-key:v1"
	iotSeedWrapContext          = "iot-device-seed-wrap:v1"
	iotDefaultMasterSecret      = "ac4n6AB6wPugeFz6gHYcoTHTQrpSWJsqYYQqSxLVA3c"
	iotDefaultFirmwareModel     = "esp32-s3-epaper"
	iotOTADownloadTokenTTL      = 15 * time.Minute
	iotMaxFirmwareBytes         = 0x3B0000
	iotMaxFirmwarePackageBytes  = iotMaxFirmwareBytes + 512*1024
)

type IoTController struct {
	web.Controller
}

func (c *IoTController) Prepare() {
	if strings.HasPrefix(c.Ctx.Request.URL.Path, "/api/iot/v1/") {
		c.logIoTTerminalDebug("request_enter", http.StatusOK, "request_received", "", map[string]interface{}{
			"content_length": c.Ctx.Request.ContentLength,
			"body_bytes":     len(c.Ctx.Input.RequestBody),
		})
	}
}

func (c *IoTController) Finish() {
	if strings.HasPrefix(c.Ctx.Request.URL.Path, "/api/iot/v1/") {
		c.logIoTTerminalDebug("request_finish", c.Ctx.Output.Status, "request_finished", "", map[string]interface{}{
			"body_bytes": len(c.Ctx.Input.RequestBody),
		})
	}
}

type iotDeviceRequest struct {
	Name             string `json:"name"`
	SN               string `json:"sn"`
	Model            string `json:"model"`
	Status           string `json:"status"`
	ResolutionWidth  int    `json:"resolution_width"`
	ResolutionHeight int    `json:"resolution_height"`
	Remark           string `json:"remark"`
}

type iotBootstrapRequest struct {
	SN              string `json:"sn"`
	Model           string `json:"model"`
	PublicKeyPEM    string `json:"public_key"`
	Timestamp       string `json:"timestamp"`
	Nonce           string `json:"nonce"`
	Signature       string `json:"signature"`
	FirmwareVersion string `json:"firmware_version"`
}

type iotSeedRefreshRequest struct {
	SN              string `json:"sn"`
	KeyVersion      string `json:"key_version"`
	Timestamp       string `json:"timestamp"`
	Nonce           string `json:"nonce"`
	Signature       string `json:"signature"`
	DeviceSignature string `json:"device_signature"`
	Confirm         bool   `json:"confirm"`
}

type iotOTAConfirmRequest struct {
	Version   string `json:"version"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
	Nonce     string `json:"nonce"`
}

type iotFirmwarePublishRequest struct {
	Target   string   `json:"target"`
	SNs      []string `json:"sns"`
	TenantID uint     `json:"tenant_id"`
	Force    bool     `json:"force"`
}

func (c *IoTController) ListDevices() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canViewIoT(auth) {
		utils.Forbidden(c.Ctx, "无权限查看IoT设备")
		return
	}
	var devices []models.IoTDevice
	query := utils.DB.Where("tenant_id = ?", auth.TenantID).Order("updated_at desc")
	if status := strings.TrimSpace(c.Ctx.Input.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&devices).Error; err != nil {
		utils.InternalServerError(c.Ctx, "获取设备列表失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(devices))
	for _, device := range devices {
		items = append(items, iotDevicePayload(device))
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"items":                      items,
		"total":                      len(items),
		"online_window_sec":          int(iotOnlineWindow.Seconds()),
		"expected_sync_interval_sec": iotRefreshInterval,
	})
}

func (c *IoTController) CreateDevice() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageIoT(auth) {
		utils.Forbidden(c.Ctx, "无权限管理IoT设备")
		return
	}
	var req iotDeviceRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.SN = normalizeIoTSN(req.SN)
	req.Model = normalizeIoTModel(req.Model)
	if req.Name == "" || req.SN == "" {
		utils.BadRequest(c.Ctx, "设备名称和SN不能为空")
		return
	}
	device := models.IoTDevice{
		TenantModel:      models.TenantModel{TenantID: auth.TenantID},
		Name:             req.Name,
		SN:               req.SN,
		Model:            req.Model,
		SecretHash:       "pending:v1",
		Status:           "pending_bind",
		ResolutionWidth:  normalizeIoTDimension(req.ResolutionWidth),
		ResolutionHeight: normalizeIoTDimension(req.ResolutionHeight),
		Remark:           strings.TrimSpace(req.Remark),
	}
	if err := utils.DB.Create(&device).Error; err != nil {
		if isDuplicateDBError(err) {
			utils.BadRequest(c.Ctx, "设备SN已存在")
			return
		}
		utils.InternalServerError(c.Ctx, "创建设备失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_IOT_DEVICE", "IoT设备["+device.SN+"]", "operation.create_iot_device", map[string]interface{}{"id": device.ID, "sn": device.SN}, "success", "")
	payload := iotDevicePayload(device)
	utils.Success(c.Ctx, payload)
}

func (c *IoTController) UpdateDevice() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageIoT(auth) {
		utils.Forbidden(c.Ctx, "无权限管理IoT设备")
		return
	}
	id, ok := pathUint(c.Ctx, ":id")
	if !ok {
		utils.BadRequest(c.Ctx, "设备ID无效")
		return
	}
	var device models.IoTDevice
	if err := utils.DB.Where("tenant_id = ? AND id = ?", auth.TenantID, id).First(&device).Error; err != nil {
		utils.NotFound(c.Ctx, "设备不存在")
		return
	}
	var req iotDeviceRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		utils.BadRequest(c.Ctx, "设备名称不能为空")
		return
	}
	updates := map[string]interface{}{
		"name":              name,
		"status":            normalizeIoTUpdateStatus(req.Status, device.PublicKeyPEM != ""),
		"resolution_width":  normalizeIoTDimension(req.ResolutionWidth),
		"resolution_height": normalizeIoTDimension(req.ResolutionHeight),
		"remark":            strings.TrimSpace(req.Remark),
	}
	if err := utils.DB.Model(&device).Updates(updates).Error; err != nil {
		utils.InternalServerError(c.Ctx, "更新设备失败")
		return
	}
	_ = utils.DB.Where("tenant_id = ? AND id = ?", auth.TenantID, id).First(&device).Error
	logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_IOT_DEVICE", "IoT设备["+device.SN+"]", "operation.update_iot_device", map[string]interface{}{"id": device.ID, "sn": device.SN}, "success", "")
	utils.Success(c.Ctx, iotDevicePayload(device))
}

func (c *IoTController) DeleteDevice() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageIoT(auth) {
		utils.Forbidden(c.Ctx, "无权限管理IoT设备")
		return
	}
	id, ok := pathUint(c.Ctx, ":id")
	if !ok {
		utils.BadRequest(c.Ctx, "设备ID无效")
		return
	}
	var device models.IoTDevice
	if err := utils.DB.Where("tenant_id = ? AND id = ?", auth.TenantID, id).First(&device).Error; err != nil {
		utils.NotFound(c.Ctx, "设备不存在")
		return
	}
	if err := utils.DB.Delete(&device).Error; err != nil {
		utils.InternalServerError(c.Ctx, "删除设备失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "DELETE_IOT_DEVICE", "IoT设备["+device.SN+"]", "operation.delete_iot_device", map[string]interface{}{"id": device.ID, "sn": device.SN}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"deleted": true})
}

func (c *IoTController) RotateDeviceSecret() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canManageIoT(auth) {
		utils.Forbidden(c.Ctx, "无权限管理IoT设备")
		return
	}
	id, ok := pathUint(c.Ctx, ":id")
	if !ok {
		utils.BadRequest(c.Ctx, "设备ID无效")
		return
	}
	var device models.IoTDevice
	if err := utils.DB.Where("tenant_id = ? AND id = ?", auth.TenantID, id).First(&device).Error; err != nil {
		utils.NotFound(c.Ctx, "设备不存在")
		return
	}
	if device.PublicKeyPEM == "" {
		utils.BadRequest(c.Ctx, "设备尚未上线绑定，无法轮换密钥")
		return
	}
	_, sealedSeed, err := newIoTDeviceSeed()
	if err != nil {
		utils.InternalServerError(c.Ctx, "生成设备密钥失败")
		return
	}
	if err := utils.DB.Model(&device).Update("pending_secret_hash", sealedSeed).Error; err != nil {
		utils.InternalServerError(c.Ctx, "轮换设备密钥失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "ROTATE_IOT_DEVICE_SECRET", "IoT设备["+device.SN+"]", "operation.rotate_iot_device_secret", map[string]interface{}{"id": device.ID, "sn": device.SN}, "success", "")
	payload := iotDevicePayload(device)
	payload["pending_rotation"] = true
	utils.Success(c.Ctx, payload)
}

func (c *IoTController) PreviewDevice() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !canViewIoT(auth) {
		utils.Forbidden(c.Ctx, "无权限查看IoT设备")
		return
	}
	id, ok := pathUint(c.Ctx, ":id")
	if !ok {
		utils.BadRequest(c.Ctx, "设备ID无效")
		return
	}
	var device models.IoTDevice
	if err := utils.DB.Where("tenant_id = ? AND id = ?", auth.TenantID, id).First(&device).Error; err != nil {
		utils.NotFound(c.Ctx, "设备不存在")
		return
	}
	year := queryYear(c.Ctx.Input.Query("year"))
	payload := buildIoTDashboardPayload(device, year)
	utils.Success(c.Ctx, payload)
}

func (c *IoTController) ListFirmwares() {
	if _, ok := requirePlatformAdmin(c.Ctx); !ok {
		return
	}
	var releases []models.IoTFirmwareRelease
	if err := utils.DB.Order("created_at desc").Limit(200).Find(&releases).Error; err != nil {
		utils.InternalServerError(c.Ctx, "获取固件列表失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(releases))
	releaseIDs := make([]uint, 0, len(releases))
	for _, release := range releases {
		releaseIDs = append(releaseIDs, release.ID)
	}
	statsByRelease := map[uint]map[string]int64{}
	if len(releaseIDs) > 0 {
		var rows []struct {
			ReleaseID uint
			Status    string
			Count     int64
		}
		if err := utils.DB.Model(&models.IoTFirmwareTarget{}).Select("release_id, status, count(*) as count").Where("release_id IN ?", releaseIDs).Group("release_id, status").Scan(&rows).Error; err != nil {
			utils.InternalServerError(c.Ctx, "获取固件发布统计失败")
			return
		}
		for _, row := range rows {
			if _, exists := statsByRelease[row.ReleaseID]; !exists {
				statsByRelease[row.ReleaseID] = map[string]int64{}
			}
			statsByRelease[row.ReleaseID][row.Status] = row.Count
		}
	}
	for _, release := range releases {
		stats := statsByRelease[release.ID]
		item := map[string]interface{}{}
		data, _ := json.Marshal(release)
		_ = json.Unmarshal(data, &item)
		item["target_count"] = sumIoTFirmwareTargetStats(stats)
		item["checked_count"] = sumIoTFirmwareTargetStatsByStatus(stats, "available", "downloading", "success", "failed", "latest", "release_missing")
		item["downloading_count"] = sumIoTFirmwareTargetStatsByStatus(stats, "downloading")
		item["success_count"] = sumIoTFirmwareTargetStatsByStatus(stats, "success")
		item["failed_count"] = sumIoTFirmwareTargetStatsByStatus(stats, "failed", "release_missing")
		items = append(items, item)
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": len(items)})
}

func (c *IoTController) ListFirmwareTargets() {
	if _, ok := requirePlatformAdmin(c.Ctx); !ok {
		return
	}
	id, ok := pathUint(c.Ctx, ":id")
	if !ok {
		utils.BadRequest(c.Ctx, "固件ID无效")
		return
	}
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 200 {
		size = 20
	}
	var total int64
	query := utils.DB.Model(&models.IoTFirmwareTarget{}).Where("release_id = ?", id)
	if status := strings.TrimSpace(c.Ctx.Input.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if sn := normalizeIoTSN(c.Ctx.Input.Query("sn")); sn != "" {
		query = query.Where("sn LIKE ?", "%"+sn+"%")
	}
	if tenantID := queryInt(c.Ctx, "tenant_id", 0); tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Count(&total).Error; err != nil {
		utils.InternalServerError(c.Ctx, "获取发布记录失败")
		return
	}
	var targets []models.IoTFirmwareTarget
	if err := query.Order("assigned_at desc, id desc").Offset((page - 1) * size).Limit(size).Find(&targets).Error; err != nil {
		utils.InternalServerError(c.Ctx, "获取发布记录失败")
		return
	}
	enrichIoTFirmwareTargets(targets)
	utils.Success(c.Ctx, map[string]interface{}{"items": targets, "total": total, "page": page, "size": size})
}

func (c *IoTController) UploadFirmware() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	remark := strings.TrimSpace(c.GetString("remark"))
	force := strings.EqualFold(strings.TrimSpace(c.GetString("force")), "true") || strings.TrimSpace(c.GetString("force")) == "1"
	file, header, err := c.GetFile("file")
	if err != nil {
		utils.BadRequestKey(c.Ctx, "iotFirmware.error.missingFile", nil)
		return
	}
	defer file.Close()
	size := uploadFileSize(header)
	if size > iotMaxFirmwarePackageBytes {
		utils.BadRequestKey(c.Ctx, "iotFirmware.error.invalidSize", nil)
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, iotMaxFirmwarePackageBytes+1))
	if err != nil {
		utils.InternalServerError(c.Ctx, "读取固件文件失败")
		return
	}
	if len(data) == 0 || int64(len(data)) > iotMaxFirmwarePackageBytes {
		utils.BadRequestKey(c.Ctx, "iotFirmware.error.invalidSize", nil)
		return
	}
	packageInfo, err := parseIoTFirmwarePackage(header.Filename, data)
	if err != nil {
		utils.BadRequestKey(c.Ctx, "iotFirmware.error.invalidPackage", map[string]interface{}{"reason": err.Error()})
		return
	}
	data = packageInfo.Data
	shaHex := packageInfo.SHA256
	if provided := strings.ToLower(strings.TrimSpace(c.GetString("sha256"))); provided != "" && provided != shaHex {
		utils.BadRequestKey(c.Ctx, "iotFirmware.error.shaMismatch", nil)
		return
	}
	node, err := services.ChooseUploadNodeRealtime()
	if err != nil {
		utils.Error(c.Ctx, http.StatusServiceUnavailable, "暂无可用文件服务节点")
		return
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", packageInfo.FileName)
	if err != nil {
		utils.InternalServerError(c.Ctx, "准备固件上传失败")
		return
	}
	if _, err := part.Write(data); err != nil {
		utils.InternalServerError(c.Ctx, "准备固件上传失败")
		return
	}
	if err := writer.Close(); err != nil {
		utils.InternalServerError(c.Ctx, "准备固件上传失败")
		return
	}
	status, payload, err := services.ProxyUpload(*node, 0, auth.Username, bytes.NewReader(body.Bytes()), writer.FormDataContentType(), int64(body.Len()))
	if err != nil || status < 200 || status >= 300 {
		utils.Error(c.Ctx, http.StatusBadGateway, "固件上传文件服务失败")
		return
	}
	var envelope struct {
		Data struct {
			FileID        string `json:"file_id"`
			OriginalName  string `json:"original_name"`
			Size          int64  `json:"size"`
			StorageNodeID string `json:"storage_node_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.Data.FileID == "" {
		utils.Error(c.Ctx, http.StatusBadGateway, "固件上传响应无效")
		return
	}
	release := models.IoTFirmwareRelease{
		Version:        packageInfo.Version,
		Model:          packageInfo.Model,
		FileID:         envelope.Data.FileID,
		StorageNodeID:  iotFirstNonEmpty(envelope.Data.StorageNodeID, node.ID),
		FileName:       iotFirstNonEmpty(envelope.Data.OriginalName, packageInfo.FileName),
		FileSize:       envelope.Data.Size,
		SHA256:         shaHex,
		Signature:      packageInfo.Signature,
		Status:         "draft",
		Force:          force,
		RolloutPercent: 100,
		CreatedBy:      auth.UserID,
		Remark:         truncateIoTText(remark, 512),
	}
	if release.FileSize <= 0 {
		release.FileSize = int64(len(data))
	}
	if err := utils.DB.Create(&release).Error; err != nil {
		_, _, _ = deleteRemoteFileByID(authContext{TenantID: 0}, release.FileID, release.StorageNodeID)
		utils.InternalServerError(c.Ctx, "保存固件发布记录失败")
		return
	}
	utils.Success(c.Ctx, release)
}

func (c *IoTController) PublishFirmware() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	id, ok := pathUint(c.Ctx, ":id")
	if !ok {
		utils.BadRequest(c.Ctx, "固件ID无效")
		return
	}
	var req iotFirmwarePublishRequest
	if !bindIoTFirmwarePublishRequest(c, &req) {
		return
	}
	var release models.IoTFirmwareRelease
	if err := utils.DB.Where("id = ?", id).First(&release).Error; err != nil {
		utils.NotFound(c.Ctx, "固件不存在")
		return
	}
	now := time.Now()
	release.Status = "published"
	release.Force = req.Force
	release.PublishedAt = &now
	release.RevokedAt = nil
	updated, err := applyIoTFirmwareTargets(release, req)
	if err != nil {
		if firmwareErr, ok := err.(iotFirmwarePublishError); ok {
			utils.BadRequestKey(c.Ctx, firmwareErr.Key, nil)
		} else {
			utils.BadRequest(c.Ctx, err.Error())
		}
		return
	}
	logControllerOperation(c.Ctx, auth, 0, "PUBLISH_IOT_FIRMWARE", "iot.firmware["+release.Version+"]", "operation.publish_iot_firmware", map[string]interface{}{"release_id": release.ID, "version": release.Version, "updated_devices": updated}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"updated_devices": updated})
}

func (c *IoTController) RevokeFirmware() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	id, ok := pathUint(c.Ctx, ":id")
	if !ok {
		utils.BadRequest(c.Ctx, "固件ID无效")
		return
	}
	now := time.Now()
	var release models.IoTFirmwareRelease
	if err := utils.DB.Where("id = ?", id).First(&release).Error; err != nil {
		utils.NotFound(c.Ctx, "固件不存在")
		return
	}
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&release).Updates(map[string]interface{}{"status": "revoked", "revoked_at": &now}).Error; err != nil {
			return err
		}
		incomplete := []string{"assigned", "available", "downloading"}
		if err := tx.Model(&models.IoTFirmwareTarget{}).
			Where("release_id = ? AND status IN ?", id, incomplete).
			Updates(map[string]interface{}{"status": "release_missing", "failure_reason": "firmware release revoked"}).Error; err != nil {
			return err
		}
		return tx.Model(&models.IoTDevice{}).
			Where("target_firmware_version = ? AND model = ? AND ota_status IN ?", release.Version, normalizeIoTModel(release.Model), incomplete).
			Updates(map[string]interface{}{"target_firmware_version": "", "ota_status": "release_missing", "ota_failure_reason": "firmware release revoked"}).Error
	}); err != nil {
		utils.InternalServerError(c.Ctx, "撤回固件失败")
		return
	}
	logControllerOperation(c.Ctx, auth, 0, "REVOKE_IOT_FIRMWARE", "iot.firmware", "operation.revoke_iot_firmware", map[string]interface{}{"release_id": id}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"id": id})
}

func (c *IoTController) DeleteFirmware() {
	_, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	id, ok := pathUint(c.Ctx, ":id")
	if !ok {
		utils.BadRequest(c.Ctx, "固件ID无效")
		return
	}
	var release models.IoTFirmwareRelease
	if err := utils.DB.Where("id = ?", id).First(&release).Error; err != nil {
		utils.NotFound(c.Ctx, "固件不存在")
		return
	}
	status, _, err := deleteRemoteFileByID(authContext{TenantID: 0}, release.FileID, release.StorageNodeID)
	if err != nil || (status < 200 || status >= 300) && status != http.StatusNotFound {
		utils.Error(c.Ctx, http.StatusBadGateway, "删除固件物理文件失败")
		return
	}
	if err := utils.DB.Unscoped().Delete(&release).Error; err != nil {
		utils.InternalServerError(c.Ctx, "删除固件记录失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"id": id})
}

func (c *IoTController) SalesDashboard() {
	if decision := applyIoTPreAuthRateLimit(c.Ctx.Input.IP()); !decision.Allowed {
		c.logIoTTerminalDebug("sales_dashboard_preauth_rate_limit", http.StatusTooManyRequests, "ip_rate_limited", "", map[string]interface{}{"retry_after_sec": int(decision.RetryAfter.Seconds())})
		writeIoTRateLimit(c, decision)
		return
	}
	device, ok := c.authenticateIoTDevice()
	if !ok {
		return
	}
	if decision := applyIoTRateLimit(device); !decision.Allowed {
		c.logIoTTerminalDebug("sales_dashboard_device_rate_limit", http.StatusTooManyRequests, "device_or_tenant_rate_limited", device.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "retry_after_sec": int(decision.RetryAfter.Seconds())})
		writeIoTRateLimit(c, decision)
		return
	}
	year := queryYear(c.Ctx.Input.Query("year"))
	payload := cachedIoTDashboardPayload(device, year)
	updateIoTDeviceActivity(c.Ctx.Input.IP(), c.Ctx.Request.UserAgent(), device, iotDeviceActivityOptions{
		Sync:            true,
		FirmwareVersion: iotFirmwareVersionFromRequest(c.Ctx.Request),
	})
	etag := fmt.Sprint(payload["etag"])
	if etag != "" && strings.TrimSpace(c.Ctx.Request.Header.Get("If-None-Match")) == etag {
		c.logIoTTerminalDebug("sales_dashboard_not_modified", http.StatusNotModified, "etag_matched_no_body_returned", device.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "year": year, "etag": etag, "body_returned": false, "metrics_returned": false})
		c.Ctx.Output.SetStatus(http.StatusNotModified)
		return
	}
	c.Ctx.Output.Header("ETag", etag)
	c.Ctx.Output.SetStatus(http.StatusOK)
	c.Data["json"] = payload
	c.logIoTTerminalDebug("sales_dashboard_success", http.StatusOK, "payload_returned", device.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "year": year, "etag": etag, "body_returned": true, "metrics_returned": true, "metrics": iotMetricsDebugPayload(payload)})
	_ = c.ServeJSON()
}

func (c *IoTController) BootstrapDevice() {
	if decision := applyIoTPreAuthRateLimit(c.Ctx.Input.IP()); !decision.Allowed {
		c.logIoTTerminalDebug("bootstrap_preauth_rate_limit", http.StatusTooManyRequests, "ip_rate_limited", "", map[string]interface{}{"retry_after_sec": int(decision.RetryAfter.Seconds())})
		writeIoTRateLimit(c, decision)
		return
	}
	var req iotBootstrapRequest
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.logIoTTerminalDebug("bootstrap_parse_json", http.StatusBadRequest, "invalid_json", "", map[string]interface{}{"body_bytes": len(c.Ctx.Input.RequestBody), "error": err.Error()})
		utils.Error(c.Ctx, http.StatusBadRequest, "请求参数错误")
		return
	}
	req.SN = normalizeIoTSN(req.SN)
	req.Model = normalizeIoTModel(req.Model)
	req.Timestamp = strings.TrimSpace(req.Timestamp)
	req.Nonce = strings.TrimSpace(req.Nonce)
	req.Signature = strings.TrimSpace(req.Signature)
	req.FirmwareVersion = truncateIoTText(strings.TrimSpace(req.FirmwareVersion), 64)
	if !validIoTBootstrapRequest(req) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("bootstrap_validate_request", http.StatusUnauthorized, "invalid_bootstrap_fields", req.SN, iotBootstrapDebugDetails(req))
		utils.Unauthorized(c.Ctx, "设备注册信息无效")
		return
	}
	if decision := applyIoTFailedRateLimit(c.Ctx.Input.IP(), req.SN); !decision.Allowed {
		c.logIoTTerminalDebug("bootstrap_failed_rate_limit", http.StatusTooManyRequests, "failed_attempt_rate_limited", req.SN, map[string]interface{}{"retry_after_sec": int(decision.RetryAfter.Seconds())})
		writeIoTRateLimit(c, decision)
		return
	}
	if !validIoTTimestamp(req.Timestamp) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("bootstrap_validate_timestamp", http.StatusUnauthorized, "timestamp_out_of_range", req.SN, iotBootstrapDebugDetails(req))
		utils.Unauthorized(c.Ctx, "设备时间戳无效")
		return
	}
	if !consumeIoTNonce(req.SN, "bootstrap:"+req.Nonce) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("bootstrap_consume_nonce", http.StatusForbidden, "nonce_replay", req.SN, iotBootstrapDebugDetails(req))
		utils.Forbidden(c.Ctx, "设备签名已使用")
		return
	}
	publicKey, normalizedPEM, fingerprint, err := parseIoTPublicKey(req.PublicKeyPEM)
	if err != nil {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("bootstrap_parse_public_key", http.StatusUnauthorized, "invalid_public_key", req.SN, map[string]interface{}{"public_key_input_bytes": len(strings.TrimSpace(req.PublicKeyPEM)), "error": err.Error()})
		utils.Unauthorized(c.Ctx, "设备公钥无效")
		return
	}
	if !verifyIoTDeviceSignature(publicKey, iotBootstrapSigningString(req.SN, fingerprint, req.Timestamp, req.Nonce, req.FirmwareVersion), req.Signature) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("bootstrap_verify_device_signature", http.StatusForbidden, "signature_mismatch", req.SN, map[string]interface{}{"public_key_fingerprint": fingerprint, "signature_len": len(req.Signature)})
		utils.Forbidden(c.Ctx, "设备签名错误")
		return
	}
	var device models.IoTDevice
	if err := utils.DB.Where("sn = ?", req.SN).First(&device).Error; err != nil {
		c.logIoTTerminalDebug("bootstrap_lookup_device", http.StatusOK, "device_not_registered_pending_claim", req.SN, map[string]interface{}{"public_key_fingerprint": fingerprint, "db_error": err.Error()})
		utils.Success(c.Ctx, map[string]interface{}{"status": "pending_claim", "retry_after": int64(iotRefreshInterval)})
		return
	}
	if device.Status == "disabled" {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("bootstrap_check_device_status", http.StatusForbidden, "device_disabled", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "device_status": device.Status})
		utils.Forbidden(c.Ctx, "设备已禁用")
		return
	}
	if device.PublicKeyFingerprint != "" && device.PublicKeyFingerprint != fingerprint {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("bootstrap_check_public_key_fingerprint", http.StatusForbidden, "public_key_fingerprint_mismatch", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "device_status": device.Status, "stored_public_key_fingerprint": device.PublicKeyFingerprint, "request_public_key_fingerprint": fingerprint})
		utils.Forbidden(c.Ctx, "设备已绑定其他公钥")
		return
	}
	now := time.Now()
	var deviceSeed string
	var sealedSeed string
	if opened, err := openIoTDeviceSeed(device.SecretHash); err == nil && opened != "" {
		deviceSeed = opened
		sealedSeed = device.SecretHash
	} else {
		deviceSeed, sealedSeed, err = newIoTDeviceSeed()
		if err != nil {
			c.logIoTTerminalDebug("bootstrap_create_seed", http.StatusInternalServerError, "new_seed_failed", req.SN, map[string]interface{}{"error": err.Error()})
			utils.InternalServerError(c.Ctx, "生成设备密钥失败")
			return
		}
	}
	encryptedSeed, err := encryptIoTDeviceSeed(normalizedPEM, deviceSeed)
	if err != nil {
		utils.InternalServerError(c.Ctx, "加密设备密钥失败")
		return
	}
	updates := map[string]interface{}{
		"secret_hash":            sealedSeed,
		"public_key_pem":         normalizedPEM,
		"public_key_fingerprint": fingerprint,
		"model":                  req.Model,
		"status":                 "active",
		"bound_at":               boundAtOrNow(device.BoundAt, now),
		"last_bootstrap_at":      &now,
		"last_seen_at":           &now,
		"last_sync_at":           &now,
		"last_ip":                strings.TrimSpace(c.Ctx.Input.IP()),
		"last_user_agent":        truncateIoTText(strings.TrimSpace(c.Ctx.Request.UserAgent()), iotMaxUserAgentLength),
	}
	firmwareVersion := iotFirstNonEmpty(normalizeIoTFirmwareVersion(req.FirmwareVersion), iotFirmwareVersionFromRequest(c.Ctx.Request))
	if shouldUpdateIoTFirmwareVersion(device.FirmwareVersion, firmwareVersion) {
		updates["firmware_version"] = firmwareVersion
	}
	if err := utils.DB.Model(&device).Updates(updates).Error; err != nil {
		c.logIoTTerminalDebug("bootstrap_save_binding", http.StatusInternalServerError, "db_update_failed", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "public_key_fingerprint": fingerprint, "error": err.Error()})
		utils.InternalServerError(c.Ctx, "绑定设备失败")
		return
	}
	c.logIoTTerminalDebug("bootstrap_success", http.StatusOK, "device_bound", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "public_key_fingerprint": fingerprint, "key_version": nowIoTKeyVersion()})
	c.logIoTTerminalDebug("bootstrap_seed_debug", http.StatusOK, "seed_encrypted_for_device", req.SN, mergeIoTDebugDetails(map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID}, iotDeviceSeedDebugDetails(deviceSeed, encryptedSeed)))
	utils.Success(c.Ctx, map[string]interface{}{
		"status":                 "bound",
		"sn":                     req.SN,
		"tenant_id":              device.TenantID,
		"key_version":            nowIoTKeyVersion(),
		"encrypted_seed":         encryptedSeed,
		"signing_key_derivation": iotKeyDerivationContext + ":key_version",
	})
}

func (c *IoTController) RefreshDeviceSeed() {
	if decision := applyIoTPreAuthRateLimit(c.Ctx.Input.IP()); !decision.Allowed {
		c.logIoTTerminalDebug("seed_refresh_preauth_rate_limit", http.StatusTooManyRequests, "ip_rate_limited", "", map[string]interface{}{"retry_after_sec": int(decision.RetryAfter.Seconds())})
		writeIoTRateLimit(c, decision)
		return
	}
	var req iotSeedRefreshRequest
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.logIoTTerminalDebug("seed_refresh_parse_json", http.StatusBadRequest, "invalid_json", "", map[string]interface{}{"body_bytes": len(c.Ctx.Input.RequestBody), "error": err.Error()})
		utils.Error(c.Ctx, http.StatusBadRequest, "请求参数错误")
		return
	}
	req.SN = normalizeIoTSN(req.SN)
	req.KeyVersion = strings.TrimSpace(req.KeyVersion)
	req.Timestamp = strings.TrimSpace(req.Timestamp)
	req.Nonce = strings.TrimSpace(req.Nonce)
	req.Signature = strings.TrimSpace(req.Signature)
	req.DeviceSignature = strings.TrimSpace(req.DeviceSignature)
	if !validIoTSeedRefreshRequest(req) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("seed_refresh_validate_request", http.StatusUnauthorized, "invalid_seed_refresh_fields", req.SN, iotSeedRefreshDebugDetails(req))
		utils.Unauthorized(c.Ctx, "设备密钥刷新信息无效")
		return
	}
	if !validIoTKeyVersion(req.KeyVersion, nowIoTKeyVersion(), previousIoTKeyVersion()) || !validIoTTimestamp(req.Timestamp) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("seed_refresh_validate_signature_info", http.StatusUnauthorized, "invalid_key_version_or_timestamp", req.SN, iotSeedRefreshDebugDetails(req))
		utils.Unauthorized(c.Ctx, "设备签名信息无效")
		return
	}
	if decision := applyIoTFailedRateLimit(c.Ctx.Input.IP(), req.SN); !decision.Allowed {
		c.logIoTTerminalDebug("seed_refresh_failed_rate_limit", http.StatusTooManyRequests, "failed_attempt_rate_limited", req.SN, map[string]interface{}{"retry_after_sec": int(decision.RetryAfter.Seconds())})
		writeIoTRateLimit(c, decision)
		return
	}
	if !consumeIoTNonce(req.SN, "seed-refresh:"+req.Nonce) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("seed_refresh_consume_nonce", http.StatusForbidden, "nonce_replay", req.SN, iotSeedRefreshDebugDetails(req))
		utils.Forbidden(c.Ctx, "设备签名已使用")
		return
	}
	var device models.IoTDevice
	if err := utils.DB.Where("sn = ?", req.SN).First(&device).Error; err != nil {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("seed_refresh_lookup_device", http.StatusForbidden, "device_not_registered", req.SN, map[string]interface{}{"db_error": err.Error()})
		utils.Forbidden(c.Ctx, "设备SN未登记")
		return
	}
	if device.Status != "active" || device.PublicKeyPEM == "" {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("seed_refresh_check_device_status", http.StatusForbidden, "device_not_bound_or_disabled", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "device_status": device.Status, "has_public_key": device.PublicKeyPEM != ""})
		utils.Forbidden(c.Ctx, "设备未绑定或已禁用")
		return
	}
	publicKey, _, _, err := parseIoTPublicKey(device.PublicKeyPEM)
	if err != nil {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("seed_refresh_parse_public_key", http.StatusForbidden, "stored_public_key_invalid", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "error": err.Error()})
		utils.Forbidden(c.Ctx, "设备公钥无效")
		return
	}
	activeSeed, err := openIoTDeviceSeed(device.SecretHash)
	if err != nil || activeSeed == "" {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("seed_refresh_open_active_seed", http.StatusForbidden, "active_seed_invalid", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "error": fmt.Sprint(err)})
		utils.Forbidden(c.Ctx, "设备密钥无效")
		return
	}
	refreshBase := iotRefreshSigningString(req.KeyVersion, req.Timestamp, req.Nonce)
	if req.Confirm {
		pendingSeed, err := openIoTDeviceSeed(device.PendingSecretHash)
		if err != nil || pendingSeed == "" {
			recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
			c.logIoTTerminalDebug("seed_refresh_confirm_open_pending_seed", http.StatusForbidden, "pending_seed_missing", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "error": fmt.Sprint(err)})
			utils.Forbidden(c.Ctx, "没有待确认密钥")
			return
		}
		if !checkIoTSignature(pendingSeed, req.KeyVersion, http.MethodPost, "/api/iot/v1/device-seed/refresh", req.Timestamp, req.Nonce, "", req.Signature) ||
			!verifyIoTDeviceSignature(publicKey, refreshBase, req.DeviceSignature) {
			recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
			c.logIoTTerminalDebug("seed_refresh_confirm_verify_signature", http.StatusForbidden, "signature_mismatch", req.SN, mergeIoTDebugDetails(mergeIoTDebugDetails(iotSeedRefreshDebugDetails(req), iotDeviceSeedDebugDetails(pendingSeed, nil)), iotSignatureDebugDetails(pendingSeed, req.KeyVersion, http.MethodPost, "/api/iot/v1/device-seed/refresh", req.Timestamp, req.Nonce, "", req.Signature)))
			utils.Forbidden(c.Ctx, "设备签名错误")
			return
		}
		updates := iotDeviceActivityUpdates(time.Now(), c.Ctx.Input.IP(), c.Ctx.Request.UserAgent(), device, iotDeviceActivityOptions{
			Force:           true,
			FirmwareVersion: iotFirmwareVersionFromRequest(c.Ctx.Request),
		})
		updates["secret_hash"] = device.PendingSecretHash
		updates["pending_secret_hash"] = ""
		if err := utils.DB.Model(&device).Updates(updates).Error; err != nil {
			c.logIoTTerminalDebug("seed_refresh_confirm_save", http.StatusInternalServerError, "db_update_failed", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "error": err.Error()})
			utils.InternalServerError(c.Ctx, "确认设备密钥失败")
			return
		}
		c.logIoTTerminalDebug("seed_refresh_confirm_success", http.StatusOK, "confirmed", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "key_version": nowIoTKeyVersion()})
		utils.Success(c.Ctx, map[string]interface{}{"status": "confirmed", "key_version": nowIoTKeyVersion()})
		return
	}
	if !checkIoTSignature(activeSeed, req.KeyVersion, http.MethodPost, "/api/iot/v1/device-seed/refresh", req.Timestamp, req.Nonce, "", req.Signature) ||
		!verifyIoTDeviceSignature(publicKey, refreshBase, req.DeviceSignature) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), req.SN)
		c.logIoTTerminalDebug("seed_refresh_verify_signature", http.StatusForbidden, "signature_mismatch", req.SN, mergeIoTDebugDetails(mergeIoTDebugDetails(iotSeedRefreshDebugDetails(req), iotDeviceSeedDebugDetails(activeSeed, nil)), iotSignatureDebugDetails(activeSeed, req.KeyVersion, http.MethodPost, "/api/iot/v1/device-seed/refresh", req.Timestamp, req.Nonce, "", req.Signature)))
		utils.Forbidden(c.Ctx, "设备签名错误")
		return
	}
	pendingSeed := ""
	pendingSealed := device.PendingSecretHash
	if opened, err := openIoTDeviceSeed(device.PendingSecretHash); err == nil && opened != "" {
		pendingSeed = opened
	} else {
		pendingSeed, pendingSealed, err = newIoTDeviceSeed()
		if err != nil {
			c.logIoTTerminalDebug("seed_refresh_create_pending_seed", http.StatusInternalServerError, "new_seed_failed", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "error": err.Error()})
			utils.InternalServerError(c.Ctx, "生成设备密钥失败")
			return
		}
		if err := utils.DB.Model(&device).Update("pending_secret_hash", pendingSealed).Error; err != nil {
			c.logIoTTerminalDebug("seed_refresh_save_pending_seed", http.StatusInternalServerError, "db_update_failed", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "error": err.Error()})
			utils.InternalServerError(c.Ctx, "保存待轮换密钥失败")
			return
		}
	}
	encryptedSeed, err := encryptIoTDeviceSeed(device.PublicKeyPEM, pendingSeed)
	if err != nil {
		c.logIoTTerminalDebug("seed_refresh_encrypt_seed", http.StatusInternalServerError, "encrypt_seed_failed", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "error": err.Error()})
		utils.InternalServerError(c.Ctx, "加密设备密钥失败")
		return
	}
	c.logIoTTerminalDebug("seed_refresh_success", http.StatusOK, "pending_confirm", req.SN, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "key_version": nowIoTKeyVersion()})
	c.logIoTTerminalDebug("seed_refresh_seed_debug", http.StatusOK, "pending_seed_encrypted_for_device", req.SN, mergeIoTDebugDetails(map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID}, iotDeviceSeedDebugDetails(pendingSeed, encryptedSeed)))
	updateIoTDeviceActivity(c.Ctx.Input.IP(), c.Ctx.Request.UserAgent(), device, iotDeviceActivityOptions{
		Force:           true,
		FirmwareVersion: iotFirmwareVersionFromRequest(c.Ctx.Request),
	})
	utils.Success(c.Ctx, map[string]interface{}{
		"status":         "pending_confirm",
		"key_version":    nowIoTKeyVersion(),
		"encrypted_seed": encryptedSeed,
	})
}

func (c *IoTController) OTACheck() {
	if decision := applyIoTPreAuthRateLimit(c.Ctx.Input.IP()); !decision.Allowed {
		writeIoTRateLimit(c, decision)
		return
	}
	device, ok := c.authenticateIoTDevice()
	if !ok {
		return
	}
	now := time.Now()
	reportedFirmwareVersion := iotFirmwareVersionFromRequest(c.Ctx.Request)
	effectiveFirmwareVersion := effectiveIoTFirmwareVersion(device.FirmwareVersion, reportedFirmwareVersion)
	updates := iotDeviceActivityUpdates(now, c.Ctx.Input.IP(), c.Ctx.Request.UserAgent(), device, iotDeviceActivityOptions{
		Force:           true,
		FirmwareVersion: reportedFirmwareVersion,
	})
	updates["last_ota_check_at"] = &now
	version := strings.TrimSpace(device.TargetFirmwareVersion)
	if version == "" || version == strings.TrimSpace(effectiveFirmwareVersion) {
		updates["ota_status"] = "latest"
		updates["ota_failure_reason"] = ""
		_ = utils.DB.Model(&models.IoTDevice{}).Where("id = ?", device.ID).Updates(updates).Error
		if version != "" {
			_ = updateIoTFirmwareTargetStatus(device, 0, "latest", map[string]interface{}{
				"checked_at":     &now,
				"last_seen_at":   &now,
				"failure_reason": "",
			})
		}
		utils.Success(c.Ctx, map[string]interface{}{"status": "already_latest", "current_version": effectiveFirmwareVersion})
		return
	}
	var release models.IoTFirmwareRelease
	model := normalizeIoTModel(device.Model)
	if err := utils.DB.Where("model = ? AND version = ? AND status = ?", model, version, "published").Order("id desc").First(&release).Error; err != nil {
		updates["ota_status"] = "release_missing"
		updates["ota_failure_reason"] = "target firmware release is not published"
		_ = utils.DB.Model(&models.IoTDevice{}).Where("id = ?", device.ID).Updates(updates).Error
		_ = updateIoTFirmwareTargetStatus(device, 0, "release_missing", map[string]interface{}{
			"checked_at":     &now,
			"last_seen_at":   &now,
			"failure_reason": "target firmware release is not published",
		})
		utils.Success(c.Ctx, map[string]interface{}{"status": "already_latest", "current_version": effectiveFirmwareVersion})
		return
	}
	token := makeIoTOTADownloadToken(device.SN, release.ID, release.Version, now.Add(iotOTADownloadTokenTTL))
	updates["ota_status"] = "available"
	updates["ota_failure_reason"] = ""
	_ = utils.DB.Model(&models.IoTDevice{}).Where("id = ?", device.ID).Updates(updates).Error
	_ = updateIoTFirmwareTargetStatus(device, release.ID, "available", map[string]interface{}{
		"checked_at":     &now,
		"last_seen_at":   &now,
		"failure_reason": "",
	})
	utils.Success(c.Ctx, map[string]interface{}{
		"status":       "update_available",
		"version":      release.Version,
		"size":         release.FileSize,
		"sha256":       release.SHA256,
		"signature":    release.Signature,
		"download_url": "/api/iot/v1/ota/download/" + token,
		"force":        release.Force,
	})
}

func (c *IoTController) OTADownload() {
	token := strings.TrimSpace(c.Ctx.Input.Param(":token"))
	claims, ok := parseIoTOTADownloadToken(token)
	if !ok {
		c.recordOTADownloadFailure(models.IoTDevice{}, 0, "download_token_invalid", "下载链接无效或已过期")
		utils.Forbidden(c.Ctx, "下载链接无效或已过期")
		return
	}
	if utils.RedisClient != nil {
		key := "iot:ota:download:" + claims.Nonce
		if used, err := utils.RedisClient.Exists(context.Background(), key).Result(); err != nil || used > 0 {
			c.recordOTADownloadFailure(models.IoTDevice{SN: claims.SN, TargetFirmwareVersion: claims.Version}, claims.ReleaseID, "download_token_reused", "下载链接已使用")
			utils.Forbidden(c.Ctx, "下载链接已使用")
			return
		}
	}
	var device models.IoTDevice
	if err := utils.DB.Where("sn = ? AND target_firmware_version = ? AND status = ?", claims.SN, claims.Version, "active").First(&device).Error; err != nil {
		c.recordOTADownloadFailure(models.IoTDevice{SN: claims.SN, TargetFirmwareVersion: claims.Version}, claims.ReleaseID, "download_task_missing", "设备没有该升级任务")
		utils.Forbidden(c.Ctx, "设备没有该升级任务")
		return
	}
	var release models.IoTFirmwareRelease
	if err := utils.DB.Where("id = ? AND version = ? AND status = ?", claims.ReleaseID, claims.Version, "published").First(&release).Error; err != nil {
		c.recordOTADownloadFailure(device, claims.ReleaseID, "download_release_missing", "固件不存在")
		utils.NotFound(c.Ctx, "固件不存在")
		return
	}
	path := fmt.Sprintf("/internal/v1/files/%s/download", release.FileID)
	for _, node := range orderedFileserverNodes(release.StorageNodeID) {
		if !node.Healthy {
			continue
		}
		resp, err := services.GetInternal(node, path, "", 0, "")
		if err != nil {
			c.logIoTTerminalDebug("ota_download_fileserver", http.StatusBadGateway, "fileserver_request_failed", device.SN, map[string]interface{}{"release_id": release.ID, "node": node.BaseURL, "error": err.Error()})
			continue
		}
		if resp.StatusCode == http.StatusNotFound {
			c.logIoTTerminalDebug("ota_download_fileserver", resp.StatusCode, "firmware_file_not_found", device.SN, map[string]interface{}{"release_id": release.ID, "node": node.BaseURL, "file_id": release.FileID})
			resp.Body.Close()
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			c.logIoTTerminalDebug("ota_download_fileserver", resp.StatusCode, "fileserver_download_failed", device.SN, map[string]interface{}{"release_id": release.ID, "node": node.BaseURL, "file_id": release.FileID})
			resp.Body.Close()
			continue
		}
		if utils.RedisClient != nil {
			key := "iot:ota:download:" + claims.Nonce
			used, err := utils.RedisClient.SetNX(context.Background(), key, "1", iotOTADownloadTokenTTL).Result()
			if err != nil || !used {
				resp.Body.Close()
				c.recordOTADownloadFailure(device, claims.ReleaseID, "download_token_reused", "下载链接已使用")
				utils.Forbidden(c.Ctx, "下载链接已使用")
				return
			}
		}
		now := time.Now()
		_ = utils.DB.Model(&models.IoTDevice{}).Where("id = ?", device.ID).Updates(map[string]interface{}{
			"ota_status":           "downloading",
			"last_ota_download_at": &now,
		}).Error
		_ = updateIoTFirmwareTargetStatus(device, release.ID, "downloading", map[string]interface{}{
			"downloaded_at":  &now,
			"last_seen_at":   &now,
			"failure_reason": "",
		})
		defer resp.Body.Close()
		c.Ctx.Output.Header("Content-Type", "application/octet-stream")
		c.Ctx.Output.Header("Content-Length", strconv.FormatInt(release.FileSize, 10))
		c.Ctx.Output.Header("X-IOT-Firmware-Version", release.Version)
		c.Ctx.Output.Header("X-IOT-Firmware-SHA256", release.SHA256)
		c.Ctx.Output.SetStatus(resp.StatusCode)
		_, _ = io.Copy(c.Ctx.ResponseWriter, resp.Body)
		return
	}
	c.recordOTADownloadFailure(device, release.ID, "download_file_unavailable", "固件文件不可用")
	utils.Error(c.Ctx, http.StatusBadGateway, "固件文件不可用")
}

func (c *IoTController) recordOTADownloadFailure(device models.IoTDevice, releaseID uint, reason string, message string) {
	now := time.Now()
	sn := normalizeIoTSN(device.SN)
	version := strings.TrimSpace(device.TargetFirmwareVersion)
	if version == "" {
		version = strings.TrimSpace(device.FirmwareVersion)
	}
	details := map[string]interface{}{
		"release_id": releaseID,
		"version":    version,
		"message":    message,
	}
	if device.ID > 0 {
		updates := map[string]interface{}{
			"ota_status":         "failed",
			"ota_failure_reason": message,
			"last_seen_at":       &now,
			"last_ip":            strings.TrimSpace(c.Ctx.Input.IP()),
			"last_user_agent":    truncateIoTText(strings.TrimSpace(c.Ctx.Request.UserAgent()), iotMaxUserAgentLength),
		}
		_ = utils.DB.Model(&models.IoTDevice{}).Where("id = ?", device.ID).Updates(updates).Error
		_ = updateIoTFirmwareTargetStatus(device, releaseID, "failed", map[string]interface{}{
			"last_seen_at":   &now,
			"failure_reason": message,
		})
		details["device_id"] = device.ID
		details["tenant_id"] = device.TenantID
	} else {
		if releaseID == 0 && sn == "" && version == "" {
			c.logIoTTerminalDebug("ota_download_failed", http.StatusForbidden, reason, sn, details)
			return
		}
		query := utils.DB.Model(&models.IoTFirmwareTarget{}).Where("1 = 1")
		if releaseID > 0 {
			query = query.Where("release_id = ?", releaseID)
		}
		if sn != "" {
			query = query.Where("sn = ?", sn)
		}
		if version != "" {
			query = query.Where("version = ?", version)
		}
		result := query.Order("assigned_at desc, id desc").Limit(1).Updates(map[string]interface{}{
			"status":         "failed",
			"failure_reason": message,
			"last_seen_at":   &now,
		})
		details["target_rows"] = result.RowsAffected
	}
	c.logIoTTerminalDebug("ota_download_failed", http.StatusForbidden, reason, sn, details)
}

func (c *IoTController) OTAConfirm() {
	if decision := applyIoTPreAuthRateLimit(c.Ctx.Input.IP()); !decision.Allowed {
		writeIoTRateLimit(c, decision)
		return
	}
	device, ok := c.authenticateIoTDevice()
	if !ok {
		return
	}
	var req iotOTAConfirmRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	version := truncateIoTText(strings.TrimSpace(req.Version), 64)
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if version == "" || (status != "success" && status != "failed") {
		utils.BadRequest(c.Ctx, "OTA确认参数无效")
		return
	}
	now := time.Now()
	updates := iotDeviceActivityUpdates(now, c.Ctx.Input.IP(), c.Ctx.Request.UserAgent(), device, iotDeviceActivityOptions{
		Force:           true,
		FirmwareVersion: iotFirmwareVersionFromRequest(c.Ctx.Request),
	})
	updates["last_ota_confirm_at"] = &now
	if status == "success" {
		updates["firmware_version"] = version
		updates["target_firmware_version"] = ""
		updates["ota_status"] = "success"
		updates["ota_failure_reason"] = ""
	} else {
		updates["ota_status"] = "failed"
		updates["ota_failure_reason"] = truncateIoTText(strings.TrimSpace(req.Reason), 512)
	}
	if err := utils.DB.Model(&models.IoTDevice{}).Where("id = ?", device.ID).Updates(updates).Error; err != nil {
		utils.InternalServerError(c.Ctx, "保存OTA确认状态失败")
		return
	}
	targetStatus := "success"
	failureReason := ""
	if status == "failed" {
		targetStatus = "failed"
		failureReason = truncateIoTText(strings.TrimSpace(req.Reason), 512)
	}
	_ = updateIoTFirmwareTargetStatus(device, 0, targetStatus, map[string]interface{}{
		"confirmed_at":   &now,
		"last_seen_at":   &now,
		"failure_reason": failureReason,
	})
	utils.Success(c.Ctx, map[string]interface{}{"status": "confirmed"})
}

func (c *IoTController) authenticateIoTDevice() (models.IoTDevice, bool) {
	if c.Ctx.Request.Method == http.MethodGet && strings.HasPrefix(c.Ctx.Request.URL.Path, "/api/iot/v1/ota/download/") {
		return models.IoTDevice{}, true
	}
	allowedDeviceAPI := map[string]string{
		http.MethodGet + " " + "/api/iot/v1/sales-dashboard": "",
		http.MethodGet + " " + "/api/iot/v1/ota/check":       "",
		http.MethodPost + " " + "/api/iot/v1/ota/confirm":    "",
	}
	if _, allowed := allowedDeviceAPI[c.Ctx.Request.Method+" "+c.Ctx.Request.URL.Path]; !allowed {
		recordIoTAuthFailure(c.Ctx.Input.IP(), "")
		c.logIoTTerminalDebug("device_auth_route", http.StatusForbidden, "invalid_device_api", "", nil)
		utils.Forbidden(c.Ctx, "设备接口无效")
		return models.IoTDevice{}, false
	}
	sn := normalizeIoTSN(c.Ctx.Request.Header.Get("X-IOT-Device-SN"))
	keyVersion := strings.TrimSpace(c.Ctx.Request.Header.Get("X-IOT-Key-Version"))
	timestamp := strings.TrimSpace(c.Ctx.Request.Header.Get("X-IOT-Timestamp"))
	nonce := strings.TrimSpace(c.Ctx.Request.Header.Get("X-IOT-Nonce"))
	signature := strings.TrimSpace(c.Ctx.Request.Header.Get("X-IOT-Signature"))
	if !validIoTAuthHeaders(sn, keyVersion, timestamp, nonce, signature) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), sn)
		c.logIoTTerminalDebug("device_auth_validate_headers", http.StatusUnauthorized, "invalid_auth_headers", sn, iotAuthHeaderDebugDetails(sn, keyVersion, timestamp, nonce, signature))
		utils.Unauthorized(c.Ctx, "设备签名信息无效")
		return models.IoTDevice{}, false
	}
	if !validIoTKeyVersion(keyVersion, nowIoTKeyVersion(), previousIoTKeyVersion()) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), sn)
		c.logIoTTerminalDebug("device_auth_validate_key_version", http.StatusUnauthorized, "invalid_key_version", sn, iotAuthHeaderDebugDetails(sn, keyVersion, timestamp, nonce, signature))
		utils.Unauthorized(c.Ctx, "设备密钥版本无效")
		return models.IoTDevice{}, false
	}
	if decision := applyIoTFailedRateLimit(c.Ctx.Input.IP(), sn); !decision.Allowed {
		c.logIoTTerminalDebug("device_auth_failed_rate_limit", http.StatusTooManyRequests, "failed_attempt_rate_limited", sn, map[string]interface{}{"retry_after_sec": int(decision.RetryAfter.Seconds())})
		writeIoTRateLimit(c, decision)
		return models.IoTDevice{}, false
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		recordIoTAuthFailure(c.Ctx.Input.IP(), sn)
		c.logIoTTerminalDebug("device_auth_parse_timestamp", http.StatusUnauthorized, "timestamp_parse_error", sn, iotAuthHeaderDebugDetails(sn, keyVersion, timestamp, nonce, signature))
		utils.Unauthorized(c.Ctx, "设备时间戳无效")
		return models.IoTDevice{}, false
	}
	now := time.Now()
	if math.Abs(float64(now.Unix()-ts)) > iotTimestampSkew.Seconds() {
		recordIoTAuthFailure(c.Ctx.Input.IP(), sn)
		c.logIoTTerminalDebug("device_auth_timestamp_skew", http.StatusUnauthorized, "timestamp_expired", sn, iotAuthHeaderDebugDetails(sn, keyVersion, timestamp, nonce, signature))
		utils.Unauthorized(c.Ctx, "设备签名已过期")
		return models.IoTDevice{}, false
	}
	var device models.IoTDevice
	if err := utils.DB.Where("sn = ?", sn).First(&device).Error; err != nil {
		recordIoTAuthFailure(c.Ctx.Input.IP(), sn)
		c.logIoTTerminalDebug("device_auth_lookup_device", http.StatusForbidden, "device_not_registered", sn, map[string]interface{}{"db_error": err.Error()})
		utils.Forbidden(c.Ctx, "设备SN未登记")
		return models.IoTDevice{}, false
	}
	if device.Status != "active" {
		recordIoTAuthFailure(c.Ctx.Input.IP(), sn)
		c.logIoTTerminalDebug("device_auth_check_status", http.StatusForbidden, "device_not_active", sn, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "device_status": device.Status})
		utils.Forbidden(c.Ctx, "设备未绑定或已禁用")
		return models.IoTDevice{}, false
	}
	if reason, blocked := tenantBlockedReason(device.TenantID, now); blocked {
		recordIoTAuthFailure(c.Ctx.Input.IP(), sn)
		c.logIoTTerminalDebug("device_auth_tenant_blocked", http.StatusForbidden, "tenant_blocked", sn, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "reason": reason})
		utils.Forbidden(c.Ctx, reason)
		return models.IoTDevice{}, false
	}
	deviceSeed, err := openIoTDeviceSeed(device.SecretHash)
	if err != nil || deviceSeed == "" {
		recordIoTAuthFailure(c.Ctx.Input.IP(), sn)
		c.logIoTTerminalDebug("device_auth_open_seed", http.StatusForbidden, "device_seed_invalid", sn, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "error": fmt.Sprint(err)})
		utils.Forbidden(c.Ctx, "设备密钥无效")
		return models.IoTDevice{}, false
	}
	bodySHA := ""
	if len(c.Ctx.Input.RequestBody) > 0 {
		sum := sha256.Sum256(c.Ctx.Input.RequestBody)
		bodySHA = hex.EncodeToString(sum[:])
	}
	if !checkIoTSignature(deviceSeed, keyVersion, c.Ctx.Request.Method, c.Ctx.Request.URL.Path, timestamp, nonce, bodySHA, signature) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), sn)
		c.logIoTTerminalDebug("device_auth_verify_signature", http.StatusForbidden, "signature_mismatch", sn, mergeIoTDebugDetails(mergeIoTDebugDetails(iotAuthHeaderDebugDetails(sn, keyVersion, timestamp, nonce, signature), iotDeviceSeedDebugDetails(deviceSeed, nil)), iotSignatureDebugDetails(deviceSeed, keyVersion, c.Ctx.Request.Method, c.Ctx.Request.URL.Path, timestamp, nonce, bodySHA, signature)))
		utils.Forbidden(c.Ctx, "设备签名错误")
		return models.IoTDevice{}, false
	}
	if !consumeIoTNonce(sn, nonce) {
		recordIoTAuthFailure(c.Ctx.Input.IP(), sn)
		c.logIoTTerminalDebug("device_auth_consume_nonce", http.StatusForbidden, "nonce_replay", sn, iotAuthHeaderDebugDetails(sn, keyVersion, timestamp, nonce, signature))
		utils.Forbidden(c.Ctx, "设备签名已使用")
		return models.IoTDevice{}, false
	}
	c.logIoTTerminalDebug("device_auth_success", http.StatusOK, "authenticated", sn, map[string]interface{}{"device_id": device.ID, "tenant_id": device.TenantID, "key_version": keyVersion})
	return device, true
}

type iotRateDecision struct {
	Allowed    bool
	RetryAfter time.Duration
}

func writeIoTRateLimit(c *IoTController, decision iotRateDecision) {
	if decision.RetryAfter > 0 {
		c.Ctx.Output.Header("Retry-After", strconv.FormatInt(int64(decision.RetryAfter.Seconds()), 10))
	}
	utils.ErrorWithData(c.Ctx, http.StatusTooManyRequests, "请求过于频繁", map[string]interface{}{"retry_after": int64(decision.RetryAfter.Seconds())})
}

func (c *IoTController) logIoTTerminalDebug(stage string, status int, reason string, sn string, extra map[string]interface{}) {
	if !iotTerminalDebugLogEnabled() {
		return
	}
	details := map[string]interface{}{
		"stage":      stage,
		"status":     status,
		"reason":     reason,
		"sn":         sn,
		"ip":         c.Ctx.Input.IP(),
		"method":     c.Ctx.Request.Method,
		"path":       c.Ctx.Request.URL.Path,
		"query":      c.Ctx.Request.URL.RawQuery,
		"ua":         truncateIoTText(c.Ctx.Request.UserAgent(), 180),
		"body_bytes": len(c.Ctx.Input.RequestBody),
	}
	for key, value := range extra {
		details[key] = value
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		utils.LogWarning("IoT terminal debug marshal_failed stage=%s status=%d reason=%s sn=%s ip=%s err=%v", stage, status, reason, sn, c.Ctx.Input.IP(), err)
		return
	}
	if status >= http.StatusBadRequest {
		utils.LogWarning("IoT terminal debug %s", string(encoded))
		return
	}
	utils.LogInfo("IoT terminal debug %s", string(encoded))
}

func iotTerminalDebugLogEnabled() bool {
	if value := strings.TrimSpace(os.Getenv("IOT_TERMINAL_DEBUG_LOG_ENABLED")); value != "" {
		enabled, err := strconv.ParseBool(value)
		return err == nil && enabled
	}
	return web.AppConfig.DefaultBool("iot_terminal_debug_log_enabled", false)
}

func iotBootstrapDebugDetails(req iotBootstrapRequest) map[string]interface{} {
	return map[string]interface{}{
		"sn_present":             req.SN != "",
		"sn_len":                 len(req.SN),
		"sn_chars_valid":         validIoTTextChars(req.SN, true),
		"public_key_present":     strings.TrimSpace(req.PublicKeyPEM) != "",
		"public_key_input_bytes": len(strings.TrimSpace(req.PublicKeyPEM)),
		"timestamp_present":      req.Timestamp != "",
		"timestamp_len":          len(req.Timestamp),
		"timestamp_digits":       validIoTDigits(req.Timestamp),
		"timestamp_skew_sec":     iotTimestampSkewSeconds(req.Timestamp),
		"nonce_present":          req.Nonce != "",
		"nonce_len":              len(req.Nonce),
		"nonce_chars_valid":      validIoTTextChars(req.Nonce, false),
		"signature_present":      req.Signature != "",
		"signature_len":          len(req.Signature),
		"signature_hex":          validIoTHex(req.Signature),
		"expected_signature_len": iotDeviceSignatureHexLength,
		"firmware_version":       req.FirmwareVersion,
	}
}

func iotSeedRefreshDebugDetails(req iotSeedRefreshRequest) map[string]interface{} {
	return map[string]interface{}{
		"sn_present":              req.SN != "",
		"sn_len":                  len(req.SN),
		"sn_chars_valid":          validIoTTextChars(req.SN, true),
		"key_version":             req.KeyVersion,
		"key_version_valid":       validIoTKeyVersion(req.KeyVersion, nowIoTKeyVersion(), previousIoTKeyVersion()),
		"timestamp_present":       req.Timestamp != "",
		"timestamp_len":           len(req.Timestamp),
		"timestamp_digits":        validIoTDigits(req.Timestamp),
		"timestamp_skew_sec":      iotTimestampSkewSeconds(req.Timestamp),
		"nonce_present":           req.Nonce != "",
		"nonce_len":               len(req.Nonce),
		"nonce_chars_valid":       validIoTTextChars(req.Nonce, false),
		"signature_len":           len(req.Signature),
		"signature_hex":           validIoTHex(req.Signature),
		"expected_signature_len":  iotHMACSignatureHexLength,
		"device_signature_len":    len(req.DeviceSignature),
		"device_signature_hex":    validIoTHex(req.DeviceSignature),
		"expected_device_sig_len": iotDeviceSignatureHexLength,
		"confirm":                 req.Confirm,
	}
}

func iotAuthHeaderDebugDetails(sn string, keyVersion string, timestamp string, nonce string, signature string) map[string]interface{} {
	return map[string]interface{}{
		"sn_present":             sn != "",
		"sn_len":                 len(sn),
		"sn_chars_valid":         validIoTTextChars(sn, true),
		"key_version":            keyVersion,
		"key_version_len":        len(keyVersion),
		"key_version_valid":      validIoTKeyVersion(keyVersion, nowIoTKeyVersion(), previousIoTKeyVersion()),
		"timestamp_present":      timestamp != "",
		"timestamp_len":          len(timestamp),
		"timestamp_digits":       validIoTDigits(timestamp),
		"timestamp_skew_sec":     iotTimestampSkewSeconds(timestamp),
		"nonce_present":          nonce != "",
		"nonce_len":              len(nonce),
		"nonce_chars_valid":      validIoTTextChars(nonce, false),
		"signature_present":      signature != "",
		"signature_len":          len(signature),
		"signature_hex":          validIoTHex(signature),
		"expected_signature_len": iotHMACSignatureHexLength,
	}
}

func mergeIoTDebugDetails(base map[string]interface{}, extra map[string]interface{}) map[string]interface{} {
	merged := map[string]interface{}{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func iotSignatureDebugDetails(deviceSeed string, keyVersion string, method string, path string, timestamp string, nonce string, bodySHA string, signature string) map[string]interface{} {
	if !iotTerminalDebugLogEnabled() {
		return nil
	}
	if bodySHA == "" {
		empty := sha256.Sum256(nil)
		bodySHA = hex.EncodeToString(empty[:])
	}
	canonicalSigningString := iotSignatureBaseString(method, path, keyVersion, timestamp, nonce, bodySHA)
	signingKey := deriveIoTSigningKey(deviceSeed, keyVersion)
	signingKeySHA := sha256.Sum256([]byte(signingKey))
	rawSigningKey, rawSigningKeyErr := hex.DecodeString(signingKey)
	return map[string]interface{}{
		"body_sha256":                                  bodySHA,
		"canonical_signing_string":                     canonicalSigningString,
		"canonical_expected_signature":                 hmacSHA256Hex(signingKey, canonicalSigningString),
		"canonical_expected_raw_signing_key_signature": hmacSHA256BytesHex(rawSigningKey, canonicalSigningString),
		"received_signature":                           strings.ToLower(strings.TrimSpace(signature)),
		"signing_key_len":                              len(signingKey),
		"signing_key_sha256":                           hex.EncodeToString(signingKeySHA[:]),
		"signing_key_hex_decode_ok":                    rawSigningKeyErr == nil,
		"signing_key_raw_len":                          len(rawSigningKey),
		"alt_expected_raw_seed_signature":              hmacSHA256Hex(deviceSeed, canonicalSigningString),
		"alt_expected_no_key_version_in_string":        hmacSHA256Hex(signingKey, strings.Join([]string{strings.ToUpper(method), path, timestamp, nonce, bodySHA}, "\n")),
		"alt_expected_no_key_version_raw_key":          hmacSHA256BytesHex(rawSigningKey, strings.Join([]string{strings.ToUpper(method), path, timestamp, nonce, bodySHA}, "\n")),
		"alt_expected_key_version_last_string":         hmacSHA256Hex(signingKey, strings.Join([]string{strings.ToUpper(method), path, timestamp, nonce, bodySHA, keyVersion}, "\n")),
		"alt_expected_legacy_path_without_key_version": hmacSHA256Hex(signingKey, strings.Join([]string{strings.ToUpper(method), path, timestamp, nonce, bodySHA}, "\n")),
		"signing_key_derivation":                       iotKeyDerivationContext + ":key_version",
		"signing_string_format":                        "METHOD\\nPATH\\nKEY_VERSION\\nTIMESTAMP\\nNONCE\\nBODY_SHA256",
	}
}

func iotDeviceSeedDebugDetails(deviceSeed string, encryptedSeed map[string]interface{}) map[string]interface{} {
	if !iotTerminalDebugLogEnabled() {
		return nil
	}
	sum := sha256.Sum256([]byte(deviceSeed))
	details := map[string]interface{}{
		"device_seed_len":    len(deviceSeed),
		"device_seed_sha256": hex.EncodeToString(sum[:]),
	}
	if encryptedSeed != nil {
		if ciphertext, ok := encryptedSeed["ciphertext"].(string); ok {
			details["encrypted_seed_ciphertext_len"] = len(ciphertext)
			cipherSum := sha256.Sum256([]byte(ciphertext))
			details["encrypted_seed_ciphertext_sha256"] = hex.EncodeToString(cipherSum[:])
		}
		if nonce, ok := encryptedSeed["nonce"].(string); ok {
			details["encrypted_seed_nonce_len"] = len(nonce)
		}
		if ephemeralPublicKey, ok := encryptedSeed["ephemeral_public_key"].(string); ok {
			details["encrypted_seed_ephemeral_public_key_len"] = len(ephemeralPublicKey)
		}
		if fingerprint, ok := encryptedSeed["device_public_key_fingerprint"].(string); ok {
			details["encrypted_seed_device_public_key_fingerprint"] = fingerprint
		}
		if contextValue, ok := encryptedSeed["context"].(string); ok {
			details["encrypted_seed_context"] = contextValue
		}
	}
	return details
}

func iotMetricsDebugPayload(payload map[string]interface{}) []map[string]interface{} {
	rawMetrics, ok := payload["metrics"].([]map[string]interface{})
	if !ok {
		return nil
	}
	items := make([]map[string]interface{}, 0, len(rawMetrics))
	for _, metric := range rawMetrics {
		items = append(items, map[string]interface{}{
			"key":       metric["key"],
			"label":     metric["label"],
			"value":     metric["value"],
			"raw_value": metric["raw_value"],
			"unit":      metric["unit"],
		})
	}
	return items
}

func iotTimestampSkewSeconds(timestamp string) interface{} {
	ts, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return "parse_error"
	}
	return time.Now().Unix() - ts
}

func applyIoTPreAuthRateLimit(ip string) iotRateDecision {
	if utils.RedisClient == nil {
		return iotRateDecision{Allowed: true}
	}
	key := "iot:preauth:ip:" + normalizeIoTRateKey(ip)
	if ok, retry := incrementIoTRate(key, iotPreAuthIPRateLimit, iotPreAuthIPRateWindow); !ok {
		return iotRateDecision{Allowed: false, RetryAfter: retry}
	}
	return iotRateDecision{Allowed: true}
}

func applyIoTFailedRateLimit(ip string, sn string) iotRateDecision {
	if utils.RedisClient == nil {
		return iotRateDecision{Allowed: true}
	}
	if sn != "" {
		key := "iot:fail:sn:" + normalizeIoTRateKey(sn)
		if ok, retry := incrementIoTRateReadOnly(key, iotFailedSNRateLimit); !ok {
			return iotRateDecision{Allowed: false, RetryAfter: retry}
		}
	}
	key := "iot:fail:ip:" + normalizeIoTRateKey(ip)
	if ok, retry := incrementIoTRateReadOnly(key, iotFailedIPRateLimit); !ok {
		return iotRateDecision{Allowed: false, RetryAfter: retry}
	}
	return iotRateDecision{Allowed: true}
}

func recordIoTAuthFailure(ip string, sn string) {
	if utils.RedisClient == nil {
		return
	}
	_, _ = incrementIoTRate("iot:fail:ip:"+normalizeIoTRateKey(ip), iotFailedIPRateLimit, iotFailedRateWindow)
	if sn != "" {
		_, _ = incrementIoTRate("iot:fail:sn:"+normalizeIoTRateKey(sn), iotFailedSNRateLimit, iotFailedRateWindow)
	}
}

func applyIoTRateLimit(device models.IoTDevice) iotRateDecision {
	if utils.RedisClient == nil {
		return iotRateDecision{Allowed: true}
	}
	if ok, retry := incrementIoTRate("iot:rate:sn:"+device.SN, 1, iotDeviceRateWindow); !ok {
		return iotRateDecision{Allowed: false, RetryAfter: retry}
	}
	if ok, retry := incrementIoTRate("iot:rate:tenant:"+strconv.FormatUint(uint64(device.TenantID), 10), iotTenantRateLimit, iotTenantRateWindow); !ok {
		return iotRateDecision{Allowed: false, RetryAfter: retry}
	}
	return iotRateDecision{Allowed: true}
}

func incrementIoTRateReadOnly(key string, limit int64) (bool, time.Duration) {
	client := utils.RedisClient
	if client == nil {
		return true, 0
	}
	ctx := context.Background()
	count, err := client.Get(ctx, key).Int64()
	if err != nil || count < limit {
		return true, 0
	}
	ttl, err := client.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		ttl = iotFailedRateWindow
	}
	return false, ttl
}

func incrementIoTRate(key string, limit int64, window time.Duration) (bool, time.Duration) {
	client := utils.RedisClient
	if client == nil {
		return true, 0
	}
	ctx := context.Background()
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return true, 0
	}
	if count == 1 {
		_ = client.Expire(ctx, key, window).Err()
	}
	if count <= limit {
		return true, 0
	}
	ttl, err := client.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		ttl = window
	}
	return false, ttl
}

func validIoTAuthHeaders(sn string, keyVersion string, timestamp string, nonce string, signature string) bool {
	if sn == "" || keyVersion == "" || timestamp == "" || nonce == "" || signature == "" {
		return false
	}
	if len(sn) > iotMaxSNLength || len(keyVersion) > iotMaxKeyVersionLength || len(timestamp) > iotMaxTimestampLength || len(nonce) > iotMaxNonceLength || len(signature) != iotHMACSignatureHexLength {
		return false
	}
	for _, ch := range sn {
		if !(ch >= 'A' && ch <= 'Z') && !(ch >= '0' && ch <= '9') && ch != '-' && ch != '_' && ch != ':' && ch != '.' {
			return false
		}
	}
	for _, ch := range keyVersion {
		if !(ch >= '0' && ch <= '9') && ch != '-' {
			return false
		}
	}
	for _, ch := range timestamp {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	for _, ch := range nonce {
		if !(ch >= 'a' && ch <= 'z') && !(ch >= 'A' && ch <= 'Z') && !(ch >= '0' && ch <= '9') && ch != '-' && ch != '_' && ch != ':' && ch != '.' {
			return false
		}
	}
	for _, ch := range signature {
		if !(ch >= 'a' && ch <= 'f') && !(ch >= 'A' && ch <= 'F') && !(ch >= '0' && ch <= '9') {
			return false
		}
	}
	return true
}

func validIoTBootstrapRequest(req iotBootstrapRequest) bool {
	if req.SN == "" || req.PublicKeyPEM == "" || req.Timestamp == "" || req.Nonce == "" || req.Signature == "" {
		return false
	}
	if len(req.SN) > iotMaxSNLength || len(req.Timestamp) > iotMaxTimestampLength || len(req.Nonce) > iotMaxNonceLength || len(req.Signature) != iotDeviceSignatureHexLength {
		return false
	}
	return validIoTTextChars(req.SN, true) && validIoTDigits(req.Timestamp) && validIoTTextChars(req.Nonce, false) && validIoTHex(req.Signature)
}

func validIoTSeedRefreshRequest(req iotSeedRefreshRequest) bool {
	if req.SN == "" || req.KeyVersion == "" || req.Timestamp == "" || req.Nonce == "" || req.Signature == "" || req.DeviceSignature == "" {
		return false
	}
	if len(req.SN) > iotMaxSNLength || len(req.KeyVersion) > iotMaxKeyVersionLength || len(req.Timestamp) > iotMaxTimestampLength || len(req.Nonce) > iotMaxNonceLength || len(req.Signature) != iotHMACSignatureHexLength || len(req.DeviceSignature) != iotDeviceSignatureHexLength {
		return false
	}
	return validIoTTextChars(req.SN, true) && validIoTDigits(req.Timestamp) && validIoTTextChars(req.Nonce, false) && validIoTHex(req.Signature) && validIoTHex(req.DeviceSignature)
}

func validIoTTimestamp(timestamp string) bool {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	return math.Abs(float64(time.Now().Unix()-ts)) <= iotTimestampSkew.Seconds()
}

func validIoTTextChars(value string, upperOnly bool) bool {
	for _, ch := range value {
		if ch >= '0' && ch <= '9' {
			continue
		}
		if upperOnly {
			if ch >= 'A' && ch <= 'Z' {
				continue
			}
		} else if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
			continue
		}
		if ch == '-' || ch == '_' || ch == ':' || ch == '.' {
			continue
		}
		return false
	}
	return true
}

func validIoTDigits(value string) bool {
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func validIoTHex(value string) bool {
	for _, ch := range value {
		if !(ch >= 'a' && ch <= 'f') && !(ch >= 'A' && ch <= 'F') && !(ch >= '0' && ch <= '9') {
			return false
		}
	}
	return true
}

func consumeIoTNonce(sn string, nonce string) bool {
	if utils.RedisClient == nil {
		return true
	}
	key := "iot:nonce:" + sn + ":" + nonce
	ok, err := utils.RedisClient.SetNX(context.Background(), key, "1", iotNonceTTL).Result()
	return err == nil && ok
}

func cachedIoTDashboardPayload(device models.IoTDevice, year int) map[string]interface{} {
	if utils.RedisClient == nil {
		return buildIoTDashboardPayload(device, year)
	}
	key := fmt.Sprintf("iot:sales_dashboard:%d:%d:%s", device.TenantID, year, iotDashboardCacheVersion)
	var cached map[string]interface{}
	if data, err := utils.RedisClient.Get(context.Background(), key).Bytes(); err == nil && json.Unmarshal(data, &cached) == nil {
		cached["sn"] = device.SN
		cached["resolution"] = map[string]interface{}{"width": device.ResolutionWidth, "height": device.ResolutionHeight}
		return cached
	}
	payload := buildIoTDashboardPayload(device, year)
	data, _ := json.Marshal(payload)
	_ = utils.RedisClient.Set(context.Background(), key, data, iotDashboardCacheTTL).Err()
	return payload
}

func buildIoTDashboardPayload(device models.IoTDevice, year int) map[string]interface{} {
	if year <= 0 {
		year = time.Now().Year()
	}
	auth := authContext{TenantID: device.TenantID, UserID: 0, Username: "iot:" + device.SN, AccountType: "iot_device"}
	summary := buildSalesTargetSummary(auth, year)
	currency := strings.TrimSpace(fmt.Sprint(summary["currency"]))
	metricBasis := strings.TrimSpace(fmt.Sprint(summary["metric_basis"]))
	metrics := []map[string]interface{}{
		iotMetric("annual_target", "年度目标", numericValue(summary["company_target"]), currency, true),
		iotMetric("current_completed", "当前完成", numericValue(summary["current_amount"]), currency, true),
		iotPercentMetric("completion_rate", "完成率", numericValue(summary["target_progress"])),
		iotMetric("order_amount", "下单金额", numericValue(summary["order_amount"]), currency, true),
		iotMetric("paid_amount", "回款金额", numericValue(summary["paid_amount"]), currency, true),
		iotMetric("order_count", "订单数", numericValue(summary["order_count"]), "", false),
	}
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	etag := iotPayloadETag(device.TenantID, year, metricBasis, metrics)
	return map[string]interface{}{
		"refresh_interval_sec": iotRefreshInterval,
		"sn":                   device.SN,
		"resolution":           map[string]interface{}{"width": device.ResolutionWidth, "height": device.ResolutionHeight},
		"currency":             currency,
		"year":                 year,
		"metric_basis":         metricBasis,
		"generated_at":         generatedAt,
		"etag":                 etag,
		"metrics":              metrics,
	}
}

func iotMetric(key string, label string, raw float64, unit string, money bool) map[string]interface{} {
	value := formatIoTNumber(raw)
	if money && unit != "" {
		value = unit + " " + value
	}
	return map[string]interface{}{"key": key, "label": label, "value": value, "raw_value": raw, "unit": unit}
}

func iotPercentMetric(key string, label string, ratio float64) map[string]interface{} {
	percent := ratio * 100
	return map[string]interface{}{"key": key, "label": label, "value": fmt.Sprintf("%.1f%%", percent), "raw_value": ratio, "unit": "%"}
}

func formatIoTNumber(value float64) string {
	absValue := math.Abs(value)
	if absValue >= 1000000000 {
		return fmt.Sprintf("%.1fB", value/1000000000)
	}
	if absValue >= 1000000 {
		return fmt.Sprintf("%.1fM", value/1000000)
	}
	if absValue >= 1000 {
		return fmt.Sprintf("%.1fK", value/1000)
	}
	if math.Abs(value-math.Round(value)) < 0.000001 {
		return strconv.FormatInt(int64(math.Round(value)), 10)
	}
	return fmt.Sprintf("%.2f", value)
}

func iotPayloadETag(tenantID uint, year int, metricBasis string, metrics []map[string]interface{}) string {
	data, _ := json.Marshal(map[string]interface{}{"tenant_id": tenantID, "year": year, "basis": metricBasis, "metrics": metrics})
	sum := sha256.Sum256(data)
	return `"` + hex.EncodeToString(sum[:12]) + `"`
}

func iotDevicePayload(device models.IoTDevice) map[string]interface{} {
	online := false
	if device.LastSeenAt != nil {
		online = time.Since(*device.LastSeenAt) <= iotOnlineWindow
	}
	return map[string]interface{}{
		"id":                      device.ID,
		"name":                    device.Name,
		"sn":                      device.SN,
		"model":                   normalizeIoTModel(device.Model),
		"status":                  device.Status,
		"firmware_version":        device.FirmwareVersion,
		"target_firmware_version": device.TargetFirmwareVersion,
		"ota_status":              device.OTAStatus,
		"ota_failure_reason":      device.OTAFailureReason,
		"last_ota_check_at":       device.LastOTACheckAt,
		"last_ota_download_at":    device.LastOTADownloadAt,
		"last_ota_confirm_at":     device.LastOTAConfirmAt,
		"resolution_width":        device.ResolutionWidth,
		"resolution_height":       device.ResolutionHeight,
		"online":                  online,
		"last_seen_at":            device.LastSeenAt,
		"last_sync_at":            device.LastSyncAt,
		"last_ip":                 device.LastIP,
		"last_user_agent":         device.LastUserAgent,
		"remark":                  device.Remark,
		"created_at":              device.CreatedAt,
		"updated_at":              device.UpdatedAt,
	}
}

type iotDeviceActivityOptions struct {
	Sync            bool
	Force           bool
	FirmwareVersion string
}

func updateIoTDeviceActivity(ip string, ua string, device models.IoTDevice, options iotDeviceActivityOptions) {
	now := time.Now()
	firmwareVersion := normalizeIoTFirmwareVersion(options.FirmwareVersion)
	if utils.RedisClient != nil {
		key := "iot:last_db_seen:" + device.SN
		ok, err := utils.RedisClient.SetNX(context.Background(), key, "1", iotDeviceUpdateInterval).Result()
		if err == nil && !ok && !options.Force && !options.Sync && !shouldUpdateIoTFirmwareVersion(device.FirmwareVersion, firmwareVersion) {
			return
		}
	}
	updates := iotDeviceActivityUpdates(now, ip, ua, device, iotDeviceActivityOptions{
		Sync:            options.Sync,
		FirmwareVersion: firmwareVersion,
	})
	_ = utils.DB.Model(&models.IoTDevice{}).Where("id = ?", device.ID).Updates(updates).Error
}

func iotDeviceActivityUpdates(now time.Time, ip string, ua string, device models.IoTDevice, options iotDeviceActivityOptions) map[string]interface{} {
	firmwareVersion := normalizeIoTFirmwareVersion(options.FirmwareVersion)
	updates := map[string]interface{}{
		"last_seen_at":    &now,
		"last_ip":         strings.TrimSpace(ip),
		"last_user_agent": truncateIoTText(strings.TrimSpace(ua), iotMaxUserAgentLength),
	}
	if options.Sync {
		updates["last_sync_at"] = &now
	}
	if shouldUpdateIoTFirmwareVersion(device.FirmwareVersion, firmwareVersion) {
		updates["firmware_version"] = firmwareVersion
	}
	return updates
}

func iotFirmwareVersionFromRequest(request *http.Request) string {
	if request == nil {
		return ""
	}
	if version := normalizeIoTFirmwareVersion(request.Header.Get("X-IOT-Firmware-Version")); version != "" {
		return version
	}
	return iotFirmwareVersionFromUserAgent(request.UserAgent())
}

func iotFirmwareVersionFromUserAgent(ua string) string {
	fields := strings.FieldsFunc(ua, func(r rune) bool {
		switch r {
		case ' ', '\t', '\r', '\n', ';', ',', '/', '(', ')', '[', ']', '{', '}', '"', '\'':
			return true
		default:
			return false
		}
	})
	for _, field := range fields {
		if version := normalizeIoTFirmwareVersion(strings.Trim(field, "/")); strings.HasPrefix(strings.ToLower(version), "v") {
			return version
		}
	}
	return ""
}

func normalizeIoTFirmwareVersion(value string) string {
	value = truncateIoTText(strings.TrimSpace(value), 64)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	start := 0
	if strings.HasPrefix(lower, "v") {
		start = 1
		if len(value) == 1 {
			return ""
		}
	}
	segments := strings.Split(value[start:], ".")
	if len(segments) < 3 || len(segments) > 6 {
		return ""
	}
	for _, segment := range segments {
		if segment == "" {
			return ""
		}
		for _, ch := range segment {
			if ch < '0' || ch > '9' {
				return ""
			}
		}
	}
	return value
}

func shouldUpdateIoTFirmwareVersion(current string, reported string) bool {
	reported = normalizeIoTFirmwareVersion(reported)
	if reported == "" {
		return false
	}
	current = normalizeIoTFirmwareVersion(current)
	if current == "" {
		return true
	}
	cmp, ok := compareIoTFirmwareVersions(reported, current)
	if !ok {
		return false
	}
	return cmp > 0
}

func effectiveIoTFirmwareVersion(current string, reported string) string {
	reported = normalizeIoTFirmwareVersion(reported)
	if shouldUpdateIoTFirmwareVersion(current, reported) {
		return reported
	}
	return current
}

func compareIoTFirmwareVersions(left string, right string) (int, bool) {
	leftParts, ok := parseIoTFirmwareVersionParts(left)
	if !ok {
		return 0, false
	}
	rightParts, ok := parseIoTFirmwareVersionParts(right)
	if !ok {
		return 0, false
	}
	limit := len(leftParts)
	if len(rightParts) > limit {
		limit = len(rightParts)
	}
	for i := 0; i < limit; i++ {
		leftValue := 0
		if i < len(leftParts) {
			leftValue = leftParts[i]
		}
		rightValue := 0
		if i < len(rightParts) {
			rightValue = rightParts[i]
		}
		if leftValue > rightValue {
			return 1, true
		}
		if leftValue < rightValue {
			return -1, true
		}
	}
	return 0, true
}

func parseIoTFirmwareVersionParts(value string) ([]int, bool) {
	value = normalizeIoTFirmwareVersion(value)
	if value == "" {
		return nil, false
	}
	value = strings.TrimPrefix(strings.TrimPrefix(value, "v"), "V")
	segments := strings.Split(value, ".")
	parts := make([]int, 0, len(segments))
	for _, segment := range segments {
		part, err := strconv.Atoi(segment)
		if err != nil {
			return nil, false
		}
		parts = append(parts, part)
	}
	return parts, true
}

type iotOTADownloadClaims struct {
	SN        string
	ReleaseID uint
	Version   string
	ExpiresAt int64
	Nonce     string
}

type iotFirmwarePackageInfo struct {
	Version   string
	Model     string
	FileName  string
	Signature string
	Data      []byte
	SHA256    string
}

func parseIoTFirmwarePackage(uploadName string, data []byte) (iotFirmwarePackageInfo, error) {
	name := filepath.Base(strings.TrimSpace(uploadName))
	version, model := parseIoTFirmwareName(name)
	if version == "" || model == "" {
		return iotFirmwarePackageInfo{}, fmt.Errorf("固件包文件名必须包含型号和版本，例如 halmify-esp32-s3-epaper-v1.0.1.zip")
	}
	if strings.ToLower(filepath.Ext(name)) != ".zip" {
		return iotFirmwarePackageInfo{}, fmt.Errorf("请上传 zip 固件包")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return iotFirmwarePackageInfo{}, fmt.Errorf("固件包无法解压")
	}
	var bin []byte
	signature := ""
	binName := ""
	for _, file := range reader.File {
		base := strings.ToLower(filepath.Base(file.Name))
		if file.FileInfo().IsDir() {
			continue
		}
		if strings.HasSuffix(base, ".bin") {
			content, err := readZipEntryLimited(file, iotMaxFirmwareBytes)
			if err != nil {
				return iotFirmwarePackageInfo{}, fmt.Errorf("读取固件bin失败")
			}
			bin = content
			binName = filepath.Base(file.Name)
		}
		if strings.HasSuffix(base, ".sig") || strings.HasSuffix(base, ".signature") {
			content, err := readZipEntryLimited(file, 16*1024)
			if err != nil {
				return iotFirmwarePackageInfo{}, fmt.Errorf("读取固件签名失败")
			}
			signature = strings.TrimSpace(string(content))
		}
	}
	if len(bin) == 0 || signature == "" {
		return iotFirmwarePackageInfo{}, fmt.Errorf("固件包必须包含 .bin 和 .sig")
	}
	if _, err := base64.StdEncoding.DecodeString(signature); err != nil {
		return iotFirmwarePackageInfo{}, fmt.Errorf("固件签名必须是 base64 编码")
	}
	sum := sha256.Sum256(bin)
	return iotFirmwarePackageInfo{
		Version:   version,
		Model:     model,
		FileName:  iotFirstNonEmpty(binName, strings.TrimSuffix(name, filepath.Ext(name))+".bin"),
		Signature: signature,
		Data:      bin,
		SHA256:    hex.EncodeToString(sum[:]),
	}, nil
}

func readZipEntryLimited(file *zip.File, limit int64) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("zip entry too large")
	}
	return data, nil
}

func parseIoTFirmwareName(name string) (string, string) {
	trimmed := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	version := ""
	versionIndex := -1
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		clean := strings.TrimSpace(part)
		lowerPart := strings.ToLower(clean)
		if strings.HasPrefix(lowerPart, "v") && len(clean) > 1 && hasDigit(clean[1:]) {
			version = clean
			versionIndex = i
			break
		}
		if hasDigit(clean) && strings.Contains(clean, ".") {
			version = clean
			versionIndex = i
			break
		}
	}
	model := ""
	if versionIndex > 0 {
		modelParts := make([]string, 0, versionIndex)
		for _, part := range parts[:versionIndex] {
			clean := strings.TrimSpace(part)
			if clean == "" {
				continue
			}
			if len(modelParts) == 0 && strings.EqualFold(clean, "halmify") {
				continue
			}
			modelParts = append(modelParts, clean)
		}
		model = normalizeIoTModel(strings.Join(modelParts, "-"))
	}
	if model == "" {
		lower := strings.ToLower(trimmed)
		for _, candidate := range []string{"esp32-s3-epaper", "esp32-p4-epaper", "s3", "p4"} {
			if strings.Contains(lower, candidate) {
				model = normalizeIoTModel(candidate)
				break
			}
		}
	}
	return truncateIoTText(version, 64), model
}

func hasDigit(value string) bool {
	for _, ch := range value {
		if ch >= '0' && ch <= '9' {
			return true
		}
	}
	return false
}

func makeIoTOTADownloadToken(sn string, releaseID uint, version string, expiresAt time.Time) string {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		nonce = []byte(strconv.FormatInt(time.Now().UnixNano(), 10))
	}
	claims := []string{
		normalizeIoTSN(sn),
		strconv.FormatUint(uint64(releaseID), 10),
		strings.TrimSpace(version),
		strconv.FormatInt(expiresAt.Unix(), 10),
		hex.EncodeToString(nonce),
	}
	raw := strings.Join(claims, ".")
	sig := hmacSHA256Hex(configuredIoTMasterSecret(), raw)
	return base64.RawURLEncoding.EncodeToString([]byte(raw + "." + sig))
}

func parseIoTOTADownloadToken(token string) (iotOTADownloadClaims, bool) {
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil {
		return iotOTADownloadClaims{}, false
	}
	parts := strings.Split(string(data), ".")
	if len(parts) < 6 {
		return iotOTADownloadClaims{}, false
	}
	signature := parts[len(parts)-1]
	rawParts := parts[:len(parts)-1]
	raw := strings.Join(rawParts, ".")
	expected := hmacSHA256Hex(configuredIoTMasterSecret(), raw)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return iotOTADownloadClaims{}, false
	}
	releaseID, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || releaseID == 0 {
		return iotOTADownloadClaims{}, false
	}
	expiresIndex := len(rawParts) - 2
	nonceIndex := len(rawParts) - 1
	version := strings.Join(rawParts[2:expiresIndex], ".")
	expiresAt, err := strconv.ParseInt(rawParts[expiresIndex], 10, 64)
	if err != nil || time.Now().Unix() > expiresAt {
		return iotOTADownloadClaims{}, false
	}
	if parts[0] == "" || version == "" || rawParts[nonceIndex] == "" {
		return iotOTADownloadClaims{}, false
	}
	return iotOTADownloadClaims{SN: normalizeIoTSN(parts[0]), ReleaseID: uint(releaseID), Version: version, ExpiresAt: expiresAt, Nonce: rawParts[nonceIndex]}, true
}

func bindIoTFirmwarePublishRequest(c *IoTController, req *iotFirmwarePublishRequest) bool {
	body := c.Ctx.Input.RequestBody
	if len(body) == 0 && c.Ctx.Request != nil && c.Ctx.Request.Body != nil {
		data, err := io.ReadAll(c.Ctx.Request.Body)
		if err == nil {
			body = data
		}
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, req); err != nil {
			utils.BadRequestKey(c.Ctx, "common.invalidRequest", nil)
			return false
		}
		normalizeIoTFirmwarePublishRequest(req)
		return true
	}
	if err := c.Ctx.Request.ParseForm(); err != nil {
		utils.BadRequestKey(c.Ctx, "common.invalidRequest", nil)
		return false
	}
	req.Target = strings.TrimSpace(c.Ctx.Request.FormValue("target"))
	req.Force = strings.EqualFold(c.Ctx.Request.FormValue("force"), "true") || c.Ctx.Request.FormValue("force") == "1"
	if tenantID, err := strconv.ParseUint(strings.TrimSpace(c.Ctx.Request.FormValue("tenant_id")), 10, 64); err == nil {
		req.TenantID = uint(tenantID)
	}
	req.SNs = splitIoTFirmwarePublishSNs(c.Ctx.Request.Form["sns"])
	req.SNs = append(req.SNs, splitIoTFirmwarePublishSNs(c.Ctx.Request.Form["sns[]"])...)
	req.SNs = append(req.SNs, splitIoTFirmwarePublishSNs([]string{c.Ctx.Request.FormValue("sn"), c.Ctx.Request.FormValue("sns_text")})...)
	normalizeIoTFirmwarePublishRequest(req)
	return true
}

func normalizeIoTFirmwarePublishRequest(req *iotFirmwarePublishRequest) {
	req.Target = strings.ToLower(strings.TrimSpace(req.Target))
	if req.Target == "" {
		if req.TenantID > 0 {
			req.Target = "tenant"
		} else if len(req.SNs) > 0 {
			req.Target = "devices"
		}
	}
	normalizedSNs := make([]string, 0, len(req.SNs))
	seen := map[string]bool{}
	for _, sn := range req.SNs {
		normalized := normalizeIoTSN(sn)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		normalizedSNs = append(normalizedSNs, normalized)
	}
	req.SNs = normalizedSNs
}

func splitIoTFirmwarePublishSNs(values []string) []string {
	result := []string{}
	for _, value := range values {
		for _, item := range strings.FieldsFunc(value, func(r rune) bool {
			return r == ',' || r == '，' || r == '\n' || r == '\r' || r == '\t' || r == ' '
		}) {
			if normalized := normalizeIoTSN(item); normalized != "" {
				result = append(result, normalized)
			}
		}
	}
	return result
}

func sumIoTFirmwareTargetStats(stats map[string]int64) int64 {
	var total int64
	for _, count := range stats {
		total += count
	}
	return total
}

func sumIoTFirmwareTargetStatsByStatus(stats map[string]int64, statuses ...string) int64 {
	var total int64
	for _, status := range statuses {
		total += stats[status]
	}
	return total
}

func enrichIoTFirmwareTargets(targets []models.IoTFirmwareTarget) {
	if len(targets) == 0 {
		return
	}
	deviceIDs := make([]uint, 0, len(targets))
	tenantIDs := make([]uint, 0, len(targets))
	for _, target := range targets {
		if target.DeviceID > 0 {
			deviceIDs = append(deviceIDs, target.DeviceID)
		}
		if target.TenantID > 0 {
			tenantIDs = append(tenantIDs, target.TenantID)
		}
	}
	deviceMap := map[uint]models.IoTDevice{}
	if len(deviceIDs) > 0 {
		var devices []models.IoTDevice
		if err := utils.DB.Where("id IN ?", uniqueUintSlice(deviceIDs)).Find(&devices).Error; err == nil {
			for _, device := range devices {
				deviceMap[device.ID] = device
			}
		}
	}
	tenantMap := map[uint]string{}
	if len(tenantIDs) > 0 {
		var tenants []models.Tenant
		if err := utils.DB.Select("id, name").Where("id IN ?", uniqueUintSlice(tenantIDs)).Find(&tenants).Error; err == nil {
			for _, tenant := range tenants {
				tenantMap[tenant.ID] = tenant.Name
			}
		}
	}
	for index := range targets {
		if device, ok := deviceMap[targets[index].DeviceID]; ok {
			targets[index].DeviceName = device.Name
			targets[index].CurrentVersion = device.FirmwareVersion
			targets[index].LastDeviceSeenAt = device.LastSeenAt
			if targets[index].LastSeenAt == nil {
				targets[index].LastSeenAt = device.LastSeenAt
			}
		}
		targets[index].TenantName = tenantMap[targets[index].TenantID]
	}
}

func updateIoTFirmwareTargetStatus(device models.IoTDevice, releaseID uint, status string, updates map[string]interface{}) error {
	version := strings.TrimSpace(device.TargetFirmwareVersion)
	if version == "" {
		version = strings.TrimSpace(device.FirmwareVersion)
	}
	if releaseID == 0 && version == "" {
		return nil
	}
	query := utils.DB.Model(&models.IoTFirmwareTarget{}).Where("device_id = ?", device.ID)
	if releaseID > 0 {
		query = query.Where("release_id = ?", releaseID)
	} else {
		query = query.Where("version = ?", version)
	}
	if _, exists := updates["status"]; !exists {
		updates["status"] = status
	}
	if _, exists := updates["last_seen_at"]; !exists {
		updates["last_seen_at"] = device.LastSeenAt
	}
	result := query.Order("assigned_at desc, id desc").Limit(1).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 || releaseID == 0 {
		return nil
	}
	var release models.IoTFirmwareRelease
	if err := utils.DB.Where("id = ?", releaseID).First(&release).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	assignedAt := time.Now()
	target := models.IoTFirmwareTarget{
		ReleaseID:     release.ID,
		DeviceID:      device.ID,
		TenantID:      device.TenantID,
		SN:            device.SN,
		Model:         normalizeIoTModel(device.Model),
		Version:       release.Version,
		Status:        status,
		Force:         release.Force,
		FailureReason: stringFromInterface(updates["failure_reason"]),
		AssignedAt:    assignedAt,
		LastSeenAt:    device.LastSeenAt,
	}
	applyIoTFirmwareTargetTimes(&target, updates)
	return utils.DB.Where("release_id = ? AND device_id = ?", release.ID, device.ID).Assign(target).FirstOrCreate(&target).Error
}

func applyIoTFirmwareTargetTimes(target *models.IoTFirmwareTarget, values map[string]interface{}) {
	if value, ok := values["checked_at"].(*time.Time); ok {
		target.CheckedAt = value
	}
	if value, ok := values["downloaded_at"].(*time.Time); ok {
		target.DownloadedAt = value
	}
	if value, ok := values["confirmed_at"].(*time.Time); ok {
		target.ConfirmedAt = value
	}
	if value, ok := values["last_seen_at"].(*time.Time); ok {
		target.LastSeenAt = value
	}
}

func uniqueUintSlice(values []uint) []uint {
	seen := map[uint]bool{}
	result := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func stringFromInterface(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func applyIoTFirmwareTargets(release models.IoTFirmwareRelease, req iotFirmwarePublishRequest) (int64, error) {
	modelWhere, modelArgs := iotFirmwareTargetModelWhere(release.Model)
	query := utils.DB.Where(modelWhere, modelArgs...)
	target := strings.ToLower(strings.TrimSpace(req.Target))
	if target == "" {
		if req.TenantID > 0 {
			target = "tenant"
		} else if len(req.SNs) > 0 {
			target = "devices"
		}
	}
	switch target {
	case "tenant":
		if req.TenantID == 0 {
			return 0, iotFirmwarePublishError{Key: "iotFirmware.error.tenantRequired"}
		}
		query = query.Where("tenant_id = ?", req.TenantID)
	case "devices", "device":
		sns := make([]string, 0, len(req.SNs))
		for _, sn := range req.SNs {
			if normalized := normalizeIoTSN(sn); normalized != "" {
				sns = append(sns, normalized)
			}
		}
		if len(sns) == 0 {
			return 0, iotFirmwarePublishError{Key: "iotFirmware.error.snsRequired"}
		}
		query = query.Where("sn IN ?", sns)
	default:
		return 0, iotFirmwarePublishError{Key: "iotFirmware.error.invalidTarget"}
	}
	var devices []models.IoTDevice
	if err := query.Find(&devices).Error; err != nil {
		return 0, err
	}
	if len(devices) == 0 {
		return 0, iotFirmwarePublishError{Key: "iotFirmware.error.noMatchingDevice"}
	}
	now := time.Now()
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&release).Updates(map[string]interface{}{
			"status":       "published",
			"force":        release.Force,
			"published_at": release.PublishedAt,
			"revoked_at":   nil,
		}).Error; err != nil {
			return err
		}
		deviceIDs := make([]uint, 0, len(devices))
		for _, device := range devices {
			deviceIDs = append(deviceIDs, device.ID)
		}
		if err := tx.Model(&models.IoTDevice{}).Where("id IN ?", deviceIDs).Updates(map[string]interface{}{
			"target_firmware_version": release.Version,
			"ota_status":              "assigned",
			"ota_failure_reason":      "",
		}).Error; err != nil {
			return err
		}
		for _, device := range devices {
			targetRecord := models.IoTFirmwareTarget{
				ReleaseID:     release.ID,
				DeviceID:      device.ID,
				TenantID:      device.TenantID,
				SN:            device.SN,
				Model:         normalizeIoTModel(device.Model),
				Version:       release.Version,
				Status:        "assigned",
				Force:         release.Force,
				FailureReason: "",
				AssignedAt:    now,
				CheckedAt:     nil,
				DownloadedAt:  nil,
				ConfirmedAt:   nil,
				LastSeenAt:    device.LastSeenAt,
			}
			if err := tx.Where("release_id = ? AND device_id = ?", release.ID, device.ID).Assign(targetRecord).FirstOrCreate(&targetRecord).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return int64(len(devices)), nil
}

type iotFirmwarePublishError struct {
	Key string
}

func (err iotFirmwarePublishError) Error() string {
	return err.Key
}

func iotFirmwareTargetModelWhere(model string) (string, []interface{}) {
	normalized := normalizeIoTModel(model)
	if normalized == iotDefaultFirmwareModel {
		return "(model IN ? OR model = '' OR model IS NULL)", []interface{}{iotFirmwareModelCandidates(normalized)}
	}
	return "model IN ?", []interface{}{iotFirmwareModelCandidates(normalized)}
}

func iotFirmwareModelCandidates(model string) []string {
	normalized := normalizeIoTModel(model)
	switch normalized {
	case "esp32-s3-epaper":
		return []string{"esp32-s3-epaper", "esp32-s3", "esp32s3", "s3"}
	case "esp32-p4-epaper":
		return []string{"esp32-p4-epaper", "esp32-p4", "esp32p4", "p4"}
	default:
		return []string{normalized}
	}
}

func iotFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func checkIoTSignature(deviceSeed string, keyVersion string, method string, path string, timestamp string, nonce string, bodySHA string, signature string) bool {
	if bodySHA == "" {
		empty := sha256.Sum256(nil)
		bodySHA = hex.EncodeToString(empty[:])
	}
	secret := deriveIoTSigningKey(deviceSeed, keyVersion)
	if secret == "" {
		return false
	}
	expected := hmacSHA256Hex(secret, iotSignatureBaseString(method, path, keyVersion, timestamp, nonce, bodySHA))
	return hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected))
}

func iotSignatureBaseString(method string, path string, keyVersion string, timestamp string, nonce string, bodySHA string) string {
	return strings.Join([]string{strings.ToUpper(method), path, keyVersion, timestamp, nonce, bodySHA}, "\n")
}

func hmacSHA256Hex(key string, message string) string {
	if key == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func hmacSHA256BytesHex(key []byte, message string) string {
	if len(key) == 0 {
		return ""
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func iotBootstrapSigningString(sn string, publicKeyFingerprint string, timestamp string, nonce string, firmwareVersion string) string {
	return strings.Join([]string{"POST", "/api/iot/v1/bootstrap", sn, publicKeyFingerprint, timestamp, nonce, firmwareVersion}, "\n")
}

func iotRefreshSigningString(keyVersion string, timestamp string, nonce string) string {
	return strings.Join([]string{"POST", "/api/iot/v1/device-seed/refresh", keyVersion, timestamp, nonce}, "\n")
}

func deriveIoTSigningKey(deviceSeed string, keyVersion string) string {
	if deviceSeed == "" {
		return ""
	}
	material := strings.Join([]string{iotKeyDerivationContext, keyVersion}, ":")
	mac := hmac.New(sha256.New, []byte(deviceSeed))
	_, _ = mac.Write([]byte(material))
	return hex.EncodeToString(mac.Sum(nil))
}

func configuredIoTMasterSecret() string {
	secret := strings.TrimSpace(os.Getenv("IOT_MASTER_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(web.AppConfig.DefaultString("iot_master_secret", ""))
	}
	if secret == "" {
		secret = iotDefaultMasterSecret
	}
	return secret
}

func newIoTDeviceSeed() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	seed := base64.RawURLEncoding.EncodeToString(raw)
	sealed, err := sealIoTDeviceSeed(seed)
	if err != nil {
		return "", "", err
	}
	return seed, sealed, nil
}

func encryptIoTDeviceSeed(publicKeyPEM string, seed string) (map[string]interface{}, error) {
	devicePublicKey, _, fingerprint, err := parseIoTPublicKey(publicKeyPEM)
	if err != nil {
		return nil, err
	}
	ephemeralPrivate, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	sharedX, _ := devicePublicKey.Curve.ScalarMult(devicePublicKey.X, devicePublicKey.Y, ephemeralPrivate.D.Bytes())
	if sharedX == nil {
		return nil, fmt.Errorf("derive shared key failed")
	}
	ephemeralPubDER, err := x509.MarshalPKIXPublicKey(&ephemeralPrivate.PublicKey)
	if err != nil {
		return nil, err
	}
	wrapKey := deriveIoTWrapKey(sharedX.Bytes(), fingerprint)
	block, err := aes.NewCipher(wrapKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(seed), nil)
	return map[string]interface{}{
		"alg":                  "ECDH-P256-HMAC-SHA256-AES-256-GCM",
		"context":              iotSeedWrapContext,
		"device_fingerprint":   fingerprint,
		"ephemeral_public_key": base64.StdEncoding.EncodeToString(ephemeralPubDER),
		"nonce":                base64.StdEncoding.EncodeToString(nonce),
		"ciphertext":           base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func deriveIoTWrapKey(sharedSecret []byte, fingerprint string) []byte {
	mac := hmac.New(sha256.New, sharedSecret)
	_, _ = mac.Write([]byte(iotSeedWrapContext))
	_, _ = mac.Write([]byte(":"))
	_, _ = mac.Write([]byte(fingerprint))
	return mac.Sum(nil)
}

func sealIoTDeviceSeed(seed string) (string, error) {
	block, err := aes.NewCipher(iotEncryptionKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(seed), nil)
	return "seed:v1:" + base64.RawURLEncoding.EncodeToString(nonce) + ":" + base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func openIoTDeviceSeed(sealed string) (string, error) {
	parts := strings.Split(strings.TrimSpace(sealed), ":")
	if len(parts) != 4 || parts[0] != "seed" || parts[1] != "v1" {
		return "", fmt.Errorf("invalid sealed device seed")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(iotEncryptionKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func iotEncryptionKey() []byte {
	sum := sha256.Sum256([]byte(configuredIoTMasterSecret()))
	return sum[:]
}

func parseIoTPublicKey(raw string) (*ecdsa.PublicKey, string, string, error) {
	raw = strings.TrimSpace(raw)
	var der []byte
	if block, _ := pem.Decode([]byte(raw)); block != nil {
		der = block.Bytes
	} else {
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(raw)
			if err != nil {
				return nil, "", "", err
			}
		}
		der = decoded
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, "", "", err
	}
	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok || ecdsaPub.Curve != elliptic.P256() {
		return nil, "", "", fmt.Errorf("unsupported public key")
	}
	normalizedDER, err := x509.MarshalPKIXPublicKey(ecdsaPub)
	if err != nil {
		return nil, "", "", err
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: normalizedDER}
	normalizedPEM := string(pem.EncodeToMemory(block))
	sum := sha256.Sum256(normalizedDER)
	return ecdsaPub, normalizedPEM, hex.EncodeToString(sum[:]), nil
}

func verifyIoTDeviceSignature(publicKey *ecdsa.PublicKey, message string, signatureHex string) bool {
	signatureBytes, err := hex.DecodeString(strings.TrimSpace(signatureHex))
	if err != nil || len(signatureBytes) != 64 {
		return false
	}
	hash := sha256.Sum256([]byte(message))
	r := new(big.Int).SetBytes(signatureBytes[:32])
	s := new(big.Int).SetBytes(signatureBytes[32:])
	return ecdsa.Verify(publicKey, hash[:], r, s)
}

func validIoTKeyVersion(keyVersion string, allowed ...string) bool {
	for _, version := range allowed {
		if keyVersion == version {
			return true
		}
	}
	return false
}

func nowIoTKeyVersion() string {
	return time.Now().UTC().Format("2006-01")
}

func previousIoTKeyVersion() string {
	return time.Now().UTC().AddDate(0, -1, 0).Format("2006-01")
}

func normalizeIoTRateKey(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 160 {
		value = value[:160]
	}
	if value == "" {
		return "unknown"
	}
	return strings.NewReplacer(" ", "_", "\n", "_", "\r", "_", "\t", "_").Replace(value)
}

func truncateIoTText(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}

func normalizeIoTSN(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeIoTModel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(" ", "-", "_", "-").Replace(value)
	switch value {
	case "s3", "esp32-s3", "esp32s3", "esp32-s3-epaper":
		return "esp32-s3-epaper"
	case "p4", "esp32-p4", "esp32p4", "esp32-p4-epaper":
		return "esp32-p4-epaper"
	case "":
		return iotDefaultFirmwareModel
	default:
		return truncateIoTText(value, 64)
	}
}

func inferIoTModelFromSN(sn string) string {
	normalized := strings.ToLower(strings.TrimSpace(sn))
	if strings.Contains(normalized, "-s3-") || strings.Contains(normalized, "_s3_") {
		return "esp32-s3-epaper"
	}
	if strings.Contains(normalized, "-p4-") || strings.Contains(normalized, "_p4_") {
		return "esp32-p4-epaper"
	}
	return ""
}

func normalizeIoTStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "inactive", "disabled":
		return "disabled"
	default:
		return "active"
	}
}

func normalizeIoTUpdateStatus(value string, bound bool) string {
	normalized := normalizeIoTStatus(value)
	if normalized == "disabled" {
		return "disabled"
	}
	if !bound {
		return "pending_bind"
	}
	return "active"
}

func boundAtOrNow(existing *time.Time, now time.Time) *time.Time {
	if existing != nil {
		return existing
	}
	return &now
}

func normalizeIoTDimension(value int) int {
	if value <= 0 {
		return iotDefaultWidth
	}
	if value < 80 {
		return 80
	}
	if value > 4096 {
		return 4096
	}
	return value
}

func canManageIoT(auth authContext) bool {
	return auth.IsTenantAdmin()
}

func canViewIoT(auth authContext) bool {
	return auth.IsTenantAdmin()
}

func isDuplicateDBError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "duplicate") || strings.Contains(text, "unique")
}
