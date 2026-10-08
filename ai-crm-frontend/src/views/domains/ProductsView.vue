<template>
  <div class="system-page products-page">
    <section class="crm-help-strip">
      <span>{{ t('domain.auto.e21aedc231') }}</span>
    </section>

    <section class="product-kpi-grid">
      <MetricCard v-for="item in kpiCards" :key="item.key" v-bind="item" />
    </section>

    <section class="product-main-grid">
      <div class="system-card product-list-card">
        <div class="list-card-search">
          <el-form :model="filters" label-width="72px" class="system-search-form">
            <el-form-item :label="t('domain.auto.cc1b21e800')"><el-input v-model="filters.keyword" :placeholder="t('domain.auto.d4392b4016')" clearable /></el-form-item>
            <el-form-item :label="t('domain.auto.62e951a692')">
              <el-select v-model="filters.status" clearable :placeholder="t('domain.auto.1a4c26d92d')">
                <el-option :label="t('domain.auto.19f18d2c22')" value="on_sale" />
                <el-option :label="t('domain.auto.cf7e9e2519')" value="testing" />
                <el-option :label="t('domain.auto.6c7dcbb73a')" value="stopped" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('domain.auto.435c5259e4')"><el-input v-model="filters.category" :placeholder="t('domain.auto.cde67b450f')" clearable /></el-form-item>
            <div class="system-search-actions">
              <el-button @click="resetFilters">{{ t('domain.auto.3d81345303') }}</el-button>
              <el-button type="primary" @click="loadPage">{{ t('domain.auto.711363c424') }}</el-button>
            </div>
          </el-form>
        </div>
        <div class="system-toolbar">
          <div class="toolbar-left">
            <el-button v-if="canManage" type="primary" @click="openCreate">{{ t('domain.auto.8a76021798') }}</el-button>
            <el-button v-if="canManage" :disabled="!selectedRecord" @click="openEdit(selectedRecord)">{{ t('domain.auto.1e02d2583c') }}</el-button>
            <el-button v-if="canManage" :disabled="!selectedRecord" @click="openAdjust(selectedRecord)">{{ t('domain.auto.aafb96792b') }}</el-button>
            <el-button :disabled="!rows.length" @click="exportObjects">{{ t('domain.auto.188896795f') }}</el-button>
          </div>
          <div class="toolbar-icons">
            <el-tooltip :content="t('domain.auto.38108eaa1d')"><button class="system-icon-btn" :aria-label="t('domain.auto.38108eaa1d')" @click="loadPage"><Icon icon="ri:refresh-line" /></button></el-tooltip>
          </div>
        </div>

        <el-table
          :data="rows"
          class="system-table product-list-table"
          row-key="id"
          v-loading="loading"
          highlight-current-row
          :current-row-key="selectedRecord?.id"
          :row-class-name="productRowClass"
          @row-click="selectRecord"
        >
          <el-table-column prop="name" :label="t('domain.auto.eb3ee63b7a')" min-width="210">
            <template #default="{ row }">
              <div class="record-title">
                <strong>{{ row.name }}</strong>
                <span>{{ productPayload(row).sku || row.code || '-' }} · {{ productPayload(row).category || t('domain.auto.b28f13ea3e') }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('domain.auto.62e951a692')" width="110"><template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
          <el-table-column :label="t('domain.auto.8ec8c8c712')" width="90"><template #default="{ row }">{{ productPayload(row).unit || '-' }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.b70c2d28a1')" width="130"><template #default="{ row }">{{ formatAmount(row.amount, row.currency) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.dfd6b7aab1')" width="150">
            <template #default="{ row }">
              <span :class="{ 'sla-overdue': isLowStock(row) }">{{ productPayload(row).stock || 0 }}</span>
              <small class="product-stock-note">{{ t('domain.auto.cbe7b234f9') }} {{ productPayload(row).stock_threshold || 0 }}</small>
            </template>
          </el-table-column>
          <el-table-column :label="t('domain.auto.37d56ad493')" width="100"><template #default="{ row }">{{ productPayload(row).tax_rate || 0 }}%</template></el-table-column>
          <el-table-column :label="t('domain.auto.d9db02d07a')" width="150"><template #default="{ row }">{{ formatTime(row.updated_at) }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.f3ea6d345e')" width="160" fixed="right">
            <template #default="{ row }">
              <div class="row-actions three" @click.stop>
                <el-tooltip :content="t('domain.auto.f7acefd2d4')"><button class="action-btn view" @click="selectRecord(row)"><Icon icon="ri:eye-line" /></button></el-tooltip>
                <el-tooltip v-if="canManage" :content="t('domain.auto.a7f814c0a4')"><button class="action-btn edit" @click="openEdit(row)"><Icon icon="ri:pencil-line" /></button></el-tooltip>
                <el-tooltip v-if="canManage" :content="t('domain.auto.aafb96792b')"><button class="action-btn success" @click="openAdjust(row)"><Icon icon="ri:archive-stack-line" /></button></el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <div class="system-pagination">
          <span>{{ t('domain.auto.3b6ef811b8') }} {{ pagination.total }} {{ t('domain.auto.bce2ef6151') }}</span>
          <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.size" layout="prev, pager, next, sizes, jumper" :total="pagination.total" @current-change="loadPage" @size-change="loadPage" />
        </div>
      </div>

      <aside class="system-card product-side-card">
        <div class="chart-panel-head">
          <strong>{{ t('domain.auto.e8ee7bcad5') }}</strong>
          <el-tag :type="lowStockRows.length ? 'danger' : 'success'">{{ lowStockRows.length }} {{ t('domain.auto.f7b2a6ee68') }}</el-tag>
        </div>
        <p class="crm-inline-help">{{ t('product.stockWarning.help') }}</p>
        <div class="product-warning-list">
          <button v-for="item in lowStockRows" :key="item.id" class="product-warning-row" @click="selectRecord(item)">
            <strong>{{ item.name }} <el-tag size="small" :type="stockRiskType(item)">{{ stockRiskLabel(item) }}</el-tag></strong>
            <span>{{ productPayload(item).sku || item.code || '-' }} {{ t('domain.auto.bc06fdbdc1') }} {{ productPayload(item).stock || 0 }} {{ t('domain.auto.9c63f8024b') }} {{ productPayload(item).stock_threshold || 0 }}</span>
          </button>
        </div>
        <div v-if="!lowStockRows.length" class="product-warning-empty">
          <strong>{{ t('product.stockWarning.emptyTitle') }}</strong>
          <span>{{ t('product.stockWarning.emptyDesc') }}</span>
        </div>

        <el-divider>{{ t('domain.auto.d54d26986a') }}</el-divider>
        <div v-if="selectedRecord" class="product-image-panel">
          <div class="product-image-frame">
            <img v-if="productImageSrc" :src="productImageSrc" :alt="selectedRecord.name" />
            <div v-else class="product-image-empty">
              <Icon :icon="imageLoading ? 'ri:loader-4-line' : 'ri:image-add-line'" :class="{ spinning: imageLoading }" />
              <span>{{ imageLoading ? t('product.image.loading') : imageLoadError || t('product.image.empty') }}</span>
            </div>
          </div>
          <el-upload v-if="canManage" :show-file-list="false" :before-upload="uploadProductImage" :disabled="imageUploading" accept="image/*">
            <el-button size="small" type="primary" plain :loading="imageUploading">{{ t('product.image.upload') }}</el-button>
          </el-upload>
        </div>
        <div v-if="selectedRecord" class="lead-summary product-summary">
          <div><span>{{ t('domain.auto.6cc98552a9') }}</span><strong>{{ selectedRecord.name }}</strong></div>
          <div><span>SKU</span><strong>{{ selectedPayload.sku || selectedRecord.code || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.435c5259e4') }}</span><strong>{{ selectedPayload.category || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.b70c2d28a1') }}</span><strong>{{ formatAmount(selectedRecord.amount, selectedRecord.currency) }}</strong></div>
          <div><span>{{ t('domain.auto.dfd6b7aab1') }}</span><strong>{{ selectedPayload.stock || 0 }}</strong></div>
          <div><span>{{ t('domain.auto.3cea1d9847') }}</span><strong>{{ selectedPayload.stock_threshold || 0 }}</strong></div>
        </div>
        <el-empty v-else :description="t('domain.auto.569f0edd42')" />
      </aside>
    </section>

    <el-dialog v-model="formDialog" :title="editingId ? t('domain.auto.1e02d2583c') : t('domain.auto.8a76021798')" width="min(980px, 96vw)" destroy-on-close>
      <el-form :model="form" label-width="96px" class="lead-dialog-form">
        <el-form-item :label="t('domain.auto.abc0ac7995')" required><el-input v-model="form.name" :placeholder="t('domain.auto.14c898695d')" /></el-form-item>
        <el-form-item label="SKU"><el-input v-model="form.payload.sku" :placeholder="t('domain.auto.0c9ab9697f')" /></el-form-item>
        <el-form-item :label="t('domain.auto.435c5259e4')"><el-input v-model="form.payload.category" :placeholder="t('domain.auto.37c8806424')" /></el-form-item>
        <el-form-item :label="t('domain.auto.62e951a692')"><el-select v-model="form.status"><el-option :label="t('domain.auto.19f18d2c22')" value="on_sale" /><el-option :label="t('domain.auto.cf7e9e2519')" value="testing" /><el-option :label="t('domain.auto.6c7dcbb73a')" value="stopped" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.8ec8c8c712')"><el-input v-model="form.payload.unit" :placeholder="t('domain.auto.fe18433168')" /></el-form-item>
        <el-form-item :label="t('domain.auto.7ed045fd3b')"><el-input-number v-model="form.amount" :min="0" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.a81ab5e100')"><el-select v-model="form.currency" filterable><el-option v-for="item in currencyOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.37d56ad493')"><el-input-number v-model="form.payload.tax_rate" :min="0" :max="100" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.684dcaa512')"><el-input-number v-model="form.payload.stock" :min="0" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.0d2e218a84')"><el-input-number v-model="form.payload.stock_threshold" :min="0" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.26670dda42')" class="wide-field"><el-input v-model="form.payload.description" type="textarea" :rows="3" :placeholder="t('domain.auto.7b695f2012')" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="formDialog = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" @click="submitForm">{{ t('domain.auto.fadf24dbc5') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="adjustDialog" :title="t('domain.auto.aafb96792b')" width="560px" destroy-on-close>
      <el-form :model="adjustForm" label-width="96px">
        <el-form-item :label="t('domain.auto.8eaccd38a5')">
          <el-radio-group v-model="adjustForm.type">
            <el-radio-button label="increase">{{ t('domain.auto.cb7c9a7bb3') }}</el-radio-button>
            <el-radio-button label="decrease">{{ t('domain.auto.bb22ac4a42') }}</el-radio-button>
            <el-radio-button label="set">{{ t('domain.auto.3b100673a3') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('domain.auto.b9ae893190')" required><el-input-number v-model="adjustForm.quantity" :min="0" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.cd612544e1')" required><el-input v-model="adjustForm.reason" type="textarea" :rows="4" :placeholder="t('domain.auto.bf09dc2043')" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="adjustDialog = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
        <el-button type="primary" @click="submitAdjust">{{ t('domain.auto.862d3e5dd8') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage } from 'element-plus'
import api from '../../api'
import { canManageDomain } from '../../permissions'
import MetricCard from '../../components/MetricCard.vue'
import { downloadProtectedFile, unwrapFileUploadResponse } from '../../utils/download'
import { currencyOptions, formatCurrencyAmount, normalizeCurrencyCode } from '../../utils/currency'
import { defaultBusinessCurrency } from '../../utils/company-format'

const rows = ref([])
const loading = ref(false)
const selectedRecord = ref(null)
const formDialog = ref(false)
const adjustDialog = ref(false)
const imageUploading = ref(false)
const imageLoading = ref(false)
const imageLoadError = ref('')
const productImageSrc = ref('')
let productImageObjectURL = ''
const editingId = ref(null)
const filters = reactive({ keyword: '', status: '', category: '' })
const pagination = reactive({ page: 1, size: 20, total: 0 })
const form = reactive(defaultForm())
const adjustForm = reactive({ type: 'increase', quantity: 0, reason: '' })

const selectedPayload = computed(() => productPayload(selectedRecord.value))
const selectedImageURL = computed(() => selectedPayload.value.image_url || selectedPayload.value.cover_url || selectedPayload.value.picture_url || '')
const canManage = computed(() => canManageDomain('products'))
const lowStockRows = computed(() => rows.value.filter((item) => isLowStock(item)))
const kpiCards = computed(() => [
  { key: 'total', title: t('domain.auto.1715eaae10'), value: rows.value.length, icon: 'ri:box-3-line', tone: 'blue', help: t('domain.auto.fde17d0154') },
  { key: 'on_sale', title: t('domain.auto.d6c79a5d24'), value: rows.value.filter((item) => item.status === 'on_sale').length, icon: 'ri:store-2-line', tone: 'green', help: t('domain.auto.4b4cc7d3bd') },
  { key: 'low', title: t('domain.auto.e8ee7bcad5'), value: lowStockRows.value.length, icon: 'ri:alarm-warning-line', tone: lowStockRows.value.length ? 'red' : 'slate', help: t('domain.auto.60eed57ebc') },
  { key: 'value', title: t('domain.auto.0f9bf8caf0'), value: formatAmount(rows.value.reduce((sum, item) => sum + Number(productPayload(item).stock || 0) * Number(item.amount || 0), 0)), icon: 'ri:coins-line', tone: 'orange', help: t('domain.auto.9b595ba106') }
])

onMounted(loadPage)
onBeforeUnmount(revokeProductImageObjectURL)

watch(
  () => [selectedRecord.value?.id, selectedImageURL.value],
  () => {
    loadSelectedProductImage()
  },
  { immediate: true }
)

async function loadPage() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/prd/products/objects', { params: cleanParams({ object_type: 'product', keyword: filters.keyword, status: filters.status, page: pagination.page, size: pagination.size }) })
    let items = res.data?.items || []
    if (filters.category) {
      const keyword = filters.category.trim().toLowerCase()
      items = items.filter((item) => String(productPayload(item).category || '').toLowerCase().includes(keyword))
    }
    rows.value = items
    pagination.total = filters.category ? items.length : (res.data?.total || 0)
    if (selectedRecord.value) {
      const fresh = rows.value.find((item) => item.id === selectedRecord.value.id)
      selectedRecord.value = fresh || null
    }
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error.message || t('domain.auto.170d01542c'))
  } finally {
    loading.value = false
  }
}

