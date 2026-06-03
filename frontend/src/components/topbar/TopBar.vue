<template>
  <header class="topbar">
    <div class="topbar-center">
      <SearchBar />
      <ThumbnailSlider />
    </div>

    <div class="topbar-right">
      <SortControl />
      <ViewToggle />
    </div>
  </header>
</template>

<script setup lang="ts">
import SearchBar from './SearchBar.vue'
import ThumbnailSlider from './ThumbnailSlider.vue'
import SortControl from './SortControl.vue'
import ViewToggle from './ViewToggle.vue'
</script>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  height: 48px;
  padding: 0 12px;
  gap: 12px;
  border-bottom: 1px solid #2a2a2a;
  background: #161616;
}
.topbar-center { flex: 1; display: flex; align-items: center; justify-content: center; gap: 12px; }
.topbar-right { display: flex; align-items: center; gap: 8px; min-width: 200px; justify-content: flex-end; }
</style>
