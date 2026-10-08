import { reactive } from 'vue'
import zhCNCoreMessages from './i18n.zh-CN.core.js'
import dashboardMessages from './i18n-dashboard-messages.js'
import iotMessages from './i18n-iot-messages.js'
import securityMessages from './i18n-security-messages.js'
import tenantSettingsMessages from './i18n-tenant-settings-messages.js'
import permissionMessages from './i18n-permission-messages.js'
import uiFixMessages from './i18n-ui-fixes-messages.js'
import crmUiFixMessages from './i18n-crm-ui-fixes-messages.js'
import orderFixMessages from './i18n-order-fixes-messages.js'
import errorMessages from './i18n-error-messages.js'
import industryMessages from './i18n-industry-messages.js'
import { localizeGeneratedText } from './i18n-strict-locales.js'

export const localeOptions = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'zh-TW', label: '繁體中文' },
  { value: 'en-US', label: 'English' },
  { value: 'es-ES', label: 'Español' },
  { value: 'fr-FR', label: 'Français' },
  { value: 'de-DE', label: 'Deutsch' },
  { value: 'ja-JP', label: '日本語' },
  { value: 'ko-KR', label: '한국어' },
  { value: 'pt-PT', label: 'Português' },
  { value: 'ar-SA', label: 'العربية' },
  { value: 'ru-RU', label: 'Русский' },
  { value: 'it-IT', label: 'Italiano' }
]

const supportedLocales = new Set(localeOptions.map((item) => item.value))
const localeAliasMap = {
  zh: 'zh-CN',
  'zh-cn': 'zh-CN',
  'zh-hans': 'zh-CN',
  'zh-sg': 'zh-CN',
  'zh-tw': 'zh-TW',
  'zh-hk': 'zh-TW',
  'zh-mo': 'zh-TW',
  'zh-hant': 'zh-TW',
  en: 'en-US',
  'en-us': 'en-US',
  'en-gb': 'en-US',
  es: 'es-ES',
  'es-es': 'es-ES',
  fr: 'fr-FR',
  'fr-fr': 'fr-FR',
  de: 'de-DE',
  'de-de': 'de-DE',
  ja: 'ja-JP',
  'ja-jp': 'ja-JP',
  ko: 'ko-KR',
  'ko-kr': 'ko-KR',
  pt: 'pt-PT',
  'pt-pt': 'pt-PT',
  'pt-br': 'pt-PT',
  ar: 'ar-SA',
  'ar-sa': 'ar-SA',
  ru: 'ru-RU',
  'ru-ru': 'ru-RU',
  it: 'it-IT',
  'it-it': 'it-IT'
}

const rtlLocales = new Set(['ar-SA'])
const loadedLocales = new Set()
const translationCache = new Map()
const externalMessages = {}
const messages = {
  'zh-CN': {
    ...zhCNCoreMessages,
    ...(dashboardMessages['zh-CN'] || {}),
    ...(iotMessages['zh-CN'] || {}),
    ...(securityMessages['zh-CN'] || {}),
    ...(tenantSettingsMessages['zh-CN'] || {}),
    ...(permissionMessages['zh-CN'] || {}),
    ...(uiFixMessages['zh-CN'] || {}),
    ...(crmUiFixMessages['zh-CN'] || {}),
    ...(orderFixMessages['zh-CN'] || {}),
    ...(errorMessages['zh-CN'] || {}),
    ...(industryMessages['zh-CN'] || {})
  }
}

let zhCNFullMessagesPromise = null

export const i18nState = reactive({
  locale: initialLocale()
})

const zhTextToKeys = new Map()
rebuildZhTextToKeys()

