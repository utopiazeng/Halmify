package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
)

type AccountController struct {
	web.Controller
}

var supportedLanguagePrefs = map[string]bool{
	"zh-CN": true,
	"zh-TW": true,
	"en-US": true,
	"es-ES": true,
	"fr-FR": true,
	"de-DE": true,
	"ja-JP": true,
	"ko-KR": true,
	"pt-PT": true,
	"ar-SA": true,
	"ru-RU": true,
	"it-IT": true,
}

type createSubAccountRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	Name         string `json:"name"`
	Gender       string `json:"gender"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	RoleID       uint   `json:"role_id"`
	RoleIDs      []uint `json:"role_ids"`
	DepartmentID uint   `json:"department_id"`
	PostID       uint   `json:"post_id"`
	LanguagePref string `json:"language_pref"`
}

type userWithRoles struct {
	models.User
	RoleIDs        []uint `json:"role_ids"`
	IsPrimaryAdmin bool   `json:"is_primary_admin"`
}

type userPickerOption struct {
	ID             uint   `json:"id"`
	Username       string `json:"username"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	RoleID         uint   `json:"role_id"`
	DepartmentID   uint   `json:"department_id"`
	PostID         uint   `json:"post_id"`
	AccountType    string `json:"account_type"`
	Status         string `json:"status"`
	LanguagePref   string `json:"language_pref"`
	DepartmentName string `json:"department_name,omitempty"`
	PostName       string `json:"post_name,omitempty"`
}

func (c *AccountController) ListUsers() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	tenantID := auth.TenantID
	if tenantID == 0 {
		utils.BadRequest(c.Ctx, "缺少租户ID")
		return
	}
	status := strings.TrimSpace(c.Ctx.Input.Query("status"))
	keyword := strings.TrimSpace(c.Ctx.Input.Query("keyword"))
	username := strings.TrimSpace(c.Ctx.Input.Query("username"))
	name := strings.TrimSpace(c.Ctx.Input.Query("name"))
	phone := strings.TrimSpace(c.Ctx.Input.Query("phone"))
	gender := strings.TrimSpace(c.Ctx.Input.Query("gender"))
	roleID := queryUint(c.Ctx, "role_id")
	departmentID := queryUint(c.Ctx, "department_id")
	postID := queryUint(c.Ctx, "post_id")
	start := parseTimeQuery(c.Ctx.Input.Query("start_time"))
	end := parseTimeQuery(c.Ctx.Input.Query("end_time"))
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)

	query := utils.DB.Model(&models.User{}).Where("tenant_id = ?", tenantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		query = query.Where("username LIKE ? OR name LIKE ? OR phone LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if phone != "" {
		query = query.Where("phone LIKE ?", "%"+phone+"%")
	}
	if gender != "" {
		query = query.Where("gender = ?", gender)
	}
	if roleID > 0 {
		query = query.Where("role_id = ? OR id IN (SELECT user_id FROM user_roles WHERE role_id = ?)", roleID, roleID)
	}
	if departmentID > 0 {
		query = query.Where("department_id = ?", departmentID)
	}
	if postID > 0 {
		query = query.Where("post_id = ?", postID)
	}
	if start != nil {
		query = query.Where("created_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("created_at <= ?", *end)
	}
	var total int64
	query.Count(&total)
	var users []models.User
	if err := query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&users).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取账号失败")
		return
	}
	if !auth.IsTenantAdmin() {
		utils.Success(c.Ctx, map[string]interface{}{"items": userPickerOptions(users), "total": total})
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": attachUserRoleIDs(users), "total": total})
}

