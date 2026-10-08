package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"regexp"
	"strings"
	"time"
)

const defaultBusinessCurrencyCode = "USD"

func validCRMDomain(domain string) bool {
	for _, item := range crmDomains {
		if item.Key == domain {
			return true
		}
	}
	return false
}

func crmObjectFromRequest(auth authContext, domain string, req crmObjectRequest) (models.CRMObject, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.ObjectType = strings.TrimSpace(req.ObjectType)
	if req.Name == "" || req.ObjectType == "" {
		return models.CRMObject{}, fmt.Errorf("名称和对象类型不能为空")
	}
	startAt, err := parseOptionalTime(req.StartAt)
	if err != nil {
		return models.CRMObject{}, fmt.Errorf("开始时间格式无效")
	}
	endAt, err := parseOptionalTime(req.EndAt)
	if err != nil {
		return models.CRMObject{}, fmt.Errorf("结束时间格式无效")
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}
	payload := req.Payload
	if payload == nil {
		payload = map[string]interface{}{}
	}
	if domain == "tickets" {
		applyTicketServiceRole(payload)
		applyTicketSLA(auth, payload)
	}
	if domain == "contacts" && (req.ObjectType == "contact" || req.ObjectType == "company") {
		applyContactSearchTokens(&req, payload)
	}
	if domain == "opportunities" && req.ObjectType == "opportunity" {
		normalizeOpportunityRequest(&req, payload)
	}
	if domain == "orders" && req.ObjectType == "order" {
		normalizeOrderRequest(&req, payload)
	}
	if domain == "products" && req.ObjectType == "product" {
		applyProductSKU(&req, payload)
	}
	if crmObjectAccessVisibilityDomain(domain) {
		normalizeCRMObjectAccessPayload(payload)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return models.CRMObject{}, fmt.Errorf("扩展字段不是有效JSON")
	}
	currency := normalizeCurrencyCode(req.Currency)
	if currency == "" {
		currency = tenantBusinessCurrency(auth.TenantID)
	}
	return models.CRMObject{
		TenantModel:  models.TenantModel{TenantID: auth.TenantID},
		Domain:       domain,
		ObjectType:   req.ObjectType,
		Name:         req.Name,
		Code:         strings.TrimSpace(req.Code),
		Status:       status,
		OwnerID:      req.OwnerID,
		DepartmentID: req.DepartmentID,
		CompanyID:    req.CompanyID,
		ContactID:    req.ContactID,
		DealID:       req.DealID,
		ProjectID:    req.ProjectID,
		Amount:       req.Amount,
		Currency:     currency,
		Score:        req.Score,
		StartAt:      startAt,
		EndAt:        endAt,
		Tags:         strings.Join(req.Tags, ","),
		PayloadJSON:  string(payloadJSON),
		CreatedBy:    auth.UserID,
		UpdatedBy:    auth.UserID,
	}, nil
}

func normalizeOpportunityRequest(req *crmObjectRequest, payload map[string]interface{}) {
	now := time.Now()
	stage := normalizeOpportunityStage(req.Status)
	if stage == "" {
		stage = normalizeOpportunityStage(fmt.Sprint(payload["stage"]))
	}
	if stage == "" {
		stage = "initial_contact"
	}
	req.Status = stage
	payload["stage"] = stage
	payload["probability"] = opportunityProbability(stage)
	if strings.TrimSpace(fmt.Sprint(payload["stage_entered_at"])) == "" {
		payload["stage_entered_at"] = now.Format(time.RFC3339)
	}
	if _, ok := payload["line_items"]; !ok {
		payload["line_items"] = []interface{}{}
	}
	opportunity := models.CRMObject{Status: stage, PayloadJSON: toJSONString(payload)}
	opportunity.UpdatedAt = now
	payload["follow_up_status"] = opportunityFollowUpStatus(opportunity, now, defaultOpportunityFollowUpRules())
}

func applyProductSKU(req *crmObjectRequest, payload map[string]interface{}) {
	sku := strings.TrimSpace(fmt.Sprint(payload["sku"]))
	if sku == "" {
		sku = strings.TrimSpace(req.Code)
	}
	if sku == "" {
		category := strings.TrimSpace(fmt.Sprint(payload["category"]))
		prefix := "SKU"
		if category != "" {
			prefix = strings.ToUpper(strings.ReplaceAll(category, " ", "-"))
			if len(prefix) > 8 {
				prefix = prefix[:8]
			}
		}
		sku = fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()%1000000)
	}
	payload["sku"] = sku
	if strings.TrimSpace(req.Code) == "" {
		req.Code = sku
	}
	if _, ok := payload["stock"]; !ok {
		payload["stock"] = 0
	}
	if _, ok := payload["stock_threshold"]; !ok {
		payload["stock_threshold"] = 0
	}
}

func applyContactSearchTokens(req *crmObjectRequest, payload map[string]interface{}) {
	name := strings.TrimSpace(req.Name)
	if name != "" {
		pinyin, initials := textPinyinTokens(name)
		if pinyin != "" {
			payload["name_pinyin"] = pinyin
		}
		if initials != "" {
			payload["name_initials"] = initials
		}
	}
	company := strings.TrimSpace(fmt.Sprint(payload["company"]))
	if company == "" {
		company = strings.TrimSpace(fmt.Sprint(payload["company_name"]))
	}
	if req.ObjectType == "company" && company == "" {
		company = name
	}
	if company != "" {
		pinyin, initials := textPinyinTokens(company)
		if pinyin != "" {
			payload["company_pinyin"] = pinyin
		}
		if initials != "" {
			payload["company_initials"] = initials
		}
	}
}

var nonAlphaNumeric = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeSearchKeyword(value string) string {
	return nonAlphaNumeric.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "")
}

func textPinyinTokens(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	var pinyin strings.Builder
	var initials strings.Builder
	lastWasLatinOrDigit := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			pinyin.WriteRune(r)
			if !lastWasLatinOrDigit {
				initials.WriteRune(r)
			}
			lastWasLatinOrDigit = true
		case r >= 'A' && r <= 'Z':
			low := r + ('a' - 'A')
			pinyin.WriteRune(low)
			if !lastWasLatinOrDigit {
				initials.WriteRune(low)
			}
			lastWasLatinOrDigit = true
		case r >= '0' && r <= '9':
			pinyin.WriteRune(r)
			if !lastWasLatinOrDigit {
				initials.WriteRune(r)
			}
			lastWasLatinOrDigit = true
		default:
			lastWasLatinOrDigit = false
			token := runePinyinToken(r)
			if token != "" {
				pinyin.WriteString(token)
				initials.WriteByte(token[0])
			}
		}
	}
	return pinyin.String(), initials.String()
}

func runePinyinToken(r rune) string {
	if token, ok := commonChinesePinyinToken[r]; ok {
		return token
	}
	return ""
}

