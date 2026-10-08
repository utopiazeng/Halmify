package controllers

import (
	"ai-crm/services"
	"ai-crm/utils"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

type ManufacturingCustomerController struct {
	web.Controller
}

func (c *ManufacturingCustomerController) GetProfile() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:customer:view")
	if !ok {
		return
	}
	customerID, valid := pathUint(c.Ctx, ":customer_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的客户ID")
		return
	}
	profile, exists, err := services.GetMfgCustomerProfile(auth.TenantID, customerID)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"profile": profile, "exists": exists})
}

func (c *ManufacturingCustomerController) SaveProfile() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:customer:manage")
	if !ok {
		return
	}
	customerID, valid := pathUint(c.Ctx, ":customer_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的客户ID")
		return
	}
	var req services.MfgCustomerProfileInput
	if !bindJSON(c.Ctx, &req) {
		return
	}
	profile, err := services.SaveMfgCustomerProfile(auth.TenantID, auth.UserID, customerID, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SAVE_MFG_CUSTOMER_PROFILE", fmt.Sprintf("客户[%d]", customerID), "operation.save_mfg_customer_profile", map[string]interface{}{"customer_id": customerID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"profile": profile})
}

func (c *ManufacturingCustomerController) ListDecisionChain() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:customer:view")
	if !ok {
		return
	}
	customerID, valid := pathUint(c.Ctx, ":customer_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的客户ID")
		return
	}
	items, err := services.ListMfgDecisionChain(auth.TenantID, customerID)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "total": len(items)})
}

func (c *ManufacturingCustomerController) CreateDecisionChain() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:customer:manage")
	if !ok {
		return
	}
	customerID, valid := pathUint(c.Ctx, ":customer_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的客户ID")
		return
	}
	var req services.MfgDecisionChainInput
	if !bindJSON(c.Ctx, &req) {
		return
	}
	item, err := services.CreateMfgDecisionChain(auth.TenantID, auth.UserID, customerID, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_MFG_DECISION_CHAIN", fmt.Sprintf("客户[%d]", customerID), "operation.create_mfg_decision_chain", map[string]interface{}{"customer_id": customerID, "contact_id": item.ContactID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingCustomerController) UpdateDecisionChain() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:customer:manage")
	if !ok {
		return
	}
	customerID, valid := pathUint(c.Ctx, ":customer_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的客户ID")
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的决策链ID")
		return
	}
	var req services.MfgDecisionChainInput
	if !bindJSON(c.Ctx, &req) {
		return
	}
	item, err := services.UpdateMfgDecisionChain(auth.TenantID, auth.UserID, customerID, id, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "UPDATE_MFG_DECISION_CHAIN", fmt.Sprintf("决策链[%d]", id), "operation.update_mfg_decision_chain", map[string]interface{}{"customer_id": customerID, "decision_chain_id": id}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"item": item})
}

func (c *ManufacturingCustomerController) DeleteDecisionChain() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:customer:manage")
	if !ok {
		return
	}
	customerID, valid := pathUint(c.Ctx, ":customer_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的客户ID")
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的决策链ID")
		return
	}
	if err := services.DeleteMfgDecisionChain(auth.TenantID, customerID, id); err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "DELETE_MFG_DECISION_CHAIN", fmt.Sprintf("决策链[%d]", id), "operation.delete_mfg_decision_chain", map[string]interface{}{"customer_id": customerID, "decision_chain_id": id}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"deleted_id": id})
}

func (c *ManufacturingCustomerController) SaveMetricSnapshot() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:metrics:view")
	if !ok {
		return
	}
	customerID, valid := pathUint(c.Ctx, ":customer_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的客户ID")
		return
	}
	var req services.MetricCustomerSnapshotInput
	if !bindJSON(c.Ctx, &req) {
		return
	}
	snapshot, err := services.SaveMetricCustomerSnapshot(auth.TenantID, auth.UserID, customerID, req)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SAVE_MFG_CUSTOMER_METRIC_SNAPSHOT", fmt.Sprintf("客户[%d]", customerID), "operation.save_mfg_customer_metric_snapshot", map[string]interface{}{"customer_id": customerID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"snapshot": snapshot})
}

