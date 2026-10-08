package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	"gorm.io/gorm"
)

type PortalController struct {
	web.Controller
}

func (c *PortalController) Articles() {
	category := c.Ctx.Input.Query("category")
	query := utils.DB.Model(&models.CRMObject{}).Where("domain = ? AND object_type = ?", "insights", "faq_cluster")
	if category != "" {
		query = query.Where("tags LIKE ? OR name LIKE ?", "%"+category+"%", "%"+category+"%")
	}
	var docs []models.CRMObject
	if err := query.Order("updated_at desc").Limit(50).Find(&docs).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取帮助中心失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"articles": docs})
}

func (c *PortalController) Ask() {
	var req struct {
		Question string `json:"question"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	answer, sources := portalAnswer(req.Question)
	utils.Success(c.Ctx, map[string]interface{}{"answer": answer, "sources": sources})
}

func (c *PortalController) CreateTicket() {
	var req struct {
		Name          string `json:"name"`
		Email         string `json:"email"`
		Phone         string `json:"phone"`
		Category      string `json:"category"`
		Priority      string `json:"priority"`
		Customer      string `json:"customer"`
		CustomerGrade string `json:"customer_grade"`
		Content       string `json:"content"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Content = strings.TrimSpace(req.Content)
	if req.Name == "" || req.Content == "" {
		utils.BadRequest(c.Ctx, "工单标题和问题描述不能为空")
		return
	}
	if strings.TrimSpace(req.Priority) == "" {
		req.Priority = "normal"
	}
	if strings.TrimSpace(req.Category) == "" {
		req.Category = "general"
	}
	tenantID := defaultTenantID()
	trackingID := fmt.Sprintf("PT-%d", time.Now().UnixNano()%1000000000)
	payload := map[string]interface{}{
		"source":         "portal",
		"email":          strings.TrimSpace(req.Email),
		"phone":          strings.TrimSpace(req.Phone),
		"customer":       strings.TrimSpace(req.Customer),
		"category":       strings.TrimSpace(req.Category),
		"priority":       strings.TrimSpace(req.Priority),
		"customer_grade": strings.TrimSpace(req.CustomerGrade),
		"content":        req.Content,
		"tracking_id":    trackingID,
		"submitted_at":   time.Now().Format(time.RFC3339),
	}
	payloadJSON, _ := json.Marshal(payload)
	ticket := models.CRMObject{
		TenantModel: models.TenantModel{TenantID: tenantID},
		Domain:      "tickets",
		ObjectType:  "ticket",
		Name:        req.Name,
		Code:        trackingID,
		Status:      "new",
		Currency:    tenantBusinessCurrency(tenantID),
		Tags:        "portal",
		PayloadJSON: string(payloadJSON),
	}
	portalTicket := models.PortalTicket{
		TrackingID: trackingID,
		Status:     "new",
		Replies:    "工单已提交，客服团队会尽快处理。",
	}
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}
		portalTicket.TicketID = ticket.ID
		return tx.Create(&portalTicket).Error
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "提交工单失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"tracking_id": trackingID, "ticket_id": ticket.ID, "status": portalTicket.Status})
}

func portalAnswer(question string) (string, []string) {
	var docs []models.CRMObject
	if utils.DB != nil {
		like := "%" + question + "%"
		_ = utils.DB.Where("domain = ? AND object_type = ? AND (name LIKE ? OR tags LIKE ?)", "insights", "faq_cluster", like, like).Limit(3).Find(&docs).Error
	}
	if len(docs) == 0 {
		return "您可以在客户门户提交问题或工单，系统会自动匹配知识库并给出处理进度。", []string{"portal-default"}
	}
	sources := make([]string, 0, len(docs))
	for _, doc := range docs {
		sources = append(sources, doc.Name)
	}
	payload := payloadMap(docs[0].PayloadJSON)
	if answer, ok := payload["answer"].(string); ok && answer != "" {
		return answer, sources
	}
	return "已匹配到帮助中心条目：" + docs[0].Name, sources
}

func (c *PortalController) TicketStatus() {
	trackingID := c.Ctx.Input.Param(":tracking_id")
	var portalTicket models.PortalTicket
	if err := utils.DB.Where("tracking_id = ?", trackingID).First(&portalTicket).Error; err != nil {
		utils.NotFound(c.Ctx, "工单不存在")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"status": portalTicket.Status, "replies": portalTicket.Replies})
}
