package controllers

import (
	"ai-crm/utils"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/redis/go-redis/v9"
)

const (
	defaultUploadMaxFileBytes            int64 = 10 * 1024 * 1024
	defaultDownloadRateWindowSeconds           = 60
	defaultDownloadUserRatePerWindow           = 60
	defaultDownloadIPRatePerWindow             = 120
	defaultDownloadTenantRatePerWindow         = 600
	defaultDownloadUserConcurrency             = 3
	defaultDownloadIPConcurrency               = 10
	defaultDownloadTenantConcurrency           = 50
	defaultDownloadConcurrencyTTLSeconds       = 300
)

var errUploadLimitExceeded = errors.New("upload file size limit exceeded")

type uploadQuotaConfig struct {
	MaxFileBytes int64
}

type downloadLimitConfig struct {
	Window            time.Duration
	UserRate          int
	IPRate            int
	TenantRate        int
	UserConcurrency   int
	IPConcurrency     int
	TenantConcurrency int
	ConcurrencyTTL    time.Duration
	FailOpen          bool
}

type downloadLimitDecision struct {
	Allowed     bool
	RetryAfter  time.Duration
	ReleaseFunc func()
}

type downloadLimitSubject struct {
	TenantID uint
	UserID   uint
	IP       string
}

func currentUploadQuotaConfig() uploadQuotaConfig {
	maxBytes := appConfigInt64WithEnv("fileserver_upload_max_file_bytes", "FILESERVER_UPLOAD_MAX_FILE_BYTES", defaultUploadMaxFileBytes)
	if maxBytes <= 0 {
		maxBytes = defaultUploadMaxFileBytes
	}
	return uploadQuotaConfig{MaxFileBytes: maxBytes}
}

func currentDownloadLimitConfig() downloadLimitConfig {
	windowSec := appConfigIntWithEnv("fileserver_download_rate_window_sec", "FILESERVER_DOWNLOAD_RATE_WINDOW_SEC", defaultDownloadRateWindowSeconds)
	ttlSec := appConfigIntWithEnv("fileserver_download_concurrency_ttl_sec", "FILESERVER_DOWNLOAD_CONCURRENCY_TTL_SEC", defaultDownloadConcurrencyTTLSeconds)
	if windowSec <= 0 {
		windowSec = defaultDownloadRateWindowSeconds
	}
	if ttlSec <= 0 {
		ttlSec = defaultDownloadConcurrencyTTLSeconds
	}
	return downloadLimitConfig{
		Window:            time.Duration(windowSec) * time.Second,
		UserRate:          positiveConfigInt("fileserver_download_user_rate_per_window", "FILESERVER_DOWNLOAD_USER_RATE_PER_WINDOW", defaultDownloadUserRatePerWindow),
		IPRate:            positiveConfigInt("fileserver_download_ip_rate_per_window", "FILESERVER_DOWNLOAD_IP_RATE_PER_WINDOW", defaultDownloadIPRatePerWindow),
		TenantRate:        positiveConfigInt("fileserver_download_tenant_rate_per_window", "FILESERVER_DOWNLOAD_TENANT_RATE_PER_WINDOW", defaultDownloadTenantRatePerWindow),
		UserConcurrency:   positiveConfigInt("fileserver_download_user_concurrency", "FILESERVER_DOWNLOAD_USER_CONCURRENCY", defaultDownloadUserConcurrency),
		IPConcurrency:     positiveConfigInt("fileserver_download_ip_concurrency", "FILESERVER_DOWNLOAD_IP_CONCURRENCY", defaultDownloadIPConcurrency),
		TenantConcurrency: positiveConfigInt("fileserver_download_tenant_concurrency", "FILESERVER_DOWNLOAD_TENANT_CONCURRENCY", defaultDownloadTenantConcurrency),
		ConcurrencyTTL:    time.Duration(ttlSec) * time.Second,
		FailOpen:          appConfigBoolWithEnv("fileserver_download_rate_limit_fail_open", "FILESERVER_DOWNLOAD_RATE_LIMIT_FAIL_OPEN", true),
	}
}

func positiveConfigInt(key string, envKey string, fallback int) int {
	value := appConfigIntWithEnv(key, envKey, fallback)
	if value <= 0 {
		return fallback
	}
	return value
}

func appConfigIntWithEnv(key string, envKey string, fallback int) int {
	if env := strings.TrimSpace(os.Getenv(envKey)); env != "" {
		if parsed, err := strconv.Atoi(env); err == nil {
			return parsed
		}
	}
	return web.AppConfig.DefaultInt(key, fallback)
}

func appConfigInt64WithEnv(key string, envKey string, fallback int64) int64 {
	if env := strings.TrimSpace(os.Getenv(envKey)); env != "" {
		if parsed, err := strconv.ParseInt(env, 10, 64); err == nil {
			return parsed
		}
	}
	return web.AppConfig.DefaultInt64(key, fallback)
}

func appConfigBoolWithEnv(key string, envKey string, fallback bool) bool {
	if env := strings.TrimSpace(os.Getenv(envKey)); env != "" {
		if parsed, err := strconv.ParseBool(env); err == nil {
			return parsed
		}
	}
	return web.AppConfig.DefaultBool(key, fallback)
}

