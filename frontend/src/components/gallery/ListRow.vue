<template>
  <div
    class="list-row"
    :class="{ selected: isSelected(file) }"
    @click="(e) => handleClick(e)"
    @dblclick="openPreview"
  >
    <span class="col-thumb"><img :src="thumbnailUrl" class="thumb" /></span>
    <span class="col-name">{{ filename }}</span>
    <span class="col-tags">{{ tagsText }}</span>
    <span class="col-date">{{ file.indexed_at }}</span>
    <span class="col-size">{{ fileSize }}</span>
    <span class="col-rating">{{ '★'.repeat(file.rating) }}{{ '☆'.repeat(5 - file.rating) }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useSelection } from '../../composables/useSelection'
import { usePreviewStore } from '../../stores/preview'
import type { File } from '../../types/file'

const props = defineProps<{ file: File }>()
const { isSelected, toggleSelection } = useSelection()
const previewStore = usePreviewStore()

const filename = computed(() => props.file.vault_path.split(/[\\/]/).pop() || '')
const thumbnailUrl = computed(() => props.file.thumbnail_path || '')
const tagsText = computed(() => '—') // TODO: Load tags for this file
const fileSize = computed(() => '—') // TODO: Fetch from metadata

const handleClick = (e: MouseEvent) => {
  if (!e.ctrlKey && !e.shiftKey) {
    previewStore.setFile(props.file)
  }
  toggleSelection(props.file, e.ctrlKey, e.shiftKey)
}

const openPreview = () => previewStore.setFile(props.file)
</script>

<style scoped>
.list-row {
  display: grid; grid-template-columns: 40px 1fr 1fr 100px 80px 60px;
  padding: 4px 8px; align-items: center; cursor: pointer; border-radius: 4px;
}
.list-row:hover { background: #1e1e1e; }
.list-row.selected { background: #2a3a5a; }
.thumb { width: 32px; height: 32px; object-fit: cover; border-radius: 3px; }
.col-name { color: #ddd; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-tags { color: #888; font-size: 12px; }
.col-date, .col-size, .col-rating { color: #666; font-size: 12px; }
</style>
