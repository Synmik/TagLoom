import { onMounted, onUnmounted } from 'vue'
import { useUIStore } from '../stores/ui'
import { useFilesStore } from '../stores/files'
import { usePreviewStore } from '../stores/preview'
import { useVaultStore } from '../stores/vault'
import { useToast } from './useToast'

/**
 * Global keyboard shortcut handler.
 * Register once in App.vue onMounted.
 *
 * Shortcuts:
 *   Ctrl+A    — Select all files matching current filters
 *   Ctrl+B    — Open batch edit modal (requires selection)
 *   Ctrl+C    — Copy original image to clipboard (single selection)
 *   Ctrl+F    — Focus search bar
 *   Ctrl+R    — Reindex/rescan the vault
 *   Ctrl+D    — Toggle favorite on selected file(s)
 *   Enter     — Open preview (same as double-click on thumbnail)
 *   Escape    — Close open modals (priority order), then clear selection
 *   ← / →     — Navigate gallery (previous / next file)
 */
export function useKeyboardShortcuts() {
  const uiStore = useUIStore()
  const filesStore = useFilesStore()
  const previewStore = usePreviewStore()
  const vaultStore = useVaultStore()
  const { info, success, error: error } = useToast()

  // Custom event channel for child components
  const listeners: Record<string, Set<() => void>> = {
    'navigate:scroll': new Set(),
  }

  /** Dispatch a window-level custom event that Gallery listens for */
  const requestFocusGallery = () => {
    window.dispatchEvent(new CustomEvent('tagloom:focus-gallery'))
  }

  const on = (event: string, fn: () => void) => {
    if (!listeners[event]) listeners[event] = new Set()
    listeners[event].add(fn)
    return () => listeners[event].delete(fn)
  }

  const emit = (event: string) => {
    listeners[event]?.forEach(fn => fn())
  }

  /** Dispatch a window-level custom event that SearchBar listens for */
  const requestFocusSearch = () => {
    window.dispatchEvent(new CustomEvent('tagloom:focus-search'))
  }

  // ── Handlers ────────────────────────────────────────────────────

  const handleNavigate = (direction: -1 | 1) => {
    const files = filesStore.files
    if (files.length === 0) return

    // Determine current file index
    const currentFile =
      previewStore.currentFile ??
      (filesStore.selectedFiles.length === 1 ? filesStore.selectedFiles[0] : null)

    let currentIndex = currentFile
      ? files.findIndex(f => f.id === currentFile.id)
      : -1

    if (currentIndex === -1) {
      // No file selected — go to first or last depending on direction
      currentIndex = direction === 1 ? 0 : files.length - 1
    } else {
      currentIndex += direction
      if (currentIndex < 0) currentIndex = files.length - 1
      if (currentIndex >= files.length) currentIndex = 0
    }

    const file = files[currentIndex]
    if (!file) return

    // Select and load preview
    filesStore.selectFile(file, false)
    previewStore.setFile(file)

    // Let Gallery scroll the cell into view
    emit('navigate:scroll')
  }

  const handleToggleFavorite = async (e: KeyboardEvent) => {
    e.preventDefault()

    // Single file selected (via preview or explicit selection)
    if (filesStore.selectedFiles.length === 1) {
      const file = filesStore.selectedFiles[0]
      await filesStore.toggleFavorite(file)
      // Also update preview store if showing same file
      if (previewStore.currentFile?.id === file.id) {
        previewStore.currentFile.is_favorite = file.is_favorite
      }
      return
    }

    // Multi-select: batch toggle
    if (filesStore.selectedFiles.length > 1) {
      const firstFile = filesStore.selectedFiles[0]
      const newFav = firstFile.is_favorite === 1 ? 0 : 1
      await filesStore.batchSetFavorite(newFav === 1)
      info(
        `${newFav === 1 ? 'Favorited' : 'Un-favorited'} ${filesStore.selectedFiles.length} files`
      )
      return
    }

    // No selection — do nothing silently
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

  /** Open preview for the currently selected file (same as double-click) */
  const handleOpenPreview = async () => {
    const file = filesStore.selectedFiles[0]
    if (!file) return

    if (!isBrowserPreviewable(file.vault_path)) {
      // Open with OS default app (TIFF, JPEGXL, SVG, video, etc.)
      try {
        await filesStore.openOriginalFile(file.id)
      } catch (e) {
        error('Failed to open file')
        console.error(e)
      }
      return
    }
    // Browser can display it — open in-app preview modal
    previewStore.setFile(file)
    previewStore.openFullPreview()
  }

  /** Copy the original image of the selected file to the system clipboard */
  const handleCopyToClipboard = async () => {
    const file = filesStore.selectedFiles[0]
    if (!file) return

    try {
      await filesStore.copyImageToClipboard(file.id)
      success('Image copied to clipboard')
    } catch (e) {
      error('Failed to copy image to clipboard')
      console.error(e)
    }
  }

  const handleSelectAll = () => {
    const files = filesStore.files
    if (files.length === 0) return
    filesStore.selectedFiles = [...files]
    info(`Selected ${files.length} files`)
  }

  const handleBatchEdit = () => {
    if (filesStore.selectedFiles.length === 0) return
    uiStore.openBatchEdit()
  }

  const handleRescan = async () => {
    if (vaultStore.isScanning) {
      info('Scan already in progress…')
      return
    }
    await vaultStore.rescanVault()
  }

  const handleCloseOrClear = () => {
    // Priority: FilePreviewModal → other modals → clear selection

    if (previewStore.previewModalOpen) {
      previewStore.closeFullPreview()
      return
    }

    if (uiStore.showBatchEdit) {
      uiStore.closeBatchEdit()
      return
    }

    if (uiStore.showVaultSettings) {
      uiStore.closeVaultSettings()
      return
    }

    if (uiStore.showTagManager) {
      uiStore.closeTagManager()
      return
    }

    // No modal open — clear selection
    filesStore.clearSelection()
  }

  // ── Key dispatcher ──────────────────────────────────────────────

  const onKeydown = (e: KeyboardEvent) => {
    const ctrl = e.ctrlKey || e.metaKey

    // Ctrl+A — select all files matching current filters
    if (ctrl && e.key === 'a') {
      const inInput = document.activeElement instanceof HTMLInputElement ||
        document.activeElement instanceof HTMLTextAreaElement ||
        document.activeElement instanceof HTMLSelectElement
      if (!inInput) {
        e.preventDefault()
        handleSelectAll()
      }
      return
    }

    // Ctrl+B — open batch edit modal
    if (ctrl && e.key === 'b') {
      e.preventDefault()
      handleBatchEdit()
      return
    }

    // Ctrl+C — copy original image to clipboard (only when not in an input)
    if (ctrl && e.key === 'c') {
      const inInput = document.activeElement instanceof HTMLInputElement ||
        document.activeElement instanceof HTMLTextAreaElement ||
        document.activeElement instanceof HTMLSelectElement
      if (!inInput && filesStore.selectedFiles.length >= 1) {
        e.preventDefault()
        handleCopyToClipboard()
      }
      return
    }

    // Ctrl+F — focus search
    if (ctrl && e.key === 'f') {
      e.preventDefault()
      requestFocusSearch()
      return
    }

    // Ctrl+R — rescan vault
    if (ctrl && e.key === 'r') {
      e.preventDefault()
      handleRescan()
      return
    }

    // Escape — close modal, defocus search, or clear selection
    if (e.key === 'Escape') {
      e.preventDefault()

      // Defocus search input and shift focus back to gallery
      const activeEl = document.activeElement
      if (activeEl instanceof HTMLInputElement && activeEl.classList.contains('search-input')) {
        activeEl.blur()
        requestFocusGallery()
        return
      }

      const inInput = activeEl instanceof HTMLInputElement ||
        activeEl instanceof HTMLTextAreaElement

      // Always close modals regardless of focus; only skip selection clear when in input
      if (uiStore.showBatchEdit || uiStore.showVaultSettings || uiStore.showTagManager ||
          previewStore.previewModalOpen) {
        handleCloseOrClear()
      } else if (!inInput) {
        filesStore.clearSelection()
      }
      return
    }

    // ← / → — navigate gallery
    // Always allow when preview modal is open; otherwise only when not in an input
    if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
      const inInput = document.activeElement instanceof HTMLInputElement ||
        document.activeElement instanceof HTMLTextAreaElement ||
        document.activeElement instanceof HTMLSelectElement

      if (previewStore.previewModalOpen || !inInput) {
        e.preventDefault()
        handleNavigate(e.key === 'ArrowLeft' ? -1 : 1)
      }
      return
    }

    // Ctrl+D — toggle favorite
    if (ctrl && e.key === 'd') {
      handleToggleFavorite(e)
      return
    }

    // Enter — open preview (same as double-click on thumbnail)
    if (e.key === 'Enter') {
      const inInput = document.activeElement instanceof HTMLInputElement ||
        document.activeElement instanceof HTMLTextAreaElement ||
        document.activeElement instanceof HTMLSelectElement
      if (!inInput && filesStore.selectedFiles.length >= 1) {
        e.preventDefault()
        handleOpenPreview()
      }
      return
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', onKeydown)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', onKeydown)
  })

  return { on }
}