var commonChinesePinyinToken = map[rune]string{
	'王': "wang", '李': "li", '张': "zhang", '刘': "liu", '陈': "chen", '杨': "yang", '赵': "zhao", '黄': "huang",
	'周': "zhou", '吴': "wu", '徐': "xu", '孙': "sun", '胡': "hu", '朱': "zhu", '高': "gao", '林': "lin",
	'何': "he", '郭': "guo", '马': "ma", '罗': "luo", '梁': "liang", '宋': "song", '郑': "zheng", '谢': "xie",
	'韩': "han", '唐': "tang", '冯': "feng", '于': "yu", '董': "dong", '萧': "xiao", '程': "cheng", '曹': "cao",
	'袁': "yuan", '邓': "deng", '许': "xu", '傅': "fu", '沈': "shen", '曾': "zeng", '彭': "peng", '吕': "lv",
	'苏': "su", '卢': "lu", '蒋': "jiang", '蔡': "cai", '贾': "jia", '丁': "ding", '魏': "wei", '薛': "xue",
	'叶': "ye", '阎': "yan", '余': "yu", '潘': "pan", '杜': "du", '戴': "dai", '夏': "xia", '钟': "zhong",
	'汪': "wang", '田': "tian", '任': "ren", '姜': "jiang", '范': "fan", '方': "fang", '石': "shi", '姚': "yao",
	'谭': "tan", '廖': "liao", '邹': "zou", '熊': "xiong", '金': "jin", '陆': "lu", '郝': "hao", '孔': "kong",
	'白': "bai", '崔': "cui", '康': "kang", '毛': "mao", '邱': "qiu", '秦': "qin", '江': "jiang", '史': "shi",
	'顾': "gu", '侯': "hou", '邵': "shao", '孟': "meng", '龙': "long", '万': "wan", '段': "duan", '雷': "lei",
	'钱': "qian", '汤': "tang", '尹': "yin", '黎': "li", '易': "yi", '常': "chang", '武': "wu", '乔': "qiao",
	'贺': "he", '赖': "lai", '龚': "gong", '文': "wen", '邢': "xing", '洪': "hong", '庄': "zhuang",
	'科': "ke", '技': "ji", '有': "you", '限': "xian", '公': "gong", '司': "si", '集': "ji", '团': "tuan",
	'智': "zhi", '能': "neng", '网': "wang", '络': "luo", '软': "ruan", '件': "jian", '信': "xin", '息': "xi",
	'数': "shu", '据': "ju", '电': "dian", '子': "zi", '商': "shang", '贸': "mao", '服': "fu", '务': "wu",
	'咨': "zi", '询': "xun", '创': "chuang", '新': "xin", '工': "gong", '业': "ye", '实': "shi", '控': "kong",
	'股': "gu", '投': "tou", '资': "zi", '北': "bei", '京': "jing", '上': "shang", '海': "hai", '深': "shen",
	'圳': "zhen", '广': "guang", '州': "zhou", '杭': "hang", '成': "cheng", '都': "du", '南': "nan", '宁': "ning",
	'福': "fu", '建': "jian", '厦': "sha", '门': "men", '天': "tian", '津': "jin", '重': "chong", '庆': "qing",
}

func crmRelationFromRequest(auth authContext, req crmRelationRequest) (models.CRMRelation, error) {
	req.SourceType = strings.TrimSpace(req.SourceType)
	req.TargetType = strings.TrimSpace(req.TargetType)
	req.RelationType = strings.TrimSpace(req.RelationType)
	if req.SourceType == "" || req.TargetType == "" || req.SourceID == 0 || req.TargetID == 0 || req.RelationType == "" {
		return models.CRMRelation{}, fmt.Errorf("来源、目标和关系类型不能为空")
	}
	if !validCRMDomain(req.SourceType) || !validCRMDomain(req.TargetType) {
		return models.CRMRelation{}, fmt.Errorf("来源或目标业务域无效")
	}
	payloadJSON, err := json.Marshal(req.Payload)
	if err != nil {
		return models.CRMRelation{}, fmt.Errorf("关系扩展字段不是有效JSON")
	}
	return models.CRMRelation{
		TenantModel:  models.TenantModel{TenantID: auth.TenantID},
		SourceType:   req.SourceType,
		SourceID:     req.SourceID,
		TargetType:   req.TargetType,
		TargetID:     req.TargetID,
		RelationType: req.RelationType,
		PayloadJSON:  string(payloadJSON),
	}, nil
}

func cloneCRMObject(auth authContext, domain, objectType, name string, source models.CRMObject) models.CRMObject {
	return models.CRMObject{
		TenantModel:  models.TenantModel{TenantID: auth.TenantID},
		Domain:       domain,
		ObjectType:   objectType,
		Name:         name,
		Status:       "active",
		OwnerID:      source.OwnerID,
		DepartmentID: source.DepartmentID,
		CompanyID:    source.CompanyID,
		ContactID:    source.ContactID,
		DealID:       source.DealID,
		ProjectID:    source.ProjectID,
		Amount:       source.Amount,
		Currency:     source.Currency,
		Score:        source.Score,
		StartAt:      source.StartAt,
		EndAt:        source.EndAt,
		Tags:         source.Tags,
		PayloadJSON:  source.PayloadJSON,
		CreatedBy:    auth.UserID,
		UpdatedBy:    auth.UserID,
	}
}

func newCRMRelation(auth authContext, sourceType string, sourceID uint, targetType string, targetID uint, relationType string) models.CRMRelation {
	return models.CRMRelation{
		TenantModel:  models.TenantModel{TenantID: auth.TenantID},
		SourceType:   sourceType,
		SourceID:     sourceID,
		TargetType:   targetType,
		TargetID:     targetID,
		RelationType: relationType,
		PayloadJSON:  "{}",
	}
}

func formCompleteness(payload map[string]interface{}) int {
	fields := []string{"email", "phone", "company", "title", "utm_source"}
	completed := 0
	for _, field := range fields {
		if strings.TrimSpace(fmt.Sprint(payload[field])) != "" {
			completed++
		}
	}
	return completed * 20
}

