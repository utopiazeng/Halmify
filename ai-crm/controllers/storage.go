package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	beegoctx "github.com/beego/beego/v2/server/web/context"
	"gorm.io/gorm"
)

const (
	baseStorageBytes       int64 = 1024 * 1024 * 1024
	storagePricePerGBYear        = 6
	storagePriceCents            = int64(storagePricePerGBYear * 100)
	seatPricePerMonth            = 1
	seatPriceCentsPerMonth       = int64(seatPricePerMonth * 100)
	seatPriceCentsPerYear        = seatPriceCentsPerMonth * 12
	billingDaysPerYear           = 365.0
)

type StorageController struct {
	web.Controller
}

type tenantStorageUsage struct {
	TenantID              uint                            `json:"tenant_id"`
	UsedBytes             uint64                          `json:"used_bytes"`
	UsedGB                float64                         `json:"used_gb"`
	BaseBytes             int64                           `json:"base_bytes"`
	BaseGB                int                             `json:"base_gb"`
	ExpansionBytes        int64                           `json:"expansion_bytes"`
	ExpansionGB           int                             `json:"expansion_gb"`
	TotalBytes            int64                           `json:"total_bytes"`
	TotalGB               float64                         `json:"total_gb"`
	RemainingBytes        int64                           `json:"remaining_bytes"`
	UsagePercent          float64                         `json:"usage_percent"`
	UploadLocked          bool                            `json:"upload_locked"`
	LockReason            string                          `json:"lock_reason"`
	ExpiringSoon          bool                            `json:"expiring_soon"`
	DaysToExpire          int                             `json:"days_to_expire"`
	NextExpireAt          *time.Time                      `json:"next_expire_at"`
	CurrentExpireAt       *time.Time                      `json:"current_expire_at"`
	UnifiedExpireAt       *time.Time                      `json:"unified_expire_at"`
	ExpiredExpansionGB    int                             `json:"expired_expansion_gb"`
	PricePerGBYear        int                             `json:"price_per_gb_year"`
	Currency              string                          `json:"currency"`
	ExpansionRecords      []tenantStorageExpansionPayload `json:"expansion_records"`
	HistoryRecords        []tenantStorageExpansionPayload `json:"history_records"`
	PendingRequestRecords []tenantStorageExpansionPayload `json:"pending_request_records"`
}

type tenantStorageExpansionPayload struct {
	ID                 uint       `json:"id"`
	CapacityGB         int        `json:"capacity_gb"`
	Years              int        `json:"years"`
	PricePerGB         int64      `json:"price_per_gb"`
	Currency           string     `json:"currency"`
	TotalAmount        int64      `json:"total_amount"`
	TotalAmountCents   int64      `json:"total_amount_cents"`
	TotalAmountDisplay string     `json:"total_amount_display"`
	Source             string     `json:"source"`
	Status             string     `json:"status"`
	ValidFrom          time.Time  `json:"valid_from"`
	ValidTo            time.Time  `json:"valid_to"`
	DaysLeft           int        `json:"days_left"`
	Remark             string     `json:"remark"`
	ConfirmedAt        *time.Time `json:"confirmed_at"`
}

type storagePurchaseRequest struct {
	CapacityGB int    `json:"capacity_gb"`
	Years      int    `json:"years"`
	Remark     string `json:"remark"`
}

type platformStorageExpansionRequest struct {
	CapacityGB int    `json:"capacity_gb"`
	Years      int    `json:"years"`
	Source     string `json:"source"`
	Status     string `json:"status"`
	ValidFrom  string `json:"valid_from"`
	ValidTo    string `json:"valid_to"`
	Remark     string `json:"remark"`
}

type entitlementQuoteRequest struct {
	Resource string `json:"resource"`
	Quantity int    `json:"quantity"`
	Years    int    `json:"years"`
	Remark   string `json:"remark"`
}

type entitlementQuotePayload struct {
	Resource             string                  `json:"resource"`
	UnitLabel            string                  `json:"unit_label"`
	Currency             string                  `json:"currency"`
	UnitPriceCents       int64                   `json:"unit_price_cents"`
	UnitPriceDisplay     string                  `json:"unit_price_display"`
	CurrentQuantity      int                     `json:"current_quantity"`
	AdditionalQuantity   int                     `json:"additional_quantity"`
	TargetQuantity       int                     `json:"target_quantity"`
	TargetYears          int                     `json:"target_years"`
	TargetMonths         int                     `json:"target_months"`
	BillingMultiplier    int                     `json:"billing_multiplier"`
	BillingUnit          string                  `json:"billing_unit"`
	CurrentExpireAt      *time.Time              `json:"current_expire_at"`
	TargetValidTo        time.Time               `json:"target_valid_to"`
	CurrentCreditCents   int64                   `json:"current_credit_cents"`
	CurrentCreditDisplay string                  `json:"current_credit_display"`
	TargetGrossCents     int64                   `json:"target_gross_cents"`
	TargetGrossDisplay   string                  `json:"target_gross_display"`
	DueCents             int64                   `json:"due_cents"`
	DueDisplay           string                  `json:"due_display"`
	Formula              string                  `json:"formula"`
	FormulaSummary       string                  `json:"formula_summary"`
	CreditLines          []entitlementCreditLine `json:"credit_lines"`
	Snapshot             map[string]interface{}  `json:"snapshot"`
}

type entitlementCreditLine struct {
	ID             uint       `json:"id"`
	Quantity       int        `json:"quantity"`
	Source         string     `json:"source"`
	Status         string     `json:"status"`
	ValidTo        time.Time  `json:"valid_to"`
	DaysLeft       int        `json:"days_left"`
	CreditCents    int64      `json:"credit_cents"`
	CreditDisplay  string     `json:"credit_display"`
	Included       bool       `json:"included"`
	ExcludedReason string     `json:"excluded_reason"`
	ConfirmedAt    *time.Time `json:"confirmed_at"`
}

type tenantSeatUsage struct {
	TenantID              uint                         `json:"tenant_id"`
	CurrentUsers          int64                        `json:"current_users"`
	MaxUsers              int                          `json:"max_users"`
	SeatCount             int                          `json:"seat_count"`
	UsagePercent          float64                      `json:"usage_percent"`
	CurrentExpireAt       *time.Time                   `json:"current_expire_at"`
	UnifiedExpireAt       *time.Time                   `json:"unified_expire_at"`
	NextExpireAt          *time.Time                   `json:"next_expire_at"`
	ExpiringSoon          bool                         `json:"expiring_soon"`
	DaysToExpire          int                          `json:"days_to_expire"`
	PricePerSeatMonth     int                          `json:"price_per_seat_month"`
	PricePerSeatYear      int                          `json:"price_per_seat_year"`
	Currency              string                       `json:"currency"`
	ExpansionRecords      []tenantSeatExpansionPayload `json:"expansion_records"`
	HistoryRecords        []tenantSeatExpansionPayload `json:"history_records"`
	PendingRequestRecords []tenantSeatExpansionPayload `json:"pending_request_records"`
}

