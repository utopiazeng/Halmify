package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (c *CRMController) MergeContacts() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		PrimaryID    uint   `json:"primary_id"`
		DuplicateIDs []uint `json:"duplicate_ids"`
		Reason       string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.PrimaryID == 0 || len(req.DuplicateIDs) == 0 {
		utils.BadRequest(c.Ctx, "主联系人和重复联系人不能为空")
		return
	}
	var primary models.CRMObject
	if err := crmScopedObjectQuery(auth, "contacts").Where("id = ?", req.PrimaryID).First(&primary).Error; err != nil {
		utils.NotFound(c.Ctx, "主联系人不存在")
		return
	}
	merged := make([]uint, 0, len(req.DuplicateIDs))
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		for _, duplicateID := range req.DuplicateIDs {
			if duplicateID == 0 || duplicateID == req.PrimaryID {
				continue
			}
			var duplicate models.CRMObject
			if err := crmScopedObjectQuery(auth, "contacts").Where("id = ?", duplicateID).First(&duplicate).Error; err != nil {
				return fmt.Errorf("重复联系人[%d]不存在", duplicateID)
			}
			primaryPayload := payloadMap(primary.PayloadJSON)
			duplicatePayload := payloadMap(duplicate.PayloadJSON)
			for key, value := range duplicatePayload {
				if _, exists := primaryPayload[key]; !exists {
					primaryPayload[key] = value
				}
			}
			primary.PayloadJSON = toJSONString(primaryPayload)
			if err := tx.Model(&models.CRMRelation{}).Where("tenant_id = ? AND source_type = ? AND source_id = ?", auth.TenantID, "contacts", duplicateID).Update("source_id", primary.ID).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.CRMRelation{}).Where("tenant_id = ? AND target_type = ? AND target_id = ?", auth.TenantID, "contacts", duplicateID).Update("target_id", primary.ID).Error; err != nil {
				return err
			}
			duplicate.Status = "merged"
			duplicate.UpdatedBy = auth.UserID
			duplicate.PayloadJSON = toJSONString(map[string]interface{}{"merged_to": primary.ID, "reason": req.Reason})
			if err := tx.Save(&duplicate).Error; err != nil {
				return err
			}
			merged = append(merged, duplicateID)
		}
		primary.UpdatedBy = auth.UserID
		return tx.Save(&primary).Error
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, err.Error())
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "MERGE_CONTACTS", fmt.Sprintf("联系人[%d]", primary.ID), "operation.merge_contacts", map[string]interface{}{"primary_id": primary.ID, "duplicate_ids": merged}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"primary": primary, "merged_ids": merged})
}

