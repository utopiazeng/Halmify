package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"time"
)

type crmObjectRequest struct {
	ObjectType   string                 `json:"object_type"`
	Name         string                 `json:"name"`
	Code         string                 `json:"code"`
	Status       string                 `json:"status"`
	OwnerID      uint                   `json:"owner_id"`
	DepartmentID uint                   `json:"department_id"`
	CompanyID    uint                   `json:"company_id"`
	ContactID    uint                   `json:"contact_id"`
	DealID       uint                   `json:"deal_id"`
	ProjectID    uint                   `json:"project_id"`
	Amount       float64                `json:"amount"`
	Currency     string                 `json:"currency"`
	Score        int                    `json:"score"`
	StartAt      string                 `json:"start_at"`
	EndAt        string                 `json:"end_at"`
	Tags         []string               `json:"tags"`
	Payload      map[string]interface{} `json:"payload"`
}

type crmRelationRequest struct {
	SourceType   string                 `json:"source_type"`
	SourceID     uint                   `json:"source_id"`
	TargetType   string                 `json:"target_type"`
	TargetID     uint                   `json:"target_id"`
	RelationType string                 `json:"relation_type"`
	Payload      map[string]interface{} `json:"payload"`
}

func (c *CRMController) ListObjects() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domain := strings.TrimSpace(c.Ctx.Input.Query(":domain"))
	if !validCRMDomain(domain) {
		utils.BadRequest(c.Ctx, "无效的业务域")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	objectType := strings.TrimSpace(c.Ctx.Input.Query("object_type"))
	status := strings.TrimSpace(c.Ctx.Input.Query("status"))
	keyword := strings.TrimSpace(c.Ctx.Input.Query("keyword"))
	ownerID := queryUint(c.Ctx, "owner_id")
	companyID := queryUint(c.Ctx, "company_id")
	contactID := queryUint(c.Ctx, "contact_id")
	dealID := queryUint(c.Ctx, "deal_id")
	projectID := queryUint(c.Ctx, "project_id")
	companyName := strings.TrimSpace(c.Ctx.Input.Query("company_name"))
	contactName := strings.TrimSpace(c.Ctx.Input.Query("contact_name"))
	phone := strings.TrimSpace(c.Ctx.Input.Query("phone"))
	email := strings.TrimSpace(c.Ctx.Input.Query("email"))
	source := strings.TrimSpace(c.Ctx.Input.Query("source_channel"))
	language := strings.TrimSpace(c.Ctx.Input.Query("language"))
	priority := strings.TrimSpace(c.Ctx.Input.Query("priority"))
	importance := strings.TrimSpace(c.Ctx.Input.Query("importance"))
	reliability := strings.TrimSpace(c.Ctx.Input.Query("reliability"))
	minScore := queryInt(c.Ctx, "min_score", 0)
	minAmount := queryFloat(c.Ctx, "min_amount", 0)
	maxAmount := queryFloat(c.Ctx, "max_amount", 0)
	excludeStatuses := splitCSV(c.Ctx.Input.Query("exclude_status"))
	page := queryInt(c.Ctx, "page", 1)
	size := queryInt(c.Ctx, "size", 20)
	if size > 100 {
		utils.Error(c.Ctx, http.StatusTooManyRequests, "分页大小超过安全限制")
		return
	}

	query := crmScopedObjectQuery(auth, domain)
	if objectType != "" {
		query = query.Where("object_type = ?", objectType)
	}
	if domain == "insights" && !auth.IsSuperAdmin() && !auth.IsTenantAdmin() && !isCRMModuleManager(auth, "insights") && !isCRMModuleManager(auth, "opportunities") {
		query = query.Where("(object_type <> ? OR status = ? OR owner_id = ? OR created_by = ?)", "deal_review", "published", auth.UserID, auth.UserID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if len(excludeStatuses) > 0 {
		query = query.Where("status NOT IN ?", excludeStatuses)
	}
	if minScore > 0 {
		query = query.Where("score >= ?", minScore)
	}
	if minAmount > 0 {
		query = query.Where("amount >= ?", minAmount)
	}
	if maxAmount > 0 {
		query = query.Where("amount <= ?", maxAmount)
	}
	if keyword != "" {
		if domain == "contacts" {
			normalizedKeyword := normalizeSearchKeyword(keyword)
			if normalizedKeyword != "" {
				query = query.Where(
					"name LIKE ? OR code LIKE ? OR tags LIKE ? OR payload_json LIKE ?",
					"%"+keyword+"%",
					"%"+keyword+"%",
					"%"+keyword+"%",
					"%"+normalizedKeyword+"%",
				)
			} else {
				query = query.Where("name LIKE ? OR code LIKE ? OR tags LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
			}
		} else {
			query = applyCRMListKeywordFilter(query, domain, objectType, keyword)
		}
	}
	if ownerID > 0 {
		query = query.Where("owner_id = ?", ownerID)
	}
	if companyID > 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if contactID > 0 {
		query = query.Where("contact_id = ?", contactID)
	}
	if dealID > 0 {
		query = query.Where("deal_id = ?", dealID)
	}
	if projectID > 0 {
		query = query.Where("project_id = ?", projectID)
	}
	if companyName != "" && supportsFuzzyCompanyContactPayloadSearch(domain, objectType) {
		query = applyPayloadJSONKeyContains(query, companyName, "company_name", "company")
	} else if companyName != "" && supportsCompanyContactPayloadSearch(domain) {
		query = applyPayloadJSONKeyLike(query, companyName, "company_name", "company")
	}
	if contactName != "" && supportsFuzzyCompanyContactPayloadSearch(domain, objectType) {
		query = applyPayloadJSONKeyContains(query, contactName, "contact_name", "contact", "name_contact")
	} else if contactName != "" && supportsCompanyContactPayloadSearch(domain) {
		query = applyPayloadJSONKeyLike(query, contactName, "contact_name", "contact", "name")
	}
	if domain == "leads" {
		if phone != "" {
			query = applyPayloadJSONKeyLike(query, phone, "phone", "mobile")
		}
		if email != "" {
			query = applyPayloadJSONKeyLike(query, email, "email")
		}
		if source != "" {
			query = applyPayloadJSONKeyLike(query, source, "source_channel", "utm_source")
		}
	}
	if domain == "tickets" && language != "" {
		query = applyPayloadJSONKeyLike(query, language, "language", "locale")
	}
	if priority != "" {
		if priority == "normal" {
			query = query.Where("(payload_json LIKE ? OR payload_json NOT LIKE ?)", `%\"priority\":\"normal\"%`, `%\"priority\":%`)
		} else {
			query = applyPayloadJSONKeyLike(query, priority, "priority")
		}
	}
	if importance != "" {
		if importance == "normal" {
			query = query.Where("(payload_json LIKE ? OR payload_json NOT LIKE ?)", `%\"importance\":\"normal\"%`, `%\"importance\":%`)
		} else {
			query = applyPayloadJSONKeyLike(query, importance, "importance")
		}
	}
	if reliability != "" {
		if reliability == "medium" {
			query = query.Where("(payload_json LIKE ? OR payload_json NOT LIKE ?)", `%\"reliability\":\"medium\"%`, `%\"reliability\":%`)
		} else {
			query = applyPayloadJSONKeyLike(query, reliability, "reliability")
		}
	}
	var total int64
	query.Count(&total)
	var items []models.CRMObject
	if err := query.Order("updated_at desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取业务对象失败")
		return
	}
	items = applyCRMFieldACL(auth, domain, items)
	items = applyCRMObjectCapabilities(auth, items)
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": total})
}

func supportsCompanyContactPayloadSearch(domain string) bool {
	switch domain {
	case "leads", "opportunities", "deals", "orders", "projects", "tickets":
		return true
	default:
		return false
	}
}

func supportsFuzzyCompanyContactPayloadSearch(domain string, objectType string) bool {
	switch domain {
	case "deals":
		return objectType == "" || objectType == "closed_deal"
	case "orders":
		return objectType == "" || objectType == "order"
	default:
		return false
	}
}

func applyCRMListKeywordFilter(query *gorm.DB, domain string, objectType string, keyword string) *gorm.DB {
	baseClause := "name LIKE ? OR code LIKE ? OR tags LIKE ?"
	baseArgs := []interface{}{"%" + keyword + "%", "%" + keyword + "%", "%" + keyword + "%"}
	payloadKeys := crmListKeywordPayloadKeys(domain, objectType)
	if clause, args := payloadJSONKeyContainsClause(keyword, payloadKeys...); clause != "" {
		baseClause += " OR " + clause
		baseArgs = append(baseArgs, args...)
	}
	return query.Where(baseClause, baseArgs...)
}

func crmListKeywordPayloadKeys(domain string, objectType string) []string {
	switch domain {
	case "deals":
		if objectType == "" || objectType == "closed_deal" {
			return []string{"company_name", "company", "contact_name", "contact", "name_contact", "contract_no", "invoice_no", "source_channel", "growth_source", "sales_owner"}
		}
	case "orders":
		if objectType == "" || objectType == "order" {
			return []string{"company_name", "company", "contact_name", "contact", "name_contact", "shipment_no", "payment_no", "invoice_no", "shipping_address", "delivery_address", "delivery_note", "carrier"}
		}
	}
	return nil
}

func applyPayloadJSONKeyContains(query *gorm.DB, value string, keys ...string) *gorm.DB {
	clause, args := payloadJSONKeyContainsClause(value, keys...)
	if clause == "" {
		value = strings.TrimSpace(value)
		if value == "" {
			return query
		}
		return query.Where("payload_json LIKE ?", "%"+value+"%")
	}
	return query.Where(clause, args...)
}

func payloadJSONKeyContainsClause(value string, keys ...string) (string, []interface{}) {
	value = strings.TrimSpace(value)
	if value == "" || len(keys) == 0 {
		return "", nil
	}
	fragment, ok := jsonStringSearchFragment(value)
	if !ok {
		return "", nil
	}
	parts := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		parts = append(parts, "payload_json LIKE ?")
		args = append(args, fmt.Sprintf("%%\"%s\":%%%s%%", key, fragment))
	}
	if len(parts) == 0 {
		return "", nil
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}

func jsonStringSearchFragment(value string) (string, bool) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", false
	}
	text := string(encoded)
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1], true
	}
	return text, true
}

