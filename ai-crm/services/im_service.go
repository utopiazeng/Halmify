package services

import (
	"ai-crm/models"
	"ai-crm/utils"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const (
	IMStatusOnline  = "online"
	IMStatusAway    = "away"
	IMStatusOffline = "offline"

	IMMessageTypeText  = "text"
	IMMessageTypeEmoji = "emoji"

	IMConversationTypeDirect  = "direct"
	IMConversationTypeSupport = "support"
)

type IMContactItem struct {
	UserID       uint       `json:"user_id"`
	Username     string     `json:"username"`
	Name         string     `json:"name"`
	AvatarURL    string     `json:"avatar_url"`
	DepartmentID uint       `json:"department_id"`
	Department   string     `json:"department"`
	PostID       uint       `json:"post_id"`
	PostName     string     `json:"post_name"`
	Status       string     `json:"status"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
}

type IMConversationItem struct {
	ConversationID         uint       `json:"conversation_id"`
	TenantID               uint       `json:"tenant_id"`
	ConversationType       string     `json:"conversation_type"`
	PeerID                 uint       `json:"peer_id"`
	PeerName               string     `json:"peer_name"`
	PeerUsername           string     `json:"peer_username"`
	PeerAvatarURL          string     `json:"peer_avatar_url"`
	PeerStatus             string     `json:"peer_status"`
	PeerDepartment         string     `json:"peer_department"`
	PeerPostName           string     `json:"peer_post_name"`
	TenantName             string     `json:"tenant_name"`
	Category               string     `json:"category"`
	CategoryLabel          string     `json:"category_label"`
	Priority               string     `json:"priority"`
	PriorityLabel          string     `json:"priority_label"`
	Title                  string     `json:"title"`
	LastMessageID          uint       `json:"last_message_id"`
	LastMessageType        string     `json:"last_message_type"`
	LastMessagePreview     string     `json:"last_message_preview"`
	LastMessagePayloadJSON string     `json:"last_message_payload_json"`
	LastMessageAt          *time.Time `json:"last_message_at"`
	UnreadCount            int64      `json:"unread_count"`
	Mute                   bool       `json:"mute"`
	Pinned                 bool       `json:"pinned"`
	RetentionNoticeText    string     `json:"retention_notice_text"`
}

type IMMessageItem struct {
	ID             uint      `json:"id"`
	ConversationID uint      `json:"conversation_id"`
	SenderID       uint      `json:"sender_id"`
	RecipientID    uint      `json:"recipient_id"`
	Type           string    `json:"type"`
	Content        string    `json:"content"`
	PayloadJSON    string    `json:"payload_json"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type IMUnreadSummary struct {
	TotalUnread int64 `json:"total_unread"`
}

type IMSendInput struct {
	Type                 string
	Content              string
	PayloadJSON          string
	ClientMsgID          string
	PlatformSupportAdmin bool
}

type IMSupportInput struct {
	Category       string
	Title          string
	Content        string
	Priority       string
	MessagePayload map[string]interface{}
}

type IMService struct{}

func NewIMService() *IMService { return &IMService{} }

var ensureIMSupportColumnsOnce sync.Once
var ensureIMSupportColumnsErr error

func ensureIMSupportConversationColumnsRuntime() error {
	ensureIMSupportColumnsOnce.Do(func() {
		if !utils.DB.Migrator().HasColumn(&models.IMConversation{}, "ConversationType") {
			if err := utils.DB.Migrator().AddColumn(&models.IMConversation{}, "ConversationType"); err != nil {
				ensureIMSupportColumnsErr = err
				return
			}
		}
		if !utils.DB.Migrator().HasColumn(&models.IMConversation{}, "Category") {
			if err := utils.DB.Migrator().AddColumn(&models.IMConversation{}, "Category"); err != nil {
				ensureIMSupportColumnsErr = err
				return
			}
		}
		if !utils.DB.Migrator().HasColumn(&models.IMConversation{}, "Priority") {
			if err := utils.DB.Migrator().AddColumn(&models.IMConversation{}, "Priority"); err != nil {
				ensureIMSupportColumnsErr = err
				return
			}
		}
		if !utils.DB.Migrator().HasColumn(&models.IMConversation{}, "Title") {
			if err := utils.DB.Migrator().AddColumn(&models.IMConversation{}, "Title"); err != nil {
				ensureIMSupportColumnsErr = err
				return
			}
		}
		if !utils.DB.Migrator().HasColumn(&models.IMConversation{}, "PayloadJSON") {
			if err := utils.DB.Migrator().AddColumn(&models.IMConversation{}, "PayloadJSON"); err != nil {
				ensureIMSupportColumnsErr = err
				return
			}
		}
		if !utils.DB.Migrator().HasColumn(&models.IMMessage{}, "PayloadJSON") {
			if err := utils.DB.Migrator().AddColumn(&models.IMMessage{}, "PayloadJSON"); err != nil {
				ensureIMSupportColumnsErr = err
				return
			}
		}
		ensureIMSupportColumnsErr = utils.DB.Model(&models.IMConversation{}).
			Where("conversation_type IS NULL OR conversation_type = ''").
			Update("conversation_type", IMConversationTypeDirect).Error
	})
	return ensureIMSupportColumnsErr
}

func normalizePair(a, b uint) (uint, uint) {
	if a < b {
		return a, b
	}
	return b, a
}

var supportCategoryLabels = map[string]string{
	"feedback":           "意见反馈",
	"feature_suggestion": "功能建议",
	"bug":                "故障问题",
	"billing_ai":         "账单/AI费用",
	"account_permission": "账号权限",
	"data_permission":    "数据/权限问题",
	"storage_purchase":   "存储增购/续费",
	"seat_purchase":      "人数增购/续费",
	"entitlement_quote":  "权益报价",
	"other":              "其他",
}

var supportPriorityLabels = map[string]string{
	"normal":   "普通",
	"urgent":   "较急",
	"critical": "紧急",
}

func normalizeSupportCategory(value string) string {
	key := strings.TrimSpace(value)
	if _, ok := supportCategoryLabels[key]; ok {
		return key
	}
	return ""
}

func normalizeSupportPriority(value string) string {
	key := strings.TrimSpace(value)
	if _, ok := supportPriorityLabels[key]; ok {
		return key
	}
	return ""
}

func platformAdminQuery(db *gorm.DB) *gorm.DB {
	return db.Where(
		"tenant_id = 0 AND status = ? AND (account_type IN ? OR role IN ?)",
		"active",
		[]string{"super_admin", "platform_super_admin"},
		[]string{"super_admin", "platform_super_admin"},
	)
}

func isPlatformAdminUserID(db *gorm.DB, userID uint) bool {
	if userID == 0 {
		return false
	}
	var count int64
	_ = platformAdminQuery(db).Where("id = ?", userID).Model(&models.User{}).Count(&count).Error
	return count > 0
}

func listPlatformAdminIDs(db *gorm.DB) ([]uint, error) {
	var users []models.User
	if err := platformAdminQuery(db).Select("id").Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(users))
	for _, user := range users {
		if user.ID > 0 {
			ids = append(ids, user.ID)
		}
	}
	return ids, nil
}

func supportTenantUserID(tx *gorm.DB, conv models.IMConversation) (uint, error) {
	var users []models.User
	if err := tx.Select("id, tenant_id").Where("id IN ?", []uint{conv.UserLow, conv.UserHigh}).Find(&users).Error; err != nil {
		return 0, err
	}
	peerMap := map[uint]models.User{}
	for _, user := range users {
		peerMap[user.ID] = user
	}
	tenantUserID := supportTenantUserIDFromMap(conv, peerMap)
	if tenantUserID == 0 {
		return 0, errors.New("租户侧会话成员不存在")
	}
	return tenantUserID, nil
}

func supportTenantUserIDFromMap(conv models.IMConversation, peerMap map[uint]models.User) uint {
	if user := peerMap[conv.UserLow]; user.ID > 0 && user.TenantID == conv.TenantID {
		return conv.UserLow
	}
	if user := peerMap[conv.UserHigh]; user.ID > 0 && user.TenantID == conv.TenantID {
		return conv.UserHigh
	}
	if user := peerMap[conv.UserLow]; user.ID > 0 && user.TenantID > 0 {
		return conv.UserLow
	}
	if user := peerMap[conv.UserHigh]; user.ID > 0 && user.TenantID > 0 {
		return conv.UserHigh
	}
	return 0
}

func supportCategoryLabel(value string) string {
	if label, ok := supportCategoryLabels[value]; ok {
		return label
	}
	return value
}

func supportPriorityLabel(value string) string {
	if label, ok := supportPriorityLabels[value]; ok {
		return label
	}
	return value
}

func toJSONText(value interface{}) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func sanitizeMessageContent(raw string) string {
	text := strings.TrimSpace(raw)
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}

func (s *IMService) ensureUserPresence(tenantID, userID uint) {
	now := time.Now()
	_ = utils.DB.Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Assign(models.IMUserPresence{
			TenantModel:  models.TenantModel{TenantID: tenantID},
			UserID:       userID,
			Status:       IMStatusOnline,
			LastActiveAt: &now,
			LastSeenAt:   &now,
		}).
		FirstOrCreate(&models.IMUserPresence{}).Error
}

func (s *IMService) ListContacts(tenantID, currentUserID uint, deptID uint, keyword string, statusFilter string, page, size int) ([]IMContactItem, int64, error) {
	query := utils.DB.Model(&models.User{}).
		Where("tenant_id = ? AND status = ?", tenantID, "active").
		Where("id <> ?", currentUserID)
	if deptID > 0 {
		query = query.Where("department_id = ?", deptID)
	}
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		query = query.Where("name LIKE ? OR username LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	var users []models.User
	if err := query.Order("name asc").Offset((page - 1) * size).Limit(size).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	if len(users) == 0 {
		return []IMContactItem{}, total, nil
	}
	userIDs := make([]uint, 0, len(users))
	deptIDs := map[uint]bool{}
	postIDs := map[uint]bool{}
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
		if u.DepartmentID > 0 {
			deptIDs[u.DepartmentID] = true
		}
		if u.PostID > 0 {
			postIDs[u.PostID] = true
		}
	}
	presenceMap := map[uint]models.IMUserPresence{}
	var presences []models.IMUserPresence
	_ = utils.DB.Where("tenant_id = ? AND user_id IN ?", tenantID, userIDs).Find(&presences).Error
	for _, item := range presences {
		presenceMap[item.UserID] = item
	}
	deptNameMap := map[uint]string{}
	if len(deptIDs) > 0 {
		ids := make([]uint, 0, len(deptIDs))
		for id := range deptIDs {
			ids = append(ids, id)
		}
		var depts []models.SystemDepartment
		_ = utils.DB.Where("tenant_id = ? AND id IN ?", tenantID, ids).Find(&depts).Error
		for _, d := range depts {
			deptNameMap[d.ID] = d.Name
		}
	}
	postNameMap := map[uint]string{}
	if len(postIDs) > 0 {
		ids := make([]uint, 0, len(postIDs))
		for id := range postIDs {
			ids = append(ids, id)
		}
		var posts []models.SystemPost
		_ = utils.DB.Where("tenant_id = ? AND id IN ?", tenantID, ids).Find(&posts).Error
		for _, p := range posts {
			postNameMap[p.ID] = p.Name
		}
	}

	items := make([]IMContactItem, 0, len(users))
	for _, u := range users {
		presence := presenceMap[u.ID]
		state := presence.Status
		if state == "" {
			state = IMStatusOffline
		}
		if statusFilter != "" && state != statusFilter {
			continue
		}
		items = append(items, IMContactItem{
			UserID:       u.ID,
			Username:     u.Username,
			Name:         u.Name,
			AvatarURL:    u.AvatarURL,
			DepartmentID: u.DepartmentID,
			Department:   deptNameMap[u.DepartmentID],
			PostID:       u.PostID,
			PostName:     postNameMap[u.PostID],
			Status:       state,
			LastSeenAt:   presence.LastSeenAt,
		})
	}

	statusRank := map[string]int{IMStatusOnline: 0, IMStatusAway: 1, IMStatusOffline: 2}
	sort.Slice(items, func(i, j int) bool {
		ri := statusRank[items[i].Status]
		rj := statusRank[items[j].Status]
		if ri != rj {
			return ri < rj
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, total, nil
}

func (s *IMService) ensureConversation(tx *gorm.DB, tenantID, a, b uint) (models.IMConversation, error) {
	low, high := normalizePair(a, b)
	var conv models.IMConversation
	err := tx.Where("tenant_id = ? AND user_low = ? AND user_high = ?", tenantID, low, high).First(&conv).Error
	if err == nil {
		return conv, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return conv, err
	}
	conv = models.IMConversation{
		TenantModel:      models.TenantModel{TenantID: tenantID},
		UserLow:          low,
		UserHigh:         high,
		ConversationType: IMConversationTypeDirect,
	}
	if err := tx.Create(&conv).Error; err != nil {
		return conv, err
	}
	return conv, nil
}

func (s *IMService) SendMessage(tenantID, senderID uint, conversationID uint, in IMSendInput) (IMMessageItem, uint, error) {
	msgType := strings.TrimSpace(in.Type)
	if msgType == "" {
		msgType = IMMessageTypeText
	}
	if msgType != IMMessageTypeText && msgType != IMMessageTypeEmoji {
		return IMMessageItem{}, 0, errors.New("不支持的消息类型")
	}
	content := sanitizeMessageContent(in.Content)
	if content == "" {
		return IMMessageItem{}, 0, errors.New("消息内容不能为空")
	}
	if len([]rune(content)) > 5000 {
		return IMMessageItem{}, 0, errors.New("文本长度不能超过5000字符")
	}

	var saved models.IMMessage
	var recipientID uint
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		var conv models.IMConversation
		if err := tx.Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&conv).Error; err != nil {
			return err
		}
		platformSupportSender := false
		if conv.UserLow != senderID && conv.UserHigh != senderID {
			if in.PlatformSupportAdmin && conv.ConversationType == IMConversationTypeSupport && isPlatformAdminUserID(tx, senderID) {
				platformSupportSender = true
			}
		}
		if conv.UserLow != senderID && conv.UserHigh != senderID && !platformSupportSender {
			return errors.New("无权在该会话发送消息")
		}
		if platformSupportSender {
			tenantUserID, err := supportTenantUserID(tx, conv)
			if err != nil {
				return err
			}
			recipientID = tenantUserID
		} else if conv.UserLow == senderID {
			recipientID = conv.UserHigh
		} else {
			recipientID = conv.UserLow
		}
		now := time.Now()
		saved = models.IMMessage{
			TenantModel:    models.TenantModel{TenantID: tenantID},
			ConversationID: conv.ID,
			SenderID:       senderID,
			RecipientID:    recipientID,
			Type:           msgType,
			Content:        content,
			PayloadJSON:    strings.TrimSpace(in.PayloadJSON),
			Status:         "sent",
			ClientMsgID:    strings.TrimSpace(in.ClientMsgID),
			ExpiresAt:      now.Add(180 * 24 * time.Hour),
		}
		if err := tx.Create(&saved).Error; err != nil {
			return err
		}
		receipt := models.IMMessageReceipt{
			TenantModel:    models.TenantModel{TenantID: tenantID},
			MessageID:      saved.ID,
			ConversationID: conv.ID,
			RecipientID:    recipientID,
			DeliveredAt:    &now,
		}
		if err := tx.Create(&receipt).Error; err != nil {
			return err
		}
		preview := content
		if msgType == IMMessageTypeEmoji {
			preview = "[emoji] " + preview
		}
		if len([]rune(preview)) > 120 {
			preview = string([]rune(preview)[:120])
		}
		return tx.Model(&models.IMConversation{}).Where("id = ?", conv.ID).Updates(map[string]interface{}{
			"last_message_id":        saved.ID,
			"last_message_sender_id": senderID,
			"last_message_type":      msgType,
			"last_message_preview":   preview,
			"last_message_at":        now,
		}).Error
	})
	if err != nil {
		return IMMessageItem{}, 0, err
	}
	return IMMessageItem{
		ID:             saved.ID,
		ConversationID: saved.ConversationID,
		SenderID:       saved.SenderID,
		RecipientID:    saved.RecipientID,
		Type:           saved.Type,
		Content:        saved.Content,
		PayloadJSON:    saved.PayloadJSON,
		Status:         saved.Status,
		CreatedAt:      saved.CreatedAt,
	}, recipientID, nil
}

func (s *IMService) ConversationTenantForParticipant(conversationID, userID uint, platformAdmin bool) (uint, error) {
	if platformAdmin {
		if err := ensureIMSupportConversationColumnsRuntime(); err != nil {
			return 0, err
		}
	}
	var conv models.IMConversation
	query := utils.DB.Where("id = ?", conversationID)
	if !platformAdmin {
		query = query.Where("conversation_type <> ?", IMConversationTypeSupport)
	}
	if err := query.First(&conv).Error; err != nil {
		return 0, err
	}
	if platformAdmin && conv.ConversationType != IMConversationTypeSupport {
		return 0, errors.New("平台管理员只能访问租户支持会话")
	}
	if platformAdmin {
		if !isPlatformAdminUserID(utils.DB, userID) {
			return 0, errors.New("无权访问该会话")
		}
		return conv.TenantID, nil
	}
	if conv.UserLow != userID && conv.UserHigh != userID {
		return 0, errors.New("无权访问该会话")
	}
	return conv.TenantID, nil
}

func (s *IMService) ListConversationMessages(tenantID, currentUserID, conversationID uint, beforeID uint, limit int, platformSupportAdmin bool) ([]IMMessageItem, error) {
	var conv models.IMConversation
	if err := utils.DB.Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&conv).Error; err != nil {
		return nil, err
	}
	allowedPlatformSupport := platformSupportAdmin && conv.ConversationType == IMConversationTypeSupport && isPlatformAdminUserID(utils.DB, currentUserID)
	if conv.UserLow != currentUserID && conv.UserHigh != currentUserID && !allowedPlatformSupport {
		return nil, errors.New("无权访问该会话")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := utils.DB.Model(&models.IMMessage{}).Where("tenant_id = ? AND conversation_id = ?", tenantID, conversationID)
	if beforeID > 0 {
		query = query.Where("id < ?", beforeID)
	}
	var rows []models.IMMessage
	if err := query.Order("id desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	items := make([]IMMessageItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, IMMessageItem{
			ID:             row.ID,
			ConversationID: row.ConversationID,
			SenderID:       row.SenderID,
			RecipientID:    row.RecipientID,
			Type:           row.Type,
			Content:        row.Content,
			PayloadJSON:    row.PayloadJSON,
			Status:         row.Status,
			CreatedAt:      row.CreatedAt,
		})
	}
	return items, nil
}