func templateNodes(template map[string]interface{}) []map[string]interface{} {
	raw, ok := template["nodes"].([]map[string]interface{})
	if ok {
		return raw
	}
	nodes := []map[string]interface{}{}
	for _, item := range interfaceList(template["nodes"]) {
		if node, ok := item.(map[string]interface{}); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func normalizeOpportunityStage(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	aliases := map[string]string{
		"pipeline":       "initial_contact",
		"contact":        "initial_contact",
		"qualified":      "need_confirmed",
		"needs":          "need_confirmed",
		"quote":          "proposal",
		"proposal_quote": "proposal",
		"review":         "negotiation",
		"closed_won":     "won",
		"closed_lost":    "lost",
		"hold":           "on_hold",
	}
	if value, ok := aliases[status]; ok {
		status = value
	}
	for _, allowed := range []string{"initial_contact", "need_confirmed", "proposal", "negotiation", "won", "lost", "on_hold"} {
		if status == allowed {
			return status
		}
	}
	return ""
}

func opportunityProbability(stage string) int {
	switch normalizeOpportunityStage(stage) {
	case "initial_contact":
		return 10
	case "need_confirmed":
		return 30
	case "proposal":
		return 50
	case "negotiation":
		return 70
	case "won":
		return 100
	default:
		return 0
	}
}

func isMQLRejectedReason(reason string) bool {
	text := strings.ToLower(strings.TrimSpace(reason))
	return strings.Contains(text, "mql") ||
		strings.Contains(text, "sdr") ||
		strings.Contains(text, "拒绝")
}

func scoreToTemperature(score int) string {
	switch {
	case score >= 75:
		return "hot"
	case score >= 45:
		return "warm"
	default:
		return "cold"
	}
}

func firstPayloadText(payload map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(fmt.Sprint(payload[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func defaultText(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func mergePayload(base map[string]interface{}, patch map[string]interface{}) map[string]interface{} {
	next := map[string]interface{}{}
	for key, value := range base {
		next[key] = value
	}
	for key, value := range patch {
		if value != nil {
			next[key] = value
		}
	}
	return next
}

func inheritAttributionPayload(source map[string]interface{}) map[string]interface{} {
	payload := map[string]interface{}{}
	for _, key := range []string{"source_channel", "campaign_id", "utm_source", "utm_medium", "utm_campaign", "attribution_trace"} {
		if value, ok := source[key]; ok && strings.TrimSpace(fmt.Sprint(value)) != "" && strings.TrimSpace(fmt.Sprint(value)) != "<nil>" {
			payload[key] = value
		}
	}
	return payload
}

func hasRelatedTimeline(auth authContext, sourceType string, sourceID uint) bool {
	var count int64
	_ = utils.DB.Model(&models.CRMRelation{}).
		Where("tenant_id = ? AND source_type = ? AND source_id = ? AND relation_type IN ?", auth.TenantID, sourceType, sourceID, []string{"timeline", "stage_timeline", "has_note"}).
		Count(&count).Error
	return count > 0
}

func createBusinessTimeline(tx *gorm.DB, auth authContext, sourceType string, sourceID uint, eventType string, summary string, payload map[string]interface{}) error {
	if payload == nil {
		payload = map[string]interface{}{}
	}
	payload["event_type"] = eventType
	payload["summary"] = summary
	payload["source_module"] = sourceType
	payload["related_object_id"] = sourceID
	operatedAt := time.Now().Format(time.RFC3339)
	payload["happened_at"] = operatedAt
	payload["operated_at"] = operatedAt
	payload["operator_id"] = auth.UserID
	payload["operator_name"] = defaultText(auth.Username, fmt.Sprintf("用户#%d", auth.UserID))
	item := models.CRMObject{
		TenantModel: models.TenantModel{TenantID: auth.TenantID},
		Domain:      "contacts",
		ObjectType:  "interaction",
		Name:        summary,
		Status:      "active",
		OwnerID:     auth.UserID,
		PayloadJSON: toJSONString(payload),
		CreatedBy:   auth.UserID,
		UpdatedBy:   auth.UserID,
	}
	if err := tx.Create(&item).Error; err != nil {
		return err
	}
	relation := newCRMRelation(auth, sourceType, sourceID, "contacts", item.ID, "timeline")
	return tx.Create(&relation).Error
}

func normalizeMatchText(value string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", ".", "", ",", "", "，", "", "。", "", "(", "", ")", "", "（", "", "）", "")
	return strings.ToLower(replacer.Replace(strings.TrimSpace(value)))
}

func mergeTagText(existing string, tags []string) string {
	seen := map[string]bool{}
	items := []string{}
	for _, tag := range append(strings.Split(existing, ","), tags...) {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		items = append(items, tag)
	}
	return strings.Join(items, ",")
}

func findCRMObjectsByPayload(auth authContext, domain string, objectType string, name string, phone string, email string, limit int) []models.CRMObject {
	query := crmScopedObjectQuery(auth, domain).Where("object_type = ?", objectType)
	conditions := []string{}
	args := []interface{}{}
	name = strings.TrimSpace(name)
	if name != "" {
		conditions = append(conditions, "name LIKE ?")
		args = append(args, "%"+name+"%")
	}
	if clause, clauseArgs := payloadJSONKeyLikeClause(phone, "phone", "mobile"); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if clause, clauseArgs := payloadJSONKeyLikeClause(email, "email"); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if len(conditions) == 0 {
		return []models.CRMObject{}
	}
	if limit <= 0 {
		limit = 10
	}
	var items []models.CRMObject
	_ = query.Where(strings.Join(conditions, " OR "), args...).Limit(limit).Find(&items).Error
	return items
}

func findCRMObjectByPayloadTx(tx *gorm.DB, auth authContext, domain string, objectType string, name string, phone string, email string) models.CRMObject {
	items := []models.CRMObject{}
	query := tx.Where("tenant_id = ? AND domain = ? AND object_type = ?", auth.TenantID, domain, objectType)
	conditions := []string{}
	args := []interface{}{}
	name = strings.TrimSpace(name)
	if name != "" {
		conditions = append(conditions, "name LIKE ?")
		args = append(args, "%"+name+"%")
	}
	if clause, clauseArgs := payloadJSONKeyLikeClause(phone, "phone", "mobile"); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if clause, clauseArgs := payloadJSONKeyLikeClause(email, "email"); clause != "" {
		conditions = append(conditions, clause)
		args = append(args, clauseArgs...)
	}
	if len(conditions) == 0 {
		return models.CRMObject{}
	}
	_ = query.Where(strings.Join(conditions, " OR "), args...).Limit(1).Find(&items).Error
	if len(items) == 0 {
		return models.CRMObject{}
	}
	return items[0]
}

func validateCRMObjectDelete(auth authContext, object models.CRMObject) error {
	if object.Domain == "contacts" && (object.ObjectType == "company" || object.ObjectType == "contact") && !auth.IsSuperAdmin() && !auth.IsTenantAdmin() {
		return fmt.Errorf("客户和联系人请先停用，物理删除仅管理员可操作")
	}
	if object.Domain == "enterprise" && object.ObjectType == "department" {
		var childCount int64
		if err := utils.DB.Model(&models.CRMObject{}).
			Where("tenant_id = ? AND domain = ? AND object_type = ? AND department_id = ?", auth.TenantID, "enterprise", "department", object.ID).
			Count(&childCount).Error; err != nil {
			return err
		}
		if childCount > 0 {
			return fmt.Errorf("该部门仍有下级部门，不能删除")
		}
		var memberCount int64
		if err := utils.DB.Model(&models.CRMObject{}).
			Where("tenant_id = ? AND department_id = ? AND NOT (domain = ? AND object_type = ? AND id = ?)", auth.TenantID, object.ID, object.Domain, object.ObjectType, object.ID).
			Count(&memberCount).Error; err != nil {
			return err
		}
		if memberCount > 0 {
			return fmt.Errorf("该部门仍有关联人员或业务数据，不能删除")
		}
	}
	if object.Domain == "leads" && object.Status != "invalid" {
		return fmt.Errorf("只能删除已作废线索，请先将线索作废")
	}
	if object.Domain == "orders" && object.ObjectType == "order" {
		if object.Status == "completed" {
			return fmt.Errorf("已完成订单不能删除")
		}
		if object.Status != "draft" && !auth.IsSuperAdmin() && !auth.IsTenantAdmin() {
			return fmt.Errorf("非草稿订单只能由管理员删除，普通用户请取消或归档订单")
		}
	}
	return nil
}

func validateMfgLockedOrderUpdate(auth authContext, domain string, before models.CRMObject, after models.CRMObject) error {
	if domain != "orders" || before.ObjectType != "order" {
		return nil
	}
	if !services.IsIndustryPackageEnabled(auth.TenantID, services.IndustryPackageManufacturing) {
		return nil
	}
	beforePayload := payloadMap(before.PayloadJSON)
	locked, _ := beforePayload["mfg_bom_locked"].(bool)
	if !locked {
		return nil
	}
	afterPayload := payloadMap(after.PayloadJSON)
	lockedKeys := []string{"mfg_bom_snapshot_id", "mfg_quote_revision_id", "line_items", "products", "quantity", "manufacturing_config"}
	for _, key := range lockedKeys {
		if fmt.Sprint(beforePayload[key]) != fmt.Sprint(afterPayload[key]) {
			return fmt.Errorf(services.MfgErrorECNRequired)
		}
	}
	if before.Amount != after.Amount || before.CompanyID != after.CompanyID || before.DealID != after.DealID {
		return fmt.Errorf(services.MfgErrorECNRequired)
	}
	return nil
}

func validateStageGate(auth authContext, domain string, object models.CRMObject) error {
	if domain != "opportunities" {
		return nil
	}
	if object.ObjectType == "opportunity" {
		if err := validateOpportunityStageGate(auth, object); err != nil {
			return err
		}
	}
	setting := crmSettingValue(auth.TenantID, "sales_process", domain)
	stageRules, _ := setting["stage_required_fields"].(map[string]interface{})
	if len(stageRules) == 0 {
		stageRules, _ = setting["required_fields"].(map[string]interface{})
	}
	required := fieldSet(stageRules[object.Status])
	if len(required) == 0 {
		return nil
	}
	payload := payloadMap(object.PayloadJSON)
	for field := range required {
		value, ok := payload[field]
		if !ok || strings.TrimSpace(fmt.Sprint(value)) == "" {
			return fmt.Errorf("进入阶段 %s 前必须填写字段 %s", object.Status, field)
		}
	}
	return nil
}

func validateOpportunityStageGate(auth authContext, opportunity models.CRMObject) error {
	stage := normalizeOpportunityStage(opportunity.Status)
	if stage == "" {
		return fmt.Errorf("商机阶段无效")
	}
	payload := payloadMap(opportunity.PayloadJSON)
	switch stage {
	case "need_confirmed":
		if !hasRelatedTimeline(auth, "opportunities", opportunity.ID) && strings.TrimSpace(fmt.Sprint(payload["last_follow_up"])) == "" {
			return fmt.Errorf("进入需求确认前，请先添加一条跟进记录")
		}
	case "proposal":
		if len(interfaceList(payload["line_items"])) == 0 {
			return fmt.Errorf("进入方案/报价前，请先添加产品或服务明细")
		}
	case "negotiation":
		quoteStatus := strings.TrimSpace(fmt.Sprint(payload["quote_status"]))
		if quoteStatus != "sent" && quoteStatus != "accepted" {
			return fmt.Errorf("进入谈判/评审前，请先发送报价单")
		}
	case "lost":
		if strings.TrimSpace(fmt.Sprint(payload["loss_reason"])) == "" {
			return fmt.Errorf("输单必须填写输单原因")
		}
		if strings.TrimSpace(fmt.Sprint(payload["competitor"])) == "" {
			return fmt.Errorf("输单必须填写竞争对手")
		}
	}
	return nil
}

func crmScopedObjectQuery(auth authContext, domain string) *gorm.DB {
	query := utils.DB.Model(&models.CRMObject{}).Where("crm_objects.tenant_id = ? AND crm_objects.domain = ?", auth.TenantID, domain)
	return applyCRMDataScope(query, auth, domain)
}

func applyCRMDataScope(query *gorm.DB, auth authContext, domain string) *gorm.DB {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() || isCRMDataPiercingUser(auth, domain) || isCRMPrivateVisibilitySupervisor(auth, domain) || isCRMModuleManager(auth, domain) {
		return query
	}
	setting := crmSettingValue(auth.TenantID, "data_scope", domain)
	mode, _ := setting["mode"].(string)
	switch mode {
	case "owner":
		aclClause, aclArgs := crmObjectACLExistsClause(auth)
		if domain == "leads" {
			args := []interface{}{"%\"visibility\":\"private\"%", "%\"private\":true%", auth.UserID, auth.UserID}
			args = append(args, aclArgs...)
			return query.Where("((crm_objects.owner_id = 0 AND crm_objects.payload_json NOT LIKE ? AND crm_objects.payload_json NOT LIKE ?) OR crm_objects.owner_id = ? OR crm_objects.created_by = ? OR "+aclClause+")", args...)
		}
		if crmObjectACLDomain(domain) {
			args := append([]interface{}{auth.UserID, auth.UserID}, aclArgs...)
			return query.Where("(crm_objects.owner_id = ? OR crm_objects.created_by = ? OR "+aclClause+")", args...)
		}
		return query.Where("crm_objects.owner_id = ? OR crm_objects.created_by = ?", auth.UserID, auth.UserID)
	case "department":
		departmentIDs := numberList(setting["department_ids"])
		if len(departmentIDs) == 0 {
			departmentIDs = numberList(setting["departments"])
		}
		aclClause, aclArgs := crmObjectACLExistsClause(auth)
		if len(departmentIDs) == 0 {
			if crmObjectACLDomain(domain) {
				args := []interface{}{auth.RoleID, "%\"visibility\":\"private\"%", "%\"private\":true%", auth.UserID, auth.UserID}
				args = append(args, aclArgs...)
				return query.Where("((crm_objects.department_id = ? AND crm_objects.payload_json NOT LIKE ? AND crm_objects.payload_json NOT LIKE ?) OR crm_objects.owner_id = ? OR crm_objects.created_by = ? OR "+aclClause+")", args...)
			}
			return query.Where("crm_objects.department_id = ?", auth.RoleID)
		}
		if crmObjectACLDomain(domain) {
			args := []interface{}{departmentIDs, "%\"visibility\":\"private\"%", "%\"private\":true%", auth.UserID, auth.UserID}
			args = append(args, aclArgs...)
			return query.Where("((crm_objects.department_id IN ? AND crm_objects.payload_json NOT LIKE ? AND crm_objects.payload_json NOT LIKE ?) OR crm_objects.owner_id = ? OR crm_objects.created_by = ? OR "+aclClause+")", args...)
		}
		return query.Where("crm_objects.department_id IN ?", departmentIDs)
	case "team", "region", "condition":
		if crmObjectACLDomain(domain) {
			return applyCRMConditionScopeWithACL(query, auth, domain, setting)
		}
		return applyCRMConditionScope(query, setting)
	default:
		return applyCRMObjectACLVisibility(query, auth, domain)
	}
}

func crmObjectACLDomain(domain string) bool {
	switch domain {
	case "campaigns", "leads", "projects", "tickets", "opportunities", "deals", "orders":
		return true
	default:
		return false
	}
}

func crmObjectAccessVisibilityDomain(domain string) bool {
	switch domain {
	case "campaigns", "leads", "projects", "opportunities", "deals", "orders":
		return true
	default:
		return false
	}
}

func normalizeCRMObjectAccessPayload(payload map[string]interface{}) {
	if payload == nil {
		return
	}
	visibility := strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["visibility"])))
	if visibility == "" {
		visibility = strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["access_visibility"])))
	}
	privateValue, hasPrivate := payload["private"]
	privateText := strings.ToLower(strings.TrimSpace(fmt.Sprint(privateValue)))
	if visibility != "private" && hasPrivate && (privateValue == true || privateText == "true") {
		visibility = "private"
	}
	if visibility != "private" {
		visibility = "public"
	}
	payload["visibility"] = visibility
	payload["private"] = visibility == "private"
}

func crmObjectACLExistsClause(auth authContext) (string, []interface{}) {
	return "EXISTS (SELECT 1 FROM crm_object_acl_entries acl WHERE acl.deleted_at IS NULL AND acl.tenant_id = crm_objects.tenant_id AND acl.domain = crm_objects.domain AND acl.object_id = crm_objects.id AND acl.user_id = ?)", []interface{}{auth.UserID}
}

func applyCRMObjectACLVisibility(query *gorm.DB, auth authContext, domain string) *gorm.DB {
	if !crmObjectACLDomain(domain) || auth.IsSuperAdmin() || auth.IsTenantAdmin() || isCRMPrivateVisibilitySupervisor(auth, domain) {
		return query
	}
	aclClause, aclArgs := crmObjectACLExistsClause(auth)
	args := []interface{}{
		"%\"visibility\":\"private\"%",
		"%\"private\":true%",
		auth.UserID,
		auth.UserID,
	}
	args = append(args, aclArgs...)
	return query.Where("((crm_objects.payload_json NOT LIKE ? AND crm_objects.payload_json NOT LIKE ?) OR crm_objects.owner_id = ? OR crm_objects.created_by = ? OR "+aclClause+")", args...)
}

func crmObjectPayloadVisibility(object models.CRMObject) string {
	payload := payloadMap(object.PayloadJSON)
	visibility := strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["visibility"])))
	if visibility == "" {
		visibility = strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["access_visibility"])))
	}
	if visibility == "" && fmt.Sprint(payload["private"]) == "true" {
		visibility = "private"
	}
	if visibility == "" {
		visibility = "public"
	}
	return visibility
}

