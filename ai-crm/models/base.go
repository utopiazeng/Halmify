package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// BaseModel 基础模型
type BaseModel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TenantModel 租户隔离基础字段
type TenantModel struct {
	TenantID uint `gorm:"index;default:0" json:"tenant_id"`
}

// Tenant 企业租户
type Tenant struct {
	BaseModel
	Name        string `gorm:"size:128;uniqueIndex;not null" json:"name"`
	Status      string `gorm:"size:32;default:active;index" json:"status"`
	AdminUserID uint   `gorm:"index" json:"admin_user_id"`
	QuotaJSON   string `gorm:"type:longtext" json:"quota_json"`
	Remark      string `gorm:"type:text" json:"remark"`
}

// TenantQuota 企业租户配额
type TenantQuota struct {
	BaseModel
	TenantID             uint       `gorm:"uniqueIndex;not null" json:"tenant_id"`
	PlanCode             string     `gorm:"size:64;default:basic;index" json:"plan_code"`
	MaxUsers             int        `gorm:"default:20" json:"max_users"`
	StorageLimit         int64      `gorm:"default:1073741824" json:"storage_limit"`
	ValidFrom            *time.Time `json:"valid_from"`
	ValidTo              *time.Time `json:"valid_to"`
	Modules              string     `gorm:"type:longtext" json:"modules"`
	ModuleOverridesJSON  string     `gorm:"type:longtext" json:"module_overrides_json"`
	FeatureSwitch        string     `gorm:"type:longtext" json:"feature_switch"`
	CurrentUsersSnapshot int        `gorm:"-" json:"current_users,omitempty"`
}

// TenantStorageExpansion 租户增购/赠送存储空间记录
type TenantStorageExpansion struct {
	BaseModel
	TenantModel
	CapacityGB       int        `gorm:"not null;default:0" json:"capacity_gb"`
	Years            int        `gorm:"not null;default:1" json:"years"`
	PricePerGB       int64      `gorm:"not null;default:6" json:"price_per_gb"`
	Currency         string     `gorm:"size:16;not null;default:USD" json:"currency"`
	TotalAmount      int64      `gorm:"not null;default:0" json:"total_amount"`
	TotalAmountCents int64      `gorm:"not null;default:0" json:"total_amount_cents"`
	Source           string     `gorm:"size:32;not null;default:paid" json:"source"`
	Status           string     `gorm:"size:32;not null;default:active;index" json:"status"`
	ValidFrom        time.Time  `gorm:"not null;index" json:"valid_from"`
	ValidTo          time.Time  `gorm:"not null;index" json:"valid_to"`
	Remark           string     `gorm:"type:text" json:"remark"`
	PurchasedBy      uint       `gorm:"index" json:"purchased_by"`
	ConfirmedBy      uint       `gorm:"index" json:"confirmed_by"`
	ConfirmedAt      *time.Time `json:"confirmed_at"`
	RequestJSON      string     `gorm:"type:longtext" json:"request_json"`
	BillingSnapshot  string     `gorm:"type:longtext" json:"billing_snapshot"`
}

// TenantSeatExpansion 租户人数扩充记录
type TenantSeatExpansion struct {
	BaseModel
	TenantModel
	SeatCount        int        `gorm:"not null;default:0" json:"seat_count"`
	Years            int        `gorm:"not null;default:1" json:"years"`
	PricePerSeat     int64      `gorm:"not null;default:1" json:"price_per_seat"`
	Currency         string     `gorm:"size:16;not null;default:USD" json:"currency"`
	TotalAmount      int64      `gorm:"not null;default:0" json:"total_amount"`
	TotalAmountCents int64      `gorm:"not null;default:0" json:"total_amount_cents"`
	Source           string     `gorm:"size:32;not null;default:paid" json:"source"`
	Status           string     `gorm:"size:32;not null;default:active;index" json:"status"`
	ValidFrom        time.Time  `gorm:"not null;index" json:"valid_from"`
	ValidTo          time.Time  `gorm:"not null;index" json:"valid_to"`
	Remark           string     `gorm:"type:text" json:"remark"`
	PurchasedBy      uint       `gorm:"index" json:"purchased_by"`
	ConfirmedBy      uint       `gorm:"index" json:"confirmed_by"`
	ConfirmedAt      *time.Time `json:"confirmed_at"`
	RequestJSON      string     `gorm:"type:longtext" json:"request_json"`
	BillingSnapshot  string     `gorm:"type:longtext" json:"billing_snapshot"`
}

// PlatformPlanTemplate 平台套餐模板
type PlatformPlanTemplate struct {
	BaseModel
	PlanCode        string `gorm:"size:64;uniqueIndex;not null" json:"plan_code"`
	Name            string `gorm:"size:128;not null" json:"name"`
	MaxUsersDefault int    `gorm:"default:20" json:"max_users_default"`
	StorageDefault  int64  `gorm:"default:1073741824" json:"storage_default"`
	ModulesJSON     string `gorm:"type:longtext" json:"modules_json"`
	Status          string `gorm:"size:32;default:active;index" json:"status"`
	Version         int    `gorm:"default:1" json:"version"`
	Remark          string `gorm:"type:text" json:"remark"`
}

// TenantExpiryNotice 租户到期提醒记录（去重+审计）
type TenantExpiryNotice struct {
	BaseModel
	TenantID   uint      `gorm:"index:idx_tenant_expiry_notice_tenant_type_date,unique;not null" json:"tenant_id"`
	NoticeType string    `gorm:"size:32;index:idx_tenant_expiry_notice_tenant_type_date,unique;not null" json:"notice_type"`
	NoticeDate time.Time `gorm:"type:date;index:idx_tenant_expiry_notice_tenant_type_date,unique;not null" json:"notice_date"`
	SentTo     string    `gorm:"size:191" json:"sent_to"`
	Status     string    `gorm:"size:32;default:generated;index" json:"status"`
	DaysLeft   int       `gorm:"default:0;index" json:"days_left"`
}

// TenantIndustryPackage records the optional primary industry package enabled
// for a tenant. Tenants without an enabled row keep the generic CRM behavior.
type TenantIndustryPackage struct {
	BaseModel
	TenantID       uint       `gorm:"index:idx_tenant_industry_primary,unique;index:idx_tenant_industry_package,unique;not null" json:"tenant_id"`
	PackageCode    string     `gorm:"size:64;index:idx_tenant_industry_package,unique;not null" json:"package_code"`
	PackageVersion string     `gorm:"size:32;default:1.0" json:"package_version"`
	Primary        bool       `gorm:"index:idx_tenant_industry_primary,unique;default:true" json:"primary"`
	Enabled        bool       `gorm:"index;default:false" json:"enabled"`
	EnabledAt      *time.Time `json:"enabled_at"`
	DisabledAt     *time.Time `json:"disabled_at"`
	ConfigJSON     string     `gorm:"type:longtext" json:"config_json"`
}

func (TenantIndustryPackage) TableName() string {
	return "tenant_industry_packages"
}