func (c *CRMController) DedupeContactObjects() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		ObjectType string `json:"object_type"`
		ObjectID   uint   `json:"object_id"`
		Name       string `json:"name"`
		Phone      string `json:"phone"`
		Email      string `json:"email"`
		Company    string `json:"company"`
		Website    string `json:"website"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.ObjectType = strings.TrimSpace(req.ObjectType)
	if req.ObjectType == "" && req.ObjectID > 0 {
		var object models.CRMObject
		if err := crmScopedObjectQuery(auth, "contacts").Where("id = ?", req.ObjectID).First(&object).Error; err == nil {
			req.ObjectType = object.ObjectType
			payload := payloadMap(object.PayloadJSON)
			if req.Name == "" {
				req.Name = object.Name
			}
			if req.Phone == "" {
				req.Phone = firstPayloadText(payload, "phone", "mobile")
			}
			if req.Email == "" {
				req.Email = firstPayloadText(payload, "email")
			}
			if req.Company == "" {
				req.Company = firstPayloadText(payload, "company", "company_name")
			}
			if req.Website == "" {
				req.Website = firstPayloadText(payload, "website", "domain")
			}
		}
	}
	if req.ObjectType != "company" && req.ObjectType != "contact" {
		utils.BadRequest(c.Ctx, "查重类型只能是公司或联系人")
		return
	}
	items := findContactDuplicates(auth, req.ObjectType, req.ObjectID, req.Name, req.Phone, req.Email, req.Company, req.Website)
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "count": len(items)})
}

func (c *CRMController) ContactCustomer360() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的客户或联系人ID")
		return
	}
	var object models.CRMObject
	if err := crmScopedObjectQuery(auth, "contacts").Where("id = ?", id).First(&object).Error; err != nil {
		utils.NotFound(c.Ctx, "客户或联系人不存在")
		return
	}
	if object.ObjectType != "company" && object.ObjectType != "contact" {
		utils.BadRequest(c.Ctx, "只能查看客户或联系人360")
		return
	}
	payload := payloadMap(object.PayloadJSON)
	keyword := object.Name
	companyID := object.ID
	contactID := object.ID
	if object.ObjectType == "contact" {
		companyName := firstPayloadText(payload, "company", "company_name")
		keyword = strings.Join(uniqueNonEmptyStrings(companyName, object.Name), "\n")
		if keyword == "" {
			keyword = object.Name
		}
		companyID = object.CompanyID
	} else {
		contactID = 0
	}
	contacts := []models.CRMObject{}
	companies := []models.CRMObject{}
	if object.ObjectType == "company" {
		contacts = contact360Objects(auth, "contacts", "contact", keyword, companyID, 0)
		companies = []models.CRMObject{object}
	} else {
		companies = contact360Objects(auth, "contacts", "company", keyword, companyID, 0)
		contacts = []models.CRMObject{object}
	}
	campaigns := contact360Objects(auth, "campaigns", "campaign", keyword, companyID, contactID)
	leads := contact360Objects(auth, "leads", "lead", keyword, companyID, contactID)
	opportunities := contact360Objects(auth, "opportunities", "opportunity", keyword, companyID, contactID)
	quotes := contact360Objects(auth, "opportunities", "quote", keyword, companyID, contactID)
	deals := contact360Objects(auth, "deals", "", keyword, companyID, contactID)
	orders := contact360Objects(auth, "orders", "order", keyword, companyID, contactID)
	projects := contact360Objects(auth, "projects", "project", keyword, companyID, contactID)
	tickets := contact360Objects(auth, "tickets", "ticket", keyword, companyID, contactID)
	interactions := contact360Interactions(auth, object, keyword)
	service := contact360ServiceSummary(auth, companyID, contactID)
	dealSummary := contact360DealSummary(auth, companyID, contactID)
	summary := contact360Summary(campaigns, leads, opportunities, quotes, deals, orders, projects, tickets, interactions, service, dealSummary)
	result := map[string]interface{}{
		"object":        object,
		"contacts":      contacts,
		"companies":     companies,
		"campaigns":     campaigns,
		"leads":         leads,
		"opportunities": opportunities,
		"quotes":        quotes,
		"deals":         deals,
		"orders":        orders,
		"projects":      projects,
		"tickets":       tickets,
		"interactions":  interactions,
		"service":       service,
		"deal_summary":  dealSummary,
		"summary":       summary,
		"recent_timeline": contact360RecentTimeline(
			campaigns,
			leads,
			opportunities,
			quotes,
			deals,
			orders,
			projects,
			tickets,
			interactions,
		),
	}
	utils.Success(c.Ctx, result)
}

func (c *CRMController) BatchUpdateContactObjects() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		IDs     []uint   `json:"ids"`
		OwnerID uint     `json:"owner_id"`
		Tags    []string `json:"tags"`
		Grade   string   `json:"grade"`
		Status  string   `json:"status"`
		Reason  string   `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if len(req.IDs) == 0 {
		utils.BadRequest(c.Ctx, "请选择要批量处理的记录")
		return
	}
	if req.OwnerID > 0 && !tenantUserExists(auth.TenantID, req.OwnerID) {
		utils.BadRequest(c.Ctx, "负责人不存在或不属于当前租户")
		return
	}
	updated := []uint{}
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		for _, id := range req.IDs {
			var object models.CRMObject
			if err := crmScopedObjectQuery(auth, "contacts").Where("id = ? AND object_type IN ?", id, []string{"company", "contact"}).First(&object).Error; err != nil {
				return err
			}
			payload := payloadMap(object.PayloadJSON)
			if req.OwnerID > 0 {
				object.OwnerID = req.OwnerID
			}
			if strings.TrimSpace(req.Grade) != "" {
				payload["grade"] = strings.TrimSpace(req.Grade)
			}
			if strings.TrimSpace(req.Status) != "" {
				object.Status = strings.TrimSpace(req.Status)
			}
			if len(req.Tags) > 0 {
				object.Tags = mergeTagText(object.Tags, req.Tags)
			}
			payload["batch_updated_at"] = time.Now().Format(time.RFC3339)
			payload["batch_update_reason"] = strings.TrimSpace(req.Reason)
			object.PayloadJSON = toJSONString(payload)
			object.UpdatedBy = auth.UserID
			if err := tx.Save(&object).Error; err != nil {
				return err
			}
			if err := createBusinessTimeline(tx, auth, "contacts", object.ID, "contact_batch_update", "批量更新", map[string]interface{}{"reason": req.Reason, "owner_id": req.OwnerID, "grade": req.Grade, "status": req.Status, "tags": req.Tags}); err != nil {
				return err
			}
			updated = append(updated, id)
		}
		return nil
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "批量更新失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "BATCH_UPDATE_CONTACT_OBJECTS", "客户联系人批量更新", "operation.batch_update_contact_objects", map[string]interface{}{"ids": updated, "count": len(updated)}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"ids": updated, "count": len(updated)})
}

