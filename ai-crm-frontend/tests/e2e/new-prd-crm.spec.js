import { expect, test } from '@playwright/test'
import { loginByAPI } from './helpers/auth.js'

const admin = {
  username: process.env.E2E_USERNAME || 'admin',
  password: process.env.E2E_PASSWORD || '123456'
}

const toast = (page, text) => page.locator('.el-message__content').filter({ hasText: text }).last()

async function login(page) {
  await loginByAPI(page, admin.username, admin.password, 'zh-CN')
  await page.goto('/workspace')
  await expect(page).toHaveURL(/\/workspace/, { timeout: 20000 })
  await expect(page.getByRole('heading', { name: '客户主面板' })).toBeVisible()
}

test('newprd 工作台展示 15 模块、功能地图和动作中心', async ({ page }) => {
  await login(page)
  await expect(page.getByRole('button', { name: /销售工作台/ }).first()).toBeVisible()

  await page.goto('/leads')
  await expect(page).toHaveURL(/\/leads/)
  await expect(page.getByRole('button', { name: '新增线索' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'SLA回收' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: '线索名称' })).toBeVisible()
})

test('线索专属页面支持新增、查重、评分、分配和转商机', async ({ page }) => {
  await login(page)
  await page.goto('/leads')
  await expect(page).toHaveURL(/\/leads/)
  await expect(page.getByRole('button', { name: '新增线索' })).toBeVisible()

  const companyName = `E2E线索公司${Date.now()}`
  await page.getByRole('button', { name: '新增线索' }).click()
  const dialog = page.getByRole('dialog', { name: '新增线索' })
  await dialog.getByLabel('公司名').fill(companyName)
  await dialog.getByLabel('联系人').fill('E2E 联系人')
  await dialog.getByLabel('手机').fill('13800001111')
  await dialog.getByLabel('邮箱').fill(`lead${Date.now()}@example.com`)
  await page.getByRole('button', { name: '提交' }).click()
  await expect(toast(page, '操作成功')).toBeVisible()

  const row = page.locator('.system-table .el-table__body tr').filter({ hasText: companyName }).first()
  await expect(row).toBeVisible()
  await row.getByRole('button').nth(3).click()
  await page.getByRole('dialog', { name: '线索评分' }).getByRole('button', { name: '保存评分' }).click()
  await expect(toast(page, '线索评分已保存')).toBeVisible()

  await row.getByRole('button').nth(2).click()
  const assignDialog = page.getByRole('dialog', { name: '分配线索' })
  await assignDialog.getByLabel('负责人').click({ force: true })
  await page.getByRole('option').first().click()
  await assignDialog.getByRole('button', { name: '确定' }).click()
  await expect(toast(page, '操作成功')).toBeVisible()

  await row.getByRole('button').nth(4).click()
  const convertDialog = page.getByRole('dialog', { name: '线索转商机' })
  await convertDialog.getByLabel('预估金额').fill('88000')
  await convertDialog.getByLabel('处理说明').fill('E2E转商机')
  await convertDialog.getByRole('button', { name: '确定' }).click()
  await expect(toast(page, '操作成功')).toBeVisible()
})