// IndustryPackageRegistry stores platform-level package metadata and manifest.
type IndustryPackageRegistry struct {
	BaseModel
	Code         string `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name         string `gorm:"size:128;not null" json:"name"`
	Version      string `gorm:"size:32;default:1.0" json:"version"`
	Description  string `gorm:"type:text" json:"description"`
	ManifestJSON string `gorm:"type:longtext" json:"manifest_json"`
	Status       string `gorm:"size:32;default:active;index" json:"status"`
}

func (IndustryPackageRegistry) TableName() string {
	return "industry_package_registry"
}

// IndustryExtensionRegistry stores package extension declarations separately
// so future industry packages can register menus, tabs, reports and APIs.
type IndustryExtensionRegistry struct {
	BaseModel
	PackageCode   string `gorm:"size:64;index:idx_industry_extension_key,unique;not null" json:"package_code"`
	ExtensionType string `gorm:"size:64;index:idx_industry_extension_key,unique;not null" json:"extension_type"`
	TargetKey     string `gorm:"size:128;index:idx_industry_extension_key,unique;not null" json:"target_key"`
	ExtensionKey  string `gorm:"size:128;index:idx_industry_extension_key,unique;not null" json:"extension_key"`
	ConfigJSON    string `gorm:"type:longtext" json:"config_json"`
	Sort          int    `gorm:"default:0;index" json:"sort"`
	Enabled       bool   `gorm:"default:true;index" json:"enabled"`
}

func (IndustryExtensionRegistry) TableName() string {
	return "industry_extension_registry"
}

// MfgCustomerProfile stores manufacturing-specific customer fields outside
// the generic CRM customer object.
type MfgCustomerProfile struct {
	BaseModel
	TenantID                  uint   `gorm:"uniqueIndex:idx_mfg_customer_profile_tenant_customer;index;not null" json:"tenant_id"`
	CustomerID                uint   `gorm:"uniqueIndex:idx_mfg_customer_profile_tenant_customer;index;not null" json:"customer_id"`
	RequiredCertifications    string `gorm:"type:text" json:"required_certifications"`
	QualityStandard           string `gorm:"size:191" json:"quality_standard"`
	PaymentTermType           string `gorm:"size:64;index" json:"payment_term_type"`
	DesignatedBrandRequired   bool   `gorm:"default:false" json:"designated_brand_required"`
	MoldFeePolicy             string `gorm:"size:191" json:"mold_fee_policy"`
	SpecialProcessRequirement string `gorm:"type:text" json:"special_process_requirement"`
	PreferredInspectionAgency string `gorm:"size:191" json:"preferred_inspection_agency"`
	PackagingRequirement      string `gorm:"type:text" json:"packaging_requirement"`
	ExtensionJSON             string `gorm:"type:longtext" json:"extension_json"`
	CreatedBy                 uint   `gorm:"index" json:"created_by"`
	UpdatedBy                 uint   `gorm:"index" json:"updated_by"`
}

func (MfgCustomerProfile) TableName() string {
	return "mfg_customer_profiles"
}

// MfgDecisionChain records manufacturing buying-center contacts for a customer.
type MfgDecisionChain struct {
	BaseModel
	TenantID           uint   `gorm:"index:idx_mfg_decision_chain_customer;index;not null" json:"tenant_id"`
	CustomerID         uint   `gorm:"index:idx_mfg_decision_chain_customer;not null" json:"customer_id"`
	ContactID          uint   `gorm:"index" json:"contact_id"`
	RoleType           string `gorm:"size:64;index" json:"role_type"`
	InfluenceLevel     string `gorm:"size:64;index" json:"influence_level"`
	RelationshipStatus string `gorm:"size:64;index" json:"relationship_status"`
	Notes              string `gorm:"type:text" json:"notes"`
	CreatedBy          uint   `gorm:"index" json:"created_by"`
	UpdatedBy          uint   `gorm:"index" json:"updated_by"`
}

func (MfgDecisionChain) TableName() string {
	return "mfg_decision_chains"
}

// MetricCustomerSnapshot stores point-in-time manufacturing customer metrics.
type MetricCustomerSnapshot struct {
	BaseModel
	TenantID         uint       `gorm:"index:idx_metric_customer_snapshot_customer_window;index;not null" json:"tenant_id"`
	CustomerID       uint       `gorm:"index:idx_metric_customer_snapshot_customer_window;not null" json:"customer_id"`
	SnapshotDate     time.Time  `gorm:"type:date;index:idx_metric_customer_snapshot_customer_window" json:"snapshot_date"`
	WindowStart      *time.Time `json:"window_start"`
	WindowEnd        *time.Time `json:"window_end"`
	HealthScore      int        `gorm:"default:0" json:"health_score"`
	GrossMargin      float64    `json:"gross_margin"`
	GrossMarginRate  float64    `json:"gross_margin_rate"`
	RevenueAmount    float64    `json:"revenue_amount"`
	ReceivableAmount float64    `json:"receivable_amount"`
	OverdueAmount    float64    `json:"overdue_amount"`
	OnTimeRate       float64    `json:"on_time_rate"`
	ComplaintCount   int        `gorm:"default:0" json:"complaint_count"`
	ServiceValue     float64    `json:"service_value"`
	SourceJSON       string     `gorm:"type:longtext" json:"source_json"`
	RefreshStatus    string     `gorm:"size:32;default:manual;index" json:"refresh_status"`
	RefreshError     string     `gorm:"type:text" json:"refresh_error"`
	GeneratedAt      time.Time  `gorm:"index" json:"generated_at"`
	CreatedBy        uint       `gorm:"index" json:"created_by"`
	UpdatedBy        uint       `gorm:"index" json:"updated_by"`
}

func (MetricCustomerSnapshot) TableName() string {
	return "metric_customer_snapshots"
}

// MfgRFQ records manufacturing request-for-quotation intake and workflow state.
type MfgRFQ struct {
	BaseModel
	TenantID             uint       `gorm:"index:idx_mfg_rfq_tenant_status;index;not null" json:"tenant_id"`
	CustomerID           uint       `gorm:"index" json:"customer_id"`
	ContactID            uint       `gorm:"index" json:"contact_id"`
	OpportunityID        uint       `gorm:"index" json:"opportunity_id"`
	RFQNo                string     `gorm:"size:64;index;not null" json:"rfq_no"`
	BusinessName         string     `gorm:"size:191;index" json:"business_name"`
	ProductFamilyID      uint       `gorm:"index" json:"product_family_id"`
	Quantity             float64    `json:"quantity"`
	TargetPrice          float64    `json:"target_price"`
	RequiredDeliveryDate *time.Time `json:"required_delivery_date"`
	DrawingFileID        string     `gorm:"size:128;index" json:"drawing_file_id"`
	DrawingStorageNodeID string     `gorm:"size:128" json:"drawing_storage_node_id"`
	NoDrawingReason      string     `gorm:"type:text" json:"no_drawing_reason"`
	Status               string     `gorm:"size:32;index:idx_mfg_rfq_tenant_status;default:draft" json:"status"`
	OwnerID              uint       `gorm:"index" json:"owner_id"`
	TechnicalOwnerID     uint       `gorm:"index" json:"technical_owner_id"`
	PayloadJSON          string     `gorm:"type:longtext" json:"payload_json"`
	CreatedBy            uint       `gorm:"index" json:"created_by"`
	UpdatedBy            uint       `gorm:"index" json:"updated_by"`
}

func (MfgRFQ) TableName() string { return "mfg_rfqs" }

type MfgRFQClarification struct {
	BaseModel
	TenantID       uint       `gorm:"index;not null" json:"tenant_id"`
	RFQID          uint       `gorm:"index;not null" json:"rfq_id"`
	Question       string     `gorm:"type:text" json:"question"`
	Answer         string     `gorm:"type:text" json:"answer"`
	OwnerID        uint       `gorm:"index" json:"owner_id"`
	Status         string     `gorm:"size:32;index;default:open" json:"status"`
	SourceType     string     `gorm:"size:64;index" json:"source_type"`
	SourceKey      string     `gorm:"size:128;index" json:"source_key"`
	SourceLabel    string     `gorm:"size:191" json:"source_label"`
	SortOrder      int        `gorm:"index" json:"sort_order"`
	LastAnsweredBy uint       `gorm:"index" json:"last_answered_by"`
	LastAnsweredAt *time.Time `json:"last_answered_at"`
	DueAt          *time.Time `json:"due_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
	CreatedBy      uint       `gorm:"index" json:"created_by"`
	UpdatedBy      uint       `gorm:"index" json:"updated_by"`
}

func (MfgRFQClarification) TableName() string { return "mfg_rfq_clarifications" }

type MfgRFQClarificationAnswerHistory struct {
	BaseModel
	TenantID        uint      `gorm:"index;not null" json:"tenant_id"`
	RFQID           uint      `gorm:"index;not null" json:"rfq_id"`
	ClarificationID uint      `gorm:"index;not null" json:"clarification_id"`
	OldAnswer       string    `gorm:"type:text" json:"old_answer"`
	NewAnswer       string    `gorm:"type:text" json:"new_answer"`
	OldStatus       string    `gorm:"size:32" json:"old_status"`
	NewStatus       string    `gorm:"size:32" json:"new_status"`
	AnsweredBy      uint      `gorm:"index" json:"answered_by"`
	AnsweredByName  string    `gorm:"size:128" json:"answered_by_name"`
	AnsweredAt      time.Time `gorm:"index" json:"answered_at"`
}

func (MfgRFQClarificationAnswerHistory) TableName() string {
	return "mfg_rfq_clarification_answer_histories"
}