func (c *CRMController) RestoreContactObject() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的客户或联系人ID")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if len(c.Ctx.Input.RequestBody) > 0 {
		_ = bindJSON(c.Ctx, &req)
	}
	var object models.CRMObject
	if err := crmScopedObjectQuery(auth, "contacts").Where("id = ?", id).First(&object).Error; err != nil {
		utils.NotFound(c.Ctx, "客户或联系人不存在")
		return
	}
	if object.ObjectType != "company" && object.ObjectType != "contact" {
		utils.BadRequest(c.Ctx, "只能恢复客户或联系人")
		return
	}
	before := object.Status
	payload := payloadMap(object.PayloadJSON)
	payload["restored_at"] = time.Now().Format(time.RFC3339)
	payload["restore_reason"] = strings.TrimSpace(req.Reason)
	object.Status = "active"
	object.PayloadJSON = toJSONString(payload)
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&object).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "contacts", object.ID, "contact_restore", "恢复记录", map[string]interface{}{"reason": req.Reason, "from": before, "to": "active"})
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "恢复失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "RESTORE_CONTACT_OBJECT", fmt.Sprintf("联系人/客户[%d]", object.ID), "operation.restore_contact_object", map[string]interface{}{"id": object.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"object": object})
}

func (c *CRMController) LinkContactCompany() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		ContactID uint   `json:"contact_id"`
		CompanyID uint   `json:"company_id"`
		Primary   bool   `json:"primary"`
		Reason    string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.ContactID == 0 || req.CompanyID == 0 {
		utils.BadRequest(c.Ctx, "联系人和公司不能为空")
		return
	}
	var contact, company models.CRMObject
	if err := crmScopedObjectQuery(auth, "contacts").Where("id = ? AND object_type = ?", req.ContactID, "contact").First(&contact).Error; err != nil {
		utils.NotFound(c.Ctx, "联系人不存在")
		return
	}
	if err := crmScopedObjectQuery(auth, "contacts").Where("id = ? AND object_type = ?", req.CompanyID, "company").First(&company).Error; err != nil {
		utils.NotFound(c.Ctx, "公司不存在")
		return
	}
	payload := payloadMap(contact.PayloadJSON)
	if req.Primary {
		contact.CompanyID = company.ID
		payload["company"] = company.Name
		payload["company_id"] = company.ID
	}
	contact.PayloadJSON = toJSONString(payload)
	relation := newCRMRelation(auth, "contacts", company.ID, "contacts", contact.ID, "company_contact")
	relation.PayloadJSON = toJSONString(map[string]interface{}{"primary": req.Primary, "reason": strings.TrimSpace(req.Reason)})
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&contact).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?", auth.TenantID, "contacts", company.ID, "contacts", contact.ID, "company_contact").FirstOrCreate(&relation).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "contacts", company.ID, "company_contact_link", "关联联系人", map[string]interface{}{"reason": req.Reason, "contact_id": contact.ID, "primary": req.Primary})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "关联失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "LINK_CONTACT_COMPANY", fmt.Sprintf("公司[%d]-联系人[%d]", company.ID, contact.ID), "operation.link_contact_company", map[string]interface{}{"company_id": company.ID, "contact_id": contact.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"company": company, "contact": contact, "relation": relation})
}

