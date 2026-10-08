<template>
  <el-config-provider :locale="elementLocale" :z-index="3000" :button="{ autoInsertSpace: true }">
    <div class="app-root">
      <div v-if="showLayout" class="art-layout art-layout-left" :class="{ collapsed }">
        <aside class="art-sidebar">
          <button class="art-logo" @click="router.push('/workspace')">
            <img class="art-logo-image" src="/halmify-logo-horizontal.svg" :alt="BRAND_NAME" />
          </button>

          <div class="art-menu-scroll">
            <el-menu
              v-for="group in groupedPrimaryRoutes"
              :key="group.key"
              :default-active="selectedKey"
              :collapse="collapsed"
              class="art-menu"
              router
              @select="onMenuSelect"
            >
              <div v-if="!collapsed" class="sider-section">{{ group.title }}</div>
              <template v-for="item in group.routes" :key="item.path">
                <el-sub-menu v-if="item.children?.length" :index="item.path">
                  <template #title>
                    <el-icon><component :is="item.meta.icon" /></el-icon>
                    <span>{{ routeTitle(item) }}</span>
                  </template>
                  <el-menu-item v-for="child in item.children" :key="child.path" :index="child.path">
                    <template #title>{{ routeTitle(child) }}</template>
                  </el-menu-item>
                </el-sub-menu>
                <el-menu-item v-else :index="item.path">
                  <el-icon><component :is="item.meta.icon" /></el-icon>
                  <template #title>{{ routeTitle(item) }}</template>
                </el-menu-item>
              </template>
            </el-menu>

            <div v-if="platformRoutes.length && !collapsed" class="sider-section">{{ t('app.platformCapabilities') }}</div>
            <el-menu
              v-if="platformRoutes.length"
              :default-active="selectedKey"
              :collapse="collapsed"
              class="art-menu"
              router
              @select="onMenuSelect"
            >
              <el-menu-item v-for="item in platformRoutes" :key="item.path" :index="item.path">
                <el-icon><component :is="item.meta.icon" /></el-icon>
                <template #title>{{ routeTitle(item) }}</template>
              </el-menu-item>
            </el-menu>
          </div>
        </aside>

        <main class="art-main">
          <header class="art-header">
            <div class="header-left">
              <ArtIconButton
                :icon="collapsed ? 'ri:menu-unfold-line' : 'ri:menu-fold-line'"
                :label="t('app.toggleMenu')"
                @click="collapsed = !collapsed"
              />
              <el-breadcrumb separator="/">
                <el-breadcrumb-item>{{ BRAND_NAME }}</el-breadcrumb-item>
                <el-breadcrumb-item>{{ currentTitle }}</el-breadcrumb-item>
              </el-breadcrumb>
              <div class="page-heading">
                <h1>{{ currentTitle }}</h1>
                <p>{{ currentDescription }}</p>
              </div>
            </div>

            <div class="header-actions">
              <el-tooltip :content="t('domain.auto.925f26b9e1')">
                <button type="button" class="header-mail-trigger" :aria-label="t('domain.auto.925f26b9e1')" @click="goSystemMessages">
                  <el-badge :value="systemUnreadCount" :hidden="!systemUnreadCount" :max="99">
                    <Icon icon="ri:mail-line" />
                  </el-badge>
                </button>
              </el-tooltip>
              <el-dropdown trigger="click" @command="changeLocale">
                <button type="button" class="header-locale-trigger" :aria-label="t('language')">
                  <Icon icon="ri:global-line" />
                </button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                      v-for="item in localeOptions"
                      :key="item.value"
                      :command="item.value"
                      :class="{ 'is-active-locale': item.value === i18nState.locale }"
                    >
                      {{ item.label }}
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              <el-popover
                v-model:visible="userPopoverVisible"
                placement="bottom-end"
                :width="280"
                trigger="click"
                popper-class="user-profile-popover"
              >
                <template #reference>
                  <div class="user-entry">
                    <el-avatar :size="30" class="user-avatar" :src="currentUserAvatar">{{ userAvatarText }}</el-avatar>
                    <span class="user-name">{{ currentUsername }}</span>
                  </div>
                </template>
                <div class="user-popover-card">
                  <div class="user-popover-head">
                    <el-avatar :size="42" class="user-avatar" :src="currentUserAvatar">{{ userAvatarText }}</el-avatar>
                    <div class="user-popover-meta">
                      <strong>{{ currentUsername }}</strong>
                      <p>{{ currentUserEmail || '-' }}</p>
                    </div>
                  </div>
                  <div class="user-popover-actions">
                    <button type="button" class="user-popover-action-btn" @click="goProfileCenter">
                      <Icon icon="ri:user-line" />
                      <span>{{ t('app.profileCenter') }}</span>
                    </button>
                    <button type="button" class="user-popover-action-btn" @click="openPasswordDialog">
                      <Icon icon="ri:key-2-line" />
                      <span>{{ t('app.changePassword') }}</span>
                    </button>
                    <button type="button" class="user-popover-logout-btn" @click="requestLogout">
                      {{ t('logout') }}
                    </button>
                  </div>
                </div>
              </el-popover>
            </div>
          </header>

          <div
            v-if="showMaintenanceNotice"
            class="crm-maintenance-notice app-maintenance-notice"
            role="status"
            aria-live="polite"
            :dir="maintenanceNotice.locale === 'ar-SA' ? 'rtl' : 'ltr'"
          >
            <div class="crm-maintenance-icon" aria-hidden="true">!</div>
            <div class="crm-maintenance-body">
              <span>{{ maintenanceNotice.label }}</span>
              <strong>{{ maintenanceNotice.title }}</strong>
              <p>{{ maintenanceNotice.text }}</p>
            </div>
          </div>

          <ArtPageContent>
            <router-view />
          </ArtPageContent>
        </main>
      </div>

      <div v-else class="login-layout">
        <router-view />
      </div>

      <el-dialog
        v-model="passwordDialogVisible"
        :title="t('app.passwordDialogTitle')"
        width="440px"
        destroy-on-close
      >
        <el-form label-position="top">
          <el-form-item :label="t('app.currentPassword')">
            <el-input
              v-model="passwordForm.current_password"
              type="password"
              show-password
              :placeholder="t('app.currentPasswordPlaceholder')"
              @keyup.enter="submitPasswordChange"
            />
          </el-form-item>
          <el-form-item :label="t('app.newPassword')">
            <el-input
              v-model="passwordForm.new_password"
              type="password"
              show-password
              :placeholder="t('app.newPasswordPlaceholder')"
              @keyup.enter="submitPasswordChange"
            />
          </el-form-item>
          <el-form-item :label="t('app.confirmPassword')">
            <el-input
              v-model="passwordForm.confirm_password"
              type="password"
              show-password
              :placeholder="t('app.confirmPasswordPlaceholder')"
              @keyup.enter="submitPasswordChange"
            />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="closePasswordDialog">{{ t('common.action.cancel') }}</el-button>
          <el-button type="primary" :loading="passwordSubmitting" @click="submitPasswordChange">
            {{ t('app.changePassword') }}
          </el-button>
        </template>
      </el-dialog>

    </div>
  </el-config-provider>
