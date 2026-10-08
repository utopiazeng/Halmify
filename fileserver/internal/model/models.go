package model

import (
	"time"

	"gorm.io/gorm"
)

type Base struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type PhysicalFile struct {
	Base
	TenantID uint64 `gorm:"not null;uniqueIndex:idx_tenant_node_md5,priority:1" json:"tenant_id"`
	NodeID   string `gorm:"type:varchar(64);not null;default:'';uniqueIndex:idx_tenant_node_md5,priority:2" json:"node_id"`
	MD5      string `gorm:"type:char(32);not null;uniqueIndex:idx_tenant_node_md5,priority:3" json:"md5"`
	FileSize uint64 `gorm:"not null" json:"file_size"`
	FilePath string `gorm:"type:varchar(500);not null" json:"file_path"`
	RefCount uint64 `gorm:"not null;default:1" json:"ref_count"`
	Status   string `gorm:"type:varchar(32);not null;default:'active';index" json:"status"`
}

type FileRecord struct {
	Base
	TenantID      uint64     `gorm:"not null;index:idx_tenant_created,priority:1;index:idx_tenant_file,priority:1;index:idx_tenant_file_status,priority:1" json:"tenant_id"`
	NodeID        string     `gorm:"type:varchar(64);not null;default:'';index" json:"node_id"`
	FileID        string     `gorm:"type:varchar(64);not null;uniqueIndex;index:idx_tenant_file_status,priority:2" json:"file_id"`
	PhysicalID    uint64     `gorm:"not null;index:idx_tenant_file,priority:2" json:"physical_id"`
	OriginalName  string     `gorm:"type:varchar(255);not null" json:"original_name"`
	FileSize      uint64     `gorm:"not null" json:"file_size"`
	Uploader      string     `gorm:"type:varchar(64);default:''" json:"uploader"`
	Status        uint8      `gorm:"not null;default:1;index;index:idx_tenant_file_status,priority:3" json:"status"`
	DeletedAtTime *time.Time `json:"deleted_at,omitempty"`
}

type DeleteTask struct {
	Base
	TenantID     uint64     `gorm:"not null;index" json:"tenant_id"`
	NodeID       string     `gorm:"type:varchar(64);not null;default:'';index;index:idx_delete_tasks_node_status_next,priority:1" json:"node_id"`
	PhysicalID   uint64     `gorm:"not null;index" json:"physical_id"`
	FilePath     string     `gorm:"type:varchar(500);not null" json:"file_path"`
	RetryCount   int        `gorm:"not null;default:0" json:"retry_count"`
	Status       string     `gorm:"type:varchar(32);not null;default:'pending';index;index:idx_delete_tasks_status_next,priority:1;index:idx_delete_tasks_status_lock_expire,priority:1;index:idx_delete_tasks_node_status_next,priority:2" json:"status"`
	WorkerID     string     `gorm:"type:varchar(128);default:'';index" json:"worker_id"`
	LockedAt     *time.Time `json:"locked_at,omitempty"`
	LockExpireAt *time.Time `gorm:"index;index:idx_delete_tasks_status_lock_expire,priority:2" json:"lock_expire_at,omitempty"`
	LastError    string     `gorm:"type:text" json:"last_error"`
	NextRunAt    time.Time  `gorm:"index;index:idx_delete_tasks_status_next,priority:2;index:idx_delete_tasks_node_status_next,priority:3" json:"next_run_at"`
}