type tenantSeatExpansionPayload struct {
	ID                 uint       `json:"id"`
	SeatCount          int        `json:"seat_count"`
	Years              int        `json:"years"`
	PricePerSeat       int64      `json:"price_per_seat"`
	Currency           string     `json:"currency"`
	TotalAmount        int64      `json:"total_amount"`
	TotalAmountCents   int64      `json:"total_amount_cents"`
	TotalAmountDisplay string     `json:"total_amount_display"`
	Source             string     `json:"source"`
	Status             string     `json:"status"`
	ValidFrom          time.Time  `json:"valid_from"`
	ValidTo            time.Time  `json:"valid_to"`
	DaysLeft           int        `json:"days_left"`
	Remark             string     `json:"remark"`
	ConfirmedAt        *time.Time `json:"confirmed_at"`
}

func (c *StorageController) Usage() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !auth.IsTenantAdmin() {
		utils.Forbidden(c.Ctx, "无权限访问存储空间")
		return
	}
	usage, err := buildTenantStorageUsage(auth.TenantID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusServiceUnavailable, err.Error())
		return
	}
	utils.Success(c.Ctx, usage)
}

func (c *StorageController) SeatUsage() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !auth.IsTenantAdmin() {
		utils.Forbidden(c.Ctx, "无权限访问人数权益")
		return
	}
	usage, err := buildTenantSeatUsage(auth.TenantID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(c.Ctx, usage)
}

func (c *StorageController) Quote() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !auth.IsTenantAdmin() {
		utils.Forbidden(c.Ctx, "仅企业管理员可计算权益报价")
		return
	}
	var req entitlementQuoteRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	quote, err := buildEntitlementQuote(auth.TenantID, req.Resource, req.Quantity, req.Years, time.Now())
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	utils.Success(c.Ctx, quote)
}

func (c *StorageController) PurchaseRequest() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !auth.IsTenantAdmin() {
		utils.Forbidden(c.Ctx, "仅企业管理员可申请增购存储空间")
		return
	}
	var req storagePurchaseRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Remark = strings.TrimSpace(req.Remark)
	if req.CapacityGB < 0 || req.CapacityGB > 1024 {
		utils.BadRequest(c.Ctx, "增购容量必须在 0-1024GB 之间")
		return
	}
	if req.Years <= 0 || req.Years > 10 {
		utils.BadRequest(c.Ctx, "购买年限必须在 1-10 年之间")
		return
	}
	quote, err := buildEntitlementQuote(auth.TenantID, "storage", req.CapacityGB, req.Years, time.Now())
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	usage, err := buildTenantStorageUsage(auth.TenantID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusServiceUnavailable, err.Error())
		return
	}
	var tenant models.Tenant
	_ = utils.DB.Select("id,name").Where("id = ?", auth.TenantID).First(&tenant).Error
	now := time.Now()
	purchase := models.TenantStorageExpansion{
		TenantModel:      models.TenantModel{TenantID: auth.TenantID},
		CapacityGB:       quote.TargetQuantity,
		Years:            req.Years,
		PricePerGB:       storagePricePerGBYear,
		Currency:         "USD",
		TotalAmount:      centsToRoundedUSD(quote.DueCents),
		TotalAmountCents: quote.DueCents,
		Source:           "paid",
		Status:           "pending",
		ValidFrom:        now,
		ValidTo:          quote.TargetValidTo,
		Remark:           req.Remark,
		PurchasedBy:      auth.UserID,
		RequestJSON: toJSONString(map[string]interface{}{
			"current_used_bytes":  usage.UsedBytes,
			"current_total_bytes": usage.TotalBytes,
			"current_expire_at":   quote.CurrentExpireAt,
			"requested_quantity":  req.CapacityGB,
			"target_quantity":     quote.TargetQuantity,
			"requested_years":     quote.TargetYears,
			"requested_valid_to":  quote.TargetValidTo,
			"upload_locked":       usage.UploadLocked,
			"lock_reason":         usage.LockReason,
		}),
		BillingSnapshot: toJSONString(quote.Snapshot),
	}
	if err := utils.DB.Create(&purchase).Error; err != nil {
		utils.LogError("创建存储增购申请失败: tenant_id=%d user_id=%d err=%v", auth.TenantID, auth.UserID, err)
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建存储增购申请失败")
		return
	}

	title := fmt.Sprintf("存储空间增购申请：新增%dGB，统一%d年", req.CapacityGB, req.Years)
	tenantName := defaultText(tenant.Name, fmt.Sprintf("租户%d", auth.TenantID))
	currentUsed := formatStorageBytes(int64(usage.UsedBytes))
	currentTotal := formatStorageBytes(usage.TotalBytes)
	uploadStatus := storageUploadStatusText(usage)
	uploadStatusKey := "storage.uploadStatus.available"
	if usage.UploadLocked {
		uploadStatusKey = "storage.uploadStatus.locked"
	}
	remark := defaultText(req.Remark, "-")
	content := fmt.Sprintf(
		"租户ID：%d\n租户名称：%s\n申请人：%s\n申请新增容量：%dGB\n扩容后容量：%dGB\n统一使用年限：%d年\n当前剩余价值：%s\n应补金额：%s\n新统一到期日：%s\n当前已用：%s\n当前总空间：%s\n上传状态：%s\n计费公式：%s\n备注：%s",
		auth.TenantID,
		tenantName,
		auth.Username,
		req.CapacityGB,
		quote.TargetQuantity,
		req.Years,
		quote.CurrentCreditDisplay,
		quote.DueDisplay,
		quote.TargetValidTo.Format("2006-01-02"),
		currentUsed,
		currentTotal,
		uploadStatus,
		quote.FormulaSummary,
		remark,
	)
	convID, msg, recipientID, err := services.NewIMService().OpenSupportConversation(auth.TenantID, auth.UserID, services.IMSupportInput{
		Category: "storage_purchase",
		Title:    title,
		Content:  content,
		Priority: "urgent",
		MessagePayload: map[string]interface{}{
			"i18n_key": "notification.storagePurchase.content",
			"i18n_params": map[string]interface{}{
				"tenant_id":              auth.TenantID,
				"tenant_name":            tenantName,
				"requester":              auth.Username,
				"capacity_gb":            req.CapacityGB,
				"years":                  req.Years,
				"total":                  quote.DueDisplay,
				"current_used":           currentUsed,
				"current_total":          currentTotal,
				"upload_status":          uploadStatus,
				"upload_status_i18n_key": uploadStatusKey,
				"remark":                 remark,
			},
			"title_i18n_key":    "notification.storagePurchase.title",
			"title_i18n_params": map[string]interface{}{"capacity_gb": req.CapacityGB, "years": req.Years},
			"fallback_title":    title,
			"fallback_content":  content,
		},
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "存储申请已创建，但发送系统管理员实时消息失败: "+err.Error())
		return
	}
	if platformAdminIDs, receiptErr := services.NewIMService().EnsureSupportReceiptsForPlatformAdmins(auth.TenantID, convID, msg.ID); receiptErr == nil {
		for _, platformAdminID := range platformAdminIDs {
			publishPlatformSystemNotificationUnread(platformAdminID)
		}
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"request":       storageExpansionPayload(purchase),
		"conversation":  convID,
		"message_id":    msg.ID,
		"recipient_id":  recipientID,
		"estimated_usd": quote.DueDisplay,
		"quote":         quote,
	})
}

