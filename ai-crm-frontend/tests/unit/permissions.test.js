import assert from 'node:assert/strict'
import { beforeEach, describe, it } from 'node:test'

const store = new Map()

globalThis.localStorage = {
  getItem(key) {
    return store.has(key) ? store.get(key) : null
  },
  setItem(key, value) {
    store.set(key, String(value))
  },
  removeItem(key) {
    store.delete(key)
  },
  clear() {
    store.clear()
  }
}

const permissions = await import('../../src/permissions.js')

function setUser(user) {
  localStorage.setItem('current_user', JSON.stringify(user))
}

describe('permissions helpers', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('returns safe defaults when current_user is missing or invalid', () => {
    assert.deepEqual(permissions.currentUser(), {})
    assert.deepEqual(permissions.userPermissions(), [])
    assert.deepEqual(permissions.enabledModules(), [])
    assert.equal(permissions.isPlatformAdminUser(), false)
    assert.equal(permissions.isTenantAdminUser(), false)

    localStorage.setItem('current_user', '{broken-json')
    assert.deepEqual(permissions.currentUser(), {})
    assert.deepEqual(permissions.userPermissions(), [])
    assert.deepEqual(permissions.enabledModules(), [])
  })

  it('detects platform administrators from explicit flag, account type, role and tenant boundary', () => {
    setUser({ is_platform_admin: true, tenant_id: 123 })
    assert.equal(permissions.isPlatformAdminUser(), true)

    setUser({ account_type: 'super_admin', tenant_id: 123 })
    assert.equal(permissions.isPlatformAdminUser(), true)

    setUser({ role: 'platform_super_admin', tenant_id: 123 })
    assert.equal(permissions.isPlatformAdminUser(), true)

    setUser({ account_type: 'admin', tenant_id: 0 })
    assert.equal(permissions.isPlatformAdminUser(), true)

    setUser({ account_type: 'admin', tenant_id: 1 })
    assert.equal(permissions.isPlatformAdminUser(), false)
  })

  it('allows platform and tenant admins to pass permission checks', () => {
    setUser({ account_type: 'tenant_admin', tenant_id: 10, permissions: [] })
    assert.equal(permissions.isAdminUser(), true)
    assert.equal(permissions.hasPermission('leads:delete'), true)
    assert.equal(permissions.hasExactPermission('orders:export'), true)

    setUser({ account_type: 'platform_super_admin', tenant_id: 0, permissions: [] })
    assert.equal(permissions.isAdminUser(), true)
    assert.equal(permissions.hasPermission('platform:settings'), true)
  })

  it('checks exact permissions and wildcard permissions for normal users', () => {
    setUser({ tenant_id: 10, permissions: ['leads:view'] })
    assert.equal(permissions.hasPermission('leads:view'), true)
    assert.equal(permissions.hasExactPermission('leads:view'), true)
    assert.equal(permissions.hasPermission('leads:create'), false)
    assert.equal(permissions.hasExactPermission('leads:create'), false)

    setUser({ tenant_id: 10, permissions: ['*'] })
    assert.equal(permissions.hasPermission('orders:delete'), true)
    assert.equal(permissions.hasExactPermission('orders:delete'), true)
  })

  it('uses domain manage permissions as action fallback without treating manage as exact implicit permission', () => {
    setUser({ tenant_id: 10, permissions: ['leads:manage'] })
    assert.equal(permissions.hasPermission('leads:view'), true)
    assert.equal(permissions.hasPermission('leads:create'), true)
    assert.equal(permissions.hasPermission('leads:update'), true)
    assert.equal(permissions.hasPermission('leads:delete'), true)
    assert.equal(permissions.hasPermission('leads:export'), true)
    assert.equal(permissions.hasPermission('leads:assign'), true)
    assert.equal(permissions.hasPermission('leads:approve'), true)
    assert.equal(permissions.hasPermission('contacts:view'), false)
    assert.equal(permissions.hasExactPermission('leads:view'), false)
    assert.equal(permissions.canViewDomain('leads'), true)
    assert.equal(permissions.canManageDomain('leads'), true)
    assert.equal(permissions.canViewDomain('contacts'), false)
  })

  it('returns enabled modules only when the user payload provides an array', () => {
    setUser({ enabled_modules: ['workspace', 'orders'] })
    assert.deepEqual(permissions.enabledModules(), ['workspace', 'orders'])

    setUser({ enabled_modules: 'workspace,orders' })
    assert.deepEqual(permissions.enabledModules(), [])
  })

  it('allows empty permission checks so callers can make optional route metadata', () => {
    setUser({ tenant_id: 10, permissions: [] })
    assert.equal(permissions.hasPermission(''), true)
    assert.equal(permissions.hasExactPermission(''), true)
  })
})