function selectRecord(row) {
  selectedRecord.value = row
}
function productRowClass({ row }) {
  return Number(selectedRecord.value?.id || 0) === Number(row.id || 0) ? 'is-selected-row' : ''
}

function openCreate() {
  editingId.value = null
  Object.assign(form, defaultForm())
  formDialog.value = true
}

function openEdit(row) {
  if (!row) return
  editingId.value = row.id
  const p = productPayload(row)
  Object.assign(form, {
    object_type: 'product',
    name: row.name,
    code: row.code || p.sku || '',
    status: row.status || 'on_sale',
    owner_id: row.owner_id || 0,
    amount: row.amount || 0,
    currency: normalizeCurrencyCode(row.currency, defaultBusinessCurrency()),
    score: row.score || 0,
    tags: splitTags(row.tags),
    payload: { unit: t('domain.auto.032231d845'), stock: 0, stock_threshold: 0, tax_rate: 0, ...p }
  })
  formDialog.value = true
}

async function submitForm() {
  if (!form.name) return ElMessage.error(t('domain.auto.cbfa98d408'))
  form.code = form.payload.sku || form.code
  if (editingId.value) await api.put(`/api/v1/prd/products/objects/${editingId.value}`, form)
  else await api.post('/api/v1/prd/products/objects', form)
  ElMessage.success(t('domain.auto.e3dc7956c8'))
  formDialog.value = false
  await loadPage()
}

