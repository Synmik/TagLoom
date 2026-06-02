<template>
  <section class="preview-section">
    <div class="preview-container" @dblclick="previewStore.openFullPreview">
      <!-- Image preview -->
      <img
        v-if="isImage"
        :src="imageUrl"
        alt="Preview"
        class="preview-media"
      />
      <!-- Video player -->
      <video
        v-else-if="isVideo"
        :src="videoUrl"
        controls
        muted
        preload="metadata"
        class="preview-media"
      />
      <!-- Fallback -->
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

const imageExtensions = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.tiff', '.tif', '.svg'])
const videoExtensions = new Set(['.mp4', '.mov', '.avi', '.webm', '.mkv', '.wmv', '.flv', '.m4v', '.3gp', '.3g2', '.vob', '.ogv', '.mpg', '.mpeg', '.m2v', '.ts', '.mts', '.m2ts', '.asf', '.rm', '.amv', '.f4v', '.dv', '.mxf'])

const fileExt = computed(() => {
  const file = previewStore.currentFile
  if (!file) return ''
  return file.vault_path.split('.').pop()?.toLowerCase() || ''
})

const isImage = computed(() => imageExtensions.has('.' + fileExt.value))
const isVideo = computed(() => videoExtensions.has('.' + fileExt.value))

const imageUrl = computed(() => {
  const file = previewStore.currentFile
  if (!file) return ''
  return `/api/original/${file.id}`
})

const videoUrl = computed(() => {
  const file = previewStore.currentFile
  if (!file) return ''
  return `/api/original/${file.id}`
})

const formatName = computed(() => {
  const file = previewStore.currentFile
  if (!file) return ''
  const ext = file.vault_path.split('.').pop()?.toUpperCase() || ''
  return ext === 'JPG' || ext === 'JPEG' ? 'JPEG' : ext
})
</script>

<style scoped>
.preview-section { text-align: center; }
.preview-container {
  aspect-ratio: 1; background: #111; border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; overflow: hidden; position: relative;
}
.preview-media {
  max-width: 100%; max-height: 100%; object-fit: contain;
}
.preview-container video {
  width: 100%; height: 100%;
}
.no-preview { color: #444; font-size: 12px; }
.format-info { margin-top: 6px; }
.format-badge {
  background: #2a2a2a; color: #aaa; font-size: 10px;
  padding: 2px 8px; border-radius: 3px; text-transform: uppercase;
}
</style>
