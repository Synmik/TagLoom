import { defineStore } from 'pinia'
import type { File, FileMetadata } from '../types/file'
import type { Tag } from '../types/tag'

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
        // @ts-ignore
        const [metadata, tags] = await Promise.all([
          window.go.main.app.GetFileMetadata(fileID),
          window.go.main.app.GetFileTags(fileID),
        ])
        this.metadata = metadata
        this.tags = tags
      } finally {
        this.isLoading = false
      }
    },
    async updateName(value: string) {
      if (!this.currentFile) return
      // @ts-ignore
      await window.go.main.app.UpdateFile({ id: this.currentFile.id, name: value })
      this.currentFile.name = value
    },
    async updateNotes(value: string) {
      if (!this.currentFile) return
      // @ts-ignore
      await window.go.main.app.UpdateFile({ id: this.currentFile.id, notes: value })
      this.currentFile.notes = value
    },
    async updateLink(value: string) {
      if (!this.currentFile) return
      // @ts-ignore
      await window.go.main.app.UpdateFile({ id: this.currentFile.id, link: value })
      this.currentFile.link = value
    },
    async setRating(rating: number) {
      if (!this.currentFile) return
      // @ts-ignore
      await window.go.main.app.UpdateFile({ id: this.currentFile.id, rating })
      this.currentFile.rating = rating
    },
    async toggleFavorite() {
      if (!this.currentFile) return
      const newFav = this.currentFile.is_favorite === 1 ? 0 : 1
      // @ts-ignore
      await window.go.main.app.UpdateFile({ id: this.currentFile.id, is_favorite: newFav })
      this.currentFile.is_favorite = newFav
    },
  },
})
