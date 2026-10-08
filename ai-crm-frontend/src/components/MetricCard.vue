<template>
  <div class="crm-metric-card" :class="[`tone-${tone || 'blue'}`, { compact }]">
    <span class="crm-metric-icon">
      <Icon :icon="icon || 'ri:bar-chart-box-line'" />
    </span>
    <div class="crm-metric-main">
      <span class="crm-metric-title">{{ title }}</span>
      <strong class="crm-metric-value">{{ value }}</strong>
      <small v-if="help || caption || description" class="crm-metric-help">{{ help || caption || description }}</small>
    </div>
    <el-tag v-if="trend" class="crm-metric-trend" size="small" effect="plain" :type="status || ''">{{ trend }}</el-tag>
  </div>
</template>

<script setup>
import { t } from '../i18n'
import { Icon } from '@iconify/vue'

defineProps({
  title: { type: String, required: true },
  value: { type: [String, Number], required: true },
  icon: { type: String, default: 'ri:bar-chart-box-line' },
  help: { type: String, default: '' },
  caption: { type: String, default: '' },
  description: { type: String, default: '' },
  trend: { type: String, default: '' },
  status: { type: String, default: '' },
  tone: { type: String, default: 'blue' },
  compact: { type: Boolean, default: false }
})
</script>
