package models

import "time"

// IMConversation 单聊会话
type IMConversation struct {
	BaseModel
	TenantModel
	UserLow             uint       `gorm:"not null;index:idx_im_conv_users,unique;index" json:"user_low"`
	UserHigh            uint       `gorm:"not null;index:idx_im_conv_users,unique;index" json:"user_high"`
	ConversationType    string     `gorm:"size:32;not null;default:direct;index" json:"conversation_type"`
	Category            string     `gorm:"size:64;index" json:"category"`
	Priority            string     `gorm:"size:32;index" json:"priority"`
	Title               string     `gorm:"size:255" json:"title"`
	PayloadJSON         string     `gorm:"type:longtext" json:"payload_json"`
	LastMessageID       uint       `gorm:"index" json:"last_message_id"`
	LastMessageSenderID uint       `gorm:"index" json:"last_message_sender_id"`
	LastMessageType     string     `gorm:"size:16;default:text" json:"last_message_type"`
	LastMessagePreview  string     `gorm:"size:255" json:"last_message_preview"`
	LastMessageAt       *time.Time `gorm:"index" json:"last_message_at"`
}

// IMMessage 会话消息
type IMMessage struct {
	BaseModel
	TenantModel
	ConversationID uint      `gorm:"not null;index:idx_im_msg_conv_id,priority:1;index" json:"conversation_id"`
	SenderID       uint      `gorm:"not null;index" json:"sender_id"`
	RecipientID    uint      `gorm:"not null;index:idx_im_msg_recipient_created,priority:1;index" json:"recipient_id"`
	Type           string    `gorm:"size:16;not null;default:text;index" json:"type"`
	Content        string    `gorm:"type:longtext" json:"content"`
	PayloadJSON    string    `gorm:"type:longtext" json:"payload_json"`
	Status         string    `gorm:"size:32;not null;default:sent;index" json:"status"`
	ClientMsgID    string    `gorm:"size:64;index" json:"client_msg_id"`
	ExpiresAt      time.Time `gorm:"not null;index" json:"expires_at"`
}

// IMMessageReceipt 消息回执
type IMMessageReceipt struct {
	BaseModel
	TenantModel
	MessageID      uint       `gorm:"not null;index:idx_im_receipt_msg_recipient,priority:1;index" json:"message_id"`
	ConversationID uint       `gorm:"not null;index:idx_im_receipt_conv_recipient,priority:1;index" json:"conversation_id"`
	RecipientID    uint       `gorm:"not null;index:idx_im_receipt_msg_recipient,priority:2;index:idx_im_receipt_conv_recipient,priority:2;index:idx_im_receipt_unread,priority:1" json:"recipient_id"`
	DeliveredAt    *time.Time `gorm:"index" json:"delivered_at"`
	ReadAt         *time.Time `gorm:"index:idx_im_receipt_unread,priority:2;index" json:"read_at"`
}

// IMUserPresence 用户在线状态
type IMUserPresence struct {
	BaseModel
	TenantModel
	UserID       uint       `gorm:"not null;index:idx_im_presence_user" json:"user_id"`
	Status       string     `gorm:"size:16;not null;default:offline;index" json:"status"`
	ManualStatus bool       `gorm:"not null;default:false" json:"manual_status"`
	LastActiveAt *time.Time `gorm:"index" json:"last_active_at"`
	LastSeenAt   *time.Time `gorm:"index" json:"last_seen_at"`
}

// IMUserSetting 用户通知设置
type IMUserSetting struct {
	BaseModel
	TenantModel
	UserID              uint `gorm:"not null;index:idx_im_user_setting_user" json:"user_id"`
	SoundEnabled        bool `gorm:"not null;default:true" json:"sound_enabled"`
	BrowserNotifyEnable bool `gorm:"not null;default:true" json:"browser_notify_enable"`
}

// IMSessionSetting 会话设置
type IMSessionSetting struct {
	BaseModel
	TenantModel
	ConversationID uint       `gorm:"not null;index:idx_im_session_setting_user_conv,priority:2" json:"conversation_id"`
	UserID         uint       `gorm:"not null;index:idx_im_session_setting_user_conv,priority:1;index" json:"user_id"`
	Mute           bool       `gorm:"not null;default:false" json:"mute"`
	Pinned         bool       `gorm:"not null;default:false" json:"pinned"`
	PinnedAt       *time.Time `gorm:"index" json:"pinned_at"`
}

// IMCleanupJob 清理任务记录
type IMCleanupJob struct {
	BaseModel
	Status          string     `gorm:"size:16;not null;default:done;index" json:"status"`
	StartedAt       time.Time  `gorm:"index" json:"started_at"`
	FinishedAt      *time.Time `gorm:"index" json:"finished_at"`
	DeletedMessages int64      `gorm:"not null;default:0" json:"deleted_messages"`
	DeletedReceipts int64      `gorm:"not null;default:0" json:"deleted_receipts"`
	Reason          string     `gorm:"size:255" json:"reason"`
}

