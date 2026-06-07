import { defineStore } from 'pinia'
import type { File, FileUpdate, FilePage, FileFilter, SortOpts } from '../types/file'
import { GetFiles, UpdateFile, SearchFiles, GenerateThumbnail, GenerateThumbnailsPool, GetThumbnailData, AddTagsToFiles, RemoveTagsFromFiles, SetRatingForFiles, SetFavoriteForFiles, DeleteFile, OpenOriginalFile, OpenFileFolder, DeleteOriginalFile } from '../api/backend'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { useFiltersStore } from './filters'
import { useUIStore } from './ui'

export const useFilesStore = defineStore('files', {
  state: () => ({
    files: [] as File[],
    selectedFiles: [] as File[],
    // When set, batch operations target ALL files in this folder (not just loaded ones)
    folderBulkEditPath: '',
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
      const uiStore = useUIStore()
      await this.loadFiles({}, { field: uiStore.sortBy, order: uiStore.sortOrder })
    },

    /** Load files incrementally for virtual scrolling.
     * Loads the first page and sets totalCount so the virtualizer knows the
     * total scroll height. Subsequent pages are loaded via loadNextPage() as
     * the user scrolls. Thumbnails are still loaded lazily per-cell. */
    async loadAllFiles() {
      this.isLoading = true
      this.page = 0
      this.files = []
      try {
        const filtersStore = useFiltersStore()
        const uiStore = useUIStore()
        const activeFilter = filtersStore.asBackendFilter
        const sortOpts = { field: uiStore.sortBy, order: uiStore.sortOrder }

        // Load first page with same page size as reloadFiles
        const result: FilePage = await Promise.race([
          GetFiles(activeFilter, sortOpts, 0, this.limit),
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

    /** Load the next page of files and append to the current list.
     * Used by infinite scroll / virtual grid to progressively load data
     * as the user scrolls. Uses the same page size as reloadFiles (this.limit). */
    async loadNextPage() {
      const files = this.files || []
      if (files.length >= this.totalCount) return // all loaded
      if (this.totalCount === 0) return
      if (this.isLoading) return

      this.isLoading = true
      try {
        const filtersStore = useFiltersStore()
        const uiStore = useUIStore()
        const activeFilter = filtersStore.asBackendFilter
        const sortOpts = { field: uiStore.sortBy, order: uiStore.sortOrder }

        // Calculate next page: files.length items are loaded across pages 0..page.
        // Since we just loaded page 'this.page', the next page is this.page + 1.
        const nextPage = this.page + 1
        const result: FilePage = await Promise.race([
          GetFiles(activeFilter, sortOpts, nextPage, this.limit),
          new Promise<FilePage>((_, reject) =>
            setTimeout(() => reject(new Error('GetFiles timeout (5s)')), 5000)
          ),
        ])
        const fileArray: File[] = result && Array.isArray(result.files) ? result.files : []
        this.files = [...this.files, ...fileArray]
        this.page = nextPage
      } catch (e) {
        console.error('filesStore.loadNextPage failed:', e)
      } finally {
        this.isLoading = false
      }
    },
    selectFile(file: File, multi: boolean = false) {
      // Clear folder bulk edit mode when user manually selects files
      this.folderBulkEditPath = ''
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
      this.folderBulkEditPath = ''
    },
    /** Set bulk edit scope to ALL files in a folder (without loading them into gallery) */
    setFolderBulkEdit(folderPath: string) {
      this.folderBulkEditPath = folderPath
      this.selectedFiles = []
    },
    /** Get all file IDs for current selection or folder bulk edit scope */
    async getAllSelectedIds(): Promise<number[]> {
      if (this.folderBulkEditPath) {
        // Fetch all file IDs for this folder from backend (without loading into gallery)
        const filtersStore = useFiltersStore()
        const activeFilter = filtersStore.asBackendFilter
        const allIds: number[] = []
        const limit = 500
        let page = 0
        let totalCount = 0

        // First get total count
        const firstResult: FilePage = await Promise.race([
          GetFiles({ ...activeFilter, folder_path: this.folderBulkEditPath }, { field: 'indexed_at', order: 'desc' }, 0, 1),
          new Promise<FilePage>((_, reject) =>
            setTimeout(() => reject(new Error('GetFiles timeout (5s)')), 5000)
          ),
        ])
        totalCount = firstResult?.total_count ?? 0

        // Fetch all pages of IDs
        for (let p = 0; p * limit < totalCount; p++) {
          const result: FilePage = await Promise.race([
            GetFiles({ ...activeFilter, folder_path: this.folderBulkEditPath }, { field: 'indexed_at', order: 'desc' }, p, limit),
            new Promise<FilePage>((_, reject) =>
              setTimeout(() => reject(new Error('GetFiles timeout (5s)')), 5000)
            ),
          ])
          const fileArray: File[] = result && Array.isArray(result.files) ? result.files : []
          for (const f of fileArray) {
            allIds.push(f.id)
          }
        }
        return allIds
      }
      return this.selectedFiles.map(f => f.id)
    },
    /** Get total count for current selection or folder bulk edit scope */
    async getTotalSelectedCount(): Promise<number> {
      if (this.folderBulkEditPath) {
        const filtersStore = useFiltersStore()
        const activeFilter = filtersStore.asBackendFilter
        try {
          const result: FilePage = await Promise.race([
            GetFiles({ ...activeFilter, folder_path: this.folderBulkEditPath }, { field: 'indexed_at', order: 'desc' }, 0, 1),
            new Promise<FilePage>((_, reject) =>
              setTimeout(() => reject(new Error('GetFiles timeout (2s)')), 2000)
            ),
          ])
          return result?.total_count ?? 0
        } catch {
          return 0
        }
      }
      return this.selectedFiles.length
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
    async searchFiles(query: string, limit: number = 500) {
      this.isLoading = true
      try {
        const results = await SearchFiles(query, limit)
        // Go returns nil slice → null in JS when no results
        const fileArray: File[] = Array.isArray(results) ? results : []
        this.files = fileArray
        this.totalCount = fileArray.length
        this.page = 0
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
      const fileIDs = await this.getAllSelectedIds()
      if (fileIDs.length === 0 || tagIDs.length === 0) return
      await AddTagsToFiles(fileIDs, tagIDs)
    },

    /** Remove tags from all selected files in a single backend call */
    async batchRemoveTags(tagIDs: number[]) {
      const fileIDs = await this.getAllSelectedIds()
      if (fileIDs.length === 0 || tagIDs.length === 0) return
      await RemoveTagsFromFiles(fileIDs, tagIDs)
    },

    /** Set rating on all selected files */
    async batchSetRating(rating: number) {
      const fileIDs = await this.getAllSelectedIds()
      if (fileIDs.length === 0) return
      await SetRatingForFiles(fileIDs, rating)
      // Optimistic update for loaded files
      for (const file of this.selectedFiles) {
        file.rating = rating
      }
    },

    /** Set favorite flag on all selected files */
    async batchSetFavorite(isFavorite: boolean) {
      const fileIDs = await this.getAllSelectedIds()
      if (fileIDs.length === 0) return
      await SetFavoriteForFiles(fileIDs, isFavorite ? 1 : 0)
      // Optimistic update for loaded files
      const favValue = isFavorite ? 1 : 0
      for (const file of this.selectedFiles) {
        file.is_favorite = favValue
      }
    },

    /** Delete a file from the vault index (DB only, not from disk) */
    async deleteFile(fileID: number) {
      await DeleteFile(fileID)
      this.removeFileLocally(fileID)
    },

    /** Open the original file with the default OS application */
    async openOriginalFile(fileID: number) {
      await OpenOriginalFile(fileID)
    },

    /** Open the parent folder of the file in the system file explorer */
    async openFileFolder(fileID: number) {
      await OpenFileFolder(fileID)
    },

    /** Delete the original file from disk, remove thumbnail, and remove from vault index */
    async deleteOriginalFile(fileID: number) {
      await DeleteOriginalFile(fileID)
      this.removeFileLocally(fileID)
    },

    /** Remove a file from local state (files list + selection) */
    removeFileLocally(fileID: number) {
      const index = this.files.findIndex(f => f.id === fileID)
      if (index >= 0) {
        this.files.splice(index, 1)
      }
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
