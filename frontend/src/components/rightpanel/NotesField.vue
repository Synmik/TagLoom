<template>
  <section class="field-section">
    <label class="field-label">Notes</label>
    <textarea
      v-model="localNotes"
      placeholder="Add notes…"
      class="field-textarea"
      rows="4"
    />
  </section>
</template>

<script setup lang="ts">
import { shallowRef, watch } from 'vue'
import { usePreviewStore } from '../../stores/preview'

const previewStore = usePreviewStore()

const localNotes = shallowRef('')

watch(
  () => previewStore.currentFile?.notes,
  (val) => { localNotes.value = val || '' },
)

watch(localNotes, (value, _prev, onCleanup) => {
  const timer = setTimeout(() => {
    previewStore.updateNotes(value)
  }, 500)
  onCleanup(() => clearTimeout(timer))
})
</script>

<style scoped>
.field-section { display: flex; flex-direction: column; gap: 6px; }
.field-label { color: #666; font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
.field-textarea {
  background: #1a1a1a; border: 1px solid #2a2a2a; color: #e8e8e8;
  border-radius: 6px; padding: 7px 10px; font-size: 13px; outline: none;
  resize: vertical;
  font-family: 'Inter', sans-serif;
  transition: border-color 0.15s;
}
.field-textarea:focus { border-color: #22c55e; }
.field-textarea::placeholder { color: #444; }
</style>
