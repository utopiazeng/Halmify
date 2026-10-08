package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
)

// AuthController 认证控制器
type AuthController struct {
	web.Controller
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// Register 用户注册
// @router /api/v1/auth/register [post]
func (c *AuthController) Register() {
	var req RegisterRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}

	var userCount int64
	utils.DB.Model(&models.User{}).Count(&userCount)
	if userCount > 0 {
		utils.Error(c.Ctx, http.StatusForbidden, "系统已初始化，请由超管或企业管理员创建账号")
		return
	}

	// 检查用户名是否已存在
	var existingUser models.User
	if err := utils.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		utils.Error(c.Ctx, http.StatusBadRequest, "用户名已存在")
		return
	}

	// 创建新用户
	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		Role:         "super_admin",
		AccountType:  "super_admin",
		Status:       "active",
		LanguagePref: "zh-CN",
	}

	// 设置密码
	if err := user.SetPassword(req.Password); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "密码加密失败")
		return
	}

	if err := utils.DB.Create(&user).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "用户创建失败")
		return
	}

	utils.Success(c.Ctx, map[string]interface{}{
		"message": "注册成功",
		"user_id": user.ID,
	})
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	MFACode      string `json:"mfa_code"`
	CaptchaToken string `json:"captcha_token"`
}

type ActivatePasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// Login 用户登录
// @router /api/v1/auth/login [post]
func (c *AuthController) Login() {
	loginConfigOnce.Do(loadLoginSecurityConfig)
	var req LoginRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)

	now := time.Now()
	ip := c.Ctx.Input.IP()

	statusCode, reason, allowed := authSecurity.checkLoginAllowed(ip, req.Username, now)
	if !allowed {
		services.LogOperation(services.OperationLogEvent{
			TenantID:       0,
			Username:       req.Username,
			Operation:      "LOGIN_RATE_LIMIT",
			Target:         "认证防护",
			DescriptionKey: "operation.login",
			Params: map[string]interface{}{
				"username": req.Username,
			},
			Result:    "failed",
			Reason:    reason,
			IP:        ip,
			UserAgent: c.Ctx.Request.UserAgent(),
		})
		utils.Error(c.Ctx, statusCode, reason)
		return
	}

	challenge, code, message := validateCaptchaToken(req.CaptchaToken, ip, c.Ctx.Request.UserAgent(), now)
	if challenge == nil {
		services.LogOperation(services.OperationLogEvent{
			TenantID:       0,
			Username:       req.Username,
			Operation:      "LOGIN_CAPTCHA",
			Target:         "滑块验证码",
			DescriptionKey: "operation.login",
			Params: map[string]interface{}{
				"username": req.Username,
			},
			Result:    "failed",
			Reason:    message,
			IP:        ip,
			UserAgent: c.Ctx.Request.UserAgent(),
		})
		utils.Error(c.Ctx, code, message)
		return
	}

	// 查找用户
	var user models.User
	if err := utils.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		consumeCaptchaChallenge(challenge.ID)
		authSecurity.markLoginFailure(ip, req.Username, now)
		time.Sleep(loginFailureResponseDelay)
		services.LogOperation(services.OperationLogEvent{
			TenantID:       0,
			Username:       req.Username,
			Operation:      "LOGIN",
			Target:         "用户[0]",
			DescriptionKey: "operation.login",
			Params:         map[string]interface{}{"username": req.Username},
			Result:         "failed",
			Reason:         "用户名或密码错误",
			IP:             ip,
			UserAgent:      c.Ctx.Request.UserAgent(),
		})
		utils.Error(c.Ctx, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	// 验证密码
	if !user.CheckPassword(req.Password) {
		consumeCaptchaChallenge(challenge.ID)
		authSecurity.markLoginFailure(ip, req.Username, now)
		time.Sleep(loginFailureResponseDelay)
		services.LogOperation(services.OperationLogEvent{
			TenantID:       user.TenantID,
			UserID:         user.ID,
			Username:       user.Username,
			Operation:      "LOGIN",
			Target:         fmt.Sprintf("用户[%d]", user.ID),
			DescriptionKey: "operation.login",
			Params:         map[string]interface{}{"user_id": user.ID},
			Result:         "failed",
			Reason:         "用户名或密码错误",
			IP:             ip,
			UserAgent:      c.Ctx.Request.UserAgent(),
		})
		utils.Error(c.Ctx, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	// 检查用户状态
	if user.Status != "active" {
		consumeCaptchaChallenge(challenge.ID)
		services.LogOperation(services.OperationLogEvent{
			TenantID:       user.TenantID,
			UserID:         user.ID,
			Username:       user.Username,
			Operation:      "LOGIN",
			Target:         "用户[" + user.Username + "]",
			DescriptionKey: "operation.login",
			Result:         "failed",
			Reason:         "用户账号已被禁用",
			IP:             ip,
			UserAgent:      c.Ctx.Request.UserAgent(),
		})
		utils.Error(c.Ctx, http.StatusForbidden, "用户账号已被禁用")
		return
	}
	if user.TenantID > 0 {
		var tenant models.Tenant
		if err := utils.DB.First(&tenant, user.TenantID).Error; err == nil && tenant.Status != "active" {
			consumeCaptchaChallenge(challenge.ID)
			services.LogOperation(services.OperationLogEvent{
				TenantID:       user.TenantID,
				UserID:         user.ID,
				Username:       user.Username,
				Operation:      "LOGIN",
				Target:         "租户[" + tenant.Name + "]",
				DescriptionKey: "operation.login",
				Result:         "failed",
				Reason:         "企业租户已停用",
				IP:             ip,
				UserAgent:      c.Ctx.Request.UserAgent(),
			})
			utils.Error(c.Ctx, http.StatusForbidden, "企业租户已停用")
			return
		}
		if reason, blocked := tenantBlockedReason(user.TenantID, time.Now()); blocked {
			consumeCaptchaChallenge(challenge.ID)
			services.LogOperation(services.OperationLogEvent{
				TenantID:       user.TenantID,
				UserID:         user.ID,
				Username:       user.Username,
				Operation:      "LOGIN",
				Target:         "租户策略",
				DescriptionKey: "operation.login",
				Result:         "failed",
				Reason:         reason,
				IP:             ip,
				UserAgent:      c.Ctx.Request.UserAgent(),
			})
			utils.ErrorWithData(c.Ctx, http.StatusForbidden, reason, map[string]interface{}{"error_code": reason})
			return
		}
	}
	if mfaRequiredForUser(user) {
		if req.MFACode == "" {
			consumeCaptchaChallenge(challenge.ID)
			utils.Success(c.Ctx, map[string]interface{}{
				"mfa_required": true,
				"user_id":      user.ID,
				"message":      "需要MFA验证码",
			})
			return
		}
		if !verifyUserMFA(user, req.MFACode) {
			consumeCaptchaChallenge(challenge.ID)
			services.LogOperation(services.OperationLogEvent{
				TenantID:       user.TenantID,
				UserID:         user.ID,
				Username:       user.Username,
				Operation:      "LOGIN_MFA",
				Target:         fmt.Sprintf("用户[%d]", user.ID),
				DescriptionKey: "operation.login_mfa",
				Result:         "failed",
				Reason:         "MFA验证码无效",
				IP:             ip,
				UserAgent:      c.Ctx.Request.UserAgent(),
			})
			authSecurity.markLoginFailure(ip, req.Username, now)
			utils.Error(c.Ctx, http.StatusUnauthorized, "MFA验证码无效")
			return
		}
	}

	sessionID := utils.NewSessionID()
	expiresAt := time.Now().Add(24 * time.Hour)
	session := models.UserSession{
		TenantModel: models.TenantModel{TenantID: user.TenantID},
		UserID:      user.ID,
		SessionID:   sessionID,
		DeviceType:  deviceType(c.Ctx.Request.UserAgent()),
		IP:          ip,
		Browser:     browserName(c.Ctx.Request.UserAgent()),
		OS:          osName(c.Ctx.Request.UserAgent()),
		UserAgent:   c.Ctx.Request.UserAgent(),
		LastActive:  time.Now(),
		ExpiresAt:   expiresAt,
	}
	if err := utils.DB.Create(&session).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建登录会话失败")
		return
	}

	// 生成JWT令牌
	accountType := effectiveAuthAccountType(user)
	token, err := utils.GenerateTokenWithSession(user.ID, user.TenantID, user.Username, user.Role, user.RoleID, accountType, sessionID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "令牌生成失败")
		return
	}

	// 更新最后登录时间
	loginAt := time.Now()
	user.LastLogin = &loginAt
	utils.DB.Save(&user)
	authSecurity.markLoginSuccess(ip, req.Username)
	consumeCaptchaChallenge(challenge.ID)
	services.LogOperation(services.OperationLogEvent{
		TenantID:       user.TenantID,
		UserID:         user.ID,
		Username:       user.Username,
		Operation:      "LOGIN",
		Target:         "用户[" + user.Username + "]",
		DescriptionKey: "operation.login",
		Params:         map[string]interface{}{"user_id": user.ID},
		Result:         "success",
		IP:             ip,
		UserAgent:      c.Ctx.Request.UserAgent(),
	})

	utils.Success(c.Ctx, map[string]interface{}{
		"token":                    token,
		"user":                     userContextPayload(user, accountType),
		"password_change_required": user.MustChangePassword,
	})
}

