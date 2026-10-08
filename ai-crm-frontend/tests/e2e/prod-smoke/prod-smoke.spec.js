import { expect, test } from '@playwright/test'
import {
  CRM_DOMAIN_TYPES,
  PROD_ROUTES,
  assertRouteHealthy,
  attachFailureWatchers,
  cleanupCreatedData,
  createAPI,
  createCRMObject,
  dataOf,
  expectAPI,
  installAuthStorage,
  itemsOf,
  loginProd,
  makeRunContext,
  markBlocked,
  markCoverage,
  writeSmokeReport
} from './helpers/prod-context.js'

test.describe.configure({ mode: 'serial' })

const ctx = makeRunContext()
let api

test.beforeEach(async ({ page }) => {
  attachFailureWatchers(page, ctx)
  if (!ctx.token) {
    await loginProd(page, ctx)
  } else {
    await installAuthStorage(page, ctx)
  }
  api = createAPI(page.request, ctx)
})

test.afterAll(async ({ browser }) => {
  const page = await browser.newPage()
  try {
    attachFailureWatchers(page, ctx)
    if (!ctx.token) await loginProd(page, ctx)
    const cleanupAPI = createAPI(page.request, ctx)
    await cleanupCreatedData(cleanupAPI, ctx)
  } finally {
    const report = writeSmokeReport(ctx)
    console.log(`prod-smoke-summary=${report.mdPath}`)
    await page.close()
  }
})

test('prod-smoke-auth: 登录、刷新、个人中心、会话、退出', async ({ page }) => {
  await expect(page.evaluate(() => localStorage.getItem('token'))).resolves.toBeTruthy()
  const me = await api.get('/api/v1/auth/me')
  await expectAPI(me, 'auth me')
  markCoverage(ctx, 'auth', 'me')

  const profile = await api.get('/api/v1/accounts/me/profile')
  await expectAPI(profile, 'profile')
  markCoverage(ctx, 'auth', 'profile')

  const sessions = await api.get('/api/v1/accounts/me/sessions')
  await expectAPI(sessions, 'sessions', [403])
  if (sessions.ok) markCoverage(ctx, 'auth', 'sessions')
  else markBlocked(ctx, 'auth', 'sessions', `HTTP ${sessions.resp.status()}`)

  const refresh = await api.post('/api/v1/auth/refresh', {})
  await expectAPI(refresh, 'refresh token')
  markCoverage(ctx, 'auth', 'refresh token')

  await page.goto('/system/user-center', { waitUntil: 'domcontentloaded' })
  await expect(page.locator('.app-main, main, .system-page').first()).toBeVisible()
  markCoverage(ctx, 'auth', 'user center page')

  const logout = await api.post('/api/v1/auth/logout', {})
  await expectAPI(logout, 'logout', [403])
  markCoverage(ctx, 'auth', 'logout')
  await loginProd(page, ctx)
  api = createAPI(page.request, ctx)
})

test('prod-smoke-navigation: 遍历所有当前账号可访问页面', async ({ page }) => {
  test.setTimeout(600000)
  for (const [routePath, title] of PROD_ROUTES) {
    await assertRouteHealthy(page, ctx, routePath, title)
  }
})

