<template>
  <header class="topbar">
    <div class="topbar-left">
      <button class="app-menu-btn" @click="toggleMenu">☰</button>
      <span class="app-title">TagLoom</span>
      <span v-if="vaultStore.currentVault" class="vault-name">
        {{ vaultStore.currentVault.name }}
      </span>
    </div>

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
import { useVaultStore } from '../../stores/vault'
import SearchBar from './SearchBar.vue'
import ThumbnailSlider from './ThumbnailSlider.vue'
import SortControl from './SortControl.vue'
import ViewToggle from './ViewToggle.vue'

const vaultStore = useVaultStore()

const toggleMenu = () => {
  // TODO: Open app menu dropdown
}
</script>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  height: 48px;
  padding: 0 12px;
  gap: 12px;
  border-bottom: 1px solid #333;
  background: #1a1a1a;
}
.topbar-left { display: flex; align-items: center; gap: 8px; min-width: 180px; }
.topbar-center { flex: 1; display: flex; align-items: center; justify-content: center; gap: 12px; }
.topbar-right { display: flex; align-items: center; gap: 8px; min-width: 200px; justify-content: flex-end; }
.app-menu-btn { background: none; border: none; color: #ccc; font-size: 18px; cursor: pointer; }
.app-title { font-weight: bold; color: #fff; }
.vault-name { color: #888; font-size: 12px; }
</style>
