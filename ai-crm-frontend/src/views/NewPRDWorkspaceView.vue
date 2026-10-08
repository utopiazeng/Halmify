<template>
  <div class="prd-page">
    <section class="workspace-hero">
      <div>
        <p class="eyebrow">NEW PRD CRM</p>
        <h2>{{ activeDomainName }}</h2>
        <p>{{ activeDomainDesc }}</p>
      </div>
      <div class="hero-actions">
        <el-button @click="sideTab = 'map'">{{ wt('featureMap') }}</el-button>
        <el-button @click="refreshAll">{{ t('common.action.refresh') }}</el-button>
        <el-button @click="exportObjects">{{ t('common.action.export') }}</el-button>
        <el-button type="primary" @click="startCreate">{{ wt('newRecord') }}</el-button>
      </div>
    </section>

    <section v-if="isWorkspaceDomain" class="kpi-grid">
      <ArtStatsCard
        v-for="item in cockpitCards"
        :key="item.key"
        :title="item.title"
        :value="item.value"
      />
    </section>

    <section v-else class="module-summary-grid">
      <ArtStatsCard
        v-for="item in moduleCards"
        :key="item.key"
        :title="item.title"
        :value="item.value"
        :description="item.description"
      />
    </section>

    <section v-if="isWorkspaceDomain" class="domain-strip">
      <button
        v-for="domain in domains"
        :key="domain.key"
        class="domain-pill"
        :class="{ active: activeDomain === domain.key }"
        @click="selectDomain(domain.key)"
      >
        <span>{{ domainDisplayName(domain) }}</span>
        <strong>{{ summaryCount(domain.key) }}</strong>
      </button>
    </section>

    <section v-if="isWorkspaceDomain" class="insight-grid">
      <div class="mini-panel">
        <div class="mini-panel-head">
          <h3>{{ wt('todayTodos') }}</h3>
          <el-button size="small" @click="selectDomain('workspace')">{{ wt('view') }}</el-button>
        </div>
        <div class="art-list">
          <div v-for="item in cockpit.tasks.slice(0, 4)" :key="item.id" class="art-list-item">
            <span>{{ item.name }}</span>
            <el-button size="small" link @click.stop="completeTask(item)">{{ wt('complete') }}</el-button>
          </div>
        </div>
        <el-empty v-if="!cockpit.tasks.length" :description="wt('noTodo')" />
      </div>

      <div class="mini-panel">
        <div class="mini-panel-head">
          <h3>{{ wt('recentActivities') }}</h3>
          <el-tag>{{ cockpit.activities.length }}</el-tag>
        </div>
        <el-timeline class="compact-timeline">
          <el-timeline-item v-for="item in cockpit.activities.slice(0, 5)" :key="`${item.domain}-${item.id}`">
            <button class="timeline-link" @click="selectDomain(item.domain)">{{ item.name }}</button>
            <span>{{ item.domain }} · {{ shortDate(item.updated_at) }}</span>
          </el-timeline-item>
        </el-timeline>
      </div>

      <div class="mini-panel">
        <div class="mini-panel-head">
          <h3>{{ wt('prdCoverage') }}</h3>
          <el-space size="small">
            <el-tag type="success">{{ activeCoverage.length }} {{ wt('items') }}</el-tag>
            <el-tag type="primary">{{ activeActions.length }} {{ wt('actions') }}</el-tag>
          </el-space>
        </div>
        <div class="coverage-list">
          <div v-for="item in activeCoverage.slice(0, 6)" :key="item.code" class="coverage-row">
            <span>{{ item.code }}</span>
            <strong>{{ item.section }}</strong>
            <el-tag type="primary">{{ item.object_type }}</el-tag>
          </div>
        </div>
      </div>
    </section>

    <section v-if="isWorkspaceDomain" class="process-lane">
      <button
        v-for="step in processSteps"
        :key="step.domain"
        class="process-step"
        :class="{ active: activeDomain === step.domain }"
        @click="selectDomain(step.domain)"
      >
        <span>{{ step.index }}</span>
        <strong>{{ step.title }}</strong>
        <small>{{ step.caption }}</small>
      </button>
    </section>

    <section v-if="activeDomain === 'dashboards'" class="big-screen" :class="{ fullscreen: bigScreenMode }">
      <div class="big-screen-head">
        <div>
          <p class="eyebrow">SALES COMMAND CENTER</p>
          <h3>{{ wt('salesRealtimeScreen') }}</h3>
        </div>
        <el-button @click="bigScreenMode = !bigScreenMode">{{ bigScreenMode ? t('common.action.exitFullscreen') : t('common.action.fullscreen') }}</el-button>
      </div>
      <div class="chart-grid">
        <div class="chart-panel">
          <h4>{{ wt('salesFunnel') }}</h4>
          <div class="funnel-chart">
            <div v-for="item in cockpit.charts.funnel" :key="item.stage" class="funnel-row">
              <span>{{ item.stage }}</span>
              <div><i :style="{ width: chartWidth(item.count, maxFunnelCount) }"></i></div>
              <strong>{{ item.count }}</strong>
            </div>
          </div>
        </div>
        <div class="chart-panel">
          <h4>{{ wt('dealTrend') }}</h4>
          <div class="bar-chart">
            <div v-for="item in cockpit.charts.trend" :key="item.month" class="bar-item">
              <i :style="{ height: chartWidth(item.amount, maxTrendAmount) }"></i>
              <span>{{ item.month.slice(5) }}</span>
            </div>
          </div>
        </div>
        <div class="chart-panel">
          <h4>{{ wt('teamLeaderboard') }}</h4>
          <div class="leaderboard">
            <div v-for="item in cockpit.charts.leaderboard" :key="item.owner_id" class="leader-row">
              <span>{{ item.name }}</span>
              <div><i :style="{ width: chartWidth(item.amount, maxLeaderAmount) }"></i></div>
              <strong>{{ formatAmount(item.amount) }}</strong>
            </div>
          </div>
        </div>
        <div class="chart-panel">
          <h4>{{ wt('targetProgress') }}</h4>
          <el-progress type="circle" :percentage="targetPercent" />
          <p class="target-copy">{{ formatAmount(cockpit.charts.target.actual || 0) }} / {{ formatAmount(cockpit.charts.target.target || 0) }}</p>
        </div>
      </div>
    </section>

    <section v-if="activeDomain !== 'dashboards'" class="kanban-board">
      <div v-for="column in kanbanColumns" :key="column.status" class="kanban-column">
        <div class="kanban-head">
          <strong>{{ column.status }}</strong>
          <el-tag>{{ column.items.length }}</el-tag>
        </div>
        <button v-for="item in column.items" :key="item.id" class="kanban-card" @click="editObject(item)">
          <strong>{{ item.name }}</strong>
          <span>{{ item.object_type }} · {{ money(item) }}</span>
        </button>
      </div>
    </section>

    <section v-if="activeDomain === 'projects'" class="customer-space">
      <div class="mini-panel-head">
        <h3>{{ wt('customerSpace') }}</h3>
        <el-tag type="primary">{{ customerSpaceItems.length }} {{ wt('itemsVisible') }}</el-tag>
      </div>
      <div class="space-grid">
        <div v-for="item in customerSpaceItems" :key="item.id" class="space-card">
          <strong>{{ item.name }}</strong>
          <span>{{ item.object_type }} · {{ item.status }}</span>
          <p>{{ displayValue(parsePayload(item.payload_json).acceptance_criteria || parsePayload(item.payload_json).share_url || parsePayload(item.payload_json).progress) }}</p>
        </div>
      </div>
    </section>

    <section class="work-grid">
      <main class="data-panel">
        <div class="panel-toolbar">
          <div class="toolbar-left">
            <el-select v-model="filters.object_type" clearable :placeholder="wt('type')" class="type-filter">
              <el-option v-for="type in activeObjectTypes" :key="type" :value="type">{{ type }}</el-option>
            </el-select>
            <el-input v-model="filters.keyword" :placeholder="wt('searchNameCodeTags')" class="keyword-filter" @keyup.enter="loadObjects" />
            <el-button @click="loadObjects">{{ t('common.action.search') }}</el-button>
          </div>
          <el-tag>{{ objects.length }} {{ wt('records') }}</el-tag>
        </div>

        <el-table
          :data="objects"
          row-key="id"
          size="default"
          class="crm-table"
          :row-class-name="rowClassName"
          @row-click="editObject"
        >
          <el-table-column prop="name" :label="wt('name')" min-width="180">
            <template #default="{ row }">
              <div class="record-title">
                <strong>{{ row.name }}</strong>
                <span>{{ row.code || row.object_type }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="object_type" :label="wt('type')" width="150" />
          <el-table-column prop="status" :label="t('common.table.status')" width="120">
            <template #default="{ row }">
              <el-tag :type="statusType(row.status)">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="wt('amount')" width="150">
            <template #default="{ row }">{{ money(row) }}</template>
          </el-table-column>
          <el-table-column :label="wt('score')" width="110">
            <template #default="{ row }">
              <el-progress :percentage="Number(row.score || 0)" :show-text="false" />
              <span class="score-text">{{ row.score || 0 }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="wt('updated')" width="140">
            <template #default="{ row }">{{ shortDate(row.updated_at) }}</template>
          </el-table-column>
        </el-table>
      </main>

      <aside class="side-panel">
        <el-tabs v-model="sideTab" class="art-tabs">
          <el-tab-pane name="edit" :label="editingId ? t('common.action.edit') : t('common.action.create')">
            <el-form label-position="top" :model="form" @submit.prevent="saveObject">
              <el-form-item :label="wt('objectType')">
                <el-select v-model="form.object_type" @change="applyObjectTypeDefaults">
                  <el-option v-for="type in activeObjectTypes" :key="type" :value="type">{{ type }}</el-option>
                </el-select>
              </el-form-item>
              <el-form-item :label="wt('name')" required>
                <el-input v-model="form.name" :placeholder="namePlaceholder" />
              </el-form-item>
              <el-row :gutter="8">
                <el-col :span="12">
                  <el-form-item :label="wt('code')"><el-input v-model="form.code" /></el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item :label="t('common.table.status')">
                    <el-select v-model="form.status">
                      <el-option v-for="status in activeStatuses" :key="status" :value="status">{{ status }}</el-option>
                    </el-select>
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="8">
                <el-col :span="12">
                  <el-form-item :label="wt('amount')"><el-input-number v-model="form.amount" class="full-input" /></el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item :label="wt('score')"><el-input-number v-model="form.score" :min="0" :max="100" class="full-input" /></el-form-item>
                </el-col>
              </el-row>
              <el-form-item :label="wt('tags')">
                <el-input v-model="tagText" :placeholder="wt('commaSeparated')" />
              </el-form-item>
              <div v-if="schemaFields.length" class="schema-form">
                <div class="schema-form-head">
                  <strong>{{ wt('businessFields') }}</strong>
                  <el-tag>{{ activeSchema?.title }}</el-tag>
                </div>
                <el-row :gutter="8">
                  <el-col v-for="field in schemaFields" :key="field.key" :span="12">
                    <el-form-item :label="field.label" :required="field.required">
                      <el-input-number
                        v-if="field.type === 'number'"
                        v-model="fieldDraft[field.key]"
                        class="full-input"
                        @change="syncFieldDraftToPayload"
                      />
                      <el-checkbox
                        v-else-if="field.type === 'boolean'"
                        v-model="fieldDraft[field.key]"
                        @change="syncFieldDraftToPayload"
                      >
                        {{ t('common.status.enabled') }}
                      </el-checkbox>
                      <el-input type="textarea"
                        v-else-if="field.type === 'json'"
                        v-model="fieldDraft[field.key]"
                        :rows="3"
                        @change="syncFieldDraftToPayload"
                      />
                      <el-input v-else v-model="fieldDraft[field.key]" @change="syncFieldDraftToPayload" />
                    </el-form-item>
                  </el-col>
                </el-row>
              </div>
              <el-form-item :label="wt('extensionJson')">
                <el-input type="textarea" v-model="payloadText" :rows="8" />
                <el-button size="small" class="inline-action" @click="fillTemplate">{{ wt('fillTemplate') }}</el-button>
              </el-form-item>
              <div class="form-actions">
                <el-button type="primary" native-type="submit" :loading="saving">{{ editingId ? t('common.action.save') : t('common.action.create') }}</el-button>
                <el-button @click="resetForm">{{ wt('clear') }}</el-button>
                <el-button v-if="editingId" type="danger" @click="deleteObject(editingId)">{{ t('common.action.delete') }}</el-button>
              </div>
            </el-form>
          </el-tab-pane>

          <el-tab-pane name="detail" :label="wt('detail')">
            <div v-if="selectedRecord" class="detail-panel">
              <div class="detail-head">
                <div>
                  <strong>{{ selectedRecord.name }}</strong>
                  <span>{{ selectedRecord.object_type }} · {{ selectedRecord.code || selectedRecord.id }}</span>
                </div>
                <el-tag :type="statusType(selectedRecord.status)">{{ selectedRecord.status }}</el-tag>
              </div>
              <div class="detail-metrics">
                <div>
                  <span>{{ wt('amount') }}</span>
                  <strong>{{ money(selectedRecord) }}</strong>
                </div>
                <div>
                  <span>{{ wt('score') }}</span>
                  <strong>{{ selectedRecord.score || 0 }}</strong>
                </div>
                <div>
                  <span>{{ wt('owner') }}</span>
                  <strong>#{{ selectedRecord.owner_id || 0 }}</strong>
                </div>
              </div>
              <div class="detail-fields">
                <div v-for="item in selectedPayloadEntries" :key="item.key" class="detail-field">
                  <span>{{ item.key }}</span>
                  <strong>{{ item.value }}</strong>
                </div>
              </div>
              <el-empty v-if="!selectedPayloadEntries.length" :description="wt('noExtendedFields')" />
              <el-divider />
              <h3 class="panel-title">{{ wt('relatedTimeline') }}</h3>
              <el-timeline class="compact-timeline">
                <el-timeline-item v-for="item in relations" :key="item.id">
                  <span>{{ item.relation_type }}</span>
                  <button class="timeline-link" @click="selectDomain(item.target_type)">
                    {{ item.source_type }}#{{ item.source_id }} -> {{ item.target_type }}#{{ item.target_id }}
                  </button>
                </el-timeline-item>
              </el-timeline>
              <el-empty v-if="!relations.length" :description="wt('noRelatedRecords')" />
            </div>
            <el-empty v-else :description="wt('selectARecord')" />
          </el-tab-pane>

          <el-tab-pane name="flow" :label="wt('flow')">
            <div class="selected-summary" v-if="selectedRecord">
              <strong>{{ selectedRecord.name }}</strong>
              <span>{{ selectedRecord.object_type }} · {{ selectedRecord.status }}</span>
            </div>
            <div class="quick-actions">
              <el-button
                v-for="action in executableActions"
                :key="action.key"
                class="block-button"
                @click="runDomainAction(action)"
              >
                {{ actionButtonTitle(action) }}
              </el-button>
              <el-button v-if="canStartApproval" class="block-button" @click="startApproval">{{ wt('startApproval') }}</el-button>
              <el-empty v-if="!executableActions.length && !canStartApproval" :description="wt('noQuickFlow')" />
            </div>

            <el-divider />
            <h3 class="panel-title">{{ wt('actionList') }}</h3>
            <div class="action-list">
              <div v-for="action in activeActions" :key="action.key" class="action-row">
                <strong>{{ action.title }}</strong>
                <span>{{ action.description }}</span>
                <code>{{ action.path }}</code>
              </div>
            </div>
            <el-empty v-if="!activeActions.length" :description="wt('noActionList')" />

            <el-divider />
            <h3 class="panel-title">{{ wt('approvalQueue') }}</h3>
            <div class="art-list">
              <div v-for="item in approvals.slice(0, 5)" :key="item.id" class="art-list-item approval-row">
                <div class="approval-item">
                  <strong>{{ item.title }}</strong>
                  <span>{{ currentApprovalNode(item)?.name || wt('currentNode') }} · {{ wt('nodeIndex', { n: item.current_node || 1 }) }}</span>
                </div>
                <div class="approval-actions">
                  <el-button size="small" link @click="actApproval(item, 'approve')">{{ wt('approve') }}</el-button>
                  <el-button size="small" link @click="actApproval(item, 'append')">{{ wt('append') }}</el-button>
                  <el-button size="small" link @click="actApproval(item, 'transfer')">{{ wt('transfer') }}</el-button>
                  <el-button size="small" link @click="actApproval(item, 'withdraw')">{{ wt('withdraw') }}</el-button>
                  <el-button size="small" link type="danger" @click="actApproval(item, 'reject')">{{ wt('reject') }}</el-button>
                </div>
              </div>
            </div>
            <el-empty v-if="!approvals.length" :description="wt('noPendingApprovals')" />

            <el-divider />
            <h3 class="panel-title">{{ wt('relatedObjects') }}</h3>
            <div class="art-list">
              <div v-for="item in relations" :key="item.id" class="art-list-item">
                <span>{{ item.source_type }}#{{ item.source_id }} -> {{ item.target_type }}#{{ item.target_id }}</span>
                <el-tag>{{ item.relation_type }}</el-tag>
              </div>
            </div>
          </el-tab-pane>

          <el-tab-pane name="settings" :label="wt('settings')">
            <el-form label-position="top" @submit.prevent="saveSetting">
              <el-form-item :label="wt('settingType')">
                <el-select v-model="settingForm.scope">
                  <el-option
                    v-for="item in settingScopeOptions"
                    :key="item.value"
                    :value="item.value"
                    :label="item.label"
                  />
                </el-select>
              </el-form-item>
              <el-form-item :label="wt('settingJson')">
                <el-input type="textarea" v-model="settingValueText" :rows="8" />
              </el-form-item>
              <div class="preset-row">
                <el-button v-for="preset in settingPresets" :key="preset.key" size="small" @click="preset.handler">
                  {{ preset.title }}
                </el-button>
              </div>
              <el-button v-if="settingForm.scope === 'integration'" class="block-button inline-action" @click="testIntegration">{{ wt('testIntegration') }}</el-button>
              <el-button v-if="settingForm.scope === 'integration'" class="block-button inline-action" @click="executeIntegration">{{ wt('executeIntegration') }}</el-button>
              <el-button v-if="settingForm.scope === 'integration' && ['hr', 'ldap'].includes(settingForm.key)" class="block-button inline-action" @click="syncHR">{{ wt('syncHR') }}</el-button>
              <el-button v-if="settingForm.scope === 'audit_policy'" class="block-button inline-action" @click="cleanupAuditLogs">{{ wt('cleanupAuditLogs') }}</el-button>
              <el-button type="primary" native-type="submit" class="block-button">{{ wt('saveSettings') }}</el-button>
              <el-alert v-if="settingSaveStatus" class="inline-action" :title="settingSaveStatus" type="success" show-icon :closable="false" />
            </el-form>
          </el-tab-pane>

          <el-tab-pane name="map" :label="wt('featureMap')">
            <div class="schema-card" v-if="activeSchema">
              <h3>{{ activeSchema.title }}</h3>
              <p>{{ activeSchema.object_type }}</p>
              <div class="schema-block">
                <span>{{ wt('requiredFields') }}</span>
                <el-tag v-for="field in activeSchema.required" :key="field">{{ field }}</el-tag>
              </div>
              <div class="schema-block">
                <span>{{ wt('acceptanceCriteria') }}</span>
                <ul>
                  <li v-for="item in activeSchema.acceptance" :key="item">{{ item }}</li>
                </ul>
              </div>
            </div>
            <div class="coverage-list full">
              <div v-for="item in activeCoverage" :key="item.code" class="coverage-row">
                <span>{{ item.code }}</span>
                <strong>{{ item.section }}</strong>
                <el-tag :type="item.implemented ? 'success' : 'warning'">{{ item.implemented ? wt('implemented') : wt('pending') }}</el-tag>
              </div>
            </div>
          </el-tab-pane>

          <el-tab-pane name="import" :label="t('common.action.import')">
            <el-form label-position="top" @submit.prevent="importObjects">
              <el-form-item label="JSON">
                <el-input type="textarea" v-model="importText" :rows="12" />
              </el-form-item>
              <el-button type="primary" native-type="submit" class="block-button">{{ wt('batchImport') }}</el-button>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </aside>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage as message } from 'element-plus'
import api from '../api'
import ArtStatsCard from '../components/art/ArtStatsCard.vue'
import { i18nState, t } from '../i18n'
import { downloadProtectedFile } from '../utils/download'
import { DEFAULT_COMPANY_SETTINGS, fetchCompanySettings } from '../utils/company-format'
import { DEFAULT_CURRENCY, formatCurrencyAmount, normalizeCurrencyCode } from '../utils/currency'
import { apiErrorMessage } from '../utils/api-error'

const route = useRoute()
const router = useRouter()
const domains = ref([])
const coverage = ref([])
const schemas = ref([])
const actions = ref([])
const summary = ref([])
const objects = ref([])
const approvals = ref([])
const cockpit = reactive({ tasks: [], activities: [], kpi: {}, charts: { funnel: [], trend: [], leaderboard: [], target: {} } })
const companySettings = reactive({ ...DEFAULT_COMPANY_SETTINGS })
const activeDomain = ref(route.meta.domain || route.params.domain || 'workspace')
const bigScreenMode = ref(false)
const editingId = ref(null)
const selectedRecord = ref(null)
const saving = ref(false)
const settingSaveStatus = ref('')
const relations = ref([])
const sideTab = ref('edit')
const tagText = ref('')
const payloadText = ref('{}')
const importText = ref('{"items":[]}')
const settingValueText = ref('{}')
const filters = reactive({ object_type: undefined, keyword: '' })
const form = reactive(defaultForm())
const fieldDraft = reactive({})
const settingForm = reactive({ scope: 'company', key: activeDomain.value })

const isWorkspaceDomain = computed(() => activeDomain.value === 'workspace')
const activeDomainMeta = computed(() => domains.value.find((item) => item.key === activeDomain.value) || {})
const routeObjectType = computed(() => route.meta.objectType || route.query.object_type || '')
const activeDomainName = computed(() => {
  if (routeObjectType.value && route.meta.titleKey) return t(route.meta.titleKey)
  return domainDisplayName(activeDomainMeta.value) || t('domain.workspace.domain.workspace.title')
})
const activeDomainDesc = computed(() => {
  if (route.meta.descKey) return t(route.meta.descKey)
  if (activeDomain.value) return t(`domain.workspace.domain.${activeDomain.value}.desc`)
  if (activeDomainMeta.value?.description) return activeDomainMeta.value.description
  return t('app.defaultDescription')
})
const activeObjectTypes = computed(() => activeDomainMeta.value.object_types || [])
const activeStatuses = computed(() => activeDomainMeta.value.statuses || ['active', 'draft'])
const activeCoverage = computed(() => coverage.value.filter((item) => item.domain === activeDomain.value))
const activeActions = computed(() => actions.value.filter((item) => item.domain === activeDomain.value))
const executableActions = computed(() => activeActions.value.filter((item) => actionHandlers[item.key] || item.key.startsWith('workspace_')))
const settingScopeOptions = computed(() => {
  const map = {
    company: ['enterprise', 'workspace'],
    sales_process: ['enterprise', 'opportunities'],
    data_scope: ['enterprise', 'accounts', 'teams'],
    field_acl: ['enterprise', 'accounts', 'contacts'],
    permission_matrix: ['accounts', 'teams', 'enterprise'],
    sla_rule: ['tickets'],
    integration: ['enterprise', 'accounts'],
    audit_policy: ['accounts', 'enterprise']
  }
  const labels = {
    company: wt('scopeCompany'),
    sales_process: wt('scopeSalesProcess'),
    data_scope: wt('scopeDataScope'),
    field_acl: wt('scopeFieldAcl'),
    permission_matrix: wt('scopePermissionMatrix'),
    sla_rule: wt('scopeSlaRule'),
    integration: wt('scopeIntegration'),
    audit_policy: wt('scopeAuditPolicy')
  }
  const allowed = Object.entries(map)
    .filter(([, domains]) => domains.includes(activeDomain.value))
    .map(([value]) => ({ value, label: labels[value] }))
  return allowed.length ? allowed : [{ value: 'company', label: wt('moduleSettings') }]
})
const settingPresets = computed(() => [
  { key: 'stage_gate', title: wt('stageGate'), domains: ['enterprise', 'opportunities'], handler: presetStageGate },
  { key: 'hidden_fields', title: wt('presetHiddenFields'), domains: ['enterprise', 'accounts', 'contacts'], handler: presetHiddenFields },
  { key: 'owner_scope', title: wt('presetOwnerScope'), domains: ['enterprise', 'accounts', 'teams'], handler: presetOwnerScope },
  { key: 'permission_matrix', title: wt('presetPermissionMatrix'), domains: ['accounts', 'teams', 'enterprise'], handler: presetPermissionMatrix },
  { key: 'sla', title: 'SLA', domains: ['tickets'], handler: presetSLARule },
  { key: 'integration', title: wt('integrationMode'), domains: ['enterprise', 'accounts'], handler: presetIntegration },
  { key: 'audit', title: wt('presetAudit'), domains: ['accounts', 'enterprise'], handler: presetAuditPolicy }
].filter((item) => item.domains.includes(activeDomain.value)))
const activeSchema = computed(() => schemas.value.find((item) => item.domain === activeDomain.value && item.object_type === form.object_type) || schemas.value.find((item) => item.domain === activeDomain.value))
const schemaFields = computed(() => {
  const schema = activeSchema.value
  if (!schema) return []
  return [
    ...(schema.required || []).map((key) => fieldSpec(key, true, schema)),
    ...(schema.optional || []).map((key) => fieldSpec(key, false, schema))
  ].slice(0, 12)
})
const namePlaceholder = computed(() => {
  if (activeSchema.value?.title) return wf('inputSchemaName', { title: activeSchema.value.title })
  return wt('inputBusinessName')
})
const selectedPayloadEntries = computed(() => Object.entries(parsePayload(selectedRecord.value?.payload_json)).slice(0, 12).map(([key, value]) => ({ key, value: displayValue(value) })))
const approvalObjectTypes = {
  orders: ['order', 'order_approval'],
  projects: ['deliverable'],
  products: ['pricing_rule']
}
const approvalTarget = computed(() => {
  const types = approvalObjectTypes[activeDomain.value] || []
  if (!types.length) return null
  if (selectedRecord.value && types.includes(selectedRecord.value.object_type)) return selectedRecord.value
  return objects.value.find((item) => types.includes(item.object_type)) || null
})
const canStartApproval = computed(() => {
  return Boolean(approvalTarget.value)
})
const cockpitCards = computed(() => [
  { key: 'leads', title: wt('leads'), value: cockpit.kpi.leads || 0 },
  { key: 'opportunities', title: wt('opportunities'), value: cockpit.kpi.opportunities || 0 },
  { key: 'orders', title: wt('orders'), value: cockpit.kpi.orders || 0 },
  { key: 'tickets', title: wt('tickets'), value: cockpit.kpi.tickets || 0 },
  { key: 'projects', title: wt('projects'), value: cockpit.kpi.projects || 0 },
  { key: 'won_amount', title: wt('wonAmount'), value: formatAmount(cockpit.kpi.won_amount || 0) }
])
const moduleCards = computed(() => [
  { key: 'records', title: wt('moduleRecords'), value: objects.value.length, description: activeDomainName.value },
  { key: 'types', title: wt('objectTypes'), value: activeObjectTypes.value.length, description: activeObjectTypes.value.slice(0, 2).join(' / ') },
  { key: 'actions', title: wt('availableActions'), value: activeActions.value.length, description: executableActions.value.slice(0, 2).map((item) => item.title).join(' / ') || wt('configImportExport') },
  { key: 'coverage', title: wt('prdCoverage'), value: activeCoverage.value.length, description: activeCoverage.value.slice(0, 2).map((item) => item.section).join(' / ') }
])
const kanbanColumns = computed(() => activeStatuses.value.map((status) => ({
  status,
  items: objects.value.filter((item) => item.status === status).slice(0, 8)
})))
const customerSpaceItems = computed(() => objects.value.filter((item) => ['project', 'milestone', 'deliverable', 'customer_space'].includes(item.object_type)).slice(0, 8))
const maxFunnelCount = computed(() => Math.max(1, ...((cockpit.charts.funnel || []).map((item) => Number(item.count || 0)))))
const maxTrendAmount = computed(() => Math.max(1, ...((cockpit.charts.trend || []).map((item) => Number(item.amount || 0)))))
const maxLeaderAmount = computed(() => Math.max(1, ...((cockpit.charts.leaderboard || []).map((item) => Number(item.amount || 0)))))
const targetPercent = computed(() => Math.round(Number(cockpit.charts.target?.progress || 0) * 100))
const processSteps = computed(() => [
  { index: '01', domain: 'campaigns', title: wt('processMarketing'), caption: wt('processMarketingCap') },
  { index: '02', domain: 'leads', title: wt('processLeadNurture'), caption: wt('processLeadNurtureCap') },
  { index: '03', domain: 'opportunities', title: wt('processSalesAdvance'), caption: wt('processSalesAdvanceCap') },
  { index: '04', domain: 'deals', title: wt('processDeals'), caption: wt('processDealsCap') },
  { index: '05', domain: 'orders', title: wt('processOrderFulfill'), caption: wt('processOrderFulfillCap') },
  { index: '06', domain: 'projects', title: wt('processProjectDelivery'), caption: wt('processProjectDeliveryCap') },
  { index: '07', domain: 'tickets', title: wt('processServiceLoop'), caption: wt('processServiceLoopCap') },
  { index: '08', domain: 'insights', title: wt('processOpsInsight'), caption: wt('processOpsInsightCap') },
  { index: '09', domain: 'workspace', title: wt('processWorkspace'), caption: wt('processWorkspaceCap') }
])

function wt(key) {
  return t(`domain.workspace.${key}`)
}

function wf(key, params = {}) {
  let text = wt(key)
  Object.entries(params).forEach(([k, v]) => {
    text = text.replace(new RegExp(`\{${k}\}`, 'g'), String(v))
  })
  return text
}

function domainDisplayName(domain) {
  if (!domain) return ''
  if (domain.key) return t(`domain.workspace.domain.${domain.key}.title`)
  return domain.name || ''
}

async function refreshAll() {
  await Promise.all([loadCompanySettings(), loadSummary(), loadObjects(), loadCockpit(), loadCoverage(), loadSchema(), loadActions(), loadApprovals()])
}

async function loadDomains() {
  const res = await api.get('/api/v1/prd/domains')
  domains.value = res.data?.items || []
  if (!activeObjectTypes.value.length && domains.value.length) activeDomain.value = domains.value[0].key
  applyRouteObjectType()
  normalizeSettingScope()
  resetForm()
}

async function loadSummary() {
  const res = await api.get('/api/v1/prd/summary')
  summary.value = res.data?.items || []
}

async function loadCockpit() {
  const res = await api.get('/api/v1/prd/cockpit')
  cockpit.tasks = res.data?.tasks || []
  cockpit.activities = res.data?.activities || []
  cockpit.kpi = res.data?.kpi || {}
  cockpit.charts = res.data?.charts || { funnel: [], trend: [], leaderboard: [], target: {} }
}

async function loadCoverage() {
  const res = await api.get('/api/v1/prd/coverage')
  coverage.value = res.data?.items || []
}

async function loadSchema() {
  const res = await api.get('/api/v1/prd/schema')
  schemas.value = res.data?.items || []
}

async function loadActions() {
  const res = await api.get('/api/v1/prd/actions')
  actions.value = res.data?.items || []
}

async function loadApprovals() {
  try {
    const res = await api.get('/api/v1/prd/approvals', { params: { status: 'pending', business_type: activeDomain.value } })
    approvals.value = res.data?.items || []
  } catch {
    approvals.value = []
  }
}

async function loadCompanySettings() {
  try {
    Object.assign(companySettings, await fetchCompanySettings())
  } catch {
    Object.assign(companySettings, DEFAULT_COMPANY_SETTINGS)
  }
}

async function loadObjects() {
  if (!activeDomain.value) return
  const params = Object.fromEntries(Object.entries(filters).filter(([, value]) => value !== '' && value !== undefined))
  const res = await api.get(`/api/v1/prd/${activeDomain.value}/objects`, { params: { ...params, page: 1, size: 80 } })
  objects.value = res.data?.items || []
  if (selectedRecord.value && !objects.value.some((item) => item.id === selectedRecord.value.id)) {
    resetForm()
  }
  if (!selectedRecord.value && objects.value[0]) editObject(objects.value[0])
}

async function selectDomain(domain) {
  activeDomain.value = domain
  settingForm.key = domain
  normalizeSettingScope()
  filters.object_type = ''
  router.replace(pathByDomain(domain))
  resetForm()
  await Promise.all([loadObjects(), loadApprovals()])
}

function startCreate() {
  resetForm()
  applyObjectTypeDefaults()
  sideTab.value = 'edit'
}

function editObject(record) {
  if (!record) return
  selectedRecord.value = record
  editingId.value = record.id
  form.object_type = record.object_type
  form.name = record.name
  form.code = record.code
  form.status = record.status
  form.amount = record.amount
  form.currency = normalizeCurrencyCode(record.currency)
  form.score = record.score
  tagText.value = record.tags || ''
  payloadText.value = prettyJSON(record.payload_json)
  hydrateFieldDraft(parseJSON(payloadText.value))
  loadRelations()
}

function rowClassName(record) {
  return record.id === editingId.value ? 'selected-row' : ''
}

function fillTemplate() {
  const template = { ...(activeDomainMeta.value.template || {}) }
  if (activeSchema.value?.defaults) Object.assign(template, activeSchema.value.defaults)
  payloadText.value = JSON.stringify(template, null, 2)
  hydrateFieldDraft(template)
}

function applyObjectTypeDefaults() {
  if (editingId.value) return
  if (!form.status || !activeStatuses.value.includes(form.status)) {
    form.status = activeStatuses.value[0] || 'active'
  }
  if (!form.name && activeSchema.value?.title) {
    form.name = `${activeSchema.value.title}-${Date.now().toString().slice(-4)}`
  }
  fillTemplate()
}

async function saveObject() {
  saving.value = true
  try {
    const payload = { ...form, tags: splitTags(tagText.value), payload: payloadWithFieldDraft() }
    if (editingId.value) {
      await api.put(`/api/v1/prd/${activeDomain.value}/objects/${editingId.value}`, payload)
      message.success(wt('saved'))
    } else {
      await api.post(`/api/v1/prd/${activeDomain.value}/objects`, payload)
      message.success(wt('created'))
    }
    resetForm()
    await refreshAll()
  } catch (error) {
    message.error(apiErrorMessage(error, wt('saveFailed')))
  } finally {
    saving.value = false
  }
}

async function deleteObject(id) {
  try {
    await api.delete(`/api/v1/prd/${activeDomain.value}/objects/${id}`)
    message.success(wt('deleted'))
    resetForm()
    await refreshAll()
  } catch (error) {
    message.error(apiErrorMessage(error, wt('deleteFailed')))
  }
}

async function loadRelations() {
  if (!editingId.value) {
    relations.value = []
    return
  }
  const res = await api.get('/api/v1/prd/relations', { params: { object_type: activeDomain.value, object_id: editingId.value } })
  relations.value = res.data?.items || []
}

async function saveSetting() {
  try {
    await api.post('/api/v1/prd/settings', { scope: settingForm.scope, key: settingForm.key, value: parseJSON(settingValueText.value) })
    settingSaveStatus.value = wt('settingsSaved')
    message.success(wt('settingsSaved'))
  } catch (error) {
    settingSaveStatus.value = ''
    message.error(apiErrorMessage(error, wt('settingsSaveFailed')))
  }
}

function presetOwnerScope() {
  settingForm.scope = 'data_scope'
  settingValueText.value = JSON.stringify({ mode: 'owner', piercing_user_ids: [] }, null, 2)
}

function presetHiddenFields() {
  settingForm.scope = 'field_acl'
  settingValueText.value = JSON.stringify({ hidden: ['cost_price', 'phone'], readonly: [], bypass_user_ids: [] }, null, 2)
}

function presetStageGate() {
  settingForm.scope = 'sales_process'
  settingForm.key = 'opportunities'
  settingValueText.value = JSON.stringify({ stage_required_fields: { proposal: ['proposal_url'], negotiation: ['quote_amount'] } }, null, 2)
}

function presetSLARule() {
  settingForm.scope = 'sla_rule'
  settingForm.key = 'tickets'
  settingValueText.value = JSON.stringify({ default_hours: 24, matrix: { 'complaint:VIP': 4, complaint: 8, repair: 24 } }, null, 2)
}

function presetPermissionMatrix() {
  settingForm.scope = 'permission_matrix'
  settingForm.key = 'modules'
  settingValueText.value = JSON.stringify({
    roles: {
      editor: { leads: 'edit', opportunities: 'edit', orders: 'read', tickets: 'edit' },
      viewer: { leads: 'read', opportunities: 'read', orders: 'read', tickets: 'read' }
    }
  }, null, 2)
}

function presetIntegration() {
  settingForm.scope = 'integration'
  settingForm.key = 'email'
  settingValueText.value = JSON.stringify({ type: 'email', endpoint: 'smtp://mail.example.com:25', enabled: false, from: 'crm@example.com', to: ['ops@example.com'] }, null, 2)
}

function presetAuditPolicy() {
  settingForm.scope = 'audit_policy'
  settingForm.key = 'retention'
  settingValueText.value = JSON.stringify({ retention_days: 180 }, null, 2)
}

async function runDomainAction(action) {
  const handler = actionHandlers[action.key]
  if (!handler) {
    message.warning(wt('actionNotWired'))
    return
  }
  const target = actionRequiresRecord(action) ? ensureActionTarget(action) : selectedRecord.value
  if (actionRequiresRecord(action) && !target) {
    message.warning(wf('selectOrCreateTypeRecord', { type: action.object_type }))
    return
  }
  await handler(target)
}

function ensureActionTarget(action) {
  if (selectedRecord.value?.object_type === action.object_type) {
    return selectedRecord.value
  }
  const target = objects.value.find((item) => item.object_type === action.object_type)
  if (target) {
    editObject(target)
  }
  return target || null
}

function actionRequiresRecord(action) {
  return !['insight_hotspots'].includes(action.key)
}

function actionButtonTitle(action) {
  if (action.key === 'opportunity_stage') return wt('advanceToNextStage')
  return action.title
}

async function testIntegration() {
  try {
    const value = parseJSON(settingValueText.value)
    await api.post('/api/v1/prd/integrations/test', { type: value.type || settingForm.key, endpoint: value.endpoint || '', config: value })
    message.success(wt('integrationCheckDone'))
  } catch (error) {
    message.error(apiErrorMessage(error, wt('integrationTestFailed')))
  }
}

async function executeIntegration() {
  try {
    const value = parseJSON(settingValueText.value)
    await api.post('/api/v1/prd/integrations/execute', {
      type: value.type || settingForm.key,
      endpoint: value.endpoint || '',
      config: value,
      payload: integrationPayload(value.type || settingForm.key)
    })
    message.success(wt('integrationExecuteDone'))
  } catch (error) {
    message.error(apiErrorMessage(error, wt('integrationExecuteFailed')))
  }
}

async function syncHR() {
  try {
    await api.post('/api/v1/prd/integrations/hr/sync')
    message.success(wt('hrSyncDone'))
    await refreshAll()
  } catch (error) {
    message.error(apiErrorMessage(error, wt('hrSyncFailed')))
  }
}

async function cleanupAuditLogs() {
  try {
    const value = parseJSON(settingValueText.value)
    await api.post('/api/v1/operation-logs/cleanup', { retention_days: Number(value.retention_days || 180) })
    message.success(wt('auditCleanupDone'))
  } catch (error) {
    message.error(apiErrorMessage(error, wt('auditCleanupFailed')))
  }
}

async function flowCall(callback, emptyMessage, record) {
  const target = record || selectedRecord.value
  if (!target) {
    message.warning(emptyMessage || wt('selectARecord'))
    return
  }
  try {
    await callback(target)
    await refreshAll()
  } catch (error) {
    message.error(apiErrorMessage(error, wt('flowExecuteFailed')))
  }
}

const convertLead = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/leads/${item.id}/convert`, { company_name: item.name, deal_name: `${item.name} ${wt('dealSuffix')}`, amount: item.amount, currency: normalizeCurrencyCode(item.currency) })
  message.success(wt('leadConverted'))
}, undefined, record)
const assignLead = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/leads/${item.id}/assign`, { owner_id: item.owner_id || currentUserId(), reason: 'manual_assign' })
  message.success(wt('leadAssigned'))
}, undefined, record)
const recycleLead = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/leads/${item.id}/recycle`, { reason: wt('reasonTimeoutNoFollow') })
  message.success(wt('leadRecycled'))
}, undefined, record)
const scoreLead = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/leads/${item.id}/score`)
  message.success(wt('leadScoreUpdated'))
}, undefined, record)
const captureCampaignLead = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/campaigns/${item.id}/capture-lead`, { name: `${item.name} ${wt('newLeadSuffix')}`, email: `lead${Date.now()}@example.com`, company: wt('campaignCustomer'), payload: { source_record_id: item.id, product_page_visits: 3, form_completeness: 80 } })
  message.success(wt('leadCaptured'))
}, undefined, record)
const generateCampaignAttribution = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/campaigns/${item.id}/attribution`)
  message.success(wt('campaignAttributionGenerated'))
}, undefined, record)
const advanceOpportunityStage = (record) => flowCall(async (item) => {
  const next = nextOpportunityStage(item.status)
  await api.post(`/api/v1/prd/flows/opportunities/${item.id}/stage`, { stage: next, comment: wf('advanceToStage', { stage: next }), payload_patch: stagePayloadPatch(next, item) })
  message.success(wt('opportunityStageAdvanced'))
}, undefined, record)
const winOpportunity = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/opportunities/${item.id}/win`)
  message.success(wt('dealAndOrderGenerated'))
}, undefined, record)
const generateOpportunityRisk = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/opportunities/${item.id}/risk`)
  message.success(wt('riskInsightGenerated'))
}, undefined, record)
async function generateHotInsights() {
  try {
    await api.post('/api/v1/prd/flows/insights/hotspots')
    message.success(wt('hotInsightGenerated'))
    await refreshAll()
  } catch (error) {
    message.error(apiErrorMessage(error, wt('hotInsightGenerateFailed')))
  }
}
const createOrderFromDeal = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/deals/${item.id}/order`)
  message.success(wt('orderGenerated'))
}, undefined, record)
const markOrderPaid = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/orders/${item.id}/pay`, { payment_no: `PAY-${Date.now()}`, payment_method: 'manual', paid_amount: item.amount || 0 })
  message.success(wt('orderPaid'))
}, undefined, record)
const shipOrder = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/orders/${item.id}/ship`, { shipment_no: `SHP-${Date.now()}`, carrier: 'manual' })
  message.success(wt('orderShipped'))
}, undefined, record)
const invoiceOrder = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/orders/${item.id}/invoice`, { invoice_no: `INV-${Date.now()}`, invoice_type: 'special_vat', tax_no: 'TAX-CRM-DEMO', sent_to: 'finance@example.com' })
  message.success(wt('orderInvoiced'))
}, undefined, record)
const returnOrder = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/orders/${item.id}/return`, { type: 'return', reason: wt('customerAfterSalesRequest') })
  message.success(wt('afterSalesTicketGenerated'))
}, undefined, record)
const fulfillOrder = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/orders/${item.id}/fulfill`)
  message.success(wt('deliveryProjectGenerated'))
}, undefined, record)
const createTicketFromOrder = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/orders/${item.id}/ticket`)
  message.success(wt('afterSalesTicketGenerated'))
}, undefined, record)
const createKnowledgeFromTicket = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/tickets/${item.id}/knowledge`)
  message.success(wt('knowledgeEntryGenerated'))
}, undefined, record)
const escalateTicketSLA = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/tickets/${item.id}/escalate`, { reason: wt('slaAboutToTimeout'), force: true })
  message.success(wt('slaEscalated'))
}, undefined, record)
const addTicketNote = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/tickets/${item.id}/note`, { visibility: 'public', content: wt('ticketNoteContent') })
  message.success(wt('ticketNoteSaved'))
}, undefined, record)
const completeProject = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/projects/${item.id}/complete`)
  message.success(wt('projectCompleted'))
}, undefined, record)
const closeTicket = (record) => flowCall(async (item) => {
  await api.post(`/api/v1/prd/flows/tickets/${item.id}/close`, { csat: 5, nps: 9, comment: wt('resolved') })
  message.success(wt('ticketClosed'))
}, undefined, record)
async function completeTask(record) {
  const item = record || selectedRecord.value
  if (!item) {
    message.warning(wt('selectATodo'))
    return
  }
  try {
    await api.post(`/api/v1/prd/flows/workspace/${item.id}/done`)
    message.success(wt('todoCompleted'))
    await refreshAll()
  } catch (error) {
    message.error(apiErrorMessage(error, wt('todoCompleteFailed')))
  }
}
async function markNotificationRead(record) {
  const item = record || selectedRecord.value
  if (!item) {
    message.warning(wt('selectANotification'))
    return
  }
  try {
    await api.post(`/api/v1/prd/flows/workspace/${item.id}/read`)
    message.success(wt('notificationRead'))
    await refreshAll()
  } catch (error) {
    message.error(apiErrorMessage(error, wt('notificationHandleFailed')))
  }
}
async function mergeSelectedContact() {
  const target = objects.value.find((item) => item.id !== editingId.value)
  if (!target) {
    message.warning(wt('noDuplicateContacts'))
    return
  }
  await flowCall(async () => {
    await api.post('/api/v1/prd/flows/contacts/merge', { primary_id: editingId.value, duplicate_ids: [target.id], reason: 'manual_merge' })
    message.success(wt('contactsMerged'))
  })
}

const actionHandlers = {
  campaign_capture_lead: captureCampaignLead,
  campaign_attribution: generateCampaignAttribution,
  lead_score: scoreLead,
  lead_assign: assignLead,
  lead_recycle: recycleLead,
  lead_convert: convertLead,
  contact_merge: mergeSelectedContact,
  insight_hotspots: generateHotInsights,
  opportunity_stage: advanceOpportunityStage,
  opportunity_win: winOpportunity,
  opportunity_risk: generateOpportunityRisk,
  deal_order: createOrderFromDeal,
  order_pay: markOrderPaid,
  order_ship: shipOrder,
  order_invoice: invoiceOrder,
  order_return: returnOrder,
  order_fulfill: fulfillOrder,
  order_ticket: createTicketFromOrder,
  project_complete: completeProject,
  ticket_note: addTicketNote,
  ticket_knowledge: createKnowledgeFromTicket,
  ticket_sla_escalate: escalateTicketSLA,
  ticket_close: closeTicket,
  workspace_done: completeTask,
  workspace_read: markNotificationRead
}

async function startApproval() {
  const target = approvalTarget.value
  if (!target) {
    message.warning(wt('selectApprovableRecord'))
    return
  }
  if (selectedRecord.value?.id !== target.id) {
    editObject(target)
  }
  try {
    const template = await ensureApprovalTemplate(activeDomain.value, target.object_type)
    await api.post(`/api/v1/prd/${activeDomain.value}/objects/${target.id}/approval`, {
      template_id: template.id,
      title: `${target.name} ${wt('approval')}`
    })
    message.success(wt('approvalStarted'))
    await refreshAll()
  } catch (error) {
    message.error(apiErrorMessage(error, wt('approvalStartFailed')))
  }
}

async function actApproval(item, action) {
  try {
    await api.post(`/api/v1/prd/approvals/${item.id}/action`, approvalActionPayload(action))
    message.success(approvalActionMessage(action))
    await refreshAll()
  } catch (error) {
    message.error(apiErrorMessage(error, wt('approvalHandleFailed')))
  }
}

async function ensureApprovalTemplate(domain, objectType) {
  const type = `${domain}:${objectType}`
  const list = await api.get('/api/v1/prd/approval-templates', { params: { type } })
  const existing = list.data?.items?.[0]
  if (existing) return existing
  const created = await api.post('/api/v1/prd/approval-templates', {
    name: defaultApprovalName(domain, objectType),
    type,
    nodes: [{ name: wt('ownerApprovalNode'), approver_role: 'manager' }],
    conditions: { domain, object_type: objectType }
  })
  return created.data
}

async function importObjects() {
  try {
    await api.post(`/api/v1/prd/${activeDomain.value}/objects/import`, parseJSON(importText.value))
    message.success(wt('importDone'))
    await refreshAll()
  } catch (error) {
    message.error(apiErrorMessage(error, wt('importFailed')))
  }
}

async function exportObjects() {
  await downloadProtectedFile(`/api/v1/prd/${activeDomain.value}/objects/export`, `${activeDomain.value}-objects-${dateStamp()}.xls`)
  message.success(t('common.action.export'))
}

function resetForm() {
  editingId.value = null
  selectedRecord.value = null
  Object.assign(form, defaultForm())
  form.object_type = filters.object_type || activeObjectTypes.value[0] || ''
  form.status = activeStatuses.value[0] || 'active'
  tagText.value = ''
  fillTemplate()
  relations.value = []
}

function applyRouteObjectType() {
  const objectType = routeObjectType.value
  filters.object_type = objectType && activeObjectTypes.value.includes(objectType) ? objectType : ''
}

function dateStamp() {
  const date = new Date()
  return `${date.getFullYear()}${String(date.getMonth() + 1).padStart(2, '0')}${String(date.getDate()).padStart(2, '0')}-${String(date.getHours()).padStart(2, '0')}${String(date.getMinutes()).padStart(2, '0')}${String(date.getSeconds()).padStart(2, '0')}`
}

function normalizeSettingScope() {
  if (!settingScopeOptions.value.some((item) => item.value === settingForm.scope)) {
    settingForm.scope = settingScopeOptions.value[0]?.value || 'company'
  }
  settingForm.key = activeDomain.value
}

const summaryCount = (domain) => summary.value.find((item) => item.domain === domain)?.total || 0
const splitTags = (value) => String(value || '').split(',').map((item) => item.trim()).filter(Boolean)
const parseJSON = (value) => JSON.parse(value || '{}')
const parseSettingValue = (value) => {
  try { return typeof value === 'string' ? JSON.parse(value || '{}') : (value || {}) } catch { return {} }
}
const parsePayload = (value) => {
  try { return typeof value === 'string' ? JSON.parse(value || '{}') : (value || {}) } catch { return {} }
}
function hydrateFieldDraft(payload) {
  Object.keys(fieldDraft).forEach((key) => delete fieldDraft[key])
  schemaFields.value.forEach((field) => {
    fieldDraft[field.key] = displayDraftValue(payload?.[field.key], field)
  })
}
function syncFieldDraftToPayload() {
  payloadText.value = JSON.stringify(payloadWithFieldDraft(), null, 2)
}
function payloadWithFieldDraft() {
  const payload = parseJSON(payloadText.value)
  schemaFields.value.forEach((field) => {
    const value = fieldDraft[field.key]
    if (value === undefined || value === '' || value === null) {
      delete payload[field.key]
    } else {
      payload[field.key] = castDraftValue(value, field)
    }
  })
  return payload
}
const displayDraftValue = (value, field) => {
  if (value === null || value === undefined) return ''
  if (field?.type === 'boolean') return value === true || value === 'true'
  if (field?.type === 'number') return Number(value || 0)
  if (field?.type === 'json' || typeof value === 'object') return JSON.stringify(value, null, 2)
  return String(value)
}
const castDraftValue = (value, field) => {
  if (field.type === 'boolean') return value === true || value === 'true'
  if (field.type === 'number') return Number(value || 0)
  if (field.type === 'json') {
    try { return JSON.parse(value || 'null') } catch { return value }
  }
  return value
}
const fieldSpec = (key, required, schema) => {
  const defaults = schema.defaults || {}
  return { key, label: key, required, type: inferFieldType(key, defaults[key]) }
}
const inferFieldType = (key, defaultValue) => {
  if (typeof defaultValue === 'boolean') return 'boolean'
  if (typeof defaultValue === 'number') return 'number'
  if (Array.isArray(defaultValue) || (defaultValue && typeof defaultValue === 'object')) return 'json'
  if (/(^|_)(amount|price|count|stock|score|rate|progress|quantity|budget|revenue|probability|hours|seconds|load|id)$/i.test(key)) return 'number'
  if (/(_id|_ids)$/i.test(key)) return 'number'
  if (/(enabled|required|visible|customer_visible|mfa_required|is_|has_)/i.test(key)) return 'boolean'
  if (/(items|roles|rules|matrix|acl|widgets|milestones|competitors|line_items|owner_split)/i.test(key)) return 'json'
  return 'text'
}
const currentApprovalNode = (item) => {
  const nodes = parsePayload(item?.nodes_with_status)
  const list = Array.isArray(nodes) ? nodes : nodes.nodes || []
  return list[(item?.current_node || 1) - 1] || list[0] || {}
}
const approvalActionPayload = (action) => {
  const payload = { action, comment: approvalActionComment(action) }
  if (action === 'append') {
    payload.name = wt('appendApproval')
    payload.target_user_id = currentUserId()
  }
  if (action === 'transfer') {
    payload.target_user_id = currentUserId()
  }
  return payload
}
const approvalActionComment = (action) => ({
  approve: wt('commentApprove'),
  reject: wt('commentReject'),
  append: wt('commentAppend'),
  transfer: wt('commentTransfer'),
  withdraw: wt('commentWithdraw')
}[action] || action)
const approvalActionMessage = (action) => ({
  approve: wt('approvalNodeApproved'),
  reject: wt('approvalRejected'),
  append: wt('approvalAppended'),
  transfer: wt('approvalTransferred'),
  withdraw: wt('approvalWithdrawn')
}[action] || wt('approvalHandled'))
const displayValue = (value) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
const prettyJSON = (value) => {
  try { return JSON.stringify(JSON.parse(value || '{}'), null, 2) } catch { return '{}' }
}
const money = (record) => formatAmount(record?.amount || 0, record?.currency)
const formatAmount = (amount, recordCurrency) => {
  const currency = normalizeCurrencyCode(recordCurrency || companySettings.currency, DEFAULT_CURRENCY)
  return formatCurrencyAmount(amount, currency, i18nState.locale)
}
const chartWidth = (value, max) => `${Math.max(8, Math.round((Number(value || 0) / Number(max || 1)) * 100))}%`
const integrationPayload = (type) => {
  if (['email', 'mail', 'smtp'].includes(type)) return { subject: wt('integrationMailSubject'), body: wt('integrationMailBody') }
  if (['finance', 'financial', 'invoice_api'].includes(type)) return { invoice_no: `INV-${Date.now()}`, amount: 1000, currency: companySettings.currency || DEFAULT_CURRENCY }
  if (['logistics', 'shipping'].includes(type)) return { order_no: `ORD-${Date.now()}`, receiver: wt('testCustomer'), address: wt('testAddress') }
  return {}
}
const shortDate = (value) => {
  if (!value) return '-'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return String(value).slice(0, 10)
  const timezone = companySettings.timezone || 'Asia/Shanghai'
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: timezone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  }).formatToParts(parsed).reduce((acc, item) => {
    acc[item.type] = item.value
    return acc
  }, {})
  if (companySettings.date_format === 'DD/MM/YYYY') return `${parts.day}/${parts.month}/${parts.year}`
  return `${parts.year}-${parts.month}-${parts.day}`
}
const statusType = (status) => {
  if (['active', 'running', 'paid', 'closed_won', 'won', 'completed'].includes(status)) return 'success'
  if (['watching', 'processing', 'proposal', 'qualified'].includes(status)) return 'primary'
  if (['at_risk', 'locked', 'recycled'].includes(status)) return 'warning'
  if (['closed_lost', 'lost', 'disabled'].includes(status)) return 'danger'
  return 'info'
}
const currentUserId = () => {
  try {
    return JSON.parse(localStorage.getItem('current_user') || '{}').id || 1
  } catch {
    return 1
  }
}
const nextOpportunityStage = (status) => {
  const stages = ['pipeline', 'proposal', 'negotiation', 'won']
  const index = stages.indexOf(status)
  return stages[index + 1] || 'proposal'
}
const stagePayloadPatch = (stage, record) => {
  const amount = Number(record?.amount || 0)
  if (stage === 'proposal') return { proposal_url: `https://example.com/proposal/${record?.id || Date.now()}` }
  if (stage === 'negotiation') return { quote_amount: amount || 100000 }
  return {}
}
const defaultApprovalName = (domain, objectType) => `${activeDomainMeta.value.name || domain}-${objectType}-${wt('defaultApproval')}`
const pathByDomain = (domain) => {
  const found = domains.value.find((item) => item.key === domain)
  return found ? `/${domain}` : `/crm-suite/${domain}`
}

watch(() => [route.meta.domain || route.params.domain, routeObjectType.value], async ([domain]) => {
  if (domain) {
    activeDomain.value = domain
    settingForm.key = activeDomain.value
    applyRouteObjectType()
    normalizeSettingScope()
    resetForm()
    await Promise.all([loadObjects(), loadApprovals()])
  }
})

watch(() => form.object_type, () => {
  if (!editingId.value && activeSchema.value?.defaults) {
    fillTemplate()
  } else {
    hydrateFieldDraft(parseJSON(payloadText.value))
  }
})

onMounted(async () => {
  await loadDomains()
  await refreshAll()
})
</script>