func (c *StorageController) SeatPurchaseRequest() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if !auth.IsTenantAdmin() {
		utils.Forbidden(c.Ctx, "仅企业管理员可申请增购人数")
		return
	}
	var req entitlementQuoteRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Remark = strings.TrimSpace(req.Remark)
	if req.Quantity < 0 || req.Quantity > 10000 {
		utils.BadRequest(c.Ctx, "增购人数必须在 0-10000 之间")
		return
	}
	if req.Years <= 0 || req.Years > 10 {
		utils.BadRequest(c.Ctx, "统一使用年限必须在 1-10 年之间")
		return
	}
	quote, err := buildEntitlementQuote(auth.TenantID, "seats", req.Quantity, req.Years, time.Now())
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	var tenant models.Tenant
	_ = utils.DB.Select("id,name").Where("id = ?", auth.TenantID).First(&tenant).Error
	now := time.Now()
	purchase := models.TenantSeatExpansion{
		TenantModel:      models.TenantModel{TenantID: auth.TenantID},
		SeatCount:        quote.TargetQuantity,
		Years:            req.Years,
		PricePerSeat:     seatPricePerMonth,
		Currency:         "USD",
		TotalAmount:      centsToRoundedUSD(quote.DueCents),
		TotalAmountCents: quote.DueCents,
		Source:           "paid",
		Status:           "pending",
		ValidFrom:        now,
		ValidTo:          quote.TargetValidTo,
		Remark:           req.Remark,
		PurchasedBy:      auth.UserID,
		RequestJSON: toJSONString(map[string]interface{}{
			"requested_quantity": req.Quantity,
			"target_quantity":    quote.TargetQuantity,
			"requested_years":    quote.TargetYears,
			"requested_valid_to": quote.TargetValidTo,
		}),
		BillingSnapshot: toJSONString(quote.Snapshot),
	}
	if err := utils.DB.Create(&purchase).Error; err != nil {
		utils.LogError("创建人数增购申请失败: tenant_id=%d user_id=%d err=%v", auth.TenantID, auth.UserID, err)
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建人数增购申请失败")
		return
	}
	tenantName := defaultText(tenant.Name, fmt.Sprintf("租户%d", auth.TenantID))
	remark := defaultText(req.Remark, "-")
	title := fmt.Sprintf("人数增购申请：新增%d人，统一%d年", req.Quantity, req.Years)
	content := fmt.Sprintf(
		"租户ID：%d\n租户名称：%s\n申请人：%s\n申请新增人数：%d人\n扩容后人数：%d人\n统一使用年限：%d年\n当前剩余价值：%s\n应补金额：%s\n新统一到期日：%s\n计费公式：%s\n备注：%s",
		auth.TenantID,
		tenantName,
		auth.Username,
		req.Quantity,
		quote.TargetQuantity,
		req.Years,
		quote.CurrentCreditDisplay,
		quote.DueDisplay,
		quote.TargetValidTo.Format("2006-01-02"),
		quote.FormulaSummary,
		remark,
	)
	convID, msg, recipientID, err := services.NewIMService().OpenSupportConversation(auth.TenantID, auth.UserID, services.IMSupportInput{
		Category: "seat_purchase",
		Title:    title,
		Content:  content,
		Priority: "urgent",
		MessagePayload: map[string]interface{}{
			"resource":       "seats",
			"quote":          quote.Snapshot,
			"fallback_title": title,
		},
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "人数申请已创建，但发送系统管理员实时消息失败: "+err.Error())
		return
	}
	if platformAdminIDs, receiptErr := services.NewIMService().EnsureSupportReceiptsForPlatformAdmins(auth.TenantID, convID, msg.ID); receiptErr == nil {
		for _, platformAdminID := range platformAdminIDs {
			publishPlatformSystemNotificationUnread(platformAdminID)
		}
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"request":      seatExpansionPayload(purchase),
		"conversation": convID,
		"message_id":   msg.ID,
		"recipient_id": recipientID,
		"quote":        quote,
	})
}

func (c *StorageController) PlatformListExpansions() {
	_, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	usage, err := buildTenantStorageUsage(tenantID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusServiceUnavailable, err.Error())
		return
	}
	utils.Success(c.Ctx, usage)
}

func (c *StorageController) PlatformQuote() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	_ = auth
	tenantID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	var req entitlementQuoteRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	quote, err := buildEntitlementQuote(tenantID, req.Resource, req.Quantity, req.Years, time.Now())
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	utils.Success(c.Ctx, quote)
}

func (c *StorageController) PlatformSendQuote() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	var req entitlementQuoteRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Remark = strings.TrimSpace(req.Remark)
	quote, err := buildEntitlementQuote(tenantID, req.Resource, req.Quantity, req.Years, time.Now())
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	tenantAdminID := tenantPrimaryAdminID(tenantID)
	if tenantAdminID == 0 {
		utils.BadRequest(c.Ctx, "租户管理员不存在")
		return
	}
	resourceText := "存储"
	unitText := quote.UnitLabel
	if quote.Resource == "seats" {
		resourceText = "人数"
	}
	title := fmt.Sprintf("%s扩充报价：新增%d%s，统一%d年", resourceText, quote.AdditionalQuantity, unitText, quote.TargetYears)
	content := fmt.Sprintf(
		"报价资源：%s\n新增数量：%d%s\n扩充后总量：%d%s\n统一使用年限：%d年\n当前剩余价值：%s\n应补金额：%s\n新统一到期日：%s\n计费公式：%s\n备注：%s",
		resourceText,
		quote.AdditionalQuantity,
		unitText,
		quote.TargetQuantity,
		unitText,
		quote.TargetYears,
		quote.CurrentCreditDisplay,
		quote.DueDisplay,
		quote.TargetValidTo.Format("2006-01-02"),
		quote.FormulaSummary,
		defaultText(req.Remark, "-"),
	)
	convID, msg, recipientID, err := services.NewIMService().OpenSupportConversation(tenantID, tenantAdminID, services.IMSupportInput{
		Category: "entitlement_quote",
		Title:    title,
		Content:  content,
		Priority: "normal",
		MessagePayload: map[string]interface{}{
			"resource":          quote.Resource,
			"quote":             quote.Snapshot,
			"platform_admin_id": auth.UserID,
			"fallback_title":    title,
			"fallback_content":  content,
		},
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "发送报价失败: "+err.Error())
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"quote": quote, "conversation": convID, "message_id": msg.ID, "recipient_id": recipientID})
}

func (c *StorageController) PlatformSeatUsage() {
	_, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	usage, err := buildTenantSeatUsage(tenantID)
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.Success(c.Ctx, usage)
}

func (c *StorageController) PlatformCreateExpansion() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	var req platformStorageExpansionRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	expansion, err := buildStorageExpansionFromPlatformRequest(tenantID, auth.UserID, req)
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	if err := utils.DB.Create(&expansion).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存存储扩容记录失败")
		return
	}
	usage, _ := buildTenantStorageUsage(tenantID)
	utils.Success(c.Ctx, map[string]interface{}{"expansion": storageExpansionPayload(expansion), "usage": usage})
}

