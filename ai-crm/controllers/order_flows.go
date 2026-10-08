package controllers

import (
	"ai-crm/models"
	"ai-crm/services"
	"ai-crm/utils"
	"archive/zip"
	"bytes"
	"fmt"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strings"
	"time"
)

func (c *CRMController) FulfillOrder() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的订单ID")
		return
	}
	var order models.CRMObject
	if err := crmScopedObjectQuery(auth, "orders").Where("id = ?", id).First(&order).Error; err != nil {
		utils.NotFound(c.Ctx, "订单不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, order, "订单"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var project models.CRMObject
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := reserveOrderInventory(tx, auth, order); err != nil {
			return err
		}
		order.Status = "confirmed"
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		project = cloneCRMObject(auth, "projects", "project", order.Name+" 交付项目", order)
		project.Status = "planning"
		project.DealID = order.DealID
		if err := tx.Create(&project).Error; err != nil {
			return err
		}
		relation := newCRMRelation(auth, "orders", order.ID, "projects", project.ID, "generated_project")
		return tx.Create(&relation).Error
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "订单履约失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "FULFILL_ORDER", fmt.Sprintf("订单[%d]", order.ID), "operation.fulfill_order", map[string]interface{}{"order_id": order.ID, "project_id": project.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"order": order, "project": project})
}

func (c *CRMController) CreateOrderFromDeal() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的成交ID")
		return
	}
	var deal models.CRMObject
	if err := crmScopedObjectQuery(auth, "deals").Where("id = ?", id).First(&deal).Error; err != nil {
		utils.NotFound(c.Ctx, "成交记录不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, deal, "成交记录"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var order models.CRMObject
	reused := false
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if existingOrder, hasOrder, err := findExistingOrderForDealTx(tx, auth, deal.ID); err != nil {
			return err
		} else if hasOrder {
			order = existingOrder
			reused = true
			order.DealID = deal.ID
			if err := normalizeCustomerChainObjectTx(tx, auth, &order, deal); err != nil {
				return err
			}
			order.UpdatedBy = auth.UserID
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
			if err := ensureCRMRelationTx(tx, auth, "deals", deal.ID, "orders", order.ID, "generated_order"); err != nil {
				return err
			}
			if err := ensureCustomerChainRelationsTx(tx, auth, order); err != nil {
				return err
			}
			return updateDealGeneratedOrderTx(tx, auth, deal, order)
		}
		var err error
		order, err = createDraftOrderForDealTx(tx, auth, deal, deal, nil)
		return err
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "成交生成订单失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_ORDER_FROM_DEAL", fmt.Sprintf("成交[%d]", deal.ID), "operation.create_order_from_deal", map[string]interface{}{"deal_id": deal.ID, "order_id": order.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"deal": deal, "order": order, "reused": reused})
}