function openAdjust(row) {
  if (!row) return
  selectedRecord.value = row
  Object.assign(adjustForm, { type: 'increase', quantity: 0, reason: '' })
  adjustDialog.value = true
}

async function submitAdjust() {
  if (!selectedRecord.value) return
  if (!adjustForm.reason) return ElMessage.error(t('domain.auto.bd59f8c970'))
  const current = Number(selectedPayload.value.stock || 0)
  const quantity = Number(adjustForm.quantity || 0)
  let nextStock = current
  if (adjustForm.type === 'increase') nextStock = current + quantity
  else if (adjustForm.type === 'decrease') nextStock = Math.max(0, current - quantity)
  else nextStock = quantity
  const updatedPayload = { ...selectedPayload.value, stock: nextStock }
  await api.put(`/api/v1/prd/products/objects/${selectedRecord.value.id}`, {
    object_type: 'product',
    name: selectedRecord.value.name,
    code: selectedRecord.value.code || updatedPayload.sku || '',
    status: selectedRecord.value.status || 'on_sale',
    owner_id: selectedRecord.value.owner_id || 0,
    amount: selectedRecord.value.amount || 0,
    currency: normalizeCurrencyCode(selectedRecord.value.currency, defaultBusinessCurrency()),
    score: selectedRecord.value.score || 0,
    tags: splitTags(selectedRecord.value.tags),
    payload: updatedPayload
  })
  await api.post('/api/v1/prd/products/objects', {
    object_type: 'inventory',
    name: `${selectedRecord.value.name}${t('domain.auto.aafb96792b')}`,
    code: `INV-${Date.now()}`,
    status: 'active',
    amount: 0,
    currency: normalizeCurrencyCode(selectedRecord.value.currency, defaultBusinessCurrency()),
    payload: {
      product_id: selectedRecord.value.id,
      product_name: selectedRecord.value.name,
      sku: updatedPayload.sku || selectedRecord.value.code || '',
      adjust_type: adjustForm.type,
      quantity,
      before_stock: current,
      after_stock: nextStock,
      reason: adjustForm.reason
    }
  }).catch(() => null)
  ElMessage.success(t('domain.auto.3be578e16b'))
  adjustDialog.value = false
  const selectedID = selectedRecord.value.id
  await loadPage()
  const fresh = rows.value.find((item) => item.id === selectedID)
  if (fresh) selectedRecord.value = fresh
}

