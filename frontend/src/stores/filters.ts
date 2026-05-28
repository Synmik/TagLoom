import { defineStore } from 'pinia'
import type { FileFilter } from '../types/file'

export interface FilterState {
  folderPath: string
  tagIds: number[]
  fileFormats: string[]
  minRating: number
  favoritesOnly: boolean
}

export const useFiltersStore = defineStore('filters', {
  state: () => ({
    activeFilters: {
      folderPath: '',
      tagIds: [] as number[],
      fileFormats: [] as string[],
      minRating: 0,
      favoritesOnly: false,
    } as FilterState,
  }),
  getters: {
    hasActiveFilters: (state: any) => {
      const f = state.activeFilters
      return f.folderPath !== '' ||
        f.tagIds.length > 0 ||
        f.fileFormats.length > 0 ||
        f.minRating > 0 ||
        f.favoritesOnly
    },
    /** Returns a snake-case FileFilter compatible with the Go backend */
    asBackendFilter: (state: any): FileFilter => {
      const f = state.activeFilters
      return {
        folder_path: f.folderPath,
        tag_ids: f.tagIds,
        file_formats: f.fileFormats,
        min_rating: f.minRating,
        favorites_only: f.favoritesOnly,
      }
    },
  },
  actions: {
    setFolderFilter(path: string) {
      this.activeFilters.folderPath = path
    },
    setTagFilter(tagIds: number[]) {
      this.activeFilters.tagIds = tagIds
    },
    toggleTagFilter(tagId: number) {
      const index = this.activeFilters.tagIds.indexOf(tagId)
      if (index >= 0) {
        this.activeFilters.tagIds.splice(index, 1)
      } else {
        this.activeFilters.tagIds.push(tagId)
      }
    },
    setFormatFilter(formats: string[]) {
      this.activeFilters.fileFormats = formats
    },
    setRatingFilter(minRating: number) {
      this.activeFilters.minRating = minRating
    },
    setFavoritesFilter(favoritesOnly: boolean) {
      this.activeFilters.favoritesOnly = favoritesOnly
    },
    clearFilters() {
      this.activeFilters = {
        folderPath: '',
        tagIds: [],
        fileFormats: [],
        minRating: 0,
        favoritesOnly: false,
      }
    },
  },
})