func (c *CRMController) MarkOrderPaid() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的订单ID")
		return
	}
	var req struct {
		PaymentNo     string  `json:"payment_no"`
		PaymentMethod string  `json:"payment_method"`
		PaidAmount    float64 `json:"paid_amount"`
		Reason        string  `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var order models.CRMObject
	if err := crmScopedObjectQuery(auth, "orders").Where("id = ?", id).First(&order).Error; err != nil {
		utils.NotFound(c.Ctx, "订单不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, order, "订单"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	if err := ensureOrderNotCompleted(order, "登记回款"); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	payload := payloadMap(order.PayloadJSON)
	if req.PaidAmount <= 0 {
		utils.BadRequest(c.Ctx, "回款金额必须大于0")
		return
	}
	receivable := currentReceivable(order, payload)
	if receivable <= 0 {
		utils.BadRequest(c.Ctx, "订单应收余额已为0，不能继续登记回款")
		return
	}
	if req.PaidAmount > receivable {
		utils.BadRequest(c.Ctx, "回款金额不能超过应收余额")
		return
	}
	payments := normalizeOrderPayments(payload["payments"])
	payment := map[string]interface{}{
		"payment_no":     strings.TrimSpace(req.PaymentNo),
		"payment_method": strings.TrimSpace(req.PaymentMethod),
		"paid_amount":    req.PaidAmount,
		"paid_at":        time.Now().Format(time.RFC3339),
		"reason":         defaultText(req.Reason, "登记回款"),
		"operator_id":    auth.UserID,
	}
	if payment["payment_no"] == "" {
		payment["payment_no"] = fmt.Sprintf("PAY-%d", time.Now().UnixNano()%1000000000)
	}
	payments = append(payments, payment)
	payload["payments"] = payments
	payload["payment_no"] = req.PaymentNo
	payload["payment_method"] = req.PaymentMethod
	payload["paid_amount"] = numericValue(payload["paid_amount"]) + req.PaidAmount
	payload["paid_at"] = time.Now().Format(time.RFC3339)
	recalculateOrderFinancials(&order, payload)
	if numericValue(payload["receivable_balance"]) > 0 {
		payload["payment_status"] = "partial_paid"
	} else {
		payload["payment_status"] = "paid"
	}
	order.PayloadJSON = toJSONString(payload)
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "orders", order.ID, "order_payment", "登记回款", map[string]interface{}{"reason": defaultText(req.Reason, "登记回款"), "payment_no": req.PaymentNo, "paid_amount": req.PaidAmount})
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "订单支付状态更新失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "MARK_ORDER_PAID", fmt.Sprintf("订单[%d]", order.ID), "operation.mark_order_paid", map[string]interface{}{"order_id": order.ID, "payment_no": req.PaymentNo}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"order": order})
}

func normalizeOrderPayments(raw interface{}) []map[string]interface{} {
	items := []map[string]interface{}{}
	for _, item := range interfaceList(raw) {
		entry, ok := item.(map[string]interface{})
		if !ok || len(entry) == 0 {
			continue
		}
		items = append(items, entry)
	}
	return items
}

func (c *CRMController) SaveOrderPaymentPlan() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的订单ID")
		return
	}
	var req struct {
		PaymentPlan []map[string]interface{} `json:"payment_plan"`
		Reason      string                   `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if len(req.PaymentPlan) == 0 {
		utils.BadRequest(c.Ctx, "付款计划不能为空")
		return
	}
	var order models.CRMObject
	if err := crmScopedObjectQuery(auth, "orders").Where("id = ?", id).First(&order).Error; err != nil {
		utils.NotFound(c.Ctx, "订单不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, order, "订单"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	if err := ensureOrderNotCompleted(order, "修改付款计划"); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	plan := normalizePaymentPlanWithCurrency(req.PaymentPlan, order.Currency)
	if err := validateOrderPaymentPlan(plan); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	payload := payloadMap(order.PayloadJSON)
	payload["payment_plan"] = plan
	recalculateOrderFinancials(&order, payload)
	order.PayloadJSON = toJSONString(payload)
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "orders", order.ID, "order_payment_plan", "更新付款计划", map[string]interface{}{"reason": defaultText(req.Reason, "更新付款计划"), "payment_plan": payload["payment_plan"], "overdue_amount": payload["overdue_amount"]})
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "保存付款计划失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SAVE_ORDER_PAYMENT_PLAN", fmt.Sprintf("订单[%d]", order.ID), "operation.save_order_payment_plan", map[string]interface{}{"order_id": order.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"order": order})
}

