<template>
  <div class="system-page platform-ai-page">
    <section class="ai-wallet-row">
      <div class="system-card ai-wallet-card">
        <span>{{ t('platformAi.tenants') }}</span>
        <strong>{{ walletRows.length }}</strong>
      </div>
      <div class="system-card ai-wallet-card">
        <span>{{ t('platformAi.totalBalance') }}</span>
        <strong>USD {{ totalBalance }}</strong>
      </div>
      <div class="system-card ai-wallet-card">
        <span>{{ t('platformAi.totalSpent') }}</span>
        <strong>USD {{ totalSpent }}</strong>
      </div>
    </section>

    <section class="system-card">
      <el-tabs v-model="tab">
        <el-tab-pane :label="t('platformAi.tab.wallets')" name="wallets">
          <el-table :data="walletRows" v-loading="loading">
            <el-table-column prop="tenant_id" :label="t('platform.tenants.tenantId')" width="110" />
            <el-table-column prop="tenant_name" :label="t('platformAi.tenant')" min-width="160" />
            <el-table-column :label="t('aiAnalysis.wallet.balance')" width="150"><template #default="{ row }">{{ row.currency }} {{ row.available_amount }}</template></el-table-column>
            <el-table-column :label="t('aiAnalysis.wallet.monthly')" width="210"><template #default="{ row }">{{ formatNumber(row.monthly_token_used) }} / {{ row.monthly_token_limit ? formatNumber(row.monthly_token_limit) : t('aiAnalysis.unlimited') }}</template></el-table-column>
            <el-table-column :label="t('aiAnalysis.wallet.spent')" width="150"><template #default="{ row }">{{ row.currency }} {{ row.total_spent_amount }}</template></el-table-column>
            <el-table-column prop="enabled" :label="t('common.table.status')" width="110"><template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? t('aiAnalysis.status.enabled') : t('aiAnalysis.status.disabled') }}</el-tag></template></el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane :label="t('platformAi.tab.applications')" name="applications">
          <el-table :data="applications" v-loading="loading">
            <el-table-column prop="id" :label="t('aiAnalysis.report.id')" width="90" />
            <el-table-column prop="tenant_name" :label="t('platformAi.tenant')" min-width="160" />
            <el-table-column prop="package_code" :label="t('aiAnalysis.package.title')" width="140" />
            <el-table-column prop="status" :label="t('common.table.status')" width="120"><template #default="{ row }"><el-tag>{{ row.status }}</el-tag></template></el-table-column>
            <el-table-column prop="note" :label="t('aiAnalysis.apply.note')" min-width="220" />
            <el-table-column :label="t('common.table.operation')" width="190">
              <template #default="{ row }">
                <el-button size="small" type="primary" :disabled="row.status !== 'pending'" @click="approve(row)">{{ t('platformAi.approve') }}</el-button>
                <el-button size="small" type="danger" :disabled="row.status !== 'pending'" @click="reject(row)">{{ t('platformAi.reject') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane :label="t('platformAi.tab.ledger')" name="ledger">
          <el-table :data="ledgerRows" v-loading="loading">
            <el-table-column prop="id" :label="t('aiAnalysis.report.id')" width="90" />
            <el-table-column prop="tenant_name" :label="t('platformAi.tenant')" min-width="150" />
            <el-table-column prop="entry_type" :label="t('platformAi.entryType')" width="120" />
            <el-table-column prop="direction" :label="t('platformAi.direction')" width="100" />
            <el-table-column :label="t('aiAnalysis.usage.cost')" width="130"><template #default="{ row }">{{ row.amount }}</template></el-table-column>
            <el-table-column :label="t('aiAnalysis.usage.tokens')" width="150"><template #default="{ row }">{{ formatNumber(row.total_tokens) }}</template></el-table-column>
            <el-table-column prop="trace_id" :label="t('platformAi.trace')" min-width="220" />
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import api from '../../api'
import { t } from '../../i18n'

const tab = ref('wallets')
const loading = ref(false)
const walletRows = ref([])
const applications = ref([])
const ledgerRows = ref([])

const totalBalance = computed(() => sumAmount(walletRows.value, 'available_amount'))
const totalSpent = computed(() => sumAmount(walletRows.value, 'total_spent_amount'))

onMounted(loadCurrent)
watch(tab, loadCurrent)

async function loadCurrent() {
  loading.value = true
  try {
    if (tab.value === 'wallets') {
      const res = await api.get('/api/v1/platform/ai-analysis/usage/summary')
      walletRows.value = res.data?.items || []
    } else if (tab.value === 'applications') {
      const res = await api.get('/api/v1/platform/ai-analysis/applications')
      applications.value = res.data?.items || []
    } else {
      const res = await api.get('/api/v1/platform/ai-analysis/ledger')
      ledgerRows.value = res.data?.items || []
    }
  } finally {
    loading.value = false
  }
}

async function approve(row) {
  await api.post(`/api/v1/platform/ai-analysis/applications/${row.id}/approve`, {})
  ElMessage.success(t('common.message.opSuccess'))
  await loadCurrent()
}

async function reject(row) {
  const reason = await ElMessageBox.prompt(t('platformAi.rejectReason'), t('platformAi.reject'), { confirmButtonText: t('common.action.submit'), cancelButtonText: t('common.action.cancel') })
  await api.post(`/api/v1/platform/ai-analysis/applications/${row.id}/reject`, { reason: reason.value || '' })
  ElMessage.success(t('common.message.opSuccess'))
  await loadCurrent()
}

function formatNumber(value) {
  return Number(value || 0).toLocaleString()
}

function sumAmount(rows, key) {
  const total = rows.reduce((sum, row) => sum + Number(row[key] || 0), 0)
  return total.toFixed(6)
}
</script>