test('线索页面支持CSV导入、培育模板、进入培育和SLA回收', async ({ page }) => {
  await login(page)
  await page.goto('/leads')
  await expect(page.getByText('线索漏斗')).toBeVisible()
  await expect(page.getByText('渠道转化')).toBeVisible()
  await expect(page.getByText('SDR 工作台')).toBeVisible()

  await page.getByRole('button', { name: '批量导入' }).click()
  await page.locator('input[type="file"]').setInputFiles({
    name: 'leads.csv',
    mimeType: 'text/csv',
    buffer: Buffer.from([
    'company_name,contact_name,phone,email,source_channel,intent_product',
    `CSV导入公司${Date.now()},CSV 联系人,13900001111,csvlead@example.com,website,AI CRM`
    ].join('\n'))
  })
  await expect(toast(page, '文件已读取')).toBeVisible()
  const importDialog = page.getByRole('dialog', { name: '批量导入线索' })
  await expect(importDialog.locator('.import-preview-summary').getByText('有效')).toBeVisible()
  await expect(importDialog.locator('.import-preview-summary').getByText('异常')).toBeVisible()
  await importDialog.getByRole('button', { name: '仅导入有效' }).click()
  await expect(toast(page, '导入完成')).toBeVisible()

  await page.getByRole('button', { name: '培育模板' }).click()
  const nurtureDialog = page.getByRole('dialog', { name: '培育模板' })
  await nurtureDialog.getByPlaceholder('请输入模板名称').fill(`E2E培育模板${Date.now()}`)
  await nurtureDialog.getByRole('button', { name: '保存模板' }).click()
  await expect(toast(page, '培育模板已保存')).toBeVisible()
  await page.keyboard.press('Escape')

  await page.getByRole('row', { name: /CSV导入公司/ }).first().click()
  const leadDrawer = page.locator('.el-drawer:visible')
  await leadDrawer.getByRole('button', { name: '进入培育' }).click()
  const nurtureActionDialog = page.getByRole('dialog', { name: '进入培育' })
  await nurtureActionDialog.getByLabel('处理说明').fill('E2E进入培育')
  await nurtureActionDialog.getByRole('button', { name: '确定' }).click()
  await expect(toast(page, '操作成功')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.locator('.el-drawer:visible')).toHaveCount(0)

  await page.getByRole('button', { name: 'SLA回收' }).click()
  await expect(toast(page, 'SLA回收已执行')).toBeVisible()
})

test('线索导入支持字段映射异常预检、ABM命中和SDR沟通记录', async ({ page }) => {
  await login(page)
  await page.goto('/leads')

  await page.getByRole('button', { name: '批量导入' }).click()
  const importDialog = page.getByRole('dialog', { name: '批量导入线索' })
  await importDialog.locator('textarea').fill([
    '公司,姓名,电话,邮箱,来源,意向产品,状态',
    `ABM目标公司${Date.now()},张SDR,13700003333,sdr@example.com,social,AI CRM,sdr_following`,
    '异常公司,,not-phone,bad-email,website,AI CRM'
  ].join('\n'))
  await importDialog.getByRole('button', { name: '解析预览' }).click()
  await expect(toast(page, '解析完成')).toBeVisible()
  await expect(importDialog.locator('.import-preview-table').getByText('可导入')).toBeVisible()
  await expect(importDialog.locator('.import-preview-table').getByText('异常').last()).toBeVisible()
  await importDialog.getByRole('button', { name: '仅导入有效' }).click()
  await expect(toast(page, '导入完成')).toBeVisible()

  await page.getByRole('button', { name: 'ABM名单' }).click()
  const abmDrawer = page.locator('.el-drawer:visible')
  await abmDrawer.locator('textarea').first().fill('ABM目标公司')
  await abmDrawer.getByRole('button', { name: '保存名单' }).click()
  await expect(toast(page, 'ABM名单已保存')).toBeVisible()
  await page.keyboard.press('Escape')

  await page.getByRole('button', { name: '进入队列' }).click()
  await expect(page.getByRole('heading', { name: 'SDR工作台' })).toBeVisible()
  await expect(page.locator('.sdr-workbench')).toBeVisible()
  const sdrWorkbench = page.locator('.sdr-workbench')
  await sdrWorkbench.locator('.sdr-queue button').first().click()
  await sdrWorkbench.getByRole('button', { name: '标记SQL' }).click()
  await expect(toast(page, '已标记为SQL')).toBeVisible()
})

test('线索自动化规则可保存、干跑并立即执行', async ({ page }) => {
  await login(page)
  await page.goto('/leads')
  await expect(page.getByText('主管预警')).toBeVisible()

  await page.getByRole('button', { name: '自动化规则' }).first().click()
  await expect(page.getByRole('heading', { name: '线索自动化规则' })).toBeVisible()
  await page.getByLabel('回收小时').fill('12')
  await page.getByLabel('间隔分钟').fill('45')
  await page.getByRole('button', { name: '保存规则' }).click()
  await expect(toast(page, '自动化规则已保存')).toBeVisible()

  await page.getByRole('button', { name: '干跑检查' }).click()
  await expect(toast(page, '自动化干跑完成')).toBeVisible()
  await expect(page.locator('.automation-result').getByText('扫描')).toBeVisible()

  await page.getByRole('button', { name: '立即执行' }).click()
  await expect(toast(page, '线索自动化已执行')).toBeVisible()
  await expect(page.locator('.automation-result').getByText('培育')).toBeVisible()
})

