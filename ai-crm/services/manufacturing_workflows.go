package services

import (
	"ai-crm/models"
	"ai-crm/utils"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	MfgErrorNotFound                  = "MFG_RECORD_NOT_FOUND"
	MfgErrorUnsupportedResource       = "MFG_UNSUPPORTED_RESOURCE"
	MfgErrorRFQMissingRequiredFields  = "MFG_RFQ_MISSING_REQUIRED_FIELDS"
	MfgErrorBOMSnapshotRequired       = "MFG_BOM_SNAPSHOT_REQUIRED"
	MfgErrorLowMarginApprovalRequired = "MFG_LOW_MARGIN_APPROVAL_REQUIRED"
	MfgErrorOrderBOMLocked            = "MFG_ORDER_BOM_LOCKED"
	MfgErrorECNRequired               = "MFG_ECN_REQUIRED"
	MfgErrorInvalidState              = "MFG_INVALID_STATE"
	MfgErrorDuplicateSerialNo         = "MFG_DUPLICATE_SERIAL_NO"
	MfgErrorIoTUnauthorized           = "MFG_IOT_UNAUTHORIZED"
	MfgErrorDrawingApprovalRequired   = "MFG_DRAWING_APPROVAL_REQUIRED"
	MfgErrorClarificationAnswerLocked = "MFG_CLARIFICATION_ANSWER_LOCKED"
	MfgErrorCurrentBOMRequired        = "MFG_CURRENT_BOM_REQUIRED"
	MfgErrorApprovalRemarkRequired    = "MFG_APPROVAL_REMARK_REQUIRED"
)

type MfgListParams struct {
	Page            int
	Size            int
	Keyword         string
	Status          string
	CustomerID      uint
	RFQID           uint
	ProductFamilyID uint
	OrderID         uint
	AssetID         uint
}

type MfgAccessContext struct {
	UserID           uint
	TenantID         uint
	IsAdmin          bool
	IsSupervisor     bool
	CanManageRFQ     bool
	CanManageCost    bool
	CanManageProduct bool
	CanManageQuote   bool
	CanApproveQuote  bool
}

type MfgFileReference struct {
	FileID        string
	StorageNodeID string
	Source        string
}

type MfgOrderCreateInput struct {
	QuoteID    uint                   `json:"quote_id"`
	OrderID    uint                   `json:"order_id"`
	CustomerID uint                   `json:"customer_id"`
	ContactID  uint                   `json:"contact_id"`
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	Payload    map[string]interface{} `json:"payload"`
}

type MfgShipmentAssetInput struct {
	OrderID         uint                   `json:"order_id"`
	SerialNumbers   []string               `json:"serial_numbers"`
	WarrantyMonths  int                    `json:"warranty_months"`
	InstallLocation string                 `json:"install_location"`
	IoTAssetRefs    map[string]string      `json:"iot_asset_refs"`
	Payload         map[string]interface{} `json:"payload"`
}

type MfgIoTAlarmInput struct {
	AssetID     uint                   `json:"asset_id"`
	IoTAssetRef string                 `json:"iot_asset_ref"`
	AlarmCode   string                 `json:"alarm_code"`
	Severity    string                 `json:"severity"`
	OccurredAt  string                 `json:"occurred_at"`
	Message     string                 `json:"message"`
	Payload     map[string]interface{} `json:"payload"`
}

type MfgServiceOpportunityInput struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
}

type MfgBOMSnapshotInput struct {
	BusinessName        string                 `json:"business_name"`
	ProductFamilyID     uint                   `json:"product_family_id"`
	ConfigModelID       uint                   `json:"config_model_id"`
	BOMTemplateID       uint                   `json:"bom_template_id"`
	RFQID               uint                   `json:"rfq_id"`
	ConfigValues        map[string]interface{} `json:"config_values"`
	BOMLines            interface{}            `json:"bom_lines"`
	SourceConfigVersion string                 `json:"source_config_version"`
	SaveAsTemplate      bool                   `json:"save_as_template"`
	TemplateNo          string                 `json:"template_no"`
	TemplateVersion     string                 `json:"template_version"`
	TemplateStatus      string                 `json:"template_status"`
	TemplateRule        interface{}            `json:"template_rule"`
}

type MfgRFQCostingResult struct {
	RFQ       models.MfgRFQ       `json:"rfq"`
	CostSheet models.MfgCostSheet `json:"cost_sheet"`
}

type mfgResourceSpec struct {
	Name           string
	Model          func() interface{}
	Slice          func() interface{}
	KeywordColumns []string
	StatusColumn   string
	NoField        string
	NoPrefix       string
	Immutable      bool
	Versioned      bool
	FileFields     []mfgFileField
}

type mfgFileField struct {
	FileField string
	NodeField string
}

func ListMfgRecords(tenantID uint, resource string, params MfgListParams) (interface{}, int64, error) {
	spec, ok := mfgResource(resource)
	if !ok {
		return nil, 0, errors.New(MfgErrorUnsupportedResource)
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Size <= 0 {
		params.Size = 20
	}
	if params.Size > 100 {
		params.Size = 100
	}
	items := spec.Slice()
	query := utils.DB.Model(spec.Model()).Where("tenant_id = ?", tenantID)
	if params.Status != "" && spec.StatusColumn != "" {
		query = query.Where(spec.StatusColumn+" = ?", strings.TrimSpace(params.Status))
	}
	if params.CustomerID > 0 && hasField(spec.Model(), "CustomerID") {
		query = query.Where("customer_id = ?", params.CustomerID)
	}
	if params.RFQID > 0 && hasField(spec.Model(), "RFQID") {
		query = query.Where("rfq_id = ?", params.RFQID)
	}
	if params.ProductFamilyID > 0 && hasField(spec.Model(), "ProductFamilyID") {
		query = query.Where("product_family_id = ?", params.ProductFamilyID)
	}
	if params.OrderID > 0 && hasField(spec.Model(), "OrderID") {
		query = query.Where("order_id = ?", params.OrderID)
	}
	if params.AssetID > 0 && hasField(spec.Model(), "AssetID") {
		query = query.Where("asset_id = ?", params.AssetID)
	}
	if keyword := strings.TrimSpace(params.Keyword); keyword != "" && len(spec.KeywordColumns) > 0 {
		parts := make([]string, 0, len(spec.KeywordColumns))
		args := make([]interface{}, 0, len(spec.KeywordColumns))
		for _, column := range spec.KeywordColumns {
			parts = append(parts, column+" LIKE ?")
			args = append(args, "%"+keyword+"%")
		}
		for _, id := range matchingMfgCRMObjectIDs(tenantID, "company", keyword) {
			if hasField(spec.Model(), "CustomerID") {
				parts = append(parts, "customer_id = ?")
				args = append(args, id)
			}
		}
		for _, id := range matchingMfgCRMObjectIDs(tenantID, "order", keyword) {
			if hasField(spec.Model(), "OrderID") {
				parts = append(parts, "order_id = ?")
				args = append(args, id)
			}
		}
		for _, id := range matchingMfgUserIDs(tenantID, keyword) {
			if hasField(spec.Model(), "SalesOwnerID") {
				parts = append(parts, "sales_owner_id = ?")
				args = append(args, id)
			}
		}
		query = query.Where("("+strings.Join(parts, " OR ")+")", args...)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("updated_at desc, id desc").Offset((params.Page - 1) * params.Size).Limit(params.Size).Find(items).Error; err != nil {
		return nil, 0, err
	}
	return derefInterface(items), total, nil
}

func SanitizeMfgRecords(resource string, value interface{}, canViewSensitive bool) interface{} {
	return SanitizeMfgRecordsForAccess(resource, value, MfgAccessContext{IsAdmin: canViewSensitive, IsSupervisor: canViewSensitive})
}

func SanitizeMfgRecordsForAccess(resource string, value interface{}, access MfgAccessContext) interface{} {
	if resource == "rfqs" {
		return sanitizeMfgRFQValue(value, access)
	}
	if resource == "cost-sheets" {
		return enrichMfgCostSheetValue(value, access)
	}
	if resource == "production-progress" {
		return enrichMfgProductionProgressValue(value)
	}
	if resource == "ecns" {
		return enrichMfgECNValue(value)
	}
	if resource == "compensation-records" {
		return enrichMfgCompensationRecordValue(value)
	}
	if resource == "bom-snapshots" {
		return enrichMfgBOMSnapshotValueForAccess(value, access)
	}
	if resource == "bom-templates" {
		return enrichMfgBOMTemplateValue(value)
	}
	if resource == "product-families" {
		return enrichMfgProductFamilyValue(value)
	}
	if resource == "rfq-clarifications" {
		return enrichMfgRFQClarificationValue(value)
	}
	if resource == "quotes" {
		return enrichMfgQuoteRevisionValueForAccess(value, access)
	}
	if resource == "sample-projects" {
		return enrichMfgSampleProjectValue(value)
	}
	if access.IsAdmin || access.IsSupervisor {
		return value
	}
	switch typed := value.(type) {
	case []models.MfgCostSheet:
		return enrichMfgCostSheetValue(typed, access)
	case []models.MfgSparePartQuote:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, sanitizeMfgSparePartQuote(item))
		}
		return items
	case models.MfgCostSheet:
		return enrichMfgCostSheet(typed, access)
	case models.MfgSparePartQuote:
		return sanitizeMfgSparePartQuote(typed)
	default:
		if resource == "cost-sheets" {
			if item, ok := value.(models.MfgCostSheet); ok {
				return enrichMfgCostSheet(item, access)
			}
		}
		if resource == "spare-part-quotes" {
			if item, ok := value.(models.MfgSparePartQuote); ok {
				return sanitizeMfgSparePartQuote(item)
			}
		}
		return value
	}
}

func GetMfgRecord(tenantID uint, resource string, id uint) (interface{}, error) {
	spec, ok := mfgResource(resource)
	if !ok {
		return nil, errors.New(MfgErrorUnsupportedResource)
	}
	item := spec.Model()
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, id).First(item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(MfgErrorNotFound)
		}
		return nil, err
	}
	return derefInterface(item), nil
}

func GetMfgRecordFileReferences(tenantID uint, resource string, id uint) ([]MfgFileReference, error) {
	spec, ok := mfgResource(resource)
	if !ok {
		return nil, errors.New(MfgErrorUnsupportedResource)
	}
	item := spec.Model()
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, id).First(item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(MfgErrorNotFound)
		}
		return nil, err
	}
	return MfgRecordFileReferences(resource, item), nil
}

func CreateMfgRecord(tenantID uint, userID uint, resource string, input map[string]interface{}) (interface{}, error) {
	if resource == "sample-projects" {
		return createMfgSampleProject(tenantID, userID, input)
	}
	if resource == "production-progress" {
		return createMfgProductionProgress(tenantID, userID, input)
	}
	spec, ok := mfgResource(resource)
	if !ok {
		return nil, errors.New(MfgErrorUnsupportedResource)
	}
	item := spec.Model()
	if err := applyMfgInput(item, input); err != nil {
		return nil, err
	}
	setUintField(item, "TenantID", tenantID)
	setUintField(item, "CreatedBy", userID)
	setUintField(item, "UpdatedBy", userID)
	setDefaultNo(item, spec.NoField, spec.NoPrefix)
	applyMfgCRMDefaults(tenantID, item)
	if err := validateMfgRecordForSave(tenantID, resource, item); err != nil {
		return nil, err
	}
	if err := utils.DB.Create(item).Error; err != nil {
		return nil, err
	}
	if resource == "rfqs" {
		if err := syncRFQRequirementClarifications(tenantID, userID, derefInterface(item)); err != nil {
			return nil, err
		}
	}
	return SanitizeMfgRecords(resource, derefInterface(item), true), nil
}

func UpdateMfgRecord(tenantID uint, userID uint, resource string, id uint, input map[string]interface{}) (interface{}, error) {
	if resource == "rfq-clarifications" {
		return updateMfgRFQClarification(tenantID, userID, id, input)
	}
	if resource == "sample-projects" {
		return updateMfgSampleProject(tenantID, userID, id, input)
	}
	if resource == "production-progress" {
		return updateMfgProductionProgress(tenantID, userID, id, input)
	}
	spec, ok := mfgResource(resource)
	if !ok {
		return nil, errors.New(MfgErrorUnsupportedResource)
	}
	if spec.Immutable {
		return nil, errors.New(MfgErrorInvalidState)
	}
	item := spec.Model()
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, id).First(item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(MfgErrorNotFound)
		}
		return nil, err
	}
	if spec.Versioned && mfgVersionedRecordLocked(item) {
		return nil, errors.New(MfgErrorInvalidState)
	}
	if err := applyMfgInput(item, input); err != nil {
		return nil, err
	}
	setUintField(item, "ID", id)
	setUintField(item, "TenantID", tenantID)
	setUintField(item, "UpdatedBy", userID)
	applyMfgCRMDefaults(tenantID, item)
	if err := validateMfgRecordForSave(tenantID, resource, item); err != nil {
		return nil, err
	}
	if err := utils.DB.Save(item).Error; err != nil {
		return nil, err
	}
	if resource == "rfqs" {
		if err := syncRFQRequirementClarifications(tenantID, userID, derefInterface(item)); err != nil {
			return nil, err
		}
	}
	return SanitizeMfgRecords(resource, derefInterface(item), true), nil
}

func updateMfgRFQClarification(tenantID uint, userID uint, id uint, input map[string]interface{}) (interface{}, error) {
	var saved models.MfgRFQClarification
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		var before models.MfgRFQClarification
		if err := tx.Where("tenant_id = ? AND id = ?", tenantID, id).First(&before).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(MfgErrorNotFound)
			}
			return err
		}
		next := before
		if err := applyMfgInput(&next, input); err != nil {
			return err
		}
		next.ID = id
		next.TenantID = tenantID
		next.CreatedBy = before.CreatedBy
		next.LastAnsweredBy = before.LastAnsweredBy
		next.LastAnsweredAt = before.LastAnsweredAt
		next.UpdatedBy = userID
		if err := validateMfgRecordForSave(tenantID, "rfq-clarifications", &next); err != nil {
			return err
		}
		history, err := applyMfgRFQClarificationAnswerAudit(before, &next, userID, time.Now(), userDisplayName(tenantID, userID))
		if err != nil {
			return err
		}
		if history != nil {
			if err := tx.Create(&history).Error; err != nil {
				return err
			}
		}
		if err := tx.Save(&next).Error; err != nil {
			return err
		}
		saved = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	return SanitizeMfgRecords("rfq-clarifications", saved, true), nil
}

func createMfgSampleProject(tenantID uint, userID uint, input map[string]interface{}) (interface{}, error) {
	var saved models.MfgSampleProject
	reused := false
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		next := models.MfgSampleProject{}
		if err := applyMfgInput(&next, input); err != nil {
			return err
		}
		next.TenantID = tenantID
		next.CreatedBy = userID
		next.UpdatedBy = userID
		if strings.TrimSpace(next.Stage) == "" {
			next.Stage = "sample_need"
		}
		next.Stage = normalizeMfgSampleStage(next.Stage)
		if strings.TrimSpace(next.RiskLevel) == "" {
			next.RiskLevel = "green"
		}
		setDefaultNo(&next, "SampleNo", "SMP")
		if err := validateMfgRecordForSave(tenantID, "sample-projects", &next); err != nil {
			return err
		}
		if next.CustomerID > 0 && next.RFQID > 0 && next.BOMSnapshotID > 0 {
			var existing models.MfgSampleProject
			err := tx.Where("tenant_id = ? AND customer_id = ? AND rfq_id = ? AND bom_snapshot_id = ? AND stage NOT IN ?",
				tenantID, next.CustomerID, next.RFQID, next.BOMSnapshotID, mfgSampleTerminalStages()).
				Order("updated_at desc, id desc").
				First(&existing).Error
			if err == nil {
				saved = existing
				reused = true
				return nil
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		if err := tx.Create(&next).Error; err != nil {
			return err
		}
		history := mfgSampleProgressHistoryFromChange(next, "", userID, "created")
		if err := tx.Create(&history).Error; err != nil {
			return err
		}
		saved = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	payload := enrichMfgSampleProject(saved)
	payload["reused"] = reused
	return payload, nil
}

func updateMfgSampleProject(tenantID uint, userID uint, id uint, input map[string]interface{}) (interface{}, error) {
	var saved models.MfgSampleProject
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		var before models.MfgSampleProject
		if err := tx.Where("tenant_id = ? AND id = ?", tenantID, id).First(&before).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(MfgErrorNotFound)
			}
			return err
		}
		next := before
		if err := applyMfgInput(&next, input); err != nil {
			return err
		}
		next.ID = id
		next.TenantID = tenantID
		next.CreatedBy = before.CreatedBy
		next.UpdatedBy = userID
		next.SampleNo = before.SampleNo
		next.Stage = normalizeMfgSampleStage(next.Stage)
		if strings.TrimSpace(next.Stage) == "" {
			next.Stage = "sample_need"
		}
		if err := validateMfgRecordForSave(tenantID, "sample-projects", &next); err != nil {
			return err
		}
		if mfgSampleProgressChanged(before, next) {
			history := mfgSampleProgressHistoryFromChange(next, before.Stage, userID, "")
			if err := tx.Create(&history).Error; err != nil {
				return err
			}
		}
		if err := tx.Save(&next).Error; err != nil {
			return err
		}
		saved = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	return enrichMfgSampleProject(saved), nil
}

func ListMfgSampleProgressHistories(tenantID uint, sampleProjectID uint) ([]map[string]interface{}, error) {
	var sample models.MfgSampleProject
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, sampleProjectID).First(&sample).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(MfgErrorNotFound)
		}
		return nil, err
	}
	var rows []models.MfgSampleProgressHistory
	if err := utils.DB.Where("tenant_id = ? AND sample_project_id = ?", tenantID, sampleProjectID).Order("created_at desc, id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, enrichMfgSampleProgressHistory(row))
	}
	return items, nil
}

func mfgSampleProgressChanged(before models.MfgSampleProject, next models.MfgSampleProject) bool {
	return normalizeMfgSampleStage(before.Stage) != normalizeMfgSampleStage(next.Stage) ||
		strings.TrimSpace(before.RiskLevel) != strings.TrimSpace(next.RiskLevel) ||
		strings.TrimSpace(before.CustomerFeedback) != strings.TrimSpace(next.CustomerFeedback)
}

func mfgSampleProgressHistoryFromChange(item models.MfgSampleProject, fromStage string, userID uint, note string) models.MfgSampleProgressHistory {
	return models.MfgSampleProgressHistory{
		TenantID:         item.TenantID,
		SampleProjectID:  item.ID,
		FromStage:        normalizeMfgSampleStage(fromStage),
		ToStage:          normalizeMfgSampleStage(item.Stage),
		RiskLevel:        item.RiskLevel,
		CustomerFeedback: item.CustomerFeedback,
		Note:             note,
		CreatedBy:        userID,
	}
}

func mfgSampleTerminalStages() []string {
	return []string{"completed", "paused"}
}

func normalizeMfgSampleStage(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	lower = strings.ReplaceAll(lower, "-", "_")
	lower = strings.ReplaceAll(lower, " ", "_")
	switch lower {
	case "planning", "sample_need", "需求确认":
		return "sample_need"
	case "material_prepare", "materials", "备料":
		return "material_prepare"
	case "sample_build", "sampling", "打样中":
		return "sample_build"
	case "internal_test", "test", "内部测试":
		return "internal_test"
	case "sample_sent", "send_sample", "寄样":
		return "sample_sent"
	case "customer_test", "客户测试":
		return "customer_test"
	case "revision", "modifying", "修改中":
		return "revision"
	case "sealed_confirm", "seal", "封样确认":
		return "sealed_confirm"
	case "mass_prepare", "mass", "量产准备":
		return "mass_prepare"
	case "completed", "done", "finish", "完成":
		return "completed"
	case "paused", "pause", "暂停":
		return "paused"
	default:
		return text
	}
}

func enrichMfgSampleProgressHistory(item models.MfgSampleProgressHistory) map[string]interface{} {
	return map[string]interface{}{
		"id":                item.ID,
		"created_at":        item.CreatedAt,
		"updated_at":        item.UpdatedAt,
		"tenant_id":         item.TenantID,
		"sample_project_id": item.SampleProjectID,
		"from_stage":        item.FromStage,
		"to_stage":          item.ToStage,
		"risk_level":        item.RiskLevel,
		"customer_feedback": item.CustomerFeedback,
		"note":              item.Note,
		"created_by":        item.CreatedBy,
		"created_by_name":   userDisplayName(item.TenantID, item.CreatedBy),
	}
}

func createMfgProductionProgress(tenantID uint, userID uint, input map[string]interface{}) (interface{}, error) {
	var saved models.MfgProductionProgress
	reused := false
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		next := models.MfgProductionProgress{}
		if err := applyMfgInput(&next, input); err != nil {
			return err
		}
		next.TenantID = tenantID
		next.CreatedBy = userID
		next.UpdatedBy = userID
		next.Milestone = normalizeMfgProductionMilestone(firstNonEmpty(next.Milestone, "order_confirmed"))
		next.ProgressStatus = normalizeMfgProductionProgressStatus(firstNonEmpty(next.ProgressStatus, "not_started"))
		next.RiskLevel = firstNonEmpty(next.RiskLevel, "green")
		next.SyncStatus = firstNonEmpty(next.SyncStatus, "local")
		next.SourceSystem = firstNonEmpty(next.SourceSystem, "manual")
		applyMfgCRMDefaults(tenantID, &next)
		if err := validateMfgRecordForSave(tenantID, "production-progress", &next); err != nil {
			return err
		}
		if next.OrderID > 0 {
			var existing models.MfgProductionProgress
			err := tx.Where("tenant_id = ? AND order_id = ? AND progress_status NOT IN ?", tenantID, next.OrderID, mfgProductionTerminalStatuses()).
				Order("updated_at desc, id desc").
				First(&existing).Error
			if err == nil {
				saved = existing
				reused = true
				return nil
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		if err := tx.Create(&next).Error; err != nil {
			return err
		}
		history := mfgProductionProgressHistoryFromChange(next, "", "", userID, "created")
		if err := tx.Create(&history).Error; err != nil {
			return err
		}
		saved = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	payload := enrichMfgProductionProgress(saved)
	payload["reused"] = reused
	return payload, nil
}

func updateMfgProductionProgress(tenantID uint, userID uint, id uint, input map[string]interface{}) (interface{}, error) {
	var saved models.MfgProductionProgress
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		var before models.MfgProductionProgress
		if err := tx.Where("tenant_id = ? AND id = ?", tenantID, id).First(&before).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(MfgErrorNotFound)
			}
			return err
		}
		next := before
		if err := applyMfgInput(&next, input); err != nil {
			return err
		}
		next.ID = id
		next.TenantID = tenantID
		next.CreatedBy = before.CreatedBy
		next.UpdatedBy = userID
		next.Milestone = normalizeMfgProductionMilestone(firstNonEmpty(next.Milestone, "order_confirmed"))
		next.ProgressStatus = normalizeMfgProductionProgressStatus(firstNonEmpty(next.ProgressStatus, "not_started"))
		next.RiskLevel = firstNonEmpty(next.RiskLevel, "green")
		next.SyncStatus = firstNonEmpty(next.SyncStatus, "local")
		next.SourceSystem = firstNonEmpty(next.SourceSystem, "manual")
		applyMfgCRMDefaults(tenantID, &next)
		if err := validateMfgRecordForSave(tenantID, "production-progress", &next); err != nil {
			return err
		}
		if mfgProductionProgressChanged(before, next) {
			history := mfgProductionProgressHistoryFromChange(next, before.Milestone, before.ProgressStatus, userID, "")
			if err := tx.Create(&history).Error; err != nil {
				return err
			}
		}
		if err := tx.Save(&next).Error; err != nil {
			return err
		}
		saved = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	return enrichMfgProductionProgress(saved), nil
}

func ListMfgProductionProgressHistories(tenantID uint, progressID uint) ([]map[string]interface{}, error) {
	var progress models.MfgProductionProgress
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, progressID).First(&progress).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(MfgErrorNotFound)
		}
		return nil, err
	}
	var rows []models.MfgProductionProgressHistory
	if err := utils.DB.Where("tenant_id = ? AND production_progress_id = ?", tenantID, progressID).Order("created_at desc, id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, enrichMfgProductionProgressHistory(row))
	}
	return items, nil
}