func applyPayloadJSONKeyLike(query *gorm.DB, value string, keys ...string) *gorm.DB {
	clause, args := payloadJSONKeyLikeClause(value, keys...)
	if clause == "" {
		value = strings.TrimSpace(value)
		if value == "" {
			return query
		}
		return query.Where("payload_json LIKE ?", "%"+value+"%")
	}
	return query.Where(clause, args...)
}

func payloadJSONKeyLikeClause(value string, keys ...string) (string, []interface{}) {
	value = strings.TrimSpace(value)
	if value == "" || len(keys) == 0 {
		return "", nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", nil
	}
	parts := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		parts = append(parts, "payload_json LIKE ?")
		args = append(args, fmt.Sprintf("%%\"%s\":%s%%", key, string(encoded)))
	}
	if len(parts) == 0 {
		return "", nil
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}

func payloadJSONKeyBoolLikeClause(value bool, keys ...string) (string, []interface{}) {
	if len(keys) == 0 {
		return "", nil
	}
	boolText := "false"
	if value {
		boolText = "true"
	}
	parts := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		parts = append(parts, "payload_json LIKE ?")
		args = append(args, fmt.Sprintf("%%\"%s\":%s%%", key, boolText))
	}
	if len(parts) == 0 {
		return "", nil
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}

func payloadJSONKeyNumberLikeClause(number uint, keys ...string) (string, []interface{}) {
	if number == 0 || len(keys) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		parts = append(parts, "payload_json LIKE ?")
		args = append(args, fmt.Sprintf("%%\"%s\":%d%%", key, number))
	}
	if len(parts) == 0 {
		return "", nil
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}

