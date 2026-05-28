import { defineStore } from 'pinia'
import type { File, FileMetadata } from '../types/file'
import type { Tag } from '../types/tag'
import { GetFileMetadata, GetFileTags, UpdateFile } from '../api/backend'

export const usePreviewStore = defineStore('preview', {
  state: () => ({
    currentFile: null as File | null,
    metadata: null as FileMetadata | null,
    tags: [] as Tag[],
    isLoading: false,
  }),
  actions: {
    async setFile(file: File | null) {
      this.currentFile = file
      this.metadata = null
      this.tags = []
      if (file) {
        await this.loadFileDetails(file.id)
      }
    },
    async loadFileDetails(fileID: number) {
      this.isLoading = true
      try {
        const [metadata, tags] = await Promise.all([
          GetFileMetadata(fileID),
          GetFileTags(fileID),
        ])
        this.metadata = metadata
        this.tags = tags
      } finally {
        this.isLoading = false
      }
    },
    // Helper: build a full FileUpdate with defaults, then override the field
    _updateField(field: string, value: any) {
      if (!this.currentFile) return Promise.resolve()
      const f = this.currentFile
      return UpdateFile({
        id: f.id,
        name: field === 'name' ? value : f.name,
        notes: field === 'notes' ? value : f.notes,
        link: field === 'link' ? value : f.link,
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
  },
})
