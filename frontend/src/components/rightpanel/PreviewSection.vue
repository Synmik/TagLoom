<template>
  <section class="preview-section">
    <div class="preview-image" @click="openFullPreview">
      <img v-if="imageUrl" :src="imageUrl" alt="Preview" />
      <span v-else class="no-preview">No preview</span>
    </div>
    <div class="format-info">
      <span class="format-badge">{{ formatName }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { usePreviewStore } from '../../stores/preview'

const previewStore = usePreviewStore()

const imageUrl = computed(() => {
  const file = previewStore.currentFile
  if (!file) return ''
  // Use the original file via the HTTP endpoint
  return `/api/original/${file.id}`
})

const formatName = computed(() => {
  const file = previewStore.currentFile
  if (!file) return ''
  const ext = file.vault_path.split('.').pop()?.toUpperCase() || ''
  return ext === 'JPG' || ext === 'JPEG' ? 'JPEG' : ext
})

const openFullPreview = () => {
  // TODO: Open FilePreviewModal
}
</script>

<style scoped>
.preview-section { text-align: center; }
.preview-image {
  aspect-ratio: 1; background: #111; border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; overflow: hidden;
}
.preview-image img { max-width: 100%; max-height: 100%; object-fit: contain; }
.no-preview { color: #444; font-size: 12px; }
.format-info { margin-top: 6px; }
.format-badge {
  background: #2a2a2a; color: #aaa; font-size: 10px;
  padding: 2px 8px; border-radius: 3px; text-transform: uppercase;
}
</style>
