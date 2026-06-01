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
        <span v-if="filesStore.isLoading" class="loading-more">Loading more…</span>
      </div>
    </div>

    <!-- Grid / List view -->
    <ThumbnailGrid v-if="uiStore.viewMode === 'grid'" />
    <ListView v-else />

    <!-- Infinite scroll sentinel -->
    <div ref="sentinel" class="sentinel"></div>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
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
const { loadingMore, loadMore, resetPage, observeSentinel } = usePagination()

const galleryRef = ref<HTMLElement | null>(null)
const sentinel = ref<HTMLElement | null>(null)

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
  icon: string
  title: string
  description: string
  actionText?: string
  actionVariant?: 'primary' | 'secondary'
}

const emptyState = computed<EmptyStateConfig>(() => {
  // No vault open at all
  if (!vaultStore.currentVault) {
    return {
      icon: '📂',
      title: 'No vault open',
      description: 'Open or create a vault to get started',
      actionText: 'Open Vault',
      actionVariant: 'primary',
    }
  }

  // Vault open — search returned no results
  if (filtersStore.hasActiveSearch) {
    return {
      icon: '🔍',
      title: 'No results found',
      description: 'No files match your search query',
      actionText: 'Clear search',
      actionVariant: 'secondary',
    }
  }

  // Vault open — active filters but no matching files (totalCount === 0)
  if (filtersStore.hasActiveFilters) {
    return {
      icon: '🔍',
      title: 'No matching files',
      description: 'No files match your current filters',
      actionText: 'Clear filters',
      actionVariant: 'secondary',
    }
  }

  // Vault open but genuinely empty
  return {
    icon: '📁',
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

let cleanup: (() => void) | undefined = undefined

onMounted(async () => {
  await loadGallery()
  cleanup = observeSentinel(sentinel.value)
})

onUnmounted(() => cleanup?.())

// Reload when non-search filters change (searchQuery is handled by useSearch composable)
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
  },
  { deep: true }
)

// Reload after scan completes
watch(
  () => vaultStore.isScanning,
  async (wasScanning, isScanning) => {
    if (wasScanning && !isScanning) {
      resetPage()
      await loadGallery()
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
      // Vault opened — load files (auto-scan may refill later)
      resetPage()
      await loadGallery()
    }
  }
)

async function loadGallery() {
  await filesStore.reloadFiles()
}
</script>

<style scoped>
.gallery {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  background: #121212;
  display: flex;
  flex-direction: column;
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
  padding: 8px 4px;
  font-size: 12px;
  color: #888;
  border-bottom: 1px solid #222;
  margin-bottom: 8px;
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

.sentinel {
  height: 1px;
  flex-shrink: 0;
}
</style>
