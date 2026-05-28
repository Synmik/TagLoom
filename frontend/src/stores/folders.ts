import { defineStore } from 'pinia'
import type { FolderNode } from '../types/vault'
import { GetCurrentVault, GetFolderTree } from '../api/backend'

export const useFoldersStore = defineStore('folders', {
  state: () => ({
    tree: [] as FolderNode[],
    expandedPaths: [] as string[],
    selectedPath: '',
    isLoading: false,
  }),
  actions: {
    async loadTree() {
      this.isLoading = true
      try {
        const vault = await GetCurrentVault()
        if (vault) {
          const root = await GetFolderTree(vault.path)
          this.tree = root ? [root] : []
        }
      } finally {
        this.isLoading = false
      }
    },
    toggleFolder(path: string) {
      const index = this.expandedPaths.indexOf(path)
      if (index >= 0) {
        this.expandedPaths.splice(index, 1)
      } else {
        this.expandedPaths.push(path)
      }
    },
    selectFolder(path: string) {
      this.selectedPath = path
    },
    clearSelection() {
      this.selectedPath = ''
    },
  },
})
