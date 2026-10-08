import { expect, test } from '@playwright/test'
import { loginByAPI } from './helpers/auth.js'

const userA = {
  username: process.env.E2E_IM_USER_A || process.env.E2E_USERNAME || '',
  password: process.env.E2E_IM_PASS_A || process.env.E2E_PASSWORD || '123456'
}

const userBEnv = {
  username: process.env.E2E_IM_USER_B || '',
  password: process.env.E2E_IM_PASS_B || '123456'
}

async function loginAWithFallback(page) {
  const usernames = []
  if (userA.username) usernames.push(userA.username)
  usernames.push('tenant_admin', 'utopia', 'admin')
  const dedup = Array.from(new Set(usernames.filter(Boolean)))
  const errors = []
  for (const username of dedup) {
    try {
      const session = await loginByAPI(page, username, userA.password)
      if (session?.user?.is_platform_admin) {
        errors.push(`login succeeded for ${username} but this is platform admin, cannot access tenant IM`)
        continue
      }
      return session
    } catch (error) {
      errors.push(String(error?.message || error))
    }
  }
  throw new Error(`all login attempts failed: ${errors.join(' | ')}`)
}

async function authedGet(page, token, url, params) {
  const resp = await page.request.get(url, {
    headers: { Authorization: `Bearer ${token}` },
    params
  })
  if (!resp.ok()) {
    const body = await resp.json().catch(() => ({}))
    throw new Error(`GET ${url} failed: status=${resp.status()} message=${body?.message || 'unknown'}`)
  }
  return resp.json()
}

async function authedPost(page, token, url, data) {
  const resp = await page.request.post(url, {
    headers: { Authorization: `Bearer ${token}` },
    data
  })
  if (!resp.ok()) {
    const body = await resp.json().catch(() => ({}))
    throw new Error(`POST ${url} failed: status=${resp.status()} message=${body?.message || 'unknown'}`)
  }
  return resp.json()
}

function hasIMPermission(role) {
  const raw = String(role?.permissions_json || role?.permissions || '')
  return raw.includes('*') || (raw.includes('im:view') && raw.includes('im:manage'))
}

async function ensureUserB(page, token) {
  if (userBEnv.username) {
    return { username: userBEnv.username, password: userBEnv.password }
  }
  // Auto-create B only when A has tenant management capability.
  const rolesBody = await authedGet(page, token, '/api/v1/rbac/roles')
  const roles = rolesBody?.data?.items || []
  expect(Array.isArray(roles)).toBeTruthy()
  expect(roles.length).toBeGreaterThan(0)
  const pickedRole = roles.find(hasIMPermission) || roles[0]
  const username = `im_e2e_${Date.now().toString(36)}`
  const password = 'E2eIm123456'
  const createBody = await authedPost(page, token, '/api/v1/accounts/users', {
    username,
    password,
    name: `IM E2E ${username}`,
    role_id: pickedRole.id,
    language_pref: 'en-US'
  })
  expect(createBody?.data?.id).toBeTruthy()
  return { id: createBody.data.id, username, password }
}

async function unreadTotal(page, token) {
  const body = await authedGet(page, token, '/api/v1/im/unread/summary')
  return Number(body?.data?.total_unread || 0)
}

async function authedPostPath(page, token, url, data) {
  return authedPost(page, token, url, data)
}

async function authedGetPath(page, token, url, params) {
  return authedGet(page, token, url, params)
}

test('IM A/B 单聊收发 + 未读 + 已读', async ({ browser }) => {
  test.setTimeout(180000)

  const contextA = await browser.newContext()
  const contextB = await browser.newContext()
  const pageA = await contextA.newPage()
  const pageB = await contextB.newPage()

  try {
    let loginA
    try {
      loginA = await loginAWithFallback(pageA)
    } catch (error) {
      test.skip(true, `缺少可用租户账号，请设置 E2E_IM_USER_A/E2E_IM_PASS_A。详情: ${error?.message || error}`)
      return
    }

    let userB
    try {
      userB = await ensureUserB(pageA, loginA.token)
    } catch (error) {
      test.skip(true, `无法准备第二账号，请设置 E2E_IM_USER_B/E2E_IM_PASS_B 或给A账号开通账号管理权限。详情: ${error?.message || error}`)
      return
    }

    const loginB = await loginByAPI(pageB, userB.username, userB.password, 'zh-CN')

    const message = `E2E_IM_${Date.now()}`
    const convBody = await authedPostPath(pageA, loginA.token, '/api/v1/im/conversations', { peer_id: userB.id || loginB.user?.id })
    const conversationID = convBody?.data?.conversation_id
    expect(conversationID).toBeTruthy()
    await authedPostPath(pageA, loginA.token, `/api/v1/im/conversations/${conversationID}/messages`, {
      type: 'text',
      content: message,
      client_msg_id: `e2e-${Date.now()}`
    })
    await expect.poll(async () => unreadTotal(pageB, loginB.token), { timeout: 20000 }).toBeGreaterThan(0)
    const messages = await authedGetPath(pageB, loginB.token, `/api/v1/im/conversations/${conversationID}/messages`, { limit: 20 })
    expect((messages?.data?.items || []).some((item) => item.content === message)).toBeTruthy()
    await authedPostPath(pageB, loginB.token, `/api/v1/im/conversations/${conversationID}/read`, {})
    await expect.poll(async () => unreadTotal(pageB, loginB.token), { timeout: 20000 }).toBe(0)

    await pageA.goto('/im')
    await pageB.goto('/im')
    await expect(pageA.locator('.im-page')).toBeVisible()
    await expect(pageB.locator('.im-page')).toBeVisible()
    await pageA.reload()
    await expect(pageA.locator('.im-conv-row').first()).toBeVisible({ timeout: 15000 })
  } finally {
    await contextA.close()
    await contextB.close()
  }
})