func (s *IMService) FindOrCreateConversationByPeer(tenantID, currentUserID, peerID uint) (uint, error) {
	if currentUserID == peerID {
		return 0, errors.New("不能给自己发消息")
	}
	var peer models.User
	if err := utils.DB.Where("tenant_id = ? AND id = ? AND status = ?", tenantID, peerID, "active").First(&peer).Error; err != nil {
		return 0, errors.New("对方用户不存在")
	}
	var conv models.IMConversation
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		c, err := s.ensureConversation(tx, tenantID, currentUserID, peerID)
		if err != nil {
			return err
		}
		conv = c
		return nil
	})
	if err != nil {
		return 0, err
	}
	return conv.ID, nil
}

func (s *IMService) ListConversations(tenantID, currentUserID uint) ([]IMConversationItem, error) {
	var convs []models.IMConversation
	if err := utils.DB.Where("tenant_id = ? AND (user_low = ? OR user_high = ?)", tenantID, currentUserID, currentUserID).
		Order("COALESCE(last_message_at, created_at) desc").
		Find(&convs).Error; err != nil {
		return nil, err
	}
	if len(convs) == 0 {
		return []IMConversationItem{}, nil
	}
	peerIDs := make([]uint, 0, len(convs))
	convIDs := make([]uint, 0, len(convs))
	for _, c := range convs {
		convIDs = append(convIDs, c.ID)
		if c.UserLow == currentUserID {
			peerIDs = append(peerIDs, c.UserHigh)
		} else {
			peerIDs = append(peerIDs, c.UserLow)
		}
	}
	var peers []models.User
	_ = utils.DB.Where("id IN ?", peerIDs).Find(&peers).Error
	peerMap := map[uint]models.User{}
	deptIDs := map[uint]bool{}
	postIDs := map[uint]bool{}
	for _, p := range peers {
		peerMap[p.ID] = p
		if p.DepartmentID > 0 {
			deptIDs[p.DepartmentID] = true
		}
		if p.PostID > 0 {
			postIDs[p.PostID] = true
		}
	}
	deptNameMap := map[uint]string{}
	if len(deptIDs) > 0 {
		ids := make([]uint, 0, len(deptIDs))
		for id := range deptIDs {
			ids = append(ids, id)
		}
		var depts []models.SystemDepartment
		_ = utils.DB.Where("tenant_id = ? AND id IN ?", tenantID, ids).Find(&depts).Error
		for _, d := range depts {
			deptNameMap[d.ID] = d.Name
		}
	}
	postNameMap := map[uint]string{}
	if len(postIDs) > 0 {
		ids := make([]uint, 0, len(postIDs))
		for id := range postIDs {
			ids = append(ids, id)
		}
		var posts []models.SystemPost
		_ = utils.DB.Where("tenant_id = ? AND id IN ?", tenantID, ids).Find(&posts).Error
		for _, p := range posts {
			postNameMap[p.ID] = p.Name
		}
	}
	var presence []models.IMUserPresence
	_ = utils.DB.Where("tenant_id = ? AND user_id IN ?", tenantID, peerIDs).Find(&presence).Error
	presenceMap := map[uint]models.IMUserPresence{}
	for _, p := range presence {
		presenceMap[p.UserID] = p
	}
	var receipts []models.IMMessageReceipt
	receiptQuery := utils.DB.Where("conversation_id IN ? AND recipient_id = ? AND read_at IS NULL", convIDs, currentUserID)
	if tenantID > 0 {
		receiptQuery = receiptQuery.Where("tenant_id = ?", tenantID)
	}
	_ = receiptQuery.Find(&receipts).Error
	unreadByConv := map[uint]int64{}
	for _, r := range receipts {
		unreadByConv[r.ConversationID]++
	}
	var settings []models.IMSessionSetting
	_ = utils.DB.Where("tenant_id = ? AND user_id = ? AND conversation_id IN ?", tenantID, currentUserID, convIDs).Find(&settings).Error
	settingMap := map[uint]models.IMSessionSetting{}
	for _, s := range settings {
		settingMap[s.ConversationID] = s
	}
	lastMessagePayloadMap := map[uint]string{}
	var lastMessages []models.IMMessage
	lastMessageIDs := make([]uint, 0, len(convs))
	for _, c := range convs {
		if c.LastMessageID > 0 {
			lastMessageIDs = append(lastMessageIDs, c.LastMessageID)
		}
	}
	if len(lastMessageIDs) > 0 {
		_ = utils.DB.Select("id, payload_json").Where("id IN ?", lastMessageIDs).Find(&lastMessages).Error
		for _, msg := range lastMessages {
			lastMessagePayloadMap[msg.ID] = msg.PayloadJSON
		}
	}

	items := make([]IMConversationItem, 0, len(convs))
	for _, c := range convs {
		peerID := c.UserLow
		if peerID == currentUserID {
			peerID = c.UserHigh
		}
		peer := peerMap[peerID]
		peerName := peer.Name
		peerUsername := peer.Username
		if c.ConversationType == IMConversationTypeSupport && strings.TrimSpace(peerName) == "" {
			peerName = "系统管理员"
			peerUsername = "system-admin"
		}
		pres := presenceMap[peerID]
		state := pres.Status
		if state == "" {
			state = IMStatusOffline
		}
		setting := settingMap[c.ID]
		items = append(items, IMConversationItem{
			ConversationID:         c.ID,
			TenantID:               c.TenantID,
			ConversationType:       defaultConversationType(c.ConversationType),
			PeerID:                 peerID,
			PeerName:               peerName,
			PeerUsername:           peerUsername,
			PeerAvatarURL:          peer.AvatarURL,
			PeerStatus:             state,
			PeerDepartment:         deptNameMap[peer.DepartmentID],
			PeerPostName:           postNameMap[peer.PostID],
			Category:               c.Category,
			CategoryLabel:          supportCategoryLabel(c.Category),
			Priority:               c.Priority,
			PriorityLabel:          supportPriorityLabel(c.Priority),
			Title:                  c.Title,
			LastMessageID:          c.LastMessageID,
			LastMessageType:        c.LastMessageType,
			LastMessagePreview:     c.LastMessagePreview,
			LastMessagePayloadJSON: lastMessagePayloadMap[c.LastMessageID],
			LastMessageAt:          c.LastMessageAt,
			UnreadCount:            unreadByConv[c.ID],
			Mute:                   setting.Mute,
			Pinned:                 setting.Pinned,
			RetentionNoticeText:    "消息与文件仅保留6个月，逾期自动删除",
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		ti, tj := time.Time{}, time.Time{}
		if items[i].LastMessageAt != nil {
			ti = *items[i].LastMessageAt
		}
		if items[j].LastMessageAt != nil {
			tj = *items[j].LastMessageAt
		}
		return ti.After(tj)
	})
	return items, nil
}

func defaultConversationType(value string) string {
	if strings.TrimSpace(value) == "" {
		return IMConversationTypeDirect
	}
	return strings.TrimSpace(value)
}

func (s *IMService) OpenSupportConversation(tenantID, tenantAdminID uint, in IMSupportInput) (uint, IMMessageItem, uint, error) {
	if err := ensureIMSupportConversationColumnsRuntime(); err != nil {
		return 0, IMMessageItem{}, 0, err
	}
	category := normalizeSupportCategory(in.Category)
	if category == "" {
		return 0, IMMessageItem{}, 0, errors.New("请选择问题类型")
	}
	priority := normalizeSupportPriority(in.Priority)
	if priority == "" {
		return 0, IMMessageItem{}, 0, errors.New("请选择紧急程度")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return 0, IMMessageItem{}, 0, errors.New("标题不能为空")
	}
	content := sanitizeMessageContent(in.Content)
	if content == "" {
		return 0, IMMessageItem{}, 0, errors.New("内容不能为空")
	}
	if len([]rune(title)) > 120 {
		return 0, IMMessageItem{}, 0, errors.New("标题不能超过120个字符")
	}

	var admin models.User
	if err := platformAdminQuery(utils.DB).Order("id asc").First(&admin).Error; err != nil {
		return 0, IMMessageItem{}, 0, errors.New("暂无可联系的系统管理员")
	}
	low, high := normalizePair(tenantAdminID, admin.ID)
	var conv models.IMConversation
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("tenant_id = ? AND user_low = ? AND user_high = ? AND conversation_type = ?", tenantID, low, high, IMConversationTypeSupport).First(&conv).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			conv = models.IMConversation{
				TenantModel:      models.TenantModel{TenantID: tenantID},
				UserLow:          low,
				UserHigh:         high,
				ConversationType: IMConversationTypeSupport,
			}
			if createErr := tx.Create(&conv).Error; createErr != nil {
				return createErr
			}
		} else if err != nil {
			return err
		}
		payload := map[string]interface{}{
			"support_tenant_id": tenantID,
			"tenant_admin_id":   tenantAdminID,
			"platform_admin_id": admin.ID,
			"category":          category,
			"category_label":    supportCategoryLabel(category),
			"priority":          priority,
			"priority_label":    supportPriorityLabel(priority),
			"title":             title,
			"updated_at":        time.Now().Format(time.RFC3339),
		}
		if in.MessagePayload != nil {
			for _, key := range []string{"title_i18n_key", "title_i18n_params", "fallback_title"} {
				if value, ok := in.MessagePayload[key]; ok {
					payload[key] = value
				}
			}
		}
		return tx.Model(&models.IMConversation{}).Where("id = ?", conv.ID).Updates(map[string]interface{}{
			"conversation_type": IMConversationTypeSupport,
			"category":          category,
			"priority":          priority,
			"title":             title,
			"payload_json":      toJSONText(payload),
		}).Error
	})
	if err != nil {
		return 0, IMMessageItem{}, 0, err
	}
	msgText := fmt.Sprintf("【%s｜%s】%s\n%s", supportCategoryLabel(category), supportPriorityLabel(priority), title, content)
	messagePayload := in.MessagePayload
	if messagePayload == nil {
		messagePayload = map[string]interface{}{
			"i18n_key": "im.support.message.content",
			"i18n_params": map[string]interface{}{
				"category": supportCategoryLabel(category),
				"priority": supportPriorityLabel(priority),
				"title":    title,
				"content":  content,
			},
			"fallback_content": msgText,
		}
	}
	msg, recipientID, err := s.SendMessage(tenantID, tenantAdminID, conv.ID, IMSendInput{Type: IMMessageTypeText, Content: msgText, PayloadJSON: toJSONText(messagePayload)})
	if err != nil {
		return 0, IMMessageItem{}, 0, err
	}
	_, _ = s.EnsureSupportReceiptsForPlatformAdmins(tenantID, conv.ID, msg.ID)
	return conv.ID, msg, recipientID, nil
}

