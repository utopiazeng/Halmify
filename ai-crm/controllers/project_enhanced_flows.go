package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

type projectProgressRequest struct {
	ProgressPercent int    `json:"progress_percent"`
	Phase           string `json:"phase"`
	HealthStatus    string `json:"health_status"`
	NextStep        string `json:"next_step"`
	RiskSummary     string `json:"risk_summary"`
	NextFollowAt    string `json:"next_follow_at"`
	DueDate         string `json:"due_date"`
	Reason          string `json:"reason"`
}

type projectFollowUpRequest struct {
	Content         string `json:"content"`
	NextFollowAt    string `json:"next_follow_at"`
	ProgressPercent int    `json:"progress_percent"`
	Visibility      string `json:"visibility"`
}

type projectReminderRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	RemindAt string `json:"remind_at"`
	OwnerIDs []uint `json:"owner_ids"`
}

type projectMilestoneRequest struct {
	MilestoneID uint   `json:"milestone_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	DueDate     string `json:"due_date"`
	Remark      string `json:"remark"`
}

func (c *CRMController) UpdateProjectProgress() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的项目ID")
		return
	}
	if err := ensureCRMModuleAccess(auth, "projects", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req projectProgressRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	project, ok := loadCRMParentObject(c.Ctx, auth, "projects", "project", id)
	if !ok {
		return
	}
	if !canManageCRMObject(auth, project) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权更新项目进度")
		return
	}
	payload := payloadMap(project.PayloadJSON)
	if req.ProgressPercent < 0 {
		req.ProgressPercent = 0
	}
	if req.ProgressPercent > 100 {
		req.ProgressPercent = 100
	}
	if req.ProgressPercent > 0 || fmt.Sprint(payload["progress"]) == "" {
		payload["progress"] = req.ProgressPercent
		project.Score = req.ProgressPercent
	}
	if value := strings.TrimSpace(req.Phase); value != "" {
		payload["phase"] = value
	}
	if value := normalizeProjectHealth(req.HealthStatus); value != "" {
		payload["health_status"] = value
	}
	if value := strings.TrimSpace(req.NextStep); value != "" {
		payload["next_step"] = value
	}
	if value := strings.TrimSpace(req.RiskSummary); value != "" {
		payload["risk_summary"] = value
	}
	nextFollowAt, err := parseOptionalTime(req.NextFollowAt)
	if err != nil {
		utils.BadRequest(c.Ctx, "下次跟进时间格式无效")
		return
	}
	if nextFollowAt != nil {
		payload["next_follow_at"] = nextFollowAt.Format(time.RFC3339)
	}
	dueDate, err := parseOptionalTime(req.DueDate)
	if err != nil {
		utils.BadRequest(c.Ctx, "计划完成时间格式无效")
		return
	}
	if dueDate != nil {
		payload["due_date"] = dueDate.Format("2006-01-02")
	}
	project.PayloadJSON = toJSONString(payload)
	project.UpdatedBy = auth.UserID
	if project.Score >= 100 {
		project.Status = "completed"
	} else if health := strings.TrimSpace(fmt.Sprint(payload["health_status"])); health == "blocked" || health == "risk" {
		project.Status = "blocked"
	} else if project.Status == "planning" {
		project.Status = "active"
	}
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&project).Error; err != nil {
			return err
		}
		if nextFollowAt != nil {
			if err := createWorkspaceReminder(tx, auth, project, "项目跟进提醒："+project.Name, "项目设置了下次跟进时间，请按时推进。", nextFollowAt, nil, "project_follow_reminder", "project_follow"); err != nil {
				return err
			}
		}
		return createBusinessTimeline(tx, auth, "projects", project.ID, "project_progress", "更新项目进度", map[string]interface{}{"reason": strings.TrimSpace(req.Reason), "progress": project.Score, "phase": payload["phase"], "health_status": payload["health_status"], "next_follow_at": payload["next_follow_at"], "due_date": payload["due_date"]})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新项目进度失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"project": project})
}

func (c *CRMController) AddProjectFollowUp() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的项目ID")
		return
	}
	if err := ensureCRMModuleAccess(auth, "projects", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req projectFollowUpRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		utils.BadRequest(c.Ctx, "跟进内容不能为空")
		return
	}
	project, ok := loadCRMParentObject(c.Ctx, auth, "projects", "project", id)
	if !ok {
		return
	}
	if !canFollowCRMObject(auth, project) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权跟进该项目")
		return
	}
	payload := payloadMap(project.PayloadJSON)
	if req.ProgressPercent >= 0 && req.ProgressPercent <= 100 {
		payload["progress"] = req.ProgressPercent
		project.Score = req.ProgressPercent
	}
	nextFollowAt, err := parseOptionalTime(req.NextFollowAt)
	if err != nil {
		utils.BadRequest(c.Ctx, "下次跟进时间格式无效")
		return
	}
	if nextFollowAt != nil {
		payload["next_follow_at"] = nextFollowAt.Format(time.RFC3339)
	}
	payload["last_follow_at"] = time.Now().Format(time.RFC3339)
	payload["last_follow_by"] = auth.UserID
	project.PayloadJSON = toJSONString(payload)
	project.UpdatedBy = auth.UserID
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&project).Error; err != nil {
			return err
		}
		if nextFollowAt != nil {
			if err := createWorkspaceReminder(tx, auth, project, "项目下次跟进："+project.Name, req.Content, nextFollowAt, nil, "project_follow_reminder", "project_follow"); err != nil {
				return err
			}
		}
		return createBusinessTimeline(tx, auth, "projects", project.ID, "project_follow_up", "添加项目跟进", map[string]interface{}{"reason": req.Content, "content": req.Content, "visibility": normalizeCRMFileVisibility(req.Visibility), "next_follow_at": payload["next_follow_at"], "progress": project.Score})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "添加项目跟进失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"project": project})
}

func (c *CRMController) CreateProjectReminder() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的项目ID")
		return
	}
	if err := ensureCRMModuleAccess(auth, "projects", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req projectReminderRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	project, ok := loadCRMParentObject(c.Ctx, auth, "projects", "project", id)
	if !ok {
		return
	}
	if !canFollowCRMObject(auth, project) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权设置该项目提醒")
		return
	}
	remindAt, err := parseOptionalTime(req.RemindAt)
	if err != nil {
		utils.BadRequest(c.Ctx, "提醒时间格式无效")
		return
	}
	title := defaultText(req.Title, "项目提醒："+project.Name)
	content := defaultText(req.Content, "请跟进项目进度。")
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := createWorkspaceReminder(tx, auth, project, title, content, remindAt, req.OwnerIDs, "project_manual_reminder", "project_reminder"); err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "projects", project.ID, "project_reminder", "创建项目提醒", map[string]interface{}{"reason": content, "title": title, "remind_at": req.RemindAt, "owner_ids": req.OwnerIDs})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建项目提醒失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"created": true})
}

func (c *CRMController) SetProjectOwner() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的项目ID")
		return
	}
	var req struct {
		OwnerID uint   `json:"owner_id"`
		Reason  string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	project, ok := loadCRMParentObject(c.Ctx, auth, "projects", "project", id)
	if !ok {
		return
	}
	if !canManageCRMObject(auth, project) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权变更项目负责人")
		return
	}
	before := project.OwnerID
	project.OwnerID = req.OwnerID
	project.UpdatedBy = auth.UserID
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&project).Error; err != nil {
			return err
		}
		if req.OwnerID > 0 {
			if err := createWorkspaceReminder(tx, auth, project, "项目负责人变更："+project.Name, "你已成为该项目负责人。", nil, []uint{req.OwnerID}, "project_owner_notice", "project_owner"); err != nil {
				return err
			}
		}
		return createBusinessTimeline(tx, auth, "projects", project.ID, "project_owner_change", "项目负责人变更", map[string]interface{}{"reason": strings.TrimSpace(req.Reason), "from_owner_id": before, "to_owner_id": req.OwnerID})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "变更项目负责人失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"project": project})
}

func (c *CRMController) SaveProjectMilestone() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的项目ID")
		return
	}
	var req projectMilestoneRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		utils.BadRequest(c.Ctx, "里程碑名称不能为空")
		return
	}
	project, ok := loadCRMParentObject(c.Ctx, auth, "projects", "project", id)
	if !ok {
		return
	}
	if !canManageCRMObject(auth, project) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权维护项目里程碑")
		return
	}
	payload := map[string]interface{}{"name": req.Name, "status": defaultText(req.Status, "open"), "due_date": strings.TrimSpace(req.DueDate), "remark": strings.TrimSpace(req.Remark), "project_id": project.ID}
	milestone := models.CRMObject{}
	if req.MilestoneID > 0 {
		_ = utils.DB.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ? AND project_id = ?", auth.TenantID, "projects", "milestone", req.MilestoneID, project.ID).First(&milestone).Error
	}
	if milestone.ID == 0 {
		milestone = models.CRMObject{TenantModel: models.TenantModel{TenantID: auth.TenantID}, Domain: "projects", ObjectType: "milestone", Name: req.Name, Status: defaultText(req.Status, "open"), OwnerID: project.OwnerID, ProjectID: project.ID, PayloadJSON: toJSONString(payload), CreatedBy: auth.UserID, UpdatedBy: auth.UserID}
	} else {
		milestone.Name = req.Name
		milestone.Status = defaultText(req.Status, milestone.Status)
		milestone.PayloadJSON = toJSONString(payload)
		milestone.UpdatedBy = auth.UserID
	}
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if milestone.ID == 0 {
			if err := tx.Create(&milestone).Error; err != nil {
				return err
			}
			if err := tx.Create(&models.CRMRelation{TenantModel: models.TenantModel{TenantID: auth.TenantID}, SourceType: "projects", SourceID: project.ID, TargetType: "projects", TargetID: milestone.ID, RelationType: "project_milestone"}).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&milestone).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "projects", project.ID, "project_milestone", "里程碑变更", map[string]interface{}{"milestone_id": milestone.ID, "name": req.Name, "status": milestone.Status, "due_date": req.DueDate, "reason": req.Remark})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存里程碑失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"milestone": milestone})
}

var projectReminderSchedulerStop chan struct{}

func StartProjectReminderScheduler() {
	if projectReminderSchedulerStop != nil {
		return
	}
	projectReminderSchedulerStop = make(chan struct{})
	ticker := time.NewTicker(6 * time.Hour)
	go func() {
		services.WithSchedulerLock("project_reminder_scheduler", runProjectReminderSchedule)
		for {
			select {
			case <-ticker.C:
				services.WithSchedulerLock("project_reminder_scheduler", runProjectReminderSchedule)
			case <-projectReminderSchedulerStop:
				ticker.Stop()
				return
			}
		}
	}()
}

func runProjectReminderSchedule() {
	var tenants []models.Tenant
	if err := utils.DB.Where("status = ?", "active").Find(&tenants).Error; err != nil {
		return
	}
	for _, tenant := range tenants {
		auth := authContext{TenantID: tenant.ID, UserID: tenantAdminUserID(tenant.ID), AccountType: "tenant_admin", Username: "system"}
		_ = runProjectReminderTenant(auth)
	}
}

func runProjectReminderTenant(auth authContext) error {
	now := time.Now()
	var projects []models.CRMObject
	if err := utils.DB.Where("tenant_id = ? AND domain = ? AND object_type = ? AND status NOT IN ?", auth.TenantID, "projects", "project", []string{"completed", "closed", "cancelled"}).Limit(500).Find(&projects).Error; err != nil {
		return err
	}
	return utils.DB.Transaction(func(tx *gorm.DB) error {
		for _, project := range projects {
			payload := payloadMap(project.PayloadJSON)
			ownerID := project.OwnerID
			if ownerID == 0 {
				ownerID = project.CreatedBy
			}
			if ownerID == 0 {
				continue
			}
			if err := maybeCreateProjectDateNotice(tx, auth, &project, payload, now, ownerID, "next_follow_at", 24*time.Hour, "project_next_follow_notice", "项目跟进临近提醒", "项目下次跟进时间临近，请及时处理。"); err != nil {
				return err
			}
			if err := maybeCreateProjectDateNotice(tx, auth, &project, payload, now, ownerID, "due_date", 72*time.Hour, "project_due_notice", "项目计划完成提醒", "项目计划完成时间临近或已逾期，请检查交付风险。"); err != nil {
				return err
			}
			lastFollowAt, _ := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload["last_follow_at"])))
			if lastFollowAt == nil && project.UpdatedAt.Before(now.AddDate(0, 0, -7)) && strings.TrimSpace(fmt.Sprint(payload["long_idle_notice_date"])) != now.Format("2006-01-02") {
				payload["long_idle_notice_date"] = now.Format("2006-01-02")
				project.PayloadJSON = toJSONString(payload)
				if err := tx.Save(&project).Error; err != nil {
					return err
				}
				if err := createWorkspaceReminder(tx, auth, project, "项目长期未跟进："+project.Name, "项目超过7天没有跟进记录，请确认进度。", nil, []uint{ownerID}, "project_idle_notice", "project_idle"); err != nil {
					return err
				}
				if err := createBusinessTimeline(tx, auth, "projects", project.ID, "project_idle_notice", "项目长期未跟进提醒", map[string]interface{}{"reason": "超过7天没有跟进记录"}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func maybeCreateProjectDateNotice(tx *gorm.DB, auth authContext, project *models.CRMObject, payload map[string]interface{}, now time.Time, ownerID uint, field string, window time.Duration, eventType string, title string, content string) error {
	dueAt, err := parseOptionalTime(strings.TrimSpace(fmt.Sprint(payload[field])))
	if err != nil || dueAt == nil {
		return nil
	}
	if dueAt.After(now.Add(window)) {
		return nil
	}
	noticeKey := eventType + "_date"
	if strings.TrimSpace(fmt.Sprint(payload[noticeKey])) == now.Format("2006-01-02") {
		return nil
	}
	payload[noticeKey] = now.Format("2006-01-02")
	project.PayloadJSON = toJSONString(payload)
	if err := tx.Save(project).Error; err != nil {
		return err
	}
	if err := createWorkspaceReminder(tx, auth, *project, title+"："+project.Name, content, dueAt, []uint{ownerID}, eventType, eventType); err != nil {
		return err
	}
	return createBusinessTimeline(tx, auth, "projects", project.ID, eventType, title, map[string]interface{}{"reason": content, field: dueAt.Format(time.RFC3339)})
}

func normalizeProjectHealth(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "healthy", "normal", "risk", "blocked", "done":
		return value
	default:
		return ""
	}
}
