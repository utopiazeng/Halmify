package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
)

type SystemController struct {
	web.Controller
}

type systemTreeNode struct {
	models.SystemDepartment
	Children []systemTreeNode `json:"children,omitempty"`
}

type menuTreeNode struct {
	models.SystemMenu
	Children []menuTreeNode `json:"children,omitempty"`
}

func (c *SystemController) ListDepartments() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	items := []models.SystemDepartment{}
	query := utils.DB.Where("tenant_id = ?", auth.TenantID)
	keyword := strings.TrimSpace(c.Ctx.Input.Query("keyword"))
	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ? OR leader LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if err := query.Order("parent_id asc, sort asc, id asc").Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取部门失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": buildDepartmentTree(items)})
}

func (c *SystemController) CreateDepartment() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var req models.SystemDepartment
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Code) == "" {
		utils.BadRequest(c.Ctx, "部门名称和部门编码不能为空")
		return
	}
	if req.Sort <= 0 {
		req.Sort = 1
	}
	if req.Status == "" {
		req.Status = "active"
	}
	req.ID = 0
	req.TenantID = auth.TenantID
	if req.ParentID > 0 && !departmentBelongsToTenant(req.ParentID, auth.TenantID) {
		utils.BadRequest(c.Ctx, "上级部门不存在")
		return
	}
	if err := utils.DB.Create(&req).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建部门失败")
		return
	}
	utils.Success(c.Ctx, req)
}

func (c *SystemController) UpdateDepartment() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的部门ID")
		return
	}
	var req models.SystemDepartment
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var item models.SystemDepartment
	if err := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).First(&item).Error; err != nil {
		utils.NotFound(c.Ctx, "部门不存在")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Code) == "" {
		utils.BadRequest(c.Ctx, "部门名称和部门编码不能为空")
		return
	}
	if req.ParentID == id {
		utils.BadRequest(c.Ctx, "上级部门不能为自身")
		return
	}
	if req.ParentID > 0 && !departmentBelongsToTenant(req.ParentID, auth.TenantID) {
		utils.BadRequest(c.Ctx, "上级部门不存在")
		return
	}
	item.ParentID = req.ParentID
	item.Name = strings.TrimSpace(req.Name)
	item.Code = strings.TrimSpace(req.Code)
	item.Leader = req.Leader
	item.Phone = req.Phone
	item.Email = req.Email
	item.Sort = req.Sort
	if item.Sort <= 0 {
		item.Sort = 1
	}
	item.Status = normalizeStatus(req.Status)
	item.Remark = req.Remark
	if err := utils.DB.Save(&item).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新部门失败")
		return
	}
	utils.Success(c.Ctx, item)
}

func (c *SystemController) DeleteDepartment() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的部门ID")
		return
	}
	var childCount int64
	utils.DB.Model(&models.SystemDepartment{}).Where("tenant_id = ? AND parent_id = ?", auth.TenantID, id).Count(&childCount)
	if childCount > 0 {
		utils.BadRequest(c.Ctx, "存在子部门，不能删除")
		return
	}
	var userCount int64
	utils.DB.Model(&models.User{}).Where("tenant_id = ? AND department_id = ?", auth.TenantID, id).Count(&userCount)
	if userCount > 0 {
		utils.BadRequest(c.Ctx, "部门仍有用户使用，不能删除")
		return
	}
	result := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).Delete(&models.SystemDepartment{})
	if result.Error != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "删除部门失败")
		return
	}
	if result.RowsAffected == 0 {
		utils.NotFound(c.Ctx, "部门不存在")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"deleted_id": id})
}

func (c *SystemController) ListPosts() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)
	query := utils.DB.Model(&models.SystemPost{}).Where("tenant_id = ?", auth.TenantID)
	keyword := strings.TrimSpace(c.Ctx.Input.Query("keyword"))
	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	query.Count(&total)
	items := []models.SystemPost{}
	if err := query.Order("sort asc, id asc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取岗位失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": total, "page": page, "size": size})
}