func (s *IMService) ListSupportConversations(platformAdminID uint, filters map[string]string) ([]IMConversationItem, error) {
	if err := ensureIMSupportConversationColumnsRuntime(); err != nil {
		return nil, err
	}
	if !isPlatformAdminUserID(utils.DB, platformAdminID) {
		return nil, errors.New("仅系统管理员可查看租户支持会话")
	}
	query := utils.DB.Where("conversation_type = ?", IMConversationTypeSupport)
	if category := normalizeSupportCategory(filters["category"]); category != "" {
		query = query.Where("category = ?", category)
	}
	if priority := normalizeSupportPriority(filters["priority"]); priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if tenantID := strings.TrimSpace(filters["tenant_id"]); tenantID != "" && tenantID != "0" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	var convs []models.IMConversation
	if err := query.Order("COALESCE(last_message_at, created_at) desc").Find(&convs).Error; err != nil {
		return nil, err
	}
	if len(convs) == 0 {
		return []IMConversationItem{}, nil
	}
	convIDs := make([]uint, 0, len(convs))
	userIDs := make([]uint, 0, len(convs)*2)
	tenantIDs := make([]uint, 0, len(convs))
	for _, c := range convs {
		convIDs = append(convIDs, c.ID)
		tenantIDs = append(tenantIDs, c.TenantID)
		userIDs = append(userIDs, c.UserLow, c.UserHigh)
	}
	var peers []models.User
	_ = utils.DB.Select("id, tenant_id, username, name, avatar_url").Where("id IN ?", userIDs).Find(&peers).Error
	peerMap := map[uint]models.User{}
	for _, p := range peers {
		peerMap[p.ID] = p
	}
	var tenants []models.Tenant
	_ = utils.DB.Where("id IN ?", tenantIDs).Find(&tenants).Error
	tenantMap := map[uint]string{}
	for _, t := range tenants {
		tenantMap[t.ID] = t.Name
	}
	var receipts []models.IMMessageReceipt
	_ = utils.DB.Where("conversation_id IN ? AND recipient_id = ? AND read_at IS NULL", convIDs, platformAdminID).Find(&receipts).Error
	unreadByConv := map[uint]int64{}
	for _, r := range receipts {
		unreadByConv[r.ConversationID]++
	}
	unreadOnly := strings.EqualFold(filters["unread_only"], "true") || filters["unread_only"] == "1"
	lastMessagePayloadMap := map[uint]string{}
	var lastMessages []models.IMMessage
	lastMessageIDs := make([]uint, 0, len(convs))
	for _, c := range convs {
		if c.LastMessageID > 0 {
			lastMessageIDs = append(lastMessageIDs, c.LastMessageID)
		}
	}
	if len(lastMessageIDs) > 0 {
		_ = utils.DB.Select("id, payload_json").Where("id IN ?", lastMessageIDs).Find(&lastMessages).Error
		for _, msg := range lastMessages {
			lastMessagePayloadMap[msg.ID] = msg.PayloadJSON
		}
	}
	items := make([]IMConversationItem, 0, len(convs))
	for _, c := range convs {
		if unreadOnly && unreadByConv[c.ID] == 0 {
			continue
		}
		peerID := supportTenantUserIDFromMap(c, peerMap)
		peer := peerMap[peerID]
		items = append(items, IMConversationItem{
			ConversationID:         c.ID,
			TenantID:               c.TenantID,
			ConversationType:       IMConversationTypeSupport,
			PeerID:                 peerID,
			PeerName:               peer.Name,
			PeerUsername:           peer.Username,
			PeerAvatarURL:          peer.AvatarURL,
			PeerStatus:             IMStatusOffline,
			TenantName:             tenantMap[c.TenantID],
			Category:               c.Category,
			CategoryLabel:          supportCategoryLabel(c.Category),
			Priority:               c.Priority,
			PriorityLabel:          supportPriorityLabel(c.Priority),
			Title:                  c.Title,
			LastMessageID:          c.LastMessageID,
			LastMessageType:        c.LastMessageType,
			LastMessagePreview:     c.LastMessagePreview,
			LastMessagePayloadJSON: lastMessagePayloadMap[c.LastMessageID],
			LastMessageAt:          c.LastMessageAt,
			UnreadCount:            unreadByConv[c.ID],
			RetentionNoticeText:    "消息与文件仅保留6个月，逾期自动删除",
		})
	}
	return items, nil
}

