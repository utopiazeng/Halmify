package utils

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"ai-crm/models"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB 全局数据库连接
var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() {
	// 从配置文件读取数据库配置
	dbHost, _ := web.AppConfig.String("db_host")
	dbPort, _ := web.AppConfig.String("db_port")
	dbUser, _ := web.AppConfig.String("db_user")
	dbPassword, _ := web.AppConfig.String("db_password")
	dbName, _ := web.AppConfig.String("db_name")
	dbSocket, _ := web.AppConfig.String("db_socket")

	// 使用环境变量覆盖配置
	if envHost := os.Getenv("DB_HOST"); envHost != "" {
		dbHost = envHost
	}
	if envPort := os.Getenv("DB_PORT"); envPort != "" {
		dbPort = envPort
	}
	if envUser := os.Getenv("DB_USER"); envUser != "" {
		dbUser = envUser
	}
	if envPassword := os.Getenv("DB_PASSWORD"); envPassword != "" {
		dbPassword = envPassword
	}
	if envName := os.Getenv("DB_NAME"); envName != "" {
		dbName = envName
	}
	if envSocket := os.Getenv("DB_SOCKET"); envSocket != "" {
		dbSocket = envSocket
	}

	// 构建 DSN。只有显式配置 db_socket/DB_SOCKET 时才走 Unix Socket；留空使用 db_host/db_port TCP。
	var dsn string
	dbSocket = strings.TrimSpace(dbSocket)
	if dbSocket != "" {
		dsn = fmt.Sprintf("%s:%s@unix(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dbUser, dbPassword, dbSocket, dbName)
	} else {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	// 连接数据库
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("数据库连接失败: %v (host=%s port=%s user=%s db=%s，可通过 DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME 覆盖)", err, dbHost, dbPort, dbUser, dbName))
	}
	configureDBPool(DB)

	// 增量迁移（多实例场景通过DB锁保证同一时刻仅一个实例执行）
	if shouldRunMigrations() {
		if err := runMigrationsWithLock(DB); err != nil {
			panic(fmt.Sprintf("数据库迁移失败: %v (db=%s user=%s)", err, dbName, dbUser))
		}
	}

	LogInfo("数据库连接成功")
}

func configureDBPool(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		LogWarning("获取SQL DB失败，跳过连接池配置: %v", err)
		return
	}

	maxOpen := readIntConfig("DB_MAX_OPEN_CONNS", "db_max_open_conns", 80)
	maxIdle := readIntConfig("DB_MAX_IDLE_CONNS", "db_max_idle_conns", 20)
	maxLifetimeMin := readIntConfig("DB_CONN_MAX_LIFETIME_MINUTES", "db_conn_max_lifetime_minutes", 30)
	maxIdleTimeMin := readIntConfig("DB_CONN_MAX_IDLE_TIME_MINUTES", "db_conn_max_idle_time_minutes", 10)

	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(time.Duration(maxLifetimeMin) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(maxIdleTimeMin) * time.Minute)
}

