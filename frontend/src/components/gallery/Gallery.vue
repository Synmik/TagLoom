<template>
  <main class="gallery" ref="galleryRef">
    <div v-if="filesStore.isLoading && filesStore.files.length === 0" class="loading">Loading files…</div>
    <EmptyState v-else-if="filesStore.files.length === 0 && !filesStore.isLoading" />
    <ThumbnailGrid v-else-if="uiStore.viewMode === 'grid'" />
    <ListView v-else />

    <!-- Infinite scroll sentinel -->
    <div ref="sentinel" class="sentinel"></div>
  </main>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import ThumbnailGrid from './ThumbnailGrid.vue'
import ListView from './ListView.vue'
import EmptyState from '../common/EmptyState.vue'
import { useUIStore } from '../../stores/ui'
import { useFilesStore } from '../../stores/files'
import { usePagination } from '../../composables/usePagination'

const uiStore = useUIStore()
const filesStore = useFilesStore()
const { observeSentinel } = usePagination()
const galleryRef = ref<HTMLElement | null>(null)
const sentinel = ref<HTMLElement | null>(null)

let cleanup: (() => void) | undefined = undefined
onMounted(async () => {
  await filesStore.loadFiles()
  cleanup = observeSentinel(sentinel.value)
})
onUnmounted(() => cleanup?.())
</script>

<style scoped>
.gallery {
  flex: 1; overflow-y: auto; padding: 12px;
  background: #121212;
}
.loading { text-align: center; color: #666; padding: 40px; }
.sentinel { height: 1px; }
</style>
