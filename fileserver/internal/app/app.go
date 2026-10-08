package app

import (
	"context"
	"database/sql"
	"fileserver/internal/config"
	"fileserver/internal/handler"
	"fileserver/internal/middleware"
	"fileserver/internal/repository"
	"fileserver/internal/service"
	"fileserver/internal/storage"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func OpenDB(cfg config.Config) (*gorm.DB, *sql.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	ConfigureDBPool(sqlDB, cfg)
	return db, sqlDB, nil
}

func ConfigureDBPool(sqlDB *sql.DB, cfg config.Config) {
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeMin) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.DBConnMaxIdleTimeMin) * time.Minute)
}

func RunMigration(cfg config.Config) error {
	db, sqlDB, err := OpenDB(cfg)
	if err != nil {
		return err
	}
	acquired, err := acquireMySQLLock(sqlDB, cfg.DBMigrationLockName, int(cfg.DBMigrationLockWait/time.Second))
	if err != nil {
		return err
	}
	if !acquired {
		return &MigrationLockError{LockName: cfg.DBMigrationLockName}
	}
	defer releaseMySQLLock(sqlDB, cfg.DBMigrationLockName)

	return repository.New(db).AutoMigrate()
}

type MigrationLockError struct {
	LockName string
}

func (e *MigrationLockError) Error() string {
	return "migration lock not acquired (" + e.LockName + ")"
}

func RunServer(cfg config.Config) error {
	db, _, err := OpenDB(cfg)
	if err != nil {
		return err
	}

	repo := repository.New(db)
	store := storage.New(cfg.StorageRoot)
	if err := store.EnsureDirs(); err != nil {
		return err
	}
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return err
	}
	auth := middleware.NewInternalAuth(cfg.AppID, cfg.AppSecret, cfg.AllowedClockSkew, cfg.NonceTTL, rdb)
	svc := service.NewFileService(db, repo, store, cfg.NodeID)
	h := handler.NewFileHandler(svc, cfg.StorageRoot)
	locker := service.NewSchedulerLocker(rdb, cfg.SchedulerLockTTL, cfg.SchedulerLockRenew)

	go func() {
		locker.WithLock(context.Background(), "sched:lock:fileserver_cleanup_pending_deletes", svc.CleanupPendingDeletes)
		ticker := time.NewTicker(cfg.DeleteWorkerInterval)
		defer ticker.Stop()
		for range ticker.C {
			locker.WithLock(context.Background(), "sched:lock:fileserver_cleanup_pending_deletes", svc.CleanupPendingDeletes)
		}
	}()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.MaxUploadBytes)
		c.Next()
	})

	internal := r.Group("/internal/v1")
	internal.Use(auth.Middleware())
	{
		internal.GET("/health", h.Health)
		internal.GET("/space", h.Space)
		internal.POST("/files/check", h.Check)
		internal.POST("/files/upload", h.Upload)
		internal.GET("/files/tenant/usage", h.TenantUsage)
		internal.DELETE("/files/tenant", h.DeleteTenantFiles)
		internal.GET("/files/:file_id/download", h.Download)
		internal.GET("/files/:file_id/info", h.Info)
		internal.DELETE("/files/:file_id", h.Delete)
	}

	log.Printf("fileserver listening on %s", cfg.Addr)
	return r.Run(cfg.Addr)
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