func mfgProductionProgressChanged(before models.MfgProductionProgress, next models.MfgProductionProgress) bool {
	return normalizeMfgProductionMilestone(before.Milestone) != normalizeMfgProductionMilestone(next.Milestone) ||
		normalizeMfgProductionProgressStatus(before.ProgressStatus) != normalizeMfgProductionProgressStatus(next.ProgressStatus) ||
		strings.TrimSpace(before.RiskLevel) != strings.TrimSpace(next.RiskLevel) ||
		strings.TrimSpace(before.RiskReason) != strings.TrimSpace(next.RiskReason)
}

func mfgProductionProgressHistoryFromChange(item models.MfgProductionProgress, fromMilestone string, fromStatus string, userID uint, note string) models.MfgProductionProgressHistory {
	return models.MfgProductionProgressHistory{
		TenantID:             item.TenantID,
		ProductionProgressID: item.ID,
		OrderID:              item.OrderID,
		FromMilestone:        normalizeMfgProductionMilestone(fromMilestone),
		ToMilestone:          normalizeMfgProductionMilestone(item.Milestone),
		FromProgressStatus:   normalizeMfgProductionProgressStatus(fromStatus),
		ToProgressStatus:     normalizeMfgProductionProgressStatus(item.ProgressStatus),
		RiskLevel:            item.RiskLevel,
		RiskReason:           item.RiskReason,
		Note:                 note,
		CreatedBy:            userID,
	}
}

func mfgProductionTerminalStatuses() []string {
	return []string{"completed", "canceled"}
}

func normalizeMfgProductionProgressStatus(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	lower = strings.ReplaceAll(lower, "-", "_")
	lower = strings.ReplaceAll(lower, " ", "_")
	switch lower {
	case "not_started", "notstarted", "未开始":
		return "not_started"
	case "in_progress", "progress", "running", "进行中":
		return "in_progress"
	case "paused", "pause", "暂停":
		return "paused"
	case "delayed", "delay", "延期":
		return "delayed"
	case "completed", "complete", "done", "完成":
		return "completed"
	case "canceled", "cancelled", "cancel", "取消":
		return "canceled"
	default:
		return text
	}
}

func normalizeMfgProductionMilestone(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	lower = strings.ReplaceAll(lower, "-", "_")
	lower = strings.ReplaceAll(lower, " ", "_")
	switch lower {
	case "order_confirmed", "order_confirm", "订单确认":
		return "order_confirmed"
	case "scheduled", "schedule", "排产":
		return "scheduled"
	case "material_prepare", "materials", "备料":
		return "material_prepare"
	case "production", "in_production", "生产中":
		return "production"
	case "quality_check", "qc", "质检":
		return "quality_check"
	case "packaging", "package", "包装":
		return "packaging"
	case "ready_to_ship", "待发货":
		return "ready_to_ship"
	case "shipped", "已发货":
		return "shipped"
	case "completed", "done", "完成":
		return "completed"
	case "paused", "pause", "暂停":
		return "paused"
	default:
		return text
	}
}

func enrichMfgProductionProgressHistory(item models.MfgProductionProgressHistory) map[string]interface{} {
	return map[string]interface{}{
		"id":                     item.ID,
		"created_at":             item.CreatedAt,
		"updated_at":             item.UpdatedAt,
		"tenant_id":              item.TenantID,
		"production_progress_id": item.ProductionProgressID,
		"order_id":               item.OrderID,
		"from_milestone":         item.FromMilestone,
		"to_milestone":           item.ToMilestone,
		"from_progress_status":   item.FromProgressStatus,
		"to_progress_status":     item.ToProgressStatus,
		"risk_level":             item.RiskLevel,
		"risk_reason":            item.RiskReason,
		"note":                   item.Note,
		"created_by":             item.CreatedBy,
		"created_by_name":        userDisplayName(item.TenantID, item.CreatedBy),
	}
}

func applyMfgRFQClarificationAnswerAudit(before models.MfgRFQClarification, next *models.MfgRFQClarification, userID uint, answeredAt time.Time, answeredByName string) (*models.MfgRFQClarificationAnswerHistory, error) {
	if next == nil {
		return nil, errors.New(MfgErrorInvalidInput)
	}
	oldAnswer := strings.TrimSpace(before.Answer)
	newAnswer := strings.TrimSpace(next.Answer)
	if oldAnswer == newAnswer {
		return nil, nil
	}
	if oldAnswer != "" && newAnswer == "" {
		return nil, errors.New(MfgErrorClarificationAnswerLocked)
	}
	if newAnswer == "" {
		return nil, nil
	}
	next.LastAnsweredBy = userID
	next.LastAnsweredAt = &answeredAt
	return &models.MfgRFQClarificationAnswerHistory{
		TenantID:        next.TenantID,
		RFQID:           next.RFQID,
		ClarificationID: next.ID,
		OldAnswer:       before.Answer,
		NewAnswer:       next.Answer,
		OldStatus:       before.Status,
		NewStatus:       next.Status,
		AnsweredBy:      userID,
		AnsweredByName:  firstNonEmpty(answeredByName, fmt.Sprintf("#%d", userID)),
		AnsweredAt:      answeredAt,
	}, nil
}

func DeleteMfgRecord(tenantID uint, resource string, id uint) error {
	spec, ok := mfgResource(resource)
	if !ok {
		return errors.New(MfgErrorUnsupportedResource)
	}
	item := spec.Model()
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, id).First(item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(MfgErrorNotFound)
		}
		return err
	}
	return utils.DB.Delete(item).Error
}

func MfgRecordFileReferences(resource string, record interface{}) []MfgFileReference {
	spec, ok := mfgResource(resource)
	if !ok {
		return nil
	}
	refs := make([]MfgFileReference, 0, len(spec.FileFields))
	for _, field := range spec.FileFields {
		fileID := stringField(record, field.FileField)
		if fileID == "" {
			continue
		}
		refs = append(refs, MfgFileReference{
			FileID:        fileID,
			StorageNodeID: stringField(record, field.NodeField),
			Source:        resource,
		})
	}
	return dedupeMfgFileReferences(refs)
}

func RemovedMfgFileReferences(before []MfgFileReference, after []MfgFileReference) []MfgFileReference {
	afterSet := map[string]bool{}
	for _, ref := range dedupeMfgFileReferences(after) {
		afterSet[ref.FileID] = true
	}
	removed := make([]MfgFileReference, 0)
	for _, ref := range dedupeMfgFileReferences(before) {
		if ref.FileID != "" && !afterSet[ref.FileID] {
			removed = append(removed, ref)
		}
	}
	return dedupeMfgFileReferences(removed)
}

func SubmitMfgRFQ(tenantID uint, userID uint, id uint) (models.MfgRFQ, error) {
	rfq, err := loadMfgRFQ(tenantID, id)
	if err != nil {
		return models.MfgRFQ{}, err
	}
	rfq.Status = "submitted"
	rfq.UpdatedBy = userID
	return rfq, utils.DB.Save(&rfq).Error
}

func StartMfgRFQCosting(tenantID uint, userID uint, id uint) (MfgRFQCostingResult, error) {
	rfq, err := loadMfgRFQ(tenantID, id)
	if err != nil {
		return MfgRFQCostingResult{}, err
	}
	check := MfgRFQReadiness(rfq)
	if !check["ready"].(bool) {
		return MfgRFQCostingResult{}, errors.New(MfgErrorRFQMissingRequiredFields)
	}
	costSheet, err := ensurePendingCostSheetForRFQ(tenantID, userID, rfq)
	if err != nil {
		return MfgRFQCostingResult{}, err
	}
	rfq.Status = "costing"
	rfq.UpdatedBy = userID
	if err := utils.DB.Save(&rfq).Error; err != nil {
		return MfgRFQCostingResult{}, err
	}
	return MfgRFQCostingResult{RFQ: rfq, CostSheet: costSheet}, nil
}

func CheckMfgRFQReadiness(tenantID uint, id uint) (map[string]interface{}, error) {
	rfq, err := loadMfgRFQ(tenantID, id)
	if err != nil {
		return nil, err
	}
	return MfgRFQReadiness(rfq), nil
}

func splitMfgCurrentBOMSnapshot(items []models.MfgBOMSnapshot) (interface{}, []models.MfgBOMSnapshot) {
	history := make([]models.MfgBOMSnapshot, 0, len(items))
	var current *models.MfgBOMSnapshot
	for _, item := range items {
		if current == nil && strings.EqualFold(strings.TrimSpace(item.Status), "active") {
			copied := item
			current = &copied
			continue
		}
		history = append(history, item)
	}
	if current == nil && len(items) > 0 {
		copied := items[0]
		current = &copied
		history = append(history[:0], items[1:]...)
	}
	if current == nil {
		return nil, history
	}
	return *current, history
}

func MfgRFQQuoteWorkspace(tenantID uint, id uint, canViewSensitive bool) (map[string]interface{}, error) {
	return MfgRFQQuoteWorkspaceForAccess(tenantID, id, MfgAccessContext{IsAdmin: canViewSensitive, IsSupervisor: canViewSensitive})
}

func MfgRFQQuoteWorkspaceForAccess(tenantID uint, id uint, access MfgAccessContext) (map[string]interface{}, error) {
	rfq, err := loadMfgRFQ(tenantID, id)
	if err != nil {
		return nil, err
	}
	canViewPrice := canViewMfgRFQPrice(rfq, access)
	var clarifications []models.MfgRFQClarification
	if err := utils.DB.Where("tenant_id = ? AND rfq_id = ?", tenantID, id).Order("updated_at desc, id desc").Find(&clarifications).Error; err != nil {
		return nil, err
	}
	var drawings []models.MfgDrawing
	if err := utils.DB.Where("tenant_id = ? AND rfq_id = ?", tenantID, id).Order("updated_at desc, id desc").Find(&drawings).Error; err != nil {
		return nil, err
	}
	var bomSnapshots []models.MfgBOMSnapshot
	if err := utils.DB.Where("tenant_id = ? AND rfq_id = ?", tenantID, id).Order("updated_at desc, id desc").Find(&bomSnapshots).Error; err != nil {
		return nil, err
	}
	currentBOMSnapshot, bomSnapshotHistory := splitMfgCurrentBOMSnapshot(bomSnapshots)
	var costSheets []models.MfgCostSheet
	if err := utils.DB.Where("tenant_id = ? AND rfq_id = ?", tenantID, id).Order("updated_at desc, id desc").Find(&costSheets).Error; err != nil {
		return nil, err
	}
	var quotes []models.MfgQuoteRevision
	if err := utils.DB.Where("tenant_id = ? AND rfq_id = ?", tenantID, id).Order("updated_at desc, revision desc, id desc").Find(&quotes).Error; err != nil {
		return nil, err
	}
	readiness := MfgRFQReadiness(rfq)
	canManageRFQ := canManageMfgRFQRecord(rfq, access)
	canCreateBOM := canCreateMfgBOMForRFQ(rfq, access)
	canCreateCost := canCreateMfgCostForRFQ(rfq, access)
	canCreateQuote := canCreateMfgQuoteForRFQ(rfq, access)
	return map[string]interface{}{
		"rfq":                  sanitizeMfgRFQ(rfq, access),
		"readiness":            readiness,
		"next_action":          readiness["next_action"],
		"clarifications":       SanitizeMfgRecordsForAccess("rfq-clarifications", clarifications, access),
		"drawings":             drawings,
		"current_bom_snapshot": SanitizeMfgRecordsForAccess("bom-snapshots", currentBOMSnapshot, access),
		"bom_snapshot_history": SanitizeMfgRecordsForAccess("bom-snapshots", bomSnapshotHistory, access),
		"bom_snapshots":        SanitizeMfgRecordsForAccess("bom-snapshots", bomSnapshots, access),
		"cost_sheets":          SanitizeMfgRecordsForAccess("cost-sheets", costSheets, access),
		"quotes":               SanitizeMfgRecordsForAccess("quotes", quotes, access),
		"permissions": map[string]interface{}{
			"can_manage_rfq":     canManageRFQ,
			"can_manage_drawing": canManageRFQ,
			"can_manage_bom":     canCreateBOM,
			"can_manage_cost":    canCreateCost,
			"can_manage_quote":   canCreateQuote,
			"can_create_bom":     canCreateBOM,
			"can_create_cost":    canCreateCost,
			"can_create_quote":   canCreateQuote,
			"can_view_sensitive": canViewPrice,
			"can_view_price":     canViewPrice,
			"can_view_cost":      canViewPrice,
			"can_view_quote":     canViewPrice,
			"cost_fields_masked": !canViewPrice,
			"price_masked":       !canViewPrice,
		},
	}, nil
}

func MfgRFQReadiness(rfq models.MfgRFQ) map[string]interface{} {
	missing := []string{}
	if rfq.CustomerID == 0 {
		missing = append(missing, "customer")
	}
	if rfq.Quantity <= 0 {
		missing = append(missing, "quantity")
	}
	if rfq.ProductFamilyID == 0 && !mfgRFQHasProductDemand(rfq) {
		missing = append(missing, "product_demand")
	}
	if strings.TrimSpace(rfq.DrawingFileID) == "" && strings.TrimSpace(rfq.NoDrawingReason) == "" {
		missing = append(missing, "drawing_or_exemption")
	}
	if rfq.RequiredDeliveryDate == nil {
		missing = append(missing, "delivery_date")
	}
	return map[string]interface{}{
		"ready":       len(missing) == 0,
		"missing":     missing,
		"next_action": firstNonEmpty(readinessNextAction(missing), "start_costing"),
	}
}

func mfgRFQHasProductDemand(rfq models.MfgRFQ) bool {
	payload := jsonMap(rfq.PayloadJSON)
	productType := mfgPayloadString(payload, "product_type")
	if productType == "standard" {
		if uintFromMfgInput(payload["product_id"]) > 0 {
			return true
		}
		return mfgPayloadString(payload, "product_name") != "" || mfgPayloadString(payload, "sku") != ""
	}
	if productType == "custom" {
		return mfgPayloadString(payload, "custom_product_name") != "" || mfgPayloadString(payload, "specification") != ""
	}
	return uintFromMfgInput(payload["product_id"]) > 0 ||
		mfgPayloadString(payload, "product_name") != "" ||
		mfgPayloadString(payload, "custom_product_name") != "" ||
		mfgPayloadString(payload, "specification") != ""
}

func mfgPayloadString(payload map[string]interface{}, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func readinessNextAction(missing []string) string {
	if len(missing) == 0 {
		return "start_costing"
	}
	switch missing[0] {
	case "customer":
		return "complete_customer"
	case "quantity", "product_family", "product_demand":
		return "complete_demand"
	case "drawing_or_exemption":
		return "complete_drawing"
	case "delivery_date":
		return "complete_delivery"
	default:
		return "complete_demand"
	}
}

func ApplyMfgKnowledge(tenantID uint, userID uint, knowledgeID uint, input map[string]interface{}) (models.MfgKnowledgeUsage, error) {
	var entry models.MfgKnowledgeEntry
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, knowledgeID).First(&entry).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.MfgKnowledgeUsage{}, errors.New(MfgErrorNotFound)
		}
		return models.MfgKnowledgeUsage{}, err
	}
	targetType := strings.TrimSpace(fmt.Sprint(input["target_type"]))
	targetID := uintFromMfgInput(input["target_id"])
	if targetType == "" || targetID == 0 {
		return models.MfgKnowledgeUsage{}, errors.New(MfgErrorInvalidInput)
	}
	item := models.MfgKnowledgeUsage{
		TenantID:       tenantID,
		KnowledgeID:    knowledgeID,
		TargetType:     targetType,
		TargetID:       targetID,
		AppliedSummary: strings.TrimSpace(fmt.Sprint(input["applied_summary"])),
		CreatedBy:      userID,
	}
	if item.AppliedSummary == "" {
		item.AppliedSummary = entry.Title
	}
	return item, utils.DB.Create(&item).Error
}

func CreateMfgClarification(tenantID uint, userID uint, rfqID uint, input map[string]interface{}) (models.MfgRFQClarification, error) {
	if _, err := loadMfgRFQ(tenantID, rfqID); err != nil {
		return models.MfgRFQClarification{}, err
	}
	item := models.MfgRFQClarification{}
	if err := applyMfgInput(&item, input); err != nil {
		return models.MfgRFQClarification{}, err
	}
	item.TenantID = tenantID
	item.RFQID = rfqID
	item.Status = firstNonEmpty(item.Status, "open")
	item.CreatedBy = userID
	item.UpdatedBy = userID
	return item, utils.DB.Create(&item).Error
}

func ListMfgRFQClarificationAnswerHistories(tenantID uint, clarificationID uint) ([]models.MfgRFQClarificationAnswerHistory, error) {
	var clarification models.MfgRFQClarification
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, clarificationID).First(&clarification).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(MfgErrorNotFound)
		}
		return nil, err
	}
	var items []models.MfgRFQClarificationAnswerHistory
	if err := utils.DB.Where("tenant_id = ? AND clarification_id = ?", tenantID, clarificationID).
		Order("answered_at desc, id desc").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func enrichMfgRFQClarificationValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case []models.MfgRFQClarification:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, enrichMfgRFQClarification(item))
		}
		return items
	case models.MfgRFQClarification:
		return enrichMfgRFQClarification(typed)
	default:
		return value
	}
}

func enrichMfgRFQClarification(item models.MfgRFQClarification) map[string]interface{} {
	return map[string]interface{}{
		"id":                    item.ID,
		"created_at":            item.CreatedAt,
		"updated_at":            item.UpdatedAt,
		"tenant_id":             item.TenantID,
		"rfq_id":                item.RFQID,
		"question":              item.Question,
		"answer":                item.Answer,
		"owner_id":              item.OwnerID,
		"status":                item.Status,
		"source_type":           item.SourceType,
		"source_key":            item.SourceKey,
		"source_label":          item.SourceLabel,
		"sort_order":            item.SortOrder,
		"last_answered_by":      item.LastAnsweredBy,
		"last_answered_by_name": userDisplayName(item.TenantID, item.LastAnsweredBy),
		"last_answered_at":      item.LastAnsweredAt,
		"due_at":                item.DueAt,
		"resolved_at":           item.ResolvedAt,
		"created_by":            item.CreatedBy,
		"updated_by":            item.UpdatedBy,
	}
}

func syncRFQRequirementClarifications(tenantID uint, userID uint, record interface{}) error {
	rfq, ok := record.(models.MfgRFQ)
	if !ok {
		if ptr, ptrOK := record.(*models.MfgRFQ); ptrOK && ptr != nil {
			rfq = *ptr
			ok = true
		}
	}
	if !ok || rfq.ID == 0 {
		return nil
	}
	items := rfqClarificationItems(rfq.PayloadJSON)
	if len(items) == 0 {
		return nil
	}
	sourceKeys := make([]string, 0, len(items))
	for _, item := range items {
		if item.SourceKey != "" && item.NeedClarification && item.Content != "" {
			sourceKeys = append(sourceKeys, item.SourceKey)
		}
	}
	if len(sourceKeys) == 0 {
		return nil
	}
	var existing []models.MfgRFQClarification
	if err := utils.DB.Where("tenant_id = ? AND rfq_id = ? AND source_type = ? AND source_key IN ?", tenantID, rfq.ID, "rfq_requirement", sourceKeys).Find(&existing).Error; err != nil {
		return err
	}
	existingByKey := map[string]bool{}
	for _, item := range existing {
		existingByKey[strings.TrimSpace(item.SourceKey)] = true
	}
	for _, item := range items {
		if !item.NeedClarification || item.Content == "" || item.SourceKey == "" || existingByKey[item.SourceKey] {
			continue
		}
		question := item.Content
		if item.Remark != "" {
			question = question + "\n" + item.Remark
		}
		clarification := models.MfgRFQClarification{
			TenantID:    tenantID,
			RFQID:       rfq.ID,
			Question:    question,
			Status:      "open",
			SourceType:  "rfq_requirement",
			SourceKey:   item.SourceKey,
			SourceLabel: item.Label,
			SortOrder:   item.SortOrder,
			CreatedBy:   userID,
			UpdatedBy:   userID,
		}
		if err := utils.DB.Create(&clarification).Error; err != nil {
			return err
		}
		existingByKey[item.SourceKey] = true
	}
	return nil
}

type rfqClarificationItem struct {
	SourceKey         string
	Label             string
	Content           string
	Remark            string
	NeedClarification bool
	SortOrder         int
}

func rfqClarificationItems(payloadJSON string) []rfqClarificationItem {
	payload := jsonMap(payloadJSON)
	rawItems, _ := payload["clarification_items"].([]interface{})
	items := make([]rfqClarificationItem, 0, len(rawItems))
	for index, raw := range rawItems {
		source := mapFromMfgInput(raw)
		key := stringFromMfgInput(source["key"])
		content := stringFromMfgInput(source["content"])
		if content == "" {
			continue
		}
		label := firstNonEmpty(stringFromMfgInput(source["label"]), stringFromMfgInput(source["category"]))
		if key == "" {
			key = fmt.Sprintf("requirement_%d", index+1)
		}
		needClarification := true
		if rawNeed, exists := source["need_clarification"]; exists {
			needClarification = boolFromMfgInput(rawNeed)
		}
		sortOrder := int(floatFromMfgInput(source["sort_order"]))
		if sortOrder <= 0 {
			sortOrder = index + 1
		}
		items = append(items, rfqClarificationItem{
			SourceKey:         key,
			Label:             label,
			Content:           content,
			Remark:            stringFromMfgInput(source["remark"]),
			NeedClarification: needClarification,
			SortOrder:         sortOrder,
		})
	}
	return items
}

func ConfigureMfgCPQ(tenantID uint, input map[string]interface{}) (map[string]interface{}, error) {
	productFamilyID := uintFromMfgInput(input["product_family_id"])
	configModelID := uintFromMfgInput(input["config_model_id"])
	bomTemplateID := uintFromMfgInput(input["bom_template_id"])
	configValues := mapFromMfgInput(input["config_values"])
	var configModel models.MfgConfigModel
	if configModelID > 0 {
		if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, configModelID).First(&configModel).Error; err != nil {
			return nil, err
		}
		productFamilyID = firstUint(productFamilyID, configModel.ProductFamilyID)
	}
	var bomTemplate models.MfgBOMTemplate
	if bomTemplateID > 0 {
		if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, bomTemplateID).First(&bomTemplate).Error; err != nil {
			return nil, err
		}
		productFamilyID = firstUint(productFamilyID, bomTemplate.ProductFamilyID)
	}
	errorsByField := validateConfigValues(configModel, configValues)
	return map[string]interface{}{
		"valid":             len(errorsByField) == 0,
		"errors":            errorsByField,
		"product_family_id": productFamilyID,
		"config_model":      configModel,
		"bom_template":      bomTemplate,
		"config_values":     configValues,
		"bom_preview":       jsonMap(bomTemplate.BOMLinesJSON),
	}, nil
}

