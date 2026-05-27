<template>
  <section class="metadata-section">
    <label class="field-label">Metadata</label>
    <div class="metadata-grid">
      <div class="meta-row"><span class="meta-label">Filename</span><span class="meta-value">{{ meta?.filename || '—' }}</span></div>
      <div class="meta-row"><span class="meta-label">Date Created</span><span class="meta-value">{{ meta?.date_created || '—' }}</span></div>
      <div class="meta-row"><span class="meta-label">Date Modified</span><span class="meta-value">{{ meta?.date_modified || '—' }}</span></div>
      <div class="meta-row"><span class="meta-label">File Size</span><span class="meta-value">{{ formatSize }}</span></div>
      <div class="meta-row"><span class="meta-label">File Format</span><span class="meta-value">{{ meta?.format_name || '—' }}</span></div>
      <div class="meta-row"><span class="meta-label">Resolution</span><span class="meta-value">{{ resolution }}</span></div>
      <div class="meta-row"><span class="meta-label">Duration</span><span class="meta-value">{{ duration }}</span></div>
    </div>

    <div class="rating-row">
      <span class="meta-label">Rating</span>
      <StarRating :rating="previewStore.currentFile?.rating || 0" @change="setRating" />
    </div>

    <div class="favorite-row">
      <button
        class="favorite-btn"
        :class="{ active: previewStore.currentFile?.is_favorite === 1 }"
        @click="toggleFavorite"
      >
        {{ previewStore.currentFile?.is_favorite === 1 ? '♥ Favorite' : '♡ Favorite' }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import StarRating from '../common/StarRating.vue'
import { usePreviewStore } from '../../stores/preview'

const previewStore = usePreviewStore()
const meta = computed(() => previewStore.metadata)

const formatSize = computed(() => {
  if (!meta.value?.size_bytes) return '—'
  const bytes = meta.value.size_bytes
  if (bytes > 1_000_000) return (bytes / 1_000_000).toFixed(1) + ' MB'
  if (bytes > 1_000) return (bytes / 1_000).toFixed(1) + ' KB'
  return bytes + ' B'
})

const resolution = computed(() => {
  const m = meta.value
  if (!m?.resolution_width) return '—'
  return `${m.resolution_width} × ${m.resolution_height}`
})

const duration = computed(() => {
  const s = meta.value?.duration_seconds
  if (!s || s === 0) return '—'
  const min = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${min}:${sec.toString().padStart(2, '0')}`
})

const setRating = (r: number) => previewStore.setRating(r)
const toggleFavorite = () => previewStore.toggleFavorite()
</script>

<style scoped>
.metadata-section { display: flex; flex-direction: column; gap: 6px; }
.field-label { color: #888; font-size: 11px; text-transform: uppercase; }
.metadata-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 4px; }
.meta-row { display: flex; flex-direction: column; gap: 1px; }
.meta-label { color: #666; font-size: 10px; }
.meta-value { color: #ccc; font-size: 12px; }
.rating-row { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
.favorite-row { margin-top: 4px; }
.favorite-btn {
  background: none; border: 1px solid #444; color: #888;
  border-radius: 4px; padding: 4px 10px; cursor: pointer; font-size: 12px;
}
.favorite-btn.active { color: #ff4444; border-color: #ff4444; }
</style>
