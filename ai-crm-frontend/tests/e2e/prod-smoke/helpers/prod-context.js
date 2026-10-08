import fs from 'node:fs'
import path from 'node:path'
import { expect } from '@playwright/test'
import { fetchCaptchaToken } from '../../helpers/auth.js'

export const PROD_ROUTES = [
  ['/workspace', '客户主面板'],
  ['/dashboards', '销售大屏'],
  ['/im', '站内消息'],
  ['/campaigns', '市场活动'],
  ['/leads', '线索'],
  ['/companies', '公司'],
  ['/contacts', '联系人'],
  ['/opportunities', '交易商机'],
  ['/deals', '成交记录'],
  ['/orders', '订单'],
  ['/tickets', '服务工单'],
  ['/projects', '客户项目'],
  ['/products', '产品服务'],
  ['/insights', '运营热点'],
  ['/ai-analysis', 'AI 经营分析'],
  ['/storage', '存储空间'],
  ['/system/tenant-settings', '租户设置'],
  ['/system/iot-devices', 'IoT设备'],
  ['/system/user-center', '个人中心'],
  ['/system/user', '用户管理'],
  ['/system/role', '角色管理'],
  ['/system/department', '部门管理'],
  ['/system/post', '岗位管理'],
  ['/system/menu', '菜单管理'],
  ['/system/log', '操作日志'],
  ['/system/login-log', '登录日志'],
  ['/system/security-log', '安全事件'],
  ['/platform/overview', '平台概览'],
  ['/platform/tenants', '租户管理'],
  ['/platform/settings', '平台配置'],
  ['/platform/plans', '套餐管理'],
  ['/platform/ai-analysis', 'AI 分析账务'],
  ['/platform/expiring', '到期提醒'],
  ['/platform/runtime', '运行监控'],
  ['/platform/iot-firmwares', 'IoT 固件'],
  ['/platform/messages', '平台消息']
]

export const CRM_DOMAIN_TYPES = {
  contacts: { company: 'active', contact: 'active', interaction: 'active' },
  campaigns: { campaign: 'draft' },
  leads: { lead: 'new' },
  opportunities: { opportunity: 'initial_contact', quote: 'draft' },
  deals: { closed_deal: 'closed_won' },
  orders: { order: 'draft' },
  tickets: { ticket: 'new' },
  projects: { project: 'planning', milestone: 'planning', deliverable: 'planning' },
  products: { product: 'on_sale', service_package: 'on_sale', pricing_rule: 'testing', inventory: 'on_sale' },
  workspace: { task: 'open', mail_notification: 'open' },
  insights: { faq_cluster: 'open', deal_risk: 'open' }
}

const reportDir = path.resolve('test-results/prod-smoke')
const runStamp = new Date().toISOString().replace(/\D/g, '').slice(0, 14)

export function makeRunContext() {
  const prefix = process.env.E2E_SMOKE_PREFIX || `E2E_SMOKE_${runStamp}`
  return {
    prefix,
    startedAt: new Date().toISOString(),
    user: null,
    token: '',
    created: {
      objects: [],
      roles: [],
      departments: [],
      posts: [],
      menus: [],
      users: [],
      iotDevices: [],
      files: []
    },
    coverage: [],
    failures: [],
    blocked: [],
    cleanup: [],
    apiErrors: []
  }
}

export async function loginProd(page, ctx) {
  const username = process.env.E2E_USERNAME
  const password = process.env.E2E_PASSWORD
  if (!username || !password) {
    throw new Error('E2E_USERNAME and E2E_PASSWORD are required for production smoke tests')
  }
  const captchaToken = await fetchCaptchaToken(page.request)
  const loginResp = await page.request.post('/api/v1/auth/login', {
    data: { username, password, captcha_token: captchaToken }
  })
  if (!loginResp.ok()) {
    const body = await loginResp.json().catch(() => ({}))
    throw new Error(`login failed for ${username}: status=${loginResp.status()} message=${body?.message || 'unknown'}`)
  }
  const loginBody = await loginResp.json()
  const token = loginBody?.data?.token
  const user = loginBody?.data?.user
  expect(token).toBeTruthy()
  expect(user?.id).toBeTruthy()
  user.language_pref = 'zh-CN'
  ctx.user = user
  ctx.token = token
  await installAuthStorage(page, ctx)
  return { token, user }
}

