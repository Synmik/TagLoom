import { ref } from 'vue'
import { useFilesStore } from '../stores/files'

export function usePagination() {
  const filesStore = useFilesStore()
  const loadingMore = ref(false)

  const loadMore = async () => {
    if (loadingMore.value || filesStore.files.length >= filesStore.totalCount) return
    loadingMore.value = true
    filesStore.page++
    await filesStore.loadFiles({}, { field: 'indexed_at', order: 'desc' }, true) // append mode
    loadingMore.value = false
  }

  const resetPage = () => {
    filesStore.page = 0
    filesStore.files = []
  }

  // Infinite scroll via IntersectionObserver
  const observeSentinel = (element: HTMLElement | null) => {
    if (!element) return
    const observer = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting) {
        loadMore()
      }
    }, { rootMargin: '200px' })
    observer.observe(element)

    return () => observer.disconnect()
  }

  return { loadingMore, loadMore, resetPage, observeSentinel }
}
