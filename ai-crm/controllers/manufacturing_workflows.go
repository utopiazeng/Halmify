package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

type ManufacturingWorkflowController struct {
	web.Controller
}

func (c *ManufacturingWorkflowController) List() {
	auth, ok := requireManufacturingPermission(c.Ctx, c.permission("view"))
	if !ok {
		return
	}
	resource := mfgPathResource(c.Ctx.Input.Param(":resource"))
	items, total, err := services.ListMfgRecords(auth.TenantID, resource, services.MfgListParams{
		Page:            queryInt(c.Ctx, "page", 1),
		Size:            queryInt(c.Ctx, "size", 20),
		Keyword:         strings.TrimSpace(c.Ctx.Input.Query("keyword")),
		Status:          strings.TrimSpace(c.Ctx.Input.Query("status")),
		CustomerID:      uint(queryInt(c.Ctx, "customer_id", 0)),
		RFQID:           uint(queryInt(c.Ctx, "rfq_id", 0)),
		ProductFamilyID: uint(queryInt(c.Ctx, "product_family_id", 0)),
		OrderID:         uint(queryInt(c.Ctx, "order_id", 0)),
		AssetID:         uint(queryInt(c.Ctx, "asset_id", 0)),
	})
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	items = services.SanitizeMfgRecordsForAccess(resource, items, c.mfgAccess(auth))
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": total})
}

func (c *ManufacturingWorkflowController) Create() {
	resource := mfgPathResource(c.Ctx.Input.Param(":resource"))
	req, ok := bindMfgMap(c)
	if !ok {
		return
	}
	auth, ok := c.requireCreatePermissionForPayload(resource, req)
	if !ok {
		return
	}
	item, err := services.CreateMfgRecord(auth.TenantID, auth.UserID, resource, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	itemID := mfgCreatedItemID(item)
	logMfgOperation(c, auth, "CREATE", resource, itemID, map[string]interface{}{"resource": resource, "id": itemID})
	item = services.SanitizeMfgRecordsForAccess(resource, item, c.mfgAccess(auth))
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) Get() {
	auth, ok := requireManufacturingPermission(c.Ctx, c.permission("view"))
	if !ok {
		return
	}
	resource := mfgPathResource(c.Ctx.Input.Param(":resource"))
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的制造记录ID")
		return
	}
	item, err := services.GetMfgRecord(auth.TenantID, resource, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	item = services.SanitizeMfgRecordsForAccess(resource, item, c.mfgAccess(auth))
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) Update() {
	resource := mfgPathResource(c.Ctx.Input.Param(":resource"))
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的制造记录ID")
		return
	}
	auth, ok := c.requireUpdatePermission(resource, id)
	if !ok {
		return
	}
	req, ok := bindMfgMap(c)
	if !ok {
		return
	}
	if resource == "rfq-clarifications" && !canManageMfgRFQClarification(auth) {
		req = answerOnlyMfgRFQClarificationInput(req)
	}
	beforeRefs, err := services.GetMfgRecordFileReferences(auth.TenantID, resource, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	item, err := services.UpdateMfgRecord(auth.TenantID, auth.UserID, resource, id, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	afterRefs := services.MfgRecordFileReferences(resource, item)
	removedRefs := services.RemovedMfgFileReferences(beforeRefs, afterRefs)
	if err := cleanupMfgFileReferences(auth, removedRefs); err != nil {
		utils.Error(c.Ctx, http.StatusBadGateway, err.Error())
		return
	}
	logMfgOperation(c, auth, "UPDATE", resource, id, map[string]interface{}{"resource": resource, "id": id})
	item = services.SanitizeMfgRecordsForAccess(resource, item, c.mfgAccess(auth))
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) Delete() {
	resource := mfgPathResource(c.Ctx.Input.Param(":resource"))
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的制造记录ID")
		return
	}
	auth, ok := c.requireDeletePermission(resource, id)
	if !ok {
		return
	}
	refs, err := services.GetMfgRecordFileReferences(auth.TenantID, resource, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	if err := cleanupMfgFileReferences(auth, refs); err != nil {
		utils.Error(c.Ctx, http.StatusBadGateway, err.Error())
		return
	}
	if err := services.DeleteMfgRecord(auth.TenantID, resource, id); err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "DELETE", resource, id, map[string]interface{}{"resource": resource, "id": id, "file_cleanup_count": len(refs)})
	utils.Success(c.Ctx, map[string]interface{}{"deleted_id": id})
}

func (c *ManufacturingWorkflowController) SubmitRFQ() {
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的 RFQ ID")
		return
	}
	auth, ok := c.requireUpdatePermission("rfqs", id)
	if !ok {
		return
	}
	item, err := services.SubmitMfgRFQ(auth.TenantID, auth.UserID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "SUBMIT_RFQ", "rfqs", id, map[string]interface{}{"id": id})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) StartRFQCosting() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:rfq:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的 RFQ ID")
		return
	}
	if !c.canCreateCostForRFQ(auth, id) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	result, err := services.StartMfgRFQCosting(auth.TenantID, auth.UserID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "START_RFQ_COSTING", "rfqs", id, map[string]interface{}{"id": id, "cost_sheet_id": result.CostSheet.ID})
	costSheet := services.SanitizeMfgRecordsForAccess("cost-sheets", result.CostSheet, c.mfgAccess(auth))
	utils.Success(c.Ctx, map[string]interface{}{"item": result.RFQ, "cost_sheet": costSheet})
}

