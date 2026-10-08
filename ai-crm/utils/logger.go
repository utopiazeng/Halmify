package utils

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

// InitLogger 初始化日志
func InitLogger() {
	logPath, _ := web.AppConfig.String("log_path")
	if logPath == "" {
		logPath = "logs/"
	}

	// 确保日志目录存在
	if err := os.MkdirAll(logPath, 0755); err != nil {
		logs.Error("创建日志目录失败: %v", err)
		return
	}

	cleanupOldLogFiles(logPath, 7*24*time.Hour)

	// 设置日志输出
	logFile := filepath.Join(logPath, "app.log")
	logs.SetLogger(logs.AdapterFile, `{"filename":"`+logFile+`","level":7,"maxlines":0,"maxsize":0,"daily":true,"maxdays":7,"color":true}`)

	// 同时输出到控制台
	logs.SetLogger(logs.AdapterConsole)

	logs.Info("日志系统初始化完成")
}

// LogInfo 记录信息日志
func LogInfo(format string, v ...interface{}) {
	logs.Info(format, v...)
}

// LogError 记录错误日志
func LogError(format string, v ...interface{}) {
	logs.Error(format, v...)
}

// LogWarning 记录警告日志
func LogWarning(format string, v ...interface{}) {
	logs.Warning(format, v...)
}

// LogDebug 记录调试日志
func LogDebug(format string, v ...interface{}) {
	logs.Debug(format, v...)
}

func cleanupOldLogFiles(logPath string, maxAge time.Duration) {
	entries, err := os.ReadDir(logPath)
	if err != nil {
		logs.Warning("读取日志目录失败: %v", err)
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() || !isLogFileName(entry.Name()) {
			continue
		}
		fullPath := filepath.Join(logPath, entry.Name())
		info, err := entry.Info()
		if err != nil {
			logs.Warning("读取日志文件信息失败: %s err=%v", fullPath, err)
			continue
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(fullPath); err != nil {
			logs.Warning("清理过期日志文件失败: %s err=%v", fullPath, err)
		}
	}
}

func isLogFileName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return lower == "app.log" || strings.HasPrefix(lower, "app.log.") || strings.HasSuffix(lower, ".log")
}
