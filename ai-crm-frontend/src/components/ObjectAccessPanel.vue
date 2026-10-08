<template>
  <section class="object-access-panel">
    <div class="object-access-head">
      <div>
        <strong>{{ objectLabel }}{{ t('domain.auto.be59a54e35') }}</strong>
        <span>{{ accessSummary }}</span>
      </div>
      <div class="object-access-actions">
        <el-tag :type="currentVisibility === 'private' ? 'warning' : 'success'">
          {{ visibilityLabel(currentVisibility) }}
        </el-tag>
        <el-button v-if="canManage" type="primary" plain @click="openDialog">{{ t('domain.auto.118ccf88ed') }}</el-button>
      </div>
    </div>

    <div class="object-access-list">
      <div v-for="entry in aclEntries" :key="entry.user_id" class="object-access-row">
        <strong>{{ userLabel(entry.user_id) }}</strong>
        <span>{{ aclLevelLabel(entry.access_level) }}</span>
      </div>
      <el-empty v-if="!aclEntries.length" :description="t('domain.auto.82d6b5397b')" />
    </div>
  </section>

  <el-dialog v-model="dialogVisible" :title="t('domain.auto.bb9acf2b59')" width="680px" destroy-on-close>
    <el-alert
      type="info"
      show-icon
      :closable="false"
      :title="t('domain.auto.2a1b0ecae0')"
    />
    <el-form label-width="92px" class="object-access-form">
      <el-form-item :label="t('domain.auto.241d8ef92f')">
        <el-radio-group v-model="accessForm.visibility" :disabled="!canManage">
          <el-radio-button label="public">{{ t('domain.auto.dfe5a318ab') }}</el-radio-button>
          <el-radio-button label="private">{{ t('domain.auto.c931219c30') }}</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item :label="t('domain.auto.48756a2d5d')" class="wide-field">
        <div class="object-acl-editor">
          <div v-for="entry in accessForm.entries" :key="entry.uid" class="object-acl-row">
            <el-select v-model="entry.user_id" filterable clearable :placeholder="t('domain.auto.412af8474e')" class="object-acl-user-select" :teleported="true">
              <el-option v-for="user in users" :key="user.id" :label="formatUser(user)" :value="user.id" />
            </el-select>
            <el-select v-model="entry.access_level" :placeholder="t('domain.auto.7e6fe7f2d7')" class="object-acl-level-select" :teleported="true">
              <el-option :label="t('domain.auto.f7acefd2d4')" value="view" />
              <el-option :label="t('domain.auto.0a96a4a257')" value="follow" />
              <el-option :label="t('domain.auto.9263e46a01')" value="handle" />
              <el-option :label="t('domain.auto.4989b5cf94')" value="manage" />
            </el-select>
            <el-button type="danger" link class="object-acl-remove" @click="removeEntry(entry.uid)">{{ t('domain.auto.3755f56f2f') }}</el-button>
          </div>
          <el-button plain @click="addEntry">{{ t('domain.auto.9fde446e8f') }}</el-button>
        </div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">{{ t('domain.auto.4d0b4688c7') }}</el-button>
      <el-button type="primary" :loading="saving" @click="saveAccess">{{ t('domain.auto.fadf24dbc5') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { t } from '../i18n'
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'
import { confirmDangerAction } from '../utils/confirm'
import { normalizeCurrencyCode } from '../utils/currency'
import { apiErrorMessage } from '../utils/api-error'

const props = defineProps({
  domain: { type: String, required: true },
  record: { type: Object, default: null },
  users: { type: Array, default: () => [] },
  canManage: { type: Boolean, default: false },
  objectLabel: { type: String, default: t('domain.auto.539cb3b0a6') }
})

const emit = defineEmits(['updated'])

const dialogVisible = ref(false)
const saving = ref(false)
const aclEntries = ref([])
const accessForm = reactive({
  visibility: 'public',
  entries: []
})
let entrySeq = 0

const currentPayload = computed(() => parsePayload(props.record))
const currentVisibility = computed(() => normalizeVisibility(currentPayload.value))
const accessSummary = computed(() => {
  const base = currentVisibility.value === 'private'
    ? `${props.objectLabel}${t('domain.auto.cb9830f9d6')}`
    : `${props.objectLabel}${t('domain.auto.fb37404cb1')}`
  return aclEntries.value.length ? `${base}${t('domain.auto.6a385b2cfd')}${aclEntries.value.length}${t('domain.auto.519729d5cf')}` : base
})

watch(
  () => props.record?.id,
  async () => {
    accessForm.visibility = currentVisibility.value
    await loadACL()
  },
  { immediate: true }
)

async function loadACL() {
  if (!props.record?.id || !props.domain) {
    aclEntries.value = []
    return
  }
  try {
    const res = await api.get(`/api/v1/prd/${props.domain}/objects/${props.record.id}/acl`)
    aclEntries.value = res.data?.items || []
  } catch {
    aclEntries.value = []
  }
}

function openDialog() {
  accessForm.visibility = currentVisibility.value
  accessForm.entries = aclEntries.value.map((item) => createEntry({
    user_id: item.user_id,
    access_level: item.access_level || 'view'
  }))
  dialogVisible.value = true
}

function addEntry() {
  accessForm.entries.push(createEntry())
}

async function removeEntry(uid) {
  await confirmDangerAction(t('domain.auto.1b3c8230fb'), t('domain.auto.d295b93bf9'))
  const index = accessForm.entries.findIndex((item) => item.uid === uid)
  if (index >= 0) accessForm.entries.splice(index, 1)
}

async function saveAccess() {
  if (!props.record?.id || saving.value) return
  saving.value = true
  try {
    await saveVisibility()
    const entries = normalizedEntries()
    await api.put(`/api/v1/prd/${props.domain}/objects/${props.record.id}/acl`, { entries })
    ElMessage.success(t('domain.auto.aa86ed013b'))
    dialogVisible.value = false
    await loadACL()
    emit('updated', props.record.id)
  } catch (error) {
    ElMessage.error(apiErrorMessage(error, t('domain.auto.e2d59efbd5')))
  } finally {
    saving.value = false
  }
}

async function saveVisibility() {
  let record = props.record
  try {
    const res = await api.get(`/api/v1/prd/${props.domain}/objects/${props.record.id}`)
    record = res.data || record
  } catch {
    record = props.record
  }
  const payload = {
    ...parsePayload(record),
    visibility: accessForm.visibility,
    private: accessForm.visibility === 'private'
  }
  await api.put(`/api/v1/prd/${props.domain}/objects/${record.id}`, {
    object_type: record.object_type,
    name: record.name,
    code: record.code || '',
    status: record.status || 'active',
    owner_id: record.owner_id || 0,
    department_id: record.department_id || 0,
    company_id: record.company_id || 0,
    contact_id: record.contact_id || 0,
    deal_id: record.deal_id || 0,
    project_id: record.project_id || 0,
    amount: Number(record.amount || 0),
    currency: normalizeCurrencyCode(record.currency),
    score: Number(record.score || 0),
    tags: splitTags(record.tags),
    payload
  })
}

function normalizedEntries() {
  const seen = new Set()
  return accessForm.entries
    .filter((item) => item.user_id)
    .map((item) => ({
      user_id: Number(item.user_id),
      access_level: item.access_level || 'view'
    }))
    .filter((item) => {
      if (seen.has(item.user_id)) return false
      seen.add(item.user_id)
      return true
    })
}

function createEntry(value = {}) {
  entrySeq += 1
  return {
    uid: value.uid || `acl_${Date.now()}_${entrySeq}`,
    user_id: value.user_id,
    access_level: value.access_level || 'view'
  }
}

function parsePayload(record) {
  try {
    return typeof record?.payload_json === 'string' ? JSON.parse(record.payload_json || '{}') : (record?.payload_json || {})
  } catch {
    return {}
  }
}

function normalizeVisibility(payload) {
  if (String(payload?.visibility || payload?.access_visibility || '').toLowerCase() === 'private') return 'private'
  if (payload?.private === true || String(payload?.private).toLowerCase() === 'true') return 'private'
  return 'public'
}

function visibilityLabel(value) {
  return value === 'private' ? t('domain.auto.c931219c30') : t('domain.auto.dfe5a318ab')
}

function aclLevelLabel(value) {
  return ({ view: t('domain.auto.f7acefd2d4'), follow: t('domain.auto.0a96a4a257'), handle: t('domain.auto.9263e46a01'), manage: t('domain.auto.4989b5cf94') }[value] || value || '-')
}

function userLabel(userID) {
  const user = props.users.find((item) => Number(item.id) === Number(userID))
  return user ? formatUser(user) : `${t('domain.auto.9ba763ea34')}${userID}`
}

function formatUser(user) {
  const label = user.name || user.username || user.email || `${t('domain.auto.9ba763ea34')}${user.id}`
  return user.department_name ? `${label} · ${user.department_name}` : label
}

function splitTags(value) {
  if (Array.isArray(value)) return value
  return String(value || '').split(',').map((item) => item.trim()).filter(Boolean)
}
</script>

<style scoped>
.object-access-panel {
  display: grid;
  gap: 12px;
  padding: 14px;
  border: 1px solid #e6eaf0;
  border-radius: 8px;
  background: #fbfcfe;
}

.object-access-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.object-access-head > div:first-child {
  display: grid;
  gap: 4px;
}

.object-access-head strong {
  color: #172033;
  font-size: 14px;
}

.object-access-head span,
.object-access-row span {
  color: #667085;
  font-size: 12px;
}

.object-access-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.object-access-list,
.object-acl-editor {
  display: grid;
  gap: 10px;
}

.object-access-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid #edf0f5;
  border-radius: 8px;
  background: #ffffff;
}

.object-acl-row {
  display: grid;
  grid-template-columns: minmax(240px, 1fr) 150px 70px;
  gap: 10px;
  align-items: center;
  min-height: 36px;
}

.object-acl-user-select,
.object-acl-level-select,
.object-acl-remove {
  width: 100%;
}

.object-acl-level-select {
  min-width: 150px;
}

.object-acl-remove {
  min-width: 70px;
}

.object-access-form {
  margin-top: 14px;
}

@media (max-width: 760px) {
  .object-access-head,
  .object-access-row {
    align-items: stretch;
    flex-direction: column;
  }

  .object-acl-row {
    grid-template-columns: 1fr;
  }

  .object-acl-level-select,
  .object-acl-remove {
    min-width: 100%;
  }
}
</style>
