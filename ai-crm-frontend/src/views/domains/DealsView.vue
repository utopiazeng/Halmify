<template>
  <div class="system-page deals-page">
    <section class="crm-help-strip"><span>{{ t('domain.auto.1266d48ed9') }}</span></section>

    <section class="deal-kpi-grid">
      <div v-for="item in kpiCards" :key="item.key" class="lead-kpi-card"><span>{{ item.title }}</span><strong>{{ item.value }}</strong></div>
    </section>

    <section class="deal-main-grid">
      <div class="system-card">
        <div class="list-card-search">
          <el-form :model="filters" label-width="72px" class="system-search-form">
            <el-form-item :label="t('domain.auto.cc1b21e800')"><el-input v-model="filters.keyword" :placeholder="t('domain.auto.24e3841b56')" clearable /></el-form-item>
            <el-form-item :label="t('orders.company')"><el-input v-model="filters.company_name" :placeholder="t('orders.searchCompany')" clearable /></el-form-item>
            <el-form-item :label="t('orders.contact')"><el-input v-model="filters.contact_name" :placeholder="t('orders.searchContact')" clearable /></el-form-item>
            <el-form-item :label="t('domain.auto.62e951a692')"><el-select v-model="filters.status" clearable :placeholder="t('domain.auto.1a4c26d92d')"><el-option :label="t('domain.auto.128b1b1447')" value="closed_won" /><el-option :label="t('domain.auto.5cfbea2b76')" value="archived" /></el-select></el-form-item>
            <el-form-item :label="t('domain.auto.974d383f36')"><el-select v-model="filters.owner_id" clearable filterable :placeholder="t('domain.auto.8b134548e7')"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
            <div class="system-search-actions">
              <el-button @click="resetFilters">{{ t('domain.auto.3d81345303') }}</el-button>
              <el-button type="primary" @click="searchPage">{{ t('domain.auto.711363c424') }}</el-button>
            </div>
          </el-form>
        </div>
        <div class="system-toolbar">
          <div class="toolbar-left">
            <el-button v-if="canManage" type="primary" @click="openCreate">{{ t('domain.auto.d981c0e88f') }}</el-button>
            <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openEdit(selectedRecord)">{{ t('domain.auto.b006f8c7a6') }}</el-button>
            <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" type="success" plain @click="createOrder(selectedRecord)">{{ t('domain.auto.2363444300') }}</el-button>
          </div>
          <el-tag>{{ rows.length }} {{ t('domain.auto.bce2ef6151') }}</el-tag>
        </div>
        <el-table :data="rows" class="system-table deal-list-table" row-key="id" v-loading="loading" :row-class-name="dealRowClass" @row-click="selectRecord">
          <el-table-column prop="name" :label="t('domain.auto.4483df6e28')" min-width="220">
            <template #default="{ row }">
              <div class="record-title">
                <strong>{{ row.name }} <el-tag v-if="objectVisibility(row) === 'private'" size="small" type="warning">{{ t('domain.auto.c931219c30') }}</el-tag></strong>
                <span>{{ dealCompany(row) }} · {{ dealPayload(row).contract_no || t('domain.auto.d0492f4eb1') }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('orders.company')" min-width="160"><template #default="{ row }">{{ dealCompany(row) }}</template></el-table-column>
          <el-table-column :label="t('orders.contact')" min-width="140"><template #default="{ row }">{{ dealContact(row) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.5ac60d781c')" width="150"><template #default="{ row }">{{ formatAmount(row.amount, row.currency) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.5715cababb')" width="130"><template #default="{ row }">{{ dealPayload(row).closed_date || row.created_at?.slice(0, 10) || '-' }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.974d383f36')" width="150"><template #default="{ row }">{{ ownerName(row.owner_id) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.bfa1cf8493')" width="150"><template #default="{ row }">{{ dealPayload(row).sales_owner || '-' }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.62e951a692')" width="110"><template #default="{ row }"><el-tag type="success">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
          <el-table-column :label="t('domain.auto.f3ea6d345e')" width="180" fixed="right"><template #default="{ row }"><div class="row-actions deal-row-actions" @click.stop><button class="action-btn deal-action-btn view" @click="selectRecord(row)">{{ t('domain.auto.8a08005231') }}</button><button v-if="canEditRecord(row)" class="action-btn deal-action-btn success" @click="createOrder(row)">{{ t('domain.auto.fd6ce2a756') }}</button></div></template></el-table-column>
        </el-table>
        <div class="system-pagination"><span>{{ t('domain.auto.3b6ef811b8') }} {{ pagination.total }} {{ t('domain.auto.bce2ef6151') }}</span><el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.size" layout="prev, pager, next, sizes" :total="pagination.total" @current-change="loadPage" @size-change="loadPage" /></div>
      </div>

      <aside class="system-card deal-side-card">
        <div class="chart-panel-head"><strong>{{ t('domain.auto.5329336ad0') }}</strong><el-tag size="small">{{ t('domain.auto.3d19b27350') }}</el-tag></div>
        <p class="crm-inline-help">{{ t('domain.auto.4d2c26625c') }}</p>
        <div v-if="selectedRecord" class="lead-summary">
          <div><span>{{ t('domain.auto.4483df6e28') }}</span><strong>{{ selectedRecord.name }}</strong></div>
          <div><span>{{ t('orders.company') }}</span><strong>{{ dealCompany(selectedRecord) }}</strong></div>
          <div><span>{{ t('orders.contact') }}</span><strong>{{ dealContact(selectedRecord) }}</strong></div>
          <div><span>{{ t('domain.auto.34943c40c9') }}</span><strong>{{ formatAmount(selectedRecord.amount, selectedRecord.currency) }}</strong></div>
          <div><span>{{ t('domain.auto.974d383f36') }}</span><strong>{{ ownerName(selectedRecord.owner_id) }}</strong></div>
          <div><span>{{ t('domain.auto.b6ea8162a5') }}</span><strong>{{ selectedPayload.contract_no || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.014e35ef3a') }}</span><strong>{{ selectedPayload.commission_note || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.c63f79e636') }}</span><strong>{{ selectedPayload.source_channel || selectedPayload.growth_source || '-' }}</strong></div>
        </div>
        <template v-if="selectedRecord">
          <ObjectAccessPanel
            domain="deals"
            :record="selectedRecord"
            :users="users"
            :can-manage="canManageRecordACL(selectedRecord)"
            :object-label="t('domain.auto.6a30a471cf')"
            @updated="refreshSelected"
          />
          <el-divider>{{ t('domain.auto.91238956f1') }}</el-divider>
          <DealChainPanel object-type="deals" :object-id="selectedRecord.id" :users="users" />
        </template>
        <el-empty v-else :description="t('domain.auto.404fd2a5e0')" />
      </aside>
    </section>

    <el-dialog v-model="formDialog" :title="editingId ? t('domain.auto.b006f8c7a6') : t('domain.auto.d981c0e88f')" width="min(900px, 96vw)" destroy-on-close>
      <el-form :model="form" label-width="96px" class="lead-dialog-form">
        <el-form-item :label="t('domain.auto.5b60c043b9')" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('domain.auto.e941d410f4')">
          <el-select v-model="form.company_id" class="full-input" clearable filterable :placeholder="t('domain.auto.42379aae38')" @change="applyCompany">
            <el-option v-for="item in companyOptions" :key="item.id" :label="companyOptionLabel(item)" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.2425bd4bc1')">
          <el-select v-model="form.contact_id" class="full-input" clearable filterable :placeholder="t('domain.auto.56a1a3ddad')" @change="applyContact">
            <el-option v-for="item in contactOptions" :key="item.id" :label="contactOptionLabel(item)" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('domain.auto.5ac60d781c')"><el-input-number v-model="form.amount" :min="0" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.a81ab5e100')"><el-select v-model="form.currency" filterable><el-option v-for="item in currencyOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.974d383f36')"><el-select v-model="form.owner_id" clearable filterable class="full-input"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.241d8ef92f')"><ObjectAccessField v-model="form.payload.visibility" :public-label="t('domain.auto.eec17a52b9')" :private-label="t('domain.auto.f308642fa0')" /></el-form-item>
        <el-form-item :label="t('domain.auto.b6ea8162a5')"><el-input v-model="form.payload.contract_no" /></el-form-item>
        <el-form-item :label="t('domain.auto.5715cababb')"><el-date-picker v-model="form.payload.closed_date" value-format="YYYY-MM-DD" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.bfa1cf8493')"><el-input v-model="form.payload.sales_owner" /></el-form-item>
        <el-form-item :label="t('domain.auto.014e35ef3a')" class="wide-field"><el-input v-model="form.payload.commission_note" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="formDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" @click="submitForm">{{ t('domain.auto.fadf24dbc5') }}</el-button></template>
    </el-dialog>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import { canEditRecord, canManageDomain, canManageRecordACL, currentUser } from '../../permissions'