func crmObjectHasACL(auth authContext, object models.CRMObject, levels ...string) bool {
	if auth.TenantID == 0 || auth.UserID == 0 || object.ID == 0 {
		return false
	}
	query := utils.DB.Model(&models.CRMObjectACLEntry{}).
		Where("tenant_id = ? AND domain = ? AND object_id = ? AND user_id = ?", auth.TenantID, object.Domain, object.ID, auth.UserID)
	if len(levels) > 0 {
		query = query.Where("access_level IN ?", levels)
	}
	var count int64
	_ = query.Count(&count).Error
	return count > 0
}

func canViewCRMObject(auth authContext, object models.CRMObject) bool {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() || isCRMPrivateVisibilitySupervisor(auth, object.Domain) || isCRMModuleManager(auth, object.Domain) {
		return true
	}
	if object.OwnerID == auth.UserID || object.CreatedBy == auth.UserID {
		return true
	}
	if crmObjectPayloadVisibility(object) != "private" {
		return true
	}
	return crmObjectHasACL(auth, object, "view", "follow", "handle", "manage")
}

func canFollowCRMObject(auth authContext, object models.CRMObject) bool {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() || isCRMModuleManager(auth, object.Domain) {
		return true
	}
	if object.OwnerID == auth.UserID || object.CreatedBy == auth.UserID {
		return true
	}
	return crmObjectHasACL(auth, object, "follow", "handle", "manage")
}