type MfgDrawing struct {
	BaseModel
	TenantID            uint       `gorm:"index;not null" json:"tenant_id"`
	CustomerID          uint       `gorm:"index" json:"customer_id"`
	RFQID               uint       `gorm:"index" json:"rfq_id"`
	FileID              string     `gorm:"size:128;index" json:"file_id"`
	StorageNodeID       string     `gorm:"size:128" json:"storage_node_id"`
	DrawingNo           string     `gorm:"size:128;index" json:"drawing_no"`
	Version             string     `gorm:"size:64;index" json:"version"`
	SourceSystem        string     `gorm:"size:64;default:crm" json:"source_system"`
	ClassificationLevel string     `gorm:"size:16;index;default:L3" json:"classification_level"`
	WatermarkRequired   bool       `gorm:"default:true" json:"watermark_required"`
	DownloadPolicy      string     `gorm:"size:64;default:audit_required" json:"download_policy"`
	ApprovalStatus      string     `gorm:"size:32;index;default:not_required" json:"approval_status"`
	ApprovedBy          uint       `gorm:"index" json:"approved_by"`
	ApprovedAt          *time.Time `json:"approved_at"`
	ExternalSendPolicy  string     `gorm:"size:64;default:approval_required" json:"external_send_policy"`
	Status              string     `gorm:"size:32;index;default:active" json:"status"`
	CreatedBy           uint       `gorm:"index" json:"created_by"`
	UpdatedBy           uint       `gorm:"index" json:"updated_by"`
}

func (MfgDrawing) TableName() string { return "mfg_drawings" }

