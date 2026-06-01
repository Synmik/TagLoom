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
      {{ sortOrder === 'asc' ? '↑' : '↓' }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
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
.sort-control { display: flex; align-items: center; gap: 4px; }
.sort-select {
  background: #2a2a2a; border: 1px solid #444; color: #ccc;
  border-radius: 4px; padding: 4px 8px; font-size: 12px; outline: none;
}
.order-btn {
  background: #2a2a2a; border: 1px solid #444; color: #ccc;
  border-radius: 4px; width: 28px; height: 28px; cursor: pointer; font-size: 14px;
}
</style>
