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
    leftPanelSplit: 50,
    showBatchEdit: false,
    showVaultSettings: false,
    showAppSettings: false,
    showTagManager: false,
    showNewVault: false,
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
    openBatchEdit() {
      this.showBatchEdit = true
    },
    closeBatchEdit() {
      this.showBatchEdit = false
    },
    openVaultSettings() {
      this.showVaultSettings = true
    },
    closeVaultSettings() {
      this.showVaultSettings = false
    },
    openAppSettings() {
      this.showAppSettings = true
    },
    closeAppSettings() {
      this.showAppSettings = false
    },
    openTagManager() {
      this.showTagManager = true
    },
    closeTagManager() {
      this.showTagManager = false
    },
    setLeftPanelWidth(width: number) {
      this.leftPanelWidth = Math.max(180, Math.min(500, width))
    },
    setRightPanelWidth(width: number) {
      this.rightPanelWidth = Math.max(200, Math.min(600, width))
    },
    openNewVault() {
      this.showNewVault = true
    },
    closeNewVault() {
      this.showNewVault = false
    },
    setLeftPanelSplit(percent: number) {
      this.leftPanelSplit = Math.max(15, Math.min(85, percent))
    },
  },
})