func payloadJSONKeyNumberEqualsClause(number uint, keys ...string) (string, []interface{}) {
	if number == 0 || len(keys) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys)*3)
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		parts = append(parts, "(JSON_UNQUOTE(JSON_EXTRACT(payload_json, ?)) = ? OR JSON_EXTRACT(payload_json, ?) = CAST(? AS JSON))")
		args = append(args, "$."+key, fmt.Sprint(number), "$."+key, fmt.Sprint(number))
	}
	if len(parts) == 0 {
		return "", nil
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}

func (c *CRMController) GetObject() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domain := strings.TrimSpace(c.Ctx.Input.Query(":domain"))
	id, valid := pathUint(c.Ctx, ":id")
	if !valid || !validCRMDomain(domain) {
		utils.BadRequest(c.Ctx, "无效的业务对象")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var object models.CRMObject
	if err := crmScopedObjectQuery(auth, domain).Where("id = ?", id).First(&object).Error; err != nil {
		utils.NotFound(c.Ctx, "业务对象不存在")
		return
	}
	if domain == "insights" && object.ObjectType == "deal_review" && !canViewOpportunityReview(auth, object) {
		utils.NotFound(c.Ctx, "业务对象不存在")
		return
	}
	items := applyCRMFieldACL(auth, domain, []models.CRMObject{object})
	items = applyCRMObjectCapabilities(auth, items)
	if len(items) > 0 {
		object = items[0]
	}
	utils.Success(c.Ctx, object)
}

func (c *CRMController) CreateObject() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domain := strings.TrimSpace(c.Ctx.Input.Query(":domain"))
	if !validCRMDomain(domain) {
		utils.BadRequest(c.Ctx, "无效的业务域")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req crmObjectRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if err := validateCRMFieldWrite(auth, domain, req.Payload, nil); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	object, err := crmObjectFromRequest(auth, domain, req)
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	if err := validateLeadRequiredFields(object); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	if domain == "leads" && object.ObjectType == "lead" {
		if err := utils.DB.Transaction(func(tx *gorm.DB) error {
			if err := ensureLeadCompanyContactTx(tx, auth, &object); err != nil {
				return err
			}
			if err := tx.Create(&object).Error; err != nil {
				return err
			}
			return createLeadCompanyContactRelationsTx(tx, auth, object)
		}); err != nil {
			logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_CRM_OBJECT", fmt.Sprintf("%s[0]", domain), "operation.create_crm_object", map[string]interface{}{"domain": domain, "name": req.Name}, "failed", err.Error())
			if isLeadRelationValidationError(err) {
				utils.BadRequest(c.Ctx, err.Error())
				return
			}
			utils.Error(c.Ctx, http.StatusInternalServerError, "创建业务对象失败")
			return
		}
		logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_CRM_OBJECT", fmt.Sprintf("%s[%d]", domain, object.ID), "operation.create_crm_object", map[string]interface{}{"domain": domain, "object_id": object.ID, "name": object.Name}, "success", "")
		utils.Success(c.Ctx, object)
		return
	}
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if domain == "contacts" && object.ObjectType == "contact" {
			if err := ensureContactCompanyTx(tx, auth, &object, "contact_create"); err != nil {
				return err
			}
		}
		if supportsCustomerChainDomain(domain, object.ObjectType) {
			if err := normalizeCustomerChainObjectTx(tx, auth, &object); err != nil {
				return err
			}
		}
		if err := tx.Create(&object).Error; err != nil {
			return err
		}
		if supportsCustomerChainDomain(domain, object.ObjectType) {
			if err := ensureCustomerChainRelationsTx(tx, auth, object); err != nil {
				return err
			}
		}
		if domain == "contacts" && object.ObjectType == "contact" {
			if err := createContactCompanyRelationTx(tx, auth, object, "contact_create"); err != nil {
				return err
			}
		}
		if domain == "contacts" && object.ObjectType == "interaction" {
			if err := createInteractionTimelineRelationTx(tx, auth, object); err != nil {
				return err
			}
		}
		if domain == "opportunities" && object.ObjectType == "opportunity" {
			return ensureCampaignObjectRelationTx(tx, auth, object, "generated_opportunity", "opportunity_create")
		}
		return nil
	}); err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_CRM_OBJECT", fmt.Sprintf("%s[0]", domain), "operation.create_crm_object", map[string]interface{}{"domain": domain, "name": req.Name}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "创建业务对象失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_CRM_OBJECT", fmt.Sprintf("%s[%d]", domain, object.ID), "operation.create_crm_object", map[string]interface{}{"domain": domain, "object_id": object.ID, "name": object.Name}, "success", "")
	utils.Success(c.Ctx, object)
}

func ensureContactCompanyTx(tx *gorm.DB, auth authContext, contact *models.CRMObject, source string) error {
	if contact == nil || contact.Domain != "contacts" || contact.ObjectType != "contact" {
		return nil
	}
	contactPayload := payloadMap(contact.PayloadJSON)
	companyName := strings.TrimSpace(firstPayloadText(contactPayload, "company", "company_name"))
	companyID := contact.CompanyID
	if companyID == 0 {
		companyID = uintFromValue(contactPayload["company_id"])
	}
	if companyID == 0 && companyName == "" {
		return nil
	}

	var company models.CRMObject
	if companyID > 0 {
		_ = tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "contacts", "company", companyID).First(&company).Error
	}
	if company.ID == 0 && companyName != "" {
		company = findCRMObjectByPayloadTx(tx, auth, "contacts", "company", companyName, "", "")
	}
	if company.ID == 0 && companyName != "" {
		companyPayload := map[string]interface{}{
			"company":             companyName,
			"company_name":        companyName,
			"source":              "contact",
			"source_contact_name": contact.Name,
			"auto_created":        true,
		}
		applyContactSearchTokens(&crmObjectRequest{ObjectType: "company", Name: companyName}, companyPayload)
		company = models.CRMObject{
			TenantModel:  models.TenantModel{TenantID: auth.TenantID},
			Domain:       "contacts",
			ObjectType:   "company",
			Name:         companyName,
			Status:       "active",
			OwnerID:      contact.OwnerID,
			DepartmentID: contact.DepartmentID,
			Currency:     contact.Currency,
			Tags:         "contact_auto_created",
			PayloadJSON:  toJSONString(companyPayload),
			CreatedBy:    auth.UserID,
			UpdatedBy:    auth.UserID,
		}
		if err := tx.Create(&company).Error; err != nil {
			return err
		}
	}
	if company.ID == 0 {
		return nil
	}

	contact.CompanyID = company.ID
	contactPayload["company"] = company.Name
	contactPayload["company_name"] = company.Name
	contactPayload["company_id"] = company.ID
	if strings.TrimSpace(fmt.Sprint(contactPayload["source"])) == "" {
		contactPayload["source"] = "manual"
	}
	contact.PayloadJSON = toJSONString(contactPayload)
	return nil
}

