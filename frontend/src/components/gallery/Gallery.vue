<template>
  <main class="gallery" ref="galleryRef" tabindex="-1">
    <!-- Loading state -->
    <div v-if="filesStore.isLoading && (!filesStore.files || filesStore.files.length === 0)" class="loading">
      <div class="spinner"></div>
      <span>Loading files…</span>
    </div>

    <!-- Empty state -->
    <EmptyState
      v-else-if="(!filesStore.files || filesStore.files.length === 0) && !filesStore.isLoading"
      :icon="emptyState.icon"
      :title="emptyState.title"
      :description="emptyState.description"
      :action-text="emptyState.actionText"
      :action-variant="emptyState.actionVariant"
      @action="handleEmptyStateAction"
    />

    <!-- File count bar -->
    <div v-if="filesStore.files && filesStore.files.length > 0" class="file-count-bar">
      <span>{{ filesStore.files.length }} / {{ filesStore.totalCount }} files</span>
      <div class="count-bar-right">
        <button
          v-if="filesStore.hasSelection"
          class="batch-edit-btn"
          @click="onBatchEdit"
        >
          Batch Edit ({{ filesStore.selectionCount }})
        </button>
        <span v-if="loadingMore" class="loading-more">Loading more…</span>
      </div>
    </div>

    <!-- Grid / List view (each has its own virtual scrolling container).
         Key on gridSize so the grid component is fully torn down & rebuilt,
         eliminating any possibility of stale DOM overlapping cells. -->
    <ThumbnailGrid
      v-if="uiStore.viewMode === 'grid'"
      :key="'grid-' + uiStore.gridSize"
      ref="gridRef"
    />
    <ListView v-else ref="listRef" />
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, type Component } from 'vue'
import { FolderOpen, Search, FolderSearch } from '@lucide/vue'
import ThumbnailGrid from './ThumbnailGrid.vue'
import ListView from './ListView.vue'
import EmptyState from '../common/EmptyState.vue'
import { useUIStore } from '../../stores/ui'
import { useFilesStore } from '../../stores/files'
import { useVaultStore } from '../../stores/vault'
import { useFiltersStore } from '../../stores/filters'
import { useTagsStore } from '../../stores/tags'
import { usePagination } from '../../composables/usePagination'
import { useKeyboardShortcuts } from '../../composables/useKeyboardShortcuts'

const uiStore = useUIStore()
const filesStore = useFilesStore()
const vaultStore = useVaultStore()
const filtersStore = useFiltersStore()
const tagsStore = useTagsStore()
const { loadingMore, loadMore, resetPage } = usePagination()

const galleryRef = ref<HTMLElement | null>(null)
const gridRef = ref<InstanceType<typeof ThumbnailGrid> | null>(null)
const listRef = ref<InstanceType<typeof ListView> | null>(null)

/** Scroll the active gallery view to the top */
function scrollGalleryToTop() {
  const activeView = uiStore.viewMode === 'grid' ? gridRef.value : listRef.value
  activeView?.scrollToTop()
}

// ── Keyboard shortcuts ────────────────────────────────────────────
const shortcuts = useKeyboardShortcuts()

shortcuts.on('navigate:scroll', () => {
  const selectedId = filesStore.selectedFiles[0]?.id
  if (!selectedId || !galleryRef.value) return
  const cell = galleryRef.value.querySelector(`[data-file-id="${selectedId}"]`) as HTMLElement | null
  cell?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
})

const handleFocusGallery = () => {
  galleryRef.value?.focus()
}

onMounted(() => {
  window.addEventListener('tagloom:focus-gallery', handleFocusGallery)
})

onUnmounted(() => {
  window.removeEventListener('tagloom:focus-gallery', handleFocusGallery)
})

function onBatchEdit() {
  tagsStore.loadTags()
  uiStore.openBatchEdit()
}

interface EmptyStateConfig {
  icon: Component | null
  title: string
  description: string
  actionText?: string
  actionVariant?: 'primary' | 'secondary'
}