func canManageCRMObject(auth authContext, object models.CRMObject) bool {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() || isCRMModuleManager(auth, object.Domain) {
		return true
	}
	if object.OwnerID == auth.UserID || object.CreatedBy == auth.UserID {
		return true
	}
	return crmObjectHasACL(auth, object, "handle", "manage")
}

func ensureCRMObjectEditAccess(auth authContext, object models.CRMObject, label string) error {
	if err := ensureCRMModuleAccess(auth, object.Domain, "edit"); err != nil {
		return err
	}
	if !canManageCRMObject(auth, object) {
		if strings.TrimSpace(label) == "" {
			label = "对象"
		}
		return fmt.Errorf("无权处理该%s", label)
	}
	return nil
}

func canDeleteCRMObjectAttachment(auth authContext, parent models.CRMObject, fileObject models.CRMObject) bool {
	if canManageCRMObject(auth, parent) {
		return true
	}
	return fileObject.CreatedBy == auth.UserID
}

func applyCRMConditionScope(query *gorm.DB, setting map[string]interface{}) *gorm.DB {
	clause, args := crmConditionScopeClause(setting)
	if clause == "" {
		return query
	}
	return query.Where(clause, args...)
}

func applyCRMConditionScopeWithACL(query *gorm.DB, auth authContext, domain string, setting map[string]interface{}) *gorm.DB {
	clause, args := crmConditionScopeClause(setting)
	if clause == "" {
		return applyCRMObjectACLVisibility(query, auth, domain)
	}
	aclClause, aclArgs := crmObjectACLExistsClause(auth)
	whereArgs := make([]interface{}, 0, len(args)+5+len(aclArgs))
	whereArgs = append(whereArgs, args...)
	whereArgs = append(whereArgs, "%\"visibility\":\"private\"%", "%\"private\":true%", auth.UserID, auth.UserID)
	whereArgs = append(whereArgs, aclArgs...)
	return query.Where("(("+clause+") AND crm_objects.payload_json NOT LIKE ? AND crm_objects.payload_json NOT LIKE ?) OR crm_objects.owner_id = ? OR crm_objects.created_by = ? OR "+aclClause+")", whereArgs...)
}

func crmConditionScopeClause(setting map[string]interface{}) (string, []interface{}) {
	conditions, _ := setting["conditions"].(map[string]interface{})
	if len(conditions) == 0 {
		conditions, _ = setting["where"].(map[string]interface{})
	}
	parts := make([]string, 0, len(conditions))
	args := make([]interface{}, 0, len(conditions))
	for key, value := range conditions {
		switch key {
		case "owner_id", "created_by", "department_id", "company_id", "contact_id", "deal_id", "project_id", "status", "object_type":
			values := interfaceList(value)
			if len(values) > 0 {
				parts = append(parts, "crm_objects."+key+" IN ?")
				args = append(args, values)
			}
		}
	}
	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, " AND "), args
}

func applyCRMFieldACL(auth authContext, domain string, items []models.CRMObject) []models.CRMObject {
	setting := crmSettingValue(auth.TenantID, "field_acl", domain)
	hiddenSet := fieldSet(setting["hidden"])
	if len(hiddenSet) == 0 || auth.IsSuperAdmin() || auth.IsTenantAdmin() || bypassCRMFieldACL(auth, setting) {
		return items
	}
	for index := range items {
		payload := payloadMap(items[index].PayloadJSON)
		for key := range hiddenSet {
			delete(payload, key)
		}
		next, _ := json.Marshal(payload)
		items[index].PayloadJSON = string(next)
	}
	return items
}