// IoTDevice 租户 IoT 展示设备
type IoTDevice struct {
	BaseModel
	TenantModel
	Name                  string     `gorm:"size:128;not null" json:"name"`
	SN                    string     `gorm:"size:128;uniqueIndex;not null" json:"sn"`
	SecretHash            string     `gorm:"size:255;not null" json:"-"`
	PendingSecretHash     string     `gorm:"size:255" json:"-"`
	PublicKeyPEM          string     `gorm:"type:text" json:"-"`
	PublicKeyFingerprint  string     `gorm:"size:128;index" json:"public_key_fingerprint"`
	FirmwareVersion       string     `gorm:"size:64" json:"firmware_version"`
	Model                 string     `gorm:"size:64;index" json:"model"`
	TargetFirmwareVersion string     `gorm:"size:64;index" json:"target_firmware_version"`
	OTAStatus             string     `gorm:"size:32;index" json:"ota_status"`
	OTAFailureReason      string     `gorm:"type:text" json:"ota_failure_reason"`
	LastOTACheckAt        *time.Time `gorm:"index" json:"last_ota_check_at"`
	LastOTADownloadAt     *time.Time `gorm:"index" json:"last_ota_download_at"`
	LastOTAConfirmAt      *time.Time `gorm:"index" json:"last_ota_confirm_at"`
	Status                string     `gorm:"size:32;not null;default:active;index" json:"status"`
	ResolutionWidth       int        `gorm:"not null;default:200" json:"resolution_width"`
	ResolutionHeight      int        `gorm:"not null;default:200" json:"resolution_height"`
	BoundAt               *time.Time `gorm:"index" json:"bound_at"`
	LastBootstrapAt       *time.Time `gorm:"index" json:"last_bootstrap_at"`
	LastSeenAt            *time.Time `gorm:"index" json:"last_seen_at"`
	LastSyncAt            *time.Time `gorm:"index" json:"last_sync_at"`
	LastIP                string     `gorm:"size:64" json:"last_ip"`
	LastUserAgent         string     `gorm:"type:text" json:"last_user_agent"`
	Remark                string     `gorm:"type:text" json:"remark"`
}

func (IoTDevice) TableName() string {
	return "iot_devices"
}

// IoTFirmwareRelease 平台 IoT 固件发布记录
type IoTFirmwareRelease struct {
	BaseModel
	Version        string     `gorm:"size:64;not null;index:idx_iot_fw_model_version,priority:2" json:"version"`
	Model          string     `gorm:"size:64;not null;default:esp32-s3-epaper;index:idx_iot_fw_model_version,priority:1;index" json:"model"`
	FileID         string     `gorm:"size:128;not null" json:"file_id"`
	StorageNodeID  string     `gorm:"size:64" json:"storage_node_id"`
	FileName       string     `gorm:"size:255" json:"file_name"`
	FileSize       int64      `gorm:"not null;default:0" json:"file_size"`
	SHA256         string     `gorm:"size:64;not null;index" json:"sha256"`
	Signature      string     `gorm:"type:text;not null" json:"signature"`
	Status         string     `gorm:"size:32;not null;default:draft;index" json:"status"`
	Force          bool       `gorm:"not null;default:false;index" json:"force"`
	RolloutPercent int        `gorm:"not null;default:100" json:"rollout_percent"`
	CreatedBy      uint       `gorm:"index" json:"created_by"`
	PublishedAt    *time.Time `gorm:"index" json:"published_at"`
	RevokedAt      *time.Time `gorm:"index" json:"revoked_at"`
	Remark         string     `gorm:"type:text" json:"remark"`
}

func (IoTFirmwareRelease) TableName() string {
	return "iot_firmware_releases"
}

// IoTFirmwareTarget 平台固件逐设备发布记录
type IoTFirmwareTarget struct {
	BaseModel
	ReleaseID       uint       `gorm:"not null;uniqueIndex:idx_iot_fw_target_release_device,priority:1;index" json:"release_id"`
	DeviceID        uint       `gorm:"not null;uniqueIndex:idx_iot_fw_target_release_device,priority:2;index" json:"device_id"`
	TenantID        uint       `gorm:"not null;default:0;index" json:"tenant_id"`
	SN              string     `gorm:"size:128;not null;index" json:"sn"`
	Model           string     `gorm:"size:64;not null;index" json:"model"`
	Version         string     `gorm:"size:64;not null;index" json:"version"`
	Status          string     `gorm:"size:32;not null;default:assigned;index" json:"status"`
	Force           bool       `gorm:"not null;default:false;index" json:"force"`
	FailureReason   string     `gorm:"type:text" json:"failure_reason"`
	AssignedAt      time.Time  `gorm:"not null;index" json:"assigned_at"`
	CheckedAt       *time.Time `gorm:"index" json:"checked_at"`
	DownloadedAt    *time.Time `gorm:"index" json:"downloaded_at"`
	ConfirmedAt     *time.Time `gorm:"index" json:"confirmed_at"`
	LastSeenAt       *time.Time `gorm:"index" json:"last_seen_at"`
	DeviceName      string     `gorm:"-" json:"device_name,omitempty"`
	CurrentVersion  string     `gorm:"-" json:"current_version,omitempty"`
	TenantName      string     `gorm:"-" json:"tenant_name,omitempty"`
	LastDeviceSeenAt *time.Time `gorm:"-" json:"last_device_seen_at,omitempty"`
}

func (IoTFirmwareTarget) TableName() string {
	return "iot_firmware_targets"
}
