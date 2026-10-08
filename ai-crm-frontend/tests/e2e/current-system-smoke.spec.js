import { expect, test } from '@playwright/test'
import { loginByAPI } from './helpers/auth.js'

const admin = {
  username: process.env.E2E_USERNAME || 'utopia01',
  password: process.env.E2E_PASSWORD || 'E2e@20260523!'
}

const routes = [
  ['/workspace', '客户主面板'],
  ['/dashboards', '销售大屏'],
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
  ['/system/role', '角色管理'],
  ['/system/user', '用户管理'],
  ['/system/department', '部门管理'],
  ['/system/post', '岗位管理'],
  ['/system/log', '操作日志'],
  ['/system/login-log', '登录日志'],
  ['/system/user-center', '个人中心']
]

test('当前租户核心页面可打开且无严重前端/API错误', async ({ page }) => {
  test.setTimeout(300000)
  const failures = []
  page.on('pageerror', (error) => failures.push(`pageerror: ${error.message}`))
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      const location = msg.location()
      const source = location.url ? ` (${location.url}:${location.lineNumber})` : ''
      failures.push(`console error: ${msg.text()}${source}`)
    }
  })
  page.on('response', (response) => {
    const url = response.url()
    if (response.status() >= 400) {
      failures.push(`response ${response.status()}: ${url}`)
    }
  })

  await loginByAPI(page, admin.username, admin.password, 'zh-CN')

  for (const [path, title] of routes) {
    await page.goto(path, { waitUntil: 'domcontentloaded', timeout: 30000 })
    await expect(page.getByRole('heading', { name: title }).first()).toBeVisible({ timeout: 15000 })
    await expect(page.locator('.app-main, main, .system-page, .domain-page').first()).toBeVisible({ timeout: 15000 })
  }

  expect(failures).toEqual([])
})

test('当前销售大屏展示目标管理与核心销售指标', async ({ page }) => {
  await loginByAPI(page, admin.username, admin.password, 'zh-CN')
  await page.goto('/dashboards')
  await expect(page.getByRole('heading', { name: '销售实时大屏' })).toBeVisible()
  await expect(page.getByText('年度目标').first()).toBeVisible()
  await expect(page.getByText('当前完成').first()).toBeVisible()
  await expect(page.getByText('完成率').first()).toBeVisible()
  await expect(page.getByRole('button', { name: '目标管理' })).toBeVisible()
  await expect(page.getByText('我的销售任务')).toBeVisible()
})