func (c *ManufacturingWorkflowController) CheckRFQReadiness() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:rfq:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的 RFQ ID")
		return
	}
	result, err := services.CheckMfgRFQReadiness(auth.TenantID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) RFQQuoteWorkspace() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:rfq:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的RFQ ID")
		return
	}
	result, err := services.MfgRFQQuoteWorkspaceForAccess(auth.TenantID, id, c.mfgAccess(auth))
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) ApplyKnowledge() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:knowledge:manage")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的经验ID")
		return
	}
	req, ok := bindMfgMap(c)
	if !ok {
		return
	}
	item, err := services.ApplyMfgKnowledge(auth.TenantID, auth.UserID, id, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "APPLY_KNOWLEDGE", "knowledge", id, map[string]interface{}{"id": id, "target_type": req["target_type"], "target_id": req["target_id"]})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) CreateRFQClarification() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:rfq:manage")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的 RFQ ID")
		return
	}
	req, ok := bindMfgMap(c)
	if !ok {
		return
	}
	item, err := services.CreateMfgClarification(auth.TenantID, auth.UserID, id, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CREATE_RFQ_CLARIFICATION", "rfqs", id, map[string]interface{}{"id": id})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) RFQClarificationAnswerHistories() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:rfq:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的技术澄清ID")
		return
	}
	items, err := services.ListMfgRFQClarificationAnswerHistories(auth.TenantID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": len(items)})
}

func (c *ManufacturingWorkflowController) ConfigureCPQ() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:cpq:manage")
	if !ok {
		return
	}
	req, ok := bindMfgMap(c)
	if !ok {
		return
	}
	result, err := services.ConfigureMfgCPQ(auth.TenantID, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) CreateBOMSnapshotFromConfig() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:product:view")
	if !ok {
		return
	}
	raw, ok := bindMfgMap(c)
	if !ok {
		return
	}
	req := services.MfgBOMSnapshotInput{
		BusinessName:        strings.TrimSpace(fmt.Sprint(raw["business_name"])),
		ProductFamilyID:     uintFromValue(raw["product_family_id"]),
		ConfigModelID:       uintFromValue(raw["config_model_id"]),
		BOMTemplateID:       uintFromValue(raw["bom_template_id"]),
		RFQID:               uintFromValue(raw["rfq_id"]),
		ConfigValues:        mapFromControllerValue(raw["config_values"]),
		BOMLines:            raw["bom_lines"],
		SourceConfigVersion: strings.TrimSpace(fmt.Sprint(raw["source_config_version"])),
		SaveAsTemplate:      boolFromControllerValue(raw["save_as_template"]),
		TemplateNo:          strings.TrimSpace(fmt.Sprint(raw["template_no"])),
		TemplateVersion:     strings.TrimSpace(fmt.Sprint(raw["template_version"])),
		TemplateStatus:      strings.TrimSpace(fmt.Sprint(raw["template_status"])),
		TemplateRule:        raw["template_rule"],
	}
	if !c.canCreateBOMForRFQ(auth, req.RFQID) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	item, err := services.CreateMfgBOMSnapshotFromConfig(auth.TenantID, auth.UserID, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CREATE_MFG_BOM_SNAPSHOT_FROM_CONFIG", "bom-snapshots", item.ID, map[string]interface{}{"id": item.ID})
	utils.Success(c.Ctx, map[string]interface{}{"item": services.SanitizeMfgRecordsForAccess("bom-snapshots", item, c.mfgAccess(auth))})
}

func (c *ManufacturingWorkflowController) CalculateCostSheet() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:cost:view")
	if !ok {
		return
	}
	req, ok := bindMfgMap(c)
	if !ok {
		return
	}
	if !c.canCreateCostForRFQ(auth, uintFromValue(req["rfq_id"])) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	item, err := services.CalculateMfgCostSheet(auth.TenantID, auth.UserID, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CALCULATE_COST_SHEET", "cost-sheets", item.ID, map[string]interface{}{"id": item.ID, "margin_level": item.MarginLevel})
	utils.Success(c.Ctx, map[string]interface{}{"item": services.SanitizeMfgRecordsForAccess("cost-sheets", item, c.mfgAccess(auth))})
}

func (c *ManufacturingWorkflowController) CompleteCostSheet() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:cost:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的成本核价ID")
		return
	}
	if !c.canCreateCostForRFQ(auth, c.rfqIDFromCostSheet(auth.TenantID, id)) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	req, ok := bindMfgMap(c)
	if !ok {
		return
	}
	item, err := services.CompleteMfgCostSheet(auth.TenantID, auth.UserID, id, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "COMPLETE_COST_SHEET", "cost-sheets", item.ID, map[string]interface{}{"id": item.ID, "margin_level": item.MarginLevel})
	utils.Success(c.Ctx, map[string]interface{}{"item": services.SanitizeMfgRecordsForAccess("cost-sheets", item, c.mfgAccess(auth))})
}

func (c *ManufacturingWorkflowController) CreateQuote() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:quote:view")
	if !ok {
		return
	}
	req, ok := bindMfgMap(c)
	if !ok {
		return
	}
	rfqID := uintFromValue(req["rfq_id"])
	if rfqID == 0 {
		rfqID = c.rfqIDFromCostSheet(auth.TenantID, uintFromValue(req["cost_sheet_id"]))
	}
	if !c.canCreateQuoteForRFQ(auth, rfqID) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	item, err := services.CreateMfgQuote(auth.TenantID, auth.UserID, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CREATE_MFG_QUOTE", "quotes", item.ID, map[string]interface{}{"id": item.ID, "quote_no": item.QuoteNo})
	utils.Success(c.Ctx, map[string]interface{}{"item": services.SanitizeMfgRecordsForAccess("quotes", item, c.mfgAccess(auth))})
}

