export const currencyOptions = [
  { value: 'CNY', label: 'CNY' },
  { value: 'USD', label: 'USD' },
  { value: 'EUR', label: 'EUR' },
  { value: 'GBP', label: 'GBP' },
  { value: 'JPY', label: 'JPY' },
  { value: 'KRW', label: 'KRW' },
  { value: 'HKD', label: 'HKD' },
  { value: 'TWD', label: 'TWD' },
  { value: 'SGD', label: 'SGD' },
  { value: 'AUD', label: 'AUD' },
  { value: 'CAD', label: 'CAD' },
  { value: 'AED', label: 'AED' }
]

export const currencyCodes = currencyOptions.map((item) => item.value)

export const DEFAULT_CURRENCY = 'USD'

export const currencySymbols = {
  CNY: '¥',
  USD: '$',
  EUR: '€',
  GBP: '£',
  JPY: '¥',
  KRW: '₩',
  HKD: 'HK$',
  TWD: 'NT$',
  SGD: 'S$',
  AUD: 'A$',
  CAD: 'C$',
  AED: 'AED '
}

export const defaultExchangeRates = {
  USD: 1,
  CNY: 0.14,
  EUR: 1.08,
  GBP: 1.26,
  JPY: 0.0069,
  KRW: 0.00072,
  HKD: 0.13,
  TWD: 0.031,
  SGD: 0.74,
  AUD: 0.65,
  CAD: 0.72,
  AED: 0.27
}

export function normalizeCurrencyCode(currency, fallback = DEFAULT_CURRENCY) {
  const value = String(currency || '').trim().toUpperCase()
  if (/^[A-Z]{3}$/.test(value)) return value
  return fallback
}

export function formatCurrencyAmount(amount, currency = DEFAULT_CURRENCY, locale = undefined, options = {}) {
  const code = normalizeCurrencyCode(currency)
  const value = Number(amount || 0)
  const maximumFractionDigits = options.maximumFractionDigits ?? (Number.isInteger(value) ? 0 : 2)
  const formatterOptions = {
    style: 'currency',
    currency: code,
    currencyDisplay: options.currencyDisplay || 'symbol',
    minimumFractionDigits: options.minimumFractionDigits ?? 0,
    maximumFractionDigits
  }
  if (options.notation) formatterOptions.notation = options.notation
  if (options.compactDisplay) formatterOptions.compactDisplay = options.compactDisplay
  try {
    return new Intl.NumberFormat(locale || undefined, formatterOptions).format(value)
  } catch {
    const symbol = currencySymbols[code] || `${code} `
    return `${symbol}${value.toLocaleString(locale || undefined, {
      minimumFractionDigits: formatterOptions.minimumFractionDigits,
      maximumFractionDigits
    })}`
  }
}