func shouldRunMigrations() bool {
	value := strings.TrimSpace(os.Getenv("DB_RUN_MIGRATIONS"))
	if value == "" {
		if conf, err := web.AppConfig.String("db_run_migrations"); err == nil {
			value = strings.TrimSpace(conf)
		}
	}
	if value == "" {
		return true
	}
	return strings.EqualFold(value, "1") || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

func runMigrationsWithLock(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	lockName := strings.TrimSpace(os.Getenv("DB_MIGRATION_LOCK_NAME"))
	if lockName == "" {
		lockName, _ = web.AppConfig.String("db_migration_lock_name")
	}
	if lockName == "" {
		lockName = "ai_crm_schema_migration_lock"
	}
	lockTimeout := readIntConfig("DB_MIGRATION_LOCK_TIMEOUT_SEC", "db_migration_lock_timeout_sec", 15)

	ok, err := acquireMySQLLock(sqlDB, lockName, lockTimeout)
	if err != nil {
		return err
	}
	if !ok {
		LogInfo("迁移锁未获取，跳过本实例迁移: %s", lockName)
		return nil
	}
	defer releaseMySQLLock(sqlDB, lockName)

	return RunMigrations()
}

func acquireMySQLLock(sqlDB *sql.DB, lockName string, timeoutSec int) (bool, error) {
	row := sqlDB.QueryRow("SELECT GET_LOCK(?, ?)", lockName, timeoutSec)
	var result sql.NullInt64
	if err := row.Scan(&result); err != nil {
		return false, err
	}
	return result.Valid && result.Int64 == 1, nil
}

func releaseMySQLLock(sqlDB *sql.DB, lockName string) {
	_, _ = sqlDB.Exec("DO RELEASE_LOCK(?)", lockName)
}

func readIntConfig(envKey, confKey string, def int) int {
	if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	if parsed, err := web.AppConfig.Int(confKey); err == nil && parsed > 0 {
		return parsed
	}
	return def
}

// AutoMigrate 自动迁移数据库表（仅用于迁移步骤内部）
func AutoMigrate() error {
	return autoMigrateWithDB(DB)
}

func autoMigrateWithDB(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Tenant{},
		&models.TenantQuota{},
		&models.TenantStorageExpansion{},
		&models.TenantSeatExpansion{},
		&models.PlatformPlanTemplate{},
		&models.TenantExpiryNotice{},
		&models.TenantIndustryPackage{},
		&models.IndustryPackageRegistry{},
		&models.IndustryExtensionRegistry{},
		&models.MfgCustomerProfile{},
		&models.MfgDecisionChain{},
		&models.MetricCustomerSnapshot{},
		&models.MfgRFQ{},
		&models.MfgRFQClarification{},
		&models.MfgRFQClarificationAnswerHistory{},
		&models.MfgDrawing{},
		&models.MfgProductFamily{},
		&models.MfgConfigModel{},
		&models.MfgBOMTemplate{},
		&models.MfgBOMSnapshot{},
		&models.MfgCostSheet{},
		&models.MfgQuoteRevision{},
		&models.MfgQuoteApprovalHistory{},
		&models.MfgSampleProject{},
		&models.MfgSampleTestRecord{},
		&models.MfgProductionProgress{},
		&models.MfgECN{},
		&models.MfgIntegrationOutbox{},
		&models.MfgCompensationRecord{},
		&models.MfgAsset{},
		&models.MfgServiceTicket{},
		&models.MfgSparePartQuote{},
		&models.MfgSalesAgreement{},
		&models.MfgAccountForecast{},
		&models.MfgKnowledgeEntry{},
		&models.MfgKnowledgeUsage{},
		&models.MetricMfgQuoteSnapshot{},
		&models.MetricMfgDeliverySnapshot{},
		&models.MetricMfgServiceSnapshot{},
		&models.User{},
		&models.ApprovalTemplate{},
		&models.ApprovalInstance{},
		&models.ApprovalActionLog{},
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserRole{},
		&models.SystemDepartment{},
		&models.SystemPost{},
		&models.SystemMenu{},
		&models.UserSession{},
		&models.CRMObject{},
		&models.CRMRelation{},
		&models.CRMObjectACLEntry{},
		&models.TenantAIWallet{},
		&models.AIUsageLog{},
		&models.AIResearchJob{},
		&models.AITokenLedgerEntry{},
		&models.TenantAIApplication{},
		&models.AIAnalysisPackage{},
		&models.CRMSetting{},
		&models.PlatformSettingVersion{},
		&models.TenantAdminActivationToken{},
		&models.PortalTicket{},
		&models.OperationLog{},
		&models.IMConversation{},
		&models.IMMessage{},
		&models.IMMessageReceipt{},
		&models.IMUserPresence{},
		&models.IMUserSetting{},
		&models.IMSessionSetting{},
		&models.IMCleanupJob{},
		&models.IoTDevice{},
		&models.IoTFirmwareRelease{},
		&models.IoTFirmwareTarget{},
	)
}