test('prod-smoke-crm-flow: CRM 对象 CRUD、查询、导入导出、核心流程', async () => {
  test.setTimeout(360000)

  const company = await createCRMObject(api, ctx, 'contacts', 'company')
  const contact = await createCRMObject(api, ctx, 'contacts', 'contact', {
    payload: { company: company?.name || `${ctx.prefix}_company`, company_id: company?.id || 0 }
  })
  const product = await createCRMObject(api, ctx, 'products', 'product')
  const campaign = await createCRMObject(api, ctx, 'campaigns', 'campaign')
  const lead = await createCRMObject(api, ctx, 'leads', 'lead')
  const opportunity = await createCRMObject(api, ctx, 'opportunities', 'opportunity', {
    payload: { company_id: company?.id || 0, contact_id: contact?.id || 0, product_id: product?.id || 0 }
  })
  const deal = await createCRMObject(api, ctx, 'deals', 'closed_deal', {
    payload: { company_id: company?.id || 0, contact_id: contact?.id || 0, product_id: product?.id || 0 }
  })
  const order = await createCRMObject(api, ctx, 'orders', 'order', {
    payload: { deal_id: deal?.id || 0, company_id: company?.id || 0, contact_id: contact?.id || 0 }
  })
  const ticket = await createCRMObject(api, ctx, 'tickets', 'ticket', {
    payload: { order_id: order?.id || 0, company_id: company?.id || 0, contact_id: contact?.id || 0 }
  })
  const project = await createCRMObject(api, ctx, 'projects', 'project', {
    payload: { order_id: order?.id || 0, company_id: company?.id || 0, contact_id: contact?.id || 0 }
  })

  const requiredObjects = { company, contact, product, campaign, lead, opportunity, deal, order, ticket, project }
  for (const [name, value] of Object.entries(requiredObjects)) {
    expect(value, `${name} created`).toBeTruthy()
  }
  markCoverage(ctx, 'crm-flow', 'create company/contact/product/campaign/lead/opportunity/deal/order/ticket/project')

  const updateLead = await api.put(`/api/v1/prd/leads/objects/${lead.id}`, {
    object_type: 'lead',
    name: `${lead.name}_UPDATED`,
    code: lead.code || `${ctx.prefix}_lead_code`,
    status: 'following',
    owner_id: ctx.user?.id || lead.owner_id || 0,
    amount: 200,
    currency: 'USD',
    score: 80,
    tags: [ctx.prefix, 'updated'],
    payload: {
      smoke_prefix: ctx.prefix,
      company_name: `${ctx.prefix}_lead_company`,
      contact_name: `${ctx.prefix}_lead_contact`,
      email: `${ctx.prefix.toLowerCase()}_updated@example.test`,
      phone: '13900000000',
      source_channel: 'e2e_smoke',
      utm_source: 'playwright',
      intent_product: 'Halmify CRM',
      smoke_updated: true
    }
  })
  await expectAPI(updateLead, 'update lead')
  markCoverage(ctx, 'crm-flow', 'update lead')

  for (const domain of Object.keys(CRM_DOMAIN_TYPES)) {
    const list = await api.get(`/api/v1/prd/${domain}/objects?page=1&size=10&keyword=${encodeURIComponent(ctx.prefix)}`)
    await expectAPI(list, `list ${domain}`, [403])
    if (list.ok) {
      markCoverage(ctx, domain, 'keyword search')
    } else {
      markBlocked(ctx, domain, 'keyword search', `HTTP ${list.resp.status()}`)
    }
  }

  const importName = `${ctx.prefix}_contacts_import`
  const imported = await api.post('/api/v1/prd/contacts/objects/import', {
    items: [{
      object_type: 'contact',
      name: importName,
      code: `${ctx.prefix}_IMPORT_CONTACT`,
      status: 'active',
      tags: [ctx.prefix, 'import'],
      payload: {
        e2e_smoke: true,
        smoke_prefix: ctx.prefix,
        email: `${ctx.prefix.toLowerCase()}_import@example.test`,
        phone: '13700000000',
        company: company.name
      }
    }]
  })
  await expectAPI(imported, 'import contacts', [403])
  if (imported.ok) {
    const importedList = await api.get(`/api/v1/prd/contacts/objects?page=1&size=10&keyword=${encodeURIComponent(importName)}`)
    const match = itemsOf(importedList).find((item) => item.name === importName)
    if (match?.id) ctx.created.objects.push({ domain: 'contacts', id: match.id, name: match.name, objectType: 'contact' })
    markCoverage(ctx, 'crm-flow', 'import contacts')
  } else {
    markBlocked(ctx, 'crm-flow', 'import contacts', `HTTP ${imported.resp.status()}`)
  }

  const exported = await api.get('/api/v1/prd/contacts/objects/export')
  await expectAPI(exported, 'export contacts', [403])
  if (exported.ok) markCoverage(ctx, 'crm-flow', 'export contacts')
  else markBlocked(ctx, 'crm-flow', 'export contacts', `HTTP ${exported.resp.status()}`)

  const relation = await api.post('/api/v1/prd/relations', {
    source_type: 'contacts',
    source_id: company.id,
    target_type: 'contacts',
    target_id: contact.id,
    relation_type: 'e2e_company_contact',
    payload: { smoke_prefix: ctx.prefix }
  })
  await expectAPI(relation, 'create relation', [403])
  markCoverage(ctx, 'crm-flow', 'relation create')

  const flowChecks = [
    ['lead score', () => api.post(`/api/v1/prd/flows/leads/${lead.id}/score`, {})],
    ['lead follow-up', () => api.post(`/api/v1/prd/flows/leads/${lead.id}/follow-up`, { content: 'E2E smoke follow-up', next_action: 'call' })],
    ['opportunity stage', () => api.post(`/api/v1/prd/flows/opportunities/${opportunity.id}/stage`, { stage: 'proposal', comment: 'E2E smoke stage' })],
    ['opportunity quote', () => api.post(`/api/v1/prd/flows/opportunities/${opportunity.id}/quote`, { reason: 'E2E smoke quote' })],
    ['opportunity risk', () => api.post(`/api/v1/prd/flows/opportunities/${opportunity.id}/risk`, {})],
    ['deal order', () => api.post(`/api/v1/prd/flows/deals/${deal.id}/order`, {})],
    ['order pay', () => api.post(`/api/v1/prd/flows/orders/${order.id}/pay`, { payment_no: `${ctx.prefix}_PAY`, payment_method: 'manual', paid_amount: 1 })],
    ['order ship', () => api.post(`/api/v1/prd/flows/orders/${order.id}/ship`, { shipment_no: `${ctx.prefix}_SHP`, carrier: 'manual' })],
    ['order invoice', () => api.post(`/api/v1/prd/flows/orders/${order.id}/invoice`, { invoice_no: `${ctx.prefix}_INV`, invoice_type: 'normal' })],
    ['order backup', () => api.get(`/api/v1/prd/flows/orders/${order.id}/backup`)],
    ['project progress', () => api.post(`/api/v1/prd/flows/projects/${project.id}/progress`, { progress: 30, comment: 'E2E smoke progress' })],
    ['project milestone', () => api.post(`/api/v1/prd/flows/projects/${project.id}/milestone`, { name: `${ctx.prefix}_milestone`, planned_date: '2026-06-30', status: 'planning' })],
    ['ticket note', () => api.post(`/api/v1/prd/flows/tickets/${ticket.id}/note`, { visibility: 'internal', content: 'E2E smoke ticket note' })],
    ['ticket knowledge suggestions', () => api.get(`/api/v1/prd/flows/tickets/${ticket.id}/knowledge-suggestions`)],
    ['contacts dedupe', () => api.post('/api/v1/prd/flows/contacts/dedupe', { keyword: ctx.prefix })],
    ['module access preview', () => api.get('/api/v1/prd/module-access-preview')],
    ['next best actions', () => api.get('/api/v1/prd/next-actions?limit=10')]
  ]

  for (const [label, fn] of flowChecks) {
    const result = await fn()
    await expectAPI(result, label, [400, 403])
    if (result.ok) {
      markCoverage(ctx, 'crm-flow', label)
    } else {
      markBlocked(ctx, 'crm-flow', label, `HTTP ${result.resp.status()} ${result.body?.message || ''}`)
    }
  }
})

