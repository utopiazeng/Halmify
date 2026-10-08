<template>
  <div class="domain-page">
    <div class="domain-toolbar">
      <el-form :inline="true" class="toolbar-form">
        <el-form-item>
          <el-input-number v-model="windowDays" :min="1" :max="180" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="load">{{ t('common.action.search') }}</el-button>
          <el-button @click="reset">{{ t('common.action.reset') }}</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-table :data="rows" v-loading="loading">
      <el-table-column prop="tenant_id" :label="t('platform.tenants.tenantId')" width="120" />
      <el-table-column prop="tenant_status" :label="t('common.table.status')" width="140" />
      <el-table-column prop="plan_code" :label="t('platform.tenants.plan')" width="140" />
      <el-table-column prop="valid_to" :label="t('platform.tenants.validTo')" width="200">
        <template #default="{ row }">{{ formatDate(row.valid_to) }}</template>
      </el-table-column>
      <el-table-column prop="days_to_expire" :label="t('platform.tenants.daysToExpire')" width="140" />
      <el-table-column prop="notice_type" :label="t('platform.expiring.noticeType')" width="140" />
      <el-table-column prop="max_users" :label="t('platform.tenants.maxUsers')" width="140" />
    </el-table>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api'

const loading = ref(false)
const rows = ref([])
const windowDays = ref(30)

onMounted(load)

async function load() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/platform/tenants/expiring', { params: { window_days: windowDays.value } })
    rows.value = res?.data?.items || []
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    loading.value = false
  }
}

function reset() {
  windowDays.value = 30
  load()
}

function formatDate(value) {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleString()
}
</script>