func (s *IMService) EnsureSupportReceiptsForPlatformAdmins(tenantID, conversationID, messageID uint) ([]uint, error) {
	adminIDs, err := listPlatformAdminIDs(utils.DB)
	if err != nil || len(adminIDs) == 0 {
		return adminIDs, err
	}
	now := time.Now()
	for _, adminID := range adminIDs {
		var count int64
		if err := utils.DB.Model(&models.IMMessageReceipt{}).
			Where("message_id = ? AND recipient_id = ?", messageID, adminID).
			Count(&count).Error; err != nil {
			return adminIDs, err
		}
		if count > 0 {
			continue
		}
		receipt := models.IMMessageReceipt{
			TenantModel:    models.TenantModel{TenantID: tenantID},
			MessageID:      messageID,
			ConversationID: conversationID,
			RecipientID:    adminID,
			DeliveredAt:    &now,
		}
		if err := utils.DB.Create(&receipt).Error; err != nil {
			return adminIDs, err
		}
	}
	return adminIDs, nil
}

func (s *IMService) MarkConversationRead(tenantID, currentUserID, conversationID uint) error {
	now := time.Now()
	query := utils.DB.Model(&models.IMMessageReceipt{}).
		Where("conversation_id = ? AND recipient_id = ? AND read_at IS NULL", conversationID, currentUserID)
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	return query.Update("read_at", now).Error
}