async function uploadProductImage(file) {
  if (!selectedRecord.value) {
    ElMessage.warning(t('product.message.selectFirst'))
    return false
  }
  if (!String(file.type || '').startsWith('image/')) {
    ElMessage.error(t('product.message.imageOnly'))
    return false
  }
  imageUploading.value = true
  try {
    const data = new FormData()
    data.append('file', file)
    const upload = await api.post('/api/v1/files/upload', data)
    const info = unwrapFileUploadResponse(upload)
    const imageURL = info.download_url || `/api/v1/files/${info.file_id}/download`
    const updatedPayload = {
      ...selectedPayload.value,
      image_file_id: info.file_id,
      image_url: imageURL,
      image_name: info.original_name || info.file_name || file.name,
      image_uploaded_at: new Date().toISOString()
    }
    await api.put(`/api/v1/prd/products/objects/${selectedRecord.value.id}`, {
      object_type: 'product',
      name: selectedRecord.value.name,
      code: selectedRecord.value.code || updatedPayload.sku || '',
      status: selectedRecord.value.status || 'on_sale',
      owner_id: selectedRecord.value.owner_id || 0,
      amount: selectedRecord.value.amount || 0,
      currency: normalizeCurrencyCode(selectedRecord.value.currency, defaultBusinessCurrency()),
      score: selectedRecord.value.score || 0,
      tags: splitTags(selectedRecord.value.tags),
      payload: updatedPayload
    })
    ElMessage.success(t('product.message.imageUploaded'))
    const selectedID = selectedRecord.value.id
    await loadPage()
    selectedRecord.value = rows.value.find((item) => item.id === selectedID) || selectedRecord.value
    await loadSelectedProductImage()
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error.message || t('product.message.imageUploadFailed'))
  } finally {
    imageUploading.value = false
  }
  return false
}

