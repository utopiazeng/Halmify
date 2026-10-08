<template>
  <el-radio-group
    :model-value="normalizedValue"
    class="object-access-field"
    @update:model-value="emitValue"
  >
    <el-radio-button label="public">{{ publicText }}</el-radio-button>
    <el-radio-button label="private">{{ privateText }}</el-radio-button>
  </el-radio-group>
</template>

<script setup>
import { t } from '../i18n'
import { computed } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: 'public' },
  publicLabel: { type: String, default: t('domain.auto.dfe5a318ab') },
  privateLabel: { type: String, default: t('domain.auto.c931219c30') }
})

const emit = defineEmits(['update:modelValue'])

const normalizedValue = computed(() => props.modelValue === 'private' ? 'private' : 'public')
const publicText = computed(() => props.publicLabel || t('domain.auto.dfe5a318ab'))
const privateText = computed(() => props.privateLabel || t('domain.auto.c931219c30'))

function emitValue(value) {
  emit('update:modelValue', value === 'private' ? 'private' : 'public')
}
</script>

<style scoped>
.object-access-field {
  width: 100%;
}

.object-access-field :deep(.el-radio-button__inner) {
  min-width: 96px;
}
</style>
