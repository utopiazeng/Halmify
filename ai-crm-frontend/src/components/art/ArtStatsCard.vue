<template>
  <div class="kpi-card art-stats-card">
    <span>{{ title }}</span>
    <strong>{{ value }}</strong>
    <small v-if="description">{{ description }}</small>
  </div>
</template>

<script setup>
defineProps({
  title: { type: String, required: true },
  value: { type: [String, Number], required: true },
  description: { type: String, default: '' }
})
</script>
