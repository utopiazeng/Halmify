<template>
  <div class="domain-page runtime-page">
    <div class="domain-toolbar">
      <el-form :inline="true" class="toolbar-form">
        <el-form-item>
          <el-button type="primary" @click="load">
            <Icon icon="ri:refresh-line" />
            <span style="margin-left: 6px;">{{ t('common.action.refresh') }}</span>
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="runtime-kpi-grid">
      <div class="runtime-kpi-card">
        <div class="runtime-kpi-icon"><Icon icon="ri:server-line" /></div>
        <div class="runtime-kpi-meta">
          <p>{{ t('platform.runtime.backendNodes') }}</p>
          <strong>{{ summary.backend_nodes || 0 }}</strong>
        </div>
      </div>
      <div class="runtime-kpi-card">
        <div class="runtime-kpi-icon ok"><Icon icon="ri:checkbox-circle-line" /></div>
        <div class="runtime-kpi-meta">
          <p>{{ t('platform.runtime.backendHealthy') }}</p>
          <strong>{{ summary.backend_healthy_nodes || 0 }}</strong>
        </div>
      </div>
      <div class="runtime-kpi-card">
        <div class="runtime-kpi-icon"><Icon icon="ri:hard-drive-2-line" /></div>
        <div class="runtime-kpi-meta">
          <p>{{ t('platform.runtime.fileserverTotal') }}</p>
          <strong>{{ summary.fileserver_total || 0 }}</strong>
        </div>
      </div>
      <div class="runtime-kpi-card">
        <div class="runtime-kpi-icon ok"><Icon icon="ri:shield-check-line" /></div>
        <div class="runtime-kpi-meta">
          <p>{{ t('platform.runtime.fileserverHealthy') }}</p>
          <strong>{{ summary.fileserver_healthy || 0 }}</strong>
        </div>
      </div>
      <div class="runtime-kpi-card">
        <div class="runtime-kpi-icon"><Icon icon="ri:database-2-line" /></div>
        <div class="runtime-kpi-meta">
          <p>{{ t('platform.runtime.dependencyTotal') }}</p>
          <strong>{{ summary.dependency_total || 0 }}</strong>
        </div>
      </div>
      <div class="runtime-kpi-card">
        <div class="runtime-kpi-icon ok"><Icon icon="ri:heart-pulse-line" /></div>
        <div class="runtime-kpi-meta">
          <p>{{ t('platform.runtime.dependencyHealthy') }}</p>
          <strong>{{ summary.dependency_healthy || 0 }}</strong>
        </div>
      </div>
    </div>

    <el-card class="system-card runtime-card" style="margin-top: 12px;">
      <template #header>{{ t('platform.runtime.backendNodeTable') }}</template>
      <div v-loading="loading" class="runtime-server-grid">
        <div v-for="node in backendNodes" :key="node.host" class="runtime-server-item">
          <div class="runtime-server-head">
            <strong>{{ node.host || '-' }}</strong>
            <span>{{ uptimeText(node.uptime_seconds) }}</span>
          </div>
          <div class="runtime-server-body">
            <div class="runtime-server-icon-stack">
              <div class="runtime-server-unit"><i></i><em></em><b></b></div>
              <div class="runtime-server-unit"><i></i><em></em><b></b></div>
              <div class="runtime-server-unit"><i></i><em></em><b></b></div>
            </div>
            <div class="runtime-server-metrics">
              <div class="runtime-metric">
                <div class="runtime-metric-title"><Icon icon="ri:cpu-line" /> {{ t('platform.runtime.cpu') }}</div>
                <el-progress :percentage="safePercent(node.cpu_usage_percent)" :show-text="true" :stroke-width="14" />
              </div>
              <div class="runtime-metric">
                <div class="runtime-metric-title"><Icon icon="ri:database-2-line" /> {{ t('platform.runtime.memory') }}</div>
                <el-progress :percentage="safePercent(node.memory_used_percent)" :show-text="true" :stroke-width="14" status="success" />
              </div>
              <div class="runtime-metric">
                <div class="runtime-metric-title"><Icon icon="ri:hard-drive-3-line" /> {{ t('platform.runtime.disk') }}</div>
                <el-progress :percentage="safePercent(node.disk_used_percent)" :show-text="true" :stroke-width="14" status="warning" />
              </div>
              <div class="runtime-extra">
                <span><Icon icon="ri:threads-line" /> Goroutines: {{ Number(node.goroutines || 0) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <el-card class="system-card runtime-card" style="margin-top: 12px;">
      <template #header>{{ t('platform.runtime.dependencyNodeTable') }}</template>
      <div v-loading="loading" class="runtime-server-grid">
        <div v-for="node in dependencyNodes" :key="node.type" class="runtime-server-item dependency-node-item">
          <div class="runtime-server-head">
            <strong>{{ node.name || node.type }}</strong>
            <el-tag :type="node.healthy ? 'success' : 'danger'" effect="light">
              {{ node.healthy ? t('common.status.enabled') : t('common.status.disabled') }}
            </el-tag>
          </div>
          <div class="runtime-server-body">
            <div class="runtime-server-icon-stack dependency">
              <div class="runtime-server-unit"><i></i><em></em><b></b></div>
              <div class="runtime-server-unit"><i></i><em></em><b></b></div>
            </div>
            <div class="runtime-server-metrics">
              <div class="runtime-extra">
                <span><Icon icon="ri:map-pin-line" /> {{ node.address || '-' }}</span>
                <span><Icon icon="ri:timer-line" /> {{ t('platform.runtime.latency') }}: {{ formatLatency(node.latency_ms) }}</span>
                <span v-if="node.version"><Icon icon="ri:information-line" /> {{ node.version }}</span>
                <span v-if="node.database"><Icon icon="ri:database-line" /> {{ node.database }}</span>
              </div>
              <div class="runtime-reasons" :class="{ muted: node.healthy }">
                <Icon :icon="node.healthy ? 'ri:checkbox-circle-line' : 'ri:alert-line'" />
                <span>{{ node.error_message || '-' }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <el-card class="system-card runtime-card" style="margin-top: 12px;">
      <template #header>{{ t('platform.runtime.fileserverNodeTable') }}</template>
      <div v-loading="loading" class="runtime-server-grid">
        <div v-for="node in fileserverNodes" :key="node.id" class="runtime-server-item file-node-item">
          <div class="runtime-server-head">
            <strong>{{ node.base_url || node.id }}</strong>
            <el-tag :type="node.healthy ? 'success' : 'danger'" effect="light">
              {{ node.healthy ? t('common.status.enabled') : t('common.status.disabled') }}
            </el-tag>
          </div>
          <div class="runtime-server-body">
            <div class="runtime-server-icon-stack file">
              <div class="runtime-server-unit"><i></i><em></em><b></b></div>
              <div class="runtime-server-unit"><i></i><em></em><b></b></div>
            </div>
            <div class="runtime-server-metrics">
              <div class="runtime-metric">
                <div class="runtime-metric-title"><Icon icon="ri:pie-chart-line" /> {{ t('platform.tenants.usagePercent') }}</div>
                <el-progress :percentage="safePercent(node.storage_percent)" :show-text="true" :stroke-width="14" :status="safePercent(node.storage_percent) > 85 ? 'exception' : 'success'" />
              </div>
              <div class="runtime-extra">
                <span><Icon icon="ri:route-line" /> {{ t('platform.runtime.candidate') }}: {{ node.is_candidate ? 'Y' : 'N' }}</span>
                <span><Icon icon="ri:hard-drive-3-line" /> {{ t('platform.runtime.total') }}: {{ formatBytes(node.total_bytes) }}</span>
                <span><Icon icon="ri:archive-drawer-line" /> {{ t('platform.runtime.used') }}: {{ formatBytes(node.used_bytes) }}</span>
                <span><Icon icon="ri:database-2-line" /> {{ t('platform.runtime.free') }}: {{ formatBytes(node.free_bytes) }}</span>
                <span><Icon icon="ri:timer-line" /> {{ t('platform.runtime.latency') }}: {{ formatLatency(node.latency_ms) }}</span>
                <span><Icon icon="ri:timer-line" /> {{ t('platform.runtime.checkedAt') }}: {{ formatTime(node.last_checked_at) }}</span>
              </div>
              <div class="runtime-reasons" :class="{ muted: !(node.reasons || []).length }">
                <Icon :icon="(node.reasons || []).length ? 'ri:alert-line' : 'ri:checkbox-circle-line'" />
                <span>{{ fileNodeReasonText(node) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Icon } from '@iconify/vue'
import api from '../../api'

const loading = ref(false)
const summary = reactive({
  backend_nodes: 0,
  backend_healthy_nodes: 0,
  fileserver_total: 0,
  fileserver_healthy: 0,
  dependency_total: 0,
  dependency_healthy: 0
})
const backendNodes = ref([])
const fileserverNodes = ref([])
const dependencyNodes = ref([])

onMounted(load)

async function load() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/platform/runtime/overview')
    const data = res?.data || {}
    Object.assign(summary, data.summary || {})
    backendNodes.value = data.backend_nodes || []
    fileserverNodes.value = data.fileserver?.nodes || []
    dependencyNodes.value = data.dependencies || []
  } catch (e) {
    ElMessage.error(e?.response?.data?.message || e.message || t('common.message.loadFailed'))
  } finally {
    loading.value = false
  }
}

function safePercent(value) {
  const num = Number(value || 0)
  if (Number.isNaN(num)) return 0
  return Math.max(0, Math.min(100, Math.round(num * 100) / 100))
}

function formatBytes(value) {
  const num = Number(value || 0)
  if (!num) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let size = num
  let index = 0
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024
    index += 1
  }
  return `${Math.round(size * 100) / 100} ${units[index]}`
}

function formatTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  const pad = (n) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function fileNodeReasonText(node) {
  const reasons = node?.reasons || []
  if (!reasons.length) return t('platform.runtime.fileNodeNormal')
  return reasons.map((item) => ({
    unhealthy: t('platform.runtime.reasonUnhealthy'),
    stale_probe: t('platform.runtime.reasonStaleProbe'),
    below_min_free_bytes: t('platform.runtime.reasonBelowMinFreeBytes'),
    high_watermark_reached: t('platform.runtime.reasonHighWatermarkReached')
  }[item] || item)).join(', ')
}

function formatLatency(value) {
  const num = Number(value || 0)
  if (!num) return '0ms'
  return `${Math.round(num * 100) / 100}ms`
}

function uptimeText(value) {
  const total = Number(value || 0)
  if (!total) return `${t('platform.runtime.uptime')}: 0s`
  const day = Math.floor(total / 86400)
  const hour = Math.floor((total % 86400) / 3600)
  const minute = Math.floor((total % 3600) / 60)
  if (day > 0) return `${day}d ${hour}h ${minute}m`
  if (hour > 0) return `${hour}h ${minute}m`
  return `${minute}m`
}
</script>

<style scoped>
.runtime-page {
  gap: 12px;
}

.runtime-kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.runtime-kpi-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  border-radius: 12px;
  border: 1px solid #e8edf6;
  background: #fff;
}

