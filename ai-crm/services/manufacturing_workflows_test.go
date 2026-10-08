package services

import (
	"ai-crm/models"
	"ai-crm/utils"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRemovedMfgFileReferencesDedupesRemovedOnly(t *testing.T) {
	before := []MfgFileReference{
		{FileID: "drawing-1", StorageNodeID: "node-a"},
		{FileID: "drawing-1", StorageNodeID: "node-a"},
		{FileID: "drawing-2", StorageNodeID: "node-b"},
	}
	after := []MfgFileReference{
		{FileID: "drawing-2", StorageNodeID: "node-b"},
	}

	removed := RemovedMfgFileReferences(before, after)
	if len(removed) != 1 {
		t.Fatalf("expected one removed ref, got %#v", removed)
	}
	if removed[0].FileID != "drawing-1" || removed[0].StorageNodeID != "node-a" {
		t.Fatalf("unexpected removed ref: %#v", removed[0])
	}
}

func TestMfgRFQFileReferencesIncludeDrawingForPhysicalCleanup(t *testing.T) {
	refs := MfgRecordFileReferences("rfqs", models.MfgRFQ{
		DrawingFileID:        "drawing-file-1",
		DrawingStorageNodeID: "node-a",
	})
	if len(refs) != 1 {
		t.Fatalf("expected one RFQ drawing ref, got %#v", refs)
	}
	if refs[0].FileID != "drawing-file-1" || refs[0].StorageNodeID != "node-a" || refs[0].Source != "rfqs" {
		t.Fatalf("unexpected RFQ drawing ref: %#v", refs[0])
	}
}

func TestSanitizeMfgRFQIncludesRowCapabilities(t *testing.T) {
	rfq := models.MfgRFQ{
		TenantID:    1,
		RFQNo:       "RFQ-CAPS-1",
		CreatedBy:   11,
		TargetPrice: 1200,
	}

	creatorPayload := sanitizeMfgRFQ(rfq, MfgAccessContext{TenantID: 1, UserID: 11})
	if creatorPayload["target_price"] == mfgMaskedPrice {
		t.Fatalf("RFQ creator should see own target price")
	}
	if creatorPayload["can_edit"] != true || creatorPayload["can_delete"] != true || creatorPayload["can_submit"] != true {
		t.Fatalf("RFQ creator should get own row capabilities, got %#v", creatorPayload)
	}
	if creatorPayload["can_start_costing"] != true || creatorPayload["can_create_bom"] != true || creatorPayload["can_create_cost"] != true || creatorPayload["can_create_quote"] != true {
		t.Fatalf("RFQ creator should get own downstream capabilities, got %#v", creatorPayload)
	}

	otherPayload := sanitizeMfgRFQ(rfq, MfgAccessContext{TenantID: 1, UserID: 12})
	if otherPayload["target_price"] != mfgMaskedPrice || otherPayload["price_masked"] != true {
		t.Fatalf("non-creator should see masked price, got %#v", otherPayload)
	}
	if otherPayload["can_edit"] == true || otherPayload["can_delete"] == true || otherPayload["can_submit"] == true || otherPayload["can_create_bom"] == true || otherPayload["can_create_cost"] == true || otherPayload["can_create_quote"] == true {
		t.Fatalf("non-creator should not get RFQ write capabilities, got %#v", otherPayload)
	}

	costPayload := sanitizeMfgRFQ(rfq, MfgAccessContext{TenantID: 1, UserID: 22, CanManageCost: true})
	if costPayload["can_start_costing"] != true {
		t.Fatalf("cost manager should get start costing capability")
	}
}

func TestMfgRFQReadinessAcceptsStandardOrCustomProductDemand(t *testing.T) {
	delivery := time.Now()
	base := models.MfgRFQ{
		CustomerID:           9,
		Quantity:             10,
		RequiredDeliveryDate: &delivery,
		NoDrawingReason:      "customer drawing pending",
	}

	missing := MfgRFQReadiness(base)
	if missing["ready"].(bool) {
		t.Fatalf("expected RFQ without product demand to be incomplete")
	}
	if got := missing["missing"].([]string); len(got) != 1 || got[0] != "product_demand" {
		t.Fatalf("expected product_demand missing, got %#v", missing)
	}
	if missing["next_action"] != "complete_demand" {
		t.Fatalf("expected next_action complete_demand, got %#v", missing)
	}

	standard := base
	standard.PayloadJSON = jsonStringFromAny(map[string]interface{}{
		"product_type": "standard",
		"product_id":   123,
		"product_name": "Servo controller",
	})
	if check := MfgRFQReadiness(standard); !check["ready"].(bool) {
		t.Fatalf("expected standard product RFQ to be ready, got %#v", check)
	}

	custom := base
	custom.PayloadJSON = jsonStringFromAny(map[string]interface{}{
		"product_type":          "custom",
		"custom_product_name":   "Custom controller",
		"specification":         "IP65 enclosure",
		"technical_requirement": "low noise",
	})
	if check := MfgRFQReadiness(custom); !check["ready"].(bool) {
		t.Fatalf("expected custom product RFQ to be ready, got %#v", check)
	}
}

func TestRFQClarificationItemsParsesRequirementList(t *testing.T) {
	payload := jsonStringFromAny(map[string]interface{}{
		"clarification_items": []map[string]interface{}{
			{
				"key":                "technical_requirements",
				"label":              "技术要求",
				"content":            "IP65",
				"remark":             "客户重点关注",
				"need_clarification": true,
				"sort_order":         2,
			},
			{
				"key":                "customer_notes",
				"label":              "客户备注",
				"content":            "仅内部记录",
				"need_clarification": false,
			},
			{
				"key":     "empty",
				"content": "",
			},
		},
	})

	items := rfqClarificationItems(payload)
	if len(items) != 2 {
		t.Fatalf("expected two non-empty clarification items, got %#v", items)
	}
	if items[0].SourceKey != "technical_requirements" || items[0].Label != "技术要求" || items[0].Content != "IP65" || items[0].Remark != "客户重点关注" || !items[0].NeedClarification || items[0].SortOrder != 2 {
		t.Fatalf("unexpected first item: %#v", items[0])
	}
	if items[1].NeedClarification {
		t.Fatalf("expected second item to be marked no-clarification: %#v", items[1])
	}
}

func TestRFQClarificationItemsDefaultsNeedClarification(t *testing.T) {
	payload := jsonStringFromAny(map[string]interface{}{
		"clarification_items": []map[string]interface{}{
			{
				"category": "quality",
				"content":  "需要确认认证标准",
			},
		},
	})

	items := rfqClarificationItems(payload)
	if len(items) != 1 {
		t.Fatalf("expected one item, got %#v", items)
	}
	if items[0].SourceKey != "requirement_1" || items[0].Label != "quality" || !items[0].NeedClarification || items[0].SortOrder != 1 {
		t.Fatalf("unexpected defaulted item: %#v", items[0])
	}
}

func TestRFQClarificationItemsIgnoreMissingRemark(t *testing.T) {
	payload := jsonStringFromAny(map[string]interface{}{
		"clarification_items": []map[string]interface{}{
			{
				"key":     "commercial_terms",
				"content": "确认付款条款",
			},
		},
	})

	items := rfqClarificationItems(payload)
	if len(items) != 1 {
		t.Fatalf("expected one item, got %#v", items)
	}
	if items[0].Remark != "" || items[0].Label != "" {
		t.Fatalf("expected missing optional fields to stay empty, got %#v", items[0])
	}
}

func TestUpdateMfgRFQClarificationRecordsAnswerHistory(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920001)
	user := models.User{
		TenantModel: models.TenantModel{TenantID: tenantID},
		Username:    "clarification-auditor",
		Password:    "x",
		Name:        "澄清回答人",
		Status:      "active",
	}
	if err := utils.DB.Create(&user).Error; err != nil {
		t.Fatalf("create answer user: %v", err)
	}
	rfq := models.MfgRFQ{
		TenantID: tenantID,
		RFQNo:    "RFQ-AUDIT-1",
	}
	if err := utils.DB.Create(&rfq).Error; err != nil {
		t.Fatalf("create rfq: %v", err)
	}
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQClarificationAnswerHistory{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQClarification{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQ{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.User{})

	clarification := models.MfgRFQClarification{
		TenantID: tenantID,
		RFQID:    rfq.ID,
		Question: "确认防护等级",
		Status:   "open",
		OwnerID:  77,
	}
	if err := utils.DB.Create(&clarification).Error; err != nil {
		t.Fatalf("create clarification: %v", err)
	}

	item, err := UpdateMfgRecord(tenantID, user.ID, "rfq-clarifications", clarification.ID, map[string]interface{}{
		"rfq_id":   rfq.ID,
		"question": "确认防护等级",
		"answer":   "IP65 可满足",
		"owner_id": 77,
		"status":   "open",
	})
	if err != nil {
		t.Fatalf("first answer update failed: %v", err)
	}
	saved := item.(models.MfgRFQClarification)
	if saved.LastAnsweredBy != user.ID || saved.LastAnsweredAt == nil {
		t.Fatalf("expected latest answer metadata, got %#v", saved)
	}
	if saved.Status != "open" {
		t.Fatalf("answer update must not auto-resolve, got %s", saved.Status)
	}
	if saved.OwnerID != 77 {
		t.Fatalf("answer update must not overwrite owner, got %d", saved.OwnerID)
	}

	histories, err := ListMfgRFQClarificationAnswerHistories(tenantID, clarification.ID)
	if err != nil {
		t.Fatalf("list histories: %v", err)
	}
	if len(histories) != 1 || histories[0].OldAnswer != "" || histories[0].NewAnswer != "IP65 可满足" || histories[0].AnsweredBy != user.ID || histories[0].AnsweredByName != "澄清回答人" {
		t.Fatalf("unexpected first history: %#v", histories)
	}

	if _, err := UpdateMfgRecord(tenantID, 35, "rfq-clarifications", clarification.ID, map[string]interface{}{
		"rfq_id":   rfq.ID,
		"question": "确认防护等级",
		"answer":   "IP65 可满足，需增加密封圈",
		"owner_id": 77,
		"status":   "resolved",
	}); err != nil {
		t.Fatalf("second answer update failed: %v", err)
	}
	histories, err = ListMfgRFQClarificationAnswerHistories(tenantID, clarification.ID)
	if err != nil {
		t.Fatalf("list histories after second answer: %v", err)
	}
	if len(histories) != 2 || histories[0].OldAnswer != "IP65 可满足" || histories[0].NewAnswer != "IP65 可满足，需增加密封圈" || histories[0].AnsweredBy != 35 {
		t.Fatalf("unexpected second history: %#v", histories)
	}
}

func TestUpdateMfgRFQClarificationRejectsAnswerDeletionAndStatusOnlyNoHistory(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920002)
	rfq := models.MfgRFQ{
		TenantID: tenantID,
		RFQNo:    "RFQ-AUDIT-2",
	}
	if err := utils.DB.Create(&rfq).Error; err != nil {
		t.Fatalf("create rfq: %v", err)
	}
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQClarificationAnswerHistory{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQClarification{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQ{})

	now := time.Now()
	clarification := models.MfgRFQClarification{
		TenantID:       tenantID,
		RFQID:          rfq.ID,
		Question:       "确认材质",
		Answer:         "使用 304",
		Status:         "open",
		LastAnsweredBy: 34,
		LastAnsweredAt: &now,
	}
	if err := utils.DB.Create(&clarification).Error; err != nil {
		t.Fatalf("create clarification: %v", err)
	}

	_, err := UpdateMfgRecord(tenantID, 35, "rfq-clarifications", clarification.ID, map[string]interface{}{
		"rfq_id":   rfq.ID,
		"question": "确认材质",
		"answer":   "  ",
		"status":   "open",
	})
	if err == nil || err.Error() != MfgErrorClarificationAnswerLocked {
		t.Fatalf("expected answer locked error, got %v", err)
	}

	if _, err := UpdateMfgRecord(tenantID, 35, "rfq-clarifications", clarification.ID, map[string]interface{}{
		"rfq_id":   rfq.ID,
		"question": "确认材质",
		"answer":   "使用 304",
		"status":   "resolved",
	}); err != nil {
		t.Fatalf("status-only update failed: %v", err)
	}
	histories, err := ListMfgRFQClarificationAnswerHistories(tenantID, clarification.ID)
	if err != nil {
		t.Fatalf("list histories: %v", err)
	}
	if len(histories) != 0 {
		t.Fatalf("status-only update must not create history, got %#v", histories)
	}
}

func TestCreateMfgBOMSnapshotFromTemplateCopiesTemplateLines(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920003)
	defer cleanupMfgBOMSnapshotTemplateTestData(t, tenantID)

	rfq := models.MfgRFQ{TenantID: tenantID, RFQNo: "RFQ-BOM-1", ProductFamilyID: 88}
	if err := utils.DB.Create(&rfq).Error; err != nil {
		t.Fatalf("create rfq: %v", err)
	}
	template := models.MfgBOMTemplate{
		TenantID:        tenantID,
		ProductFamilyID: 88,
		TemplateNo:      "BOMT-TEST-1",
		Version:         "v3",
		BOMLinesJSON: jsonStringFromAny([]map[string]interface{}{
			{"item_code": "A-100", "item_name": "电机", "qty": 2},
		}),
		Status: "draft",
	}
	if err := utils.DB.Create(&template).Error; err != nil {
		t.Fatalf("create template: %v", err)
	}

	snapshot, err := CreateMfgBOMSnapshotFromConfig(tenantID, 41, MfgBOMSnapshotInput{
		BOMTemplateID: template.ID,
		RFQID:         rfq.ID,
	})
	if err != nil {
		t.Fatalf("create snapshot from template: %v", err)
	}
	if snapshot.SourceTemplateID != template.ID || snapshot.SourceConfigVersion != "v3" {
		t.Fatalf("expected template source/version, got %#v", snapshot)
	}
	lines := bomLinesFromJSON(snapshot.BOMLinesJSON)
	if len(lines) != 1 || mapFromMfgInput(lines[0])["item_code"] != "A-100" {
		t.Fatalf("expected copied template lines, got %#v", lines)
	}

	template.BOMLinesJSON = jsonStringFromAny([]map[string]interface{}{
		{"item_code": "CHANGED", "item_name": "变更后物料", "qty": 1},
	})
	if err := utils.DB.Save(&template).Error; err != nil {
		t.Fatalf("update template: %v", err)
	}
	var reloaded models.MfgBOMSnapshot
	if err := utils.DB.First(&reloaded, snapshot.ID).Error; err != nil {
		t.Fatalf("reload snapshot: %v", err)
	}
	lines = bomLinesFromJSON(reloaded.BOMLinesJSON)
	if len(lines) != 1 || mapFromMfgInput(lines[0])["item_code"] != "A-100" {
		t.Fatalf("snapshot must keep immutable copied lines, got %#v", lines)
	}
}

func TestCreateMfgBOMSnapshotManualLinesDoNotLinkTemplate(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920004)
	defer cleanupMfgBOMSnapshotTemplateTestData(t, tenantID)

	snapshot, err := CreateMfgBOMSnapshotFromConfig(tenantID, 42, MfgBOMSnapshotInput{
		SourceConfigVersion: "v1",
		BOMLines: []map[string]interface{}{
			{"item_code": "M-100", "item_name": "手工件", "qty": 1},
		},
	})
	if err != nil {
		t.Fatalf("create manual snapshot: %v", err)
	}
	if snapshot.SourceTemplateID != 0 {
		t.Fatalf("manual snapshot should not link template, got %#v", snapshot)
	}
	lines := bomLinesFromJSON(snapshot.BOMLinesJSON)
	if len(lines) != 1 || mapFromMfgInput(lines[0])["item_code"] != "M-100" {
		t.Fatalf("expected manual lines, got %#v", lines)
	}
}