func (c *CRMController) ShipOrder() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的订单ID")
		return
	}
	var req struct {
		ShipmentNo    string `json:"shipment_no"`
		Carrier       string `json:"carrier"`
		ShipmentScope string `json:"shipment_scope"`
		DeliveryNote  string `json:"delivery_note"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	var order models.CRMObject
	if err := crmScopedObjectQuery(auth, "orders").Where("id = ?", id).First(&order).Error; err != nil {
		utils.NotFound(c.Ctx, "订单不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, order, "订单"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	if err := ensureOrderNotCompleted(order, "登记发货"); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	payload := payloadMap(order.PayloadJSON)
	req.ShipmentNo = strings.TrimSpace(req.ShipmentNo)
	req.Carrier = strings.TrimSpace(req.Carrier)
	req.ShipmentScope = strings.TrimSpace(req.ShipmentScope)
	req.DeliveryNote = strings.TrimSpace(req.DeliveryNote)
	if req.ShipmentScope != "partial" {
		req.ShipmentScope = "full"
	}
	if req.ShipmentNo == "" {
		req.ShipmentNo = fmt.Sprintf("SHP-%d", time.Now().UnixNano()%1000000000)
	}
	shippedAt := time.Now().Format(time.RFC3339)
	targetStatus := "shipped"
	if req.ShipmentScope == "partial" {
		targetStatus = "partial_shipped"
	}
	if err := validateOrderReadyForStatus(order, targetStatus); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	if order.Status != targetStatus && !canOrderStatusTransition(order.Status, targetStatus) {
		utils.BadRequest(c.Ctx, fmt.Sprintf("不允许从 %s 变更到 %s", order.Status, targetStatus))
		return
	}
	shipment := map[string]interface{}{
		"shipment_no":    req.ShipmentNo,
		"carrier":        req.Carrier,
		"shipment_scope": req.ShipmentScope,
		"delivery_note":  req.DeliveryNote,
		"shipped_at":     shippedAt,
		"operator_id":    auth.UserID,
	}
	payload["shipment_no"] = req.ShipmentNo
	payload["carrier"] = req.Carrier
	payload["shipment_scope"] = req.ShipmentScope
	payload["delivery_note"] = req.DeliveryNote
	payload["shipped_at"] = shippedAt
	payload["shipments"] = append(normalizeOrderShipments(payload["shipments"]), shipment)
	order.Status = targetStatus
	order.PayloadJSON = toJSONString(payload)
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		summary := "订单发货"
		if req.ShipmentScope == "partial" {
			summary = "订单部分发货"
		}
		return createBusinessTimeline(tx, auth, "orders", order.ID, "order_ship", summary, map[string]interface{}{"reason": defaultText(req.DeliveryNote, summary), "shipment_no": req.ShipmentNo, "carrier": req.Carrier, "shipment_scope": req.ShipmentScope})
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "订单发货失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "SHIP_ORDER", fmt.Sprintf("订单[%d]", order.ID), "operation.ship_order", map[string]interface{}{"order_id": order.ID, "shipment_no": req.ShipmentNo}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"order": order})
}

func normalizeOrderShipments(raw interface{}) []interface{} {
	items := []interface{}{}
	for _, item := range interfaceList(raw) {
		entry, ok := item.(map[string]interface{})
		if !ok || len(entry) == 0 {
			continue
		}
		items = append(items, entry)
	}
	return items
}

func (c *CRMController) InvoiceOrder() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的订单ID")
		return
	}
	var req struct {
		InvoiceNo              string `json:"invoice_no"`
		InvoiceType            string `json:"invoice_type"`
		TaxNo                  string `json:"tax_no"`
		SentTo                 string `json:"sent_to"`
		InvoiceExemptionReason string `json:"invoice_exemption_reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	req.InvoiceNo = strings.TrimSpace(req.InvoiceNo)
	req.InvoiceExemptionReason = strings.TrimSpace(req.InvoiceExemptionReason)
	if req.InvoiceNo == "" && req.InvoiceExemptionReason == "" {
		utils.BadRequest(c.Ctx, "发票号和免票原因至少填写一项")
		return
	}
	var order models.CRMObject
	if err := crmScopedObjectQuery(auth, "orders").Where("id = ?", id).First(&order).Error; err != nil {
		utils.NotFound(c.Ctx, "订单不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, order, "订单"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	if err := ensureOrderNotCompleted(order, "修改发票或免票记录"); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	payload := payloadMap(order.PayloadJSON)
	invoice := map[string]interface{}{
		"invoice_no":               req.InvoiceNo,
		"invoice_type":             req.InvoiceType,
		"tax_no":                   req.TaxNo,
		"sent_to":                  req.SentTo,
		"invoice_exemption_reason": req.InvoiceExemptionReason,
		"invoiced_at":              time.Now().Format(time.RFC3339),
	}
	if req.InvoiceNo != "" {
		payload["invoice_no"] = req.InvoiceNo
		payload["invoice_type"] = req.InvoiceType
	} else {
		delete(payload, "invoice_no")
		delete(payload, "invoice_type")
	}
	if req.InvoiceExemptionReason != "" {
		payload["invoice_exemption_reason"] = req.InvoiceExemptionReason
		payload["invoice_exempted_at"] = time.Now().Format(time.RFC3339)
	}
	payload["invoice"] = invoice
	payload["locked_fields"] = []string{"line_items", "amount", "currency"}
	order.PayloadJSON = toJSONString(payload)
	invoiceName := "免票记录"
	if req.InvoiceNo != "" {
		invoiceName = "发票：" + req.InvoiceNo
	}
	invoiceObject := cloneCRMObject(auth, "orders", "invoice", invoiceName, order)
	invoiceObject.Status = "active"
	invoiceObject.PayloadJSON = toJSONString(invoice)
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		if err := tx.Create(&invoiceObject).Error; err != nil {
			return err
		}
		if err := tx.Create(&[]models.CRMRelation{
			newCRMRelation(auth, "orders", order.ID, "orders", invoiceObject.ID, "generated_invoice"),
		}).Error; err != nil {
			return err
		}
		summary := "订单开票"
		if req.InvoiceNo == "" {
			summary = "登记免票原因"
		}
		return createBusinessTimeline(tx, auth, "orders", order.ID, "order_invoice", summary, map[string]interface{}{"reason": defaultText(req.InvoiceExemptionReason, summary), "invoice_no": req.InvoiceNo, "invoice_type": req.InvoiceType, "invoice_exemption_reason": req.InvoiceExemptionReason})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "订单开票失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "INVOICE_ORDER", fmt.Sprintf("订单[%d]", order.ID), "operation.invoice_order", map[string]interface{}{"order_id": order.ID, "invoice_id": invoiceObject.ID, "invoice_no": req.InvoiceNo}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"order": order, "invoice": invoiceObject})
}

func (c *CRMController) BackupOrder() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的订单ID")
		return
	}
	if err := ensureCRMModuleAccess(auth, "orders", "read"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var order models.CRMObject
	if err := crmScopedObjectQuery(auth, "orders").Where("id = ? AND object_type = ?", id, "order").First(&order).Error; err != nil {
		utils.NotFound(c.Ctx, "订单不存在")
		return
	}
	if !canViewCRMObject(auth, order) {
		utils.Error(c.Ctx, http.StatusForbidden, "无权导出该订单")
		return
	}
	exportCtx := newExportSecurityContext(c.Ctx, auth)
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	missingFiles := []map[string]interface{}{}
	manifest := map[string]interface{}{
		"order_id":      order.ID,
		"order_name":    order.Name,
		"generated_at":  time.Now().Format(time.RFC3339),
		"generated_by":  auth.UserID,
		"trace_id":      exportCtx.TraceID,
		"watermark":     exportCtx.Params(),
		"missing_files": missingFiles,
	}
	orderPayload := payloadMap(order.PayloadJSON)
	if err := writeExcelHTMLZipFile(zipWriter, "watermark.xls", exportCtx.WatermarkTable()); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "写入导出水印失败")
		return
	}
	if err := writeOrderBackupOrderXLS(zipWriter, "order.xls", order, orderPayload); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "写入订单备份失败")
		return
	}
	if err := writeOrderLineItemsXLS(zipWriter, "line_items.xls", interfaceList(orderPayload["line_items"])); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "写入订单明细备份失败")
		return
	}
	if err := writeOrderPaymentsXLS(zipWriter, "payments.xls", orderPaymentBackup(order, orderPayload)); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "写入回款备份失败")
		return
	}
	if err := writeOrderInvoiceXLS(zipWriter, "invoices.xls", orderInvoiceBackup(auth, order, orderPayload)); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "写入发票备份失败")
		return
	}
	if err := writeOrderShipmentXLS(zipWriter, "shipments.xls", orderShipmentBackup(order, orderPayload)); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "写入发货备份失败")
		return
	}
	timeline := orderTimelineObjects(auth, order.ID)
	if err := writeOrderTimelineXLS(zipWriter, "timeline.xls", timeline); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "写入时间线备份失败")
		return
	}
	relations := orderRelationBackup(auth, order.ID)
	if err := writeOrderRelationsXLS(zipWriter, "relations.xls", relations); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "写入关联备份失败")
		return
	}
	files := orderFileObjects(auth, order.ID)
	fileItems := make([]map[string]interface{}, 0, len(files))
	for _, fileObject := range files {
		filePayload := payloadMap(fileObject.PayloadJSON)
		fileItem := map[string]interface{}{"id": fileObject.ID, "name": fileObject.Name, "payload": filePayload}
		fileItems = append(fileItems, fileItem)
		fileID := strings.TrimSpace(fmt.Sprint(filePayload["file_id"]))
		if fileID == "" {
			missingFiles = append(missingFiles, map[string]interface{}{"file_object_id": fileObject.ID, "file_name": fileObject.Name, "reason": "缺少 file_id"})
			continue
		}
		if err := addOrderBackupRemoteFile(zipWriter, auth, fileObject, filePayload); err != nil {
			missingFiles = append(missingFiles, map[string]interface{}{"file_object_id": fileObject.ID, "file_id": fileID, "file_name": fileObject.Name, "reason": err.Error()})
		}
	}
	manifest["files"] = fileItems
	manifest["missing_files"] = missingFiles
	if err := writeOrderManifestXLS(zipWriter, "manifest.xls", manifest); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "写入备份清单失败")
		return
	}
	if err := zipWriter.Close(); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "生成订单备份失败")
		return
	}
	filename := fmt.Sprintf("order-%d-backup-%s.zip", order.ID, exportCtx.TraceID)
	logControllerOperation(c.Ctx, auth, auth.TenantID, "EXPORT_ORDER_BACKUP", fmt.Sprintf("订单[%d]", order.ID), "operation.export_order_backup", mergeExportParams(exportCtx, map[string]interface{}{"order_id": order.ID, "missing_files": len(missingFiles), "file_count": len(fileItems)}), "success", "")
	c.Ctx.Output.Header("Content-Type", "application/zip")
	applyExportTraceHeader(c.Ctx, exportCtx)
	c.Ctx.Output.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	_, _ = c.Ctx.ResponseWriter.Write(buffer.Bytes())
}

