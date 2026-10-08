package controllers

import (
	"ai-crm/models"
	"strings"

	"gorm.io/gorm"
)

var customerCountryPayloadKeys = []string{
	"country",
	"customer_country",
	"company_country",
	"shipping_country",
	"billing_country",
	"region_country",
	"market_country",
	"country_name",
	"region",
	"headquarters",
	"market",
	"markets",
	"markets_text",
	"hq_country",
}

func normalizeCustomerChainObjectTx(tx *gorm.DB, auth authContext, target *models.CRMObject, sources ...models.CRMObject) error {
	if target == nil {
		return nil
	}
	payload := payloadMap(target.PayloadJSON)
	sourcePayloads := make([]map[string]interface{}, 0, len(sources))
	var sourceCompany models.CRMObject
	var sourceContact models.CRMObject
	companyID := firstPositiveUint(target.CompanyID, uintFromValue(payload["company_id"]))
	contactID := firstPositiveUint(target.ContactID, uintFromValue(payload["contact_id"]))
	for _, source := range sources {
		sourcePayload := payloadMap(source.PayloadJSON)
		sourcePayloads = append(sourcePayloads, sourcePayload)
		if source.Domain == "contacts" && source.ObjectType == "company" && source.ID > 0 {
			sourceCompany = source
			companyID = firstPositiveUint(companyID, source.ID)
		}
		if source.Domain == "contacts" && source.ObjectType == "contact" && source.ID > 0 {
			sourceContact = source
			contactID = firstPositiveUint(contactID, source.ID)
		}
		companyID = firstPositiveUint(companyID, source.CompanyID, uintFromValue(sourcePayload["company_id"]))
		contactID = firstPositiveUint(contactID, source.ContactID, uintFromValue(sourcePayload["contact_id"]))
	}
	if sourceCompany.ID == 0 && companyID > 0 && tx != nil && auth.TenantID > 0 {
		_ = tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "contacts", "company", companyID).First(&sourceCompany).Error
	}
	if sourceContact.ID == 0 && contactID > 0 && tx != nil && auth.TenantID > 0 {
		_ = tx.Where("tenant_id = ? AND domain = ? AND object_type = ? AND id = ?", auth.TenantID, "contacts", "contact", contactID).First(&sourceContact).Error
	}

	if companyID > 0 {
		target.CompanyID = companyID
		payload["company_id"] = companyID
	}
	if contactID > 0 {
		target.ContactID = contactID
		payload["contact_id"] = contactID
	}
	companyName := strings.TrimSpace(sourceCompany.Name)
	if companyName == "" {
		companyName = firstPayloadTextFromPayloads(append([]map[string]interface{}{payload}, sourcePayloads...), "company", "company_name", "customer", "customer_name", "account_name")
	}
	if companyName != "" {
		payload["company"] = companyName
		payload["company_name"] = companyName
		if strings.TrimSpace(firstPayloadText(payload, "customer", "customer_name", "account_name")) == "" {
			payload["customer"] = companyName
			payload["customer_name"] = companyName
		}
	}
	contactName := strings.TrimSpace(sourceContact.Name)
	if contactName == "" {
		contactName = firstPayloadTextFromPayloads(append([]map[string]interface{}{payload}, sourcePayloads...), "contact_name", "contact", "contact_person", "buyer_name")
	}
	if contactName != "" {
		payload["contact_name"] = contactName
		payload["contact"] = contactName
	}
	country := customerCountryFromPayload(payloadMap(sourceCompany.PayloadJSON))
	if country == "" {
		country = customerCountryFromPayloads(append([]map[string]interface{}{payload}, sourcePayloads...))
	}
	if country != "" {
		payload["country"] = country
		payload["customer_country"] = country
	}
	target.PayloadJSON = toJSONString(payload)
	return nil
}

func ensureCustomerChainRelationsTx(tx *gorm.DB, auth authContext, object models.CRMObject) error {
	if tx == nil || object.ID == 0 || object.Domain == "" {
		return nil
	}
	if object.CompanyID > 0 {
		if err := ensureCRMRelationTx(tx, auth, "contacts", object.CompanyID, object.Domain, object.ID, "linked_to"); err != nil {
			return err
		}
	}
	if object.ContactID > 0 {
		if err := ensureCRMRelationTx(tx, auth, "contacts", object.ContactID, object.Domain, object.ID, "linked_to"); err != nil {
			return err
		}
	}
	if object.CompanyID > 0 && object.ContactID > 0 {
		if err := ensureCRMRelationTx(tx, auth, "contacts", object.CompanyID, "contacts", object.ContactID, "company_contact"); err != nil {
			return err
		}
	}
	return nil
}

func supportsCustomerChainDomain(domain string, objectType string) bool {
	if domain == "campaigns" && objectType == "campaign" {
		return true
	}
	if domain == "leads" && objectType == "lead" {
		return true
	}
	if domain == "opportunities" && (objectType == "opportunity" || objectType == "quote") {
		return true
	}
	if domain == "deals" && objectType == "closed_deal" {
		return true
	}
	if domain == "orders" && objectType == "order" {
		return true
	}
	return false
}

func firstPayloadTextFromPayloads(payloads []map[string]interface{}, keys ...string) string {
	for _, payload := range payloads {
		if text := firstPayloadText(payload, keys...); text != "" {
			return text
		}
	}
	return ""
}

func customerCountryFromPayload(payload map[string]interface{}) string {
	return firstPayloadText(payload, customerCountryPayloadKeys...)
}

func customerCountryFromPayloads(payloads []map[string]interface{}) string {
	for _, payload := range payloads {
		if country := customerCountryFromPayload(payload); country != "" {
			return country
		}
	}
	return ""
}
