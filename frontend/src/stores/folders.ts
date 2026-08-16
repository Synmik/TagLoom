import { defineStore, storeToRefs } from "pinia";
import { watch } from "vue";
import type { FolderNode } from "../types/vault";
import { GetCurrentVault, GetFolderTree } from "../api/backend";
import { useVaultStore } from "./vault";

export const useFoldersStore = defineStore("folders", {
  state: () => ({
    tree: [] as FolderNode[],
    expandedPaths: [] as string[],
    selectedPath: "",
    isLoading: false,
  }),
  actions: {
    async loadTree() {
      this.isLoading = true;
      try {
        const vault = await GetCurrentVault();
        if (vault) {
          const root = await GetFolderTree(vault.path);
          this.tree = root ? [root] : [];
          // Expand root folder by default
          if (root) {
            const idx = this.expandedPaths.indexOf(root.path);
            if (idx === -1) {
              this.expandedPaths.push(root.path);
            }
          }
        } else {
          this.tree = [];
        }
      } finally {
        this.isLoading = false;
      }
    },

    /** Reload folder tree whenever the vault changes — called from App.vue onMounted */
    _watchVault() {
      const vaultStore = useVaultStore();
      const { currentVault } = storeToRefs(vaultStore);

      watch(currentVault, async (vault) => {
        if (vault) {
          await this.loadTree();
        } else {
          this.tree = [];
          this.expandedPaths = [];
          this.selectedPath = "";
        }
      });
    },
    toggleFolder(path: string) {
      const index = this.expandedPaths.indexOf(path);
      if (index >= 0) {
        this.expandedPaths.splice(index, 1);
      } else {
        this.expandedPaths.push(path);
      }
    },
    selectFolder(path: string) {
      this.selectedPath = path;
    },
    clearSelection() {
      this.selectedPath = "";
    },
  },
});
