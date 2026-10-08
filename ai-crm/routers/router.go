package routers

import (
	"ai-crm/controllers"
	"ai-crm/middleware"

	"github.com/beego/beego/v2/server/web"
)

type route struct {
	path    string
	methods string
}

func registerRoutes(controller web.ControllerInterface, routes []route) {
	for _, item := range routes {
		web.Router(item.path, controller, item.methods)
	}
}

// InitRouter registers the newprd.md product surface only.
//
// Legacy P0/P1/P2 agent demo APIs are intentionally not mounted here. The
// application now exposes the unified CRM platform plus the minimum platform
// services required to operate it: auth, tenant, account, RBAC, audit and the
// customer portal.
func InitRouter() {
	web.InsertFilter("*", web.BeforeRouter, middleware.CORS)
	web.InsertFilter("/api/v1/*", web.BeforeExec, middleware.RequireAuth)
	web.InsertFilter("/api/v1/*", web.BeforeExec, middleware.SecurityRiskGuard)
	web.InsertFilter("/api/v1/*", web.FinishRouter, middleware.OperationAudit)

	registerRoutes(&controllers.OfficialWebsiteController{}, []route{
		{"/", "get:Home"},
		{"/officialwebsite", "get:Home"},
		{"/officialwebsite/", "get:Home"},
		{"/officialwebsite/assets/:name", "get:Asset"},
	})

	registerRoutes(&controllers.PublicConfigController{}, []route{
		{"/api/public/maintenance-notice", "get:MaintenanceNotice"},
	})

	registerRoutes(&controllers.AuthController{}, []route{
		{"/api/v1/auth/register", "post:Register"},
		{"/api/v1/auth/login", "post:Login"},
		{"/api/v1/auth/activate/password", "post:ActivatePassword"},
		{"/api/v1/auth/captcha/challenge", "get:SliderChallenge"},
		{"/api/v1/auth/captcha/verify", "post:VerifySlider"},
		{"/api/v1/auth/sso/oidc", "post:OIDCLogin"},
		{"/api/v1/auth/mfa/setup", "post:SetupMFA"},
		{"/api/v1/auth/mfa/verify", "post:VerifyMFA"},
		{"/api/v1/auth/me", "get:Me"},
		{"/api/v1/auth/logout", "post:Logout"},
		{"/api/v1/auth/refresh", "post:RefreshToken"},
	})

	registerRoutes(&controllers.TenantController{}, []route{
		{"/api/v1/tenants", "post:CreateTenant;get:ListTenants"},
		{"/api/v1/tenants/:id/status", "put:UpdateTenantStatus"},
		{"/api/v1/tenants/:id/quota", "put:UpdateTenantQuota"},
		{"/api/v1/tenants/:id/admin-password", "put:ResetTenantAdminPassword"},
		{"/api/v1/tenants/:id", "delete:DeleteTenant"},
	})

	registerRoutes(&controllers.PlatformController{}, []route{
		{"/api/v1/platform/tenants/summary", "get:TenantSummaries"},
		{"/api/v1/platform/tenants", "post:CreateTenant"},
		{"/api/v1/platform/tenants/:id", "put:UpdateTenant"},
		{"/api/v1/platform/tenants/:id/admin-password/reset", "post:ResetTenantAdminPassword"},
		{"/api/v1/platform/tenants/expiring", "get:ListExpiringTenants"},
		{"/api/v1/platform/runtime/overview", "get:RuntimeOverview"},
		{"/api/v1/platform/plans", "get:ListPlans;post:CreatePlan"},
		{"/api/v1/platform/plans/:code", "put:UpdatePlan"},
		{"/api/v1/platform/settings", "get:ListSettings"},
		{"/api/v1/platform/settings/draft", "post:SaveSettingDraft"},
		{"/api/v1/platform/settings/publish", "post:PublishSetting"},
		{"/api/v1/platform/settings/rollback", "post:RollbackSetting"},
		{"/api/v1/platform/ai/pricing/sync", "post:SyncAIPricing"},
		{"/api/v1/platform/ai-analysis/applications", "get:ListAIAnalysisApplications"},
		{"/api/v1/platform/ai-analysis/applications/:id/approve", "post:ApproveAIAnalysisApplication"},
		{"/api/v1/platform/ai-analysis/applications/:id/reject", "post:RejectAIAnalysisApplication"},
		{"/api/v1/platform/ai-analysis/packages", "get:ListAIAnalysisPackages;put:SaveAIAnalysisPackage;post:SaveAIAnalysisPackage"},
		{"/api/v1/platform/ai-analysis/usage/summary", "get:AIAnalysisUsageSummary"},
		{"/api/v1/platform/ai-analysis/tenants/usage", "get:AIAnalysisUsageSummary"},
		{"/api/v1/platform/ai-analysis/ledger", "get:AIAnalysisLedger"},
		{"/api/v1/platform/ai-analysis/tenants/:tenant_id/wallet", "get:AIAnalysisTenantWallet"},
	})

	registerRoutes(&controllers.IndustryPackageController{}, []route{
		{"/api/v1/industry/packages", "get:ListPackages"},
		{"/api/v1/tenant/industry-package", "get:GetTenantIndustryPackage"},
		{"/api/v1/platform/tenants/:id/industry-package", "put:UpdatePlatformTenantIndustryPackage"},
		{"/api/v1/prd/extensions/manifest", "get:ExtensionsManifest"},
		{"/api/v1/mfg/status", "get:ManufacturingStatus"},
	})

	registerRoutes(&controllers.ManufacturingCustomerController{}, []route{
		{"/api/v1/mfg/customers/:customer_id/profile", "get:GetProfile;put:SaveProfile"},
		{"/api/v1/mfg/customers/:customer_id/decision-chain", "get:ListDecisionChain;post:CreateDecisionChain"},
		{"/api/v1/mfg/customers/:customer_id/decision-chain/:id", "put:UpdateDecisionChain;delete:DeleteDecisionChain"},
		{"/api/v1/mfg/customers/:customer_id/metrics", "get:GetMetrics;post:SaveMetricSnapshot"},
		{"/api/v1/mfg/customers/:customer_id/360", "get:Customer360"},
	})

	registerRoutes(&controllers.ManufacturingWorkflowController{}, []route{
		{"/api/v1/mfg/dashboard", "get:Dashboard"},
		{"/api/v1/mfg/dashboard/refresh-metrics", "post:RefreshMetricSnapshots"},
		{"/api/v1/mfg/dashboard/drilldown", "get:MetricDrilldown"},
		{"/api/v1/mfg/integration-outbox", "get:ListIntegrationOutbox"},
		{"/api/v1/mfg/integration-outbox/:id/retry", "post:RetryIntegrationOutbox"},
		{"/api/v1/mfg/integration-outbox/:id/dead-letter", "post:DeadLetterIntegrationOutbox"},
		{"/api/v1/mfg/rfqs/:id/submit", "post:SubmitRFQ"},
		{"/api/v1/mfg/rfqs/:id/readiness-check", "post:CheckRFQReadiness"},
		{"/api/v1/mfg/rfqs/:id/quote-workspace", "get:RFQQuoteWorkspace"},
		{"/api/v1/mfg/rfqs/:id/clarifications", "post:CreateRFQClarification"},
		{"/api/v1/mfg/rfq-clarifications/:id/answer-histories", "get:RFQClarificationAnswerHistories"},
		{"/api/v1/mfg/rfqs/:id/start-costing", "post:StartRFQCosting"},
		{"/api/v1/mfg/knowledge/:id/apply", "post:ApplyKnowledge"},
		{"/api/v1/mfg/drawings/:id/view", "post:ViewDrawing"},
		{"/api/v1/mfg/drawings/:id/download", "post:DownloadDrawing"},
		{"/api/v1/mfg/drawings/:id/external-send", "post:ExternalSendDrawing"},
		{"/api/v1/mfg/drawings/:id/approve-access", "post:ApproveDrawingAccess"},
		{"/api/v1/mfg/cpq/configure", "post:ConfigureCPQ"},
		{"/api/v1/mfg/bom-snapshots/from-config", "post:CreateBOMSnapshotFromConfig"},
		{"/api/v1/mfg/cost-sheets/calculate", "post:CalculateCostSheet"},
		{"/api/v1/mfg/cost-sheets/:id/complete", "post:CompleteCostSheet"},
		{"/api/v1/mfg/quotes", "post:CreateQuote"},
		{"/api/v1/mfg/quotes/order", "post:CreateOrderFromQuote"},
		{"/api/v1/mfg/quotes/:id/order-prefill", "get:OrderPrefillFromQuote"},
		{"/api/v1/mfg/quotes/:id/submit-approval", "post:SubmitQuoteApproval"},
		{"/api/v1/mfg/quotes/:id/approve", "post:ApproveQuote"},
		{"/api/v1/mfg/quotes/:id/reject", "post:RejectQuote"},
		{"/api/v1/mfg/quotes/:id/send", "post:SendQuote"},
		{"/api/v1/mfg/quotes/:id/customer-view", "post:TrackQuoteCustomerView"},
		{"/api/v1/mfg/quotes/:id/customer-confirm", "post:ConfirmQuoteByCustomer"},
		{"/api/v1/mfg/quotes/:id/revisions", "get:QuoteRevisions"},
		{"/api/v1/mfg/quotes/:id/revisions", "post:CreateQuoteRevision"},
		{"/api/v1/mfg/quotes/:id/diff", "get:QuoteDiff"},
		{"/api/v1/mfg/sample-projects/:id/test-records", "post:CreateSampleTestRecord"},
		{"/api/v1/mfg/sample-projects/:id/progress-history", "get:SampleProgressHistory"},
		{"/api/v1/mfg/production-progress/:id/raise-risk", "post:RaiseProductionRisk"},
		{"/api/v1/mfg/production-progress/:id/progress-history", "get:ProductionProgressHistory"},
		{"/api/v1/mfg/assets/from-shipment", "post:CreateAssetsFromShipment"},
		{"/api/v1/mfg/assets/:id/service-history", "get:AssetServiceHistory"},
		{"/api/v1/mfg/assets/:id/customer-view", "get:AssetCustomerView"},
		{"/api/v1/mfg/assets/:id/iot-authorize", "post:AuthorizeAssetIoT"},
		{"/api/v1/mfg/assets/:id/iot-revoke", "post:RevokeAssetIoT"},
		{"/api/v1/mfg/service-tickets/from-iot-alarm", "post:CreateServiceTicketFromIoTAlarm"},
		{"/api/v1/mfg/service-tickets/:id/spare-part-quote", "post:CreateSparePartQuoteForServiceTicket"},
		{"/api/v1/mfg/service-tickets/:id/create-opportunity", "post:CreateOpportunityFromServiceTicket"},
		{"/api/v1/mfg/ecns/:id/impact-analysis", "post:ECNImpactAnalysis"},
		{"/api/v1/mfg/ecns/:id/submit-approval", "post:SubmitECNApproval"},
		{"/api/v1/mfg/ecns/:id/approve", "post:ApproveECN"},
		{"/api/v1/mfg/ecns/:id/reject", "post:RejectECN"},
		{"/api/v1/mfg/ecns/:id/customer-confirm", "post:ConfirmECN"},
		{"/api/v1/mfg/ecns/:id/effective", "post:EffectiveECN"},
		{"/api/v1/mfg/compensation-records/:id/submit-approval", "post:SubmitCompensationApproval"},
		{"/api/v1/mfg/compensation-records/:id/approve", "post:ApproveCompensation"},
		{"/api/v1/mfg/compensation-records/:id/reject", "post:RejectCompensation"},
		{"/api/v1/mfg/compensation-records/:id/close", "post:CloseCompensation"},
		{"/api/v1/mfg/:resource", "get:List;post:Create"},
		{"/api/v1/mfg/:resource/:id", "get:Get;put:Update;delete:Delete"},
	})

	registerRoutes(&controllers.AccountController{}, []route{
		{"/api/v1/accounts/users", "post:CreateUser;get:ListUsers"},
		{"/api/v1/accounts/users/:id", "put:UpdateUser;delete:DeleteUser"},
		{"/api/v1/accounts/users/:id/password", "put:ResetPassword"},
		{"/api/v1/accounts/me/language", "put:UpdateLanguagePref"},
	})

	registerRoutes(&controllers.ProfileController{}, []route{
		{"/api/v1/accounts/me/profile", "get:GetProfile;put:UpdateProfile"},
		{"/api/v1/accounts/me/avatar", "post:UploadAvatar"},
		{"/api/v1/accounts/me/password", "put:ChangePassword"},
		{"/api/v1/accounts/me/sessions", "get:ListSessions;delete:RevokeOtherSessions"},
	})

	registerRoutes(&controllers.RBACController{}, []route{
		{"/api/v1/rbac/roles", "post:CreateRole;get:ListRoles"},
		{"/api/v1/rbac/roles/:id", "put:UpdateRole;delete:DeleteRole"},
		{"/api/v1/rbac/permissions", "get:ListPermissions"},
		{"/api/v1/rbac/permissions/catalog", "get:PermissionCatalog"},
		{"/api/v1/rbac/role-permissions", "post:AssignPermissions"},
	})

	registerRoutes(&controllers.SystemController{}, []route{
		{"/api/v1/system/departments", "post:CreateDepartment;get:ListDepartments"},
		{"/api/v1/system/departments/:id", "put:UpdateDepartment;delete:DeleteDepartment"},
		{"/api/v1/system/posts", "post:CreatePost;get:ListPosts"},
		{"/api/v1/system/posts/:id", "put:UpdatePost;delete:DeletePost"},
		{"/api/v1/system/menus", "post:CreateMenu;get:ListMenus"},
		{"/api/v1/system/menus/:id", "put:UpdateMenu;delete:DeleteMenu"},
	})

	registerRoutes(&controllers.OperationLogController{}, []route{
		{"/api/v1/operation-logs", "get:ListLogs"},
		{"/api/v1/operation-logs/export", "get:ExportLogs"},
		{"/api/v1/operation-logs/login/export", "get:ExportLoginLogs"},
		{"/api/v1/operation-logs/cleanup", "post:CleanupLogs"},
	})

	registerRoutes(&controllers.IoTController{}, []route{
		{"/api/v1/iot/devices", "get:ListDevices;post:CreateDevice"},
		{"/api/v1/iot/devices/:id", "put:UpdateDevice;delete:DeleteDevice"},
		{"/api/v1/iot/devices/:id/rotate-secret", "post:RotateDeviceSecret"},
		{"/api/v1/iot/devices/:id/preview", "get:PreviewDevice"},
		{"/api/v1/platform/iot/firmwares", "get:ListFirmwares;post:UploadFirmware"},
		{"/api/v1/platform/iot/firmwares/:id/targets", "get:ListFirmwareTargets"},
		{"/api/v1/platform/iot/firmwares/:id/publish", "post:PublishFirmware"},
		{"/api/v1/platform/iot/firmwares/:id/revoke", "post:RevokeFirmware"},
		{"/api/v1/platform/iot/firmwares/:id", "delete:DeleteFirmware"},
		{"/api/iot/v1/bootstrap", "post:BootstrapDevice"},
		{"/api/iot/v1/device-seed/refresh", "post:RefreshDeviceSeed"},
		{"/api/iot/v1/sales-dashboard", "get:SalesDashboard"},
		{"/api/iot/v1/ota/check", "get:OTACheck"},
		{"/api/iot/v1/ota/download/:token", "get:OTADownload"},
		{"/api/iot/v1/ota/confirm", "post:OTAConfirm"},
	})

	registerRoutes(&controllers.CRMController{}, []route{
		{"/api/v1/prd/domains", "get:ListDomains"},
		{"/api/v1/prd/coverage", "get:Coverage"},
		{"/api/v1/prd/schema", "get:Schema"},
		{"/api/v1/prd/actions", "get:Actions"},
		{"/api/v1/prd/summary", "get:Summary"},
		{"/api/v1/prd/cockpit", "get:Cockpit"},
		{"/api/v1/prd/analytics", "get:AnalyticsSummary"},
		{"/api/v1/prd/next-actions", "get:NextBestActions"},
		{"/api/v1/prd/settings", "get:ListSettings;post:SaveSetting"},
		{"/api/v1/prd/settings/exchange-rates/sync", "post:SyncExchangeRates"},
		{"/api/v1/prd/module-access-preview", "get:ModuleAccessPreview"},
		{"/api/v1/prd/approval-templates", "get:ListApprovalTemplates;post:CreateApprovalTemplate"},
		{"/api/v1/prd/approvals", "get:ListApprovalInstances"},
		{"/api/v1/prd/approvals/:id/action", "post:ActApprovalInstance"},
		{"/api/v1/prd/approvals/pending-summary", "get:PendingApprovalSummary"},
		{"/api/v1/prd/integrations/test", "post:TestIntegration"},
		{"/api/v1/prd/integrations/execute", "post:ExecuteIntegration"},
		{"/api/v1/prd/integrations/hr/sync", "post:SyncHR"},
		{"/api/v1/prd/integrations/health", "get:IntegrationHealth"},
		{"/api/v1/prd/relations", "get:ListRelations;post:CreateRelation"},
		{"/api/v1/prd/relations/:id", "delete:DeleteRelation"},
		{"/api/v1/prd/leads/views", "get:ListLeadViews;post:SaveLeadView"},
		{"/api/v1/prd/contacts/views", "get:ListContactViews;post:SaveContactView"},
		{"/api/v1/prd/leads/dashboard", "get:LeadDashboard"},
		{"/api/v1/prd/leads/nurture/templates", "get:ListLeadNurtureTemplates;post:SaveLeadNurtureTemplate"},
		{"/api/v1/prd/leads/nurture/templates/:key", "put:UpdateLeadNurtureTemplate;delete:DeleteLeadNurtureTemplate"},
		{"/api/v1/prd/leads/automation/rules", "get:ListLeadAutomationRules;post:SaveLeadAutomationRules"},
		{"/api/v1/prd/flows/leads/automation/run", "post:RunLeadAutomation"},
		{"/api/v1/prd/flows/leads/dedupe", "post:DedupeLeads"},
		{"/api/v1/prd/flows/leads/sla-recycle", "post:RunLeadSLARecycle"},
		{"/api/v1/prd/flows/leads/:id/status", "post:ChangeLeadStatus"},
		{"/api/v1/prd/flows/leads/:id/merge", "post:MergeLeads"},
		{"/api/v1/prd/flows/leads/:id/release", "post:ReleaseLead"},
		{"/api/v1/prd/flows/leads/:id/follow-up", "post:AddLeadFollowUp"},
		{"/api/v1/prd/flows/leads/:id/nurture", "post:StartLeadNurture"},
		{"/api/v1/prd/flows/leads/:id/convert", "post:ConvertLead"},
		{"/api/v1/prd/flows/leads/:id/assign", "post:AssignLead"},
		{"/api/v1/prd/flows/leads/auto-assign", "post:AutoAssignLeads"},
		{"/api/v1/prd/flows/leads/:id/recycle", "post:RecycleLead"},
		{"/api/v1/prd/flows/leads/:id/score", "post:ScoreLead"},
		{"/api/v1/prd/flows/deal-chain", "get:GetDealChain"},
		{"/api/v1/prd/flows/sales-targets/summary", "get:SalesTargetSummary"},
		{"/api/v1/prd/flows/sales-targets/my", "get:MySalesTargets"},
		{"/api/v1/prd/flows/sales-targets/company", "post:SaveCompanySalesTarget"},
		{"/api/v1/prd/flows/sales-targets/allocate", "post:AllocateSalesTargets"},
		{"/api/v1/prd/flows/sales-targets/my-plan", "post:SaveMySalesPlan"},
		{"/api/v1/prd/flows/campaigns/:id/capture-lead", "post:CaptureCampaignLead"},
		{"/api/v1/prd/flows/campaigns/:id/attribution", "post:GenerateCampaignAttribution"},
		{"/api/v1/prd/flows/campaigns/:id/performance", "get:CampaignPerformance"},
		{"/api/v1/prd/flows/campaigns/:id/follow-up", "post:AddCampaignFollowUp"},
		{"/api/v1/prd/flows/campaigns/:id/files", "get:ListCampaignFiles;post:AddCampaignFile"},
		{"/api/v1/prd/flows/campaigns/:id/files/:file_object_id", "delete:DeleteCampaignFile"},
		{"/api/v1/prd/flows/insights/hotspots", "post:GenerateHotInsights"},
		{"/api/v1/prd/flows/opportunities/:id/stage", "post:AdvanceOpportunityStage"},
		{"/api/v1/prd/flows/opportunities/:id/follow-up", "post:AddOpportunityFollowUp"},
		{"/api/v1/prd/flows/opportunities/:id/reminder", "post:CreateOpportunityReminder"},
		{"/api/v1/prd/flows/opportunities/conflicts", "get:DetectOpportunityConflicts"},
		{"/api/v1/prd/flows/opportunities/:id/quote", "post:CreateQuoteFromOpportunity"},
		{"/api/v1/prd/flows/opportunities/:id/review", "get:GetOpportunityReview;post:SaveOpportunityReview"},
		{"/api/v1/prd/flows/opportunities/:id/win", "post:WinOpportunity"},
		{"/api/v1/prd/flows/opportunities/:id/risk", "post:GenerateOpportunityRisk"},
		{"/api/v1/prd/flows/opportunities/reviews/:id/submit", "post:SubmitOpportunityReview"},
		{"/api/v1/prd/flows/opportunities/reviews/:id/publish", "post:PublishOpportunityReview"},
		{"/api/v1/prd/flows/opportunities/reviews/:id/archive", "post:ArchiveOpportunityReview"},
		{"/api/v1/prd/flows/opportunity-reviews/:id/submit", "post:SubmitOpportunityReview"},
		{"/api/v1/prd/flows/opportunity-reviews/:id/publish", "post:PublishOpportunityReview"},
		{"/api/v1/prd/flows/opportunity-reviews/:id/archive", "post:ArchiveOpportunityReview"},
		{"/api/v1/prd/flows/quotes/:id/send", "post:SendQuote"},
		{"/api/v1/prd/flows/quotes/:id/accept", "post:AcceptQuote"},
		{"/api/v1/prd/flows/quotes/:id/order", "post:CreateOrderFromQuote"},
		{"/api/v1/prd/flows/deals/:id/order", "post:CreateOrderFromDeal"},
		{"/api/v1/prd/flows/orders/:id/status", "post:ChangeOrderStatus"},
		{"/api/v1/prd/flows/orders/:id/fulfill", "post:FulfillOrder"},
		{"/api/v1/prd/flows/orders/:id/pay", "post:MarkOrderPaid"},
		{"/api/v1/prd/flows/orders/:id/payment-plan", "post:SaveOrderPaymentPlan"},
		{"/api/v1/prd/flows/orders/:id/ship", "post:ShipOrder"},
		{"/api/v1/prd/flows/orders/:id/invoice", "post:InvoiceOrder"},
		{"/api/v1/prd/flows/orders/:id/return", "post:ReturnOrder"},
		{"/api/v1/prd/flows/orders/:id/ticket", "post:CreateTicketFromOrder"},
		{"/api/v1/prd/flows/orders/:id/files", "get:ListOrderFiles;post:AddOrderFile"},
		{"/api/v1/prd/flows/orders/:id/files/:file_object_id", "delete:DeleteOrderFile"},
		{"/api/v1/prd/flows/orders/:id/backup", "get:BackupOrder"},
		{"/api/v1/prd/flows/projects/:id/complete", "post:CompleteProject"},
		{"/api/v1/prd/flows/projects/:id/progress", "post:UpdateProjectProgress"},
		{"/api/v1/prd/flows/projects/:id/follow-up", "post:AddProjectFollowUp"},
		{"/api/v1/prd/flows/projects/:id/reminder", "post:CreateProjectReminder"},
		{"/api/v1/prd/flows/projects/:id/owner", "post:SetProjectOwner"},
		{"/api/v1/prd/flows/projects/:id/milestone", "post:SaveProjectMilestone"},
		{"/api/v1/prd/flows/projects/:id/files", "get:ListProjectFiles;post:AddProjectFile"},
		{"/api/v1/prd/flows/projects/:id/files/:file_object_id", "delete:DeleteProjectFile"},
		{"/api/v1/prd/flows/tickets/escalation-queue", "get:TicketEscalationQueue"},
		{"/api/v1/prd/flows/tickets/transfer-queue", "get:TicketTransferQueue"},
		{"/api/v1/prd/flows/tickets/transfer-stats", "get:TicketTransferStats"},
		{"/api/v1/prd/flows/tickets/:id/status", "post:ChangeTicketStatus"},
		{"/api/v1/prd/flows/tickets/:id/assign", "post:AssignTicket"},
		{"/api/v1/prd/flows/tickets/:id/transfer", "post:TransferTicket"},
		{"/api/v1/prd/flows/tickets/:id/auto-assign", "post:AutoAssignTicket"},
		{"/api/v1/prd/flows/tickets/:id/opportunity", "post:CreateOpportunityFromTicket"},
		{"/api/v1/prd/flows/tickets/:id/bug", "post:CreateBugTaskFromTicket"},
		{"/api/v1/prd/flows/tickets/:id/bug-status", "post:SyncTicketBugStatus"},
		{"/api/v1/prd/flows/tickets/:id/note", "post:AddTicketNote"},
		{"/api/v1/prd/flows/tickets/:id/knowledge", "post:CreateKnowledgeFromTicket"},
		{"/api/v1/prd/flows/tickets/:id/knowledge-suggestions", "get:TicketKnowledgeSuggestions"},
		{"/api/v1/prd/flows/tickets/:id/escalate", "post:EscalateTicketSLA"},
		{"/api/v1/prd/flows/tickets/:id/close", "post:CloseTicket"},
		{"/api/v1/prd/flows/tickets/:id/files", "get:ListTicketFiles;post:AddTicketFile"},
		{"/api/v1/prd/flows/tickets/:id/files/:file_object_id", "delete:DeleteTicketFile"},
		{"/api/v1/prd/flows/workspace/:id/done", "post:CompleteWorkspaceTask"},
		{"/api/v1/prd/flows/workspace/:id/read", "post:MarkWorkspaceNotificationRead"},
		{"/api/v1/prd/flows/contacts/dedupe", "post:DedupeContactObjects"},
		{"/api/v1/prd/flows/contacts/batch", "post:BatchUpdateContactObjects"},
		{"/api/v1/prd/flows/contacts/merge", "post:MergeContacts"},
		{"/api/v1/prd/flows/contacts/link-company", "post:LinkContactCompany"},
		{"/api/v1/prd/flows/contacts/unlink-company", "post:UnlinkContactCompany"},
		{"/api/v1/prd/contacts/company-research/jobs", "get:ListCompanyResearchJobs;post:CreateCompanyResearchJob"},
		{"/api/v1/prd/contacts/company-research/jobs/:id", "get:GetCompanyResearchJob"},
		{"/api/v1/prd/contacts/company-research/jobs/:id/apply", "post:ApplyCompanyResearchJob"},
		{"/api/v1/prd/ai/usage/summary", "get:AIUsageSummary"},
		{"/api/v1/prd/ai/usage/logs", "get:AIUsageLogs"},
		{"/api/v1/prd/ai/pricing", "get:AIPricing"},
		{"/api/v1/prd/flows/tenant-ai-analysis/status", "get:TenantAIAnalysisStatus"},
		{"/api/v1/prd/flows/tenant-ai-analysis/settings", "get:GetTenantAIAnalysisSettings;put:SaveTenantAIAnalysisSettings"},
		{"/api/v1/prd/flows/tenant-ai-analysis/applications", "get:ListTenantAIApplications;post:CreateTenantAIApplication"},
		{"/api/v1/prd/flows/tenant-ai-analysis/estimate", "post:EstimateTenantAIAnalysis"},
		{"/api/v1/prd/flows/tenant-ai-analysis/jobs", "get:ListTenantAIAnalysisJobs;post:CreateTenantAIAnalysisJob"},
		{"/api/v1/prd/flows/tenant-ai-analysis/jobs/:id", "get:GetTenantAIAnalysisJob"},
		{"/api/v1/prd/flows/tenant-ai-analysis/jobs/:id/export.pdf", "get:ExportTenantAIAnalysisPDF"},
		{"/api/v1/prd/flows/tenant-ai-analysis/usage/summary", "get:TenantAIUsageSummary"},
		{"/api/v1/prd/flows/tenant-ai-analysis/usage/ledger", "get:TenantAIUsageLedger"},
		{"/api/v1/prd/flows/contacts/:id/360", "get:ContactCustomer360"},
		{"/api/v1/prd/flows/contacts/:id/restore", "post:RestoreContactObject"},
		{"/api/v1/prd/:domain/objects", "get:ListObjects;post:CreateObject"},
		{"/api/v1/prd/:domain/objects/import", "post:ImportObjects"},
		{"/api/v1/prd/:domain/objects/export", "get:ExportObjects"},
		{"/api/v1/prd/:domain/objects/:id/acl", "get:GetObjectACL;put:UpdateObjectACL"},
		{"/api/v1/prd/:domain/objects/:id/approval", "post:StartObjectApproval"},
		{"/api/v1/prd/:domain/objects/:id", "get:GetObject;put:UpdateObject;delete:DeleteObject"},
	})

	registerRoutes(&controllers.PortalController{}, []route{
		{"/api/v1/portal/articles", "get:Articles"},
		{"/api/v1/portal/ask", "post:Ask"},
		{"/api/v1/portal/tickets", "post:CreateTicket"},
		{"/api/v1/portal/tickets/:tracking_id", "get:TicketStatus"},
	})

	registerRoutes(&controllers.FileGatewayController{}, []route{
		{"/api/v1/disk/space", "get:DiskSpace"},
		{"/api/v1/files/nodes", "get:ListNodes"},
		{"/api/v1/files/check", "post:Check"},
		{"/api/v1/files/upload", "post:Upload"},
		{"/api/v1/files/:file_id/download", "get:Download"},
		{"/api/v1/files/:file_id/info", "get:Info"},
		{"/api/v1/files/:file_id", "delete:Delete"},
	})

	registerRoutes(&controllers.StorageController{}, []route{
		{"/api/v1/storage/usage", "get:Usage"},
		{"/api/v1/entitlements/seats/usage", "get:SeatUsage"},
		{"/api/v1/entitlements/quote", "post:Quote"},
		{"/api/v1/storage/purchase-requests", "post:PurchaseRequest"},
		{"/api/v1/entitlements/seats/purchase-requests", "post:SeatPurchaseRequest"},
		{"/api/v1/platform/tenants/:id/entitlements/quote", "post:PlatformQuote"},
		{"/api/v1/platform/tenants/:id/entitlements/send-quote", "post:PlatformSendQuote"},
		{"/api/v1/platform/tenants/:id/storage-expansions", "get:PlatformListExpansions;post:PlatformCreateExpansion"},
		{"/api/v1/platform/tenants/:id/storage-expansions/:expansion_id", "put:PlatformUpdateExpansion"},
		{"/api/v1/platform/tenants/:id/storage-expansions/:expansion_id/approve", "post:PlatformApproveStorageExpansion"},
		{"/api/v1/platform/tenants/:id/storage-expansions/:expansion_id/reject", "post:PlatformRejectStorageExpansion"},
		{"/api/v1/platform/tenants/:id/seat-expansions", "get:PlatformSeatUsage;post:PlatformCreateSeatExpansion"},
		{"/api/v1/platform/tenants/:id/seat-expansions/:expansion_id/approve", "post:PlatformApproveSeatExpansion"},
		{"/api/v1/platform/tenants/:id/seat-expansions/:expansion_id/reject", "post:PlatformRejectSeatExpansion"},
	})

	registerRoutes(&controllers.IMController{}, []route{
		{"/internal/official/demo-requests", "post:OfficialDemoRequest"},
		{"/api/v1/im/ws", "get:WS"},
		{"/api/v1/im/contacts", "get:ListContacts"},
		{"/api/v1/im/conversations", "get:ListConversations;post:OpenConversation"},
		{"/api/v1/im/conversations/:id/messages", "get:ListMessages;post:SendMessage"},
		{"/api/v1/im/conversations/:id/read", "post:MarkRead"},
		{"/api/v1/im/conversations/:id/settings", "patch:UpdateConversationSetting"},
		{"/api/v1/im/support/conversation", "post:OpenSupportConversation"},
		{"/api/v1/im/support/conversations", "get:ListSupportConversations"},
		{"/api/v1/im/presence", "patch:UpdatePresence"},
		{"/api/v1/im/unread/summary", "get:UnreadSummary"},
		{"/api/v1/im/system-notifications/unread-count", "get:SystemNotificationUnreadCount"},
		{"/api/v1/im/system-notifications/:id/read", "post:SystemNotificationRead"},
		{"/api/v1/im/system-notifications/read-all", "post:SystemNotificationsReadAll"},
		{"/api/v1/im/system-notifications/archive", "post:ArchiveSystemNotifications"},
		{"/api/v1/im/system-notifications/cleanup", "post:CleanupSystemNotifications"},
		{"/api/v1/im/system-notifications/broadcast", "post:BroadcastSystemNotification"},
		{"/api/v1/im/system-notifications", "get:SystemNotifications"},
	})
}