test('prod-smoke-files: 上传、信息、下载、删除文件服务链路', async () => {
  const project = await createCRMObject(api, ctx, 'projects', 'project')
  expect(project?.id).toBeTruthy()

  const nodes = await api.get('/api/v1/files/nodes')
  await expectAPI(nodes, 'file nodes', [403])
  if (nodes.ok) markCoverage(ctx, 'files', 'nodes')

  const upload = await api.multipart('/api/v1/files/upload', {
    file: {
      name: `${ctx.prefix}.txt`,
      mimeType: 'text/plain',
      buffer: Buffer.from(`production smoke ${ctx.prefix}\n`)
    }
  })
  await expectAPI(upload, 'file upload', [403, 503])
  if (!upload.ok) {
    markBlocked(ctx, 'files', 'upload', `HTTP ${upload.resp.status()} ${upload.body?.message || ''}`)
    return
  }
  const uploaded = dataOf(upload)
  const fileID = uploaded?.file_id || uploaded?.id
  expect(fileID).toBeTruthy()
  ctx.created.files.push({ file_id: fileID })
  markCoverage(ctx, 'files', 'upload')

  const info = await api.get(`/api/v1/files/${fileID}/info`)
  await expectAPI(info, 'file info')
  markCoverage(ctx, 'files', 'info')

  const linked = await api.post(`/api/v1/prd/flows/projects/${project.id}/files`, {
    file_id: fileID,
    file_name: `${ctx.prefix}.txt`,
    file_size: Number(uploaded?.file_size || uploaded?.size || 0),
    file_type: 'text/plain',
    storage_node_id: uploaded?.storage_node_id || '',
    download_url: uploaded?.download_url || `/api/v1/files/${fileID}/download`,
    category: 'other',
    version: 'v1',
    visibility: 'internal',
    remark: 'production smoke project file'
  })
  await expectAPI(linked, 'project file link', [403])
  const linkedData = dataOf(linked)
  const linkedFile = linkedData?.file || linkedData
  if (linked.ok && linkedFile?.id) {
    ctx.created.objects.push({ domain: 'projects', id: linkedFile.id, name: linkedFile.name, objectType: 'project_file' })
    markCoverage(ctx, 'files', 'project file link')
  } else {
    markBlocked(ctx, 'files', 'project file link', `HTTP ${linked.resp.status()}`)
  }

  const listLinked = await api.get(`/api/v1/prd/flows/projects/${project.id}/files`)
  await expectAPI(listLinked, 'project file list', [403])
  if (listLinked.ok) markCoverage(ctx, 'files', 'project file list')

  const download = await api.get(`/api/v1/files/${fileID}/download`)
  await expectAPI(download, 'file download', [403])
  if (download.ok) markCoverage(ctx, 'files', 'download')
  else markBlocked(ctx, 'files', 'download', `HTTP ${download.resp.status()}`)

  if (linkedFile?.id) {
    const unlink = await api.delete(`/api/v1/prd/flows/projects/${project.id}/files/${linkedFile.id}`)
    await expectAPI(unlink, 'project file delete', [403])
    if (unlink.ok) {
      ctx.created.objects = ctx.created.objects.filter((item) => !(item.domain === 'projects' && item.id === linkedFile.id))
      markCoverage(ctx, 'files', 'project file delete')
    }
  }

  const deleted = await api.delete(`/api/v1/files/${fileID}`)
  await expectAPI(deleted, 'file delete', [403, 404])
  if (deleted.ok || deleted.resp.status() === 404) {
    ctx.created.files = ctx.created.files.filter((item) => item.file_id !== fileID)
    markCoverage(ctx, 'files', 'delete')
  } else {
    markBlocked(ctx, 'files', 'delete', `HTTP ${deleted.resp.status()}`)
  }
})