func (c *CRMController) UnlinkContactCompany() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		ContactID uint   `json:"contact_id"`
		CompanyID uint   `json:"company_id"`
		Reason    string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if req.ContactID == 0 || req.CompanyID == 0 {
		utils.BadRequest(c.Ctx, "联系人和公司不能为空")
		return
	}
	var contact, company models.CRMObject
	if err := crmScopedObjectQuery(auth, "contacts").Where("id = ? AND object_type = ?", req.ContactID, "contact").First(&contact).Error; err != nil {
		utils.NotFound(c.Ctx, "联系人不存在")
		return
	}
	if err := crmScopedObjectQuery(auth, "contacts").Where("id = ? AND object_type = ?", req.CompanyID, "company").First(&company).Error; err != nil {
		utils.NotFound(c.Ctx, "公司不存在")
		return
	}
	payload := payloadMap(contact.PayloadJSON)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "解除公司关联"
	}
	if contact.CompanyID == company.ID {
		contact.CompanyID = 0
		payload["company"] = ""
		payload["company_id"] = 0
	}
	contact.PayloadJSON = toJSONString(payload)
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&contact).Error; err != nil {
			return err
		}
		if err := tx.Where(
			"tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?",
			auth.TenantID, "contacts", company.ID, "contacts", contact.ID, "company_contact",
		).Delete(&models.CRMRelation{}).Error; err != nil {
			return err
		}
		if err := tx.Where(
			"tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?",
			auth.TenantID, "contacts", contact.ID, "contacts", company.ID, "company_contact",
		).Delete(&models.CRMRelation{}).Error; err != nil {
			return err
		}
		if err := createBusinessTimeline(tx, auth, "contacts", company.ID, "company_contact_unlink", "解除联系人关联", map[string]interface{}{"reason": reason, "contact_id": contact.ID}); err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "contacts", contact.ID, "contact_company_unlink", "解除公司关联", map[string]interface{}{"reason": reason, "company_id": company.ID})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "解除关联失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "UNLINK_CONTACT_COMPANY", fmt.Sprintf("公司[%d]-联系人[%d]", company.ID, contact.ID), "operation.unlink_contact_company", map[string]interface{}{"company_id": company.ID, "contact_id": contact.ID, "reason": reason}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"company": company, "contact": contact})
}

