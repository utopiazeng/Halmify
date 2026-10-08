<template>
  <div class="system-page insights-page">
    <section class="crm-help-strip"><span>{{ t('domain.auto.9193abe728') }}</span></section>

    <section class="insight-kpi-grid">
      <MetricCard v-for="item in kpiCards" :key="item.key" v-bind="item" />
    </section>

    <section class="insight-main-grid">
      <div class="system-card">
        <div class="list-card-search">
          <el-form :model="filters" label-width="72px" class="system-search-form">
            <el-form-item :label="t('domain.auto.cc1b21e800')"><el-input v-model="filters.keyword" :placeholder="t('domain.auto.d36ad2bdf2')" clearable /></el-form-item>
            <el-form-item :label="t('domain.auto.e4e46c7235')"><el-select v-model="filters.object_type" clearable :placeholder="t('domain.auto.fa6968d9f0')"><el-option v-for="item in typeOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
            <div class="system-search-actions">
              <el-button @click="resetFilters">{{ t('domain.auto.3d81345303') }}</el-button>
              <el-button type="primary" @click="loadPage">{{ t('domain.auto.711363c424') }}</el-button>
            </div>
          </el-form>
        </div>
        <div class="system-toolbar">
          <div class="toolbar-left">
            <el-button v-if="canManage" type="primary" plain @click="generateHotspots">{{ t('domain.auto.86f62c98e6') }}</el-button>
            <el-button @click="loadPage">{{ t('domain.auto.38108eaa1d') }}</el-button>
          </div>
          <el-tag>{{ pagination.total }} {{ t('domain.auto.302b14f0af') }}</el-tag>
        </div>
        <el-table :data="rows" class="system-table" row-key="id" v-loading="loading" @row-click="selectRecord">
          <el-table-column prop="name" :label="t('domain.auto.0a1b27bb9c')" min-width="260"><template #default="{ row }"><div class="record-title"><strong>{{ row.name }}</strong><span>{{ insightRowMeta(row) }}</span></div></template></el-table-column>
          <el-table-column :label="t('domain.auto.2f58566ce0')" width="120"><template #default="{ row }"><el-progress :percentage="Number(row.score || insightPayload(row).risk_score || 0)" /></template></el-table-column>
          <el-table-column :label="t('domain.auto.62e951a692')" width="110"><template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
          <el-table-column :label="t('domain.auto.c5134eb19c')" min-width="260"><template #default="{ row }">{{ insightSummary(row) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.d9db02d07a')" width="150"><template #default="{ row }">{{ formatTime(row.updated_at) }}</template></el-table-column>
        </el-table>
        <div class="system-pagination"><span>{{ t('domain.auto.3b6ef811b8') }} {{ pagination.total }} {{ t('domain.auto.bce2ef6151') }}</span><el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.size" layout="prev, pager, next, sizes" :total="pagination.total" @current-change="loadPage" @size-change="loadPage" /></div>
      </div>

      <aside class="system-card insight-side-card">
        <div class="chart-panel-head"><strong>{{ t('domain.auto.bd6f09e188') }}</strong><el-tag size="small">{{ t('domain.auto.e4636e7f16') }}</el-tag></div>
        <div v-if="selectedRecord && selectedRecord.object_type !== 'deal_review'" class="lead-summary">
          <div><span>{{ t('domain.auto.0a1b27bb9c') }}</span><strong>{{ selectedRecord.name }}</strong></div>
          <div><span>{{ t('domain.auto.e4e46c7235') }}</span><strong>{{ typeLabel(selectedRecord.object_type) }}</strong></div>
          <div><span>{{ t('domain.auto.1ff9c3d001') }}</span><strong>{{ selectedPayload.reason || selectedPayload.source || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.c5134eb19c') }}</span><strong>{{ selectedPayload.suggestion || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.0041e018c3') }}</span><strong>{{ formatAmount(selectedPayload.amount || selectedRecord.amount, selectedRecord.currency) }}</strong></div>
        </div>
        <article v-else-if="selectedRecord" class="deal-review-reader">
          <div class="deal-review-reader-head">
            <el-tag :type="selectedPayload.review_type === 'lost' ? 'danger' : 'success'">{{ reviewTypeLabel(selectedPayload.review_type) }}</el-tag>
            <el-tag :type="statusType(selectedRecord.status)">{{ statusLabel(selectedRecord.status) }}</el-tag>
          </div>
          <h2>{{ selectedRecord.name }}</h2>
          <p>{{ selectedPayload.summary || '-' }}</p>
          <div class="deal-review-facts">
            <div><span>{{ t('insights.review.fact.opportunity') }}</span><strong>{{ selectedPayload.opportunity_name || '-' }}</strong></div>
            <div><span>{{ t('insights.review.fact.customer') }}</span><strong>{{ selectedPayload.company || '-' }}</strong></div>
            <div><span>{{ t('insights.review.fact.contact') }}</span><strong>{{ selectedPayload.contact_name || '-' }}</strong></div>
            <div><span>{{ t('insights.review.fact.amount') }}</span><strong>{{ formatAmount(selectedPayload.amount || selectedRecord.amount, selectedRecord.currency) }}</strong></div>
            <div><span>{{ t('opportunity.review.field.industry') }}</span><strong>{{ selectedPayload.industry || '-' }}</strong></div>
            <div><span>{{ t('opportunity.review.field.competitor') }}</span><strong>{{ selectedPayload.competitor || '-' }}</strong></div>
            <div><span>{{ t('insights.review.fact.product') }}</span><strong>{{ reviewProducts(selectedPayload) }}</strong></div>
            <div><span>{{ t('insights.review.fact.publishedAt') }}</span><strong>{{ formatTime(selectedPayload.published_at) }}</strong></div>
          </div>
          <section>
            <h3>{{ t('opportunity.review.field.background') }}</h3>
            <p>{{ selectedPayload.background || '-' }}</p>
          </section>
          <section>
            <h3>{{ t('opportunity.review.field.customerNeeds') }}</h3>
            <p>{{ selectedPayload.customer_needs || '-' }}</p>
          </section>
          <section>
            <h3>{{ t('opportunity.review.field.decisionProcess') }}</h3>
            <p>{{ selectedPayload.decision_process || '-' }}</p>
          </section>
          <section>
            <h3>{{ selectedPayload.review_type === 'lost' ? t('insights.review.section.whyLost') : t('insights.review.section.whyWon') }}</h3>
            <p>{{ selectedPayload.review_type === 'lost' ? (selectedPayload.loss_reasons || '-') : (selectedPayload.win_reasons || '-') }}</p>
          </section>
          <section>
            <h3>{{ t('opportunity.review.field.obstacles') }}</h3>
            <p>{{ selectedPayload.obstacles || '-' }}</p>
          </section>
          <section>
            <h3>{{ t('opportunity.review.field.lessons') }}</h3>
            <p>{{ selectedPayload.lessons || '-' }}</p>
          </section>
          <section>
            <h3>{{ t('opportunity.review.reusableExperience') }}</h3>
            <p>{{ selectedPayload.reusable_playbook || '-' }}</p>
          </section>
          <section>
            <h3>{{ t('insights.review.section.notApplicableBoundary') }}</h3>
            <p>{{ selectedPayload.avoid_next_time || '-' }}</p>
          </section>
          <section>
            <h3>{{ t('insights.review.section.processReview') }}</h3>
            <p>{{ selectedPayload.key_actions || '-' }}</p>
          </section>
          <div class="deal-review-tags">
            <el-tag v-for="item in reviewTags(selectedPayload)" :key="item" size="small">{{ item }}</el-tag>
            <el-tag v-for="item in reviewKeywords(selectedPayload)" :key="`keyword-${item}`" size="small" type="info">{{ item }}</el-tag>
          </div>
        </article>
        <el-empty v-else :description="t('domain.auto.cf0fcd0025')" />
      </aside>
    </section>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import { canManageDomain } from '../../permissions'
