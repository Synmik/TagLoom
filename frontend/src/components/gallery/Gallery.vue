<template>
  <main class="gallery" ref="galleryRef">
    <!-- Loading state -->
    <div v-if="filesStore.isLoading && filesStore.files.length === 0" class="loading">
      <div class="spinner"></div>
      <span>Loading files…</span>
    </div>

    <!-- Empty state -->
    <EmptyState
      v-else-if="filesStore.files.length === 0 && !filesStore.isLoading"
      :message="vaultStore.currentVault ? 'No files in vault' : 'Open a vault to get started'"
    />

    <!-- File count bar -->
    <div v-if="filesStore.files.length > 0" class="file-count-bar">
      <span>{{ filesStore.files.length }} / {{ filesStore.totalCount }} files</span>
      <span v-if="filesStore.isLoading" class="loading-more">Loading more…</span>
    </div>

    <!-- Grid / List view -->
    <ThumbnailGrid v-if="uiStore.viewMode === 'grid'" />
    <ListView v-else />

    <!-- Infinite scroll sentinel -->
    <div ref="sentinel" class="sentinel"></div>
  </main>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import ThumbnailGrid from './ThumbnailGrid.vue'
import ListView from './ListView.vue'
import EmptyState from '../common/EmptyState.vue'
import { useUIStore } from '../../stores/ui'
import { useFilesStore } from '../../stores/files'
import { useVaultStore } from '../../stores/vault'
import { useFiltersStore } from '../../stores/filters'
import { usePagination } from '../../composables/usePagination'

const uiStore = useUIStore()
const filesStore = useFilesStore()
const vaultStore = useVaultStore()
const filtersStore = useFiltersStore()
const { loadingMore, loadMore, resetPage, observeSentinel } = usePagination()

const galleryRef = ref<HTMLElement | null>(null)
const sentinel = ref<HTMLElement | null>(null)

let cleanup: (() => void) | undefined = undefined

onMounted(async () => {
  await loadGallery()
  cleanup = observeSentinel(sentinel.value)
})

onUnmounted(() => cleanup?.())

// Reload when filters change
watch(
  () => filtersStore.activeFilters,
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
