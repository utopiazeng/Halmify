<template>
  <div class="iot-firmware-page">
    <section class="iot-firmware-header">
      <div>
        <h2>{{ t('iotFirmware.title') }}</h2>
        <p>{{ t('iotFirmware.desc') }}</p>
      </div>
      <div class="iot-firmware-actions">
        <el-button @click="loadFirmwares">
          <Icon icon="ri:refresh-line" />
          <span>{{ t('iot.refresh') }}</span>
        </el-button>
        <el-button type="primary" @click="uploadVisible = true">
          <Icon icon="ri:upload-cloud-2-line" />
          <span>{{ t('iotFirmware.upload') }}</span>
        </el-button>
      </div>
    </section>

    <section class="iot-firmware-panel">
      <el-table :data="firmwares" v-loading="loading">
        <el-table-column prop="version" :label="t('iotFirmware.version')" min-width="130" />
        <el-table-column :label="t('iot.model')" min-width="150">
          <template #default="{ row }">{{ modelLabel(row.model) }}</template>
        </el-table-column>
        <el-table-column :label="t('iotFirmware.status')" width="120">
          <template #default="{ row }">
            <el-tag :type="releaseStatusTag(row.status)" effect="light">{{ t(`iotFirmware.status.${row.status || 'draft'}`) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('iotFirmware.size')" width="120">
          <template #default="{ row }">{{ formatSize(row.file_size) }}</template>
        </el-table-column>
        <el-table-column :label="t('iotFirmware.targetCount')" width="110">
          <template #default="{ row }">{{ row.target_count || 0 }}</template>
        </el-table-column>
        <el-table-column :label="t('iotFirmware.checkedCount')" width="110">
          <template #default="{ row }">{{ row.checked_count || 0 }}</template>
        </el-table-column>
        <el-table-column :label="t('iotFirmware.downloadingCount')" width="110">
          <template #default="{ row }">{{ row.downloading_count || 0 }}</template>
        </el-table-column>
        <el-table-column :label="t('iotFirmware.successCount')" width="110">
          <template #default="{ row }">{{ row.success_count || 0 }}</template>
        </el-table-column>
        <el-table-column :label="t('iotFirmware.failedCount')" width="110">
          <template #default="{ row }">{{ row.failed_count || 0 }}</template>
        </el-table-column>
        <el-table-column prop="sha256" :label="t('iotFirmware.sha256')" min-width="220" show-overflow-tooltip />
        <el-table-column :label="t('iot.actions')" width="400" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openTargets(row)">{{ t('iotFirmware.targets') }}</el-button>
            <el-button size="small" type="primary" @click="openPublish(row)">{{ t('iotFirmware.publish') }}</el-button>
            <el-button size="small" @click="revoke(row)">{{ t('iotFirmware.revoke') }}</el-button>
            <el-button size="small" type="danger" @click="deleteFirmware(row)">{{ t('iot.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-dialog v-model="uploadVisible" :title="t('iotFirmware.upload')" width="680px">
      <el-form label-position="top">
        <el-form-item :label="t('iotFirmware.packageFile')">
          <input type="file" accept=".zip,application/zip" @change="onFileChange" />
          <p class="iot-firmware-help">{{ t('iotFirmware.packageHelp') }}</p>
        </el-form-item>
        <div v-if="packagePreview.name" class="iot-firmware-preview">
          <div><span>{{ t('iotFirmware.file') }}</span><strong>{{ packagePreview.name }}</strong></div>
          <div><span>{{ t('iotFirmware.version') }}</span><strong>{{ packagePreview.version || '-' }}</strong></div>
          <div><span>{{ t('iot.model') }}</span><strong>{{ modelLabel(packagePreview.model) }}</strong></div>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="uploadVisible = false">{{ t('iot.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="uploadFirmware">{{ t('iotFirmware.upload') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="publishVisible" :title="t('iotFirmware.publish')" width="640px">
      <el-form label-position="top">
        <el-form-item :label="t('iotFirmware.target')">
          <el-radio-group v-model="publishForm.target">
            <el-radio-button label="devices">{{ t('iotFirmware.targetDevices') }}</el-radio-button>
            <el-radio-button label="tenant">{{ t('iotFirmware.targetTenant') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="publishForm.target === 'devices'" :label="t('iotFirmware.sns')">
          <el-input v-model="publishForm.snsText" type="textarea" :rows="4" :placeholder="t('iotFirmware.snsPlaceholder')" />
        </el-form-item>
        <el-form-item v-else :label="t('iotFirmware.tenantId')">
          <el-input-number v-model="publishForm.tenant_id" :min="1" />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="publishForm.force">{{ t('iotFirmware.force') }}</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="publishVisible = false">{{ t('iot.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="publishFirmware">{{ t('iotFirmware.publish') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="targetsVisible" :title="targetsTitle" width="92%" class="iot-firmware-targets-dialog">
      <div class="iot-firmware-target-toolbar">
        <el-select v-model="targetFilters.status" clearable :placeholder="t('iotFirmware.filterStatus')" @change="loadTargets(1)">
          <el-option v-for="status in targetStatusOptions" :key="status" :label="targetStatusLabel(status)" :value="status" />
        </el-select>
        <el-input v-model="targetFilters.sn" clearable :placeholder="t('iotFirmware.searchSN')" @keyup.enter="loadTargets(1)" @clear="loadTargets(1)" />
        <el-input-number v-model="targetFilters.tenant_id" :min="0" :placeholder="t('iotFirmware.filterTenant')" @change="loadTargets(1)" />
        <el-button @click="loadTargets(1)">
          <Icon icon="ri:search-line" />
          <span>{{ t('iotFirmware.search') }}</span>
        </el-button>
        <el-button @click="loadTargets(targetPagination.page)">
          <Icon icon="ri:refresh-line" />
          <span>{{ t('iot.refresh') }}</span>
        </el-button>
      </div>
      <el-table :data="targets" v-loading="targetsLoading" class="iot-firmware-target-table">
        <el-table-column prop="sn" :label="t('iot.sn')" min-width="170" show-overflow-tooltip />
        <el-table-column prop="device_name" :label="t('iot.name')" min-width="150" show-overflow-tooltip />
        <el-table-column prop="tenant_name" :label="t('iotFirmware.tenant')" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">{{ row.tenant_name || row.tenant_id || '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('iot.model')" min-width="150">
          <template #default="{ row }">{{ modelLabel(row.model) }}</template>
        </el-table-column>
        <el-table-column prop="current_version" :label="t('iotFirmware.currentVersion')" min-width="130" />
        <el-table-column prop="version" :label="t('iotFirmware.targetVersion')" min-width="130" />
        <el-table-column :label="t('iot.otaStatus')" min-width="130">
          <template #default="{ row }">
            <el-tag :type="targetStatusTag(row.status)" effect="light">{{ targetStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('iotFirmware.assignedAt')" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.assigned_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('iotFirmware.checkedAt')" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.checked_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('iotFirmware.downloadedAt')" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.downloaded_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('iotFirmware.confirmedAt')" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.confirmed_at) }}</template>
        </el-table-column>
        <el-table-column :label="t('iot.lastSeen')" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.last_device_seen_at || row.last_seen_at) }}</template>
        </el-table-column>
        <el-table-column prop="failure_reason" :label="t('iotFirmware.failureReason')" min-width="220" show-overflow-tooltip />
      </el-table>
      <div class="iot-firmware-pagination">
        <span>{{ t('common.table.total', { total: targetPagination.total }) }}</span>
        <el-pagination
          v-model:current-page="targetPagination.page"
          v-model:page-size="targetPagination.size"
          layout="prev, pager, next, sizes"
          :total="targetPagination.total"
          @current-change="handleTargetPageChange"
          @size-change="handleTargetSizeChange"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../../api'
import { t } from '../../i18n'
import { apiErrorMessage } from '../../utils/api-error'

const FIRMWARE_UPLOAD_TIMEOUT_MS = 5 * 60 * 1000
const loading = ref(false)
const saving = ref(false)
const uploadVisible = ref(false)
const publishVisible = ref(false)
const targetsVisible = ref(false)
const targetsLoading = ref(false)
const firmwares = ref([])
const targets = ref([])
const selectedFirmware = ref(null)
const selectedTargetsFirmware = ref(null)
const selectedFile = ref(null)
const packagePreview = reactive({ name: '', version: '', model: '' })
const publishForm = reactive({ target: 'devices', snsText: '', tenant_id: 1, force: false })
const targetFilters = reactive({ status: '', sn: '', tenant_id: 0 })
const targetPagination = reactive({ page: 1, size: 20, total: 0 })
const targetStatusOptions = ['assigned', 'available', 'downloading', 'success', 'failed', 'latest', 'release_missing']
const targetRefreshTimer = ref(null)
const targetsTitle = computed(() => selectedTargetsFirmware.value
  ? `${t('iotFirmware.targets')} · ${selectedTargetsFirmware.value.version || ''}`
  : t('iotFirmware.targets'))

onMounted(loadFirmwares)
onUnmounted(stopTargetRefresh)

watch(targetsVisible, (visible) => {
  if (visible) startTargetRefresh()
  else stopTargetRefresh()
})

async function loadFirmwares(options = {}) {
  if (!options.silent) loading.value = true
  try {
    const res = await api.get('/api/v1/platform/iot/firmwares')
    firmwares.value = res?.data?.items || []
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || t('iotFirmware.loadFailed'))
  } finally {
    if (!options.silent) loading.value = false
  }
}

function onFileChange(event) {
  selectedFile.value = event?.target?.files?.[0] || null
  Object.assign(packagePreview, previewFirmwarePackage(selectedFile.value?.name || ''))
}

async function uploadFirmware() {
  if (!selectedFile.value || !packagePreview.version || !packagePreview.model) {
    ElMessage.warning(t('iotFirmware.required'))
    return
  }
  saving.value = true
  try {
    const form = new FormData()
    form.append('file', selectedFile.value)
    await api.post('/api/v1/platform/iot/firmwares', form, { timeout: FIRMWARE_UPLOAD_TIMEOUT_MS })
    uploadVisible.value = false
    Object.assign(packagePreview, { name: '', version: '', model: '' })
    selectedFile.value = null
    ElMessage.success(t('iotFirmware.uploaded'))
    await loadFirmwares()
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('iotFirmware.uploadFailed')))
  } finally {
    saving.value = false
  }
}

function previewFirmwarePackage(name) {
  const clean = String(name || '').trim()
  const base = clean.replace(/\.[^.]+$/, '')
  const parts = base.split(/[-_ ]+/).filter(Boolean)
  let version = ''
  let versionIndex = -1
  for (let i = parts.length - 1; i >= 0; i -= 1) {
    if (/^v?\d+(?:\.\d+){1,3}$/i.test(parts[i])) {
      version = parts[i]
      versionIndex = i
      break
    }
  }
  const model = versionIndex > 0
    ? parts
      .slice(0, versionIndex)
      .filter((part, index) => !(index === 0 && part.toLowerCase() === 'halmify'))
      .join('-')
      .toLowerCase()
    : ''
  return { name: clean, version, model }
}

function openPublish(row) {
  selectedFirmware.value = row
  Object.assign(publishForm, { target: 'devices', snsText: '', tenant_id: 1, force: Boolean(row.force) })
  publishVisible.value = true
}

async function publishFirmware() {
  if (!selectedFirmware.value) return
  const payload = {
    target: publishForm.target,
    force: publishForm.force,
    tenant_id: publishForm.target === 'tenant' ? Number(publishForm.tenant_id || 0) : 0,
    sns: publishForm.target === 'devices' ? publishForm.snsText.split(/[\s,，]+/).filter(Boolean) : []
  }
  if (payload.target === 'devices' && !payload.sns.length) {
    ElMessage.warning(t('iotFirmware.snsRequired'))
    return
  }
  saving.value = true
  try {
    const res = await api.post(`/api/v1/platform/iot/firmwares/${selectedFirmware.value.id}/publish`, payload, {
      headers: { 'Content-Type': 'application/json' }
    })
    publishVisible.value = false
    ElMessage.success(t('iotFirmware.published', { count: res?.data?.updated_devices || 0 }))
    await loadFirmwares()
    if (targetsVisible.value) await loadTargets(targetPagination.page)
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || t('iotFirmware.publishFailed'))
  } finally {
    saving.value = false
  }
}

async function openTargets(row) {
  selectedTargetsFirmware.value = row
  Object.assign(targetFilters, { status: '', sn: '', tenant_id: 0 })
  Object.assign(targetPagination, { page: 1, size: 20, total: 0 })
  targets.value = []
  targetsVisible.value = true
  await loadTargets(1)
}

function startTargetRefresh() {
  stopTargetRefresh()
  targetRefreshTimer.value = window.setInterval(async () => {
    if (!targetsVisible.value || targetsLoading.value || !selectedTargetsFirmware.value) return
    await loadTargets(targetPagination.page, { silent: true })
    await loadFirmwares({ silent: true })
  }, 5000)
}

function stopTargetRefresh() {
  if (!targetRefreshTimer.value) return
  window.clearInterval(targetRefreshTimer.value)
  targetRefreshTimer.value = null
}

async function loadTargets(page = targetPagination.page, options = {}) {
  if (!selectedTargetsFirmware.value) return
  targetPagination.page = Number(page || 1)
  if (!options.silent) targetsLoading.value = true
  try {
    const params = {
      page: targetPagination.page,
      size: targetPagination.size,
      status: targetFilters.status || undefined,
      sn: targetFilters.sn || undefined,
      tenant_id: Number(targetFilters.tenant_id || 0) || undefined
    }
    const res = await api.get(`/api/v1/platform/iot/firmwares/${selectedTargetsFirmware.value.id}/targets`, { params })
    targets.value = res?.data?.items || []
    targetPagination.total = Number(res?.data?.total || 0)
    targetPagination.page = Number(res?.data?.page || targetPagination.page)
    targetPagination.size = Number(res?.data?.size || targetPagination.size)
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || t('iotFirmware.targetsLoadFailed'))
  } finally {
    if (!options.silent) targetsLoading.value = false
  }
}

function handleTargetPageChange(page) {
  loadTargets(page)
}

function handleTargetSizeChange(size) {
  targetPagination.size = Number(size || 20)
  loadTargets(1)
}

async function revoke(row) {
  await api.post(`/api/v1/platform/iot/firmwares/${row.id}/revoke`)
  ElMessage.success(t('iotFirmware.revoked'))
  await loadFirmwares()
}

async function deleteFirmware(row) {
  try {
    await ElMessageBox.confirm(t('iotFirmware.deleteConfirm'), t('iot.deleteTitle'), {
      type: 'warning',
      confirmButtonText: t('iot.delete'),
      cancelButtonText: t('iot.cancel')
    })
    await api.delete(`/api/v1/platform/iot/firmwares/${row.id}`)
    ElMessage.success(t('iot.deleted'))
    await loadFirmwares()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error?.response?.data?.message || error?.message || t('iot.deleteFailed'))
  }
}

function modelLabel(model) {
  if (model === 'esp32-p4-epaper') return t('iot.model.p4')
  if (model === 'esp32-s3-epaper') return t('iot.model.s3')
  return model || t('iot.modelUnknown')
}

function releaseStatusTag(status) {
  if (status === 'published') return 'success'
  if (status === 'revoked') return 'info'
  return 'warning'
}

function targetStatusLabel(status) {
  return t(`iot.ota.status.${status || 'assigned'}`)
}

function targetStatusTag(status) {
  if (status === 'success') return 'success'
  if (status === 'failed' || status === 'release_missing') return 'danger'
  if (status === 'downloading') return 'warning'
  if (status === 'available') return 'primary'
  if (status === 'latest') return 'info'
  return 'info'
}

function formatSize(value) {
  const size = Number(value || 0)
  if (size >= 1024 * 1024) return `${(size / 1024 / 1024).toFixed(2)} MB`
  if (size >= 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${size} B`
}

function formatDateTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}
</script>

<style scoped>
.iot-firmware-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.iot-firmware-header,
.iot-firmware-panel {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.iot-firmware-header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 20px;
}

.iot-firmware-header h2 {
  margin: 0;
  font-size: 22px;
  color: #111827;
}

.iot-firmware-header p {
  margin: 8px 0 0;
  color: #6b7280;
}

.iot-firmware-actions {
  display: flex;
  gap: 8px;
  align-items: flex-start;
}

.iot-firmware-actions .el-button {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.iot-firmware-panel {
  padding: 12px;
}

.iot-firmware-target-toolbar {
  display: grid;
  grid-template-columns: minmax(140px, 180px) minmax(180px, 1fr) minmax(140px, 180px) auto auto;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
}

.iot-firmware-target-toolbar .el-button {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.iot-firmware-target-table {
  width: 100%;
}

.iot-firmware-pagination {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  margin-top: 12px;
  color: #6b7280;
}

.iot-firmware-full {
  width: 100%;
}

.iot-firmware-help {
  margin: 8px 0 0;
  color: #6b7280;
  font-size: 13px;
  line-height: 1.5;
}

.iot-firmware-preview {
  display: grid;
  gap: 8px;
  padding: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #f9fafb;
}

.iot-firmware-preview div {
  display: grid;
  grid-template-columns: 120px minmax(0, 1fr);
  gap: 10px;
}

.iot-firmware-preview span {
  color: #6b7280;
}

.iot-firmware-preview strong {
  min-width: 0;
  color: #111827;
  overflow-wrap: anywhere;
}

@media (max-width: 760px) {
  .iot-firmware-header {
    flex-direction: column;
  }

  .iot-firmware-target-toolbar {
    grid-template-columns: 1fr;
  }

  .iot-firmware-pagination {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