async function loadSelectedProductImage() {
  revokeProductImageObjectURL()
  imageLoadError.value = ''
  const url = selectedImageURL.value
  if (!selectedRecord.value || !url) {
    imageLoading.value = false
    return
  }
  if (isPublicImageURL(url)) {
    productImageSrc.value = url
    imageLoading.value = false
    return
  }
  imageLoading.value = true
  try {
    const blob = await api.get(url, { responseType: 'blob' })
    productImageObjectURL = URL.createObjectURL(blob instanceof Blob ? blob : new Blob([blob]))
    productImageSrc.value = productImageObjectURL
  } catch (error) {
    productImageSrc.value = ''
    imageLoadError.value = error?.response?.data?.message || error.message || t('product.message.imageLoadFailed')
  } finally {
    imageLoading.value = false
  }
}

function revokeProductImageObjectURL() {
  if (productImageObjectURL) {
    URL.revokeObjectURL(productImageObjectURL)
    productImageObjectURL = ''
  }
  productImageSrc.value = ''
}

function isPublicImageURL(url) {
  const value = String(url || '').trim()
  return value.startsWith('blob:') || value.startsWith('data:') || /^https?:\/\//i.test(value)
}

async function exportObjects() {
  await downloadProtectedFile('/api/v1/prd/products/objects/export', `products-${dateStamp()}.xls`)
  ElMessage.success(t('common.action.export'))
}

