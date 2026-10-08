package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
)

type IMController struct {
	web.Controller
}

var imSvc = services.NewIMService()

type systemUnreadCacheEntry struct {
	Unread    int64
	ExpiresAt time.Time
}

var systemUnreadCache = struct {
	sync.Mutex
	Items map[string]systemUnreadCacheEntry
}{
	Items: map[string]systemUnreadCacheEntry{},
}

func (c *IMController) ListContacts() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)
	items, total, err := imSvc.ListContacts(
		auth.TenantID,
		auth.UserID,
		queryUint(c.Ctx, "dept_id"),
		strings.TrimSpace(c.Ctx.Input.Query("q")),
		strings.TrimSpace(c.Ctx.Input.Query("status")),
		page,
		size,
	)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取联系人失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"items": items,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func (c *IMController) OpenConversation() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	var req struct {
		PeerID uint `json:"peer_id"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	convID, err := imSvc.FindOrCreateConversationByPeer(auth.TenantID, auth.UserID, req.PeerID)
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"conversation_id": convID})
}

func (c *IMController) ListConversations() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		if auth.IsPlatformAdmin() {
			c.ListSupportConversations()
			return
		}
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	items, err := imSvc.ListConversations(auth.TenantID, auth.UserID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取会话列表失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items})
}

func (c *IMController) SendMessage() {
	auth := currentAuth(c.Ctx)
	conversationID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效会话ID")
		return
	}
	tenantID := auth.TenantID
	if tenantID == 0 && auth.IsPlatformAdmin() {
		resolved, err := imSvc.ConversationTenantForParticipant(conversationID, auth.UserID, true)
		if err != nil {
			utils.Forbidden(c.Ctx, err.Error())
			return
		}
		tenantID = resolved
	}
	if tenantID == 0 {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	var req struct {
		Type        string `json:"type"`
		Content     string `json:"content"`
		ClientMsgID string `json:"client_msg_id"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	msg, peerID, err := imSvc.SendMessage(tenantID, auth.UserID, conversationID, services.IMSendInput{
		Type:                 req.Type,
		Content:              req.Content,
		ClientMsgID:          req.ClientMsgID,
		PlatformSupportAdmin: auth.TenantID == 0 && auth.IsPlatformAdmin(),
	})
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	recipientPushTenantID := tenantID
	senderPushTenantID := tenantID
	if auth.IsPlatformAdmin() {
		senderPushTenantID = 0
	} else if isPlatformUserID(peerID) {
		recipientPushTenantID = 0
	}
	services.PublishIMEvent("im.message.new", recipientPushTenantID, peerID, msg)
	services.PublishIMEvent("im.message.ack", senderPushTenantID, auth.UserID, map[string]interface{}{
		"conversation_id": conversationID,
		"message_id":      msg.ID,
		"status":          "sent",
	})
	if summary, sumErr := imSvc.UnreadSummary(recipientPushTenantID, peerID); sumErr == nil {
		services.PublishIMEvent("im.conversation.unread", recipientPushTenantID, peerID, summary)
	}
	utils.Success(c.Ctx, msg)
}

func (c *IMController) ListMessages() {
	auth := currentAuth(c.Ctx)
	conversationID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效会话ID")
		return
	}
	tenantID := auth.TenantID
	if tenantID == 0 && auth.IsPlatformAdmin() {
		resolved, err := imSvc.ConversationTenantForParticipant(conversationID, auth.UserID, true)
		if err != nil {
			utils.Forbidden(c.Ctx, err.Error())
			return
		}
		tenantID = resolved
	}
	if tenantID == 0 {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	items, err := imSvc.ListConversationMessages(
		tenantID,
		auth.UserID,
		conversationID,
		queryUint(c.Ctx, "before_id"),
		queryInt(c.Ctx, "limit", 20),
		auth.TenantID == 0 && auth.IsPlatformAdmin(),
	)
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items})
}