func (c *AccountController) CreateUser() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID := auth.TenantID
	var req createSubAccountRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || strings.TrimSpace(req.Password) == "" {
		utils.BadRequest(c.Ctx, "用户名和密码不能为空")
		return
	}
	if len(strings.TrimSpace(req.Password)) < 6 {
		utils.BadRequest(c.Ctx, "密码至少需要6位")
		return
	}
	if req.LanguagePref == "" {
		req.LanguagePref = "zh-CN"
	}
	if !isSupportedLanguagePref(req.LanguagePref) {
		utils.BadRequest(c.Ctx, "language_pref 不在支持范围内")
		return
	}
	roleIDs := normalizeRoleIDs(req.RoleIDs, req.RoleID)
	if len(roleIDs) == 0 {
		utils.BadRequest(c.Ctx, "请选择角色")
		return
	}
	if !rolesBelongToTenant(roleIDs, tenantID) {
		utils.BadRequest(c.Ctx, "角色不存在或不属于当前租户")
		return
	}
	if ok, msg := canCreateMoreUsers(tenantID); !ok {
		utils.Error(c.Ctx, http.StatusBadRequest, msg)
		return
	}
	user := models.User{
		TenantModel:  models.TenantModel{TenantID: tenantID},
		Username:     req.Username,
		Name:         strings.TrimSpace(req.Name),
		Gender:       strings.TrimSpace(req.Gender),
		Phone:        strings.TrimSpace(req.Phone),
		Email:        req.Email,
		Role:         "sub_user",
		RoleID:       roleIDs[0],
		DepartmentID: req.DepartmentID,
		PostID:       req.PostID,
		AccountType:  "sub_user",
		Status:       "active",
		LanguagePref: req.LanguagePref,
	}
	if user.DepartmentID > 0 && !departmentBelongsToTenant(user.DepartmentID, tenantID) {
		utils.BadRequest(c.Ctx, "部门不存在或不属于当前租户")
		return
	}
	if user.PostID > 0 && !postBelongsToTenant(user.PostID, tenantID) {
		utils.BadRequest(c.Ctx, "岗位不存在或不属于当前租户")
		return
	}
	if err := user.SetPassword(req.Password); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "密码加密失败")
		return
	}
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return replaceUserRoles(tx, user.ID, roleIDs)
	}); err != nil {
		logControllerOperation(c.Ctx, auth, tenantID, "CREATE_USER", "用户管理", "operation.create_user", map[string]interface{}{"username": req.Username}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建子账号失败")
		return
	}
	logControllerOperation(c.Ctx, auth, tenantID, "CREATE_USER", "用户管理", "operation.create_user", map[string]interface{}{"user_id": user.ID, "username": user.Username, "role_ids": roleIDs}, "success", "")
	utils.Success(c.Ctx, userWithRoles{User: user, RoleIDs: roleIDs})
}

