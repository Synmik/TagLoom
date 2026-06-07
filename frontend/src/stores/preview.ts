import { defineStore, storeToRefs } from 'pinia'
import { watch } from 'vue'
import type { File, FileMetadata } from '../types/file'
import type { Tag } from '../types/tag'
import { GetFileMetadata, GetFileTags, UpdateFile } from '../api/backend'
import { useFilesStore } from './files'

export const usePreviewStore = defineStore('preview', {
  state: () => ({
    currentFile: null as File | null,
    metadata: null as FileMetadata | null,
    tags: [] as Tag[],
    isLoading: false,
    previewModalOpen: false,
    _loadSeq: 0, // monotonic counter to discard stale loadFileDetails responses
  }),
  actions: {
    async setFile(file: File | null) {
      // Skip if the same file is already loaded — prevents duplicate calls
      // from both handleClick and the _syncSelection watcher racing
      if (file && this.currentFile && this.currentFile.id === file.id) {
        return
      }
      this.currentFile = file
      this.metadata = null
      this.tags = []
      if (file) {
        await this.loadFileDetails(file.id)
      }
    },

    /** Sync with filesStore selection — auto-loads preview when single file is selected */
    _syncSelection() {
      const filesStore = useFilesStore()
      const { selectedFiles } = storeToRefs(filesStore)

      watch(selectedFiles, (files) => {
        if (files.length === 1) {
          this.setFile(files[0])
        } else if (files.length === 0) {
          // Don't clear — keep last file visible until user explicitly deselects
        } else {
          // Multi-select: keep showing the last single-selected file's details
        }
      })
    },
    async loadFileDetails(fileID: number) {
      this.isLoading = true
      const seq = ++this._loadSeq
      // Fetch metadata and tags independently so one failure doesn't block the other
      const metadataPromise = GetFileMetadata(fileID).catch((e) => {
        console.warn(`[previewStore] metadata fetch failed for ${fileID}:`, e)
        return null
      })
      const tagsPromise = GetFileTags(fileID).catch((e) => {
        console.warn(`[previewStore] tags fetch failed for ${fileID}:`, e)
        return []
      })
      const [metadata, tags] = await Promise.all([metadataPromise, tagsPromise])
      // Only apply if no newer load has started
      if (seq === this._loadSeq) {
        this.metadata = metadata
        this.tags = Array.isArray(tags) ? tags : []
        this.isLoading = false
      }
    },
    // Helper: build a full FileUpdate with defaults, then override the field
    _updateField(field: string, value: any) {
      if (!this.currentFile) return Promise.resolve()
      const f = this.currentFile
      // Coerce nullable string fields (name, notes, link) to empty string
      // so Go backend receives a valid string instead of undefined
      return UpdateFile({
        id: f.id,
        name: field === 'name' ? value : (f.name ?? ''),
        notes: field === 'notes' ? value : (f.notes ?? ''),
        link: field === 'link' ? value : (f.link ?? ''),
        rating: field === 'rating' ? value : f.rating,
        is_favorite: field === 'is_favorite' ? value : f.is_favorite,
      }).then(() => {
        if (this.currentFile) {
          (this.currentFile as any)[field] = value
        }
      })
    },
    async updateName(value: string) {
      return this._updateField('name', value)
    },
    async updateNotes(value: string) {
      return this._updateField('notes', value)
    },
    async updateLink(value: string) {
      return this._updateField('link', value)
    },
    async setRating(rating: number) {
      return this._updateField('rating', rating)
    },
    async toggleFavorite() {
      if (!this.currentFile) return
      const newFav = this.currentFile.is_favorite === 1 ? 0 : 1
      return this._updateField('is_favorite', newFav)
    },

    openFullPreview() {
      this.previewModalOpen = true
    },
    closeFullPreview() {
      this.previewModalOpen = false
    },
  },
})