func (c *IMController) MarkRead() {
	auth := currentAuth(c.Ctx)
	conversationID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效会话ID")
		return
	}
	tenantID := auth.TenantID
	if tenantID == 0 && auth.IsPlatformAdmin() {
		resolved, err := imSvc.ConversationTenantForParticipant(conversationID, auth.UserID, true)
		if err != nil {
			utils.Forbidden(c.Ctx, err.Error())
			return
		}
		tenantID = resolved
	}
	if tenantID == 0 {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	if err := imSvc.MarkConversationRead(tenantID, auth.UserID, conversationID); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "标记已读失败")
		return
	}
	pushTenantID := tenantID
	if auth.IsPlatformAdmin() {
		pushTenantID = 0
	}
	if summary, err := imSvc.UnreadSummary(pushTenantID, auth.UserID); err == nil {
		services.PublishIMEvent("im.conversation.unread", pushTenantID, auth.UserID, summary)
	}
	utils.Success(c.Ctx, map[string]interface{}{"conversation_id": conversationID, "read": true})
}

func (c *IMController) UpdateConversationSetting() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	conversationID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效会话ID")
		return
	}
	var req struct {
		Mute   *bool `json:"mute"`
		Pinned *bool `json:"pinned"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.Mute == nil && req.Pinned == nil {
		utils.BadRequest(c.Ctx, "至少传入 mute 或 pinned")
		return
	}
	if err := imSvc.UpdateConversationSetting(auth.TenantID, auth.UserID, conversationID, req.Mute, req.Pinned); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"conversation_id": conversationID, "updated": true})
}

func (c *IMController) UpdatePresence() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	var req struct {
		Status string `json:"status"`
		Manual bool   `json:"manual"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if err := imSvc.UpdatePresence(auth.TenantID, auth.UserID, strings.TrimSpace(req.Status), req.Manual); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	services.PublishIMEvent("im.presence.changed", auth.TenantID, auth.UserID, map[string]interface{}{
		"user_id": auth.UserID,
		"status":  req.Status,
	})
	utils.Success(c.Ctx, map[string]interface{}{"status": req.Status})
}

func (c *IMController) UnreadSummary() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		if auth.IsPlatformAdmin() {
			summary, err := imSvc.UnreadSummary(0, auth.UserID)
			if err != nil {
				utils.Error(c.Ctx, http.StatusInternalServerError, "获取未读失败")
				return
			}
			utils.Success(c.Ctx, summary)
			return
		}
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	summary, err := imSvc.UnreadSummary(auth.TenantID, auth.UserID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取未读失败")
		return
	}
	utils.Success(c.Ctx, summary)
}

