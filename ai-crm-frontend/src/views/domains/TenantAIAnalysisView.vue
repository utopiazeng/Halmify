<template>
  <div class="system-page ai-analysis-page">
    <section v-if="!aiConfigured" class="ai-analysis-open">
      <div class="ai-analysis-landing-hero">
        <div class="ai-analysis-hero-copy">
          <el-tag type="success">{{ t('aiAnalysis.open.badge') }}</el-tag>
          <h1>{{ t('aiAnalysis.open.heroTitle') }}</h1>
          <p>{{ t('aiAnalysis.open.heroDesc') }}</p>
          <div class="ai-analysis-hero-actions">
            <el-button type="primary" size="large" @click="focusSettings">{{ t('aiAnalysis.settings.primaryCta') }}</el-button>
            <span>{{ t('aiAnalysis.settings.ctaNote') }}</span>
          </div>
        </div>
        <div class="ai-analysis-hero-visual" aria-hidden="true">
          <div class="ai-analysis-report-preview">
            <div class="ai-analysis-preview-head">
              <span>{{ t('aiAnalysis.preview.title') }}</span>
              <strong>{{ t('aiAnalysis.preview.status') }}</strong>
            </div>
            <div class="ai-analysis-preview-summary">{{ t('aiAnalysis.preview.summary') }}</div>
            <div class="ai-analysis-preview-list">
              <div v-for="item in previewItems" :key="item.key">
                <strong>{{ item.title }}</strong>
                <span>{{ item.desc }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="ai-analysis-promise-strip">
        <div v-for="item in promiseItems" :key="item.key" class="ai-analysis-promise">
          <strong>{{ item.title }}</strong>
          <span>{{ item.desc }}</span>
        </div>
      </div>

      <div class="ai-analysis-landing-grid">
        <div class="ai-analysis-outcomes">
          <div class="ai-analysis-section-head">
            <span>{{ t('aiAnalysis.open.outcomeEyebrow') }}</span>
            <h2>{{ t('aiAnalysis.open.outcomeTitle') }}</h2>
            <p>{{ t('aiAnalysis.open.outcomeDesc') }}</p>
          </div>
          <div class="ai-analysis-value-grid">
            <div v-for="item in outcomeCards" :key="item.key" class="ai-analysis-value">
              <strong>{{ item.title }}</strong>
              <span>{{ item.desc }}</span>
            </div>
          </div>
        </div>
        <aside class="ai-analysis-pricing">
          <div class="ai-analysis-pricing-head">
            <strong>{{ t('aiAnalysis.pricing.title') }}</strong>
            <em>{{ t('aiAnalysis.pricing.currentModel', { model: pricingModelName }) }}</em>
            <span>{{ t('aiAnalysis.pricing.note') }}</span>
          </div>
          <el-table :data="pricingRows" size="small">
            <el-table-column prop="metric" :label="t('aiAnalysis.pricing.metric')" min-width="160" />
            <el-table-column prop="price" :label="t('aiAnalysis.pricing.pricePerMillion')" width="180" />
          </el-table>
          <p class="ai-analysis-billing-note">{{ t('aiAnalysis.settings.billingNote') }}</p>
        </aside>
      </div>
    </section>

    <section ref="settingsRef" class="system-card ai-settings-card">
      <div class="system-toolbar">
        <div>
          <strong>{{ t('aiAnalysis.settings.title') }}</strong>
          <p>{{ t('aiAnalysis.settings.desc') }}</p>
        </div>
        <el-tag :type="aiConfigured ? 'success' : 'warning'">{{ aiConfigured ? t('aiAnalysis.settings.configured') : t('aiAnalysis.settings.notConfigured') }}</el-tag>
      </div>
      <el-form :model="settingsForm" label-position="top" class="ai-settings-form">
        <el-form-item :label="t('aiAnalysis.settings.provider')">
          <el-select v-model="settingsForm.provider" @change="applyProviderDefaults">
            <el-option v-for="item in providerOptions" :key="item.value" :value="item.value" :label="item.label" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('aiAnalysis.settings.apiUrl')">
          <el-input v-model="settingsForm.api_url" />
        </el-form-item>
        <el-form-item :label="t('aiAnalysis.settings.model')">
          <el-select v-model="settingsForm.model" filterable allow-create>
            <el-option v-for="item in selectedProviderModels" :key="item" :value="item" :label="item" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('aiAnalysis.settings.apiKey')">
          <el-input v-model="settingsForm.api_key" type="password" show-password :placeholder="apiKeyPlaceholder" />
        </el-form-item>
        <div class="ai-settings-actions">
          <el-button type="primary" :loading="savingSettings" :disabled="!status.can_manage_settings" @click="saveSettings">{{ t('aiAnalysis.settings.save') }}</el-button>
          <el-button @click="openProviderGuide">{{ t('aiAnalysis.settings.openGuide') }}</el-button>
          <span>{{ t('aiAnalysis.settings.adminOnly') }}</span>
        </div>
      </el-form>
    </section>

    <section v-if="status.enabled" class="ai-wallet-row">
      <div class="system-card ai-wallet-card">
        <span>{{ t('aiAnalysis.settings.currentProvider') }}</span>
        <strong>{{ selectedProviderLabel }}</strong>
      </div>
      <div class="system-card ai-wallet-card">
        <span>{{ t('aiAnalysis.settings.currentModel') }}</span>
        <strong>{{ status.ai_settings?.model || '-' }}</strong>
      </div>
      <div class="system-card ai-wallet-card">
        <span>{{ t('aiAnalysis.wallet.monthly') }}</span>
        <strong>{{ formatNumber(status.wallet?.monthly_token_used) }} / {{ status.wallet?.monthly_token_limit ? formatNumber(status.wallet.monthly_token_limit) : t('aiAnalysis.unlimited') }}</strong>
      </div>
    </section>

    <section v-if="status.enabled" class="system-card ai-generate-card">
      <div class="system-toolbar">
        <div class="toolbar-left">
          <el-select v-model="form.period_type" style="width: 150px">
            <el-option value="month" :label="t('aiAnalysis.period.month')" />
            <el-option value="quarter" :label="t('aiAnalysis.period.quarter')" />
            <el-option value="year" :label="t('aiAnalysis.period.year')" />
            <el-option value="custom" :label="t('aiAnalysis.period.custom')" />
          </el-select>
          <el-date-picker v-if="form.period_type === 'custom'" v-model="customRange" type="daterange" value-format="YYYY-MM-DD" />
          <el-select v-model="form.depth" style="width: 140px">
            <el-option value="executive" :label="t('aiAnalysis.depth.executive')" />
            <el-option value="deep" :label="t('aiAnalysis.depth.deep')" />
          </el-select>
          <el-button :loading="estimating" :disabled="estimateDisabled" @click="estimateCost">{{ t('aiAnalysis.action.estimate') }}</el-button>
          <el-button type="primary" :loading="creating" :disabled="generateDisabled" @click="createJob">{{ t('aiAnalysis.action.generate') }}</el-button>
        </div>
        <el-tag v-if="estimate">{{ t('aiAnalysis.estimate.tokens') }} {{ formatNumber(estimate.estimated_total_tokens) }}</el-tag>
      </div>
      <el-alert v-if="generateDisabledReason" class="ai-generate-alert" :closable="false" type="warning" :title="generateDisabledReason" />
      <div v-if="activeJobProgress.visible" class="ai-progress-row">
        <span>{{ activeJobProgress.text }}</span>
        <el-progress :percentage="activeJobProgress.percent" :status="activeJobProgress.status" />
      </div>
      <div class="ai-section-picker">
        <span>{{ t('aiAnalysis.section.pickLabel') }}</span>
        <el-checkbox-group v-model="form.sections" class="ai-section-checks">
          <el-checkbox v-for="item in sectionOptions" :key="item.value" :label="item.value">{{ item.label }}</el-checkbox>
        </el-checkbox-group>
      </div>
    </section>

    <section v-if="status.enabled" class="ai-analysis-main">
      <div class="system-card">
        <div class="system-toolbar">
          <strong>{{ t('aiAnalysis.report.history') }}</strong>
          <el-button @click="loadJobs">{{ t('common.action.refresh') }}</el-button>
        </div>
        <el-table :data="jobs" v-loading="loadingJobs" @row-click="selectJob">
          <el-table-column prop="id" :label="t('aiAnalysis.report.id')" width="90" />
          <el-table-column prop="status" :label="t('common.table.status')" width="120">
            <template #default="{ row }"><el-tag :type="jobStatusType(row.status)">{{ jobStatusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column :label="t('aiAnalysis.report.period')" min-width="210">
            <template #default="{ row }">{{ row.input?.period?.label || '-' }}</template>
          </el-table-column>
          <el-table-column :label="t('aiAnalysis.usage.tokens')" width="160">
            <template #default="{ row }">{{ formatNumber(row.usage?.total_tokens || 0) }}</template>
          </el-table-column>
          <el-table-column :label="t('aiAnalysis.usage.referenceCost')" width="130">
            <template #default="{ row }">{{ row.usage?.currency || '' }} {{ row.usage?.cost_amount || '0.000000' }}</template>
          </el-table-column>
        </el-table>
      </div>

      <article ref="reportRef" class="system-card ai-report-card">
        <div class="system-toolbar">
          <strong>{{ t('aiAnalysis.report.detail') }}</strong>
          <el-button :disabled="!selectedJob || selectedJob.status !== 'succeeded'" @click="exportPDF">{{ t('aiAnalysis.action.exportPdf') }}</el-button>
        </div>
        <template v-if="selectedJob">
          <h2>{{ t('aiAnalysis.report.title') }}</h2>
          <p class="ai-report-summary">{{ selectedJob.result?.executive_summary || selectedJob.error_message || '-' }}</p>
          <div class="ai-report-meta">
            <span>{{ t('aiAnalysis.report.period') }}: {{ selectedJob.input?.period?.label || '-' }}</span>
            <span>{{ t('aiAnalysis.usage.tokens') }}: {{ formatNumber(selectedJob.usage?.total_tokens || 0) }}</span>
            <span>{{ t('aiAnalysis.usage.referenceCost') }}: {{ selectedJob.usage?.currency || '' }} {{ selectedJob.usage?.cost_amount || '0.000000' }}</span>
          </div>
          <section v-for="block in reportBlocks" :key="block.key" class="ai-report-section">
            <h3>{{ block.title }}</h3>
            <div v-if="Array.isArray(block.items) && block.items.length" class="ai-report-list">
              <div v-for="(item, idx) in block.items" :key="idx">
                <strong>{{ item.title || item.name || item.action || item.priority || '-' }}</strong>
                <p>{{ item.summary || item.comment || item.impact || item.owner_suggestion || item.value || '-' }}</p>
              </div>
            </div>
            <el-empty v-else :description="t('aiAnalysis.report.empty')" />
          </section>
        </template>
        <el-empty v-else :description="t('aiAnalysis.report.select')" />
      </article>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import { t } from '../../i18n'

const status = ref({})
const jobs = ref([])
const selectedJob = ref(null)
const estimate = ref(null)
const loadingJobs = ref(false)
const creating = ref(false)
const estimating = ref(false)
const savingSettings = ref(false)
const customRange = ref([])
const reportRef = ref(null)
const settingsRef = ref(null)
const pricing = ref(null)
const pollingTimer = ref(null)

const form = reactive({
  period_type: 'month',
  depth: 'executive',
  sections: ['overview', 'customers', 'sales_pipeline', 'revenue_orders', 'products_projects', 'service_delivery', 'team_execution']
})
const settingsForm = reactive({ provider: 'deepseek', api_url: 'https://api.deepseek.com', model: 'deepseek-v4-flash', api_key: '' })

const sectionOptions = computed(() => [
  { value: 'overview', label: t('aiAnalysis.section.overview') },
  { value: 'customers', label: t('aiAnalysis.section.customers') },
  { value: 'sales_pipeline', label: t('aiAnalysis.section.sales') },
  { value: 'revenue_orders', label: t('aiAnalysis.section.revenue') },
  { value: 'products_projects', label: t('aiAnalysis.section.products') },
  { value: 'service_delivery', label: t('aiAnalysis.section.service') },
  { value: 'team_execution', label: t('aiAnalysis.section.team') }
])

const promiseItems = computed(() => [
  { key: 'speed', title: t('aiAnalysis.promise.speed.title'), desc: t('aiAnalysis.promise.speed.desc') },
  { key: 'evidence', title: t('aiAnalysis.promise.evidence.title'), desc: t('aiAnalysis.promise.evidence.desc') },
  { key: 'contribution', title: t('aiAnalysis.promise.contribution.title'), desc: t('aiAnalysis.promise.contribution.desc') },
  { key: 'billing', title: t('aiAnalysis.promise.billing.title'), desc: t('aiAnalysis.promise.billing.desc') }
])

const outcomeCards = computed(() => [
  { key: 'customer', title: t('aiAnalysis.value.customer.title'), desc: t('aiAnalysis.value.customer.desc') },
  { key: 'sales', title: t('aiAnalysis.value.sales.title'), desc: t('aiAnalysis.value.sales.desc') },
  { key: 'risk', title: t('aiAnalysis.value.risk.title'), desc: t('aiAnalysis.value.risk.desc') },
  { key: 'action', title: t('aiAnalysis.value.action.title'), desc: t('aiAnalysis.value.action.desc') }
])

const previewItems = computed(() => [
  { key: 'customer', title: t('aiAnalysis.preview.customer.title'), desc: t('aiAnalysis.preview.customer.desc') },
  { key: 'pipeline', title: t('aiAnalysis.preview.pipeline.title'), desc: t('aiAnalysis.preview.pipeline.desc') },
  { key: 'cash', title: t('aiAnalysis.preview.cash.title'), desc: t('aiAnalysis.preview.cash.desc') }
])

const pricingRows = computed(() => {
  const rows = pricingSnapshotRows(pricing.value)
  return rows.length ? rows : fallbackPricingRows()
})

const pricingModelName = computed(() => {
  const model = (pricing.value?.models || []).find((item) => item?.model || item?.display_name)
  return model?.display_name || model?.model || 'deepseek-v4-flash'
})

const aiConfigured = computed(() => Boolean(status.value?.configured || status.value?.ai_settings?.configured))
const providerOptions = computed(() => status.value?.provider_options || [])
const selectedProvider = computed(() => providerOptions.value.find((item) => item.value === settingsForm.provider) || providerOptions.value[0] || {})
const selectedProviderModels = computed(() => selectedProvider.value?.models || [])
const selectedProviderLabel = computed(() => providerOptions.value.find((item) => item.value === status.value?.ai_settings?.provider)?.label || status.value?.ai_settings?.provider || '-')
const apiKeyPlaceholder = computed(() => status.value?.ai_settings?.has_api_key ? status.value.ai_settings.masked_api_key : t('aiAnalysis.settings.apiKeyPlaceholder'))
const runningJob = computed(() => jobs.value.find((item) => ['pending', 'running'].includes(String(item.status || ''))))
const hasRunningJob = computed(() => Boolean(runningJob.value))
const estimateCanGenerate = computed(() => estimate.value ? estimate.value.can_generate !== false : true)
const estimateBlockReason = computed(() => estimate.value?.block_reason || '')
const estimateDisabled = computed(() => creating.value || estimating.value || hasRunningJob.value || !aiConfigured.value)
const generateDisabled = computed(() => creating.value || estimating.value || hasRunningJob.value || !estimateCanGenerate.value || !aiConfigured.value)
const generateDisabledReason = computed(() => {
  if (!aiConfigured.value) return t('aiAnalysis.message.settingsRequired')
  if (hasRunningJob.value) return t('aiAnalysis.message.running')
  if (estimateBlockReason.value) return estimateBlockReason.value
  return ''
})
const activeJobProgress = computed(() => {
  const job = runningJob.value || selectedJob.value
  const statusValue = String(job?.status || '')
  if (!['pending', 'running', 'succeeded', 'failed', 'billing_failed'].includes(statusValue)) {
    return { visible: false, percent: 0, text: '', status: undefined }
  }
  const percentMap = { pending: 20, running: 65, succeeded: 100, failed: 100, billing_failed: 100 }
  const statusMap = { succeeded: 'success', failed: 'exception', billing_failed: 'exception' }
  const textMap = {
    pending: t('aiAnalysis.progress.pending'),
    running: t('aiAnalysis.progress.running'),
    succeeded: t('aiAnalysis.progress.succeeded'),
    failed: t('aiAnalysis.progress.failed'),
    billing_failed: t('aiAnalysis.progress.billingFailed')
  }
  return { visible: true, percent: percentMap[statusValue] || 0, text: textMap[statusValue] || '-', status: statusMap[statusValue] }
})

const reportBlocks = computed(() => {
  const r = selectedJob.value?.result || {}
  return [
    { key: 'metrics', title: t('aiAnalysis.report.metrics'), items: r.key_metrics },
    { key: 'customers', title: t('aiAnalysis.report.customers'), items: r.customer_insights },
    { key: 'sales', title: t('aiAnalysis.report.sales'), items: r.sales_pipeline },
    { key: 'revenue', title: t('aiAnalysis.report.revenue'), items: r.revenue_and_orders },
    { key: 'service', title: t('aiAnalysis.report.service'), items: r.service_and_delivery },
    { key: 'team', title: t('aiAnalysis.report.team'), items: r.team_execution },
    { key: 'risks', title: t('aiAnalysis.report.risks'), items: r.risks },
    { key: 'opportunities', title: t('aiAnalysis.report.opportunities'), items: r.opportunities },
    { key: 'actions', title: t('aiAnalysis.report.actions'), items: r.recommended_actions }
  ]
})

onMounted(async () => {
  await loadStatus()
  if (status.value.enabled) await loadJobs()
  startJobPolling()
})

onUnmounted(() => {
  stopJobPolling()
})

async function loadStatus() {
  const res = await api.get('/api/v1/prd/flows/tenant-ai-analysis/status')
  status.value = res.data || {}
  pricing.value = status.value.pricing || null
  syncSettingsForm()
  if (!pricingSnapshotRows(pricing.value).length) {
    try {
      const priceRes = await api.get('/api/v1/prd/ai/pricing')
      if (priceRes.data?.pricing?.models?.length) pricing.value = priceRes.data.pricing
    } catch {
      pricing.value = null
    }
  }
}

function syncSettingsForm() {
  const settings = status.value?.ai_settings || {}
  settingsForm.provider = settings.provider || 'deepseek'
  settingsForm.api_url = settings.api_url || providerOptions.value.find((item) => item.value === settingsForm.provider)?.default_url || 'https://api.deepseek.com'
  settingsForm.model = settings.model || providerOptions.value.find((item) => item.value === settingsForm.provider)?.default_model || 'deepseek-v4-flash'
  settingsForm.api_key = ''
}

function pricingSnapshotRows(snapshot) {
  const model = (snapshot?.models || []).find((item) => Array.isArray(item.rates) && item.rates.length)
  return (model?.rates || [])
    .filter((rate) => rate && (rate.tenant_price_amount || rate.tenant_price_micros_per_million))
    .map((rate) => ({
      metric: pricingMetricLabel(rate),
      price: `${rate.currency || snapshot?.currency || 'USD'} ${rate.tenant_price_amount || microsToAmount(rate.tenant_price_micros_per_million)}`
    }))
}

function pricingMetricLabel(rate) {
  const metric = String(rate?.metric || '')
  if (metric === 'input_cache_hit') return t('aiAnalysis.pricing.inputCacheHit')
  if (metric === 'input_cache_miss') return t('aiAnalysis.pricing.inputCacheMiss')
  if (metric === 'output') return t('aiAnalysis.pricing.output')
  return rate?.label || metric || '-'
}

function fallbackPricingRows() {
  return [
    { metric: t('aiAnalysis.pricing.inputCacheHit'), price: 'USD 0.003220' },
    { metric: t('aiAnalysis.pricing.inputCacheMiss'), price: 'USD 0.161000' },
    { metric: t('aiAnalysis.pricing.output'), price: 'USD 0.322000' }
  ]
}

function microsToAmount(value) {
  const amount = Number(value || 0) / 1000000
  if (!Number.isFinite(amount) || amount <= 0) {
    return '0.000000'
  }
  return amount.toFixed(6)
}

function applyProviderDefaults() {
  const option = providerOptions.value.find((item) => item.value === settingsForm.provider)
  if (!option) return
  settingsForm.api_url = option.default_url || settingsForm.api_url
  settingsForm.model = option.default_model || settingsForm.model
  settingsForm.api_key = ''
}

function focusSettings() {
  settingsRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function openProviderGuide() {
  const url = selectedProvider.value?.help_url
  if (url) window.open(url, '_blank', 'noopener,noreferrer')
}

async function saveSettings() {
  if (!status.value.can_manage_settings) {
    ElMessage.warning(t('aiAnalysis.settings.noPermission'))
    return
  }
  savingSettings.value = true
  try {
    const payload = { ...settingsForm }
    if (!payload.api_key && status.value?.ai_settings?.has_api_key) {
      payload.api_key = status.value.ai_settings.masked_api_key
    }
    const res = await api.put('/api/v1/prd/flows/tenant-ai-analysis/settings', payload)
    status.value = { ...status.value, ai_settings: res.data?.settings, configured: res.data?.settings?.configured }
    syncSettingsForm()
    ElMessage.success(t('aiAnalysis.settings.saved'))
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    savingSettings.value = false
  }
}

function requestPayload() {
  const payload = { ...form }
  if (form.period_type === 'custom' && customRange.value?.length === 2) {
    payload.start_date = customRange.value[0]
    payload.end_date = customRange.value[1]
  }
  return payload
}

async function estimateCost() {
  if (estimating.value || creating.value) return
  if (!aiConfigured.value) {
    ElMessage.warning(t('aiAnalysis.message.settingsRequired'))
    focusSettings()
    return
  }
  estimating.value = true
  try {
    const res = await api.post('/api/v1/prd/flows/tenant-ai-analysis/estimate', requestPayload())
    estimate.value = res.data
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    estimating.value = false
  }
}

async function createJob() {
  if (generateDisabled.value) {
    if (generateDisabledReason.value) ElMessage.warning(generateDisabledReason.value)
    if (!aiConfigured.value) focusSettings()
    return
  }
  creating.value = true
  try {
    const key = `ai-analysis-${Date.now()}-${Math.random().toString(16).slice(2)}`
    const res = await api.post('/api/v1/prd/flows/tenant-ai-analysis/jobs', requestPayload(), { headers: { 'Idempotency-Key': key } })
    selectedJob.value = res.data
    ElMessage.success(t('aiAnalysis.message.created'))
    await loadJobs()
    startJobPolling()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    creating.value = false
  }
}

async function loadJobs() {
  loadingJobs.value = true
  try {
    const res = await api.get('/api/v1/prd/flows/tenant-ai-analysis/jobs')
    jobs.value = res.data?.items || []
    if (!selectedJob.value && jobs.value.length) selectedJob.value = jobs.value[0]
    if (selectedJob.value?.id) {
      const latest = jobs.value.find((item) => Number(item.id) === Number(selectedJob.value.id))
      if (latest) selectedJob.value = latest
    }
    if (!hasRunningJob.value) stopJobPolling()
  } finally {
    loadingJobs.value = false
  }
}

function startJobPolling() {
  if (pollingTimer.value || !status.value.enabled) return
  pollingTimer.value = setInterval(async () => {
    if (!hasRunningJob.value) {
      stopJobPolling()
      return
    }
    await loadJobs()
  }, 3000)
}

function stopJobPolling() {
  if (!pollingTimer.value) return
  clearInterval(pollingTimer.value)
  pollingTimer.value = null
}

function selectJob(row) {
  selectedJob.value = row
}

function exportPDF() {
  const html = reportRef.value?.innerHTML || ''
  const win = window.open('', '_blank')
  if (!win) return
  win.document.write(`<html><head><title>${t('aiAnalysis.report.title')}</title><style>body{font-family:Arial,sans-serif;padding:24px;color:#172033}h2,h3{margin:16px 0 8px}.system-toolbar button{display:none}.ai-report-meta{display:flex;gap:12px;flex-wrap:wrap;color:#667085}.ai-report-list>div{border:1px solid #e5e7eb;margin:8px 0;padding:10px;border-radius:6px}</style></head><body>${html}</body></html>`)
  win.document.close()
  win.focus()
  win.print()
}

function formatNumber(value) {
  return Number(value || 0).toLocaleString()
}

function jobStatusLabel(value) {
  return ({ pending: t('aiAnalysis.status.pending'), running: t('aiAnalysis.status.running'), succeeded: t('aiAnalysis.status.succeeded'), failed: t('aiAnalysis.status.failed'), billing_failed: t('aiAnalysis.status.billingFailed') }[value] || value || '-')
}

function jobStatusType(value) {
  return ({ succeeded: 'success', failed: 'danger', billing_failed: 'danger', running: 'warning', pending: 'info' }[value] || 'info')
}
</script>