function resetFilters() {
  Object.assign(filters, { keyword: '', status: '', category: '' })
  pagination.page = 1
  loadPage()
}

function dateStamp() {
  const date = new Date()
  return `${date.getFullYear()}${String(date.getMonth() + 1).padStart(2, '0')}${String(date.getDate()).padStart(2, '0')}-${String(date.getHours()).padStart(2, '0')}${String(date.getMinutes()).padStart(2, '0')}${String(date.getSeconds()).padStart(2, '0')}`
}

function defaultForm() {
  return { object_type: 'product', name: '', code: '', status: 'on_sale', owner_id: 0, amount: 0, currency: defaultBusinessCurrency(), score: 0, tags: [], payload: { sku: '', category: '', unit: t('domain.auto.032231d845'), tax_rate: 0, stock: 0, stock_threshold: 0, image_url: '', description: '' } }
}
function productPayload(row) { try { return typeof row?.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row?.payload_json || {}) } catch { return {} } }
function stockRiskGap(row) { const p = productPayload(row); return Number(p.stock || 0) - Number(p.stock_threshold || 0) }
function stockRiskBuffer(row) {
  const threshold = Number(productPayload(row).stock_threshold || 0)
  return Math.max(5, Math.ceil(threshold * 0.1))
}
function isLowStock(row) {
  const p = productPayload(row)
  const threshold = Number(p.stock_threshold || 0)
  if (threshold <= 0) return false
  return stockRiskGap(row) <= stockRiskBuffer(row)
}
function stockRiskLabel(row) { return stockRiskGap(row) <= 0 ? t('product.stockRisk.below') : t('product.stockRisk.near') }
function stockRiskType(row) { return stockRiskGap(row) <= 0 ? 'danger' : 'warning' }
function splitTags(value) { return String(value || '').split(',').map((item) => item.trim()).filter(Boolean) }
function cleanParams(params) { return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== '' && value !== undefined && value !== null)) }
function statusLabel(value) { return ({ on_sale: t('domain.auto.19f18d2c22'), stopped: t('domain.auto.6c7dcbb73a'), testing: t('domain.auto.cf7e9e2519'), active: t('domain.auto.d4e9ca3dd4') }[value] || value || '-') }
function statusType(value) { return ({ on_sale: 'success', testing: 'warning', stopped: 'info', active: 'success' }[value] || 'info') }
function formatAmount(amount, currency = defaultBusinessCurrency()) { return formatCurrencyAmount(amount, normalizeCurrencyCode(currency, defaultBusinessCurrency())) }
function formatTime(value) { if (!value) return '-'; const date = new Date(value); return Number.isNaN(date.getTime()) ? String(value).slice(0, 19) : `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}` }
</script>
