<template>
  <div class="system-page accounts-page">
    <section class="crm-help-strip">
      <span>{{ t('domain.auto.29873dae44') }}</span>
    </section>

    <section class="account-kpi-grid">
      <MetricCard v-for="item in kpiCards" :key="item.key" v-bind="item" />
    </section>

    <section class="account-main-grid">
      <div class="system-card">
        <div class="system-toolbar">
          <div class="toolbar-left">
            <el-button type="primary" plain @click="goSystem('user')">{{ t('domain.auto.baf84751a2') }}</el-button>
            <el-button plain @click="goSystem('role')">{{ t('domain.auto.3d0d88d5d4') }}</el-button>
          </div>
          <div class="toolbar-icons">
            <el-tooltip :content="t('domain.auto.38108eaa1d')">
              <button class="system-icon-btn" :aria-label="t('domain.auto.38108eaa1d')" @click="loadPage"><Icon icon="ri:refresh-line" /></button>
            </el-tooltip>
          </div>
        </div>

        <el-table :data="users" class="system-table" row-key="id" v-loading="loading">
          <el-table-column prop="username" :label="t('domain.auto.9013849179')" min-width="140">
            <template #default="{ row }">
              <div class="record-title">
                <strong>{{ row.username || row.email || `${t('domain.auto.9ba763ea34')}${row.id}` }}</strong>
                <span>{{ row.name || row.nickname || '-' }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('domain.auto.6b26695e4d')" min-width="180">
            <template #default="{ row }">{{ roleText(row) }}</template>
          </el-table-column>
          <el-table-column :label="t('domain.auto.f63da5b695')" width="130">
            <template #default="{ row }">
              <el-tag :type="isTenantAdmin(row) ? 'danger' : 'info'">{{ accountTypeLabel(row.account_type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('domain.auto.62e951a692')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'disabled' ? 'info' : 'success'">{{ row.status === 'disabled' ? t('domain.auto.d989e55188') : t('domain.auto.d4e9ca3dd4') }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('domain.auto.2f4eba2ee0')" min-width="170">
            <template #default="{ row }">{{ row.department_name || '-' }} / {{ row.post_name || '-' }}</template>
          </el-table-column>
        </el-table>
      </div>

      <aside class="system-card">
        <div class="chart-panel-head">
          <strong>{{ t('domain.auto.59e6fa2a61') }}</strong>
          <el-tag size="small">{{ t('domain.auto.53a989d8cc') }}</el-tag>
        </div>
        <div class="team-note-list">
          <div v-for="item in roleNotes" :key="item.title" class="team-note-row">
            <strong>{{ item.title }}</strong>
            <span>{{ item.text }}</span>
          </div>
        </div>

        <el-divider>{{ t('domain.auto.9723b3828b') }}</el-divider>
        <div class="team-note-list">
          <div v-for="item in roles.slice(0, 8)" :key="item.id" class="team-note-row">
            <strong>{{ item.name || item.code || `${t('domain.auto.6b26695e4d')}${item.id}` }}</strong>
            <span>{{ item.code || t('domain.auto.6e28ab92a5') }} · {{ rolePermissionText(item) }}</span>
          </div>
        </div>
        <el-empty v-if="!roles.length" :description="t('domain.auto.b128e1525b')" />
      </aside>
    </section>

    <section class="system-card">
      <div class="chart-panel-head">
        <strong>{{ t('domain.auto.342e0fd958') }}</strong>
        <el-tag size="small">view / manage</el-tag>
      </div>
      <p class="crm-inline-help">{{ t('domain.auto.11ff5608f9') }}</p>
      <el-table :data="moduleRows" class="system-table" row-key="resource">
        <el-table-column prop="label" :label="t('domain.auto.c1bebd4ab3')" min-width="140" />
        <el-table-column :label="t('domain.auto.8761bd187c')" min-width="180"><template #default="{ row }"><code>{{ row.resource }}:view</code></template></el-table-column>
        <el-table-column :label="t('domain.auto.d6f01f855a')" min-width="180"><template #default="{ row }"><code>{{ row.resource }}:manage</code></template></el-table-column>
        <el-table-column prop="desc" :label="t('domain.auto.26670dda42')" min-width="260" />
      </el-table>
    </section>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import MetricCard from '../../components/MetricCard.vue'

const router = useRouter()
const users = ref([])
const roles = ref([])
const loading = ref(false)

const moduleRows = [
  { resource: 'leads', label: t('domain.auto.8de0b8fb1d'), desc: t('domain.auto.0767801d88') },
  { resource: 'contacts', label: t('domain.auto.b274256b6e'), desc: t('domain.auto.212be877da') },
  { resource: 'opportunities', label: t('domain.auto.c4beee8569'), desc: t('domain.auto.33d89cd421') },
  { resource: 'orders', label: t('domain.auto.81b3798beb'), desc: t('domain.auto.da01056810') },
  { resource: 'tickets', label: t('domain.auto.85c230ed15'), desc: t('domain.auto.b668cc4873') },
  { resource: 'campaigns', label: t('domain.auto.5fa95b1b73'), desc: t('domain.auto.8639995ea2') },
  { resource: 'products', label: t('domain.auto.76363b4539'), desc: t('domain.auto.8dfd9799bf') },
  { resource: 'dashboards', label: t('domain.auto.e164aa4223'), desc: t('domain.auto.33c55b8f9e') }
]

const roleNotes = [
  { title: t('domain.auto.82f0ed0c7d'), text: t('domain.auto.b2f5d0b7fa') },
  { title: t('domain.auto.f7751161e4'), text: t('domain.auto.cdf21ae90f') },
  { title: t('domain.auto.396d9b8c30'), text: t('domain.auto.238d94c5cd') },
  { title: t('domain.auto.e357fc952d'), text: t('domain.auto.099489fe5d') }
]

const kpiCards = computed(() => [
  { key: 'users', title: t('domain.auto.ef68fea4ff'), value: users.value.length, icon: 'ri:user-settings-line', tone: 'blue', help: t('domain.auto.a09d6de21b') },
  { key: 'active', title: t('domain.auto.df67335bf9'), value: users.value.filter((item) => item.status !== 'disabled').length, icon: 'ri:checkbox-circle-line', tone: 'green', help: t('domain.auto.00a871ede8') },
  { key: 'roles', title: t('domain.auto.4f65a1b654'), value: roles.value.length, icon: 'ri:shield-user-line', tone: 'purple', help: t('domain.auto.e9663b948c') },
  { key: 'admins', title: t('domain.auto.284e268846'), value: users.value.filter((item) => isTenantAdmin(item)).length, icon: 'ri:admin-line', tone: 'orange', help: t('domain.auto.59cd23fbf4') }
])

onMounted(loadPage)

async function loadPage() {
  loading.value = true
  try {
    const [userRes, roleRes] = await Promise.all([
      api.get('/api/v1/accounts/users', { params: { page: 1, size: 200 } }),
      api.get('/api/v1/rbac/roles')
    ])
    users.value = userRes.data?.items || []
    roles.value = roleRes.data?.items || []
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error.message || t('domain.auto.797d24d80c'))
  } finally {
    loading.value = false
  }
}

function goSystem(section) {
  router.push(`/system/${section}`)
}


function isTenantAdmin(user) {
  return ['super_admin', 'tenant_admin'].includes(user.account_type) || ['super_admin', 'tenant_admin'].includes(user.role)
}

function accountTypeLabel(value) {
  return ({ super_admin: t('domain.auto.6f22a707cd'), tenant_admin: t('domain.auto.82f0ed0c7d'), user: t('domain.auto.bd8d896aad') }[value] || value || t('domain.auto.bd8d896aad'))
}

function roleText(user) {
  const names = user.role_names || user.roles || user.role_name || user.role
  if (Array.isArray(names)) return names.join('、') || '-'
  return names || '-'
}

function rolePermissionText(role) {
  const permissions = splitPermissions(role.permissions_json || role.permissions)
  const manageCount = permissions.filter((item) => item.endsWith(':manage')).length
  if (manageCount) return `${manageCount}${t('domain.auto.be7adfbd5b')}`
  if (permissions.length) return `${permissions.length}${t('domain.auto.e55a19bb1e')}`
  return t('domain.auto.47968d97a6')
}

function splitPermissions(value = '') {
  if (Array.isArray(value)) return value
  return String(value).split(',').map((item) => item.trim()).filter(Boolean)
}
</script>
