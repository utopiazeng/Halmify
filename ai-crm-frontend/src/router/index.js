import { createRouter, createWebHistory } from 'vue-router'
import { navigationRoutes, routes } from './routes'
import api from '../api'
import { ensureLocaleMessages, i18nState } from '../i18n'
import { currentUser, enabledModules, hasPermission, isAdminUser, isPlatformAdminUser, isTenantAdminUser, userPermissions } from '../permissions'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

router.beforeEach(async (to, from, next) => {
  let industryPackages = null
  if (to.path === '/dashboard') {
    next('/workspace')
    return
  }
  if (to.meta.requiresAuth && !localStorage.getItem('token')) {
    next('/')
    return
  }
  if (to.meta.requiresAuth) {
    await ensureLocaleMessages(i18nState.locale)
  }
  if (to.meta.requiresAuth) {
    await refreshCurrentUser()
  }
  if (to.meta.requiresAuth && !localStorage.getItem('token')) {
    next('/')
    return
  }
  if (to.meta.requiresAuth && !isPlatformAdminUser()) {
    industryPackages = await ensureIndustryPackages(Boolean(to.meta.industryPackage))
  }
  if (to.meta.requiresAuth && isPlatformAdminUser()) {
    if (isTenantBusinessRoute(to.path)) {
      next('/platform/overview')
      return
    }
  }
  if (to.meta.requiresAuth && !canAccess(to.meta)) {
    const firstRoute = navigationRoutes.find((item) => canAccess(item.meta))
    if (isPlatformAdminUser()) {
      next(firstRoute?.path || '/platform/overview')
      return
    }
    next(firstRoute?.path || '/workspace')
    return
  }
  if (to.meta.requiresAuth && to.meta.industryPackage && !isIndustryPackageEnabled(to.meta.industryPackage, industryPackages)) {
    next('/workspace')
    return
  }
  next()
})

let refreshPromise = null
async function refreshCurrentUser() {
  if (refreshPromise) return refreshPromise
  refreshPromise = api.get('/api/v1/auth/me')
    .then((res) => {
      if (res?.data) {
        localStorage.setItem('current_user', JSON.stringify(res.data))
      }
    })
    .catch((error) => {
      if (error?.response?.status === 401) {
        localStorage.removeItem('token')
        localStorage.removeItem('current_user')
      }
    })
    .finally(() => {
      refreshPromise = null
    })
  return refreshPromise
}

let industryPackagePromise = null
async function ensureIndustryPackages(force = false) {
  if (!force && localStorage.getItem('industry_packages_loaded') === '1') return cachedIndustryPackages()
  if (industryPackagePromise) return industryPackagePromise
  industryPackagePromise = api.get('/api/v1/prd/extensions/manifest')
    .then((res) => {
      const manifest = industryManifestPayload(res)
      const packages = Array.isArray(manifest?.packages) ? manifest.packages : []
      localStorage.setItem('industry_packages', JSON.stringify(packages))
      localStorage.setItem('industry_packages_loaded', '1')
      return packages
    })
    .catch(() => {
      localStorage.setItem('industry_packages', '[]')
      localStorage.setItem('industry_packages_loaded', '1')
      return []
    })
    .finally(() => {
      industryPackagePromise = null
    })
  return industryPackagePromise
}

export function canAccess(metaOrPermission) {
  const meta = typeof metaOrPermission === 'string' ? { permission: metaOrPermission } : (metaOrPermission || {})
  if (meta.systemSection === 'menu' && !isPlatformAdminUser()) {
    return false
  }
  const permissions = userPermissions()
  const modules = new Set(enabledModules())
  const platformAdmin = isPlatformAdminUser()
  if (meta.group === 'platform' && !platformAdmin) {
    return false
  }
  if (meta.group === 'admin' && !isTenantAdminUser()) {
    return false
  }
  if (platformAdmin) {
    if (meta.group === 'platform') return true
    return false
  }
  if (meta.domain && modules.size > 0 && !modules.has(meta.domain)) {
    return false
  }
  if (isAdminUser()) return true
  if (meta.industryPackage) return true
  if (meta.domain && modules.has(meta.domain) && !isRestrictedTenantRoute(meta)) return true
  if (!meta.permission || permissions.includes('*')) return true
  if (meta.menuPermission) {
    return hasPermission(meta.menuPermission) || hasPermission(meta.permission)
  }
  return hasPermission(meta.permission)
}

function isRestrictedTenantRoute(meta) {
  if (!meta) return true
  if (meta.group === 'admin') return true
  if (meta.tenantRequired) return true
  if (meta.systemSection) return true
  return false
}

function isTenantBusinessRoute(path) {
  return [
    '/workspace', '/projects', '/products', '/campaigns', '/contacts', '/companies', '/leads',
    '/opportunities', '/deals', '/orders', '/tickets', '/dashboards', '/teams', '/accounts', '/insights', '/im', '/storage', '/crm-suite/'
  ].some((prefix) => path.startsWith(prefix))
}

function isIndustryPackageEnabled(packageCode, packages) {
  if (!packageCode) return false
  const source = Array.isArray(packages) ? packages : cachedIndustryPackages()
  return source.some((item) => item?.code === packageCode)
}

function cachedIndustryPackages() {
  try {
    const packages = JSON.parse(localStorage.getItem('industry_packages') || '[]')
    return Array.isArray(packages) ? packages : []
  } catch {
    return []
  }
}

function industryManifestPayload(res) {
  if (res?.data?.packages || res?.data?.menuExtensions) return res.data
  if (res?.packages || res?.menuExtensions) return res
  return {}
}

export default router
