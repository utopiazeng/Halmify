import { t } from '../i18n'

export function apiErrorMessage(error, fallback = t('common.message.loadFailed')) {
  const body = error?.response?.data || error?.data || error || {}
  const params = body.message_params || body.data?.message_params || {}
  const details = detailedErrorMessage(body)
  if (details) return details
  const legacyMessage = body.data?.legacy_message || body.legacy_message
  if (legacyMessage) {
    const localized = localizeBackendText(legacyMessage)
    if (localized) return localized
  }
  const rawMessageKey = body.message_key || body.data?.message_key || keyFromErrorCode(body.error_code || body.data?.error_code)
  const messageKey = normalizeMessageKey(rawMessageKey)
  if (messageKey) {
    const translated = t(messageKey, params)
    if (translated && translated !== messageKey) return translated
  }
  const message = body.message || body.error
  if (message) return localizeBackendText(message)
  if (error?.message) return localizeBackendText(error.message)
  return fallback
}

export function normalizeApiErrorResponse(error) {
  const body = error?.response?.data
  if (!body || typeof body !== 'object') return error
  const message = apiErrorMessage(error, '')
  if (message) body.message = message
  return error
}

export function localizeBackendText(message) {
  const text = String(message || '').trim()
  if (!text) return ''
  const key = legacyMessageKeys[text]
  if (key) return t(key)
  const normalizedKey = text.replace(/\s+/g, '.')
  if (/^[a-z][a-z0-9_]*(\.[a-z0-9_]+)+$/i.test(normalizedKey)) {
    const mappedKey = normalizeMessageKey(normalizedKey)
    const direct = t(mappedKey)
    if (direct && direct !== mappedKey) return direct
    const commonKey = legacyMessageKeys[mappedKey]
    if (commonKey) return t(commonKey)
  }
  return text
}

function normalizeMessageKey(key) {
  const normalized = String(key || '').trim().replace(/\s+/g, '.')
  if (!normalized) return ''
  return backendMessageKeyAliases[normalized] || normalized
}

function keyFromErrorCode(code) {
  const normalized = String(code || '').trim().toUpperCase()
  if (!normalized) return ''
  return errorCodeKeys[normalized] || ''
}

function detailedErrorMessage(body) {
  const candidates = [
    body.data?.details,
    body.data?.errors,
    body.details,
    body.errors
  ]
  for (const candidate of candidates) {
    const text = normalizeDetailMessage(candidate)
    if (text) return text
  }
  return ''
}

function normalizeDetailMessage(value) {
  if (!value) return ''
  if (typeof value === 'string') return localizeBackendText(value)
  if (Array.isArray(value)) {
    return value
      .map((item) => normalizeDetailMessage(item))
      .filter(Boolean)
      .join('；')
  }
  if (typeof value === 'object') {
    const direct = value.message || value.reason || value.error || value.detail
    const field = value.field || value.name || value.key
    if (direct) {
      const text = localizeBackendText(direct)
      return field ? `${field}: ${text}` : text
    }
    return Object.entries(value)
      .map(([key, item]) => {
        const text = normalizeDetailMessage(item)
        return text ? `${key}: ${text}` : ''
      })
      .filter(Boolean)
      .join('；')
  }
  return ''
}

const errorCodeKeys = {
  AUTH_INVALID_CREDENTIALS: 'auth.invalidCredentials',
  AUTH_CAPTCHA_EXPIRED: 'auth.captchaExpired',
  AUTH_CAPTCHA_FAILED: 'auth.captchaFailed',
  AUTH_CAPTCHA_UNAVAILABLE: 'auth.captchaUnavailable',
  AUTH_CAPTCHA_REQUIRED: 'auth.captchaRequired',
  AUTH_LOGIN_PROTECTION_BUSY: 'auth.loginProtectionBusy',
  AUTH_LOGIN_RATE_LIMITED: 'auth.loginRateLimited',
  AUTH_TOKEN_MISSING: 'auth.tokenMissing',
  AUTH_TOKEN_INVALID: 'auth.tokenInvalid',
  AUTH_SESSION_EXPIRED: 'auth.sessionExpired',
  AUTH_DEMO_ACCOUNT_PASSWORD_LOCKED: 'auth.demoAccountPasswordLocked',
  PASSWORD_CHANGE_REQUIRED: 'auth.passwordChangeRequired',
  TENANT_DISABLED: 'login.error.tenantDisabled',
  TENANT_EXPIRED: 'login.error.tenantExpired',
  SECURITY_RISK_BLOCKED: 'security.riskBlocked',
  MFG_CLARIFICATION_ANSWER_LOCKED: 'industry.manufacturing.error.clarificationAnswerLocked',
  MFG_APPROVAL_REMARK_REQUIRED: 'industry.manufacturing.error.approvalRemarkRequired',
  MFG_UNSUPPORTED_RESOURCE: 'industry.manufacturing.error.unsupportedMetricDrilldown',
  COMMON_PERMISSION_DENIED: 'common.message.permissionDenied',
  COMMON_INVALID_REQUEST: 'common.message.invalidRequest',
  COMMON_OPERATION_FAILED: 'common.message.loadFailed'
}

const backendMessageKeyAliases = {
  'common.badRequest': 'common.message.invalidRequest',
  'common.invalidRequest': 'common.message.invalidRequest',
  'common.permissionDenied': 'common.message.permissionDenied',
  'common.forbidden': 'common.message.permissionDenied',
  'common.moduleNotEnabled': 'common.message.moduleNotEnabled',
  'common.createFailed': 'common.message.createFailed',
  'common.updateFailed': 'common.message.updateFailed',
  'common.deleteFailed': 'common.message.deleteFailed'
}

const legacyMessageKeys = {
  '用户名或密码错误': 'auth.invalidCredentials',
  '滑动验证码已失效，请刷新重试': 'auth.captchaExpired',
  '滑动验证未通过，请重试': 'auth.captchaFailed',
  '验证码服务不可用': 'auth.captchaUnavailable',
  '缺少认证令牌': 'auth.tokenMissing',
  '无效的令牌': 'auth.tokenInvalid',
  '登录会话已失效，请重新登录': 'auth.sessionExpired',
  '登录会话不存在，请重新登录': 'auth.sessionExpired',
  '用户不存在或会话无效': 'auth.sessionExpired',
  '首次登录需先修改密码': 'auth.passwordChangeRequired',
  '无权限访问': 'common.message.permissionDenied',
  '请求参数错误': 'common.message.invalidRequest',
  'common.badRequest': 'common.message.invalidRequest',
  'common invalidRequest': 'common.message.invalidRequest',
  'common.invalidRequest': 'common.message.invalidRequest',
  'common.permissionDenied': 'common.message.permissionDenied',
  'common forbidden': 'common.message.permissionDenied',
  'common.forbidden': 'common.message.permissionDenied',
  'common moduleNotEnabled': 'common.message.moduleNotEnabled',
  'common.moduleNotEnabled': 'common.message.moduleNotEnabled',
  'common createFailed': 'common.message.createFailed',
  'common.createFailed': 'common.message.createFailed',
  'common updateFailed': 'common.message.updateFailed',
  'common.updateFailed': 'common.message.updateFailed',
  'common deleteFailed': 'common.message.deleteFailed',
  'common.deleteFailed': 'common.message.deleteFailed'
}