test('prod-smoke-admin: 用户、角色、部门、岗位、菜单、日志、IoT、存储', async () => {
  test.setTimeout(300000)

  const roles = await api.get('/api/v1/rbac/roles')
  await expectAPI(roles, 'list roles', [403])
  const roleItems = itemsOf(roles)
  if (roles.ok) markCoverage(ctx, 'admin', 'list roles')
  else markBlocked(ctx, 'admin', 'list roles', `HTTP ${roles.resp.status()}`)

  const role = await api.post('/api/v1/rbac/roles', {
    name: `${ctx.prefix}_角色`,
    code: `${ctx.prefix}_role`.toLowerCase(),
    description: 'production smoke role',
    status: 'active',
    permissions: []
  })
  await expectAPI(role, 'create role', [403])
  const roleData = dataOf(role)
  if (role.ok && roleData?.id) {
    ctx.created.roles.push({ id: roleData.id })
    markCoverage(ctx, 'admin', 'create role')
  } else {
    markBlocked(ctx, 'admin', 'create role', `HTTP ${role.resp.status()}`)
  }

  const department = await api.post('/api/v1/system/departments', {
    name: `${ctx.prefix}_部门`,
    code: `${ctx.prefix}_dept`.toLowerCase(),
    sort: 999,
    status: 'active',
    remark: 'production smoke department'
  })
  await expectAPI(department, 'create department', [403])
  const departmentData = dataOf(department)
  if (department.ok && departmentData?.id) {
    ctx.created.departments.push({ id: departmentData.id })
    markCoverage(ctx, 'admin', 'create department')
  } else {
    markBlocked(ctx, 'admin', 'create department', `HTTP ${department.resp.status()}`)
  }

  const post = await api.post('/api/v1/system/posts', {
    name: `${ctx.prefix}_岗位`,
    code: `${ctx.prefix}_post`.toLowerCase(),
    sort: 999,
    status: 'active',
    remark: 'production smoke post'
  })
  await expectAPI(post, 'create post', [403])
  const postData = dataOf(post)
  if (post.ok && postData?.id) {
    ctx.created.posts.push({ id: postData.id })
    markCoverage(ctx, 'admin', 'create post')
  } else {
    markBlocked(ctx, 'admin', 'create post', `HTTP ${post.resp.status()}`)
  }

  const menu = await api.post('/api/v1/system/menus', {
    name: `${ctx.prefix}_菜单`,
    type: 'button',
    path: '',
    permission: `${ctx.prefix}:test`,
    sort: 999,
    status: 'disabled',
    hidden: true,
    remark: 'production smoke menu'
  })
  await expectAPI(menu, 'create menu', [403])
  const menuData = dataOf(menu)
  if (menu.ok && menuData?.id) {
    ctx.created.menus.push({ id: menuData.id })
    markCoverage(ctx, 'admin', 'create menu')
  } else {
    markBlocked(ctx, 'admin', 'create menu', `HTTP ${menu.resp.status()}`)
  }

  const createUserRoleID = roleData?.id || roleItems[0]?.id
  const user = await api.post('/api/v1/accounts/users', {
    username: `${ctx.prefix}_user`.toLowerCase(),
    password: 'E2eSmoke@12345',
    name: `${ctx.prefix}_用户`,
    email: `${ctx.prefix.toLowerCase()}_user@example.test`,
    role_ids: createUserRoleID ? [createUserRoleID] : [],
    department_id: departmentData?.id || 0,
    post_id: postData?.id || 0,
    language_pref: 'zh-CN'
  })
  await expectAPI(user, 'create user', [400, 403])
  const userData = dataOf(user)
  if (user.ok && userData?.id) {
    ctx.created.users.push({ id: userData.id })
    markCoverage(ctx, 'admin', 'create user')
  } else {
    markBlocked(ctx, 'admin', 'create user', `HTTP ${user.resp.status()} ${user.body?.message || ''}`)
  }

  const readEndpoints = [
    ['admin', 'list users', '/api/v1/accounts/users?page=1&size=10'],
    ['admin', 'list departments', '/api/v1/system/departments'],
    ['admin', 'list posts', '/api/v1/system/posts?page=1&size=10'],
    ['admin', 'list menus', '/api/v1/system/menus'],
    ['admin', 'operation logs', '/api/v1/operation-logs?page=1&size=10'],
    ['admin', 'login logs export', '/api/v1/operation-logs/login/export'],
    ['storage', 'usage', '/api/v1/storage/usage'],
    ['storage', 'disk space', '/api/v1/disk/space'],
    ['storage', 'seat usage', '/api/v1/entitlements/seats/usage'],
    ['iot', 'list devices', '/api/v1/iot/devices'],
    ['im', 'contacts', '/api/v1/im/contacts'],
    ['im', 'conversations', '/api/v1/im/conversations'],
    ['im', 'unread summary', '/api/v1/im/unread/summary'],
    ['ai', 'usage summary', '/api/v1/prd/ai/usage/summary'],
    ['ai', 'pricing', '/api/v1/prd/ai/pricing'],
    ['ai', 'tenant status', '/api/v1/prd/flows/tenant-ai-analysis/status']
  ]
  for (const [module, label, url] of readEndpoints) {
    const result = await api.get(url)
    await expectAPI(result, label, [403])
    if (result.ok) markCoverage(ctx, module, label)
    else markBlocked(ctx, module, label, `HTTP ${result.resp.status()}`)
  }

  const device = await api.post('/api/v1/iot/devices', {
    name: `${ctx.prefix}_设备`,
    sn: `${ctx.prefix}_SN`,
    model: 'esp32-s3-epaper',
    status: 'pending_bind',
    resolution_width: 200,
    resolution_height: 200,
    remark: 'production smoke device'
  })
  await expectAPI(device, 'create iot device', [403])
  const deviceData = dataOf(device)
  if (device.ok && deviceData?.id) {
    ctx.created.iotDevices.push({ id: deviceData.id })
    markCoverage(ctx, 'iot', 'create device')
    const preview = await api.get(`/api/v1/iot/devices/${deviceData.id}/preview`)
    await expectAPI(preview, 'iot preview', [403])
    if (preview.ok) markCoverage(ctx, 'iot', 'preview device')
  } else {
    markBlocked(ctx, 'iot', 'create device', `HTTP ${device.resp.status()}`)
  }
})