func applyCRMObjectCapabilities(auth authContext, items []models.CRMObject) []models.CRMObject {
	for index := range items {
		object := items[index]
		items[index].CanView = canViewCRMObject(auth, object)
		items[index].CanFollow = canFollowCRMObject(auth, object)
		items[index].CanEdit = canManageCRMObject(auth, object)
		items[index].CanDelete = canManageCRMObject(auth, object)
		items[index].CanManageACL = canManageCRMObject(auth, object)
	}
	return items
}

func validateCRMFieldWrite(auth authContext, domain string, next map[string]interface{}, before map[string]interface{}) error {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() {
		return nil
	}
	acl := crmSettingValue(auth.TenantID, "field_acl", domain)
	if bypassCRMFieldACL(auth, acl) {
		return nil
	}
	return validateCRMFieldWriteWithACL(next, before, acl)
}

func validateCRMFieldWriteWithACL(next map[string]interface{}, before map[string]interface{}, acl map[string]interface{}) error {
	blocked := fieldSet(acl["hidden"])
	for key := range fieldSet(acl["readonly"]) {
		blocked[key] = true
	}
	for key := range fieldSet(acl["read_only"]) {
		blocked[key] = true
	}
	if len(blocked) == 0 {
		return nil
	}
	for key := range blocked {
		nextValue, touched := next[key]
		if !touched {
			continue
		}
		if before == nil {
			return fmt.Errorf("字段 %s 无写入权限", key)
		}
		if !reflect.DeepEqual(nextValue, before[key]) {
			return fmt.Errorf("字段 %s 为只读或隐藏字段，不能修改", key)
		}
	}
	return nil
}

func ensureCRMModuleAccess(auth authContext, domain string, action string) error {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() {
		return nil
	}
	level := crmModuleAccessLevel(auth, domain)
	switch action {
	case "edit", "write", "manage":
		if level != "" && level != "edit" && level != "manage" {
			return fmt.Errorf("当前角色对模块 %s 只有%s权限", domain, crmModuleAccessLabel(level))
		}
	case "read", "view":
		if level == "none" {
			return fmt.Errorf("当前角色无权访问模块 %s", domain)
		}
	}
	return nil
}

func crmModuleAccessLevel(auth authContext, domain string) string {
	setting := crmSettingValue(auth.TenantID, "permission_matrix", "modules")
	if len(setting) == 0 {
		return ""
	}
	roleKeys := roleMatrixKeys(auth)
	for _, sectionKey := range []string{"roles", "matrix"} {
		section, _ := setting[sectionKey].(map[string]interface{})
		for _, roleKey := range roleKeys {
			if moduleRules, ok := section[roleKey].(map[string]interface{}); ok {
				if level := strings.ToLower(strings.TrimSpace(fmt.Sprint(moduleRules[domain]))); level != "" {
					return level
				}
			}
		}
	}
	if moduleRules, ok := setting[fmt.Sprint(auth.RoleID)].(map[string]interface{}); ok {
		return strings.ToLower(strings.TrimSpace(fmt.Sprint(moduleRules[domain])))
	}
	if moduleRules, ok := setting[auth.Role].(map[string]interface{}); ok {
		return strings.ToLower(strings.TrimSpace(fmt.Sprint(moduleRules[domain])))
	}
	return ""
}

func roleMatrixKeys(auth authContext) []string {
	keys := []string{}
	seen := map[string]bool{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		keys = append(keys, value)
	}
	for _, roleID := range roleIDsForAuth(auth) {
		add(fmt.Sprint(roleID))
		add("role:" + fmt.Sprint(roleID))
	}
	add(auth.Role)
	return keys
}

func crmModuleAccessLabel(level string) string {
	switch level {
	case "none":
		return "无访问"
	case "read", "readonly", "read_only", "view":
		return "只读"
	default:
		return level
	}
}

func bypassCRMFieldACL(auth authContext, setting map[string]interface{}) bool {
	return uintInList(auth.UserID, numberList(setting["bypass_user_ids"])) ||
		uintInList(auth.UserID, numberList(setting["piercing_user_ids"])) ||
		uintInList(auth.UserID, numberList(setting["penetration_user_ids"]))
}

func isCRMDataPiercingUser(auth authContext, domain string) bool {
	setting := crmSettingValue(auth.TenantID, "data_scope", domain)
	return uintInList(auth.UserID, numberList(setting["piercing_user_ids"])) ||
		uintInList(auth.UserID, numberList(setting["penetration_user_ids"])) ||
		uintInList(auth.UserID, numberList(setting["bypass_user_ids"]))
}

func isCRMPrivateVisibilitySupervisor(auth authContext, domain string) bool {
	if !crmSupervisorPrivateVisibilityDomain(domain) {
		return false
	}
	return authHasSupervisorRole(auth)
}

func crmSupervisorPrivateVisibilityDomain(domain string) bool {
	switch domain {
	case "campaigns", "leads", "opportunities", "orders":
		return true
	default:
		return false
	}
}

func authHasSupervisorRole(auth authContext) bool {
	if auth.TenantID == 0 {
		return false
	}
	if isSupervisorRoleToken(auth.Role) {
		return true
	}
	roleIDs := roleIDsForAuth(auth)
	if len(roleIDs) == 0 {
		return false
	}
	var roles []models.Role
	if err := utils.DB.Select("id", "name", "code").Where("tenant_id = ? AND id IN ?", auth.TenantID, roleIDs).Find(&roles).Error; err != nil {
		return false
	}
	for _, role := range roles {
		if isSupervisorRoleToken(role.Name) || isSupervisorRoleToken(role.Code) {
			return true
		}
	}
	return false
}

func isSupervisorRoleToken(value string) bool {
	token := strings.ToUpper(strings.TrimSpace(value))
	switch token {
	case "主管", "SDR主管", "R_SUPERVISOR", "R_SDR_SUPERVISOR":
		return true
	default:
		return false
	}
}

func isCRMModuleManager(auth authContext, domain string) bool {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() {
		return true
	}
	return crmModuleAccessLevel(auth, domain) == "manage"
}

func crmSettingValue(tenantID uint, scope string, key string) map[string]interface{} {
	var setting models.CRMSetting
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", tenantID, scope, key).First(&setting).Error; err != nil {
		return map[string]interface{}{}
	}
	var value map[string]interface{}
	if err := json.Unmarshal([]byte(setting.ValueJSON), &value); err != nil {
		return map[string]interface{}{}
	}
	return value
}

func tenantBusinessCurrency(tenantID uint) string {
	currency := normalizeCurrencyCode(fmt.Sprint(crmSettingValue(tenantID, "company", "enterprise")["currency"]))
	if currency == "" {
		return defaultBusinessCurrencyCode
	}
	return currency
}