test('线索页面支持看板视图和客户空间视图', async ({ page }) => {
  await login(page)
  await page.goto('/leads')
  await expect(page.getByText('看板', { exact: true })).toBeVisible()

  await page.getByText('看板', { exact: true }).click()
  await expect(page.locator('.lead-kanban-board')).toBeVisible()
  await expect(page.getByText('新线索').first()).toBeVisible()

  await page.getByText('客户空间', { exact: true }).click()
  await expect(page.locator('.lead-customer-space')).toBeVisible()
  await expect(page.getByRole('button', { name: '转商机' }).first()).toBeVisible()
})

test('各菜单只展示本模块内容和动作', async ({ page }) => {
  await login(page)
  await expect(page).toHaveURL(/\/workspace/)

  await page.goto('/leads')
  await expect(page).toHaveURL(/\/leads/)
  await expect(page.getByRole('heading', { name: '线索', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: '新增线索' })).toBeVisible()
  await expect(page.getByRole('button', { name: /营销获客/ })).toHaveCount(0)
  await expect(page.getByRole('button', { name: '订单发货' })).toHaveCount(0)

  await page.goto('/orders')
  await expect(page.getByRole('columnheader', { name: '订单' })).toBeVisible()
  await expect(page.getByText('AI 线索评分')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '登记发货' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'AI 线索评分' })).toHaveCount(0)
})

test('商机阶段门禁可通过快捷动作补齐必填字段后推进', async ({ page }) => {
  await login(page)
  await page.goto('/opportunities')
  await expect(page).toHaveURL(/\/opportunities/)

  const firstRow = page.locator('.system-table .el-table__body tr').first()
  await firstRow.click()
  const opportunityDrawer = page.locator('.el-drawer:visible')
  await opportunityDrawer.getByRole('button', { name: '推进阶段' }).click()
  const stageDialog = page.getByRole('dialog', { name: '推进商机阶段' })
  await stageDialog.getByLabel('处理内容').fill('E2E推进阶段')
  await stageDialog.getByRole('button', { name: '确定' }).click()
  await expect(page.getByText('暂无推进记录，点击上方按钮处理后会显示在这里。')).toHaveCount(0)
})

test('多模块关键按钮可执行并反馈', async ({ page }) => {
  await login(page)

  await page.goto('/campaigns')
  await expect(page.getByRole('heading', { name: '市场活动管理' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: '活动捕获线索' })).toBeVisible()
  await page.getByRole('button', { name: '活动捕获线索' }).click()
  await expect(page.getByText('已捕获线索')).toBeVisible()
  await page.getByRole('button', { name: '生成活动归因' }).click()
  await expect(page.getByText('活动归因已生成')).toBeVisible()
  await expect(page.getByText('归因触点明细')).toBeVisible()
  await expect(page.getByText('渠道 ROI')).toBeVisible()

  await page.goto('/tickets')
  await page.locator('.system-table .el-table__body tr').first().click()
  const ticketDrawer = page.locator('.el-drawer:visible')
  await ticketDrawer.getByRole('button', { name: '添加处理记录' }).click()
  const noteDialog = page.getByRole('dialog', { name: '添加处理记录' })
  await noteDialog.getByLabel('处理内容').fill('E2E工单处理')
  await noteDialog.getByRole('button', { name: '确定' }).click()
  await expect(noteDialog).toHaveCount(0)

  await page.goto('/orders')
  await page.locator('.system-table .el-table__body tr').first().click()
  const orderDrawer = page.locator('.el-drawer:visible')
  await orderDrawer.getByRole('button', { name: '登记回款' }).click()
  const payDialog = page.getByRole('dialog', { name: '登记回款' })
  await payDialog.getByLabel('处理内容').fill('E2E登记回款')
  await payDialog.getByRole('button', { name: '确定' }).click()
  await expect(page.locator('.el-dialog:visible')).toHaveCount(0)
})

