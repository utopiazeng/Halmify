package controllers

import (
	"encoding/json"
	"os"
	"strings"

	"ai-crm/utils"

	"github.com/beego/beego/v2/server/web"
)

type PublicConfigController struct {
	web.Controller
}

type maintenanceNoticePayload struct {
	Visible bool   `json:"visible"`
	Locale  string `json:"locale"`
	Label   string `json:"label"`
	Title   string `json:"title"`
	Text    string `json:"text"`
}

type maintenanceNoticeCopy struct {
	Label string
	Title string
}

var maintenanceNoticeCopies = map[string]maintenanceNoticeCopy{
	"zh-CN": {Label: "维护公告", Title: "CRM 系统停服维护公告"},
	"zh-TW": {Label: "維護公告", Title: "CRM 系統停服維護公告"},
	"en-US": {Label: "Maintenance", Title: "CRM system maintenance notice"},
	"es-ES": {Label: "Mantenimiento", Title: "Aviso de mantenimiento del sistema CRM"},
	"fr-FR": {Label: "Maintenance", Title: "Avis de maintenance du systeme CRM"},
	"de-DE": {Label: "Wartung", Title: "Wartungshinweis fuer das CRM-System"},
	"ja-JP": {Label: "メンテナンス", Title: "CRM システム停止メンテナンスのお知らせ"},
	"ko-KR": {Label: "유지보수", Title: "CRM 시스템 점검 중단 안내"},
	"pt-PT": {Label: "Manutencao", Title: "Aviso de manutencao do sistema CRM"},
	"ar-SA": {Label: "إشعار صيانة", Title: "إشعار توقف صيانة نظام CRM"},
	"ru-RU": {Label: "Обслуживание", Title: "Уведомление о техническом обслуживании CRM"},
	"it-IT": {Label: "Manutenzione", Title: "Avviso di manutenzione del sistema CRM"},
}

var maintenanceNoticeLocaleAliases = map[string]string{
	"zh":    "zh-CN",
	"zh-cn": "zh-CN",
	"zh-tw": "zh-TW",
	"en":    "en-US",
	"en-us": "en-US",
	"es":    "es-ES",
	"es-es": "es-ES",
	"fr":    "fr-FR",
	"fr-fr": "fr-FR",
	"de":    "de-DE",
	"de-de": "de-DE",
	"ja":    "ja-JP",
	"ja-jp": "ja-JP",
	"ko":    "ko-KR",
	"ko-kr": "ko-KR",
	"pt":    "pt-PT",
	"pt-pt": "pt-PT",
	"ar":    "ar-SA",
	"ar-sa": "ar-SA",
	"ru":    "ru-RU",
	"ru-ru": "ru-RU",
	"it":    "it-IT",
	"it-it": "it-IT",
}

func (c *PublicConfigController) MaintenanceNotice() {
	locale := normalizeMaintenanceNoticeLocale(c.GetString("locale"))
	copy := maintenanceNoticeCopies[locale]
	text := resolveMaintenanceNoticeText(readMaintenanceNoticeConfig(), locale)
	utils.Success(c.Ctx, maintenanceNoticePayload{
		Visible: text != "",
		Locale:  locale,
		Label:   copy.Label,
		Title:   copy.Title,
		Text:    text,
	})
}

func readMaintenanceNoticeConfig() string {
	if value := strings.TrimSpace(os.Getenv("CRM_MAINTENANCE_NOTICE")); value != "" {
		return value
	}
	return strings.TrimSpace(web.AppConfig.DefaultString("crm_maintenance_notice", ""))
}

func normalizeMaintenanceNoticeLocale(locale string) string {
	locale = strings.TrimSpace(locale)
	if copy, ok := maintenanceNoticeCopies[locale]; ok && copy.Title != "" {
		return locale
	}
	if mapped := maintenanceNoticeLocaleAliases[strings.ToLower(locale)]; mapped != "" {
		return mapped
	}
	return "zh-CN"
}

func resolveMaintenanceNoticeText(raw string, locale string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	localized := map[string]string{}
	if strings.HasPrefix(raw, "{") && json.Unmarshal([]byte(raw), &localized) == nil {
		return firstMaintenanceNoticeText(localized,
			locale,
			strings.ToLower(locale),
			strings.Split(locale, "-")[0],
			"default",
			"en-US",
			"en",
			"zh-CN",
			"zh",
		)
	}
	return raw
}

func firstMaintenanceNoticeText(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(values[key]); value != "" {
			return value
		}
	}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