func (c *SystemController) CreatePost() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var req models.SystemPost
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Code) == "" {
		utils.BadRequest(c.Ctx, "岗位名称和岗位编码不能为空")
		return
	}
	req.ID = 0
	req.TenantID = auth.TenantID
	if req.Sort <= 0 {
		req.Sort = 1
	}
	req.Status = normalizeStatus(req.Status)
	if err := utils.DB.Create(&req).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建岗位失败")
		return
	}
	utils.Success(c.Ctx, req)
}

func (c *SystemController) UpdatePost() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的岗位ID")
		return
	}
	var req models.SystemPost
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var item models.SystemPost
	if err := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).First(&item).Error; err != nil {
		utils.NotFound(c.Ctx, "岗位不存在")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Code) == "" {
		utils.BadRequest(c.Ctx, "岗位名称和岗位编码不能为空")
		return
	}
	item.Name = strings.TrimSpace(req.Name)
	item.Code = strings.TrimSpace(req.Code)
	item.Sort = req.Sort
	if item.Sort <= 0 {
		item.Sort = 1
	}
	item.Status = normalizeStatus(req.Status)
	item.Remark = req.Remark
	if err := utils.DB.Save(&item).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新岗位失败")
		return
	}
	utils.Success(c.Ctx, item)
}

func (c *SystemController) DeletePost() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的岗位ID")
		return
	}
	var userCount int64
	utils.DB.Model(&models.User{}).Where("tenant_id = ? AND post_id = ?", auth.TenantID, id).Count(&userCount)
	if userCount > 0 {
		utils.BadRequest(c.Ctx, "岗位仍有用户使用，不能删除")
		return
	}
	result := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).Delete(&models.SystemPost{})
	if result.Error != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "删除岗位失败")
		return
	}
	if result.RowsAffected == 0 {
		utils.NotFound(c.Ctx, "岗位不存在")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"deleted_id": id})
}

func (c *SystemController) ListMenus() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Error(c.Ctx, http.StatusForbidden, "缺少租户上下文")
		return
	}
	visibleOnly := strings.TrimSpace(c.Ctx.Input.Query("visible")) == "1"
	if !visibleOnly {
		if _, ok := requireTenantAdmin(c.Ctx); !ok {
			return
		}
	}
	items := []models.SystemMenu{}
	query := utils.DB.Where("tenant_id = ?", auth.TenantID)
	if visibleOnly {
		query = query.Where("status = ? AND hidden = ?", "active", false)
	}
	if err := query.Order("parent_id asc, sort asc, id asc").Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取菜单失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": buildMenuTree(items)})
}

func (c *SystemController) CreateMenu() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var req models.SystemMenu
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		utils.BadRequest(c.Ctx, "菜单名称不能为空")
		return
	}
	req.ID = 0
	req.TenantID = auth.TenantID
	if req.Sort <= 0 {
		req.Sort = 1
	}
	if req.Type == "" {
		req.Type = "menu"
	}
	req.Status = normalizeStatus(req.Status)
	if req.ParentID > 0 && !menuBelongsToTenant(req.ParentID, auth.TenantID) {
		utils.BadRequest(c.Ctx, "上级菜单不存在")
		return
	}
	if err := utils.DB.Create(&req).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建菜单失败")
		return
	}
	utils.Success(c.Ctx, req)
}

