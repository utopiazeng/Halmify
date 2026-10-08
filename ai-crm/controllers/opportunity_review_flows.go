package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

const opportunityReviewRelationType = "opportunity_review"

type opportunityReviewRequest struct {
	ReviewType       string                 `json:"review_type"`
	Title            string                 `json:"title"`
	Summary          string                 `json:"summary"`
	Industry         string                 `json:"industry"`
	Competitor       string                 `json:"competitor"`
	Background       string                 `json:"background"`
	CustomerNeeds    string                 `json:"customer_needs"`
	DecisionProcess  string                 `json:"decision_process"`
	KeyActions       string                 `json:"key_actions"`
	Obstacles        string                 `json:"obstacles"`
	WinReasons       string                 `json:"win_reasons"`
	LossReasons      string                 `json:"loss_reasons"`
	Lessons          string                 `json:"lessons"`
	ReusablePlaybook string                 `json:"reusable_playbook"`
	AvoidNextTime    string                 `json:"avoid_next_time"`
	Tags             []string               `json:"tags"`
	Keywords         []string               `json:"keywords"`
	Payload          map[string]interface{} `json:"payload"`
}

func (c *CRMController) GetOpportunityReview() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的商机ID")
		return
	}
	opportunity, ok := loadVisibleOpportunity(c, auth, id)
	if !ok {
		return
	}
	reviewType := normalizeOpportunityReviewType(c.Ctx.Input.Query("review_type"))
	if reviewType != "" {
		review, found, err := findOpportunityReview(auth, opportunity.ID, reviewType)
		if err != nil {
			utils.Error(c.Ctx, http.StatusInternalServerError, "获取商机复盘失败")
			return
		}
		if !found || !canViewOpportunityReview(auth, review) {
			utils.Success(c.Ctx, map[string]interface{}{"opportunity": opportunity, "review": nil})
			return
		}
		utils.Success(c.Ctx, map[string]interface{}{"opportunity": opportunity, "review": review})
		return
	}

	reviews := map[string]interface{}{"won": nil, "lost": nil}
	var fallbackReview interface{}
	for _, typed := range []string{"won", "lost"} {
		review, found, err := findOpportunityReview(auth, opportunity.ID, typed)
		if err != nil {
			utils.Error(c.Ctx, http.StatusInternalServerError, "获取商机复盘失败")
			return
		}
		if found && canViewOpportunityReview(auth, review) {
			reviews[typed] = review
			if fallbackReview == nil {
				fallbackReview = review
			}
		}
	}
	utils.Success(c.Ctx, map[string]interface{}{"opportunity": opportunity, "review": fallbackReview, "reviews": reviews})
}

