import { chromium } from '@playwright/test'

const base = process.env.PLAYWRIGHT_BASE_URL || 'http://127.0.0.1:5177'
const routes = [
  '/',
  '/workspace',
  '/campaigns',
  '/leads',
  '/companies',
  '/contacts',
  '/opportunities',
  '/deals',
  '/orders',
  '/tickets',
  '/projects',
  '/products',
  '/dashboards',
  '/insights',
  '/im',
  '/system/user',
  '/system/role',
  '/system/menu',
  '/system/data-scope',
  '/system/operation-logs',
  '/system/settings',
  '/platform/overview',
  '/platform/tenants',
  '/platform/settings',
  '/platform/plans',
  '/platform/expiring',
  '/platform/runtime'
]

const forbidden = [
  'Missing translation',
  'Label',
  'Interface text',
  'CRM interface text',
  'Admin settings',
  'Lead handling',
  'Deal progress',
  'Campaign results',
  'Order finance',
  'Project progress',
  'Contact details',
  'PublicOpportunity',
  'PrivateOpportunity',
  'BatchDeleteInvalidLateLead',
  'SDR 跟进中',
  'decision_maker'
]

const modules = [
  'workspace',
  'projects',
  'products',
  'campaigns',
  'contacts',
  'leads',
  'opportunities',
  'deals',
  'orders',
  'tickets',
  'dashboards',
  'im',
  'teams',
  'accounts',
  'insights',
  'platform'
]

const permissions = ['*']
for (const mod of modules) {
  for (const action of ['view', 'create', 'update', 'delete', 'manage', 'assign', 'approve', 'export']) {
    permissions.push(`${mod}:${action}`)
  }
  permissions.push(`menu:${mod}`)
}
permissions.push('platform:overview', 'platform:tenants', 'platform:settings')

function responseData(url) {
  const path = new URL(url).pathname
  if (path.endsWith('/auth/me')) {
    return {
      id: 1,
      name: 'Tenant Admin',
      username: 'tenant_admin',
      role: 'tenant_admin',
      account_type: 'tenant_admin',
      tenant_id: 1,
      language_pref: 'en-US',
      permissions,
      enabled_modules: modules
    }
  }
  if (path.includes('/accounts/users')) {
    return {
      items: [
        { id: 1, name: 'Tenant Admin', username: 'tenant_admin' },
        { id: 2, name: 'Sales Rep', username: 'sales' }
      ],
      total: 2
    }
  }
  if (path.includes('/im/unread/summary')) return { total_unread: 0, system_unread: 0, conversation_unread: 0 }
  if (path.includes('/im/contacts')) return { items: [{ id: 2, name: 'Sales Rep', username: 'sales', online: true }] }
  if (path.includes('/im/conversations') && path.includes('/messages')) return { items: [] }
  if (path.includes('/im/conversations')) return { items: [] }
  if (path.includes('/im/system-notifications')) return { items: [], total: 0 }
  if (path.includes('/prd/summary')) return { domains: [] }
  if (path.includes('/prd/cockpit')) return { kpi: {}, trends: [], alerts: [] }
  if (path.includes('/prd/coverage')) return { items: [] }
  if (path.includes('/prd/schema')) return { domains: [] }
  if (path.includes('/prd/actions')) return { items: [] }
  if (path.includes('/prd/leads/dashboard')) return { kpi: {}, alerts: {}, workload: [] }
  if (path.includes('/prd/leads/views')) return { items: [] }
  if (path.includes('/prd/leads/nurture/templates')) return { items: [] }
  if (path.includes('/prd/leads/automation/rules')) return { items: [] }
  if (path.includes('/prd/relations')) return { items: [] }
  if (path.includes('/performance')) {
    return { leads: [], opportunities: [], deals: [], orders: [], payments: [], timeline: [], files: [], summary: {} }
  }
  if (path.includes('/prd/settings')) return { items: [], value: {} }
  if (path.includes('/platform/')) return { items: [], total: 0, summary: {}, metrics: {} }
  if (path.includes('/objects')) return { items: [], total: 0 }
  if (path.includes('/prd/') || path.includes('/operation-logs')) return { items: [], total: 0, kpi: {}, summary: {}, data: [] }
  return { items: [], total: 0 }
}

const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox'] })
const context = await browser.newContext({ viewport: { width: 1440, height: 960 }, deviceScaleFactor: 1 })
await context.route('**/api/**', async (route) => {
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ code: 0, data: responseData(route.request().url()) })
  })
})

const page = await context.newPage()
await page.addInitScript((state) => {
  localStorage.setItem('locale', 'en-US')
  localStorage.setItem('token', 'playwright-token')
  localStorage.setItem('current_user', JSON.stringify(state))
}, {
  id: 1,
  name: 'Tenant Admin',
  username: 'tenant_admin',
  role: 'tenant_admin',
  account_type: 'tenant_admin',
  tenant_id: 1,
  language_pref: 'en-US',
  permissions,
  enabled_modules: modules
})

const failures = []
for (const route of routes) {
  try {
    await page.goto(`${base}${route}`, { waitUntil: 'domcontentloaded', timeout: 25000 })
    await page.waitForTimeout(700)
    const result = await page.evaluate(({ forbiddenWords }) => {
      const bodyText = document.body?.innerText || ''
      const issues = []
      const pageWidth = document.documentElement.clientWidth
      const scrollWidth = document.documentElement.scrollWidth
      if (scrollWidth - pageWidth > 8) issues.push(`horizontal overflow ${scrollWidth - pageWidth}px`)
      for (const word of forbiddenWords) {
        if (bodyText.includes(word)) issues.push(`forbidden text: ${word}`)
      }
      const cnMatches = Array.from(new Set(bodyText.match(/[\u4e00-\u9fff]{2,}/g) || []))
        .filter((text) => !['线索', '商机', '订单', '公司', '联系人', '市场活动', '成交记录', '服务工单', '客户项目', '产品服务', '销售大屏', '站内消息', '系统管理', '平台概览'].includes(text))
        .slice(0, 15)
      if (cnMatches.length) issues.push(`Chinese text: ${cnMatches.join(' | ')}`)

      const selectors = 'button, .el-button, .el-tabs__item, .el-tag, .el-form-item__label, th .cell, .el-menu-item, .el-sub-menu__title'
      const overflow = []
      for (const el of Array.from(document.querySelectorAll(selectors))) {
        const rect = el.getBoundingClientRect()
        if (!rect.width || !rect.height || rect.width < 8 || rect.height < 8) continue
        if (el.scrollWidth - el.clientWidth > 3 || el.scrollHeight - el.clientHeight > 8) {
          const text = (el.innerText || el.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 80)
          if (text) {
            overflow.push(`${text} sw=${el.scrollWidth} cw=${el.clientWidth} sh=${el.scrollHeight} ch=${el.clientHeight}`)
          }
        }
      }
      if (overflow.length) issues.push(`element overflow: ${overflow.slice(0, 8).join(' || ')}`)
      return { issues }
    }, { forbiddenWords: forbidden })
    if (result.issues.length) failures.push({ route, issues: result.issues })
    console.log(`${result.issues.length ? 'FAIL' : 'OK'} ${route}`)
    for (const issue of result.issues) console.log(`  - ${issue}`)
  } catch (error) {
    failures.push({ route, issues: [String(error?.message || error)] })
    console.log(`ERROR ${route}: ${error?.message || error}`)
  }
}

await browser.close()

if (failures.length) {
  console.log('\nPLAYWRIGHT_FAIL')
  console.log(JSON.stringify(failures, null, 2))
  process.exit(1)
}

console.log('\nPLAYWRIGHT_OK')
