<template>
  <div class="domain-page">
    <div class="domain-toolbar">
      <el-form :inline="true" class="toolbar-form">
        <el-form-item><el-button type="primary" @click="openDraft">{{ t('platform.settings.newDraft') }}</el-button></el-form-item>
        <el-form-item><el-button @click="load">{{ t('common.action.refresh') }}</el-button></el-form-item>
      </el-form>
    </div>

    <el-table :data="rows" v-loading="loading">
      <el-table-column prop="id" label="ID" width="100" />
      <el-table-column prop="group" :label="t('platform.settings.group')" width="160" />
      <el-table-column prop="key" :label="t('platform.settings.key')" width="220" />
      <el-table-column prop="version" :label="t('platform.settings.version')" width="100" />
      <el-table-column prop="status" :label="t('common.table.status')" width="120" />
      <el-table-column prop="effective_at" :label="t('platform.settings.effectiveAt')" width="180" />
      <el-table-column :label="t('common.table.operation')" min-width="240">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="publish(row)">{{ t('platform.settings.publish') }}</el-button>
          <el-button size="small" @click="rollback(row)">{{ t('platform.settings.rollback') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="draftVisible" :title="t('platform.settings.newDraft')" width="700px">
      <el-form :model="draft" label-width="120px">
        <el-form-item :label="t('platform.settings.group')"><el-input v-model="draft.group" /></el-form-item>
        <el-form-item :label="t('platform.settings.key')"><el-input v-model="draft.key" /></el-form-item>
        <el-form-item :label="t('platform.settings.valueJson')"><el-input v-model="draft.value_json" type="textarea" :rows="8" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="draftVisible = false">{{ t('common.action.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="saveDraft">{{ t('common.action.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api'

const loading = ref(false)
const saving = ref(false)
const draftVisible = ref(false)
const rows = ref([])
const draft = reactive({ group: '', key: '', value_json: '{}' })

onMounted(load)

async function load() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/platform/settings', { params: { status: 'all' } })
    rows.value = res?.data?.items || []
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openDraft() {
  draft.group = ''
  draft.key = ''
  draft.value_json = '{}'
  draftVisible.value = true
}

async function saveDraft() {
  let value = {}
  try {
    value = JSON.parse(draft.value_json || '{}')
  } catch {
    ElMessage.warning(t('platform.settings.invalidJson'))
    return
  }
  saving.value = true
  try {
    await api.post('/api/v1/platform/settings/draft', {
      group: draft.group,
      key: draft.key,
      value
    })
    ElMessage.success(t('common.message.opSuccess'))
    draftVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    saving.value = false
  }
}

async function publish(row) {
  try {
    await api.post('/api/v1/platform/settings/publish', { id: row.id })
    ElMessage.success(t('common.message.opSuccess'))
    await load()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  }
}

async function rollback(row) {
  try {
    await api.post('/api/v1/platform/settings/rollback', { group: row.group, key: row.key })
    ElMessage.success(t('common.message.opSuccess'))
    await load()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  }
}
</script>
