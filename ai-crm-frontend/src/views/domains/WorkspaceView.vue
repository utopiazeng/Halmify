<template>
  <div class="system-page workspace-page">
    <section class="workspace-role-strip">
      <button v-for="item in roleTabs" :key="item.key" class="workspace-role-tab" :class="{ active: activeTab === item.key }" @click="activeTab = item.key">
        <strong>{{ item.title }}</strong>
        <span>{{ item.desc }}</span>
      </button>
    </section>

    <section class="crm-help-strip">
      <span>{{ activeHelp }}</span>
    </section>

    <section class="workspace-kpi-grid">
      <MetricCard v-for="item in kpiCards" :key="item.key" v-bind="item" />
    </section>

    <section class="workspace-main-grid">
      <div class="system-card workspace-panel">
        <div class="chart-panel-head">
          <strong>{{ activePanelTitle }}</strong>
          <el-button size="small" @click="refreshAll">{{ t('domain.auto.38108eaa1d') }}</el-button>
        </div>

        <div v-if="activeTab === 'sales'" class="workspace-focus-list">
          <button v-for="item in salesFocus" :key="`${item.domain}-${item.id}`" class="workspace-focus-row" @click="goDomain(item.domain)">
            <div>
              <strong>{{ item.name }}</strong>
              <span>{{ domainLabel(item.domain) }} · {{ statusLabel(item.status) }} · {{ formatAmount(item.amount, item.currency) }}</span>
            </div>
            <el-tag>{{ item.action }}</el-tag>
          </button>
          <el-empty v-if="!salesFocus.length" :description="t('domain.auto.328a4c17d9')" />
        </div>

        <div v-else-if="activeTab === 'service'" class="workspace-focus-list">
          <button v-for="item in serviceFocus" :key="item.id" class="workspace-focus-row" :class="{ urgent: isTicketOverdue(item) }" @click="goDomain('tickets')">
            <div>
              <strong>{{ item.name }}</strong>
              <span>{{ ticketPriorityLabel(payload(item).priority) }} · {{ ticketSlaText(item) }}</span>
            </div>
            <el-tag :type="isTicketOverdue(item) ? 'danger' : 'warning'">{{ statusLabel(item.status) }}</el-tag>
          </button>
          <el-empty v-if="!serviceFocus.length" :description="t('domain.auto.6b328205cd')" />
        </div>

        <div v-else class="workspace-manager-grid">
          <div class="manager-chart">
            <div class="chart-panel-head"><strong>{{ t('domain.auto.8da56ada07') }}</strong><el-tag size="small">{{ t('domain.auto.d2053849e8') }}</el-tag></div>
            <div class="dashboard-funnel-chart">
              <div v-for="item in cockpit.charts.funnel" :key="item.stage" class="dashboard-funnel-row light">
                <span>{{ stageLabel(item.stage) }}</span>
                <div><i :style="{ width: chartWidth(item.count, maxFunnelCount) }"></i></div>
                <strong>{{ item.count }}</strong>
              </div>
            </div>
          </div>
          <div class="manager-chart">
            <div class="chart-panel-head"><strong>SLA</strong><el-tag size="small">{{ t('domain.auto.63aaa4cee4') }}</el-tag></div>
            <div class="dashboard-service-grid">
              <div><span>{{ t('domain.auto.6c7444b500') }}</span><strong>{{ analytics.service_quality.total || 0 }}</strong></div>
              <div><span>{{ t('domain.auto.8c71083008') }}</span><strong>{{ analytics.service_quality.overdue || 0 }}</strong></div>
              <div><span>{{ t('domain.auto.3cde56d71e') }}</span><strong>{{ analytics.service_quality.closed || 0 }}</strong></div>
              <div><span>{{ t('domain.auto.e80ecbfeb6') }}</span><strong>{{ scoreText(analytics.service_quality.avg_csat) }}</strong></div>
            </div>
          </div>
        </div>
      </div>

      <aside class="system-card workspace-panel">
        <div class="chart-panel-head">
          <strong>{{ t('domain.auto.846209a810') }}</strong>
          <el-tag>{{ cockpit.tasks.length }}</el-tag>
        </div>
        <div class="dashboard-task-list">
          <div v-for="item in cockpit.tasks" :key="item.id" class="dashboard-task-item">
            <div>
              <strong>{{ item.name }}</strong>
              <span>{{ item.object_type }} · {{ statusLabel(item.status) }}</span>
            </div>
            <el-button v-if="canManageWorkspace" size="small" link @click="completeTask(item)">{{ t('domain.auto.33246f6a5e') }}</el-button>
          </div>
        </div>
        <el-empty v-if="!cockpit.tasks.length" :description="t('domain.auto.033984ae97')" />
        <el-divider>{{ t('domain.auto.62cc51cc74') }}</el-divider>
        <div class="dashboard-task-list">
          <div v-for="(item, idx) in nextActions" :key="`${item.domain}-${item.object_id}-${idx}`" class="dashboard-task-item">
            <div>
              <strong>{{ item.title }}</strong>
              <span>{{ domainLabel(item.domain) }} · {{ item.message }}</span>
            </div>
            <div>
              <el-tag :type="priorityType(item.priority)" size="small">{{ priorityLabel(item.priority) }}</el-tag>
              <el-button size="small" link @click="goDomain(item.domain)">{{ t('domain.auto.a0a1f34396') }}</el-button>
            </div>
          </div>
        </div>
        <el-empty v-if="!nextActions.length" :description="t('domain.auto.af11ed1f66')" />
      </aside>
    </section>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '../../api'