func findContactDuplicates(auth authContext, objectType string, excludeID uint, name string, phone string, email string, company string, website string) []models.CRMObject {
	query := crmScopedObjectQuery(auth, "contacts").Where("object_type = ?", objectType)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	conditions := []string{}
	args := []interface{}{}
	if objectType == "company" {
		for _, value := range []string{name, company, website, normalizeMatchText(name), normalizeMatchText(company)} {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			sub := []string{"name LIKE ?"}
			subArgs := []interface{}{"%" + value + "%"}
			if clause, clauseArgs := payloadJSONKeyLikeClause(value, "company", "company_name", "website", "domain"); clause != "" {
				sub = append(sub, clause)
				subArgs = append(subArgs, clauseArgs...)
			}
			conditions = append(conditions, "("+strings.Join(sub, " OR ")+")")
			args = append(args, subArgs...)
		}
	} else {
		for _, value := range []string{name, phone, email} {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			sub := []string{"name LIKE ?"}
			subArgs := []interface{}{"%" + value + "%"}
			if clause, clauseArgs := payloadJSONKeyLikeClause(value, "phone", "mobile", "email", "contact_name", "contact", "name"); clause != "" {
				sub = append(sub, clause)
				subArgs = append(subArgs, clauseArgs...)
			}
			conditions = append(conditions, "("+strings.Join(sub, " OR ")+")")
			args = append(args, subArgs...)
		}
	}
	if len(conditions) == 0 {
		return []models.CRMObject{}
	}
	var items []models.CRMObject
	_ = query.Where(strings.Join(conditions, " OR "), args...).Limit(20).Find(&items).Error
	return items
}

func contact360Objects(auth authContext, domain string, objectType string, keyword string, companyID uint, contactID uint) []models.CRMObject {
	query := crmScopedObjectQuery(auth, domain)
	if objectType != "" {
		query = query.Where("object_type = ?", objectType)
	}
	filters := []string{}
	args := []interface{}{}
	keywords := splitContact360Keywords(keyword)
	if companyID > 0 {
		sub := []string{"company_id = ?"}
		subArgs := []interface{}{companyID}
		if clause, clauseArgs := payloadJSONKeyNumberLikeClause(companyID, "company_id"); clause != "" {
			sub = append(sub, clause)
			subArgs = append(subArgs, clauseArgs...)
		}
		filters = append(filters, "("+strings.Join(sub, " OR ")+")")
		args = append(args, subArgs...)
	}
	if contactID > 0 {
		sub := []string{"contact_id = ?"}
		subArgs := []interface{}{contactID}
		if clause, clauseArgs := payloadJSONKeyNumberLikeClause(contactID, "contact_id"); clause != "" {
			sub = append(sub, clause)
			subArgs = append(subArgs, clauseArgs...)
		}
		filters = append(filters, "("+strings.Join(sub, " OR ")+")")
		args = append(args, subArgs...)
	}
	for _, keyword := range keywords {
		sub := []string{"name LIKE ?"}
		subArgs := []interface{}{"%" + keyword + "%"}
		if clause, clauseArgs := payloadJSONKeyLikeClause(keyword, "company", "company_name", "contact", "contact_name", "email", "phone", "mobile", "website", "domain", "title", "summary"); clause != "" {
			sub = append(sub, clause)
			subArgs = append(subArgs, clauseArgs...)
		}
		filters = append(filters, "("+strings.Join(sub, " OR ")+")")
		args = append(args, subArgs...)
	}
	if len(filters) == 0 {
		return []models.CRMObject{}
	}
	var items []models.CRMObject
	_ = query.Where("("+strings.Join(filters, " OR ")+")", args...).Order("updated_at desc").Limit(100).Find(&items).Error
	return items
}