func (c *CRMController) SaveOpportunityReview() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的商机ID")
		return
	}
	var req opportunityReviewRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.ReviewType = normalizeOpportunityReviewType(req.ReviewType)
	if req.ReviewType == "" {
		utils.BadRequest(c.Ctx, "复盘类型不能为空")
		return
	}
	var opportunity models.CRMObject
	if err := crmScopedObjectQuery(auth, "opportunities").Where("id = ? AND object_type = ?", id, "opportunity").First(&opportunity).Error; err != nil {
		utils.NotFound(c.Ctx, "商机不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, opportunity, "商机"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	reviewPayload := buildOpportunityReviewPayload(auth, opportunity, req)
	reviewName := strings.TrimSpace(req.Title)
	if reviewName == "" {
		reviewName = opportunityReviewDefaultTitle(opportunity, req.ReviewType)
	}
	var review models.CRMObject
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		existing, found, err := findOpportunityReviewTx(tx, auth, opportunity.ID, req.ReviewType)
		if err != nil {
			return err
		}
		if found {
			review = existing
			if review.Status != "draft" {
				return fmt.Errorf("该复盘已提交或发布，不能编辑")
			}
			if !canManageOpportunityReviewDraft(auth, review, opportunity) {
				return fmt.Errorf("无权更新该商机复盘")
			}
			review.Name = reviewName
			review.OwnerID = opportunity.OwnerID
			review.DepartmentID = opportunity.DepartmentID
			review.CompanyID = opportunity.CompanyID
			review.ContactID = opportunity.ContactID
			review.Amount = opportunity.Amount
			review.Currency = opportunity.Currency
			review.Score = opportunityReviewScore(req.ReviewType)
			review.Tags = strings.Join(compactTextList(req.Tags), ",")
			review.PayloadJSON = toJSONString(reviewPayload)
			review.UpdatedBy = auth.UserID
			if err := tx.Save(&review).Error; err != nil {
				return err
			}
		} else {
			review = cloneCRMObject(auth, "insights", "deal_review", reviewName, opportunity)
			review.Status = "draft"
			review.Score = opportunityReviewScore(req.ReviewType)
			review.Tags = strings.Join(compactTextList(req.Tags), ",")
			review.PayloadJSON = toJSONString(reviewPayload)
			if err := tx.Create(&review).Error; err != nil {
				return err
			}
		}
		relation := newCRMRelation(auth, "opportunities", opportunity.ID, "insights", review.ID, opportunityReviewRelationType)
		relation.PayloadJSON = toJSONString(map[string]interface{}{"review_type": req.ReviewType})
		if err := tx.Where(
			"tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?",
			auth.TenantID, "opportunities", opportunity.ID, "insights", review.ID, opportunityReviewRelationType,
		).FirstOrCreate(&relation).Error; err != nil {
			return err
		}
		relation.PayloadJSON = toJSONString(map[string]interface{}{"review_type": req.ReviewType})
		if err := tx.Save(&relation).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "opportunities", opportunity.ID, "opportunity_review_saved", "保存商机复盘", map[string]interface{}{"review_id": review.ID, "review_type": req.ReviewType})
	})
	if err != nil {
		if strings.Contains(err.Error(), "无权") {
			utils.Error(c.Ctx, http.StatusForbidden, err.Error())
			return
		}
		if strings.Contains(err.Error(), "不能编辑") {
			utils.Error(c.Ctx, http.StatusBadRequest, err.Error())
			return
		}
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存商机复盘失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SAVE_OPPORTUNITY_REVIEW", fmt.Sprintf("商机复盘[%d]", review.ID), "operation.save_opportunity_review", map[string]interface{}{"opportunity_id": opportunity.ID, "review_id": review.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"opportunity": opportunity, "review": review})
}

func (c *CRMController) SubmitOpportunityReview() {
	c.changeOpportunityReviewStatus("pending_review", "提交商机复盘审核")
}

func (c *CRMController) PublishOpportunityReview() {
	c.changeOpportunityReviewStatus("published", "发布商机复盘")
}

func (c *CRMController) ArchiveOpportunityReview() {
	c.changeOpportunityReviewStatus("archived", "归档商机复盘")
}

func (c *CRMController) changeOpportunityReviewStatus(status string, title string) {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的复盘ID")
		return
	}
	var review models.CRMObject
	if err := crmScopedObjectQuery(auth, "insights").Where("id = ? AND object_type = ?", id, "deal_review").First(&review).Error; err != nil {
		utils.NotFound(c.Ctx, "复盘不存在")
		return
	}
	payload := payloadMap(review.PayloadJSON)
	opportunityID := uintFromValue(payload["opportunity_id"])
	var opportunity models.CRMObject
	hasOpportunity := false
	if opportunityID > 0 {
		if err := crmScopedObjectQuery(auth, "opportunities").Where("id = ? AND object_type = ?", opportunityID, "opportunity").First(&opportunity).Error; err == nil {
			hasOpportunity = true
		}
	}
	if status == "pending_review" {
		canSubmitReview := auth.IsSuperAdmin() || auth.IsTenantAdmin() || review.OwnerID == auth.UserID || review.CreatedBy == auth.UserID
		if !canSubmitReview && hasOpportunity {
			canSubmitReview = ensureCRMObjectEditAccess(auth, opportunity, "商机") == nil
		}
		if !canSubmitReview {
			utils.Error(c.Ctx, http.StatusForbidden, "无权提交该复盘")
			return
		}
	} else if !auth.IsSuperAdmin() && !auth.IsTenantAdmin() && !isCRMModuleManager(auth, "insights") && !isCRMModuleManager(auth, "opportunities") {
		utils.Error(c.Ctx, http.StatusForbidden, "无权审核该复盘")
		return
	}
	now := time.Now().Format(time.RFC3339)
	switch status {
	case "pending_review":
		payload["submitted_at"] = now
		payload["submitted_by"] = auth.UserID
	case "published":
		payload["published_at"] = now
		payload["published_by"] = auth.UserID
	case "archived":
		payload["archived_at"] = now
		payload["archived_by"] = auth.UserID
	}
	review.Status = status
	review.PayloadJSON = toJSONString(payload)
	review.UpdatedBy = auth.UserID
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&review).Error; err != nil {
			return err
		}
		if opportunityID > 0 {
			return createBusinessTimeline(tx, auth, "opportunities", opportunityID, "opportunity_review_status", title, map[string]interface{}{"review_id": review.ID, "status": status})
		}
		return nil
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新复盘状态失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CHANGE_OPPORTUNITY_REVIEW_STATUS", fmt.Sprintf("复盘[%d]", review.ID), "operation.change_opportunity_review_status", map[string]interface{}{"review_id": review.ID, "status": status}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"review": review})
}