func (c *StorageController) PlatformCreateSeatExpansion() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	var req entitlementQuoteRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Resource = "seats"
	req.Remark = strings.TrimSpace(req.Remark)
	quote, err := buildEntitlementQuote(tenantID, "seats", req.Quantity, req.Years, time.Now())
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	now := time.Now()
	expansion := models.TenantSeatExpansion{
		TenantModel:      models.TenantModel{TenantID: tenantID},
		SeatCount:        quote.TargetQuantity,
		Years:            req.Years,
		PricePerSeat:     seatPricePerMonth,
		Currency:         "USD",
		TotalAmount:      centsToRoundedUSD(quote.DueCents),
		TotalAmountCents: quote.DueCents,
		Source:           "paid",
		Status:           "pending",
		ValidFrom:        now,
		ValidTo:          quote.TargetValidTo,
		Remark:           req.Remark,
		ConfirmedBy:      auth.UserID,
		BillingSnapshot:  toJSONString(quote.Snapshot),
	}
	if err := utils.DB.Create(&expansion).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存人数扩容记录失败")
		return
	}
	usage, _ := buildTenantSeatUsage(tenantID)
	utils.Success(c.Ctx, map[string]interface{}{"expansion": seatExpansionPayload(expansion), "usage": usage, "quote": quote})
}

func (c *StorageController) PlatformUpdateExpansion() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的租户ID")
		return
	}
	expansionID, valid := pathUint(c.Ctx, ":expansion_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的扩容记录ID")
		return
	}
	var existing models.TenantStorageExpansion
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, expansionID).First(&existing).Error; err != nil {
		utils.NotFound(c.Ctx, "扩容记录不存在")
		return
	}
	var req platformStorageExpansionRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	next, err := buildStorageExpansionFromPlatformRequest(tenantID, auth.UserID, req)
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	existing.CapacityGB = next.CapacityGB
	existing.Years = next.Years
	existing.PricePerGB = next.PricePerGB
	existing.Currency = next.Currency
	existing.TotalAmount = next.TotalAmount
	existing.Source = next.Source
	existing.Status = next.Status
	existing.ValidFrom = next.ValidFrom
	existing.ValidTo = next.ValidTo
	existing.Remark = next.Remark
	existing.ConfirmedBy = auth.UserID
	now := time.Now()
	existing.ConfirmedAt = &now
	if err := utils.DB.Save(&existing).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新存储扩容记录失败")
		return
	}
	usage, _ := buildTenantStorageUsage(tenantID)
	utils.Success(c.Ctx, map[string]interface{}{"expansion": storageExpansionPayload(existing), "usage": usage})
}

func (c *StorageController) PlatformApproveStorageExpansion() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID, expansionID, ok := platformTenantExpansionIDs(c.Ctx, ":expansion_id")
	if !ok {
		utils.BadRequest(c.Ctx, "无效的扩容记录")
		return
	}
	var record models.TenantStorageExpansion
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, expansionID).First(&record).Error; err != nil {
		utils.NotFound(c.Ctx, "扩容记录不存在")
		return
	}
	if record.Status != "pending" {
		utils.BadRequest(c.Ctx, "只能审核待处理记录")
		return
	}
	now := time.Now()
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.TenantStorageExpansion{}).
			Where("tenant_id = ? AND status = ? AND id <> ?", tenantID, "active", record.ID).
			Updates(map[string]interface{}{"status": "cancelled", "remark": gorm.Expr("CONCAT(COALESCE(remark,''), ?)", "\nReplaced by unified entitlement recalculation")}).Error; err != nil {
			return err
		}
		return tx.Model(&record).Updates(map[string]interface{}{
			"status":       "active",
			"valid_from":   now,
			"confirmed_by": auth.UserID,
			"confirmed_at": &now,
		}).Error
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "审核存储扩容失败")
		return
	}
	notifyTenantEntitlementApproved(tenantID, "存储", record.CapacityGB, "GB", record.TotalAmountCents, record.ValidTo)
	usage, _ := buildTenantStorageUsage(tenantID)
	utils.Success(c.Ctx, map[string]interface{}{"expansion": storageExpansionPayload(record), "usage": usage})
}

func (c *StorageController) PlatformRejectStorageExpansion() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID, expansionID, ok := platformTenantExpansionIDs(c.Ctx, ":expansion_id")
	if !ok {
		utils.BadRequest(c.Ctx, "无效的扩容记录")
		return
	}
	now := time.Now()
	if err := utils.DB.Model(&models.TenantStorageExpansion{}).
		Where("tenant_id = ? AND id = ? AND status = ?", tenantID, expansionID, "pending").
		Updates(map[string]interface{}{"status": "cancelled", "confirmed_by": auth.UserID, "confirmed_at": &now}).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "拒绝存储扩容失败")
		return
	}
	usage, _ := buildTenantStorageUsage(tenantID)
	utils.Success(c.Ctx, usage)
}

func (c *StorageController) PlatformApproveSeatExpansion() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID, expansionID, ok := platformTenantExpansionIDs(c.Ctx, ":expansion_id")
	if !ok {
		utils.BadRequest(c.Ctx, "无效的人数扩容记录")
		return
	}
	var record models.TenantSeatExpansion
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, expansionID).First(&record).Error; err != nil {
		utils.NotFound(c.Ctx, "人数扩容记录不存在")
		return
	}
	if record.Status != "pending" {
		utils.BadRequest(c.Ctx, "只能审核待处理记录")
		return
	}
	now := time.Now()
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.TenantSeatExpansion{}).
			Where("tenant_id = ? AND status = ? AND id <> ?", tenantID, "active", record.ID).
			Updates(map[string]interface{}{"status": "cancelled", "remark": gorm.Expr("CONCAT(COALESCE(remark,''), ?)", "\nReplaced by unified entitlement recalculation")}).Error; err != nil {
			return err
		}
		if err := tx.Model(&record).Updates(map[string]interface{}{
			"status":       "active",
			"valid_from":   now,
			"confirmed_by": auth.UserID,
			"confirmed_at": &now,
		}).Error; err != nil {
			return err
		}
		return ensureTenantQuotaSeatsTx(tx, tenantID, record.SeatCount, record.ValidTo)
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "审核人数扩容失败")
		return
	}
	notifyTenantEntitlementApproved(tenantID, "人数", record.SeatCount, "人", record.TotalAmountCents, record.ValidTo)
	usage, _ := buildTenantSeatUsage(tenantID)
	utils.Success(c.Ctx, map[string]interface{}{"expansion": seatExpansionPayload(record), "usage": usage})
}