func CreateMfgBOMSnapshotFromConfig(tenantID uint, userID uint, input MfgBOMSnapshotInput) (models.MfgBOMSnapshot, error) {
	configPayload := map[string]interface{}{
		"product_family_id": input.ProductFamilyID,
		"config_model_id":   input.ConfigModelID,
		"bom_template_id":   input.BOMTemplateID,
		"config_values":     input.ConfigValues,
	}
	result, err := ConfigureMfgCPQ(tenantID, configPayload)
	if err != nil {
		return models.MfgBOMSnapshot{}, err
	}
	if result["valid"] == false {
		return models.MfgBOMSnapshot{}, errors.New(MfgErrorInvalidInput)
	}
	var template models.MfgBOMTemplate
	if input.BOMTemplateID > 0 {
		if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, input.BOMTemplateID).First(&template).Error; err != nil {
			return models.MfgBOMSnapshot{}, err
		}
	}
	bomLinesJSON := firstNonEmpty(template.BOMLinesJSON, jsonStringFromAny(result["bom_preview"]))
	if input.BOMLines != nil {
		if manualLines := jsonStringFromAny(input.BOMLines); manualLines != "{}" && manualLines != "[]" {
			bomLinesJSON = manualLines
		}
	}
	productFamilyID := uintFromMfgInput(result["product_family_id"])
	if productFamilyID == 0 && input.RFQID > 0 {
		if rfq, err := loadMfgRFQ(tenantID, input.RFQID); err == nil {
			productFamilyID = rfq.ProductFamilyID
		}
	}
	sourceTemplateID := input.BOMTemplateID
	sourceConfigVersion := firstNonEmpty(input.SourceConfigVersion, template.Version)
	var item models.MfgBOMSnapshot
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if input.RFQID > 0 {
			if err := tx.Model(&models.MfgBOMSnapshot{}).
				Where("tenant_id = ? AND rfq_id = ? AND LOWER(TRIM(status)) = ?", tenantID, input.RFQID, "active").
				Updates(map[string]interface{}{"status": "superseded", "updated_by": userID}).Error; err != nil {
				return err
			}
		}
		if input.SaveAsTemplate {
			templateVersion := firstNonEmpty(input.TemplateVersion, sourceConfigVersion, "v1")
			newTemplate := models.MfgBOMTemplate{
				TenantID:        tenantID,
				ProductFamilyID: productFamilyID,
				TemplateNo:      strings.TrimSpace(input.TemplateNo),
				Version:         templateVersion,
				BOMLinesJSON:    bomLinesJSON,
				RuleJSON:        jsonStringFromAny(input.TemplateRule),
				Status:          firstNonEmpty(input.TemplateStatus, "draft"),
				CreatedBy:       userID,
				UpdatedBy:       userID,
			}
			setDefaultNo(&newTemplate, "TemplateNo", "BOMT")
			if err := validateMfgRecordForSave(tenantID, "bom-templates", &newTemplate); err != nil {
				return err
			}
			if err := tx.Create(&newTemplate).Error; err != nil {
				return err
			}
			sourceTemplateID = newTemplate.ID
			sourceConfigVersion = templateVersion
		}
		item = models.MfgBOMSnapshot{
			TenantID:            tenantID,
			SnapshotNo:          nextMfgNo("BOMS"),
			BusinessName:        strings.TrimSpace(input.BusinessName),
			SourceTemplateID:    sourceTemplateID,
			SourceConfigVersion: sourceConfigVersion,
			RFQID:               input.RFQID,
			ConfigValuesJSON:    jsonStringFromAny(input.ConfigValues),
			BOMLinesJSON:        bomLinesJSON,
			Status:              "active",
			CreatedBy:           userID,
			UpdatedBy:           userID,
		}
		return tx.Create(&item).Error
	})
	return item, err
}

func currentMfgBOMSnapshotForRFQ(tenantID uint, rfqID uint) (models.MfgBOMSnapshot, error) {
	var snapshot models.MfgBOMSnapshot
	if rfqID == 0 {
		return snapshot, errors.New(MfgErrorBOMSnapshotRequired)
	}
	err := utils.DB.Where("tenant_id = ? AND rfq_id = ? AND LOWER(TRIM(status)) = ?", tenantID, rfqID, "active").
		Order("updated_at desc, id desc").
		First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return snapshot, errors.New(MfgErrorCurrentBOMRequired)
	}
	return snapshot, err
}

func ensureCurrentMfgBOMSnapshot(tenantID uint, rfqID uint, bomSnapshotID uint) (models.MfgBOMSnapshot, error) {
	current, err := currentMfgBOMSnapshotForRFQ(tenantID, rfqID)
	if err != nil {
		return models.MfgBOMSnapshot{}, err
	}
	if current.ID != bomSnapshotID {
		return models.MfgBOMSnapshot{}, errors.New(MfgErrorCurrentBOMRequired)
	}
	return current, nil
}

func applyMfgCostSheetCalculation(item *models.MfgCostSheet, input map[string]interface{}, tenantID uint) {
	totalCost := item.MaterialCost + item.LaborCost + item.OutsourcingCost + item.ManufacturingOverhead
	if _, ok := input["quote_amount"]; ok {
		item.Amount = floatFromMfgInput(input["quote_amount"])
	}
	item.GrossMargin = item.Amount - totalCost
	if item.Amount > 0 {
		item.GrossMarginRate = item.GrossMargin / item.Amount
	} else {
		item.GrossMarginRate = 0
	}
	item.MarginLevel = marginLevelForTenant(tenantID, item.GrossMarginRate)
}

func CalculateMfgCostSheet(tenantID uint, userID uint, input map[string]interface{}) (models.MfgCostSheet, error) {
	rfqID := uintFromMfgInput(input["rfq_id"])
	bomSnapshotID := uintFromMfgInput(input["bom_snapshot_id"])
	if rfqID == 0 {
		return models.MfgCostSheet{}, errors.New(MfgErrorInvalidInput)
	}
	if _, err := loadMfgRFQ(tenantID, rfqID); err != nil {
		return models.MfgCostSheet{}, err
	}
	if bomSnapshotID == 0 {
		return models.MfgCostSheet{}, errors.New(MfgErrorBOMSnapshotRequired)
	}
	if _, err := ensureCurrentMfgBOMSnapshot(tenantID, rfqID, bomSnapshotID); err != nil {
		return models.MfgCostSheet{}, err
	}
	item := models.MfgCostSheet{}
	err := utils.DB.Where("tenant_id = ? AND rfq_id = ? AND bom_snapshot_id = ? AND status <> ?", tenantID, rfqID, bomSnapshotID, "archived").
		Order("updated_at desc, id desc").
		First(&item).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.MfgCostSheet{}, err
	}
	isNew := errors.Is(err, gorm.ErrRecordNotFound)
	if isNew {
		item = models.MfgCostSheet{
			TenantID:      tenantID,
			RFQID:         rfqID,
			BOMSnapshotID: bomSnapshotID,
			CreatedBy:     userID,
		}
	}
	if err := applyMfgInput(&item, input); err != nil {
		return models.MfgCostSheet{}, err
	}
	item.TenantID = tenantID
	item.RFQID = rfqID
	item.BOMSnapshotID = bomSnapshotID
	applyMfgCostSheetCalculation(&item, input, tenantID)
	item.Status = firstNonEmpty(item.Status, "calculated")
	item.SensitiveCostJSON = jsonStringFromAny(firstAny(input["sensitive_cost"], input["sensitive_cost_json"]))
	item.UpdatedBy = userID
	if isNew {
		return item, utils.DB.Create(&item).Error
	}
	return item, utils.DB.Save(&item).Error
}

func CompleteMfgCostSheet(tenantID uint, userID uint, id uint, input map[string]interface{}) (models.MfgCostSheet, error) {
	var item models.MfgCostSheet
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.MfgCostSheet{}, errors.New(MfgErrorNotFound)
		}
		return models.MfgCostSheet{}, err
	}
	if err := applyMfgInput(&item, input); err != nil {
		return models.MfgCostSheet{}, err
	}
	if item.BOMSnapshotID == 0 {
		item.BOMSnapshotID = uintFromMfgInput(input["bom_snapshot_id"])
	}
	if item.RFQID == 0 || item.BOMSnapshotID == 0 {
		return models.MfgCostSheet{}, errors.New(MfgErrorBOMSnapshotRequired)
	}
	if _, err := ensureCurrentMfgBOMSnapshot(tenantID, item.RFQID, item.BOMSnapshotID); err != nil {
		return models.MfgCostSheet{}, err
	}
	applyMfgCostSheetCalculation(&item, input, tenantID)
	item.Status = "calculated"
	item.SensitiveCostJSON = jsonStringFromAny(firstAny(input["sensitive_cost"], input["sensitive_cost_json"], jsonMap(item.SensitiveCostJSON)))
	item.UpdatedBy = userID
	if err := utils.DB.Save(&item).Error; err != nil {
		return models.MfgCostSheet{}, err
	}
	if item.RFQID > 0 {
		_ = utils.DB.Model(&models.MfgRFQ{}).Where("tenant_id = ? AND id = ?", tenantID, item.RFQID).Updates(map[string]interface{}{"status": "costed", "updated_by": userID}).Error
	}
	return item, nil
}

func ensurePendingCostSheetForRFQ(tenantID uint, userID uint, rfq models.MfgRFQ) (models.MfgCostSheet, error) {
	var existing models.MfgCostSheet
	err := utils.DB.Where("tenant_id = ? AND rfq_id = ?", tenantID, rfq.ID).Order("updated_at desc, id desc").First(&existing).Error
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.MfgCostSheet{}, err
	}
	var bomSnapshot models.MfgBOMSnapshot
	bomSnapshotID := uint(0)
	err = utils.DB.Where("tenant_id = ? AND rfq_id = ? AND LOWER(TRIM(status)) = ?", tenantID, rfq.ID, "active").Order("updated_at desc, id desc").First(&bomSnapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = utils.DB.Where("tenant_id = ? AND rfq_id = ?", tenantID, rfq.ID).Order("updated_at desc, id desc").First(&bomSnapshot).Error
	}
	if err == nil {
		bomSnapshotID = bomSnapshot.ID
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.MfgCostSheet{}, err
	}
	item := buildPendingCostSheetForRFQ(tenantID, userID, rfq, bomSnapshotID)
	if err := utils.DB.Create(&item).Error; err != nil {
		return models.MfgCostSheet{}, err
	}
	return item, nil
}

func buildPendingCostSheetForRFQ(tenantID uint, userID uint, rfq models.MfgRFQ, bomSnapshotID uint) models.MfgCostSheet {
	return models.MfgCostSheet{
		TenantID:      tenantID,
		RFQID:         rfq.ID,
		BOMSnapshotID: bomSnapshotID,
		Amount:        rfq.TargetPrice,
		Status:        "pending",
		MarginLevel:   "pending",
		SensitiveCostJSON: jsonStringFromAny(map[string]interface{}{
			"source":            "rfq_start_costing",
			"rfq_no":            rfq.RFQNo,
			"rfq_payload":       jsonMap(rfq.PayloadJSON),
			"bom_missing":       bomSnapshotID == 0,
			"next_action":       firstNonEmpty(costingNextAction(bomSnapshotID == 0), "calculate_cost"),
			"assigned_role":     "pricing_specialist",
			"technical_role":    "sales_engineer",
			"risk_flags":        costingRiskFlags(rfq, bomSnapshotID == 0),
			"target_price":      rfq.TargetPrice,
			"required_delivery": rfq.RequiredDeliveryDate,
		}),
		CreatedBy: userID,
		UpdatedBy: userID,
	}
}

func costingNextAction(bomMissing bool) string {
	if bomMissing {
		return "create_bom_snapshot"
	}
	return "calculate_cost"
}

func costingRiskFlags(rfq models.MfgRFQ, bomMissing bool) []string {
	flags := []string{}
	if bomMissing {
		flags = append(flags, "bom_missing")
	}
	if strings.TrimSpace(rfq.DrawingFileID) == "" {
		flags = append(flags, "drawing_exempted")
	}
	if rfq.TargetPrice <= 0 {
		flags = append(flags, "target_price_missing")
	}
	return flags
}

func CreateMfgQuote(tenantID uint, userID uint, input map[string]interface{}) (models.MfgQuoteRevision, error) {
	item := models.MfgQuoteRevision{}
	if err := applyMfgInput(&item, input); err != nil {
		return models.MfgQuoteRevision{}, err
	}
	if item.BOMSnapshotID == 0 {
		return models.MfgQuoteRevision{}, errors.New(MfgErrorBOMSnapshotRequired)
	}
	if item.RFQID == 0 {
		var cost models.MfgCostSheet
		if item.CostSheetID == 0 {
			return models.MfgQuoteRevision{}, errors.New(MfgErrorInvalidInput)
		}
		if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, item.CostSheetID).First(&cost).Error; err != nil {
			return models.MfgQuoteRevision{}, err
		}
		item.RFQID = cost.RFQID
		if cost.BOMSnapshotID != 0 && cost.BOMSnapshotID != item.BOMSnapshotID {
			return models.MfgQuoteRevision{}, errors.New(MfgErrorCurrentBOMRequired)
		}
	}
	if _, err := ensureCurrentMfgBOMSnapshot(tenantID, item.RFQID, item.BOMSnapshotID); err != nil {
		return models.MfgQuoteRevision{}, err
	}
	item.TenantID = tenantID
	item.QuoteNo = firstNonEmpty(item.QuoteNo, nextMfgNo("MQ"))
	item.Revision = nextMfgQuoteRevision(tenantID, item.RFQID)
	item.Currency = firstNonEmpty(item.Currency, "CNY")
	item.ApprovalStatus = "draft"
	item.SendStatus = firstNonEmpty(item.SendStatus, "not_sent")
	item.Status = firstNonEmpty(item.Status, "draft")
	item.TermsJSON = jsonStringFromAny(firstAny(input["terms"], input["terms_json"]))
	item.CreatedBy = userID
	item.UpdatedBy = userID
	return item, utils.DB.Create(&item).Error
}

func nextMfgQuoteRevision(tenantID uint, rfqID uint) int {
	if rfqID == 0 {
		return 1
	}
	var maxRevision int
	if err := utils.DB.Model(&models.MfgQuoteRevision{}).
		Where("tenant_id = ? AND rfq_id = ?", tenantID, rfqID).
		Select("COALESCE(MAX(revision), 0)").
		Scan(&maxRevision).Error; err != nil {
		return 1
	}
	return maxInt(maxRevision+1, 1)
}

func CreateMfgQuoteRevision(tenantID uint, userID uint, id uint) (models.MfgQuoteRevision, error) {
	source, err := loadMfgQuote(tenantID, id)
	if err != nil {
		return models.MfgQuoteRevision{}, err
	}
	var maxRevision int
	if err := utils.DB.Model(&models.MfgQuoteRevision{}).
		Where("tenant_id = ? AND rfq_id = ?", tenantID, source.RFQID).
		Select("COALESCE(MAX(revision), 0)").
		Scan(&maxRevision).Error; err != nil {
		return models.MfgQuoteRevision{}, err
	}
	item := models.MfgQuoteRevision{
		TenantID:       tenantID,
		QuoteNo:        source.QuoteNo,
		Revision:       maxInt(maxRevision+1, source.Revision+1),
		RFQID:          source.RFQID,
		CostSheetID:    source.CostSheetID,
		BOMSnapshotID:  source.BOMSnapshotID,
		Amount:         source.Amount,
		Currency:       firstNonEmpty(source.Currency, "CNY"),
		TermsJSON:      source.TermsJSON,
		ApprovalStatus: "draft",
		SendStatus:     "not_sent",
		Status:         "draft",
		CreatedBy:      userID,
		UpdatedBy:      userID,
	}
	return item, utils.DB.Create(&item).Error
}

func ChangeMfgQuoteApproval(tenantID uint, userID uint, id uint, status string) (models.MfgQuoteRevision, error) {
	return ChangeMfgQuoteApprovalWithRemark(tenantID, userID, id, status, "")
}

func ChangeMfgQuoteApprovalWithRemark(tenantID uint, userID uint, id uint, status string, remark string) (models.MfgQuoteRevision, error) {
	quote, err := loadMfgQuote(tenantID, id)
	if err != nil {
		return models.MfgQuoteRevision{}, err
	}
	status = strings.ToLower(strings.TrimSpace(status))
	remark = strings.TrimSpace(remark)
	if (status == "approved" || status == "rejected") && remark == "" {
		return models.MfgQuoteRevision{}, errors.New(MfgErrorApprovalRemarkRequired)
	}
	now := time.Now()
	oldApprovalStatus := quote.ApprovalStatus
	quote.ApprovalStatus = status
	quote.UpdatedBy = userID
	if status == "approved" {
		quote.Status = "approved"
		quote.ApprovedBy = userID
		quote.ApprovedAt = &now
	} else if status == "rejected" {
		quote.Status = "rejected"
		quote.RejectedBy = userID
		quote.RejectedAt = &now
	}
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&quote).Error; err != nil {
			return err
		}
		history := models.MfgQuoteApprovalHistory{
			TenantID:          quote.TenantID,
			RFQID:             quote.RFQID,
			QuoteID:           quote.ID,
			QuoteNo:           quote.QuoteNo,
			Revision:          quote.Revision,
			Action:            quoteApprovalAction(status),
			OldApprovalStatus: oldApprovalStatus,
			NewApprovalStatus: quote.ApprovalStatus,
			Remark:            remark,
			OperatorID:        userID,
			OperatorName:      firstNonEmpty(userDisplayName(tenantID, userID), fmt.Sprintf("#%d", userID)),
			OperatedAt:        now,
		}
		return tx.Create(&history).Error
	})
	return quote, err
}

func quoteApprovalAction(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "submit"
	case "approved":
		return "approve"
	case "rejected":
		return "reject"
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func SendMfgQuote(tenantID uint, userID uint, id uint) (models.MfgQuoteRevision, error) {
	quote, err := loadMfgQuote(tenantID, id)
	if err != nil {
		return models.MfgQuoteRevision{}, err
	}
	if quote.ApprovalStatus != "approved" {
		return models.MfgQuoteRevision{}, errors.New(MfgErrorInvalidState)
	}
	if quote.CostSheetID > 0 {
		var cost models.MfgCostSheet
		if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, quote.CostSheetID).First(&cost).Error; err == nil {
			if cost.MarginLevel == "low" && quote.ApprovalStatus != "approved" {
				return models.MfgQuoteRevision{}, errors.New(MfgErrorLowMarginApprovalRequired)
			}
		}
	}
	quote.SendStatus = "sent"
	quote.Status = "sent"
	quote.UpdatedBy = userID
	return quote, utils.DB.Save(&quote).Error
}

func TrackMfgQuoteCustomerView(tenantID uint, id uint) (models.MfgQuoteRevision, error) {
	quote, err := loadMfgQuote(tenantID, id)
	if err != nil {
		return models.MfgQuoteRevision{}, err
	}
	now := time.Now()
	quote.CustomerViewedAt = &now
	quote.CustomerViewCount++
	return quote, utils.DB.Save(&quote).Error
}

func ConfirmMfgQuoteByCustomer(tenantID uint, userID uint, id uint) (models.MfgQuoteRevision, error) {
	quote, err := loadMfgQuote(tenantID, id)
	if err != nil {
		return models.MfgQuoteRevision{}, err
	}
	if quote.SendStatus != "sent" {
		return models.MfgQuoteRevision{}, errors.New(MfgErrorInvalidState)
	}
	quote.Status = "confirmed"
	quote.UpdatedBy = userID
	return quote, utils.DB.Save(&quote).Error
}

func MfgQuoteCustomerView(tenantID uint, id uint) (map[string]interface{}, error) {
	quote, err := TrackMfgQuoteCustomerView(tenantID, id)
	if err != nil {
		return nil, err
	}
	return sanitizeMfgQuoteForCustomer(quote), nil
}

func MfgQuoteDiff(tenantID uint, id uint) (map[string]interface{}, error) {
	quote, err := loadMfgQuote(tenantID, id)
	if err != nil {
		return nil, err
	}
	var previous models.MfgQuoteRevision
	_ = utils.DB.Where("tenant_id = ? AND rfq_id = ? AND revision < ?", tenantID, quote.RFQID, quote.Revision).
		Order("revision desc, id desc").
		First(&previous).Error
	previousChanged := previous.ID > 0
	return map[string]interface{}{
		"quote":    enrichMfgQuoteRevision(quote),
		"previous": enrichMfgQuoteRevision(previous),
		"diff": map[string]interface{}{
			"amount_delta":  quote.Amount - previous.Amount,
			"terms_changed": previousChanged && strings.TrimSpace(quote.TermsJSON) != strings.TrimSpace(previous.TermsJSON),
			"bom_changed":   previousChanged && quote.BOMSnapshotID != previous.BOMSnapshotID,
			"has_previous":  previousChanged,
		},
	}, nil
}

func MfgQuoteRevisions(tenantID uint, id uint) (map[string]interface{}, error) {
	quote, err := loadMfgQuote(tenantID, id)
	if err != nil {
		return nil, err
	}
	var items []models.MfgQuoteRevision
	if err := utils.DB.Where("tenant_id = ? AND rfq_id = ?", tenantID, quote.RFQID).
		Order("revision asc, id asc").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return map[string]interface{}{"quote_no": quote.QuoteNo, "rfq_id": quote.RFQID, "items": enrichMfgQuoteRevisionValue(items), "total": len(items)}, nil
}

func RecordMfgDrawingAccess(tenantID uint, userID uint, drawingID uint, action string) (map[string]interface{}, error) {
	var drawing models.MfgDrawing
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, drawingID).First(&drawing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(MfgErrorNotFound)
		}
		return nil, err
	}
	action = firstNonEmpty(strings.TrimSpace(action), "view")
	if err := validateMfgDrawingAccessPolicy(drawing, action); err != nil {
		return nil, err
	}
	audit := models.OperationLog{
		TenantModel:    models.TenantModel{TenantID: tenantID},
		UserID:         userID,
		Operation:      "MFG_DRAWING_" + strings.ToUpper(strings.ReplaceAll(action, "-", "_")),
		Target:         fmt.Sprintf("mfg_drawings[%d]", drawing.ID),
		DescriptionKey: "operation.manufacturing_drawing_access",
		ParamsJSON: jsonStringFromAny(map[string]interface{}{
			"drawing_id":           drawing.ID,
			"file_id":              drawing.FileID,
			"classification_level": drawing.ClassificationLevel,
			"watermark_required":   drawing.WatermarkRequired,
			"download_policy":      drawing.DownloadPolicy,
			"approval_status":      drawing.ApprovalStatus,
			"external_send_policy": drawing.ExternalSendPolicy,
			"action":               action,
		}),
		Result: "success",
		Source: "web",
	}
	if err := utils.DB.Create(&audit).Error; err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"drawing":              drawing,
		"action":               action,
		"audit_log_id":         audit.ID,
		"watermark_required":   drawing.WatermarkRequired,
		"download_policy":      drawing.DownloadPolicy,
		"approval_status":      drawing.ApprovalStatus,
		"external_send_policy": drawing.ExternalSendPolicy,
		"classification_level": drawing.ClassificationLevel,
	}, nil
}

func ApproveMfgDrawingAccess(tenantID uint, userID uint, drawingID uint) (models.MfgDrawing, error) {
	var drawing models.MfgDrawing
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, drawingID).First(&drawing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return drawing, errors.New(MfgErrorNotFound)
		}
		return drawing, err
	}
	now := time.Now()
	drawing.ApprovalStatus = "approved"
	drawing.ApprovedBy = userID
	drawing.ApprovedAt = &now
	drawing.UpdatedBy = userID
	return drawing, utils.DB.Save(&drawing).Error
}