import { canManageDomain } from '../../permissions'
import MetricCard from '../../components/MetricCard.vue'
import { formatCurrencyAmount, normalizeCurrencyCode } from '../../utils/currency'
import { defaultBusinessCurrency } from '../../utils/company-format'

const router = useRouter()
const activeTab = ref('sales')
const cockpit = reactive({ tasks: [], activities: [], kpi: {}, charts: { funnel: [], trend: [], leaderboard: [], target: {} } })
const analytics = reactive({ service_quality: {} })
const leads = ref([])
const opportunities = ref([])
const tickets = ref([])
const nextActions = ref([])
const canManageWorkspace = computed(() => canManageDomain('workspace'))
const roleTabs = [
  { key: 'sales', title: t('domain.auto.080b609dec'), desc: t('domain.auto.64ed6cb83e') },
  { key: 'service', title: t('domain.auto.445f20c714'), desc: t('domain.auto.2c23b980e4') },
  { key: 'manager', title: t('domain.auto.24abe501b2'), desc: t('domain.auto.7222cdf0ad') }
]
const activeHelp = computed(() => ({
  sales: t('domain.auto.e7ef29dcee'),
  service: t('domain.auto.3d3a7ed1bf'),
  manager: t('domain.auto.49b5fd4a00')
}[activeTab.value]))
const activePanelTitle = computed(() => ({ sales: t('domain.auto.5088b60f84'), service: t('domain.auto.e726f90083'), manager: t('domain.auto.c2c9679809') }[activeTab.value]))
const kpiCards = computed(() => [
  { key: 'leads', title: t('domain.auto.8de0b8fb1d'), value: cockpit.kpi.leads || 0, icon: 'ri:contacts-line', tone: 'blue', help: t('domain.auto.e2d8dbabd2') },
  { key: 'opps', title: t('domain.auto.c4beee8569'), value: cockpit.kpi.opportunities || 0, icon: 'ri:briefcase-4-line', tone: 'purple', help: t('domain.auto.260617751a') },
  { key: 'orders', title: t('domain.auto.81b3798beb'), value: cockpit.kpi.orders || 0, icon: 'ri:shopping-cart-line', tone: 'green', help: t('domain.auto.8f3dc61681') },
  { key: 'tickets', title: t('domain.auto.85c230ed15'), value: cockpit.kpi.tickets || 0, icon: 'ri:customer-service-2-line', tone: 'orange', help: t('domain.auto.17e2f8d878') }
])
const salesFocus = computed(() => [
  ...leads.value.slice(0, 4).map((item) => ({ ...item, domain: 'leads', action: t('domain.auto.92d7235c82') })),
  ...opportunities.value.slice(0, 6).map((item) => ({ ...item, domain: 'opportunities', action: item.status === 'proposal' ? t('domain.auto.7311cad9da') : t('domain.auto.74984d48e8') }))
].slice(0, 8))
const serviceFocus = computed(() => tickets.value.filter((item) => !['closed', 'resolved'].includes(item.status)).slice(0, 8))
const maxFunnelCount = computed(() => Math.max(1, ...((cockpit.charts.funnel || []).map((item) => Number(item.count || 0)))))

onMounted(refreshAll)