func (c *StorageController) PlatformRejectSeatExpansion() {
	auth, ok := requirePlatformAdmin(c.Ctx)
	if !ok {
		return
	}
	tenantID, expansionID, ok := platformTenantExpansionIDs(c.Ctx, ":expansion_id")
	if !ok {
		utils.BadRequest(c.Ctx, "无效的人数扩容记录")
		return
	}
	now := time.Now()
	if err := utils.DB.Model(&models.TenantSeatExpansion{}).
		Where("tenant_id = ? AND id = ? AND status = ?", tenantID, expansionID, "pending").
		Updates(map[string]interface{}{"status": "cancelled", "confirmed_by": auth.UserID, "confirmed_at": &now}).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "拒绝人数扩容失败")
		return
	}
	usage, _ := buildTenantSeatUsage(tenantID)
	utils.Success(c.Ctx, usage)
}

func buildStorageExpansionFromPlatformRequest(tenantID uint, adminID uint, req platformStorageExpansionRequest) (models.TenantStorageExpansion, error) {
	req.Source = strings.TrimSpace(req.Source)
	req.Status = strings.TrimSpace(req.Status)
	req.Remark = strings.TrimSpace(req.Remark)
	if req.Source == "" {
		req.Source = "paid"
	}
	if req.Source != "paid" && req.Source != "gift" {
		return models.TenantStorageExpansion{}, fmt.Errorf("扩容来源必须为 paid 或 gift")
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if req.Status != "active" && req.Status != "expired" && req.Status != "cancelled" && req.Status != "pending" {
		return models.TenantStorageExpansion{}, fmt.Errorf("扩容状态无效")
	}
	if req.CapacityGB < 0 || req.CapacityGB > 1024 {
		return models.TenantStorageExpansion{}, fmt.Errorf("扩容容量必须在 0-1024GB 之间")
	}
	if req.Years <= 0 || req.Years > 10 {
		req.Years = 1
	}
	validFrom := time.Now()
	if parsed := parseDatePtr(req.ValidFrom); parsed != nil {
		validFrom = *parsed
	}
	quote, err := buildEntitlementQuote(tenantID, "storage", req.CapacityGB, req.Years, validFrom)
	if err != nil {
		return models.TenantStorageExpansion{}, err
	}
	validTo := quote.TargetValidTo
	explicitValidTo := parseDatePtr(req.ValidTo)
	if parsed := explicitValidTo; parsed != nil {
		validTo = *parsed
	}
	if !validTo.After(validFrom) {
		return models.TenantStorageExpansion{}, fmt.Errorf("到期时间必须晚于开始时间")
	}
	totalCents := quote.DueCents
	targetQuantity := quote.TargetQuantity
	if explicitValidTo != nil {
		totalCents = calculateTargetDateDueCents(tenantID, "storage", req.CapacityGB, validTo, validFrom)
		targetQuantity = quote.CurrentQuantity + req.CapacityGB
	}
	total := centsToRoundedUSD(totalCents)
	if req.Source == "gift" {
		total = 0
		totalCents = 0
		targetQuantity = req.CapacityGB
	}
	now := time.Now()
	return models.TenantStorageExpansion{
		TenantModel:      models.TenantModel{TenantID: tenantID},
		CapacityGB:       targetQuantity,
		Years:            req.Years,
		PricePerGB:       storagePricePerGBYear,
		Currency:         "USD",
		TotalAmount:      total,
		TotalAmountCents: totalCents,
		Source:           req.Source,
		Status:           req.Status,
		ValidFrom:        validFrom,
		ValidTo:          validTo,
		Remark:           req.Remark,
		ConfirmedBy:      adminID,
		ConfirmedAt:      &now,
		BillingSnapshot:  toJSONString(quote.Snapshot),
	}, nil
}

func buildTenantStorageUsage(tenantID uint) (tenantStorageUsage, error) {
	if tenantID == 0 {
		return tenantStorageUsage{}, fmt.Errorf("缺少租户上下文")
	}
	used, err := services.FileserverTenantUsedBytes(tenantID)
	if err != nil {
		return tenantStorageUsage{}, fmt.Errorf("查询租户存储空间失败: %w", err)
	}
	now := time.Now()
	var records []models.TenantStorageExpansion
	if err := utils.DB.Where("tenant_id = ?", tenantID).Order("valid_to asc, id asc").Find(&records).Error; err != nil {
		return tenantStorageUsage{}, err
	}
	activeGB := 0
	expiredGB := 0
	pending := make([]tenantStorageExpansionPayload, 0)
	items := make([]tenantStorageExpansionPayload, 0, len(records))
	for _, record := range records {
		if record.Status == "pending" {
			pending = append(pending, storageExpansionPayload(record))
			continue
		}
		items = append(items, storageExpansionPayload(record))
		if isExpiredStorageExpansion(record, now) {
			expiredGB += record.CapacityGB
			continue
		}
	}
	activeGB, _, currentExpireAt := activeStorageEntitlementFromRecords(records, now)
	totalBytes := baseStorageBytes + int64(activeGB)*baseStorageBytes
	remaining := totalBytes - int64(used)
	if remaining < 0 {
		remaining = 0
	}
	usagePercent := 0.0
	if totalBytes > 0 {
		usagePercent = roundFloat(float64(used) / float64(totalBytes) * 100)
	}
	days := 0
	expiringSoon := false
	if currentExpireAt != nil {
		days = int(currentExpireAt.Sub(now).Hours() / 24)
		expiringSoon = days >= 0 && days <= 30
	}
	uploadLocked := int64(used) > totalBytes
	lockReason := ""
	if uploadLocked {
		lockReason = "当前已用空间超过有效配额，请续费或清理后继续上传"
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ValidTo.Equal(items[j].ValidTo) {
			return items[i].ID > items[j].ID
		}
		return items[i].ValidTo.After(items[j].ValidTo)
	})
	sort.Slice(pending, func(i, j int) bool {
		if pending[i].ValidTo.Equal(pending[j].ValidTo) {
			return pending[i].ID > pending[j].ID
		}
		return pending[i].ValidTo.After(pending[j].ValidTo)
	})
	return tenantStorageUsage{
		TenantID:              tenantID,
		UsedBytes:             used,
		UsedGB:                roundFloat(float64(used) / float64(baseStorageBytes)),
		BaseBytes:             baseStorageBytes,
		BaseGB:                1,
		ExpansionBytes:        int64(activeGB) * baseStorageBytes,
		ExpansionGB:           activeGB,
		TotalBytes:            totalBytes,
		TotalGB:               roundFloat(float64(totalBytes) / float64(baseStorageBytes)),
		RemainingBytes:        remaining,
		UsagePercent:          usagePercent,
		UploadLocked:          uploadLocked,
		LockReason:            lockReason,
		ExpiringSoon:          expiringSoon,
		DaysToExpire:          days,
		NextExpireAt:          currentExpireAt,
		CurrentExpireAt:       currentExpireAt,
		UnifiedExpireAt:       currentExpireAt,
		ExpiredExpansionGB:    expiredGB,
		PricePerGBYear:        storagePricePerGBYear,
		Currency:              "USD",
		ExpansionRecords:      items,
		HistoryRecords:        items,
		PendingRequestRecords: pending,
	}, nil
}