func (c *ManufacturingWorkflowController) CreateQuoteRevision() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:quote:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的报价ID")
		return
	}
	if !c.canManageQuoteByID(auth, id) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	item, err := services.CreateMfgQuoteRevision(auth.TenantID, auth.UserID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CREATE_MFG_QUOTE_REVISION", "quotes", id, map[string]interface{}{"source_id": id, "id": item.ID, "quote_no": item.QuoteNo, "revision": item.Revision})
	utils.Success(c.Ctx, map[string]interface{}{"item": services.SanitizeMfgRecordsForAccess("quotes", item, c.mfgAccess(auth))})
}

func (c *ManufacturingWorkflowController) SubmitQuoteApproval() {
	c.changeQuoteApproval("pending", "SUBMIT_MFG_QUOTE_APPROVAL")
}
func (c *ManufacturingWorkflowController) ApproveQuote() {
	c.changeQuoteApproval("approved", "APPROVE_MFG_QUOTE")
}
func (c *ManufacturingWorkflowController) RejectQuote() {
	c.changeQuoteApproval("rejected", "REJECT_MFG_QUOTE")
}

func (c *ManufacturingWorkflowController) SendQuote() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:quote:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的报价ID")
		return
	}
	if !c.canManageQuoteByID(auth, id) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	item, err := services.SendMfgQuote(auth.TenantID, auth.UserID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "SEND_MFG_QUOTE", "quotes", id, map[string]interface{}{"id": id})
	utils.Success(c.Ctx, map[string]interface{}{"item": services.SanitizeMfgRecordsForAccess("quotes", item, c.mfgAccess(auth))})
}

func (c *ManufacturingWorkflowController) TrackQuoteCustomerView() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:quote:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的报价ID")
		return
	}
	item, err := services.MfgQuoteCustomerView(auth.TenantID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "TRACK_MFG_QUOTE_CUSTOMER_VIEW", "quotes", id, map[string]interface{}{"id": id})
	utils.Success(c.Ctx, map[string]interface{}{"item": services.SanitizeMfgRecordsForAccess("quotes", item, c.mfgAccess(auth))})
}

func (c *ManufacturingWorkflowController) ConfirmQuoteByCustomer() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:quote:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的报价ID")
		return
	}
	if !c.canManageQuoteByID(auth, id) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	item, err := services.ConfirmMfgQuoteByCustomer(auth.TenantID, auth.UserID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CONFIRM_MFG_QUOTE_BY_CUSTOMER", "quotes", id, map[string]interface{}{"id": id})
	utils.Success(c.Ctx, map[string]interface{}{"item": services.SanitizeMfgRecordsForAccess("quotes", item, c.mfgAccess(auth))})
}

func (c *ManufacturingWorkflowController) ViewDrawing() {
	c.recordDrawingAccess("view", "mfg:rfq:view")
}

func (c *ManufacturingWorkflowController) DownloadDrawing() {
	c.recordDrawingAccess("download", "mfg:rfq:manage")
}

func (c *ManufacturingWorkflowController) ExternalSendDrawing() {
	c.recordDrawingAccess("external-send", "mfg:rfq:manage")
}

func (c *ManufacturingWorkflowController) ApproveDrawingAccess() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:rfq:manage")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的图纸ID")
		return
	}
	item, err := services.ApproveMfgDrawingAccess(auth.TenantID, auth.UserID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "APPROVE_MFG_DRAWING_ACCESS", "drawings", id, map[string]interface{}{"id": id})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) CreateOrderFromQuote() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:quote:view")
	if !ok {
		return
	}
	var req services.MfgOrderCreateInput
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if !c.canManageQuoteByID(auth, req.QuoteID) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	result, err := services.CreateMfgOrderFromQuote(auth.TenantID, auth.UserID, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CREATE_MFG_ORDER_FROM_QUOTE", "quotes", req.QuoteID, map[string]interface{}{"quote_id": req.QuoteID})
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) OrderPrefillFromQuote() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:quote:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的报价ID")
		return
	}
	if !c.canManageQuoteByID(auth, id) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	result, err := services.MfgOrderPrefillFromQuote(auth.TenantID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) CreateAssetsFromShipment() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:asset:manage")
	if !ok {
		return
	}
	var req services.MfgShipmentAssetInput
	if !bindJSON(c.Ctx, &req) {
		return
	}
	result, err := services.CreateMfgAssetsFromShipment(auth.TenantID, auth.UserID, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CREATE_MFG_ASSETS_FROM_SHIPMENT", "assets", 0, map[string]interface{}{"order_id": req.OrderID, "asset_count": len(req.SerialNumbers)})
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) AssetServiceHistory() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:asset:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的资产ID")
		return
	}
	result, err := services.MfgAssetServiceHistory(auth.TenantID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) AssetCustomerView() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:asset:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的资产ID")
		return
	}
	result, err := services.MfgAssetCustomerView(auth.TenantID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "TRACK_MFG_ASSET_CUSTOMER_VIEW", "assets", id, map[string]interface{}{"id": id})
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) CreateServiceTicketFromIoTAlarm() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:service:manage")
	if !ok {
		return
	}
	var req services.MfgIoTAlarmInput
	if !bindJSON(c.Ctx, &req) {
		return
	}
	result, err := services.CreateMfgServiceTicketFromIoTAlarm(auth.TenantID, auth.UserID, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CREATE_MFG_SERVICE_TICKET_FROM_IOT_ALARM", "service-tickets", 0, map[string]interface{}{"asset_id": req.AssetID, "alarm_code": req.AlarmCode})
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) AuthorizeAssetIoT() {
	c.updateAssetIoTAuthorization(true, "AUTHORIZE_MFG_ASSET_IOT")
}