func CreateMfgOrderFromQuote(tenantID uint, userID uint, input MfgOrderCreateInput) (map[string]interface{}, error) {
	if input.QuoteID == 0 {
		return nil, errors.New(MfgErrorInvalidState)
	}
	quote, err := loadMfgQuote(tenantID, input.QuoteID)
	if err != nil {
		return nil, err
	}
	if quote.BOMSnapshotID == 0 {
		return nil, errors.New(MfgErrorBOMSnapshotRequired)
	}
	if quote.Status != "confirmed" {
		return nil, errors.New(MfgErrorInvalidState)
	}
	var order models.CRMObject
	reused := false
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if input.OrderID > 0 {
			if err := tx.Where("tenant_id = ? AND id = ? AND domain = ? AND object_type = ?", tenantID, input.OrderID, "orders", "order").First(&order).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New(MfgErrorNotFound)
				}
				return err
			}
			if mfgOrderBOMLocked(order) && uintFromPayload(order.PayloadJSON, "mfg_bom_snapshot_id") != quote.BOMSnapshotID {
				return errors.New(MfgErrorOrderBOMLocked)
			}
			reused = true
		} else {
			order = models.CRMObject{
				TenantModel: models.TenantModel{TenantID: tenantID},
				Domain:      "orders",
				ObjectType:  "order",
				Name:        firstNonEmpty(input.Name, quote.QuoteNo+" 制造订单"),
				Code:        nextMfgNo("ORD"),
				Status:      firstNonEmpty(input.Status, "confirmed"),
				CompanyID:   firstUint(input.CustomerID, mfgQuoteCustomerID(tenantID, quote.RFQID)),
				ContactID:   input.ContactID,
				Amount:      quote.Amount,
				Currency:    quote.Currency,
				CreatedBy:   userID,
				UpdatedBy:   userID,
			}
		}
		payload := payloadFromJSON(order.PayloadJSON)
		for key, value := range input.Payload {
			payload[key] = value
		}
		payload["mfg_quote_revision_id"] = quote.ID
		payload["mfg_quote_no"] = quote.QuoteNo
		payload["mfg_bom_snapshot_id"] = quote.BOMSnapshotID
		payload["mfg_bom_locked"] = true
		payload["mfg_bom_locked_at"] = time.Now().Format(time.RFC3339)
		payload["mfg_order_status"] = "bom_locked"
		order.PayloadJSON = jsonStringFromAny(payload)
		order.UpdatedBy = userID
		if order.ID == 0 {
			if err := tx.Create(&order).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&order).Error; err != nil {
			return err
		}
		snapshot := models.MfgBOMSnapshot{}
		if err := tx.Where("tenant_id = ? AND id = ?", tenantID, quote.BOMSnapshotID).First(&snapshot).Error; err != nil {
			return err
		}
		snapshot.OrderID = order.ID
		snapshot.QuoteID = quote.ID
		if err := tx.Save(&snapshot).Error; err != nil {
			return err
		}
		relation := mfgCRMRelation(tenantID, "mfg_quotes", quote.ID, "orders", order.ID, "mfg_quote_generated_order")
		if err := tx.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND target_type = ? AND target_id = ? AND relation_type = ?", tenantID, relation.SourceType, relation.SourceID, relation.TargetType, relation.TargetID, relation.RelationType).FirstOrCreate(&relation).Error; err != nil {
			return err
		}
		if err := enqueueMfgIntegrationOutboxTx(tx, tenantID, userID, "OrderConfirmed", "erp", "orders", order.ID, map[string]interface{}{
			"order_id":             order.ID,
			"quote_revision_id":    quote.ID,
			"quote_no":             quote.QuoteNo,
			"bom_snapshot_id":      quote.BOMSnapshotID,
			"customer_id":          order.CompanyID,
			"amount":               order.Amount,
			"currency":             order.Currency,
			"integration_status":   "not_configured",
			"compensation_state":   "pending",
			"minimum_payload_only": true,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"quote": quote, "order": order, "reused": reused, "bom_locked": true}, nil
}

func MfgOrderPrefillFromQuote(tenantID uint, quoteID uint) (map[string]interface{}, error) {
	if quoteID == 0 {
		return nil, errors.New(MfgErrorInvalidState)
	}
	quote, err := loadMfgQuote(tenantID, quoteID)
	if err != nil {
		return nil, err
	}
	if quote.BOMSnapshotID == 0 {
		return nil, errors.New(MfgErrorBOMSnapshotRequired)
	}
	rfq, err := loadMfgRFQ(tenantID, quote.RFQID)
	if err != nil {
		return nil, err
	}
	var snapshot models.MfgBOMSnapshot
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, quote.BOMSnapshotID).First(&snapshot).Error; err != nil {
		return nil, err
	}
	rfqPayload := jsonMap(rfq.PayloadJSON)
	productID := uintFromMfgInput(rfqPayload["product_id"])
	quantity := rfq.Quantity
	if quantity <= 0 {
		quantity = 1
	}
	unitPrice := 0.0
	if quote.Amount > 0 && quantity > 0 {
		unitPrice = quote.Amount / quantity
	}
	productName := firstNonEmpty(
		stringFromMfgInput(rfqPayload["product_name"]),
		stringFromMfgInput(rfqPayload["custom_product_name"]),
		stringFromMfgInput(rfqPayload["sku"]),
		rfq.BusinessName,
		rfq.RFQNo,
	)
	line := map[string]interface{}{
		"product_id":          productID,
		"sku":                 stringFromMfgInput(rfqPayload["sku"]),
		"name":                productName,
		"product_name":        productName,
		"quantity":            quantity,
		"unit_price":          unitPrice,
		"price":               unitPrice,
		"tax_rate":            0,
		"mfg_rfq_id":          rfq.ID,
		"mfg_rfq_no":          rfq.RFQNo,
		"mfg_quote_id":        quote.ID,
		"mfg_quote_no":        quote.QuoteNo,
		"mfg_bom_snapshot":    snapshot.ID,
		"mfg_bom_snapshot_id": snapshot.ID,
	}
	payload := map[string]interface{}{
		"company_id":              rfq.CustomerID,
		"company":                 firstNonEmpty(stringFromMfgInput(rfqPayload["customer_name"]), fmt.Sprintf("#%d", rfq.CustomerID)),
		"company_name":            stringFromMfgInput(rfqPayload["customer_name"]),
		"contact_id":              rfq.ContactID,
		"contact_name":            stringFromMfgInput(rfqPayload["contact_name"]),
		"payment_status":          "pending",
		"payment_terms":           "Net 30",
		"shipping_address":        "",
		"priority":                "normal",
		"visibility":              "public",
		"line_items":              []map[string]interface{}{line},
		"mfg_rfq_id":              rfq.ID,
		"mfg_rfq_no":              rfq.RFQNo,
		"mfg_rfq_business_name":   rfq.BusinessName,
		"mfg_quote_revision_id":   quote.ID,
		"mfg_quote_no":            quote.QuoteNo,
		"mfg_bom_snapshot_id":     snapshot.ID,
		"mfg_bom_snapshot_no":     snapshot.SnapshotNo,
		"mfg_bom_business_name":   snapshot.BusinessName,
		"mfg_source_quote_status": quote.Status,
	}
	name := firstNonEmpty(
		rfq.BusinessName,
		stringFromMfgInput(rfqPayload["custom_product_name"]),
		stringFromMfgInput(rfqPayload["product_name"]),
		quote.QuoteNo,
	) + " 订单"
	form := map[string]interface{}{
		"object_type": "order",
		"name":        name,
		"code":        "",
		"status":      "draft",
		"company_id":  rfq.CustomerID,
		"contact_id":  rfq.ContactID,
		"amount":      quote.Amount,
		"currency":    quote.Currency,
		"score":       0,
		"tags":        []string{},
		"payload":     payload,
	}
	return map[string]interface{}{
		"form":         form,
		"quote":        enrichMfgQuoteRevisionForAccess(quote, MfgAccessContext{TenantID: tenantID, IsAdmin: true}),
		"rfq":          sanitizeMfgRFQ(rfq, MfgAccessContext{TenantID: tenantID, IsAdmin: true}),
		"bom_snapshot": enrichMfgBOMSnapshot(snapshot),
	}, nil
}

func CreateMfgAssetsFromShipment(tenantID uint, userID uint, input MfgShipmentAssetInput) (map[string]interface{}, error) {
	if input.OrderID == 0 || len(input.SerialNumbers) == 0 {
		return nil, errors.New(MfgErrorInvalidInput)
	}
	order, err := loadMfgOrder(tenantID, input.OrderID)
	if err != nil {
		return nil, err
	}
	payload := payloadFromJSON(order.PayloadJSON)
	bomSnapshotID := uintFromMap(payload, "mfg_bom_snapshot_id")
	warrantyMonths := input.WarrantyMonths
	if warrantyMonths <= 0 {
		warrantyMonths = 12
	}
	warrantyStart := time.Now()
	warrantyEnd := warrantyStart.AddDate(0, warrantyMonths, 0)
	created := []models.MfgAsset{}
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		for _, rawSN := range input.SerialNumbers {
			sn := strings.TrimSpace(rawSN)
			if sn == "" {
				continue
			}
			var existing int64
			if err := tx.Model(&models.MfgAsset{}).Where("tenant_id = ? AND serial_no = ?", tenantID, sn).Count(&existing).Error; err != nil {
				return err
			}
			if existing > 0 {
				return errors.New(MfgErrorDuplicateSerialNo)
			}
			asset := models.MfgAsset{
				TenantID:        tenantID,
				CustomerID:      order.CompanyID,
				OrderID:         order.ID,
				BOMSnapshotID:   bomSnapshotID,
				SerialNo:        sn,
				BatchNo:         stringFromMfgInput(firstAny(input.Payload["batch_no"], payload["shipment_no"])),
				WarrantyStart:   &warrantyStart,
				WarrantyEnd:     &warrantyEnd,
				InstallLocation: strings.TrimSpace(input.InstallLocation),
				Status:          "active",
				IoTAssetRef:     strings.TrimSpace(input.IoTAssetRefs[sn]),
				CreatedBy:       userID,
				UpdatedBy:       userID,
			}
			if asset.IoTAssetRef != "" {
				now := time.Now()
				asset.IoTAuthorized = true
				asset.IoTAuthorizedAt = &now
			}
			if err := tx.Create(&asset).Error; err != nil {
				return err
			}
			created = append(created, asset)
		}
		if len(created) == 0 {
			return errors.New(MfgErrorInvalidInput)
		}
		payload["mfg_asset_count"] = intFromAny(payload["mfg_asset_count"]) + len(created)
		payload["mfg_last_asset_created_at"] = time.Now().Format(time.RFC3339)
		order.PayloadJSON = jsonStringFromAny(payload)
		order.UpdatedBy = userID
		return tx.Save(&order).Error
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"order": order, "assets": created, "total": len(created)}, nil
}

func MfgAssetServiceHistory(tenantID uint, assetID uint) (map[string]interface{}, error) {
	var asset models.MfgAsset
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, assetID).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(MfgErrorNotFound)
		}
		return nil, err
	}
	var tickets []models.MfgServiceTicket
	if err := utils.DB.Where("tenant_id = ? AND asset_id = ?", tenantID, assetID).Order("updated_at desc, id desc").Find(&tickets).Error; err != nil {
		return nil, err
	}
	var quotes []models.MfgSparePartQuote
	if err := utils.DB.Where("tenant_id = ? AND asset_id = ?", tenantID, assetID).Order("updated_at desc, id desc").Find(&quotes).Error; err != nil {
		return nil, err
	}
	return map[string]interface{}{"asset": asset, "tickets": tickets, "spare_part_quotes": quotes}, nil
}

func MfgAssetCustomerView(tenantID uint, assetID uint) (map[string]interface{}, error) {
	var asset models.MfgAsset
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, assetID).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(MfgErrorNotFound)
		}
		return nil, err
	}
	var tickets []models.MfgServiceTicket
	if err := utils.DB.Where("tenant_id = ? AND asset_id = ?", tenantID, assetID).
		Order("updated_at desc, id desc").
		Limit(20).
		Find(&tickets).Error; err != nil {
		return nil, err
	}
	var quotes []models.MfgSparePartQuote
	if err := utils.DB.Where("tenant_id = ? AND asset_id = ?", tenantID, assetID).
		Order("updated_at desc, id desc").
		Limit(20).
		Find(&quotes).Error; err != nil {
		return nil, err
	}
	sanitizedTickets := make([]map[string]interface{}, 0, len(tickets))
	for _, ticket := range tickets {
		sanitizedTickets = append(sanitizedTickets, sanitizeMfgServiceTicketForCustomer(ticket))
	}
	sanitizedQuotes := make([]map[string]interface{}, 0, len(quotes))
	for _, quote := range quotes {
		sanitizedQuotes = append(sanitizedQuotes, sanitizeMfgSparePartQuote(quote))
	}
	return map[string]interface{}{
		"asset":             sanitizeMfgAssetForCustomer(asset),
		"service_tickets":   sanitizedTickets,
		"spare_part_quotes": sanitizedQuotes,
		"customer_view":     true,
	}, nil
}

func CreateMfgSparePartQuoteForTicket(tenantID uint, userID uint, ticketID uint, input map[string]interface{}) (models.MfgSparePartQuote, error) {
	var ticket models.MfgServiceTicket
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, ticketID).First(&ticket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.MfgSparePartQuote{}, errors.New(MfgErrorNotFound)
		}
		return models.MfgSparePartQuote{}, err
	}
	item := models.MfgSparePartQuote{}
	if err := applyMfgInput(&item, input); err != nil {
		return models.MfgSparePartQuote{}, err
	}
	item.TenantID = tenantID
	item.ServiceTicketID = ticket.ID
	item.AssetID = firstUint(item.AssetID, ticket.AssetID)
	item.QuoteNo = firstNonEmpty(item.QuoteNo, nextMfgNo("SPQ"))
	item.LineItemsJSON = jsonStringFromAny(firstAny(input["line_items"], input["line_items_json"]))
	item.Status = firstNonEmpty(item.Status, "draft")
	item.CreatedBy = userID
	item.UpdatedBy = userID
	return item, utils.DB.Create(&item).Error
}

func CreateMfgSampleTestRecord(tenantID uint, userID uint, sampleProjectID uint, input map[string]interface{}) (models.MfgSampleTestRecord, error) {
	var sample models.MfgSampleProject
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, sampleProjectID).First(&sample).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.MfgSampleTestRecord{}, errors.New(MfgErrorNotFound)
		}
		return models.MfgSampleTestRecord{}, err
	}
	item := models.MfgSampleTestRecord{}
	if err := applyMfgInput(&item, input); err != nil {
		return models.MfgSampleTestRecord{}, err
	}
	item.TenantID = tenantID
	item.SampleProjectID = sampleProjectID
	item.CreatedBy = userID
	item.UpdatedBy = userID
	if item.TestResult == "failed" {
		sample.RiskLevel = "red"
		sample.UpdatedBy = userID
	}
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		if sample.RiskLevel == "red" {
			return tx.Save(&sample).Error
		}
		return nil
	})
	return item, err
}

func RaiseMfgProductionRisk(tenantID uint, userID uint, progressID uint, input map[string]interface{}) (models.MfgProductionProgress, error) {
	var item models.MfgProductionProgress
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, progressID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, errors.New(MfgErrorNotFound)
		}
		return item, err
	}
	before := item
	item.RiskLevel = firstNonEmpty(stringFromMfgInput(input["risk_level"]), "red")
	item.RiskReason = firstNonEmpty(stringFromMfgInput(input["risk_reason"]), stringFromMfgInput(input["reason"]), item.RiskReason)
	item.SyncStatus = firstNonEmpty(item.SyncStatus, "local")
	item.ProgressStatus = normalizeMfgProductionProgressStatus(firstNonEmpty(item.ProgressStatus, "in_progress"))
	item.UpdatedBy = userID
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if mfgProductionProgressChanged(before, item) {
			history := mfgProductionProgressHistoryFromChange(item, before.Milestone, before.ProgressStatus, userID, "risk_raised")
			if err := tx.Create(&history).Error; err != nil {
				return err
			}
		}
		if err := tx.Save(&item).Error; err != nil {
			return err
		}
		return enqueueMfgIntegrationOutboxTx(tx, tenantID, userID, "ProductionRiskRaised", "mes", "production-progress", item.ID, map[string]interface{}{
			"production_progress_id": item.ID,
			"order_id":               item.OrderID,
			"risk_level":             item.RiskLevel,
			"risk_reason":            item.RiskReason,
			"integration_status":     "not_configured",
			"compensation_state":     "pending",
		})
	})
	return item, err
}

func CreateMfgServiceTicketFromIoTAlarm(tenantID uint, userID uint, input MfgIoTAlarmInput) (map[string]interface{}, error) {
	asset, err := findMfgAssetForAlarm(tenantID, input.AssetID, input.IoTAssetRef)
	if err != nil {
		return nil, err
	}
	if !mfgAssetIoTAuthorized(asset) {
		return nil, errors.New(MfgErrorIoTUnauthorized)
	}
	alarmCode := firstNonEmpty(input.AlarmCode, "iot_alarm")
	severity := firstNonEmpty(input.Severity, "medium")
	var ticket models.MfgServiceTicket
	reused := false
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("tenant_id = ? AND asset_id = ? AND fault_code = ? AND status NOT IN ?", tenantID, asset.ID, alarmCode, []string{"closed", "resolved"}).
			Order("id desc").
			First(&ticket).Error
		if err == nil {
			reused = true
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		} else {
			ticket = models.MfgServiceTicket{
				TenantID:    tenantID,
				AssetID:     asset.ID,
				CustomerID:  asset.CustomerID,
				FaultCode:   alarmCode,
				ServiceType: "iot_alarm",
				Status:      "open",
				CreatedBy:   userID,
				UpdatedBy:   userID,
			}
			if err := tx.Create(&ticket).Error; err != nil {
				return err
			}
		}
		ticket.ServiceResult = firstNonEmpty(input.Message, ticket.ServiceResult)
		ticket.UpdatedBy = userID
		return tx.Save(&ticket).Error
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"asset": asset, "ticket": ticket, "reused": reused, "severity": severity}, nil
}

func UpdateMfgAssetIoTAuthorization(tenantID uint, userID uint, assetID uint, authorized bool, iotAssetRef string) (models.MfgAsset, error) {
	var asset models.MfgAsset
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, assetID).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return asset, errors.New(MfgErrorNotFound)
		}
		return asset, err
	}
	now := time.Now()
	if authorized {
		ref := firstNonEmpty(iotAssetRef, asset.IoTAssetRef)
		if strings.TrimSpace(ref) == "" {
			return asset, errors.New(MfgErrorIoTUnauthorized)
		}
		asset.IoTAssetRef = strings.TrimSpace(ref)
		asset.IoTAuthorized = true
		asset.IoTAuthorizedAt = &now
		asset.IoTRevokedAt = nil
	} else {
		asset.IoTAuthorized = false
		asset.IoTRevokedAt = &now
	}
	asset.UpdatedBy = userID
	return asset, utils.DB.Save(&asset).Error
}

func CreateMfgOpportunityFromServiceTicket(tenantID uint, userID uint, ticketID uint, input MfgServiceOpportunityInput) (map[string]interface{}, error) {
	var ticket models.MfgServiceTicket
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, ticketID).First(&ticket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(MfgErrorNotFound)
		}
		return nil, err
	}
	opportunity := models.CRMObject{
		TenantModel: models.TenantModel{TenantID: tenantID},
		Domain:      "opportunities",
		ObjectType:  "opportunity",
		Name:        firstNonEmpty(input.Name, fmt.Sprintf("售后转商机-%d", ticket.ID)),
		Code:        nextMfgNo("OPP"),
		Status:      "initial_contact",
		CompanyID:   ticket.CustomerID,
		Amount:      input.Amount,
		Currency:    "CNY",
		CreatedBy:   userID,
		UpdatedBy:   userID,
		PayloadJSON: jsonStringFromAny(map[string]interface{}{
			"stage":                 "initial_contact",
			"probability":           10,
			"growth_source":         "mfg_service",
			"source_mfg_ticket_id":  ticket.ID,
			"source_mfg_asset_id":   ticket.AssetID,
			"service_upsell_reason": strings.TrimSpace(input.Reason),
		}),
	}
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&opportunity).Error; err != nil {
			return err
		}
		ticket.ServiceToSalesFlag = true
		ticket.OpportunityID = opportunity.ID
		ticket.UpdatedBy = userID
		if err := tx.Save(&ticket).Error; err != nil {
			return err
		}
		relation := mfgCRMRelation(tenantID, "mfg_service_tickets", ticket.ID, "opportunities", opportunity.ID, "mfg_service_upsell")
		return tx.Create(&relation).Error
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"ticket": ticket, "opportunity": opportunity}, nil
}

func RefreshMfgMetricSnapshots(tenantID uint, userID uint) (map[string]interface{}, error) {
	today := time.Now().Truncate(24 * time.Hour)
	var rfqCount int64
	var quoteAmount float64
	var lowMarginCount int64
	var costCount int64
	var redCount int64
	var yellowCount int64
	var greenCount int64
	var ticketCount int64
	var serviceToOpportunity int64
	_ = utils.DB.Model(&models.MfgRFQ{}).Where("tenant_id = ?", tenantID).Count(&rfqCount).Error
	_ = utils.DB.Model(&models.MfgQuoteRevision{}).Where("tenant_id = ?", tenantID).Select("COALESCE(SUM(amount), 0)").Scan(&quoteAmount).Error
	_ = utils.DB.Model(&models.MfgCostSheet{}).Where("tenant_id = ?", tenantID).Count(&costCount).Error
	_ = utils.DB.Model(&models.MfgCostSheet{}).Where("tenant_id = ? AND margin_level = ?", tenantID, "low").Count(&lowMarginCount).Error
	_ = utils.DB.Model(&models.MfgProductionProgress{}).Where("tenant_id = ? AND risk_level = ?", tenantID, "red").Count(&redCount).Error
	_ = utils.DB.Model(&models.MfgProductionProgress{}).Where("tenant_id = ? AND risk_level = ?", tenantID, "yellow").Count(&yellowCount).Error
	_ = utils.DB.Model(&models.MfgProductionProgress{}).Where("tenant_id = ? AND (risk_level = ? OR risk_level = '')", tenantID, "green").Count(&greenCount).Error
	_ = utils.DB.Model(&models.MfgServiceTicket{}).Where("tenant_id = ?", tenantID).Count(&ticketCount).Error
	_ = utils.DB.Model(&models.MfgServiceTicket{}).Where("tenant_id = ? AND service_to_sales_flag = ?", tenantID, true).Count(&serviceToOpportunity).Error
	dataSource := jsonStringFromAny(map[string]interface{}{
		"mode":         "local_snapshot",
		"generated":    time.Now().Format(time.RFC3339),
		"source":       []string{"mfg_rfqs", "mfg_quote_revisions", "mfg_cost_sheets", "mfg_production_progress", "mfg_service_tickets"},
		"external":     "not_configured",
		"refreshed_by": userID,
	})
	quoteSnapshot := models.MetricMfgQuoteSnapshot{
		TenantID:           tenantID,
		StatDate:           today,
		RFQCount:           int(rfqCount),
		QuoteAmount:        quoteAmount,
		LowMarginQuoteRate: safeRatio(lowMarginCount, costCount),
		RefreshStatus:      "generated",
		DataSourceJSON:     dataSource,
		RefreshError:       "",
	}
	deliverySnapshot := models.MetricMfgDeliverySnapshot{
		TenantID:       tenantID,
		StatDate:       today,
		GreenCount:     int(greenCount),
		YellowCount:    int(yellowCount),
		RedCount:       int(redCount),
		RefreshStatus:  "generated",
		DataSourceJSON: dataSource,
		RefreshError:   "",
	}
	serviceSnapshot := models.MetricMfgServiceSnapshot{
		TenantID:                 tenantID,
		StatDate:                 today,
		TicketCount:              int(ticketCount),
		ServiceToOpportunityRate: safeRatio(serviceToOpportunity, ticketCount),
		RefreshStatus:            "generated",
		DataSourceJSON:           dataSource,
		RefreshError:             "",
	}
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND stat_date = ?", tenantID, today).Assign(quoteSnapshot).FirstOrCreate(&quoteSnapshot).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND stat_date = ?", tenantID, today).Assign(deliverySnapshot).FirstOrCreate(&deliverySnapshot).Error; err != nil {
			return err
		}
		return tx.Where("tenant_id = ? AND stat_date = ?", tenantID, today).Assign(serviceSnapshot).FirstOrCreate(&serviceSnapshot).Error
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"quote": quoteSnapshot, "delivery": deliverySnapshot, "service": serviceSnapshot, "refreshed_by": userID}, nil
}

func ListMfgIntegrationOutbox(tenantID uint, params MfgListParams) (map[string]interface{}, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Size <= 0 {
		params.Size = 20
	}
	if params.Size > 100 {
		params.Size = 100
	}
	query := utils.DB.Model(&models.MfgIntegrationOutbox{}).Where("tenant_id = ?", tenantID)
	if params.Status != "" {
		query = query.Where("status = ?", strings.TrimSpace(params.Status))
	}
	if params.Keyword != "" {
		keyword := "%" + strings.TrimSpace(params.Keyword) + "%"
		query = query.Where("(event_type LIKE ? OR target_system LIKE ? OR business_type LIKE ? OR idempotency_key LIKE ? OR last_error LIKE ?)", keyword, keyword, keyword, keyword, keyword)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []models.MfgIntegrationOutbox
	if err := query.Order("updated_at desc, id desc").Offset((params.Page - 1) * params.Size).Limit(params.Size).Find(&items).Error; err != nil {
		return nil, err
	}
	return map[string]interface{}{"items": items, "total": total}, nil
}

func RetryMfgIntegrationOutbox(tenantID uint, userID uint, id uint) (models.MfgIntegrationOutbox, error) {
	var item models.MfgIntegrationOutbox
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, errors.New(MfgErrorNotFound)
		}
		return item, err
	}
	now := time.Now()
	item.Status = "pending"
	item.CompensationState = "retrying"
	item.RetryCount++
	item.LastError = ""
	item.NextRetryAt = &now
	item.LastAttemptAt = &now
	item.UpdatedBy = userID
	return item, utils.DB.Save(&item).Error
}

