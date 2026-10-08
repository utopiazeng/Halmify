package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
)

type ProfileController struct {
	web.Controller
}

const maxAvatarUploadBytes int64 = 5 * 1024 * 1024

var allowedAvatarMIMETypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

func (c *ProfileController) GetProfile() {
	auth := currentAuth(c.Ctx)
	var user models.User
	if err := utils.DB.First(&user, auth.UserID).Error; err != nil {
		utils.NotFound(c.Ctx, "用户不存在")
		return
	}
	utils.Success(c.Ctx, profilePayload(user))
}

func (c *ProfileController) UpdateProfile() {
	auth := currentAuth(c.Ctx)
	var req struct {
		Name      string `json:"name"`
		Gender    string `json:"gender"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		Bio       string `json:"bio"`
		Address   string `json:"address"`
		AvatarURL string `json:"avatar_url"`
		CoverURL  string `json:"cover_url"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var user models.User
	if err := utils.DB.First(&user, auth.UserID).Error; err != nil {
		utils.NotFound(c.Ctx, "用户不存在")
		return
	}
	user.Name = strings.TrimSpace(req.Name)
	user.Gender = strings.TrimSpace(req.Gender)
	user.Email = strings.TrimSpace(req.Email)
	user.Phone = strings.TrimSpace(req.Phone)
	user.Bio = req.Bio
	user.Address = strings.TrimSpace(req.Address)
	user.AvatarURL = sanitizeProfileImageURL(req.AvatarURL)
	user.CoverURL = strings.TrimSpace(req.CoverURL)
	if err := utils.DB.Save(&user).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存资料失败")
		return
	}
	utils.Success(c.Ctx, profilePayload(user))
}

func sanitizeProfileImageURL(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	if strings.HasPrefix(text, "data:image/svg+xml") || strings.HasPrefix(text, "data:image/png") || strings.HasPrefix(text, "data:image/jpeg") || strings.HasPrefix(text, "data:image/webp") || strings.HasPrefix(text, "data:image/gif") {
		return text
	}
	if strings.HasPrefix(text, "/api/v1/files/") || strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		return text
	}
	return ""
}

