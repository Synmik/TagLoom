<template>
  <div class="app-layout">
    <TitleBar />
    <TopBar />
    <div class="app-body">
      <LeftPanel />
      <div class="resize-handle left" @mousedown="onLeftResizeStart"></div>
      <Gallery />
      <div class="resize-handle right" @mousedown="onRightResizeStart"></div>
      <RightPanel />
    </div>
    <ScanProgressBar />
    <ToastContainer />
    <FilePreviewModal v-if="previewStore.previewModalOpen" @close="previewStore.closeFullPreview" />
    <BatchEditModal v-if="uiStore.showBatchEdit" @close="uiStore.closeBatchEdit" />
    <VaultSettingsModal v-if="uiStore.showVaultSettings" @close="uiStore.closeVaultSettings" />
    <TagManagerModal v-if="uiStore.showTagManager" @close="uiStore.closeTagManager" />
    <NewVaultModal v-if="uiStore.showNewVault" @close="uiStore.closeNewVault" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import TitleBar from './components/common/TitleBar.vue'
import TopBar from './components/topbar/TopBar.vue'
import LeftPanel from './components/leftpanel/LeftPanel.vue'
import Gallery from './components/gallery/Gallery.vue'
import RightPanel from './components/rightpanel/RightPanel.vue'
import ScanProgressBar from './components/common/ScanProgressBar.vue'
import ToastContainer from './components/common/ToastContainer.vue'
import FilePreviewModal from './components/modals/FilePreviewModal.vue'
import BatchEditModal from './components/modals/BatchEditModal.vue'
import VaultSettingsModal from './components/modals/VaultSettingsModal.vue'
import TagManagerModal from './components/modals/TagManagerModal.vue'
import NewVaultModal from './components/modals/NewVaultModal.vue'
import { usePreviewStore } from './stores/preview'
import { useUIStore } from './stores/ui'
import { useTagsStore } from './stores/tags'
import { useFoldersStore } from './stores/folders'
import { useVaultStore } from './stores/vault'
import { useKeyboardShortcuts } from './composables/useKeyboardShortcuts'

const previewStore = usePreviewStore()
const uiStore = useUIStore()

// ── Panel resize logic ────────────────────────────────────────────

const onLeftResizeStart = (e: MouseEvent) => {
  e.preventDefault()
  const startX = e.clientX
  const startWidth = uiStore.leftPanelWidth

  const onMouseMove = (ev: MouseEvent) => {
    const delta = ev.clientX - startX
    uiStore.setLeftPanelWidth(startWidth + delta)
  }

  const onMouseUp = () => {
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }

  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

const onRightResizeStart = (e: MouseEvent) => {
  e.preventDefault()
  const startX = e.clientX
  const startWidth = uiStore.rightPanelWidth

  const onMouseMove = (ev: MouseEvent) => {
    const delta = startX - ev.clientX
    uiStore.setRightPanelWidth(startWidth + delta)
  }

  const onMouseUp = () => {
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }

  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

onMounted(async () => {
  previewStore._syncSelection()
  useTagsStore()._watchVault()
  useFoldersStore()._watchVault()
  await useVaultStore().autoOpenLastVault()

  // ── Keyboard shortcuts ──────────────────────────────────────────
  useKeyboardShortcuts()
})
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0d0d0d; color: #e8e8e8; }
</style>

<style scoped>
.app-layout { display: flex; flex-direction: column; height: 100vh; overflow: hidden; }
.app-body { display: flex; flex: 1; overflow: hidden; }

.resize-handle {
  width: 4px;
  min-width: 4px;
  background: transparent;
  cursor: col-resize;
  flex-shrink: 0;
  transition: background 0.15s;
  position: relative;
  z-index: 10;
}
.resize-handle:hover,
.resize-handle:active {
  background: #22c55e;
}
</style>
