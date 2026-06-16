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
  <ConfirmDialog
    v-if="showDeleteConfirm"
    :message="deleteConfirmMessage"
    confirm-text="Delete"
    @confirm="confirmDeleteOriginal"
    @cancel="showDeleteConfirm = false"
  />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useSelection } from '../../composables/useSelection'
import { useContextMenu, type ContextMenuItem } from '../../composables/useContextMenu'
import { usePreviewStore } from '../../stores/preview'
import { useFilesStore } from '../../stores/files'
import { useUIStore } from '../../stores/ui'
import { useVaultStore } from '../../stores/vault'
import { ClipboardSetText } from '../../../wailsjs/runtime/runtime'
import { useToast } from '../../composables/useToast'
import ContextMenu from '../common/ContextMenu.vue'
import ConfirmDialog from '../common/ConfirmDialog.vue'
import type { File } from '../../types/file'

const props = defineProps<{ file: File }>()
const { isSelected, toggleSelection } = useSelection()
const previewStore = usePreviewStore()
const filesStore = useFilesStore()
const uiStore = useUIStore()
const vaultStore = useVaultStore()
const { success, error: toastError } = useToast()
const { visible, x, y, items, open, close } = useContextMenu()
const showDeleteConfirm = ref(false)

const filename = computed(() => props.file.vault_path.split(/[\\/]/).pop() || '')
const absolutePath = computed(() => {
  const vp = vaultStore.currentVault?.path
  if (!vp) return props.file.vault_path
  // vault_path is relative from vault root; join with vault path
  return vp + '\\' + props.file.vault_path
})
const tagsText = computed(() => '—') // TODO: Load tags for this file
const deleteConfirmMessage = computed(() =>
  `Move "${filename.value}" to Recycle Bin?\n\nThe original file will be moved to Recycle Bin and the thumbnail removed.`
)

const confirmDeleteOriginal = async () => {
  showDeleteConfirm.value = false
  try {
    await filesStore.deleteOriginalFile(props.file.id)
    success('File deleted')
    await filesStore.reloadFiles()
  } catch (e) {
    toastError('Failed to delete file')
    console.error(e)
  }
}

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

// Extensions that browsers can display in <img> / <video> tags.
// Everything else opens with the OS default application.
const browserPreviewable = new Set([
  '.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.avif',
])

const isBrowserPreviewable = (path: string): boolean => {
  const ext = path.split('.').pop()?.toLowerCase() || ''
  return browserPreviewable.has(`.${ext}`)
}

const openPreview = async () => {
  if (!isBrowserPreviewable(props.file.vault_path)) {
    await filesStore.openOriginalFile(props.file.id)
    return
  }
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
      icon: 'heart',
      action: async () => {
        await filesStore.toggleFavorite(props.file)
      },
    },
    { type: 'divider' },
    {
      type: 'item',
      label: 'Open file',
      icon: 'file-text',
      action: async () => {
        try {
          await filesStore.openOriginalFile(props.file.id)
        } catch (e) {
          toastError('Failed to open file')
          console.error(e)
        }
      },
    },
    {
      type: 'item',
      label: 'Open file location',
      icon: 'folder-open',
      action: async () => {
        try {
          await filesStore.openFileFolder(props.file.id)
        } catch (e) {
          toastError('Failed to open folder')
          console.error(e)
        }
      },
    },
    { type: 'divider' },
    {
      type: 'item',
      label: 'Copy filename',
      icon: 'file',
      action: () => copyToClipboard(filename.value, 'Filename'),
    },
    {
      type: 'item',
      label: 'Copy path',
      icon: 'folder',
      action: () => copyToClipboard(absolutePath.value, 'Path'),
    },
    { type: 'divider' },
    {
      type: 'item',
      label: 'Batch edit',
      icon: 'pencil',
      action: () => {
        uiStore.openBatchEdit()
      },
    },
    { type: 'divider' },
    {
      type: 'item',
      label: 'Delete from vault',
      icon: 'trash2',
      action: async () => {
        await filesStore.deleteFile(props.file.id)
        success('File removed from vault')
        await filesStore.reloadFiles()
      },
    },
    {
      type: 'item',
      label: 'Delete original file',
      icon: 'alert-triangle',
      action: () => {
        close()
        showDeleteConfirm.value = true
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
  border-radius: 6px;
  height: 44px;
  padding: 4px 10px;
  box-sizing: border-box;
}
.list-row:hover { background: #1a1a1a; }
.list-row.selected { background: #14532d; }
.col-name { color: #ccc; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-tags { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; min-height: 0; }
.tag-chip {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 10px;
  font-size: 11px;
  line-height: 1.4;
  white-space: nowrap;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tag-more { color: #555; font-size: 11px; }
.no-tags { color: #666; font-size: 12px; }
.col-date, .col-rating { color: #666; font-size: 12px; }
</style>
