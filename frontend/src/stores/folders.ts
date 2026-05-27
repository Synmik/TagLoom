import { defineStore } from 'pinia'
import type { FolderNode } from '../types/vault'

export const useFoldersStore = defineStore('folders', {
  state: () => ({
    tree: [] as FolderNode[],
    expandedPaths: [] as string[],
    selectedPath: '' as string,
    isLoading: false,
  }),
  actions: {
    async loadTree() {
      this.isLoading = true
      try {
        // @ts-ignore
        const vault = await window.go.main.app.GetCurrentVault()
        if (vault) {
          // @ts-ignore
          const root = await window.go.main.app.GetFolderTree(vault.path)
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