func createContactCompanyRelationTx(tx *gorm.DB, auth authContext, contact models.CRMObject, source string) error {
	if contact.ID == 0 || contact.CompanyID == 0 || contact.Domain != "contacts" || contact.ObjectType != "contact" {
		return nil
	}
	var company models.CRMObject
	if err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "contacts", "company", contact.CompanyID).First(&company).Error; err != nil {
		return nil
	}
	relation := newCRMRelation(auth, "contacts", company.ID, "contacts", contact.ID, "company_contact")
	relation.PayloadJSON = toJSONString(map[string]interface{}{"primary": true, "source": source})
	if err := tx.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?", auth.TenantID, "contacts", company.ID, "contacts", contact.ID, "company_contact").FirstOrCreate(&relation).Error; err != nil {
		return err
	}
	return nil
}

func ensureLeadCompanyContactTx(tx *gorm.DB, auth authContext, lead *models.CRMObject) error {
	if lead == nil {
		return nil
	}
	leadPayload := payloadMap(lead.PayloadJSON)
	companyName := strings.TrimSpace(firstPayloadText(leadPayload, "company_name", "company"))
	contactName := strings.TrimSpace(firstPayloadText(leadPayload, "contact_name", "contact", "name"))
	companyID := lead.CompanyID
	if companyID == 0 {
		companyID = uintFromValue(leadPayload["company_id"])
	}
	contactID := lead.ContactID
	if contactID == 0 {
		contactID = uintFromValue(leadPayload["contact_id"])
	}

	var company models.CRMObject
	if companyID > 0 {
		if err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "contacts", "company", companyID).First(&company).Error; err != nil {
			return fmt.Errorf("线索关联的公司不存在")
		}
	}
	if company.ID == 0 && companyName != "" {
		company = models.CRMObject{
			TenantModel:  models.TenantModel{TenantID: auth.TenantID},
			Domain:       "contacts",
			ObjectType:   "company",
			Name:         companyName,
			Status:       "active",
			OwnerID:      lead.OwnerID,
			DepartmentID: lead.DepartmentID,
			Currency:     lead.Currency,
			Tags:         "lead_auto_created",
			PayloadJSON: toJSONString(map[string]interface{}{
				"company_name":     companyName,
				"source":           "lead",
				"source_lead_name": lead.Name,
				"source_channel":   firstPayloadText(leadPayload, "source_channel", "utm_source"),
			}),
			CreatedBy: auth.UserID,
			UpdatedBy: auth.UserID,
		}
		if err := tx.Create(&company).Error; err != nil {
			return err
		}
	}

	var contact models.CRMObject
	if contactID > 0 {
		if err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "contacts", "contact", contactID).First(&contact).Error; err != nil {
			return fmt.Errorf("线索关联的联系人不存在")
		}
	}
	if contact.ID == 0 && contactName != "" {
		contactPayload := mergePayload(leadPayload, map[string]interface{}{
			"contact_name":     contactName,
			"company":          company.Name,
			"company_id":       company.ID,
			"source":           "lead",
			"source_lead_name": lead.Name,
		})
		contact = models.CRMObject{
			TenantModel:  models.TenantModel{TenantID: auth.TenantID},
			Domain:       "contacts",
			ObjectType:   "contact",
			Name:         contactName,
			Status:       "active",
			OwnerID:      lead.OwnerID,
			DepartmentID: lead.DepartmentID,
			CompanyID:    company.ID,
			Currency:     lead.Currency,
			Tags:         "lead_auto_created",
			PayloadJSON:  toJSONString(contactPayload),
			CreatedBy:    auth.UserID,
			UpdatedBy:    auth.UserID,
		}
		if err := tx.Create(&contact).Error; err != nil {
			return err
		}
	}
	if contact.ID > 0 && company.ID > 0 {
		contactPayload := payloadMap(contact.PayloadJSON)
		changed := false
		if contact.CompanyID == 0 {
			contact.CompanyID = company.ID
			changed = true
		}
		if strings.TrimSpace(fmt.Sprint(contactPayload["company"])) == "" {
			contactPayload["company"] = company.Name
			changed = true
		}
		if uintFromValue(contactPayload["company_id"]) == 0 {
			contactPayload["company_id"] = company.ID
			changed = true
		}
		if changed {
			contact.PayloadJSON = toJSONString(contactPayload)
			if err := tx.Save(&contact).Error; err != nil {
				return err
			}
		}
		relation := newCRMRelation(auth, "contacts", company.ID, "contacts", contact.ID, "company_contact")
		relation.PayloadJSON = toJSONString(map[string]interface{}{"primary": true, "source": "lead_create"})
		if err := tx.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?", auth.TenantID, "contacts", company.ID, "contacts", contact.ID, "company_contact").FirstOrCreate(&relation).Error; err != nil {
			return err
		}
	}
	if company.ID > 0 {
		lead.CompanyID = company.ID
		leadPayload["company_id"] = company.ID
		leadPayload["company_name"] = company.Name
	}
	if contact.ID > 0 {
		lead.ContactID = contact.ID
		leadPayload["contact_id"] = contact.ID
		leadPayload["contact_name"] = contact.Name
		contactPayload := payloadMap(contact.PayloadJSON)
		if strings.TrimSpace(firstPayloadText(leadPayload, "phone", "mobile")) == "" {
			if contactPhone := firstPayloadText(contactPayload, "phone", "mobile"); contactPhone != "" {
				leadPayload["phone"] = contactPhone
			}
		}
		if strings.TrimSpace(firstPayloadText(leadPayload, "email")) == "" {
			if contactEmail := firstPayloadText(contactPayload, "email"); contactEmail != "" {
				leadPayload["email"] = contactEmail
			}
		}
	}
	lead.PayloadJSON = toJSONString(leadPayload)
	return nil
}

