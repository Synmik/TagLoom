<template>
  <div class="view-toggle">
    <button
      :class="['view-btn', { active: uiStore.viewMode === 'grid' }]"
      @click="uiStore.setViewMode('grid')"
      title="Grid view"
    >
      <Grid3X3 :size="16" />
    </button>
    <button
      :class="['view-btn', { active: uiStore.viewMode === 'list' }]"
      @click="uiStore.setViewMode('list')"
      title="List view"
    >
      <List :size="16" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { Grid3X3, List } from '@lucide/vue'
import { useUIStore } from '../../stores/ui'
const uiStore = useUIStore()
</script>

<style scoped>
.view-toggle { display: flex; gap: 2px; }
.view-btn {
  background: #2a2a2a; border: 1px solid #444; color: #888;
  border-radius: 4px; width: 28px; height: 28px; cursor: pointer; font-size: 16px;
}
.view-btn.active { background: #5b8af5; color: #fff; border-color: #5b8af5; }
</style>
