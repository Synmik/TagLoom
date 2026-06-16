<template>
  <div class="app-layout" ref="appRoot">
    <TitleBar />
    <TopBar />
    <div class="app-body">
      <LeftPanel />
      <div class="resize-handle left" @mousedown="onLeftResizeStart"></div>
      <Gallery />
      <div class="resize-handle right" @mousedown="onRightResizeStart"></div>
      <RightPanel />
    </div>
    <!-- Drag & drop overlay -->
    <div v-if="dragDrop.isDragging.value" class="drop-overlay">
      <div class="drop-overlay-content">
        <ArrowDown class="drop-icon" :size="48" />
        <span class="drop-label">Drop files to import</span>
      </div>
    </div>
    <!-- Import menu (copy / move) -->
    <ImportMenu
      v-if="dragDrop.showMenu.value && dragDrop.menuData.value"
      :files="(dragDrop.menuData.value as any).files"
      :x="(dragDrop.menuData.value as any).x"
      :y="(dragDrop.menuData.value as any).y"
      @close="dragDrop.closeMenu"
      @copy="handleImport(false)"
      @move="handleImport(true)"
    />
    <ScanProgressBar />
    <ToastContainer />
    <FilePreviewModal v-if="previewStore.previewModalOpen" @close="previewStore.closeFullPreview" />
    <BatchEditModal v-if="uiStore.showBatchEdit" @close="uiStore.closeBatchEdit" />
    <VaultSettingsModal v-if="uiStore.showVaultSettings" @close="uiStore.closeVaultSettings" />
    <AppSettingsModal v-if="uiStore.showAppSettings" @close="uiStore.closeAppSettings" />
    <TagManagerModal v-if="uiStore.showTagManager" @close="uiStore.closeTagManager" />
    <NewVaultModal v-if="uiStore.showNewVault" @close="uiStore.closeNewVault" />
    <AboutModal v-if="uiStore.showAbout" @close="uiStore.closeAbout" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
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
import AppSettingsModal from './components/modals/AppSettingsModal.vue'
import TagManagerModal from './components/modals/TagManagerModal.vue'
import NewVaultModal from './components/modals/NewVaultModal.vue'
import AboutModal from './components/modals/AboutModal.vue'
import ImportMenu from './components/common/ImportMenu.vue'
import { usePreviewStore } from './stores/preview'
import { useUIStore } from './stores/ui'
import { useTagsStore } from './stores/tags'
import { useFoldersStore } from './stores/folders'
import { useVaultStore } from './stores/vault'
import { useFilesStore } from './stores/files'
import { useKeyboardShortcuts } from './composables/useKeyboardShortcuts'
import { useDragDrop } from './composables/useDragDrop'
import { GetAppSettings, ImportFile } from './api/backend'
import { ArrowDown } from '@lucide/vue'

const previewStore = usePreviewStore()
const uiStore = useUIStore()
const foldersStore = useFoldersStore()
const vaultStore = useVaultStore()

// ── Drag & drop ──────────────────────────────────────────────────

const appRoot = ref<HTMLElement | null>(null)
const dragDrop = useDragDrop()

async function handleImport(move: boolean) {
  const files = dragDrop.menuData.value?.files
  if (!files || files.length === 0) {
    dragDrop.closeMenu()
    return
  }

  // Import into currently selected folder (empty = vault root).
  // Root folder's selectedPath is the full vault path — treat as vault root.
  const targetFolder = foldersStore.selectedPath === vaultStore.currentVault?.path ? '' : foldersStore.selectedPath
  dragDrop.closeMenu()

  let imported = 0
  let skipped = 0
  const errors: string[] = []

  for (const f of files) {
    try {
      const result = await ImportFile(f.path, move, targetFolder)
      if (result?.imported) imported += result.imported
      if (result?.skipped) skipped += result.skipped
      if (result?.errors) errors.push(...result.errors)
    } catch (e: any) {
      errors.push(`${f.name}: ${e.message || String(e)}`)
    }
  }

  // Reload gallery
  await useFilesStore().loadFiles()

  // Show result toast(s)
  const { useToast } = await import('./composables/useToast')
  const toast = useToast()
  if (imported > 0) {
    toast.success(`Imported ${imported} file${imported > 1 ? 's' : ''}`)
  }
  if (skipped > 0) {
    toast.info(`Skipped ${skipped} file${skipped > 1 ? 's' : ''} (already in vault)`)
  }
  for (const err of errors) {
    toast.error(err)
  }
}

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
  // Set up drag-and-drop handlers on the app root element
  if (appRoot.value) {
    dragDrop.setupHandlers(appRoot.value)
  }

  previewStore._syncSelection()
  useTagsStore()._watchVault()
  useFoldersStore()._watchVault()

  // Load global app settings to check auto-open preference
  let settings
  try {
    settings = await GetAppSettings()
  } catch {
    // Non-fatal: proceed without settings
  }
  if (settings?.auto_open_last_vault) {
    await useVaultStore().autoOpenLastVault()
  }

  // ── Keyboard shortcuts ──────────────────────────────────────────
  useKeyboardShortcuts()
})

onUnmounted(() => {
  if (appRoot.value) {
    dragDrop.teardownHandlers(appRoot.value)
  }
})
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0d0d0d; color: #e8e8e8; }
</style>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  --wails-drop-target: drop;
}
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

/* ── Drop overlay ────────────────────────────────────────────────── */
.drop-overlay {
  position: fixed;
  inset: 0;
  z-index: 9000;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}

.drop-overlay-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: #22c55e;
}

.drop-icon {
  animation: drop-bounce 1s ease-in-out infinite;
}

.drop-label {
  font-size: 20px;
  font-weight: 600;
  letter-spacing: 0.5px;
}

@keyframes drop-bounce {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(8px); }
}
</style>