func (c *ManufacturingWorkflowController) RevokeAssetIoT() {
	c.updateAssetIoTAuthorization(false, "REVOKE_MFG_ASSET_IOT")
}

func (c *ManufacturingWorkflowController) CreateOpportunityFromServiceTicket() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:service:manage")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的服务工单ID")
		return
	}
	var req services.MfgServiceOpportunityInput
	if !bindJSON(c.Ctx, &req) {
		return
	}
	result, err := services.CreateMfgOpportunityFromServiceTicket(auth.TenantID, auth.UserID, id, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CREATE_MFG_SERVICE_OPPORTUNITY", "service-tickets", id, map[string]interface{}{"ticket_id": id})
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) CreateSparePartQuoteForServiceTicket() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:service:manage")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的服务工单ID")
		return
	}
	req, ok := bindMfgMap(c)
	if !ok {
		return
	}
	item, err := services.CreateMfgSparePartQuoteForTicket(auth.TenantID, auth.UserID, id, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CREATE_MFG_SPARE_PART_QUOTE", "service-tickets", id, map[string]interface{}{"ticket_id": id, "quote_id": item.ID})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) RefreshMetricSnapshots() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:metrics:manage")
	if !ok {
		return
	}
	result, err := services.RefreshMfgMetricSnapshots(auth.TenantID, auth.UserID)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "REFRESH_MFG_METRIC_SNAPSHOTS", "metrics", 0, map[string]interface{}{})
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) ListIntegrationOutbox() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:metrics:view")
	if !ok {
		return
	}
	result, err := services.ListMfgIntegrationOutbox(auth.TenantID, services.MfgListParams{
		Page:    queryInt(c.Ctx, "page", 1),
		Size:    queryInt(c.Ctx, "size", 20),
		Status:  strings.TrimSpace(c.Ctx.Input.Query("status")),
		Keyword: strings.TrimSpace(c.Ctx.Input.Query("keyword")),
	})
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) RetryIntegrationOutbox() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:metrics:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的集成补偿记录ID")
		return
	}
	item, err := services.RetryMfgIntegrationOutbox(auth.TenantID, auth.UserID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "RETRY_MFG_INTEGRATION_OUTBOX", "integration-outbox", id, map[string]interface{}{"id": id})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) DeadLetterIntegrationOutbox() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:metrics:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的集成补偿记录ID")
		return
	}
	req := map[string]interface{}{}
	if len(c.Ctx.Input.RequestBody) > 0 {
		var bindOK bool
		req, bindOK = bindMfgMap(c)
		if !bindOK {
			return
		}
	}
	item, err := services.DeadLetterMfgIntegrationOutbox(auth.TenantID, auth.UserID, id, fmt.Sprint(req["reason"]))
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "DEAD_LETTER_MFG_INTEGRATION_OUTBOX", "integration-outbox", id, map[string]interface{}{"id": id})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) MetricDrilldown() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:metrics:view")
	if !ok {
		return
	}
	result, err := services.MfgMetricDrilldown(auth.TenantID, strings.TrimSpace(c.Ctx.Input.Query("metric")), services.MfgListParams{
		Page:       queryInt(c.Ctx, "page", 1),
		Size:       queryInt(c.Ctx, "size", 20),
		CustomerID: uint(queryInt(c.Ctx, "customer_id", 0)),
		OrderID:    uint(queryInt(c.Ctx, "order_id", 0)),
		AssetID:    uint(queryInt(c.Ctx, "asset_id", 0)),
		Status:     strings.TrimSpace(c.Ctx.Input.Query("status")),
		Keyword:    strings.TrimSpace(c.Ctx.Input.Query("keyword")),
	})
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) QuoteDiff() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:quote:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的报价ID")
		return
	}
	result, err := services.MfgQuoteDiff(auth.TenantID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) QuoteRevisions() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:quote:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的报价ID")
		return
	}
	result, err := services.MfgQuoteRevisions(auth.TenantID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) CreateSampleTestRecord() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:sample:manage")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的打样项目ID")
		return
	}
	req, ok := bindMfgMap(c)
	if !ok {
		return
	}
	item, err := services.CreateMfgSampleTestRecord(auth.TenantID, auth.UserID, id, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "CREATE_MFG_SAMPLE_TEST_RECORD", "sample-projects", id, map[string]interface{}{"sample_project_id": id, "test_record_id": item.ID})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) SampleProgressHistory() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:sample:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的打样项目ID")
		return
	}
	items, err := services.ListMfgSampleProgressHistories(auth.TenantID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": len(items)})
}

func (c *ManufacturingWorkflowController) RaiseProductionRisk() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:production:manage")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的生产进度ID")
		return
	}
	req, ok := bindMfgMap(c)
	if !ok {
		return
	}
	item, err := services.RaiseMfgProductionRisk(auth.TenantID, auth.UserID, id, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, "RAISE_MFG_PRODUCTION_RISK", "production-progress", id, map[string]interface{}{"id": id, "risk_level": item.RiskLevel})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) ProductionProgressHistory() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:production:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的生产进度ID")
		return
	}
	items, err := services.ListMfgProductionProgressHistories(auth.TenantID, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": len(items)})
}