func isLeadRelationValidationError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "线索关联的公司不存在") || strings.Contains(message, "线索关联的联系人不存在")
}

func createLeadCompanyContactRelationsTx(tx *gorm.DB, auth authContext, lead models.CRMObject) error {
	if lead.CompanyID > 0 {
		relation := newCRMRelation(auth, "leads", lead.ID, "contacts", lead.CompanyID, "linked_company")
		if err := tx.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?", auth.TenantID, "leads", lead.ID, "contacts", lead.CompanyID, "linked_company").FirstOrCreate(&relation).Error; err != nil {
			return err
		}
	}
	if lead.ContactID > 0 {
		relation := newCRMRelation(auth, "leads", lead.ID, "contacts", lead.ContactID, "linked_contact")
		if err := tx.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?", auth.TenantID, "leads", lead.ID, "contacts", lead.ContactID, "linked_contact").FirstOrCreate(&relation).Error; err != nil {
			return err
		}
	}
	leadPayload := payloadMap(lead.PayloadJSON)
	campaignID := uintFromValue(leadPayload["campaign_id"])
	if campaignID > 0 {
		var campaign models.CRMObject
		if err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "campaigns", "campaign", campaignID).First(&campaign).Error; err == nil {
			relation := newCRMRelation(auth, "campaigns", campaign.ID, "leads", lead.ID, "generated_lead")
			relation.PayloadJSON = toJSONString(map[string]interface{}{
				"source":        "lead_create",
				"campaign_id":   campaign.ID,
				"campaign_name": campaign.Name,
				"utm_source":    leadPayload["utm_source"],
				"utm_medium":    leadPayload["utm_medium"],
				"utm_campaign":  leadPayload["utm_campaign"],
			})
			if err := tx.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?", auth.TenantID, "campaigns", campaign.ID, "leads", lead.ID, "generated_lead").FirstOrCreate(&relation).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureCampaignObjectRelationTx(tx *gorm.DB, auth authContext, object models.CRMObject, relationType string, source string) error {
	payload := payloadMap(object.PayloadJSON)
	campaignID := uintFromValue(payload["campaign_id"])
	if campaignID == 0 {
		return nil
	}
	var campaign models.CRMObject
	if err := tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "campaigns", "campaign", campaignID).First(&campaign).Error; err != nil {
		return nil
	}
	relation := newCRMRelation(auth, "campaigns", campaign.ID, object.Domain, object.ID, relationType)
	relation.PayloadJSON = toJSONString(map[string]interface{}{
		"source":        source,
		"campaign_id":   campaign.ID,
		"campaign_name": campaign.Name,
		"utm_source":    payload["utm_source"],
		"utm_medium":    payload["utm_medium"],
		"utm_campaign":  payload["utm_campaign"],
	})
	return tx.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?", auth.TenantID, "campaigns", campaign.ID, object.Domain, object.ID, relationType).FirstOrCreate(&relation).Error
}

func createInteractionTimelineRelationTx(tx *gorm.DB, auth authContext, interaction models.CRMObject) error {
	if interaction.Domain != "contacts" || interaction.ObjectType != "interaction" || interaction.ID == 0 {
		return nil
	}
	payload := payloadMap(interaction.PayloadJSON)
	sourceType := strings.TrimSpace(fmt.Sprint(payload["related_type"]))
	sourceID := uintFromValue(payload["related_id"])
	if sourceType == "contact" || sourceType == "company" || sourceType == "interaction" {
		sourceType = "contacts"
	}
	if sourceType == "" {
		sourceType = strings.TrimSpace(fmt.Sprint(payload["source_module"]))
	}
	if sourceID == 0 {
		sourceID = uintFromValue(payload["related_object_id"])
	}
	if sourceType == "" || sourceID == 0 || !validCRMDomain(sourceType) {
		return nil
	}
	var source models.CRMObject
	if err := tx.Where("tenant_id = ? AND domain = ? AND id = ?", auth.TenantID, sourceType, sourceID).First(&source).Error; err != nil {
		return nil
	}
	relation := newCRMRelation(auth, sourceType, sourceID, "contacts", interaction.ID, "timeline")
	relation.PayloadJSON = toJSONString(map[string]interface{}{
		"source":      "interaction_create",
		"object_type": source.ObjectType,
	})
	return tx.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?", auth.TenantID, sourceType, sourceID, "contacts", interaction.ID, "timeline").FirstOrCreate(&relation).Error
}

