package service

import (
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"fileserver/internal/model"
	"fileserver/internal/repository"
	"fileserver/internal/storage"

	"github.com/segmentio/ksuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FileService struct {
	repo   *repository.Repo
	store  *storage.Store
	db     *gorm.DB
	nodeID string
}

func NewFileService(db *gorm.DB, repo *repository.Repo, store *storage.Store, nodeID string) *FileService {
	return &FileService{db: db, repo: repo, store: store, nodeID: strings.TrimSpace(nodeID)}
}

func (s *FileService) scopedPhysicalLookup(tenantID uint64, md5Hex string) (uint64, string, string) {
	return tenantID, s.nodeID, strings.ToLower(strings.TrimSpace(md5Hex))
}

func (s *FileService) newDeleteTask(tenantID uint64, physicalID uint64, filePath string) model.DeleteTask {
	return model.DeleteTask{
		TenantID:   tenantID,
		NodeID:     s.nodeID,
		PhysicalID: physicalID,
		FilePath:   filePath,
		Status:     "pending",
		NextRunAt:  time.Now(),
	}
}

func (s *FileService) NodeInfo() (map[string]interface{}, error) {
	if s.nodeID == "" {
		return nil, errors.New("missing fileserver node_id")
	}
	return map[string]interface{}{"node_id": s.nodeID}, nil
}

func (s *FileService) Check(tenantID uint64, md5Hex string, fileSize uint64) (bool, string, error) {
	rec := model.FileRecord{}
	err := s.db.
		Joins("JOIN physical_files ON physical_files.id = file_records.physical_id").
		Where("file_records.tenant_id = ? AND file_records.node_id = ? AND file_records.status = 1 AND physical_files.node_id = ? AND physical_files.md5 = ? AND physical_files.file_size = ?", tenantID, s.nodeID, s.nodeID, strings.ToLower(md5Hex), fileSize).
		Order("file_records.created_at desc").
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, rec.FileID, nil
}

func (s *FileService) Upload(tenantID uint64, uploader string, originalName string, reader multipart.File) (map[string]interface{}, error) {
	md5Hex, fileSize, tempPath, err := s.store.ComputeMD5AndWriteTemp(reader)
	if err != nil {
		return nil, err
	}
	finalRel := s.store.FinalPath(tenantID, md5Hex)
	result := map[string]interface{}{}
	shouldStoreFinal := false
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		insert := model.PhysicalFile{
			TenantID: tenantID,
			NodeID:   s.nodeID,
			MD5:      md5Hex,
			FileSize: fileSize,
			FilePath: finalRel,
			RefCount: 1,
			Status:   "active",
		}

		physicalID := uint64(0)
		createResult := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "node_id"}, {Name: "md5"}},
			DoNothing: true,
		}).Create(&insert)
		if createResult.Error != nil {
			return createResult.Error
		}
		if createResult.RowsAffected > 0 {
			shouldStoreFinal = true
			physicalID = insert.ID
		} else {
			scopedTenantID, scopedNodeID, scopedMD5 := s.scopedPhysicalLookup(tenantID, md5Hex)
			existing, qerr := s.findPhysicalForUpload(tx, scopedTenantID, scopedNodeID, scopedMD5)
			if qerr != nil {
				return fmt.Errorf("physical file duplicate index is inconsistent for tenant=%d node=%s md5=%s: %w", scopedTenantID, scopedNodeID, scopedMD5, qerr)
			}
			if s.shouldRevivePhysical(existing) {
				if err := tx.Unscoped().Model(&model.PhysicalFile{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
					"deleted_at": nil,
					"file_size":  fileSize,
					"file_path":  finalRel,
					"ref_count":  1,
					"status":     "active",
				}).Error; err != nil {
					return err
				}
				if err := tx.Model(&model.DeleteTask{}).Where("tenant_id = ? AND node_id = ? AND physical_id = ? AND status IN ?", tenantID, s.nodeID, existing.ID, []string{"pending", "running"}).Updates(map[string]interface{}{
					"status":         "done",
					"last_error":     "",
					"worker_id":      "",
					"locked_at":      nil,
					"lock_expire_at": nil,
				}).Error; err != nil {
					return err
				}
				shouldStoreFinal = true
			} else {
				if err := tx.Model(&model.PhysicalFile{}).Where("id = ?", existing.ID).Update("ref_count", gorm.Expr("ref_count + 1")).Error; err != nil {
					return err
				}
			}
			physicalID = existing.ID
		}

		rec := model.FileRecord{
			TenantID:     tenantID,
			NodeID:       s.nodeID,
			FileID:       ksuid.New().String(),
			PhysicalID:   physicalID,
			OriginalName: originalName,
			FileSize:     fileSize,
			Uploader:     uploader,
			Status:       1,
		}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}

		result = map[string]interface{}{
			"file_id":         rec.FileID,
			"original_name":   rec.OriginalName,
			"size":            rec.FileSize,
			"md5":             md5Hex,
			"storage_node_id": s.nodeID,
		}
		return nil
	}); err != nil {
		_ = os.Remove(tempPath)
		return nil, err
	}
	if shouldStoreFinal {
		if err := s.store.MoveToFinal(tempPath, finalRel); err != nil {
			_ = s.markPhysicalFileUnavailable(tenantID, md5Hex, err.Error())
			return nil, err
		}
	} else {
		_ = os.Remove(tempPath)
	}
	return result, nil
}

