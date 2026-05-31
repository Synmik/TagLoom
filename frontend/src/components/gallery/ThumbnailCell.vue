<template>
  <div
    class="thumbnail-cell"
    :class="{ selected: isSelected(file) }"
    @click="(e) => handleClick(e)"
    @dblclick="openPreview"
    @contextmenu.prevent="onContextMenu"
  >
    <div class="thumbnail-wrapper">
      <img :src="thumbnailUrl" :alt="filename" class="thumbnail" loading="lazy" />
      <span class="format-badge">{{ formatName }}</span>
      <span v-if="file.is_favorite === 1" class="favorite-badge">♥</span>
    </div>
    <div class="file-name">{{ filename }}</div>
  </div>
  <ContextMenu :visible="visible" :x="x" :y="y" :items="items" @close="close" />
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useSelection } from '../../composables/useSelection'
import { useContextMenu, type ContextMenuItem } from '../../composables/useContextMenu'
import { usePreviewStore } from '../../stores/preview'
import { useFilesStore } from '../../stores/files'
import { useUIStore } from '../../stores/ui'
import { useVaultStore } from '../../stores/vault'
import { ClipboardSetText } from '../../../wailsjs/runtime/runtime'
import { useToast } from '../../composables/useToast'
import ContextMenu from '../common/ContextMenu.vue'
import type { File } from '../../types/file'

const props = defineProps<{ file: File }>()
const { isSelected, toggleSelection } = useSelection()
const previewStore = usePreviewStore()
const filesStore = useFilesStore()
const uiStore = useUIStore()
const vaultStore = useVaultStore()
const { success, error: toastError } = useToast()
const { visible, x, y, items, open, close } = useContextMenu()

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

const thumbnailUrlValue = computed(() => {
  // Include vault path as cache-busting parameter so switching vaults
  // forces the browser to fetch fresh thumbnails instead of serving
  // stale cached images from the previous vault.
  const vp = vaultPath.value
  const bust = vp ? `&vp=${encodeURIComponent(vp)}` : ''
  return `/api/thumbnail/${props.file.id}?nocache=${Date.now()}${bust}`
})

onMounted(async () => {
  // Use HTTP endpoint if available, fallback to base64
  const img = new Image()
  img.onload = () => {
    thumbnailUrl.value = thumbnailUrlValue.value
    isLoading.value = false
  }
  img.onerror = async () => {
    // Fallback to base64 data URL
    const dataUrl = await filesStore.getThumbnail(props.file.id)
    if (dataUrl) {
      thumbnailUrl.value = dataUrl
    }
    isLoading.value = false
  }
  img.src = thumbnailUrlValue.value
})

const handleClick = async (e: MouseEvent) => {
  // Single click (no modifiers): select file AND open its preview in the right panel
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
  // Ensure this file is selected before opening batch edit
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
.thumbnail-cell {
  cursor: pointer; border-radius: 6px; overflow: hidden;
  border: 2px solid transparent; transition: border-color 0.15s;
}
.thumbnail-cell.selected { border-color: #5b8af5; }
.thumbnail-wrapper {
  position: relative; aspect-ratio: 1; background: #1e1e1e; overflow: hidden;
}
.thumbnail { width: 100%; height: 100%; object-fit: contain; }
.format-badge {
  position: absolute; top: 4px; left: 4px;
  background: rgba(0,0,0,0.7); color: #fff; font-size: 9px;
  padding: 2px 5px; border-radius: 3px; text-transform: uppercase;
}
.favorite-badge {
  position: absolute; top: 4px; right: 4px; color: #ff4444; font-size: 14px;
}
.file-name {
  padding: 4px 6px; font-size: 11px; color: #aaa;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
</style>