func TestCreateMfgBOMSnapshotSupersedesPreviousActiveRFQBOM(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920007)
	defer cleanupMfgBOMSnapshotTemplateTestData(t, tenantID)

	rfq := models.MfgRFQ{TenantID: tenantID, RFQNo: "RFQ-BOM-CURRENT", ProductFamilyID: 77}
	if err := utils.DB.Create(&rfq).Error; err != nil {
		t.Fatalf("create rfq: %v", err)
	}

	first, err := CreateMfgBOMSnapshotFromConfig(tenantID, 45, MfgBOMSnapshotInput{
		RFQID:               rfq.ID,
		SourceConfigVersion: "v1",
		BOMLines: []map[string]interface{}{
			{"item_code": "OLD-1", "item_name": "旧物料", "qty": 1},
		},
	})
	if err != nil {
		t.Fatalf("create first snapshot: %v", err)
	}
	second, err := CreateMfgBOMSnapshotFromConfig(tenantID, 46, MfgBOMSnapshotInput{
		RFQID:               rfq.ID,
		SourceConfigVersion: "v2",
		BOMLines: []map[string]interface{}{
			{"item_code": "NEW-1", "item_name": "新物料", "qty": 2},
		},
	})
	if err != nil {
		t.Fatalf("create second snapshot: %v", err)
	}

	var reloadedFirst models.MfgBOMSnapshot
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, first.ID).First(&reloadedFirst).Error; err != nil {
		t.Fatalf("reload first snapshot: %v", err)
	}
	if reloadedFirst.Status != "superseded" {
		t.Fatalf("expected first snapshot superseded, got %s", reloadedFirst.Status)
	}
	if second.Status != "active" {
		t.Fatalf("expected second snapshot active, got %s", second.Status)
	}

	workspace, err := MfgRFQQuoteWorkspace(tenantID, rfq.ID, true)
	if err != nil {
		t.Fatalf("load quote workspace: %v", err)
	}
	current, ok := workspace["current_bom_snapshot"].(map[string]interface{})
	if !ok || uintFromMfgInput(current["id"]) != second.ID {
		t.Fatalf("expected current BOM to be second snapshot, got %#v", workspace["current_bom_snapshot"])
	}
	if current["amount"] != nil || current["quote_amount"] != nil {
		t.Fatalf("BOM current payload must not expose amount fields, got %#v", current)
	}
	history, ok := workspace["bom_snapshot_history"].([]map[string]interface{})
	if !ok || len(history) != 1 || uintFromMfgInput(history[0]["id"]) != first.ID || history[0]["status"] != "superseded" {
		t.Fatalf("expected first snapshot in history, got %#v", workspace["bom_snapshot_history"])
	}
}

