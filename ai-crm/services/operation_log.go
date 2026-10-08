package services

import (
	"ai-crm/models"
	"ai-crm/utils"
	"encoding/json"
	"os"
	"strconv"
	"time"
)

type OperationLogEvent struct {
	TenantID       uint
	UserID         uint
	Username       string
	Operation      string
	Target         string
	DescriptionKey string
	Params         map[string]interface{}
	Result         string
	Reason         string
	IP             string
	UserAgent      string
	Source         string
}

var operationLogQueue = make(chan OperationLogEvent, 256)

func InitOperationLogger() {
	go func() {
		for event := range operationLogQueue {
			writeOperationLog(event)
		}
	}()
}

func LogOperation(event OperationLogEvent) {
	if event.Result == "" {
		event.Result = "success"
	}
	if event.Source == "" {
		event.Source = "web"
	}
	select {
	case operationLogQueue <- event:
	default:
		go writeOperationLog(event)
	}
}

func CleanupOperationLogs(retention time.Duration) {
	if retention <= 0 {
		retention = 180 * 24 * time.Hour
	}
	cutoff := time.Now().Add(-retention)
	if err := utils.DB.Unscoped().Where("operation <> ? AND created_at < ?", "LOGIN", cutoff).Delete(&models.OperationLog{}).Error; err != nil {
		utils.LogWarning("操作日志清理失败: %v", err)
	}
	loginCutoff := time.Now().AddDate(-1, 0, 0)
	if err := utils.DB.Unscoped().Where("operation = ? AND created_at < ?", "LOGIN", loginCutoff).Delete(&models.OperationLog{}).Error; err != nil {
		utils.LogWarning("登录日志清理失败: %v", err)
	}
}

func CleanupOperationLogsWithLock(retention time.Duration) {
	withJobLock("operation_log_cleanup", func() {
		CleanupOperationLogs(retention)
	})
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		withJobLock("operation_log_cleanup", func() {
			CleanupOperationLogs(retention)
		})
	}
}

func OperationLogRetention() time.Duration {
	days := 180
	if value := os.Getenv("OPERATION_LOG_RETENTION_DAYS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			days = parsed
		}
	}
	return time.Duration(days) * 24 * time.Hour
}

func writeOperationLog(event OperationLogEvent) {
	paramsJSON := ""
	if event.Params != nil {
		if data, err := json.Marshal(event.Params); err == nil {
			paramsJSON = string(data)
		}
	}
	log := models.OperationLog{
		TenantModel:    models.TenantModel{TenantID: event.TenantID},
		UserID:         event.UserID,
		Username:       event.Username,
		Operation:      event.Operation,
		Target:         event.Target,
		DescriptionKey: event.DescriptionKey,
		ParamsJSON:     paramsJSON,
		Result:         event.Result,
		Reason:         event.Reason,
		IP:             event.IP,
		UserAgent:      event.UserAgent,
		Source:         event.Source,
	}
	if err := utils.DB.Create(&log).Error; err != nil {
		utils.LogWarning("操作日志写入失败: %v", err)
	}
}