func orderPaymentBackup(order models.CRMObject, payload map[string]interface{}) map[string]interface{} {
	payments := normalizeOrderPayments(payload["payments"])
	if len(payments) == 0 && numericValue(payload["paid_amount"]) > 0 {
		payments = append(payments, map[string]interface{}{
			"payment_no":     payload["payment_no"],
			"payment_method": payload["payment_method"],
			"paid_amount":    payload["paid_amount"],
			"paid_at":        payload["paid_at"],
		})
	}
	return map[string]interface{}{
		"order_id":           order.ID,
		"currency":           order.Currency,
		"paid_amount":        payload["paid_amount"],
		"receivable_balance": payload["receivable_balance"],
		"payment_status":     payload["payment_status"],
		"payment_plan":       payload["payment_plan"],
		"overdue_amount":     payload["overdue_amount"],
		"items":              payments,
	}
}

func writeOrderBackupOrderXLS(zipWriter *zip.Writer, name string, order models.CRMObject, payload map[string]interface{}) error {
	return writeExcelHTMLZipFile(zipWriter, name,
		excelTable{
			Title:   "订单基础信息",
			Headers: []string{"字段", "值"},
			Rows: [][]interface{}{
				{"订单ID", order.ID},
				{"订单名称", order.Name},
				{"订单编码", order.Code},
				{"状态", order.Status},
				{"金额", order.Amount},
				{"币种", order.Currency},
				{"负责人ID", order.OwnerID},
				{"公司ID", order.CompanyID},
				{"联系人ID", order.ContactID},
				{"成交ID", order.DealID},
				{"创建时间", order.CreatedAt},
				{"更新时间", order.UpdatedAt},
			},
		},
		excelTable{Title: "订单扩展字段", Headers: []string{"字段", "值"}, Rows: flattenMapRows("", payload)},
	)
}