func ensureCRMObjectVisible(auth authContext, objectType string, id uint) error {
	var count int64
	if err := crmScopedObjectQuery(auth, objectType).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func filterVisibleRelations(auth authContext, items []models.CRMRelation) []models.CRMRelation {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() {
		return items
	}
	visible := make([]models.CRMRelation, 0, len(items))
	for _, item := range items {
		if ensureCRMObjectVisible(auth, item.SourceType, item.SourceID) != nil {
			continue
		}
		if ensureCRMObjectVisible(auth, item.TargetType, item.TargetID) != nil {
			continue
		}
		visible = append(visible, item)
	}
	return visible
}

func defaultTenantID() uint {
	var tenant models.Tenant
	if err := utils.DB.Where("status = ?", "active").Order("id asc").First(&tenant).Error; err == nil && tenant.ID > 0 {
		return tenant.ID
	}
	return 1
}

func payloadMap(payloadJSON string) map[string]interface{} {
	payload := map[string]interface{}{}
	_ = json.Unmarshal([]byte(payloadJSON), &payload)
	return payload
}

func fieldSet(value interface{}) map[string]bool {
	items := map[string]bool{}
	for _, item := range interfaceList(value) {
		text := strings.TrimSpace(fmt.Sprint(item))
		if text != "" {
			items[text] = true
		}
	}
	return items
}

func numberList(value interface{}) []uint {
	values := interfaceList(value)
	items := make([]uint, 0, len(values))
	for _, value := range values {
		parsed := uintFromValue(value)
		if parsed > 0 {
			items = append(items, parsed)
		}
	}
	return items
}

func uintFromValue(value interface{}) uint {
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return uint(typed)
		}
	case int:
		if typed > 0 {
			return uint(typed)
		}
	case int64:
		if typed > 0 {
			return uint(typed)
		}
	case uint:
		return typed
	case string:
		var parsed uint
		if _, err := fmt.Sscanf(typed, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return 0
}

func interfaceList(value interface{}) []interface{} {
	switch typed := value.(type) {
	case []interface{}:
		return typed
	case []map[string]interface{}:
		items := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items
	case []string:
		items := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items
	case []uint:
		items := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items
	case []int:
		items := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items
	default:
		if value == nil {
			return []interface{}{}
		}
		return []interface{}{value}
	}
}

func uintInList(value uint, values []uint) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func parseOptionalTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("invalid time")
}

func approvalNodesFromJSON(value string) ([]map[string]interface{}, error) {
	var raw interface{}
	if err := json.Unmarshal([]byte(value), &raw); err != nil {
		return nil, err
	}
	nodes := make([]map[string]interface{}, 0)
	switch typed := raw.(type) {
	case []interface{}:
		for index, item := range typed {
			node, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("审批节点[%d]格式无效", index+1)
			}
			nodes = append(nodes, normalizeApprovalNode(index+1, node))
		}
	case map[string]interface{}:
		for index, item := range interfaceList(typed["nodes"]) {
			node, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("审批节点[%d]格式无效", index+1)
			}
			nodes = append(nodes, normalizeApprovalNode(index+1, node))
		}
	default:
		return nil, fmt.Errorf("审批节点格式无效")
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("审批节点不能为空")
	}
	return nodes, nil
}

func approvalNodeHasApprover(node map[string]interface{}) bool {
	if len(node) == 0 {
		return false
	}
	return uintFromValue(node["approver_id"]) > 0 ||
		uintFromValue(node["user_id"]) > 0 ||
		len(numberList(node["approver_ids"])) > 0 ||
		len(numberList(node["user_ids"])) > 0 ||
		uintFromValue(node["approver_role_id"]) > 0 ||
		uintFromValue(node["role_id"]) > 0 ||
		strings.TrimSpace(fmt.Sprint(node["approver_role"])) != "" ||
		strings.TrimSpace(fmt.Sprint(node["role"])) != ""
}

func approvalActorMatches(auth authContext, node map[string]interface{}) bool {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() {
		return true
	}
	if uintFromValue(node["approver_id"]) == auth.UserID || uintFromValue(node["user_id"]) == auth.UserID {
		return true
	}
	if uintInList(auth.UserID, numberList(node["approver_ids"])) || uintInList(auth.UserID, numberList(node["user_ids"])) {
		return true
	}
	roleIDs := roleIDsForAuth(auth)
	if uintInList(uintFromValue(node["approver_role_id"]), roleIDs) || uintInList(uintFromValue(node["role_id"]), roleIDs) {
		return true
	}
	role := strings.TrimSpace(fmt.Sprint(node["approver_role"]))
	if role == "" {
		role = strings.TrimSpace(fmt.Sprint(node["role"]))
	}
	if role != "" && (role == auth.Role || role == fmt.Sprint(auth.RoleID) || role == "role:"+fmt.Sprint(auth.RoleID)) {
		return true
	}
	return !approvalNodeHasApprover(node)
}

func numericValue(value interface{}) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case uint:
		return float64(typed)
	case string:
		var parsed float64
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%f", &parsed); err == nil {
			return parsed
		}
	}
	return 0
}

func mapValue(value interface{}) map[string]interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed
	case map[string]string:
		items := map[string]interface{}{}
		for key, value := range typed {
			items[key] = value
		}
		return items
	default:
		return map[string]interface{}{}
	}
}

func stringListValue(value interface{}, fallback []string) []string {
	values := interfaceList(value)
	items := make([]string, 0, len(values))
	for _, item := range values {
		text := strings.TrimSpace(fmt.Sprint(item))
		if text != "" && text != "<nil>" {
			items = append(items, text)
		}
	}
	if len(items) == 0 {
		return fallback
	}
	return items
}

func boolValue(value interface{}, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		text := strings.ToLower(strings.TrimSpace(typed))
		if text == "true" || text == "1" || text == "yes" || text == "enabled" {
			return true
		}
		if text == "false" || text == "0" || text == "no" || text == "disabled" {
			return false
		}
	case float64, float32, int, int64, uint:
		return numericValue(value) > 0
	}
	return fallback
}