func DeadLetterMfgIntegrationOutbox(tenantID uint, userID uint, id uint, reason string) (models.MfgIntegrationOutbox, error) {
	var item models.MfgIntegrationOutbox
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, errors.New(MfgErrorNotFound)
		}
		return item, err
	}
	now := time.Now()
	item.Status = "dead_letter"
	item.CompensationState = "manual_review"
	item.LastError = firstNonEmpty(strings.TrimSpace(reason), "manual dead letter")
	item.LastAttemptAt = &now
	item.UpdatedBy = userID
	return item, utils.DB.Save(&item).Error
}

func MfgMetricDrilldown(tenantID uint, metric string, params MfgListParams) (map[string]interface{}, error) {
	metric = strings.TrimSpace(metric)
	switch metric {
	case "rfq", "rfqs", "growth":
		items, total, err := ListMfgRecords(tenantID, "rfqs", params)
		return map[string]interface{}{"metric": metric, "target": "rfqs", "items": enrichMfgMetricDrilldownItems(tenantID, items), "total": total}, err
	case "quote", "quotes", "profit":
		items, total, err := ListMfgRecords(tenantID, "quotes", params)
		return map[string]interface{}{"metric": metric, "target": "quotes", "items": enrichMfgMetricDrilldownItems(tenantID, SanitizeMfgRecords("quotes", items, true)), "total": total}, err
	case "delivery", "production":
		items, total, err := ListMfgRecords(tenantID, "production-progress", params)
		return map[string]interface{}{"metric": metric, "target": "production-progress", "items": enrichMfgMetricDrilldownItems(tenantID, items), "total": total}, err
	case "asset", "assets", "lifecycle":
		items, total, err := ListMfgRecords(tenantID, "assets", params)
		return map[string]interface{}{"metric": metric, "target": "assets", "items": enrichMfgMetricDrilldownItems(tenantID, items), "total": total}, err
	case "service", "quality":
		items, total, err := ListMfgRecords(tenantID, "service-tickets", params)
		return map[string]interface{}{"metric": metric, "target": "service-tickets", "items": enrichMfgMetricDrilldownItems(tenantID, items), "total": total}, err
	case "ecn", "ecns":
		items, total, err := ListMfgRecords(tenantID, "ecns", params)
		return map[string]interface{}{"metric": metric, "target": "ecns", "items": enrichMfgMetricDrilldownItems(tenantID, items), "total": total}, err
	case "compensation", "compensation-records":
		items, total, err := ListMfgRecords(tenantID, "compensation-records", params)
		return map[string]interface{}{"metric": metric, "target": "compensation-records", "items": enrichMfgMetricDrilldownItems(tenantID, items), "total": total}, err
	case "cash", "customer":
		var items []models.MetricCustomerSnapshot
		query := utils.DB.Where("tenant_id = ?", tenantID).Order("snapshot_date desc, id desc")
		if params.CustomerID > 0 {
			query = query.Where("customer_id = ?", params.CustomerID)
		}
		if params.Page <= 0 {
			params.Page = 1
		}
		if params.Size <= 0 {
			params.Size = 20
		}
		var total int64
		if err := query.Model(&models.MetricCustomerSnapshot{}).Count(&total).Error; err != nil {
			return nil, err
		}
		if err := query.Offset((params.Page - 1) * params.Size).Limit(params.Size).Find(&items).Error; err != nil {
			return nil, err
		}
		return map[string]interface{}{"metric": metric, "target": "customer-metrics", "items": enrichMfgMetricDrilldownItems(tenantID, items), "total": total}, nil
	default:
		return nil, errors.New(MfgErrorUnsupportedResource)
	}
}

func enrichMfgMetricDrilldownItems(tenantID uint, value interface{}) interface{} {
	switch typed := value.(type) {
	case []map[string]interface{}:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, enrichMfgMetricDrilldownMap(tenantID, item))
		}
		return items
	case []models.MetricCustomerSnapshot:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			payload := structToMap(item)
			items = append(items, enrichMfgMetricDrilldownMap(tenantID, payload))
		}
		return items
	default:
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice {
			items := make([]map[string]interface{}, 0, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				items = append(items, enrichMfgMetricDrilldownMap(tenantID, structToMap(rv.Index(i).Interface())))
			}
			return items
		}
	}
	return value
}

func enrichMfgMetricDrilldownMap(tenantID uint, item map[string]interface{}) map[string]interface{} {
	if item == nil {
		return item
	}
	if strings.TrimSpace(stringFromMfgInput(item["customer_name"])) != "" {
		return item
	}
	customerID := uintFromMfgInput(firstAny(item["customer_id"], item["CustomerID"]))
	if customerID > 0 {
		item["customer_id"] = customerID
		item["customer_name"] = crmObjectName(tenantID, customerID)
	}
	return item
}

func structToMap(value interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return result
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return result
	}
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			name = field.Name
		}
		result[name] = rv.Field(i).Interface()
	}
	return result
}

func validateMfgDrawingAccessPolicy(drawing models.MfgDrawing, action string) error {
	action = strings.TrimSpace(action)
	if action == "" || action == "view" {
		return nil
	}
	policy := strings.ToLower(strings.TrimSpace(drawing.DownloadPolicy))
	externalPolicy := strings.ToLower(strings.TrimSpace(drawing.ExternalSendPolicy))
	approvalStatus := strings.ToLower(strings.TrimSpace(drawing.ApprovalStatus))
	classification := strings.ToUpper(strings.TrimSpace(drawing.ClassificationLevel))
	approvalRequired := policy == "approval_required" ||
		(action == "external-send" && externalPolicy == "approval_required") ||
		classification == "L4"
	if approvalRequired && approvalStatus != "approved" {
		return errors.New(MfgErrorDrawingApprovalRequired)
	}
	return nil
}

func normalizeMfgDrawingPolicy(item interface{}) {
	classification := strings.ToUpper(firstNonEmpty(stringField(item, "ClassificationLevel"), "L3"))
	setStringField(item, "ClassificationLevel", classification)
	policy := strings.ToLower(firstNonEmpty(stringField(item, "DownloadPolicy"), "audit_required"))
	externalPolicy := strings.ToLower(firstNonEmpty(stringField(item, "ExternalSendPolicy"), "approval_required"))
	approvalStatus := strings.ToLower(firstNonEmpty(stringField(item, "ApprovalStatus"), "not_required"))
	if classification == "L4" && policy == "audit_required" {
		policy = "approval_required"
	}
	if policy == "approval_required" || externalPolicy == "approval_required" || classification == "L4" {
		if approvalStatus == "not_required" {
			approvalStatus = "pending"
		}
	} else if approvalStatus == "" {
		approvalStatus = "not_required"
	}
	setStringField(item, "DownloadPolicy", policy)
	setStringField(item, "ExternalSendPolicy", externalPolicy)
	setStringField(item, "ApprovalStatus", approvalStatus)
}

func mfgAssetIoTAuthorized(asset models.MfgAsset) bool {
	if strings.TrimSpace(asset.IoTAssetRef) == "" {
		return false
	}
	if !asset.IoTAuthorized {
		return false
	}
	return asset.IoTRevokedAt == nil
}

func ChangeMfgECNStatus(tenantID uint, userID uint, id uint, action string, input map[string]interface{}) (models.MfgECN, error) {
	ecn, err := loadMfgECN(tenantID, id)
	if err != nil {
		return models.MfgECN{}, err
	}
	approvalStatus := strings.ToLower(firstNonEmpty(ecn.ApprovalStatus, "draft"))
	status := strings.ToLower(firstNonEmpty(ecn.Status, "draft"))
	oldApprovalStatus := firstNonEmpty(ecn.ApprovalStatus, "draft")
	remark := strings.TrimSpace(stringFromMfgInput(input["remark"]))
	now := time.Now()
	switch action {
	case "impact-analysis":
		if status == "effective" || approvalStatus == "pending" || approvalStatus == "approved" {
			return models.MfgECN{}, errors.New(MfgErrorInvalidState)
		}
		if value := jsonStringFromAny(input["changed_fields"]); value != "{}" {
			ecn.ChangedFieldsJSON = value
		}
		if value := jsonStringFromAny(input["bom_diff"]); value != "{}" {
			ecn.BOMDiffJSON = value
		}
		ecn.CostImpact = floatFromMfgInput(firstAny(input["cost_impact"], ecn.CostImpact))
		ecn.DeliveryImpactDays = int(floatFromMfgInput(firstAny(input["delivery_impact_days"], ecn.DeliveryImpactDays)))
		ecn.QualityImpact = firstNonEmpty(stringFromMfgInput(input["quality_impact"]), ecn.QualityImpact)
		ecn.Status = "impact_analyzed"
	case "submit-approval":
		if status == "effective" || approvalStatus == "pending" || approvalStatus == "approved" {
			return models.MfgECN{}, errors.New(MfgErrorInvalidState)
		}
		ecn.ApprovalStatus = "pending"
	case "approve":
		if approvalStatus != "pending" {
			return models.MfgECN{}, errors.New(MfgErrorInvalidState)
		}
		ecn.ApprovalStatus = "approved"
	case "reject":
		if approvalStatus != "pending" {
			return models.MfgECN{}, errors.New(MfgErrorInvalidState)
		}
		if remark == "" {
			return models.MfgECN{}, errors.New(MfgErrorApprovalRemarkRequired)
		}
		ecn.ApprovalStatus = "rejected"
		ecn.CustomerConfirmed = false
		if status == "" || status == "draft" {
			ecn.Status = "draft"
		}
	case "customer-confirm":
		if approvalStatus != "approved" || status == "effective" {
			return models.MfgECN{}, errors.New(MfgErrorInvalidState)
		}
		ecn.CustomerConfirmed = true
	case "effective":
		if ecn.ApprovalStatus != "approved" || !ecn.CustomerConfirmed {
			return models.MfgECN{}, errors.New(MfgErrorInvalidState)
		}
		ecn.Status = "effective"
	default:
		return models.MfgECN{}, errors.New(MfgErrorInvalidState)
	}
	ecn.UpdatedBy = userID
	err = utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&ecn).Error; err != nil {
			return err
		}
		if isMfgApprovalHistoryAction(action) {
			if err := createMfgApprovalHistoryTx(tx, tenantID, "ecns", ecn.ID, ecn.ECNNo, approvalHistoryActionFromWorkflowAction(action), oldApprovalStatus, firstNonEmpty(ecn.ApprovalStatus, "draft"), remark, userID, now); err != nil {
				return err
			}
		}
		if action == "effective" {
			return enqueueMfgIntegrationOutboxTx(tx, tenantID, userID, "ECNEffective", "plm_mes", "ecns", ecn.ID, map[string]interface{}{
				"ecn_id":               ecn.ID,
				"order_id":             ecn.OrderID,
				"old_bom_snapshot_id":  ecn.OldBOMSnapshotID,
				"new_bom_snapshot_id":  ecn.NewBOMSnapshotID,
				"cost_impact":          ecn.CostImpact,
				"delivery_impact_days": ecn.DeliveryImpactDays,
				"integration_status":   "not_configured",
				"compensation_state":   "pending",
			})
		}
		return nil
	})
	return ecn, err
}

func ChangeMfgCompensationStatus(tenantID uint, userID uint, id uint, action string, input map[string]interface{}) (models.MfgCompensationRecord, error) {
	item, err := loadMfgCompensationRecord(tenantID, id)
	if err != nil {
		return models.MfgCompensationRecord{}, err
	}
	approvalStatus := strings.ToLower(firstNonEmpty(item.ApprovalStatus, "draft"))
	status := strings.ToLower(firstNonEmpty(item.Status, "open"))
	oldApprovalStatus := firstNonEmpty(item.ApprovalStatus, "draft")
	remark := strings.TrimSpace(stringFromMfgInput(input["remark"]))
	now := time.Now()
	switch action {
	case "submit-approval":
		if status == "closed" || !stringIn(approvalStatus, "draft", "rejected", "") {
			return models.MfgCompensationRecord{}, errors.New(MfgErrorInvalidState)
		}
		item.ApprovalStatus = "pending"
		item.Status = "processing"
	case "approve":
		if approvalStatus != "pending" {
			return models.MfgCompensationRecord{}, errors.New(MfgErrorInvalidState)
		}
		item.ApprovalStatus = "approved"
		item.ApprovedBy = userID
		item.ApprovedAt = &now
		item.ApprovalRemark = remark
		item.Status = "processing"
	case "reject":
		if approvalStatus != "pending" {
			return models.MfgCompensationRecord{}, errors.New(MfgErrorInvalidState)
		}
		if remark == "" {
			return models.MfgCompensationRecord{}, errors.New(MfgErrorApprovalRemarkRequired)
		}
		item.ApprovalStatus = "rejected"
		item.ApprovedBy = userID
		item.ApprovedAt = &now
		item.ApprovalRemark = remark
		item.Status = "open"
	case "close":
		if approvalStatus != "approved" || status == "closed" {
			return models.MfgCompensationRecord{}, errors.New(MfgErrorInvalidState)
		}
		item.Status = "closed"
	default:
		return models.MfgCompensationRecord{}, errors.New(MfgErrorInvalidState)
	}
	item.UpdatedBy = userID
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&item).Error; err != nil {
			return err
		}
		if isMfgApprovalHistoryAction(action) {
			return createMfgApprovalHistoryTx(tx, tenantID, "compensation-records", item.ID, item.CompensationNo, approvalHistoryActionFromWorkflowAction(action), oldApprovalStatus, firstNonEmpty(item.ApprovalStatus, "draft"), remark, userID, now)
		}
		return nil
	}); err != nil {
		return models.MfgCompensationRecord{}, err
	}
	return item, nil
}

func MfgDashboard(tenantID uint) (map[string]interface{}, error) {
	count := func(model interface{}, where string, args ...interface{}) int64 {
		var total int64
		query := utils.DB.Model(model).Where("tenant_id = ?", tenantID)
		if where != "" {
			query = query.Where(where, args...)
		}
		_ = query.Count(&total).Error
		return total
	}
	var quoteAmount float64
	_ = utils.DB.Model(&models.MfgQuoteRevision{}).Where("tenant_id = ?", tenantID).Select("COALESCE(SUM(amount), 0)").Scan(&quoteAmount).Error
	var quoteSnapshot models.MetricMfgQuoteSnapshot
	var deliverySnapshot models.MetricMfgDeliverySnapshot
	var serviceSnapshot models.MetricMfgServiceSnapshot
	_ = utils.DB.Where("tenant_id = ?", tenantID).Order("stat_date desc, id desc").First(&quoteSnapshot).Error
	_ = utils.DB.Where("tenant_id = ?", tenantID).Order("stat_date desc, id desc").First(&deliverySnapshot).Error
	_ = utils.DB.Where("tenant_id = ?", tenantID).Order("stat_date desc, id desc").First(&serviceSnapshot).Error
	counts := map[string]int64{
		"rfq_count":                     count(&models.MfgRFQ{}, ""),
		"open_rfq_count":                count(&models.MfgRFQ{}, "status NOT IN ?", []string{"won", "lost", "canceled"}),
		"product_family_count":          count(&models.MfgProductFamily{}, ""),
		"bom_snapshot_count":            count(&models.MfgBOMSnapshot{}, ""),
		"low_margin_quote_count":        count(&models.MfgCostSheet{}, "margin_level = ?", "low"),
		"sample_count":                  count(&models.MfgSampleProject{}, ""),
		"red_delivery_count":            count(&models.MfgProductionProgress{}, "risk_level = ?", "red"),
		"ecn_pending_count":             count(&models.MfgECN{}, "approval_status = ?", "pending"),
		"asset_count":                   count(&models.MfgAsset{}, ""),
		"service_ticket_count":          count(&models.MfgServiceTicket{}, ""),
		"integration_pending_count":     count(&models.MfgIntegrationOutbox{}, "status IN ?", []string{"pending", "failed"}),
		"integration_dead_letter_count": count(&models.MfgIntegrationOutbox{}, "status = ?", "dead_letter"),
	}
	snapshots := map[string]interface{}{
		"quote":    quoteSnapshot,
		"delivery": deliverySnapshot,
		"service":  serviceSnapshot,
	}
	return map[string]interface{}{
		"rfq_count":                     count(&models.MfgRFQ{}, ""),
		"open_rfq_count":                count(&models.MfgRFQ{}, "status NOT IN ?", []string{"won", "lost", "canceled"}),
		"product_family_count":          count(&models.MfgProductFamily{}, ""),
		"bom_snapshot_count":            count(&models.MfgBOMSnapshot{}, ""),
		"low_margin_quote_count":        count(&models.MfgCostSheet{}, "margin_level = ?", "low"),
		"quote_amount":                  quoteAmount,
		"sample_count":                  count(&models.MfgSampleProject{}, ""),
		"red_delivery_count":            count(&models.MfgProductionProgress{}, "risk_level = ?", "red"),
		"ecn_pending_count":             count(&models.MfgECN{}, "approval_status = ?", "pending"),
		"asset_count":                   count(&models.MfgAsset{}, ""),
		"service_ticket_count":          count(&models.MfgServiceTicket{}, ""),
		"integration_pending_count":     count(&models.MfgIntegrationOutbox{}, "status IN ?", []string{"pending", "failed"}),
		"integration_dead_letter_count": count(&models.MfgIntegrationOutbox{}, "status = ?", "dead_letter"),
		"snapshots":                     snapshots,
		"metric_groups":                 mfgDashboardMetricGroups(counts, quoteAmount, quoteSnapshot, deliverySnapshot, serviceSnapshot),
		"refresh_status": map[string]interface{}{
			"quote":    quoteSnapshot.RefreshStatus,
			"delivery": deliverySnapshot.RefreshStatus,
			"service":  serviceSnapshot.RefreshStatus,
		},
		"refresh_errors": map[string]interface{}{
			"quote":    quoteSnapshot.RefreshError,
			"delivery": deliverySnapshot.RefreshError,
			"service":  serviceSnapshot.RefreshError,
		},
	}, nil
}

func validateMfgRecordForSave(tenantID uint, resource string, item interface{}) error {
	switch resource {
	case "rfqs":
		customerID := uintField(item, "CustomerID")
		if customerID > 0 {
			if err := ensureManufacturingCustomer(tenantID, customerID); err != nil {
				return err
			}
		}
	case "product-families":
		normalizeMfgProductFamilyStatus(item)
		familyCode := strings.TrimSpace(stringField(item, "FamilyCode"))
		if familyCode == "" {
			return errors.New(MfgErrorInvalidInput)
		}
		query := utils.DB.Model(&models.MfgProductFamily{}).Where("tenant_id = ? AND family_code = ?", tenantID, familyCode)
		if id := uintField(item, "ID"); id > 0 {
			query = query.Where("id <> ?", id)
		}
		var count int64
		if err := query.Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New(MfgErrorInvalidInput)
		}
	case "rfq-clarifications":
		if rfqID := uintField(item, "RFQID"); rfqID == 0 {
			return errors.New(MfgErrorInvalidInput)
		} else if _, err := loadMfgRFQ(tenantID, rfqID); err != nil {
			return err
		}
	case "drawings":
		if rfqID := uintField(item, "RFQID"); rfqID > 0 {
			if _, err := loadMfgRFQ(tenantID, rfqID); err != nil {
				return err
			}
		}
		normalizeMfgDrawingPolicy(item)
	case "sample-projects":
		if customerID := uintField(item, "CustomerID"); customerID > 0 {
			if err := ensureManufacturingCustomer(tenantID, customerID); err != nil {
				return err
			}
		}
	case "bom-snapshots":
		if stringField(item, "BOMLinesJSON") == "" {
			setStringField(item, "BOMLinesJSON", jsonStringFromAny(nil))
		}
	case "quotes":
		if uintField(item, "BOMSnapshotID") == 0 {
			return errors.New(MfgErrorBOMSnapshotRequired)
		}
	case "assets":
		if customerID := uintField(item, "CustomerID"); customerID > 0 {
			if err := ensureManufacturingCustomer(tenantID, customerID); err != nil {
				return err
			}
		}
		if stringField(item, "IoTAssetRef") != "" && !boolField(item, "IoTAuthorized") {
			setBoolField(item, "IoTAuthorized", true)
			now := time.Now()
			setTimePtrField(item, "IoTAuthorizedAt", &now)
		}
		serialNo := stringField(item, "SerialNo")
		if serialNo != "" {
			id := uintField(item, "ID")
			query := utils.DB.Model(&models.MfgAsset{}).Where("tenant_id = ? AND serial_no = ?", tenantID, serialNo)
			if id > 0 {
				query = query.Where("id <> ?", id)
			}
			var count int64
			if err := query.Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errors.New(MfgErrorDuplicateSerialNo)
			}
		}
	}
	return nil
}

func normalizeMfgProductFamilyStatus(item interface{}) {
	status := strings.ToLower(strings.TrimSpace(stringField(item, "Status")))
	switch status {
	case "inactive", "disabled", "disable", "off", "closed", "deleted":
		setStringField(item, "Status", "inactive")
	default:
		setStringField(item, "Status", "active")
	}
}

func enrichMfgProductFamilyValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case []models.MfgProductFamily:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, mfgProductFamilyDTO(item))
		}
		return items
	case models.MfgProductFamily:
		return mfgProductFamilyDTO(typed)
	case *models.MfgProductFamily:
		if typed == nil {
			return nil
		}
		return mfgProductFamilyDTO(*typed)
	default:
		return value
	}
}

func mfgProductFamilyDTO(item models.MfgProductFamily) map[string]interface{} {
	status := strings.ToLower(strings.TrimSpace(item.Status))
	if status != "inactive" {
		status = "active"
	}
	return map[string]interface{}{
		"id":          item.ID,
		"created_at":  item.CreatedAt,
		"updated_at":  item.UpdatedAt,
		"tenant_id":   item.TenantID,
		"family_code": item.FamilyCode,
		"name":        item.Name,
		"description": item.Description,
		"status":      status,
		"owner_id":    item.OwnerID,
		"created_by":  item.CreatedBy,
		"updated_by":  item.UpdatedBy,
	}
}

