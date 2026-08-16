<template>
  <div class="view-toggle">
    <button
      :class="['view-btn', { active: uiStore.viewMode === 'grid' }]"
      title="Grid view"
      @click="uiStore.setViewMode('grid')"
    >
      <Grid3X3 :size="16" />
    </button>
    <button
      :class="['view-btn', { active: uiStore.viewMode === 'list' }]"
      title="List view"
      @click="uiStore.setViewMode('list')"
    >
      <List :size="16" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { Grid3X3, List } from "@lucide/vue";
import { useUIStore } from "../../stores/ui";
const uiStore = useUIStore();
</script>

<style scoped>
.view-toggle {
  display: flex;
  gap: 2px;
  background: #1a1a1a;
  border-radius: 6px;
  padding: 2px;
}
.view-btn {
  background: transparent;
  border: none;
  color: #666;
  border-radius: 5px;
  width: 28px;
  height: 26px;
  cursor: pointer;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}
.view-btn:hover {
  color: #ccc;
}
.view-btn.active {
  background: #22c55e;
  color: #000;
}
</style>