func writeOrderLineItemsXLS(zipWriter *zip.Writer, name string, items []interface{}) error {
	rows := [][]interface{}{}
	for index, item := range items {
		row, _ := item.(map[string]interface{})
		rows = append(rows, []interface{}{
			index + 1,
			row["product_id"],
			firstOpportunityReviewText(fmt.Sprint(row["product_name"]), fmt.Sprint(row["name"])),
			row["sku"],
			row["quantity"],
			firstOpportunityReviewText(fmt.Sprint(row["unit_price"]), fmt.Sprint(row["price"])),
			row["discount"],
			row["tax_rate"],
		})
	}
	return writeExcelHTMLZipFile(zipWriter, name, excelTable{Title: "订单明细", Headers: []string{"序号", "产品ID", "产品名称", "SKU", "数量", "单价", "折扣", "税率"}, Rows: rows})
}

func writeOrderPaymentsXLS(zipWriter *zip.Writer, name string, payment map[string]interface{}) error {
	rows := [][]interface{}{}
	for index, item := range interfaceList(payment["items"]) {
		row, _ := item.(map[string]interface{})
		rows = append(rows, []interface{}{index + 1, row["payment_no"], row["payment_method"], row["paid_amount"], row["paid_at"]})
	}
	return writeExcelHTMLZipFile(zipWriter, name,
		excelTable{Title: "回款汇总", Headers: []string{"字段", "值"}, Rows: [][]interface{}{
			{"订单ID", payment["order_id"]},
			{"币种", payment["currency"]},
			{"已回款金额", payment["paid_amount"]},
			{"应收余额", payment["receivable_balance"]},
			{"回款状态", payment["payment_status"]},
			{"逾期金额", payment["overdue_amount"]},
		}},
		excelTable{Title: "回款明细", Headers: []string{"序号", "回款编号", "回款方式", "回款金额", "回款时间"}, Rows: rows},
	)
}

func writeOrderInvoiceXLS(zipWriter *zip.Writer, name string, invoice map[string]interface{}) error {
	generatedRows := [][]interface{}{}
	for _, item := range interfaceList(invoice["generated_invoice_objects"]) {
		if object, ok := item.(models.CRMObject); ok {
			generatedRows = append(generatedRows, []interface{}{object.ID, object.Name, object.Code, object.Status, object.Amount, object.Currency, object.CreatedAt})
		}
	}
	return writeExcelHTMLZipFile(zipWriter, name,
		excelTable{Title: "发票信息", Headers: []string{"字段", "值"}, Rows: [][]interface{}{
			{"订单ID", invoice["order_id"]},
			{"发票号", invoice["invoice_no"]},
			{"发票类型", invoice["invoice_type"]},
			{"免开原因", invoice["invoice_exemption_reason"]},
			{"免开时间", invoice["invoice_exempted_at"]},
		}},
		excelTable{Title: "发票对象", Headers: []string{"ID", "名称", "编码", "状态", "金额", "币种", "创建时间"}, Rows: generatedRows},
	)
}