func TestMfgCostingUsesSingleCurrentCostSheetPerCurrentBOM(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920008)
	defer cleanupMfgBOMSnapshotTemplateTestData(t, tenantID)
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgCostSheet{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgQuoteRevision{})

	rfq := models.MfgRFQ{TenantID: tenantID, RFQNo: "RFQ-COST-CURRENT", Quantity: 2, TargetPrice: 100, PayloadJSON: `{"currency":"CNY"}`}
	if err := utils.DB.Create(&rfq).Error; err != nil {
		t.Fatalf("create rfq: %v", err)
	}
	firstBOM, err := CreateMfgBOMSnapshotFromConfig(tenantID, 45, MfgBOMSnapshotInput{
		RFQID: rfq.ID,
		BOMLines: []map[string]interface{}{
			{"item_code": "OLD", "item_name": "old", "qty": 1},
		},
	})
	if err != nil {
		t.Fatalf("create first bom: %v", err)
	}
	currentBOM, err := CreateMfgBOMSnapshotFromConfig(tenantID, 45, MfgBOMSnapshotInput{
		RFQID: rfq.ID,
		BOMLines: []map[string]interface{}{
			{"item_code": "NEW", "item_name": "new", "qty": 1},
		},
	})
	if err != nil {
		t.Fatalf("create current bom: %v", err)
	}

	if _, err := CalculateMfgCostSheet(tenantID, 46, map[string]interface{}{
		"rfq_id":          rfq.ID,
		"bom_snapshot_id": firstBOM.ID,
		"quote_amount":    100,
	}); err == nil || err.Error() != MfgErrorCurrentBOMRequired {
		t.Fatalf("expected historical BOM rejected for costing, got %v", err)
	}
	firstCost, err := CalculateMfgCostSheet(tenantID, 46, map[string]interface{}{
		"rfq_id":          rfq.ID,
		"bom_snapshot_id": currentBOM.ID,
		"quote_amount":    100,
		"material_cost":   30,
	})
	if err != nil {
		t.Fatalf("calculate first cost: %v", err)
	}
	secondCost, err := CalculateMfgCostSheet(tenantID, 47, map[string]interface{}{
		"rfq_id":          rfq.ID,
		"bom_snapshot_id": currentBOM.ID,
		"quote_amount":    120,
		"material_cost":   40,
	})
	if err != nil {
		t.Fatalf("calculate second cost: %v", err)
	}
	if firstCost.ID != secondCost.ID || secondCost.Amount != 120 || secondCost.MaterialCost != 40 {
		t.Fatalf("expected costing upsert, first=%#v second=%#v", firstCost, secondCost)
	}
	var count int64
	if err := utils.DB.Model(&models.MfgCostSheet{}).Where("tenant_id = ? AND rfq_id = ? AND bom_snapshot_id = ?", tenantID, rfq.ID, currentBOM.ID).Count(&count).Error; err != nil {
		t.Fatalf("count cost sheets: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one current cost sheet, got %d", count)
	}
	if _, err := CreateMfgQuote(tenantID, 48, map[string]interface{}{
		"rfq_id":          rfq.ID,
		"cost_sheet_id":   secondCost.ID,
		"bom_snapshot_id": firstBOM.ID,
		"amount":          120,
		"currency":        "CNY",
	}); err == nil || err.Error() != MfgErrorCurrentBOMRequired {
		t.Fatalf("expected historical BOM rejected for quote, got %v", err)
	}
}

