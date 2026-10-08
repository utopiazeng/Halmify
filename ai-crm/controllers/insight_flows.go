package controllers

import (
	"ai-crm/models"
	"ai-crm/utils"
	"net/http"
	"time"
)

func (c *CRMController) GenerateHotInsights() {
	auth, ok := requireTenantUser(c.Ctx)
	if !ok {
		return
	}
	if err := ensureCRMModuleAccess(auth, "insights", "edit"); err != nil {
		utils.Error(c.Ctx, http.StatusForbidden, err.Error())
		return
	}
	var topDeal models.CRMObject
	_ = crmScopedObjectQuery(auth, "deals").Where("amount > 0").Order("amount desc, updated_at desc").First(&topDeal).Error
	var topProduct models.CRMObject
	_ = crmScopedObjectQuery(auth, "products").Order("score desc, amount desc, updated_at desc").First(&topProduct).Error
	now := time.Now().Format(time.RFC3339)
	items := []models.CRMObject{}
	if topDeal.ID > 0 {
		payload := map[string]interface{}{
			"source":       "deal_amount",
			"deal_id":      topDeal.ID,
			"amount":       topDeal.Amount,
			"currency":     topDeal.Currency,
			"generated_at": now,
			"suggestion":   "优先安排复购跟进并补齐客户画像",
		}
		hotCustomer := cloneCRMObject(auth, "insights", "hot_customer", "热点客户："+topDeal.Name, topDeal)
		hotCustomer.Status = "watching"
		hotCustomer.Score = 80
		hotCustomer.PayloadJSON = toJSONString(payload)
		items = append(items, hotCustomer)
	}
	if topProduct.ID > 0 {
		payload := payloadMap(topProduct.PayloadJSON)
		payload["source"] = "product_signal"
		payload["product_id"] = topProduct.ID
		payload["generated_at"] = now
		payload["suggestion"] = "关注库存、报价和内容曝光，联动销售活动"
		hotProduct := cloneCRMObject(auth, "insights", "hot_product", "热点产品："+topProduct.Name, topProduct)
		hotProduct.Status = "watching"
		if hotProduct.Score == 0 {
			hotProduct.Score = 70
		}
		hotProduct.PayloadJSON = toJSONString(payload)
		items = append(items, hotProduct)
	}
	if len(items) == 0 {
		utils.BadRequest(c.Ctx, "暂无可生成热点洞察的数据")
		return
	}
	if err := utils.DB.Create(&items).Error; err != nil {
		utils.Error(c.Ctx, http.StatusInternalServerError, "生成热点洞察失败")
		return
	}
	logControllerOperation(c.Ctx, auth, auth.TenantID, "GENERATE_HOT_INSIGHTS", "运营热点", "operation.generate_hot_insights", map[string]interface{}{"count": len(items)}, "success", "")
	utils.Success(c.Ctx, map[string]interface{}{"items": items, "count": len(items)})
}