.runtime-kpi-icon {
  width: 42px;
  height: 42px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  background: #edf4ff;
  color: #3569e8;
}

.runtime-kpi-icon.ok {
  background: #eafbf3;
  color: #11a468;
}

.runtime-kpi-icon :deep(svg) {
  width: 22px;
  height: 22px;
}

.runtime-kpi-meta p {
  margin: 0;
  color: #64748b;
  font-size: 13px;
}

.runtime-kpi-meta strong {
  color: #1e293b;
  font-size: 22px;
}

.runtime-card {
  padding-bottom: 8px;
}

.runtime-server-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(460px, 1fr));
  gap: 12px;
}

.runtime-server-item {
  border: 1px solid #e8edf6;
  border-radius: 12px;
  padding: 12px;
}

.runtime-server-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.runtime-server-head strong {
  color: #1f2937;
  font-size: 14px;
}

.runtime-server-head span {
  color: #7f8ea3;
  font-size: 12px;
}

.runtime-server-body {
  display: grid;
  grid-template-columns: 146px minmax(0, 1fr);
  gap: 12px;
  align-items: center;
}

.runtime-server-icon-stack {
  display: grid;
  gap: 6px;
}

.runtime-server-unit {
  height: 34px;
  border-radius: 7px;
  background: #3f4651;
  position: relative;
}