func (c *ManufacturingCustomerController) GetMetrics() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:metrics:view")
	if !ok {
		return
	}
	customerID, valid := pathUint(c.Ctx, ":customer_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的客户ID")
		return
	}
	snapshot, exists, err := services.LatestMetricCustomerSnapshot(auth.TenantID, customerID)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, map[string]interface{}{"snapshot": snapshot, "exists": exists})
}

func (c *ManufacturingCustomerController) Customer360() {
	auth, ok := requireManufacturingPermission(c.Ctx, "mfg:customer:view")
	if !ok {
		return
	}
	customerID, valid := pathUint(c.Ctx, ":customer_id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的客户ID")
		return
	}
	payload, err := services.MfgCustomer360(auth.TenantID, customerID)
	if err != nil {
		writeManufacturingError(c.Ctx, err)
		return
	}
	utils.Success(c.Ctx, payload)
}

func requireManufacturingPermission(ctx *context.Context, permission string) (authContext, bool) {
	auth := currentAuth(ctx)
	if auth.TenantID == 0 {
		utils.Error(ctx, http.StatusForbidden, "缺少租户上下文")
		return auth, false
	}
	if isManufacturingReadPermission(permission) {
		if !services.IsIndustryPackageEnabled(auth.TenantID, services.IndustryPackageManufacturing) {
			writeIndustryPackageError(ctx, errors.New(services.IndustryPackageErrorDisabled))
			return auth, false
		}
		return auth, true
	}
	auth, ok := requirePermission(ctx, permission)
	if !ok {
		return auth, false
	}
	if !services.IsIndustryPackageEnabled(auth.TenantID, services.IndustryPackageManufacturing) {
		writeIndustryPackageError(ctx, errors.New(services.IndustryPackageErrorDisabled))
		return auth, false
	}
	return auth, true
}

func isManufacturingReadPermission(permission string) bool {
	permission = strings.TrimSpace(permission)
	if permission == "" {
		return false
	}
	if permission == "mfg:view" {
		return true
	}
	return strings.HasPrefix(permission, "mfg:") && strings.HasSuffix(permission, ":view")
}

func writeManufacturingError(ctx *context.Context, err error) {
	message := err.Error()
	switch message {
	case services.MfgErrorCustomerNotFound:
		utils.ErrorWithData(ctx, http.StatusNotFound, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorContactNotFound:
		utils.ErrorWithData(ctx, http.StatusNotFound, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorInvalidInput:
		utils.ErrorWithData(ctx, http.StatusBadRequest, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorNotFound:
		utils.ErrorWithData(ctx, http.StatusNotFound, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorUnsupportedResource:
		utils.ErrorWithData(ctx, http.StatusBadRequest, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorRFQMissingRequiredFields:
		utils.ErrorWithData(ctx, http.StatusBadRequest, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorBOMSnapshotRequired:
		utils.ErrorWithData(ctx, http.StatusBadRequest, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorLowMarginApprovalRequired:
		utils.ErrorWithData(ctx, http.StatusBadRequest, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorOrderBOMLocked:
		utils.ErrorWithData(ctx, http.StatusConflict, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorECNRequired:
		utils.ErrorWithData(ctx, http.StatusConflict, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorDuplicateSerialNo:
		utils.ErrorWithData(ctx, http.StatusConflict, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorIoTUnauthorized:
		utils.ErrorWithData(ctx, http.StatusForbidden, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorDrawingApprovalRequired:
		utils.ErrorWithData(ctx, http.StatusForbidden, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorClarificationAnswerLocked:
		utils.ErrorWithData(ctx, http.StatusBadRequest, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorApprovalRemarkRequired:
		utils.ErrorWithData(ctx, http.StatusBadRequest, message, map[string]interface{}{"error_code": message})
	case services.MfgErrorInvalidState:
		utils.ErrorWithData(ctx, http.StatusBadRequest, message, map[string]interface{}{"error_code": message})
	case services.IndustryPackageErrorDisabled:
		writeIndustryPackageError(ctx, err)
	default:
		utils.Error(ctx, http.StatusInternalServerError, "制造行业包操作失败")
	}
}
