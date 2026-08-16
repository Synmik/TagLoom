import { ref, watch } from "vue";
import { useFilesStore } from "../stores/files";
import { useFiltersStore } from "../stores/filters";

export function useSearch() {
  const query = ref("");
  const filesStore = useFilesStore();
  const filtersStore = useFiltersStore();
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  // When the store's searchQuery is cleared externally (e.g. "Clear Filters"),
  // sync the local input back to empty and reload the full file list
  watch(
    () => filtersStore.activeFilters.searchQuery,
    async (storeQuery) => {
      if (storeQuery === "" && query.value !== "") {
        query.value = "";
        await filesStore.reloadFiles();
      }
    },
  );

  const search = async () => {
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(async () => {
      // Keep the store in sync so the gallery can detect an active search
      filtersStore.activeFilters.searchQuery = query.value;
      if (query.value.trim()) {
        await filesStore.searchFiles(query.value.trim());
      } else {
        await filesStore.reloadFiles();
      }
    }, 300);
  };

  const clearSearch = () => {
    query.value = "";
    filtersStore.activeFilters.searchQuery = "";
    filesStore.reloadFiles();
  };

  return { query, search, clearSearch };
}