func currentStorageUnifiedExpireAt(tenantID uint, now time.Time) *time.Time {
	var records []models.TenantStorageExpansion
	if err := utils.DB.Where("tenant_id = ? AND status = ?", tenantID, "active").Find(&records).Error; err != nil {
		return nil
	}
	_, _, expireAt := activeStorageEntitlementFromRecords(records, now)
	return expireAt
}

func activeStorageEntitlementFromRecords(records []models.TenantStorageExpansion, now time.Time) (int, int, *time.Time) {
	activeGB := 0
	expiredGB := 0
	var unifiedExpireAt *time.Time
	for _, record := range records {
		if isExpiredStorageExpansion(record, now) {
			expiredGB += record.CapacityGB
			continue
		}
		if record.Status != "active" {
			continue
		}
		activeGB += record.CapacityGB
		if unifiedExpireAt == nil || record.ValidTo.After(*unifiedExpireAt) {
			v := record.ValidTo
			unifiedExpireAt = &v
		}
	}
	return activeGB, expiredGB, unifiedExpireAt
}

func isExpiredStorageExpansion(record models.TenantStorageExpansion, now time.Time) bool {
	return record.Status == "expired" || (record.Status == "active" && !record.ValidTo.After(now))
}

func buildTenantSeatUsage(tenantID uint) (tenantSeatUsage, error) {
	if tenantID == 0 {
		return tenantSeatUsage{}, fmt.Errorf("缺少租户上下文")
	}
	now := time.Now()
	var records []models.TenantSeatExpansion
	if err := utils.DB.Where("tenant_id = ?", tenantID).Order("valid_to asc, id asc").Find(&records).Error; err != nil {
		return tenantSeatUsage{}, err
	}
	activeSeats := 0
	pending := make([]tenantSeatExpansionPayload, 0)
	items := make([]tenantSeatExpansionPayload, 0, len(records))
	var expireAt *time.Time
	for _, record := range records {
		if record.Status == "pending" {
			pending = append(pending, seatExpansionPayload(record))
			continue
		}
		items = append(items, seatExpansionPayload(record))
		if record.Status != "active" || !record.ValidTo.After(now) {
			continue
		}
		activeSeats += record.SeatCount
		if expireAt == nil || record.ValidTo.After(*expireAt) {
			v := record.ValidTo
			expireAt = &v
		}
	}
	if activeSeats <= 0 {
		var quota models.TenantQuota
		if err := utils.DB.Where("tenant_id = ?", tenantID).First(&quota).Error; err == nil {
			activeSeats = quota.MaxUsers
			expireAt = quota.ValidTo
		}
	}
	var currentUsers int64
	_ = utils.DB.Model(&models.User{}).Where("tenant_id = ? AND status <> ?", tenantID, "deleted").Count(&currentUsers).Error
	usagePercent := 0.0
	if activeSeats > 0 {
		usagePercent = roundFloat(float64(currentUsers) / float64(activeSeats) * 100)
	}
	days := 0
	expiringSoon := false
	if expireAt != nil {
		days = int(expireAt.Sub(now).Hours() / 24)
		expiringSoon = days >= 0 && days <= 30
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ValidTo.Equal(items[j].ValidTo) {
			return items[i].ID > items[j].ID
		}
		return items[i].ValidTo.After(items[j].ValidTo)
	})
	sort.Slice(pending, func(i, j int) bool {
		if pending[i].ValidTo.Equal(pending[j].ValidTo) {
			return pending[i].ID > pending[j].ID
		}
		return pending[i].ValidTo.After(pending[j].ValidTo)
	})
	return tenantSeatUsage{
		TenantID:              tenantID,
		CurrentUsers:          currentUsers,
		MaxUsers:              activeSeats,
		SeatCount:             activeSeats,
		UsagePercent:          usagePercent,
		CurrentExpireAt:       expireAt,
		UnifiedExpireAt:       expireAt,
		NextExpireAt:          expireAt,
		ExpiringSoon:          expiringSoon,
		DaysToExpire:          days,
		PricePerSeatMonth:     seatPricePerMonth,
		PricePerSeatYear:      seatPricePerMonth * 12,
		Currency:              "USD",
		ExpansionRecords:      items,
		HistoryRecords:        items,
		PendingRequestRecords: pending,
	}, nil
}

func buildEntitlementQuote(tenantID uint, resource string, additionalQuantity int, years int, now time.Time) (entitlementQuotePayload, error) {
	resource = normalizeEntitlementResource(resource)
	if resource == "" {
		return entitlementQuotePayload{}, fmt.Errorf("权益类型必须为 storage 或 seats")
	}
	if additionalQuantity < 0 {
		return entitlementQuotePayload{}, fmt.Errorf("新增数量不能小于 0")
	}
	if years <= 0 || years > 10 {
		return entitlementQuotePayload{}, fmt.Errorf("统一使用年限必须在 1-10 年之间")
	}
	currentQuantity, currentExpireAt, credit, lines, err := entitlementCurrentState(tenantID, resource, now)
	if err != nil {
		return entitlementQuotePayload{}, err
	}
	return buildEntitlementQuoteFromState(resource, additionalQuantity, years, now, currentQuantity, currentExpireAt, credit, lines)
}