type MfgProductFamily struct {
	BaseModel
	TenantID    uint   `gorm:"uniqueIndex:idx_mfg_product_family_code;index;not null" json:"tenant_id"`
	FamilyCode  string `gorm:"size:64;uniqueIndex:idx_mfg_product_family_code;not null" json:"family_code"`
	Name        string `gorm:"size:191;index;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Status      string `gorm:"size:32;index;default:active" json:"status"`
	OwnerID     uint   `gorm:"index" json:"owner_id"`
	CreatedBy   uint   `gorm:"index" json:"created_by"`
	UpdatedBy   uint   `gorm:"index" json:"updated_by"`
}

func (MfgProductFamily) TableName() string { return "mfg_product_families" }

type MfgConfigModel struct {
	BaseModel
	TenantID          uint   `gorm:"index;not null" json:"tenant_id"`
	ProductFamilyID   uint   `gorm:"index;not null" json:"product_family_id"`
	ModelNo           string `gorm:"size:64;index;not null" json:"model_no"`
	Version           string `gorm:"size:64;index;not null" json:"version"`
	ParametersJSON    string `gorm:"type:longtext" json:"parameters_json"`
	ConstraintsJSON   string `gorm:"type:longtext" json:"constraints_json"`
	DefaultValuesJSON string `gorm:"type:longtext" json:"default_values_json"`
	Status            string `gorm:"size:32;index;default:draft" json:"status"`
	CreatedBy         uint   `gorm:"index" json:"created_by"`
	UpdatedBy         uint   `gorm:"index" json:"updated_by"`
}

func (MfgConfigModel) TableName() string { return "mfg_config_models" }

type MfgBOMTemplate struct {
	BaseModel
	TenantID        uint   `gorm:"index;not null" json:"tenant_id"`
	ProductFamilyID uint   `gorm:"index;not null" json:"product_family_id"`
	TemplateNo      string `gorm:"size:64;index;not null" json:"template_no"`
	Version         string `gorm:"size:64;index;not null" json:"version"`
	BOMLinesJSON    string `gorm:"type:longtext" json:"bom_lines_json"`
	RuleJSON        string `gorm:"type:longtext" json:"rule_json"`
	Status          string `gorm:"size:32;index;default:draft" json:"status"`
	CreatedBy       uint   `gorm:"index" json:"created_by"`
	UpdatedBy       uint   `gorm:"index" json:"updated_by"`
}

func (MfgBOMTemplate) TableName() string { return "mfg_bom_templates" }

type MfgBOMSnapshot struct {
	BaseModel
	TenantID            uint   `gorm:"index;not null" json:"tenant_id"`
	SnapshotNo          string `gorm:"size:64;index;not null" json:"snapshot_no"`
	BusinessName        string `gorm:"size:191;index" json:"business_name"`
	SourceTemplateID    uint   `gorm:"index" json:"source_template_id"`
	SourceConfigVersion string `gorm:"size:64" json:"source_config_version"`
	RFQID               uint   `gorm:"index" json:"rfq_id"`
	QuoteID             uint   `gorm:"index" json:"quote_id"`
	OrderID             uint   `gorm:"index" json:"order_id"`
	ConfigValuesJSON    string `gorm:"type:longtext" json:"config_values_json"`
	BOMLinesJSON        string `gorm:"type:longtext" json:"bom_lines_json"`
	Status              string `gorm:"size:32;index;default:active" json:"status"`
	CreatedBy           uint   `gorm:"index" json:"created_by"`
	UpdatedBy           uint   `gorm:"index" json:"updated_by"`
}

func (MfgBOMSnapshot) TableName() string { return "mfg_bom_snapshots" }

type MfgCostSheet struct {
	BaseModel
	TenantID              uint    `gorm:"index;not null" json:"tenant_id"`
	RFQID                 uint    `gorm:"index" json:"rfq_id"`
	BOMSnapshotID         uint    `gorm:"index" json:"bom_snapshot_id"`
	MaterialCost          float64 `json:"material_cost"`
	LaborCost             float64 `json:"labor_cost"`
	OutsourcingCost       float64 `json:"outsourcing_cost"`
	ManufacturingOverhead float64 `json:"manufacturing_overhead"`
	Amount                float64 `json:"amount"`
	GrossMargin           float64 `json:"gross_margin"`
	GrossMarginRate       float64 `json:"gross_margin_rate"`
	MarginLevel           string  `gorm:"size:32;index" json:"margin_level"`
	Status                string  `gorm:"size:32;index;default:draft" json:"status"`
	SensitiveCostJSON     string  `gorm:"type:longtext" json:"sensitive_cost_json,omitempty"`
	CreatedBy             uint    `gorm:"index" json:"created_by"`
	UpdatedBy             uint    `gorm:"index" json:"updated_by"`
}

func (MfgCostSheet) TableName() string { return "mfg_cost_sheets" }

type MfgQuoteRevision struct {
	BaseModel
	TenantID          uint       `gorm:"index;not null" json:"tenant_id"`
	QuoteNo           string     `gorm:"size:64;index;not null" json:"quote_no"`
	Revision          int        `gorm:"index;default:1" json:"revision"`
	RFQID             uint       `gorm:"index" json:"rfq_id"`
	CostSheetID       uint       `gorm:"index" json:"cost_sheet_id"`
	BOMSnapshotID     uint       `gorm:"index" json:"bom_snapshot_id"`
	Amount            float64    `json:"amount"`
	Currency          string     `gorm:"size:16;default:CNY" json:"currency"`
	TermsJSON         string     `gorm:"type:longtext" json:"terms_json"`
	ApprovalStatus    string     `gorm:"size:32;index;default:draft" json:"approval_status"`
	ApprovedBy        uint       `gorm:"index" json:"approved_by"`
	ApprovedAt        *time.Time `json:"approved_at"`
	RejectedBy        uint       `gorm:"index" json:"rejected_by"`
	RejectedAt        *time.Time `json:"rejected_at"`
	SendStatus        string     `gorm:"size:32;index;default:not_sent" json:"send_status"`
	CustomerViewedAt  *time.Time `json:"customer_viewed_at"`
	CustomerViewCount int        `gorm:"default:0" json:"customer_view_count"`
	Status            string     `gorm:"size:32;index;default:draft" json:"status"`
	CreatedBy         uint       `gorm:"index" json:"created_by"`
	UpdatedBy         uint       `gorm:"index" json:"updated_by"`
}

func (MfgQuoteRevision) TableName() string { return "mfg_quote_revisions" }

type MfgQuoteApprovalHistory struct {
	BaseModel
	TenantID          uint      `gorm:"index;not null" json:"tenant_id"`
	RFQID             uint      `gorm:"index" json:"rfq_id"`
	QuoteID           uint      `gorm:"index;not null" json:"quote_id"`
	QuoteNo           string    `gorm:"size:64;index;not null" json:"quote_no"`
	Revision          int       `gorm:"index" json:"revision"`
	Action            string    `gorm:"size:32;index;not null" json:"action"`
	OldApprovalStatus string    `gorm:"size:32" json:"old_approval_status"`
	NewApprovalStatus string    `gorm:"size:32" json:"new_approval_status"`
	Remark            string    `gorm:"type:text" json:"remark"`
	OperatorID        uint      `gorm:"index" json:"operator_id"`
	OperatorName      string    `gorm:"size:128" json:"operator_name"`
	OperatedAt        time.Time `gorm:"index" json:"operated_at"`
}

func (MfgQuoteApprovalHistory) TableName() string {
	return "mfg_quote_approval_histories"
}

type MfgApprovalHistory struct {
	BaseModel
	TenantID          uint      `gorm:"index;not null" json:"tenant_id"`
	Resource          string    `gorm:"size:64;index;not null" json:"resource"`
	ResourceID        uint      `gorm:"index;not null" json:"resource_id"`
	ResourceNo        string    `gorm:"size:128;index" json:"resource_no"`
	Action            string    `gorm:"size:32;index;not null" json:"action"`
	OldApprovalStatus string    `gorm:"size:32" json:"old_approval_status"`
	NewApprovalStatus string    `gorm:"size:32" json:"new_approval_status"`
	Remark            string    `gorm:"type:text" json:"remark"`
	OperatorID        uint      `gorm:"index" json:"operator_id"`
	OperatorName      string    `gorm:"size:128" json:"operator_name"`
	OperatedAt        time.Time `gorm:"index" json:"operated_at"`
}

func (MfgApprovalHistory) TableName() string {
	return "mfg_approval_histories"
}

type MfgSampleProject struct {
	BaseModel
	TenantID             uint       `gorm:"index;not null" json:"tenant_id"`
	CustomerID           uint       `gorm:"index" json:"customer_id"`
	OpportunityID        uint       `gorm:"index" json:"opportunity_id"`
	RFQID                uint       `gorm:"index" json:"rfq_id"`
	SampleNo             string     `gorm:"size:64;index;not null" json:"sample_no"`
	BOMSnapshotID        uint       `gorm:"index" json:"bom_snapshot_id"`
	Stage                string     `gorm:"size:32;index;default:planning" json:"stage"`
	RiskLevel            string     `gorm:"size:16;index;default:green" json:"risk_level"`
	OwnerID              uint       `gorm:"index" json:"owner_id"`
	EngineeringOwnerID   uint       `gorm:"index" json:"engineering_owner_id"`
	PlannedFinishAt      *time.Time `json:"planned_finish_at"`
	ActualFinishAt       *time.Time `json:"actual_finish_at"`
	CustomerFeedback     string     `gorm:"type:text" json:"customer_feedback"`
	MassProductionStatus string     `gorm:"size:32;index;default:not_ready" json:"mass_production_status"`
	CreatedBy            uint       `gorm:"index" json:"created_by"`
	UpdatedBy            uint       `gorm:"index" json:"updated_by"`
}

func (MfgSampleProject) TableName() string { return "mfg_sample_projects" }

type MfgSampleProgressHistory struct {
	BaseModel
	TenantID         uint   `gorm:"index;not null" json:"tenant_id"`
	SampleProjectID  uint   `gorm:"index;not null" json:"sample_project_id"`
	FromStage        string `gorm:"size:64;index" json:"from_stage"`
	ToStage          string `gorm:"size:64;index" json:"to_stage"`
	RiskLevel        string `gorm:"size:16;index" json:"risk_level"`
	CustomerFeedback string `gorm:"type:text" json:"customer_feedback"`
	Note             string `gorm:"type:text" json:"note"`
	CreatedBy        uint   `gorm:"index" json:"created_by"`
}

func (MfgSampleProgressHistory) TableName() string { return "mfg_sample_progress_histories" }

type MfgSampleTestRecord struct {
	BaseModel
	TenantID         uint       `gorm:"index;not null" json:"tenant_id"`
	SampleProjectID  uint       `gorm:"index;not null" json:"sample_project_id"`
	TestItem         string     `gorm:"size:191;index" json:"test_item"`
	TestResult       string     `gorm:"size:32;index" json:"test_result"`
	FailureReason    string     `gorm:"type:text" json:"failure_reason"`
	AttachmentFileID string     `gorm:"size:128;index" json:"attachment_file_id"`
	StorageNodeID    string     `gorm:"size:128" json:"storage_node_id"`
	TestedBy         uint       `gorm:"index" json:"tested_by"`
	TestedAt         *time.Time `json:"tested_at"`
	CreatedBy        uint       `gorm:"index" json:"created_by"`
	UpdatedBy        uint       `gorm:"index" json:"updated_by"`
}

func (MfgSampleTestRecord) TableName() string { return "mfg_sample_test_records" }

type MfgProductionProgress struct {
	BaseModel
	TenantID       uint       `gorm:"index;not null" json:"tenant_id"`
	OrderID        uint       `gorm:"index" json:"order_id"`
	CustomerID     uint       `gorm:"index" json:"customer_id"`
	SalesOwnerID   uint       `gorm:"index" json:"sales_owner_id"`
	ERPOrderNo     string     `gorm:"size:64;index" json:"erp_order_no"`
	MESWorkOrderNo string     `gorm:"size:64;index" json:"mes_work_order_no"`
	Milestone      string     `gorm:"size:128;index" json:"milestone"`
	ProgressStatus string     `gorm:"size:32;index;default:not_started" json:"progress_status"`
	PlannedAt      *time.Time `json:"planned_at"`
	ActualAt       *time.Time `json:"actual_at"`
	RiskLevel      string     `gorm:"size:16;index;default:green" json:"risk_level"`
	RiskReason     string     `gorm:"type:text" json:"risk_reason"`
	SourceSystem   string     `gorm:"size:64;default:manual" json:"source_system"`
	SyncStatus     string     `gorm:"size:32;index;default:local" json:"sync_status"`
	CreatedBy      uint       `gorm:"index" json:"created_by"`
	UpdatedBy      uint       `gorm:"index" json:"updated_by"`
}

func (MfgProductionProgress) TableName() string { return "mfg_production_progress" }

type MfgProductionProgressHistory struct {
	BaseModel
	TenantID             uint   `gorm:"index;not null" json:"tenant_id"`
	ProductionProgressID uint   `gorm:"index;not null" json:"production_progress_id"`
	OrderID              uint   `gorm:"index" json:"order_id"`
	FromMilestone        string `gorm:"size:128;index" json:"from_milestone"`
	ToMilestone          string `gorm:"size:128;index" json:"to_milestone"`
	FromProgressStatus   string `gorm:"size:32;index" json:"from_progress_status"`
	ToProgressStatus     string `gorm:"size:32;index" json:"to_progress_status"`
	RiskLevel            string `gorm:"size:16;index" json:"risk_level"`
	RiskReason           string `gorm:"type:text" json:"risk_reason"`
	Note                 string `gorm:"type:text" json:"note"`
	CreatedBy            uint   `gorm:"index" json:"created_by"`
}

func (MfgProductionProgressHistory) TableName() string { return "mfg_production_progress_histories" }

type MfgECN struct {
	BaseModel
	TenantID           uint    `gorm:"index;not null" json:"tenant_id"`
	ECNNo              string  `gorm:"size:64;index;not null" json:"ecn_no"`
	OrderID            uint    `gorm:"index" json:"order_id"`
	OldBOMSnapshotID   uint    `gorm:"index" json:"old_bom_snapshot_id"`
	NewBOMSnapshotID   uint    `gorm:"index" json:"new_bom_snapshot_id"`
	ChangeReason       string  `gorm:"type:text" json:"change_reason"`
	ChangedFieldsJSON  string  `gorm:"type:longtext" json:"changed_fields_json"`
	BOMDiffJSON        string  `gorm:"type:longtext" json:"bom_diff_json"`
	ImpactLevel        string  `gorm:"size:32;index;default:normal" json:"impact_level"`
	CostImpact         float64 `json:"cost_impact"`
	CostImpactCurrency string  `gorm:"size:16;default:CNY" json:"cost_impact_currency"`
	DeliveryImpactDays int     `json:"delivery_impact_days"`
	ImpactScope        string  `gorm:"type:text" json:"impact_scope"`
	QualityImpact      string  `gorm:"type:text" json:"quality_impact"`
	ApprovalStatus     string  `gorm:"size:32;index;default:draft" json:"approval_status"`
	CustomerConfirmed  bool    `gorm:"default:false" json:"customer_confirmed"`
	SyncStatus         string  `gorm:"size:32;index;default:not_configured" json:"sync_status"`
	Status             string  `gorm:"size:32;index;default:draft" json:"status"`
	CreatedBy          uint    `gorm:"index" json:"created_by"`
	UpdatedBy          uint    `gorm:"index" json:"updated_by"`
}

func (MfgECN) TableName() string { return "mfg_ecns" }

type MfgIntegrationOutbox struct {
	BaseModel
	TenantID          uint       `gorm:"index;not null" json:"tenant_id"`
	EventType         string     `gorm:"size:96;index;not null" json:"event_type"`
	TargetSystem      string     `gorm:"size:64;index;not null" json:"target_system"`
	BusinessType      string     `gorm:"size:64;index" json:"business_type"`
	BusinessID        uint       `gorm:"index" json:"business_id"`
	IdempotencyKey    string     `gorm:"size:128;uniqueIndex;not null" json:"idempotency_key"`
	PayloadJSON       string     `gorm:"type:longtext" json:"payload_json"`
	Status            string     `gorm:"size:32;index;default:pending" json:"status"`
	RetryCount        int        `gorm:"default:0" json:"retry_count"`
	MaxRetryCount     int        `gorm:"default:3" json:"max_retry_count"`
	LastError         string     `gorm:"type:text" json:"last_error"`
	CompensationState string     `gorm:"size:32;index;default:pending" json:"compensation_state"`
	NextRetryAt       *time.Time `json:"next_retry_at"`
	LastAttemptAt     *time.Time `json:"last_attempt_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	CreatedBy         uint       `gorm:"index" json:"created_by"`
	UpdatedBy         uint       `gorm:"index" json:"updated_by"`
}