func (c *IMController) OpenSupportConversation() {
	auth := currentAuth(c.Ctx)
	if !auth.IsTenantAdmin() || auth.TenantID == 0 {
		utils.Forbidden(c.Ctx, "仅企业管理员可联系系统管理员")
		return
	}
	var req struct {
		Category string `json:"category"`
		Title    string `json:"title"`
		Content  string `json:"content"`
		Priority string `json:"priority"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	convID, msg, recipientID, err := imSvc.OpenSupportConversation(auth.TenantID, auth.UserID, services.IMSupportInput{
		Category: req.Category,
		Title:    req.Title,
		Content:  req.Content,
		Priority: req.Priority,
	})
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	if platformAdminIDs, receiptErr := imSvc.EnsureSupportReceiptsForPlatformAdmins(auth.TenantID, convID, msg.ID); receiptErr == nil && len(platformAdminIDs) > 0 {
		for _, platformAdminID := range platformAdminIDs {
			services.PublishIMEvent("im.message.new", 0, platformAdminID, msg)
			if summary, sumErr := imSvc.UnreadSummary(0, platformAdminID); sumErr == nil {
				services.PublishIMEvent("im.conversation.unread", 0, platformAdminID, summary)
			}
		}
	} else {
		services.PublishIMEvent("im.message.new", 0, recipientID, msg)
		if summary, sumErr := imSvc.UnreadSummary(0, recipientID); sumErr == nil {
			services.PublishIMEvent("im.conversation.unread", 0, recipientID, summary)
		}
	}
	services.PublishIMEvent("im.message.ack", auth.TenantID, auth.UserID, map[string]interface{}{
		"conversation_id": convID,
		"message_id":      msg.ID,
		"status":          "sent",
	})
	utils.Success(c.Ctx, map[string]interface{}{
		"conversation_id": convID,
		"message":         msg,
	})
}

func (c *IMController) ListSupportConversations() {
	auth := currentAuth(c.Ctx)
	if !auth.IsPlatformAdmin() {
		utils.Forbidden(c.Ctx, "仅系统管理员可查看租户支持会话")
		return
	}
	items, err := imSvc.ListSupportConversations(auth.UserID, map[string]string{
		"category":    strings.TrimSpace(c.Ctx.Input.Query("category")),
		"priority":    strings.TrimSpace(c.Ctx.Input.Query("priority")),
		"unread_only": strings.TrimSpace(c.Ctx.Input.Query("unread_only")),
		"tenant_id":   strings.TrimSpace(c.Ctx.Input.Query("tenant_id")),
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取租户支持会话失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items})
}

func (c *IMController) SystemNotifications() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 && !auth.IsPlatformAdmin() {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	status := strings.TrimSpace(c.Ctx.Input.Query("status"))
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)
	if size <= 0 || size > 100 {
		size = 20
	}
	query := personalSystemNotificationBaseQuery(auth)
	switch status {
	case "", "active":
		query = query.Where("status IN ?", []string{"open", "new", "read"})
	case "unread":
		query = query.Where("status IN ?", []string{"open", "new"})
	case "read":
		query = query.Where("status = ?", "read")
	case "archived":
		query = query.Where("status = ?", "archived")
	case "all":
	default:
		query = query.Where("status = ?", status)
	}
	var total int64
	query.Count(&total)
	var items []models.CRMObject
	if err := query.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取系统消息失败")
		return
	}
	unread, _ := systemNotificationUnreadCount(auth)
	utils.Success(c.Ctx, map[string]interface{}{
		"items":  items,
		"total":  total,
		"unread": unread,
		"page":   page,
		"size":   size,
	})
}

type systemNotificationsArchiveRequest struct {
	IDs    []uint `json:"ids"`
	Status string `json:"status"`
}

type systemNotificationsCleanupRequest struct {
	BeforeDays int `json:"before_days"`
}

type systemNotificationBroadcastRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Category string `json:"category"`
	Priority string `json:"priority"`
}

func (c *IMController) SystemNotificationRead() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 && !auth.IsPlatformAdmin() {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的通知ID")
		return
	}
	var notification models.CRMObject
	if err := personalSystemNotificationBaseQuery(auth).Where("id = ?", id).First(&notification).Error; err != nil {
		utils.NotFound(c.Ctx, "通知不存在")
		return
	}
	payload := payloadMap(notification.PayloadJSON)
	payload["read_at"] = time.Now().Format(time.RFC3339)
	payload["read_by"] = auth.UserID
	notification.Status = "read"
	notification.PayloadJSON = toJSONString(payload)
	notification.UpdatedBy = auth.UserID
	if err := utils.DB.Save(&notification).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "通知标记已读失败")
		return
	}
	publishSystemNotificationUnread(auth, notification.OwnerID)
	utils.Success(c.Ctx, map[string]interface{}{"notification": notification})
}

type officialDemoRequest struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Remark    string `json:"remark"`
	Locale    string `json:"locale"`
	Source    string `json:"source"`
	SourceIP  string `json:"source_ip"`
	UserAgent string `json:"user_agent"`
}

var nonBusinessDemoEmailDomains = map[string]bool{
	"126.com": true, "139.com": true, "163.com": true, "aol.com": true, "att.net": true, "comcast.net": true,
	"duck.com": true, "facebook.com": true, "fastmail.com": true, "foxmail.com": true, "free.fr": true,
	"gmail.com": true, "googlemail.com": true, "gmx.com": true, "gmx.de": true, "hotmail.co.uk": true,
	"hotmail.com": true, "icloud.com": true, "inbox.com": true, "live.com": true, "mail.com": true,
	"mail.ru": true, "me.com": true, "msn.com": true, "naver.com": true, "outlook.com": true,
	"pm.me": true, "proton.me": true, "protonmail.com": true, "qq.com": true, "sina.com": true,
	"sohu.com": true, "t-online.de": true, "tutanota.com": true, "web.de": true, "yahoo.co.jp": true,
	"yahoo.co.uk": true, "yahoo.com": true, "yandex.com": true, "yandex.ru": true, "yeah.net": true,
	"zoho.com": true,
}

var disposableDemoEmailDomains = map[string]bool{
	"10minutemail.com": true, "20minutemail.com": true, "anonaddy.com": true, "dispostable.com": true,
	"guerrillamail.com": true, "mailinator.com": true, "mailnesia.com": true, "moakt.com": true,
	"sharklasers.com": true, "temp-mail.org": true, "tempmail.com": true, "throwawaymail.com": true,
	"trashmail.com": true, "yopmail.com": true,
}

func (c *IMController) OfficialDemoRequest() {
	expectedToken, _ := web.AppConfig.String("official_site_internal_token")
	expectedToken = strings.TrimSpace(expectedToken)
	if expectedToken == "" || c.Ctx.Input.Header("X-Official-Site-Token") != expectedToken {
		utils.Forbidden(c.Ctx, "无权提交官网预约")
		return
	}
	var req officialDemoRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Remark = strings.TrimSpace(req.Remark)
	req.Locale = strings.TrimSpace(req.Locale)
	req.Source = strings.TrimSpace(req.Source)
	req.SourceIP = strings.TrimSpace(req.SourceIP)
	req.UserAgent = strings.TrimSpace(req.UserAgent)
	if req.Name == "" || req.Email == "" {
		utils.BadRequest(c.Ctx, "姓名和企业邮箱不能为空")
		return
	}
	if len([]rune(req.Name)) > 80 || len([]rune(req.Remark)) > 1000 {
		utils.BadRequest(c.Ctx, "预约信息长度超出限制")
		return
	}
	if err := validateOfficialDemoBusinessEmail(req.Email); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	var admins []models.User
	if err := utils.DB.
		Where("tenant_id = 0 AND status = ? AND (account_type IN ? OR role IN ?)", "active", []string{"super_admin", "platform_super_admin"}, []string{"super_admin", "platform_super_admin"}).
		Order("id asc").
		Find(&admins).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取系统管理员失败")
		return
	}
	if len(admins) == 0 {
		utils.Error(c.Ctx, http.StatusInternalServerError, "没有可接收官网预约的系统管理员")
		return
	}
	now := time.Now()
	content := fmt.Sprintf("官网收到新的预约演示申请：%s（%s）", req.Name, req.Email)
	if req.Remark != "" {
		content = fmt.Sprintf("%s\n备注：%s", content, req.Remark)
	}
	notifications := make([]models.CRMObject, 0, len(admins))
	for _, admin := range admins {
		payload := map[string]interface{}{
			"category":     "official_demo_request",
			"priority":     "important",
			"content":      content,
			"summary":      content,
			"sender_name":  "官网预约",
			"demo_name":    req.Name,
			"demo_email":   req.Email,
			"demo_remark":  req.Remark,
			"locale":       req.Locale,
			"source":       req.Source,
			"source_ip":    req.SourceIP,
			"user_agent":   req.UserAgent,
			"published_at": now.Format(time.RFC3339),
		}
		notifications = append(notifications, models.CRMObject{
			TenantModel: models.TenantModel{TenantID: 0},
			Domain:      "workspace",
			ObjectType:  "mail_notification",
			Name:        "官网预约演示申请",
			Code:        "official_demo_request",
			Status:      "new",
			OwnerID:     admin.ID,
			PayloadJSON: toJSONString(payload),
		})
	}
	if err := utils.DB.CreateInBatches(&notifications, 100).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建官网预约通知失败")
		return
	}
	for _, admin := range admins {
		publishPlatformSystemNotificationUnread(admin.ID)
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"published":       true,
		"recipient_count": len(admins),
	})
}

func validateOfficialDemoBusinessEmail(value string) error {
	_, err := mail.ParseAddress(value)
	if err != nil {
		return fmt.Errorf("企业邮箱格式无效")
	}
	parts := strings.Split(value, "@")
	if len(parts) != 2 || parts[0] == "" || !strings.Contains(parts[1], ".") {
		return fmt.Errorf("企业邮箱格式无效")
	}
	domain := strings.ToLower(strings.TrimSuffix(parts[1], "."))
	if domain == "" || strings.Contains(domain, "..") || net.ParseIP(domain) != nil {
		return fmt.Errorf("企业邮箱格式无效")
	}
	if nonBusinessDemoEmailDomains[domain] || disposableDemoEmailDomains[domain] {
		return fmt.Errorf("请使用企业邮箱地址，暂不接受个人或临时邮箱")
	}
	if strings.HasSuffix(domain, ".local") || strings.HasSuffix(domain, ".test") || strings.HasSuffix(domain, ".invalid") {
		return fmt.Errorf("请使用可验证的企业邮箱域名")
	}
	mx, err := net.LookupMX(domain)
	if err != nil || len(mx) == 0 {
		return fmt.Errorf("企业邮箱域名未通过 MX 记录验证")
	}
	return nil
}

func (c *IMController) SystemNotificationsReadAll() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 && !auth.IsPlatformAdmin() {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	result := personalSystemNotificationBaseQuery(auth).
		Where("status IN ?", []string{"open", "new"}).
		Updates(map[string]interface{}{
			"status":     "read",
			"updated_by": auth.UserID,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "全部标记已读失败")
		return
	}
	invalidateSystemNotificationUnreadCache(auth.TenantID, auth.UserID)
	unread, _ := systemNotificationUnreadCount(auth)
	utils.Success(c.Ctx, map[string]interface{}{
		"updated": result.RowsAffected,
		"unread":  unread,
	})
}

func (c *IMController) ArchiveSystemNotifications() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 && !auth.IsPlatformAdmin() {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	var req systemNotificationsArchiveRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	query := personalSystemNotificationBaseQuery(auth).Where("status <> ?", "archived")
	if len(req.IDs) > 0 {
		query = query.Where("id IN ?", req.IDs)
	} else {
		switch strings.TrimSpace(req.Status) {
		case "", "read":
			query = query.Where("status = ?", "read")
		case "unread":
			query = query.Where("status IN ?", []string{"open", "new"})
		case "active":
			query = query.Where("status IN ?", []string{"open", "new", "read"})
		case "all":
		default:
			utils.BadRequest(c.Ctx, "不支持的清理条件")
			return
		}
	}
	result := query.Updates(map[string]interface{}{
		"status":     "archived",
		"updated_by": auth.UserID,
		"updated_at": time.Now(),
	})
	if result.Error != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "归档系统消息失败")
		return
	}
	invalidateSystemNotificationUnreadCache(auth.TenantID, auth.UserID)
	unread, _ := systemNotificationUnreadCount(auth)
	utils.Success(c.Ctx, map[string]interface{}{
		"archived": result.RowsAffected,
		"unread":   unread,
	})
}

func (c *IMController) CleanupSystemNotifications() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 && !auth.IsPlatformAdmin() {
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	var req systemNotificationsCleanupRequest
	if len(c.Ctx.Input.RequestBody) > 0 {
		if !bindJSON(c.Ctx, &req) {
			return
		}
	}
	beforeDays := req.BeforeDays
	if beforeDays <= 0 {
		beforeDays = 30
	}
	cutoff := time.Now().AddDate(0, 0, -beforeDays)
	result := personalSystemNotificationBaseQuery(auth).
		Where("status = ?", "read").
		Where("created_at < ?", cutoff).
		Updates(map[string]interface{}{
			"status":     "archived",
			"updated_by": auth.UserID,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "清理系统消息失败")
		return
	}
	invalidateSystemNotificationUnreadCache(auth.TenantID, auth.UserID)
	unread, _ := systemNotificationUnreadCount(auth)
	utils.Success(c.Ctx, map[string]interface{}{
		"archived":    result.RowsAffected,
		"before_days": beforeDays,
		"unread":      unread,
	})
}

func (c *IMController) BroadcastSystemNotification() {
	auth := currentAuth(c.Ctx)
	if !auth.IsTenantAdmin() || auth.TenantID == 0 {
		utils.Forbidden(c.Ctx, "仅企业管理员可发布全员通知")
		return
	}
	var req systemNotificationBroadcastRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	req.Category = strings.TrimSpace(req.Category)
	req.Priority = strings.TrimSpace(req.Priority)
	if req.Title == "" || req.Content == "" {
		utils.BadRequest(c.Ctx, "通知标题和内容不能为空")
		return
	}
	if req.Category == "" {
		req.Category = "announcement"
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}
	var users []models.User
	if err := utils.DB.
		Where("tenant_id = ? AND status = ?", auth.TenantID, "active").
		Order("id asc").
		Find(&users).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取通知接收人失败")
		return
	}
	if len(users) == 0 {
		utils.BadRequest(c.Ctx, "当前租户没有可接收通知的启用用户")
		return
	}
	now := time.Now()
	senderName := auth.Username
	if senderName == "" {
		senderName = "企业管理员"
	}
	notifications := make([]models.CRMObject, 0, len(users))
	for _, user := range users {
		payload := map[string]interface{}{
			"category":       req.Category,
			"priority":       req.Priority,
			"content":        req.Content,
			"summary":        req.Content,
			"sender_name":    senderName,
			"sender_user_id": auth.UserID,
			"broadcast":      true,
			"published_at":   now.Format(time.RFC3339),
		}
		notifications = append(notifications, models.CRMObject{
			TenantModel:  models.TenantModel{TenantID: auth.TenantID},
			Domain:       "workspace",
			ObjectType:   "mail_notification",
			Name:         req.Title,
			Code:         "tenant_broadcast",
			Status:       "new",
			OwnerID:      user.ID,
			DepartmentID: user.DepartmentID,
			PayloadJSON:  toJSONString(payload),
			CreatedBy:    auth.UserID,
			UpdatedBy:    auth.UserID,
		})
	}
	if err := utils.DB.CreateInBatches(&notifications, 200).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "发布全员通知失败")
		return
	}
	for _, user := range users {
		invalidateSystemNotificationUnreadCache(auth.TenantID, user.ID)
		publishSystemNotificationUnread(auth, user.ID)
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "BROADCAST_SYSTEM_NOTIFICATION", req.Title, "operation.broadcast_system_notification", map[string]interface{}{"recipient_count": len(users), "category": req.Category, "priority": req.Priority}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{
		"published":       true,
		"recipient_count": len(users),
	})
}

func (c *IMController) SystemNotificationUnreadCount() {
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		if auth.IsPlatformAdmin() {
			unread, err := systemNotificationUnreadCount(auth)
			if err != nil {
				utils.Error(c.Ctx, http.StatusInternalServerError, "获取系统未读数失败")
				return
			}
			utils.Success(c.Ctx, map[string]interface{}{
				"unread":     unread,
				"updated_at": time.Now().Format(time.RFC3339),
			})
			return
		}
		utils.Forbidden(c.Ctx, "缺少租户上下文")
		return
	}
	unread, err := systemNotificationUnreadCount(auth)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取系统未读数失败")
		return
	}
	c.Ctx.Output.Header("Cache-Control", "private, max-age=30")
	utils.Success(c.Ctx, map[string]interface{}{
		"unread":     unread,
		"updated_at": time.Now().Format(time.RFC3339),
	})
}

func personalSystemNotificationBaseQuery(auth authContext) *gorm.DB {
	return utils.DB.Model(&models.CRMObject{}).
		Where("tenant_id = ? AND domain = ? AND object_type IN ? AND owner_id = ?", auth.TenantID, "workspace", []string{"mail_notification", "customer_activity"}, auth.UserID)
}

func systemNotificationUnreadCount(auth authContext) (int64, error) {
	cacheKey := fmt.Sprintf("%d:%d", auth.TenantID, auth.UserID)
	now := time.Now()
	systemUnreadCache.Lock()
	if item, ok := systemUnreadCache.Items[cacheKey]; ok && item.ExpiresAt.After(now) {
		systemUnreadCache.Unlock()
		return item.Unread, nil
	}
	systemUnreadCache.Unlock()

	var unread int64
	err := personalSystemNotificationBaseQuery(auth).
		Where("object_type IN ? AND owner_id = ? AND status IN ?", []string{"mail_notification", "customer_activity"}, auth.UserID, []string{"open", "new"}).
		Count(&unread).Error
	if err == nil {
		systemUnreadCache.Lock()
		systemUnreadCache.Items[cacheKey] = systemUnreadCacheEntry{
			Unread:    unread,
			ExpiresAt: now.Add(30 * time.Second),
		}
		systemUnreadCache.Unlock()
	}
	return unread, err
}

func invalidateSystemNotificationUnreadCache(tenantID uint, userID uint) {
	systemUnreadCache.Lock()
	delete(systemUnreadCache.Items, fmt.Sprintf("%d:%d", tenantID, userID))
	systemUnreadCache.Unlock()
}

func publishSystemNotificationUnread(auth authContext, userID uint) {
	if auth.TenantID == 0 && auth.IsPlatformAdmin() {
		publishPlatformSystemNotificationUnread(userID)
		return
	}
	if auth.TenantID == 0 || userID == 0 {
		return
	}
	invalidateSystemNotificationUnreadCache(auth.TenantID, userID)
	go func() {
		time.Sleep(500 * time.Millisecond)
		targetAuth := auth
		targetAuth.UserID = userID
		unread, err := systemNotificationUnreadCount(targetAuth)
		if err != nil {
			return
		}
		services.PublishIMEvent("im.system.unread", auth.TenantID, userID, map[string]interface{}{
			"unread":     unread,
			"updated_at": time.Now().Format(time.RFC3339),
		})
	}()
}

func publishPlatformSystemNotificationUnread(userID uint) {
	if userID == 0 {
		return
	}
	auth := authContext{TenantID: 0, UserID: userID, AccountType: "super_admin", Role: "super_admin"}
	invalidateSystemNotificationUnreadCache(0, userID)
	go func() {
		time.Sleep(500 * time.Millisecond)
		unread, err := systemNotificationUnreadCount(auth)
		if err != nil {
			return
		}
		services.PublishIMEvent("im.system.unread", 0, userID, map[string]interface{}{
			"unread":     unread,
			"updated_at": time.Now().Format(time.RFC3339),
		})
	}()
}

func isPlatformUserID(userID uint) bool {
	if userID == 0 {
		return false
	}
	var user models.User
	if err := utils.DB.Select("id,tenant_id,account_type,role").Where("id = ?", userID).First(&user).Error; err != nil {
		return false
	}
	return isPlatformAdminIdentity(user.AccountType, user.Role, user.TenantID)
}

func (c *IMController) WS() {
	c.EnableRender = false
	auth := currentAuth(c.Ctx)
	if auth.TenantID == 0 {
		if !auth.IsPlatformAdmin() {
			utils.Forbidden(c.Ctx, "缺少租户上下文")
			return
		}
	}
	if strings.ToLower(c.Ctx.Request.Header.Get("Upgrade")) != "websocket" {
		utils.BadRequest(c.Ctx, "仅支持 websocket 连接")
		return
	}
	key := strings.TrimSpace(c.Ctx.Request.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		utils.BadRequest(c.Ctx, "缺少 Sec-WebSocket-Key")
		return
	}

	conn, err := upgradeWebSocket(c.Ctx.ResponseWriter, c.Ctx.Request, key)
	if err != nil {
		return
	}
	defer conn.Close()

	if auth.TenantID > 0 {
		_ = imSvc.UpdatePresence(auth.TenantID, auth.UserID, services.IMStatusOnline, false)
		services.PublishIMEvent("im.presence.changed", auth.TenantID, auth.UserID, map[string]interface{}{"user_id": auth.UserID, "status": services.IMStatusOnline})
	}
	listenerTenantID := auth.TenantID
	if auth.IsPlatformAdmin() {
		listenerTenantID = 0
	}
	listener, cleanup := services.RegisterIMListener(listenerTenantID, auth.UserID)
	defer cleanup()

	_ = writeWSJSON(conn, map[string]interface{}{
		"event":     "im.system.retention_notice",
		"tenant_id": listenerTenantID,
		"user_id":   auth.UserID,
		"timestamp": time.Now().UnixMilli(),
		"data": map[string]interface{}{
			"text": "消息与文件仅保留6个月，逾期自动删除",
		},
	})
	if unread, err := systemNotificationUnreadCount(auth); err == nil && auth.TenantID > 0 {
		_ = writeWSJSON(conn, services.IMEvent{
			Event:     "im.system.unread",
			TenantID:  auth.TenantID,
			UserID:    auth.UserID,
			Timestamp: time.Now().UnixMilli(),
			Data: map[string]interface{}{
				"unread":     unread,
				"updated_at": time.Now().Format(time.RFC3339),
			},
		})
	}

	for {
		select {
		case evt, ok := <-listener:
			if !ok {
				return
			}
			if err := writeWSJSON(conn, evt); err != nil {
				if auth.TenantID > 0 {
					_ = imSvc.UpdatePresence(auth.TenantID, auth.UserID, services.IMStatusOffline, false)
					services.PublishIMEvent("im.presence.changed", auth.TenantID, auth.UserID, map[string]interface{}{"user_id": auth.UserID, "status": services.IMStatusOffline})
				}
				return
			}
		case <-time.After(20 * time.Second):
			if err := writeWSPing(conn); err != nil {
				if auth.TenantID > 0 {
					_ = imSvc.UpdatePresence(auth.TenantID, auth.UserID, services.IMStatusOffline, false)
					services.PublishIMEvent("im.presence.changed", auth.TenantID, auth.UserID, map[string]interface{}{"user_id": auth.UserID, "status": services.IMStatusOffline})
				}
				return
			}
		}
	}
}

func websocketAcceptKey(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func upgradeWebSocket(w http.ResponseWriter, _ *http.Request, key string) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("response writer does not support hijacking")
	}
	conn, bufrw, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}
	accept := websocketAcceptKey(key)
	_, _ = bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	_, _ = bufrw.WriteString("Upgrade: websocket\r\n")
	_, _ = bufrw.WriteString("Connection: Upgrade\r\n")
	_, _ = bufrw.WriteString("Sec-WebSocket-Accept: " + accept + "\r\n\r\n")
	if err := bufrw.Flush(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

type wsConn interface {
	Write([]byte) (int, error)
	Close() error
}

func writeWSFrame(conn wsConn, opcode byte, payload []byte) error {
	header := []byte{0x80 | opcode}
	l := len(payload)
	switch {
	case l < 126:
		header = append(header, byte(l))
	case l <= 65535:
		header = append(header, 126, byte(l>>8), byte(l))
	default:
		header = append(header, 127,
			byte(uint64(l)>>56), byte(uint64(l)>>48), byte(uint64(l)>>40), byte(uint64(l)>>32),
			byte(uint64(l)>>24), byte(uint64(l)>>16), byte(uint64(l)>>8), byte(uint64(l)),
		)
	}
	if _, err := conn.Write(header); err != nil {
		return err
	}
	if l > 0 {
		if _, err := conn.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

func writeWSJSON(conn wsConn, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return writeWSFrame(conn, 0x1, data)
}

func writeWSPing(conn wsConn) error {
	return writeWSFrame(conn, 0x9, []byte("ping"))
}
