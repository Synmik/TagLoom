<script setup lang="ts">
import type { ToastMessage } from '../../composables/useToast'

const props = defineProps<{
  toast: ToastMessage
}>()

defineEmits<{
  dismiss: [id: number]
}>()

const iconMap: Record<string, string> = {
  success: '✓',
  error: '✕',
  info: 'ℹ',
}
</script>

<template>
  <div class="toast" :class="toast.type" @click="$emit('dismiss', toast.id)">
    <span class="toast-icon">{{ iconMap[toast.type] || 'ℹ' }}</span>
    <span class="toast-message">{{ toast.message }}</span>
  </div>
</template>

<style scoped>
.toast {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  min-width: 260px;
  max-width: 420px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
  animation: toast-in 0.25s ease-out;
  user-select: none;
}

.toast:hover {
  opacity: 0.9;
}

.toast-icon {
  font-size: 15px;
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}

.toast-message {
  flex: 1;
  line-height: 1.4;
}

/* ── Types ──────────────────────────────────────────────────────── */
.toast.success {
  background: #1b3a2a;
  border: 1px solid #2d6b47;
  color: #7dcea0;
}

.toast.success .toast-icon {
  background: #2d6b47;
  color: #fff;
}

.toast.error {
  background: #3a1b1b;
  border: 1px solid #6b2d2d;
  color: #f19494;
}

.toast.error .toast-icon {
  background: #6b2d2d;
  color: #fff;
}

.toast.info {
  background: #1b2a3a;
  border: 1px solid #2d4a6b;
  color: #94c3f1;
}

.toast.info .toast-icon {
  background: #2d4a6b;
  color: #fff;
}

/* ── Animations ─────────────────────────────────────────────────── */
@keyframes toast-in {
  from {
    opacity: 0;
    transform: translateY(-8px) scale(0.96);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}
</style>
