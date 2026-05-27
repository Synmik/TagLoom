<template>
  <div class="search-bar">
    <span class="search-icon">🔍</span>
    <input
      v-model="query"
      @input="search"
      @keyup.escape="clearSearch"
      type="text"
      placeholder="Search files, tags, notes…"
      class="search-input"
    />
    <button v-if="query" class="clear-btn" @click="clearSearch">✕</button>
  </div>
</template>

<script setup lang="ts">
import { useSearch } from '../../composables/useSearch'
const { query, search, clearSearch } = useSearch()
</script>

<style scoped>
.search-bar {
  display: flex;
  align-items: center;
  flex: 1;
  max-width: 400px;
  background: #2a2a2a;
  border: 1px solid #444;
  border-radius: 6px;
  padding: 0 8px;
  height: 32px;
}
.search-icon { color: #888; margin-right: 6px; }
.search-input {
  flex: 1;
  background: none;
  border: none;
  color: #fff;
  font-size: 13px;
  outline: none;
}
.clear-btn {
  background: none;
  border: none;
  color: #888;
  cursor: pointer;
  font-size: 14px;
}
</style>
