<template>
  <div class="titlebar" style="--wails-draggable: drag">
    <div class="titlebar-left">
      <div class="menu-wrapper" ref="menuRef">
        <button
          class="app-menu-btn"
          style="--wails-draggable: no-drag"
          @click="showMenu = !showMenu"
        >
          <Menu :size="16" />
        </button>
        <transition name="menu-fade">
          <div v-if="showMenu" class="app-menu">
            <button class="menu-item" @click="onNewVault">
              <Plus :size="16" class="menu-icon" /> New Vault
            </button>
            <button class="menu-item" @click="onOpenVault">
              <FolderOpen :size="16" class="menu-icon" /> Open Vault
            </button>
            <button
              v-if="vaultStore.currentVault"
              class="menu-item"
              @click="onCloseVault"
            >
              <X :size="16" class="menu-icon" /> Close Vault
            </button>
            <button class="menu-item" @click="onRescanVault">
              <RefreshCw :size="16" class="menu-icon" /> Rescan Folder
            </button>
            <button class="menu-item" @click="onFullScan">
              <FolderSearch :size="16" class="menu-icon" /> Full Scan
            </button>
            <div class="menu-divider"></div>
            <button class="menu-item" @click="onVaultSettings">
              <Settings :size="16" class="menu-icon" /> Vault Settings
            </button>
            <button class="menu-item" @click="onTagManager">
              <Tags :size="16" class="menu-icon" /> Tag Manager
            </button>
            <div class="menu-divider"></div>
            <button class="menu-item" @click="onExit">
              <LogOut :size="16" class="menu-icon" /> Exit
            </button>
          </div>
        </transition>
      </div>

      <span class="app-title">
        <img class="logo-icon" src="../../assets/appicon_small.png" alt="TagLoom" />
        TagLoom
      </span>
      <span v-if="vaultStore.currentVault" class="vault-name">
        — {{ vaultStore.currentVault.name }}
      </span>
    </div>

    <div class="titlebar-center drag-region" style="--wails-draggable: no-drag" @dblclick="toggleMaximise"></div>

    <div class="titlebar-right" style="--wails-draggable: no-drag">
      <button
        class="window-btn minimize"
        title="Minimize"
        @click="minimise"
      >
        <Minus :size="14" />
      </button>
      <button
        class="window-btn maximize"
        :title="isMaximised ? 'Restore' : 'Maximize'"
        @click="toggleMaximise"
      >
        <Maximize v-if="!isMaximised" :size="14" />
        <Minimize2 v-else :size="14" />
      </button>
      <button
        class="window-btn close"
        title="Close"
        @click="close"
      >
        <X :size="14" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Menu, Plus, FolderOpen, X, RefreshCw, FolderSearch, Settings, Tags, LogOut, Minus, Maximize, Minimize2 } from '@lucide/vue'
import { useVaultStore } from '../../stores/vault'
import { useUIStore } from '../../stores/ui'
import {
  WindowMinimise,
  WindowMaximise,
  WindowUnmaximise,
  WindowIsMaximised,
  Quit,
} from '../../../wailsjs/runtime/runtime'

const vaultStore = useVaultStore()
const uiStore = useUIStore()
const showMenu = ref(false)
const menuRef = ref<HTMLElement | null>(null)
const isMaximised = ref(false)

// Track maximised state
const updateMaximisedState = () => {
  WindowIsMaximised().then((state: boolean) => {
    isMaximised.value = state
  })
}

const minimise = () => {
  WindowMinimise()
}

const toggleMaximise = async () => {
  const maximised = await WindowIsMaximised()
  if (maximised) {
    WindowUnmaximise()
  } else {
    WindowMaximise()
  }
  isMaximised.value = !maximised
}

const close = () => {
  Quit()
}

const onNewVault = () => {
  showMenu.value = false
  uiStore.openNewVault()
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
const onVaultSettings = () => {
  showMenu.value = false
  uiStore.openVaultSettings()
}
const onTagManager = () => {
  showMenu.value = false
  uiStore.openTagManager()
}
const onExit = () => {
  showMenu.value = false
  Quit()
}

// Close menu on outside click
const closeMenu = (e: MouseEvent) => {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    showMenu.value = false
  }
}

onMounted(() => {
  document.addEventListener('mousedown', closeMenu)
  updateMaximisedState()
})

onUnmounted(() => {
  document.removeEventListener('mousedown', closeMenu)
})
</script>

<style scoped>
.titlebar {
  display: flex;
  align-items: center;
  height: 36px;
  background: #111111;
  border-bottom: 1px solid #1a1a1a;
  user-select: none;
  -webkit-user-select: none;
  position: relative;
  z-index: 100;
}

.titlebar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding-left: 8px;
}

.titlebar-center {
  flex: 1;
}

.titlebar-right {
  display: flex;
  align-items: center;
  height: 100%;
  flex-shrink: 0;
}

/* ── Menu ────────────────────────────────────────────── */
.menu-wrapper {
  position: relative;
}

.app-menu-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: none;
  border: none;
  color: #aaa;
  cursor: pointer;
  border-radius: 4px;
  transition: background 0.15s, color 0.15s;
}

.app-menu-btn:hover {
  background: #1e1e1e;
  color: #e8e8e8;
}

.app-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 8px;
  min-width: 200px;
  padding: 4px;
  z-index: 200;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.6);
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  text-align: left;
  background: none;
  border: none;
  color: #ccc;
  padding: 7px 12px;
  font-size: 13px;
  font-family: 'Inter', sans-serif;
  cursor: pointer;
  border-radius: 5px;
  transition: background 0.1s;
}

.menu-item:hover {
  background: #14532d;
  color: #e8e8e8;
}

.menu-icon {
  flex-shrink: 0;
}

.menu-divider {
  height: 1px;
  background: #2a2a2a;
  margin: 4px 8px;
}

/* ── Title ───────────────────────────────────────────── */
.app-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  font-size: 13px;
  color: #e8e8e8;
  white-space: nowrap;
}

.logo-icon {
  width: 18px;
  height: 18px;
  object-fit: contain;
}

.vault-name {
  color: #666;
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 250px;
}

/* ── Window buttons ──────────────────────────────────── */
.window-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 100%;
  background: none;
  border: none;
  color: #aaa;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.window-btn:hover {
  color: #e8e8e8;
}

.minimize:hover {
  background: #1e1e1e;
}

.maximize:hover {
  background: #1e1e1e;
}

.close {
  border-radius: 0 8px 0 0;
}

.close:hover {
  background: #dc2626;
  color: #fff;
}

/* ── Transitions ─────────────────────────────────────── */
.menu-fade-enter-active,
.menu-fade-leave-active {
  transition: opacity 0.15s;
}

.menu-fade-enter-from,
.menu-fade-leave-to {
  opacity: 0;
}
</style>