const emptyState = computed<EmptyStateConfig>(() => {
  if (!vaultStore.currentVault) {
    return {
      icon: FolderOpen,
      title: 'No vault open',
      description: 'Open or create a vault to get started',
      actionText: 'Open Vault',
      actionVariant: 'primary',
    }
  }

  if (filtersStore.hasActiveSearch) {
    return {
      icon: Search,
      title: 'No results found',
      description: 'No files match your search query',
      actionText: 'Clear search',
      actionVariant: 'secondary',
    }
  }

  if (filtersStore.hasActiveFilters) {
    return {
      icon: Search,
      title: 'No matching files',
      description: 'No files match your current filters',
      actionText: 'Clear filters',
      actionVariant: 'secondary',
    }
  }

  return {
    icon: FolderSearch,
    title: 'No files in this vault',
    description: 'The vault appears empty — rescan to index files',
    actionText: 'Rescan vault',
    actionVariant: 'secondary',
  }
})

function handleEmptyStateAction() {
  const state = emptyState.value

  if (state.actionText === 'Open Vault') {
    vaultStore.pickAndOpenVault()
  } else if (state.actionText === 'Clear filters') {
    filtersStore.clearFilters()
    resetPage()
    loadGallery()
  } else if (state.actionText === 'Clear search') {
    filtersStore.activeFilters.searchQuery = ''
    filesStore.loadFiles()
  } else if (state.actionText === 'Rescan vault') {
    vaultStore.rescanVault()
  }
}

onMounted(async () => {
  await loadGallery()
})

// Load more pages as user scrolls — triggered by ThumbnailGrid's IntersectionObserver
// (no eager loading of all files anymore)

// Reload when non-search filters change
watch(
  () => ({
    folderPath: filtersStore.activeFilters.folderPath,
    tagIds: [...filtersStore.activeFilters.tagIds],
    fileFormats: [...filtersStore.activeFilters.fileFormats],
    minRating: filtersStore.activeFilters.minRating,
    favoritesOnly: filtersStore.activeFilters.favoritesOnly,
  }),
  async () => {
    resetPage()
    await loadGallery()
    scrollGalleryToTop()
  },
  { deep: true }
)

// Reload when thumbnail size changes — forces a clean re-render of all
// grid cells with the new size, avoiding stale-position collapse.
watch(
  () => uiStore.gridSize,
  async () => {
    resetPage()
    await loadGallery()
    scrollGalleryToTop()
  }
)

// Scroll to top whenever page resets (sort / vault open etc. via reloadFiles)
watch(
  () => filesStore.page,
  (page) => {
    if (page === 0) scrollGalleryToTop()
  }
)

// Reload after scan completes
watch(
  () => vaultStore.isScanning,
  async (wasScanning, isScanning) => {
    if (wasScanning && !isScanning) {
      resetPage()
      await loadGallery()
      scrollGalleryToTop()
    }
  }
)

// Clear gallery when vault is closed; reload when vault is opened
watch(
  () => vaultStore.currentVault,
  async (vault) => {
    if (!vault) {
      filesStore.files = []
      filesStore.selectedFiles = []
      filesStore.totalCount = 0
      filesStore.page = 0
      filesStore.clearThumbnailCache()
      filtersStore.clearFilters()
    } else {
      resetPage()
      await loadGallery()
      scrollGalleryToTop()
    }
  }
)

async function loadGallery() {
  // Load first page for quick initial render.
  // Remaining pages are loaded incrementally as user scrolls.
  await filesStore.reloadFiles()
}
</script>

<style scoped>
.gallery {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #121212;
  min-height: 0;
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  flex: 1;
  color: #666;
  padding: 40px;
  font-size: 14px;
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid #333;
  border-top-color: #5b8af5;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.file-count-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  font-size: 12px;
  color: #888;
  border-bottom: 1px solid #222;
  margin-bottom: 0;
  flex-shrink: 0;
}
.count-bar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.batch-edit-btn {
  background: #5b8af5;
  color: #fff;
  border: none;
  border-radius: 4px;
  padding: 4px 10px;
  font-size: 12px;
  cursor: pointer;
}
.batch-edit-btn:hover {
  background: #4a7ae4;
}

.loading-more {
  color: #5b8af5;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
