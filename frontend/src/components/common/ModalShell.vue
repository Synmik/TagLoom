<script setup lang="ts">
import { X } from "@lucide/vue";

defineProps<{
  /** Header title of the modal. */
  title: string;
  /** CSS width for the modal box (default 480px). */
  width?: string;
}>();

defineEmits<{ close: [] }>();
</script>

<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal" :style="width ? { width } : undefined">
      <div class="modal-header">
        <h3>{{ title }}</h3>
        <button class="close-btn" @click="$emit('close')"><X :size="16" /></button>
      </div>
      <div class="modal-body">
        <slot />
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  background: #111111;
  border-radius: 12px;
  width: 480px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  border: 1px solid #222;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 18px;
  border-bottom: 1px solid #1a1a1a;
}

.modal-header h3 {
  margin: 0;
  color: #e8e8e8;
  font-size: 14px;
  font-weight: 600;
  font-family: "Inter", sans-serif;
}

.close-btn {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  font-size: 16px;
  padding: 4px;
  border-radius: 4px;
  transition:
    color 0.15s,
    background 0.15s;
}

.close-btn:hover {
  color: #e8e8e8;
  background: #1a1a1a;
}

.modal-body {
  padding: 18px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 18px;
}
</style>
