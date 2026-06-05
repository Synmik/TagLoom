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
.field-section { display: flex; flex-direction: column; gap: 6px; }
.field-label { color: #666; font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
.field-input {
  background: #1a1a1a; border: 1px solid #2a2a2a; color: #e8e8e8;
  border-radius: 6px; padding: 7px 10px; font-size: 13px;
  font-family: 'Inter', sans-serif;
  outline: none; transition: border-color 0.15s;
}
.field-input:focus { border-color: #22c55e; }
.field-input::placeholder { color: #444; }
</style>
