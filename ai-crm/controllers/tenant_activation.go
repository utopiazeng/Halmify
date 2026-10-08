package controllers

import (
	"ai-crm/models"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
)

const (
	activationPurposeTenantAdminBootstrap = "tenant_admin_bootstrap"
	activationStatusPending               = "pending"
	activationStatusUsed                  = "used"
	activationStatusRevoked               = "revoked"
	defaultActivationTTL                  = 24 * time.Hour
)

func createTenantAdminActivationToken(tx *gorm.DB, tenantID, userID, createdBy uint, issuedIP string) (plainToken string, link string, expiresAt time.Time, err error) {
	token := randomToken(40)
	hash := sha256HexString(token)
	now := time.Now()
	expiresAt = now.Add(activationTokenTTL())

	if err = tx.Model(&models.TenantAdminActivationToken{}).
		Where("user_id = ? AND purpose = ? AND status = ?", userID, activationPurposeTenantAdminBootstrap, activationStatusPending).
		Updates(map[string]interface{}{"status": activationStatusRevoked, "revoked_at": &now}).Error; err != nil {
		return "", "", time.Time{}, err
	}

	record := models.TenantAdminActivationToken{
		TenantModel: models.TenantModel{TenantID: tenantID},
		UserID:      userID,
		TokenHash:   hash,
		Purpose:     activationPurposeTenantAdminBootstrap,
		Status:      activationStatusPending,
		ExpiresAt:   expiresAt,
		CreatedBy:   createdBy,
		IssuedIP:    strings.TrimSpace(issuedIP),
	}
	if err = tx.Create(&record).Error; err != nil {
		return "", "", time.Time{}, err
	}
	return token, buildActivationLink(token), expiresAt, nil
}

func consumeTenantAdminActivationToken(tx *gorm.DB, token string) (*models.TenantAdminActivationToken, error) {
	hash := sha256HexString(strings.TrimSpace(token))
	now := time.Now()
	record := &models.TenantAdminActivationToken{}
	if err := tx.Where("token_hash = ?", hash).First(record).Error; err != nil {
		return nil, err
	}
	if record.Status != activationStatusPending {
		return nil, fmt.Errorf("token status invalid")
	}
	if now.After(record.ExpiresAt) {
		_ = tx.Model(&models.TenantAdminActivationToken{}).Where("id = ?", record.ID).Updates(map[string]interface{}{
			"status":     activationStatusRevoked,
			"revoked_at": &now,
		}).Error
		return nil, fmt.Errorf("token expired")
	}
	if err := tx.Model(&models.TenantAdminActivationToken{}).Where("id = ?", record.ID).Updates(map[string]interface{}{
		"status":  activationStatusUsed,
		"used_at": &now,
	}).Error; err != nil {
		return nil, err
	}
	record.Status = activationStatusUsed
	record.UsedAt = &now
	return record, nil
}

func activationTokenTTL() time.Duration {
	hours := web.AppConfig.DefaultInt("tenant_admin_activation_ttl_hours", 24)
	if env := strings.TrimSpace(os.Getenv("TENANT_ADMIN_ACTIVATION_TTL_HOURS")); env != "" {
		if parsed, err := strconv.Atoi(env); err == nil && parsed > 0 {
			hours = parsed
		}
	}
	if hours <= 0 {
		return defaultActivationTTL
	}
	return time.Duration(hours) * time.Hour
}

func buildActivationLink(token string) string {
	base, _ := web.AppConfig.String("tenant_admin_activation_base_url")
	base = strings.TrimSpace(base)
	if env := strings.TrimSpace(os.Getenv("TENANT_ADMIN_ACTIVATION_BASE_URL")); env != "" {
		base = env
	}
	if base == "" {
		frontendBase, _ := web.AppConfig.String("frontend_base_url")
		frontendBase = strings.TrimSpace(frontendBase)
		if frontendBase == "" {
			frontendBase = "http://localhost:5173"
		}
		base = frontendBase
	}
	base = strings.TrimRight(base, "/")
	return base + "/activate-account?token=" + token
}

func randomToken(length int) string {
	if length < 24 {
		length = 24
	}
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("tok_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)[:length]
}

func sha256HexString(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func revokeTenantAdminSessions(tx *gorm.DB, userID uint) error {
	now := time.Now()
	return tx.Model(&models.UserSession{}).
		Where("user_id = ? AND revoked = ?", userID, false).
		Updates(map[string]interface{}{"revoked": true, "revoked_at": &now}).Error
}

func revokePendingActivationTokens(tx *gorm.DB, userID uint) error {
	now := time.Now()
	return tx.Model(&models.TenantAdminActivationToken{}).
		Where("user_id = ? AND purpose = ? AND status = ?", userID, activationPurposeTenantAdminBootstrap, activationStatusPending).
		Updates(map[string]interface{}{"status": activationStatusRevoked, "revoked_at": &now}).Error
}
