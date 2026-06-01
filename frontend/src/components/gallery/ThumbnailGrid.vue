<template>
  <div
    ref="scrollContainerRef"
    class="grid-scroll-container"
  >
    <div class="thumbnail-grid" :style="{ height: `${totalHeight}px` }">
      <div
        v-for="cell in visibleCells"
        :key="cell.file.id"
        :style="cell.style"
      >
        <ThumbnailCell :file="cell.file" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, nextTick, type Ref } from 'vue'
import ThumbnailCell from './ThumbnailCell.vue'
import { useUIStore } from '../../stores/ui'
import { useFilesStore } from '../../stores/files'
import { useVirtualGrid } from '../../composables/useVirtualGrid'
import type { File } from '../../types/file'

const uiStore = useUIStore()
const filesStore = useFilesStore()

const scrollContainerRef = ref<HTMLElement | null>(null)

const cellSizeRef = computed(() => uiStore.gridPixelSize)
const filesRef: Ref<File[]> = computed(() => filesStore.files)

const { visibleCells, totalHeight, setContainerSize, attachScroll } = useVirtualGrid(
  filesRef,
  cellSizeRef,
  8,   // gap between cells
  3,   // overscan rows
  12,  // padding (matches .thumbnail-grid padding)
)

let scrollCleanup: (() => void) | undefined
let resizeObserver: ResizeObserver | undefined

onMounted(() => {
  nextTick(() => {
    if (scrollContainerRef.value) {
      scrollCleanup = attachScroll(scrollContainerRef.value)
      setContainerSize(
        scrollContainerRef.value.clientWidth,
        scrollContainerRef.value.clientHeight,
      )
    }
  })

  // Watch for container size changes
  resizeObserver = new ResizeObserver((entries) => {
    for (const entry of entries) {
      setContainerSize(entry.contentRect.width, entry.contentRect.height)
    }
  })
  if (scrollContainerRef.value) {
    resizeObserver.observe(scrollContainerRef.value)
  }
})

onUnmounted(() => {
  scrollCleanup?.()
  resizeObserver?.disconnect()
})

// Re-attach when grid size changes (affects column count)
watch(() => uiStore.gridSize, () => {
  if (scrollContainerRef.value) {
    setContainerSize(
      scrollContainerRef.value.clientWidth,
      scrollContainerRef.value.clientHeight,
    )
  }
})
</script>

<style scoped>
.grid-scroll-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 0;
}

.thumbnail-grid {
  position: relative;
  width: 100%;
  min-height: 1px;
  padding: 12px;
}
</style>
