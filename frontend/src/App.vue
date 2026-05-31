<template>
  <div class="app-layout">
    <TopBar />
    <div class="app-body">
      <LeftPanel />
      <Gallery />
      <RightPanel />
    </div>
    <ScanProgressBar />
    <FilePreviewModal v-if="previewStore.previewModalOpen" @close="previewStore.closeFullPreview" />
    <BatchEditModal v-if="uiStore.showBatchEdit" @close="uiStore.closeBatchEdit" />
    <VaultSettingsModal v-if="uiStore.showVaultSettings" @close="uiStore.closeVaultSettings" />
    <TagManagerModal v-if="uiStore.showTagManager" @close="uiStore.closeTagManager" />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import TopBar from './components/topbar/TopBar.vue'
import LeftPanel from './components/leftpanel/LeftPanel.vue'
import Gallery from './components/gallery/Gallery.vue'
import RightPanel from './components/rightpanel/RightPanel.vue'
import ScanProgressBar from './components/common/ScanProgressBar.vue'
import FilePreviewModal from './components/modals/FilePreviewModal.vue'
import BatchEditModal from './components/modals/BatchEditModal.vue'
import VaultSettingsModal from './components/modals/VaultSettingsModal.vue'
import TagManagerModal from './components/modals/TagManagerModal.vue'
import { usePreviewStore } from './stores/preview'
import { useUIStore } from './stores/ui'
import { useTagsStore } from './stores/tags'
import { useFoldersStore } from './stores/folders'
import { useVaultStore } from './stores/vault'

const previewStore = usePreviewStore()
const uiStore = useUIStore()

onMounted(async () => {
  previewStore._syncSelection()
  useTagsStore()._watchVault()
  useFoldersStore()._watchVault()
  await useVaultStore().autoOpenLastVault()
})
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #121212; color: #ddd; }
</style>

<style scoped>
.app-layout { display: flex; flex-direction: column; height: 100vh; overflow: hidden; }
.app-body { display: flex; flex: 1; overflow: hidden; }
</style>