func userPermissions(user models.User) []string {
	accountType := effectiveAuthAccountType(user)
	if isPlatformAdminIdentity(accountType, user.Role, user.TenantID) || accountType == "tenant_admin" {
		return []string{"*"}
	}
	roleIDs := userRoleIDs(user)
	if len(roleIDs) == 0 {
		return []string{}
	}
	var roles []models.Role
	if err := utils.DB.Where("tenant_id = ? AND id IN ?", user.TenantID, roleIDs).Find(&roles).Error; err != nil {
		return []string{}
	}
	permissionSet := map[string]bool{}
	permissions := make([]string, 0)
	for _, role := range roles {
		for _, permission := range splitCSV(role.PermissionsJSON) {
			if permissionSet[permission] {
				continue
			}
			permissionSet[permission] = true
			permissions = append(permissions, permission)
		}
	}
	return permissions
}

func effectiveAuthAccountType(user models.User) string {
	accountType := strings.TrimSpace(user.AccountType)
	if accountType == "" {
		accountType = strings.TrimSpace(user.Role)
	}
	if user.TenantID > 0 && user.ID > 0 && accountType != "tenant_admin" && isTenantPrimaryAdmin(user.TenantID, user.ID) {
		return "tenant_admin"
	}
	return accountType
}

func userRoleSummaries(user models.User) ([]string, []string) {
	roleIDs := userRoleIDs(user)
	if len(roleIDs) == 0 {
		return []string{}, []string{}
	}
	var roles []models.Role
	if err := utils.DB.Select("name, code").Where("tenant_id = ? AND id IN ?", user.TenantID, roleIDs).Find(&roles).Error; err != nil {
		return []string{}, []string{}
	}
	names := make([]string, 0, len(roles))
	codes := make([]string, 0, len(roles))
	for _, role := range roles {
		if name := strings.TrimSpace(role.Name); name != "" {
			names = append(names, name)
		}
		if code := strings.TrimSpace(role.Code); code != "" {
			codes = append(codes, code)
		}
	}
	return names, codes
}