func splitContact360Keywords(keyword string) []string {
	parts := strings.FieldsFunc(keyword, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	return uniqueNonEmptyStrings(parts...)
}

func uniqueNonEmptyStrings(values ...string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func contact360Summary(campaigns, leads, opportunities, quotes, deals, orders, projects, tickets, interactions []models.CRMObject, service map[string]interface{}, dealSummary map[string]interface{}) map[string]interface{} {
	summary := map[string]interface{}{
		"campaign_count":         len(campaigns),
		"lead_count":             len(leads),
		"opportunity_count":      len(opportunities),
		"quote_count":            len(quotes),
		"deal_count":             len(deals),
		"order_count":            len(orders),
		"project_count":          len(projects),
		"ticket_count":           len(tickets),
		"interaction_count":      len(interactions),
		"active_opportunities":   intValue(dealSummary["active_opportunities"]),
		"closed_deals":           intValue(dealSummary["closed_deals"]),
		"open_orders":            intValue(dealSummary["open_orders"]),
		"ticket_processing":      intValue(service["ticket_processing"]),
		"active_pipeline_amount": numericValue(dealSummary["active_pipeline_amount"]),
		"closed_revenue":         numericValue(dealSummary["closed_revenue"]),
		"open_order_amount":      numericValue(dealSummary["open_order_amount"]),
		"open_receivable":        numericValue(dealSummary["open_receivable"]),
		"avg_csat":               numericValue(service["avg_csat"]),
	}
	if len(interactions) > 0 {
		summary["last_interaction_at"] = interactions[0].CreatedAt.Format(time.RFC3339)
	}
	if summary["active_opportunities"] == 0 {
		active := 0
		activeAmount := 0.0
		for _, item := range opportunities {
			if containsString(activeOpportunityStages, item.Status) {
				active++
				activeAmount += item.Amount
			}
		}
		summary["active_opportunities"] = active
		if numericValue(summary["active_pipeline_amount"]) == 0 {
			summary["active_pipeline_amount"] = activeAmount
		}
	}
	if summary["closed_deals"] == 0 {
		closed := 0
		closedRevenue := 0.0
		for _, item := range deals {
			if item.Status == "closed_won" || item.Status == "archived" || item.ObjectType == "closed_deal" {
				closed++
				closedRevenue += item.Amount
			}
		}
		summary["closed_deals"] = closed
		if numericValue(summary["closed_revenue"]) == 0 {
			summary["closed_revenue"] = closedRevenue
		}
	}
	if numericValue(summary["open_receivable"]) == 0 || intValue(summary["open_orders"]) == 0 {
		openOrders := 0
		openReceivable := 0.0
		openOrderAmount := 0.0
		for _, item := range orders {
			if containsString([]string{"completed", "canceled"}, item.Status) {
				continue
			}
			payload := payloadMap(item.PayloadJSON)
			openOrders++
			openOrderAmount += item.Amount
			openReceivable += currentReceivable(item, payload)
		}
		if intValue(summary["open_orders"]) == 0 {
			summary["open_orders"] = openOrders
		}
		if numericValue(summary["open_order_amount"]) == 0 {
			summary["open_order_amount"] = openOrderAmount
		}
		if numericValue(summary["open_receivable"]) == 0 {
			summary["open_receivable"] = openReceivable
		}
	}
	if summary["ticket_processing"] == 0 {
		processing := 0
		for _, item := range tickets {
			if containsString([]string{"new", "assigned", "processing", "waiting_customer", "reopened"}, item.Status) {
				processing++
			}
		}
		summary["ticket_processing"] = processing
	}
	return summary
}

func contact360RecentTimeline(groups ...[]models.CRMObject) []map[string]interface{} {
	type timelineEntry struct {
		item models.CRMObject
		at   time.Time
	}
	entries := []timelineEntry{}
	for _, group := range groups {
		for _, item := range group {
			at := item.UpdatedAt
			if item.ObjectType == "interaction" {
				at = item.CreatedAt
			}
			entries = append(entries, timelineEntry{item: item, at: at})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].at.After(entries[j].at)
	})
	if len(entries) > 20 {
		entries = entries[:20]
	}
	result := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		item := entry.item
		payload := payloadMap(item.PayloadJSON)
		summary := firstPayloadText(payload, "summary", "note", "description", "source_channel", "category")
		operatorID := uintFromValue(payload["operator_id"])
		if operatorID == 0 {
			operatorID = item.CreatedBy
		}
		operatedAt := firstPayloadText(payload, "operated_at", "happened_at")
		if operatedAt == "" {
			operatedAt = entry.at.Format(time.RFC3339)
		}
		result = append(result, map[string]interface{}{
			"domain":      item.Domain,
			"object_type": item.ObjectType,
			"id":          item.ID,
			"title":       item.Name,
			"status":      item.Status,
			"amount":      item.Amount,
			"currency":    item.Currency,
			"happened_at": entry.at.Format(time.RFC3339),
			"summary":     summary,
			"operator_id":  operatorID,
			"operator_name": firstPayloadText(payload, "operator_name", "operator"),
			"operated_at":  operatedAt,
		})
	}
	return result
}