func (c *CRMController) UpdateObject() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domain := strings.TrimSpace(c.Ctx.Input.Query(":domain"))
	id, valid := pathUint(c.Ctx, ":id")
	if !valid || !validCRMDomain(domain) {
		utils.BadRequest(c.Ctx, "无效的业务对象")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req crmObjectRequest
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var object models.CRMObject
	if err := crmScopedObjectQuery(auth, domain).Where("id = ?", id).First(&object).Error; err != nil {
		utils.NotFound(c.Ctx, "业务对象不存在")
		return
	}
	if crmObjectACLDomain(domain) && !canManageCRMObject(auth, object) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权修改该对象")
		return
	}
	if err := validateCRMFieldWrite(auth, domain, req.Payload, payloadMap(object.PayloadJSON)); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	if domain == "leads" && isTerminalLeadStatus(object.Status) && !auth.IsSuperAdmin() && !auth.IsTenantAdmin() {
		utils.BadRequest(c.Ctx, "已转换或已作废线索不可修改")
		return
	}
	before := object
	next, err := crmObjectFromRequest(auth, domain, req)
	if err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	if err := validateLeadRequiredFields(next); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	if domain == "orders" && object.Status != "canceled" && next.Status == "canceled" && !authCanCancelOrder(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "只有租户管理员或订单管理权限用户才能取消订单")
		return
	}
	if err := validateMfgLockedOrderUpdate(auth, domain, object, next); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	preserveCompletedOrderLockedFields(object, &next)
	if domain == "leads" && isTerminalLeadStatus(object.Status) && next.Status != object.Status {
		utils.BadRequest(c.Ctx, "已转换或已作废线索不可回退状态")
		return
	}
	if err := validateStageGate(auth, domain, next); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	object.ObjectType = next.ObjectType
	object.Name = next.Name
	object.Code = next.Code
	object.Status = next.Status
	object.OwnerID = next.OwnerID
	object.DepartmentID = next.DepartmentID
	object.CompanyID = next.CompanyID
	object.ContactID = next.ContactID
	object.DealID = next.DealID
	object.ProjectID = next.ProjectID
	object.Amount = next.Amount
	object.Currency = next.Currency
	object.Score = next.Score
	object.StartAt = next.StartAt
	object.EndAt = next.EndAt
	object.Tags = next.Tags
	object.PayloadJSON = next.PayloadJSON
	object.UpdatedBy = auth.UserID
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if domain == "contacts" && object.ObjectType == "contact" {
			if err := ensureContactCompanyTx(tx, auth, &object, "contact_update"); err != nil {
				return err
			}
		}
		if domain == "leads" && object.ObjectType == "lead" {
			if err := ensureLeadCompanyContactTx(tx, auth, &object); err != nil {
				return err
			}
		}
		if supportsCustomerChainDomain(domain, object.ObjectType) {
			if err := normalizeCustomerChainObjectTx(tx, auth, &object); err != nil {
				return err
			}
		}
		if err := tx.Save(&object).Error; err != nil {
			return err
		}
		if supportsCustomerChainDomain(domain, object.ObjectType) {
			if err := ensureCustomerChainRelationsTx(tx, auth, object); err != nil {
				return err
			}
		}
		if domain == "leads" && object.ObjectType == "lead" {
			if err := createLeadCompanyContactRelationsTx(tx, auth, object); err != nil {
				return err
			}
		}
		if domain == "contacts" && object.ObjectType == "contact" {
			if err := createContactCompanyRelationTx(tx, auth, object, "contact_update"); err != nil {
				return err
			}
		}
		if domain == "contacts" && object.ObjectType == "interaction" {
			if err := createInteractionTimelineRelationTx(tx, auth, object); err != nil {
				return err
			}
		}
		if domain == "opportunities" && object.ObjectType == "opportunity" {
			if err := ensureCampaignObjectRelationTx(tx, auth, object, "generated_opportunity", "opportunity_update"); err != nil {
				return err
			}
		}
		return autoSyncTicketFromBugTaskTx(tx, auth, before, object)
	}); err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_CRM_OBJECT", fmt.Sprintf("%s[%d]", domain, id), "operation.update_crm_object", map[string]interface{}{"domain": domain, "object_id": id, "before": before, "after": object}, "failed", err.Error())
		if isLeadRelationValidationError(err) {
			utils.BadRequest(c.Ctx, err.Error())
			return
		}
		utils.Error(c.Ctx, http.StatusInternalServerError, "更新业务对象失败")
		return
	}
	if err := cleanupCRMObjectReplacedFiles(auth, before, object); err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "CLEANUP_REPLACED_CRM_OBJECT_FILES", fmt.Sprintf("%s[%d]", domain, id), "operation.cleanup_replaced_crm_object_files", map[string]interface{}{"domain": domain, "object_id": id}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusBadGateway, err.Error())
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_CRM_OBJECT", fmt.Sprintf("%s[%d]", domain, id), "operation.update_crm_object", map[string]interface{}{"domain": domain, "object_id": id, "before": before, "after": object}, "success", "")
	utils.Success(c.Ctx, object)
}

func (c *CRMController) DeleteObject() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domain := strings.TrimSpace(c.Ctx.Input.Query(":domain"))
	id, valid := pathUint(c.Ctx, ":id")
	if !valid || !validCRMDomain(domain) {
		utils.BadRequest(c.Ctx, "无效的业务对象")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var object models.CRMObject
	if err := crmScopedObjectQuery(auth, domain).Where("id = ?", id).First(&object).Error; err != nil {
		utils.NotFound(c.Ctx, "业务对象不存在")
		return
	}
	if err := validateCRMObjectDelete(auth, object); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	if crmObjectACLDomain(domain) && !canManageCRMObject(auth, object) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权删除该对象")
		return
	}
	if err := cleanupCRMObjectFilesForDelete(auth, object); err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "DELETE_CRM_OBJECT_FILES", fmt.Sprintf("%s[%d]", domain, id), "operation.delete_crm_object_files", map[string]interface{}{"domain": domain, "object_id": id}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusBadGateway, err.Error())
		return
	}
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		childTypes := childFileTypesForObject(object)
		if len(childTypes) > 0 {
			var childFiles []models.CRMObject
			if err := tx.Model(&models.CRMObject{}).
				Joins("JOIN crm_relations rel ON rel.tenant_id = crm_objects.tenant_id AND rel.target_type = crm_objects.domain AND rel.target_id = crm_objects.id").
				Where("crm_objects.tenant_id = ? AND crm_objects.domain = ? AND crm_objects.object_type IN ?", auth.TenantID, domain, childTypes).
				Where("rel.source_type = ? AND rel.source_id = ?", domain, id).
				Find(&childFiles).Error; err != nil {
				return err
			}
			for _, child := range childFiles {
				if err := tx.Delete(&child).Error; err != nil {
					return err
				}
			}
		}
		if err := tx.Delete(&object).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND ((source_type = ? AND source_id = ?) OR (target_type = ? AND target_id = ?))", auth.TenantID, domain, id, domain, id).Delete(&models.CRMRelation{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		logControllerOperation(c.Ctx, auth, auth.TenantID, "DELETE_CRM_OBJECT", fmt.Sprintf("%s[%d]", domain, id), "operation.delete_crm_object", map[string]interface{}{"domain": domain, "object_id": id, "before": object}, "failed", err.Error())
		utils.Error(c.Ctx, http.StatusInternalServerError, "删除业务对象失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "DELETE_CRM_OBJECT", fmt.Sprintf("%s[%d]", domain, id), "operation.delete_crm_object", map[string]interface{}{"domain": domain, "object_id": id, "before": object}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"deleted_id": id})
}