func writeOrderShipmentXLS(zipWriter *zip.Writer, name string, shipment map[string]interface{}) error {
	return writeExcelHTMLZipFile(zipWriter, name, excelTable{Title: "发货信息", Headers: []string{"字段", "值"}, Rows: [][]interface{}{
		{"订单ID", shipment["order_id"]},
		{"订单状态", shipment["status"]},
		{"发货单号", shipment["shipment_no"]},
		{"承运方", shipment["carrier"]},
		{"发货时间", shipment["shipped_at"]},
		{"发货范围", shipment["shipment_scope"]},
		{"发货备注", shipment["delivery_note"]},
		{"交付地址", shipment["delivery_address"]},
		{"收货地址", shipment["shipping_address"]},
	}})
}

func writeOrderTimelineXLS(zipWriter *zip.Writer, name string, items []models.CRMObject) error {
	rows := [][]interface{}{}
	for _, item := range items {
		rows = append(rows, []interface{}{item.ID, item.Name, item.ObjectType, item.Status, item.OwnerID, item.CreatedAt, firstPayloadText(payloadMap(item.PayloadJSON), "content", "summary", "note")})
	}
	return writeExcelHTMLZipFile(zipWriter, name, excelTable{Title: "订单时间线", Headers: []string{"ID", "名称", "类型", "状态", "负责人ID", "创建时间", "内容"}, Rows: rows})
}

func writeOrderRelationsXLS(zipWriter *zip.Writer, name string, data map[string]interface{}) error {
	rows := [][]interface{}{}
	for _, item := range interfaceList(data["items"]) {
		row, _ := item.(map[string]interface{})
		relation, _ := row["relation"].(models.CRMRelation)
		sourceName := ""
		targetName := ""
		if source, ok := row["source"].(models.CRMObject); ok {
			sourceName = source.Name
		}
		if target, ok := row["target"].(models.CRMObject); ok {
			targetName = target.Name
		}
		rows = append(rows, []interface{}{relation.ID, relation.SourceType, relation.SourceID, sourceName, relation.RelationType, relation.TargetType, relation.TargetID, targetName, relation.CreatedAt})
	}
	return writeExcelHTMLZipFile(zipWriter, name, excelTable{Title: "订单关联关系", Headers: []string{"关系ID", "来源类型", "来源ID", "来源名称", "关系类型", "目标类型", "目标ID", "目标名称", "创建时间"}, Rows: rows})
}

func writeOrderManifestXLS(zipWriter *zip.Writer, name string, manifest map[string]interface{}) error {
	fileRows := [][]interface{}{}
	for _, item := range interfaceList(manifest["files"]) {
		row, _ := item.(map[string]interface{})
		fileRows = append(fileRows, []interface{}{row["id"], row["name"], fmt.Sprint(row["payload"])})
	}
	missingRows := [][]interface{}{}
	for _, item := range interfaceList(manifest["missing_files"]) {
		row, _ := item.(map[string]interface{})
		missingRows = append(missingRows, []interface{}{row["file_object_id"], row["file_id"], row["file_name"], row["reason"]})
	}
	return writeExcelHTMLZipFile(zipWriter, name,
		excelTable{Title: "备份清单", Headers: []string{"字段", "值"}, Rows: [][]interface{}{
			{"订单ID", manifest["order_id"]},
			{"订单名称", manifest["order_name"]},
			{"生成时间", manifest["generated_at"]},
			{"生成人ID", manifest["generated_by"]},
		}},
		excelTable{Title: "附件清单", Headers: []string{"文件对象ID", "文件名称", "文件信息"}, Rows: fileRows},
		excelTable{Title: "缺失附件", Headers: []string{"文件对象ID", "文件ID", "文件名称", "原因"}, Rows: missingRows},
	)
}

func orderInvoiceBackup(auth authContext, order models.CRMObject, payload map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"order_id":                  order.ID,
		"invoice_no":                payload["invoice_no"],
		"invoice_type":              payload["invoice_type"],
		"invoice_exemption_reason":  payload["invoice_exemption_reason"],
		"invoice_exempted_at":       payload["invoice_exempted_at"],
		"current":                   payload["invoice"],
		"generated_invoice_objects": orderRelatedObjects(auth, order.ID, "generated_invoice", "orders", "invoice"),
	}
}

func orderShipmentBackup(order models.CRMObject, payload map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"order_id":         order.ID,
		"status":           order.Status,
		"shipment_no":      payload["shipment_no"],
		"carrier":          payload["carrier"],
		"shipped_at":       payload["shipped_at"],
		"shipment_scope":   payload["shipment_scope"],
		"delivery_note":    payload["delivery_note"],
		"delivery_address": payload["delivery_address"],
		"shipping_address": payload["shipping_address"],
	}
}

