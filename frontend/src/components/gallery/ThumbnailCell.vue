<template>
  <div
    class="thumbnail-cell"
    :class="{ selected: isSelected(file) }"
    :data-file-id="file.id"
    @click="(e) => handleClick(e)"
    @dblclick="openPreview"
    @contextmenu.prevent="onContextMenu"
  >
    <div class="thumbnail-wrapper">
      <img :src="thumbnailUrl" :alt="filename" class="thumbnail" draggable="false" />
      <span class="format-badge">{{ formatName }}</span>
      <Heart v-if="file.is_favorite === 1" :size="14" class="favorite-badge" fill="currentColor" />
    </div>
    <div class="file-name">{{ filename }}</div>
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
import { computed, ref, onMounted } from 'vue'
import { Heart, FolderOpen, FolderSearch, FileText, Folder, Pencil, Trash2, AlertTriangle } from '@lucide/vue'
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

const thumbnailUrl = ref('')
const isLoading = ref(true)

const filename = computed(() => {
  const parts = props.file.vault_path.split(/[\\/]/)
  return parts[parts.length - 1] || props.file.vault_path
})

const formatName = computed(() => {
  const ext = props.file.vault_path.split('.').pop()?.toUpperCase() || ''
  return ext === 'JPG' || ext === 'JPEG' ? 'JPEG' : ext
})

const vaultPath = computed(() => vaultStore.currentVault?.path || '')
const absolutePath = computed(() => {
  if (!vaultPath.value) return props.file.vault_path
  return vaultPath.value + '\\' + props.file.vault_path
})
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

// Stable thumbnail URL — no Date.now() so the browser can cache the
// response and reuse it across sessions.  The vp= parameter (below)
// invalidates the cache only when the vault changes.
const thumbnailSrc = computed(() => {
  const vp = vaultPath.value
  const bust = vp ? `&vp=${encodeURIComponent(vp)}` : ''
  return `/api/thumbnail/${props.file.id}${bust}`
})

onMounted(async () => {
  // Preload via a hidden Image so we know when the thumbnail is ready.
  // The preload and the actual <img> use the exact same URL, so the
  // browser serves the preloaded response from its HTTP cache instead
  // of making a second request.
  const img = new Image()
  img.onload = () => {
    thumbnailUrl.value = thumbnailSrc.value
    isLoading.value = false
  }
  img.onerror = async () => {
    // Fallback to base64 data URL when the HTTP endpoint fails
    const dataUrl = await filesStore.getThumbnail(props.file.id)
    if (dataUrl) {
      thumbnailUrl.value = dataUrl
    }
    isLoading.value = false
  }
  img.src = thumbnailSrc.value
})

const handleClick = async (e: MouseEvent) => {
  // Single click (no modifiers): select file AND open its preview in the right panel
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
    // Open with OS default app (TIFF, JPEGXL, SVG, video, etc.)
    await filesStore.openOriginalFile(props.file.id)
    return
  }
  // Browser can display it — open in-app preview modal
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
  // Ensure this file is selected before opening batch edit
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
      icon: 'folder-open',
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
      icon: 'folder-search',
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
      icon: 'file-text',
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
      icon: 'trash',
      action: async () => {
        await filesStore.deleteFile(props.file.id)
        success('File removed from vault')
        await filesStore.reloadFiles()
      },
    },
    {
      type: 'item',
      label: 'Delete original file',
      icon: 'shredder',
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
.thumbnail-cell {
  cursor: pointer; border-radius: 8px; overflow: hidden;
  border: 2px solid transparent; transition: border-color 0.15s;
}
.thumbnail-cell.selected { border-color: #22c55e; }
.thumbnail-wrapper {
  position: relative; aspect-ratio: 1; background: #161616; overflow: hidden;
}
.thumbnail { width: 100%; height: 100%; object-fit: contain; }
.format-badge {
  position: absolute; top: 4px; left: 4px;
  background: rgba(0,0,0,0.75); color: #ccc; font-size: 9px;
  font-weight: 600;
  padding: 2px 6px; border-radius: 4px; text-transform: uppercase;
  letter-spacing: 0.3px;
}
.favorite-badge {
  position: absolute; top: 4px; right: 4px; color: #ef4444; font-size: 14px;
}
.file-name {
  padding: 5px 8px; font-size: 11px; color: #999;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
</style>