func userContextPayload(user models.User, accountType string) map[string]interface{} {
	if accountType == "" {
		accountType = user.AccountType
	}
	if accountType == "" {
		accountType = user.Role
	}
	isPlatformAdmin := isPlatformAdminIdentity(accountType, user.Role, user.TenantID)
	enabledModules := []string{}
	var tenantExpireAt *time.Time
	tenantDaysToExpire := 0
	if user.TenantID > 0 {
		if quota, err := getTenantQuota(user.TenantID); err == nil && quota != nil {
			enabledModules = tenantEffectiveModules(quota)
			tenantExpireAt = quota.ValidTo
			tenantDaysToExpire = daysToExpire(quota.ValidTo)
		}
	}
	roleNames, roleCodes := userRoleSummaries(user)
	return map[string]interface{}{
		"id":                    user.ID,
		"tenant_id":             user.TenantID,
		"username":              user.Username,
		"email":                 user.Email,
		"role":                  user.Role,
		"role_id":               user.RoleID,
		"role_ids":              userRoleIDs(user),
		"role_names":            roleNames,
		"role_codes":            roleCodes,
		"account_type":          accountType,
		"language_pref":         user.LanguagePref,
		"name":                  user.Name,
		"gender":                user.Gender,
		"phone":                 user.Phone,
		"avatar_url":            user.AvatarURL,
		"cover_url":             user.CoverURL,
		"department_id":         user.DepartmentID,
		"post_id":               user.PostID,
		"permissions":           userPermissions(user),
		"enabled_modules":       enabledModules,
		"tenant_expire_at":      tenantExpireAt,
		"tenant_days_to_expire": tenantDaysToExpire,
		"is_platform_admin":     isPlatformAdmin,
		"must_change_password":  user.MustChangePassword,
	}
}

