<template>
  <aside class="left-panel">
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
        <button class="icon-btn" @click="showTagManager = true" title="Manage Tags">⚙</button>
      </div>
      <TagTree />
    </section>

    <TagManagerModal v-if="showTagManager" @close="showTagManager = false" />
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import FolderTree from './FolderTree.vue'
import TagTree from './TagTree.vue'
import TagManagerModal from '../modals/TagManagerModal.vue'
import { useVaultStore } from '../../stores/vault'

const vaultStore = useVaultStore()
const showTagManager = ref(false)

const openVault = () => vaultStore.pickAndOpenVault()
</script>

<style scoped>
.left-panel {
  width: 280px;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #333;
  background: #1e1e1e;
  overflow: hidden;
}
.panel-section { padding: 8px; }
.tags-section { flex: 1; overflow-y: auto; border-top: 1px solid #333; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.section-header h3 { color: #aaa; font-size: 11px; text-transform: uppercase; letter-spacing: 1px; margin: 0; }
.icon-btn { background: none; border: none; cursor: pointer; font-size: 14px; }
</style>
