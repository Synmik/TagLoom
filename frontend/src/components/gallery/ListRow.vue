<template>
  <div
    class="list-row"
    :class="{ selected: isSelected(file) }"
    :data-file-id="file.id"
    @click="(e) => handleClick(e)"
    @dblclick="openPreview"
    @contextmenu.prevent="onContextMenu"
  >
    <span class="col-name">{{ filename }}</span>
    <span class="col-tags">
      <template v-if="file.tags && file.tags.length">
        <span
          v-for="tag in file.tags.slice(0, 3)"
          :key="tag.id"
          class="tag-chip"
          :style="{ background: tag.color || '#555', color: tag.color ? 'white' : '#ccc' }"
        >{{ tag.name }}</span>
        <span v-if="file.tags.length > 3" class="tag-more">+{{ file.tags.length - 3 }}</span>
      </template>
      <span v-else class="no-tags">—</span>
    </span>
    <span class="col-date">{{ formattedDate }}</span>
    <span class="col-rating">{{ '★'.repeat(file.rating) }}{{ '☆'.repeat(5 - file.rating) }}</span>
  </div>
  <ContextMenu :visible="visible" :x="x" :y="y" :items="items" @close="close" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useSelection } from '../../composables/useSelection'
import { useContextMenu, type ContextMenuItem } from '../../composables/useContextMenu'
import { usePreviewStore } from '../../stores/preview'
import { useFilesStore } from '../../stores/files'
import { useUIStore } from '../../stores/ui'
import { ClipboardSetText } from '../../../wailsjs/runtime/runtime'
import { useToast } from '../../composables/useToast'
import ContextMenu from '../common/ContextMenu.vue'
import type { File } from '../../types/file'

const props = defineProps<{ file: File }>()
const { isSelected, toggleSelection } = useSelection()
const previewStore = usePreviewStore()
const filesStore = useFilesStore()
const uiStore = useUIStore()
const { success, error: toastError } = useToast()
const { visible, x, y, items, open, close } = useContextMenu()

const filename = computed(() => props.file.vault_path.split(/[\\/]/).pop() || '')

const formattedDate = computed(() => {
  if (!props.file.date_modified) return '—'
  const d = new Date(props.file.date_modified)
  return d.toLocaleString('en-US', { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
})

const handleClick = async (e: MouseEvent) => {
  if (!e.ctrlKey && !e.shiftKey) {
    await previewStore.setFile(props.file)
  }
  toggleSelection(props.file, e.ctrlKey, e.shiftKey)
}

const openPreview = () => {
  previewStore.setFile(props.file)
  previewStore.openFullPreview()
}

// ── Context menu ──────────────────────────────────────────────────

const copyToClipboard = async (text: string, label: string) => {
  const ok = await ClipboardSetText(text)
  if (ok) {
    success(`${label} copied`)
  } else {
    toastError(`Failed to copy ${label.toLowerCase()}`)
  }
}

const onContextMenu = (e: MouseEvent) => {
  if (!isSelected(props.file)) {
    toggleSelection(props.file, false, false)
  }

  const menuItems: ContextMenuItem[] = [
    {
      type: 'item',
      label: props.file.is_favorite === 1 ? 'Remove from favorites' : 'Add to favorites',
      icon: '♥',
      action: async () => {
        await filesStore.toggleFavorite(props.file)
      },
    },
    { type: 'divider' },
    {
      type: 'item',
      label: 'Copy filename',
      icon: '📄',
      action: () => copyToClipboard(filename.value, 'Filename'),
    },
    {
      type: 'item',
      label: 'Copy path',
      icon: '📂',
      action: () => copyToClipboard(props.file.vault_path, 'Path'),
    },
    { type: 'divider' },
    {
      type: 'item',
      label: 'Batch edit',
      icon: '✏️',
      action: () => {
        uiStore.openBatchEdit()
      },
    },
    { type: 'divider' },
    {
      type: 'item',
      label: 'Delete from vault',
      icon: '🗑️',
      action: async () => {
        await filesStore.deleteFile(props.file.id)
        success('File removed from vault')
        await filesStore.reloadFiles()
      },
    },
  ]

  open(e, menuItems)
}
</script>

<style scoped>
.list-row {
  display: grid;
  grid-template-columns: 1fr 1fr 160px 60px;
  align-items: center;
  cursor: pointer;
  border-radius: 4px;
  height: 44px;
  padding: 4px 8px;
  box-sizing: border-box;
}
.list-row:hover { background: #1e1e1e; }
.list-row.selected { background: #2a3a5a; }
.col-name { color: #ddd; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-tags { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; min-height: 0; }
.tag-chip {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
  line-height: 1.4;
  white-space: nowrap;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tag-more { color: #666; font-size: 11px; }
.no-tags { color: #888; font-size: 12px; }
.col-date, .col-rating { color: #666; font-size: 12px; }
</style>
