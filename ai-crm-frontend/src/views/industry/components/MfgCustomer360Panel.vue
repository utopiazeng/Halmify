<template>
  <section class="mfg-customer-panel" :class="{ embedded }">
    <el-empty v-if="!customerId" :description="t('industry.manufacturing.customer360.selectCompanyHint')" />
    <template v-else>
      <div class="mfg-customer-panel__title">
        <div>
          <span>{{ t('industry.manufacturing.customer360.currentCustomer') }}</span>
          <h3>{{ customerName || customerData.name || '-' }}</h3>
        </div>
        <el-button :loading="loading" @click="loadCustomer360">
          {{ t('industry.manufacturing.customer360.reload360') }}
        </el-button>
      </div>

      <el-row :gutter="12" class="mfg-kpis">
        <el-col v-for="item in kpiCards" :key="item.key" :xs="12" :md="6">
          <div class="mfg-kpi">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </el-col>
      </el-row>

      <el-tabs v-model="activeTab" class="mfg-tabs">
        <el-tab-pane :label="t('industry.manufacturing.tabs.profile')" name="profile">
          <el-form label-position="top" class="mfg-form">
            <el-row :gutter="12">
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.profile.requiredCertifications')">
                  <el-input v-model="profileForm.required_certifications" type="textarea" :rows="3" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="12">
                <el-form-item :label="t('industry.manufacturing.profile.specialProcessRequirement')">
                  <el-input v-model="profileForm.special_process_requirement" type="textarea" :rows="3" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="8">
                <el-form-item :label="t('industry.manufacturing.profile.qualityStandard')">
                  <el-input v-model="profileForm.quality_standard" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="8">
                <el-form-item :label="t('industry.manufacturing.profile.paymentTermType')">
                  <el-input v-model="profileForm.payment_term_type" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="8">
                <el-form-item :label="t('industry.manufacturing.profile.moldFeePolicy')">
                  <el-input v-model="profileForm.mold_fee_policy" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="8">
                <el-form-item :label="t('industry.manufacturing.profile.preferredInspectionAgency')">
                  <el-input v-model="profileForm.preferred_inspection_agency" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="8">
                <el-form-item :label="t('industry.manufacturing.profile.packagingRequirement')">
                  <el-input v-model="profileForm.packaging_requirement" />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :md="8">
                <el-form-item :label="t('industry.manufacturing.profile.designatedBrandRequired')">
                  <el-switch v-model="profileForm.designated_brand_required" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-button type="primary" :loading="profileSaving" @click="saveProfile">
              {{ t('industry.manufacturing.customer360.saveProfile') }}
            </el-button>
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="t('industry.manufacturing.tabs.decisionChain')" name="decision">
          <div class="mfg-toolbar">
            <el-button type="primary" @click="openDecisionDialog()">
              {{ t('industry.manufacturing.decision.add') }}
            </el-button>
          </div>
          <el-table :data="decisionItems" border>
            <el-table-column prop="role_type" :label="t('industry.manufacturing.decision.roleType')" min-width="140" />
            <el-table-column prop="contact_id" :label="t('industry.manufacturing.decision.contactId')" width="120" />
            <el-table-column prop="influence_level" :label="t('industry.manufacturing.decision.influenceLevel')" width="130" />
            <el-table-column prop="relationship_status" :label="t('industry.manufacturing.decision.relationshipStatus')" width="150" />
            <el-table-column prop="notes" :label="t('industry.manufacturing.decision.notes')" min-width="180" />
            <el-table-column :label="t('industry.manufacturing.action.operations')" width="180" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="openDecisionDialog(row)">
                  {{ t('industry.manufacturing.action.edit') }}
                </el-button>
                <el-button size="small" type="danger" @click="deleteDecision(row)">
                  {{ t('industry.manufacturing.action.delete') }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane :label="t('industry.manufacturing.tabs.profit')" name="metrics">
          <el-form label-position="top" class="mfg-form">
            <el-row :gutter="12">
              <el-col v-for="field in metricFields" :key="field.key" :xs="24" :md="8">
                <el-form-item :label="field.label">
                  <el-input-number v-model="metricForm[field.key]" :min="field.min" :max="field.max" :step="field.step || 1" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-alert :title="t('industry.manufacturing.customer360.metricsNote')" type="info" show-icon :closable="false" />
            <el-button type="primary" :loading="metricSaving" @click="saveMetricSnapshot">
              {{ t('industry.manufacturing.customer360.saveMetricSnapshot') }}
            </el-button>
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="t('industry.manufacturing.tabs.rfqs')" name="rfqs">
          <mfg-mini-table :items="rfqItems" primary="rfq_no" :empty-text="t('industry.manufacturing.customer360.emptyRfqs')" />
        </el-tab-pane>
        <el-tab-pane :label="t('industry.manufacturing.tabs.samples')" name="samples">
          <mfg-mini-table :items="sampleItems" primary="sample_no" :empty-text="t('industry.manufacturing.customer360.emptySamples')" />
        </el-tab-pane>
        <el-tab-pane :label="t('industry.manufacturing.tabs.assets')" name="assets">
          <mfg-mini-table :items="assetItems" primary="serial_no" :empty-text="t('industry.manufacturing.customer360.emptyAssets')" />
        </el-tab-pane>
      </el-tabs>
    </template>

    <el-dialog v-model="decisionDialogVisible" :title="decisionDialogTitle" width="520px">
      <el-form label-position="top">
        <el-form-item :label="t('industry.manufacturing.decision.contactId')">
          <el-input-number v-model="decisionForm.contact_id" :min="0" />
        </el-form-item>
        <el-form-item :label="t('industry.manufacturing.decision.roleType')">
          <el-select v-model="decisionForm.role_type">
            <el-option v-for="item in decisionRoles" :key="item" :label="decisionRoleLabel(item)" :value="item" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('industry.manufacturing.decision.influenceLevel')">
          <el-select v-model="decisionForm.influence_level">
            <el-option value="high" :label="t('industry.manufacturing.decision.level.high')" />
            <el-option value="medium" :label="t('industry.manufacturing.decision.level.medium')" />
            <el-option value="low" :label="t('industry.manufacturing.decision.level.low')" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('industry.manufacturing.decision.relationshipStatus')">
          <el-select v-model="decisionForm.relationship_status">
            <el-option value="strong" :label="t('industry.manufacturing.decision.relationship.strong')" />
            <el-option value="neutral" :label="t('industry.manufacturing.decision.relationship.neutral')" />
            <el-option value="weak" :label="t('industry.manufacturing.decision.relationship.weak')" />
            <el-option value="unknown" :label="t('industry.manufacturing.decision.relationship.unknown')" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('industry.manufacturing.decision.notes')">
          <el-input v-model="decisionForm.notes" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="decisionDialogVisible = false">{{ t('industry.manufacturing.action.cancel') }}</el-button>
        <el-button type="primary" :loading="decisionSaving" @click="saveDecision">{{ t('industry.manufacturing.action.save') }}</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup>
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import api from '../../../api'
import { t } from '../../../i18n'

const props = defineProps({
  customerId: { type: Number, default: 0 },
  customerName: { type: String, default: '' },
  embedded: { type: Boolean, default: false }
})

const loading = ref(false)
const activeTab = ref('profile')
const customerData = ref({})
const decisionItems = ref([])
const rfqItems = ref([])
const sampleItems = ref([])
const assetItems = ref([])
const metricsData = ref({})
const profileSaving = ref(false)
const decisionSaving = ref(false)
const metricSaving = ref(false)
const decisionDialogVisible = ref(false)
const editingDecisionID = ref(0)
const decisionRoles = ['decision_maker', 'influencer', 'user', 'technical', 'procurement', 'finance']

const profileForm = reactive({
  required_certifications: '',
  quality_standard: '',
  payment_term_type: '',
  designated_brand_required: false,
  mold_fee_policy: '',
  special_process_requirement: '',
  preferred_inspection_agency: '',
  packaging_requirement: '',
  extension: {}
})

const decisionForm = reactive({
  contact_id: 0,
  role_type: 'decision_maker',
  influence_level: 'medium',
  relationship_status: 'unknown',
  notes: ''
})

const metricForm = reactive({
  health_score: 70,
  gross_margin_rate: 0,
  revenue_amount: 0,
  receivable_amount: 0,
  overdue_amount: 0,
  service_value: 0,
  source: { input: 'manual' },
  refresh_status: 'manual'
})

const decisionDialogTitle = computed(() => editingDecisionID.value
  ? t('industry.manufacturing.decision.edit')
  : t('industry.manufacturing.decision.add'))

const kpiCards = computed(() => [
  { key: 'health', label: t('industry.manufacturing.customer360.healthScore'), value: metricsData.value.health_score ?? '-' },
  { key: 'margin', label: t('industry.manufacturing.customer360.grossMarginRate'), value: percent(metricsData.value.gross_margin_rate) },
  { key: 'overdue', label: t('industry.manufacturing.customer360.overdueAmount'), value: money(metricsData.value.overdue_amount) },
  { key: 'service', label: t('industry.manufacturing.customer360.serviceValue'), value: money(metricsData.value.service_value) }
])

const metricFields = computed(() => [
  { key: 'health_score', label: t('industry.manufacturing.customer360.healthScore'), min: 0, max: 100 },
  { key: 'gross_margin_rate', label: t('industry.manufacturing.customer360.grossMarginRate'), min: 0, max: 1, step: 0.01 },
  { key: 'revenue_amount', label: t('industry.manufacturing.customer360.revenueAmount'), min: 0 },
  { key: 'receivable_amount', label: t('industry.manufacturing.customer360.receivableAmount'), min: 0 },
  { key: 'overdue_amount', label: t('industry.manufacturing.customer360.overdueAmount'), min: 0 },
  { key: 'service_value', label: t('industry.manufacturing.customer360.serviceValue'), min: 0 }
])

const MfgMiniTable = {
  props: {
    items: { type: Array, default: () => [] },
    primary: { type: String, default: 'id' },
    emptyText: { type: String, default: '' }
  },
  setup(tableProps) {
    return () => tableProps.items.length
      ? h('div', { class: 'mfg-mini-list' }, tableProps.items.map((item) => h('div', { class: 'mfg-mini-list__item', key: item.id || item[tableProps.primary] }, [
        h('strong', item[tableProps.primary] || item.name || '-'),
        h('span', `${item.status || item.stage || '-'} · ${formatDate(item.updated_at)}`)
      ])))
      : h('div', { class: 'mfg-mini-empty' }, tableProps.emptyText)
  }
}

onMounted(loadCustomer360)
watch(() => props.customerId, () => loadCustomer360())

async function loadCustomer360() {
  if (!props.customerId) return
  loading.value = true
  try {
    const res = await api.get(`/api/v1/mfg/customers/${props.customerId}/360`)
    const payload = res.data || {}
    customerData.value = payload.customer || {}
    applyProfile(payload.profile?.data || {})
    decisionItems.value = payload.decision_chain?.items || []
    rfqItems.value = payload.rfqs?.items || []
    sampleItems.value = payload.samples?.items || []
    assetItems.value = payload.assets?.items || []
    metricsData.value = payload.metrics?.data || {}
    applyMetric(metricsData.value)
  } finally {
    loading.value = false
  }
}

function applyProfile(profile) {
  profileForm.required_certifications = profile.required_certifications || ''
  profileForm.quality_standard = profile.quality_standard || ''
  profileForm.payment_term_type = profile.payment_term_type || ''
  profileForm.designated_brand_required = Boolean(profile.designated_brand_required)
  profileForm.mold_fee_policy = profile.mold_fee_policy || ''
  profileForm.special_process_requirement = profile.special_process_requirement || ''
  profileForm.preferred_inspection_agency = profile.preferred_inspection_agency || ''
  profileForm.packaging_requirement = profile.packaging_requirement || ''
  profileForm.extension = parseJSON(profile.extension_json) || {}
}

function applyMetric(metric) {
  metricForm.health_score = Number(metric.health_score || 70)
  metricForm.gross_margin_rate = Number(metric.gross_margin_rate || 0)
  metricForm.revenue_amount = Number(metric.revenue_amount || 0)
  metricForm.receivable_amount = Number(metric.receivable_amount || 0)
  metricForm.overdue_amount = Number(metric.overdue_amount || 0)
  metricForm.service_value = Number(metric.service_value || 0)
}

async function saveProfile() {
  if (!props.customerId) return
  profileSaving.value = true
  try {
    await api.put(`/api/v1/mfg/customers/${props.customerId}/profile`, { ...profileForm })
    ElMessage.success(t('industry.manufacturing.customer360.saved'))
    await loadCustomer360()
  } finally {
    profileSaving.value = false
  }
}

function openDecisionDialog(row = null) {
  editingDecisionID.value = row?.id || 0
  decisionForm.contact_id = row?.contact_id || 0
  decisionForm.role_type = row?.role_type || 'decision_maker'
  decisionForm.influence_level = row?.influence_level || 'medium'
  decisionForm.relationship_status = row?.relationship_status || 'unknown'
  decisionForm.notes = row?.notes || ''
  decisionDialogVisible.value = true
}

async function saveDecision() {
  if (!props.customerId) return
  decisionSaving.value = true
  try {
    const payload = { ...decisionForm }
    if (editingDecisionID.value) {
      await api.put(`/api/v1/mfg/customers/${props.customerId}/decision-chain/${editingDecisionID.value}`, payload)
    } else {
      await api.post(`/api/v1/mfg/customers/${props.customerId}/decision-chain`, payload)
    }
    decisionDialogVisible.value = false
    ElMessage.success(t('industry.manufacturing.customer360.saved'))
    await loadCustomer360()
  } finally {
    decisionSaving.value = false
  }
}

async function deleteDecision(row) {
  if (!props.customerId || !row?.id) return
  await api.delete(`/api/v1/mfg/customers/${props.customerId}/decision-chain/${row.id}`)
  ElMessage.success(t('industry.manufacturing.customer360.deleted'))
  await loadCustomer360()
}

async function saveMetricSnapshot() {
  if (!props.customerId) return
  metricSaving.value = true
  try {
    await api.post(`/api/v1/mfg/customers/${props.customerId}/metrics`, { ...metricForm })
    ElMessage.success(t('industry.manufacturing.customer360.saved'))
    await loadCustomer360()
  } finally {
    metricSaving.value = false
  }
}

function parseJSON(raw) {
  if (!raw) return {}
  try {
    return typeof raw === 'string' ? JSON.parse(raw) : raw
  } catch {
    return {}
  }
}

function percent(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return '-'
  return `${(number * 100).toFixed(1)}%`
}

function money(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return '-'
  return number.toLocaleString(undefined, { maximumFractionDigits: 0 })
}

function formatDate(value) {
  if (!value) return '-'
  return String(value).slice(0, 10)
}

function decisionRoleLabel(role) {
  const labels = {
    decision_maker: t('industry.manufacturing.decision.role.decision_maker'),
    influencer: t('industry.manufacturing.decision.role.influencer'),
    user: t('industry.manufacturing.decision.role.user'),
    technical: t('industry.manufacturing.decision.role.technical'),
    procurement: t('industry.manufacturing.decision.role.procurement'),
    finance: t('industry.manufacturing.decision.role.finance')
  }
  return labels[role] || role
}
</script>

<style scoped>
.mfg-customer-panel {
  min-width: 0;
}

.mfg-customer-panel.embedded {
  padding: 0;
}

.mfg-customer-panel__title,
.mfg-toolbar {
  margin-bottom: 14px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.mfg-customer-panel__title span {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.mfg-customer-panel__title h3 {
  margin: 4px 0 0;
  font-size: 22px;
  letter-spacing: 0;
}

.mfg-kpis {
  margin-bottom: 14px;
}

.mfg-kpi {
  min-height: 86px;
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.mfg-kpi span {
  display: block;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.mfg-kpi strong {
  display: block;
  margin-top: 10px;
  font-size: 24px;
}

.mfg-tabs {
  margin-top: 4px;
}

.mfg-form {
  max-width: 980px;
}

.mfg-form .el-button {
  margin-top: 12px;
}

.mfg-mini-list {
  display: grid;
  gap: 8px;
}

.mfg-mini-list__item {
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.mfg-mini-list__item strong,
.mfg-mini-list__item span {
  display: block;
}

.mfg-mini-list__item span,
.mfg-mini-empty {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
</style>