async function refreshAll() {
  await Promise.all([loadCockpit(), loadAnalytics(), loadLeads(), loadOpportunities(), loadTickets(), loadNextActions()])
}
async function loadCockpit() {
  const res = await api.get('/api/v1/prd/cockpit')
  cockpit.tasks = res.data?.tasks || []
  cockpit.activities = res.data?.activities || []
  cockpit.kpi = res.data?.kpi || {}
  cockpit.charts = res.data?.charts || { funnel: [], trend: [], leaderboard: [], target: {} }
}
async function loadAnalytics() {
  const res = await api.get('/api/v1/prd/analytics')
  analytics.service_quality = res.data?.service_quality || {}
}
async function loadLeads() {
  const res = await api.get('/api/v1/prd/leads/objects', { params: { object_type: 'lead', exclude_status: 'invalid,converted', page: 1, size: 8 } })
  leads.value = res.data?.items || []
}
async function loadOpportunities() {
  const res = await api.get('/api/v1/prd/opportunities/objects', { params: { object_type: 'opportunity', exclude_status: 'won,lost', page: 1, size: 8 } })
  opportunities.value = res.data?.items || []
}
async function loadTickets() {
  const res = await api.get('/api/v1/prd/tickets/objects', { params: { object_type: 'ticket', page: 1, size: 20 } })
  tickets.value = res.data?.items || []
}
async function loadNextActions() {
  try {
    const res = await api.get('/api/v1/prd/next-actions', { params: { limit: 8 } })
    nextActions.value = res.data?.items || []
  } catch {
    nextActions.value = []
  }
}
async function completeTask(item) {
  await api.post(`/api/v1/prd/flows/workspace/${item.id}/done`)
  ElMessage.success(t('domain.auto.a18980c255'))
  await loadCockpit()
}
function goDomain(domain) { router.push(`/${domain}`) }
function payload(row) { try { return typeof row?.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row?.payload_json || {}) } catch { return {} } }
function isTicketOverdue(row) { const due = payload(row).sla_due_at || payload(row).sla_resolve_due_at; return due ? new Date(due).getTime() < Date.now() : false }
function ticketSlaText(row) { const due = payload(row).sla_due_at || payload(row).sla_resolve_due_at; return due ? `${isTicketOverdue(row) ? t('domain.auto.411ebfdad7') : t('domain.auto.7f090d0a3e')} ${String(due).replace('T', ' ').slice(0, 16)}` : t('domain.auto.c264b8f920') }
function ticketPriorityLabel(value) { return ({ urgent: t('domain.auto.27429f465b'), high: t('domain.auto.73f5b58830'), normal: t('domain.auto.7cda072d45'), low: t('domain.auto.2217b22f64') }[value] || t('domain.auto.7cda072d45')) }
function domainLabel(value) { return ({ leads: t('domain.auto.8de0b8fb1d'), opportunities: t('domain.auto.c4beee8569'), orders: t('domain.auto.81b3798beb'), tickets: t('domain.auto.85c230ed15') }[value] || value) }
function statusLabel(value) { return ({ new: t('domain.auto.0cda8d1c71'), open: t('domain.auto.59a9eb4e65'), following: t('domain.auto.a5b46b1a52'), initial_contact: t('domain.auto.45b80954f8'), need_confirmed: t('domain.auto.eb5a8bd262'), proposal: t('domain.auto.df33ce33bb'), negotiation: t('domain.auto.83b022f008'), assigned: t('domain.auto.732b2f5de1'), processing: t('domain.auto.fcb979ef0b'), waiting_customer: t('domain.auto.69b352c10e'), resolved: t('domain.auto.5e045cba10'), closed: t('domain.auto.f628761bf5'), done: t('domain.auto.33246f6a5e') }[value] || value || '-') }
function stageLabel(value) { return ({ initial_contact: t('domain.auto.45b80954f8'), need_confirmed: t('domain.auto.eb5a8bd262'), proposal: t('domain.auto.df33ce33bb'), negotiation: t('domain.auto.83b022f008'), won: t('domain.auto.e65d23ddf1'), lost: t('domain.auto.5375e3d681'), on_hold: t('domain.auto.f5f7939ed7') }[value] || value || '-') }
function priorityType(value) { return ({ high: 'danger', medium: 'warning', low: 'info' }[value] || 'info') }
function priorityLabel(value) { return ({ high: t('domain.auto.b096b3f5ac'), medium: t('domain.auto.0869071c92'), low: t('domain.auto.b9ee259b7f') }[value] || t('domain.auto.7cda072d45')) }
function chartWidth(value, max) { return `${Math.max(8, Math.round((Number(value || 0) / Number(max || 1)) * 100))}%` }
function scoreText(value) { const score = Number(value || 0); return score ? score.toFixed(1) : '-' }
function formatAmount(amount, currency = defaultBusinessCurrency()) { return formatCurrencyAmount(amount, normalizeCurrencyCode(currency, defaultBusinessCurrency())) }
</script>