import MetricCard from '../../components/MetricCard.vue'
import { formatCurrencyAmount, normalizeCurrencyCode } from '../../utils/currency'
import { defaultBusinessCurrency } from '../../utils/company-format'

const rows = ref([])
const loading = ref(false)
const selectedRecord = ref(null)
const filters = reactive({ keyword: '', object_type: '' })
const pagination = reactive({ page: 1, size: 20, total: 0 })
const canManage = computed(() => canManageDomain('insights'))
const selectedPayload = computed(() => insightPayload(selectedRecord.value))
const typeOptions = [
  { value: 'hot_customer', label: t('domain.auto.13c841925a') },
  { value: 'hot_product', label: t('domain.auto.88f2d818ff') },
  { value: 'deal_risk', label: t('domain.auto.15b95dc5fa') },
  { value: 'faq_cluster', label: t('domain.auto.f4d4876567') },
  { value: 'deal_review', label: t('opportunity.review.title') },
  { value: 'touch_attribution', label: t('domain.auto.075376d630') }
]
const kpiCards = computed(() => [
  { key: 'total', title: t('domain.auto.bd771f8b8c'), value: rows.value.length, icon: 'ri:lightbulb-flash-line', tone: 'blue', help: t('domain.auto.bd771f8b8c') },
  { key: 'watching', title: t('domain.auto.e82f893586'), value: rows.value.filter((item) => item.status === 'watching' || item.status === 'open').length, icon: 'ri:eye-line', tone: 'purple', help: t('domain.auto.16ea0775ad') },
  { key: 'risk', title: t('domain.auto.eed09e83d6'), value: rows.value.filter((item) => item.object_type === 'deal_risk').length, icon: 'ri:alert-line', tone: 'orange', help: t('domain.auto.15b95dc5fa') },
  { key: 'score', title: t('domain.auto.48b372c351'), value: Math.max(0, ...rows.value.map((item) => Number(item.score || insightPayload(item).risk_score || 0))), icon: 'ri:speed-up-line', tone: 'green', help: t('domain.auto.48b372c351') }
])

