import { defineStore } from 'pinia'
import type { File, FileUpdate, FilePage, FileFilter, SortOpts } from '../types/file'
import { GetFiles, UpdateFile, SearchFiles, GenerateThumbnail, GetThumbnailData } from '../api/backend'

export const useFilesStore = defineStore('files', {
  state: () => ({
    files: [] as File[],
    selectedFiles: [] as File[],
    totalCount: 0,
    page: 0,
    limit: 50,
    isLoading: false,
    // Cache of fileID → data URL for thumbnails
    thumbnailCache: new Map<number, string>(),
  }),
  getters: {
    hasSelection: (state: any) => state.selectedFiles.length > 0,
    selectionCount: (state: any) => state.selectedFiles.length,
  },
  actions: {
    async loadFiles(
      filter: Partial<FileFilter> = {},
      sort: SortOpts = { field: 'indexed_at', order: 'desc' },
    ) {
      this.isLoading = true
      try {
        const result: FilePage = await GetFiles(
          { folder_path: '', tag_ids: [], file_formats: [], min_rating: 0, favorites_only: false, ...filter },
          sort,
          this.page,
          this.limit,
        )
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
    async updateFile(update: Partial<FileUpdate> & { id: number }) {
      const { id, ...rest } = update
      const full: FileUpdate = {
        id,
        name: '',
        notes: '',
        link: '',
        rating: 0,
        is_favorite: 0,
        ...rest,
      }
      await UpdateFile(full)
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
      const results = await SearchFiles(query, limit)
      this.files = results
      this.totalCount = results.length
    },

    /** Generate a thumbnail for a single file and return its data URL */
    async generateThumbnail(fileID: number): Promise<string | null> {
      try {
        await GenerateThumbnail(fileID)
        const dataUrl = await GetThumbnailData(fileID)
        this.thumbnailCache.set(fileID, dataUrl)
        return dataUrl
      } catch (e) {
        console.warn('Failed to generate thumbnail for', fileID, e)
        return null
      }
    },

    /** Get a cached thumbnail data URL, or generate one if missing */
    async getThumbnail(fileID: number): Promise<string | null> {
      const cached = this.thumbnailCache.get(fileID)
      if (cached) return cached
      return this.generateThumbnail(fileID)
    },

    /** Generate thumbnails for all currently loaded files */
    async generateAllThumbnails() {
      const fileIDs = this.files.map(f => f.id)
      const results = await Promise.allSettled(
        fileIDs.map(id => this.generateThumbnail(id))
      )
      const success = results.filter(r => r.status === 'fulfilled' && r.value !== null).length
      console.log(`Generated ${success}/${fileIDs.length} thumbnails`)
    },

    /** Clear the thumbnail cache */
    clearThumbnailCache() {
      this.thumbnailCache.clear()
    },
  },
})