</template>

<script setup>
import { computed, inject, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus/es/components/message/index'
import { Icon } from '@iconify/vue'
import { navigationRoutes } from './router/routes'
import { canAccess } from './router'
import { hasPermission, isAdminUser, isPlatformAdminUser } from './permissions'
import { i18nState, localeOptions, setLocale, t } from './i18n'
import api from './api'
import { BRAND_NAME } from './brand'
import ArtIconButton from './components/art/ArtIconButton.vue'
import ArtPageContent from './components/art/ArtPageContent.vue'
import { fetchCompanySettings } from './utils/company-format'
import { resolveProtectedImageUrl, revokeObjectUrl } from './utils/protectedImage'
import { Monitor } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const elementLocales = inject('elementLocales', {})
const collapsed = ref(false)
const backendMenus = ref([])
const industryManifest = ref(emptyIndustryManifest())
const authVersion = ref(0)
const passwordDialogVisible = ref(false)
const passwordSubmitting = ref(false)
const userPopoverVisible = ref(false)
const currentUser = ref(readCurrentUser())
const currentUserAvatarDisplay = ref('')
const systemUnreadCount = ref(0)
const maintenanceNotice = ref(null)
const SYSTEM_UNREAD_FALLBACK_INTERVAL_MS = 10 * 60 * 1000
let systemUnreadTimer = null
let systemUnreadSocket = null
let systemUnreadReconnectTimer = null
let systemUnreadRefreshTimer = null
let systemUnreadReconnectDelay = 1000
const passwordForm = reactive({
  current_password: '',
  new_password: '',
  confirm_password: ''
})
const hiddenManufacturingMenuKeys = new Set(['mfg_home', 'mfg_asset_service', 'mfg_analytics'])
const hiddenManufacturingMenuPaths = new Set(['/manufacturing/home', '/manufacturing/asset-service', '/manufacturing/analytics'])

const showLayout = computed(() => route.meta.layout !== 'auth')
const selectedKey = computed(() => route.path)
const visibleRoutes = computed(() => {
  // Ensure menu recalculates when login state in localStorage changes.
  void authVersion.value
  const base = [
    ...navigationRoutes.filter((item) => !item.meta.hidden && canAccess(item.meta)),
    ...industryMenuRoutes.value
  ]
  if (isPlatformAdminUser()) return base.filter((item) => item.meta.group === 'platform')
  if (isAdminUser()) return base
  if (!backendMenus.value.length) return base
  const menuPaths = new Set(flattenMenus(backendMenus.value).filter((item) => item.path && item.type !== 'button').map((item) => item.path))
  const systemRoutes = base.filter((item) => item.path.startsWith('/system/') && menuPaths.has(item.path))
  const nonSystemRoutes = base.filter((item) => !item.path.startsWith('/system/'))
  return [...nonSystemRoutes, ...systemRoutes]
})
const primaryRoutes = computed(() => visibleRoutes.value.filter((item) => item.meta.group !== 'platform'))
const platformRoutes = computed(() => sortRoutes(visibleRoutes.value.filter((item) => item.meta.group === 'platform')))
const industryMenuRoutes = computed(() => {
  const extensions = Array.isArray(industryManifest.value?.menuExtensions) ? industryManifest.value.menuExtensions : []
  return extensions
    .filter((item) => item?.path)
    .map((item) => industryRouteFromManifest(item))
})
const menuGroups = [
  { key: 'workbench', title: t('domain.auto.a5361c67df') },
  { key: 'customer', title: t('domain.auto.560ddea4f1') },
  { key: 'sales', title: t('domain.auto.8bb389d54a') },
  { key: 'service', title: t('domain.auto.36f2e259de') },
  { key: 'operation', title: t('domain.auto.0031fc6c70') },
  { key: 'manufacturing', title: t('industry.manufacturing.menu.title') },
  { key: 'admin', title: t('domain.auto.ac7873286d') }
]
const groupedPrimaryRoutes = computed(() => menuGroups
  .map((group) => ({
    ...group,
    title: group.key === 'manufacturing' ? t('industry.manufacturing.menu.title') : t(`menuGroup.${group.key}`),
    routes: sortRoutes(primaryRoutes.value.filter((item) => (item.meta.group || 'workbench') === group.key))
  }))
  .filter((group) => group.routes.length))
const currentTitle = computed(() => translatedMeta(route.meta, 'title') || t(t('domain.auto.016834c9ae')))
const currentDescription = computed(() => translatedMeta(route.meta, 'description') || t('app.defaultDescription'))
const elementLocale = computed(() => elementLocales[i18nState.locale] || elementLocales['zh-CN'])
const showMaintenanceNotice = computed(() => Boolean(maintenanceNotice.value?.visible && maintenanceNotice.value?.text))
const currentUsername = computed(() => {
  const user = currentUser.value || {}
  return user.username || user.name || '-'
})
const currentUserEmail = computed(() => {
  const user = currentUser.value || {}
  return user.email || ''
})
const currentUserAvatar = computed(() => {
  const user = currentUser.value || {}
  return currentUserAvatarDisplay.value || user.avatar_url || ''
})
const userAvatarText = computed(() => (currentUsername.value || 'U').slice(0, 1).toUpperCase())

const onMenuSelect = (index) => {
  if (index !== route.path) {
    router.push(index)
  }
}

const performLogout = () => {
  api.post('/api/v1/auth/logout').catch(() => {})
  localStorage.removeItem('token')
  localStorage.removeItem('current_user')
  localStorage.removeItem('industry_packages')
  localStorage.removeItem('industry_packages_loaded')
  currentUser.value = {}
  disconnectSystemUnreadSocket()
  authVersion.value += 1
  router.push('/')
}

async function requestLogout() {
  const { ElMessageBox } = await import('element-plus/es/components/message-box/index')
  try {
    await ElMessageBox.confirm(
      t('app.logoutConfirm'),
      t('app.logoutTitle'),
      {
        type: 'warning',
        confirmButtonText: t('logout'),
        cancelButtonText: t('common.action.cancel')
      }
    )
  } catch {
    return
  }
  userPopoverVisible.value = false
  performLogout()
}

function goProfileCenter() {
  userPopoverVisible.value = false
  router.push('/system/user-center')
}

async function refreshCurrentUserAvatar(options = {}) {
  let source = String(currentUser.value?.avatar_url || '').trim()
  if (!source && options.fetchProfile && localStorage.getItem('token')) {
    try {
      const res = await api.get('/api/v1/accounts/me/profile')
      const profile = res?.data || {}
      if (profile.avatar_url) {
        const nextUser = { ...(currentUser.value || {}), ...profile }
        localStorage.setItem('current_user', JSON.stringify(nextUser))
        currentUser.value = nextUser
        source = String(profile.avatar_url || '').trim()
      }
    } catch {
      // keep cached user info if profile refresh fails
    }
  }
  const previous = currentUserAvatarDisplay.value
  if (!source) {
    currentUserAvatarDisplay.value = ''
    revokeObjectUrl(previous)
    return
  }
  try {
    const next = await resolveProtectedImageUrl(source)
    currentUserAvatarDisplay.value = next
    if (previous && previous !== next) revokeObjectUrl(previous)
  } catch {
    currentUserAvatarDisplay.value = ''
    revokeObjectUrl(previous)
  }
}

function goSystemMessages() {
  if (isPlatformAdminUser()) {
    router.push('/platform/messages')
    return
  }
  router.push({ path: '/im', query: { tab: 'system' } })
}

async function changeLocale(locale) {
  if (!locale || locale === i18nState.locale) return
  await setLocale(locale)
  await loadMaintenanceNotice()
  window.location.reload()
}

function readCurrentUser() {
  try {
    return JSON.parse(localStorage.getItem('current_user') || '{}')
  } catch {
    return {}
  }
}

function syncCurrentUserFromStorage() {
  currentUser.value = readCurrentUser()
  refreshCurrentUserAvatar({ fetchProfile: true })
}

function handleCurrentUserUpdated() {
  syncCurrentUserFromStorage()
  authVersion.value += 1
  localStorage.removeItem('industry_packages_loaded')
  loadIndustryManifest()
  disconnectSystemUnreadSocket()
  connectSystemUnreadSocket()
}

function flattenMenus(items) {
  return (items || []).flatMap((item) => [item, ...flattenMenus(item.children || [])])
}

function sortRoutes(items) {
  return [...items].sort((a, b) => Number(a.meta.order || 999) - Number(b.meta.order || 999))
}

function translatedMeta(meta, field) {
  if (!meta) return ''
  const keyField = field === 'title' ? 'titleKey' : 'descKey'
  if (meta[keyField]) return t(meta[keyField])
  return meta[field] || ''
}

function routeTitle(item) {
  return translatedMeta(item?.meta, 'title')
}

function industryRouteFromManifest(item) {
  const isManufacturingRoot = (item.key === 'manufacturing' || item.path === '/manufacturing')
  const rawChildren = Array.isArray(item.children) && item.children.length ? item.children : (isManufacturingRoot ? defaultManufacturingMenuChildren() : [])
  const children = isManufacturingRoot ? filterManufacturingMenuChildren(rawChildren) : rawChildren
  const route = {
    path: item.path,
    name: `industry-${item.key || item.path}`,
    meta: {
      titleKey: item.titleKey,
      descKey: item.descKey,
      title: item.titleKey,
      description: item.descKey,
      icon: industryIcon(item.icon),
      group: isManufacturingRoot ? 'manufacturing' : (item.group || 'manufacturing'),
      order: Number(item.order || 90),
      industryPackage: item.packageCode || 'manufacturing',
      permission: item.permission || 'mfg:view'
    }
  }
  route.children = children
    .filter((child) => child?.path)
    .map((child) => ({
      path: child.path,
      name: `industry-${child.key || child.path}`,
      meta: {
        titleKey: child.titleKey,
        descKey: child.descKey,
        title: child.titleKey,
        description: child.descKey,
        icon: industryIcon(child.icon || item.icon),
        group: item.group || 'manufacturing',
        order: Number(child.order || 90),
        industryPackage: child.packageCode || item.packageCode || 'manufacturing',
        permission: child.permission || item.permission || 'mfg:view',
        navMode: child.navMode || item.navMode || '',
        workflowGroup: child.workflowGroup || '',
        beginnerVisible: child.beginnerVisible !== false,
        legacyModules: Array.isArray(child.legacyModules) ? child.legacyModules : []
      }
    }))
    .filter((child) => isManufacturingRoot || canAccess(child.meta))
    .sort((a, b) => Number(a.meta.order || 999) - Number(b.meta.order || 999))
  return route
}

function filterManufacturingMenuChildren(children) {
  return (children || [])
    .filter((child) => !hiddenManufacturingMenuKeys.has(child?.key) && !hiddenManufacturingMenuPaths.has(child?.path))
    .map((child) => ({
      ...child,
      children: Array.isArray(child.children) ? filterManufacturingMenuChildren(child.children) : child.children
    }))
}

function defaultManufacturingMenuChildren() {
  return [
    { key: 'mfg_quote_to_order', path: '/manufacturing/quote-to-order', titleKey: 'industry.manufacturing.workflow.quoteToOrder', descKey: 'industry.manufacturing.workflow.quoteToOrderDesc', permission: 'mfg:rfq:view', order: 2, navMode: 'workflow', workflowGroup: 'quote_to_order', beginnerVisible: true, legacyModules: ['rfqs', 'clarifications', 'bom-snapshots', 'cost-sheets', 'quotes'] },
    { key: 'mfg_sample_to_mass', path: '/manufacturing/sample-to-mass', titleKey: 'industry.manufacturing.workflow.sampleToMass', descKey: 'industry.manufacturing.workflow.sampleToMassDesc', permission: 'mfg:sample:view', order: 3, navMode: 'workflow', workflowGroup: 'sample_to_mass', beginnerVisible: true, legacyModules: ['samples', 'production'] },
    { key: 'mfg_delivery_change', path: '/manufacturing/delivery-change', titleKey: 'industry.manufacturing.workflow.deliveryChange', descKey: 'industry.manufacturing.workflow.deliveryChangeDesc', permission: 'mfg:production:view', order: 4, navMode: 'workflow', workflowGroup: 'delivery_change', beginnerVisible: true, legacyModules: ['ecns', 'compensation-records'] },
    { key: 'mfg_settings', path: '/manufacturing/settings', titleKey: 'industry.manufacturing.workflow.settings', descKey: 'industry.manufacturing.workflow.settingsDesc', permission: 'mfg:product:view', order: 7, navMode: 'workflow', workflowGroup: 'settings', beginnerVisible: true, legacyModules: ['product-families', 'bom-templates'] }
  ]
}

async function loadBackendMenus() {
  if (!localStorage.getItem('token')) {
    backendMenus.value = []
    return
  }
  if (isPlatformAdminUser()) {
    backendMenus.value = []
    return
  }
  try {
    const res = await api.get('/api/v1/system/menus', { params: { visible: 1 } })
    backendMenus.value = res.data?.items || []
  } catch {
    backendMenus.value = []
  }
}

async function loadIndustryManifest() {
  if (!localStorage.getItem('token') || isPlatformAdminUser()) {
    industryManifest.value = emptyIndustryManifest()
    localStorage.setItem('industry_packages', '[]')
    localStorage.setItem('industry_packages_loaded', '1')
    return
  }
  try {
    const res = await api.get('/api/v1/prd/extensions/manifest')
    industryManifest.value = normalizeIndustryManifest(industryManifestPayload(res))
    localStorage.setItem('industry_packages', JSON.stringify(industryManifest.value.packages || []))
    localStorage.setItem('industry_packages_loaded', '1')
  } catch {
    industryManifest.value = emptyIndustryManifest()
    localStorage.setItem('industry_packages', '[]')
    localStorage.setItem('industry_packages_loaded', '1')
  }
}

function emptyIndustryManifest() {
  return {
    packages: [],
    menuExtensions: [],
    pageTabs: {},
    objectActions: {},
    dashboardWidgets: []
  }
}

function normalizeIndustryManifest(value) {
  return {
    ...emptyIndustryManifest(),
    ...(value || {}),
    packages: Array.isArray(value?.packages) ? value.packages : [],
    menuExtensions: Array.isArray(value?.menuExtensions) ? value.menuExtensions : [],
    pageTabs: value?.pageTabs || {},
    objectActions: value?.objectActions || {},
    dashboardWidgets: Array.isArray(value?.dashboardWidgets) ? value.dashboardWidgets : []
  }
}

function industryManifestPayload(res) {
  if (res?.data?.packages || res?.data?.menuExtensions) return res.data
  if (res?.packages || res?.menuExtensions) return res
  return {}
}

function industryIcon(name) {
  if (name === 'Monitor') return Monitor
  return Monitor
}

async function loadMaintenanceNotice() {
  try {
    const res = await api.get('/api/public/maintenance-notice', {
      params: { locale: i18nState.locale }
    })
    maintenanceNotice.value = res?.data || null
  } catch {
    maintenanceNotice.value = null
  }
}

async function loadTenantCompanySettings() {
  if (!localStorage.getItem('token') || isPlatformAdminUser()) return
  if (!hasPermission('enterprise:view') && !hasPermission('enterprise:manage')) return
  await fetchCompanySettings()
}

async function loadSystemUnread() {
  if (!canUseTenantIM() && !canUsePlatformIM()) {
    systemUnreadCount.value = 0
    return
  }
  try {
    const res = await api.get('/api/v1/im/system-notifications/unread-count')
    systemUnreadCount.value = Number(res.data?.unread || 0)
  } catch {
    systemUnreadCount.value = 0
  }
}

function scheduleSystemUnreadRefresh(delay = 0) {
  if (systemUnreadRefreshTimer) return
  systemUnreadRefreshTimer = setTimeout(() => {
    systemUnreadRefreshTimer = null
    void loadSystemUnread()
  }, delay)
}

function startSystemUnreadPolling() {
  if (systemUnreadTimer) clearInterval(systemUnreadTimer)
  // WebSocket pushes unread changes in real time; this is only a low-frequency fallback.
  systemUnreadTimer = setInterval(() => scheduleSystemUnreadRefresh(), SYSTEM_UNREAD_FALLBACK_INTERVAL_MS)
}

function connectSystemUnreadSocket() {
  const token = localStorage.getItem('token')
  if (!token || (!canUseTenantIM() && !canUsePlatformIM()) || systemUnreadSocket) return
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const ws = new WebSocket(`${protocol}//${window.location.host}/api/v1/im/ws?token=${encodeURIComponent(token)}`)
  let wasOpen = false
  systemUnreadSocket = ws
  ws.onopen = () => {
    wasOpen = true
    systemUnreadReconnectDelay = 1000
    window.__AI_CRM_IM_WS_CONNECTED = true
    window.dispatchEvent(new CustomEvent('im-ws-state', { detail: { connected: true } }))
  }
  ws.onmessage = (event) => {
    try {
      const payload = JSON.parse(event.data)
      window.dispatchEvent(new CustomEvent('im-ws-event', { detail: payload }))
      if (payload?.event === 'im.system.unread') {
        systemUnreadCount.value = Number(payload.data?.unread || 0)
      }
    } catch {
      // Ignore malformed websocket payloads.
    }
  }
  ws.onclose = () => {
    if (systemUnreadSocket === ws) systemUnreadSocket = null
    window.__AI_CRM_IM_WS_CONNECTED = false
    window.dispatchEvent(new CustomEvent('im-ws-state', { detail: { connected: false } }))
    if (wasOpen) scheduleSystemUnreadRefresh(1000)
    if (!localStorage.getItem('token') || (!canUseTenantIM() && !canUsePlatformIM())) return
    clearTimeout(systemUnreadReconnectTimer)
    systemUnreadReconnectTimer = setTimeout(connectSystemUnreadSocket, systemUnreadReconnectDelay)
    systemUnreadReconnectDelay = Math.min(systemUnreadReconnectDelay * 2, 30000)
  }
  ws.onerror = () => {
    ws.close()
  }
}

function disconnectSystemUnreadSocket() {
  clearTimeout(systemUnreadReconnectTimer)
  systemUnreadReconnectTimer = null
  clearTimeout(systemUnreadRefreshTimer)
  systemUnreadRefreshTimer = null
  window.__AI_CRM_IM_WS_CONNECTED = false
  if (!systemUnreadSocket) return
  const socket = systemUnreadSocket
  systemUnreadSocket = null
  socket.close()
}

function canUseTenantIM() {
  if (!showLayout.value || !localStorage.getItem('token')) return false
  const user = currentUser.value || {}
  if (isPlatformAdminUser()) return false
  if (Number(user.tenant_id || 0) <= 0) return false
  const modules = Array.isArray(user.enabled_modules) ? user.enabled_modules : []
  return modules.length === 0 || modules.includes('im')
}

function canUsePlatformIM() {
  return showLayout.value && Boolean(localStorage.getItem('token')) && isPlatformAdminUser()
}

function openPasswordDialog() {
  userPopoverVisible.value = false
  passwordDialogVisible.value = true
}

async function refreshCurrentUser() {
  if (!localStorage.getItem('token')) return
  try {
    const res = await api.get('/api/v1/auth/me')
    if (res?.data) {
      localStorage.setItem('current_user', JSON.stringify(res.data))
      syncCurrentUserFromStorage()
      authVersion.value += 1
    }
  } catch {
    // ignore and use cached local user info
  }
}

async function maybeShowLoginWelcomeNotice() {
  if (sessionStorage.getItem('login_success_notice') !== '1') return
  sessionStorage.removeItem('login_success_notice')
  const { ElNotification } = await import('element-plus/es/components/notification/index')
  ElNotification.success({
    title: t('common.status.success'),
    message: t('app.loginWelcomeMessage'),
    position: 'top-right',
    duration: 2500
  })
}

function closePasswordDialog() {
  passwordDialogVisible.value = false
  passwordForm.current_password = ''
  passwordForm.new_password = ''
  passwordForm.confirm_password = ''
}

async function submitPasswordChange() {
  if (passwordSubmitting.value) return
  if (!passwordForm.current_password) {
    ElMessage.error(t('app.currentPasswordRequired'))
    return
  }
  if (!passwordForm.new_password) {
    ElMessage.error(t('app.newPasswordRequired'))
    return
  }
  if (passwordForm.new_password.length < 6) {
    ElMessage.error(t('app.passwordMinLength'))
    return
  }
  if (passwordForm.new_password !== passwordForm.confirm_password) {
    ElMessage.error(t('system.message.passwordMismatch'))
    return
  }
  passwordSubmitting.value = true
  try {
    await api.put('/api/v1/accounts/me/password', {
      current_password: passwordForm.current_password,
      new_password: passwordForm.new_password
    })
    ElMessage.success(t('app.passwordChangedRelogin'))
    closePasswordDialog()
    performLogout()
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || t('common.message.loadFailed'))
  } finally {
    passwordSubmitting.value = false
  }
}