onMounted(loadPage)
async function loadPage() { loading.value = true; try { const res = await api.get('/api/v1/prd/insights/objects', { params: cleanParams({ ...filters, page: pagination.page, size: pagination.size }) }); rows.value = res.data?.items || []; pagination.total = res.data?.total || 0 } finally { loading.value = false } }
async function generateHotspots() { await api.post('/api/v1/prd/flows/insights/hotspots'); ElMessage.success(t('domain.auto.f0ea79ebd0')); await loadPage() }
function selectRecord(row) { selectedRecord.value = row }
function resetFilters() { Object.assign(filters, { keyword: '', object_type: '' }); pagination.page = 1; loadPage() }
function insightPayload(row) { try { return typeof row?.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row?.payload_json || {}) } catch { return {} } }
function cleanParams(params) { return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== '' && value !== undefined && value !== null)) }
function typeLabel(value) { return typeOptions.find((item) => item.value === value)?.label || value || '-' }
function statusLabel(value) { return ({ open: t('domain.auto.59a9eb4e65'), watching: t('domain.auto.e82f893586'), resolved: t('domain.auto.219bde30cd'), active: t('domain.auto.d4e9ca3dd4'), draft: t('opportunity.review.status.draft'), pending_review: t('opportunity.review.status.pendingReview'), published: t('opportunity.review.status.published'), archived: t('opportunity.review.status.archived') }[value] || value || '-') }
function statusType(value) { return ({ resolved: 'success', active: 'success', published: 'success', pending_review: 'warning', watching: 'warning', archived: 'danger', draft: 'info' }[value] || 'warning') }
function reviewTypeLabel(value) { return ({ won: t('opportunity.review.type.won'), lost: t('opportunity.review.type.lost') }[value] || t('opportunity.review.title')) }
function insightRowMeta(row) {
  const p = insightPayload(row)
  if (row.object_type === 'deal_review') return `${reviewTypeLabel(p.review_type)} · ${p.company || '-'} · ${formatAmount(p.amount || row.amount, row.currency)}`
  return `${typeLabel(row.object_type)} · ${p.source || t('domain.auto.0dd7df1d68')}`
}
function insightSummary(row) {
  const p = insightPayload(row)
  if (row.object_type === 'deal_review') return p.summary || p.reusable_playbook || p.win_reasons || p.loss_reasons || '-'
  return p.suggestion || '-'
}
function reviewTags(payload) {
  return Array.isArray(payload.tags) ? payload.tags : []
}
function reviewKeywords(payload) {
  return Array.isArray(payload.keywords) ? payload.keywords : []
}
function reviewProducts(payload) {
  if (Array.isArray(payload.products) && payload.products.length) return payload.products.join('、')
  return '-'
}
function formatTime(value) { if (!value) return '-'; const date = new Date(value); return Number.isNaN(date.getTime()) ? String(value).slice(0, 19) : `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}` }
function formatAmount(amount, currency = defaultBusinessCurrency()) { return formatCurrencyAmount(amount, normalizeCurrencyCode(currency, defaultBusinessCurrency())) }
</script>