func parseTimeValue(value interface{}) (time.Time, bool) {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func salesFunnel(auth authContext) []map[string]interface{} {
	stages := []string{"initial_contact", "need_confirmed", "proposal", "negotiation", "won", "lost", "on_hold"}
	items := make([]map[string]interface{}, 0, len(stages))
	for _, stage := range stages {
		var count int64
		var amount float64
		_ = crmScopedObjectQuery(auth, "opportunities").Where("crm_objects.status = ?", stage).Count(&count).Error
		_ = crmScopedObjectQuery(auth, "opportunities").Select("COALESCE(SUM(crm_objects.amount), 0)").Where("crm_objects.status = ?", stage).Scan(&amount).Error
		items = append(items, map[string]interface{}{"stage": stage, "count": count, "amount": amount})
	}
	return items
}

func salesTrend(auth authContext, year int) []map[string]interface{} {
	items := []map[string]interface{}{}
	if year <= 0 {
		year = time.Now().Year()
	}
	loc := time.Now().Location()
	metricBasis := salesTargetSettingMetricBasis(auth.TenantID)
	for month := 1; month <= 12; month++ {
		start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
		end := start.AddDate(0, 1, 0)
		amount := aggregateMetricAmount(salesOrderAggregateForRange(auth.TenantID, start, end), metricBasis)
		items = append(items, map[string]interface{}{"month": start.Format("2006-01"), "amount": amount})
	}
	return items
}

func normalizeAttributionTrace(value interface{}) []map[string]interface{} {
	touches := []map[string]interface{}{}
	for _, raw := range interfaceList(value) {
		switch typed := raw.(type) {
		case map[string]interface{}:
			touch := map[string]interface{}{}
			for key, value := range typed {
				touch[key] = value
			}
			touches = append(touches, touch)
		case string:
			text := strings.TrimSpace(typed)
			if text != "" {
				touches = append(touches, map[string]interface{}{"touch": text, "channel": text, "stage": "trace", "source": "trace"})
			}
		}
	}
	return touches
}

func weightAttributionTouches(touches []map[string]interface{}, convertedAt time.Time, halfLifeDays float64) []map[string]interface{} {
	if len(touches) == 0 {
		return touches
	}
	rawWeights := make([]float64, len(touches))
	total := 0.0
	for index, touch := range touches {
		positionWeight := 0.2
		switch {
		case len(touches) == 1:
			positionWeight = 1
		case index == 0:
			positionWeight = 0.35
		case index == len(touches)-1:
			positionWeight = 0.35
		default:
			positionWeight = 0.3 / float64(len(touches)-2)
		}
		decay := 1.0
		if occurredAt, ok := parseTimeValue(touch["occurred_at"]); ok && !convertedAt.IsZero() {
			ageDays := convertedAt.Sub(occurredAt).Hours() / 24
			if ageDays < 0 {
				ageDays = 0
			}
			decay = 1 / (1 + ageDays/halfLifeDays)
		}
		rawWeights[index] = positionWeight * decay
		total += rawWeights[index]
		touch["decay_factor"] = decay
	}
	if total <= 0 {
		total = 1
	}
	for index := range touches {
		touches[index]["weight"] = rawWeights[index] / total
	}
	return touches
}

func relatedObjects(auth authContext, sourceType string, sourceID uint, targetType string, relationType string) []models.CRMObject {
	var relations []models.CRMRelation
	query := utils.DB.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ?", auth.TenantID, sourceType, sourceID, targetType)
	if relationType != "" {
		query = query.Where("relation_type = ?", relationType)
	}
	_ = query.Find(&relations).Error
	ids := []uint{}
	for _, relation := range relations {
		ids = append(ids, relation.TargetID)
	}
	if len(ids) == 0 {
		return []models.CRMObject{}
	}
	var objects []models.CRMObject
	_ = crmScopedObjectQuery(auth, targetType).Where("crm_objects.id IN ?", ids).Find(&objects).Error
	return objects
}

func mapsToInterfaces(items []map[string]interface{}) []interface{} {
	if len(items) == 0 {
		return []interface{}{}
	}
	next := make([]interface{}, 0, len(items))
	for _, item := range items {
		next = append(next, item)
	}
	return next
}

func sumLineItems(items []interface{}) float64 {
	var total float64
	for _, raw := range items {
		line := mapValue(raw)
		qty := numericValue(line["quantity"])
		if qty <= 0 {
			qty = 1
		}
		price := numericValue(line["unit_price"])
		if price <= 0 {
			price = numericValue(line["price"])
		}
		discount := numericValue(line["discount"])
		lineTotal := qty * price
		if discount > 0 && discount < 100 {
			lineTotal = lineTotal * (100 - discount) / 100
		}
		total += lineTotal
	}
	return total
}

func sumLineItemTax(items []interface{}) float64 {
	var total float64
	for _, raw := range items {
		line := mapValue(raw)
		qty := numericValue(line["quantity"])
		if qty <= 0 {
			qty = 1
		}
		price := numericValue(line["unit_price"])
		if price <= 0 {
			price = numericValue(line["price"])
		}
		discount := numericValue(line["discount"])
		lineTotal := qty * price
		if discount > 0 && discount < 100 {
			lineTotal = lineTotal * (100 - discount) / 100
		}
		taxRate := numericValue(line["tax_rate"])
		total += lineTotal * taxRate / 100
	}
	return total
}

func normalizePaymentPlan(items []map[string]interface{}) []map[string]interface{} {
	return normalizePaymentPlanWithCurrency(items, defaultBusinessCurrencyCode)
}

func normalizePaymentPlanWithCurrency(items []map[string]interface{}, defaultCurrency string) []map[string]interface{} {
	defaultCurrency = normalizeCurrencyCode(defaultCurrency)
	if defaultCurrency == "" {
		defaultCurrency = defaultBusinessCurrencyCode
	}
	next := make([]map[string]interface{}, 0, len(items))
	for index, item := range items {
		name := strings.TrimSpace(fmt.Sprint(item["name"]))
		if name == "" || strings.EqualFold(name, "<nil>") || strings.EqualFold(name, "null") || strings.EqualFold(name, "undefined") {
			name = fmt.Sprintf("第%d期", index+1)
		}
		amount := numericValue(item["amount"])
		dueDate := strings.TrimSpace(fmt.Sprint(item["due_date"]))
		if strings.EqualFold(dueDate, "<nil>") || strings.EqualFold(dueDate, "null") || strings.EqualFold(dueDate, "undefined") {
			dueDate = ""
		}
		currency := normalizeCurrencyCode(fmt.Sprint(item["currency"]))
		if currency == "" {
			currency = defaultCurrency
		}
		next = append(next, map[string]interface{}{"name": name, "amount": amount, "currency": currency, "due_date": dueDate})
	}
	return next
}

func normalizePaymentPlanFromInterface(value interface{}) []map[string]interface{} {
	return normalizePaymentPlanFromInterfaceWithCurrency(value, defaultBusinessCurrencyCode)
}

func normalizePaymentPlanFromInterfaceWithCurrency(value interface{}, defaultCurrency string) []map[string]interface{} {
	items := interfaceList(value)
	next := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		next = append(next, mapValue(item))
	}
	return normalizePaymentPlanWithCurrency(next, defaultCurrency)
}

func overduePaymentAmount(plan []map[string]interface{}, paidAmount float64) float64 {
	var dueAmount float64
	today := time.Now()
	for _, item := range plan {
		dueDate, ok := parseDateOnly(fmt.Sprint(item["due_date"]))
		if !ok || dueDate.After(today) {
			continue
		}
		dueAmount += numericValue(item["amount"])
	}
	overdue := dueAmount - paidAmount
	if overdue < 0 {
		return 0
	}
	return overdue
}

func parseDateOnly(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func exchangeRateForCurrency(setting map[string]interface{}, currency string) float64 {
	rates := mapValue(setting["exchange_rates"])
	if rate := numericValue(rates[currency]); rate > 0 {
		return rate
	}
	if strings.EqualFold(currency, fmt.Sprint(setting["base_currency"])) {
		return 1
	}
	return 1
}

func applyTaxCodesToLineItems(auth authContext, items []interface{}) {
	if len(items) == 0 {
		return
	}
	rates := taxCodeRates(orderFinanceSetting(auth))
	for _, raw := range items {
		line := mapValue(raw)
		taxCode := strings.TrimSpace(fmt.Sprint(line["tax_code"]))
		if taxCode == "" {
			continue
		}
		if rate, ok := rates[taxCode]; ok && numericValue(line["tax_rate"]) == 0 {
			line["tax_rate"] = rate
		}
	}
}

func taxCodeRates(setting map[string]interface{}) map[string]float64 {
	rates := map[string]float64{}
	for _, raw := range interfaceList(setting["tax_codes"]) {
		item := mapValue(raw)
		code := strings.TrimSpace(fmt.Sprint(item["code"]))
		if code == "" {
			continue
		}
		rates[code] = numericValue(item["rate"])
	}
	return rates
}

func stringInSlice(value string, items []string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func firstPositiveUint(values ...uint) uint {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
