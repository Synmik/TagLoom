import { defineStore } from 'pinia'

export type ViewMode = 'grid' | 'list'
export type GridSize = 'small' | 'medium' | 'large'
export type SortField = 'indexed_at' | 'filename' | 'name' | 'file_size' | 'rating' | 'date_modified'

export const useUIStore = defineStore('ui', {
  state: () => ({
    viewMode: 'grid' as ViewMode,
    gridSize: 'medium' as GridSize,
    sortBy: 'indexed_at' as SortField,
    sortOrder: 'desc' as 'asc' | 'desc',
    leftPanelWidth: 280,
    rightPanelWidth: 320,
  }),
  getters: {
    gridPixelSize: (state) => {
      const sizes = { small: 128, medium: 192, large: 256 }
      return sizes[state.gridSize]
    },
  },
  actions: {
    setViewMode(mode: ViewMode) {
      this.viewMode = mode
    },
    toggleViewMode() {
      this.viewMode = this.viewMode === 'grid' ? 'list' : 'grid'
    },
    setGridSize(size: GridSize) {
      this.gridSize = size
    },
    cycleGridSize() {
      const sizes: GridSize[] = ['small', 'medium', 'large']
      const index = sizes.indexOf(this.gridSize)
      this.gridSize = sizes[(index + 1) % sizes.length]
    },
    setSort(field: SortField, order: 'asc' | 'desc' = 'desc') {
      this.sortBy = field
      this.sortOrder = order
    },
    toggleSortOrder() {
      this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc'
    },
  },
})
