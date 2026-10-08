import axios from 'axios'
import { confirmDangerAction, hasRecentDeleteConfirmation } from './utils/confirm'
import { i18nState } from './i18n'
import { normalizeApiErrorResponse } from './utils/api-error'

const SYSTEM_UNREAD_LEGACY_URL = '/api/v1/im/system-notifications'
const SYSTEM_UNREAD_COUNT_URL = '/api/v1/im/system-notifications/unread-count'
const SYSTEM_UNREAD_CACHE_TTL_MS = 5 * 60 * 1000
const CRM_OBJECTS_MAX_PAGE_SIZE = 100
let systemUnreadCache = null

const api = axios.create({
  baseURL: import.meta.env.BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

api.interceptors.request.use(async (config) => {
  const token = localStorage.getItem('token')
  config.headers = config.headers || {}
  config.headers['Accept-Language'] = i18nState.locale || 'zh-CN'
  normalizeCRMObjectsPageSize(config)
  if (isSystemUnreadBadgeRequest(config)) {
    const cached = currentSystemUnreadCache(token)
    if (cached || isDocumentHidden()) {
      const body = cached || makeSystemUnreadBody(0)
      config.adapter = () => Promise.resolve({
        data: body,
        status: 200,
        statusText: 'OK',
        headers: { 'x-local-cache': 'system-unread' },
        config,
        request: null
      })
    } else {
      config.url = SYSTEM_UNREAD_COUNT_URL
      config.params = {}
      config.systemUnreadBadgeRequest = true
      config.systemUnreadToken = token || ''
    }
  }
  if (shouldInvalidateSystemUnreadCache(config)) {
    invalidateSystemUnreadCache()
  }
  if (String(config.method || '').toLowerCase() === 'delete' && !config.skipDeleteConfirm && !hasRecentDeleteConfirmation()) {
    await confirmDangerAction()
  }
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  if (config.data instanceof FormData) {
    delete config.headers['Content-Type']
  }
  return config
})

api.interceptors.response.use(
  (response) => {
    if (response.config?.systemUnreadBadgeRequest) {
      systemUnreadCache = {
        body: response.data,
        expiresAt: Date.now() + SYSTEM_UNREAD_CACHE_TTL_MS,
        token: response.config.systemUnreadToken || ''
      }
    }
    if (response.config?.responseType === 'blob') return response.data
    return response.data
  },
  (error) => {
    normalizeApiErrorResponse(error)
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('current_user')
      if (shouldRedirectToLogin(error.config)) {
        window.location.replace(import.meta.env.BASE_URL)
      }
    }
    return Promise.reject(error)
  }
)

function isSystemUnreadBadgeRequest(config) {
  if (String(config.method || 'get').toLowerCase() !== 'get') return false
  const url = String(config.url || '')
  if (urlPath(url) !== SYSTEM_UNREAD_LEGACY_URL) return false
  const params = queryParams(config)
  return Number(params.get('page') || 1) === 1 &&
    Number(params.get('size') || 0) === 1 &&
    !params.has('status')
}

function normalizeCRMObjectsPageSize(config) {
  if (String(config.method || 'get').toLowerCase() !== 'get') return
  if (!/^\/api\/v1\/prd\/[^/]+\/objects$/.test(urlPath(String(config.url || '')))) return
  const params = config.params || {}
  const size = Number(params.size)
  if (!Number.isFinite(size) || size <= CRM_OBJECTS_MAX_PAGE_SIZE) return
  config.params = {
    ...params,
    size: CRM_OBJECTS_MAX_PAGE_SIZE
  }
}

function shouldInvalidateSystemUnreadCache(config) {
  const method = String(config.method || 'get').toLowerCase()
  if (method === 'get') return false
  const path = urlPath(String(config.url || ''))
  return /^\/api\/v1\/prd\/flows\/workspace\/\d+\/read$/.test(path) ||
    /^\/api\/v1\/im\/system-notifications\/\d+\/read$/.test(path) ||
    path === SYSTEM_UNREAD_LEGACY_URL ||
    path === SYSTEM_UNREAD_COUNT_URL
}

function currentSystemUnreadCache(token) {
  if (!systemUnreadCache || systemUnreadCache.token !== (token || '') || systemUnreadCache.expiresAt <= Date.now()) {
    systemUnreadCache = null
    return null
  }
  return systemUnreadCache.body
}

function invalidateSystemUnreadCache() {
  systemUnreadCache = null
}

function makeSystemUnreadBody(unread) {
  return {
    code: 200,
    message: 'success',
    data: {
      unread,
      cached: true,
      updated_at: new Date().toISOString()
    }
  }
}

function queryParams(config) {
  const params = new URLSearchParams()
  const rawURL = String(config.url || '')
  const query = rawURL.includes('?') ? rawURL.slice(rawURL.indexOf('?') + 1) : ''
  new URLSearchParams(query).forEach((value, key) => params.set(key, value))
  Object.entries(config.params || {}).forEach(([key, value]) => {
    if (value !== undefined && value !== null) params.set(key, value)
  })
  return params
}

function urlPath(rawURL) {
  const value = String(rawURL || '')
  try {
    return new URL(value, window.location.origin).pathname
  } catch {
    return value.split('?')[0]
  }
}

function shouldRedirectToLogin(config) {
  const currentPath = window.location.pathname || '/'
  if (currentPath === '/' || currentPath === '/activate-account') return false

  const requestPath = urlPath(String(config?.url || ''))
  if (requestPath === '/api/v1/auth/login') return false
  if (requestPath === '/api/v1/auth/captcha/challenge') return false
  if (requestPath === '/api/v1/auth/captcha/verify') return false
  return true
}

function isDocumentHidden() {
  return typeof document !== 'undefined' && document.hidden
}

export default api