func (s *FileService) findPhysicalForUpload(tx *gorm.DB, tenantID uint64, nodeID string, md5Hex string) (model.PhysicalFile, error) {
	var pf model.PhysicalFile
	err := tx.Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_id = ? AND node_id = ? AND md5 = ?", tenantID, nodeID, md5Hex).
		First(&pf).Error
	return pf, err
}

func (s *FileService) shouldRevivePhysical(pf model.PhysicalFile) bool {
	return pf.DeletedAt.Valid || pf.Status != "active" || pf.RefCount == 0
}

func (s *FileService) markPhysicalFileUnavailable(tenantID uint64, md5Hex string, reason string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var pf model.PhysicalFile
		if err := tx.Where("tenant_id = ? AND node_id = ? AND md5 = ?", tenantID, s.nodeID, md5Hex).First(&pf).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.FileRecord{}).Where("tenant_id = ? AND physical_id = ? AND status = 1", tenantID, pf.ID).Update("status", 3).Error; err != nil {
			return err
		}
		return tx.Model(&model.PhysicalFile{}).Where("id = ?", pf.ID).Updates(map[string]interface{}{"status": "unavailable", "ref_count": 0}).Error
	})
}

func (s *FileService) Info(tenantID uint64, fileID string) (map[string]interface{}, error) {
	rec, err := s.repo.FindActiveRecord(tenantID, s.nodeID, fileID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"file_id":         rec.FileID,
		"original_name":   rec.OriginalName,
		"size":            rec.FileSize,
		"uploader":        rec.Uploader,
		"created_at":      rec.CreatedAt,
		"storage_node_id": rec.NodeID,
	}, nil
}

func (s *FileService) Delete(tenantID uint64, fileID string) error {
	var deletePath string
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		var rec model.FileRecord
		if err := tx.Where("tenant_id = ? AND node_id = ? AND file_id = ? AND status = 1", tenantID, s.nodeID, fileID).Clauses(clause.Locking{Strength: "UPDATE"}).First(&rec).Error; err != nil {
			return err
		}
		var pf model.PhysicalFile
		if err := tx.Where("id = ? AND tenant_id = ? AND node_id = ?", rec.PhysicalID, tenantID, s.nodeID).Clauses(clause.Locking{Strength: "UPDATE"}).First(&pf).Error; err != nil {
			return err
		}
		lastReference := pf.RefCount <= 1
		if lastReference {
			if err := s.store.Delete(pf.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			deletePath = pf.FilePath
		}
		now := time.Now()
		if err := tx.Model(&model.FileRecord{}).Where("id = ?", rec.ID).Updates(map[string]interface{}{"status": 2, "deleted_at_time": &now}).Error; err != nil {
			return err
		}
		if lastReference {
			if err := tx.Model(&model.PhysicalFile{}).Where("id = ?", pf.ID).Updates(map[string]interface{}{"ref_count": 0, "status": "pending_delete"}).Error; err != nil {
				return err
			}
			return s.repo.CreateDeleteTask(tx, s.newDeleteTask(tenantID, pf.ID, pf.FilePath))
		}
		return tx.Model(&model.PhysicalFile{}).Where("id = ? AND ref_count > 0", pf.ID).Update("ref_count", gorm.Expr("ref_count - 1")).Error
	}); err != nil {
		return err
	}
	if deletePath != "" {
		s.store.CleanupEmptyParents(deletePath)
	}
	return nil
}

type TenantUsage struct {
	UsedBytes     uint64 `json:"used_bytes"`
	PhysicalCount int64  `json:"physical_count"`
	RecordCount   int64  `json:"record_count"`
}

