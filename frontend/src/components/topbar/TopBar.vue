<template>
  <header class="topbar">
    <div class="topbar-left">
      <div class="menu-wrapper" ref="menuRef">
        <button class="app-menu-btn" @click="showMenu = !showMenu">☰</button>
        <transition name="menu-fade">
          <div v-if="showMenu" class="app-menu">
            <button class="menu-item" @click="onOpenVault">📂 Open Vault</button>
            <button v-if="vaultStore.currentVault" class="menu-item" @click="onCloseVault">✕ Close Vault</button>
            <button class="menu-item" @click="onRescanVault">🔄 Rescan Folder</button>
            <button class="menu-item" @click="onFullScan">📂 Full Scan</button>
            <div class="menu-divider"></div>
            <button class="menu-item" @click="onVaultSettings">⚙ Vault Settings</button>
            <button class="menu-item" @click="onTagManager">🏷 Tag Manager</button>
            <div class="menu-divider"></div>
            <button class="menu-item" @click="onExit">🚪 Exit</button>
          </div>
        </transition>
      </div>
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
import { ref } from 'vue'
import { useVaultStore } from '../../stores/vault'
import { useUIStore } from '../../stores/ui'
import SearchBar from './SearchBar.vue'
import ThumbnailSlider from './ThumbnailSlider.vue'
import SortControl from './SortControl.vue'
import ViewToggle from './ViewToggle.vue'
// Quit the app via Wails runtime (injected at build time)
const Quit = (): void => {
  // @ts-ignore - window.runtime is injected by Wails
  window.runtime?.Quit?.()
}

const vaultStore = useVaultStore()
const uiStore = useUIStore()
const showMenu = ref(false)
const menuRef = ref<HTMLElement | null>(null)

const toggleMenu = () => { showMenu.value = !showMenu.value }

// Close menu on outside click
const closeMenu = (e: MouseEvent) => {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    showMenu.value = false
  }
}

const onOpenVault = async () => {
  showMenu.value = false
  await vaultStore.pickAndOpenVault()
}
const onCloseVault = async () => {
  showMenu.value = false
  await vaultStore.closeVault()
}
const onRescanVault = async () => {
  showMenu.value = false
  await vaultStore.rescanVault()
}
const onFullScan = async () => {
  showMenu.value = false
  await vaultStore.scanVault()
}
const onVaultSettings = () => { showMenu.value = false; uiStore.openVaultSettings() }
const onTagManager = () => { showMenu.value = false; uiStore.openTagManager() }
const onExit = () => {
  showMenu.value = false
  Quit()
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

.menu-wrapper { position: relative; }
.app-menu-btn { background: none; border: none; color: #ccc; font-size: 18px; cursor: pointer; }

.app-menu {
  position: absolute; top: 100%; left: 0;
  background: #1e1e1e; border: 1px solid #333; border-radius: 6px;
  min-width: 180px; padding: 4px 0; z-index: 50;
  box-shadow: 0 8px 24px rgba(0,0,0,0.4);
}
.menu-item {
  display: block; width: 100%; text-align: left;
  background: none; border: none; color: #ddd;
  padding: 8px 16px; font-size: 13px; cursor: pointer;
}
.menu-item:hover { background: #2a3a5a; }
.menu-divider { height: 1px; background: #333; margin: 4px 0; }

.app-title { font-weight: bold; color: #fff; }
.vault-name { color: #888; font-size: 12px; }

.menu-fade-enter-active, .menu-fade-leave-active { transition: opacity 0.15s; }
.menu-fade-enter-from, .menu-fade-leave-to { opacity: 0; }
</style>