watch(
  () => route.fullPath,
  async () => {
    syncCurrentUserFromStorage()
    maybeShowLoginWelcomeNotice()
    if (localStorage.getItem('token')) {
      syncCurrentUserFromStorage()
      scheduleSystemUnreadRefresh()
      connectSystemUnreadSocket()
    } else {
      disconnectSystemUnreadSocket()
    }
    authVersion.value += 1
    await loadBackendMenus()
    await loadIndustryManifest()
  }
)

onMounted(async () => {
  window.addEventListener('current-user-updated', handleCurrentUserUpdated)
  await loadMaintenanceNotice()
  await loadTenantCompanySettings()
  syncCurrentUserFromStorage()
  maybeShowLoginWelcomeNotice()
  authVersion.value += 1
  syncCurrentUserFromStorage()
  scheduleSystemUnreadRefresh()
  await loadBackendMenus()
  await loadIndustryManifest()
  setTimeout(connectSystemUnreadSocket, 0)
  startSystemUnreadPolling()
})

onUnmounted(() => {
  window.removeEventListener('current-user-updated', handleCurrentUserUpdated)
  revokeObjectUrl(currentUserAvatarDisplay.value)
  if (systemUnreadTimer) clearInterval(systemUnreadTimer)
  disconnectSystemUnreadSocket()
})
</script>

<style>
.el-message,
.el-notification,
.el-message-box,
.el-message-box__wrapper,
.el-overlay.is-message-box {
  z-index: 2147483647 !important;
}
</style>
