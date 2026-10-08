package services

import (
	"ai-crm/models"
	"ai-crm/utils"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	MfgErrorCustomerNotFound = "MFG_CUSTOMER_NOT_FOUND"
	MfgErrorContactNotFound  = "MFG_CONTACT_NOT_FOUND"
	MfgErrorInvalidInput     = "MFG_INVALID_INPUT"
)

type MfgCustomerProfileInput struct {
	RequiredCertifications    string                 `json:"required_certifications"`
	QualityStandard           string                 `json:"quality_standard"`
	PaymentTermType           string                 `json:"payment_term_type"`
	DesignatedBrandRequired   bool                   `json:"designated_brand_required"`
	MoldFeePolicy             string                 `json:"mold_fee_policy"`
	SpecialProcessRequirement string                 `json:"special_process_requirement"`
	PreferredInspectionAgency string                 `json:"preferred_inspection_agency"`
	PackagingRequirement      string                 `json:"packaging_requirement"`
	Extension                 map[string]interface{} `json:"extension"`
}

type MfgDecisionChainInput struct {
	ContactID          uint   `json:"contact_id"`
	RoleType           string `json:"role_type"`
	InfluenceLevel     string `json:"influence_level"`
	RelationshipStatus string `json:"relationship_status"`
	Notes              string `json:"notes"`
}

type MetricCustomerSnapshotInput struct {
	SnapshotDate     string                 `json:"snapshot_date"`
	WindowStart      *time.Time             `json:"window_start"`
	WindowEnd        *time.Time             `json:"window_end"`
	HealthScore      int                    `json:"health_score"`
	GrossMargin      float64                `json:"gross_margin"`
	GrossMarginRate  float64                `json:"gross_margin_rate"`
	RevenueAmount    float64                `json:"revenue_amount"`
	ReceivableAmount float64                `json:"receivable_amount"`
	OverdueAmount    float64                `json:"overdue_amount"`
	OnTimeRate       float64                `json:"on_time_rate"`
	ComplaintCount   int                    `json:"complaint_count"`
	ServiceValue     float64                `json:"service_value"`
	Source           map[string]interface{} `json:"source"`
	RefreshStatus    string                 `json:"refresh_status"`
	RefreshError     string                 `json:"refresh_error"`
}

func GetMfgCustomerProfile(tenantID uint, customerID uint) (models.MfgCustomerProfile, bool, error) {
	if err := ensureManufacturingCustomer(tenantID, customerID); err != nil {
		return models.MfgCustomerProfile{}, false, err
	}
	var item models.MfgCustomerProfile
	err := utils.DB.Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.MfgCustomerProfile{TenantID: tenantID, CustomerID: customerID}, false, nil
		}
		return models.MfgCustomerProfile{}, false, err
	}
	return item, true, nil
}

func SaveMfgCustomerProfile(tenantID uint, userID uint, customerID uint, input MfgCustomerProfileInput) (models.MfgCustomerProfile, error) {
	if err := ensureManufacturingCustomer(tenantID, customerID); err != nil {
		return models.MfgCustomerProfile{}, err
	}
	extensionJSON := "{}"
	if input.Extension != nil {
		raw, err := json.Marshal(input.Extension)
		if err != nil {
			return models.MfgCustomerProfile{}, err
		}
		extensionJSON = string(raw)
	}
	var item models.MfgCustomerProfile
	err := utils.DB.Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).First(&item).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.MfgCustomerProfile{}, err
	}
	item.TenantID = tenantID
	item.CustomerID = customerID
	item.RequiredCertifications = strings.TrimSpace(input.RequiredCertifications)
	item.QualityStandard = strings.TrimSpace(input.QualityStandard)
	item.PaymentTermType = strings.TrimSpace(input.PaymentTermType)
	item.DesignatedBrandRequired = input.DesignatedBrandRequired
	item.MoldFeePolicy = strings.TrimSpace(input.MoldFeePolicy)
	item.SpecialProcessRequirement = strings.TrimSpace(input.SpecialProcessRequirement)
	item.PreferredInspectionAgency = strings.TrimSpace(input.PreferredInspectionAgency)
	item.PackagingRequirement = strings.TrimSpace(input.PackagingRequirement)
	item.ExtensionJSON = extensionJSON
	item.UpdatedBy = userID
	if item.ID == 0 {
		item.CreatedBy = userID
		if err := utils.DB.Create(&item).Error; err != nil {
			return models.MfgCustomerProfile{}, err
		}
		return item, nil
	}
	return item, utils.DB.Save(&item).Error
}

func ListMfgDecisionChain(tenantID uint, customerID uint) ([]models.MfgDecisionChain, error) {
	if err := ensureManufacturingCustomer(tenantID, customerID); err != nil {
		return nil, err
	}
	var items []models.MfgDecisionChain
	err := utils.DB.Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Order("id asc").
		Find(&items).Error
	return items, err
}

