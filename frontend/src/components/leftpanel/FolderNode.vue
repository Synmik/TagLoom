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
        <ChevronDown v-if="isExpanded" :size="12" />
        <ChevronRight v-else :size="12" />
      </span>
      <span v-else class="expand-icon spacer"></span>
      <Folder class="folder-icon" :size="14" />
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
  <ContextMenu :visible="visible" :x="x" :y="y" :items="items" @close="close" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ChevronDown, ChevronRight, Folder, Ban, FolderOpen } from '@lucide/vue'
import { useFoldersStore } from '../../stores/folders'
import { useFiltersStore } from '../../stores/filters'
import { useContextMenu, type ContextMenuItem } from '../../composables/useContextMenu'
import { ClipboardSetText } from '../../../wailsjs/runtime/runtime'
import { AddExcludedFolder } from '../../api/backend'
import { useToast } from '../../composables/useToast'
import ContextMenu from '../common/ContextMenu.vue'
import type { FolderNode as FolderNodeType } from '../../types/vault'

const props = defineProps<{
  node: FolderNodeType
  depth: number
}>()

const foldersStore = useFoldersStore()
const filtersStore = useFiltersStore()
const { success, error: toastError } = useToast()
const { visible, x, y, items, open, close } = useContextMenu()

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
const onContext = (e: MouseEvent) => {
  const menuItems: ContextMenuItem[] = [
    {
      type: 'item',
      label: 'Exclude from indexing',
      icon: 'ban',
      action: async () => {
        try {
          await AddExcludedFolder(props.node.path)
          success(`Excluded "${props.node.name}"`)
        } catch (err: any) {
          toastError(err.message || 'Failed to exclude folder')
        }
      },
    },
    { type: 'divider' },
    {
      type: 'item',
      label: 'Copy path',
      icon: 'folder-open',
      action: () => {
        ClipboardSetText(props.node.path).then((ok) => {
          if (ok) success('Path copied')
          else toastError('Failed to copy path')
        })
      },
    },
  ]

  open(e, menuItems)
}
</script>

<style scoped>
.node-row {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 10px; cursor: pointer; border-radius: 6px; margin: 1px 0;
  font-size: 13px;
}
.node-row:hover { background: #1e1e1e; }
.node-row.selected { background: #14532d; color: #e8e8e8; }
.expand-icon { font-size: 10px; width: 14px; text-align: center; color: #666; flex-shrink: 0; }
.folder-icon { font-size: 14px; color: #888; flex-shrink: 0; }
.folder-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #ccc; }
.file-count { font-size: 11px; color: #555; background: #1a1a1a; padding: 1px 6px; border-radius: 10px; }
</style>
