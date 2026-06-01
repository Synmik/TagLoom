<template>
  <div
    ref="scrollContainerRef"
    class="list-scroll-container"
  >
    <!-- Sticky header (outside virtual area) -->
    <div class="list-header">
      <span class="col-thumb"></span>
      <span class="col-name">Name</span>
      <span class="col-tags">Tags</span>
      <span class="col-date">Date</span>
      <span class="col-size">Size</span>
      <span class="col-rating">Rating</span>
    </div>
    <!-- Virtualized content area -->
    <div class="list-body" :style="{ height: `${totalHeight}px` }">
      <div
        v-for="vi in visibleItems"
        :key="vi.item.id"
        :style="vi.style"
      >
        <ListRow :file="vi.item" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, nextTick, type Ref } from 'vue'
import ListRow from './ListRow.vue'
import { useFilesStore } from '../../stores/files'
import { useVirtualList } from '../../composables/useVirtualList'
import type { File } from '../../types/file'

const filesStore = useFilesStore()

const scrollContainerRef = ref<HTMLElement | null>(null)

const ROW_HEIGHT = 44

const filesRef: Ref<File[]> = computed(() => filesStore.files)

const { visibleItems, totalHeight, setContainerHeight, attachScroll } = useVirtualList(
  filesRef,
  ROW_HEIGHT,
  5,  // overscan
  12, // horizontal padding
)

let scrollCleanup: (() => void) | undefined
let resizeObserver: ResizeObserver | undefined

onMounted(() => {
  nextTick(() => {
    if (scrollContainerRef.value) {
      scrollCleanup = attachScroll(scrollContainerRef.value)
      // Account for header height when measuring viewport
      const headerEl = scrollContainerRef.value.querySelector('.list-header') as HTMLElement | null
      const headerHeight = headerEl?.offsetHeight ?? 37
      setContainerHeight(scrollContainerRef.value.clientHeight - headerHeight)
    }
  })

  resizeObserver = new ResizeObserver((entries) => {
    for (const entry of entries) {
      if (scrollContainerRef.value) {
        const headerEl = scrollContainerRef.value.querySelector('.list-header') as HTMLElement | null
        const headerHeight = headerEl?.offsetHeight ?? 37
        setContainerHeight(entry.contentRect.height - headerHeight)
      }
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
</script>

<style scoped>
.list-scroll-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 0;
  font-size: 13px;
  padding: 0 12px 12px;
}

.list-header {
  display: grid;
  grid-template-columns: 40px 1fr 1fr 100px 80px 60px;
  padding: 6px 8px;
  color: #888;
  font-size: 11px;
  text-transform: uppercase;
  border-bottom: 1px solid #333;
  position: sticky;
  top: 0;
  background: #121212;
  z-index: 1;
}

.list-body {
  position: relative;
  width: 100%;
}
</style>