func buildEntitlementQuoteFromState(resource string, additionalQuantity int, years int, now time.Time, currentQuantity int, currentExpireAt *time.Time, credit int64, lines []entitlementCreditLine) (entitlementQuotePayload, error) {
	resource = normalizeEntitlementResource(resource)
	if resource == "" {
		return entitlementQuotePayload{}, fmt.Errorf("权益类型必须为 storage 或 seats")
	}
	if additionalQuantity < 0 {
		return entitlementQuotePayload{}, fmt.Errorf("新增数量不能小于 0")
	}
	if years <= 0 || years > 10 {
		return entitlementQuotePayload{}, fmt.Errorf("统一使用年限必须在 1-10 年之间")
	}
	targetQuantity := currentQuantity + additionalQuantity
	if targetQuantity <= 0 {
		return entitlementQuotePayload{}, fmt.Errorf("当前没有可续期的有效权益，请填写要开通的容量或人数")
	}
	unitPrice := entitlementDisplayUnitPriceCents(resource)
	annualUnitPrice := entitlementAnnualUnitPriceCents(resource)
	targetMonths := years * 12
	billingMultiplier := years
	billingUnit := "year"
	grossUnitPrice := annualUnitPrice
	if resource == "seats" {
		billingMultiplier = targetMonths
		billingUnit = "month"
		grossUnitPrice = unitPrice
	}
	gross := int64(targetQuantity) * int64(billingMultiplier) * grossUnitPrice
	due := gross - credit
	if due < 0 {
		due = 0
	}
	targetValidTo := now.AddDate(years, 0, 0)
	unitLabel := entitlementUnitLabel(resource)
	formula := "应补金额 = 目标总权益 × 目标年限 × 单价 - 当前付费剩余价值"
	summary := fmt.Sprintf("%s = %d%s × %d年 × %s - %s = %s", formula, targetQuantity, unitLabel, years, centsDisplay(grossUnitPrice, "USD"), centsDisplay(credit, "USD"), centsDisplay(due, "USD"))
	if resource == "seats" {
		formula = "应补金额 = 目标总权益 × 目标月数 × 月单价 - 当前付费剩余价值"
		summary = fmt.Sprintf("%s = %d%s × %d个月 × %s - %s = %s", formula, targetQuantity, unitLabel, targetMonths, centsDisplay(unitPrice, "USD"), centsDisplay(credit, "USD"), centsDisplay(due, "USD"))
	}
	snapshot := map[string]interface{}{
		"resource":                resource,
		"current_quantity":        currentQuantity,
		"additional_quantity":     additionalQuantity,
		"target_quantity":         targetQuantity,
		"target_years":            years,
		"target_months":           targetMonths,
		"billing_multiplier":      billingMultiplier,
		"billing_unit":            billingUnit,
		"unit_price_cents":        unitPrice,
		"annual_unit_price_cents": annualUnitPrice,
		"current_credit_cents":    credit,
		"target_gross_cents":      gross,
		"due_cents":               due,
		"target_valid_to":         targetValidTo,
		"current_expire_at":       currentExpireAt,
		"formula":                 formula,
		"formula_summary":         summary,
		"credit_lines":            lines,
		"billing_days_per_year":   billingDaysPerYear,
		"free_base_excluded":      resource == "storage",
	}
	return entitlementQuotePayload{
		Resource:             resource,
		UnitLabel:            unitLabel,
		Currency:             "USD",
		UnitPriceCents:       unitPrice,
		UnitPriceDisplay:     centsDisplay(unitPrice, "USD"),
		CurrentQuantity:      currentQuantity,
		AdditionalQuantity:   additionalQuantity,
		TargetQuantity:       targetQuantity,
		TargetYears:          years,
		TargetMonths:         targetMonths,
		BillingMultiplier:    billingMultiplier,
		BillingUnit:          billingUnit,
		CurrentExpireAt:      currentExpireAt,
		TargetValidTo:        targetValidTo,
		CurrentCreditCents:   credit,
		CurrentCreditDisplay: centsDisplay(credit, "USD"),
		TargetGrossCents:     gross,
		TargetGrossDisplay:   centsDisplay(gross, "USD"),
		DueCents:             due,
		DueDisplay:           centsDisplay(due, "USD"),
		Formula:              formula,
		FormulaSummary:       summary,
		CreditLines:          lines,
		Snapshot:             snapshot,
	}, nil
}

func entitlementCurrentState(tenantID uint, resource string, now time.Time) (int, *time.Time, int64, []entitlementCreditLine, error) {
	if resource == "storage" {
		var records []models.TenantStorageExpansion
		if err := utils.DB.Where("tenant_id = ?", tenantID).Find(&records).Error; err != nil {
			return 0, nil, 0, nil, err
		}
		current := 0
		var expireAt *time.Time
		var credit int64
		lines := make([]entitlementCreditLine, 0)
		for _, record := range records {
			if record.Status != "active" || !record.ValidTo.After(now) {
				continue
			}
			current += record.CapacityGB
			if expireAt == nil || record.ValidTo.After(*expireAt) {
				v := record.ValidTo
				expireAt = &v
			}
			line := creditLine(record.ID, record.CapacityGB, record.Source, record.Status, record.ValidTo, record.ConfirmedAt, storagePriceCents, now)
			if line.Included {
				credit += line.CreditCents
			}
			lines = append(lines, line)
		}
		return current, expireAt, credit, lines, nil
	}
	var records []models.TenantSeatExpansion
	if err := utils.DB.Where("tenant_id = ?", tenantID).Find(&records).Error; err != nil {
		return 0, nil, 0, nil, err
	}
	current := 0
	var expireAt *time.Time
	var credit int64
	lines := make([]entitlementCreditLine, 0)
	for _, record := range records {
		if record.Status != "active" || !record.ValidTo.After(now) {
			continue
		}
		current += record.SeatCount
		if expireAt == nil || record.ValidTo.After(*expireAt) {
			v := record.ValidTo
			expireAt = &v
		}
		line := creditLine(record.ID, record.SeatCount, record.Source, record.Status, record.ValidTo, record.ConfirmedAt, seatPriceCentsPerYear, now)
		if line.Included {
			credit += line.CreditCents
		}
		lines = append(lines, line)
	}
	if current == 0 {
		var quota models.TenantQuota
		if err := utils.DB.Where("tenant_id = ?", tenantID).First(&quota).Error; err == nil && quota.MaxUsers > 0 {
			current = quota.MaxUsers
			expireAt = quota.ValidTo
			if quota.ValidTo != nil && quota.ValidTo.After(now) {
				days := int(math.Ceil(quota.ValidTo.Sub(now).Hours() / 24))
				credit = proratedCreditCents(quota.MaxUsers, seatPriceCentsPerYear, days)
				lines = append(lines, entitlementCreditLine{Quantity: quota.MaxUsers, Source: "legacy_quota", Status: "active", ValidTo: *quota.ValidTo, DaysLeft: days, CreditCents: credit, CreditDisplay: centsDisplay(credit, "USD"), Included: true})
			}
		}
	}
	return current, expireAt, credit, lines, nil
}

func creditLine(id uint, quantity int, source string, status string, validTo time.Time, confirmedAt *time.Time, unitPriceCents int64, now time.Time) entitlementCreditLine {
	days := int(math.Ceil(validTo.Sub(now).Hours() / 24))
	line := entitlementCreditLine{ID: id, Quantity: quantity, Source: source, Status: status, ValidTo: validTo, DaysLeft: days, ConfirmedAt: confirmedAt}
	if source != "paid" && source != "legacy" {
		line.Included = false
		line.ExcludedReason = "赠送或非付费权益不折抵"
		return line
	}
	if days <= 0 || status != "active" {
		line.Included = false
		line.ExcludedReason = "权益已过期"
		return line
	}
	line.Included = true
	line.CreditCents = proratedCreditCents(quantity, unitPriceCents, days)
	line.CreditDisplay = centsDisplay(line.CreditCents, "USD")
	return line
}

func proratedCreditCents(quantity int, unitPriceCents int64, days int) int64 {
	if quantity <= 0 || unitPriceCents <= 0 || days <= 0 {
		return 0
	}
	return int64(math.Ceil(float64(quantity) * float64(unitPriceCents) * float64(days) / billingDaysPerYear))
}

func calculateTargetDateDueCents(tenantID uint, resource string, additionalQuantity int, validTo time.Time, now time.Time) int64 {
	current, _, credit, _, err := entitlementCurrentState(tenantID, resource, now)
	if err != nil {
		return 0
	}
	days := int(math.Ceil(validTo.Sub(now).Hours() / 24))
	gross := proratedCreditCents(current+additionalQuantity, entitlementAnnualUnitPriceCents(resource), days)
	due := gross - credit
	if due < 0 {
		return 0
	}
	return due
}

func normalizeEntitlementResource(resource string) string {
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "storage", "storage_gb":
		return "storage"
	case "seat", "seats", "users":
		return "seats"
	default:
		return ""
	}
}

func entitlementDisplayUnitPriceCents(resource string) int64 {
	if resource == "seats" {
		return seatPriceCentsPerMonth
	}
	return storagePriceCents
}