func loadVisibleOpportunity(c *CRMController, auth authContext, id uint) (models.CRMObject, bool) {
	var opportunity models.CRMObject
	if err := crmScopedObjectQuery(auth, "opportunities").Where("id = ? AND object_type = ?", id, "opportunity").First(&opportunity).Error; err != nil {
		utils.NotFound(c.Ctx, "商机不存在")
		return opportunity, false
	}
	if !canViewCRMObject(auth, opportunity) {
		utils.NotFound(c.Ctx, "商机不存在")
		return opportunity, false
	}
	return opportunity, true
}

func findOpportunityReview(auth authContext, opportunityID uint, reviewType string) (models.CRMObject, bool, error) {
	return findOpportunityReviewTx(utils.DB, auth, opportunityID, reviewType)
}

func findOpportunityReviewTx(tx *gorm.DB, auth authContext, opportunityID uint, reviewType string) (models.CRMObject, bool, error) {
	reviewType = normalizeOpportunityReviewType(reviewType)
	if reviewType == "" {
		return models.CRMObject{}, false, nil
	}
	var relations []models.CRMRelation
	err := tx.Where(
		"tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND relation_type = ?",
		auth.TenantID, "opportunities", opportunityID, "insights", opportunityReviewRelationType,
	).Order("updated_at desc").Find(&relations).Error
	if err != nil {
		return models.CRMObject{}, false, err
	}
	for _, relation := range relations {
		relationPayload := payloadMap(relation.PayloadJSON)
		if normalizeOpportunityReviewType(firstOpportunityReviewText(fmt.Sprint(relationPayload["review_type"]))) != "" &&
			normalizeOpportunityReviewType(firstOpportunityReviewText(fmt.Sprint(relationPayload["review_type"]))) != reviewType {
			continue
		}
		var review models.CRMObject
		err = tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "insights", "deal_review", relation.TargetID).First(&review).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return models.CRMObject{}, false, err
		}
		reviewPayload := payloadMap(review.PayloadJSON)
		if uintFromValue(reviewPayload["opportunity_id"]) != opportunityID {
			continue
		}
		if normalizeOpportunityReviewType(firstOpportunityReviewText(fmt.Sprint(reviewPayload["review_type"]))) != reviewType {
			continue
		}
		return review, true, nil
	}
	return models.CRMObject{}, false, nil
}

func canViewOpportunityReview(auth authContext, review models.CRMObject) bool {
	if review.Status == "published" {
		return true
	}
	return auth.IsSuperAdmin() || auth.IsTenantAdmin() || review.OwnerID == auth.UserID || review.CreatedBy == auth.UserID || isCRMModuleManager(auth, "opportunities")
}

func canManageOpportunityReviewDraft(auth authContext, review models.CRMObject, opportunity models.CRMObject) bool {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() || review.OwnerID == auth.UserID || review.CreatedBy == auth.UserID {
		return true
	}
	return ensureCRMObjectEditAccess(auth, opportunity, "商机") == nil
}