test('销售大屏展示真实图表并支持大屏模式和目标保存', async ({ page }) => {
  await login(page)
  await page.goto('/dashboards')

  await expect(page.getByRole('heading', { name: '销售实时大屏' })).toBeVisible()
  await expect(page.locator('.dashboard-chart-panel').filter({ hasText: '销售漏斗' }).first()).toBeVisible()
  await expect(page.locator('.dashboard-chart-panel').filter({ hasText: '成交趋势' }).first()).toBeVisible()
  await expect(page.locator('.dashboard-chart-panel').filter({ hasText: '团队排行' }).first()).toBeVisible()
  await expect(page.locator('.dashboard-chart-panel').filter({ hasText: '目标进度' }).first()).toBeVisible()

  await page.getByRole('button', { name: '大屏模式' }).click()
  await expect(page.locator('.dashboard-command.fullscreen')).toBeVisible()
  await page.getByRole('button', { name: '退出大屏' }).click()
  await expect(page.locator('.dashboard-command.fullscreen')).toHaveCount(0)

  await page.getByRole('button', { name: '目标配置' }).click()
  await page.getByRole('dialog', { name: '销售目标配置' }).getByLabel('目标金额').fill('1200000')
  await page.getByRole('button', { name: '保存配置' }).click()
  await expect(toast(page, '销售目标已保存')).toBeVisible()

  await page.getByRole('button', { name: '保存快照' }).click()
  await expect(toast(page, '大屏快照已保存')).toBeVisible()
})

test('联系人和公司页面支持独立列表、详情和列设置', async ({ page }) => {
  await login(page)
  await page.goto('/contacts')
  await expect(page.getByRole('button', { name: '新增联系人' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: '联系人' })).toBeVisible()
  await expect(page.getByText('AI 线索评分')).toHaveCount(0)

  await page.goto('/companies')
  await expect(page.getByRole('button', { name: '新增公司' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: '公司名称' })).toBeVisible()

  await page.getByRole('button', { name: '列设置' }).click()
  await expect(page.getByRole('heading', { name: '列设置' })).toBeVisible()
  await page.keyboard.press('Escape')

  const viewSwitcher = page.locator('.contact-company-page .el-segmented')
  await viewSwitcher.getByText('看板', { exact: true }).click()
  await expect(page.locator('.contact-kanban-board')).toBeVisible()

  await viewSwitcher.getByText('客户空间', { exact: true }).click()
  await expect(page.locator('.contact-customer-space')).toBeVisible()
  await expect(page.getByRole('button', { name: '查看空间' }).first()).toBeVisible()
  await expect(page.getByRole('button', { name: '添加互动' }).first()).toBeVisible()
})

test('企业默认货币和日期格式保存后驱动工作台展示', async ({ page }) => {
  await login(page)
  await page.goto('/enterprise')
  await expect(page).toHaveURL(/\/enterprise/)

  await page.getByRole('button', { name: /企业信息/ }).click()
  await page.locator('.enterprise-form .el-form-item').filter({ hasText: '默认货币' }).locator('.el-select').click()
  await page.getByRole('option', { name: 'USD' }).click()
  await page.getByRole('button', { name: '保存配置' }).click()
  await expect(toast(page, '配置已保存')).toBeVisible()

  await page.goto('/deals')
  await page.getByRole('button', { name: '新增成交' }).click()
  const createDealDialog = page.getByRole('dialog', { name: '新增成交' })
  await createDealDialog.getByLabel('成交名称').fill(`E2E-USD-成交-${Date.now()}`)
  await createDealDialog.getByLabel('成交金额').fill('12345')
  await createDealDialog.getByLabel('币种').click({ force: true })
  await page.getByRole('option', { name: 'USD' }).click()
  await createDealDialog.getByRole('button', { name: '保存' }).click()
  await expect(page.locator('.system-table .el-table__body').getByText(/\$/).first()).toBeVisible()
})