type TenantDeleteResult struct {
	DeletedBytes         uint64 `json:"deleted_bytes"`
	DeletedPhysicalCount int64  `json:"deleted_physical_count"`
	DeletedRecordCount   int64  `json:"deleted_record_count"`
}

func (s *FileService) TenantUsage(tenantID uint64) (TenantUsage, error) {
	var row struct {
		UsedBytes     uint64
		PhysicalCount int64
	}
	if err := s.db.Model(&model.PhysicalFile{}).
		Select("COALESCE(SUM(file_size), 0) AS used_bytes, COUNT(*) AS physical_count").
		Where("tenant_id = ? AND node_id = ? AND status = ?", tenantID, s.nodeID, "active").
		Scan(&row).Error; err != nil {
		return TenantUsage{}, err
	}
	var recordCount int64
	if err := s.db.Model(&model.FileRecord{}).
		Where("tenant_id = ? AND node_id = ? AND status = ?", tenantID, s.nodeID, 1).
		Count(&recordCount).Error; err != nil {
		return TenantUsage{}, err
	}
	return TenantUsage{UsedBytes: row.UsedBytes, PhysicalCount: row.PhysicalCount, RecordCount: recordCount}, nil
}

func (s *FileService) DeleteTenantFiles(tenantID uint64) (TenantDeleteResult, error) {
	var physicals []model.PhysicalFile
	if err := s.db.Unscoped().
		Where("tenant_id = ? AND node_id = ?", tenantID, s.nodeID).
		Find(&physicals).Error; err != nil {
		return TenantDeleteResult{}, err
	}

	result := TenantDeleteResult{DeletedPhysicalCount: int64(len(physicals))}
	for _, pf := range physicals {
		result.DeletedBytes += pf.FileSize
		if err := s.store.Delete(pf.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return result, err
		}
	}

	now := time.Now()
	activeRecordCount := int64(0)
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		recordResult := tx.Model(&model.FileRecord{}).
			Where("tenant_id = ? AND node_id = ? AND status = ?", tenantID, s.nodeID, 1).
			Updates(map[string]interface{}{"status": 2, "deleted_at_time": &now})
		if recordResult.Error != nil {
			return recordResult.Error
		}
		activeRecordCount = recordResult.RowsAffected
		if err := tx.Unscoped().
			Where("tenant_id = ? AND node_id = ?", tenantID, s.nodeID).
			Delete(&model.PhysicalFile{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().
			Where("tenant_id = ? AND node_id = ?", tenantID, s.nodeID).
			Delete(&model.DeleteTask{}).Error
	}); err != nil {
		return result, err
	}

	result.DeletedRecordCount = activeRecordCount
	for _, pf := range physicals {
		s.store.CleanupEmptyParents(pf.FilePath)
	}
	return result, nil
}

func (s *FileService) FilePathForDownload(tenantID uint64, fileID string) (string, string, error) {
	rec, err := s.repo.FindActiveRecord(tenantID, s.nodeID, fileID)
	if err != nil {
		return "", "", err
	}
	pf, err := s.repo.FindPhysicalByID(rec.PhysicalID)
	if err != nil {
		return "", "", err
	}
	return pf.FilePath, rec.OriginalName, nil
}

func (s *FileService) CleanupPendingDeletes() {
	workerID := fmt.Sprintf("fileserver-%s-%d", s.nodeID, os.Getpid())
	lockTTL := 60 * time.Second
	tasks, err := s.repo.ClaimPendingDeleteTasks(s.nodeID, workerID, 100, lockTTL)
	if err != nil {
		return
	}
	for _, task := range tasks {
		var pf model.PhysicalFile
		if err := s.db.Where("tenant_id = ? AND node_id = ? AND id = ? AND status = ? AND ref_count = 0", task.TenantID, s.nodeID, task.PhysicalID, "pending_delete").First(&pf).Error; err != nil {
			_ = s.repo.MarkDeleteTaskDone(task.ID)
			continue
		}
		if err := s.store.Delete(task.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			_ = s.repo.RetryDeleteTask(task.ID, err.Error(), task.RetryCount+1)
			continue
		}
		_ = s.repo.RemovePhysicalAfterDelete(task.TenantID, s.nodeID, task.PhysicalID)
		_ = s.repo.MarkDeleteTaskDone(task.ID)
		s.store.CleanupEmptyParents(task.FilePath)
	}
}

func isDup(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "1062")
}