func TestCreateMfgBOMSnapshotSaveAsTemplateCreatesTemplateAndLinksSnapshot(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920005)
	defer cleanupMfgBOMSnapshotTemplateTestData(t, tenantID)

	rfq := models.MfgRFQ{TenantID: tenantID, RFQNo: "RFQ-BOM-2", ProductFamilyID: 99}
	if err := utils.DB.Create(&rfq).Error; err != nil {
		t.Fatalf("create rfq: %v", err)
	}
	snapshot, err := CreateMfgBOMSnapshotFromConfig(tenantID, 43, MfgBOMSnapshotInput{
		RFQID:               rfq.ID,
		SourceConfigVersion: "v8",
		BOMLines: []map[string]interface{}{
			{"item_code": "S-100", "item_name": "可沉淀物料", "qty": 5},
		},
		SaveAsTemplate:  true,
		TemplateNo:      "",
		TemplateVersion: "v8",
		TemplateStatus:  "draft",
		TemplateRule: map[string]interface{}{
			"remark": "适用于该 RFQ",
		},
	})
	if err != nil {
		t.Fatalf("create snapshot and template: %v", err)
	}
	if snapshot.SourceTemplateID == 0 {
		t.Fatalf("expected snapshot linked to created template, got %#v", snapshot)
	}
	var template models.MfgBOMTemplate
	if err := utils.DB.Where("tenant_id = ? AND id = ?", tenantID, snapshot.SourceTemplateID).First(&template).Error; err != nil {
		t.Fatalf("load created template: %v", err)
	}
	if template.ProductFamilyID != rfq.ProductFamilyID || template.TemplateNo == "" || template.Version != "v8" || template.Status != "draft" {
		t.Fatalf("unexpected created template: %#v", template)
	}
	lines := bomLinesFromJSON(template.BOMLinesJSON)
	if len(lines) != 1 || mapFromMfgInput(lines[0])["item_code"] != "S-100" {
		t.Fatalf("expected saved template lines, got %#v", lines)
	}
}

func TestCreateMfgBOMSnapshotSaveAsTemplateRollsBackWhenTemplateFails(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920006)
	defer cleanupMfgBOMSnapshotTemplateTestData(t, tenantID)

	_, err := CreateMfgBOMSnapshotFromConfig(tenantID, 44, MfgBOMSnapshotInput{
		SourceConfigVersion: "v1",
		BOMLines: []map[string]interface{}{
			{"item_code": "FAIL-1", "item_name": "失败物料", "qty": 1},
		},
		SaveAsTemplate:  true,
		TemplateNo:      strings.Repeat("X", 80),
		TemplateVersion: "v1",
	})
	if err == nil {
		t.Fatalf("expected template create failure")
	}
	var snapshotCount int64
	if err := utils.DB.Model(&models.MfgBOMSnapshot{}).Where("tenant_id = ?", tenantID).Count(&snapshotCount).Error; err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	if snapshotCount != 0 {
		t.Fatalf("snapshot should roll back with failed template, got %d", snapshotCount)
	}
}

func cleanupMfgBOMSnapshotTemplateTestData(t *testing.T, tenantID uint) {
	t.Helper()
	utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgBOMSnapshot{})
	utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgBOMTemplate{})
	utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQ{})
}

func TestApplyMfgRFQClarificationAnswerAuditRules(t *testing.T) {
	now := time.Now()
	before := models.MfgRFQClarification{
		TenantID:       2,
		RFQID:          8,
		Question:       "确认材质",
		Answer:         "使用 304",
		OwnerID:        77,
		Status:         "open",
		LastAnsweredBy: 34,
	}
	next := before
	next.Answer = "使用 316"
	next.Status = "resolved"
	history, err := applyMfgRFQClarificationAnswerAudit(before, &next, 35, now, "技术张三")
	if err != nil {
		t.Fatalf("answer audit failed: %v", err)
	}
	if history == nil || history.OldAnswer != "使用 304" || history.NewAnswer != "使用 316" || history.OldStatus != "open" || history.NewStatus != "resolved" || history.AnsweredBy != 35 || history.AnsweredByName != "技术张三" {
		t.Fatalf("unexpected history: %#v", history)
	}
	if next.LastAnsweredBy != 35 || next.LastAnsweredAt == nil || !next.LastAnsweredAt.Equal(now) {
		t.Fatalf("expected latest answer metadata, got %#v", next)
	}
	if next.OwnerID != 77 {
		t.Fatalf("answer audit must not overwrite owner, got %d", next.OwnerID)
	}

	statusOnly := before
	statusOnly.Status = "resolved"
	history, err = applyMfgRFQClarificationAnswerAudit(before, &statusOnly, 35, now, "技术张三")
	if err != nil {
		t.Fatalf("status-only audit failed: %v", err)
	}
	if history != nil {
		t.Fatalf("status-only update must not create history, got %#v", history)
	}
	if statusOnly.LastAnsweredBy != 34 {
		t.Fatalf("status-only update must keep previous answerer, got %d", statusOnly.LastAnsweredBy)
	}

	deleted := before
	deleted.Answer = ""
	if _, err := applyMfgRFQClarificationAnswerAudit(before, &deleted, 35, now, "技术张三"); err == nil || err.Error() != MfgErrorClarificationAnswerLocked {
		t.Fatalf("expected locked error for answer deletion, got %v", err)
	}
}