test('系统管理模块可打开核心页面并展示真实数据', async ({ page }) => {
  await login(page)

  await page.goto('/system/user')
  await expect(page.getByRole('heading', { name: '用户管理' })).toBeVisible()
  await expect(page.getByRole('button', { name: '新增用户' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: '用户名' })).toBeVisible()

  await page.goto('/system/department')
  await expect(page.getByRole('heading', { name: '部门管理' })).toBeVisible()
  await expect(page.getByRole('button', { name: '新增部门' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: '部门名称' })).toBeVisible()

  await page.goto('/system/menu')
  await expect(page.getByRole('heading', { name: '菜单管理' })).toBeVisible()
  await expect(page.getByRole('button', { name: '新增菜单' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: '菜单名称' })).toBeVisible()

  await page.goto('/system/user-center')
  await expect(page.getByRole('heading', { name: '个人中心' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '基本设置' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '登录会话' })).toBeVisible()
})

test('系统管理角色部门岗位工具栏可排序全屏和设置列', async ({ page }) => {
  test.setTimeout(90000)
  await login(page)

  for (const item of [
    { path: '/system/role', heading: '角色管理', column: '角色编码' },
    { path: '/system/department', heading: '部门管理', column: '部门编码' },
    { path: '/system/post', heading: '岗位管理', column: '岗位编码' }
  ]) {
    await page.goto(item.path)
    await expect(page.getByRole('heading', { name: item.heading })).toBeVisible()

    await page.getByRole('button', { name: '排序' }).click()
    await page.getByRole('button', { name: '全屏' }).click()
    await expect(page.getByRole('button', { name: '退出全屏' })).toBeVisible()
    await page.getByRole('button', { name: '退出全屏' }).click()
    await expect(page.getByRole('button', { name: '全屏' })).toBeVisible()

    await page.getByRole('button', { name: '列设置' }).click()
    await expect(page.getByRole('heading', { name: '列设置' })).toBeVisible()
    await page.locator('.el-drawer').getByText(item.column, { exact: true }).click()
    await expect(page.getByRole('columnheader', { name: item.column })).toHaveCount(0)
    await page.locator('.el-drawer').getByText(item.column, { exact: true }).click()
    await expect(page.getByRole('columnheader', { name: item.column })).toBeVisible()
    await page.getByRole('button', { name: '关闭此对话框' }).click()
  }

  await page.goto('/system/department')
  await expect(page.locator('.row-actions.three').first()).toBeVisible()
})

test('系统管理菜单日志和个人中心关键交互可用', async ({ page }) => {
  test.setTimeout(90000)
  await login(page)

  await page.goto('/system/menu')
  await expect(page.getByRole('heading', { name: '菜单管理' })).toBeVisible()
  await page.getByRole('button', { name: '排序' }).click()
  await page.getByRole('button', { name: '全屏' }).click()
  await expect(page.getByRole('button', { name: '退出全屏' })).toBeVisible()
  await page.getByRole('button', { name: '退出全屏' }).click()
  await expect(page.getByRole('button', { name: '全屏' })).toBeVisible()
  await page.getByRole('button', { name: '列设置' }).click()
  await expect(page.getByRole('heading', { name: '列设置' })).toBeVisible()
  await page.locator('.el-drawer').getByText('权限标识', { exact: true }).click()
  await expect(page.getByRole('columnheader', { name: '权限标识' })).toHaveCount(0)
  await page.locator('.el-drawer').getByText('权限标识', { exact: true }).click()
  await page.getByRole('button', { name: '关闭此对话框' }).click()

  for (const item of [
    { path: '/system/login-log', heading: '登录日志' },
    { path: '/system/operation-log', heading: '操作日志' }
  ]) {
    await page.goto(item.path)
    await expect(page.getByRole('heading', { name: item.heading })).toBeVisible()
    await expect(page.getByRole('button', { name: '清空' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: '导出' })).toBeVisible()
    await page.getByRole('button', { name: '全屏' }).click()
    await expect(page.getByRole('button', { name: '退出全屏' })).toBeVisible()
    await page.getByRole('button', { name: '退出全屏' }).click()
  }

  await page.goto('/system/user-center')
  await expect(page.getByRole('heading', { name: '个人中心' })).toBeVisible()
  await expect(page.getByText('头像地址')).toBeVisible()
  await expect(page.getByText('封面地址')).toBeVisible()
  await expect(page.getByText('所属部门')).toBeVisible()
  await expect(page.locator('.session-pagination')).toBeVisible()
})
