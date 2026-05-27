<template>
  <section class="field-section">
    <label class="field-label">Notes</label>
    <textarea
      :value="previewStore.currentFile?.notes || ''"
      @input="updateNotes"
      placeholder="Add notes…"
      class="field-textarea"
      rows="4"
    />
  </section>
</template>

<script setup lang="ts">
import { usePreviewStore } from '../../stores/preview'
const previewStore = usePreviewStore()
const updateNotes = (e: Event) => previewStore.updateNotes((e.target as HTMLTextAreaElement).value)
</script>

<style scoped>
.field-section { display: flex; flex-direction: column; gap: 4px; }
.field-label { color: #888; font-size: 11px; text-transform: uppercase; }
.field-textarea {
  background: #2a2a2a; border: 1px solid #444; color: #fff;
  border-radius: 4px; padding: 6px 8px; font-size: 13px; outline: none;
  resize: vertical; font-family: inherit;
}
.field-textarea:focus { border-color: #5b8af5; }
</style>