func orderTimelineObjects(auth authContext, orderID uint) []models.CRMObject {
	var relations []models.CRMRelation
	_ = utils.DB.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND relation_type = ?", auth.TenantID, "orders", orderID, "timeline").Find(&relations).Error
	ids := []uint{}
	for _, relation := range relations {
		if relation.TargetType == "contacts" {
			ids = append(ids, relation.TargetID)
		}
	}
	if len(ids) == 0 {
		return []models.CRMObject{}
	}
	var items []models.CRMObject
	_ = crmScopedObjectQuery(auth, "contacts").Where("crm_objects.id IN ?", ids).Order("created_at asc").Find(&items).Error
	return items
}

func orderRelationBackup(auth authContext, orderID uint) map[string]interface{} {
	var relations []models.CRMRelation
	_ = utils.DB.Where("tenant_id = ? AND ((source_type = ? AND source_id = ?) OR (target_type = ? AND target_id = ?))", auth.TenantID, "orders", orderID, "orders", orderID).Order("id asc").Find(&relations).Error
	summaries := []map[string]interface{}{}
	for _, relation := range relations {
		item := map[string]interface{}{"relation": relation}
		if relation.SourceType == "orders" && relation.SourceID == orderID {
			if object, ok := loadBackupObjectSummary(auth, relation.TargetType, relation.TargetID); ok {
				item["target"] = object
			}
		} else if relation.TargetType == "orders" && relation.TargetID == orderID {
			if object, ok := loadBackupObjectSummary(auth, relation.SourceType, relation.SourceID); ok {
				item["source"] = object
			}
		}
		summaries = append(summaries, item)
	}
	return map[string]interface{}{"items": summaries, "total": len(summaries)}
}

func orderRelatedObjects(auth authContext, orderID uint, relationType string, targetDomain string, objectType string) []models.CRMObject {
	var relations []models.CRMRelation
	_ = utils.DB.Where("tenant_id = ? AND source_type = ? AND source_id = ? AND relation_type = ? AND target_type = ?", auth.TenantID, "orders", orderID, relationType, targetDomain).Find(&relations).Error
	ids := []uint{}
	for _, relation := range relations {
		ids = append(ids, relation.TargetID)
	}
	if len(ids) == 0 {
		return []models.CRMObject{}
	}
	var items []models.CRMObject
	query := crmScopedObjectQuery(auth, targetDomain).Where("crm_objects.id IN ?", ids)
	if objectType != "" {
		query = query.Where("crm_objects.object_type = ?", objectType)
	}
	_ = query.Order("created_at asc").Find(&items).Error
	return items
}

func loadBackupObjectSummary(auth authContext, domain string, id uint) (models.CRMObject, bool) {
	if !validCRMDomain(domain) {
		return models.CRMObject{}, false
	}
	var object models.CRMObject
	if err := crmScopedObjectQuery(auth, domain).Where("crm_objects.id = ?", id).First(&object).Error; err != nil {
		return models.CRMObject{}, false
	}
	return object, true
}

func addOrderBackupRemoteFile(zipWriter *zip.Writer, auth authContext, fileObject models.CRMObject, filePayload map[string]interface{}) error {
	fileID := strings.TrimSpace(fmt.Sprint(filePayload["file_id"]))
	storageNodeID := strings.TrimSpace(fmt.Sprint(filePayload["storage_node_id"]))
	path := fmt.Sprintf("/internal/v1/files/%s/download", fileID)
	for _, node := range orderedFileserverNodes(storageNodeID) {
		if !node.Healthy {
			continue
		}
		resp, err := services.GetInternal(node, path, "", auth.TenantID, "")
		if err != nil {
			continue
		}
		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return fmt.Errorf("文件服务返回 %d", resp.StatusCode)
		}
		defer resp.Body.Close()
		writer, err := zipWriter.Create("files/" + safeZipPathName(fileObject.Name, fileID))
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, resp.Body)
		return err
	}
	return fmt.Errorf("文件不存在或文件服务不可用")
}

func safeZipPathName(name string, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	name = replacer.Replace(name)
	if name == "" {
		return "file"
	}
	return name
}

