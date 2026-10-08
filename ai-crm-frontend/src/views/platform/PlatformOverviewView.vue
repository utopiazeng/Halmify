<template>
  <div class="domain-page">
    <div class="domain-toolbar">
      <el-form :inline="true" class="toolbar-form">
        <el-form-item :label="t('platform.overview.status')">
          <el-select v-model="filters.status" clearable style="width: 180px">
            <el-option :label="t('platform.common.all')" value="" />
            <el-option :label="t('common.status.enabled')" value="active" />
            <el-option :label="t('common.status.disabled')" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="load">{{ t('common.action.search') }}</el-button>
          <el-button @click="reset">{{ t('common.action.reset') }}</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="kpi-grid kpi-grid-compact">
      <ArtStatsCard :label="t('platform.overview.totalTenants')" :value="kpi.total_tenants" />
      <ArtStatsCard :label="t('platform.overview.activeTenants')" :value="kpi.active_tenants" />
      <ArtStatsCard :label="t('platform.overview.expiringSoon')" :value="kpi.expiring_soon" />
      <ArtStatsCard :label="t('platform.overview.avgUsage')" :value="`${kpi.avg_usage_percent}%`" />
    </div>

    <el-table :data="rows" v-loading="loading" style="margin-top: 12px">
      <el-table-column prop="tenant_id" :label="t('platform.tenants.tenantId')" width="120" />
      <el-table-column prop="status" :label="t('common.table.status')" width="120">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="plan" :label="t('platform.tenants.plan')" width="140" />
      <el-table-column prop="module_set" :label="t('platform.tenants.modules')" min-width="220">
        <template #default="{ row }">{{ (row.module_set || []).join(', ') || '-' }}</template>
      </el-table-column>
      <el-table-column prop="storage_quota" :label="t('platform.tenants.storageQuota')" width="160">
        <template #default="{ row }">{{ formatBytes(row.storage_quota) }}</template>
      </el-table-column>
      <el-table-column prop="usage_percent" :label="t('platform.tenants.usagePercent')" width="140">
        <template #default="{ row }">{{ Number(row.usage_percent || 0).toFixed(2) }}%</template>
      </el-table-column>
      <el-table-column prop="expire_at" :label="t('platform.tenants.expireAt')" width="180">
        <template #default="{ row }">{{ formatDate(row.expire_at) }}</template>
      </el-table-column>
      <el-table-column prop="admin_account_masked" :label="t('platform.tenants.adminMasked')" width="160" />
    </el-table>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import ArtStatsCard from '../../components/art/ArtStatsCard.vue'

const loading = ref(false)
const rows = ref([])
const filters = reactive({ status: '' })
const kpi = reactive({
  total_tenants: 0,
  active_tenants: 0,
  expiring_soon: 0,
  avg_usage_percent: 0
})

onMounted(load)

async function load() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/platform/tenants/summary', { params: { status: filters.status || undefined, page: 1, size: 200 } })
    rows.value = res?.data?.items || []
    summarize()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    loading.value = false
  }
}

function summarize() {
  const list = rows.value
  kpi.total_tenants = list.length
  kpi.active_tenants = list.filter((x) => x.status === 'active').length
  kpi.expiring_soon = list.filter((x) => {
    if (!x.expire_at) return false
    const ts = new Date(x.expire_at).getTime()
    if (!ts) return false
    const days = (ts - Date.now()) / 86400000
    return days >= 0 && days <= 30
  }).length
  const usage = list.map((x) => Number(x.usage_percent || 0)).filter((x) => Number.isFinite(x))
  kpi.avg_usage_percent = usage.length ? Number((usage.reduce((a, b) => a + b, 0) / usage.length).toFixed(2)) : 0
}

function reset() {
  filters.status = ''
  load()
}

function formatDate(value) {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleString()
}

function formatBytes(value) {
  const n = Number(value || 0)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let val = n
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024
    i++
  }
  return `${val.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}
</script>
