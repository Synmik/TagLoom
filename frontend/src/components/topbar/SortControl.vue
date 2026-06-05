<template>
  <div class="sort-control">
    <select v-model="sortBy" @change="applySort" class="sort-select">
      <option value="indexed_at">Date Indexed</option>
      <option value="filename">Filename</option>
      <option value="name">Name</option>
      <option value="file_size">File Size</option>
      <option value="rating">Rating</option>
      <option value="date_modified">Date Modified</option>
    </select>
    <button @click="toggleOrder" class="order-btn" :title="sortOrder === 'asc' ? 'Ascending' : 'Descending'">
      <ArrowUp v-if="sortOrder === 'asc'" :size="14" />
      <ArrowDown v-else :size="14" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowUp, ArrowDown } from '@lucide/vue'
import { useUIStore } from '../../stores/ui'
import { useFilesStore } from '../../stores/files'
import { useFiltersStore } from '../../stores/filters'

const uiStore = useUIStore()
const filesStore = useFilesStore()
const filtersStore = useFiltersStore()

const sortBy = computed({
  get: () => uiStore.sortBy,
  set: (val) => uiStore.setSort(val as any, uiStore.sortOrder),
})
const sortOrder = computed(() => uiStore.sortOrder)

const applySort = async () => {
  await filesStore.reloadFiles()
}
const toggleOrder = async () => {
  uiStore.toggleSortOrder()
  await applySort()
}
</script>

<style scoped>
.sort-control { display: flex; align-items: center; gap: 2px; background: #1a1a1a; border-radius: 6px; padding: 2px; }
.sort-select {
  background: transparent; border: none; color: #ccc;
  padding: 4px 8px; font-size: 12px; outline: none;
  font-family: 'Inter', sans-serif;
  cursor: pointer;
  border-radius: 5px;
  appearance: none;
  -webkit-appearance: none;
}
.sort-select:hover { background: #222; }
.sort-select option { background: #1a1a1a; color: #ccc; }
.order-btn {
  background: transparent; border: none; color: #666;
  border-radius: 5px; width: 26px; height: 26px; cursor: pointer; font-size: 14px;
  display: flex; align-items: center; justify-content: center;
  transition: all 0.15s;
}
.order-btn:hover { background: #222; color: #ccc; }
</style>
