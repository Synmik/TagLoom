import { defineStore } from 'pinia'
import type { File, FileUpdate, FilePage, FileFilter, SortOpts } from '../types/file'
import { GetFiles, UpdateFile, SearchFiles, GenerateThumbnail, GenerateThumbnailsPool, GetThumbnailData, AddTagsToFiles, RemoveTagsFromFiles, SetRatingForFiles, SetFavoriteForFiles, DeleteFile } from '../api/backend'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { useFiltersStore } from './filters'

export const useFilesStore = defineStore('files', {
  state: () => ({
    files: [] as File[],
    selectedFiles: [] as File[],
    totalCount: 0,
    page: 0,
    limit: 200,
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
      append: boolean = false,
    ) {
      this.isLoading = true
      try {
        const filtersStore = useFiltersStore()
        const activeFilter = filtersStore.asBackendFilter
        const mergedFilter = { ...activeFilter, ...filter }

        const result: FilePage = await Promise.race([
          GetFiles(mergedFilter, sort, this.page, this.limit),
          new Promise<FilePage>((_, reject) =>
            setTimeout(() => reject(new Error('GetFiles timeout (2s)')), 2000)
          ),
        ])
        // Go returns nil slice → null in JS; result itself can also be null
        const fileArray: File[] = result && Array.isArray(result.files) ? result.files : []
        if (append) {
          this.files = [...this.files, ...fileArray]
        } else {
          this.files = fileArray
        }
        this.totalCount = result?.total_count ?? 0
      } catch (e) {
        console.error('filesStore.loadFiles failed:', e)
        if (!append) {
          this.files = []
          this.totalCount = 0
        }
      } finally {
        this.isLoading = false
      }
    },

    /** Reset to page 0 and reload with current filters */
    async reloadFiles() {
      this.page = 0
      this.files = []
      await this.loadFiles()
    },

    /** Load all files at once (for virtual scrolling). Uses a high limit so the
     * virtualizer has the full metadata array to work with. Thumbnails are still
     * loaded lazily per-cell. */
    async loadAllFiles() {
      this.isLoading = true
      try {
        const filtersStore = useFiltersStore()
        const activeFilter = filtersStore.asBackendFilter

        // Load in large chunks (1000 per request) until we have everything
        const ALL_LIMIT = 10_000
        const result: FilePage = await Promise.race([
          GetFiles(activeFilter, { field: 'indexed_at', order: 'desc' }, 0, ALL_LIMIT),
          new Promise<FilePage>((_, reject) =>
            setTimeout(() => reject(new Error('GetFiles timeout (5s)')), 5000)
          ),
        ])
        const fileArray: File[] = result && Array.isArray(result.files) ? result.files : []
        this.files = fileArray
        this.totalCount = result?.total_count ?? 0
        this.page = 0
      } catch (e) {
        console.error('filesStore.loadAllFiles failed:', e)
        this.files = []
        this.totalCount = 0
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
      this.isLoading = true
      try {
        const results = await SearchFiles(query, limit)
        // Go returns nil slice → null in JS when no results
        const fileArray: File[] = Array.isArray(results) ? results : []
        this.files = fileArray
        this.totalCount = fileArray.length
      } catch (e) {
        console.error('filesStore.searchFiles failed:', e)
        this.files = []
        this.totalCount = 0
      } finally {
        this.isLoading = false
      }
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

    // ── Batch operations ──

    /** Add tags to all selected files in a single backend call */
    async batchAddTags(tagIDs: number[]) {
      const fileIDs = this.selectedFiles.map(f => f.id)
      if (fileIDs.length === 0 || tagIDs.length === 0) return
      await AddTagsToFiles(fileIDs, tagIDs)
    },

    /** Remove tags from all selected files in a single backend call */
    async batchRemoveTags(tagIDs: number[]) {
      const fileIDs = this.selectedFiles.map(f => f.id)
      if (fileIDs.length === 0 || tagIDs.length === 0) return
      await RemoveTagsFromFiles(fileIDs, tagIDs)
    },

    /** Set rating on all selected files */
    async batchSetRating(rating: number) {
      const fileIDs = this.selectedFiles.map(f => f.id)
      if (fileIDs.length === 0) return
      await SetRatingForFiles(fileIDs, rating)
      // Optimistic update
      for (const file of this.selectedFiles) {
        file.rating = rating
      }
    },

    /** Set favorite flag on all selected files */
    async batchSetFavorite(isFavorite: boolean) {
      const fileIDs = this.selectedFiles.map(f => f.id)
      if (fileIDs.length === 0) return
      await SetFavoriteForFiles(fileIDs, isFavorite ? 1 : 0)
      // Optimistic update
      const favValue = isFavorite ? 1 : 0
      for (const file of this.selectedFiles) {
        file.is_favorite = favValue
      }
    },

    /** Delete a file from the vault index (DB only, not from disk) */
    async deleteFile(fileID: number) {
      await DeleteFile(fileID)
      // Remove from local state
      const index = this.files.findIndex(f => f.id === fileID)
      if (index >= 0) {
        this.files.splice(index, 1)
      }
      // Remove from selection if selected
      const selIndex = this.selectedFiles.findIndex(f => f.id === fileID)
      if (selIndex >= 0) {
        this.selectedFiles.splice(selIndex, 1)
      }
      this.totalCount--
    },

    /** Generate thumbnails for all files using a 4-worker pool */
    async generateAllThumbnailsPool() {
      this.isLoading = true

      const progressUnsub = EventsOn('thumb:progress', (data: { current: number; total: number; generated: number }) => {
        this.isLoading = false
        console.log(`Thumbnail pool: ${data.current}/${data.total} (${data.generated} generated)`)
      })

      const completeUnsub = EventsOn('thumb:complete', (data: { generated: number; failed: number; total: number }) => {
        this.isLoading = false
        console.log(`Thumbnail pool complete: ${data.generated} generated, ${data.failed} failed out of ${data.total}`)
      })

      try {
        await GenerateThumbnailsPool()
      } catch (e) {
        console.error('Thumbnail pool failed:', e)
      } finally {
        this.isLoading = false
        progressUnsub()
        completeUnsub()
      }
    },
  },
})
