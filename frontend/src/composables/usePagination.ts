import { ref, computed } from 'vue'
import { useFilesStore } from '../stores/files'

export function usePagination() {
  const filesStore = useFilesStore()
  const loadingMore = ref(false)

  // Number of files still unloaded (need to fetch more pages)
  const hasMore = computed(() => {
    const files = filesStore.files || []
    return files.length < filesStore.totalCount
  })

  const loadMore = async () => {
    if (loadingMore.value || !hasMore.value) return
    if (filesStore.totalCount === 0) return
    loadingMore.value = true
    await filesStore.loadNextPage()
    loadingMore.value = false
  }

  const resetPage = () => {
    filesStore.page = 0
    filesStore.files = []
  }

  return { loadingMore, loadMore, resetPage, hasMore }
}