.runtime-server-unit i {
  position: absolute;
  left: 12px;
  top: 12px;
  width: 10px;
  height: 10px;
  border-radius: 2px;
  background: #dce1e8;
}

.runtime-server-unit em,
.runtime-server-unit b {
  position: absolute;
  top: 13px;
  width: 7px;
  height: 7px;
  border-radius: 1px;
}

.runtime-server-unit em {
  right: 29px;
  background: #34d399;
}

.runtime-server-unit b {
  right: 17px;
  background: #3b82f6;
}

.runtime-server-metrics {
  display: grid;
  gap: 8px;
}

.runtime-metric-title {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: #374151;
  font-size: 12px;
  margin-bottom: 2px;
}

.runtime-metric-title :deep(svg),
.runtime-extra :deep(svg),
.runtime-reasons :deep(svg) {
  width: 14px;
  height: 14px;
}

.runtime-extra {
  color: #64748b;
  font-size: 12px;
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
}

.runtime-reasons {
  display: inline-flex;
  gap: 5px;
  align-items: flex-start;
  color: #ef4444;
  font-size: 12px;
}

.runtime-reasons.muted {
  color: #16a34a;
}

.file-node-item .runtime-server-body {
  grid-template-columns: 110px minmax(0, 1fr);
}

.dependency-node-item .runtime-server-body {
  grid-template-columns: 110px minmax(0, 1fr);
}

@media (max-width: 1100px) {
  .runtime-server-grid {
    grid-template-columns: 1fr;
  }

  .runtime-server-body {
    grid-template-columns: 1fr;
  }
}
</style>
