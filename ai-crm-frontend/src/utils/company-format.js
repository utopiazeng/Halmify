import api from '../api'
import { DEFAULT_CURRENCY, currencySymbols, formatCurrencyAmount, normalizeCurrencyCode } from './currency'

const COMPANY_SETTINGS_STORAGE_KEY = 'tenant_company_settings'

export const DEFAULT_COMPANY_SETTINGS = {
  timezone: 'Asia/Shanghai',
  currency: DEFAULT_CURRENCY,
  date_format: 'YYYY-MM-DD'
}

export const CURRENCY_SYMBOLS = currencySymbols

export function parseCompanySettingValue(value) {
  if (!value) return {}
  if (typeof value === 'string') {
    try {
      return JSON.parse(value || '{}')
    } catch {
      return {}
    }
  }
  if (typeof value === 'object') return value
  return {}
}

export async function fetchCompanySettings() {
  try {
    const res = await api.get('/api/v1/prd/settings', { params: { scope: 'company', key: 'enterprise' } })
    const raw = res.data?.items?.[0]?.value_json
    const value = parseCompanySettingValue(raw)
    const settings = {
      timezone: value.timezone || DEFAULT_COMPANY_SETTINGS.timezone,
      currency: normalizeCurrencyCode(value.currency, DEFAULT_COMPANY_SETTINGS.currency),
      date_format: value.date_format || DEFAULT_COMPANY_SETTINGS.date_format
    }
    cacheCompanySettings(settings)
    return settings
  } catch {
    return cachedCompanySettings()
  }
}

export async function saveCompanySettings(settings = {}) {
  const current = await fetchCompanySettings()
  const next = {
    ...current,
    ...settings,
    currency: normalizeCurrencyCode(settings.currency || current.currency, DEFAULT_COMPANY_SETTINGS.currency)
  }
  await api.post('/api/v1/prd/settings', { scope: 'company', key: 'enterprise', value: next })
  cacheCompanySettings(next)
  return next
}

export function cachedCompanySettings() {
  try {
    const raw = JSON.parse(localStorage.getItem(COMPANY_SETTINGS_STORAGE_KEY) || '{}')
    return {
      timezone: raw.timezone || DEFAULT_COMPANY_SETTINGS.timezone,
      currency: normalizeCurrencyCode(raw.currency, DEFAULT_COMPANY_SETTINGS.currency),
      date_format: raw.date_format || DEFAULT_COMPANY_SETTINGS.date_format
    }
  } catch {
    return { ...DEFAULT_COMPANY_SETTINGS }
  }
}

export function cacheCompanySettings(settings = {}) {
  const next = {
    timezone: settings.timezone || DEFAULT_COMPANY_SETTINGS.timezone,
    currency: normalizeCurrencyCode(settings.currency, DEFAULT_COMPANY_SETTINGS.currency),
    date_format: settings.date_format || DEFAULT_COMPANY_SETTINGS.date_format
  }
  try {
    localStorage.setItem(COMPANY_SETTINGS_STORAGE_KEY, JSON.stringify(next))
    window.dispatchEvent(new CustomEvent('tenant-company-settings-updated', { detail: next }))
  } catch {
    // localStorage may be unavailable in private or embedded contexts.
  }
  return next
}

export function defaultBusinessCurrency() {
  return cachedCompanySettings().currency
}

export async function loadDefaultBusinessCurrency() {
  const settings = await fetchCompanySettings()
  return settings.currency
}

export function formatAmountByCompany(amount, recordCurrency, settings = DEFAULT_COMPANY_SETTINGS) {
  const currency = normalizeCurrencyCode(recordCurrency || settings.currency, defaultBusinessCurrency())
  return formatCurrencyAmount(amount, currency)
}

export function formatTimeByCompany(value, settings = DEFAULT_COMPANY_SETTINGS) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value).replace('T', ' ').slice(0, 19)
  const timezone = settings.timezone || DEFAULT_COMPANY_SETTINGS.timezone
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: timezone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  }).formatToParts(date).reduce((acc, item) => {
    acc[item.type] = item.value
    return acc
  }, {})

  if (settings.date_format === 'DD/MM/YYYY') return `${parts.day}/${parts.month}/${parts.year} ${parts.hour}:${parts.minute}`
  return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}`
}
