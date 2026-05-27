import { ref, computed } from 'vue'
import { useFilesStore } from '../stores/files'

export function useSearch() {
  const query = ref('')
  const filesStore = useFilesStore()
  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  const search = async () => {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(async () => {
      if (query.value.trim()) {
        await filesStore.searchFiles(query.value.trim())
      } else {
        await filesStore.loadFiles()
      }
    }, 300)
  }

  const clearSearch = () => {
    query.value = ''
    filesStore.loadFiles()
  }

  return { query, search, clearSearch }
}
