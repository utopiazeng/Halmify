<template>
  <span class="art-logo-mark" :style="markStyle">{{ text }}</span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  text: { type: String, default: 'U' },
  size: { type: Number, default: 36 }
})

const markStyle = computed(() => ({
  width: `${props.size}px`,
  height: `${props.size}px`
}))
</script>
