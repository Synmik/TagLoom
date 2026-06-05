<template>
  <div class="search-bar">
    <Search :size="14" class="search-icon" />
    <input
      ref="inputRef"
      v-model="query"
      @input="search"
      @keyup.escape="onEscape"
      type="text"
      placeholder="Search files, tags, notes…"
      class="search-input"
    />
    <button v-if="query" class="clear-btn" @click="clearSearch"><X :size="14" /></button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Search, X } from '@lucide/vue'
import { useSearch } from '../../composables/useSearch'

const { query, search, clearSearch } = useSearch()
const inputRef = ref<HTMLInputElement | null>(null)

const focus = () => {
  inputRef.value?.focus()
  inputRef.value?.select()
}

const onEscape = () => {
  clearSearch()
  inputRef.value?.blur()
}

// Listen for Ctrl+F shortcut from useKeyboardShortcuts
onMounted(() => {
  window.addEventListener('tagloom:focus-search', focus)
})

onUnmounted(() => {
  window.removeEventListener('tagloom:focus-search', focus)
})

defineExpose({ focus })
</script>

<style scoped>
.search-bar {
  display: flex;
  align-items: center;
  flex: 1;
  max-width: 400px;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 8px;
  padding: 0 12px;
  height: 34px;
  transition: border-color 0.15s;
}
.search-bar:focus-within {
  border-color: #22c55e;
}
.search-icon { color: #666; margin-right: 8px; flex-shrink: 0; }
.search-input {
  flex: 1;
  background: none;
  border: none;
  color: #e8e8e8;
  font-size: 13px;
  font-family: 'Inter', sans-serif;
  outline: none;
}
.search-input::placeholder { color: #555; }
.clear-btn {
  background: none;
  border: none;
  color: #555;
  cursor: pointer;
  font-size: 14px;
  padding: 0;
  border-radius: 3px;
  transition: color 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.clear-btn:hover { color: #e8e8e8; }
</style>
