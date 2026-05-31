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
.field-section { display: flex; flex-direction: column; gap: 4px; }
.field-label { color: #888; font-size: 11px; text-transform: uppercase; }
.field-textarea {
  background: #2a2a2a; border: 1px solid #444; color: #fff;
  border-radius: 4px; padding: 6px 8px; font-size: 13px; outline: none;
  resize: vertical; font-family: inherit;
}
.field-textarea:focus { border-color: #5b8af5; }
</style>