func (c *ManufacturingWorkflowController) ECNImpactAnalysis() {
	c.changeECN("impact-analysis", "MFG_ECN_IMPACT_ANALYSIS")
}
func (c *ManufacturingWorkflowController) SubmitECNApproval() {
	c.changeECN("submit-approval", "SUBMIT_MFG_ECN_APPROVAL")
}
func (c *ManufacturingWorkflowController) ApproveECN() { c.changeECN("approve", "APPROVE_MFG_ECN") }
func (c *ManufacturingWorkflowController) RejectECN()  { c.changeECN("reject", "REJECT_MFG_ECN") }
func (c *ManufacturingWorkflowController) ConfirmECN() {
	c.changeECN("customer-confirm", "CONFIRM_MFG_ECN")
}
func (c *ManufacturingWorkflowController) EffectiveECN() {
	c.changeECN("effective", "EFFECTIVE_MFG_ECN")
}

func (c *ManufacturingWorkflowController) SubmitCompensationApproval() {
	c.changeCompensation("submit-approval", "SUBMIT_MFG_COMPENSATION_APPROVAL")
}
func (c *ManufacturingWorkflowController) ApproveCompensation() {
	c.changeCompensation("approve", "APPROVE_MFG_COMPENSATION")
}
func (c *ManufacturingWorkflowController) RejectCompensation() {
	c.changeCompensation("reject", "REJECT_MFG_COMPENSATION")
}
func (c *ManufacturingWorkflowController) CloseCompensation() {
	c.changeCompensation("close", "CLOSE_MFG_COMPENSATION")
}

func (c *ManufacturingWorkflowController) Dashboard() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:metrics:view")
	if !ok {
		return
	}
	result, err := services.MfgDashboard(auth.TenantID)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) updateAssetIoTAuthorization(authorized bool, operation string) {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:asset:manage")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的资产ID")
		return
	}
	req := map[string]interface{}{}
	if len(c.Ctx.Input.RequestBody) > 0 {
		var bindOK bool
		req, bindOK = bindMfgMap(c)
		if !bindOK {
			return
		}
	}
	item, err := services.UpdateMfgAssetIoTAuthorization(auth.TenantID, auth.UserID, id, authorized, fmt.Sprint(req["iot_asset_ref"]))
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, operation, "assets", id, map[string]interface{}{"id": id, "iot_authorized": authorized})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) changeQuoteApproval(status string, operation string) {
	if status == "approved" || status == "rejected" {
		auth, ok := requireManufacturingPermission(c.Ctx, "mfg:quote:approve")
		if !ok {
			return
		}
		if !authCanApproveMfgQuote(auth) {
			utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
			return
		}
		id, valid := pathUint(c.Ctx, ":id")
		if !valid {
			utils.BadRequest(c.Ctx, "无效的报价ID")
			return
		}
		req := bindMfgOptionalMap(c)
		remark := mfgRequestString(req, "remark")
		item, err := services.ChangeMfgQuoteApprovalWithRemark(auth.TenantID, auth.UserID, id, status, remark)
		if err != nil {
			writeManufacturingError(c.Ctx, err)
			return
		}
		logMfgOperation(c, auth, operation, "quotes", id, map[string]interface{}{"id": id, "approval_status": status, "remark": remark})
		utils.Success(c.Ctx, map[string]interface{}{"item": services.SanitizeMfgRecordsForAccess("quotes", item, c.mfgAccess(auth))})
		return
	}
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:quote:view")
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的报价ID")
		return
	}
	if !c.canManageQuoteByID(auth, id) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	req := bindMfgOptionalMap(c)
	remark := mfgRequestString(req, "remark")
	item, err := services.ChangeMfgQuoteApprovalWithRemark(auth.TenantID, auth.UserID, id, status, remark)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, operation, "quotes", id, map[string]interface{}{"id": id, "approval_status": status, "remark": remark})
	utils.Success(c.Ctx, map[string]interface{}{"item": services.SanitizeMfgRecordsForAccess("quotes", item, c.mfgAccess(auth))})
}

func authCanApproveMfgQuote(auth authContext) bool {
	return auth.IsSuperAdmin() || auth.IsTenantAdmin() || authIsSupervisor(auth) || authHasExactPermission(auth, "mfg:quote:approve")
}

func authCanApproveMfgECN(auth authContext) bool {
	return auth.IsSuperAdmin() || auth.IsTenantAdmin() || authIsSupervisor(auth) || authHasExactPermission(auth, "mfg:ecn:approve")
}

func authCanApproveMfgCompensation(auth authContext) bool {
	return auth.IsSuperAdmin() || auth.IsTenantAdmin() || authIsSupervisor(auth) || authHasExactPermission(auth, "mfg:compensation:approve")
}

