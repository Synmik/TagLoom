<template>
  <div class="list-view">
    <div class="list-header">
      <span class="col-thumb"></span>
      <span class="col-name">Name</span>
      <span class="col-tags">Tags</span>
      <span class="col-date">Date</span>
      <span class="col-size">Size</span>
      <span class="col-rating">Rating</span>
    </div>
    <ListRow v-for="file in filesStore.files" :key="file.id" :file="file" />
  </div>
</template>

<script setup lang="ts">
import ListRow from './ListRow.vue'
import { useFilesStore } from '../../stores/files'
const filesStore = useFilesStore()
</script>

<style scoped>
.list-view { font-size: 13px; }
.list-header {
  display: grid; grid-template-columns: 40px 1fr 1fr 100px 80px 60px;
  padding: 6px 8px; color: #888; font-size: 11px; text-transform: uppercase;
  border-bottom: 1px solid #333; position: sticky; top: 0; background: #121212;
}
</style>