func TestBuildPendingCostSheetForRFQCarriesContextAndBOMRisk(t *testing.T) {
	delivery := time.Now()
	rfq := models.MfgRFQ{
		BaseModel:            models.BaseModel{ID: 88},
		RFQNo:                "RFQ-1",
		CustomerID:           10,
		Quantity:             3,
		TargetPrice:          2000,
		RequiredDeliveryDate: &delivery,
		NoDrawingReason:      "customer has no drawing",
		PayloadJSON: jsonStringFromAny(map[string]interface{}{
			"customer_name":          "Acme Manufacturing",
			"product_type":           "custom",
			"custom_product_name":    "Custom controller",
			"technical_requirements": "IP65",
			"quality_requirements":   "CE",
			"currency":               "CNY",
		}),
	}

	item := buildPendingCostSheetForRFQ(2, 34, rfq, 0)
	if item.Status != "pending" || item.MarginLevel != "pending" {
		t.Fatalf("expected pending cost sheet, got status=%s margin=%s", item.Status, item.MarginLevel)
	}
	if item.Amount != 2000 || item.RFQID != rfq.ID || item.BOMSnapshotID != 0 {
		t.Fatalf("unexpected pending cost sheet core fields: %#v", item)
	}
	context := jsonMap(item.SensitiveCostJSON)
	if context["bom_missing"] != true || context["next_action"] != "create_bom_snapshot" {
		t.Fatalf("expected BOM missing next action, got %#v", context)
	}
	if context["assigned_role"] != "pricing_specialist" || context["technical_role"] != "sales_engineer" {
		t.Fatalf("expected role hints, got %#v", context)
	}
	riskFlags, ok := context["risk_flags"].([]interface{})
	if !ok || len(riskFlags) == 0 {
		t.Fatalf("expected risk flags in context, got %#v", context["risk_flags"])
	}
}

func TestSanitizeMfgCostSheetMasksSensitiveCosts(t *testing.T) {
	item := models.MfgCostSheet{
		TenantID:              7,
		RFQID:                 11,
		BOMSnapshotID:         13,
		MaterialCost:          100,
		LaborCost:             50,
		OutsourcingCost:       25,
		ManufacturingOverhead: 10,
		Amount:                300,
		GrossMargin:           115,
		GrossMarginRate:       0.38,
		MarginLevel:           "high",
		SensitiveCostJSON:     `{"supplier":"secret"}`,
	}

	payload, ok := SanitizeMfgRecords("cost-sheets", item, false).(map[string]interface{})
	if !ok {
		t.Fatalf("expected sanitized map")
	}
	if payload["material_cost"] != "***" || payload["amount"] != "***" || payload["gross_margin_rate"] != "***" {
		t.Fatalf("expected price fields to be masked with ***, got %#v", payload)
	}
	if _, exists := payload["sensitive_cost_json"]; exists {
		t.Fatalf("expected sensitive_cost_json to be masked, got %#v", payload)
	}
	if payload["price_masked"] != true {
		t.Fatalf("expected price_masked=true, got %#v", payload)
	}
}

func TestSanitizeMfgRFQPriceByCreatorAndRole(t *testing.T) {
	rfq := models.MfgRFQ{
		TenantID:    7,
		RFQNo:       "RFQ-PRICE",
		TargetPrice: 88,
		CreatedBy:   42,
	}
	creator := sanitizeMfgRFQ(rfq, MfgAccessContext{UserID: 42})
	if creator["target_price"] != float64(88) {
		t.Fatalf("creator should see target price, got %#v", creator)
	}
	other := sanitizeMfgRFQ(rfq, MfgAccessContext{UserID: 43})
	if other["target_price"] != "***" || other["price_masked"] != true {
		t.Fatalf("non-creator should see masked price, got %#v", other)
	}
	supervisor := sanitizeMfgRFQ(rfq, MfgAccessContext{UserID: 43, IsSupervisor: true})
	if supervisor["target_price"] != float64(88) {
		t.Fatalf("supervisor should see target price, got %#v", supervisor)
	}
	approver := sanitizeMfgRFQ(rfq, MfgAccessContext{UserID: 43, CanApproveQuote: true})
	if approver["target_price"] != float64(88) {
		t.Fatalf("quote approver should see target price for approval, got %#v", approver)
	}
}

func TestChangeMfgQuoteApprovalRecordsActorAndTime(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920020)
	rfq := models.MfgRFQ{
		TenantID: tenantID,
		RFQNo:    "RFQ-APPROVAL-1",
	}
	if err := utils.DB.Create(&rfq).Error; err != nil {
		t.Fatalf("create rfq: %v", err)
	}
	quote := models.MfgQuoteRevision{
		TenantID:       tenantID,
		QuoteNo:        "MQ-APPROVAL-1",
		RFQID:          rfq.ID,
		Amount:         100,
		Currency:       "CNY",
		ApprovalStatus: "pending",
		Status:         "draft",
	}
	if err := utils.DB.Create(&quote).Error; err != nil {
		t.Fatalf("create quote: %v", err)
	}
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgQuoteApprovalHistory{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgQuoteRevision{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQ{})

	if _, err := ChangeMfgQuoteApprovalWithRemark(tenantID, 81, quote.ID, "approved", ""); err == nil || err.Error() != MfgErrorApprovalRemarkRequired {
		t.Fatalf("expected approval remark required, got %v", err)
	}

	approved, err := ChangeMfgQuoteApprovalWithRemark(tenantID, 81, quote.ID, "approved", "价格和毛利符合要求")
	if err != nil {
		t.Fatalf("approve quote: %v", err)
	}
	if approved.ApprovalStatus != "approved" || approved.Status != "approved" || approved.ApprovedBy != 81 || approved.ApprovedAt == nil {
		t.Fatalf("expected approval audit fields, got %#v", approved)
	}
	var approvalHistory models.MfgQuoteApprovalHistory
	if err := utils.DB.Where("tenant_id = ? AND quote_id = ? AND action = ?", tenantID, quote.ID, "approve").First(&approvalHistory).Error; err != nil {
		t.Fatalf("load approval history: %v", err)
	}
	if approvalHistory.Remark != "价格和毛利符合要求" || approvalHistory.OperatorID != 81 || approvalHistory.OldApprovalStatus != "pending" || approvalHistory.NewApprovalStatus != "approved" {
		t.Fatalf("unexpected approval history: %#v", approvalHistory)
	}

	rejected, err := ChangeMfgQuoteApprovalWithRemark(tenantID, 82, quote.ID, "rejected", "客户条款需要重做")
	if err != nil {
		t.Fatalf("reject quote: %v", err)
	}
	if rejected.ApprovalStatus != "rejected" || rejected.Status != "rejected" || rejected.RejectedBy != 82 || rejected.RejectedAt == nil {
		t.Fatalf("expected rejection audit fields, got %#v", rejected)
	}
	var historyCount int64
	if err := utils.DB.Model(&models.MfgQuoteApprovalHistory{}).Where("tenant_id = ? AND quote_id = ?", tenantID, quote.ID).Count(&historyCount).Error; err != nil {
		t.Fatalf("count approval history: %v", err)
	}
	if historyCount != 2 {
		t.Fatalf("expected two approval history rows, got %d", historyCount)
	}
}

