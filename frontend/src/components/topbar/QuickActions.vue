<template>
  <div class="quick-actions">
    <button class="action-btn" title="New Vault" @click="onNewVault">
      <Plus :size="15" />
    </button>
    <button class="action-btn" title="Open Vault" @click="onOpenVault">
      <FolderOpen :size="15" />
    </button>
    <button class="action-btn" title="Rescan Folder" @click="onRescanVault">
      <RefreshCw :size="15" />
    </button>
    <button class="action-btn" title="Full Scan" @click="onFullScan">
      <FolderSearch :size="15" />
    </button>
    <button class="action-btn" title="Vault Settings" @click="onVaultSettings">
      <Settings :size="15" />
    </button>
    <button class="action-btn" title="Tag Manager" @click="onTagManager">
      <Tags :size="15" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { Plus, FolderOpen, RefreshCw, FolderSearch, Settings, Tags } from '@lucide/vue'
import { useVaultStore } from '../../stores/vault'
import { useUIStore } from '../../stores/ui'

const vaultStore = useVaultStore()
const uiStore = useUIStore()

const onNewVault = () => uiStore.openNewVault()
const onOpenVault = () => vaultStore.pickAndOpenVault()
const onRescanVault = () => vaultStore.rescanVault()
const onFullScan = () => vaultStore.scanVault()
const onVaultSettings = () => uiStore.openVaultSettings()
const onTagManager = () => uiStore.openTagManager()
</script>

<style scoped>
.quick-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}

.action-btn:hover {
  background: #1a1a1a;
  color: #e8e8e8;
}
</style>
