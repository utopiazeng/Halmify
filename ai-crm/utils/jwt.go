package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT声明
type JWTClaims struct {
	UserID      uint   `json:"user_id"`
	TenantID    uint   `json:"tenant_id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	RoleID      uint   `json:"role_id"`
	AccountType string `json:"account_type"`
	SessionID   string `json:"session_id"`
	jwt.RegisteredClaims
}

// JWTSecret JWT密钥
var JWTSecret = []byte("your-secret-key-change-in-production")

func init() {
	secret, _ := web.AppConfig.String("jwt_secret")
	if envSecret := os.Getenv("JWT_SECRET"); envSecret != "" {
		secret = envSecret
	}
	if secret != "" {
		JWTSecret = []byte(secret)
	}
}

// GenerateToken 生成JWT令牌
func GenerateToken(userID uint, username, role string) (string, error) {
	return GenerateTokenWithContext(userID, 0, username, role, 0, role)
}

// GenerateTokenWithContext 生成带租户和账号类型的JWT令牌
func GenerateTokenWithContext(userID, tenantID uint, username, role string, roleID uint, accountType string) (string, error) {
	return GenerateTokenWithSession(userID, tenantID, username, role, roleID, accountType, "")
}

// GenerateTokenWithSession 生成带租户、账号类型和会话ID的JWT令牌
func GenerateTokenWithSession(userID, tenantID uint, username, role string, roleID uint, accountType, sessionID string) (string, error) {
	claims := JWTClaims{
		UserID:      userID,
		TenantID:    tenantID,
		Username:    username,
		Role:        role,
		RoleID:      roleID,
		AccountType: accountType,
		SessionID:   sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ai-crm",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// NewSessionID 创建不可预测的会话ID
func NewSessionID() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// ParseToken 解析JWT令牌
func ParseToken(tokenString string) (*JWTClaims, error) {
	// 移除Bearer前缀
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return JWTSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