func (s *IMService) UpdateConversationSetting(tenantID, currentUserID, conversationID uint, mute *bool, pinned *bool) error {
	var conv models.IMConversation
	if err := utils.DB.Where("id = ? AND tenant_id = ?", conversationID, tenantID).First(&conv).Error; err != nil {
		return err
	}
	if conv.UserLow != currentUserID && conv.UserHigh != currentUserID {
		return errors.New("无权设置该会话")
	}

	var setting models.IMSessionSetting
	err := utils.DB.Where("tenant_id = ? AND user_id = ? AND conversation_id = ?", tenantID, currentUserID, conversationID).First(&setting).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = models.IMSessionSetting{
			TenantModel:    models.TenantModel{TenantID: tenantID},
			UserID:         currentUserID,
			ConversationID: conversationID,
		}
	}
	if mute != nil {
		setting.Mute = *mute
	}
	if pinned != nil {
		setting.Pinned = *pinned
		if *pinned {
			now := time.Now()
			setting.PinnedAt = &now
		} else {
			setting.PinnedAt = nil
		}
	}
	if setting.ID == 0 {
		return utils.DB.Create(&setting).Error
	}
	return utils.DB.Save(&setting).Error
}

func (s *IMService) UpdatePresence(tenantID, userID uint, status string, manual bool) error {
	switch status {
	case IMStatusOnline, IMStatusAway, IMStatusOffline:
	default:
		return errors.New("不支持的状态")
	}
	now := time.Now()
	values := map[string]interface{}{
		"status":        status,
		"manual_status": manual,
		"last_seen_at":  now,
	}
	if status == IMStatusOnline {
		values["last_active_at"] = now
	}
	if err := utils.DB.Model(&models.IMUserPresence{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Updates(values).Error; err != nil {
		return err
	}
	var count int64
	_ = utils.DB.Model(&models.IMUserPresence{}).Where("tenant_id = ? AND user_id = ?", tenantID, userID).Count(&count).Error
	if count == 0 {
		rec := models.IMUserPresence{
			TenantModel:  models.TenantModel{TenantID: tenantID},
			UserID:       userID,
			Status:       status,
			ManualStatus: manual,
			LastSeenAt:   &now,
		}
		if status == IMStatusOnline {
			rec.LastActiveAt = &now
		}
		return utils.DB.Create(&rec).Error
	}
	return nil
}

func (s *IMService) UnreadSummary(tenantID, userID uint) (IMUnreadSummary, error) {
	var total int64
	query := utils.DB.Model(&models.IMMessageReceipt{}).Where("recipient_id = ? AND read_at IS NULL", userID)
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.Count(&total).Error; err != nil {
		return IMUnreadSummary{}, err
	}
	return IMUnreadSummary{TotalUnread: total}, nil
}

func (s *IMService) CleanupExpiredMessages(retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 180
	}
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	job := models.IMCleanupJob{
		Status:    "running",
		StartedAt: time.Now(),
		Reason:    fmt.Sprintf("retention_days=%d", retentionDays),
	}
	if err := utils.DB.Create(&job).Error; err != nil {
		return err
	}

	var msgIDs []uint
	if err := utils.DB.Model(&models.IMMessage{}).
		Where("expires_at <= ?", cutoff).
		Limit(5000).
		Pluck("id", &msgIDs).Error; err != nil {
		return err
	}
	deletedMessages := int64(0)
	deletedReceipts := int64(0)
	if len(msgIDs) > 0 {
		if err := utils.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("message_id IN ?", msgIDs).Delete(&models.IMMessageReceipt{}).Error; err != nil {
				return err
			}
			deletedReceipts = tx.RowsAffected
			if err := tx.Where("id IN ?", msgIDs).Delete(&models.IMMessage{}).Error; err != nil {
				return err
			}
			deletedMessages = tx.RowsAffected
			return nil
		}); err != nil {
			_ = utils.DB.Model(&models.IMCleanupJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
				"status":      "failed",
				"finished_at": time.Now(),
				"reason":      err.Error(),
			}).Error
			return err
		}
	}
	now := time.Now()
	return utils.DB.Model(&models.IMCleanupJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
		"status":           "done",
		"finished_at":      now,
		"deleted_messages": deletedMessages,
		"deleted_receipts": deletedReceipts,
	}).Error
}
