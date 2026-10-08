<template>
  <div class="domain-page">
    <div class="domain-toolbar">
      <el-form :inline="true" class="toolbar-form">
        <el-form-item>
          <el-button @click="load">{{ t('common.action.refresh') }}</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-table :data="rows" v-loading="loading">
      <el-table-column prop="plan_code" :label="t('platform.plans.code')" width="120" />
      <el-table-column prop="name" :label="t('platform.plans.name')" width="160" />
      <el-table-column prop="max_users_default" :label="t('platform.tenants.maxUsers')" width="140" />
      <el-table-column prop="storage_default" :label="t('platform.tenants.storageQuota')" width="160">
        <template #default="{ row }">{{ formatBytes(row.storage_default) }}</template>
      </el-table-column>
      <el-table-column prop="modules" :label="t('platform.tenants.modulesPreview')" min-width="300">
        <template #default="{ row }">{{ (row.modules || []).join(', ') }}</template>
      </el-table-column>
      <el-table-column :label="t('common.table.operation')" width="120">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="openEdit(row)">{{ t('common.action.edit') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="editVisible" :title="t('platform.plans.editTitle')" width="720px">
      <el-form :model="form" label-width="160px">
        <el-form-item :label="t('platform.plans.code')"><el-input v-model="form.plan_code" disabled /></el-form-item>
        <el-form-item :label="t('platform.plans.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('platform.tenants.maxUsers')"><el-input-number v-model="form.max_users_default" :min="1" /></el-form-item>
        <el-form-item :label="t('platform.tenants.storageQuota')"><el-input :model-value="t('storage.text.baseIncluded')" disabled /></el-form-item>
        <el-form-item :label="t('platform.tenants.modules')">
          <el-select v-model="form.modules" multiple style="width: 100%">
            <el-option v-for="item in moduleOptions" :key="item" :label="moduleLabel(item)" :value="item" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">{{ t('common.action.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submit">{{ t('common.action.submit') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import { apiErrorMessage } from '../../utils/api-error'

const loading = ref(false)
const saving = ref(false)
const rows = ref([])
const editVisible = ref(false)
const form = reactive(defaultForm())
const moduleOptions = ['workspace', 'projects', 'products', 'campaigns', 'contacts', 'leads', 'opportunities', 'deals', 'orders', 'tickets', 'dashboards', 'teams', 'insights', 'accounts', 'im', 'audit_log']

onMounted(load)

function defaultForm() {
  return {
    plan_code: 'basic',
    name: '',
    max_users_default: 20,
    storage_default: 1024 * 1024 * 1024,
    modules: []
  }
}

async function load() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/platform/plans')
    rows.value = (res?.data?.items || []).filter((item) => item.plan_code === 'basic' || item.plan_code === 'plus')
  } catch (e) {
    ElMessage.error(apiErrorMessage(e, t('common.message.loadFailed')))
  } finally {
    loading.value = false
  }
}

function openEdit(row) {
  Object.assign(form, defaultForm(), {
    plan_code: row.plan_code,
    name: row.name,
    max_users_default: row.max_users_default,
    storage_default: row.storage_default,
    modules: Array.isArray(row.modules) ? [...row.modules] : []
  })
  editVisible.value = true
}

async function submit() {
  saving.value = true
  try {
    await api.put(`/api/v1/platform/plans/${form.plan_code}`, {
      name: form.name,
      max_users_default: form.max_users_default,
      storage_default: 1024 * 1024 * 1024,
      modules: form.modules,
      status: 'active'
    })
    ElMessage.success(t('common.message.opSuccess'))
    editVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(apiErrorMessage(e, t('common.message.loadFailed')))
  } finally {
    saving.value = false
  }
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
function moduleLabel(code) {
  return ({
    workspace: t('domain.auto.dd3edc066c'),
    projects: t('domain.auto.6dd16d11ee'),
    products: t('domain.auto.76363b4539'),
    campaigns: t('domain.auto.5fa95b1b73'),
    contacts: t('domain.auto.b274256b6e'),
    leads: t('domain.auto.8de0b8fb1d'),
    opportunities: t('domain.auto.2559c4e3c0'),
    deals: t('domain.auto.6a30a471cf'),
    orders: t('domain.auto.81b3798beb'),
    tickets: t('domain.auto.17e2f8d878'),
    dashboards: t('domain.auto.e164aa4223'),
    teams: t('domain.auto.b060bbbc90'),
    insights: t('domain.auto.47b27c79bd'),
    accounts: t('domain.auto.11b83f7d0f'),
    im: t('domain.auto.925f26b9e1'),
    audit_log: t('domain.auto.1db2e8417b')
  }[code] || code)
}

</script>
