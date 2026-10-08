<template>
  <div class="system-page project-page">
    <section class="crm-help-strip"><span>{{ t('domain.auto.6da8ac8ad9') }}</span></section>

    <section class="project-main-grid enhanced-project-grid">
      <div class="system-card">
        <div class="list-card-search">
          <el-form :model="filters" label-width="72px" class="system-search-form">
            <el-form-item :label="t('domain.auto.621219ff98')"><el-input v-model="filters.keyword" :placeholder="t('domain.auto.8b84844647')" clearable /></el-form-item>
            <el-form-item :label="t('domain.auto.62e951a692')"><el-select v-model="filters.status" clearable :placeholder="t('domain.auto.1a4c26d92d')"><el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
            <div class="system-search-actions">
              <el-button @click="resetFilters">{{ t('domain.auto.3d81345303') }}</el-button>
              <el-button type="primary" @click="loadPage">{{ t('domain.auto.711363c424') }}</el-button>
            </div>
          </el-form>
        </div>
        <div class="crm-stage-board">
          <button class="crm-stage-card" :class="{ active: !filters.status }" @click="quickStatus('')">
            <strong>{{ t('domain.auto.c352e4df56') }}</strong><span>{{ t('domain.auto.ad91043311') }}</span><em>{{ pagination.total }} {{ t('domain.auto.f7b2a6ee68') }}</em>
          </button>
          <button v-for="item in statusOptions" :key="item.value" class="crm-stage-card" :class="{ active: filters.status === item.value }" @click="quickStatus(item.value)">
            <strong>{{ item.label }}</strong><span>{{ item.desc }}</span><em>{{ statusCount(item.value) }} {{ t('domain.auto.f7b2a6ee68') }}</em>
          </button>
        </div>
        <div class="system-toolbar">
          <div class="toolbar-left">
            <el-button v-if="canManage" type="primary" @click="openCreate">{{ t('domain.auto.3f6da571c7') }}</el-button>
            <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openEdit(selectedRecord)">{{ t('domain.auto.a7f814c0a4') }}</el-button>
            <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" @click="openProgress(selectedRecord)">{{ t('domain.auto.6d5434c787') }}</el-button>
            <el-button v-if="canFollowRecord(selectedRecord)" :disabled="!selectedRecord" @click="openFollow(selectedRecord)">{{ t('domain.auto.a68cc3cf80') }}</el-button>
            <el-button v-if="canEditRecord(selectedRecord)" :disabled="!selectedRecord" type="success" plain @click="completeProject(selectedRecord)">{{ t('domain.auto.a4d932a5e3') }}</el-button>
          </div>
          <el-tag>{{ pagination.total }} {{ t('domain.auto.bce2ef6151') }}</el-tag>
        </div>
        <el-table
          :data="rows"
          class="system-table project-list-table"
          row-key="id"
          v-loading="loading"
          highlight-current-row
          :current-row-key="selectedRecord?.id"
          :row-class-name="projectRowClass"
          @row-click="selectProject"
        >
          <el-table-column prop="name" :label="t('domain.auto.22336e6b89')" min-width="240">
            <template #default="{ row }">
              <div class="record-title">
                <strong>{{ row.name }} <el-tag v-if="projectPayload(row).visibility === 'private'" size="small" type="warning">{{ t('domain.auto.c931219c30') }}</el-tag></strong>
                <span>{{ projectPayload(row).company || '-' }} {{ t('domain.auto.b63f504dfd') }} {{ ownerName(row.owner_id) }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('domain.auto.62e951a692')" width="120"><template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
          <el-table-column :label="t('domain.auto.28ff5ffe95')" width="110"><template #default="{ row }"><el-tag :type="healthType(projectPayload(row).health_status)">{{ healthLabel(projectPayload(row).health_status) }}</el-tag></template></el-table-column>
          <el-table-column :label="t('domain.auto.acf014bff8')" width="150"><template #default="{ row }"><el-progress :percentage="Number(projectPayload(row).progress || row.score || 0)" /></template></el-table-column>
          <el-table-column :label="t('domain.auto.ec46366045')" width="150"><template #default="{ row }"><span :class="{ 'sla-overdue': isPast(projectPayload(row).next_follow_at) }">{{ shortDate(projectPayload(row).next_follow_at) }}</span></template></el-table-column>
          <el-table-column :label="t('domain.auto.0d1e788e84')" width="130"><template #default="{ row }"><span :class="{ 'sla-overdue': isPast(projectPayload(row).due_date) && row.status !== 'completed' }">{{ shortDate(projectPayload(row).due_date) }}</span></template></el-table-column>
          <el-table-column :label="t('domain.auto.adfadb319f')" width="90"><template #default="{ row }">{{ projectPayload(row).file_count || '-' }}</template></el-table-column>
          <el-table-column :label="t('domain.auto.f3ea6d345e')" width="150" fixed="right">
            <template #default="{ row }">
              <div class="row-actions three" @click.stop>
                <el-tooltip :content="t('domain.auto.8a08005231')">
                  <button class="action-btn view" :aria-label="t('domain.auto.8a08005231')" @click="openDetail(row)"><Icon icon="ri:eye-line" /></button>
                </el-tooltip>
                <el-tooltip v-if="canEditRecord(row)" :content="t('domain.auto.cd95688761')">
                  <button class="action-btn edit" :aria-label="t('domain.auto.cd95688761')" @click="openEdit(row)"><Icon icon="ri:pencil-line" /></button>
                </el-tooltip>
                <el-tooltip v-if="canFollowRecord(row)" :content="t('domain.auto.af69218387')">
                  <button class="action-btn success" :aria-label="t('domain.auto.af69218387')" @click="openFollow(row)"><Icon icon="ri:chat-check-line" /></button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>
        <div class="system-pagination"><span>{{ t('domain.auto.3b6ef811b8') }} {{ pagination.total }} {{ t('domain.auto.bce2ef6151') }}</span><el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.size" layout="prev, pager, next, sizes" :total="pagination.total" @current-change="loadPage" @size-change="loadPage" /></div>
      </div>

      <aside class="system-card project-side-card">
        <div class="chart-panel-head"><strong>{{ t('domain.auto.439e3c242e') }}</strong><el-tag size="small">{{ t('domain.auto.bd1ab15a74') }}</el-tag></div>
        <div v-if="selectedRecord" class="lead-summary">
          <div><span>{{ t('domain.auto.22336e6b89') }}</span><strong>{{ selectedRecord.name }}</strong></div>
          <div><span>{{ t('domain.auto.974d383f36') }}</span><strong>{{ ownerName(selectedRecord.owner_id) }}</strong></div>
          <div><span>{{ t('domain.auto.08ff8ca47a') }}</span><strong>{{ selectedPayload.progress || selectedRecord.score || 0 }}%</strong></div>
          <div><span>{{ t('domain.auto.2e604c06f0') }}</span><strong>{{ healthLabel(selectedPayload.health_status) }}</strong></div>
          <div><span>{{ t('domain.auto.ea0ef2ae72') }}</span><strong>{{ selectedPayload.next_step || '-' }}</strong></div>
          <div><span>{{ t('domain.auto.3c5d2a19e5') }}</span><strong>{{ selectedPayload.acceptance_criteria || '-' }}</strong></div>
        </div>
        <el-empty v-else :description="t('domain.auto.7fe3fd827f')" />
      </aside>
    </section>

    <el-drawer v-model="detailDrawer" size="58%" :title="selectedRecord?.name || t('domain.auto.be80e2b3c0')" destroy-on-close>
      <div v-if="selectedRecord" class="lead-detail project-detail">
        <div class="lead-summary">
          <div><span>{{ t('domain.auto.974d383f36') }}</span><strong>{{ ownerName(selectedRecord.owner_id) }}</strong></div>
          <div><span>{{ t('domain.auto.acf014bff8') }}</span><strong>{{ selectedPayload.progress || selectedRecord.score || 0 }}%</strong></div>
          <div><span>{{ t('domain.auto.2e604c06f0') }}</span><strong>{{ healthLabel(selectedPayload.health_status) }}</strong></div>
          <div><span>{{ t('domain.auto.0d1e788e84') }}</span><strong>{{ shortDate(selectedPayload.due_date) }}</strong></div>
          <div><span>{{ t('domain.auto.ec46366045') }}</span><strong>{{ formatTime(selectedPayload.next_follow_at) }}</strong></div>
          <div><span>{{ t('domain.auto.241d8ef92f') }}</span><strong>{{ selectedPayload.visibility === 'private' ? t('domain.auto.19bc86cefb') : t('domain.auto.27df2bdde5') }}</strong></div>
        </div>
        <div class="detail-actions">
          <el-button v-if="canEditRecord(selectedRecord)" @click="openProgress(selectedRecord)">{{ t('domain.auto.6d5434c787') }}</el-button>
          <el-button v-if="canFollowRecord(selectedRecord)" @click="openFollow(selectedRecord)">{{ t('domain.auto.a68cc3cf80') }}</el-button>
          <el-button v-if="canFollowRecord(selectedRecord)" @click="openReminder(selectedRecord)">{{ t('domain.auto.c07f8b771c') }}</el-button>
          <el-button v-if="canFollowRecord(selectedRecord)" @click="openMilestone(selectedRecord)">{{ t('domain.auto.9b6eabd7ee') }}</el-button>
        </div>
        <el-tabs v-model="detailTab">
          <el-tab-pane :label="t('domain.auto.f86ba74abb')" name="timeline">
            <el-timeline><el-timeline-item v-for="item in timeline" :key="item.id" :timestamp="formatTime(item.created_at)"><strong>{{ projectPayload(item).summary || item.name }}</strong><small class="timeline-operator">{{ timelineOperator(item) }}</small><p>{{ projectPayload(item).reason || projectPayload(item).content || projectPayload(item).file_name || '-' }}</p></el-timeline-item></el-timeline>
            <el-empty v-if="!timeline.length" :description="t('domain.auto.9777ebee32')" />
          </el-tab-pane>
          <el-tab-pane :label="t('domain.auto.0d76bd4e8d')" name="files">
            <div class="detail-actions"><el-button v-if="canFollowRecord(selectedRecord)" type="primary" @click="openFileDialog('project')">{{ t('domain.auto.da905dfb4f') }}</el-button></div>
            <el-table :data="files" class="system-table" row-key="id">
              <el-table-column prop="name" :label="t('domain.auto.49deaf7da2')" min-width="220" />
              <el-table-column :label="t('domain.auto.435c5259e4')" width="120"><template #default="{ row }">{{ fileCategoryLabel(projectPayload(row).category) }}</template></el-table-column>
              <el-table-column :label="t('domain.auto.989d1affa0')" width="90"><template #default="{ row }">{{ projectPayload(row).version || '-' }}</template></el-table-column>
              <el-table-column :label="t('domain.auto.61890b022c')" width="130"><template #default="{ row }">{{ fileVisibilityLabel(projectPayload(row).visibility) }}</template></el-table-column>
              <el-table-column :label="t('domain.auto.fd20702c73')" width="110"><template #default="{ row }">{{ formatSize(projectPayload(row).file_size) }}</template></el-table-column>
              <el-table-column :label="t('domain.auto.f3ea6d345e')" width="150"><template #default="{ row }"><el-button link type="primary" @click="downloadFile(row)">{{ t('domain.auto.df5766f734') }}</el-button><el-button v-if="canEditRecord(selectedRecord)" link type="danger" @click="deleteFile(row)">{{ t('domain.auto.3755f56f2f') }}</el-button></template></el-table-column>
            </el-table>
            <el-empty v-if="!files.length" :description="t('domain.auto.642c03d788')" />
          </el-tab-pane>
          <el-tab-pane :label="t('domain.auto.f61ea80b01')" name="acl">
            <ObjectAccessPanel
              domain="projects"
              :record="selectedRecord"
              :users="users"
              :can-manage="canManageRecordACL(selectedRecord)"
              :object-label="t('domain.auto.22336e6b89')"
              @updated="refreshSelected"
            />
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-drawer>

    <el-dialog v-model="formDialog" :title="editingId ? t('domain.auto.b18abd2257') : t('domain.auto.3f6da571c7')" width="min(980px, 96vw)" destroy-on-close>
      <el-form :model="form" label-width="96px" class="lead-dialog-form">
        <el-form-item :label="t('domain.auto.3e7255522b')" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('domain.auto.e941d410f4')"><el-input v-model="form.payload.company" /></el-form-item>
        <el-form-item :label="t('domain.auto.62e951a692')"><el-select v-model="form.status"><el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.974d383f36')"><el-select v-model="form.owner_id" clearable filterable class="full-input"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.241d8ef92f')"><ObjectAccessField v-model="form.payload.visibility" :public-label="t('domain.auto.27df2bdde5')" :private-label="t('domain.auto.19bc86cefb')" /></el-form-item>
        <el-form-item :label="t('domain.auto.6914bd603d')"><el-input-number v-model="form.amount" :min="0" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.acf014bff8')"><el-slider v-model="form.payload.progress" :min="0" :max="100" /></el-form-item>
        <el-form-item :label="t('domain.auto.2e604c06f0')"><el-select v-model="form.payload.health_status"><el-option :label="t('domain.auto.f78d037abc')" value="healthy" /><el-option :label="t('domain.auto.32f446363a')" value="risk" /><el-option :label="t('domain.auto.a00db105bd')" value="blocked" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.0d1e788e84')"><el-date-picker v-model="form.payload.due_date" value-format="YYYY-MM-DD" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.ec46366045')"><el-date-picker v-model="form.payload.next_follow_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.ea0ef2ae72')"><el-input v-model="form.payload.next_step" /></el-form-item>
        <el-form-item :label="t('domain.auto.acf24ab9e0')" class="wide-field"><el-input v-model="form.payload.risk_summary" type="textarea" :rows="2" /></el-form-item>
        <el-form-item :label="t('domain.auto.3c5d2a19e5')" class="wide-field"><el-input v-model="form.payload.acceptance_criteria" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="formDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" @click="submitForm">{{ t('domain.auto.fadf24dbc5') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="progressDialog" :title="t('domain.auto.0cb96c453d')" width="620px" destroy-on-close>
      <el-form :model="progressForm" label-width="100px">
        <el-form-item :label="t('domain.auto.acf014bff8')"><el-slider v-model="progressForm.progress_percent" :min="0" :max="100" /></el-form-item>
        <el-form-item :label="t('domain.auto.4ca39faad0')"><el-input v-model="progressForm.phase" /></el-form-item>
        <el-form-item :label="t('domain.auto.2e604c06f0')"><el-select v-model="progressForm.health_status"><el-option :label="t('domain.auto.f78d037abc')" value="healthy" /><el-option :label="t('domain.auto.32f446363a')" value="risk" /><el-option :label="t('domain.auto.a00db105bd')" value="blocked" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.0d1e788e84')"><el-date-picker v-model="progressForm.due_date" value-format="YYYY-MM-DD" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.ec46366045')"><el-date-picker v-model="progressForm.next_follow_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.ea0ef2ae72')"><el-input v-model="progressForm.next_step" /></el-form-item>
        <el-form-item :label="t('domain.auto.acf24ab9e0')"><el-input v-model="progressForm.risk_summary" type="textarea" :rows="3" /></el-form-item>
        <el-form-item :label="t('domain.auto.26670dda42')"><el-input v-model="progressForm.reason" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="progressDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" @click="submitProgress">{{ t('domain.auto.fadf24dbc5') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="followDialog" :title="t('domain.auto.5b3c0d62f2')" width="620px" destroy-on-close>
      <el-form :model="followForm" label-width="100px">
        <el-form-item :label="t('domain.auto.ab264d91c6')" required><el-input v-model="followForm.content" type="textarea" :rows="4" /></el-form-item>
        <el-form-item :label="t('domain.auto.08ff8ca47a')"><el-slider v-model="followForm.progress_percent" :min="0" :max="100" /></el-form-item>
        <el-form-item :label="t('domain.auto.ec46366045')"><el-date-picker v-model="followForm.next_follow_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.61890b022c')"><el-select v-model="followForm.visibility"><el-option :label="t('domain.auto.d162da4eaf')" value="internal" /><el-option :label="t('domain.auto.b8692e7870')" value="customer_visible" /><el-option :label="t('domain.auto.c931219c30')" value="private" /></el-select></el-form-item>
      </el-form>
      <template #footer><el-button @click="followDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" @click="submitFollow">{{ t('domain.auto.fadf24dbc5') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="reminderDialog" :title="t('domain.auto.f284f95a5c')" width="620px" destroy-on-close>
      <el-form :model="reminderForm" label-width="100px">
        <el-form-item :label="t('domain.auto.748d7dc7e3')"><el-input v-model="reminderForm.title" /></el-form-item>
        <el-form-item :label="t('domain.auto.4f36c3b75a')"><el-date-picker v-model="reminderForm.remind_at" type="datetime" value-format="YYYY-MM-DD HH:mm:ss" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.892a872784')"><el-select v-model="reminderForm.owner_ids" multiple filterable class="full-input"><el-option v-for="item in ownerOptions" :key="item.id" :label="item.label" :value="item.id" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.163aec9194')"><el-input v-model="reminderForm.content" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="reminderDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" @click="submitReminder">{{ t('domain.auto.fcbd093292') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="milestoneDialog" :title="t('domain.auto.c42c45a324')" width="620px" destroy-on-close>
      <el-form :model="milestoneForm" label-width="100px">
        <el-form-item :label="t('domain.auto.1be7ae4fc2')" required><el-input v-model="milestoneForm.name" /></el-form-item>
        <el-form-item :label="t('domain.auto.62e951a692')"><el-select v-model="milestoneForm.status"><el-option :label="t('domain.auto.062e5e670f')" value="open" /><el-option :label="t('domain.auto.6f1972e48e')" value="doing" /><el-option :label="t('domain.auto.e99b48a29b')" value="done" /><el-option :label="t('domain.auto.65a02a484e')" value="overdue" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.3144a998f0')"><el-date-picker v-model="milestoneForm.due_date" value-format="YYYY-MM-DD" class="full-input" /></el-form-item>
        <el-form-item :label="t('domain.auto.e0361480e3')"><el-input v-model="milestoneForm.remark" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="milestoneDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" @click="submitMilestone">{{ t('domain.auto.fadf24dbc5') }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="fileDialog" :title="t('domain.auto.1bd8fa6924')" width="620px" destroy-on-close>
      <el-form :model="fileForm" label-width="100px">
        <el-form-item :label="t('domain.auto.21a6f5a8d9')" required><input type="file" @change="onFileChange" /></el-form-item>
        <el-form-item :label="t('domain.auto.435c5259e4')"><el-select v-model="fileForm.category"><el-option v-for="item in fileCategories" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.989d1affa0')"><el-input v-model="fileForm.version" /></el-form-item>
        <el-form-item :label="t('domain.auto.61890b022c')"><el-select v-model="fileForm.visibility"><el-option :label="t('domain.auto.d162da4eaf')" value="internal" /><el-option :label="t('domain.auto.b8692e7870')" value="customer_visible" /><el-option :label="t('domain.auto.c931219c30')" value="private" /></el-select></el-form-item>
        <el-form-item :label="t('domain.auto.e0361480e3')"><el-input v-model="fileForm.remark" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer><el-button @click="fileDialog = false">{{ t('domain.auto.4d632deeea') }}</el-button><el-button type="primary" :loading="fileUploading" @click="submitFile">{{ t('domain.auto.a977650db1') }}</el-button></template>
    </el-dialog>

  </div>
</template>

<script setup>

import { t } from '../../i18n'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Icon } from '@iconify/vue'
import api from '../../api'
import { canEditRecord, canFollowRecord, canManageDomain, canManageRecordACL } from '../../permissions'
import { confirmDangerAction } from '../../utils/confirm'
import { downloadProtectedFile, unwrapFileUploadResponse } from '../../utils/download'
import { timelineOperatorText } from '../../utils/timeline'
import ObjectAccessPanel from '../../components/ObjectAccessPanel.vue'
import ObjectAccessField from '../../components/ObjectAccessField.vue'
import { formatCurrencyAmount, normalizeCurrencyCode } from '../../utils/currency'
import { defaultBusinessCurrency } from '../../utils/company-format'
import { apiErrorMessage } from '../../utils/api-error'

const rows = ref([])
const route = useRoute()
const users = ref([])
const loading = ref(false)
const selectedRecord = ref(null)
const timeline = ref([])
const files = ref([])
const detailDrawer = ref(false)
const detailTab = ref('timeline')
const formDialog = ref(false)
const progressDialog = ref(false)
const followDialog = ref(false)
const reminderDialog = ref(false)
const milestoneDialog = ref(false)
const fileDialog = ref(false)
const fileUploading = ref(false)
const selectedUploadFile = ref(null)
const editingId = ref(null)
const filters = reactive({ keyword: '', status: '' })
const pagination = reactive({ page: 1, size: 20, total: 0 })
const form = reactive(defaultForm())
const progressForm = reactive({ progress_percent: 0, phase: '', health_status: 'healthy', next_step: '', risk_summary: '', next_follow_at: '', due_date: '', reason: '' })
const followForm = reactive({ content: '', next_follow_at: '', progress_percent: 0, visibility: 'internal' })
const reminderForm = reactive({ title: '', content: '', remind_at: '', owner_ids: [] })
const milestoneForm = reactive({ name: '', status: 'open', due_date: '', remark: '' })
const fileForm = reactive({ category: 'other', version: 'v1', visibility: 'internal', remark: '' })
const canManage = computed(() => canManageDomain('projects'))
const selectedPayload = computed(() => projectPayload(selectedRecord.value))
const ownerOptions = computed(() => users.value.map((item) => ({ id: item.id, label: userDisplayName(item) })))
const statusOptions = [
  { value: 'planning', label: t('domain.auto.846552c433'), desc: t('domain.auto.513f4488de') },
  { value: 'active', label: t('domain.auto.6f1972e48e'), desc: t('domain.auto.65f7d2165b') },
  { value: 'blocked', label: t('domain.auto.06439d95d0'), desc: t('domain.auto.ae3b687dc9') },
  { value: 'completed', label: t('domain.auto.e99b48a29b'), desc: t('domain.auto.c6a8487273') }
]
const fileCategories = [
  { value: 'contract', label: t('domain.auto.50e5b3d4ff') },
  { value: 'proposal', label: t('domain.auto.d257f5b5df') },
  { value: 'meeting', label: t('domain.auto.1a45a4b0f7') },
  { value: 'delivery', label: t('domain.auto.848a9c3760') },
  { value: 'acceptance', label: t('domain.auto.39535110be') },
  { value: 'other', label: t('domain.auto.614f418c41') }
]

onMounted(async () => {
  await Promise.all([loadUsers(), loadPage()])
  await openProjectFromRoute()
})
watch(() => route.query.id, async () => { await openProjectFromRoute() })
async function loadUsers() { try { const res = await api.get('/api/v1/accounts/users', { params: { page: 1, size: 300, status: 'active' } }); users.value = res.data?.items || [] } catch { users.value = [] } }
async function loadPage() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/prd/projects/objects', { params: cleanParams({ object_type: 'project', ...filters, page: pagination.page, size: pagination.size }) })
    rows.value = res.data?.items || []
    pagination.total = res.data?.total || 0
    if (selectedRecord.value) selectedRecord.value = rows.value.find((item) => item.id === selectedRecord.value.id) || selectedRecord.value
  } catch (error) { ElMessage.error(apiErrorMessage(error, t('domain.auto.a65938f554'))) } finally { loading.value = false }
}
function selectProject(row) { selectedRecord.value = row }
function projectRowClass({ row }) {
  return Number(selectedRecord.value?.id || 0) === Number(row.id || 0) ? 'is-selected-row' : ''
}
async function openDetail(row) { selectedRecord.value = row; detailDrawer.value = true; detailTab.value = 'timeline'; await Promise.all([loadTimeline(row), loadFiles(row)]) }
async function openProjectFromRoute() {
  const id = Number(route.query.id || 0)
  if (!id || Number(selectedRecord.value?.id || 0) === id) return
  let row = rows.value.find((item) => Number(item.id) === id)
  if (!row) {
    try {
      const res = await api.get(`/api/v1/prd/projects/objects/${id}`)
      row = res.data
    } catch (error) {
      ElMessage.error(apiErrorMessage(error, t('domain.auto.a65938f554')))
      return
    }
  }
  await openDetail(row)
}
async function loadTimeline(row) { const rel = await api.get('/api/v1/prd/relations', { params: { object_type: 'projects', object_id: row.id, relation_type: 'timeline' } }); const ids = (rel.data?.items || []).map((item) => item.target_type === 'contacts' ? item.target_id : item.source_id); const items = await Promise.all(ids.map((id) => api.get(`/api/v1/prd/contacts/objects/${id}`).catch(() => null))); timeline.value = items.map((res) => res?.data).filter(Boolean) }
async function loadFiles(row) { try { const res = await api.get(`/api/v1/prd/flows/projects/${row.id}/files`); files.value = res.data?.items || [] } catch { files.value = [] } }
function openCreate() { editingId.value = null; Object.assign(form, defaultForm()); formDialog.value = true }
function openEdit(row) { if (!row) return; editingId.value = row.id; Object.assign(form, { object_type: 'project', name: row.name, code: row.code || '', status: row.status || 'planning', owner_id: row.owner_id || undefined, amount: row.amount || 0, currency: normalizeCurrencyCode(row.currency, defaultBusinessCurrency()), score: row.score || 0, tags: splitTags(row.tags), payload: { ...defaultForm().payload, progress: row.score || 0, ...projectPayload(row) } }); formDialog.value = true }
async function submitForm() {
  if (!form.name) return ElMessage.error(t('domain.auto.de9b6c5b55'))
  try {
    form.score = Number(form.payload.progress || 0)
    if (editingId.value) await api.put(`/api/v1/prd/projects/objects/${editingId.value}`, form)
    else await api.post('/api/v1/prd/projects/objects', form)
    ElMessage.success(t('domain.auto.7e68eb622d'))
    formDialog.value = false
    await loadPage()
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.a65938f554')))
  }
}
function openProgress(row) { if (!row) return; selectedRecord.value = row; const p = projectPayload(row); Object.assign(progressForm, { progress_percent: Number(p.progress || row.score || 0), phase: p.phase || '', health_status: p.health_status || 'healthy', next_step: p.next_step || '', risk_summary: p.risk_summary || '', next_follow_at: normalizeDateInput(p.next_follow_at), due_date: shortDate(p.due_date), reason: '' }); progressDialog.value = true }
async function submitProgress() {
  const row = selectedRecord.value
  try {
    await api.post(`/api/v1/prd/flows/projects/${row.id}/progress`, progressForm)
    ElMessage.success(t('domain.auto.ec8a3da0b5'))
    progressDialog.value = false
    await refreshSelected(row.id)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.a65938f554')))
  }
}
function openFollow(row) { if (!row) return; selectedRecord.value = row; Object.assign(followForm, { content: '', next_follow_at: '', progress_percent: Number(projectPayload(row).progress || row.score || 0), visibility: 'internal' }); followDialog.value = true }
async function submitFollow() {
  if (!followForm.content) return ElMessage.error(t('domain.auto.b7cfc87770'))
  const row = selectedRecord.value
  try {
    await api.post(`/api/v1/prd/flows/projects/${row.id}/follow-up`, followForm)
    ElMessage.success(t('domain.auto.89e6d84e48'))
    followDialog.value = false
    await refreshSelected(row.id)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.a65938f554')))
  }
}
function openReminder(row) { selectedRecord.value = row; Object.assign(reminderForm, { title: `${t('domain.auto.b117b758fe')}${row.name}`, content: '', remind_at: '', owner_ids: row.owner_id ? [row.owner_id] : [] }); reminderDialog.value = true }
async function submitReminder() {
  const row = selectedRecord.value
  try {
    await api.post(`/api/v1/prd/flows/projects/${row.id}/reminder`, reminderForm)
    ElMessage.success(t('domain.auto.b2b668e7a5'))
    reminderDialog.value = false
    await loadTimeline(row)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.a65938f554')))
  }
}
function openMilestone(row) { selectedRecord.value = row; Object.assign(milestoneForm, { name: '', status: 'open', due_date: '', remark: '' }); milestoneDialog.value = true }
async function submitMilestone() {
  if (!milestoneForm.name) return ElMessage.error(t('domain.auto.2d1eb76a24'))
  const row = selectedRecord.value
  try {
    await api.post(`/api/v1/prd/flows/projects/${row.id}/milestone`, milestoneForm)
    ElMessage.success(t('domain.auto.be646deb0d'))
    milestoneDialog.value = false
    await loadTimeline(row)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.a65938f554')))
  }
}
function openFileDialog() { selectedUploadFile.value = null; Object.assign(fileForm, { category: 'other', version: 'v1', visibility: 'internal', remark: '' }); fileDialog.value = true }
function onFileChange(event) { selectedUploadFile.value = event.target.files?.[0] || null }
async function submitFile() {
  if (!selectedUploadFile.value || !selectedRecord.value) return ElMessage.error(t('domain.auto.eba557edfa'))
  fileUploading.value = true
  try {
    const data = new FormData(); data.append('file', selectedUploadFile.value)
    const upload = await api.post('/api/v1/files/upload', data)
    const info = unwrapFileUploadResponse(upload)
    await api.post(`/api/v1/prd/flows/projects/${selectedRecord.value.id}/files`, { ...fileForm, file_id: info.file_id, file_name: info.original_name || selectedUploadFile.value.name, file_size: info.size || selectedUploadFile.value.size, file_type: selectedUploadFile.value.type, storage_node_id: info.storage_node_id, download_url: info.download_url })
    ElMessage.success(t('domain.auto.7f9e1b1b7c')); fileDialog.value = false; await Promise.all([loadFiles(selectedRecord.value), loadTimeline(selectedRecord.value)])
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.e9bcb5373a')))
  } finally { fileUploading.value = false }
}
async function downloadFile(row) {
  const filePayload = projectPayload(row)
  const fileID = filePayload.file_id
  const url = filePayload.download_url || (fileID ? `/api/v1/files/${fileID}/download` : '')
  try {
    await downloadProtectedFile(url, filePayload.file_name || row.name)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.7a8d3dbaaf')))
  }
}
async function deleteFile(row) {
  await confirmDangerAction(`${t('domain.auto.3a61f7a246')}${row.name || projectPayload(row).file_name || t('domain.auto.16a3a69043')}${t('domain.auto.30baaa0f0c')}`, t('domain.auto.abeece0622'))
  await api.delete(`/api/v1/prd/flows/projects/${selectedRecord.value.id}/files/${row.id}`, { skipDeleteConfirm: true })
  ElMessage.success(t('domain.auto.ce8e746a98'))
  await Promise.all([loadFiles(selectedRecord.value), loadTimeline(selectedRecord.value)])
}
async function completeProject(row) {
  if (!row) return
  try {
    await api.post(`/api/v1/prd/flows/projects/${row.id}/complete`)
    ElMessage.success(t('domain.auto.a74e3b9afd'))
    await refreshSelected(row.id)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.a65938f554')))
  }
}
async function refreshSelected(id) { await loadPage(); const next = rows.value.find((item) => item.id === id); if (next) { selectedRecord.value = next; await Promise.all([loadTimeline(next), loadFiles(next)]) } }
function quickStatus(status) { filters.status = filters.status === status ? '' : status; loadPage() }
function resetFilters() { Object.assign(filters, { keyword: '', status: '' }); pagination.page = 1; loadPage() }
function defaultForm() { return { object_type: 'project', name: '', code: '', status: 'planning', owner_id: undefined, amount: 0, currency: defaultBusinessCurrency(), score: 0, tags: [], payload: { company: '', progress: 0, due_date: '', next_follow_at: '', delivery_owner: '', next_step: '', risk_summary: '', health_status: 'healthy', visibility: 'public', acceptance_criteria: '' } } }
function projectPayload(row) { try { return typeof row?.payload_json === 'string' ? JSON.parse(row.payload_json || '{}') : (row?.payload_json || {}) } catch { return {} } }
function statusCount(status) { return rows.value.filter((item) => item.status === status).length }
function splitTags(value) { return String(value || '').split(',').map((item) => item.trim()).filter(Boolean) }
function cleanParams(params) { return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== '' && value !== undefined && value !== null)) }
function statusLabel(value) { return statusOptions.find((item) => item.value === value)?.label || value || '-' }
function statusType(value) { return ({ planning: 'info', active: 'primary', blocked: 'danger', completed: 'success' }[value] || 'info') }
function healthLabel(value) { return ({ healthy: t('domain.auto.f78d037abc'), normal: t('domain.auto.f78d037abc'), risk: t('domain.auto.32f446363a'), blocked: t('domain.auto.a00db105bd'), done: t('domain.auto.33246f6a5e') }[value] || t('domain.auto.f78d037abc')) }
function healthType(value) { return ({ healthy: 'success', normal: 'success', risk: 'warning', blocked: 'danger', done: 'success' }[value] || 'success') }
function ownerName(ownerID) { const id = Number(ownerID || 0); if (!id) return t('domain.auto.3bb92e6efc'); const user = users.value.find((item) => Number(item.id) === id); return user ? userDisplayName(user) : `${t('domain.auto.9ba763ea34')}${id}` }
function userDisplayName(user) { const label = user.name || user.username || user.email || `${t('domain.auto.9ba763ea34')}${user.id}`; return user.department_name ? `${label} · ${user.department_name}` : label }
function timelineOperator(item) { return timelineOperatorText(item, users.value, formatTime) }
function fileCategoryLabel(value) { return fileCategories.find((item) => item.value === value)?.label || t('domain.auto.614f418c41') }
function fileVisibilityLabel(value) { return ({ internal: t('domain.auto.d162da4eaf'), customer_visible: t('domain.auto.b8692e7870'), private: t('domain.auto.c931219c30') }[value] || t('domain.auto.d162da4eaf')) }
function formatSize(value) { const n = Number(value || 0); if (n >= 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`; if (n >= 1024) return `${(n / 1024).toFixed(1)} KB`; return `${n} B` }
function normalizeDateInput(value) { if (!value) return ''; const date = new Date(value); return Number.isNaN(date.getTime()) ? String(value).slice(0, 19) : `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}` }
function shortDate(value) { if (!value) return '-'; return String(value).slice(0, 10) }
function isPast(value) { if (!value) return false; const date = new Date(value); return !Number.isNaN(date.getTime()) && date.getTime() < Date.now() }
function formatTime(value) { if (!value) return '-'; const date = new Date(value); return Number.isNaN(date.getTime()) ? String(value).slice(0, 19) : `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}` }
function formatAmount(amount, currency = defaultBusinessCurrency()) { return formatCurrencyAmount(amount, normalizeCurrencyCode(currency, defaultBusinessCurrency())) }
</script>

<style scoped>
.timeline-operator {
  display: block;
  margin-top: 4px;
  color: #667085;
  font-size: 12px;
  line-height: 1.4;
}

.enhanced-project-grid {
  grid-template-columns: minmax(0, 1fr) 320px;
}
.project-detail :deep(.el-tabs__content) {
  padding-top: 12px;
}
.sla-overdue {
  color: #dc2626;
  font-weight: 700;
}
</style>
