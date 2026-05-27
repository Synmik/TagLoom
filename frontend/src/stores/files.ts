import { defineStore } from 'pinia'
import type { File, FileUpdate, FilePage, FileFilter, SortOpts } from '../types/file'

export const useFilesStore = defineStore('files', {
  state: () => ({
    files: [] as File[],
    selectedFiles: [] as File[],
    totalCount: 0,
    page: 0,
    limit: 50,
    isLoading: false,
  }),
  getters: {
    hasSelection: (state) => state.selectedFiles.length > 0,
    selectionCount: (state) => state.selectedFiles.length,
  },
  actions: {
    async loadFiles(filter: FileFilter = {} as FileFilter, sort: SortOpts = { field: 'indexed_at', order: 'desc' }) {
      this.isLoading = true
      try {
        // @ts-ignore
        const result: FilePage = await window.go.main.app.GetFiles(filter, sort, this.page, this.limit)
        this.files = result.files
        this.totalCount = result.total_count
      } finally {
        this.isLoading = false
      }
    },
    selectFile(file: File, multi: boolean = false) {
      if (multi) {
        const index = this.selectedFiles.findIndex(f => f.id === file.id)
        if (index >= 0) {
          this.selectedFiles.splice(index, 1)
        } else {
          this.selectedFiles.push(file)
        }
      } else {
        this.selectedFiles = [file]
      }
    },
    clearSelection() {
      this.selectedFiles = []
    },
    async updateFile(update: FileUpdate) {
      // @ts-ignore
      await window.go.main.app.UpdateFile(update)
      // Update local state
      const index = this.files.findIndex(f => f.id === update.id)
      if (index >= 0) {
        Object.assign(this.files[index], update)
      }
    },
    async toggleFavorite(file: File) {
      const newFav = file.is_favorite === 1 ? 0 : 1
      await this.updateFile({ id: file.id, is_favorite: newFav })
    },
    async searchFiles(query: string, limit: number = 100) {
      // @ts-ignore
      const results = await window.go.main.app.SearchFiles(query, limit)
      this.files = results
      this.totalCount = results.length
    },
  },
})
