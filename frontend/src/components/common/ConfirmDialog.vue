<template>
  <Teleport to="body">
    <div class="confirm-overlay" @click.self="cancel">
      <div class="confirm-modal">
        <div class="confirm-header">
          <div class="confirm-icon"><AlertTriangle :size="20" /></div>
          <h3>{{ title }}</h3>
        </div>
        <p class="confirm-message">{{ message }}</p>
        <div class="confirm-actions">
          <button class="cancel-btn" @click="cancel">Cancel</button>
          <button class="confirm-btn" @click="confirm">{{ confirmText }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { AlertTriangle } from '@lucide/vue'

const props = withDefaults(defineProps<{
  title?: string
  message?: string
  confirmText?: string
  cancelText?: string
}>(), {
  title: 'Confirm',
  message: 'Are you sure?',
  confirmText: 'Confirm',
  cancelText: 'Cancel',
})

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const confirm = () => emit('confirm')
const cancel = () => emit('cancel')
</script>

<style scoped>
.confirm-overlay {
  position: fixed; inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex; align-items: center; justify-content: center;
  z-index: 200;
}
.confirm-modal {
  background: #111111;
  border: 1px solid #222;
  border-radius: 12px;
  width: 340px;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
  overflow: hidden;
}
.confirm-header {
  display: flex; align-items: center; gap: 10px;
  padding: 14px 18px;
  border-bottom: 1px solid #1a1a1a;
}
.confirm-icon { color: #ef4444; }
.confirm-header h3 {
  margin: 0;
  color: #e8e8e8;
  font-size: 14px;
  font-weight: 600;
  font-family: 'Inter', sans-serif;
}
.confirm-message {
  margin: 0;
  padding: 14px 18px;
  color: #aaa;
  font-size: 13px;
  line-height: 1.5;
  font-family: 'Inter', sans-serif;
}
.confirm-actions {
  display: flex; gap: 8px;
  padding: 12px 18px;
  border-top: 1px solid #1a1a1a;
}
.cancel-btn {
  flex: 1;
  background: #1a1a1a;
  color: #ccc;
  border: 1px solid #2a2a2a;
  border-radius: 6px;
  padding: 9px;
  cursor: pointer;
  font-size: 13px;
  font-family: 'Inter', sans-serif;
  transition: background 0.15s, border-color 0.15s;
}
.cancel-btn:hover { background: #222; border-color: #333; }
.confirm-btn {
  flex: 1;
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.25);
  border-radius: 6px;
  padding: 9px;
  cursor: pointer;
  font-size: 13px;
  font-family: 'Inter', sans-serif;
  transition: background 0.15s;
}
.confirm-btn:hover { background: rgba(239, 68, 68, 0.25); }
</style>
