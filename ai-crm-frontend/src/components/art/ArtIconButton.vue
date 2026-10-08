<template>
  <button class="art-icon-button" type="button" :aria-label="label">
    <Icon v-if="icon" :icon="icon" />
    <slot />
  </button>
</template>

<script setup>
import { Icon } from '@iconify/vue'

defineProps({
  icon: { type: String, default: '' },
  label: { type: String, default: 'button' }
})
</script>