func buildOpportunityReviewPayload(auth authContext, opportunity models.CRMObject, req opportunityReviewRequest) map[string]interface{} {
	oppPayload := payloadMap(opportunity.PayloadJSON)
	payload := map[string]interface{}{}
	for key, value := range req.Payload {
		payload[key] = value
	}
	payload["review_type"] = req.ReviewType
	payload["opportunity_id"] = opportunity.ID
	payload["opportunity_name"] = opportunity.Name
	payload["company"] = firstPayloadText(oppPayload, "company", "company_name", "customer_name", "account_name")
	payload["contact_name"] = firstPayloadText(oppPayload, "contact_name", "contact", "contact_person", "buyer_name")
	payload["amount"] = opportunity.Amount
	payload["currency"] = opportunity.Currency
	payload["products"] = opportunityReviewProducts(oppPayload)
	payload["industry"] = strings.TrimSpace(req.Industry)
	if req.ReviewType == "lost" {
		payload["competitor"] = firstOpportunityReviewText(strings.TrimSpace(req.Competitor), firstPayloadText(oppPayload, "competitor"))
	} else {
		payload["competitor"] = strings.TrimSpace(req.Competitor)
	}
	payload["background"] = strings.TrimSpace(req.Background)
	payload["customer_needs"] = strings.TrimSpace(req.CustomerNeeds)
	payload["decision_process"] = strings.TrimSpace(req.DecisionProcess)
	payload["key_actions"] = strings.TrimSpace(req.KeyActions)
	payload["obstacles"] = strings.TrimSpace(req.Obstacles)
	if req.ReviewType == "won" {
		payload["win_reasons"] = firstOpportunityReviewText(strings.TrimSpace(req.WinReasons), firstPayloadText(oppPayload, "won_reason", "win_reason"))
		payload["loss_reasons"] = strings.TrimSpace(req.LossReasons)
	} else {
		payload["win_reasons"] = strings.TrimSpace(req.WinReasons)
		payload["loss_reasons"] = firstOpportunityReviewText(strings.TrimSpace(req.LossReasons), firstPayloadText(oppPayload, "loss_reason", "lost_reason"))
	}
	payload["lessons"] = strings.TrimSpace(req.Lessons)
	payload["reusable_playbook"] = strings.TrimSpace(req.ReusablePlaybook)
	payload["avoid_next_time"] = strings.TrimSpace(req.AvoidNextTime)
	payload["summary"] = strings.TrimSpace(req.Summary)
	payload["tags"] = compactTextList(req.Tags)
	payload["keywords"] = compactTextList(req.Keywords)
	payload["source"] = "opportunity_review"
	payload["created_from"] = "opportunity_review_flow"
	payload["updated_by"] = auth.UserID
	payload["updated_at"] = time.Now().Format(time.RFC3339)
	return payload
}

func normalizeOpportunityReviewType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "win", "won", "closed_won":
		return "won"
	case "loss", "lost", "closed_lost":
		return "lost"
	default:
		return ""
	}
}

func opportunityReviewDefaultTitle(opportunity models.CRMObject, reviewType string) string {
	label := "赢单复盘"
	if reviewType == "lost" {
		label = "输单复盘"
	}
	return label + "：" + opportunity.Name
}

func opportunityReviewScore(reviewType string) int {
	if reviewType == "won" {
		return 85
	}
	return 70
}

func opportunityReviewProducts(payload map[string]interface{}) []string {
	lines := interfaceList(payload["line_items"])
	products := make([]string, 0, len(lines))
	for _, item := range lines {
		if row, ok := item.(map[string]interface{}); ok {
			name := firstOpportunityReviewText(fmt.Sprint(row["name"]), fmt.Sprint(row["product_name"]), fmt.Sprint(row["sku"]))
			if strings.TrimSpace(name) != "" {
				products = append(products, strings.TrimSpace(name))
			}
		}
	}
	return products
}

func compactTextList(items []string) []string {
	result := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		text := strings.TrimSpace(item)
		if text == "" || seen[text] {
			continue
		}
		seen[text] = true
		result = append(result, text)
	}
	return result
}

func firstOpportunityReviewText(values ...string) string {
	for _, value := range values {
		text := strings.TrimSpace(value)
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}
