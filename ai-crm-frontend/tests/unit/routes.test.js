import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

import { authRoutes, navigationRoutes, routes } from '../../src/router/routes.js'

describe('router metadata', () => {
  it('keeps route path and name values unique', () => {
    const paths = new Set()
    const names = new Set()

    for (const route of routes) {
      assert.equal(paths.has(route.path), false, `duplicate path: ${route.path}`)
      paths.add(route.path)

      assert.equal(names.has(route.name), false, `duplicate name: ${route.name}`)
      names.add(route.name)
    }
  })

  it('keeps auth routes public and navigation routes protected', () => {
    for (const route of authRoutes) {
      assert.notEqual(route.meta?.requiresAuth, true, `${route.path} should stay public`)
    }

    for (const route of navigationRoutes) {
      assert.equal(route.meta?.requiresAuth, true, `${route.path} must require auth`)
    }
  })

  it('defines access metadata for every protected navigation route', () => {
    for (const route of navigationRoutes) {
      const meta = route.meta || {}
      const hasAccessRule = Boolean(meta.permission || meta.group === 'platform' || meta.systemSection === 'user-center')
      assert.equal(hasAccessRule, true, `${route.path} must define permission, platform group or user-center exception`)
      assert.ok(meta.titleKey, `${route.path} must define titleKey for i18n`)
    }
  })

  it('keeps core tenant and platform routes registered', () => {
    const paths = new Set(navigationRoutes.map((route) => route.path))
    for (const path of [
      '/workspace',
      '/leads',
      '/contacts',
      '/companies',
      '/opportunities',
      '/deals',
      '/orders',
      '/tickets',
      '/projects',
      '/products',
      '/im',
      '/storage',
      '/platform/overview',
      '/platform/tenants',
      '/platform/plans',
      '/platform/runtime',
      '/platform/iot-firmwares'
    ]) {
      assert.equal(paths.has(path), true, `missing route: ${path}`)
    }
  })

  it('marks platform routes with platform group and platform permissions', () => {
    const platformRoutes = navigationRoutes.filter((route) => route.path.startsWith('/platform/'))
    assert.ok(platformRoutes.length > 0, 'expected platform routes')

    for (const route of platformRoutes) {
      assert.equal(route.meta?.group, 'platform', `${route.path} must stay in platform group`)
      assert.match(route.meta?.permission || '', /^platform:/, `${route.path} must use platform permission`)
    }
  })
})
