<template>
  <section class="field-section">
    <label class="field-label">Name</label>
    <input
      :value="previewStore.currentFile?.name || ''"
      @input="updateName"
      placeholder="Enter name…"
      class="field-input"
    />
  </section>
</template>

<script setup lang="ts">
import { usePreviewStore } from '../../stores/preview'
const previewStore = usePreviewStore()
const updateName = (e: Event) => previewStore.updateName((e.target as HTMLInputElement).value)
</script>

<style scoped>
.field-section { display: flex; flex-direction: column; gap: 4px; }
.field-label { color: #888; font-size: 11px; text-transform: uppercase; }
.field-input {
  background: #2a2a2a; border: 1px solid #444; color: #fff;
  border-radius: 4px; padding: 6px 8px; font-size: 13px; outline: none;
}
.field-input:focus { border-color: #5b8af5; }
</style>