func (c *AccountController) UpdateUser() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的用户ID")
		return
	}
	var req struct {
		Name         string `json:"name"`
		Gender       string `json:"gender"`
		Phone        string `json:"phone"`
		Email        string `json:"email"`
		RoleID       uint   `json:"role_id"`
		RoleIDs      []uint `json:"role_ids"`
		DepartmentID uint   `json:"department_id"`
		PostID       uint   `json:"post_id"`
		Status       string `json:"status"`
		LanguagePref string `json:"language_pref"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var user models.User
	if err := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).First(&user).Error; err != nil {
		utils.NotFound(c.Ctx, "用户不存在")
		return
	}
	if isTenantPrimaryAdmin(auth.TenantID, user.ID) && req.Status == "disabled" {
		utils.ErrorWithData(c.Ctx, http.StatusBadRequest, "PRIMARY_TENANT_ADMIN_PROTECTED", map[string]interface{}{"error_code": "PRIMARY_TENANT_ADMIN_PROTECTED"})
		return
	}
	if user.AccountType == "tenant_admin" && user.ID == auth.UserID && req.Status == "disabled" {
		utils.BadRequest(c.Ctx, "不能禁用当前管理员账号")
		return
	}
	roleIDs := normalizeRoleIDs(req.RoleIDs, req.RoleID)
	if len(roleIDs) > 0 {
		if !rolesBelongToTenant(roleIDs, auth.TenantID) {
			utils.BadRequest(c.Ctx, "角色不存在或不属于当前租户")
			return
		}
		user.RoleID = roleIDs[0]
	}
	user.Name = strings.TrimSpace(req.Name)
	user.Gender = strings.TrimSpace(req.Gender)
	user.Phone = strings.TrimSpace(req.Phone)
	user.Email = strings.TrimSpace(req.Email)
	if req.DepartmentID > 0 && !departmentBelongsToTenant(req.DepartmentID, auth.TenantID) {
		utils.BadRequest(c.Ctx, "部门不存在或不属于当前租户")
		return
	}
	if req.PostID > 0 && !postBelongsToTenant(req.PostID, auth.TenantID) {
		utils.BadRequest(c.Ctx, "岗位不存在或不属于当前租户")
		return
	}
	user.DepartmentID = req.DepartmentID
	user.PostID = req.PostID
	if req.Status != "" {
		if req.Status != "active" && req.Status != "disabled" {
			utils.BadRequest(c.Ctx, "状态必须为 active 或 disabled")
			return
		}
		user.Status = req.Status
	}
	if req.LanguagePref != "" {
		if !isSupportedLanguagePref(req.LanguagePref) {
			utils.BadRequest(c.Ctx, "language_pref 不在支持范围内")
			return
		}
		user.LanguagePref = req.LanguagePref
	}
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		if len(roleIDs) > 0 {
			return replaceUserRoles(tx, user.ID, roleIDs)
		}
		return nil
	}); err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_USER", "用户管理", "operation.update_user", map[string]interface{}{"user_id": id, "username": user.Username}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新用户失败")
		return
	}
	if len(roleIDs) == 0 {
		roleIDs = userRoleIDs(user)
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_USER", "用户管理", "operation.update_user", map[string]interface{}{"user_id": id, "username": user.Username, "role_ids": roleIDs}, "success", "")
	utils.Success(c.Ctx, userWithRoles{User: user, RoleIDs: roleIDs})
}

func (c *AccountController) ResetPassword() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的用户ID")
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if len(strings.TrimSpace(req.Password)) < 6 {
		utils.BadRequest(c.Ctx, "新密码至少需要6位")
		return
	}
	var user models.User
	if err := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).First(&user).Error; err != nil {
		utils.NotFound(c.Ctx, "用户不存在")
		return
	}
	if isImmutableDemoAccount(user) {
		utils.ForbiddenKey(c.Ctx, "auth.demoAccountPasswordLocked", nil)
		return
	}
	if isTenantPrimaryAdmin(auth.TenantID, user.ID) {
		utils.ErrorWithData(c.Ctx, http.StatusBadRequest, "PRIMARY_TENANT_ADMIN_PROTECTED", map[string]interface{}{"error_code": "PRIMARY_TENANT_ADMIN_PROTECTED"})
		return
	}
	if err := user.SetPassword(req.Password); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "密码加密失败")
		return
	}
	if err := utils.DB.Save(&user).Error; err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "RESET_PASSWORD", "用户管理", "operation.reset_password", map[string]interface{}{"user_id": id, "username": user.Username}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "重置密码失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "RESET_PASSWORD", "用户管理", "operation.reset_password", map[string]interface{}{"user_id": id, "username": user.Username}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"user_id": id})
}

func (c *AccountController) DeleteUser() {
	auth := currentAuth(c.Ctx)
	if !auth.IsSuperAdmin() && !auth.IsTenantAdmin() {
		utils.Error(c.Ctx, http.StatusForbidden, "仅企业管理员或超管可执行该操作")
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的用户ID")
		return
	}
	if id == auth.UserID {
		utils.BadRequest(c.Ctx, "不能删除当前登录账号")
		return
	}
	var user models.User
	query := utils.DB.Where("id = ?", id)
	if auth.IsTenantAdmin() {
		if auth.TenantID == 0 {
			utils.Error(c.Ctx, http.StatusForbidden, "缺少租户上下文")
			return
		}
		query = query.Where("tenant_id = ?", auth.TenantID)
	}
	if err := query.First(&user).Error; err != nil {
		utils.NotFound(c.Ctx, "用户不存在")
		return
	}
	deleteTenantFiles := auth.IsSuperAdmin() && user.AccountType == "tenant_admin" && user.TenantID > 0
	if !deleteTenantFiles && isTenantPrimaryAdmin(auth.TenantID, id) {
		utils.ErrorWithData(c.Ctx, http.StatusBadRequest, "PRIMARY_TENANT_ADMIN_PROTECTED", map[string]interface{}{"error_code": "PRIMARY_TENANT_ADMIN_PROTECTED"})
		return
	}
	if deleteTenantFiles {
		purgeResult, err := purgeTenantCompletely(auth, user.TenantID)
		if err != nil {
			logControllerOperation(c.Ctx, auth, user.TenantID, "DELETE_USER", "用户管理", "operation.delete_user", map[string]interface{}{"user_id": id, "username": user.Username}, "failed", err.Error())
			utils.Error(c.Ctx, http.StatusBadGateway, "删除租户失败: "+err.Error())
			return
		}
		logControllerOperation(c.Ctx, auth, 0, "DELETE_TENANT", fmt.Sprintf("租户[%d]", user.TenantID), "operation.delete_tenant", map[string]interface{}{"tenant_id": user.TenantID, "admin_user_id": id, "username": user.Username, "tenant_file_cleanup": purgeResult.FileCleanup, "tenant_db_cleanup": purgeResult.DBCleanup}, "success", "")
		utils.Success(c.Ctx, map[string]interface{}{"deleted_id": id, "deleted_admin_user_id": id, "deleted_tenant_id": user.TenantID, "file_cleanup": purgeResult.FileCleanup, "db_cleanup": purgeResult.DBCleanup})
		return
	}
	var rowsAffected int64
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND tenant_id = ?", id, user.TenantID).Delete(&models.User{})
		if result.Error != nil {
			return result.Error
		}
		rowsAffected = result.RowsAffected
		if rowsAffected == 0 {
			return nil
		}
		return tx.Where("user_id = ?", id).Delete(&models.UserRole{}).Error
	})
	if err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "DELETE_USER", "用户管理", "operation.delete_user", map[string]interface{}{"user_id": id}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "删除用户失败")
		return
	}
	if rowsAffected == 0 {
		utils.NotFound(c.Ctx, "用户不存在")
		return
	}
	logControllerOperation(c.Ctx, auth, user.TenantID, "DELETE_USER", "用户管理", "operation.delete_user", map[string]interface{}{"user_id": id, "username": user.Username}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"deleted_id": id})
}

func (c *AccountController) UpdateLanguagePref() {
	auth := currentAuth(c.Ctx)
	var req struct {
		Language string `json:"language"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Language = strings.TrimSpace(req.Language)
	if !isSupportedLanguagePref(req.Language) {
		utils.BadRequest(c.Ctx, "language 不在支持范围内")
		return
	}
	if err := utils.DB.Model(&models.User{}).Where("id = ?", auth.UserID).Update("language_pref", req.Language).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存语言偏好失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"language_pref": req.Language})
}