func mfgResource(resource string) (mfgResourceSpec, bool) {
	resource = strings.Trim(strings.ToLower(strings.TrimSpace(resource)), "/")
	specs := map[string]mfgResourceSpec{
		"rfqs":                 {Name: "rfqs", Model: func() interface{} { return &models.MfgRFQ{} }, Slice: func() interface{} { return &[]models.MfgRFQ{} }, KeywordColumns: []string{"rfq_no", "business_name", "payload_json"}, StatusColumn: "status", NoField: "RFQNo", NoPrefix: "RFQ", FileFields: []mfgFileField{{FileField: "DrawingFileID", NodeField: "DrawingStorageNodeID"}}},
		"rfq-clarifications":   {Name: "rfq-clarifications", Model: func() interface{} { return &models.MfgRFQClarification{} }, Slice: func() interface{} { return &[]models.MfgRFQClarification{} }, KeywordColumns: []string{"question", "answer"}, StatusColumn: "status"},
		"drawings":             {Name: "drawings", Model: func() interface{} { return &models.MfgDrawing{} }, Slice: func() interface{} { return &[]models.MfgDrawing{} }, KeywordColumns: []string{"drawing_no", "version", "file_id"}, StatusColumn: "status", FileFields: []mfgFileField{{FileField: "FileID", NodeField: "StorageNodeID"}}},
		"product-families":     {Name: "product-families", Model: func() interface{} { return &models.MfgProductFamily{} }, Slice: func() interface{} { return &[]models.MfgProductFamily{} }, KeywordColumns: []string{"family_code", "name", "description"}, StatusColumn: "status", NoField: "FamilyCode", NoPrefix: "PF"},
		"config-models":        {Name: "config-models", Model: func() interface{} { return &models.MfgConfigModel{} }, Slice: func() interface{} { return &[]models.MfgConfigModel{} }, KeywordColumns: []string{"model_no", "version"}, StatusColumn: "status", NoField: "ModelNo", NoPrefix: "CFG", Versioned: true},
		"bom-templates":        {Name: "bom-templates", Model: func() interface{} { return &models.MfgBOMTemplate{} }, Slice: func() interface{} { return &[]models.MfgBOMTemplate{} }, KeywordColumns: []string{"template_no", "version"}, StatusColumn: "status", NoField: "TemplateNo", NoPrefix: "BOMT", Versioned: true},
		"bom-snapshots":        {Name: "bom-snapshots", Model: func() interface{} { return &models.MfgBOMSnapshot{} }, Slice: func() interface{} { return &[]models.MfgBOMSnapshot{} }, KeywordColumns: []string{"snapshot_no", "business_name"}, StatusColumn: "status", NoField: "SnapshotNo", NoPrefix: "BOMS", Immutable: true},
		"cost-sheets":          {Name: "cost-sheets", Model: func() interface{} { return &models.MfgCostSheet{} }, Slice: func() interface{} { return &[]models.MfgCostSheet{} }, KeywordColumns: []string{"margin_level", "status"}, StatusColumn: "status"},
		"quotes":               {Name: "quotes", Model: func() interface{} { return &models.MfgQuoteRevision{} }, Slice: func() interface{} { return &[]models.MfgQuoteRevision{} }, KeywordColumns: []string{"quote_no", "terms_json"}, StatusColumn: "status", NoField: "QuoteNo", NoPrefix: "MQ", Versioned: true},
		"sample-projects":      {Name: "sample-projects", Model: func() interface{} { return &models.MfgSampleProject{} }, Slice: func() interface{} { return &[]models.MfgSampleProject{} }, KeywordColumns: []string{"sample_no", "customer_feedback"}, StatusColumn: "stage", NoField: "SampleNo", NoPrefix: "SMP"},
		"sample-test-records":  {Name: "sample-test-records", Model: func() interface{} { return &models.MfgSampleTestRecord{} }, Slice: func() interface{} { return &[]models.MfgSampleTestRecord{} }, KeywordColumns: []string{"test_item", "failure_reason"}, StatusColumn: "test_result", FileFields: []mfgFileField{{FileField: "AttachmentFileID", NodeField: "StorageNodeID"}}},
		"production-progress":  {Name: "production-progress", Model: func() interface{} { return &models.MfgProductionProgress{} }, Slice: func() interface{} { return &[]models.MfgProductionProgress{} }, KeywordColumns: []string{"erp_order_no", "mes_work_order_no", "milestone", "progress_status", "risk_reason"}, StatusColumn: "risk_level"},
		"ecns":                 {Name: "ecns", Model: func() interface{} { return &models.MfgECN{} }, Slice: func() interface{} { return &[]models.MfgECN{} }, KeywordColumns: []string{"ecn_no", "change_reason", "impact_scope"}, StatusColumn: "status", NoField: "ECNNo", NoPrefix: "ECN"},
		"compensation-records": {Name: "compensation-records", Model: func() interface{} { return &models.MfgCompensationRecord{} }, Slice: func() interface{} { return &[]models.MfgCompensationRecord{} }, KeywordColumns: []string{"compensation_no", "reason", "responsibility_split", "improvement_actions"}, StatusColumn: "status", NoField: "CompensationNo", NoPrefix: "CMP"},
		"assets":               {Name: "assets", Model: func() interface{} { return &models.MfgAsset{} }, Slice: func() interface{} { return &[]models.MfgAsset{} }, KeywordColumns: []string{"serial_no", "batch_no", "install_location", "iot_asset_ref"}, StatusColumn: "status", NoField: "SerialNo", NoPrefix: "SN"},
		"service-tickets":      {Name: "service-tickets", Model: func() interface{} { return &models.MfgServiceTicket{} }, Slice: func() interface{} { return &[]models.MfgServiceTicket{} }, KeywordColumns: []string{"fault_code", "service_type", "service_result"}, StatusColumn: "status"},
		"spare-part-quotes":    {Name: "spare-part-quotes", Model: func() interface{} { return &models.MfgSparePartQuote{} }, Slice: func() interface{} { return &[]models.MfgSparePartQuote{} }, KeywordColumns: []string{"quote_no", "line_items_json"}, StatusColumn: "status", NoField: "QuoteNo", NoPrefix: "SPQ"},
		"agreements":           {Name: "agreements", Model: func() interface{} { return &models.MfgSalesAgreement{} }, Slice: func() interface{} { return &[]models.MfgSalesAgreement{} }, KeywordColumns: []string{"agreement_no", "terms_json"}, StatusColumn: "status", NoField: "AgreementNo", NoPrefix: "AGR"},
		"forecasts":            {Name: "forecasts", Model: func() interface{} { return &models.MfgAccountForecast{} }, Slice: func() interface{} { return &[]models.MfgAccountForecast{} }, KeywordColumns: []string{"forecast_no", "period", "assumption_json"}, StatusColumn: "status", NoField: "ForecastNo", NoPrefix: "FCST"},
		"knowledge":            {Name: "knowledge", Model: func() interface{} { return &models.MfgKnowledgeEntry{} }, Slice: func() interface{} { return &[]models.MfgKnowledgeEntry{} }, KeywordColumns: []string{"title", "problem", "root_cause", "solution", "tags"}, StatusColumn: "status"},
		"knowledge-usages":     {Name: "knowledge-usages", Model: func() interface{} { return &models.MfgKnowledgeUsage{} }, Slice: func() interface{} { return &[]models.MfgKnowledgeUsage{} }, KeywordColumns: []string{"target_type", "applied_summary"}, StatusColumn: ""},
	}
	spec, ok := specs[resource]
	return spec, ok
}

func applyMfgInput(item interface{}, input map[string]interface{}) error {
	normalized := map[string]interface{}{}
	for key, value := range input {
		normalized[key] = value
	}
	if value, ok := normalized["quote_amount"]; ok {
		normalized["amount"] = value
	}
	jsonAliases := map[string]string{
		"payload":        "payload_json",
		"terms":          "terms_json",
		"line_items":     "line_items_json",
		"bom_lines":      "bom_lines_json",
		"rules":          "rule_json",
		"rule":           "rule_json",
		"parameters":     "parameters_json",
		"constraints":    "constraints_json",
		"default_values": "default_values_json",
		"config_values":  "config_values_json",
		"changed_fields": "changed_fields_json",
		"bom_diff":       "bom_diff_json",
		"sensitive_cost": "sensitive_cost_json",
		"source":         "source_json",
	}
	for from, to := range jsonAliases {
		if value, ok := normalized[from]; ok {
			normalized[to] = jsonStringFromAny(value)
		}
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, item); err != nil {
		return errors.New(MfgErrorInvalidInput)
	}
	return nil
}

func applyMfgCRMDefaults(tenantID uint, item interface{}) {
	if !hasField(item, "OrderID") {
		return
	}
	orderID := uintField(item, "OrderID")
	if orderID == 0 {
		return
	}
	customerID, salesOwnerID := resolveMfgCRMOwnership(tenantID, orderID, uintField(item, "CustomerID"), uintField(item, "SalesOwnerID"))
	if customerID > 0 {
		setUintField(item, "CustomerID", customerID)
	}
	if salesOwnerID > 0 {
		setUintField(item, "SalesOwnerID", salesOwnerID)
	}
}

func matchingMfgCRMObjectIDs(tenantID uint, objectType string, keyword string) []uint {
	if strings.TrimSpace(keyword) == "" {
		return nil
	}
	var rows []models.CRMObject
	if err := utils.DB.Select("id").
		Where("tenant_id = ? AND object_type = ? AND (name LIKE ? OR code LIKE ?)", tenantID, objectType, "%"+keyword+"%", "%"+keyword+"%").
		Limit(30).
		Find(&rows).Error; err != nil {
		return nil
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func matchingMfgUserIDs(tenantID uint, keyword string) []uint {
	if strings.TrimSpace(keyword) == "" {
		return nil
	}
	var rows []models.User
	if err := utils.DB.Select("id").
		Where("tenant_id = ? AND (name LIKE ? OR username LIKE ?)", tenantID, "%"+keyword+"%", "%"+keyword+"%").
		Limit(30).
		Find(&rows).Error; err != nil {
		return nil
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func GetMfgRFQ(tenantID uint, id uint) (models.MfgRFQ, error) {
	return loadMfgRFQ(tenantID, id)
}

func GetMfgQuoteRFQID(tenantID uint, id uint) (uint, error) {
	quote, err := loadMfgQuote(tenantID, id)
	if err != nil {
		return 0, err
	}
	return quote.RFQID, nil
}

func loadMfgRFQ(tenantID uint, id uint) (models.MfgRFQ, error) {
	var item models.MfgRFQ
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, errors.New(MfgErrorNotFound)
		}
		return item, err
	}
	return item, nil
}

func loadMfgQuote(tenantID uint, id uint) (models.MfgQuoteRevision, error) {
	var item models.MfgQuoteRevision
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, errors.New(MfgErrorNotFound)
		}
		return item, err
	}
	return item, nil
}

func loadMfgECN(tenantID uint, id uint) (models.MfgECN, error) {
	var item models.MfgECN
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, errors.New(MfgErrorNotFound)
		}
		return item, err
	}
	return item, nil
}

func loadMfgCompensationRecord(tenantID uint, id uint) (models.MfgCompensationRecord, error) {
	var item models.MfgCompensationRecord
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, errors.New(MfgErrorNotFound)
		}
		return item, err
	}
	return item, nil
}

func ensureMfgBOMSnapshot(tenantID uint, id uint) error {
	var count int64
	if err := utils.DB.Model(&models.MfgBOMSnapshot{}).Where("tenant_id = ? AND id = ?", tenantID, id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New(MfgErrorBOMSnapshotRequired)
	}
	return nil
}

func validateConfigValues(model models.MfgConfigModel, values map[string]interface{}) map[string]string {
	result := map[string]string{}
	constraints := jsonMap(model.ConstraintsJSON)
	requiredRaw, _ := constraints["required"].([]interface{})
	for _, item := range requiredRaw {
		key := strings.TrimSpace(fmt.Sprint(item))
		if key == "" {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(values[key])) == "" || fmt.Sprint(values[key]) == "<nil>" {
			result[key] = "required"
		}
	}
	if ranges, ok := constraints["range"].(map[string]interface{}); ok {
		for key, raw := range ranges {
			rule := mapFromMfgInput(raw)
			value := floatFromMfgInput(values[key])
			if minRaw, exists := rule["min"]; exists && value < floatFromMfgInput(minRaw) {
				result[key] = "range"
			}
			if maxRaw, exists := rule["max"]; exists && value > floatFromMfgInput(maxRaw) {
				result[key] = "range"
			}
		}
	}
	if mutexRules, ok := constraints["mutex"].([]interface{}); ok {
		for _, raw := range mutexRules {
			keys := stringSliceFromAny(raw)
			present := []string{}
			for _, key := range keys {
				if strings.TrimSpace(fmt.Sprint(values[key])) != "" && fmt.Sprint(values[key]) != "<nil>" {
					present = append(present, key)
				}
			}
			if len(present) > 1 {
				for _, key := range present {
					result[key] = "mutex"
				}
			}
		}
	}
	if dependencies, ok := constraints["dependency"].(map[string]interface{}); ok {
		for key, raw := range dependencies {
			requiredKeys := stringSliceFromAny(raw)
			if strings.TrimSpace(fmt.Sprint(values[key])) == "" || fmt.Sprint(values[key]) == "<nil>" {
				continue
			}
			for _, requiredKey := range requiredKeys {
				if strings.TrimSpace(fmt.Sprint(values[requiredKey])) == "" || fmt.Sprint(values[requiredKey]) == "<nil>" {
					result[requiredKey] = "dependency"
				}
			}
		}
	}
	return result
}

func marginLevel(rate float64) string {
	if rate < 0.15 {
		return "low"
	}
	if rate < 0.3 {
		return "medium"
	}
	return "high"
}

func marginLevelForTenant(tenantID uint, rate float64) string {
	lowThreshold := 0.15
	mediumThreshold := 0.3
	var setting models.CRMSetting
	if err := utils.DB.Where("tenant_id = ? AND scope = ? AND `key` = ?", tenantID, "manufacturing", "margin_policy").First(&setting).Error; err == nil {
		payload := mapFromMfgInput(setting.ValueJSON)
		if value := floatFromMfgInput(payload["low_margin_threshold"]); value > 0 {
			lowThreshold = value
		}
		if value := floatFromMfgInput(payload["medium_margin_threshold"]); value > lowThreshold {
			mediumThreshold = value
		}
	}
	if rate < lowThreshold {
		return "low"
	}
	if rate < mediumThreshold {
		return "medium"
	}
	return "high"
}

func mfgVersionedRecordLocked(item interface{}) bool {
	status := strings.ToLower(strings.TrimSpace(stringField(item, "Status")))
	approvalStatus := strings.ToLower(strings.TrimSpace(stringField(item, "ApprovalStatus")))
	sendStatus := strings.ToLower(strings.TrimSpace(stringField(item, "SendStatus")))
	switch status {
	case "active", "approved", "sent", "effective", "sealed", "published":
		return true
	}
	switch approvalStatus {
	case "pending", "approved":
		return true
	}
	return sendStatus == "sent"
}

func setDefaultNo(item interface{}, fieldName string, prefix string) {
	if fieldName == "" || stringField(item, fieldName) != "" {
		return
	}
	setStringField(item, fieldName, nextMfgNo(prefix))
}

func nextMfgNo(prefix string) string {
	return fmt.Sprintf("%s-%s", strings.ToUpper(prefix), time.Now().Format("20060102150405"))
}

func jsonStringFromAny(value interface{}) string {
	if value == nil {
		return "{}"
	}
	if text, ok := value.(string); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			return "{}"
		}
		if json.Valid([]byte(text)) {
			return text
		}
	}
	raw, err := json.Marshal(value)
	if err != nil || len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}

func payloadFromJSON(raw string) map[string]interface{} {
	payload := map[string]interface{}{}
	_ = json.Unmarshal([]byte(raw), &payload)
	return payload
}

const mfgMaskedPrice = "***"

func canViewMfgRFQPrice(rfq models.MfgRFQ, access MfgAccessContext) bool {
	if access.IsAdmin || access.IsSupervisor || access.CanApproveQuote {
		return true
	}
	return access.UserID > 0 && rfq.CreatedBy == access.UserID
}

func canManageMfgRFQRecord(rfq models.MfgRFQ, access MfgAccessContext) bool {
	return access.IsAdmin || access.IsSupervisor || access.CanManageRFQ || (access.UserID > 0 && rfq.CreatedBy == access.UserID)
}

func canCreateMfgBOMForRFQ(rfq models.MfgRFQ, access MfgAccessContext) bool {
	return access.IsAdmin || access.IsSupervisor || access.CanManageProduct || canManageMfgRFQRecord(rfq, access)
}

func canCreateMfgCostForRFQ(rfq models.MfgRFQ, access MfgAccessContext) bool {
	return access.IsAdmin || access.IsSupervisor || access.CanManageCost || canManageMfgRFQRecord(rfq, access)
}

func canCreateMfgQuoteForRFQ(rfq models.MfgRFQ, access MfgAccessContext) bool {
	return access.IsAdmin || access.IsSupervisor || access.CanManageQuote || canManageMfgRFQRecord(rfq, access)
}

func canViewMfgPriceForRFQ(tenantID uint, rfqID uint, access MfgAccessContext) bool {
	if access.IsAdmin || access.IsSupervisor {
		return true
	}
	if rfqID == 0 || access.UserID == 0 {
		return false
	}
	rfq, err := loadMfgRFQ(tenantID, rfqID)
	return err == nil && canViewMfgRFQPrice(rfq, access)
}

func maskMfgPriceFields(payload map[string]interface{}, keys ...string) {
	for _, key := range keys {
		if _, ok := payload[key]; ok {
			payload[key] = mfgMaskedPrice
		}
	}
	payload["price_masked"] = true
}

func maskMfgPayloadPrices(payload map[string]interface{}) {
	maskMfgPriceFields(payload,
		"target_price",
		"amount",
		"quote_amount",
		"unit_price",
		"material_cost",
		"labor_cost",
		"outsourcing_cost",
		"manufacturing_overhead",
		"gross_margin",
		"gross_margin_rate",
		"cost_impact",
		"compensation_amount",
	)
}

func sanitizeMfgRFQValue(value interface{}, access MfgAccessContext) interface{} {
	switch typed := value.(type) {
	case []models.MfgRFQ:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, sanitizeMfgRFQ(item, access))
		}
		return items
	case models.MfgRFQ:
		return sanitizeMfgRFQ(typed, access)
	default:
		return value
	}
}

func sanitizeMfgRFQ(item models.MfgRFQ, access MfgAccessContext) map[string]interface{} {
	canManageRFQ := canManageMfgRFQRecord(item, access)
	canCreateBOM := canCreateMfgBOMForRFQ(item, access)
	canCreateCost := canCreateMfgCostForRFQ(item, access)
	canCreateQuote := canCreateMfgQuoteForRFQ(item, access)
	payload := map[string]interface{}{
		"id":                      item.ID,
		"created_at":              item.CreatedAt,
		"updated_at":              item.UpdatedAt,
		"tenant_id":               item.TenantID,
		"customer_id":             item.CustomerID,
		"contact_id":              item.ContactID,
		"opportunity_id":          item.OpportunityID,
		"rfq_no":                  item.RFQNo,
		"business_name":           item.BusinessName,
		"product_family_id":       item.ProductFamilyID,
		"quantity":                item.Quantity,
		"target_price":            item.TargetPrice,
		"required_delivery_date":  item.RequiredDeliveryDate,
		"drawing_file_id":         item.DrawingFileID,
		"drawing_storage_node_id": item.DrawingStorageNodeID,
		"no_drawing_reason":       item.NoDrawingReason,
		"status":                  item.Status,
		"owner_id":                item.OwnerID,
		"technical_owner_id":      item.TechnicalOwnerID,
		"payload_json":            item.PayloadJSON,
		"rfq_payload":             jsonMap(item.PayloadJSON),
		"created_by":              item.CreatedBy,
		"updated_by":              item.UpdatedBy,
		"can_edit":                canManageRFQ,
		"can_delete":              canManageRFQ,
		"can_submit":              canManageRFQ,
		"can_start_costing":       canCreateCost,
		"can_create_bom":          canCreateBOM,
		"can_create_cost":         canCreateCost,
		"can_create_quote":        canCreateQuote,
	}
	if !canViewMfgRFQPrice(item, access) {
		maskMfgPayloadPrices(payload)
	}
	return payload
}

func sanitizeMfgCostSheet(item models.MfgCostSheet) map[string]interface{} {
	payload := map[string]interface{}{
		"id":                item.ID,
		"created_at":        item.CreatedAt,
		"updated_at":        item.UpdatedAt,
		"tenant_id":         item.TenantID,
		"rfq_id":            item.RFQID,
		"bom_snapshot_id":   item.BOMSnapshotID,
		"amount":            item.Amount,
		"gross_margin_rate": item.GrossMarginRate,
		"margin_level":      item.MarginLevel,
		"status":            item.Status,
		"sensitive_masked":  true,
	}
	addMfgCostSheetContext(payload, item)
	return payload
}

func enrichMfgCostSheetValue(value interface{}, access MfgAccessContext) interface{} {
	switch typed := value.(type) {
	case []models.MfgCostSheet:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, enrichMfgCostSheet(item, access))
		}
		return items
	case models.MfgCostSheet:
		return enrichMfgCostSheet(typed, access)
	default:
		return value
	}
}

func enrichMfgCostSheet(item models.MfgCostSheet, access MfgAccessContext) map[string]interface{} {
	payload := map[string]interface{}{
		"id":                     item.ID,
		"created_at":             item.CreatedAt,
		"updated_at":             item.UpdatedAt,
		"tenant_id":              item.TenantID,
		"rfq_id":                 item.RFQID,
		"bom_snapshot_id":        item.BOMSnapshotID,
		"material_cost":          item.MaterialCost,
		"labor_cost":             item.LaborCost,
		"outsourcing_cost":       item.OutsourcingCost,
		"manufacturing_overhead": item.ManufacturingOverhead,
		"amount":                 item.Amount,
		"gross_margin":           item.GrossMargin,
		"gross_margin_rate":      item.GrossMarginRate,
		"margin_level":           item.MarginLevel,
		"status":                 item.Status,
	}
	if access.IsAdmin || access.IsSupervisor || canViewMfgPriceForRFQ(item.TenantID, item.RFQID, access) {
		payload["sensitive_cost_json"] = item.SensitiveCostJSON
	}
	addMfgCostSheetContext(payload, item)
	if !canViewMfgPriceForRFQ(item.TenantID, item.RFQID, access) {
		maskMfgPayloadPrices(payload)
	}
	return payload
}

func addMfgCostSheetContext(payload map[string]interface{}, item models.MfgCostSheet) {
	context := jsonMap(item.SensitiveCostJSON)
	for key, value := range context {
		if key == "rfq_payload" || key == "risk_flags" || key == "next_action" || key == "assigned_role" || key == "technical_role" || key == "bom_missing" {
			payload[key] = value
		}
	}
	if item.RFQID == 0 {
		return
	}
	if utils.DB == nil {
		return
	}
	var rfq models.MfgRFQ
	if err := utils.DB.Where("tenant_id = ? AND id = ?", item.TenantID, item.RFQID).First(&rfq).Error; err != nil {
		return
	}
	rfqPayload := jsonMap(rfq.PayloadJSON)
	payload["rfq_no"] = rfq.RFQNo
	payload["rfq_business_name"] = rfq.BusinessName
	payload["customer_id"] = rfq.CustomerID
	payload["contact_id"] = rfq.ContactID
	payload["opportunity_id"] = rfq.OpportunityID
	payload["product_family_id"] = rfq.ProductFamilyID
	payload["quantity"] = rfq.Quantity
	payload["target_price"] = rfq.TargetPrice
	payload["required_delivery_date"] = rfq.RequiredDeliveryDate
	payload["drawing_file_id"] = rfq.DrawingFileID
	payload["drawing_storage_node_id"] = rfq.DrawingStorageNodeID
	payload["no_drawing_reason"] = rfq.NoDrawingReason
	payload["rfq_status"] = rfq.Status
	payload["rfq_payload"] = rfqPayload
	payload["customer_name"] = mfgPayloadString(rfqPayload, "customer_name")
	payload["contact_name"] = mfgPayloadString(rfqPayload, "contact_name")
	payload["opportunity_name"] = mfgPayloadString(rfqPayload, "opportunity_name")
	payload["product_name"] = firstNonEmpty(mfgPayloadString(rfqPayload, "product_name"), mfgPayloadString(rfqPayload, "custom_product_name"), mfgPayloadString(rfqPayload, "sku"))
	payload["currency"] = firstNonEmpty(mfgPayloadString(rfqPayload, "currency"), "CNY")
	payload["technical_requirements"] = mfgPayloadString(rfqPayload, "technical_requirements")
	payload["quality_requirements"] = mfgPayloadString(rfqPayload, "quality_requirements")
	payload["packaging_requirements"] = mfgPayloadString(rfqPayload, "packaging_requirements")
	payload["customer_notes"] = mfgPayloadString(rfqPayload, "customer_notes")

	effectiveBOMSnapshotID := item.BOMSnapshotID
	if effectiveBOMSnapshotID == 0 {
		var snapshot models.MfgBOMSnapshot
		err := utils.DB.Where("tenant_id = ? AND rfq_id = ? AND LOWER(TRIM(status)) = ?", item.TenantID, item.RFQID, "active").Order("updated_at desc, id desc").First(&snapshot).Error
		if err != nil {
			err = utils.DB.Where("tenant_id = ? AND rfq_id = ?", item.TenantID, item.RFQID).Order("updated_at desc, id desc").First(&snapshot).Error
		}
		if err == nil {
			effectiveBOMSnapshotID = snapshot.ID
			payload["latest_bom_snapshot_id"] = snapshot.ID
			payload["original_bom_snapshot_id"] = item.BOMSnapshotID
		}
	}
	bomMissing := effectiveBOMSnapshotID == 0
	payload["bom_snapshot_id"] = effectiveBOMSnapshotID
	payload["bom_missing"] = bomMissing
	payload["next_action"] = costingNextAction(bomMissing)
	payload["risk_flags"] = costingRiskFlags(rfq, bomMissing)
}