func (c *CRMController) ReturnOrder() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的订单ID")
		return
	}
	var req struct {
		Reason string `json:"reason"`
		Type   string `json:"type"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	if strings.TrimSpace(req.Type) == "" {
		req.Type = "return"
	}
	var order models.CRMObject
	if err := crmScopedObjectQuery(auth, "orders").Where("id = ?", id).First(&order).Error; err != nil {
		utils.NotFound(c.Ctx, "订单不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, order, "订单"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	if err := ensureOrderNotCompleted(order, "发起售后退货"); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	payload := payloadMap(order.PayloadJSON)
	payload["after_sales_status"] = req.Type
	payload["return_reason"] = req.Reason
	payload["return_requested_at"] = time.Now().Format(time.RFC3339)
	order.Status = "canceled"
	order.PayloadJSON = toJSONString(payload)
	ticket := cloneCRMObject(auth, "tickets", "ticket", "订单售后："+order.Name, order)
	ticket.Status = "new"
	ticket.PayloadJSON = toJSONString(map[string]interface{}{"category": "after_sales", "priority": "normal", "order_id": order.ID, "reason": req.Reason, "type": req.Type})
	err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}
		if err := tx.Create(&[]models.CRMRelation{
			newCRMRelation(auth, "orders", order.ID, "tickets", ticket.ID, "return_ticket"),
		}).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "orders", order.ID, "order_after_sales", "订单售后", map[string]interface{}{"reason": req.Reason, "ticket_id": ticket.ID})
	})
	if err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "订单售后失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "RETURN_ORDER", fmt.Sprintf("订单[%d]", order.ID), "operation.return_order", map[string]interface{}{"order_id": order.ID, "ticket_id": ticket.ID, "type": req.Type}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"order": order, "ticket": ticket})
}

func (c *CRMController) ChangeOrderStatus() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的订单ID")
		return
	}
	var req struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if !bindJSON(c.Ctx, &req) {
		return
	}
	status := normalizeOrderStatus(req.Status)
	if status == "" {
		utils.BadRequest(c.Ctx, "订单状态无效")
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if status == "canceled" && reason == "" {
		utils.BadRequest(c.Ctx, "取消订单必须填写原因")
		return
	}
	if status == "canceled" && !authCanCancelOrder(auth) {
		utils.Error(c.Ctx, http.StatusForbidden, "只有租户管理员或订单管理权限用户才能取消订单")
		return
	}
	var order models.CRMObject
	if err := crmScopedObjectQuery(auth, "orders").Where("id = ?", id).First(&order).Error; err != nil {
		utils.NotFound(c.Ctx, "订单不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, order, "订单"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	if order.Status == "completed" || order.Status == "canceled" {
		utils.BadRequest(c.Ctx, "终态订单不能再变更状态")
		return
	}
	if !canOrderStatusTransition(order.Status, status) {
		utils.BadRequest(c.Ctx, fmt.Sprintf("不允许从 %s 变更到 %s", order.Status, status))
		return
	}
	if err := validateOrderReadyForStatus(order, status); err != nil {
		utils.BadRequest(c.Ctx, err.Error())
		return
	}
	before := order.Status
	payload := payloadMap(order.PayloadJSON)
	payload["status_reason"] = reason
	payload["status_changed_at"] = time.Now().Format(time.RFC3339)
	payload["status_changed_by"] = auth.UserID
	if status == "canceled" {
		payload["cancel_reason"] = reason
		payload["canceled_at"] = time.Now().Format(time.RFC3339)
	}
	order.Status = status
	order.PayloadJSON = toJSONString(payload)
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "orders", order.ID, "order_status", fmt.Sprintf("%s -> %s", before, status), map[string]interface{}{"reason": reason, "from": before, "to": status})
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "订单状态变更失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CHANGE_ORDER_STATUS", fmt.Sprintf("订单[%d]", order.ID), "operation.change_order_status", map[string]interface{}{"order_id": order.ID, "from": before, "to": status}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"order": order})
}

func (c *CRMController) CreateTicketFromOrder() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	id, valid := pathUint(c.Ctx, ":id")
	if !valid {
		utils.BadRequest(c.Ctx, "无效的订单ID")
		return
	}
	var order models.CRMObject
	if err := crmScopedObjectQuery(auth, "orders").Where("id = ?", id).First(&order).Error; err != nil {
		utils.NotFound(c.Ctx, "订单不存在")
		return
	}
	if err := ensureCRMObjectEditAccess(auth, order, "订单"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	ticket := cloneCRMObject(auth, "tickets", "ticket", order.Name+" 售后工单", order)
	ticket.Status = "new"
	ticket.PayloadJSON = toJSONString(mergePayload(payloadMap(order.PayloadJSON), map[string]interface{}{"category": "after_sales", "priority": "normal", "order_id": order.ID, "customer_grade": firstPayloadText(payloadMap(order.PayloadJSON), "customer_grade", "grade")}))
	if err := utils.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&ticket).Error; err != nil {
			return err
		}
		relation := newCRMRelation(auth, "orders", order.ID, "tickets", ticket.ID, "generated_ticket")
		if err := tx.Create(&relation).Error; err != nil {
			return err
		}
		return createBusinessTimeline(tx, auth, "orders", order.ID, "order_ticket", "订单创建售后工单", map[string]interface{}{"reason": "订单创建售后工单", "ticket_id": ticket.ID})
	}); err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "订单生成工单失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "CREATE_TICKET_FROM_ORDER", fmt.Sprintf("订单[%d]", order.ID), "operation.create_ticket_from_order", map[string]interface{}{"order_id": order.ID, "ticket_id": ticket.ID}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"order": order, "ticket": ticket})
}
