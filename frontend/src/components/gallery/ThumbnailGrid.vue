<template>
  <div class="thumbnail-grid" :style="{ gridTemplateColumns: `repeat(auto-fill, minmax(${cellSize}px, 1fr))` }">
    <ThumbnailCell v-for="file in filesStore.files" :key="file.id" :file="file" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ThumbnailCell from './ThumbnailCell.vue'
import { useUIStore } from '../../stores/ui'
import { useFilesStore } from '../../stores/files'

const uiStore = useUIStore()
const filesStore = useFilesStore()

const cellSize = computed(() => uiStore.gridPixelSize)
</script>

<style scoped>
.thumbnail-grid {
  display: grid;
  gap: 8px;
}
</style>
