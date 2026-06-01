import { ref } from 'vue'
import { useFilesStore } from '../stores/files'

export function usePagination() {
  const filesStore = useFilesStore()
  const loadingMore = ref(false)

  const loadMore = async () => {
    const files = filesStore.files || []
    if (loadingMore.value || files.length >= filesStore.totalCount) return
    if (filesStore.totalCount === 0) return
    loadingMore.value = true
    filesStore.page++
    await filesStore.loadFiles()
    loadingMore.value = false
  }

  const resetPage = () => {
    filesStore.page = 0
    filesStore.files = []
  }

  return { loadingMore, loadMore, resetPage }
}
