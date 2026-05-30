import { ref } from 'vue'
import { useFilesStore } from '../stores/files'

export function usePagination() {
  const filesStore = useFilesStore()
  const loadingMore = ref(false)

  const loadMore = async () => {
    const files = filesStore.files || []
    if (loadingMore.value || files.length >= filesStore.totalCount) return
    // Don't load more if gallery is empty (totalCount is 0) — avoid triggering
    // infinite scroll on empty states
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
