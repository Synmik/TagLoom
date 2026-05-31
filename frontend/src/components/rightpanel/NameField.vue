<template>
  <section class="field-section">
    <label class="field-label">Name</label>
    <input
      v-model="localName"
      placeholder="Enter name…"
      class="field-input"
    />
  </section>
</template>

<script setup lang="ts">
import { shallowRef, watch } from 'vue'
import { usePreviewStore } from '../../stores/preview'

const previewStore = usePreviewStore()

// Local copy — synced from store when file changes
const localName = shallowRef('')

watch(
  () => previewStore.currentFile?.name,
  (val) => { localName.value = val || '' },
)

// Debounced auto-save (500ms)
watch(localName, (value, _prev, onCleanup) => {
  const timer = setTimeout(() => {
    previewStore.updateName(value)
  }, 500)
  onCleanup(() => clearTimeout(timer))
})
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