func (MfgIntegrationOutbox) TableName() string { return "mfg_integration_outbox" }

type MfgCompensationRecord struct {
	BaseModel
	TenantID            uint       `gorm:"index;not null" json:"tenant_id"`
	CompensationNo      string     `gorm:"size:64;index" json:"compensation_no"`
	OrderID             uint       `gorm:"index" json:"order_id"`
	CustomerID          uint       `gorm:"index" json:"customer_id"`
	SalesOwnerID        uint       `gorm:"index" json:"sales_owner_id"`
	Reason              string     `gorm:"type:text" json:"reason"`
	Amount              float64    `json:"amount"`
	Currency            string     `gorm:"size:16;default:CNY" json:"currency"`
	ResponsibilitySplit string     `gorm:"type:text" json:"responsibility_split"`
	ImprovementActions  string     `gorm:"type:text" json:"improvement_actions"`
	CompensationDate    *time.Time `json:"compensation_date"`
	Status              string     `gorm:"size:32;index;default:open" json:"status"`
	ApprovalStatus      string     `gorm:"size:32;index;default:draft" json:"approval_status"`
	ApprovedBy          uint       `gorm:"index" json:"approved_by"`
	ApprovedAt          *time.Time `json:"approved_at"`
	ApprovalRemark      string     `gorm:"type:text" json:"approval_remark"`
	CreatedBy           uint       `gorm:"index" json:"created_by"`
	UpdatedBy           uint       `gorm:"index" json:"updated_by"`
}

func (MfgCompensationRecord) TableName() string { return "mfg_compensation_records" }

type MfgAsset struct {
	BaseModel
	TenantID        uint       `gorm:"index;not null" json:"tenant_id"`
	CustomerID      uint       `gorm:"index" json:"customer_id"`
	OrderID         uint       `gorm:"index" json:"order_id"`
	BOMSnapshotID   uint       `gorm:"index" json:"bom_snapshot_id"`
	SerialNo        string     `gorm:"size:128;index;not null" json:"serial_no"`
	BatchNo         string     `gorm:"size:128;index" json:"batch_no"`
	ProductID       uint       `gorm:"index" json:"product_id"`
	WarrantyStart   *time.Time `json:"warranty_start"`
	WarrantyEnd     *time.Time `json:"warranty_end"`
	InstallLocation string     `gorm:"size:255" json:"install_location"`
	Status          string     `gorm:"size:32;index;default:active" json:"status"`
	IoTAssetRef     string     `gorm:"size:128;index" json:"iot_asset_ref"`
	IoTAuthorized   bool       `gorm:"default:false;index" json:"iot_authorized"`
	IoTAuthorizedAt *time.Time `json:"iot_authorized_at"`
	IoTRevokedAt    *time.Time `json:"iot_revoked_at"`
	CreatedBy       uint       `gorm:"index" json:"created_by"`
	UpdatedBy       uint       `gorm:"index" json:"updated_by"`
}

func (MfgAsset) TableName() string { return "mfg_assets" }

type MfgServiceTicket struct {
	BaseModel
	TenantID           uint   `gorm:"index;not null" json:"tenant_id"`
	AssetID            uint   `gorm:"index" json:"asset_id"`
	CustomerID         uint   `gorm:"index" json:"customer_id"`
	TicketID           uint   `gorm:"index" json:"ticket_id"`
	FaultCode          string `gorm:"size:64;index" json:"fault_code"`
	ServiceType        string `gorm:"size:64;index" json:"service_type"`
	ServiceResult      string `gorm:"type:text" json:"service_result"`
	ServiceToSalesFlag bool   `gorm:"default:false;index" json:"service_to_sales_flag"`
	OpportunityID      uint   `gorm:"index" json:"opportunity_id"`
	Status             string `gorm:"size:32;index;default:open" json:"status"`
	CreatedBy          uint   `gorm:"index" json:"created_by"`
	UpdatedBy          uint   `gorm:"index" json:"updated_by"`
}

func (MfgServiceTicket) TableName() string { return "mfg_service_tickets" }

type MfgSparePartQuote struct {
	BaseModel
	TenantID        uint    `gorm:"index;not null" json:"tenant_id"`
	ServiceTicketID uint    `gorm:"index" json:"service_ticket_id"`
	AssetID         uint    `gorm:"index" json:"asset_id"`
	QuoteNo         string  `gorm:"size:64;index;not null" json:"quote_no"`
	LineItemsJSON   string  `gorm:"type:longtext" json:"line_items_json"`
	Amount          float64 `json:"amount"`
	Status          string  `gorm:"size:32;index;default:draft" json:"status"`
	CreatedBy       uint    `gorm:"index" json:"created_by"`
	UpdatedBy       uint    `gorm:"index" json:"updated_by"`
}

func (MfgSparePartQuote) TableName() string { return "mfg_spare_part_quotes" }

type MfgSalesAgreement struct {
	BaseModel
	TenantID              uint       `gorm:"index;not null" json:"tenant_id"`
	CustomerID            uint       `gorm:"index;not null" json:"customer_id"`
	ProductFamilyID       uint       `gorm:"index" json:"product_family_id"`
	AgreementNo           string     `gorm:"size:64;index;not null" json:"agreement_no"`
	Year                  int        `gorm:"index" json:"year"`
	CommittedQuantity     float64    `json:"committed_quantity"`
	FulfilledQuantity     float64    `json:"fulfilled_quantity"`
	TargetAmount          float64    `json:"target_amount"`
	FulfilledAmount       float64    `json:"fulfilled_amount"`
	ForecastDeviationRate float64    `json:"forecast_deviation_rate"`
	StartDate             *time.Time `json:"start_date"`
	EndDate               *time.Time `json:"end_date"`
	RiskLevel             string     `gorm:"size:16;index;default:green" json:"risk_level"`
	Status                string     `gorm:"size:32;index;default:draft" json:"status"`
	TermsJSON             string     `gorm:"type:longtext" json:"terms_json"`
	CreatedBy             uint       `gorm:"index" json:"created_by"`
	UpdatedBy             uint       `gorm:"index" json:"updated_by"`
}

func (MfgSalesAgreement) TableName() string { return "mfg_sales_agreements" }

type MfgAccountForecast struct {
	BaseModel
	TenantID         uint       `gorm:"index;not null" json:"tenant_id"`
	CustomerID       uint       `gorm:"index;not null" json:"customer_id"`
	ProductFamilyID  uint       `gorm:"index" json:"product_family_id"`
	ForecastNo       string     `gorm:"size:64;index;not null" json:"forecast_no"`
	Period           string     `gorm:"size:32;index" json:"period"`
	ForecastQuantity float64    `json:"forecast_quantity"`
	ActualQuantity   float64    `json:"actual_quantity"`
	ForecastAmount   float64    `json:"forecast_amount"`
	ActualAmount     float64    `json:"actual_amount"`
	DeviationRate    float64    `json:"deviation_rate"`
	ConfidenceLevel  string     `gorm:"size:32;index;default:medium" json:"confidence_level"`
	Source           string     `gorm:"size:64;index;default:manual" json:"source"`
	OwnerID          uint       `gorm:"index" json:"owner_id"`
	NextReviewAt     *time.Time `json:"next_review_at"`
	Status           string     `gorm:"size:32;index;default:active" json:"status"`
	AssumptionJSON   string     `gorm:"type:longtext" json:"assumption_json"`
	CreatedBy        uint       `gorm:"index" json:"created_by"`
	UpdatedBy        uint       `gorm:"index" json:"updated_by"`
}