func TestMfgDeliveryChangeApprovalHistories(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920026)
	ecn := models.MfgECN{
		TenantID:       tenantID,
		ECNNo:          "ECN-HISTORY-1",
		ApprovalStatus: "draft",
		Status:         "draft",
	}
	if err := utils.DB.Create(&ecn).Error; err != nil {
		t.Fatalf("create ecn: %v", err)
	}
	compensation := models.MfgCompensationRecord{
		TenantID:       tenantID,
		CompensationNo: "CMP-HISTORY-1",
		ApprovalStatus: "draft",
		Status:         "open",
	}
	if err := utils.DB.Create(&compensation).Error; err != nil {
		t.Fatalf("create compensation: %v", err)
	}
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgApprovalHistory{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgECN{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgCompensationRecord{})

	if _, err := ChangeMfgECNStatus(tenantID, 91, ecn.ID, "submit-approval", map[string]interface{}{}); err != nil {
		t.Fatalf("submit ecn approval: %v", err)
	}
	if _, err := ChangeMfgECNStatus(tenantID, 92, ecn.ID, "reject", map[string]interface{}{}); err == nil || err.Error() != MfgErrorApprovalRemarkRequired {
		t.Fatalf("expected ecn reject remark required, got %v", err)
	}
	if _, err := ChangeMfgECNStatus(tenantID, 92, ecn.ID, "reject", map[string]interface{}{"remark": "影响范围需要补充"}); err != nil {
		t.Fatalf("reject ecn: %v", err)
	}

	if _, err := ChangeMfgCompensationStatus(tenantID, 93, compensation.ID, "submit-approval", map[string]interface{}{}); err != nil {
		t.Fatalf("submit compensation approval: %v", err)
	}
	if _, err := ChangeMfgCompensationStatus(tenantID, 94, compensation.ID, "reject", map[string]interface{}{}); err == nil || err.Error() != MfgErrorApprovalRemarkRequired {
		t.Fatalf("expected compensation reject remark required, got %v", err)
	}
	if _, err := ChangeMfgCompensationStatus(tenantID, 94, compensation.ID, "reject", map[string]interface{}{"remark": "责任划分不清楚"}); err != nil {
		t.Fatalf("reject compensation: %v", err)
	}

	ecnHistories := listMfgApprovalHistories(tenantID, "ecns", ecn.ID)
	if len(ecnHistories) != 2 || ecnHistories[0]["action"] != "reject" || ecnHistories[0]["remark"] != "影响范围需要补充" {
		t.Fatalf("unexpected ecn histories: %#v", ecnHistories)
	}
	compensationHistories := listMfgApprovalHistories(tenantID, "compensation-records", compensation.ID)
	if len(compensationHistories) != 2 || compensationHistories[0]["action"] != "reject" || compensationHistories[0]["remark"] != "责任划分不清楚" {
		t.Fatalf("unexpected compensation histories: %#v", compensationHistories)
	}
}

func TestCreateMfgQuoteForcesDraftApprovalStatus(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920021)
	rfq := models.MfgRFQ{
		TenantID: tenantID,
		RFQNo:    "RFQ-APPROVAL-2",
	}
	if err := utils.DB.Create(&rfq).Error; err != nil {
		t.Fatalf("create rfq: %v", err)
	}
	bom := models.MfgBOMSnapshot{
		TenantID:   tenantID,
		RFQID:      rfq.ID,
		SnapshotNo: "BOM-APPROVAL-2",
		Status:     "active",
	}
	if err := utils.DB.Create(&bom).Error; err != nil {
		t.Fatalf("create bom: %v", err)
	}
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgQuoteRevision{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgBOMSnapshot{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQ{})

	quote, err := CreateMfgQuote(tenantID, 81, map[string]interface{}{
		"rfq_id":          rfq.ID,
		"bom_snapshot_id": bom.ID,
		"amount":          100,
		"currency":        "CNY",
		"approval_status": "pending",
		"status":          "pending",
	})
	if err != nil {
		t.Fatalf("create quote: %v", err)
	}
	if quote.ApprovalStatus != "draft" {
		t.Fatalf("new quote must start as approval draft, got %#v", quote)
	}
}

func TestEnrichMfgQuoteUsesApprovalAuditFallbackWhenHistoryMissing(t *testing.T) {
	now := time.Now()
	item := models.MfgQuoteRevision{
		TenantID:       1,
		QuoteNo:        "MQ-APPROVAL-FALLBACK",
		Revision:       2,
		RFQID:          3,
		ApprovalStatus: "approved",
		Status:         "approved",
		ApprovedBy:     81,
		ApprovedAt:     &now,
	}
	payload := enrichMfgQuoteRevisionForAccess(item, MfgAccessContext{IsAdmin: true})
	histories, ok := payload["approval_histories"].([]map[string]interface{})
	if !ok || len(histories) != 1 {
		t.Fatalf("expected fallback approval history, got %#v", payload["approval_histories"])
	}
	if histories[0]["action"] != "approve" || histories[0]["new_approval_status"] != "approved" || histories[0]["operator_id"] != uint(81) {
		t.Fatalf("unexpected fallback approval history: %#v", histories[0])
	}
}

