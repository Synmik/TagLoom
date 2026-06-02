<template>
  <aside class="left-panel" :style="panelStyle">
    <section class="panel-section">
      <div class="section-header">
        <h3>Folders</h3>
        <button class="icon-btn" @click="openVault" title="Open Vault">📂</button>
      </div>
      <FolderTree />
    </section>

    <section class="panel-section tags-section">
      <div class="section-header">
        <h3>Tags</h3>
        <button class="icon-btn" @click="openTagManager(null)" title="Create Tag">+</button>
      </div>
      <TagTree @edit="openTagManager($event)" />
    </section>

    <TagManagerModal v-if="showTagManager" :tag="editingTag" @close="closeTagManager" />
  </aside>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import FolderTree from './FolderTree.vue'
import TagTree from './TagTree.vue'
import TagManagerModal from '../modals/TagManagerModal.vue'
import { useVaultStore } from '../../stores/vault'
import { useUIStore } from '../../stores/ui'
import type { Tag } from '../../types/tag'

const uiStore = useUIStore()
const panelStyle = computed(() => ({ width: `${uiStore.leftPanelWidth}px` }))

const vaultStore = useVaultStore()
const showTagManager = ref(false)
const editingTag = ref<Tag | null>(null)

const openVault = () => vaultStore.pickAndOpenVault()

const openTagManager = (tag: Tag | null) => {
  editingTag.value = tag
  showTagManager.value = true
}

const closeTagManager = () => {
  showTagManager.value = false
  editingTag.value = null
}
</script>

<style scoped>
.left-panel {
  display: flex;
  flex-direction: column;
  border-right: 1px solid #333;
  background: #1e1e1e;
  flex-shrink: 0;
}
.panel-section { padding: 8px; }
.tags-section { flex: 1; overflow-y: auto; border-top: 1px solid #333; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.section-header h3 { color: #aaa; font-size: 11px; text-transform: uppercase; letter-spacing: 1px; margin: 0; }
.icon-btn { background: none; border: none; cursor: pointer; font-size: 14px; }
</style>