test('prod-smoke-platform: 平台管理、套餐、租户、配置、账务、运行监控、固件只读/草稿冒烟', async () => {
  test.setTimeout(240000)
  const platformEndpoints = [
    ['overview', '/api/v1/platform/tenants/summary'],
    ['tenants', '/api/v1/tenants'],
    ['tenant summaries', '/api/v1/platform/tenants/summary'],
    ['expiring tenants', '/api/v1/platform/tenants/expiring'],
    ['runtime overview', '/api/v1/platform/runtime/overview'],
    ['plans', '/api/v1/platform/plans'],
    ['settings', '/api/v1/platform/settings'],
    ['ai applications', '/api/v1/platform/ai-analysis/applications'],
    ['ai packages', '/api/v1/platform/ai-analysis/packages'],
    ['ai usage summary', '/api/v1/platform/ai-analysis/usage/summary'],
    ['ai ledger', '/api/v1/platform/ai-analysis/ledger'],
    ['iot firmwares', '/api/v1/platform/iot/firmwares']
  ]

  for (const [label, url] of platformEndpoints) {
    const result = await api.get(url)
    await expectAPI(result, `platform ${label}`, [403])
    if (result.ok) {
      markCoverage(ctx, 'platform', label)
    } else {
      markBlocked(ctx, 'platform', label, `HTTP ${result.resp.status()}`)
    }
  }

  const settingsDraft = await api.post('/api/v1/platform/settings/draft', {
    key: `${ctx.prefix}_setting`,
    value: 'smoke',
    description: 'production smoke draft setting'
  })
  await expectAPI(settingsDraft, 'platform setting draft', [400, 403])
  if (settingsDraft.ok) markCoverage(ctx, 'platform', 'settings draft')
  else markBlocked(ctx, 'platform', 'settings draft', `HTTP ${settingsDraft.resp.status()} ${settingsDraft.body?.message || ''}`)

  const quote = await api.post('/api/v1/entitlements/quote', {
    capacity_gb: 1,
    years: 1,
    seats: 1
  })
  await expectAPI(quote, 'entitlement quote', [400, 403])
  if (quote.ok) markCoverage(ctx, 'platform', 'entitlement quote')
  else markBlocked(ctx, 'platform', 'entitlement quote', `HTTP ${quote.resp.status()} ${quote.body?.message || ''}`)
})

test('prod-smoke-cleanup-report: 汇总失败、权限不可达和清理结果', async () => {
  const fatalFailures = ctx.failures.filter((item) => item.type !== 'console_error' || !/favicon|ResizeObserver/i.test(item.message || ''))
  const apiFailures = ctx.apiErrors.filter((item) => item.status >= 500 || item.status === 401)
  const cleanupFailures = ctx.cleanup.filter((item) => !item.ok && item.status !== 404)
  markCoverage(ctx, 'cleanup', 'summary', cleanupFailures.length === 0 ? 'passed' : 'failed', `${cleanupFailures.length} cleanup failures`)
  expect({
    fatalFailures,
    apiFailures,
    cleanupFailures
  }).toEqual({
    fatalFailures: [],
    apiFailures: [],
    cleanupFailures: []
  })
})