export async function ensureLocaleMessages(locale = i18nState.locale) {
  const normalized = normalizeLocale(locale)
  if (!normalized || loadedLocales.has(normalized)) return
  if (normalized === 'zh-CN') {
    const fullMessages = await loadZhCNFullMessages()
    messages['zh-CN'] = {
      ...messages['zh-CN'],
      ...fullMessages,
      ...(dashboardMessages['zh-CN'] || {}),
      ...(iotMessages['zh-CN'] || {}),
      ...(securityMessages['zh-CN'] || {}),
      ...(tenantSettingsMessages['zh-CN'] || {}),
      ...(permissionMessages['zh-CN'] || {}),
      ...(uiFixMessages['zh-CN'] || {}),
      ...(crmUiFixMessages['zh-CN'] || {}),
      ...(orderFixMessages['zh-CN'] || {}),
      ...(errorMessages['zh-CN'] || {}),
      ...(industryMessages['zh-CN'] || {}),
      ...(externalMessages['zh-CN'] || {})
    }
    loadedLocales.add('zh-CN')
    rebuildZhTextToKeys()
    translationCache.clear()
    return
  }
  const localeMessages = await loadLocaleMessages(normalized)
  messages[normalized] = {
    ...(messages[normalized] || {}),
    ...localeMessages,
    ...(dashboardMessages[normalized] || {}),
    ...(iotMessages[normalized] || {}),
    ...(securityMessages[normalized] || {}),
    ...(tenantSettingsMessages[normalized] || {}),
    ...(permissionMessages[normalized] || {}),
    ...(uiFixMessages[normalized] || {}),
    ...(crmUiFixMessages[normalized] || {}),
    ...(orderFixMessages[normalized] || {}),
    ...(errorMessages[normalized] || {}),
    ...(industryMessages[normalized] || {}),
    ...(externalMessages[normalized] || {})
  }
  loadedLocales.add(normalized)
  translationCache.clear()
}

export function applyExternalMessages(pack) {
  const packMessages = pack?.messages || {}
  for (const locale of localeOptions.map((item) => item.value)) {
    const localeMessages = packMessages[locale]
    if (!localeMessages || typeof localeMessages !== 'object') continue
    externalMessages[locale] = {
      ...(externalMessages[locale] || {}),
      ...localeMessages
    }
    if (messages[locale]) {
      messages[locale] = {
        ...(messages[locale] || {}),
        ...localeMessages
      }
    }
  }
  if (packMessages['zh-CN']) rebuildZhTextToKeys()
  translationCache.clear()
}

export function t(key, params = {}) {
  const hasParams = params && Object.keys(params).length > 0
  const locale = i18nState.locale
  const cacheKey = hasParams ? '' : `${locale}:${key}`
  if (cacheKey && translationCache.has(cacheKey)) return translationCache.get(cacheKey)
  const finish = (value) => {
    const translated = hasParams ? interpolate(value, params) : String(value ?? '')
    if (cacheKey) translationCache.set(cacheKey, translated)
    return translated
  }

  const direct = messages[locale]?.[key]
  if (isUsableTranslation(direct)) return finish(direct)
  return finish(resolveFallbackText(key, locale))
}

export function resolveKeyFromZhText(text) {
  const normalized = String(text || '').trim()
  if (!normalized) return ''
  const candidates = zhTextToKeys.get(normalized) || []
  if (!candidates.length) return ''
  const exactDomain = candidates.find((item) => /^domain\.auto\.[a-f0-9]+$/i.test(item))
  return exactDomain || candidates[0] || ''
}

export function normalizeLocale(locale) {
  if (!locale) return ''
  const value = String(locale).trim()
  if (!value) return ''
  if (supportedLocales.has(value)) return value
  return localeAliasMap[value.toLowerCase()] || ''
}

export function currentLocaleOption() {
  return localeOptions.find((item) => item.value === i18nState.locale) || localeOptions[0]
}

export async function setLocale(locale, options = {}) {
  const nextLocale = normalizeLocale(locale)
  if (!nextLocale) return
  await ensureLocaleMessages(nextLocale)
  const syncServer = options.syncServer !== false
  i18nState.locale = nextLocale
  translationCache.clear()
  localStorage.setItem('locale', nextLocale)
  applyDocumentLocale(nextLocale)
  const user = currentUser()
  user.language_pref = nextLocale
  localStorage.setItem('current_user', JSON.stringify(user))
  if (!syncServer || !localStorage.getItem('token')) return
  try {
    const { default: api } = await import('./api.js')
    await api.put('/api/v1/accounts/me/language', { language: nextLocale })
  } catch {
    // Local language preference still applies if the backend save fails.
  }
}

export function currentUser() {
  try {
    return JSON.parse(localStorage.getItem('current_user') || '{}')
  } catch {
    return {}
  }
}

async function loadZhCNFullMessages() {
  if (!zhCNFullMessagesPromise) {
    zhCNFullMessagesPromise = import('./i18n.zh-CN.generated.js').then((module) => module.default)
  }
  return zhCNFullMessagesPromise
}