func TestEnrichMfgQuoteUsesApprovalOperationLogFallbackWhenHistoryMissing(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920023)
	now := time.Now()
	rfq := models.MfgRFQ{
		TenantID: tenantID,
		RFQNo:    "RFQ-APPROVAL-LOG-FALLBACK",
	}
	if err := utils.DB.Create(&rfq).Error; err != nil {
		t.Fatalf("create rfq: %v", err)
	}
	quote := models.MfgQuoteRevision{
		TenantID:       tenantID,
		QuoteNo:        "MQ-APPROVAL-LOG-FALLBACK",
		Revision:       1,
		RFQID:          rfq.ID,
		ApprovalStatus: "approved",
		Status:         "approved",
		ApprovedBy:     81,
		ApprovedAt:     &now,
	}
	if err := utils.DB.Create(&quote).Error; err != nil {
		t.Fatalf("create quote: %v", err)
	}
	log := models.OperationLog{
		TenantModel: models.TenantModel{TenantID: tenantID},
		UserID:      81,
		Username:    "审批主管",
		Operation:   "APPROVE_MFG_QUOTE",
		Target:      fmt.Sprintf("mfg/quotes[%d]", quote.ID),
		ParamsJSON:  `{"id":1,"approval_status":"approved","remark":"毛利率达到审批要求"}`,
		Result:      "success",
		Source:      "web",
	}
	if err := utils.DB.Create(&log).Error; err != nil {
		t.Fatalf("create operation log: %v", err)
	}
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.OperationLog{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgQuoteApprovalHistory{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgQuoteRevision{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQ{})

	payload := enrichMfgQuoteRevisionForAccess(quote, MfgAccessContext{IsAdmin: true})
	histories, ok := payload["approval_histories"].([]map[string]interface{})
	if !ok || len(histories) != 1 {
		t.Fatalf("expected operation log fallback approval history, got %#v", payload["approval_histories"])
	}
	if histories[0]["action"] != "approve" || histories[0]["remark"] != "毛利率达到审批要求" || histories[0]["operator_name"] != "审批主管" {
		t.Fatalf("unexpected operation log fallback approval history: %#v", histories[0])
	}
}

func TestEnrichMfgQuoteUsesCRMCustomerNameFallbackAndDescendingApprovalHistory(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920024)
	customer := models.CRMObject{
		TenantModel: models.TenantModel{TenantID: tenantID},
		Domain:      "companies",
		ObjectType:  "company",
		Name:        "真实客户名称",
		Code:        "CUST-REAL",
		Status:      "active",
	}
	if err := utils.DB.Create(&customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}
	rfq := models.MfgRFQ{
		TenantID:   tenantID,
		RFQNo:      "RFQ-CUSTOMER-FALLBACK",
		CustomerID: customer.ID,
		PayloadJSON: jsonStringFromAny(map[string]interface{}{
			"product_name": "测试产品",
		}),
	}
	if err := utils.DB.Create(&rfq).Error; err != nil {
		t.Fatalf("create rfq: %v", err)
	}
	quote := models.MfgQuoteRevision{
		TenantID:       tenantID,
		QuoteNo:        "MQ-CUSTOMER-FALLBACK",
		Revision:       1,
		RFQID:          rfq.ID,
		ApprovalStatus: "approved",
		Status:         "approved",
	}
	if err := utils.DB.Create(&quote).Error; err != nil {
		t.Fatalf("create quote: %v", err)
	}
	older := time.Now().Add(-time.Hour)
	newer := time.Now()
	histories := []models.MfgQuoteApprovalHistory{
		{TenantID: tenantID, RFQID: rfq.ID, QuoteID: quote.ID, QuoteNo: quote.QuoteNo, Revision: 1, Action: "submit", OldApprovalStatus: "draft", NewApprovalStatus: "pending", OperatorID: 11, OperatorName: "提交人", OperatedAt: older},
		{TenantID: tenantID, RFQID: rfq.ID, QuoteID: quote.ID, QuoteNo: quote.QuoteNo, Revision: 1, Action: "approve", OldApprovalStatus: "pending", NewApprovalStatus: "approved", OperatorID: 12, OperatorName: "审批人", OperatedAt: newer},
	}
	if err := utils.DB.Create(&histories).Error; err != nil {
		t.Fatalf("create approval histories: %v", err)
	}
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgQuoteApprovalHistory{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgQuoteRevision{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQ{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.CRMObject{})

	payload := enrichMfgQuoteRevisionForAccess(quote, MfgAccessContext{IsAdmin: true})
	if payload["customer_name"] != "真实客户名称" {
		t.Fatalf("expected real customer name fallback, got %#v", payload["customer_name"])
	}
	items, ok := payload["approval_histories"].([]map[string]interface{})
	if !ok || len(items) != 2 {
		t.Fatalf("expected approval history items, got %#v", payload["approval_histories"])
	}
	if items[0]["action"] != "approve" || items[1]["action"] != "submit" {
		t.Fatalf("expected descending approval history, got %#v", items)
	}
}

func TestCreateMfgQuoteIncrementsRevisionPerRFQ(t *testing.T) {
	if utils.DB == nil {
		t.Skip("database is not configured")
	}
	tenantID := uint(920022)
	rfq := models.MfgRFQ{
		TenantID: tenantID,
		RFQNo:    "RFQ-REVISION-1",
	}
	if err := utils.DB.Create(&rfq).Error; err != nil {
		t.Fatalf("create rfq: %v", err)
	}
	bom := models.MfgBOMSnapshot{
		TenantID:   tenantID,
		RFQID:      rfq.ID,
		SnapshotNo: "BOM-REVISION-1",
		Status:     "active",
	}
	if err := utils.DB.Create(&bom).Error; err != nil {
		t.Fatalf("create bom: %v", err)
	}
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgQuoteApprovalHistory{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgQuoteRevision{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgBOMSnapshot{})
	defer utils.DB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.MfgRFQ{})

	for index := 1; index <= 3; index++ {
		quote, err := CreateMfgQuote(tenantID, 81, map[string]interface{}{
			"rfq_id":          rfq.ID,
			"bom_snapshot_id": bom.ID,
			"amount":          100 + index,
			"currency":        "CNY",
		})
		if err != nil {
			t.Fatalf("create quote %d: %v", index, err)
		}
		if quote.Revision != index {
			t.Fatalf("expected revision %d, got %#v", index, quote)
		}
	}
}

func TestSanitizeMfgQuoteForCustomerHidesCostAndBOMSnapshot(t *testing.T) {
	item := models.MfgQuoteRevision{
		QuoteNo:       "MQ-1",
		Revision:      2,
		RFQID:         3,
		CostSheetID:   4,
		BOMSnapshotID: 5,
		Amount:        99,
		Currency:      "USD",
	}

	payload := sanitizeMfgQuoteForCustomer(item)
	if _, exists := payload["cost_sheet_id"]; exists {
		t.Fatalf("expected cost_sheet_id to be hidden, got %#v", payload)
	}
	if _, exists := payload["bom_snapshot_id"]; exists {
		t.Fatalf("expected bom_snapshot_id to be hidden, got %#v", payload)
	}
	if payload["quote_no"] != "MQ-1" || payload["amount"] != float64(99) {
		t.Fatalf("expected customer quote summary, got %#v", payload)
	}
}

func TestValidateConfigValuesSupportsRequiredRangeMutexAndDependency(t *testing.T) {
	model := models.MfgConfigModel{
		ConstraintsJSON: jsonStringFromAny(map[string]interface{}{
			"required":   []interface{}{"voltage"},
			"range":      map[string]interface{}{"power": map[string]interface{}{"min": 10, "max": 20}},
			"mutex":      []interface{}{[]interface{}{"option_a", "option_b"}},
			"dependency": map[string]interface{}{"sensor": []interface{}{"controller"}},
		}),
	}
	errorsByField := validateConfigValues(model, map[string]interface{}{
		"power":    25,
		"option_a": true,
		"option_b": true,
		"sensor":   "enabled",
	})

	expected := map[string]string{
		"voltage":    "required",
		"power":      "range",
		"option_a":   "mutex",
		"option_b":   "mutex",
		"controller": "dependency",
	}
	for key, want := range expected {
		if errorsByField[key] != want {
			t.Fatalf("expected %s=%s, got %#v", key, want, errorsByField)
		}
	}
}

func TestValidateMfgDrawingAccessPolicyRequiresApprovalForL4Download(t *testing.T) {
	drawing := models.MfgDrawing{
		ClassificationLevel: "L4",
		DownloadPolicy:      "approval_required",
		ApprovalStatus:      "pending",
	}

	if err := validateMfgDrawingAccessPolicy(drawing, "view"); err != nil {
		t.Fatalf("view should only audit without approval gate, got %v", err)
	}
	if err := validateMfgDrawingAccessPolicy(drawing, "download"); err == nil || err.Error() != MfgErrorDrawingApprovalRequired {
		t.Fatalf("expected approval required for L4 download, got %v", err)
	}
	drawing.ApprovalStatus = "approved"
	if err := validateMfgDrawingAccessPolicy(drawing, "download"); err != nil {
		t.Fatalf("approved L4 download should pass, got %v", err)
	}
}

func TestValidateMfgDrawingAccessPolicyRequiresApprovalForExternalSend(t *testing.T) {
	drawing := models.MfgDrawing{
		ClassificationLevel: "L3",
		DownloadPolicy:      "audit_required",
		ExternalSendPolicy:  "approval_required",
		ApprovalStatus:      "pending",
	}

	if err := validateMfgDrawingAccessPolicy(drawing, "download"); err != nil {
		t.Fatalf("audit-only download should pass, got %v", err)
	}
	if err := validateMfgDrawingAccessPolicy(drawing, "external-send"); err == nil || err.Error() != MfgErrorDrawingApprovalRequired {
		t.Fatalf("expected approval required for external send, got %v", err)
	}
}

func TestMfgAssetIoTAuthorizedRequiresExplicitActiveAuthorization(t *testing.T) {
	now := testingNow()
	cases := []struct {
		name  string
		asset models.MfgAsset
		want  bool
	}{
		{name: "empty ref", asset: models.MfgAsset{IoTAuthorized: true}, want: false},
		{name: "not authorized", asset: models.MfgAsset{IoTAssetRef: "iot-1"}, want: false},
		{name: "authorized", asset: models.MfgAsset{IoTAssetRef: "iot-1", IoTAuthorized: true}, want: true},
		{name: "revoked", asset: models.MfgAsset{IoTAssetRef: "iot-1", IoTAuthorized: true, IoTRevokedAt: &now}, want: false},
	}
	for _, tc := range cases {
		if got := mfgAssetIoTAuthorized(tc.asset); got != tc.want {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, got)
		}
	}
}

func TestMfgVersionedRecordLockedProtectsPublishedManufacturingVersions(t *testing.T) {
	if !mfgVersionedRecordLocked(&models.MfgConfigModel{Status: "active"}) {
		t.Fatalf("expected active config model to be locked")
	}
	if !mfgVersionedRecordLocked(&models.MfgBOMTemplate{Status: "published"}) {
		t.Fatalf("expected published BOM template to be locked")
	}
	if !mfgVersionedRecordLocked(&models.MfgQuoteRevision{ApprovalStatus: "approved"}) {
		t.Fatalf("expected approved quote revision to be locked")
	}
	if mfgVersionedRecordLocked(&models.MfgConfigModel{Status: "draft"}) {
		t.Fatalf("expected draft config model to remain editable")
	}
}

func TestSanitizeMfgAssetForCustomerHidesInternalFields(t *testing.T) {
	now := testingNow()
	payload := sanitizeMfgAssetForCustomer(models.MfgAsset{
		TenantID:        7,
		CustomerID:      9,
		OrderID:         11,
		BOMSnapshotID:   13,
		SerialNo:        "SN-1",
		IoTAssetRef:     "iot-secret",
		IoTAuthorized:   true,
		IoTAuthorizedAt: &now,
	})

	for _, key := range []string{"tenant_id", "bom_snapshot_id", "iot_asset_ref", "iot_authorized", "iot_authorized_at", "iot_revoked_at"} {
		if _, exists := payload[key]; exists {
			t.Fatalf("expected %s to be hidden from customer asset view, got %#v", key, payload)
		}
	}
	if payload["serial_no"] != "SN-1" || payload["sensitive_masked"] != true {
		t.Fatalf("expected sanitized customer asset summary, got %#v", payload)
	}
}

func TestMfgDashboardMetricGroupsCoverRequiredDecisionDimensions(t *testing.T) {
	now := testingNow()
	groups := mfgDashboardMetricGroups(
		map[string]int64{
			"rfq_count":              2,
			"open_rfq_count":         1,
			"low_margin_quote_count": 1,
			"asset_count":            3,
			"service_ticket_count":   4,
		},
		1200,
		models.MetricMfgQuoteSnapshot{RefreshStatus: "generated", LowMarginQuoteRate: 0.25, BaseModel: models.BaseModel{UpdatedAt: now}},
		models.MetricMfgDeliverySnapshot{RefreshStatus: "generated", RedCount: 1, BaseModel: models.BaseModel{UpdatedAt: now}},
		models.MetricMfgServiceSnapshot{RefreshStatus: "generated", TicketCount: 4, BaseModel: models.BaseModel{UpdatedAt: now}},
	)

	expected := map[string]bool{
		"growth":     false,
		"efficiency": false,
		"quality":    false,
		"delivery":   false,
		"profit":     false,
		"cash":       false,
		"lifecycle":  false,
	}
	for _, group := range groups {
		key, _ := group["key"].(string)
		if _, ok := expected[key]; ok {
			expected[key] = true
		}
		if group["drilldown"] == "" {
			t.Fatalf("expected group %s to expose drilldown target: %#v", key, group)
		}
	}
	for key, seen := range expected {
		if !seen {
			t.Fatalf("expected dashboard metric group %s, got %#v", key, groups)
		}
	}
}

func TestDefaultMarginLevelBoundaries(t *testing.T) {
	if got := marginLevel(0.149); got != "low" {
		t.Fatalf("expected low below default threshold, got %s", got)
	}
	if got := marginLevel(0.2); got != "medium" {
		t.Fatalf("expected medium between thresholds, got %s", got)
	}
	if got := marginLevel(0.35); got != "high" {
		t.Fatalf("expected high above threshold, got %s", got)
	}
}

func testingNow() time.Time {
	return time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
}
