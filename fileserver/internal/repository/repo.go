package repository

import (
	"fileserver/internal/model"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repo struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Repo { return &Repo{DB: db} }

func (r *Repo) AutoMigrate() error {
	if err := r.dropLegacyPhysicalMD5UniqueIndex(); err != nil {
		return err
	}
	return r.DB.AutoMigrate(&model.PhysicalFile{}, &model.FileRecord{}, &model.DeleteTask{})
}

func (r *Repo) dropLegacyPhysicalMD5UniqueIndex() error {
	if !r.DB.Migrator().HasTable(&model.PhysicalFile{}) {
		return nil
	}
	var indexes []struct {
		Name string `gorm:"column:INDEX_NAME"`
	}
	err := r.DB.Raw(`
		SELECT INDEX_NAME
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'physical_files'
		  AND NON_UNIQUE = 0
		GROUP BY INDEX_NAME
		HAVING SUM(COLUMN_NAME = 'md5') > 0
		   AND SUM(COLUMN_NAME = 'node_id') = 0
	`).Scan(&indexes).Error
	if err != nil {
		return err
	}
	for _, index := range indexes {
		name := strings.TrimSpace(index.Name)
		if name == "" || strings.EqualFold(name, "PRIMARY") {
			continue
		}
		escaped := strings.ReplaceAll(name, "`", "``")
		if err := r.DB.Exec(fmt.Sprintf("DROP INDEX `%s` ON `physical_files`", escaped)).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) FindActiveRecord(tenantID uint64, nodeID string, fileID string) (model.FileRecord, error) {
	var rec model.FileRecord
	err := r.DB.Where("tenant_id = ? AND node_id = ? AND file_id = ? AND status = 1", tenantID, nodeID, fileID).First(&rec).Error
	return rec, err
}

func (r *Repo) FindPhysicalByTenantNodeMD5(tenantID uint64, nodeID string, md5Hex string) (model.PhysicalFile, error) {
	var pf model.PhysicalFile
	err := r.DB.Where("tenant_id = ? AND node_id = ? AND md5 = ?", tenantID, nodeID, md5Hex).First(&pf).Error
	return pf, err
}

func (r *Repo) FindPhysicalByID(id uint64) (model.PhysicalFile, error) {
	var pf model.PhysicalFile
	err := r.DB.First(&pf, id).Error
	return pf, err
}

func (r *Repo) CreateDeleteTask(tx *gorm.DB, task model.DeleteTask) error {
	return tx.Create(&task).Error
}

func (r *Repo) RequeueExpiredRunningTasks(nodeID string, now time.Time) error {
	return r.DB.Model(&model.DeleteTask{}).
		Where("node_id = ? AND status = ? AND lock_expire_at IS NOT NULL AND lock_expire_at <= ?", nodeID, "running", now).
		Updates(map[string]interface{}{
			"status":         "pending",
			"worker_id":      "",
			"locked_at":      nil,
			"lock_expire_at": nil,
			"next_run_at":    now,
		}).Error
}

func (r *Repo) ClaimPendingDeleteTasks(nodeID string, workerID string, limit int, lockTTL time.Duration) ([]model.DeleteTask, error) {
	now := time.Now()
	if err := r.RequeueExpiredRunningTasks(nodeID, now); err != nil {
		return nil, err
	}

	var claimed []model.DeleteTask
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var pendingIDs []uint64
		if err := tx.
			Model(&model.DeleteTask{}).
			Where("node_id = ? AND status = ? AND next_run_at <= ?", nodeID, "pending", now).
			Order("id asc").
			Limit(limit).
			Pluck("id", &pendingIDs).Error; err != nil {
			return err
		}
		if len(pendingIDs) == 0 {
			claimed = []model.DeleteTask{}
			return nil
		}
		lockExpireAt := now.Add(lockTTL)
		if err := tx.Model(&model.DeleteTask{}).
			Where("node_id = ? AND id IN ? AND status = ?", nodeID, pendingIDs, "pending").
			Updates(map[string]interface{}{
				"status":         "running",
				"worker_id":      workerID,
				"locked_at":      now,
				"lock_expire_at": lockExpireAt,
			}).Error; err != nil {
			return err
		}
		return tx.Where("node_id = ? AND id IN ? AND status = ? AND worker_id = ?", nodeID, pendingIDs, "running", workerID).Order("id asc").Find(&claimed).Error
	})
	return claimed, err
}

func (r *Repo) MarkDeleteTaskDone(id uint64) error {
	return r.DB.Model(&model.DeleteTask{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":         "done",
		"last_error":     "",
		"worker_id":      "",
		"locked_at":      nil,
		"lock_expire_at": nil,
	}).Error
}

func (r *Repo) RetryDeleteTask(id uint64, errMsg string, retryCount int) error {
	next := time.Now().Add(time.Duration(retryCount*retryCount) * time.Second)
	return r.DB.Model(&model.DeleteTask{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":         "pending",
		"worker_id":      "",
		"locked_at":      nil,
		"lock_expire_at": nil,
		"last_error":     errMsg,
		"retry_count":    retryCount,
		"next_run_at":    next,
	}).Error
}

func (r *Repo) RemovePhysicalAfterDelete(tenantID uint64, nodeID string, physicalID uint64) error {
	return r.DB.Unscoped().
		Where("tenant_id = ? AND node_id = ? AND id = ? AND status = ? AND ref_count = 0", tenantID, nodeID, physicalID, "pending_delete").
		Delete(&model.PhysicalFile{}).Error
}