func validateUploadSize(auth authContext, header *multipart.FileHeader) error {
	cfg := currentUploadQuotaConfig()
	if header != nil && header.Size > cfg.MaxFileBytes {
		return errUploadLimitExceeded
	}
	if err := ensureTenantStorageAvailable(auth.TenantID, uploadFileSize(header)); err != nil {
		return err
	}
	return nil
}

func uploadFileSize(header *multipart.FileHeader) int64 {
	if header == nil || header.Size < 0 {
		return 0
	}
	return header.Size
}

func copyUploadWithLimit(dst io.Writer, src io.Reader, maxBytes int64) error {
	if maxBytes <= 0 {
		_, err := io.Copy(dst, src)
		return err
	}
	limited := &io.LimitedReader{R: src, N: maxBytes + 1}
	written, err := io.Copy(dst, limited)
	if err != nil {
		return err
	}
	if written > maxBytes || limited.N <= 0 {
		return errUploadLimitExceeded
	}
	return nil
}

func applyDownloadLimit(auth authContext, ip string) downloadLimitDecision {
	cfg := currentDownloadLimitConfig()
	subject := downloadLimitSubject{TenantID: auth.TenantID, UserID: auth.UserID, IP: strings.TrimSpace(ip)}
	decision, err := applyDownloadLimitWithRedis(context.Background(), utils.RedisClient, cfg, subject)
	if err == nil {
		return decision
	}
	utils.LogWarning("文件下载限流异常 tenant=%d user=%d ip=%s err=%v", auth.TenantID, auth.UserID, subject.IP, err)
	if cfg.FailOpen {
		return downloadLimitDecision{Allowed: true}
	}
	return downloadLimitDecision{Allowed: false, RetryAfter: cfg.Window}
}

func applyDownloadLimitWithRedis(ctx context.Context, client *redis.Client, cfg downloadLimitConfig, subject downloadLimitSubject) (downloadLimitDecision, error) {
	if client == nil {
		return downloadLimitDecision{}, errors.New("redis client is nil")
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Duration(defaultDownloadRateWindowSeconds) * time.Second
	}
	if cfg.ConcurrencyTTL <= 0 {
		cfg.ConcurrencyTTL = time.Duration(defaultDownloadConcurrencyTTLSeconds) * time.Second
	}
	rateKeys := []struct {
		key   string
		limit int
	}{
		{downloadLimitKey("rate", "tenant", strconv.FormatUint(uint64(subject.TenantID), 10)), cfg.TenantRate},
		{downloadLimitKey("rate", "user", strconv.FormatUint(uint64(subject.UserID), 10)), cfg.UserRate},
		{downloadLimitKey("rate", "ip", subject.IP), cfg.IPRate},
	}
	for _, item := range rateKeys {
		if item.limit <= 0 || strings.HasSuffix(item.key, ":") {
			continue
		}
		count, ttl, err := incrementFixedWindow(ctx, client, item.key, cfg.Window)
		if err != nil {
			return downloadLimitDecision{}, err
		}
		if count > int64(item.limit) {
			return downloadLimitDecision{Allowed: false, RetryAfter: ttl}, nil
		}
	}

	concurrencyKeys := []struct {
		key   string
		limit int
	}{
		{downloadLimitKey("conc", "tenant", strconv.FormatUint(uint64(subject.TenantID), 10)), cfg.TenantConcurrency},
		{downloadLimitKey("conc", "user", strconv.FormatUint(uint64(subject.UserID), 10)), cfg.UserConcurrency},
		{downloadLimitKey("conc", "ip", subject.IP), cfg.IPConcurrency},
	}
	acquired := make([]string, 0, len(concurrencyKeys))
	release := func() {
		for i := len(acquired) - 1; i >= 0; i-- {
			if err := client.Decr(context.Background(), acquired[i]).Err(); err != nil {
				utils.LogWarning("文件下载并发计数释放失败 key=%s err=%v", acquired[i], err)
			}
		}
	}
	for _, item := range concurrencyKeys {
		if item.limit <= 0 || strings.HasSuffix(item.key, ":") {
			continue
		}
		count, err := client.Incr(ctx, item.key).Result()
		if err != nil {
			release()
			return downloadLimitDecision{}, err
		}
		_ = client.Expire(ctx, item.key, cfg.ConcurrencyTTL).Err()
		if count > int64(item.limit) {
			_ = client.Decr(ctx, item.key).Err()
			release()
			return downloadLimitDecision{Allowed: false}, nil
		}
		acquired = append(acquired, item.key)
	}
	return downloadLimitDecision{Allowed: true, ReleaseFunc: release}, nil
}

func incrementFixedWindow(ctx context.Context, client *redis.Client, key string, window time.Duration) (int64, time.Duration, error) {
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return 0, 0, err
	}
	if count == 1 {
		if err := client.Expire(ctx, key, window).Err(); err != nil {
			return 0, 0, err
		}
		return count, window, nil
	}
	ttl, err := client.TTL(ctx, key).Result()
	if err != nil {
		return 0, 0, err
	}
	if ttl <= 0 {
		ttl = window
		_ = client.Expire(ctx, key, window).Err()
	}
	return count, ttl, nil
}

func downloadLimitKey(kind string, dimension string, value string) string {
	return "fileserver:download:" + kind + ":" + dimension + ":" + strings.TrimSpace(value)
}