export async function installAuthStorage(page, ctx) {
  await page.context().addInitScript(({ tokenValue, userValue }) => {
    localStorage.setItem('token', tokenValue)
    localStorage.setItem('current_user', JSON.stringify(userValue || {}))
    localStorage.setItem('locale', 'zh-CN')
  }, { tokenValue: ctx.token, userValue: ctx.user })
  await page.goto('/workspace', { waitUntil: 'domcontentloaded' })
}

export function attachFailureWatchers(page, ctx) {
  page.on('pageerror', (error) => {
    ctx.failures.push({ type: 'pageerror', message: error.message })
  })
  page.on('console', (msg) => {
    if (msg.type() !== 'error') return
    const text = msg.text()
    if (/favicon|ResizeObserver loop/i.test(text)) return
    const location = msg.location()
    ctx.failures.push({
      type: 'console_error',
      message: text,
      url: location.url,
      line: location.lineNumber
    })
  })
  page.on('response', (response) => {
    const url = response.url()
    const status = response.status()
    if (status < 500) return
    if (!/\/api\/v1\//.test(url)) return
    ctx.failures.push({ type: 'api_5xx', status, url })
  })
}

export function createAPI(request, ctx) {
  async function call(method, url, options = {}) {
    const headers = {
      Accept: 'application/json',
      'Accept-Language': 'zh-CN',
      ...(ctx.token ? { Authorization: `Bearer ${ctx.token}` } : {}),
      ...(options.headers || {})
    }
    const resp = await request[method](url, { ...options, headers })
    const contentType = resp.headers()['content-type'] || ''
    let body = null
    if (contentType.includes('application/json')) {
      body = await resp.json().catch(() => null)
    } else {
      const text = await resp.text().catch(() => '')
      body = text ? { raw: text.slice(0, 500) } : null
    }
    const record = { method: method.toUpperCase(), url, status: resp.status(), body }
    if (resp.status() >= 500 || resp.status() === 401) {
      ctx.apiErrors.push(record)
    }
    return { resp, body, record, ok: resp.ok() }
  }
  return {
    get: (url, options) => call('get', url, options),
    post: (url, data, options = {}) => call('post', url, { ...options, data }),
    multipart: (url, multipart, options = {}) => call('post', url, { ...options, multipart }),
    put: (url, data, options = {}) => call('put', url, { ...options, data }),
    patch: (url, data, options = {}) => call('patch', url, { ...options, data }),
    delete: (url, options) => call('delete', url, options)
  }
}

export function dataOf(result) {
  return result?.body?.data ?? result?.body
}

export function itemsOf(result) {
  const data = dataOf(result)
  if (Array.isArray(data)) return data
  if (Array.isArray(data?.items)) return data.items
  return []
}

export async function expectAPI(result, label, allow = []) {
  if (result.ok) return dataOf(result)
  if (allow.includes(result.resp.status())) {
    return null
  }
  const message = result.body?.message || result.body?.raw || 'unknown'
  throw new Error(`${label} failed: status=${result.resp.status()} message=${message}`)
}

export function markCoverage(ctx, module, action, status = 'passed', detail = '') {
  ctx.coverage.push({ module, action, status, detail, at: new Date().toISOString() })
}

export function markBlocked(ctx, module, action, detail) {
  ctx.blocked.push({ module, action, detail, at: new Date().toISOString() })
  markCoverage(ctx, module, action, 'blocked', detail)
}

export function smokePayload(ctx, domain, objectType, index = 1, extra = {}) {
  const base = `${ctx.prefix}_${domain}_${objectType}_${index}`
  const payload = {
    e2e_smoke: true,
    smoke_prefix: ctx.prefix,
    external_ref: base,
    remark: `${base} automated production smoke test data`
  }
  if (domain === 'contacts' && objectType === 'company') {
    Object.assign(payload, {
      company_name: base,
      industry: 'Automation',
      grade: 'normal',
      source: 'e2e_smoke'
    })
  }
  if (domain === 'contacts' && objectType === 'contact') {
    Object.assign(payload, {
      contact_name: base,
      email: `${base.toLowerCase()}@example.test`,
      phone: '13800000000',
      company: extra.companyName || `${ctx.prefix}_company`
    })
  }
  if (domain === 'leads') {
    Object.assign(payload, {
      company_name: `${ctx.prefix}_lead_company`,
      contact_name: `${ctx.prefix}_lead_contact`,
      email: `${base.toLowerCase()}@example.test`,
      phone: '13900000000',
      source_channel: 'e2e_smoke',
      utm_source: 'playwright',
      intent_product: 'Halmify CRM'
    })
  }
  if (domain === 'opportunities') {
    Object.assign(payload, {
      stage: 'initial_contact',
      probability: 20,
      next_activity_date: '2026-06-30',
      company_name: extra.companyName || `${ctx.prefix}_company`,
      contact_name: extra.contactName || `${ctx.prefix}_contact`,
      line_items: []
    })
  }
  if (domain === 'deals') {
    Object.assign(payload, {
      signed_at: '2026-06-22',
      contract_no: `${base}_CONTRACT`,
      invoice_no: `${base}_INV`,
      line_items: []
    })
  }
  if (domain === 'orders') {
    Object.assign(payload, {
      payment_status: 'pending',
      shipment_no: '',
      invoice_type: '',
      line_items: [{ product_id: 0, quantity: 1, unit_price: 1 }]
    })
  }
  if (domain === 'tickets') {
    Object.assign(payload, {
      category: 'e2e_smoke',
      priority: 'normal',
      customer_grade: 'normal',
      public_reply: '',
      internal_note: 'created by production smoke'
    })
  }
  if (domain === 'projects') {
    Object.assign(payload, {
      manager: ctx.user?.name || ctx.user?.username || 'e2e',
      milestones: [],
      progress: 0,
      customer_visible: false
    })
  }
  if (domain === 'products') {
    Object.assign(payload, {
      sku: base,
      unit: '套',
      tax_rate: 0,
      stock: 10,
      stock_threshold: 1,
      category: 'E2E'
    })
  }
  return payload
}

export async function createCRMObject(api, ctx, domain, objectType, options = {}) {
  const status = options.status || CRM_DOMAIN_TYPES[domain]?.[objectType] || 'active'
  const index = ctx.created.objects.length + 1
  const name = options.name || `${ctx.prefix}_${domain}_${objectType}_${index}`
  const payload = {
    ...smokePayload(ctx, domain, objectType, index, options.payload || {}),
    ...(options.payload || {})
  }
  const result = await api.post(`/api/v1/prd/${domain}/objects`, {
    object_type: objectType,
    name,
    code: options.code || `${ctx.prefix}_${domain}_${objectType}_${index}`,
    status,
    owner_id: options.ownerID || ctx.user?.id || 0,
    amount: options.amount || 100,
    currency: options.currency || 'USD',
    score: options.score || 10,
    tags: [ctx.prefix, 'prod_smoke'],
    payload
  })
  const data = await expectAPI(result, `create ${domain}/${objectType}`, [403])
  if (!data) {
    markBlocked(ctx, domain, `create ${objectType}`, `permission denied`)
    return null
  }
  ctx.created.objects.push({ domain, id: data.id, name: data.name, objectType })
  return data
}

export async function cleanupCreatedData(api, ctx) {
  await discoverPrefixedCRMObjects(api, ctx)
  for (const device of [...ctx.created.iotDevices].reverse()) {
    const result = await api.delete(`/api/v1/iot/devices/${device.id}`)
    ctx.cleanup.push({ type: 'iot_device', id: device.id, ok: result.ok, status: result.resp.status() })
  }
  for (const user of [...ctx.created.users].reverse()) {
    const result = await api.delete(`/api/v1/accounts/users/${user.id}`)
    ctx.cleanup.push({ type: 'user', id: user.id, ok: result.ok, status: result.resp.status() })
  }
  for (const menu of [...ctx.created.menus].reverse()) {
    const result = await api.delete(`/api/v1/system/menus/${menu.id}`)
    ctx.cleanup.push({ type: 'menu', id: menu.id, ok: result.ok, status: result.resp.status() })
  }
  for (const role of [...ctx.created.roles].reverse()) {
    const result = await api.delete(`/api/v1/rbac/roles/${role.id}`)
    ctx.cleanup.push({ type: 'role', id: role.id, ok: result.ok, status: result.resp.status() })
  }
  for (const post of [...ctx.created.posts].reverse()) {
    const result = await api.delete(`/api/v1/system/posts/${post.id}`)
    ctx.cleanup.push({ type: 'post', id: post.id, ok: result.ok, status: result.resp.status() })
  }
  for (const department of [...ctx.created.departments].reverse()) {
    const result = await api.delete(`/api/v1/system/departments/${department.id}`)
    ctx.cleanup.push({ type: 'department', id: department.id, ok: result.ok, status: result.resp.status() })
  }
  for (const object of [...ctx.created.objects].reverse()) {
    if (object.domain === 'leads') {
      await markLeadInvalidForCleanup(api, object)
    }
    const result = await api.delete(`/api/v1/prd/${object.domain}/objects/${object.id}`)
    ctx.cleanup.push({ type: object.domain, id: object.id, ok: result.ok, status: result.resp.status() })
  }
  for (const file of [...ctx.created.files].reverse()) {
    const result = await api.delete(`/api/v1/files/${file.file_id || file.id}`)
    ctx.cleanup.push({ type: 'file', id: file.file_id || file.id, ok: result.ok, status: result.resp.status() })
  }
}

async function discoverPrefixedCRMObjects(api, ctx) {
  const known = new Set(ctx.created.objects.map((item) => `${item.domain}:${item.id}`))
  for (const domain of Object.keys(CRM_DOMAIN_TYPES)) {
    const result = await api.get(`/api/v1/prd/${domain}/objects?page=1&size=100&keyword=${encodeURIComponent(ctx.prefix)}`)
    if (!result.ok) continue
    for (const item of itemsOf(result)) {
      if (!item?.id || known.has(`${domain}:${item.id}`)) continue
      const payload = objectPayload(item)
      const name = String(item.name || '')
      const code = String(item.code || '')
      const tagged = String(item.tags || '').includes(ctx.prefix)
      const payloadTagged = String(payload.smoke_prefix || payload.external_ref || '').includes(ctx.prefix)
      if (!name.includes(ctx.prefix) && !code.includes(ctx.prefix) && !tagged && !payloadTagged) continue
      ctx.created.objects.push({ domain, id: item.id, name: item.name, objectType: item.object_type })
      known.add(`${domain}:${item.id}`)
    }
  }
}

async function markLeadInvalidForCleanup(api, object) {
  const current = await api.get(`/api/v1/prd/leads/objects/${object.id}`)
  if (!current.ok) return
  const data = dataOf(current)
  if (!data || data.status === 'invalid') return
  await api.put(`/api/v1/prd/leads/objects/${object.id}`, {
    object_type: data.object_type || object.objectType || 'lead',
    name: data.name || object.name || `cleanup_${object.id}`,
    code: data.code || '',
    status: 'invalid',
    owner_id: data.owner_id || 0,
    department_id: data.department_id || 0,
    company_id: data.company_id || 0,
    contact_id: data.contact_id || 0,
    amount: data.amount || 0,
    currency: data.currency || 'USD',
    score: data.score || 0,
    tags: tagsArray(data.tags),
    payload: objectPayload(data)
  })
}

function objectPayload(item) {
  if (item?.payload && typeof item.payload === 'object') return item.payload
  if (item?.payload_json && typeof item.payload_json === 'string') {
    try {
      return JSON.parse(item.payload_json)
    } catch {
      return {}
    }
  }
  return {}
}

function tagsArray(tags) {
  if (Array.isArray(tags)) return tags
  if (typeof tags === 'string') {
    return tags.split(',').map((item) => item.trim()).filter(Boolean)
  }
  return []
}

export async function assertRouteHealthy(page, ctx, routePath, title) {
  const errorsBefore = ctx.failures.length
  let response = null
  try {
    response = await page.goto(routePath, { waitUntil: 'domcontentloaded', timeout: 45000 })
  } catch (error) {
    ctx.failures.push({ type: 'navigation_timeout', url: routePath, message: error.message })
    markCoverage(ctx, 'navigation', routePath, 'failed', `navigation timeout: ${title}`)
    return
  }
  const status = response?.status() || 0
  if ([401, 403, 404].includes(status)) {
    markBlocked(ctx, 'navigation', routePath, `HTTP ${status}`)
    return
  }
  expect(status, `${routePath} status`).toBeLessThan(500)
  const shell = page.locator('.app-main, main, .system-page, .domain-page, .platform-page, .im-page').first()
  await expect(shell, `${routePath} main shell`).toBeVisible({ timeout: 15000 })
  const heading = page.getByRole('heading', { name: title }).first()
  if (await heading.count()) {
    await expect(heading).toBeVisible({ timeout: 10000 })
  }
  const visibleFatal = page.getByText(/401|403|500|Internal Server Error|接口异常|登录已过期|无权限/).first()
  if (await visibleFatal.count()) {
    const text = await visibleFatal.textContent().catch(() => '')
    if (/500|Internal Server Error|接口异常|登录已过期/.test(text || '')) {
      ctx.failures.push({ type: 'visible_fatal', url: routePath, message: text })
      markCoverage(ctx, 'navigation', routePath, 'failed', `visible fatal: ${text}`)
      return
    }
  }
  const newErrors = ctx.failures.slice(errorsBefore)
  if (newErrors.length > 0) {
    markCoverage(ctx, 'navigation', routePath, 'failed', `frontend/API fatal errors: ${JSON.stringify(newErrors.slice(0, 3))}`)
    return
  }
  markCoverage(ctx, 'navigation', routePath, 'passed', title)
}

export function writeSmokeReport(ctx) {
  fs.mkdirSync(reportDir, { recursive: true })
  const jsonPath = path.join(reportDir, `${ctx.prefix}-summary.json`)
  const mdPath = path.join(reportDir, `${ctx.prefix}-summary.md`)
  const endedAt = new Date().toISOString()
  const summary = {
    ...ctx,
    endedAt,
    counts: {
      coverage: ctx.coverage.length,
      failures: ctx.failures.length,
      blocked: ctx.blocked.length,
      cleanup: ctx.cleanup.length,
      cleanupFailed: ctx.cleanup.filter((item) => !item.ok && item.status !== 404).length,
      apiErrors: ctx.apiErrors.length
    }
  }
  fs.writeFileSync(jsonPath, JSON.stringify(summary, null, 2))
  const lines = [
    `# Production CRM Smoke Report`,
    ``,
    `- Prefix: \`${ctx.prefix}\``,
    `- Started: ${ctx.startedAt}`,
    `- Ended: ${endedAt}`,
    `- User: ${ctx.user?.username || ctx.user?.name || 'unknown'}`,
    `- Coverage items: ${summary.counts.coverage}`,
    `- Blocked items: ${summary.counts.blocked}`,
    `- Fatal failures: ${summary.counts.failures}`,
    `- API errors: ${summary.counts.apiErrors}`,
    `- Cleanup records: ${summary.counts.cleanup}`,
    `- Cleanup failed: ${summary.counts.cleanupFailed}`,
    ``,
    `## Coverage`,
    ...ctx.coverage.map((item) => `- ${item.status}: ${item.module} / ${item.action}${item.detail ? ` - ${item.detail}` : ''}`),
    ``,
    `## Blocked`,
    ...(ctx.blocked.length ? ctx.blocked.map((item) => `- ${item.module} / ${item.action}: ${item.detail}`) : ['- none']),
    ``,
    `## Failures`,
    ...(ctx.failures.length ? ctx.failures.map((item) => `- ${item.type || 'failure'}: ${item.message || item.url || JSON.stringify(item)}`) : ['- none']),
    ``,
    `## Cleanup`,
    ...(ctx.cleanup.length ? ctx.cleanup.map((item) => `- ${item.type} ${item.id}: ${item.ok ? 'ok' : `status ${item.status}`}`) : ['- none'])
  ]
  fs.writeFileSync(mdPath, `${lines.join('\n')}\n`)
  return { jsonPath, mdPath, summary }
}