func (c *ProfileController) UploadAvatar() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	file, header, err := c.GetFile("file")
	if err != nil {
		utils.BadRequest(c.Ctx, "缺少头像文件")
		return
	}
	defer file.Close()
	if header.Size > maxAvatarUploadBytes {
		utils.BadRequest(c.Ctx, "头像文件不能超过5MB")
		return
	}
	if err := ensureTenantStorageAvailable(auth.TenantID, header.Size); err != nil {
		if errors.Is(err, errTenantStorageInsufficient) {
			utils.Error(c.Ctx, http.StatusForbidden, "租户存储空间不足，请清理资料或升级空间")
			return
		}
		if errors.Is(err, errTenantStorageUploadLocked) {
			utils.Error(c.Ctx, http.StatusForbidden, "存储增购已到期，请续费后继续上传")
			return
		}
		utils.Error(c.Ctx, http.StatusServiceUnavailable, err.Error())
		return
	}
	contentType, err := detectMultipartFileType(file)
	if err != nil {
		utils.BadRequest(c.Ctx, "头像文件读取失败")
		return
	}
	if !allowedAvatarMIMETypes[contentType] {
		utils.BadRequest(c.Ctx, "仅支持 JPG、PNG、WEBP、GIF 图片")
		return
	}

	node, err := services.ChooseUploadNode()
	if err != nil {
		utils.Error(c.Ctx, http.StatusServiceUnavailable, "暂无可用文件服务节点")
		return
	}
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	contentTypeHeader := writer.FormDataContentType()
	go func() {
		defer pipeWriter.Close()
		defer writer.Close()
		part, err := writer.CreateFormFile("file", header.Filename)
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			_ = pipeWriter.CloseWithError(err)
		}
	}()

	status, payload, err := services.ProxyUpload(*node, auth.TenantID, auth.Username, pipeReader, contentTypeHeader, -1)
	if err != nil {
		utils.Error(c.Ctx, http.StatusBadGateway, "头像上传失败: "+err.Error())
		return
	}
	if status < 200 || status >= 300 {
		writeRawJSON(c.Ctx, status, payload)
		return
	}
	var uploadResp struct {
		Data struct {
			FileID string `json:"file_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &uploadResp); err != nil || strings.TrimSpace(uploadResp.Data.FileID) == "" {
		utils.Error(c.Ctx, http.StatusBadGateway, "头像上传返回异常")
		return
	}
	avatarURL := "/api/v1/files/" + strings.TrimSpace(uploadResp.Data.FileID) + "/download"
	var user models.User
	if err := utils.DB.First(&user, auth.UserID).Error; err != nil {
		utils.NotFound(c.Ctx, "用户不存在")
		return
	}
	user.AvatarURL = avatarURL
	if err := utils.DB.Save(&user).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存头像失败")
		return
	}
	utils.Success(c.Ctx, profilePayload(user))
}

func (c *ProfileController) ChangePassword() {
	auth := currentAuth(c.Ctx)
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if len(strings.TrimSpace(req.NewPassword)) < 6 {
		utils.BadRequest(c.Ctx, "新密码至少需要6位")
		return
	}
	var user models.User
	if err := utils.DB.First(&user, auth.UserID).Error; err != nil {
		utils.NotFound(c.Ctx, "用户不存在")
		return
	}
	if isImmutableDemoAccount(user) {
		utils.ForbiddenKey(c.Ctx, "auth.demoAccountPasswordLocked", nil)
		return
	}
	if !user.CheckPassword(req.CurrentPassword) {
		utils.BadRequest(c.Ctx, "当前密码不正确")
		return
	}
	if err := user.SetPassword(req.NewPassword); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "密码加密失败")
		return
	}
	now := time.Now()
	user.MustChangePassword = false
	if err := utils.DB.Save(&user).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新密码失败")
		return
	}
	utils.DB.Model(&models.UserSession{}).
		Where("user_id = ? AND revoked = ?", auth.UserID, false).
		Updates(map[string]interface{}{"revoked": true, "revoked_at": &now})
	utils.Success(c.Ctx, map[string]interface{}{"message": "密码已更新"})
}

func (c *ProfileController) ListSessions() {
	auth := currentAuth(c.Ctx)
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 5)
	if size <= 0 || size > 20 {
		size = 5
	}
	var total int64
	utils.DB.Model(&models.UserSession{}).Where("user_id = ?", auth.UserID).Count(&total)
	items := []models.UserSession{}
	if err := utils.DB.Where("user_id = ?", auth.UserID).Order("last_active desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取登录会话失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": total, "page": page, "size": size, "current_session_id": auth.SessionID})
}

func (c *ProfileController) RevokeOtherSessions() {
	auth := currentAuth(c.Ctx)
	now := time.Now()
	if err := utils.DB.Model(&models.UserSession{}).
		Where("user_id = ? AND session_id <> ? AND revoked = ?", auth.UserID, auth.SessionID, false).
		Updates(map[string]interface{}{"revoked": true, "revoked_at": &now}).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "退出其他会话失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"message": "已退出其他会话"})
}

func profilePayload(user models.User) map[string]interface{} {
	var department models.SystemDepartment
	var post models.SystemPost
	if user.DepartmentID > 0 {
		utils.DB.First(&department, user.DepartmentID)
	}
	if user.PostID > 0 {
		utils.DB.First(&post, user.PostID)
	}
	return map[string]interface{}{
		"id":                   user.ID,
		"tenant_id":            user.TenantID,
		"username":             user.Username,
		"name":                 user.Name,
		"gender":               user.Gender,
		"email":                user.Email,
		"phone":                user.Phone,
		"bio":                  user.Bio,
		"address":              user.Address,
		"avatar_url":           user.AvatarURL,
		"cover_url":            user.CoverURL,
		"department_id":        user.DepartmentID,
		"department_name":      department.Name,
		"post_id":              user.PostID,
		"post_name":            post.Name,
		"role_ids":             userRoleIDs(user),
		"must_change_password": user.MustChangePassword,
	}
}

func detectMultipartFileType(file multipart.File) (string, error) {
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return http.DetectContentType(buf[:n]), nil
}
