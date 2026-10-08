<template>
  <div class="domain-page">
    <div class="domain-toolbar">
      <el-form :inline="true" class="toolbar-form">
        <el-form-item>
          <el-button type="primary" @click="openCreate">{{ t('common.action.create') }}</el-button>
          <el-button @click="load">{{ t('common.action.refresh') }}</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-table :data="rows" v-loading="loading">
      <el-table-column prop="tenant_id" :label="t('platform.tenants.tenantId')" width="120" />
      <el-table-column prop="status" :label="t('common.table.status')" width="120" />
      <el-table-column prop="plan_code" :label="t('platform.tenants.plan')" width="120" />
      <el-table-column prop="max_users" :label="t('platform.tenants.maxUsers')" width="120" />
      <el-table-column prop="current_users" :label="t('platform.tenants.currentUsers')" width="120" />
      <el-table-column prop="valid_to" :label="t('platform.tenants.validTo')" width="180">
        <template #default="{ row }">{{ formatDate(row.valid_to) }}</template>
      </el-table-column>
      <el-table-column prop="days_to_expire" :label="t('platform.tenants.daysToExpire')" width="120" />
      <el-table-column prop="admin_account_masked" :label="t('platform.tenants.adminMasked')" width="180" />
      <el-table-column prop="storage_quota" :label="t('platform.tenants.storageQuota')" width="160">
        <template #default="{ row }">{{ formatBytes(row.storage_quota) }}</template>
      </el-table-column>
      <el-table-column prop="storage_used_mb" :label="t('platform.tenants.storageUsedMB')" width="150">
        <template #default="{ row }">{{ formatMB(row.storage_used_mb) }}</template>
      </el-table-column>
      <el-table-column :label="t('platform.tenants.storageStatus')" width="180">
        <template #default="{ row }">
          <el-tag :type="row.storage_upload_locked ? 'danger' : row.storage_expiring_soon ? 'warning' : 'success'">
            {{ row.storage_upload_locked ? t('storage.status.uploadPaused') : row.storage_expiring_soon ? t('storage.status.expiringSoon') : t('storage.status.uploadAvailable') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="expansion_storage_gb" :label="t('platform.tenants.storageExpansion')" width="150">
        <template #default="{ row }">{{ row.expansion_storage_gb || 0 }} GB</template>
      </el-table-column>
      <el-table-column prop="storage_pending_count" :label="t('platform.tenants.storagePending')" width="140" />
      <el-table-column :label="t('platform.tenants.industryPackage')" width="170">
        <template #default="{ row }">
          <el-tag :type="row.industry_package_enabled ? 'success' : 'info'">
            {{ row.industry_package_enabled ? industryPackageLabel(row.industry_package_code, row.industry_package_name) : t('platform.tenants.industryPackageNone') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('domain.auto.9b90db1a3f')" width="120">
        <template #default="{ row }"><el-tag :type="row.ai_enabled ? 'success' : 'info'">{{ row.ai_enabled ? t('domain.auto.320d2e28d1') : t('domain.auto.18d76e1386') }}</el-tag></template>
      </el-table-column>
      <el-table-column :label="t('domain.auto.05c7222cda')" width="130">
        <template #default="{ row }">${{ row.ai_balance_amount || '0.000000' }} USD</template>
      </el-table-column>
      <el-table-column :label="t('domain.auto.65149760c2')" width="170">
        <template #default="{ row }">{{ formatNumber(row.ai_monthly_token_used) }} / {{ row.ai_monthly_token_limit ? formatNumber(row.ai_monthly_token_limit) : t('domain.auto.09c4fc252c') }}</template>
      </el-table-column>
      <el-table-column :label="t('common.table.operation')" min-width="220">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="openEdit(row)">{{ t('common.action.edit') }}</el-button>
          <el-button size="small" @click="openStorage(row)">{{ t('platform.tenants.storageManage') }}</el-button>
          <el-button size="small" @click="resetAdminPassword(row)">{{ t('platform.tenants.resetAdminPassword') }}</el-button>
          <el-button size="small" type="danger" @click="deleteTenant(row)">{{ t('common.action.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createVisible" :title="t('platform.tenants.createTenant')" width="620px">
      <el-form :model="form" label-width="120px">
        <el-form-item :label="t('platform.tenants.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('platform.tenants.adminUsername')"><el-input v-model="form.admin_username" /></el-form-item>
        <el-form-item :label="t('platform.tenants.adminEmail')"><el-input v-model="form.admin_email" /></el-form-item>
        <el-form-item :label="t('platform.tenants.plan')">
          <el-select v-model="form.quota.plan_code" style="width: 100%">
            <el-option v-for="plan in plans" :key="plan.plan_code" :value="plan.plan_code" :label="plan.name || plan.plan_code" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('platform.tenants.industryPackage')">
          <el-select v-model="form.industry_package_code" style="width: 100%" @change="onCreateIndustryPackageChange">
            <el-option value="" :label="t('platform.tenants.industryPackageNone')" />
            <el-option v-for="item in industryPackages" :key="item.code" :value="item.code" :label="industryPackageLabel(item.code, item.name)" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('platform.tenants.maxUsers')"><el-input-number v-model="form.quota.max_users" :min="1" /></el-form-item>
        <el-form-item :label="t('platform.tenants.storageQuota')"><el-input :model-value="t('storage.text.baseIncluded')" disabled /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">{{ t('common.action.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitCreate">{{ t('common.action.submit') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editVisible" :title="t('platform.tenants.editTenant')" width="680px">
      <el-form :model="editForm" label-width="150px">
        <el-form-item :label="t('common.table.status')">
          <el-select v-model="editForm.status" style="width: 100%">
            <el-option value="active" :label="t('common.status.enabled')" />
            <el-option value="disabled" :label="t('common.status.disabled')" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('platform.tenants.plan')">
          <el-select v-model="editForm.plan_code" style="width: 100%">
            <el-option v-for="plan in plans" :key="plan.plan_code" :value="plan.plan_code" :label="plan.name || plan.plan_code" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('platform.tenants.industryPackage')">
          <el-select v-model="editForm.industry_package_code" style="width: 100%" @change="onEditIndustryPackageChange">
            <el-option value="" :label="t('platform.tenants.industryPackageNone')" />
            <el-option v-for="item in industryPackages" :key="item.code" :value="item.code" :label="industryPackageLabel(item.code, item.name)" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('platform.tenants.maxUsers')"><el-input-number v-model="editForm.max_users" :min="1" /></el-form-item>
        <el-form-item :label="t('platform.tenants.storageQuota')"><el-input :model-value="t('storage.text.adjustInStorageDialog')" disabled /></el-form-item>
        <el-form-item :label="t('platform.tenants.validTo')"><el-date-picker v-model="editForm.valid_to" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" style="width: 100%" /></el-form-item>
        <el-divider>{{ t('domain.auto.9b90db1a3f') }}</el-divider>
        <el-form-item :label="t('domain.auto.2d4d2eed87')">
          <el-switch v-model="editForm.ai_enabled" />
        </el-form-item>
        <el-form-item :label="t('domain.auto.c406b54fb1')">
          <el-input-number v-model="editForm.ai_balance_amount" :min="0" :precision="2" :step="100" style="width: 100%" />
        </el-form-item>
        <el-form-item :label="t('domain.auto.f76bf4f2d6')">
          <el-input-number v-model="editForm.ai_monthly_token_limit" :min="0" :step="100000" style="width: 100%" />
        </el-form-item>
        <el-form-item :label="t('platform.tenants.modulesPreview')">
          <el-input :model-value="planModulesPreview(editForm.plan_code)" disabled type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">{{ t('common.action.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitEdit">{{ t('common.action.submit') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="passwordVisible" :title="t('platform.tenants.resetAdminPassword')" width="520px">
      <el-form :model="passwordForm" label-width="120px">
        <el-form-item :label="t('platform.tenants.adminMasked')">
          <el-input :model-value="passwordForm.admin_account" disabled />
        </el-form-item>
        <el-form-item :label="t('app.newPassword')">
          <div class="password-row">
            <el-input
              v-model="passwordForm.new_password"
              type="password"
              show-password
              :placeholder="t('app.newPasswordPlaceholder')"
              autocomplete="new-password"
            />
            <el-button @click="generateRandomPassword">{{ t('common.action.generatePassword') }}</el-button>
          </div>
        </el-form-item>
        <el-form-item :label="t('app.confirmPassword')">
          <el-input
            v-model="passwordForm.confirm_password"
            type="password"
            show-password
            :placeholder="t('app.confirmPasswordPlaceholder')"
            autocomplete="new-password"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="passwordVisible = false">{{ t('common.action.cancel') }}</el-button>
        <el-button type="primary" :loading="savingPassword" @click="submitResetAdminPassword">{{ t('common.action.submit') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="storageVisible" :title="t('platform.entitlements.dialogTitle')" width="980px">
      <el-tabs v-model="entitlementResource" @tab-change="loadStorage">
        <el-tab-pane :label="t('entitlements.resource.storage')" name="storage" />
        <el-tab-pane :label="t('entitlements.resource.seats')" name="seats" />
      </el-tabs>
      <div v-if="storageUsage" class="platform-storage-summary">
        <div class="platform-storage-card">
          <span>{{ entitlementResource === 'storage' ? t('storage.metric.used') : t('entitlements.seats.currentUsers') }}</span>
          <strong>{{ entitlementResource === 'storage' ? formatBytes(storageUsage.used_bytes) : `${storageUsage.current_users || 0} ${t('entitlements.unit.person')}` }}</strong>
        </div>
        <div class="platform-storage-card">
          <span>{{ entitlementResource === 'storage' ? t('storage.metric.total') : t('entitlements.seats.maxUsers') }}</span>
          <strong>{{ entitlementResource === 'storage' ? formatBytes(storageUsage.total_bytes) : `${storageUsage.max_users || 0} ${t('entitlements.unit.person')}` }}</strong>
        </div>
        <div class="platform-storage-card">
          <span>{{ entitlementResource === 'storage' ? t('storage.metric.expansion') : t('entitlements.seats.active') }}</span>
          <strong>{{ entitlementResource === 'storage' ? `${storageUsage.expansion_gb || 0} GB` : `${storageUsage.seat_count || 0} ${t('entitlements.unit.person')}` }}</strong>
        </div>
        <div class="platform-storage-card">
          <span>{{ t('storage.metric.nextExpire') }}</span>
          <strong>{{ formatDate(storageUnifiedExpireAt) }}</strong>
        </div>
        <div class="platform-storage-card">
          <span>{{ t('platform.tenants.storageStatus') }}</span>
          <strong>{{ storageUsage.upload_locked ? t('storage.status.uploadPaused') : t('storage.status.uploadAvailable') }}</strong>
        </div>
      </div>

      <el-form :model="storageForm" label-width="140px" class="platform-storage-form">
        <el-form-item :label="entitlementResource === 'storage' ? t('storage.dialog.capacity') : t('entitlements.seats.addQuantity')">
          <el-input-number v-model="storageForm.quantity" :min="0" :max="entitlementResource === 'storage' ? 1024 : 10000" @change="loadEntitlementQuote" />
          <div class="platform-storage-hint">{{ t('entitlements.form.zeroQuantityHint') }}</div>
        </el-form-item>
        <el-form-item :label="t('entitlements.dialog.targetYears')">
          <el-input-number v-model="storageForm.years" :min="1" :max="10" @change="loadEntitlementQuote" />
        </el-form-item>
        <el-form-item :label="t('storage.dialog.amount')">
          <el-input :model-value="entitlementQuote?.due_display || '-'" disabled />
        </el-form-item>
        <el-form-item v-if="entitlementQuote" :label="t('entitlements.quote.formula')">
          <div class="platform-storage-hint">{{ entitlementQuoteFormulaSummary }}</div>
        </el-form-item>
        <el-form-item :label="t('storage.dialog.remark')">
          <el-input v-model="storageForm.remark" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="savingStorage" @click="submitStorageExpansion">{{ t('platform.entitlements.createPending') }}</el-button>
          <el-button :loading="savingStorage" @click="sendEntitlementQuote">{{ t('platform.entitlements.sendQuote') }}</el-button>
          <el-button @click="resetStorageForm">{{ t('common.action.clear') }}</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="storageRecords" v-loading="loadingStorage">
        <el-table-column :label="entitlementResource === 'storage' ? t('storage.table.capacity') : t('entitlements.seats.quantity')" width="120">
          <template #default="{ row }">{{ entitlementResource === 'storage' ? `${row.capacity_gb} GB` : `${row.seat_count} ${t('entitlements.unit.person')}` }}</template>
        </el-table-column>
        <el-table-column prop="years" :label="t('storage.table.years')" width="90" />
        <el-table-column prop="status" :label="t('common.table.status')" width="120">
          <template #default="{ row }"><el-tag :type="storageStatusTagType(row.status)">{{ t(`storage.status.${row.status || 'unknown'}`) }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="valid_to" :label="t('storage.table.validTo')" width="180">
          <template #default="{ row }">{{ formatDate(row.valid_to) }}</template>
        </el-table-column>
        <el-table-column prop="total_amount_cents" :label="t('storage.table.amount')" width="150">
          <template #default="{ row }">{{ row.total_amount_display || formatCents(row.total_amount_cents, row.currency) }}</template>
        </el-table-column>
        <el-table-column prop="remark" :label="t('storage.table.remark')" min-width="160" />
        <el-table-column :label="t('common.table.operation')" width="180">
          <template #default="{ row }">
            <template v-if="row.status === 'pending'">
              <el-button size="small" type="success" @click="approveEntitlement(row)">{{ t('platform.entitlements.approve') }}</el-button>
              <el-button size="small" type="danger" @click="rejectEntitlement(row)">{{ t('platform.entitlements.reject') }}</el-button>
            </template>
            <el-button v-else size="small" @click="editStorageExpansion(row)">{{ t('common.action.edit') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="storageVisible = false">{{ t('common.action.close') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../../api'

const loading = ref(false)
const saving = ref(false)
const createVisible = ref(false)
const editVisible = ref(false)
const passwordVisible = ref(false)
const savingPassword = ref(false)
const storageVisible = ref(false)
const loadingStorage = ref(false)
const savingStorage = ref(false)
const storageUsage = ref(null)
const storageRecords = ref([])
const entitlementResource = ref('storage')
const entitlementQuote = ref(null)
const rows = ref([])
const plans = ref([])
const industryPackages = ref([])
const form = reactive(defaultForm())
const editForm = reactive(defaultEditForm())
const editingTenantID = ref(0)
const passwordForm = reactive(defaultPasswordForm())
const storageTenantID = ref(0)
const storageForm = reactive(defaultStorageForm())
const storageUnifiedExpireAt = computed(() => (
  storageUsage.value?.current_expire_at || storageUsage.value?.unified_expire_at || storageUsage.value?.next_expire_at || ''
))
const entitlementQuoteFormulaSummary = computed(() => formatEntitlementQuoteFormula(entitlementQuote.value, entitlementResource.value))

onMounted(async () => {
  await Promise.all([load(), loadPlans(), loadIndustryPackages()])
})

function defaultForm() {
  return {
    name: '',
    admin_username: '',
    admin_email: '',
    quota: {
      max_users: 20,
      storage_limit: 1024 * 1024 * 1024,
      plan_code: 'basic'
    },
    industry_package_code: '',
    industry_package_enabled: false
  }
}

async function load() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/platform/tenants/summary', { params: { page: 1, size: 200 } })
    rows.value = res?.data?.items || []
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadPlans() {
  try {
    const res = await api.get('/api/v1/platform/plans')
    plans.value = res?.data?.items || []
  } catch {
    plans.value = []
  }
}

async function loadIndustryPackages() {
  try {
    const res = await api.get('/api/v1/industry/packages')
    industryPackages.value = res?.data?.items || []
  } catch {
    industryPackages.value = []
  }
}

function openCreate() {
  Object.assign(form, defaultForm())
  createVisible.value = true
}

async function submitCreate() {
  if (!form.name || !form.admin_username) {
    ElMessage.warning(t('platform.tenants.nameRequired'))
    return
  }
  saving.value = true
  try {
    const res = await api.post('/api/v1/platform/tenants', form)
    createVisible.value = false
    ElMessage.success(t('common.message.opSuccess'))
    if (res?.data?.activation_link_once) {
      ElMessage.info(`${t('platform.tenants.activationLink')}: ${res.data.activation_link_once}`)
    }
    await load()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    saving.value = false
  }
}

function defaultEditForm() {
  return {
    status: 'active',
    plan_code: 'basic',
    max_users: 20,
    storage_limit: 1024 * 1024 * 1024,
    valid_to: '',
    ai_enabled: false,
    ai_balance_amount: 0,
    ai_monthly_token_limit: 1000000,
    industry_package_code: '',
    industry_package_enabled: false
  }
}

function openEdit(row) {
  editingTenantID.value = Number(row.tenant_id || 0)
  Object.assign(editForm, defaultEditForm(), {
    status: row.status || 'active',
    plan_code: row.plan_code || 'basic',
    max_users: Number(row.max_users || 20),
    storage_limit: Number(row.base_storage_bytes || 1024 * 1024 * 1024),
    valid_to: row.valid_to || row.expire_at || '',
    ai_enabled: Boolean(row.ai_enabled),
    ai_balance_amount: Number(row.ai_balance_amount || 0),
    ai_monthly_token_limit: Number(row.ai_monthly_token_limit || 1000000),
    industry_package_code: row.industry_package_enabled ? (row.industry_package_code || '') : '',
    industry_package_enabled: Boolean(row.industry_package_enabled)
  })
  editVisible.value = true
}

function defaultStorageForm() {
  return {
    id: 0,
    quantity: 0,
    years: 1,
    remark: ''
  }
}

async function openStorage(row) {
  storageTenantID.value = Number(row.tenant_id || 0)
  resetStorageForm()
  storageVisible.value = true
  await loadStorage()
}

async function loadStorage() {
  if (!storageTenantID.value) return
  loadingStorage.value = true
  try {
    const url = entitlementResource.value === 'storage'
      ? `/api/v1/platform/tenants/${storageTenantID.value}/storage-expansions`
      : `/api/v1/platform/tenants/${storageTenantID.value}/seat-expansions`
    const res = await api.get(url)
    storageUsage.value = res?.data || null
    storageRecords.value = [
      ...(res?.data?.pending_request_records || []),
      ...(res?.data?.history_records || res?.data?.expansion_records || [])
    ]
    await loadEntitlementQuote()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    loadingStorage.value = false
  }
}

function resetStorageForm() {
  Object.assign(storageForm, defaultStorageForm())
  entitlementQuote.value = null
  loadEntitlementQuote()
}

function editStorageExpansion(row) {
  Object.assign(storageForm, defaultStorageForm(), {
    id: Number(row.id || 0),
    quantity: Number(entitlementResource.value === 'storage' ? row.capacity_gb || 1 : row.seat_count || 1),
    years: Number(row.years || 1),
    remark: row.remark || ''
  })
  loadEntitlementQuote()
}

async function loadEntitlementQuote() {
  if (!storageTenantID.value || storageForm.quantity === null || storageForm.quantity === undefined || !storageForm.years) return
  try {
    const res = await api.post(`/api/v1/platform/tenants/${storageTenantID.value}/entitlements/quote`, {
      resource: entitlementResource.value,
      quantity: storageForm.quantity,
      years: storageForm.years
    })
    entitlementQuote.value = res?.data || null
  } catch {
    entitlementQuote.value = null
  }
}

async function submitStorageExpansion() {
  if (!storageTenantID.value) return
  savingStorage.value = true
  try {
    const payload = entitlementResource.value === 'storage' ? {
      capacity_gb: storageForm.quantity,
      years: storageForm.years,
      remark: storageForm.remark || ''
    } : {
      resource: 'seats',
      quantity: storageForm.quantity,
      years: storageForm.years,
      remark: storageForm.remark || ''
    }
    if (storageForm.id && entitlementResource.value === 'storage') {
      await api.put(`/api/v1/platform/tenants/${storageTenantID.value}/storage-expansions/${storageForm.id}`, payload)
    } else {
      const url = entitlementResource.value === 'storage'
        ? `/api/v1/platform/tenants/${storageTenantID.value}/storage-expansions`
        : `/api/v1/platform/tenants/${storageTenantID.value}/seat-expansions`
      await api.post(url, payload)
    }
    ElMessage.success(t('common.message.opSuccess'))
    resetStorageForm()
    await Promise.all([loadStorage(), load()])
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    savingStorage.value = false
  }
}

async function sendEntitlementQuote() {
  if (!storageTenantID.value) return
  savingStorage.value = true
  try {
    await api.post(`/api/v1/platform/tenants/${storageTenantID.value}/entitlements/send-quote`, {
      resource: entitlementResource.value,
      quantity: storageForm.quantity,
      years: storageForm.years,
      remark: storageForm.remark || ''
    })
    ElMessage.success(t('platform.entitlements.quoteSent'))
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    savingStorage.value = false
  }
}

async function approveEntitlement(row) {
  await actEntitlement(row, 'approve')
}

async function rejectEntitlement(row) {
  await actEntitlement(row, 'reject')
}

async function actEntitlement(row, action) {
  if (!storageTenantID.value || !row?.id) return
  savingStorage.value = true
  try {
    const base = entitlementResource.value === 'storage' ? 'storage-expansions' : 'seat-expansions'
    await api.post(`/api/v1/platform/tenants/${storageTenantID.value}/${base}/${row.id}/${action}`)
    ElMessage.success(t('common.message.opSuccess'))
    await Promise.all([loadStorage(), load()])
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    savingStorage.value = false
  }
}

function defaultPasswordForm() {
  return {
    tenant_id: 0,
    admin_account: '',
    new_password: '',
    confirm_password: ''
  }
}

async function submitEdit() {
  if (!editingTenantID.value) return
  saving.value = true
  try {
    await api.put(`/api/v1/platform/tenants/${editingTenantID.value}`, {
      status: editForm.status,
      plan_code: editForm.plan_code,
      max_users: editForm.max_users,
      storage_limit: editForm.storage_limit,
      valid_to: editForm.valid_to || '',
      ai_enabled: editForm.ai_enabled,
      ai_balance_amount: editForm.ai_balance_amount,
      ai_monthly_token_limit: editForm.ai_monthly_token_limit,
      industry_package_code: editForm.industry_package_code || '',
      industry_package_enabled: Boolean(editForm.industry_package_code && editForm.industry_package_enabled)
    })
    ElMessage.success(t('common.message.opSuccess'))
    editVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    saving.value = false
  }
}

function onCreateIndustryPackageChange(value) {
  form.industry_package_enabled = Boolean(value)
}

function onEditIndustryPackageChange(value) {
  editForm.industry_package_enabled = Boolean(value)
}

function industryPackageLabel(code, name) {
  if (code === 'manufacturing') return t('industry.manufacturing.name')
  return name || code || t('platform.tenants.industryPackageNone')
}

function resetAdminPassword(row) {
  Object.assign(passwordForm, defaultPasswordForm(), {
    tenant_id: Number(row.tenant_id || 0),
    admin_account: row.admin_account_masked || '-'
  })
  passwordVisible.value = true
}

function generateRandomPassword() {
  const lower = 'abcdefghijkmnpqrstuvwxyz'
  const upper = 'ABCDEFGHJKLMNPQRSTUVWXYZ'
  const digits = '23456789'
  const symbols = '!@#$%*'
  const all = lower + upper + digits + symbols
  const length = 14
  const chars = []

  chars.push(randomChar(lower))
  chars.push(randomChar(upper))
  chars.push(randomChar(digits))
  chars.push(randomChar(symbols))
  for (let i = chars.length; i < length; i += 1) {
    chars.push(randomChar(all))
  }

  for (let i = chars.length - 1; i > 0; i -= 1) {
    const j = Math.floor(Math.random() * (i + 1))
    const tmp = chars[i]
    chars[i] = chars[j]
    chars[j] = tmp
  }

  const generated = chars.join('')
  passwordForm.new_password = generated
  passwordForm.confirm_password = generated
}

function randomChar(pool) {
  return pool[Math.floor(Math.random() * pool.length)]
}

async function submitResetAdminPassword() {
  if (!passwordForm.tenant_id) return
  if (!passwordForm.new_password) {
    ElMessage.warning(t('app.newPasswordRequired'))
    return
  }
  if (String(passwordForm.new_password).length < 8) {
    ElMessage.warning(t('app.passwordMinLength8'))
    return
  }
  if (passwordForm.new_password !== passwordForm.confirm_password) {
    ElMessage.warning(t('system.message.passwordMismatch'))
    return
  }
  savingPassword.value = true
  try {
    const res = await api.post(`/api/v1/platform/tenants/${passwordForm.tenant_id}/admin-password/reset`, {
      new_password: passwordForm.new_password
    })
    if (res?.data?.initial_password_once) {
      ElMessage.success(`${t('platform.tenants.initialPassword')}: ${res.data.initial_password_once}`)
    } else {
      ElMessage.success(t('common.message.opSuccess'))
    }
    passwordVisible.value = false
    Object.assign(passwordForm, defaultPasswordForm())
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    savingPassword.value = false
  }
}

async function deleteTenant(row) {
  const tenantID = Number(row.tenant_id || 0)
  if (!tenantID) return
  try {
    await ElMessageBox.confirm(
      t('platform.tenants.confirmDeleteTenant').replace('{id}', String(tenantID)),
      t('platform.tenants.deleteTenantTitle'),
      { type: 'warning', confirmButtonText: t('common.action.delete'), cancelButtonText: t('common.action.cancel') }
    )
  } catch {
    return
  }
  loading.value = true
  try {
    await api.delete(`/api/v1/tenants/${tenantID}`)
    ElMessage.success(t('common.message.opSuccess'))
    await load()
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    loading.value = false
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
    i++
  }
  return `${val.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}
function formatMB(value) {
  return `${Number(value || 0).toFixed(2)} MB`
}
function formatNumber(value) {
  return Number(value || 0).toLocaleString()
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

function formatEntitlementQuoteFormula(payload, resource) {
  if (!payload) return ''
  const params = {
    quantity: payload.target_quantity || 0,
    unit: resource === 'storage' ? 'GB' : t('entitlements.unit.person'),
    years: payload.target_years || storageForm.years || 0,
    months: payload.target_months || Number(payload.target_years || storageForm.years || 0) * 12,
    price: payload.unit_price_display || formatCents(payload.unit_price_cents, payload.currency),
    credit: payload.current_credit_display || formatCents(payload.current_credit_cents, payload.currency),
    due: payload.due_display || formatCents(payload.due_cents, payload.currency)
  }
  return resource === 'seats'
    ? t('entitlements.quote.seatFormulaSummary', params)
    : t('entitlements.quote.storageFormulaSummary', params)
}

function planModulesPreview(planCode) {
  const plan = (plans.value || []).find((item) => item.plan_code === planCode)
  const modules = Array.isArray(plan?.modules) ? plan.modules : []
  return modules.join(', ')
}

function storageStatusTagType(status) {
  return {
    active: 'success',
    pending: 'warning',
    expired: 'danger',
    cancelled: 'info'
  }[status] || 'info'
}
</script>

<style scoped>
.password-row {
  display: flex;
  gap: 8px;
  width: 100%;
}

.password-row .el-input {
  flex: 1;
}

.platform-storage-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.platform-storage-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #f8fafc;
}

.platform-storage-card span {
  color: #64748b;
  font-size: 12px;
}

.platform-storage-card strong {
  color: #0f172a;
}

.platform-storage-hint {
  width: 100%;
  margin-top: 6px;
  color: #64748b;
  font-size: 12px;
  line-height: 1.4;
}

.platform-storage-form {
  padding: 12px 0;
  border-top: 1px solid #e5e7eb;
}

@media (max-width: 900px) {
  .platform-storage-summary {
    grid-template-columns: 1fr;
  }
}
</style>
