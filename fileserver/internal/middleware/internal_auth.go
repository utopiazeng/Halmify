package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type InternalAuth struct {
	AppID            string
	AppSecret        string
	AllowedClockSkew time.Duration
	NonceTTL         time.Duration
	Redis            *redis.Client
}

func NewInternalAuth(appID, appSecret string, skew, nonceTTL time.Duration, rdb *redis.Client) *InternalAuth {
	return &InternalAuth{AppID: appID, AppSecret: appSecret, AllowedClockSkew: skew, NonceTTL: nonceTTL, Redis: rdb}
}

func (m *InternalAuth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		appID := c.GetHeader("X-App-Id")
		timestamp := c.GetHeader("X-Timestamp")
		nonce := c.GetHeader("X-Nonce")
		signature := c.GetHeader("X-Signature")
		tenantID := c.GetHeader("X-Tenant-ID")
		if appID != m.AppID || timestamp == "" || nonce == "" || signature == "" || tenantID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid internal auth", "data": nil})
			c.Abort()
			return
		}
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid timestamp", "data": nil})
			c.Abort()
			return
		}
		now := time.Now()
		if now.Sub(time.Unix(ts, 0)) > m.AllowedClockSkew || time.Unix(ts, 0).Sub(now) > m.AllowedClockSkew {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "timestamp expired", "data": nil})
			c.Abort()
			return
		}
		nonceKey := fmt.Sprintf("fs_nonce:%s:%s:%s", appID, tenantID, nonce)
		ok, err := m.Redis.SetNX(c, nonceKey, "1", m.NonceTTL).Result()
		if err != nil || !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "replay attack detected", "data": nil})
			c.Abort()
			return
		}
		bodyHash := sha256Hex(nil)
		if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
			bodyRaw, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(bodyRaw))
			bodyHash = sha256Hex(bodyRaw)
		}
		canonical := strings.Join([]string{
			c.Request.Method,
			c.Request.URL.Path,
			c.Request.URL.RawQuery,
			bodyHash,
			timestamp,
			nonce,
			tenantID,
		}, "\n")
		expected := hmacHex(m.AppSecret, canonical)
		if !hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(signature))) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid signature", "data": nil})
			c.Abort()
			return
		}
		tenantNum, _ := strconv.ParseUint(tenantID, 10, 64)
		c.Set("tenant_id", tenantNum)
		c.Next()
	}
}

func sha256Hex(body []byte) string {
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}

func hmacHex(secret, raw string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(raw))
	return hex.EncodeToString(mac.Sum(nil))
}
