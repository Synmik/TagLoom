<template>
  <div ref="scrollContainerRef" class="grid-scroll-container">
    <div class="thumbnail-grid" :style="{ height: `${totalHeight}px` }">
      <div v-for="cell in visibleCells" :key="cell.file.id" :style="cell.style">
        <ThumbnailCell :file="cell.file" />
      </div>

      <!-- Sentinel: triggers loading next page when user scrolls near end of loaded data -->
      <div
        v-if="hasMore"
        ref="sentinelRef"
        class="load-sentinel"
        :style="{
          position: 'absolute',
          top: `${sentinelTop}px`,
          left: '0',
          width: '100%',
          height: '1px',
        }"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, nextTick, type Ref } from "vue";
import ThumbnailCell from "./ThumbnailCell.vue";
import { useUIStore } from "../../stores/ui";
import { useFilesStore } from "../../stores/files";
import { useVirtualGrid } from "../../composables/useVirtualGrid";
import { usePagination } from "../../composables/usePagination";
import type { File } from "../../types/file";

const uiStore = useUIStore();
const filesStore = useFilesStore();
const { loadMore, hasMore } = usePagination();

const scrollContainerRef = ref<HTMLElement | null>(null);
const sentinelRef = ref<HTMLElement | null>(null);

const cellSizeRef = computed(() => uiStore.gridPixelSize);
const filesRef: Ref<File[]> = computed(() => filesStore.files);
const totalCountRef = computed(() => filesStore.totalCount);

const { visibleCells, totalHeight, numColumns, setContainerSize, attachScroll, scrollTo } =
  useVirtualGrid(
    filesRef,
    cellSizeRef,
    8, // gap between cells
    3, // overscan rows
    12, // padding (matches .thumbnail-grid padding)
    totalCountRef,
  );

// Position sentinel near the last LOADED file (not total count).
// When user scrolls to this position, trigger loading next page.
const rowHeight = computed(() => uiStore.gridPixelSize + 28);
const sentinelTop = computed(() => {
  const loadedCount = filesStore.files.length;
  if (loadedCount === 0) return 0;
  const cols = numColumns.value;
  const lastLoadedRow = Math.ceil(loadedCount / cols);
  return 12 + lastLoadedRow * rowHeight.value; // 12 = padding
});

let scrollCleanup: (() => void) | undefined;
let resizeObserver: ResizeObserver | undefined;
let sentinelObserver: IntersectionObserver | undefined;

onMounted(() => {
  nextTick(() => {
    if (scrollContainerRef.value) {
      scrollCleanup = attachScroll(scrollContainerRef.value);
      setContainerSize(scrollContainerRef.value.clientWidth, scrollContainerRef.value.clientHeight);
    }

    // Observe sentinel immediately if hasMore is already true on mount.
    // The watch(hasMore) below is lazy (doesn't fire on initial creation),
    // so without this the sentinel is never observed when ThumbnailGrid
    // mounts with data already loaded.
    if (hasMore.value && sentinelRef.value && sentinelObserver) {
      sentinelObserver.observe(sentinelRef.value);
    }
  });

  // Watch for container size changes
  resizeObserver = new ResizeObserver((entries) => {
    for (const entry of entries) {
      setContainerSize(entry.contentRect.width, entry.contentRect.height);
    }
  });
  if (scrollContainerRef.value) {
    resizeObserver.observe(scrollContainerRef.value);
  }

  // Sentinel: load next page when user scrolls near end of loaded data
  sentinelObserver = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          loadMore();
        }
      }
    },
    { root: null, rootMargin: "200px", threshold: 0 },
  );
});

onUnmounted(() => {
  scrollCleanup?.();
  resizeObserver?.disconnect();
  sentinelObserver?.disconnect();
});

// ── Scroll to top (called by Gallery on sort/filter change) ──
function scrollToTop() {
  if (scrollContainerRef.value) {
    scrollContainerRef.value.scrollTop = 0;
  }
}

defineExpose({ scrollToTop });

// Observe/unobserve sentinel element when it appears/disappears.
// Use nextTick to wait for the v-if DOM update — otherwise sentinelRef
// is still null when hasMore flips to true on first page load.
watch(
  () => hasMore.value,
  async (more) => {
    await nextTick();
    if (more && sentinelRef.value && sentinelObserver) {
      sentinelObserver.observe(sentinelRef.value);
    } else {
      sentinelObserver?.disconnect();
    }
  },
);

// Re-observe when files array changes (new page loaded).
// nextTick ensures the sentinel element is in the DOM before observing.
watch(
  () => filesStore.files.length,
  async () => {
    await nextTick();
    if (hasMore.value && sentinelRef.value && sentinelObserver) {
      sentinelObserver.disconnect();
      sentinelObserver.observe(sentinelRef.value);
    }
  },
);

// When grid size changes, measure the container BEFORE Vue renders so
// visibleCells computes with both the new cellSize and the correct
// containerWidth simultaneously. Otherwise Vue renders cells with the
// new cellSize but old column count, causing overlap.
watch(
  () => uiStore.gridSize,
  () => {
    scrollTo(0);
    if (scrollContainerRef.value) {
      scrollContainerRef.value.scrollTop = 0;
      setContainerSize(scrollContainerRef.value.clientWidth, scrollContainerRef.value.clientHeight);
    }
  },
);
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