func contact360Interactions(auth authContext, object models.CRMObject, keyword string) []models.CRMObject {
	ids := []uint{}
	var relations []models.CRMRelation
	_ = utils.DB.Where("tenant_id = ? AND relation_type = ? AND ((source_type = ? AND source_id = ?) OR (target_type = ? AND target_id = ?))", auth.TenantID, "timeline", "contacts", object.ID, "contacts", object.ID).Find(&relations).Error
	for _, relation := range relations {
		if relation.TargetType == "contacts" {
			ids = append(ids, relation.TargetID)
		}
		if relation.SourceType == "contacts" {
			ids = append(ids, relation.SourceID)
		}
	}
	query := crmScopedObjectQuery(auth, "contacts").Where("object_type = ?", "interaction")
	keywords := splitContact360Keywords(keyword)
	filters := []string{}
	args := []interface{}{}
	if len(ids) > 0 {
		filters = append(filters, "id IN ?")
		args = append(args, ids)
	}
	if len(keywords) > 0 {
		for _, keyword := range keywords {
			keyword = strings.TrimSpace(keyword)
			if keyword == "" {
				continue
			}
			sub := []string{"name LIKE ?"}
			subArgs := []interface{}{"%" + keyword + "%"}
			if clause, clauseArgs := payloadJSONKeyLikeClause(keyword, "summary", "content", "email", "phone", "mobile", "company", "company_name", "contact", "contact_name", "interaction_type"); clause != "" {
				sub = append(sub, clause)
				subArgs = append(subArgs, clauseArgs...)
			}
			filters = append(filters, "("+strings.Join(sub, " OR ")+")")
			args = append(args, subArgs...)
		}
	}
	if object.ID > 0 {
		sub := []string{}
		subArgs := []interface{}{}
		if clause, clauseArgs := payloadJSONKeyNumberLikeClause(object.ID, "related_id", "related_object_id"); clause != "" {
			sub = append(sub, clause)
			subArgs = append(subArgs, clauseArgs...)
		}
		if object.ObjectType == "contact" {
			if clause, clauseArgs := payloadJSONKeyLikeClause("contact", "related_type"); clause != "" {
				sub = append(sub, clause)
				subArgs = append(subArgs, clauseArgs...)
			}
		}
		if object.ObjectType == "company" {
			if clause, clauseArgs := payloadJSONKeyLikeClause("company", "related_type"); clause != "" {
				sub = append(sub, clause)
				subArgs = append(subArgs, clauseArgs...)
			}
		}
		if len(sub) > 0 {
			filters = append(filters, "("+strings.Join(sub, " OR ")+")")
			args = append(args, subArgs...)
		}
	}
	if len(filters) == 0 {
		return []models.CRMObject{}
	}
	var items []models.CRMObject
	_ = query.Where("("+strings.Join(filters, " OR ")+")", args...).Order("created_at desc").Limit(100).Find(&items).Error
	return items
}