func (c *SystemController) UpdateMenu() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的菜单ID")
		return
	}
	var req models.SystemMenu
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var item models.SystemMenu
	if err := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).First(&item).Error; err != nil {
		utils.NotFound(c.Ctx, "菜单不存在")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		utils.BadRequest(c.Ctx, "菜单名称不能为空")
		return
	}
	if req.ParentID == id {
		utils.BadRequest(c.Ctx, "上级菜单不能为自身")
		return
	}
	if req.ParentID > 0 && !menuBelongsToTenant(req.ParentID, auth.TenantID) {
		utils.BadRequest(c.Ctx, "上级菜单不存在")
		return
	}
	item.ParentID = req.ParentID
	item.Type = req.Type
	if item.Type == "" {
		item.Type = "menu"
	}
	item.Name = strings.TrimSpace(req.Name)
	item.Path = strings.TrimSpace(req.Path)
	item.Permission = strings.TrimSpace(req.Permission)
	item.Icon = strings.TrimSpace(req.Icon)
	item.Sort = req.Sort
	if item.Sort <= 0 {
		item.Sort = 1
	}
	item.Status = normalizeStatus(req.Status)
	item.Hidden = req.Hidden
	item.ShowBadge = req.ShowBadge
	item.ComponentKey = req.ComponentKey
	if err := utils.DB.Save(&item).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新菜单失败")
		return
	}
	utils.Success(c.Ctx, item)
}

func (c *SystemController) DeleteMenu() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的菜单ID")
		return
	}
	var childCount int64
	utils.DB.Model(&models.SystemMenu{}).Where("tenant_id = ? AND parent_id = ?", auth.TenantID, id).Count(&childCount)
	if childCount > 0 {
		utils.BadRequest(c.Ctx, "存在子菜单，不能删除")
		return
	}
	result := utils.DB.Where("id = ? AND tenant_id = ?", id, auth.TenantID).Delete(&models.SystemMenu{})
	if result.Error != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "删除菜单失败")
		return
	}
	if result.RowsAffected == 0 {
		utils.NotFound(c.Ctx, "菜单不存在")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"deleted_id": id})
}

func buildDepartmentTree(items []models.SystemDepartment) []systemTreeNode {
	children := map[uint][]models.SystemDepartment{}
	for _, item := range items {
		children[item.ParentID] = append(children[item.ParentID], item)
	}
	var walk func(parent uint) []systemTreeNode
	walk = func(parent uint) []systemTreeNode {
		nodes := []systemTreeNode{}
		for _, item := range children[parent] {
			nodes = append(nodes, systemTreeNode{SystemDepartment: item, Children: walk(item.ID)})
		}
		return nodes
	}
	return walk(0)
}

func buildMenuTree(items []models.SystemMenu) []menuTreeNode {
	children := map[uint][]models.SystemMenu{}
	for _, item := range items {
		children[item.ParentID] = append(children[item.ParentID], item)
	}
	var walk func(parent uint) []menuTreeNode
	walk = func(parent uint) []menuTreeNode {
		nodes := []menuTreeNode{}
		for _, item := range children[parent] {
			nodes = append(nodes, menuTreeNode{SystemMenu: item, Children: walk(item.ID)})
		}
		return nodes
	}
	return walk(0)
}

func departmentBelongsToTenant(id, tenantID uint) bool {
	var count int64
	utils.DB.Model(&models.SystemDepartment{}).Where("id = ? AND tenant_id = ?", id, tenantID).Count(&count)
	return count > 0
}

func postBelongsToTenant(id, tenantID uint) bool {
	if id == 0 {
		return true
	}
	var count int64
	utils.DB.Model(&models.SystemPost{}).Where("id = ? AND tenant_id = ?", id, tenantID).Count(&count)
	return count > 0
}

func menuBelongsToTenant(id, tenantID uint) bool {
	var count int64
	utils.DB.Model(&models.SystemMenu{}).Where("id = ? AND tenant_id = ?", id, tenantID).Count(&count)
	return count > 0
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" || status == "enabled" {
		return "active"
	}
	if status == "disabled" {
		return "disabled"
	}
	return status
}

func roleBelongsToTenantWithDB(tx *gorm.DB, roleID, tenantID uint) bool {
	var count int64
	tx.Model(&models.Role{}).Where("id = ? AND tenant_id = ?", roleID, tenantID).Count(&count)
	return count > 0
}
