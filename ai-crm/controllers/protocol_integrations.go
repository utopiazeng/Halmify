package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

func (c *CRMController) ExecuteIntegration() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		Type     string                 `json:"type"`
		Endpoint string                 `json:"endpoint"`
		Config   map[string]interface{} `json:"config"`
		Payload  map[string]interface{} `json:"payload"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	result, err := runRealIntegration(auth, req.Type, req.Endpoint, req.Config, req.Payload, false)
	status := "success"
	if err != nil {
		status = "failed"
		result["ok"] = false
		result["message"] = err.Error()
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "EXECUTE_INTEGRATION", req.Type, "operation.execute_integration", result, status, fmt.Sprint(result["message"]))
	if err != nil {
		utils.Error(c.Ctx, http.StatusBadGateway, err.Error())
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *CRMController) SyncHR() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	config := crmSettingValue(auth.TenantID, "integration", "hr")
	if len(config) == 0 {
		config = crmSettingValue(auth.TenantID, "integration", "ldap")
	}
	result, err := runRealIntegration(auth, fmt.Sprint(config["type"]), fmt.Sprint(config["endpoint"]), config, map[string]interface{}{}, false)
	if err != nil {
		utils.Error(c.Ctx, http.StatusBadGateway, err.Error())
		return
	}
	imported := 0
	if users, ok := result["users"].([]interface{}); ok {
		imported = upsertHRUsers(auth, users)
	}
	result["imported"] = imported
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SYNC_HR", "HR同步", "operation.sync_hr", result, "success", "")
	utils.Success(c.Ctx, result)
}

func (c *AuthController) SetupMFA() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	secret, err := newTOTPSecret()
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "MFA密钥生成失败")
		return
	}
	value := map[string]interface{}{
		"secret":     secret,
		"enabled":    false,
		"created_at": time.Now().Format(time.RFC3339),
	}
	if err := saveCRMSetting(auth.TenantID, "mfa_secret", fmt.Sprint(auth.UserID), value, auth.UserID); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "MFA配置保存失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"secret": secret,
		"otpauth_url": fmt.Sprintf("otpauth://totp/AI-CRM:%s?secret=%s&issuer=AI-CRM&period=30&digits=6",
			url.QueryEscape(auth.Username), secret),
	})
}

func (c *AuthController) VerifyMFA() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	setting := crmSettingValue(auth.TenantID, "mfa_secret", fmt.Sprint(auth.UserID))
	secret := strings.TrimSpace(fmt.Sprint(setting["secret"]))
	if secret == "" || !verifyTOTP(secret, req.Code, time.Now()) {
		utils.Error(c.Ctx, http.StatusUnauthorized, "MFA验证码无效")
		return
	}
	setting["enabled"] = true
	setting["verified_at"] = time.Now().Format(time.RFC3339)
	if err := saveCRMSetting(auth.TenantID, "mfa_secret", fmt.Sprint(auth.UserID), setting, auth.UserID); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "MFA启用失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"enabled": true})
}

func (c *AuthController) OIDCLogin() {
	var req struct {
		TenantID    uint   `json:"tenant_id"`
		Provider    string `json:"provider"`
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	tenantID := req.TenantID
	if tenantID == 0 {
		tenantID = defaultTenantID()
	}
	config := crmSettingValue(tenantID, "integration", "sso")
	if req.Provider != "" {
		config = crmSettingValue(tenantID, "integration", req.Provider)
	}
	if len(config) == 0 {
		utils.BadRequest(c.Ctx, "未配置SSO提供方")
		return
	}
	profile, err := oidcUserProfile(config, req.AccessToken, req.IDToken)
	if err != nil {
		utils.Error(c.Ctx, http.StatusUnauthorized, err.Error())
		return
	}
	user, err := upsertSSOUser(tenantID, profile)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "SSO用户同步失败")
		return
	}
	token, err := utils.GenerateTokenWithContext(user.ID, user.TenantID, user.Username, user.Role, user.RoleID, user.AccountType)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "令牌生成失败")
		return
	}
	services.LogOperation(services.OperationLogEvent{
		TenantID:       user.TenantID,
		UserID:         user.ID,
		Username:       user.Username,
		Operation:      "SSO_LOGIN",
		Target:         fmt.Sprintf("用户[%d]", user.ID),
		DescriptionKey: "operation.sso_login",
		Result:         "success",
		IP:             c.Ctx.Input.IP(),
		UserAgent:      c.Ctx.Request.UserAgent(),
	})
	utils.Success(c.Ctx, map[string]interface{}{"token": token, "user": userContextPayload(user, user.AccountType)})
}

func runRealIntegration(auth authContext, integrationType, endpoint string, config, payload map[string]interface{}, dryRun bool) (map[string]interface{}, error) {
	integrationType = strings.ToLower(strings.TrimSpace(integrationType))
	if integrationType == "" {
		integrationType = strings.ToLower(strings.TrimSpace(fmt.Sprint(config["type"])))
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(fmt.Sprint(config["endpoint"]))
	}
	result := map[string]interface{}{
		"type":       integrationType,
		"endpoint":   endpoint,
		"checked_at": time.Now().Format(time.RFC3339),
		"ok":         true,
	}
	if !integrationEnabled(config) || dryRun {
		result["mode"] = "protocol_contract_check"
		result["message"] = "配置已通过协议契约检查，enabled=true 时执行真实外呼"
		return result, validateIntegrationConfig(integrationType, endpoint, config)
	}
	switch integrationType {
	case "email", "mail", "smtp":
		return sendSMTPMail(endpoint, config, payload, result)
	case "finance", "financial", "invoice_api":
		return callSignedJSONAPI("finance", endpoint, config, payload, result)
	case "logistics", "shipping":
		return callSignedJSONAPI("logistics", endpoint, config, payload, result)
	case "hr", "hr_sync":
		return callHRAPI(endpoint, config, result)
	case "ldap", "ldaps":
		return testLDAP(endpoint, config, result)
	case "sso", "oidc":
		return testOIDC(config, result)
	case "mfa", "totp":
		secret := strings.TrimSpace(fmt.Sprint(config["secret"]))
		if secret == "" {
			generated, err := newTOTPSecret()
			if err != nil {
				return result, err
			}
			result["secret"] = generated
			result["message"] = "已生成TOTP密钥"
			return result, nil
		}
		result["code_valid"] = verifyTOTP(secret, fmt.Sprint(payload["code"]), time.Now())
		result["message"] = "TOTP校验完成"
		return result, nil
	default:
		return result, fmt.Errorf("不支持的集成类型: %s", integrationType)
	}
}

func validateIntegrationConfig(integrationType, endpoint string, config map[string]interface{}) error {
	switch strings.ToLower(integrationType) {
	case "email", "mail", "smtp", "finance", "financial", "invoice_api", "logistics", "shipping", "hr", "hr_sync", "ldap", "ldaps":
		if strings.TrimSpace(endpoint) == "" {
			return fmt.Errorf("未配置端点")
		}
	case "sso", "oidc":
		if strings.TrimSpace(fmt.Sprint(config["issuer"])) == "" && strings.TrimSpace(fmt.Sprint(config["discovery_url"])) == "" {
			return fmt.Errorf("OIDC必须配置issuer或discovery_url")
		}
	}
	return nil
}

func integrationEnabled(config map[string]interface{}) bool {
	value, ok := config["enabled"]
	return ok && (value == true || strings.EqualFold(fmt.Sprint(value), "true"))
}

func sendSMTPMail(endpoint string, config, payload, result map[string]interface{}) (map[string]interface{}, error) {
	host, port, err := smtpHostPort(endpoint)
	if err != nil {
		return result, err
	}
	from := firstNonEmpty(fmt.Sprint(payload["from"]), fmt.Sprint(config["from"]))
	to := stringList(payload["to"])
	if len(to) == 0 {
		to = stringList(config["to"])
	}
	if from == "" || len(to) == 0 {
		return result, fmt.Errorf("邮件发送必须包含from和to")
	}
	subject := firstNonEmpty(fmt.Sprint(payload["subject"]), "Halmify CRM 通知")
	body := firstNonEmpty(fmt.Sprint(payload["body"]), fmt.Sprint(payload["content"]))
	message := []byte("From: " + from + "\r\nTo: " + strings.Join(to, ",") + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body)
	var auth smtp.Auth
	username := strings.TrimSpace(fmt.Sprint(config["username"]))
	password := strings.TrimSpace(fmt.Sprint(config["password"]))
	if username != "" || password != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}
	addr := net.JoinHostPort(host, port)
	if strings.HasPrefix(endpoint, "smtps://") {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return result, err
		}
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return result, err
		}
		defer client.Close()
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return result, err
			}
		}
		if err := client.Mail(from); err != nil {
			return result, err
		}
		for _, recipient := range to {
			if err := client.Rcpt(recipient); err != nil {
				return result, err
			}
		}
		writer, err := client.Data()
		if err != nil {
			return result, err
		}
		if _, err := writer.Write(message); err != nil {
			return result, err
		}
		if err := writer.Close(); err != nil {
			return result, err
		}
		if err := client.Quit(); err != nil {
			return result, err
		}
	} else if err := smtp.SendMail(addr, auth, from, to, message); err != nil {
		return result, err
	}
	result["message"] = "邮件已通过SMTP发送"
	result["recipients"] = to
	return result, nil
}

func callSignedJSONAPI(apiType, endpoint string, config, payload, result map[string]interface{}) (map[string]interface{}, error) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	req.Header.Set("Content-Type", "application/json")
	apiKey := strings.TrimSpace(fmt.Sprint(config["api_key"]))
	secret := strings.TrimSpace(fmt.Sprint(config["secret"]))
	timestamp := fmt.Sprint(time.Now().Unix())
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(timestamp))
		mac.Write(body)
		req.Header.Set("X-Timestamp", timestamp)
		req.Header.Set("X-Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	result["status_code"] = resp.StatusCode
	result["response"] = string(data)
	if resp.StatusCode >= 300 {
		return result, fmt.Errorf("%s API返回状态码 %d", apiType, resp.StatusCode)
	}
	result["message"] = apiType + " API调用完成"
	return result, nil
}

func callHRAPI(endpoint string, config, result map[string]interface{}) (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return result, err
	}
	if token := strings.TrimSpace(fmt.Sprint(config["token"])); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 300 {
		result["status_code"] = resp.StatusCode
		return result, fmt.Errorf("HR API返回状态码 %d", resp.StatusCode)
	}
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return result, err
	}
	users := extractHRUsers(parsed)
	result["users"] = users
	result["count"] = len(users)
	result["message"] = "HR API同步完成"
	return result, nil
}

func testLDAP(endpoint string, config, result map[string]interface{}) (map[string]interface{}, error) {
	host, port, useTLS, err := ldapAddress(endpoint)
	if err != nil {
		return result, err
	}
	addr := net.JoinHostPort(host, port)
	var conn net.Conn
	if useTLS {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
	}
	if err != nil {
		return result, err
	}
	defer conn.Close()
	result["message"] = "LDAP端点连接成功"
	result["protocol"] = map[string]interface{}{"host": host, "port": port, "tls": useTLS}
	return result, nil
}

func testOIDC(config, result map[string]interface{}) (map[string]interface{}, error) {
	discoveryURL := oidcDiscoveryURL(config)
	if discoveryURL == "" {
		return result, fmt.Errorf("OIDC discovery_url无效")
	}
	resp, err := (&http.Client{Timeout: 12 * time.Second}).Get(discoveryURL)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return result, fmt.Errorf("OIDC discovery返回状态码 %d", resp.StatusCode)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return result, err
	}
	result["issuer"] = doc["issuer"]
	result["authorization_endpoint"] = doc["authorization_endpoint"]
	result["token_endpoint"] = doc["token_endpoint"]
	result["userinfo_endpoint"] = doc["userinfo_endpoint"]
	result["message"] = "OIDC discovery完成"
	return result, nil
}

func oidcUserProfile(config map[string]interface{}, accessToken, idToken string) (map[string]interface{}, error) {
	if accessToken == "" && idToken == "" {
		return nil, fmt.Errorf("SSO登录必须提供access_token或id_token")
	}
	userinfo := strings.TrimSpace(fmt.Sprint(config["userinfo_endpoint"]))
	if userinfo == "" {
		docResult, err := testOIDC(config, map[string]interface{}{})
		if err != nil {
			return nil, err
		}
		userinfo = strings.TrimSpace(fmt.Sprint(docResult["userinfo_endpoint"]))
	}
	if accessToken != "" && userinfo != "" {
		req, _ := http.NewRequest(http.MethodGet, userinfo, nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err := (&http.Client{Timeout: 12 * time.Second}).Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("OIDC userinfo返回状态码 %d", resp.StatusCode)
		}
		var profile map[string]interface{}
		if err := json.Unmarshal(body, &profile); err != nil {
			return nil, err
		}
		return profile, nil
	}
	return decodeJWTClaims(idToken)
}

func upsertSSOUser(tenantID uint, profile map[string]interface{}) (models.User, error) {
	email := firstNonEmpty(fmt.Sprint(profile["email"]), fmt.Sprint(profile["preferred_username"]))
	username := firstNonEmpty(fmt.Sprint(profile["preferred_username"]), fmt.Sprint(profile["name"]), email)
	if username == "" {
		username = fmt.Sprint(profile["sub"])
	}
	if username == "" {
		return models.User{}, fmt.Errorf("SSO用户缺少用户名")
	}
	var user models.User
	err := utils.DB.Where("tenant_id = ? AND (username = ? OR email = ?)", tenantID, username, email).First(&user).Error
	if err == nil {
		user.Email = email
		user.Status = "active"
		return user, utils.DB.Save(&user).Error
	}
	user = models.User{TenantModel: models.TenantModel{TenantID: tenantID}, Username: username, Email: email, Role: "sub_user", AccountType: "sub_user", Status: "active", LanguagePref: "zh-CN"}
	_ = user.SetPassword(base64.RawURLEncoding.EncodeToString([]byte(username + time.Now().String())))
	return user, utils.DB.Create(&user).Error
}

func upsertHRUsers(auth authContext, users []interface{}) int {
	imported := 0
	for _, raw := range users {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name := firstNonEmpty(fmt.Sprint(item["name"]), fmt.Sprint(item["username"]), fmt.Sprint(item["email"]))
		if name == "" {
			continue
		}
		payload := map[string]interface{}{}
		for key, value := range item {
			payload[key] = value
		}
		object := models.CRMObject{
			TenantModel: models.TenantModel{TenantID: auth.TenantID},
			Domain:      "teams",
			ObjectType:  "team_member",
			Name:        name,
			Code:        firstNonEmpty(fmt.Sprint(item["employee_no"]), fmt.Sprint(item["id"])),
			Status:      "active",
			PayloadJSON: toJSONString(payload),
			CreatedBy:   auth.UserID,
			UpdatedBy:   auth.UserID,
		}
		key := object.Code
		if key == "" {
			key = name
		}
		if err := utils.DB.Where("tenant_id = ? AND domain = ? AND object_type = ? AND (code = ? OR name = ?)", auth.TenantID, "teams", "team_member", key, name).Assign(object).FirstOrCreate(&object).Error; err == nil {
			imported++
		}
	}
	return imported
}

func saveCRMSetting(tenantID uint, scope, key string, value map[string]interface{}, userID uint) error {
	valueJSON, _ := json.Marshal(value)
	setting := models.CRMSetting{TenantModel: models.TenantModel{TenantID: tenantID}, Scope: scope, Key: key, ValueJSON: string(valueJSON), UpdatedBy: userID}
	return utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", tenantID, scope, key).Assign(setting).FirstOrCreate(&setting).Error
}

func mfaRequiredForUser(user models.User) bool {
	if user.TenantID == 0 {
		return false
	}
	secret := crmSettingValue(user.TenantID, "mfa_secret", fmt.Sprint(user.ID))
	if secret["enabled"] == true || strings.EqualFold(fmt.Sprint(secret["enabled"]), "true") {
		return true
	}
	policy := crmSettingValue(user.TenantID, "integration", "mfa")
	if len(policy) == 0 {
		policy = crmSettingValue(user.TenantID, "mfa_policy", "default")
	}
	return policy["required"] == true || strings.EqualFold(fmt.Sprint(policy["required"]), "true")
}

func verifyUserMFA(user models.User, code string) bool {
	setting := crmSettingValue(user.TenantID, "mfa_secret", fmt.Sprint(user.ID))
	secret := strings.TrimSpace(fmt.Sprint(setting["secret"]))
	return secret != "" && verifyTOTP(secret, code, time.Now())
}

func newTOTPSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), "="), nil
}

func verifyTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	for offset := -1; offset <= 1; offset++ {
		if totpCode(secret, now.Add(time.Duration(offset)*30*time.Second)) == code {
			return true
		}
	}
	return false
}

func totpCode(secret string, now time.Time) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return ""
	}
	counter := uint64(now.Unix() / 30)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", value%1000000)
}

func smtpHostPort(endpoint string) (string, string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return "", "", fmt.Errorf("SMTP端点无效")
	}
	host := parsed.Hostname()
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "smtps" {
			port = "465"
		} else {
			port = "25"
		}
	}
	return host, port, nil
}

func ldapAddress(endpoint string) (string, string, bool, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return "", "", false, fmt.Errorf("LDAP端点无效")
	}
	port := parsed.Port()
	useTLS := parsed.Scheme == "ldaps"
	if port == "" {
		if useTLS {
			port = "636"
		} else {
			port = "389"
		}
	}
	return parsed.Hostname(), port, useTLS, nil
}

func oidcDiscoveryURL(config map[string]interface{}) string {
	if discovery := strings.TrimSpace(fmt.Sprint(config["discovery_url"])); discovery != "" {
		return discovery
	}
	issuer := strings.TrimRight(strings.TrimSpace(fmt.Sprint(config["issuer"])), "/")
	if issuer == "" {
		return ""
	}
	return issuer + "/.well-known/openid-configuration"
}

func decodeJWTClaims(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("id_token格式无效")
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func extractHRUsers(parsed interface{}) []interface{} {
	switch typed := parsed.(type) {
	case []interface{}:
		return typed
	case map[string]interface{}:
		for _, key := range []string{"users", "employees", "items", "data"} {
			if list, ok := typed[key].([]interface{}); ok {
				return list
			}
		}
	}
	return []interface{}{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func stringList(value interface{}) []string {
	items := []string{}
	for _, item := range interfaceList(value) {
		text := strings.TrimSpace(fmt.Sprint(item))
		if text != "" && text != "<nil>" {
			items = append(items, text)
		}
	}
	sort.Strings(items)
	return items
}

func objectsByDomain(auth authContext, domain string, limit int) []models.CRMObject {
	var items []models.CRMObject
	_ = crmScopedObjectQuery(auth, domain).Order("updated_at desc").Limit(limit).Find(&items).Error
	return items
}

func createIntegrationObject(tx *gorm.DB, auth authContext, domain, objectType, name string, payload map[string]interface{}) error {
	object := models.CRMObject{
		TenantModel: models.TenantModel{TenantID: auth.TenantID},
		Domain:      domain,
		ObjectType:  objectType,
		Name:        name,
		Status:      "active",
		PayloadJSON: toJSONString(payload),
		CreatedBy:   auth.UserID,
		UpdatedBy:   auth.UserID,
	}
	return tx.Create(&object).Error
}