func (MfgAccountForecast) TableName() string { return "mfg_account_forecasts" }

type MfgKnowledgeEntry struct {
	BaseModel
	TenantID        uint   `gorm:"index;not null" json:"tenant_id"`
	Title           string `gorm:"size:191;index;not null" json:"title"`
	KnowledgeType   string `gorm:"size:64;index;default:quote" json:"knowledge_type"`
	Scene           string `gorm:"size:64;index" json:"scene"`
	ProductFamilyID uint   `gorm:"index" json:"product_family_id"`
	CustomerID      uint   `gorm:"index" json:"customer_id"`
	Problem         string `gorm:"type:text" json:"problem"`
	RootCause       string `gorm:"type:text" json:"root_cause"`
	Solution        string `gorm:"type:text" json:"solution"`
	RiskTip         string `gorm:"type:text" json:"risk_tip"`
	Tags            string `gorm:"size:255" json:"tags"`
	Status          string `gorm:"size:32;index;default:draft" json:"status"`
	CreatedBy       uint   `gorm:"index" json:"created_by"`
	UpdatedBy       uint   `gorm:"index" json:"updated_by"`
}

func (MfgKnowledgeEntry) TableName() string { return "mfg_knowledge_entries" }

type MfgKnowledgeUsage struct {
	BaseModel
	TenantID       uint   `gorm:"index;not null" json:"tenant_id"`
	KnowledgeID    uint   `gorm:"index;not null" json:"knowledge_id"`
	TargetType     string `gorm:"size:64;index;not null" json:"target_type"`
	TargetID       uint   `gorm:"index;not null" json:"target_id"`
	AppliedSummary string `gorm:"type:text" json:"applied_summary"`
	CreatedBy      uint   `gorm:"index" json:"created_by"`
}

func (MfgKnowledgeUsage) TableName() string { return "mfg_knowledge_usages" }

type MetricMfgQuoteSnapshot struct {
	BaseModel
	TenantID           uint      `gorm:"index;not null" json:"tenant_id"`
	StatDate           time.Time `gorm:"type:date;index" json:"stat_date"`
	RFQCount           int       `json:"rfq_count"`
	QuoteAmount        float64   `json:"quote_amount"`
	AvgQuoteCycleHours float64   `json:"avg_quote_cycle_hours"`
	CostingCycleHours  float64   `json:"costing_cycle_hours"`
	ApprovalCycleHours float64   `json:"approval_cycle_hours"`
	LowMarginQuoteRate float64   `json:"low_margin_quote_rate"`
	RefreshStatus      string    `gorm:"size:32;index;default:manual" json:"refresh_status"`
	DataSourceJSON     string    `gorm:"type:longtext" json:"data_source_json"`
	RefreshError       string    `gorm:"type:text" json:"refresh_error"`
}

func (MetricMfgQuoteSnapshot) TableName() string { return "metric_mfg_quote_snapshots" }

type MetricMfgDeliverySnapshot struct {
	BaseModel
	TenantID               uint      `gorm:"index;not null" json:"tenant_id"`
	StatDate               time.Time `gorm:"type:date;index" json:"stat_date"`
	OnTimeRate             float64   `json:"on_time_rate"`
	AvgDelayDays           float64   `json:"avg_delay_days"`
	GreenCount             int       `json:"green_count"`
	YellowCount            int       `json:"yellow_count"`
	RedCount               int       `json:"red_count"`
	ExceptionResponseHours float64   `json:"exception_response_hours"`
	RefreshStatus          string    `gorm:"size:32;index;default:manual" json:"refresh_status"`
	DataSourceJSON         string    `gorm:"type:longtext" json:"data_source_json"`
	RefreshError           string    `gorm:"type:text" json:"refresh_error"`
}

func (MetricMfgDeliverySnapshot) TableName() string { return "metric_mfg_delivery_snapshots" }

type MetricMfgServiceSnapshot struct {
	BaseModel
	TenantID                 uint      `gorm:"index;not null" json:"tenant_id"`
	StatDate                 time.Time `gorm:"type:date;index" json:"stat_date"`
	TicketCount              int       `json:"ticket_count"`
	FirstFixRate             float64   `json:"first_fix_rate"`
	ServiceToOpportunityRate float64   `json:"service_to_opportunity_rate"`
	AssetRepurchaseRate      float64   `json:"asset_repurchase_rate"`
	WarrantyExpiringCount    int       `json:"warranty_expiring_count"`
	RefreshStatus            string    `gorm:"size:32;index;default:manual" json:"refresh_status"`
	DataSourceJSON           string    `gorm:"type:longtext" json:"data_source_json"`
	RefreshError             string    `gorm:"type:text" json:"refresh_error"`
}

func (MetricMfgServiceSnapshot) TableName() string { return "metric_mfg_service_snapshots" }

// User 用户模型
type User struct {
	BaseModel
	TenantModel
	Username           string     `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Password           string     `gorm:"size:255;not null" json:"-"`
	Name               string     `gorm:"size:64;index" json:"name"`
	Gender             string     `gorm:"size:16;index" json:"gender"`
	Phone              string     `gorm:"size:32;index" json:"phone"`
	Email              string     `gorm:"size:191" json:"email"`
	Role               string     `gorm:"size:32;default:user" json:"role"`
	RoleID             uint       `gorm:"index" json:"role_id"`
	DepartmentID       uint       `gorm:"index" json:"department_id"`
	PostID             uint       `gorm:"index" json:"post_id"`
	AccountType        string     `gorm:"size:32;default:sub_user;index" json:"account_type"`
	Status             string     `gorm:"size:32;default:active;index" json:"status"`
	MustChangePassword bool       `gorm:"default:false;index" json:"must_change_password"`
	LanguagePref       string     `gorm:"size:16;default:zh-CN" json:"language_pref"`
	Bio                string     `gorm:"type:text" json:"bio"`
	Address            string     `gorm:"size:255" json:"address"`
	AvatarURL          string     `gorm:"type:longtext" json:"avatar_url"`
	CoverURL           string     `gorm:"size:255" json:"cover_url"`
	LastLogin          *time.Time `json:"last_login"`
}

// PlatformSettingVersion 平台全局配置版本
type PlatformSettingVersion struct {
	BaseModel
	Scope            string     `gorm:"size:32;default:platform;index" json:"scope"`
	Group            string     `gorm:"size:64;index:idx_platform_setting_group_key_ver" json:"group"`
	Key              string     `gorm:"size:128;index:idx_platform_setting_group_key_ver" json:"key"`
	ValueJSON        string     `gorm:"type:longtext" json:"value_json"`
	Version          int        `gorm:"index:idx_platform_setting_group_key_ver" json:"version"`
	Status           string     `gorm:"size:32;index" json:"status"`
	EffectiveAt      *time.Time `json:"effective_at"`
	CreatedBy        uint       `gorm:"index" json:"created_by"`
	PublishedBy      uint       `gorm:"index" json:"published_by"`
	PublishedAt      *time.Time `json:"published_at"`
	RolledBackFromID uint       `gorm:"index" json:"rolled_back_from_id"`
}

// TenantAdminActivationToken 租户管理员激活令牌（仅存摘要，不存明文）
type TenantAdminActivationToken struct {
	BaseModel
	TenantModel
	UserID    uint       `gorm:"index;not null" json:"user_id"`
	TokenHash string     `gorm:"size:64;uniqueIndex;not null" json:"-"`
	Purpose   string     `gorm:"size:32;index;not null" json:"purpose"`
	Status    string     `gorm:"size:32;index;default:pending" json:"status"`
	ExpiresAt time.Time  `gorm:"index;not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	CreatedBy uint       `gorm:"index" json:"created_by"`
	IssuedIP  string     `gorm:"size:64" json:"issued_ip"`
}

// SetPassword 设置密码
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// ApprovalTemplate 审批模板
type ApprovalTemplate struct {
	BaseModel
	TenantModel
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nodes      string `gorm:"type:longtext" json:"nodes"`
	Conditions string `gorm:"type:longtext" json:"conditions"`
}

// ApprovalInstance 审批实例
type ApprovalInstance struct {
	BaseModel
	TenantModel
	TemplateID   uint   `json:"template_id"`
	Title        string `json:"title"`
	BusinessType string `json:"business_type"`
	BusinessID   uint   `json:"business_id"`
	Status       string `json:"status"`
	CurrentNode  int    `json:"current_node"`
	NodesStatus  string `gorm:"type:longtext" json:"nodes_with_status"`
}

