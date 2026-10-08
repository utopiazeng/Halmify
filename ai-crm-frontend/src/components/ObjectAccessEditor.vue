<template>
  <section class="object-access-editor">
    <div class="access-scope-row">
      <ObjectAccessField
        :model-value="visibility"
        :public-label="publicLabel"
        :private-label="privateLabel"
        @update:model-value="emitVisibility"
      />
      <el-tag v-if="entries.length" type="info" effect="light">{{ t('domain.auto.6a385b2cfd') }} {{ entries.length }} {{ t('domain.auto.4912771a42') }}</el-tag>
    </div>
    <div v-if="isPrivate" class="access-help">
      {{ t('domain.auto.2fa135d5de') }}
    </div>
    <div v-if="isPrivate" class="access-entry-list">
      <div v-for="entry in entries" :key="entry.uid" class="access-entry-row">
        <el-select v-model="entry.user_id" filterable clearable :placeholder="t('domain.auto.1dd1098108')" class="access-user-select">
          <el-option v-for="user in users" :key="user.id" :label="formatUser(user)" :value="user.id" />
        </el-select>
        <el-select v-model="entry.access_level" :placeholder="t('domain.auto.7e6fe7f2d7')" class="access-level-select">
          <el-option :label="t('domain.auto.f7acefd2d4')" value="view" />
          <el-option :label="t('domain.auto.0a96a4a257')" value="follow" />
          <el-option :label="t('domain.auto.9263e46a01')" value="handle" />
          <el-option :label="t('domain.auto.4989b5cf94')" value="manage" />
        </el-select>
        <el-button type="danger" link @click="removeEntry(entry.uid)">{{ t('domain.auto.3755f56f2f') }}</el-button>
      </div>
      <el-button plain @click="addEntry">{{ t('domain.auto.e87fa91276') }}</el-button>
    </div>
  </section>
</template>

<script setup>
import { t } from '../i18n'
import { computed, watch } from 'vue'
import ObjectAccessField from './ObjectAccessField.vue'

const props = defineProps({
  visibility: { type: String, default: 'public' },
  entries: { type: Array, default: () => [] },
  users: { type: Array, default: () => [] },
  publicLabel: { type: String, default: t('domain.auto.dfe5a318ab') },
  privateLabel: { type: String, default: t('domain.auto.c931219c30') }
})

const emit = defineEmits(['update:visibility', 'update:entries'])

const entries = computed({
  get: () => props.entries,
  set: (value) => emit('update:entries', value)
})
const isPrivate = computed(() => props.visibility === 'private')

watch(
  () => props.visibility,
  (value) => {
    if (value !== 'private' && entries.value.length) {
      emit('update:entries', [])
    }
  }
)

function emitVisibility(value) {
  emit('update:visibility', value === 'private' ? 'private' : 'public')
}

function addEntry() {
  const next = [...entries.value, createEntry()]
  emit('update:entries', next)
}

function removeEntry(uid) {
  emit('update:entries', entries.value.filter((item) => item.uid !== uid))
}

function createEntry(value = {}) {
  return {
    uid: value.uid || `acl_${Date.now()}_${Math.random().toString(16).slice(2, 8)}`,
    user_id: value.user_id,
    access_level: value.access_level || 'view'
  }
}

function formatUser(user) {
  const label = user.name || user.username || user.email || `${t('domain.auto.9ba763ea34')}${user.id}`
  return user.department_name ? `${label} · ${user.department_name}` : label
}
</script>

<style scoped>
.object-access-editor {
  display: grid;
  gap: 10px;
  width: 100%;
}

.access-scope-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.access-help {
  color: #667085;
  font-size: 12px;
  line-height: 1.6;
}

.access-entry-list {
  display: grid;
  gap: 10px;
}

.access-entry-row {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) 140px 64px;
  gap: 10px;
  align-items: center;
}

.access-user-select,
.access-level-select {
  width: 100%;
}

@media (max-width: 720px) {
  .access-entry-row {
    grid-template-columns: 1fr;
  }
}
</style>
