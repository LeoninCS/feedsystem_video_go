<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ value: unknown }>()

const text = computed(() => {
  try {
    return JSON.stringify(props.value, null, 2)
  } catch {
    return String(props.value)
  }
})
</script>

<template>
  <pre class="pre mono">{{ text }}</pre>
</template>