import DealChainPanel from '../../components/DealChainPanel.vue'
import ObjectAccessPanel from '../../components/ObjectAccessPanel.vue'
import ObjectAccessField from '../../components/ObjectAccessField.vue'
import { playCelebration } from '../../utils/celebration'
import { useCustomerChainOptions } from '../../utils/customerChain'
import { currencyOptions, formatCurrencyAmount, normalizeCurrencyCode } from '../../utils/currency'
import { defaultBusinessCurrency } from '../../utils/company-format'
import { apiErrorMessage } from '../../utils/api-error'

const rows = ref([])
const users = ref([])
const loading = ref(false)
const selectedRecord = ref(null)
const formDialog = ref(false)
const editingId = ref(null)
const filters = reactive({ keyword: '', company_name: '', contact_name: '', status: '', owner_id: undefined })
const pagination = reactive({ page: 1, size: 20, total: 0 })
const form = reactive(defaultForm())
const {
  companyOptions,
  contactOptions,
  loadCustomerOptions,
  companyOptionLabel,
  contactOptionLabel,
  applyCompany,
  applyContact,
  syncCustomerChainFromRecord,
  normalizeCustomerChainForSubmit
} = useCustomerChainOptions(form)
const canManage = computed(() => canManageDomain('deals'))
const selectedPayload = computed(() => dealPayload(selectedRecord.value))
const ownerOptions = computed(() => users.value.map((item) => ({ id: item.id, label: userDisplayName(item) })))
const kpiCards = computed(() => [
  { key: 'count', title: t('domain.auto.a88ede0377'), value: rows.value.length },
  { key: 'amount', title: t('domain.auto.5ac60d781c'), value: formatAmount(rows.value.reduce((sum, item) => sum + Number(item.amount || 0), 0)) },
  { key: 'orders', title: t('domain.auto.b13e5970f0'), value: rows.value.filter((item) => !dealPayload(item).order_id).length },
  { key: 'avg', title: t('domain.auto.1b85e42e98'), value: formatAmount(rows.value.length ? rows.value.reduce((sum, item) => sum + Number(item.amount || 0), 0) / rows.value.length : 0) }
])

