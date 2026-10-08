<template>
  <div class="system-page dashboard-page">
    <section class="dashboard-command" :class="{ fullscreen: bigScreenMode }">
      <div class="dashboard-command-head">
        <div>
          <h2>{{ t('dashboard.title') }}</h2>
          <p>{{ t('dashboard.subtitle') }}</p>
        </div>
        <div class="dashboard-actions">
          <div class="dashboard-year-filter">
            <span>{{ t('dashboard.dataYear') }}</span>
            <el-input-number
              v-model="dashboardYear"
              :min="2020"
              :max="2100"
              controls-position="right"
              size="small"
              @change="handleDashboardYearChange"
            />
          </div>
          <el-button @click="refreshAll">{{ t('common.action.refresh') }}</el-button>
          <el-button type="primary" @click="bigScreenMode = !bigScreenMode">{{ bigScreenMode ? t('dashboard.exitScreenMode') : t('dashboard.screenMode') }}</el-button>
        </div>
      </div>

      <div class="dashboard-notice">
        <Icon icon="ri:volume-up-line" />
        <span>{{ t('dashboard.notice') }}</span>
      </div>

      <div class="dashboard-kpi-grid">
        <MetricCard v-for="item in kpiCards" :key="item.key" v-bind="item" />
      </div>

      <div class="dashboard-chart-grid">
        <div class="dashboard-chart-panel dashboard-chart-panel-wide dashboard-panel-span-8">
          <div class="chart-panel-head">
            <div>
              <strong>{{ t('dashboard.salesTrend') }}</strong>
              <span>{{ t('dashboard.salesTrendDesc') }}</span>
            </div>
            <el-tag size="small">{{ metricBasisLabel }}</el-tag>
          </div>
          <div class="dashboard-area-chart">
            <div class="dashboard-area-plot">
              <div class="dashboard-y-axis">
                <span v-for="tick in trendYAxisTicks" :key="tick.value" :style="{ top: `${tick.top}%` }">{{ formatTrendAxisAmount(tick.value) }}</span>
              </div>
              <svg viewBox="0 0 720 260" preserveAspectRatio="none" :aria-label="t('dashboard.salesTrendChart')">
                <g class="chart-grid-lines">
                  <line v-for="tick in trendYAxisTicks" :key="tick.y" x1="0" x2="720" :y1="tick.y" :y2="tick.y" />
                </g>
                <path class="area-fill" :d="trendAreaPath" />
                <path class="area-line" :d="trendLinePath" />
                <circle v-for="point in trendPoints" :key="`${point.x}-${point.y}`" :cx="point.x" :cy="point.y" r="4" />
              </svg>
            </div>
            <div class="dashboard-axis-labels">
              <span v-for="item in trendRows" :key="item.month">{{ shortMonth(item.month) }}</span>
            </div>
          </div>
        </div>

        <div class="dashboard-chart-panel dashboard-panel-span-4">
          <div class="chart-panel-head">
            <div>
              <strong>{{ t('dashboard.teamTargetRanking') }}</strong>
              <span>{{ t('dashboard.teamTargetRankingDesc') }}</span>
            </div>
            <el-button v-if="canManageDashboard" size="small" plain @click="openTargetDialog">{{ t('dashboard.targetManagement') }}</el-button>
          </div>
          <div class="dashboard-ranking-controls">
            <el-segmented v-model="rankingMode" :options="rankingModeOptions" size="small" />
            <el-date-picker
              v-if="rankingMode === 'month'"
              v-model="rankingMonth"
              type="month"
              value-format="YYYY-MM"
              size="small"
              :clearable="false"
              @change="loadSalesTargetSummary"
            />
          </div>
          <div class="dashboard-leaderboard">
            <div v-for="item in rankingRows" :key="item.owner_id" class="dashboard-leader-row">
              <span>{{ item.name }}</span>
              <div>
                <i :style="{ width: chartWidth(item.actual || item.amount, maxRankingAmount) }"></i>
                <small>{{ formatAmount(item.actual || 0) }} / {{ formatAmount(item.target || 0) }}</small>
              </div>
              <strong>{{ progressText(item.progress) }}</strong>
            </div>
            <div v-if="rankingMode === 'quarter'" class="dashboard-quarter-grid">
              <div v-for="quarter in memberQuarterRows" :key="quarter.period" class="dashboard-quarter-block">
                <strong>{{ quarter.period }}</strong>
                <div v-for="row in quarter.rows" :key="`${quarter.period}-${row.owner_id}`" class="dashboard-quarter-row">
                  <span>{{ row.name }}</span>
                  <i :style="{ width: chartWidth(row.actual || 0, quarterMaxAmount(quarter)) }"></i>
                  <small>{{ progressText(row.progress) }}</small>
                </div>
              </div>
            </div>
          </div>
          <p v-if="unassignedSales.actual" class="crm-inline-help">{{ t('dashboard.unassignedSalesHint', { amount: formatAmount(unassignedSales.actual || 0) }) }}</p>
          <el-empty v-if="!rankingRows.length && rankingMode !== 'quarter'" :description="t('dashboard.emptyTeamRanking')" />
        </div>

        <div class="dashboard-chart-panel target-panel dashboard-panel-span-4">
          <div class="chart-panel-head">
            <div>
              <strong>{{ t('dashboard.annualTargetProgress') }}</strong>
              <span>{{ t('dashboard.metricBasisYear', { basis: metricBasisLabel, year: salesTargets.year || currentYear }) }}</span>
            </div>
            <el-tag size="small">{{ completionRateText }}</el-tag>
          </div>
          <el-progress type="dashboard" :percentage="targetPercent" :format="() => completionRateText" :width="168" />
          <p>{{ formatAmount(cockpit.charts.target.actual || 0) }} / {{ formatAmount(cockpit.charts.target.target || 0) }}</p>
          <p class="crm-inline-help">{{ yoyCaption(salesTargets.yoy) }}</p>
        </div>

        <div class="dashboard-chart-panel dashboard-panel-span-8">
          <div class="chart-panel-head">
            <div>
              <strong>{{ t('dashboard.mySalesTask') }}</strong>
              <span>{{ t('dashboard.mySalesTaskDesc') }}</span>
            </div>
            <el-button type="primary" plain size="small" @click="openPlanDialog()">{{ t('dashboard.editPlan') }}</el-button>
          </div>
          <div class="sales-task-grid">
            <div class="sales-task-stat">
              <span>{{ t('dashboard.annualTarget') }}</span>
              <strong>{{ formatAmount(myProgress.target || myGoal.target || 0) }}</strong>
              <small>{{ t('dashboard.metricBasisOnly', { basis: metricBasisLabel }) }}</small>
            </div>
            <div class="sales-task-stat">
              <span>{{ t('dashboard.currentCompleted') }}</span>
              <strong>{{ formatAmount(myProgress.actual || 0) }}</strong>
              <small>{{ yoyCaption(myProgress.yoy) }}</small>
            </div>
            <div class="sales-task-stat">
              <span>{{ t('dashboard.orderCount') }}</span>
              <strong>{{ myProgress.order_count || 0 }}</strong>
              <small>{{ t('dashboard.orderPaidSummary', { order: formatCompact(myProgress.order_amount || 0), paid: formatCompact(myProgress.paid_amount || 0) }) }}</small>
            </div>
          </div>
          <el-progress :percentage="myProgressPercent" :format="() => progressText(myProgress.progress)" />
          <div class="sales-period-list">
            <div class="sales-period-row">
              <span>{{ t('dashboard.currentQuarter', { period: currentQuarter.period || '-' }) }}</span>
              <strong>{{ formatAmount(currentQuarter.actual || 0) }}</strong>
              <small>{{ yoyCaption(currentQuarter.yoy) }}</small>
            </div>
            <div class="sales-period-row">
              <span>{{ t('dashboard.currentMonth', { period: currentMonth.period || '-' }) }}</span>
              <strong>{{ formatAmount(currentMonth.actual || 0) }}</strong>
              <small>{{ yoyCaption(currentMonth.yoy) }}</small>
            </div>
          </div>
          <div class="sales-plan-list">
            <div v-for="item in myPlanRows" :key="item.id || `${item.period_type}-${item.period}`" class="sales-plan-row">
              <div>
                <strong>{{ periodTypeLabel(item.period_type) }} {{ item.period }}</strong>
                <span>{{ item.action_plan || t('dashboard.emptyActionPlan') }}</span>
                <div class="sales-plan-meta">
                  <small><span>{{ t('dashboard.targetAmount') }}</span><strong>{{ formatAmount(item.target || 0) }}</strong></small>
                  <small><span>{{ t('dashboard.plannedOrderCount') }}</span><strong>{{ item.planned_order_count || 0 }}</strong></small>
                  <small><span>{{ t('dashboard.focusCustomers') }}</span><strong>{{ listText(item.focus_customers) }}</strong></small>
                  <small><span>{{ t('dashboard.focusProducts') }}</span><strong>{{ listText(item.focus_products) }}</strong></small>
                </div>
              </div>
              <el-button size="small" link @click="openPlanDialog(item)">{{ t('common.action.edit') }}</el-button>
            </div>
          </div>
          <el-empty v-if="!myPlanRows.length" :description="t('dashboard.emptyMyPlan')" />
        </div>

      </div>
    </section>

    <section class="dashboard-analytics-grid">
      <div class="system-card dashboard-analytics-panel">
        <div class="chart-panel-head">
          <strong>{{ t('dashboard.countrySalesRanking') }}</strong>
          <el-tag size="small">{{ t('dashboard.yoy') }}</el-tag>
        </div>
        <el-table :data="analytics.country_sales" class="system-table" size="small">
          <el-table-column prop="country" :label="t('dashboard.countryRegion')" min-width="120" />
          <el-table-column :label="t('dashboard.salesAmount')" width="130"><template #default="{ row }">{{ formatAmount(row.amount || 0) }}</template></el-table-column>
          <el-table-column :label="t('dashboard.lastYearSamePeriod')" width="130"><template #default="{ row }">{{ yoyCaption(row.yoy) }}</template></el-table-column>
        </el-table>
        <el-empty v-if="!analytics.country_sales.length" :description="t('dashboard.emptyCountrySales')" />
      </div>

      <div class="system-card dashboard-analytics-panel">
        <div class="chart-panel-head">
          <strong>{{ t('dashboard.customerSalesRanking') }}</strong>
          <el-tag size="small">{{ t('dashboard.orderAggregation') }}</el-tag>
        </div>
        <el-table :data="analytics.customer_sales" class="system-table" size="small">
          <el-table-column prop="customer" :label="t('dashboard.customer')" min-width="150" />
          <el-table-column :label="t('dashboard.salesAmount')" width="130"><template #default="{ row }">{{ formatAmount(row.amount || 0) }}</template></el-table-column>
          <el-table-column :label="t('dashboard.yoy')" width="120"><template #default="{ row }">{{ yoyCaption(row.yoy) }}</template></el-table-column>
        </el-table>
        <el-empty v-if="!analytics.customer_sales.length" :description="t('dashboard.emptyCustomerSales')" />
      </div>

      <div class="system-card dashboard-analytics-panel">
        <div class="chart-panel-head">
          <strong>{{ t('dashboard.hotProductRanking') }}</strong>
          <el-tag size="small">{{ t('dashboard.topCount', { count: analytics.product_sales_top.length }) }}</el-tag>
        </div>
        <div class="sales-analysis-list">
          <div v-for="item in analytics.product_sales_top" :key="item.key" class="sales-analysis-row">
            <div>
              <strong>{{ item.product }}</strong>
              <span>{{ t('dashboard.productSalesCount', { quantity: Number(item.quantity || 0), count: Number(item.count || 0) }) }}</span>
            </div>
            <small>{{ formatAmount(item.amount || 0) }}</small>
          </div>
        </div>
        <el-empty v-if="!analytics.product_sales_top.length" :description="t('dashboard.emptyProductSales')" />
      </div>

      <div class="system-card dashboard-analytics-panel">
        <div class="chart-panel-head">
          <strong>{{ t('dashboard.lowProductRanking') }}</strong>
          <el-tag size="small" type="warning">{{ t('dashboard.lowSales') }}</el-tag>
        </div>
        <div class="sales-analysis-list">
          <div v-for="item in analytics.product_sales_bottom" :key="item.key" class="sales-analysis-row muted">
            <div>
              <strong>{{ item.product }}</strong>
              <span>{{ t('dashboard.productSalesCount', { quantity: Number(item.quantity || 0), count: Number(item.count || 0) }) }}</span>
            </div>
            <small>{{ formatAmount(item.amount || 0) }}</small>
          </div>
        </div>
        <el-empty v-if="!analytics.product_sales_bottom.length" :description="t('dashboard.emptyLowProductSales')" />
      </div>

    </section>

    <el-dialog v-model="targetDialog" :title="t('dashboard.targetManagement')" width="min(980px, 94vw)" destroy-on-close>
      <el-form :model="targetForm" label-width="112px" class="sales-target-form">
        <el-form-item :label="t('dashboard.targetYear')">
          <el-input-number v-model="targetForm.year" :min="2020" :max="2100" class="full-input" />
        </el-form-item>
        <el-form-item :label="t('dashboard.companyAnnualTarget')" class="sales-target-form-wide">
          <div class="sales-target-amount-field">
            <el-input-number v-model="targetForm.target" :min="0" class="full-input" />
            <el-select v-model="targetForm.amount_unit" class="sales-target-unit-select">
              <el-option v-for="item in amountUnitOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </div>
        </el-form-item>
        <el-form-item :label="t('dashboard.completionMetric')">
          <el-radio-group v-model="targetForm.metric_basis">
            <el-radio-button label="payment">{{ t('dashboard.paymentAmount') }}</el-radio-button>
            <el-radio-button label="order_amount">{{ t('dashboard.orderAmount') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div class="sales-allocation-summary">
        <div><span>{{ t('dashboard.allocated') }}</span><strong>{{ formatTargetAmount(allocationStats.allocated) }}</strong></div>
        <div><span>{{ t('dashboard.unallocated') }}</span><strong :class="{ danger: allocationStats.unallocated < 0 }">{{ formatTargetAmount(allocationStats.unallocated) }}</strong></div>
        <div><span>{{ t('dashboard.allocationStatus') }}</span><el-tag :type="allocationStats.unallocated < 0 ? 'danger' : allocationStats.unallocated === 0 ? 'success' : 'warning'">{{ allocationStatusText }}</el-tag></div>
      </div>
      <el-alert
        v-if="!allocationRows.length"
        :title="t('dashboard.noSalesRoleHint')"
        type="warning"
        show-icon
        :closable="false"
        class="sales-allocation-empty"
      />
      <el-table :data="allocationRows" class="system-table sales-allocation-table" size="small">
        <el-table-column prop="name" :label="t('dashboard.salesPerson')" min-width="150" />
        <el-table-column :label="t('dashboard.annualTargetWithUnit', { unit: amountUnitLabel(targetForm.amount_unit) })" min-width="180">
          <template #default="{ row }">
            <el-input-number v-model="row.target" :min="0" controls-position="right" class="full-input" />
          </template>
        </el-table-column>
        <el-table-column :label="t('dashboard.plannedOrderCount')" min-width="160">
          <template #default="{ row }">
            <el-input-number v-model="row.planned_order_count" :min="0" controls-position="right" class="full-input" />
          </template>
        </el-table-column>
        <el-table-column :label="t('dashboard.currentCompleted')" width="130">
          <template #default="{ row }">{{ formatAmount(row.actual || 0) }}</template>
        </el-table-column>
        <el-table-column :label="t('dashboard.completionRate')" width="100">
          <template #default="{ row }">{{ progressText(row.progress) }}</template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button :disabled="targetSaving" @click="targetDialog = false">{{ t('common.action.cancel') }}</el-button>
        <el-button type="primary" :loading="targetSaving" @click="saveTarget">{{ t('dashboard.saveTarget') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="planDialog" :title="t('dashboard.mySalesPlan')" width="min(720px, 94vw)" destroy-on-close>
      <el-form :model="planForm" label-width="112px">
        <el-form-item :label="t('dashboard.planPeriod')">
          <el-segmented v-model="planForm.period_type" :options="planPeriodOptions" @change="syncPlanPeriod" />
        </el-form-item>
        <el-form-item :label="t('dashboard.period')">
          <el-select v-if="planForm.period_type === 'quarter'" v-model="planForm.period" class="full-input">
            <el-option v-for="item in quarterOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
          <el-date-picker v-else-if="planForm.period_type === 'month'" v-model="planForm.period" type="month" value-format="YYYY-MM" class="full-input" />
          <el-input v-else v-model="planForm.period" class="full-input" disabled />
        </el-form-item>
        <el-form-item :label="t('dashboard.targetAmount')">
          <el-input-number v-model="planForm.target" :min="0" class="full-input" />
        </el-form-item>
        <el-form-item :label="t('dashboard.plannedOrderCount')">
          <el-input-number v-model="planForm.planned_order_count" :min="0" class="full-input" />
        </el-form-item>
        <el-form-item :label="t('dashboard.focusCustomers')">
          <el-select v-model="planForm.focus_customers" multiple filterable allow-create default-first-option class="full-input" :placeholder="t('dashboard.focusCustomerPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('dashboard.focusProducts')">
          <el-select v-model="planForm.focus_products" multiple filterable allow-create default-first-option class="full-input" :placeholder="t('dashboard.focusProductPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('dashboard.actionPlan')">
          <el-input v-model="planForm.action_plan" type="textarea" :rows="5" :placeholder="t('dashboard.actionPlanPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button :disabled="planSaving" @click="planDialog = false">{{ t('common.action.cancel') }}</el-button>
        <el-button type="primary" :loading="planSaving" @click="savePlan">{{ t('dashboard.savePlan') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>

import { i18nState, t } from '../../i18n'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import { hasPermission } from '../../permissions'
import MetricCard from '../../components/MetricCard.vue'
import { DEFAULT_COMPANY_SETTINGS, fetchCompanySettings } from '../../utils/company-format'
import { DEFAULT_CURRENCY, formatCurrencyAmount, normalizeCurrencyCode } from '../../utils/currency'

const currentYear = new Date().getFullYear()
const cockpit = reactive({ tasks: [], activities: [], kpi: {}, charts: { funnel: [], trend: [], leaderboard: [], target: {} } })
const salesTargets = reactive({ year: currentYear, metric_basis: 'payment', amount_unit: 'yuan', company_goal: {}, my_goal: {}, my_progress: {}, current_quarter: {}, current_month: {}, my_plans: [], team_rows: [], users: [], allocatable_users: [] })
const analytics = reactive({ country_sales: [], customer_sales: [], product_sales_top: [], product_sales_bottom: [], payment_summary: {} })
const bigScreenMode = ref(false)
const targetDialog = ref(false)
const planDialog = ref(false)
const targetSaving = ref(false)
const planSaving = ref(false)
const rankingMode = ref('year')
const rankingMonth = ref(currentMonthValue())
const dashboardYear = ref(currentYear)
const targetForm = reactive({ year: currentYear, target: 100, metric_basis: 'payment', amount_unit: 'yuan' })
const allocationRows = ref([])
const planForm = reactive({ year: currentYear, period_type: 'month', period: currentMonthValue(), target: 0, planned_order_count: 0, focus_customers: [], focus_products: [], action_plan: '' })
const companySettings = reactive({ ...DEFAULT_COMPANY_SETTINGS })
const amountUnitOptions = computed(() => [
  { label: amountUnitOptionLabel('yuan'), value: 'yuan', multiplier: 1 },
  { label: amountUnitOptionLabel('wan'), value: 'wan', multiplier: 10000 },
  { label: amountUnitOptionLabel('yi'), value: 'yi', multiplier: 100000000 }
])
const planPeriodOptions = computed(() => [
  { label: t('dashboard.periodYear'), value: 'year' },
  { label: t('dashboard.periodQuarter'), value: 'quarter' },
  { label: t('dashboard.periodMonth'), value: 'month' }
])
const rankingModeOptions = computed(() => [
  { label: t('dashboard.periodYear'), value: 'year' },
  { label: t('dashboard.periodMonth'), value: 'month' },
  { label: t('dashboard.periodQuarter'), value: 'quarter' }
])

const kpiCards = computed(() => [
  { key: 'target', title: t('dashboard.annualTarget'), value: formatAmount(salesTargets.company_target || cockpit.kpi.company_target || 0), caption: t('dashboard.companyTargetYear', { year: salesTargets.year || currentYear }), icon: 'ri:flag-2-line', tone: 'blue' },
  { key: 'actual', title: t('dashboard.currentCompleted'), value: formatAmount(salesTargets.current_amount || cockpit.kpi.target_actual || 0), caption: yoyCaption(salesTargets.yoy), icon: 'ri:coins-line', tone: 'green' },
  { key: 'progress', title: t('dashboard.completionRate'), value: completionRateText.value, caption: t('dashboard.metricBasisOnly', { basis: metricBasisLabel.value }), icon: 'ri:speed-up-line', tone: 'cyan' },
  { key: 'order_amount', title: t('dashboard.orderAmount'), value: formatAmount(salesTargets.order_amount || cockpit.kpi.order_amount || 0), caption: yoyCaption(salesTargets.order_amount_yoy), icon: 'ri:shopping-cart-line', tone: 'orange' },
  { key: 'paid_amount', title: t('dashboard.paymentAmount'), value: formatAmount(salesTargets.paid_amount || cockpit.kpi.paid_amount || 0), caption: yoyCaption(salesTargets.paid_amount_yoy), icon: 'ri:bank-card-line', tone: 'purple' },
  { key: 'order_count', title: t('dashboard.orderCount'), value: salesTargets.order_count || cockpit.kpi.order_count || 0, caption: yoyCaption(salesTargets.order_count_yoy), icon: 'ri:file-list-3-line', tone: 'slate' },
  { key: 'yoy', title: t('dashboard.lastYearGrowthRate'), value: yoyRateValue(salesTargets.yoy), caption: yoyCaption(salesTargets.yoy), icon: 'ri:line-chart-line', tone: 'green' }
])
const maxTrendAmount = computed(() => Math.max(1, ...((cockpit.charts.trend || []).map((item) => Number(item.amount || 0)))))
const maxRankingAmount = computed(() => Math.max(1, ...((rankingRows.value || []).map((item) => Number(item.actual || item.amount || 0)))))
const targetPercent = computed(() => progressPercent(cockpit.charts.target?.progress || salesTargets.target_progress || 0))
const completionRateText = computed(() => progressText(cockpit.charts.target?.progress || salesTargets.target_progress || 0))
const metricBasisLabel = computed(() => metricBasisText(salesTargets.metric_basis || cockpit.kpi.metric_basis || targetForm.metric_basis))
const dashboardCurrency = computed(() => normalizeCurrencyCode(
  salesTargets.currency || salesTargets.company_goal?.currency || cockpit.kpi.currency || companySettings.currency,
  DEFAULT_CURRENCY
))
const myGoal = computed(() => salesTargets.my_goal || {})
const myProgress = computed(() => salesTargets.my_progress || {})
const currentQuarter = computed(() => salesTargets.current_quarter || {})
const currentMonth = computed(() => salesTargets.current_month || {})
const myProgressPercent = computed(() => progressPercent(myProgress.value.progress || 0))
const myPlanRows = computed(() => (salesTargets.my_plans || []).filter((item) => item.period_type !== 'year'))
const rankingRows = computed(() => {
  if (rankingMode.value === 'month') return salesTargets.member_month_rows || []
  if (rankingMode.value === 'quarter') return []
  return salesTargets.team_rows?.length ? salesTargets.team_rows : (cockpit.charts.leaderboard || [])
})
const memberQuarterRows = computed(() => salesTargets.member_quarter_rows || [])
const unassignedSales = computed(() => salesTargets.unassigned_sales || {})
const quarterOptions = computed(() => [1, 2, 3, 4].map((quarter) => ({ label: t('dashboard.quarterOption', { year: salesTargets.year || currentYear, quarter }), value: `${salesTargets.year || currentYear}-Q${quarter}` })))
const allocationStats = computed(() => {
  const unit = normalizeAmountUnit(targetForm.amount_unit)
  const allocated = allocationRows.value.reduce((sum, item) => sum + fromDisplayAmount(item.target, unit), 0)
  const target = fromDisplayAmount(targetForm.target, unit)
  return { allocated, unallocated: target - allocated }
})
const allocationStatusText = computed(() => {
  if (allocationStats.value.unallocated < 0) return t('dashboard.overallocated')
  if (allocationStats.value.unallocated === 0) return t('dashboard.fullyAllocated')
  return t('dashboard.stillUnallocated')
})
const canManageDashboard = computed(() => hasPermission('dashboards:update') || hasPermission('dashboards:manage') || hasPermission('teams:update') || hasPermission('teams:manage'))
const trendRows = computed(() => {
  const rows = cockpit.charts.trend || []
  return rows.length ? rows : [{ month: '01', amount: 0 }, { month: '02', amount: 0 }, { month: '03', amount: 0 }, { month: '04', amount: 0 }]
})
const trendChart = {
  top: 40,
  bottom: 220,
  height: 180
}
const trendYAxisTicks = computed(() => {
  const max = maxTrendAmount.value || 1
  return [4, 3, 2, 1, 0].map((step) => {
    const ratio = step / 4
    return {
      value: max * ratio,
      y: Math.round(trendChart.bottom - ratio * trendChart.height),
      top: ((trendChart.bottom - ratio * trendChart.height) / 260) * 100
    }
  })
})
const trendPoints = computed(() => {
  const rows = trendRows.value
  const max = maxTrendAmount.value || 1
  const step = rows.length > 1 ? 720 / (rows.length - 1) : 720
  return rows.map((item, index) => ({
    x: Math.round(index * step),
    y: Math.round(trendChart.bottom - (Number(item.amount || 0) / max) * trendChart.height)
  }))
})
const trendLinePath = computed(() => pointsToPath(trendPoints.value))
const trendAreaPath = computed(() => {
  const points = trendPoints.value
  if (!points.length) return ''
  return `${pointsToPath(points)} L ${points[points.length - 1].x} 238 L ${points[0].x} 238 Z`
})

watch(() => targetForm.amount_unit, (nextUnit, previousUnit) => {
  if (!targetDialog.value || !previousUnit || normalizeAmountUnit(nextUnit) === normalizeAmountUnit(previousUnit)) return
  convertTargetUnit(previousUnit, nextUnit)
})

onMounted(refreshAll)

async function refreshAll() {
  await Promise.all([loadCockpit(), loadAnalytics(), loadCompanySettings()])
}

function selectedDashboardYear() {
  const year = Number(dashboardYear.value || currentYear)
  return year >= 2020 && year <= 2100 ? year : currentYear
}

async function handleDashboardYearChange() {
  dashboardYear.value = selectedDashboardYear()
  targetForm.year = dashboardYear.value
  rankingMonth.value = normalizeRankingMonthForYear(rankingMonth.value, dashboardYear.value)
  await Promise.all([loadCockpit(), loadAnalytics()])
}

async function loadCompanySettings() {
  Object.assign(companySettings, await fetchCompanySettings())
}

async function loadCockpit() {
  const res = await api.get('/api/v1/prd/cockpit', { params: { year: selectedDashboardYear() } })
  cockpit.tasks = res.data?.tasks || []
  cockpit.activities = res.data?.activities || []
  cockpit.kpi = res.data?.kpi || {}
  cockpit.charts = res.data?.charts || { funnel: [], trend: [], leaderboard: [], target: {} }
  applySalesTargets(res.data?.sales_targets || {})
}

async function loadAnalytics() {
  const res = await api.get('/api/v1/prd/analytics', { params: { year: selectedDashboardYear() } })
  const data = res.data || {}
  analytics.country_sales = data.country_sales || []
  analytics.customer_sales = data.customer_sales || []
  analytics.product_sales_top = data.product_sales_top || []
  analytics.product_sales_bottom = data.product_sales_bottom || []
  analytics.payment_summary = data.payment_summary || {}
}

async function loadSalesTargetSummary() {
  const year = Number(targetForm.year || dashboardYear.value || salesTargets.year || currentYear)
  const res = await api.get('/api/v1/prd/flows/sales-targets/summary', { params: { year, month: rankingMonth.value || currentMonthValue() } })
  applySalesTargets(res.data || {})
}

async function openTargetDialog() {
  targetForm.year = selectedDashboardYear()
  await loadSalesTargetSummary()
  targetForm.year = Number(salesTargets.year || currentYear)
  targetForm.amount_unit = normalizeAmountUnit(salesTargets.amount_unit || targetForm.amount_unit)
  targetForm.target = toDisplayAmount(Number(salesTargets.company_target || cockpit.charts.target?.target || 0), targetForm.amount_unit)
  targetForm.metric_basis = salesTargets.metric_basis || targetForm.metric_basis || 'payment'
  allocationRows.value = buildAllocationRows()
  targetDialog.value = true
}

async function saveTarget() {
  if (targetSaving.value) return
  targetSaving.value = true
  try {
    const year = Number(targetForm.year || currentYear)
    const amountUnit = normalizeAmountUnit(targetForm.amount_unit)
    const companyRes = await api.post('/api/v1/prd/flows/sales-targets/company', {
      year,
      target: fromDisplayAmount(targetForm.target, amountUnit),
      metric_basis: targetForm.metric_basis || 'payment',
      amount_unit: amountUnit
    })
    const allocateRes = await api.post('/api/v1/prd/flows/sales-targets/allocate', {
      year,
      allocations: allocationRows.value
        .filter((item) => Number(item.owner_id || 0) > 0)
        .map((item) => ({ owner_id: Number(item.owner_id), target: fromDisplayAmount(item.target, amountUnit), planned_order_count: Number(item.planned_order_count || 0) }))
    })
    applySalesTargets(allocateRes.data?.summary || companyRes.data?.summary || {})
    syncCockpitTargetFromSalesTargets()
    allocationRows.value = buildAllocationRows()
    targetDialog.value = false
    ElMessage.success(t('dashboard.targetSaved'))
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || t('dashboard.targetSaveFailed'))
  } finally {
    targetSaving.value = false
  }
}

function openPlanDialog(row = null) {
  const year = Number(dashboardYear.value || salesTargets.year || currentYear)
  planForm.year = year
  if (row) {
    planForm.period_type = row.period_type || 'month'
    planForm.period = row.period || defaultPlanPeriod(row.period_type || 'month', year)
    planForm.target = Number(row.target || 0)
    planForm.planned_order_count = Number(row.planned_order_count || 0)
    planForm.focus_customers = arrayText(row.focus_customers)
    planForm.focus_products = arrayText(row.focus_products)
    planForm.action_plan = row.action_plan || ''
  } else {
    planForm.period_type = 'month'
    planForm.period = normalizeRankingMonthForYear('', year)
    planForm.target = 0
    planForm.planned_order_count = 0
    planForm.focus_customers = []
    planForm.focus_products = []
    planForm.action_plan = ''
  }
  syncPlanPeriod()
  planDialog.value = true
}

async function savePlan() {
  if (planSaving.value) return
  planSaving.value = true
  try {
    const res = await api.post('/api/v1/prd/flows/sales-targets/my-plan', {
      year: Number(planForm.year || currentYear),
      period_type: planForm.period_type,
      period: planForm.period,
      target: Number(planForm.target || 0),
      planned_order_count: Number(planForm.planned_order_count || 0),
      focus_customers: planForm.focus_customers || [],
      focus_products: planForm.focus_products || [],
      action_plan: planForm.action_plan || ''
    })
    applySalesTargets(res.data?.summary || {})
    syncCockpitTargetFromSalesTargets()
    planDialog.value = false
    ElMessage.success(t('dashboard.planSaved'))
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || t('dashboard.planSaveFailed'))
  } finally {
    planSaving.value = false
  }
}

function applySalesTargets(data) {
  const next = data || {}
  Object.assign(salesTargets, {
    year: next.year || currentYear,
    metric_basis: next.metric_basis || 'payment',
    metric_basis_label: next.metric_basis_label || metricBasisText(next.metric_basis || 'payment'),
    amount_unit: normalizeAmountUnit(next.amount_unit || next.company_goal?.amount_unit || salesTargets.amount_unit || targetForm.amount_unit),
    amount_unit_label: amountUnitLabel(next.amount_unit || next.company_goal?.amount_unit || salesTargets.amount_unit || targetForm.amount_unit),
    currency: normalizeCurrencyCode(next.currency || next.company_goal?.currency || companySettings.currency, dashboardCurrency.value),
    company_goal: next.company_goal || {},
    company_target: Number(next.company_target || next.company_goal?.target || 0),
    current_amount: Number(next.current_amount || 0),
    target_progress: Number(next.target_progress || 0),
    allocated_target: Number(next.allocated_target || 0),
    unallocated_target: Number(next.unallocated_target || 0),
    overallocated: Boolean(next.overallocated),
    order_amount: Number(next.order_amount || 0),
    paid_amount: Number(next.paid_amount || 0),
    order_count: Number(next.order_count || 0),
    yoy: next.yoy || {},
    order_amount_yoy: next.order_amount_yoy || {},
    paid_amount_yoy: next.paid_amount_yoy || {},
    order_count_yoy: next.order_count_yoy || {},
    team_rows: next.team_rows || [],
    selected_month: next.selected_month || rankingMonth.value || currentMonthValue(),
    member_month_rows: next.member_month_rows || [],
    member_quarter_rows: next.member_quarter_rows || [],
    unassigned_sales: next.unassigned_sales || {},
    users: next.users || [],
    allocatable_users: next.allocatable_users || [],
    my_goal: next.my_goal || {},
    my_plans: next.my_plans || [],
    my_progress: next.my_progress || {},
    current_quarter: next.current_quarter || {},
    current_month: next.current_month || {}
  })
  dashboardYear.value = Number(salesTargets.year || dashboardYear.value || currentYear)
  targetForm.year = Number(salesTargets.year || currentYear)
  targetForm.amount_unit = normalizeAmountUnit(salesTargets.amount_unit || targetForm.amount_unit)
  targetForm.target = toDisplayAmount(Number(salesTargets.company_target || 0), targetForm.amount_unit)
  targetForm.metric_basis = salesTargets.metric_basis || targetForm.metric_basis || 'payment'
  rankingMonth.value = salesTargets.selected_month || normalizeRankingMonthForYear(rankingMonth.value, dashboardYear.value)
}

function syncCockpitTargetFromSalesTargets() {
  cockpit.charts.target = {
    ...(cockpit.charts.target || {}),
    target: Number(salesTargets.company_target || 0),
    actual: Number(salesTargets.current_amount || 0),
    progress: Number(salesTargets.target_progress || 0)
  }
  cockpit.kpi.company_target = Number(salesTargets.company_target || 0)
  cockpit.kpi.target_actual = Number(salesTargets.current_amount || 0)
  cockpit.kpi.metric_basis = salesTargets.metric_basis || 'payment'
}

function buildAllocationRows() {
  const byOwner = new Map((salesTargets.team_rows || []).map((row) => [Number(row.owner_id), row]))
  const usersByID = new Map()
  ;[
    ...(salesTargets.allocatable_users || [])
  ].forEach((user) => {
    const ownerID = Number(user.id || user.owner_id || 0)
    if (ownerID > 0 && !usersByID.has(ownerID)) usersByID.set(ownerID, user)
  })
  return Array.from(usersByID.values()).map((user) => {
    const ownerID = Number(user.id || user.owner_id || 0)
    const teamRow = byOwner.get(ownerID) || {}
    return {
      owner_id: ownerID,
      name: user.name || teamRow.name || t('dashboard.salesPersonFallback', { id: ownerID }),
      target: toDisplayAmount(Number(user.target ?? teamRow.target ?? 0), targetForm.amount_unit),
      planned_order_count: Number(user.planned_order_count || teamRow.planned_order_count || 0),
      actual: Number(teamRow.actual || 0),
      progress: Number(teamRow.progress || 0)
    }
  })
}

function convertTargetUnit(previousUnit, nextUnit) {
  const currentUnit = normalizeAmountUnit(previousUnit)
  const normalizedNextUnit = normalizeAmountUnit(nextUnit)
  const targetAmount = fromDisplayAmount(targetForm.target, currentUnit)
  targetForm.amount_unit = normalizedNextUnit
  targetForm.target = toDisplayAmount(targetAmount, normalizedNextUnit)
  allocationRows.value = allocationRows.value.map((row) => ({
    ...row,
    target: toDisplayAmount(fromDisplayAmount(row.target, currentUnit), normalizedNextUnit)
  }))
}

function syncPlanPeriod() {
  planForm.period = normalizePlanPeriod(planForm.period_type, planForm.period, Number(planForm.year || salesTargets.year || currentYear))
}

function chartWidth(value, max) { return `${Math.max(8, Math.round((Number(value || 0) / Number(max || 1)) * 100))}%` }
function quarterMaxAmount(quarter) { return Math.max(1, ...((quarter?.rows || []).map((item) => Number(item.actual || 0)))) }
function pointsToPath(points) {
  if (!points.length) return ''
  return points.map((point, index) => `${index === 0 ? 'M' : 'L'} ${point.x} ${point.y}`).join(' ')
}
function progressPercent(value) {
  const ratio = Number(value || 0)
  if (ratio <= 0) return 0
  return Math.min(100, Math.max(0.01, Number((ratio * 100).toFixed(2))))
}
function progressText(value) {
  const ratio = Number(value || 0)
  if (ratio <= 0) return '0%'
  const percent = ratio * 100
  if (percent < 0.01) return `${percent.toFixed(4)}%`
  if (percent < 0.1) return `${percent.toFixed(3)}%`
  if (percent < 1) return `${percent.toFixed(2)}%`
  return `${(Math.round(percent * 10) / 10).toLocaleString()}%`
}
function formatAmount(amount, currency = dashboardCurrency.value, unit = salesTargets.amount_unit || targetForm.amount_unit) {
  const normalizedUnit = normalizeAmountUnit(unit)
  if (shouldUseConfiguredAmountUnit(normalizedUnit)) {
    return `${formatCurrencyNumber(toDisplayAmount(amount, normalizedUnit), currency)}${amountUnitSuffix(normalizedUnit)}`
  }
  return normalizedUnit === 'yuan' ? formatCurrencyNumber(amount, currency) : formatCurrencyCompact(amount, currency)
}
function formatTargetAmount(amount) {
  return formatAmount(amount, dashboardCurrency.value, targetForm.amount_unit)
}
function formatCompact(amount) {
  return formatCurrencyCompact(amount)
}
function formatTrendAxisAmount(amount) {
  return formatCurrencyCompact(amount)
}
function shortMonth(value) { return String(value || '').slice(5) || '-' }
function metricBasisText(value) {
  return value === 'order_amount' ? t('dashboard.orderAmount') : t('dashboard.paymentAmount')
}
function normalizeAmountUnit(value) {
  return ['yuan', 'wan', 'yi'].includes(value) ? value : 'yuan'
}
function amountUnitLabel(value) {
  return amountUnitOptions.value.find((item) => item.value === normalizeAmountUnit(value))?.label || amountUnitOptionLabel('yuan')
}
function amountUnitOptionLabel(value) {
  const unit = normalizeAmountUnit(value)
  return `${dashboardCurrency.value} ${amountUnitScaleLabel(unit)}`
}
function amountUnitScaleLabel(value) {
  switch (normalizeAmountUnit(value)) {
    case 'wan':
      return t('dashboard.currencyUnitTenThousand')
    case 'yi':
      return t('dashboard.currencyUnitHundredMillion')
    default:
      return t('dashboard.currencyUnitBase')
  }
}
function amountUnitSuffix(value) {
  const unit = normalizeAmountUnit(value)
  return unit === 'yuan' ? '' : amountUnitScaleLabel(unit)
}
function amountUnitMultiplier(value) {
  return amountUnitOptions.value.find((item) => item.value === normalizeAmountUnit(value))?.multiplier || 1
}
function shouldUseConfiguredAmountUnit(unit) {
  return ['wan', 'yi'].includes(normalizeAmountUnit(unit)) && ['zh-CN', 'zh-TW', 'ja-JP', 'ko-KR'].includes(i18nState.locale)
}
function formatCurrencyNumber(amount, currency = dashboardCurrency.value, options = {}) {
  const value = trimNumber(amount)
  return formatCurrencyAmount(value, currency, i18nState.locale, {
    currencyDisplay: 'symbol',
    maximumFractionDigits: Number.isInteger(value) ? 0 : 2,
    ...options
  })
}
function formatCurrencyCompact(amount, currency = dashboardCurrency.value) {
  const value = trimNumber(amount)
  return formatCurrencyNumber(value, currency, {
    notation: 'compact',
    compactDisplay: 'short',
    maximumFractionDigits: Math.abs(value) >= 1000 ? 1 : 0
  })
}
function toDisplayAmount(value, unit = targetForm.amount_unit) {
  const amount = Number(value || 0)
  const multiplier = amountUnitMultiplier(unit)
  if (!multiplier) return amount
  return trimNumber(amount / multiplier)
}
function fromDisplayAmount(value, unit = targetForm.amount_unit) {
  return trimNumber(Number(value || 0) * amountUnitMultiplier(unit))
}
function trimNumber(value) {
  const number = Number(value || 0)
  if (!Number.isFinite(number)) return 0
  return Number(number.toFixed(6))
}
function formatPlainNumber(value) {
  return trimNumber(value).toLocaleString(undefined, { maximumFractionDigits: 6 })
}
function yoyCaption(value) {
  const yoy = value || {}
  if (!yoy.has_last_year && Number(yoy.last_year_amount || 0) <= 0) return t('dashboard.noLastYearData')
  const rate = Number(yoy.yoy_rate || 0)
  return t('dashboard.samePeriodRate', { rate: `${rate >= 0 ? '+' : ''}${(rate * 100).toFixed(1)}%` })
}
function yoyRateValue(value) {
  const yoy = value || {}
  if (!yoy.has_last_year && Number(yoy.last_year_amount || 0) <= 0) return t('dashboard.none')
  const rate = Number(yoy.yoy_rate || 0)
  return `${rate >= 0 ? '+' : ''}${(rate * 100).toFixed(1)}%`
}
function periodTypeLabel(value) {
  return ({ year: t('dashboard.periodYear'), quarter: t('dashboard.periodQuarter'), month: t('dashboard.periodMonth') }[value] || t('dashboard.plan'))
}
function arrayText(value) {
  if (Array.isArray(value)) return value.filter(Boolean).map((item) => String(item))
  if (!value) return []
  return String(value).split(/[,，]/).map((item) => item.trim()).filter(Boolean)
}
function listText(value) {
  const items = arrayText(value)
  return items.length ? items.join(t('dashboard.listSeparator')) : t('dashboard.none')
}
function currentMonthValue() {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}
function normalizeRankingMonthForYear(value, year = currentYear) {
  const text = String(value || '')
  const month = /^\d{4}-(\d{2})$/.test(text) ? text.slice(5, 7) : String(new Date().getMonth() + 1).padStart(2, '0')
  return `${year}-${month}`
}
function defaultPlanPeriod(periodType, year = currentYear) {
  if (periodType === 'year') return String(year)
  if (periodType === 'quarter') {
    const quarter = Math.floor(new Date().getMonth() / 3) + 1
    return `${year}-Q${quarter}`
  }
  return normalizeRankingMonthForYear('', year)
}
function normalizePlanPeriod(periodType, period, year = currentYear) {
  if (periodType === 'year') return String(year)
  if (periodType === 'quarter') {
    const text = String(period || '')
    if (/^\d{4}-Q[1-4]$/.test(text)) return text
    return defaultPlanPeriod('quarter', year)
  }
  const text = String(period || '')
  if (/^\d{4}-\d{2}$/.test(text)) return text
  return defaultPlanPeriod('month', year)
}
</script>
