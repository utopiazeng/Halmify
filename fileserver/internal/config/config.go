package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr                 string
	NodeID               string
	StorageRoot          string
	DBDSN                string
	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxLifetimeMin int
	DBConnMaxIdleTimeMin int
	DBMigrationLockName  string
	DBMigrationLockWait  time.Duration
	RedisAddr            string
	RedisPassword        string
	RedisDB              int
	AppID                string
	AppSecret            string
	MaxUploadBytes       int64
	AllowedClockSkew     time.Duration
	NonceTTL             time.Duration
	DeleteWorkerInterval time.Duration
	SchedulerLockTTL     time.Duration
	SchedulerLockRenew   time.Duration
}

func Load() (Config, error) {
	configPath := strings.TrimSpace(os.Getenv("FILESERVER_CONFIG"))
	if configPath == "" {
		configPath = "conf/app.conf"
	}
	values, err := loadKVConfig(configPath)
	if err != nil {
		return Config{}, err
	}

	addr := readString(values, "server_addr", "FILESERVER_ADDR")
	nodeID := readString(values, "node_id", "FILESERVER_NODE_ID")
	storageRoot := readString(values, "storage_root", "FILESERVER_STORAGE_ROOT")
	dbHost := readString(values, "db_host", "FILESERVER_DB_HOST")
	dbPort := readString(values, "db_port", "FILESERVER_DB_PORT")
	dbUser := readString(values, "db_user", "FILESERVER_DB_USER")
	dbPassword := readString(values, "db_password", "FILESERVER_DB_PASSWORD")
	dbName := readString(values, "db_name", "FILESERVER_DB_NAME")
	dbParams := readString(values, "db_params", "FILESERVER_DB_PARAMS")
	dbSocket := readString(values, "db_socket", "FILESERVER_DB_SOCKET")
	dbMaxOpen := readInt(values, "db_max_open_conns", "FILESERVER_DB_MAX_OPEN_CONNS")
	dbMaxIdle := readInt(values, "db_max_idle_conns", "FILESERVER_DB_MAX_IDLE_CONNS")
	dbLifeMin := readInt(values, "db_conn_max_lifetime_minutes", "FILESERVER_DB_CONN_MAX_LIFETIME_MINUTES")
	dbIdleMin := readInt(values, "db_conn_max_idle_time_minutes", "FILESERVER_DB_CONN_MAX_IDLE_TIME_MINUTES")
	dbMigrationLockName := readString(values, "db_migration_lock_name", "FILESERVER_DB_MIGRATION_LOCK_NAME")
	dbMigrationLockWaitSec := readInt(values, "db_migration_lock_wait_sec", "FILESERVER_DB_MIGRATION_LOCK_WAIT_SEC")
	redisHost := readString(values, "redis_host", "FILESERVER_REDIS_HOST")
	redisPort := readString(values, "redis_port", "FILESERVER_REDIS_PORT")
	redisPassword := readString(values, "redis_password", "FILESERVER_REDIS_PASSWORD")
	redisDB := readInt(values, "redis_db", "FILESERVER_REDIS_DB")
	appID := readString(values, "app_id", "FILESERVER_APP_ID")
	appSecret := readString(values, "app_secret", "FILESERVER_APP_SECRET")
	maxUploadBytes := readInt64(values, "max_upload_bytes", "FILESERVER_MAX_UPLOAD_BYTES")
	skewSec := readInt(values, "allowed_clock_skew_sec", "FILESERVER_ALLOWED_CLOCK_SKEW_SEC")
	nonceSec := readInt(values, "nonce_ttl_sec", "FILESERVER_NONCE_TTL_SEC")
	workerSec := readInt(values, "delete_worker_interval_sec", "FILESERVER_DELETE_WORKER_INTERVAL_SEC")
	lockTTLSec := readInt(values, "scheduler_lock_ttl_sec", "FILESERVER_SCHEDULER_LOCK_TTL_SEC")
	lockRenewSec := readInt(values, "scheduler_lock_renew_sec", "FILESERVER_SCHEDULER_LOCK_RENEW_SEC")

	missing := make([]string, 0)
	if addr == "" {
		missing = append(missing, "server_addr")
	}
	if storageRoot == "" {
		missing = append(missing, "storage_root")
	}
	if nodeID == "" {
		missing = append(missing, "node_id")
	}
	if dbHost == "" || dbPort == "" || dbUser == "" || dbName == "" {
		missing = append(missing, "db_host/db_port/db_user/db_name")
	}
	if redisHost == "" || redisPort == "" {
		missing = append(missing, "redis_host/redis_port")
	}
	if appID == "" || appSecret == "" {
		missing = append(missing, "app_id/app_secret")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required config in %s: %s", configPath, strings.Join(missing, ", "))
	}
	if dbParams == "" {
		dbParams = "charset=utf8mb4&parseTime=True&loc=Local"
	}
	if skewSec <= 0 {
		skewSec = 300
	}
	if nonceSec <= 0 {
		nonceSec = 300
	}
	if workerSec <= 0 {
		workerSec = 3
	}
	if lockTTLSec <= 0 {
		lockTTLSec = 90
	}
	if lockRenewSec <= 0 {
		lockRenewSec = 30
	}
	if dbMaxOpen <= 0 {
		dbMaxOpen = 60
	}
	if dbMaxIdle <= 0 {
		dbMaxIdle = 20
	}
	if dbMaxIdle > dbMaxOpen {
		dbMaxIdle = dbMaxOpen
	}
	if dbLifeMin <= 0 {
		dbLifeMin = 30
	}
	if dbIdleMin <= 0 {
		dbIdleMin = 10
	}
	if dbMigrationLockName == "" {
		dbMigrationLockName = "fileserver_schema_migration_lock"
	}
	if dbMigrationLockWaitSec <= 0 {
		dbMigrationLockWaitSec = 15
	}
	if maxUploadBytes <= 0 {
		maxUploadBytes = 1024 * 1024 * 1024
	}

	dbDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", dbUser, dbPassword, dbHost, dbPort, dbName, dbParams)
	if strings.TrimSpace(dbSocket) != "" {
		dbDSN = fmt.Sprintf("%s:%s@unix(%s)/%s?%s", dbUser, dbPassword, dbSocket, dbName, dbParams)
	}

	cfg := Config{
		Addr:                 addr,
		NodeID:               nodeID,
		StorageRoot:          strings.TrimRight(storageRoot, "/"),
		DBDSN:                dbDSN,
		DBMaxOpenConns:       dbMaxOpen,
		DBMaxIdleConns:       dbMaxIdle,
		DBConnMaxLifetimeMin: dbLifeMin,
		DBConnMaxIdleTimeMin: dbIdleMin,
		DBMigrationLockName:  dbMigrationLockName,
		DBMigrationLockWait:  time.Duration(dbMigrationLockWaitSec) * time.Second,
		RedisAddr:            redisHost + ":" + redisPort,
		RedisPassword:        redisPassword,
		RedisDB:              redisDB,
		AppID:                appID,
		AppSecret:            appSecret,
		MaxUploadBytes:       maxUploadBytes,
		AllowedClockSkew:     time.Duration(skewSec) * time.Second,
		NonceTTL:             time.Duration(nonceSec) * time.Second,
		DeleteWorkerInterval: time.Duration(workerSec) * time.Second,
		SchedulerLockTTL:     time.Duration(lockTTLSec) * time.Second,
		SchedulerLockRenew:   time.Duration(lockRenewSec) * time.Second,
	}
	return cfg, nil
}

func loadKVConfig(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file failed: %w", err)
	}
	defer f.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		values[key] = val
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config file failed: %w", err)
	}
	return values, nil
}

func readString(values map[string]string, key, envKey string) string {
	if val := strings.TrimSpace(os.Getenv(envKey)); val != "" {
		return val
	}
	return strings.TrimSpace(values[key])
}

func readInt(values map[string]string, key, envKey string) int {
	if val := strings.TrimSpace(os.Getenv(envKey)); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
		return 0
	}
	raw := strings.TrimSpace(values[key])
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return parsed
}

func readInt64(values map[string]string, key, envKey string) int64 {
	if val := strings.TrimSpace(os.Getenv(envKey)); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			return parsed
		}
		return 0
	}
	raw := strings.TrimSpace(values[key])
	if raw == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}
