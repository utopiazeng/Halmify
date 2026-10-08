<template>
  <div class="iot-page">
    <section class="iot-hero">
      <div>
        <h2>{{ t('iot.title') }}</h2>
        <p>{{ t('iot.desc') }}</p>
      </div>
      <div class="iot-hero-actions">
        <el-button @click="helpVisible = true">
          <Icon icon="ri:book-open-line" />
          <span>{{ t('iot.guide.open') }}</span>
        </el-button>
        <el-button @click="loadDevices">
          <Icon icon="ri:refresh-line" />
          <span>{{ t('iot.refresh') }}</span>
        </el-button>
        <el-button type="primary" @click="openCreate">
          <Icon icon="ri:add-line" />
          <span>{{ t('iot.add') }}</span>
        </el-button>
      </div>
    </section>

    <section class="iot-panel">
      <el-table :data="devices" v-loading="loading" class="iot-table">
        <el-table-column :label="t('iot.name')" min-width="150">
          <template #default="{ row }">
            <div class="iot-device-name">
              <strong>{{ row.name || '-' }}</strong>
              <span>{{ row.remark || '-' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="sn" :label="t('iot.sn')" min-width="170" show-overflow-tooltip />
        <el-table-column :label="t('iot.model')" min-width="150">
          <template #default="{ row }">{{ modelLabel(row.model) }}</template>
        </el-table-column>
        <el-table-column :label="t('iot.firmware')" min-width="170">
          <template #default="{ row }">
            <div class="iot-device-name">
              <strong>{{ row.firmware_version || '-' }}</strong>
              <span>{{ row.target_firmware_version ? `${t('iot.otaTarget')}: ${row.target_firmware_version}` : t('iot.otaNoTarget') }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('iot.otaStatus')" min-width="140">
          <template #default="{ row }">
            <el-tag :type="otaTagType(row.ota_status)" effect="light">{{ otaStatusLabel(row.ota_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('iot.resolution')" width="120">
          <template #default="{ row }">{{ resolutionText(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('iot.status')" width="110">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" effect="light">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('iot.onlineState')" width="110">
          <template #default="{ row }">
            <el-tag :type="row.online ? 'success' : 'info'" effect="light">
              {{ row.online ? t('iot.online') : t('iot.offline') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('iot.lastSync')" min-width="170">
          <template #default="{ row }">
            <div class="iot-device-name">
              <strong>{{ formatTime(row.last_sync_at) }}</strong>
              <span>{{ t('iot.lastSeen') }}: {{ formatTime(row.last_seen_at) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="last_ip" :label="t('iot.lastIp')" min-width="140">
          <template #default="{ row }">{{ row.last_ip || '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('iot.actions')" width="340" fixed="right">
          <template #default="{ row }">
            <div class="iot-actions">
              <el-button size="small" @click="openEdit(row)">{{ t('iot.edit') }}</el-button>
              <el-button size="small" @click="openPreview(row)">{{ t('iot.preview') }}</el-button>
              <el-dropdown trigger="click">
                <el-button size="small">
                  <Icon icon="ri:more-2-line" />
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="rotateSecret(row)">{{ t('iot.rotate') }}</el-dropdown-item>
                    <el-dropdown-item divided @click="deleteDevice(row)">{{ t('iot.delete') }}</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-dialog v-model="formVisible" :title="formMode === 'create' ? t('iot.createTitle') : t('iot.editTitle')" width="620px">
      <el-form :model="form" label-position="top" class="iot-form">
        <el-form-item :label="t('iot.name')">
          <el-input v-model="form.name" :placeholder="t('iot.namePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('iot.sn')">
          <el-input v-model="form.sn" :disabled="formMode !== 'create'" :placeholder="t('iot.snPlaceholder')" />
        </el-form-item>
        <div class="iot-form-grid">
          <el-form-item :label="t('iot.width')">
            <el-input-number v-model="form.resolution_width" :min="80" :max="4096" />
          </el-form-item>
          <el-form-item :label="t('iot.height')">
            <el-input-number v-model="form.resolution_height" :min="80" :max="4096" />
          </el-form-item>
        </div>
        <el-form-item :label="t('iot.status')">
          <el-select v-model="form.status" class="iot-full">
            <el-option value="active" :label="t('iot.status.active')" />
            <el-option value="disabled" :label="t('iot.status.disabled')" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('iot.remark')">
          <el-input v-model="form.remark" type="textarea" :rows="3" :placeholder="t('iot.remarkPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">{{ t('iot.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">{{ t('iot.save') }}</el-button>
      </template>
    </el-dialog>


    <el-dialog v-model="previewVisible" :title="t('iot.previewTitle')" width="680px">
      <div v-loading="previewLoading" class="iot-preview">
        <div class="iot-preview-meta">
          <span>{{ t('iot.year') }}: {{ previewData.year || currentYear }}</span>
          <span>{{ t('iot.currency') }}: {{ previewData.currency || '-' }}</span>
          <span>{{ t('iot.refreshInterval') }}: {{ previewData.refresh_interval_sec || 600 }}s</span>
        </div>
        <div class="iot-metric-grid">
          <div v-for="metric in previewMetrics" :key="metric.key" class="iot-metric-card">
            <span>{{ metricLabel(metric) }}</span>
            <strong>{{ metric.value || '-' }}</strong>
          </div>
        </div>
      </div>
    </el-dialog>

    <el-drawer v-model="helpVisible" :title="t('iot.guide.title')" size="min(720px, 92vw)" destroy-on-close>
      <div class="iot-guide">
        <section class="iot-guide-intro">
          <div class="iot-guide-intro-icon">
            <Icon icon="ri:cpu-line" />
          </div>
          <div>
            <p class="iot-guide-kicker">{{ t('iot.guide.kicker') }}</p>
            <h3>{{ t('iot.guide.heading') }}</h3>
            <p>{{ t('iot.guide.summary') }}</p>
          </div>
        </section>

        <section class="iot-guide-stat-grid" :aria-label="t('iot.guide.quickFacts')">
          <div v-for="item in guideStats" :key="item.key" class="iot-guide-stat">
            <Icon :icon="item.icon" />
            <span>{{ t(item.label) }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </section>

        <section class="iot-guide-timeout">
          <Icon icon="ri:alarm-warning-line" />
          <div>
            <h4>{{ t('iot.guide.timeout.title') }}</h4>
            <p>{{ t('iot.guide.timeout.desc') }}</p>
            <code>{{ t('iot.guide.timeout.notice') }}</code>
          </div>
        </section>

        <section class="iot-guide-section">
          <div class="iot-guide-section-title">
            <Icon icon="ri:route-line" />
            <h4>{{ t('iot.guide.stepsTitle') }}</h4>
          </div>
          <div class="iot-guide-steps">
            <article v-for="(step, index) in guideSteps" :key="step.key" class="iot-guide-step">
              <div class="iot-guide-step-index">{{ index + 1 }}</div>
              <div>
                <h5>{{ t(step.title) }}</h5>
                <p>{{ t(step.desc) }}</p>
                <code v-if="step.code">{{ step.code }}</code>
              </div>
            </article>
          </div>
        </section>

        <section class="iot-guide-section">
          <div class="iot-guide-section-title">
            <Icon icon="ri:settings-3-line" />
            <h4>{{ t('iot.guide.configTitle') }}</h4>
          </div>
          <div class="iot-guide-config">
            <div v-for="row in guideConfigRows" :key="row.key" class="iot-guide-config-row">
              <span>{{ t(row.label) }}</span>
              <p>{{ t(row.desc) }}</p>
            </div>
          </div>
        </section>

        <section class="iot-guide-section">
          <div class="iot-guide-section-title">
            <Icon icon="ri:pulse-line" />
            <h4>{{ t('iot.guide.statusTitle') }}</h4>
          </div>
          <div class="iot-guide-status-grid">
            <div v-for="item in guideStatuses" :key="item.key" class="iot-guide-status">
              <el-tag :type="item.type" effect="light">{{ t(item.label) }}</el-tag>
              <p>{{ t(item.desc) }}</p>
            </div>
          </div>
        </section>

        <section class="iot-guide-section">
          <div class="iot-guide-section-title">
            <Icon icon="ri:question-answer-line" />
            <h4>{{ t('iot.guide.faqTitle') }}</h4>
          </div>
          <el-collapse class="iot-guide-faq">
            <el-collapse-item v-for="item in guideFaqs" :key="item.key" :title="t(item.title)" :name="item.key">
              <p>{{ t(item.answer) }}</p>
            </el-collapse-item>
          </el-collapse>
        </section>

        <section class="iot-guide-security">
          <Icon icon="ri:shield-check-line" />
          <div>
            <h4>{{ t('iot.guide.securityTitle') }}</h4>
            <p>{{ t('iot.guide.securityDesc') }}</p>
          </div>
        </section>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../../api'
import { t } from '../../i18n'

const loading = ref(false)
const saving = ref(false)
const devices = ref([])
const formVisible = ref(false)
const formMode = ref('create')
const editingId = ref(null)
const previewVisible = ref(false)
const previewLoading = ref(false)
const previewData = ref({})
const helpVisible = ref(false)
const form = reactive(defaultForm())
const currentYear = new Date().getFullYear()

const previewMetrics = computed(() => Array.isArray(previewData.value?.metrics) ? previewData.value.metrics : [])
const guideStats = [
  { key: 'ap', icon: 'ri:wifi-line', label: 'iot.guide.stat.ap', value: 'HALMIFY-XXXX' },
  { key: 'password', icon: 'ri:key-2-line', label: 'iot.guide.stat.password', value: '123456789' },
  { key: 'setup', icon: 'ri:global-line', label: 'iot.guide.stat.setup', value: 'http://192.168.4.1' },
  { key: 'platform', icon: 'ri:cloud-line', label: 'iot.guide.stat.platform', value: 'https://crm.halmify.com' },
  { key: 'interval', icon: 'ri:timer-line', label: 'iot.guide.stat.interval', value: '10 min' }
]
const guideSteps = ['sn', 'register', 'wifi', 'configure', 'verify'].map((key) => ({
  key,
  title: `iot.guide.step.${key}.title`,
  desc: `iot.guide.step.${key}.desc`,
  code: key === 'wifi' ? 'HALMIFY-XXXX / 123456789' : key === 'configure' ? 'http://192.168.4.1' : ''
}))
const guideConfigRows = ['wifi', 'manual', 'password', 'platform', 'sleepStart', 'sleepEnd'].map((key) => ({
  key,
  label: `iot.guide.config.${key}.label`,
  desc: `iot.guide.config.${key}.desc`
}))
const guideStatuses = [
  { key: 'pending', label: 'iot.guide.status.pending.label', desc: 'iot.guide.status.pending.desc', type: 'warning' },
  { key: 'online', label: 'iot.guide.status.online.label', desc: 'iot.guide.status.online.desc', type: 'success' },
  { key: 'offline', label: 'iot.guide.status.offline.label', desc: 'iot.guide.status.offline.desc', type: 'info' },
  { key: 'seed', label: 'iot.guide.status.seed.label', desc: 'iot.guide.status.seed.desc', type: 'primary' },
  { key: 'lastSync', label: 'iot.guide.status.lastSync.label', desc: 'iot.guide.status.lastSync.desc', type: 'info' }
]
const guideFaqs = ['ap', 'setupPage', 'wifiFail', 'pending', 'offline'].map((key) => ({
  key,
  title: `iot.guide.faq.${key}.title`,
  answer: `iot.guide.faq.${key}.answer`
}))

onMounted(loadDevices)

function defaultForm() {
  return {
    name: '',
    sn: '',
    status: 'active',
    resolution_width: 200,
    resolution_height: 200,
    remark: ''
  }
}

async function loadDevices() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/iot/devices')
    devices.value = res?.data?.items || []
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || t('iot.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  formMode.value = 'create'
  editingId.value = null
  Object.assign(form, defaultForm())
  formVisible.value = true
}

function openEdit(row) {
  formMode.value = 'edit'
  editingId.value = row.id
  Object.assign(form, {
    name: row.name || '',
    sn: row.sn || '',
    status: row.status || 'active',
    resolution_width: Number(row.resolution_width || 200),
    resolution_height: Number(row.resolution_height || 200),
    remark: row.remark || ''
  })
  formVisible.value = true
}

async function submitForm() {
  if (!String(form.name || '').trim() || (formMode.value === 'create' && !String(form.sn || '').trim())) {
    ElMessage.warning(t('iot.required'))
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.name,
      sn: form.sn,
      status: form.status,
      resolution_width: form.resolution_width,
      resolution_height: form.resolution_height,
      remark: form.remark
    }
    const res = formMode.value === 'create'
      ? await api.post('/api/v1/iot/devices', payload)
      : await api.put(`/api/v1/iot/devices/${editingId.value}`, payload)
    formVisible.value = false
    ElMessage.success(formMode.value === 'create' ? t('iot.created') : t('iot.updated'))
    await loadDevices()
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || t('iot.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function deleteDevice(row) {
  try {
    await ElMessageBox.confirm(t('iot.deleteConfirm'), t('iot.deleteTitle'), {
      type: 'warning',
      confirmButtonText: t('iot.delete'),
      cancelButtonText: t('iot.cancel')
    })
    await api.delete(`/api/v1/iot/devices/${row.id}`)
    ElMessage.success(t('iot.deleted'))
    await loadDevices()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error?.response?.data?.message || error?.message || t('iot.deleteFailed'))
  }
}

async function rotateSecret(row) {
  try {
    await ElMessageBox.confirm(t('iot.rotateConfirm'), t('iot.rotateTitle'), {
      type: 'warning',
      confirmButtonText: t('iot.rotate'),
      cancelButtonText: t('iot.cancel')
    })
    await api.post(`/api/v1/iot/devices/${row.id}/rotate-secret`)
    ElMessage.success(t('iot.rotated'))
    await loadDevices()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error?.response?.data?.message || error?.message || t('iot.rotateFailed'))
  }
}

async function openPreview(row) {
  previewVisible.value = true
  previewLoading.value = true
  previewData.value = {}
  try {
    const res = await api.get(`/api/v1/iot/devices/${row.id}/preview`, { params: { year: currentYear } })
    previewData.value = res?.data || {}
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || t('iot.previewFailed'))
  } finally {
    previewLoading.value = false
  }
}


function statusLabel(status) {
  if (status === 'pending_bind') return t('iot.status.pendingBind')
  return status === 'active' ? t('iot.status.active') : t('iot.status.disabled')
}

function statusTagType(status) {
  if (status === 'active') return 'success'
  if (status === 'pending_bind') return 'warning'
  return 'info'
}

function modelLabel(model) {
  const normalized = String(model || '').trim()
  if (!normalized) return t('iot.modelUnknown')
  if (normalized === 'esp32-s3-epaper') return t('iot.model.s3')
  if (normalized === 'esp32-p4-epaper') return t('iot.model.p4')
  return normalized
}

function otaStatusLabel(status) {
  const key = String(status || 'latest').trim() || 'latest'
  return t(`iot.ota.status.${key}`)
}

function otaTagType(status) {
  if (status === 'success' || status === 'latest') return 'success'
  if (status === 'assigned' || status === 'available' || status === 'downloading') return 'warning'
  if (status === 'failed' || status === 'release_missing') return 'danger'
  return 'info'
}

function resolutionText(row) {
  return `${Number(row?.resolution_width || 200)}x${Number(row?.resolution_height || 200)}`
}

function metricLabel(metric) {
  return t(`iot.metric.${metric?.key || ''}`) || metric?.label || '-'
}

function formatTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  const pad = (number) => String(number).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}
</script>

<style scoped>
.iot-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.iot-hero,
.iot-panel {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.iot-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 20px;
}

.iot-hero h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: #111827;
}

.iot-hero p {
  margin: 8px 0 0;
  color: #6b7280;
  line-height: 1.6;
}

.iot-hero-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.iot-hero-actions .el-button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.iot-panel {
  padding: 12px;
}

.iot-table {
  width: 100%;
}

.iot-device-name {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.iot-device-name strong {
  color: #111827;
}

.iot-device-name span {
  color: #6b7280;
  font-size: 12px;
}

.iot-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.iot-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.iot-full {
  width: 100%;
}


.iot-preview {
  min-height: 180px;
}

.iot-preview-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 14px;
  color: #6b7280;
  font-size: 13px;
}

.iot-metric-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.iot-metric-card {
  min-height: 92px;
  padding: 14px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.iot-metric-card span {
  color: #6b7280;
  font-size: 13px;
}

.iot-metric-card strong {
  display: block;
  margin-top: 10px;
  color: #111827;
  font-size: 24px;
  line-height: 1.2;
  overflow-wrap: anywhere;
}

.iot-guide {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-bottom: 10px;
}

.iot-guide-intro,
.iot-guide-section,
.iot-guide-timeout,
.iot-guide-security {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.iot-guide-intro {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 14px;
  padding: 18px;
}

.iot-guide-intro-icon {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  border-radius: 8px;
  background: #eef2ff;
  color: #4f46e5;
  font-size: 24px;
}

.iot-guide-kicker {
  margin: 0 0 4px;
  color: #4f46e5;
  font-size: 12px;
  font-weight: 700;
}

.iot-guide-intro h3,
.iot-guide-section h4,
.iot-guide-security h4,
.iot-guide-step h5 {
  margin: 0;
  color: #111827;
}

.iot-guide-intro h3 {
  font-size: 20px;
  line-height: 1.35;
}

.iot-guide-intro p:last-child,
.iot-guide-step p,
.iot-guide-status p,
.iot-guide-security p,
.iot-guide-faq p {
  margin: 6px 0 0;
  color: #6b7280;
  line-height: 1.6;
}

.iot-guide-stat-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.iot-guide-stat {
  min-width: 0;
  padding: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.iot-guide-stat svg,
.iot-guide-stat .iconify {
  color: #0f766e;
  font-size: 18px;
}

.iot-guide-stat span {
  display: block;
  margin-top: 8px;
  color: #6b7280;
  font-size: 12px;
}

.iot-guide-stat strong {
  display: block;
  margin-top: 4px;
  color: #111827;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 13px;
  overflow-wrap: anywhere;
}

.iot-guide-section {
  padding: 16px;
}

.iot-guide-section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
}

.iot-guide-section-title .iconify {
  color: #2563eb;
  font-size: 20px;
}

.iot-guide-section h4,
.iot-guide-security h4 {
  font-size: 16px;
}

.iot-guide-steps {
  display: grid;
  gap: 10px;
}

.iot-guide-step {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  padding: 12px;
  border: 1px solid #edf0f5;
  border-radius: 8px;
  background: #f9fafb;
}

.iot-guide-step-index {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  border-radius: 50%;
  background: #111827;
  color: #fff;
  font-size: 13px;
  font-weight: 700;
}

.iot-guide-step h5 {
  font-size: 14px;
}

.iot-guide-step code {
  display: inline-block;
  margin-top: 8px;
  padding: 5px 7px;
  border-radius: 6px;
  background: #fff;
  color: #111827;
  font-size: 12px;
  overflow-wrap: anywhere;
}

.iot-guide-config {
  overflow: hidden;
  border: 1px solid #edf0f5;
  border-radius: 8px;
}

.iot-guide-config-row {
  display: grid;
  grid-template-columns: 160px minmax(0, 1fr);
  gap: 12px;
  padding: 12px;
  background: #fff;
}

.iot-guide-config-row + .iot-guide-config-row {
  border-top: 1px solid #edf0f5;
}

.iot-guide-config-row span {
  color: #111827;
  font-weight: 700;
  overflow-wrap: anywhere;
}

.iot-guide-config-row p {
  margin: 0;
  color: #6b7280;
  line-height: 1.5;
}

.iot-guide-status-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.iot-guide-status {
  min-height: 104px;
  padding: 12px;
  border: 1px solid #edf0f5;
  border-radius: 8px;
  background: #fff;
}

.iot-guide-faq {
  border-top: 0;
  border-bottom: 0;
}

.iot-guide-security {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  padding: 14px;
  background: #fffbeb;
  border-color: #fde68a;
}

.iot-guide-timeout {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  padding: 14px;
  background: #fff7ed;
  border-color: #fed7aa;
}

.iot-guide-timeout .iconify {
  color: #c2410c;
  font-size: 22px;
}

.iot-guide-timeout h4 {
  margin: 0;
  color: #111827;
  font-size: 16px;
}

.iot-guide-timeout p {
  margin: 6px 0 0;
  color: #6b7280;
  line-height: 1.6;
}

.iot-guide-timeout code {
  display: inline-block;
  margin-top: 8px;
  padding: 5px 7px;
  border-radius: 6px;
  background: #fff;
  color: #9a3412;
  font-size: 12px;
  font-weight: 700;
}

.iot-guide-security .iconify {
  color: #b45309;
  font-size: 22px;
}

@media (max-width: 760px) {
  .iot-hero {
    flex-direction: column;
  }

  .iot-hero-actions {
    justify-content: flex-start;
  }

  .iot-form-grid,
  .iot-metric-grid {
    grid-template-columns: 1fr;
  }

  .iot-guide-intro,
  .iot-guide-security,
  .iot-guide-timeout,
  .iot-guide-config-row,
  .iot-guide-status-grid,
  .iot-guide-stat-grid {
    grid-template-columns: 1fr;
  }

  .iot-guide-config-row {
    gap: 6px;
  }
}
</style>