func CreateMfgDecisionChain(tenantID uint, userID uint, customerID uint, input MfgDecisionChainInput) (models.MfgDecisionChain, error) {
	if err := ensureManufacturingCustomer(tenantID, customerID); err != nil {
		return models.MfgDecisionChain{}, err
	}
	if input.ContactID > 0 {
		if err := ensureManufacturingContact(tenantID, input.ContactID, customerID); err != nil {
			return models.MfgDecisionChain{}, err
		}
	}
	item := models.MfgDecisionChain{
		TenantID:           tenantID,
		CustomerID:         customerID,
		ContactID:          input.ContactID,
		RoleType:           normalizeDecisionRole(input.RoleType),
		InfluenceLevel:     normalizeInfluenceLevel(input.InfluenceLevel),
		RelationshipStatus: normalizeRelationshipStatus(input.RelationshipStatus),
		Notes:              strings.TrimSpace(input.Notes),
		CreatedBy:          userID,
		UpdatedBy:          userID,
	}
	if item.RoleType == "" {
		return models.MfgDecisionChain{}, errors.New(MfgErrorInvalidInput)
	}
	return item, utils.DB.Create(&item).Error
}

func UpdateMfgDecisionChain(tenantID uint, userID uint, customerID uint, id uint, input MfgDecisionChainInput) (models.MfgDecisionChain, error) {
	if err := ensureManufacturingCustomer(tenantID, customerID); err != nil {
		return models.MfgDecisionChain{}, err
	}
	var item models.MfgDecisionChain
	if err := utils.DB.Where("tenant_id = ? AND customer_id = ? AND id = ?", tenantID, customerID, id).First(&item).Error; err != nil {
		return models.MfgDecisionChain{}, err
	}
	if input.ContactID > 0 {
		if err := ensureManufacturingContact(tenantID, input.ContactID, customerID); err != nil {
			return models.MfgDecisionChain{}, err
		}
	}
	roleType := normalizeDecisionRole(input.RoleType)
	if roleType == "" {
		return models.MfgDecisionChain{}, errors.New(MfgErrorInvalidInput)
	}
	item.ContactID = input.ContactID
	item.RoleType = roleType
	item.InfluenceLevel = normalizeInfluenceLevel(input.InfluenceLevel)
	item.RelationshipStatus = normalizeRelationshipStatus(input.RelationshipStatus)
	item.Notes = strings.TrimSpace(input.Notes)
	item.UpdatedBy = userID
	return item, utils.DB.Save(&item).Error
}

func DeleteMfgDecisionChain(tenantID uint, customerID uint, id uint) error {
	if err := ensureManufacturingCustomer(tenantID, customerID); err != nil {
		return err
	}
	return utils.DB.Where("tenant_id = ? AND customer_id = ? AND id = ?", tenantID, customerID, id).
		Delete(&models.MfgDecisionChain{}).Error
}

func SaveMetricCustomerSnapshot(tenantID uint, userID uint, customerID uint, input MetricCustomerSnapshotInput) (models.MetricCustomerSnapshot, error) {
	if err := ensureManufacturingCustomer(tenantID, customerID); err != nil {
		return models.MetricCustomerSnapshot{}, err
	}
	snapshotDate := time.Now()
	if strings.TrimSpace(input.SnapshotDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(input.SnapshotDate))
		if err != nil {
			return models.MetricCustomerSnapshot{}, errors.New(MfgErrorInvalidInput)
		}
		snapshotDate = parsed
	}
	sourceJSON := "{}"
	if input.Source != nil {
		raw, err := json.Marshal(input.Source)
		if err != nil {
			return models.MetricCustomerSnapshot{}, err
		}
		sourceJSON = string(raw)
	}
	item := models.MetricCustomerSnapshot{
		TenantID:         tenantID,
		CustomerID:       customerID,
		SnapshotDate:     snapshotDate,
		WindowStart:      input.WindowStart,
		WindowEnd:        input.WindowEnd,
		HealthScore:      clampInt(input.HealthScore, 0, 100),
		GrossMargin:      input.GrossMargin,
		GrossMarginRate:  input.GrossMarginRate,
		RevenueAmount:    input.RevenueAmount,
		ReceivableAmount: input.ReceivableAmount,
		OverdueAmount:    input.OverdueAmount,
		OnTimeRate:       input.OnTimeRate,
		ComplaintCount:   maxInt(input.ComplaintCount, 0),
		ServiceValue:     input.ServiceValue,
		SourceJSON:       sourceJSON,
		RefreshStatus:    normalizeRefreshStatus(input.RefreshStatus),
		RefreshError:     strings.TrimSpace(input.RefreshError),
		GeneratedAt:      time.Now(),
		CreatedBy:        userID,
		UpdatedBy:        userID,
	}
	return item, utils.DB.Create(&item).Error
}

