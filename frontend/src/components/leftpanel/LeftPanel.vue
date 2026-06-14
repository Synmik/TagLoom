<template>
  <aside class="left-panel" :style="panelStyle">
    <!-- Filters bar -->
    <div class="filters-bar">
      <button
        class="filter-btn"
        :class="{ active: filtersStore.activeFilters.favoritesOnly }"
        @click="toggleFavorites"
        title="Show only favorites"
      >
        <Star :size="14" :fill="filtersStore.activeFilters.favoritesOnly ? 'currentColor' : 'none'" />
        <span>Favorites</span>
      </button>
      <button
        class="filter-btn"
        :class="{ active: filtersStore.activeFilters.untaggedOnly }"
        @click="toggleUntagged"
        title="Show only untagged items"
      >
        <Tags :size="14" />
        <span>Untagged</span>
      </button>
    </div>

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
import { FolderOpen, Plus, Star, Tags } from '@lucide/vue'
import FolderTree from './FolderTree.vue'
import TagTree from './TagTree.vue'
import TagManagerModal from '../modals/TagManagerModal.vue'
import { useVaultStore } from '../../stores/vault'
import { useUIStore } from '../../stores/ui'
import { useFiltersStore } from '../../stores/filters'
import type { Tag } from '../../types/tag'

const uiStore = useUIStore()
const filtersStore = useFiltersStore()
const panelStyle = computed(() => ({ width: `${uiStore.leftPanelWidth}px` }))

const vaultStore = useVaultStore()
const showTagManager = ref(false)
const editingTag = ref<Tag | null>(null)

const openVault = () => vaultStore.pickAndOpenVault()

const toggleFavorites = () => {
  const newState = !filtersStore.activeFilters.favoritesOnly
  filtersStore.setFavoritesFilter(newState)
}

const toggleUntagged = () => {
  const newState = !filtersStore.activeFilters.untaggedOnly
  filtersStore.setUntaggedFilter(newState)
}

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
  height: 100%;
  overflow: hidden;
  border-right: 1px solid #1a1a1a;
  background: #111111;
  flex-shrink: 0;
}
.filters-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 8px;
  border-bottom: 1px solid #1a1a1a;
  flex-shrink: 0;
}
.filter-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  background: #1a1a1a;
  color: #888;
  border: 1px solid #2a2a2a;
  border-radius: 6px;
  padding: 4px 10px;
  font-size: 11px;
  font-family: 'Inter', sans-serif;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}
.filter-btn:hover {
  color: #e8e8e8;
  border-color: #444;
}
.filter-btn.active {
  background: #22c55e;
  color: #000;
  border-color: #22c55e;
}
.panel-section {
  overflow-y: auto;
  padding: 0 8px 8px 8px;
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
  background: #22c55e;
}
.tags-section {
  border-top: 1px solid #1a1a1a;
}
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0 4px 0;
  margin-bottom: 8px;
  position: sticky;
  top: 0;
  background: #111111;
  z-index: 1;
}
.section-header h3 {
  color: #888;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.8px;
  margin: 0;
}
.icon-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: #888;
  border-radius: 4px;
  padding: 2px;
  transition: color 0.15s;
}
.icon-btn:hover {
  color: #e8e8e8;
}
</style>