function loadLocaleMessages(locale) {
  switch (locale) {
    case 'zh-TW':
      return import('./i18n-messages/zh-TW.js').then((module) => module.default)
    case 'en-US':
      return import('./i18n-messages/en-US.js').then((module) => module.default)
    case 'es-ES':
      return import('./i18n-messages/es-ES.js').then((module) => module.default)
    case 'fr-FR':
      return import('./i18n-messages/fr-FR.js').then((module) => module.default)
    case 'de-DE':
      return import('./i18n-messages/de-DE.js').then((module) => module.default)
    case 'ja-JP':
      return import('./i18n-messages/ja-JP.js').then((module) => module.default)
    case 'ko-KR':
      return import('./i18n-messages/ko-KR.js').then((module) => module.default)
    case 'pt-PT':
      return import('./i18n-messages/pt-PT.js').then((module) => module.default)
    case 'ar-SA':
      return import('./i18n-messages/ar-SA.js').then((module) => module.default)
    case 'ru-RU':
      return import('./i18n-messages/ru-RU.js').then((module) => module.default)
    case 'it-IT':
      return import('./i18n-messages/it-IT.js').then((module) => module.default)
    default:
      return Promise.resolve({})
  }
}

function rebuildZhTextToKeys() {
  zhTextToKeys.clear()
  for (const [key, value] of Object.entries(messages['zh-CN'] || {})) {
    if (isAutoDomainKey(key)) continue
    const text = String(value || '').trim()
    if (!text || !hasChineseText(text)) continue
    const list = zhTextToKeys.get(text) || []
    list.push(key)
    zhTextToKeys.set(text, list)
  }
}

function initialLocale() {
  const stored = normalizeLocale(localStorage.getItem('locale'))
  if (stored) {
    applyDocumentLocale(stored)
    return stored
  }
  const userLocale = normalizeLocale(currentUser().language_pref)
  if (userLocale) {
    applyDocumentLocale(userLocale)
    return userLocale
  }
  const browser = detectBrowserLocale()
  applyDocumentLocale(browser)
  return browser
}

function detectBrowserLocale() {
  const candidates = Array.isArray(navigator.languages) && navigator.languages.length
    ? navigator.languages
    : [navigator.language]
  for (const candidate of candidates) {
    const normalized = normalizeLocale(candidate)
    if (normalized) return normalized
  }
  return 'zh-CN'
}

function applyDocumentLocale(locale) {
  if (typeof document === 'undefined') return
  document.documentElement.lang = locale
  document.documentElement.dir = rtlLocales.has(locale) ? 'rtl' : 'ltr'
}

function resolveFallbackText(key, locale) {
  const zhValue = messages['zh-CN']?.[key]
  if (isUsableTranslation(zhValue)) {
    if (locale === 'zh-CN') return String(zhValue)
    return localizeGeneratedText(zhValue, locale, key) || humanizeMissingKey(key, locale)
  }
  return humanizeMissingKey(key, locale) || missingTranslation(locale)
}

function missingTranslation(locale) {
  return messages[locale]?.['common.message.translationMissing'] ||
    messages['zh-CN']?.['common.message.translationMissing'] ||
    '翻译缺失'
}

function humanizeMissingKey(key, locale) {
  const raw = String(key || '').trim()
  if (!raw) return ''
  const cleaned = raw
    .replace(/^domain\.auto\.[a-f0-9]+$/i, '')
    .replace(/^(domain|navDesc|nav|menuGroup|app|campaign|im)\./, '')
    .replace(/[._-]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  if (!cleaned) return ''
  if (locale === 'zh-CN' || locale === 'zh-TW') return cleaned
  return cleaned.replace(/\b\w/g, (letter) => letter.toUpperCase())
}

function interpolate(value, params = {}) {
  return String(value ?? '').replace(/\{([A-Za-z0-9_]+)\}/g, (match, name) => (
    params[name] === undefined || params[name] === null ? match : String(params[name])
  ))
}

function isUsableTranslation(value) {
  if (value === undefined || value === null) return false
  const text = String(value).trim()
  if (!text) return false
  if (isAutoDomainKey(text)) return false
  if (isGeneratedPlaceholder(text)) return false
  return !isGeneratedFragment(text)
}

function isAutoDomainKey(value) {
  return typeof value === 'string' && value.length === 22 && value.startsWith('domain.auto.')
}

function isGeneratedPlaceholder(value) {
  const text = String(value || '').trim()
  return /^(CRM interface text|Text unavailable|Missing translation|文字未設定|翻译缺失|翻譯缺失)(\s+[a-f0-9]{4})?$/i.test(text)
}

function isGeneratedFragment(value) {
  return /&&|<\/?|v-model|pagination|Icon icon|completionWarnings|class=|plain>|{{|}}|\\+$/.test(String(value || ''))
}

function hasChineseText(value) {
  return /[\u3400-\u9FFF]/.test(String(value || ''))
}