func roleBelongsToTenant(roleID, tenantID uint) bool {
	var count int64
	utils.DB.Model(&models.Role{}).Where("id = ? AND tenant_id = ?", roleID, tenantID).Count(&count)
	return count > 0
}

func rolesBelongToTenant(roleIDs []uint, tenantID uint) bool {
	roleIDs = normalizeRoleIDs(roleIDs, 0)
	if len(roleIDs) == 0 {
		return false
	}
	var count int64
	utils.DB.Model(&models.Role{}).Where("tenant_id = ? AND id IN ?", tenantID, roleIDs).Count(&count)
	return int(count) == len(roleIDs)
}

func normalizeRoleIDs(roleIDs []uint, fallback uint) []uint {
	seen := map[uint]bool{}
	items := make([]uint, 0, len(roleIDs)+1)
	for _, roleID := range roleIDs {
		if roleID == 0 || seen[roleID] {
			continue
		}
		seen[roleID] = true
		items = append(items, roleID)
	}
	if len(items) == 0 && fallback != 0 {
		items = append(items, fallback)
	}
	return items
}

func replaceUserRoles(tx *gorm.DB, userID uint, roleIDs []uint) error {
	if err := tx.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
		return err
	}
	roles := make([]models.UserRole, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		roles = append(roles, models.UserRole{UserID: userID, RoleID: roleID})
	}
	if len(roles) == 0 {
		return nil
	}
	return tx.Create(&roles).Error
}

