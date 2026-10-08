<template>
  <div class="system-page teams-page">
    <section class="crm-help-strip"><span>{{ t('domain.auto.31ca60e04d') }}</span></section>

    <section class="team-kpi-grid">
      <MetricCard v-for="item in kpiCards" :key="item.key" v-bind="item" />
    </section>

    <section class="team-main-grid">
      <div class="system-card">
        <div class="chart-panel-head"><strong>{{ t('domain.auto.c8b6f50b8b') }}</strong><el-button size="small" @click="refreshAll">{{ t('domain.auto.38108eaa1d') }}</el-button></div>
        <el-table :data="users" class="system-table" row-key="id" v-loading="loading">
          <el-table-column prop="name" :label="t('domain.auto.c1ee9f0190')" min-width="170"><template #default="{ row }"><div class="record-title"><strong>{{ row.name || row.username }}</strong><span>{{ row.email || row.phone || '-' }}</span></div></template></el-table-column>
          <el-table-column :label="t('domain.auto.6b26695e4d')" min-width="160"><template #default="{ row }">{{ roleNames(row.role_ids || [row.role_id]) || '-' }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.91061a56c0')" width="150"><template #default="{ row }">{{ row.department_name || departmentName(row.department_id) || '-' }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.62e951a692')" width="100"><template #default="{ row }"><el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ row.status === 'active' ? t('domain.auto.d4e9ca3dd4') : row.status }}</el-tag></template></el-table-column>
        </el-table>
      </div>

      <aside class="system-card">
        <div class="chart-panel-head"><strong>{{ t('domain.auto.dea0f3a00d') }}</strong><el-tag size="small">{{ t('domain.auto.3dfb868491') }}</el-tag></div>
        <div class="dashboard-leaderboard team-leaderboard">
          <div v-for="item in cockpit.charts.leaderboard" :key="item.owner_id" class="dashboard-leader-row light">
            <span>{{ item.name }}</span>
            <div><i :style="{ width: chartWidth(item.amount, maxLeaderAmount) }"></i></div>
            <strong>{{ formatAmount(item.amount) }}</strong>
          </div>
        </div>
        <el-empty v-if="!cockpit.charts.leaderboard.length" :description="t('domain.auto.fee046faec')" />
      </aside>
    </section>

    <section class="team-main-grid">
      <div class="system-card">
        <div class="chart-panel-head"><strong>{{ t('domain.auto.4c9cc379da') }}</strong><el-tag size="small">{{ t('domain.auto.089dbc6bed') }}</el-tag></div>
        <div class="team-note-list">
          <div v-for="item in roleNotes" :key="item.title" class="team-note-row"><strong>{{ item.title }}</strong><span>{{ item.text }}</span></div>
        </div>
      </div>
      <div class="system-card">
        <div class="chart-panel-head"><strong>{{ t('domain.auto.91061a56c0') }}</strong><el-tag>{{ flatDepartments.length }}</el-tag></div>
        <div class="team-note-list"><div v-for="item in flatDepartments" :key="item.id" class="team-note-row"><strong>{{ item.name }}</strong><span>{{ item.leader || t('domain.auto.d7762033ab') }} · {{ item.status || 'active' }}</span></div></div>
      </div>
    </section>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, onMounted, reactive, ref } from 'vue'
import api from '../../api'
import MetricCard from '../../components/MetricCard.vue'
import { formatCurrencyAmount, normalizeCurrencyCode } from '../../utils/currency'
import { defaultBusinessCurrency } from '../../utils/company-format'
import { hasPermission } from '../../permissions'

const users = ref([])
const roles = ref([])
const departments = ref([])
const loading = ref(false)
const cockpit = reactive({ charts: { leaderboard: [] } })
const roleNotes = [
  { title: t('domain.auto.82f0ed0c7d'), text: t('domain.auto.566244feb5') },
  { title: t('domain.auto.2f5185a1ca'), text: t('domain.auto.93185e4eee') },
  { title: t('domain.auto.dc4e1e6b18'), text: t('domain.auto.3c1f035176') }
]
const flatDepartments = computed(() => flattenTree(departments.value))
const kpiCards = computed(() => [
  { key: 'users', title: t('domain.auto.06d22c61cc'), value: users.value.length, icon: 'ri:team-line', tone: 'blue', help: t('domain.auto.7de0251fdd') },
  { key: 'roles', title: t('domain.auto.4f65a1b654'), value: roles.value.length, icon: 'ri:shield-user-line', tone: 'purple', help: t('domain.auto.b220b63f95') },
  { key: 'departments', title: t('domain.auto.6b261d01c0'), value: flatDepartments.value.length, icon: 'ri:organization-chart', tone: 'orange', help: t('domain.auto.11bdd0b9a7') },
  { key: 'active', title: t('domain.auto.af398b27e0'), value: users.value.filter((item) => item.status === 'active').length, icon: 'ri:user-follow-line', tone: 'green', help: t('domain.auto.af398b27e0') }
])
const maxLeaderAmount = computed(() => Math.max(1, ...((cockpit.charts.leaderboard || []).map((item) => Number(item.amount || 0)))))

onMounted(refreshAll)
async function refreshAll() {
  loading.value = true
  try {
    const [userRes, roleRes, deptRes, cockpitRes] = await Promise.all([
      api.get('/api/v1/accounts/users', { params: { page: 1, size: 200 } }),
      hasPermission('accounts:manage') ? api.get('/api/v1/rbac/roles') : Promise.resolve({ data: { items: [] } }),
      hasPermission('system:department:view') ? api.get('/api/v1/system/departments') : Promise.resolve({ data: { items: [] } }),
      api.get('/api/v1/prd/cockpit')
    ])
    users.value = userRes.data?.items || []
    roles.value = roleRes.data?.items || []
    departments.value = deptRes.data?.items || []
    cockpit.charts = cockpitRes.data?.charts || { leaderboard: [] }
  } finally { loading.value = false }
}
function roleNames(ids) { return (ids || []).map((id) => roles.value.find((role) => Number(role.id) === Number(id))?.name).filter(Boolean).join('、') }
function departmentName(id) { return flatDepartments.value.find((item) => Number(item.id) === Number(id))?.name }
function flattenTree(items) { return (items || []).flatMap((item) => [item, ...flattenTree(item.children || [])]) }
function chartWidth(value, max) { return `${Math.max(8, Math.round((Number(value || 0) / Number(max || 1)) * 100))}%` }
function formatAmount(amount, currency = defaultBusinessCurrency()) { return formatCurrencyAmount(amount, normalizeCurrencyCode(currency, defaultBusinessCurrency())) }
</script>