// ApprovalActionLog 审批操作日志
type ApprovalActionLog struct {
	BaseModel
	TenantModel
	InstanceID uint   `json:"instance_id"`
	Action     string `json:"action"`
	Comment    string `gorm:"type:text" json:"comment"`
	Operator   string `json:"operator"`
}

// Role 角色
type Role struct {
	BaseModel
	TenantModel
	Name            string `gorm:"size:64;not null" json:"name"`
	Code            string `gorm:"size:64;index" json:"code"`
	Description     string `json:"description"`
	Status          string `gorm:"size:32;default:active;index" json:"status"`
	IsPreset        bool   `gorm:"default:false" json:"is_preset"`
	PermissionsJSON string `gorm:"type:longtext" json:"permissions_json"`
	IsSystem        bool   `gorm:"default:false;index" json:"is_system"`
}

// SystemDepartment 系统部门
type SystemDepartment struct {
	BaseModel
	TenantModel
	ParentID uint   `gorm:"index" json:"parent_id"`
	Name     string `gorm:"size:128;not null;index" json:"name"`
	Code     string `gorm:"size:64;index" json:"code"`
	Leader   string `gorm:"size:64" json:"leader"`
	Phone    string `gorm:"size:32" json:"phone"`
	Email    string `gorm:"size:191" json:"email"`
	Sort     int    `gorm:"default:1" json:"sort"`
	Status   string `gorm:"size:32;default:active;index" json:"status"`
	Remark   string `gorm:"type:text" json:"remark"`
}

// SystemPost 系统岗位
type SystemPost struct {
	BaseModel
	TenantModel
	Name   string `gorm:"size:128;not null;index" json:"name"`
	Code   string `gorm:"size:64;index" json:"code"`
	Sort   int    `gorm:"default:1" json:"sort"`
	Status string `gorm:"size:32;default:active;index" json:"status"`
	Remark string `gorm:"type:text" json:"remark"`
}

// SystemMenu 系统菜单
type SystemMenu struct {
	BaseModel
	TenantModel
	ParentID     uint   `gorm:"index" json:"parent_id"`
	Type         string `gorm:"size:32;default:menu;index" json:"type"`
	Name         string `gorm:"size:128;not null;index" json:"name"`
	Path         string `gorm:"size:191;index" json:"path"`
	Permission   string `gorm:"size:128;index" json:"permission"`
	Icon         string `gorm:"size:128" json:"icon"`
	Sort         int    `gorm:"default:1" json:"sort"`
	Status       string `gorm:"size:32;default:active;index" json:"status"`
	Hidden       bool   `gorm:"default:false;index" json:"hidden"`
	ShowBadge    bool   `gorm:"default:false" json:"show_badge"`
	ComponentKey string `gorm:"size:128" json:"component_key"`
}

// UserSession 登录会话
type UserSession struct {
	BaseModel
	TenantModel
	UserID     uint       `gorm:"index" json:"user_id"`
	SessionID  string     `gorm:"size:96;uniqueIndex;not null" json:"session_id"`
	DeviceType string     `gorm:"size:32;index" json:"device_type"`
	IP         string     `gorm:"size:64" json:"ip"`
	Browser    string     `gorm:"size:64" json:"browser"`
	OS         string     `gorm:"size:64" json:"os"`
	UserAgent  string     `gorm:"type:text" json:"user_agent"`
	LastActive time.Time  `gorm:"index" json:"last_active"`
	ExpiresAt  time.Time  `gorm:"index" json:"expires_at"`
	Revoked    bool       `gorm:"default:false;index" json:"revoked"`
	RevokedAt  *time.Time `json:"revoked_at"`
}

// Permission 权限
type Permission struct {
	BaseModel
	Resource string `gorm:"size:64;uniqueIndex:idx_permission_resource_action" json:"resource"`
	Action   string `gorm:"size:64;uniqueIndex:idx_permission_resource_action" json:"action"`
	NameKey  string `gorm:"size:128" json:"name_key"`
}

// RolePermission 角色权限
type RolePermission struct {
	BaseModel
	RoleID       uint   `json:"role_id"`
	PermissionID uint   `json:"permission_id"`
	DataScope    string `json:"data_scope"`
}

// UserRole 用户角色
type UserRole struct {
	BaseModel
	UserID uint `json:"user_id"`
	RoleID uint `json:"role_id"`
}

// CRMObject 新PRD统一业务对象
// 用于承载企业配置、组织、项目、产品、活动、联系人、成交、线索、仪表盘、
// 团队、热点、交易、工单、订单、主面板等15个业务域的一体化数据。
type CRMObject struct {
	BaseModel
	TenantModel
	Domain       string     `gorm:"size:64;index:idx_crm_object_domain_type" json:"domain"`
	ObjectType   string     `gorm:"size:64;index:idx_crm_object_domain_type" json:"object_type"`
	Name         string     `gorm:"size:191;index" json:"name"`
	Code         string     `gorm:"size:64;index" json:"code"`
	Status       string     `gorm:"size:32;index" json:"status"`
	OwnerID      uint       `gorm:"index" json:"owner_id"`
	DepartmentID uint       `gorm:"index" json:"department_id"`
	CompanyID    uint       `gorm:"index" json:"company_id"`
	ContactID    uint       `gorm:"index" json:"contact_id"`
	DealID       uint       `gorm:"index" json:"deal_id"`
	ProjectID    uint       `gorm:"index" json:"project_id"`
	Amount       float64    `json:"amount"`
	Currency     string     `gorm:"size:16" json:"currency"`
	Score        int        `json:"score"`
	StartAt      *time.Time `json:"start_at"`
	EndAt        *time.Time `json:"end_at"`
	Tags         string     `gorm:"type:text" json:"tags"`
	PayloadJSON  string     `gorm:"type:longtext" json:"payload_json"`
	CreatedBy    uint       `gorm:"index" json:"created_by"`
	UpdatedBy    uint       `gorm:"index" json:"updated_by"`
	CanView      bool       `gorm:"-" json:"can_view,omitempty"`
	CanFollow    bool       `gorm:"-" json:"can_follow,omitempty"`
	CanEdit      bool       `gorm:"-" json:"can_edit,omitempty"`
	CanDelete    bool       `gorm:"-" json:"can_delete,omitempty"`
	CanManageACL bool       `gorm:"-" json:"can_manage_acl,omitempty"`
}

// CRMRelation 新PRD统一对象关系
type CRMRelation struct {
	BaseModel
	TenantModel
	SourceType   string `gorm:"size:64;index" json:"source_type"`
	SourceID     uint   `gorm:"index" json:"source_id"`
	TargetType   string `gorm:"size:64;index" json:"target_type"`
	TargetID     uint   `gorm:"index" json:"target_id"`
	RelationType string `gorm:"size:64;index" json:"relation_type"`
	PayloadJSON  string `gorm:"type:longtext" json:"payload_json"`
}

// CRMObjectACLEntry 业务对象级协作权限。
// 用于私密项目/工单指定可查看、跟进、处理或管理的协作人。
type CRMObjectACLEntry struct {
	BaseModel
	TenantModel
	Domain      string `gorm:"size:64;uniqueIndex:idx_crm_object_acl_scope;index" json:"domain"`
	ObjectID    uint   `gorm:"uniqueIndex:idx_crm_object_acl_scope;index" json:"object_id"`
	UserID      uint   `gorm:"uniqueIndex:idx_crm_object_acl_scope;index" json:"user_id"`
	AccessLevel string `gorm:"size:32;index" json:"access_level"`
	CreatedBy   uint   `gorm:"index" json:"created_by"`
	UpdatedBy   uint   `gorm:"index" json:"updated_by"`
}

func (CRMObjectACLEntry) TableName() string {
	return "crm_object_acl_entries"
}

// TenantAIWallet 记录租户 AI 钱包、额度和用量快照。
type TenantAIWallet struct {
	BaseModel
	TenantModel
	Currency            string `gorm:"size:16;default:USD" json:"currency"`
	BalanceMicros       int64  `gorm:"default:0" json:"balance_micros"`
	ReservedMicros      int64  `gorm:"default:0" json:"reserved_micros"`
	TotalSpentMicros    int64  `gorm:"default:0" json:"total_spent_micros"`
	MonthlyTokenLimit   int64  `gorm:"default:0" json:"monthly_token_limit"`
	MonthlyTokenUsed    int64  `gorm:"default:0" json:"monthly_token_used"`
	HardStop            bool   `gorm:"default:true" json:"hard_stop"`
	Enabled             bool   `gorm:"default:false;index" json:"enabled"`
	LastUsageResetMonth string `gorm:"size:16;index" json:"last_usage_reset_month"`
}

