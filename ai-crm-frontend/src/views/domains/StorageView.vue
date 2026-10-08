<template>
  <div class="storage-page">
    <div class="storage-hero">
      <div>
        <div class="storage-eyebrow">{{ t('entitlements.page.eyebrow') }}</div>
        <h1>{{ t('entitlements.page.title') }}</h1>
        <p>{{ t('entitlements.page.subtitle') }}</p>
      </div>
      <div class="storage-hero-actions">
        <el-button @click="load">{{ t('common.action.refresh') }}</el-button>
        <el-button type="primary" @click="openPurchase(activeResource)">{{ t('entitlements.action.request') }}</el-button>
      </div>
    </div>

    <el-skeleton v-if="loading && !storageUsage" :rows="8" animated />
    <template v-else>
      <el-tabs v-model="activeResource" class="entitlement-tabs">
        <el-tab-pane :label="t('entitlements.resource.storage')" name="storage" />
        <el-tab-pane :label="t('entitlements.resource.seats')" name="seats" />
      </el-tabs>

      <div class="storage-grid">
        <section class="storage-panel storage-usage-panel">
          <div class="storage-panel-head">
            <div>
              <div class="storage-panel-label">{{ currentLabels.used }}</div>
              <div class="storage-used">{{ currentUsed }}</div>
            </div>
            <el-tag :type="currentStatusType">{{ currentStatusText }}</el-tag>
          </div>
          <el-progress
            :percentage="currentPercent"
            :stroke-width="16"
            :status="currentPercent >= 90 ? 'exception' : currentPercent >= 75 ? 'warning' : 'success'"
          />
          <div class="storage-progress-meta">
            <span>{{ currentLabels.total }}: {{ currentTotal }}</span>
            <span>{{ currentLabels.expire }}: {{ formatDate(currentExpireAt) }}</span>
          </div>
          <div class="entitlement-policy-note">{{ currentPolicyText }}</div>
        </section>

        <section class="storage-panel storage-console">
          <div class="storage-console-row">
            <span>{{ currentLabels.base }}</span>
            <strong>{{ currentBase }}</strong>
          </div>
          <div class="storage-console-row">
            <span>{{ currentLabels.expansion }}</span>
            <strong>{{ currentExpansion }}</strong>
          </div>
          <div class="storage-console-row">
            <span>{{ t('storage.metric.nextExpire') }}</span>
            <strong>{{ formatDate(currentExpireAt) }}</strong>
          </div>
          <div class="storage-console-row">
            <span>{{ t('storage.metric.price') }}</span>
            <strong>{{ currentUnitPrice }}</strong>
          </div>
        </section>
      </div>

      <section class="storage-panel">
        <div class="storage-panel-head">
          <div>
            <div class="storage-panel-label">{{ t('storage.records.title') }}</div>
            <div class="storage-panel-note">{{ t('storage.records.subtitle') }}</div>
          </div>
        </div>
        <el-table :data="currentRecords" v-loading="loading">
          <el-table-column :prop="activeResource === 'storage' ? 'capacity_gb' : 'seat_count'" :label="currentLabels.quantity" width="140">
            <template #default="{ row }">{{ activeResource === 'storage' ? `${row.capacity_gb} GB` : `${row.seat_count} ${t('entitlements.unit.person')}` }}</template>
          </el-table-column>
          <el-table-column prop="years" :label="t('storage.table.years')" width="110" />
          <el-table-column prop="status" :label="t('common.table.status')" width="130">
            <template #default="{ row }"><el-tag :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="valid_to" :label="t('storage.table.validTo')" width="190">
            <template #default="{ row }">{{ formatDate(row.valid_to) }}</template>
          </el-table-column>
          <el-table-column prop="total_amount_cents" :label="t('storage.table.amount')" width="150">
            <template #default="{ row }">{{ row.total_amount_display || formatCents(row.total_amount_cents, row.currency) }}</template>
          </el-table-column>
          <el-table-column prop="remark" :label="t('storage.table.remark')" min-width="180" />
        </el-table>
      </section>
    </template>

    <el-dialog v-model="purchaseVisible" :title="purchaseTitle" width="680px">
      <el-form :model="purchaseForm" label-width="170px">
        <el-form-item :label="purchaseQuantityLabel">
          <el-input-number v-model="purchaseForm.quantity" :min="0" :max="purchaseResource === 'storage' ? 1024 : 10000" style="width: 100%" @change="loadQuote" />
          <div class="entitlement-form-hint">{{ t('entitlements.form.zeroQuantityHint') }}</div>
        </el-form-item>
        <el-form-item :label="t('entitlements.dialog.targetYears')">
          <el-input-number v-model="purchaseForm.years" :min="1" :max="10" style="width: 100%" @change="loadQuote" />
        </el-form-item>
        <div v-if="quote" class="quote-box">
          <div><span>{{ t('entitlements.quote.unitPrice') }}</span><strong>{{ quote.unit_price_display }} / {{ quoteUnitLabel }} / {{ quotePricePeriodLabel }}</strong></div>
          <div><span>{{ t('entitlements.quote.currentCredit') }}</span><strong>{{ quote.current_credit_display }}</strong></div>
          <div><span>{{ t('entitlements.quote.targetQuantity') }}</span><strong>{{ quote.target_quantity }} {{ quoteUnitLabel }}</strong></div>
          <div><span>{{ t('entitlements.quote.targetValidTo') }}</span><strong>{{ formatDate(quote.target_valid_to) }}</strong></div>
          <div class="quote-due"><span>{{ t('entitlements.quote.due') }}</span><strong>{{ quote.due_display }}</strong></div>
          <p>{{ quoteFormulaSummary }}</p>
        </div>
        <el-form-item :label="t('storage.dialog.remark')">
          <el-input v-model="purchaseForm.remark" type="textarea" :rows="3" :placeholder="t('storage.dialog.remarkPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="purchaseVisible = false">{{ t('common.action.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitPurchase">{{ t('storage.action.submitRequest') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import { t } from '../../i18n'

const loading = ref(false)
const quoteLoading = ref(false)
const submitting = ref(false)
const purchaseVisible = ref(false)
const activeResource = ref('storage')
const purchaseResource = ref('storage')
const storageUsage = ref(null)
const seatUsage = ref(null)
const quote = ref(null)
const purchaseForm = reactive(defaultPurchaseForm())

const currentPercent = computed(() => Math.max(0, Math.min(100, Number(currentUsage.value?.usage_percent || 0))))
const currentUsage = computed(() => activeResource.value === 'storage' ? storageUsage.value : seatUsage.value)
const currentExpireAt = computed(() => currentUsage.value?.current_expire_at || currentUsage.value?.unified_expire_at || currentUsage.value?.next_expire_at || '')
const currentRecords = computed(() => [
  ...(currentUsage.value?.pending_request_records || []),
  ...(currentUsage.value?.history_records || currentUsage.value?.expansion_records || [])
])
const currentLabels = computed(() => activeResource.value === 'storage'
  ? { used: t('storage.metric.used'), total: t('storage.metric.total'), expire: t('storage.metric.nextExpire'), base: t('storage.metric.base'), expansion: t('storage.metric.expansion'), quantity: t('storage.table.capacity') }
  : { used: t('entitlements.seats.currentUsers'), total: t('entitlements.seats.maxUsers'), expire: t('storage.metric.nextExpire'), base: t('entitlements.seats.base'), expansion: t('entitlements.seats.active'), quantity: t('entitlements.seats.quantity') })
const currentUsed = computed(() => activeResource.value === 'storage' ? formatBytes(storageUsage.value?.used_bytes) : `${seatUsage.value?.current_users || 0} ${t('entitlements.unit.person')}`)
const currentTotal = computed(() => activeResource.value === 'storage' ? formatBytes(storageUsage.value?.total_bytes) : `${seatUsage.value?.max_users || 0} ${t('entitlements.unit.person')}`)
const currentBase = computed(() => activeResource.value === 'storage' ? formatBytes(storageUsage.value?.base_bytes) : t('entitlements.seats.noFreeBase'))
const currentExpansion = computed(() => activeResource.value === 'storage' ? `${storageUsage.value?.expansion_gb || 0} GB` : `${seatUsage.value?.seat_count || 0} ${t('entitlements.unit.person')}`)
const currentUnitPrice = computed(() => activeResource.value === 'storage'
  ? `${storageUsage.value?.price_per_gb_year || 6} USD / GB / ${t('storage.unit.year')}`
  : `${seatUsage.value?.price_per_seat_month || 1} USD / ${t('entitlements.unit.person')} / ${t('storage.unit.month')}`)
const currentStatusType = computed(() => currentUsage.value?.upload_locked ? 'danger' : currentUsage.value?.expiring_soon ? 'warning' : 'success')
const currentStatusText = computed(() => currentUsage.value?.upload_locked ? t('storage.status.uploadPaused') : currentUsage.value?.expiring_soon ? t('storage.status.expiringSoon') : t('storage.status.uploadAvailable'))
const currentPolicyText = computed(() => activeResource.value === 'storage' ? t('entitlements.policy.storageExpiry') : t('entitlements.policy.seatExpiry'))
const purchaseTitle = computed(() => purchaseResource.value === 'storage' ? t('storage.dialog.title') : t('entitlements.seats.dialogTitle'))
const purchaseQuantityLabel = computed(() => purchaseResource.value === 'storage' ? t('storage.dialog.capacity') : t('entitlements.seats.addQuantity'))
const quoteUnitLabel = computed(() => purchaseResource.value === 'storage' ? 'GB' : t('entitlements.unit.person'))
const quotePricePeriodLabel = computed(() => purchaseResource.value === 'storage' ? t('storage.unit.year') : t('storage.unit.month'))
const quoteFormulaSummary = computed(() => formatQuoteFormula(quote.value, purchaseResource.value))

watch([() => purchaseForm.quantity, () => purchaseForm.years, purchaseResource], loadQuote)
onMounted(load)

function defaultPurchaseForm() {
  return { quantity: 1, years: 1, remark: '' }
}

async function load() {
  loading.value = true
  try {
    const [storageRes, seatRes] = await Promise.all([
      api.get('/api/v1/storage/usage'),
      api.get('/api/v1/entitlements/seats/usage')
    ])
    storageUsage.value = storageRes?.data || null
    seatUsage.value = seatRes?.data || null
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openPurchase(resource) {
  purchaseResource.value = resource
  Object.assign(purchaseForm, defaultPurchaseForm())
  quote.value = null
  purchaseVisible.value = true
  loadQuote()
}

async function loadQuote() {
  if (!purchaseVisible.value || purchaseForm.quantity === null || purchaseForm.quantity === undefined || !purchaseForm.years) return
  quoteLoading.value = true
  try {
    const res = await api.post('/api/v1/entitlements/quote', {
      resource: purchaseResource.value,
      quantity: purchaseForm.quantity,
      years: purchaseForm.years
    })
    quote.value = res?.data || null
  } catch {
    quote.value = null
  } finally {
    quoteLoading.value = false
  }
}

async function submitPurchase() {
  if (purchaseForm.quantity === null || purchaseForm.quantity === undefined || !purchaseForm.years) {
    ElMessage.warning(t('storage.message.invalidRequest'))
    return
  }
  submitting.value = true
  try {
    if (purchaseResource.value === 'storage') {
      await api.post('/api/v1/storage/purchase-requests', { capacity_gb: purchaseForm.quantity, years: purchaseForm.years, remark: purchaseForm.remark })
    } else {
      await api.post('/api/v1/entitlements/seats/purchase-requests', { resource: 'seats', quantity: purchaseForm.quantity, years: purchaseForm.years, remark: purchaseForm.remark })
    }
    ElMessage.success(t('storage.message.requestSent'))
    purchaseVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    submitting.value = false
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
    i += 1
  }
  return `${val.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}

function formatDate(value) {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleString()
}

function formatCents(value, currency = 'USD') {
  return `${(Number(value || 0) / 100).toFixed(2)} ${currency || 'USD'}`
}

function formatQuoteFormula(payload, resource) {
  if (!payload) return ''
  const params = {
    quantity: payload.target_quantity || 0,
    unit: resource === 'storage' ? 'GB' : t('entitlements.unit.person'),
    years: payload.target_years || purchaseForm.years || 0,
    months: payload.target_months || Number(payload.target_years || purchaseForm.years || 0) * 12,
    price: payload.unit_price_display || formatCents(payload.unit_price_cents, payload.currency),
    credit: payload.current_credit_display || formatCents(payload.current_credit_cents, payload.currency),
    due: payload.due_display || formatCents(payload.due_cents, payload.currency)
  }
  return resource === 'seats'
    ? t('entitlements.quote.seatFormulaSummary', params)
    : t('entitlements.quote.storageFormulaSummary', params)
}

function statusLabel(status) {
  return t(`storage.status.${status || 'unknown'}`)
}

function statusTagType(status) {
  return { active: 'success', pending: 'warning', expired: 'danger', cancelled: 'info' }[status] || 'info'
}
</script>

<style scoped>
.storage-page {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.storage-hero {
  display: flex;
  justify-content: space-between;
  gap: 18px;
  align-items: flex-end;
  padding: 28px;
  border-radius: 8px;
  background: linear-gradient(135deg, #f8fafc 0%, #eefcf7 48%, #fff7ed 100%);
  border: 1px solid #e5e7eb;
}

.storage-eyebrow,
.storage-panel-label {
  color: #0f766e;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}

.storage-hero h1 {
  margin: 8px 0;
  font-size: 28px;
  line-height: 1.2;
  color: #0f172a;
}

.storage-hero p,
.storage-panel-note {
  margin: 0;
  color: #64748b;
}

.storage-hero-actions,
.storage-panel-head,
.storage-progress-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.storage-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(280px, 0.8fr);
  gap: 18px;
}

.storage-panel {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.05);
  padding: 20px;
}

.storage-used {
  margin-top: 6px;
  font-size: 34px;
  font-weight: 800;
  color: #0f172a;
}

.storage-usage-panel :deep(.el-progress) {
  margin-top: 26px;
}

.storage-progress-meta {
  margin-top: 14px;
  color: #64748b;
  font-size: 13px;
}

.entitlement-policy-note,
.entitlement-form-hint {
  color: #64748b;
  font-size: 13px;
  line-height: 1.5;
}

.entitlement-policy-note {
  margin-top: 10px;
}

.entitlement-form-hint {
  margin-top: 6px;
}

.storage-console {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.storage-console-row,
.quote-box > div {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  border-radius: 8px;
  background: #f8fafc;
}

.storage-console-row span,
.quote-box span {
  color: #64748b;
}

.storage-console-row strong,
.quote-box strong {
  color: #0f172a;
}

.quote-box {
  display: grid;
  gap: 8px;
  margin: 0 0 18px 170px;
}

.quote-box .quote-due {
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
}

.quote-box p {
  margin: 2px 0 0;
  color: #475569;
  font-size: 13px;
}

@media (max-width: 900px) {
  .storage-hero,
  .storage-panel-head,
  .storage-progress-meta {
    align-items: stretch;
    flex-direction: column;
  }

  .storage-grid {
    grid-template-columns: 1fr;
  }

  .quote-box {
    margin-left: 0;
  }
}
</style>