func (c *AuthController) Me() {
	auth := currentAuth(c.Ctx)
	var user models.User
	if err := utils.DB.First(&user, auth.UserID).Error; err != nil {
		utils.Error(c.Ctx, http.StatusUnauthorized, "用户不存在")
		return
	}
	if user.Status != "active" {
		utils.Error(c.Ctx, http.StatusForbidden, "用户账号已被禁用")
		return
	}
	if user.TenantID > 0 {
		if reason, blocked := tenantBlockedReason(user.TenantID, time.Now()); blocked {
			utils.Error(c.Ctx, http.StatusForbidden, reason)
			return
		}
	}
	utils.Success(c.Ctx, userContextPayload(user, auth.AccountType))
}

func tenantBlockedReason(tenantID uint, now time.Time) (string, bool) {
	var tenant models.Tenant
	if err := utils.DB.Select("id,status").Where("id = ?", tenantID).First(&tenant).Error; err == nil {
		if tenant.Status != "active" {
			return "TENANT_DISABLED", true
		}
	}
	if quota, err := getTenantQuota(tenantID); err == nil && quota != nil && quota.ValidTo != nil && now.After(*quota.ValidTo) {
		return "TENANT_EXPIRED", true
	}
	return "", false
}

// Logout 用户登出
// @router /api/v1/auth/logout [post]
func (c *AuthController) Logout() {
	auth := currentAuth(c.Ctx)
	services.LogOperation(services.OperationLogEvent{
		TenantID:       auth.TenantID,
		UserID:         auth.UserID,
		Username:       auth.Username,
		Operation:      "LOGOUT",
		Target:         fmt.Sprintf("用户[%d]", auth.UserID),
		DescriptionKey: "operation.logout",
		Result:         "success",
		IP:             c.Ctx.Input.IP(),
		UserAgent:      c.Ctx.Request.UserAgent(),
	})
	if auth.SessionID != "" {
		now := time.Now()
		utils.DB.Model(&models.UserSession{}).Where("session_id = ?", auth.SessionID).Updates(map[string]interface{}{"revoked": true, "revoked_at": &now})
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"message": "登出成功",
	})
}