func userRoleIDs(user models.User) []uint {
	var roleIDs []uint
	if user.ID != 0 {
		utils.DB.Model(&models.UserRole{}).Where("user_id = ?", user.ID).Order("id asc").Pluck("role_id", &roleIDs)
	}
	return normalizeRoleIDs(roleIDs, user.RoleID)
}

func isSupportedLanguagePref(language string) bool {
	return supportedLanguagePrefs[strings.TrimSpace(language)]
}

func attachUserRoleIDs(users []models.User) []userWithRoles {
	if len(users) == 0 {
		return []userWithRoles{}
	}
	userIDs := make([]uint, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}
	var userRoles []models.UserRole
	utils.DB.Where("user_id IN ?", userIDs).Order("id asc").Find(&userRoles)
	roleMap := map[uint][]uint{}
	for _, item := range userRoles {
		roleMap[item.UserID] = normalizeRoleIDs(append(roleMap[item.UserID], item.RoleID), 0)
	}
	primaryAdminIDs := map[uint]uint{}
	tenantSeen := map[uint]bool{}
	for _, user := range users {
		if user.TenantID == 0 || tenantSeen[user.TenantID] {
			continue
		}
		tenantSeen[user.TenantID] = true
		primaryAdminIDs[user.TenantID] = tenantPrimaryAdminID(user.TenantID)
	}
	items := make([]userWithRoles, 0, len(users))
	for _, user := range users {
		items = append(items, userWithRoles{
			User:           user,
			RoleIDs:        normalizeRoleIDs(roleMap[user.ID], user.RoleID),
			IsPrimaryAdmin: primaryAdminIDs[user.TenantID] == user.ID,
		})
	}
	return items
}

func userPickerOptions(users []models.User) []userPickerOption {
	if len(users) == 0 {
		return []userPickerOption{}
	}
	items := make([]userPickerOption, 0, len(users))
	for _, user := range users {
		items = append(items, userPickerOption{
			ID:           user.ID,
			Username:     user.Username,
			Name:         user.Name,
			Email:        user.Email,
			Phone:        user.Phone,
			RoleID:       user.RoleID,
			DepartmentID: user.DepartmentID,
			PostID:       user.PostID,
			AccountType:  user.AccountType,
			Status:       user.Status,
			LanguagePref: user.LanguagePref,
		})
	}
	return items
}

func tenantPrimaryAdminID(tenantID uint) uint {
	if tenantID == 0 {
		return 0
	}
	var tenant models.Tenant
	if err := utils.DB.Select("id, admin_user_id").Where("id = ?", tenantID).First(&tenant).Error; err == nil && tenant.AdminUserID > 0 {
		return tenant.AdminUserID
	}
	var admin models.User
	if err := utils.DB.Select("id").Where("tenant_id = ? AND account_type = ?", tenantID, "tenant_admin").Order("id asc").First(&admin).Error; err == nil {
		return admin.ID
	}
	return 0
}

func isTenantPrimaryAdmin(tenantID, userID uint) bool {
	return userID > 0 && tenantPrimaryAdminID(tenantID) == userID
}

func canCreateMoreUsers(tenantID uint) (bool, string) {
	var quota models.TenantQuota
	if err := utils.DB.Where("tenant_id = ?", tenantID).First(&quota).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, ""
		}
		return false, "读取租户配额失败"
	}
	if quota.MaxUsers <= 0 {
		return true, ""
	}
	var count int64
	utils.DB.Model(&models.User{}).Where("tenant_id = ? AND status <> ?", tenantID, "deleted").Count(&count)
	if int(count) >= quota.MaxUsers {
		return false, "已达到最大账号数配额"
	}
	return true, ""
}
