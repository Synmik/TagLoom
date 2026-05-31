<template>
  <section class="field-section">
    <label class="field-label">Link</label>
    <input
      v-model="localLink"
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

const localLink = shallowRef('')

watch(
  () => previewStore.currentFile?.link,
  (val) => { localLink.value = val || '' },
)

// Simple URL validation: empty is allowed, otherwise must be a valid URL
const isInvalid = computed(() => {
  const v = localLink.value.trim()
  if (!v) return false
  try { new URL(v) } catch { return true }
  return false
})

watch(localLink, (value, _prev, onCleanup) => {
  const timer = setTimeout(() => {
    const trimmed = value.trim()
    // Only save if empty or valid
    if (!trimmed || !isInvalid.value) {
      previewStore.updateLink(trimmed)
    }
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
.invalid-url { border-color: #e55 !important; }
.error-msg { color: #e55; font-size: 10px; margin-top: 2px; }
</style>