func (c *ManufacturingWorkflowController) changeECN(action string, operation string) {
	permission := "mfg:ecn:manage"
	if action == "approve" || action == "reject" {
		permission = "mfg:ecn:approve"
	}
	auth, ok := requireManufacturingPermission(c.Ctx, permission)
	if !ok {
		return
	}
	if (action == "approve" || action == "reject") && !authCanApproveMfgECN(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的 ECN ID")
		return
	}
	req := map[string]interface{}{}
	if len(c.Ctx.Input.RequestBody) > 0 {
		var bindOK bool
		req, bindOK = bindMfgMap(c)
		if !bindOK {
			return
		}
	}
	item, err := services.ChangeMfgECNStatus(auth.TenantID, auth.UserID, id, action, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, operation, "ecns", id, map[string]interface{}{"id": id, "action": action})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) changeCompensation(action string, operation string) {
	permission := "mfg:compensation:manage"
	if action == "approve" || action == "reject" {
		permission = "mfg:compensation:approve"
	}
	auth, ok := requireManufacturingPermission(c.Ctx, permission)
	if !ok {
		return
	}
	if (action == "approve" || action == "reject") && !authCanApproveMfgCompensation(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的赔偿记录ID")
		return
	}
	req := map[string]interface{}{}
	if len(c.Ctx.Input.RequestBody) > 0 {
		var bindOK bool
		req, bindOK = bindMfgMap(c)
		if !bindOK {
			return
		}
	}
	item, err := services.ChangeMfgCompensationStatus(auth.TenantID, auth.UserID, id, action, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logMfgOperation(c, auth, operation, "compensation-records", id, map[string]interface{}{"id": id, "action": action})
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingWorkflowController) recordDrawingAccess(action string, permission string) {
	auth, ok := requireManufacturingPermission(c.Ctx, permission)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的图纸ID")
		return
	}
	result, err := services.RecordMfgDrawingAccess(auth.TenantID, auth.UserID, id, action)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, result)
}

func (c *ManufacturingWorkflowController) permission(action string) string {
	resource := mfgPathResource(c.Ctx.Input.Param(":resource"))
	switch resource {
	case "rfqs", "rfq-clarifications", "drawings":
		return "mfg:rfq:" + action
	case "agreements", "forecasts":
		return "mfg:agreement:" + action
	case "product-families", "config-models", "bom-templates", "bom-snapshots":
		return "mfg:product:" + action
	case "cost-sheets":
		return "mfg:cost:" + action
	case "quotes":
		return "mfg:quote:" + action
	case "sample-projects", "sample-test-records":
		return "mfg:sample:" + action
	case "production-progress":
		return "mfg:production:" + action
	case "ecns":
		return "mfg:ecn:" + action
	case "compensation-records":
		return "mfg:compensation:" + action
	case "assets":
		return "mfg:asset:" + action
	case "service-tickets", "spare-part-quotes":
		return "mfg:service:" + action
	case "knowledge", "knowledge-usages":
		return "mfg:knowledge:" + action
	default:
		return "mfg:" + action
	}
}

func (c *ManufacturingWorkflowController) requireCreatePermission(resource string) (authContext, bool) {
	if resource == "rfqs" {
		return requireManufacturingRFQCreatePermission(c.Ctx)
	}
	return requireManufacturingPermission(c.Ctx, c.permission("manage"))
}

func (c *ManufacturingWorkflowController) requireCreatePermissionForPayload(resource string, input map[string]interface{}) (authContext, bool) {
	auth := currentAuth(c.Ctx)
	switch resource {
	case "bom-snapshots":
		if c.canCreateBOMForRFQ(auth, uintFromValue(input["rfq_id"])) {
			return auth, true
		}
	case "cost-sheets":
		if c.canCreateCostForRFQ(auth, uintFromValue(input["rfq_id"])) {
			return auth, true
		}
	case "quotes":
		rfqID := uintFromValue(input["rfq_id"])
		if rfqID == 0 {
			rfqID = c.rfqIDFromCostSheet(auth.TenantID, uintFromValue(input["cost_sheet_id"]))
		}
		if c.canCreateQuoteForRFQ(auth, rfqID) {
			return auth, true
		}
	}
	return c.requireCreatePermission(resource)
}

func (c *ManufacturingWorkflowController) requireUpdatePermission(resource string, id uint) (authContext, bool) {
	if resource == "rfq-clarifications" {
		return requireManufacturingPermission(c.Ctx, "mfg:rfq:view")
	}
	if resource == "rfqs" {
		auth, ok := requireManufacturingRFQCreatePermission(c.Ctx)
		if !ok {
			return auth, false
		}
		rfq, err := services.GetMfgRFQ(auth.TenantID, id)
		if err != nil {
			writeManufacturingError(c.Ctx, err)
			return auth, false
		}
		if canManageManufacturingRFQ(auth, rfq) {
			return auth, true
		}
		utils.Error(c.Ctx, http.StatusForbidden, "无权限访问")
		return auth, false
	}
	auth, ok := requireManufacturingPermission(c.Ctx, c.permission("manage"))
	if !ok {
		return auth, false
	}
	if !c.ensureMfgRecordEditable(auth, resource, id) {
		return auth, false
	}
	return auth, true
}

func (c *ManufacturingWorkflowController) requireDeletePermission(resource string, id uint) (authContext, bool) {
	if resource == "rfq-clarifications" {
		return requireManufacturingPermission(c.Ctx, c.permission("manage"))
	}
	return c.requireUpdatePermission(resource, id)
}

func (c *ManufacturingWorkflowController) ensureMfgRecordEditable(auth authContext, resource string, id uint) bool {
	switch resource {
	case "ecns", "compensation-records":
	default:
		return true
	}
	item, err := services.GetMfgRecord(auth.TenantID, resource, id)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return false
	}
	if isMfgRecordEditable(resource, item) {
		return true
	}
	utils.Error(c.Ctx, http.StatusBadRequest, "当前状态不允许编辑或删除")
	return false
}