func enrichMfgBOMSnapshotValue(value interface{}) interface{} {
	return enrichMfgBOMSnapshotValueForAccess(value, MfgAccessContext{IsAdmin: true})
}

func enrichMfgBOMSnapshotValueForAccess(value interface{}, access MfgAccessContext) interface{} {
	switch typed := value.(type) {
	case []models.MfgBOMSnapshot:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, enrichMfgBOMSnapshotForAccess(item, access))
		}
		return items
	case models.MfgBOMSnapshot:
		return enrichMfgBOMSnapshotForAccess(typed, access)
	default:
		return value
	}
}

func enrichMfgBOMTemplateValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case []models.MfgBOMTemplate:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, enrichMfgBOMTemplate(item))
		}
		return items
	case models.MfgBOMTemplate:
		return enrichMfgBOMTemplate(typed)
	default:
		return value
	}
}

func enrichMfgBOMTemplate(item models.MfgBOMTemplate) map[string]interface{} {
	bomLines := bomLinesFromJSON(item.BOMLinesJSON)
	return map[string]interface{}{
		"id":                item.ID,
		"created_at":        item.CreatedAt,
		"updated_at":        item.UpdatedAt,
		"tenant_id":         item.TenantID,
		"product_family_id": item.ProductFamilyID,
		"template_no":       item.TemplateNo,
		"version":           item.Version,
		"bom_lines_json":    item.BOMLinesJSON,
		"bom_lines":         bomLines,
		"bom_line_count":    len(bomLines),
		"rule_json":         item.RuleJSON,
		"rule":              jsonMap(item.RuleJSON),
		"status":            item.Status,
		"created_by":        item.CreatedBy,
		"updated_by":        item.UpdatedBy,
	}
}

func enrichMfgProductionProgressValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case []models.MfgProductionProgress:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, enrichMfgProductionProgress(item))
		}
		return items
	case models.MfgProductionProgress:
		return enrichMfgProductionProgress(typed)
	default:
		return value
	}
}

func enrichMfgProductionProgress(item models.MfgProductionProgress) map[string]interface{} {
	customerID, salesOwnerID := resolveMfgCRMOwnership(item.TenantID, item.OrderID, item.CustomerID, item.SalesOwnerID)
	payload := map[string]interface{}{
		"id":                item.ID,
		"created_at":        item.CreatedAt,
		"updated_at":        item.UpdatedAt,
		"tenant_id":         item.TenantID,
		"order_id":          item.OrderID,
		"order_name":        crmObjectName(item.TenantID, item.OrderID),
		"order_code":        crmObjectCode(item.TenantID, item.OrderID),
		"customer_id":       customerID,
		"customer_name":     crmObjectName(item.TenantID, customerID),
		"sales_owner_id":    salesOwnerID,
		"sales_owner_name":  userDisplayName(item.TenantID, salesOwnerID),
		"erp_order_no":      item.ERPOrderNo,
		"mes_work_order_no": item.MESWorkOrderNo,
		"milestone":         item.Milestone,
		"progress_status":   normalizeMfgProductionProgressStatus(item.ProgressStatus),
		"planned_at":        item.PlannedAt,
		"actual_at":         item.ActualAt,
		"risk_level":        item.RiskLevel,
		"risk_reason":       item.RiskReason,
		"source_system":     item.SourceSystem,
		"sync_status":       item.SyncStatus,
		"created_by":        item.CreatedBy,
		"updated_by":        item.UpdatedBy,
	}
	var count int64
	if utils.DB != nil {
		_ = utils.DB.Model(&models.MfgProductionProgressHistory{}).Where("tenant_id = ? AND production_progress_id = ?", item.TenantID, item.ID).Count(&count).Error
	}
	payload["progress_history_count"] = count
	return payload
}

func enrichMfgSampleProjectValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case []models.MfgSampleProject:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, enrichMfgSampleProject(item))
		}
		return items
	case models.MfgSampleProject:
		return enrichMfgSampleProject(typed)
	default:
		return value
	}
}

func enrichMfgSampleProject(item models.MfgSampleProject) map[string]interface{} {
	payload := map[string]interface{}{
		"id":                     item.ID,
		"created_at":             item.CreatedAt,
		"updated_at":             item.UpdatedAt,
		"tenant_id":              item.TenantID,
		"customer_id":            item.CustomerID,
		"customer_name":          crmObjectName(item.TenantID, item.CustomerID),
		"opportunity_id":         item.OpportunityID,
		"rfq_id":                 item.RFQID,
		"sample_no":              item.SampleNo,
		"bom_snapshot_id":        item.BOMSnapshotID,
		"stage":                  normalizeMfgSampleStage(item.Stage),
		"risk_level":             item.RiskLevel,
		"owner_id":               item.OwnerID,
		"owner_name":             userDisplayName(item.TenantID, item.OwnerID),
		"engineering_owner_id":   item.EngineeringOwnerID,
		"engineering_owner_name": userDisplayName(item.TenantID, item.EngineeringOwnerID),
		"planned_finish_at":      item.PlannedFinishAt,
		"actual_finish_at":       item.ActualFinishAt,
		"customer_feedback":      item.CustomerFeedback,
		"mass_production_status": item.MassProductionStatus,
		"created_by":             item.CreatedBy,
		"updated_by":             item.UpdatedBy,
	}
	addMfgRFQContext(payload, item.TenantID, item.RFQID)
	addMfgBOMSnapshotContext(payload, item.TenantID, item.BOMSnapshotID)
	var count int64
	if utils.DB != nil {
		_ = utils.DB.Model(&models.MfgSampleProgressHistory{}).Where("tenant_id = ? AND sample_project_id = ?", item.TenantID, item.ID).Count(&count).Error
	}
	payload["progress_history_count"] = count
	return payload
}

func enrichMfgECNValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case []models.MfgECN:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, enrichMfgECN(item))
		}
		return items
	case models.MfgECN:
		return enrichMfgECN(typed)
	default:
		return value
	}
}

func enrichMfgECN(item models.MfgECN) map[string]interface{} {
	payload := map[string]interface{}{
		"id":                   item.ID,
		"created_at":           item.CreatedAt,
		"updated_at":           item.UpdatedAt,
		"tenant_id":            item.TenantID,
		"ecn_no":               item.ECNNo,
		"order_id":             item.OrderID,
		"order_name":           crmObjectName(item.TenantID, item.OrderID),
		"old_bom_snapshot_id":  item.OldBOMSnapshotID,
		"new_bom_snapshot_id":  item.NewBOMSnapshotID,
		"change_reason":        item.ChangeReason,
		"changed_fields_json":  item.ChangedFieldsJSON,
		"bom_diff_json":        item.BOMDiffJSON,
		"impact_level":         item.ImpactLevel,
		"cost_impact":          item.CostImpact,
		"cost_impact_currency": item.CostImpactCurrency,
		"delivery_impact_days": item.DeliveryImpactDays,
		"impact_scope":         item.ImpactScope,
		"quality_impact":       item.QualityImpact,
		"approval_status":      firstNonEmpty(item.ApprovalStatus, "draft"),
		"customer_confirmed":   item.CustomerConfirmed,
		"sync_status":          item.SyncStatus,
		"status":               firstNonEmpty(item.Status, "draft"),
		"created_by":           item.CreatedBy,
		"updated_by":           item.UpdatedBy,
		"approval_histories":   listMfgApprovalHistories(item.TenantID, "ecns", item.ID),
	}
	addMfgBOMSnapshotContextWithPrefix(payload, item.TenantID, item.OldBOMSnapshotID, "old_")
	addMfgBOMSnapshotContextWithPrefix(payload, item.TenantID, item.NewBOMSnapshotID, "new_")
	return payload
}

func enrichMfgCompensationRecordValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case []models.MfgCompensationRecord:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, enrichMfgCompensationRecord(item))
		}
		return items
	case models.MfgCompensationRecord:
		return enrichMfgCompensationRecord(typed)
	default:
		return value
	}
}

func enrichMfgCompensationRecord(item models.MfgCompensationRecord) map[string]interface{} {
	customerID, salesOwnerID := resolveMfgCRMOwnership(item.TenantID, item.OrderID, item.CustomerID, item.SalesOwnerID)
	payload := map[string]interface{}{
		"id":                   item.ID,
		"created_at":           item.CreatedAt,
		"updated_at":           item.UpdatedAt,
		"tenant_id":            item.TenantID,
		"compensation_no":      item.CompensationNo,
		"order_id":             item.OrderID,
		"order_name":           crmObjectName(item.TenantID, item.OrderID),
		"customer_id":          customerID,
		"customer_name":        crmObjectName(item.TenantID, customerID),
		"sales_owner_id":       salesOwnerID,
		"sales_owner_name":     userDisplayName(item.TenantID, salesOwnerID),
		"reason":               item.Reason,
		"amount":               item.Amount,
		"currency":             item.Currency,
		"responsibility_split": item.ResponsibilitySplit,
		"improvement_actions":  item.ImprovementActions,
		"compensation_date":    item.CompensationDate,
		"status":               item.Status,
		"approval_status":      firstNonEmpty(item.ApprovalStatus, "draft"),
		"approved_by":          item.ApprovedBy,
		"approved_by_name":     userDisplayName(item.TenantID, item.ApprovedBy),
		"approved_at":          item.ApprovedAt,
		"approval_remark":      item.ApprovalRemark,
		"created_by":           item.CreatedBy,
		"updated_by":           item.UpdatedBy,
	}
	histories := listMfgApprovalHistories(item.TenantID, "compensation-records", item.ID)
	if len(histories) == 0 {
		histories = fallbackMfgCompensationApprovalHistoriesFromAuditFields(item)
	}
	payload["approval_histories"] = histories
	return payload
}

func resolveMfgCRMOwnership(tenantID uint, orderID uint, customerID uint, salesOwnerID uint) (uint, uint) {
	if orderID == 0 {
		return customerID, salesOwnerID
	}
	var order models.CRMObject
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, orderID).First(&order).Error; err != nil {
		return customerID, salesOwnerID
	}
	if customerID == 0 {
		customerID = order.CompanyID
	}
	if salesOwnerID == 0 {
		salesOwnerID = order.OwnerID
	}
	return customerID, salesOwnerID
}

func crmObjectName(tenantID uint, id uint) string {
	if id == 0 || utils.DB == nil {
		return ""
	}
	var obj models.CRMObject
	if err := utils.DB.Select("id", "name", "code").Where("tenant_id = ? AND id = ?", tenantID, id).First(&obj).Error; err != nil {
		return ""
	}
	return firstNonEmpty(obj.Name, obj.Code)
}

func crmObjectCode(tenantID uint, id uint) string {
	if id == 0 || utils.DB == nil {
		return ""
	}
	var obj models.CRMObject
	if err := utils.DB.Select("id", "code").Where("tenant_id = ? AND id = ?", tenantID, id).First(&obj).Error; err != nil {
		return ""
	}
	return obj.Code
}

func userDisplayName(tenantID uint, id uint) string {
	if id == 0 || utils.DB == nil {
		return ""
	}
	var user models.User
	if err := utils.DB.Select("id", "name", "username").Where("tenant_id = ? AND id = ?", tenantID, id).First(&user).Error; err != nil {
		return ""
	}
	return firstNonEmpty(user.Name, user.Username, fmt.Sprintf("#%d", id))
}

func enrichMfgBOMSnapshot(item models.MfgBOMSnapshot) map[string]interface{} {
	return enrichMfgBOMSnapshotForAccess(item, MfgAccessContext{IsAdmin: true})
}

func enrichMfgBOMSnapshotForAccess(item models.MfgBOMSnapshot, access MfgAccessContext) map[string]interface{} {
	bomLines := bomLinesFromJSON(item.BOMLinesJSON)
	payload := map[string]interface{}{
		"id":                    item.ID,
		"created_at":            item.CreatedAt,
		"updated_at":            item.UpdatedAt,
		"tenant_id":             item.TenantID,
		"snapshot_no":           item.SnapshotNo,
		"business_name":         item.BusinessName,
		"source_template_id":    item.SourceTemplateID,
		"source_config_version": item.SourceConfigVersion,
		"rfq_id":                item.RFQID,
		"quote_id":              item.QuoteID,
		"order_id":              item.OrderID,
		"config_values_json":    item.ConfigValuesJSON,
		"bom_lines_json":        item.BOMLinesJSON,
		"bom_lines":             bomLines,
		"bom_line_count":        len(bomLines),
		"status":                item.Status,
		"created_by":            item.CreatedBy,
		"updated_by":            item.UpdatedBy,
	}
	addMfgRFQContext(payload, item.TenantID, item.RFQID)
	addMfgLatestCostContext(payload, item.TenantID, item.RFQID, item.ID)
	addMfgLatestQuoteContext(payload, item.TenantID, item.RFQID, item.ID)
	if !canViewMfgPriceForRFQ(item.TenantID, item.RFQID, access) {
		maskMfgPayloadPrices(payload)
	}
	return payload
}

func enrichMfgQuoteRevisionValue(value interface{}) interface{} {
	return enrichMfgQuoteRevisionValueForAccess(value, MfgAccessContext{IsAdmin: true})
}

func enrichMfgQuoteRevisionValueForAccess(value interface{}, access MfgAccessContext) interface{} {
	switch typed := value.(type) {
	case []models.MfgQuoteRevision:
		items := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, enrichMfgQuoteRevisionForAccess(item, access))
		}
		return items
	case models.MfgQuoteRevision:
		return enrichMfgQuoteRevisionForAccess(typed, access)
	default:
		return value
	}
}

func enrichMfgQuoteRevision(item models.MfgQuoteRevision) map[string]interface{} {
	return enrichMfgQuoteRevisionForAccess(item, MfgAccessContext{IsAdmin: true})
}

func enrichMfgQuoteRevisionForAccess(item models.MfgQuoteRevision, access MfgAccessContext) map[string]interface{} {
	payload := map[string]interface{}{
		"id":                  item.ID,
		"created_at":          item.CreatedAt,
		"updated_at":          item.UpdatedAt,
		"tenant_id":           item.TenantID,
		"quote_no":            item.QuoteNo,
		"revision":            item.Revision,
		"rfq_id":              item.RFQID,
		"cost_sheet_id":       item.CostSheetID,
		"bom_snapshot_id":     item.BOMSnapshotID,
		"amount":              item.Amount,
		"quote_amount":        item.Amount,
		"currency":            item.Currency,
		"terms_json":          item.TermsJSON,
		"approval_status":     item.ApprovalStatus,
		"approved_by":         item.ApprovedBy,
		"approved_by_name":    userDisplayName(item.TenantID, item.ApprovedBy),
		"approved_at":         item.ApprovedAt,
		"rejected_by":         item.RejectedBy,
		"rejected_by_name":    userDisplayName(item.TenantID, item.RejectedBy),
		"rejected_at":         item.RejectedAt,
		"send_status":         item.SendStatus,
		"customer_viewed_at":  item.CustomerViewedAt,
		"customer_view_count": item.CustomerViewCount,
		"status":              item.Status,
		"created_by":          item.CreatedBy,
		"updated_by":          item.UpdatedBy,
	}
	addMfgRFQContext(payload, item.TenantID, item.RFQID)
	addMfgBOMSnapshotContext(payload, item.TenantID, item.BOMSnapshotID)
	addMfgCostSheetContextByID(payload, item.TenantID, item.CostSheetID)
	payload["is_latest_revision"] = isLatestMfgQuoteRevision(item)
	approvalHistories := enrichMfgQuoteApprovalHistories(listMfgQuoteApprovalHistories(item.TenantID, item.ID))
	if len(approvalHistories) == 0 {
		approvalHistories = fallbackMfgQuoteApprovalHistoriesFromOperationLogs(item)
	}
	if len(approvalHistories) == 0 {
		approvalHistories = fallbackMfgQuoteApprovalHistoriesFromAuditFields(item)
	}
	payload["approval_histories"] = approvalHistories
	if !canViewMfgPriceForRFQ(item.TenantID, item.RFQID, access) {
		maskMfgPayloadPrices(payload)
	}
	return payload
}

func listMfgQuoteApprovalHistories(tenantID uint, quoteID uint) []models.MfgQuoteApprovalHistory {
	if quoteID == 0 {
		return []models.MfgQuoteApprovalHistory{}
	}
	var items []models.MfgQuoteApprovalHistory
	if err := utils.DB.Where("tenant_id = ? AND quote_id = ?", tenantID, quoteID).
		Order("operated_at desc, id desc").
		Find(&items).Error; err != nil {
		return []models.MfgQuoteApprovalHistory{}
	}
	return items
}

func enrichMfgQuoteApprovalHistories(items []models.MfgQuoteApprovalHistory) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]interface{}{
			"id":                  item.ID,
			"created_at":          item.CreatedAt,
			"tenant_id":           item.TenantID,
			"rfq_id":              item.RFQID,
			"quote_id":            item.QuoteID,
			"quote_no":            item.QuoteNo,
			"revision":            item.Revision,
			"action":              item.Action,
			"old_approval_status": item.OldApprovalStatus,
			"new_approval_status": item.NewApprovalStatus,
			"remark":              item.Remark,
			"operator_id":         item.OperatorID,
			"operator_name":       firstNonEmpty(item.OperatorName, fmt.Sprintf("#%d", item.OperatorID)),
			"operated_at":         item.OperatedAt,
		})
	}
	return result
}

func isMfgApprovalHistoryAction(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "submit-approval", "approve", "reject":
		return true
	default:
		return false
	}
}

func approvalHistoryActionFromWorkflowAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "submit-approval":
		return "submit"
	case "approve":
		return "approve"
	case "reject":
		return "reject"
	default:
		return action
	}
}

func createMfgApprovalHistoryTx(tx *gorm.DB, tenantID uint, resource string, resourceID uint, resourceNo string, action string, oldStatus string, newStatus string, remark string, operatorID uint, operatedAt time.Time) error {
	history := models.MfgApprovalHistory{
		TenantID:          tenantID,
		Resource:          resource,
		ResourceID:        resourceID,
		ResourceNo:        resourceNo,
		Action:            action,
		OldApprovalStatus: firstNonEmpty(oldStatus, "draft"),
		NewApprovalStatus: firstNonEmpty(newStatus, "draft"),
		Remark:            strings.TrimSpace(remark),
		OperatorID:        operatorID,
		OperatorName:      firstNonEmpty(userDisplayName(tenantID, operatorID), fmt.Sprintf("#%d", operatorID)),
		OperatedAt:        operatedAt,
	}
	return tx.Create(&history).Error
}

func listMfgApprovalHistories(tenantID uint, resource string, resourceID uint) []map[string]interface{} {
	if resourceID == 0 || utils.DB == nil {
		return []map[string]interface{}{}
	}
	var items []models.MfgApprovalHistory
	if err := utils.DB.Where("tenant_id = ? AND resource = ? AND resource_id = ?", tenantID, resource, resourceID).
		Order("operated_at desc, id desc").
		Find(&items).Error; err != nil {
		return []map[string]interface{}{}
	}
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]interface{}{
			"id":                  item.ID,
			"created_at":          item.CreatedAt,
			"tenant_id":           item.TenantID,
			"resource":            item.Resource,
			"resource_id":         item.ResourceID,
			"resource_no":         item.ResourceNo,
			"action":              item.Action,
			"old_approval_status": item.OldApprovalStatus,
			"new_approval_status": item.NewApprovalStatus,
			"remark":              item.Remark,
			"operator_id":         item.OperatorID,
			"operator_name":       firstNonEmpty(item.OperatorName, fmt.Sprintf("#%d", item.OperatorID)),
			"operated_at":         item.OperatedAt,
		})
	}
	return result
}

func fallbackMfgCompensationApprovalHistoriesFromAuditFields(item models.MfgCompensationRecord) []map[string]interface{} {
	status := strings.ToLower(strings.TrimSpace(item.ApprovalStatus))
	if !stringIn(status, "approved", "rejected") || item.ApprovedAt == nil {
		return []map[string]interface{}{}
	}
	action := "approve"
	if status == "rejected" {
		action = "reject"
	}
	return []map[string]interface{}{{
		"id":                  0,
		"tenant_id":           item.TenantID,
		"resource":            "compensation-records",
		"resource_id":         item.ID,
		"resource_no":         item.CompensationNo,
		"action":              action,
		"old_approval_status": "pending",
		"new_approval_status": status,
		"remark":              item.ApprovalRemark,
		"operator_id":         item.ApprovedBy,
		"operator_name":       firstNonEmpty(userDisplayName(item.TenantID, item.ApprovedBy), fmt.Sprintf("#%d", item.ApprovedBy)),
		"operated_at":         item.ApprovedAt,
	}}
}

func fallbackMfgQuoteApprovalHistoriesFromOperationLogs(item models.MfgQuoteRevision) []map[string]interface{} {
	if utils.DB == nil || item.ID == 0 {
		return []map[string]interface{}{}
	}
	operations := []string{"SUBMIT_MFG_QUOTE_APPROVAL", "APPROVE_MFG_QUOTE", "REJECT_MFG_QUOTE"}
	var logs []models.OperationLog
	if err := utils.DB.Where("tenant_id = ? AND target = ? AND operation IN ? AND result = ?", item.TenantID, fmt.Sprintf("mfg/quotes[%d]", item.ID), operations, "success").
		Order("created_at desc, id desc").
		Find(&logs).Error; err != nil {
		return []map[string]interface{}{}
	}
	result := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		action, oldStatus, newStatus := quoteApprovalHistoryStateFromOperation(log.Operation)
		if action == "" {
			continue
		}
		params := map[string]interface{}{}
		if strings.TrimSpace(log.ParamsJSON) != "" {
			_ = json.Unmarshal([]byte(log.ParamsJSON), &params)
		}
		if value := strings.TrimSpace(fmt.Sprint(params["approval_status"])); value != "" && value != "<nil>" {
			newStatus = value
		}
		remark := ""
		if value, ok := params["remark"]; ok {
			remark = strings.TrimSpace(fmt.Sprint(value))
		}
		result = append(result, map[string]interface{}{
			"id":                  0,
			"tenant_id":           item.TenantID,
			"rfq_id":              item.RFQID,
			"quote_id":            item.ID,
			"quote_no":            item.QuoteNo,
			"revision":            item.Revision,
			"action":              action,
			"old_approval_status": oldStatus,
			"new_approval_status": newStatus,
			"remark":              remark,
			"operator_id":         log.UserID,
			"operator_name":       firstNonEmpty(log.Username, userDisplayName(item.TenantID, log.UserID), fmt.Sprintf("#%d", log.UserID)),
			"operated_at":         log.CreatedAt,
		})
	}
	return result
}

func quoteApprovalHistoryStateFromOperation(operation string) (string, string, string) {
	switch strings.ToUpper(strings.TrimSpace(operation)) {
	case "SUBMIT_MFG_QUOTE_APPROVAL":
		return "submit", "draft", "pending"
	case "APPROVE_MFG_QUOTE":
		return "approve", "pending", "approved"
	case "REJECT_MFG_QUOTE":
		return "reject", "pending", "rejected"
	default:
		return "", "", ""
	}
}