func entitlementAnnualUnitPriceCents(resource string) int64 {
	if resource == "seats" {
		return seatPriceCentsPerYear
	}
	return storagePriceCents
}

func entitlementUnitLabel(resource string) string {
	if resource == "seats" {
		return "人"
	}
	return "GB"
}

func centsDisplay(cents int64, currency string) string {
	if strings.TrimSpace(currency) == "" {
		currency = "USD"
	}
	return fmt.Sprintf("%.2f %s", float64(cents)/100, currency)
}

func centsToRoundedUSD(cents int64) int64 {
	if cents <= 0 {
		return 0
	}
	return int64(math.Ceil(float64(cents) / 100))
}

func platformTenantExpansionIDs(ctx *beegoctx.Context, expansionKey string) (uint, uint, bool) {
	tenantID, valid := pathUint(ctx, ":id")
	if !valid {
		return 0, 0, false
	}
	expansionID, valid := pathUint(ctx, expansionKey)
	if !valid {
		return 0, 0, false
	}
	return tenantID, expansionID, true
}

func ensureTenantQuotaSeatsTx(tx *gorm.DB, tenantID uint, seats int, validTo time.Time) error {
	var quota models.TenantQuota
	if err := tx.Where("tenant_id = ?", tenantID).First(&quota).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		quota = models.TenantQuota{TenantID: tenantID, PlanCode: "basic", StorageLimit: baseStorageBytes}
	}
	quota.MaxUsers = seats
	quota.ValidTo = &validTo
	if quota.StorageLimit <= 0 {
		quota.StorageLimit = baseStorageBytes
	}
	if quota.ID == 0 {
		if err := tx.Create(&quota).Error; err != nil {
			return err
		}
	} else if err := tx.Save(&quota).Error; err != nil {
		return err
	}
	return tx.Model(&models.Tenant{}).Where("id = ?", tenantID).Update("quota_json", quotaSummaryJSON(quota)).Error
}

func notifyTenantEntitlementApproved(tenantID uint, resource string, quantity int, unit string, amountCents int64, validTo time.Time) {
	tenantAdminID := tenantPrimaryAdminID(tenantID)
	if tenantAdminID == 0 {
		return
	}
	title := fmt.Sprintf("%s扩充已审核通过", resource)
	content := fmt.Sprintf("%s扩充已审核通过。\n当前权益：%d%s\n应补金额：%s\n统一到期日：%s", resource, quantity, unit, centsDisplay(amountCents, "USD"), validTo.Format("2006-01-02"))
	_, _, _, _ = services.NewIMService().OpenSupportConversation(tenantID, tenantAdminID, services.IMSupportInput{
		Category: "entitlement_quote",
		Title:    title,
		Content:  content,
		Priority: "normal",
		MessagePayload: map[string]interface{}{
			"resource":         resource,
			"quantity":         quantity,
			"unit":             unit,
			"amount_cents":     amountCents,
			"valid_to":         validTo,
			"fallback_title":   title,
			"fallback_content": content,
		},
	})
}

func storageExpansionPayload(record models.TenantStorageExpansion) tenantStorageExpansionPayload {
	totalCents := record.TotalAmountCents
	if totalCents <= 0 && record.TotalAmount > 0 {
		totalCents = record.TotalAmount * 100
	}
	return tenantStorageExpansionPayload{
		ID:                 record.ID,
		CapacityGB:         record.CapacityGB,
		Years:              record.Years,
		PricePerGB:         record.PricePerGB,
		Currency:           record.Currency,
		TotalAmount:        record.TotalAmount,
		TotalAmountCents:   totalCents,
		TotalAmountDisplay: centsDisplay(totalCents, record.Currency),
		Source:             record.Source,
		Status:             record.Status,
		ValidFrom:          record.ValidFrom,
		ValidTo:            record.ValidTo,
		DaysLeft:           int(time.Until(record.ValidTo).Hours() / 24),
		Remark:             record.Remark,
		ConfirmedAt:        record.ConfirmedAt,
	}
}

func seatExpansionPayload(record models.TenantSeatExpansion) tenantSeatExpansionPayload {
	totalCents := record.TotalAmountCents
	if totalCents <= 0 && record.TotalAmount > 0 {
		totalCents = record.TotalAmount * 100
	}
	return tenantSeatExpansionPayload{
		ID:                 record.ID,
		SeatCount:          record.SeatCount,
		Years:              record.Years,
		PricePerSeat:       record.PricePerSeat,
		Currency:           record.Currency,
		TotalAmount:        record.TotalAmount,
		TotalAmountCents:   totalCents,
		TotalAmountDisplay: centsDisplay(totalCents, record.Currency),
		Source:             record.Source,
		Status:             record.Status,
		ValidFrom:          record.ValidFrom,
		ValidTo:            record.ValidTo,
		DaysLeft:           int(time.Until(record.ValidTo).Hours() / 24),
		Remark:             record.Remark,
		ConfirmedAt:        record.ConfirmedAt,
	}
}

func storageUploadStatusText(usage tenantStorageUsage) string {
	if usage.UploadLocked {
		return usage.LockReason
	}
	return "可上传"
}

func formatStorageBytes(value int64) string {
	if value <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	v := float64(value)
	idx := 0
	for v >= 1024 && idx < len(units)-1 {
		v = v / 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%.0f %s", v, units[idx])
	}
	return fmt.Sprintf("%.2f %s", v, units[idx])
}

func tenantStorageLimit(tenantID uint) int64 {
	usage, err := buildTenantStorageUsage(tenantID)
	if err != nil {
		return baseStorageBytes
	}
	return usage.TotalBytes
}

var errTenantStorageUploadLocked = fmt.Errorf("tenant storage upload locked")
var errTenantStorageInsufficient = fmt.Errorf("tenant storage quota exceeded")

func ensureTenantStorageAvailable(tenantID uint, incomingBytes int64) error {
	if tenantID == 0 || incomingBytes <= 0 {
		return nil
	}
	usage, err := buildTenantStorageUsage(tenantID)
	if err != nil {
		return err
	}
	if usage.UploadLocked {
		return errTenantStorageUploadLocked
	}
	if usage.UsedBytes+uint64(incomingBytes) > uint64(usage.TotalBytes) {
		return errTenantStorageInsufficient
	}
	return nil
}

func storageExpansionCountByStatus(tenantID uint, statuses ...string) int64 {
	var count int64
	query := utils.DB.Model(&models.TenantStorageExpansion{}).Where("tenant_id = ?", tenantID)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	_ = query.Count(&count).Error
	return count
}

func storageUsageForPlatformSummary(tenantID uint) (tenantStorageUsage, error) {
	usage, err := buildTenantStorageUsage(tenantID)
	if errorsIsNoReachableUsage(err) {
		return tenantStorageUsage{TenantID: tenantID, BaseBytes: baseStorageBytes, BaseGB: 1, TotalBytes: baseStorageBytes, TotalGB: 1, PricePerGBYear: storagePricePerGBYear, Currency: "USD"}, nil
	}
	return usage, err
}

func errorsIsNoReachableUsage(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no reachable fileserver tenant usage endpoint")
}
