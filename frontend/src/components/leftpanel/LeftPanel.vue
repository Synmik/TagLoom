<template>
  <aside class="left-panel" :style="panelStyle">
    <section class="panel-section" :style="{ height: `${uiStore.leftPanelSplit}%` }">
      <div class="section-header">
        <h3>Folders</h3>
        <button class="icon-btn" @click="openVault" title="Open Vault"><FolderOpen :size="14" /></button>
      </div>
      <FolderTree />
    </section>

    <div class="divider" @mousedown="startDrag"></div>

    <section class="panel-section tags-section" :style="{ height: `${100 - uiStore.leftPanelSplit}%` }">
      <div class="section-header">
        <h3>Tags</h3>
        <button class="icon-btn" @click="openTagManager(null)" title="Create Tag"><Plus :size="14" /></button>
      </div>
      <TagTree @edit="openTagManager($event)" />
    </section>

    <TagManagerModal v-if="showTagManager" :tag="editingTag" @close="closeTagManager" />
  </aside>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount } from 'vue'
import { FolderOpen, Plus } from '@lucide/vue'
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

// Draggable divider
const isDragging = ref(false)

const startDrag = (e: MouseEvent) => {
  e.preventDefault()
  isDragging.value = true
  document.body.style.cursor = 'ns-resize'
  document.body.style.userSelect = 'none'
  window.addEventListener('mousemove', onDrag)
  window.addEventListener('mouseup', stopDrag)
}

const onDrag = (e: MouseEvent) => {
  if (!isDragging.value) return
  const panel = (e.target as HTMLElement).closest('.left-panel') as HTMLElement
  if (!panel) return
  const rect = panel.getBoundingClientRect()
  const ratio = ((e.clientY - rect.top) / rect.height) * 100
  uiStore.setLeftPanelSplit(ratio)
}

const stopDrag = () => {
  isDragging.value = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  window.removeEventListener('mousemove', onDrag)
  window.removeEventListener('mouseup', stopDrag)
}

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', onDrag)
  window.removeEventListener('mouseup', stopDrag)
})
</script>

<style scoped>
.left-panel {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  border-right: 1px solid #333;
  background: #1e1e1e;
  flex-shrink: 0;
}
.panel-section {
  overflow-y: auto;
  padding: 8px;
  min-height: 0;
}
.divider {
  height: 4px;
  background: transparent;
  cursor: ns-resize;
  flex-shrink: 0;
  transition: background 0.15s;
}
.divider:hover,
.divider:active {
  background: #4a9eff;
}
.tags-section {
  border-top: 1px solid #333;
}
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.section-header h3 {
  color: #aaa;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin: 0;
}
.icon-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
}
</style>