func LatestMetricCustomerSnapshot(tenantID uint, customerID uint) (models.MetricCustomerSnapshot, bool, error) {
	if err := ensureManufacturingCustomer(tenantID, customerID); err != nil {
		return models.MetricCustomerSnapshot{}, false, err
	}
	var item models.MetricCustomerSnapshot
	err := utils.DB.Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Order("snapshot_date desc, id desc").
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.MetricCustomerSnapshot{TenantID: tenantID, CustomerID: customerID}, false, nil
		}
		return models.MetricCustomerSnapshot{}, false, err
	}
	return item, true, nil
}

func MfgCustomer360(tenantID uint, customerID uint) (map[string]interface{}, error) {
	customer, err := ManufacturingCustomerObject(tenantID, customerID)
	if err != nil {
		return nil, err
	}
	profile, profileExists, err := GetMfgCustomerProfile(tenantID, customerID)
	if err != nil {
		return nil, err
	}
	decisionChain, err := ListMfgDecisionChain(tenantID, customerID)
	if err != nil {
		return nil, err
	}
	snapshot, snapshotExists, err := LatestMetricCustomerSnapshot(tenantID, customerID)
	if err != nil {
		return nil, err
	}
	var rfqs []models.MfgRFQ
	_ = utils.DB.Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Order("updated_at desc, id desc").
		Limit(10).
		Find(&rfqs).Error
	var samples []models.MfgSampleProject
	_ = utils.DB.Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Order("updated_at desc, id desc").
		Limit(10).
		Find(&samples).Error
	var assets []models.MfgAsset
	_ = utils.DB.Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Order("updated_at desc, id desc").
		Limit(10).
		Find(&assets).Error
	return map[string]interface{}{
		"customer": map[string]interface{}{
			"id":      customer.ID,
			"name":    customer.Name,
			"code":    customer.Code,
			"status":  customer.Status,
			"payload": jsonMap(customer.PayloadJSON),
		},
		"profile": map[string]interface{}{
			"exists": profileExists,
			"data":   profile,
		},
		"decision_chain": map[string]interface{}{
			"items": decisionChain,
			"total": len(decisionChain),
		},
		"rfqs": map[string]interface{}{
			"items":  rfqs,
			"total":  len(rfqs),
			"status": "ready",
		},
		"samples": map[string]interface{}{
			"items":  samples,
			"total":  len(samples),
			"status": "ready",
		},
		"assets": map[string]interface{}{
			"items":  assets,
			"total":  len(assets),
			"status": "ready",
		},
		"metrics": map[string]interface{}{
			"exists": snapshotExists,
			"data":   snapshot,
		},
	}, nil
}

func ManufacturingCustomerObject(tenantID uint, customerID uint) (models.CRMObject, error) {
	var customer models.CRMObject
	err := utils.DB.Where("tenant_id = ? AND id = ? AND domain = ? AND object_type = ?", tenantID, customerID, "contacts", "company").First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.CRMObject{}, errors.New(MfgErrorCustomerNotFound)
		}
		return models.CRMObject{}, err
	}
	return customer, nil
}

func ensureManufacturingCustomer(tenantID uint, customerID uint) error {
	_, err := ManufacturingCustomerObject(tenantID, customerID)
	return err
}

func ensureManufacturingContact(tenantID uint, contactID uint, customerID uint) error {
	var contact models.CRMObject
	err := utils.DB.Where("tenant_id = ? AND id = ? AND domain = ? AND object_type = ?", tenantID, contactID, "contacts", "contact").First(&contact).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(MfgErrorContactNotFound)
		}
		return err
	}
	if customerID > 0 && contact.CompanyID > 0 && contact.CompanyID != customerID {
		return errors.New(MfgErrorContactNotFound)
	}
	return nil
}

func normalizeDecisionRole(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "decision_maker", "influencer", "user", "technical", "procurement", "finance":
		return value
	default:
		return ""
	}
}

func normalizeInfluenceLevel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "high", "medium", "low":
		return value
	default:
		return "medium"
	}
}

func normalizeRelationshipStatus(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "strong", "neutral", "weak", "unknown":
		return value
	default:
		return "unknown"
	}
}

func normalizeRefreshStatus(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "manual", "generated", "synced", "failed":
		return value
	default:
		return "manual"
	}
}

func jsonMap(raw string) map[string]interface{} {
	out := map[string]interface{}{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func maxInt(value int, min int) int {
	if value < min {
		return min
	}
	return value
}