onMounted(async () => { await Promise.all([loadUsers(), loadCustomerOptions(), loadPage()]) })
async function loadUsers() { try { const res = await api.get('/api/v1/accounts/users', { params: { page: 1, size: 200, status: 'active' } }); users.value = res.data?.items || [] } catch { users.value = [] } }
async function loadPage() { loading.value = true; try { const res = await api.get('/api/v1/prd/deals/objects', { params: cleanParams({ object_type: 'closed_deal', ...filters, page: pagination.page, size: pagination.size }) }); rows.value = res.data?.items || []; pagination.total = res.data?.total || 0; if (selectedRecord.value) selectedRecord.value = rows.value.find((item) => item.id === selectedRecord.value.id) || selectedRecord.value } catch (error) { ElMessage.error(apiErrorMessage(error, t('domain.auto.f6b7a41e56'))) } finally { loading.value = false } }
function searchPage() { pagination.page = 1; loadPage() }
function selectRecord(row) { selectedRecord.value = row }
function dealRowClass({ row }) { return selectedRecord.value && Number(selectedRecord.value.id) === Number(row.id) ? 'is-selected-row' : '' }
function openCreate() { editingId.value = null; Object.assign(form, defaultForm()); formDialog.value = true; if (!companyOptions.value.length || !contactOptions.value.length) loadCustomerOptions() }
function openEdit(row) { if (!row) return; editingId.value = row.id; Object.assign(form, { object_type: 'closed_deal', name: row.name, code: row.code || '', status: row.status || 'closed_won', owner_id: row.owner_id || 0, company_id: row.company_id || 0, contact_id: row.contact_id || 0, amount: row.amount || 0, currency: normalizeCurrencyCode(row.currency, defaultBusinessCurrency()), score: row.score || 100, tags: splitTags(row.tags), payload: { ...defaultForm().payload, ...dealPayload(row) } }); syncCustomerChainFromRecord(row); formDialog.value = true; if (!companyOptions.value.length || !contactOptions.value.length) loadCustomerOptions() }
async function submitForm() { if (!form.name) return ElMessage.error(t('domain.auto.e6a93d99bd')); normalizeCustomerChainForSubmit(); const isCreate = !editingId.value; try { if (editingId.value) await api.put(`/api/v1/prd/deals/objects/${editingId.value}`, form); else await api.post('/api/v1/prd/deals/objects', form); if (isCreate) playCelebration('win'); ElMessage.success(t('domain.auto.92b53f92d3')); formDialog.value = false; await loadPage() } catch (error) { ElMessage.error(apiErrorMessage(error, t('domain.auto.b412eb86ab'))) } }
async function createOrder(row) { if (!row) return; try { await api.post(`/api/v1/prd/flows/deals/${row.id}/order`); playCelebration('win'); ElMessage.success(t('domain.auto.7894c404dd')); await loadPage() } catch (error) { ElMessage.error(apiErrorMessage(error, t('domain.auto.ca37aec661'))) } }
function resetFilters() { Object.assign(filters, { keyword: '', company_name: '', contact_name: '', status: '', owner_id: undefined }); pagination.page = 1; loadPage() }
async function refreshSelected(id) { await loadPage(); const next = rows.value.find((item) => Number(item.id) === Number(id)); if (next) selectedRecord.value = next }
function defaultForm() { return { object_type: 'closed_deal', name: '', code: '', status: 'closed_won', owner_id: Number(currentUser().id || 0), company_id: 0, contact_id: 0, amount: 0, currency: defaultBusinessCurrency(), score: 100, tags: [], payload: { company_id: undefined, company: '', company_name: '', contact_id: undefined, contact: '', contact_name: '', contract_no: '', closed_date: '', sales_owner: '', commission_note: '', visibility: 'public' } } }
function dealPayload(row) { try { return typeof row?.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row?.payload_json || {}) } catch { return {} } }
function dealCompany(row) { const p = dealPayload(row); return p.company || p.company_name || '-' }
function dealContact(row) { const p = dealPayload(row); return p.contact_name || p.contact || p.name_contact || '-' }
function objectVisibility(row) { const p = dealPayload(row); return p.visibility === 'private' || p.access_visibility === 'private' || p.private === true || p.private === 'true' ? 'private' : 'public' }
function splitTags(value) { return String(value || '').split(',').map((item) => item.trim()).filter(Boolean) }
function cleanParams(params) { return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== '' && value !== undefined && value !== null)) }
function statusLabel(value) { return ({ closed_won: t('domain.auto.128b1b1447'), archived: t('domain.auto.5cfbea2b76') }[value] || value || '-') }
function formatAmount(amount, currency = defaultBusinessCurrency()) { return formatCurrencyAmount(amount, normalizeCurrencyCode(currency, defaultBusinessCurrency())) }
function ownerName(ownerID) { const id = Number(ownerID || 0); if (!id) return t('domain.auto.3bb92e6efc'); const user = users.value.find((item) => Number(item.id) === id); return user ? userDisplayName(user) : `${t('domain.auto.9ba763ea34')}${id}` }
function userDisplayName(user) { const label = user.name || user.username || user.email || `${t('domain.auto.9ba763ea34')}${user.id}`; return user.department_name ? `${label} · ${user.department_name}` : label }
</script>
