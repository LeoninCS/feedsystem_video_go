<script setup lang="ts">
import { useToastStore } from '../stores/toast'

const toast = useToastStore()
</script>

<template>
  <div class="toast-wrap" aria-live="polite" aria-relevant="additions removals">
    <div v-for="t in toast.toasts" :key="t.id" class="toast" :class="t.type">
      <div class="toast-msg">{{ t.message }}</div>
      <button class="toast-x" type="button" aria-label="关闭" @click="toast.remove(t.id)">×</button>
    </div>
  </div>
</template>

<style scoped>
.toast-wrap {
  position: fixed;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  display: grid;
  gap: 10px;
  z-index: 200;
  width: min(520px, calc(100vw - 24px));
  pointer-events: none;
}

.toast {
  pointer-events: auto;
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 10px;
  align-items: center;
  border-radius: 14px;
  padding: 10px 12px;
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(0, 0, 0, 0.55);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.35);
}

.toast.success {
  border-color: rgba(34, 197, 94, 0.35);
}

.toast.error {
  border-color: rgba(254, 44, 85, 0.45);
}

.toast.info {
  border-color: rgba(37, 244, 238, 0.3);
}

.toast-msg {
  font-size: 13px;
  line-height: 1.35;
  color: rgba(255, 255, 255, 0.92);
}

.toast-x {
  width: 30px;
  height: 30px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.88);
  cursor: pointer;
  font-size: 18px;
  line-height: 1;
  padding: 0;
}
</style>
