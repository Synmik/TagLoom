<template>
  <div
    ref="scrollContainerRef"
    class="list-scroll-container"
  >
    <!-- Sticky header (outside virtual area) -->
    <div class="list-header">
      <span class="col-name">Name</span>
      <span class="col-tags">Tags</span>
      <span class="col-date">Modified</span>
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

      <!-- Sentinel: triggers loading next page when user scrolls near end of loaded data -->
      <div
        v-if="hasMore"
        ref="sentinelRef"
        class="load-sentinel"
        :style="{ position: 'absolute', top: `${sentinelTop}px`, left: '0', width: '100%', height: '1px' }"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, nextTick, type Ref } from 'vue'
import ListRow from './ListRow.vue'
import { useFilesStore } from '../../stores/files'
import { useVirtualList } from '../../composables/useVirtualList'
import { usePagination } from '../../composables/usePagination'
import type { File } from '../../types/file'

const filesStore = useFilesStore()
const { loadMore, hasMore } = usePagination()

const scrollContainerRef = ref<HTMLElement | null>(null)
const sentinelRef = ref<HTMLElement | null>(null)

const ROW_HEIGHT = 44

const filesRef: Ref<File[]> = computed(() => filesStore.files)
const totalCountRef = computed(() => filesStore.totalCount)

const { visibleItems, totalHeight, setContainerHeight, attachScroll, scrollTo } = useVirtualList(
  filesRef,
  ROW_HEIGHT,
  5,  // overscan
  12, // horizontal padding
  totalCountRef,
)

// Position sentinel near the last LOADED file
const sentinelTop = computed(() => {
  const loadedCount = filesStore.files.length
  return loadedCount * ROW_HEIGHT
})

let scrollCleanup: (() => void) | undefined
let resizeObserver: ResizeObserver | undefined
let sentinelObserver: IntersectionObserver | undefined

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

  // Sentinel: load next page when user scrolls near end of loaded data
  sentinelObserver = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          loadMore()
        }
      }
    },
    { root: null, rootMargin: '200px', threshold: 0 }
  )
})

onUnmounted(() => {
  scrollCleanup?.()
  resizeObserver?.disconnect()
  sentinelObserver?.disconnect()
})

// ── Scroll to top (called by Gallery on sort/filter change) ──
function scrollToTop() {
  if (scrollContainerRef.value) {
    scrollContainerRef.value.scrollTop = 0
  }
}

defineExpose({ scrollToTop })

// Observe/unobserve sentinel element when it appears/disappears.
// Use nextTick to wait for the v-if DOM update — otherwise sentinelRef
// is still null when hasMore flips to true on first page load.
watch(
  () => hasMore.value,
  async (more) => {
    await nextTick()
    if (more && sentinelRef.value && sentinelObserver) {
      sentinelObserver.observe(sentinelRef.value)
    } else {
      sentinelObserver?.disconnect()
    }
  }
)

// Re-observe when files array changes (new page loaded).
// nextTick ensures the sentinel element is in the DOM before observing.
watch(
  () => filesStore.files.length,
  async () => {
    await nextTick()
    if (hasMore.value && sentinelRef.value && sentinelObserver) {
      sentinelObserver.disconnect()
      sentinelObserver.observe(sentinelRef.value)
    }
  }
)
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
  grid-template-columns: 1fr 1fr 160px 60px;
  padding: 8px 10px;
  color: #666;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border-bottom: 1px solid #1a1a1a;
  position: sticky;
  top: 0;
  background: #0d0d0d;
  z-index: 1;
}

.list-body {
  position: relative;
  width: 100%;
}
</style>