func (c *CRMController) GetObjectACL() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domain := strings.TrimSpace(c.Ctx.Input.Query(":domain"))
	id, valid := pathUint(c.Ctx, ":id")
	if !valid || !crmObjectACLDomain(domain) {
		utils.BadRequest(c.Ctx, "无效的业务对象")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var object models.CRMObject
	if err := crmScopedObjectQuery(auth, domain).Where("id = ?", id).First(&object).Error; err != nil {
		utils.NotFound(c.Ctx, "业务对象不存在")
		return
	}
	if !canViewCRMObject(auth, object) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权查看该对象协作权限")
		return
	}
	var entries []models.CRMObjectACLEntry
	if err := utils.DB.Where("tenant_id = ? AND domain = ? AND object_id = ?", auth.TenantID, domain, id).Order("id asc").Find(&entries).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "获取协作权限失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": entries})
}

func (c *CRMController) UpdateObjectACL() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domain := strings.TrimSpace(c.Ctx.Input.Query(":domain"))
	id, valid := pathUint(c.Ctx, ":id")
	if !valid || !crmObjectACLDomain(domain) {
		utils.BadRequest(c.Ctx, "无效的业务对象")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		Entries []struct {
			UserID      uint   `json:"user_id"`
			AccessLevel string `json:"access_level"`
		} `json:"entries"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var object models.CRMObject
	if err := crmScopedObjectQuery(auth, domain).Where("id = ?", id).First(&object).Error; err != nil {
		utils.NotFound(c.Ctx, "业务对象不存在")
		return
	}
	if !canManageCRMObject(auth, object) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权配置该对象协作权限")
		return
	}
	entries := make([]models.CRMObjectACLEntry, 0, len(req.Entries))
	seen := map[uint]bool{}
	for _, item := range req.Entries {
		level := strings.ToLower(strings.TrimSpace(item.AccessLevel))
		if item.UserID == 0 || seen[item.UserID] {
			continue
		}
		if level != "view" && level != "follow" && level != "handle" && level != "manage" {
			utils.BadRequest(c.Ctx, "协作权限只支持 view/follow/handle/manage")
			return
		}
		var user models.User
		if err := utils.DB.Where("id = ? AND tenant_id = ? AND status = ?", item.UserID, auth.TenantID, "active").First(&user).Error; err != nil {
			utils.BadRequest(c.Ctx, "协作用户不存在或不可用")
			return
		}
		seen[item.UserID] = true
		entries = append(entries, models.CRMObjectACLEntry{
			TenantModel: models.TenantModel{TenantID: auth.TenantID},
			Domain:      domain,
			ObjectID:    id,
			UserID:      item.UserID,
			AccessLevel: level,
			CreatedBy:   auth.UserID,
			UpdatedBy:   auth.UserID,
		})
	}
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND domain = ? AND object_id = ?", auth.TenantID, domain, id).Delete(&models.CRMObjectACLEntry{}).Error; err != nil {
			return err
		}
		if len(entries) > 0 {
			if err := tx.Create(&entries).Error; err != nil {
				return err
			}
		}
		return createBusinessTimeline(tx, auth, domain, id, "acl_update", "更新协作权限", map[string]interface{}{"entries": entries})
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存协作权限失败")
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": entries})
}

func (c *CRMController) ImportObjects() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domain := strings.TrimSpace(c.Ctx.Input.Query(":domain"))
	if !validCRMDomain(domain) {
		utils.BadRequest(c.Ctx, "无效的业务域")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var req struct {
		Items []crmObjectRequest `json:"items"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if len(req.Items) == 0 {
		utils.BadRequest(c.Ctx, "导入数据不能为空")
		return
	}
	if len(req.Items) > 500 {
		utils.BadRequest(c.Ctx, "单次最多导入500条")
		return
	}
	items := make([]models.CRMObject, 0, len(req.Items))
	for _, item := range req.Items {
		if err := validateCRMFieldWrite(auth, domain, item.Payload, nil); err != nil {
			utils.BadRequest(c.Ctx, err.Error())
			return
		}
		object, err := crmObjectFromRequest(auth, domain, item)
		if err != nil {
			utils.BadRequest(c.Ctx, err.Error())
			return
		}
		if err := validateLeadRequiredFields(object); err != nil {
			utils.BadRequest(c.Ctx, err.Error())
			return
		}
		items = append(items, object)
	}
	if err := utils.DB.Create(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "批量导入失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "IMPORT_CRM_OBJECTS", domain, "operation.import_crm_objects", map[string]interface{}{"domain": domain, "count": len(items)}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"count": len(items)})
}

func (c *CRMController) ExportObjects() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	domain := strings.TrimSpace(c.Ctx.Input.Query(":domain"))
	if !validCRMDomain(domain) {
		utils.BadRequest(c.Ctx, "无效的业务域")
		return
	}
	if err := ensureCRMModuleAccess(auth, domain, "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	if !auth.IsSuperAdmin() && !auth.IsTenantAdmin() && !authHasPermissionAny(auth, domain+":export", domain+":manage") {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限导出该模块数据")
		return
	}
	exportCtx := newExportSecurityContext(c.Ctx, auth)
	query := crmScopedObjectQuery(auth, domain)
	var items []models.CRMObject
	if err := query.Order("updated_at desc").Limit(1000).Find(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "导出失败")
		return
	}
	items = applyCRMFieldACL(auth, domain, items)
	logControllerOperation(c.Ctx, auth, auth.TenantID, "EXPORT_CRM_OBJECTS", domain, "operation.export_crm_objects", mergeExportParams(exportCtx, map[string]interface{}{"domain": domain, "count": len(items)}), "success", "")
	rows := make([][]interface{}, 0, len(items))
	for _, item := range items {
		rows = append(rows, []interface{}{
			item.ID,
			item.Domain,
			item.ObjectType,
			item.Name,
			item.Code,
			item.Status,
			item.OwnerID,
			item.DepartmentID,
			item.CompanyID,
			item.ContactID,
			item.DealID,
			item.ProjectID,
			item.Amount,
			item.Currency,
			item.Score,
			item.Tags,
			item.CreatedAt,
			item.UpdatedAt,
			item.PayloadJSON,
		})
	}
	var buffer bytes.Buffer
	if err := writeExcelHTML(&buffer, exportCtx.WatermarkTable(), excelTable{
		Title:   "业务对象导出",
		Headers: []string{"ID", "业务域", "对象类型", "名称", "编码", "状态", "负责人ID", "部门ID", "公司ID", "联系人ID", "成交ID", "项目ID", "金额", "币种", "评分", "标签", "创建时间", "更新时间", "扩展字段"},
		Rows:    rows,
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "导出失败")
		return
	}
	filename := fmt.Sprintf("%s-objects-%s-%s.xls", domain, time.Now().Format("20060102150405"), exportCtx.TraceID)
	c.Ctx.Output.Header("Content-Type", "application/vnd.ms-excel; charset=utf-8")
	applyExportTraceHeader(c.Ctx, exportCtx)
	c.Ctx.Output.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	_, _ = c.Ctx.ResponseWriter.Write(buffer.Bytes())
}