func fallbackMfgQuoteApprovalHistoriesFromAuditFields(item models.MfgQuoteRevision) []map[string]interface{} {
	status := strings.ToLower(strings.TrimSpace(item.ApprovalStatus))
	if status == "approved" && item.ApprovedAt != nil {
		return []map[string]interface{}{{
			"id":                  0,
			"tenant_id":           item.TenantID,
			"rfq_id":              item.RFQID,
			"quote_id":            item.ID,
			"quote_no":            item.QuoteNo,
			"revision":            item.Revision,
			"action":              "approve",
			"old_approval_status": "pending",
			"new_approval_status": "approved",
			"remark":              "",
			"operator_id":         item.ApprovedBy,
			"operator_name":       firstNonEmpty(userDisplayName(item.TenantID, item.ApprovedBy), fmt.Sprintf("#%d", item.ApprovedBy)),
			"operated_at":         item.ApprovedAt,
		}}
	}
	if status == "rejected" && item.RejectedAt != nil {
		return []map[string]interface{}{{
			"id":                  0,
			"tenant_id":           item.TenantID,
			"rfq_id":              item.RFQID,
			"quote_id":            item.ID,
			"quote_no":            item.QuoteNo,
			"revision":            item.Revision,
			"action":              "reject",
			"old_approval_status": "pending",
			"new_approval_status": "rejected",
			"remark":              "",
			"operator_id":         item.RejectedBy,
			"operator_name":       firstNonEmpty(userDisplayName(item.TenantID, item.RejectedBy), fmt.Sprintf("#%d", item.RejectedBy)),
			"operated_at":         item.RejectedAt,
		}}
	}
	return []map[string]interface{}{}
}

func addMfgRFQContext(payload map[string]interface{}, tenantID uint, rfqID uint) {
	if rfqID == 0 || utils.DB == nil {
		return
	}
	var rfq models.MfgRFQ
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, rfqID).First(&rfq).Error; err != nil {
		return
	}
	rfqPayload := jsonMap(rfq.PayloadJSON)
	payload["rfq_no"] = rfq.RFQNo
	payload["rfq_business_name"] = rfq.BusinessName
	payload["customer_id"] = rfq.CustomerID
	payload["contact_id"] = rfq.ContactID
	payload["opportunity_id"] = rfq.OpportunityID
	payload["product_family_id"] = rfq.ProductFamilyID
	payload["quantity"] = rfq.Quantity
	payload["target_price"] = rfq.TargetPrice
	payload["required_delivery_date"] = rfq.RequiredDeliveryDate
	payload["drawing_file_id"] = rfq.DrawingFileID
	payload["drawing_storage_node_id"] = rfq.DrawingStorageNodeID
	payload["no_drawing_reason"] = rfq.NoDrawingReason
	payload["rfq_status"] = rfq.Status
	payload["rfq_payload"] = rfqPayload
	payload["customer_name"] = firstNonEmpty(mfgPayloadString(rfqPayload, "customer_name"), crmObjectName(tenantID, rfq.CustomerID))
	payload["contact_name"] = firstNonEmpty(mfgPayloadString(rfqPayload, "contact_name"), crmObjectName(tenantID, rfq.ContactID))
	payload["opportunity_name"] = firstNonEmpty(mfgPayloadString(rfqPayload, "opportunity_name"), crmObjectName(tenantID, rfq.OpportunityID))
	payload["product_name"] = firstNonEmpty(mfgPayloadString(rfqPayload, "product_name"), mfgPayloadString(rfqPayload, "custom_product_name"), mfgPayloadString(rfqPayload, "sku"))
	payload["currency"] = firstNonEmpty(stringFromMfgInput(payload["currency"]), mfgPayloadString(rfqPayload, "currency"), "CNY")
}

func addMfgLatestCostContext(payload map[string]interface{}, tenantID uint, rfqID uint, bomSnapshotID uint) {
	if utils.DB == nil {
		return
	}
	var cost models.MfgCostSheet
	query := utils.DB.Where("tenant_id = ?", tenantID)
	if bomSnapshotID > 0 {
		query = query.Where("bom_snapshot_id = ?", bomSnapshotID)
	} else if rfqID > 0 {
		query = query.Where("rfq_id = ?", rfqID)
	} else {
		return
	}
	if err := query.Order("updated_at desc, id desc").First(&cost).Error; err != nil {
		return
	}
	payload["latest_cost_sheet_id"] = cost.ID
	payload["cost_sheet_status"] = cost.Status
	payload["margin_level"] = cost.MarginLevel
	if cost.Amount > 0 {
		payload["quote_amount"] = cost.Amount
		payload["amount"] = cost.Amount
	}
}

func addMfgLatestQuoteContext(payload map[string]interface{}, tenantID uint, rfqID uint, bomSnapshotID uint) {
	if utils.DB == nil {
		return
	}
	var quote models.MfgQuoteRevision
	query := utils.DB.Where("tenant_id = ?", tenantID)
	if bomSnapshotID > 0 {
		query = query.Where("bom_snapshot_id = ?", bomSnapshotID)
	} else if rfqID > 0 {
		query = query.Where("rfq_id = ?", rfqID)
	} else {
		return
	}
	if err := query.Order("revision desc, updated_at desc, id desc").First(&quote).Error; err != nil {
		return
	}
	payload["latest_quote_id"] = quote.ID
	payload["latest_quote_no"] = quote.QuoteNo
	payload["latest_quote_revision"] = quote.Revision
	if quote.Amount > 0 {
		payload["quote_amount"] = quote.Amount
		payload["amount"] = quote.Amount
	}
}

func addMfgBOMSnapshotContext(payload map[string]interface{}, tenantID uint, bomSnapshotID uint) {
	if bomSnapshotID == 0 || utils.DB == nil {
		return
	}
	var snapshot models.MfgBOMSnapshot
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, bomSnapshotID).First(&snapshot).Error; err != nil {
		return
	}
	payload["bom_snapshot_no"] = snapshot.SnapshotNo
	payload["bom_business_name"] = snapshot.BusinessName
	payload["bom_line_count"] = len(bomLinesFromJSON(snapshot.BOMLinesJSON))
	if uintFromMfgInput(payload["rfq_id"]) == 0 && snapshot.RFQID > 0 {
		payload["rfq_id"] = snapshot.RFQID
		addMfgRFQContext(payload, tenantID, snapshot.RFQID)
	}
}

func addMfgBOMSnapshotContextWithPrefix(payload map[string]interface{}, tenantID uint, bomSnapshotID uint, prefix string) {
	if bomSnapshotID == 0 || utils.DB == nil {
		return
	}
	var snapshot models.MfgBOMSnapshot
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, bomSnapshotID).First(&snapshot).Error; err != nil {
		return
	}
	payload[prefix+"bom_snapshot_no"] = snapshot.SnapshotNo
	payload[prefix+"bom_business_name"] = snapshot.BusinessName
}

func addMfgCostSheetContextByID(payload map[string]interface{}, tenantID uint, costSheetID uint) {
	if costSheetID == 0 || utils.DB == nil {
		return
	}
	var cost models.MfgCostSheet
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, costSheetID).First(&cost).Error; err != nil {
		return
	}
	payload["cost_sheet_status"] = cost.Status
	payload["margin_level"] = cost.MarginLevel
	payload["gross_margin_rate"] = cost.GrossMarginRate
	payload["cost_sheet_label"] = mfgCostSheetLabel(tenantID, cost)
	if cost.Amount > 0 && floatFromMfgInput(payload["amount"]) <= 0 {
		payload["amount"] = cost.Amount
		payload["quote_amount"] = cost.Amount
	}
	if uintFromMfgInput(payload["rfq_id"]) == 0 && cost.RFQID > 0 {
		payload["rfq_id"] = cost.RFQID
		addMfgRFQContext(payload, tenantID, cost.RFQID)
	}
}

func mfgCostSheetLabel(tenantID uint, cost models.MfgCostSheet) string {
	context := ""
	if cost.RFQID > 0 {
		var rfq models.MfgRFQ
		if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, cost.RFQID).First(&rfq).Error; err == nil {
			context = rfq.RFQNo
		}
	}
	if context == "" && cost.BOMSnapshotID > 0 {
		var snapshot models.MfgBOMSnapshot
		if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, cost.BOMSnapshotID).First(&snapshot).Error; err == nil {
			context = snapshot.SnapshotNo
		}
	}
	parts := []string{"成本核价"}
	if strings.TrimSpace(context) != "" {
		parts = append(parts, strings.TrimSpace(context))
	}
	if strings.TrimSpace(cost.Status) != "" {
		parts = append(parts, strings.TrimSpace(cost.Status))
	}
	return strings.Join(parts, " - ")
}

func bomLinesFromJSON(raw string) []interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var lines []interface{}
	if err := json.Unmarshal([]byte(raw), &lines); err == nil {
		return lines
	}
	wrapped := jsonMap(raw)
	if items, ok := wrapped["items"].([]interface{}); ok {
		return items
	}
	if items, ok := wrapped["lines"].([]interface{}); ok {
		return items
	}
	return nil
}

func isLatestMfgQuoteRevision(item models.MfgQuoteRevision) bool {
	if utils.DB == nil {
		return true
	}
	var latest models.MfgQuoteRevision
	query := utils.DB.Where("tenant_id = ?", item.TenantID)
	if item.RFQID > 0 {
		query = query.Where("rfq_id = ?", item.RFQID)
	} else if item.QuoteNo != "" {
		query = query.Where("quote_no = ?", item.QuoteNo)
	} else {
		return true
	}
	if err := query.Order("revision desc, id desc").First(&latest).Error; err != nil {
		return true
	}
	return latest.ID == item.ID
}

func sanitizeMfgSparePartQuote(item models.MfgSparePartQuote) map[string]interface{} {
	return map[string]interface{}{
		"id":                item.ID,
		"created_at":        item.CreatedAt,
		"updated_at":        item.UpdatedAt,
		"tenant_id":         item.TenantID,
		"service_ticket_id": item.ServiceTicketID,
		"asset_id":          item.AssetID,
		"quote_no":          item.QuoteNo,
		"amount":            item.Amount,
		"status":            item.Status,
		"line_items_masked": true,
	}
}

func sanitizeMfgAssetForCustomer(item models.MfgAsset) map[string]interface{} {
	return map[string]interface{}{
		"id":               item.ID,
		"created_at":       item.CreatedAt,
		"updated_at":       item.UpdatedAt,
		"customer_id":      item.CustomerID,
		"order_id":         item.OrderID,
		"serial_no":        item.SerialNo,
		"batch_no":         item.BatchNo,
		"product_id":       item.ProductID,
		"warranty_start":   item.WarrantyStart,
		"warranty_end":     item.WarrantyEnd,
		"install_location": item.InstallLocation,
		"status":           item.Status,
		"sensitive_masked": true,
	}
}

func sanitizeMfgServiceTicketForCustomer(item models.MfgServiceTicket) map[string]interface{} {
	return map[string]interface{}{
		"id":                    item.ID,
		"created_at":            item.CreatedAt,
		"updated_at":            item.UpdatedAt,
		"asset_id":              item.AssetID,
		"customer_id":           item.CustomerID,
		"fault_code":            item.FaultCode,
		"service_type":          item.ServiceType,
		"service_result":        item.ServiceResult,
		"service_to_sales_flag": item.ServiceToSalesFlag,
		"opportunity_id":        item.OpportunityID,
		"status":                item.Status,
	}
}

func sanitizeMfgQuoteForCustomer(item models.MfgQuoteRevision) map[string]interface{} {
	return map[string]interface{}{
		"id":                  item.ID,
		"quote_no":            item.QuoteNo,
		"revision":            item.Revision,
		"rfq_id":              item.RFQID,
		"amount":              item.Amount,
		"currency":            item.Currency,
		"terms":               jsonMap(item.TermsJSON),
		"approval_status":     item.ApprovalStatus,
		"send_status":         item.SendStatus,
		"customer_viewed_at":  item.CustomerViewedAt,
		"customer_view_count": item.CustomerViewCount,
		"status":              item.Status,
	}
}

func mfgDashboardMetricGroups(counts map[string]int64, quoteAmount float64, quoteSnapshot models.MetricMfgQuoteSnapshot, deliverySnapshot models.MetricMfgDeliverySnapshot, serviceSnapshot models.MetricMfgServiceSnapshot) []map[string]interface{} {
	dataUpdatedAt := latestTime(quoteSnapshot.UpdatedAt, deliverySnapshot.UpdatedAt, serviceSnapshot.UpdatedAt)
	groups := []map[string]interface{}{
		{
			"key":            "growth",
			"drilldown":      "growth",
			"updated_at":     dataUpdatedAt,
			"refresh_status": quoteSnapshot.RefreshStatus,
			"refresh_error":  quoteSnapshot.RefreshError,
			"metrics": map[string]interface{}{
				"rfq_count":      counts["rfq_count"],
				"open_rfq_count": counts["open_rfq_count"],
				"quote_amount":   quoteAmount,
			},
		},
		{
			"key":            "efficiency",
			"drilldown":      "quote",
			"updated_at":     quoteSnapshot.UpdatedAt,
			"refresh_status": quoteSnapshot.RefreshStatus,
			"refresh_error":  quoteSnapshot.RefreshError,
			"metrics": map[string]interface{}{
				"avg_quote_cycle_hours": quoteSnapshot.AvgQuoteCycleHours,
				"costing_cycle_hours":   quoteSnapshot.CostingCycleHours,
				"approval_cycle_hours":  quoteSnapshot.ApprovalCycleHours,
			},
		},
		{
			"key":            "quality",
			"drilldown":      "quality",
			"updated_at":     serviceSnapshot.UpdatedAt,
			"refresh_status": serviceSnapshot.RefreshStatus,
			"refresh_error":  serviceSnapshot.RefreshError,
			"metrics": map[string]interface{}{
				"ticket_count":       counts["service_ticket_count"],
				"first_fix_rate":     serviceSnapshot.FirstFixRate,
				"service_exceptions": serviceSnapshot.TicketCount,
			},
		},
		{
			"key":            "delivery",
			"drilldown":      "delivery",
			"updated_at":     deliverySnapshot.UpdatedAt,
			"refresh_status": deliverySnapshot.RefreshStatus,
			"refresh_error":  deliverySnapshot.RefreshError,
			"metrics": map[string]interface{}{
				"green_count":  deliverySnapshot.GreenCount,
				"yellow_count": deliverySnapshot.YellowCount,
				"red_count":    deliverySnapshot.RedCount,
				"on_time_rate": deliverySnapshot.OnTimeRate,
			},
		},
		{
			"key":            "profit",
			"drilldown":      "profit",
			"updated_at":     quoteSnapshot.UpdatedAt,
			"refresh_status": quoteSnapshot.RefreshStatus,
			"refresh_error":  quoteSnapshot.RefreshError,
			"metrics": map[string]interface{}{
				"low_margin_quote_count": counts["low_margin_quote_count"],
				"low_margin_quote_rate":  quoteSnapshot.LowMarginQuoteRate,
				"quote_amount":           quoteAmount,
			},
		},
		{
			"key":            "cash",
			"drilldown":      "cash",
			"updated_at":     dataUpdatedAt,
			"refresh_status": firstNonEmpty(quoteSnapshot.RefreshStatus, "manual"),
			"refresh_error":  quoteSnapshot.RefreshError,
			"metrics": map[string]interface{}{
				"customer_snapshot_target": "customer-metrics",
				"quote_amount":             quoteAmount,
			},
		},
		{
			"key":            "lifecycle",
			"drilldown":      "lifecycle",
			"updated_at":     serviceSnapshot.UpdatedAt,
			"refresh_status": serviceSnapshot.RefreshStatus,
			"refresh_error":  serviceSnapshot.RefreshError,
			"metrics": map[string]interface{}{
				"asset_count":                 counts["asset_count"],
				"asset_repurchase_rate":       serviceSnapshot.AssetRepurchaseRate,
				"warranty_expiring_count":     serviceSnapshot.WarrantyExpiringCount,
				"service_to_opportunity_rate": serviceSnapshot.ServiceToOpportunityRate,
			},
		},
	}
	return groups
}

func latestTime(values ...time.Time) time.Time {
	var latest time.Time
	for _, value := range values {
		if value.After(latest) {
			latest = value
		}
	}
	return latest
}

func mfgOrderBOMLocked(order models.CRMObject) bool {
	payload := payloadFromJSON(order.PayloadJSON)
	value, ok := payload["mfg_bom_locked"].(bool)
	return ok && value
}

func uintFromPayload(raw string, key string) uint {
	return uintFromMap(payloadFromJSON(raw), key)
}

func uintFromMap(payload map[string]interface{}, key string) uint {
	return uintFromMfgInput(payload[key])
}

func intFromAny(value interface{}) int {
	return int(floatFromMfgInput(value))
}

func mfgQuoteCustomerID(tenantID uint, rfqID uint) uint {
	if rfqID == 0 {
		return 0
	}
	rfq, err := loadMfgRFQ(tenantID, rfqID)
	if err != nil {
		return 0
	}
	return rfq.CustomerID
}

func loadMfgOrder(tenantID uint, orderID uint) (models.CRMObject, error) {
	var order models.CRMObject
	if err := utils.DB.Where("tenant_id = ? AND id = ? AND domain = ? AND object_type = ?", tenantID, orderID, "orders", "order").First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return order, errors.New(MfgErrorNotFound)
		}
		return order, err
	}
	return order, nil
}

func findMfgAssetForAlarm(tenantID uint, assetID uint, iotAssetRef string) (models.MfgAsset, error) {
	var asset models.MfgAsset
	query := utils.DB.Where("tenant_id = ?", tenantID)
	if assetID > 0 {
		query = query.Where("id = ?", assetID)
	} else {
		iotAssetRef = strings.TrimSpace(iotAssetRef)
		if iotAssetRef == "" {
			return asset, errors.New(MfgErrorIoTUnauthorized)
		}
		query = query.Where("iot_asset_ref = ?", iotAssetRef)
	}
	if err := query.First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return asset, errors.New(MfgErrorNotFound)
		}
		return asset, err
	}
	if strings.TrimSpace(iotAssetRef) != "" && strings.TrimSpace(asset.IoTAssetRef) != strings.TrimSpace(iotAssetRef) {
		return asset, errors.New(MfgErrorIoTUnauthorized)
	}
	return asset, nil
}

func mfgCRMRelation(tenantID uint, sourceType string, sourceID uint, targetType string, targetID uint, relationType string) models.CRMRelation {
	return models.CRMRelation{
		TenantModel:  models.TenantModel{TenantID: tenantID},
		SourceType:   sourceType,
		SourceID:     sourceID,
		TargetType:   targetType,
		TargetID:     targetID,
		RelationType: relationType,
		PayloadJSON:  "{}",
	}
}

func enqueueMfgIntegrationOutboxTx(tx *gorm.DB, tenantID uint, userID uint, eventType string, targetSystem string, businessType string, businessID uint, payload map[string]interface{}) error {
	eventType = strings.TrimSpace(eventType)
	targetSystem = strings.TrimSpace(targetSystem)
	businessType = strings.TrimSpace(businessType)
	if eventType == "" || targetSystem == "" || businessType == "" || businessID == 0 {
		return errors.New(MfgErrorInvalidInput)
	}
	idempotencyKey := strings.Join([]string{
		fmt.Sprint(tenantID),
		eventType,
		targetSystem,
		businessType,
		fmt.Sprint(businessID),
	}, ":")
	item := models.MfgIntegrationOutbox{
		TenantID:          tenantID,
		EventType:         eventType,
		TargetSystem:      targetSystem,
		BusinessType:      businessType,
		BusinessID:        businessID,
		IdempotencyKey:    idempotencyKey,
		PayloadJSON:       jsonStringFromAny(payload),
		Status:            "pending",
		MaxRetryCount:     3,
		CompensationState: "pending",
		CreatedBy:         userID,
		UpdatedBy:         userID,
	}
	return tx.Where("tenant_id = ? AND idempotency_key = ?", tenantID, idempotencyKey).
		Attrs(item).
		FirstOrCreate(&item).Error
}

func safeRatio(numerator int64, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func mapFromMfgInput(value interface{}) map[string]interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed
	case string:
		result := map[string]interface{}{}
		_ = json.Unmarshal([]byte(typed), &result)
		return result
	default:
		return map[string]interface{}{}
	}
}

func uintFromMfgInput(value interface{}) uint {
	return uint(floatFromMfgInput(value))
}

func floatFromMfgInput(value interface{}) float64 {
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
	case uint64:
		return float64(typed)
	case string:
		var out float64
		_, _ = fmt.Sscanf(strings.TrimSpace(typed), "%f", &out)
		return out
	default:
		return 0
	}
}

func boolFromMfgInput(value interface{}) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "y", "on":
			return true
		default:
			return false
		}
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case uint:
		return typed != 0
	case uint64:
		return typed != 0
	case float64:
		return typed != 0
	default:
		return false
	}
}

func stringFromMfgInput(value interface{}) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func stringSliceFromAny(value interface{}) []string {
	switch typed := value.(type) {
	case []interface{}:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" && text != "<nil>" {
				items = append(items, text)
			}
		}
		return items
	case []string:
		return typed
	case string:
		if strings.Contains(typed, ",") {
			parts := strings.Split(typed, ",")
			items := make([]string, 0, len(parts))
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					items = append(items, part)
				}
			}
			return items
		}
		if strings.TrimSpace(typed) != "" {
			return []string{strings.TrimSpace(typed)}
		}
	}
	return nil
}

func firstAny(values ...interface{}) interface{} {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstUint(values ...uint) uint {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func stringIn(value string, candidates ...string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range candidates {
		if value == strings.ToLower(strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func derefInterface(value interface{}) interface{} {
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		return rv.Elem().Interface()
	}
	return value
}

func hasField(item interface{}, name string) bool {
	rv := reflect.Indirect(reflect.ValueOf(item))
	return rv.IsValid() && rv.FieldByName(name).IsValid()
}

func uintField(item interface{}, name string) uint {
	rv := reflect.Indirect(reflect.ValueOf(item))
	if !rv.IsValid() {
		return 0
	}
	field := rv.FieldByName(name)
	if !field.IsValid() {
		return 0
	}
	switch field.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint(field.Uint())
	default:
		return 0
	}
}

func stringField(item interface{}, name string) string {
	if name == "" {
		return ""
	}
	rv := reflect.Indirect(reflect.ValueOf(item))
	if !rv.IsValid() {
		return ""
	}
	field := rv.FieldByName(name)
	if !field.IsValid() || field.Kind() != reflect.String {
		return ""
	}
	return strings.TrimSpace(field.String())
}

func boolField(item interface{}, name string) bool {
	rv := reflect.Indirect(reflect.ValueOf(item))
	if !rv.IsValid() {
		return false
	}
	field := rv.FieldByName(name)
	if !field.IsValid() || field.Kind() != reflect.Bool {
		return false
	}
	return field.Bool()
}

func setUintField(item interface{}, name string, value uint) {
	rv := reflect.Indirect(reflect.ValueOf(item))
	if !rv.IsValid() {
		return
	}
	field := rv.FieldByName(name)
	if field.IsValid() && field.CanSet() && field.Kind() >= reflect.Uint && field.Kind() <= reflect.Uint64 {
		field.SetUint(uint64(value))
	}
}

func setStringField(item interface{}, name string, value string) {
	rv := reflect.Indirect(reflect.ValueOf(item))
	if !rv.IsValid() {
		return
	}
	field := rv.FieldByName(name)
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
		field.SetString(value)
	}
}

func setBoolField(item interface{}, name string, value bool) {
	rv := reflect.Indirect(reflect.ValueOf(item))
	if !rv.IsValid() {
		return
	}
	field := rv.FieldByName(name)
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.Bool {
		field.SetBool(value)
	}
}

func setTimePtrField(item interface{}, name string, value *time.Time) {
	rv := reflect.Indirect(reflect.ValueOf(item))
	if !rv.IsValid() {
		return
	}
	field := rv.FieldByName(name)
	if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.Ptr {
		return
	}
	if value == nil {
		field.Set(reflect.Zero(field.Type()))
		return
	}
	if field.Type() == reflect.TypeOf(&time.Time{}) {
		field.Set(reflect.ValueOf(value))
	}
}

func dedupeMfgFileReferences(refs []MfgFileReference) []MfgFileReference {
	seen := map[string]bool{}
	items := make([]MfgFileReference, 0, len(refs))
	for _, ref := range refs {
		ref.FileID = strings.TrimSpace(ref.FileID)
		ref.StorageNodeID = strings.TrimSpace(ref.StorageNodeID)
		if ref.FileID == "" || seen[ref.FileID] {
			continue
		}
		seen[ref.FileID] = true
		items = append(items, ref)
	}
	return items
}
