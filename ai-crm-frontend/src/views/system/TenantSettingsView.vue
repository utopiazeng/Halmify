<template>
  <div class="tenant-settings-page">
    <section class="tenant-settings-hero">
      <div>
        <h2>{{ t('tenantSettings.title') }}</h2>
        <p>{{ t('tenantSettings.desc') }}</p>
      </div>
      <el-tag effect="light">{{ settings.currency }}</el-tag>
    </section>

    <section class="tenant-settings-panel">
      <el-form :model="settings" label-position="top" class="tenant-settings-form">
        <el-form-item :label="t('tenantSettings.defaultCurrency')">
          <el-select v-model="settings.currency" filterable class="tenant-settings-select">
            <el-option
              v-for="item in currencyOptions"
              :key="item.value"
              :label="currencyOptionLabel(item)"
              :value="item.value"
            />
          </el-select>
          <p class="tenant-settings-help">{{ t('tenantSettings.defaultCurrencyHelp') }}</p>
        </el-form-item>

        <div class="tenant-settings-current">
          <span>{{ t('tenantSettings.currentCurrency') }}</span>
          <strong>{{ currentCurrencyLabel }}</strong>
        </div>

        <div class="tenant-settings-note">
          <Icon icon="ri:information-line" />
          <div>
            <strong>{{ t('tenantSettings.billingExcluded') }}</strong>
            <p>{{ t('tenantSettings.billingExcludedDesc') }}</p>
          </div>
        </div>

        <el-button type="primary" :loading="saving" :disabled="!canManageTenantSettings" @click="saveSettings">
          {{ t('tenantSettings.save') }}
        </el-button>
      </el-form>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { Icon } from '@iconify/vue'
import { ElMessage } from 'element-plus'
import { t } from '../../i18n'
import { hasPermission } from '../../permissions'
import { currencyOptions, currencySymbols } from '../../utils/currency'
import { DEFAULT_COMPANY_SETTINGS, fetchCompanySettings, saveCompanySettings } from '../../utils/company-format'

const settings = reactive({ ...DEFAULT_COMPANY_SETTINGS })
const saving = ref(false)
const canManageTenantSettings = computed(() => hasPermission('enterprise:manage'))

const currentCurrencyLabel = computed(() => currencyOptionLabel({ value: settings.currency, label: settings.currency }))

onMounted(loadSettings)

function currencyOptionLabel(item) {
  const code = item?.value || DEFAULT_COMPANY_SETTINGS.currency
  const symbol = currencySymbols[code] || code
  return `${symbol} ${item?.label || code}`
}

async function loadSettings() {
  try {
    Object.assign(settings, await fetchCompanySettings())
  } catch {
    ElMessage.error(t('tenantSettings.loadFailed'))
  }
}

async function saveSettings() {
  if (!canManageTenantSettings.value) return
  saving.value = true
  try {
    Object.assign(settings, await saveCompanySettings({ currency: settings.currency }))
    ElMessage.success(t('tenantSettings.saved'))
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || t('tenantSettings.saveFailed'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.tenant-settings-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.tenant-settings-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 20px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.tenant-settings-hero h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: #111827;
}

.tenant-settings-hero p {
  margin: 8px 0 0;
  color: #6b7280;
  line-height: 1.6;
}

.tenant-settings-panel {
  padding: 20px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.tenant-settings-form {
  max-width: 620px;
}

.tenant-settings-select {
  width: 280px;
  max-width: 100%;
}

.tenant-settings-help {
  margin: 8px 0 0;
  color: #6b7280;
  font-size: 13px;
  line-height: 1.6;
}

.tenant-settings-current,
.tenant-settings-note {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 18px;
}

.tenant-settings-current span {
  color: #6b7280;
}

.tenant-settings-current strong {
  color: #111827;
}

.tenant-settings-note {
  align-items: flex-start;
  padding: 14px;
  border: 1px solid #dbeafe;
  border-radius: 8px;
  background: #eff6ff;
  color: #1d4ed8;
}

.tenant-settings-note svg {
  width: 20px;
  height: 20px;
  margin-top: 2px;
  flex: 0 0 auto;
}

.tenant-settings-note strong {
  display: block;
  color: #1e3a8a;
}

.tenant-settings-note p {
  margin: 4px 0 0;
  color: #1d4ed8;
  line-height: 1.6;
}
</style>