func contact360ServiceSummary(auth authContext, companyID uint, contactID uint) map[string]interface{} {
	filter := func(query *gorm.DB) *gorm.DB {
		query = query.Where("object_type = ?", "ticket")
		if companyID > 0 {
			query = query.Where("company_id = ?", companyID)
		}
		if contactID > 0 {
			query = query.Where("contact_id = ?", contactID)
		}
		return query
	}
	base := filter(crmScopedObjectQuery(auth, "tickets"))
	var total int64
	_ = base.Count(&total).Error
	var last30 int64
	_ = filter(crmScopedObjectQuery(auth, "tickets")).
		Where("created_at >= ?", time.Now().Add(-30*24*time.Hour)).
		Count(&last30).Error
	var processing int64
	_ = filter(crmScopedObjectQuery(auth, "tickets")).
		Where("status IN ?", []string{"new", "assigned", "processing", "waiting_customer", "reopened"}).
		Count(&processing).Error
	var tickets []models.CRMObject
	_ = filter(crmScopedObjectQuery(auth, "tickets")).
		Where("status = ?", "closed").
		Order("updated_at desc").
		Limit(100).
		Find(&tickets).Error
	csatSum := 0.0
	csatCount := 0.0
	samples := []map[string]interface{}{}
	for _, ticket := range tickets {
		payload := payloadMap(ticket.PayloadJSON)
		satisfaction := mapValue(payload["satisfaction"])
		csat := int(numericValue(satisfaction["csat"]))
		nps := int(numericValue(satisfaction["nps"]))
		if csat <= 0 && nps == 0 {
			continue
		}
		if csat > 0 {
			csatSum += float64(csat)
			csatCount++
		}
		samples = append(samples, map[string]interface{}{
			"ticket_id": ticket.ID,
			"name":      ticket.Name,
			"csat":      csat,
			"nps":       nps,
			"at":        ticket.UpdatedAt.Format(time.RFC3339),
		})
		if len(samples) >= 6 {
			break
		}
	}
	avgCSAT := 0.0
	if csatCount > 0 {
		avgCSAT = csatSum / csatCount
	}
	return map[string]interface{}{
		"ticket_total":         total,
		"ticket_last_30d":      last30,
		"ticket_processing":    processing,
		"avg_csat":             avgCSAT,
		"satisfaction_samples": samples,
	}
}

func contact360DealSummary(auth authContext, companyID uint, contactID uint) map[string]interface{} {
	summary := map[string]interface{}{
		"active_pipeline_amount": 0.0,
		"closed_revenue":         0.0,
		"open_order_amount":      0.0,
		"open_receivable":        0.0,
		"active_opportunities":   0,
		"closed_deals":           0,
		"open_orders":            0,
	}
	filter := func(query *gorm.DB) *gorm.DB {
		if companyID > 0 {
			query = query.Where("company_id = ?", companyID)
		}
		if contactID > 0 {
			query = query.Where("contact_id = ?", contactID)
		}
		return query
	}
	var opportunities []models.CRMObject
	_ = filter(crmScopedObjectQuery(auth, "opportunities")).
		Where("object_type = ? AND status IN ?", "opportunity", activeOpportunityStages).
		Find(&opportunities).Error
	for _, item := range opportunities {
		summary["active_pipeline_amount"] = numericValue(summary["active_pipeline_amount"]) + item.Amount
		summary["active_opportunities"] = intValue(summary["active_opportunities"]) + 1
	}
	var deals []models.CRMObject
	_ = filter(crmScopedObjectQuery(auth, "deals")).
		Where("object_type IN ? AND status IN ?", []string{"closed_deal", "deal"}, []string{"closed_won", "archived"}).
		Find(&deals).Error
	for _, item := range deals {
		summary["closed_revenue"] = numericValue(summary["closed_revenue"]) + item.Amount
		summary["closed_deals"] = intValue(summary["closed_deals"]) + 1
	}
	var orders []models.CRMObject
	_ = filter(crmScopedObjectQuery(auth, "orders")).
		Where("object_type = ? AND status NOT IN ?", "order", []string{"completed", "canceled"}).
		Find(&orders).Error
	for _, item := range orders {
		payload := payloadMap(item.PayloadJSON)
		summary["open_order_amount"] = numericValue(summary["open_order_amount"]) + item.Amount
		summary["open_receivable"] = numericValue(summary["open_receivable"]) + currentReceivable(item, payload)
		summary["open_orders"] = intValue(summary["open_orders"]) + 1
	}
	return summary
}