func isMfgRecordEditable(resource string, item interface{}) bool {
	switch resource {
	case "ecns":
		ecn, ok := item.(models.MfgECN)
		if !ok {
			return false
		}
		status := strings.ToLower(strings.TrimSpace(ecn.Status))
		approval := strings.ToLower(strings.TrimSpace(ecn.ApprovalStatus))
		return status != "effective" && !stringInSlice(approval, []string{"pending", "approved"})
	case "compensation-records":
		record, ok := item.(models.MfgCompensationRecord)
		if !ok {
			return false
		}
		status := strings.ToLower(strings.TrimSpace(record.Status))
		approval := strings.ToLower(strings.TrimSpace(record.ApprovalStatus))
		return status != "closed" && stringInSlice(approval, []string{"", "draft", "rejected"})
	default:
		return true
	}
}

func requireManufacturingRFQCreatePermission(ctx *context.Context) (authContext, bool) {
	auth := currentAuth(ctx)
	if auth.TenantID == 0 {
		utils.Error(ctx, http.StatusForbidden, "缺少租户上下文")
		return auth, false
	}
	if !services.IsIndustryPackageEnabled(auth.TenantID, services.IndustryPackageManufacturing) {
		writeIndustryPackageError(ctx, fmt.Errorf(services.IndustryPackageErrorDisabled))
		return auth, false
	}
	if canCreateManufacturingRFQ(auth) {
		return auth, true
	}
	utils.Error(ctx, http.StatusForbidden, "无权限访问")
	return auth, false
}

func canCreateManufacturingRFQ(auth authContext) bool {
	return auth.IsSuperAdmin() ||
		auth.IsTenantAdmin() ||
		authIsSupervisor(auth) ||
		authIsSales(auth) ||
		authHasPermissionAny(auth, "mfg:rfq:manage")
}

func canManageManufacturingRFQ(auth authContext, rfq models.MfgRFQ) bool {
	if auth.UserID > 0 && rfq.CreatedBy == auth.UserID {
		return true
	}
	return auth.IsSuperAdmin() ||
		auth.IsTenantAdmin() ||
		authIsSupervisor(auth) ||
		authHasPermissionAny(auth, "mfg:rfq:manage")
}

func canManageMfgRFQClarification(auth authContext) bool {
	return auth.IsSuperAdmin() ||
		auth.IsTenantAdmin() ||
		authIsSupervisor(auth) ||
		authHasPermissionAny(auth, "mfg:rfq:manage")
}

func answerOnlyMfgRFQClarificationInput(input map[string]interface{}) map[string]interface{} {
	filtered := map[string]interface{}{}
	for _, key := range []string{"rfq_id", "answer", "status"} {
		if value, ok := input[key]; ok {
			filtered[key] = value
		}
	}
	return filtered
}

func (c *ManufacturingWorkflowController) canCreateBOMForRFQ(auth authContext, rfqID uint) bool {
	if authHasPermissionAny(auth, "mfg:product:manage") {
		return true
	}
	return c.canAdvanceOwnManufacturingRFQ(auth, rfqID)
}

func (c *ManufacturingWorkflowController) canCreateCostForRFQ(auth authContext, rfqID uint) bool {
	if authHasPermissionAny(auth, "mfg:cost:manage") {
		return true
	}
	return c.canAdvanceOwnManufacturingRFQ(auth, rfqID)
}

func (c *ManufacturingWorkflowController) canCreateQuoteForRFQ(auth authContext, rfqID uint) bool {
	if authHasPermissionAny(auth, "mfg:quote:manage") {
		return true
	}
	return c.canAdvanceOwnManufacturingRFQ(auth, rfqID)
}

func (c *ManufacturingWorkflowController) canManageQuoteByID(auth authContext, quoteID uint) bool {
	if auth.TenantID == 0 || quoteID == 0 {
		return false
	}
	if authHasPermissionAny(auth, "mfg:quote:manage") {
		return true
	}
	rfqID, err := services.GetMfgQuoteRFQID(auth.TenantID, quoteID)
	if err != nil {
		return false
	}
	return c.canCreateQuoteForRFQ(auth, rfqID)
}

func (c *ManufacturingWorkflowController) canAdvanceOwnManufacturingRFQ(auth authContext, rfqID uint) bool {
	if auth.TenantID == 0 || rfqID == 0 {
		return false
	}
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() || authIsSupervisor(auth) {
		return true
	}
	rfq, err := services.GetMfgRFQ(auth.TenantID, rfqID)
	if err != nil {
		return false
	}
	return canManageManufacturingRFQ(auth, rfq)
}

func (c *ManufacturingWorkflowController) rfqIDFromCostSheet(tenantID uint, costSheetID uint) uint {
	if tenantID == 0 || costSheetID == 0 {
		return 0
	}
	item, err := services.GetMfgRecord(tenantID, "cost-sheets", costSheetID)
	if err != nil {
		return 0
	}
	if cost, ok := item.(models.MfgCostSheet); ok {
		return cost.RFQID
	}
	if cost, ok := item.(*models.MfgCostSheet); ok && cost != nil {
		return cost.RFQID
	}
	return 0
}

func authIsSales(auth authContext) bool {
	if auth.TenantID == 0 {
		return false
	}
	roleIDs := roleIDsForAuth(auth)
	if len(roleIDs) == 0 {
		return false
	}
	var roles []models.Role
	if err := utils.DB.Where("tenant_id = ? AND id IN ?", auth.TenantID, roleIDs).Find(&roles).Error; err != nil {
		return false
	}
	for _, role := range roles {
		if isSalesRole(role) {
			return true
		}
	}
	return false
}

func mapFromControllerValue(value interface{}) map[string]interface{} {
	if typed, ok := value.(map[string]interface{}); ok {
		return typed
	}
	return map[string]interface{}{}
}