func (TenantAIWallet) TableName() string {
	return "tenant_ai_wallets"
}

// AIUsageLog 是租户 AI 调用的审计和计费用量明细。
type AIUsageLog struct {
	BaseModel
	TenantModel
	UserID                    uint   `gorm:"index" json:"user_id"`
	Feature                   string `gorm:"size:64;index" json:"feature"`
	ObjectDomain              string `gorm:"size:64;index" json:"object_domain"`
	ObjectType                string `gorm:"size:64;index" json:"object_type"`
	ObjectID                  uint   `gorm:"index" json:"object_id"`
	Provider                  string `gorm:"size:64;index" json:"provider"`
	Model                     string `gorm:"size:128;index" json:"model"`
	PromptTokens              int64  `gorm:"default:0" json:"prompt_tokens"`
	CompletionTokens          int64  `gorm:"default:0" json:"completion_tokens"`
	TotalTokens               int64  `gorm:"default:0;index" json:"total_tokens"`
	UnitPriceMicrosPerMillion int64  `gorm:"default:0" json:"unit_price_micros_per_million"`
	CostMicros                int64  `gorm:"default:0" json:"cost_micros"`
	Currency                  string `gorm:"size:16;default:USD" json:"currency"`
	Status                    string `gorm:"size:32;index" json:"status"`
	TraceID                   string `gorm:"size:96;index" json:"trace_id"`
	LatencyMS                 int64  `gorm:"default:0" json:"latency_ms"`
	ErrorMessage              string `gorm:"type:text" json:"error_message"`
	MetadataJSON              string `gorm:"type:longtext" json:"metadata_json"`
}

func (AIUsageLog) TableName() string {
	return "ai_usage_logs"
}

// AIResearchJob 保存公司 AI 深度调研任务、结果和来源证据。
type AIResearchJob struct {
	BaseModel
	TenantModel
	UserID       uint       `gorm:"index" json:"user_id"`
	ObjectDomain string     `gorm:"size:64;index" json:"object_domain"`
	ObjectType   string     `gorm:"size:64;index" json:"object_type"`
	ObjectID     uint       `gorm:"index" json:"object_id"`
	Status       string     `gorm:"size:32;index" json:"status"`
	InputJSON    string     `gorm:"type:longtext" json:"input_json"`
	ResultJSON   string     `gorm:"type:longtext" json:"result_json"`
	SourceJSON   string     `gorm:"type:longtext" json:"source_json"`
	Confidence   float64    `json:"confidence"`
	UsageLogID   uint       `gorm:"index" json:"usage_log_id"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	FinishedAt   *time.Time `json:"finished_at"`
}

func (AIResearchJob) TableName() string {
	return "ai_research_jobs"
}

// AITokenLedgerEntry 是 AI token 钱包的可复算账本流水。
type AITokenLedgerEntry struct {
	BaseModel
	TenantModel
	UserID               uint   `gorm:"index" json:"user_id"`
	UsageLogID           uint   `gorm:"index" json:"usage_log_id"`
	JobID                uint   `gorm:"index" json:"job_id"`
	EntryType            string `gorm:"size:32;index" json:"entry_type"`
	Direction            string `gorm:"size:16;index" json:"direction"`
	AmountMicros         int64  `gorm:"default:0" json:"amount_micros"`
	PromptTokens         int64  `gorm:"default:0" json:"prompt_tokens"`
	CompletionTokens     int64  `gorm:"default:0" json:"completion_tokens"`
	CachedInputTokens    int64  `gorm:"default:0" json:"cached_input_tokens"`
	UncachedInputTokens  int64  `gorm:"default:0" json:"uncached_input_tokens"`
	OutputTokens         int64  `gorm:"default:0" json:"output_tokens"`
	TotalTokens          int64  `gorm:"default:0;index" json:"total_tokens"`
	PriceSnapshotJSON    string `gorm:"type:longtext" json:"price_snapshot_json"`
	BalanceBeforeMicros  int64  `gorm:"default:0" json:"balance_before_micros"`
	BalanceAfterMicros   int64  `gorm:"default:0" json:"balance_after_micros"`
	ReservedBeforeMicros int64  `gorm:"default:0" json:"reserved_before_micros"`
	ReservedAfterMicros  int64  `gorm:"default:0" json:"reserved_after_micros"`
	Status               string `gorm:"size:32;index" json:"status"`
	TraceID              string `gorm:"size:96;index" json:"trace_id"`
	IdempotencyKey       string `gorm:"size:128;index" json:"idempotency_key"`
	Description          string `gorm:"type:text" json:"description"`
}

func (AITokenLedgerEntry) TableName() string {
	return "ai_token_ledger_entries"
}

// TenantAIApplication 记录租户管理员提交的 AI 分析开通/购买申请。
type TenantAIApplication struct {
	BaseModel
	TenantModel
	UserID          uint       `gorm:"index" json:"user_id"`
	ApplicationType string     `gorm:"size:32;default:enable;index" json:"application_type"`
	PackageCode     string     `gorm:"size:64;index" json:"package_code"`
	ExpectedUsage   string     `gorm:"size:128" json:"expected_usage"`
	ContactName     string     `gorm:"size:128" json:"contact_name"`
	ContactEmail    string     `gorm:"size:191" json:"contact_email"`
	ContactPhone    string     `gorm:"size:64" json:"contact_phone"`
	Note            string     `gorm:"type:text" json:"note"`
	Status          string     `gorm:"size:32;default:pending;index" json:"status"`
	ApprovedBy      uint       `gorm:"index" json:"approved_by"`
	ApprovedAt      *time.Time `json:"approved_at"`
	RejectedBy      uint       `gorm:"index" json:"rejected_by"`
	RejectedAt      *time.Time `json:"rejected_at"`
	RejectReason    string     `gorm:"type:text" json:"reject_reason"`
}

func (TenantAIApplication) TableName() string {
	return "tenant_ai_applications"
}

// AIAnalysisPackage 是平台配置的 AI 分析购买套餐。
type AIAnalysisPackage struct {
	BaseModel
	Code              string `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name              string `gorm:"size:128;not null" json:"name"`
	Currency          string `gorm:"size:16;default:USD" json:"currency"`
	PriceMicros       int64  `gorm:"default:0" json:"price_micros"`
	BalanceMicros     int64  `gorm:"default:0" json:"balance_micros"`
	MonthlyTokenLimit int64  `gorm:"default:0" json:"monthly_token_limit"`
	ValidityDays      int    `gorm:"default:0" json:"validity_days"`
	Description       string `gorm:"type:text" json:"description"`
	Status            string `gorm:"size:32;default:active;index" json:"status"`
	Sort              int    `gorm:"default:0;index" json:"sort"`
}

func (AIAnalysisPackage) TableName() string {
	return "ai_analysis_packages"
}

// CRMSetting 新PRD租户级配置
type CRMSetting struct {
	BaseModel
	TenantModel
	Scope     string `gorm:"size:64;index:idx_crm_setting_scope_key" json:"scope"`
	Key       string `gorm:"size:128;index:idx_crm_setting_scope_key" json:"key"`
	ValueJSON string `gorm:"type:longtext" json:"value_json"`
	UpdatedBy uint   `gorm:"index" json:"updated_by"`
}

// PortalTicket 公开门户工单
type PortalTicket struct {
	BaseModel
	TrackingID string `gorm:"size:64;uniqueIndex;not null" json:"tracking_id"`
	TicketID   uint   `json:"ticket_id"`
	Status     string `json:"status"`
	Replies    string `gorm:"type:longtext" json:"replies"`
}

// OperationLog 系统操作日志
type OperationLog struct {
	BaseModel
	TenantModel
	UserID         uint   `gorm:"index" json:"user_id"`
	Username       string `gorm:"size:64;index" json:"username"`
	Operation      string `gorm:"size:64;index" json:"operation"`
	Target         string `gorm:"size:128;index" json:"target"`
	DescriptionKey string `gorm:"size:128" json:"description_key"`
	ParamsJSON     string `gorm:"type:longtext" json:"params_json"`
	Result         string `gorm:"size:32;index" json:"result"`
	Reason         string `gorm:"type:text" json:"reason"`
	IP             string `gorm:"size:64" json:"ip"`
	UserAgent      string `gorm:"type:text" json:"user_agent"`
	Source         string `gorm:"size:32;default:web" json:"source"`
}
