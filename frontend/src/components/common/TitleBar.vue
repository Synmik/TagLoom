<template>
  <div class="titlebar" style="--wails-draggable: drag">
    <div class="titlebar-left">
      <div class="menu-wrapper" ref="menuRef">
        <button
          class="app-menu-btn"
          style="--wails-draggable: no-drag"
          @click="showMenu = !showMenu"
        >
          <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
            <path d="M2 3h12v1H2zm0 4h12v1H2zm0 4h12v1H2z" />
          </svg>
        </button>
        <transition name="menu-fade">
          <div v-if="showMenu" class="app-menu">
            <button class="menu-item" @click="onNewVault">
              <span class="menu-icon">➕</span> New Vault
            </button>
            <button class="menu-item" @click="onOpenVault">
              <span class="menu-icon">📂</span> Open Vault
            </button>
            <button
              v-if="vaultStore.currentVault"
              class="menu-item"
              @click="onCloseVault"
            >
              <span class="menu-icon">✕</span> Close Vault
            </button>
            <button class="menu-item" @click="onRescanVault">
              <span class="menu-icon">🔄</span> Rescan Folder
            </button>
            <button class="menu-item" @click="onFullScan">
              <span class="menu-icon">📂</span> Full Scan
            </button>
            <div class="menu-divider"></div>
            <button class="menu-item" @click="onVaultSettings">
              <span class="menu-icon">⚙</span> Vault Settings
            </button>
            <button class="menu-item" @click="onTagManager">
              <span class="menu-icon">🏷</span> Tag Manager
            </button>
            <div class="menu-divider"></div>
            <button class="menu-item" @click="onExit">
              <span class="menu-icon">🚪</span> Exit
            </button>
          </div>
        </transition>
      </div>

      <span class="app-title">
        <svg class="logo-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l7.17 7.17a2 2 0 0 1 0 2.83z" />
          <circle cx="7" cy="7" r="1.5" fill="currentColor" stroke="none" />
        </svg>
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
        <svg width="12" height="12" viewBox="0 0 12 12">
          <line x1="1" y1="6" x2="11" y2="6" stroke="currentColor" stroke-width="1.5" />
        </svg>
      </button>
      <button
        class="window-btn maximize"
        :title="isMaximised ? 'Restore' : 'Maximize'"
        @click="toggleMaximise"
      >
        <svg v-if="!isMaximised" width="12" height="12" viewBox="0 0 12 12">
          <rect x="1" y="1" width="10" height="10" fill="none" stroke="currentColor" stroke-width="1.5" />
        </svg>
        <svg v-else width="12" height="12" viewBox="0 0 12 12">
          <rect x="3" y="3" width="8" height="8" fill="none" stroke="currentColor" stroke-width="1.2" />
          <polyline points="3,3 3,1 1,1 1,3" fill="none" stroke="currentColor" stroke-width="1.2" />
        </svg>
      </button>
      <button
        class="window-btn close"
        title="Close"
        @click="close"
      >
        <svg width="12" height="12" viewBox="0 0 12 12">
          <line x1="2" y1="2" x2="10" y2="10" stroke="currentColor" stroke-width="1.5" />
          <line x1="10" y1="2" x2="2" y2="10" stroke="currentColor" stroke-width="1.5" />
        </svg>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
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
  background: #1a1a1a;
  border-bottom: 1px solid #2a2a2a;
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
  background: #2a2a2a;
  color: #fff;
}

.app-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  background: #1e1e1e;
  border: 1px solid #333;
  border-radius: 8px;
  min-width: 200px;
  padding: 4px;
  z-index: 200;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  text-align: left;
  background: none;
  border: none;
  color: #ddd;
  padding: 8px 12px;
  font-size: 13px;
  cursor: pointer;
  border-radius: 4px;
  transition: background 0.1s;
}

.menu-item:hover {
  background: #2a3a5a;
}

.menu-icon {
  font-size: 14px;
  width: 20px;
  text-align: center;
}

.menu-divider {
  height: 1px;
  background: #333;
  margin: 4px 8px;
}

/* ── Title ───────────────────────────────────────────── */
.app-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  font-size: 13px;
  color: #fff;
  white-space: nowrap;
}

.logo-icon {
  color: #5b8af5;
}

.vault-name {
  color: #777;
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
  color: #fff;
}

.minimize:hover {
  background: #2a2a2a;
}

.maximize:hover {
  background: #2a2a2a;
}

.close {
  border-radius: 0 8px 0 0;
}

.close:hover {
  background: #c42b1c;
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
