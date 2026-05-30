<template>
  <div class="folder-node">
    <div
      class="node-row"
      :class="{ selected: foldersStore.selectedPath === node.path }"
      :style="{ paddingLeft: depth * 16 + 8 + 'px' }"
      @click="selectFolder"
      @contextmenu.prevent="onContext"
    >
      <span v-if="hasChildren" class="expand-icon" @click.stop="toggleExpand">
        {{ isExpanded ? '▼' : '▶' }}
      </span>
      <span v-else class="expand-icon spacer">·</span>
      <span class="folder-icon">📁</span>
      <span class="folder-name">{{ node.name }}</span>
      <span class="file-count">{{ node.file_count }}</span>
    </div>
    <template v-if="isExpanded && node.children?.length">
      <FolderNode
        v-for="child in node.children"
        :key="child.path"
        :node="child"
        :depth="depth + 1"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useFoldersStore } from '../../stores/folders'
import { useFiltersStore } from '../../stores/filters'
import type { FolderNode as FolderNodeType } from '../../types/vault'

const props = defineProps<{
  node: FolderNodeType
  depth: number
}>()

const foldersStore = useFoldersStore()
const filtersStore = useFiltersStore()

const hasChildren = computed(() => props.node.children?.length > 0)
const isExpanded = computed(() => foldersStore.expandedPaths.includes(props.node.path))

const toggleExpand = () => foldersStore.toggleFolder(props.node.path)
const selectFolder = () => {
  // Toggle: clicking the same selected folder deselects it → shows all files
  if (foldersStore.selectedPath === props.node.path) {
    foldersStore.clearSelection()
    filtersStore.setFolderFilter('')
  } else {
    foldersStore.selectFolder(props.node.path)
    filtersStore.setFolderFilter(props.node.path)
  }
}
const onContext = () => {
  // TODO: Context menu - "Exclude from indexing"
}
</script>

<style scoped>
.node-row {
  display: flex; align-items: center; gap: 4px;
  padding: 3px 8px; cursor: pointer; border-radius: 4px; margin: 1px 0;
}
.node-row:hover { background: #2a2a2a; }
.node-row.selected { background: #5b8af5; color: #fff; }
.expand-icon { font-size: 10px; width: 14px; text-align: center; color: #888; }
.folder-icon { font-size: 14px; }
.folder-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #ddd; }
.file-count { font-size: 11px; color: #666; }
</style>