func boolFromControllerValue(value interface{}) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		text := strings.ToLower(strings.TrimSpace(typed))
		return text == "true" || text == "1" || text == "yes" || text == "y"
	case float64:
		return typed != 0
	case int:
		return typed != 0
	case uint:
		return typed != 0
	default:
		return false
	}
}

func (c *ManufacturingWorkflowController) canViewSensitive(resource string, auth authContext) bool {
	if auth.IsSuperAdmin() || auth.IsTenantAdmin() {
		return true
	}
	switch resource {
	case "cost-sheets":
		return authHasPermissionAny(auth, "mfg:cost:view_sensitive", "mfg:cost:manage")
	case "quotes":
		return authHasPermissionAny(auth, "mfg:quote:manage", "mfg:cost:manage")
	case "spare-part-quotes":
		return authHasPermissionAny(auth, "mfg:service:manage")
	default:
		return true
	}
}

func (c *ManufacturingWorkflowController) mfgAccess(auth authContext) services.MfgAccessContext {
	return services.MfgAccessContext{
		UserID:           auth.UserID,
		TenantID:         auth.TenantID,
		IsAdmin:          auth.IsSuperAdmin() || auth.IsTenantAdmin(),
		IsSupervisor:     authIsSupervisor(auth),
		CanManageRFQ:     authHasPermissionAny(auth, "mfg:rfq:manage"),
		CanManageCost:    authHasPermissionAny(auth, "mfg:cost:manage"),
		CanManageProduct: authHasPermissionAny(auth, "mfg:product:manage"),
		CanManageQuote:   authHasPermissionAny(auth, "mfg:quote:manage"),
		CanApproveQuote:  authHasPermissionAny(auth, "mfg:quote:approve"),
	}
}

func authIsSupervisor(auth authContext) bool {
	if auth.TenantID == 0 {
		return false
	}
	roleIDs := roleIDsForAuth(auth)
	if len(roleIDs) == 0 {
		return false
	}
	var roles []struct {
		Name string
		Code string
	}
	if err := utils.DB.Model(&models.Role{}).Where("tenant_id = ? AND id IN ?", auth.TenantID, roleIDs).Find(&roles).Error; err != nil {
		return false
	}
	for _, role := range roles {
		name := strings.TrimSpace(role.Name)
		code := strings.ToUpper(strings.TrimSpace(role.Code))
		if name == "主管" || name == "SDR主管" || code == "R_SUPERVISOR" || code == "R_SDR_SUPERVISOR" {
			return true
		}
	}
	return false
}

func bindMfgMap(c web.ControllerInterface) (map[string]interface{}, bool) {
	var req map[string]interface{}
	controller, ok := c.(*ManufacturingWorkflowController)
	if !ok {
		return map[string]interface{}{}, false
	}
	if !bindJSON(controller.Ctx, &req) {
		return map[string]interface{}{}, false
	}
	if req == nil {
		req = map[string]interface{}{}
	}
	return req, true
}

func bindMfgOptionalMap(c web.ControllerInterface) map[string]interface{} {
	var req map[string]interface{}
	controller, ok := c.(*ManufacturingWorkflowController)
	if !ok || controller.Ctx == nil || len(controller.Ctx.Input.RequestBody) == 0 {
		return map[string]interface{}{}
	}
	if !bindJSON(controller.Ctx, &req) || req == nil {
		return map[string]interface{}{}
	}
	return req
}

func mfgRequestString(req map[string]interface{}, key string) string {
	if req == nil {
		return ""
	}
	value, ok := req[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if ok {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func cleanupMfgFileReferences(auth authContext, refs []services.MfgFileReference) error {
	for _, ref := range refs {
		status, _, err := deleteRemoteFileByID(auth, ref.FileID, ref.StorageNodeID)
		if err != nil {
			return fmt.Errorf("删除制造文件资源 %s 失败: %w", ref.FileID, err)
		}
		if status != http.StatusNotFound && (status < 200 || status >= 300) {
			return fmt.Errorf("删除制造文件资源 %s 失败，文件服务返回 %d", ref.FileID, status)
		}
	}
	return nil
}

func logMfgOperation(c *ManufacturingWorkflowController, auth authContext, operation string, resource string, id uint, params map[string]interface{}) {
	logControllerOperation(c.Ctx, auth, auth.TenantID, operation, fmt.Sprintf("mfg/%s[%d]", resource, id), "operation.manufacturing", params, "success", "")
}

func mfgCreatedItemID(item interface{}) uint {
	rv := reflect.Indirect(reflect.ValueOf(item))
	if !rv.IsValid() {
		return 0
	}
	if rv.Kind() == reflect.Map {
		for _, key := range []string{"id", "ID"} {
			value := rv.MapIndex(reflect.ValueOf(key))
			if value.IsValid() {
				return uintFromReflectValue(value)
			}
		}
		return 0
	}
	if rv.Kind() != reflect.Struct {
		return 0
	}
	field := rv.FieldByName("ID")
	return uintFromReflectValue(field)
}

func uintFromReflectValue(value reflect.Value) uint {
	value = reflect.Indirect(value)
	if !value.IsValid() {
		return 0
	}
	switch value.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint(value.Uint())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value.Int() > 0 {
			return uint(value.Int())
		}
	case reflect.Float32, reflect.Float64:
		if value.Float() > 0 {
			return uint(value.Float())
		}
	}
	return 0
}

func mfgPathResource(value string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(value)), "/")
}