func (c *CRMController) Summary() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	type row struct {
		Domain string `json:"domain"`
		Total  int64  `json:"total"`
	}
	var rows []row
	for _, domain := range crmDomains {
		var total int64
		if err := crmScopedObjectQuery(auth, domain.Key).Count(&total).Error; err != nil {
			utils.Error(c.Ctx, http.StatusInternalServerError, "获取总览失败")
			return
		}
		if total > 0 {
			rows = append(rows, row{Domain: domain.Key, Total: total})
		}
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": rows})
}

func (c *CRMController) Cockpit() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	var tasks []models.CRMObject
	_ = crmScopedObjectQuery(auth, "workspace").Where("status IN ?", []string{"open", "new"}).Order("created_at desc").Limit(10).Find(&tasks).Error
	var activities []models.CRMObject
	for _, domain := range crmDomains {
		var domainItems []models.CRMObject
		_ = crmScopedObjectQuery(auth, domain.Key).Order("updated_at desc").Limit(3).Find(&domainItems).Error
		activities = append(activities, domainItems...)
	}
	if len(activities) > 10 {
		activities = activities[:10]
	}
	counts := map[string]int64{}
	for _, domain := range []string{"leads", "opportunities", "orders", "tickets", "projects"} {
		var total int64
		crmScopedObjectQuery(auth, domain).Count(&total)
		counts[domain] = total
	}
	var wonAmount float64
	_ = crmScopedObjectQuery(auth, "deals").Select("COALESCE(SUM(crm_objects.amount), 0)").Where("crm_objects.status = ?", "closed_won").Scan(&wonAmount).Error
	funnel := salesFunnel(auth)
	year := queryYear(c.Ctx.Input.Query("year"))
	trend := salesTrend(auth, year)
	salesTargets := salesTargetSummaryForCockpit(auth, year)
	leaderboard, _ := salesTargets["team_rows"].([]map[string]interface{})
	if len(leaderboard) > 8 {
		leaderboard = leaderboard[:8]
	}
	target := map[string]interface{}{
		"target":       numericValue(salesTargets["company_target"]),
		"actual":       numericValue(salesTargets["current_amount"]),
		"progress":     numericValue(salesTargets["target_progress"]),
		"metric_basis": salesTargets["metric_basis"],
		"yoy":          salesTargets["yoy"],
	}
	utils.Success(c.Ctx, map[string]interface{}{
		"tasks":      tasks,
		"activities": activities,
		"charts": map[string]interface{}{
			"funnel":      funnel,
			"trend":       trend,
			"leaderboard": leaderboard,
			"target":      target,
		},
		"kpi": map[string]interface{}{
			"leads":            counts["leads"],
			"opportunities":    counts["opportunities"],
			"orders":           counts["orders"],
			"tickets":          counts["tickets"],
			"projects":         counts["projects"],
			"won_amount":       wonAmount,
			"company_target":   numericValue(salesTargets["company_target"]),
			"target_amount":    numericValue(salesTargets["company_target"]),
			"target_actual":    numericValue(salesTargets["current_amount"]),
			"target_progress":  numericValue(salesTargets["target_progress"]),
			"metric_basis":     salesTargets["metric_basis"],
			"order_amount":     numericValue(salesTargets["order_amount"]),
			"paid_amount":      numericValue(salesTargets["paid_amount"]),
			"order_count":      numericValue(salesTargets["order_count"]),
			"yoy":              salesTargets["yoy"],
			"order_amount_yoy": salesTargets["order_amount_yoy"],
			"paid_amount_yoy":  salesTargets["paid_amount_yoy"],
			"order_count_yoy":  salesTargets["order_count_yoy"],
		},
		"sales_targets": salesTargets,
	})
}

func (c *CRMController) TestIntegration() {
	auth, ok := requireTenantAdmin(c.Ctx)
	if !ok {
		return
	}
	var req struct {
		Type     string                 `json:"type"`
		Endpoint string                 `json:"endpoint"`
		Config   map[string]interface{} `json:"config"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.Type = strings.TrimSpace(req.Type)
	if req.Type == "" {
		utils.BadRequest(c.Ctx, "集成类型不能为空")
		return
	}
	result, err := runRealIntegration(auth, req.Type, req.Endpoint, req.Config, map[string]interface{}{}, true)
	if err != nil {
		result["ok"] = false
		result["message"] = err.Error()
		logControllerOperation(c.Ctx, auth, auth.TenantID, "TEST_INTEGRATION", req.Type, "operation.test_integration", result, "failed", err.Error())
		utils.Success(c.Ctx, result)
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "TEST_INTEGRATION", req.Type, "operation.test_integration", result, "success", "")
	utils.Success(c.Ctx, result)
}