// RefreshToken 刷新令牌
// @router /api/v1/auth/refresh [post]
func (c *AuthController) RefreshToken() {
	// 从请求头中获取令牌
	authHeader := c.Ctx.Request.Header.Get("Authorization")
	if authHeader == "" {
		utils.Error(c.Ctx, http.StatusUnauthorized, "缺少认证令牌")
		return
	}

	// 解析令牌
	claims, err := utils.ParseToken(authHeader)
	if err != nil {
		utils.Error(c.Ctx, http.StatusUnauthorized, "无效的令牌")
		return
	}

	// 生成新令牌
	token, err := utils.GenerateTokenWithSession(claims.UserID, claims.TenantID, claims.Username, claims.Role, claims.RoleID, claims.AccountType, claims.SessionID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "令牌生成失败")
		return
	}

	utils.Success(c.Ctx, map[string]interface{}{
		"token": token,
	})
}

func (c *AuthController) ActivatePassword() {
	var req ActivatePasswordRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	req.NewPassword = strings.TrimSpace(req.NewPassword)
	if req.Token == "" {
		utils.BadRequest(c.Ctx, "激活令牌不能为空")
		return
	}
	if len(req.NewPassword) < 8 {
		utils.BadRequest(c.Ctx, "新密码至少需要8位")
		return
	}
	var user models.User
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		tokenRecord, err := consumeTenantAdminActivationToken(tx, req.Token)
		if err != nil {
			return err
		}
		if err := tx.Where("id = ? AND account_type = ?", tokenRecord.UserID, "tenant_admin").First(&user).Error; err != nil {
			return err
		}
		if isImmutableDemoAccount(user) {
			return fmt.Errorf("demo account password is immutable")
		}
		if err := user.SetPassword(req.NewPassword); err != nil {
			return err
		}
		user.MustChangePassword = false
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		if err := revokeTenantAdminSessions(tx, user.ID); err != nil {
			return err
		}
		return revokePendingActivationTokens(tx, user.ID)
	})
	if err != nil {
		if isImmutableDemoAccount(user) {
			utils.ForbiddenKey(c.Ctx, "auth.demoAccountPasswordLocked", nil)
			return
		}
		utils.Error(c.Ctx, http.StatusBadRequest, "激活链接无效或已过期")
		return
	}
	services.LogOperation(services.OperationLogEvent{
		TenantID:       user.TenantID,
		UserID:         user.ID,
		Username:       user.Username,
		Operation:      "ACTIVATE_TENANT_ADMIN_PASSWORD",
		Target:         fmt.Sprintf("用户[%d]", user.ID),
		DescriptionKey: "operation.activate_tenant_admin_password",
		Result:         "success",
		IP:             c.Ctx.Input.IP(),
		UserAgent:      c.Ctx.Request.UserAgent(),
	})
	utils.Success(c.Ctx, map[string]interface{}{
		"user_id":              user.ID,
		"tenant_id":            user.TenantID,
		"must_change_password": false,
		"activation_consumed":  true,
		"message":              "激活成功，请使用新密码登录",
	})
}

func deviceType(ua string) string {
	lower := strings.ToLower(ua)
	if strings.Contains(lower, "mobile") || strings.Contains(lower, "android") || strings.Contains(lower, "iphone") {
		return "mobile"
	}
	return "pc"
}

func browserName(ua string) string {
	lower := strings.ToLower(ua)
	switch {
	case strings.Contains(lower, "edg/"):
		return "Edge"
	case strings.Contains(lower, "firefox"):
		return "Firefox"
	case strings.Contains(lower, "chrome"):
		return "Chrome"
	case strings.Contains(lower, "safari"):
		return "Safari"
	default:
		return "未知"
	}
}

func osName(ua string) string {
	lower := strings.ToLower(ua)
	switch {
	case strings.Contains(lower, "windows"):
		return "Windows"
	case strings.Contains(lower, "mac os") || strings.Contains(lower, "macintosh"):
		return "macOS"
	case strings.Contains(lower, "linux"):
		return "Linux"
	case strings.Contains(lower, "android"):
		return "Android"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		return "iOS"
	default:
		return "未知"
	}
}
