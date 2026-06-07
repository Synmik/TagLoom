<template>
  <section class="field-section">
    <label class="field-label">Link</label>
    <input
      :value="localLink"
      @input="onInput"
      placeholder="https://…"
      class="field-input"
      :class="{ 'invalid-url': isInvalid }"
    />
    <span v-if="isInvalid" class="error-msg">Enter a valid URL (e.g. https://example.com)</span>
  </section>
</template>

<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue'
import { usePreviewStore } from '../../stores/preview'

const previewStore = usePreviewStore()

const editingFileId = shallowRef<number | null>(null)
const localLink = shallowRef('')

watch(
  () => previewStore.currentFile,
  (file) => {
    editingFileId.value = file?.id ?? null
    localLink.value = file?.link || ''
  },
  { immediate: true },
)

function onInput(event: Event) {
  localLink.value = (event.target as HTMLInputElement).value
}

// Simple URL validation: empty is allowed, otherwise must be a valid URL
const isInvalid = computed(() => {
  const v = localLink.value.trim()
  if (!v) return false
  try { new URL(v) } catch { return true }
  return false
})

// Debounced auto-save (500ms) — guarded by file ID + value comparison
watch(localLink, (value, _prev, onCleanup) => {
  const currentFile = previewStore.currentFile
  if (!currentFile) return

  const currentLink = currentFile.link || ''
  // If the value matches what's already in the store, this is a
  // programmatic reset (file switch), not a user edit — skip entirely.
  if (value === currentLink) return

  const savedFileId = currentFile.id

  const timer = setTimeout(() => {
    if (previewStore.currentFile?.id !== savedFileId) return

    const trimmed = value.trim()
    // Only save if empty or a valid URL
    if (!trimmed || !isInvalid.value) {
      previewStore.updateLink(trimmed)
    }
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
.invalid-url { border-color: #ef4444 !important; }
.error-msg { color: #ef4444; font-size: 10px; margin-top: 2px; }
</style>
