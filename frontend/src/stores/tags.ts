import { defineStore, storeToRefs } from "pinia";
import { watch } from "vue";
import type { Tag, TagCreate, TagUpdate } from "../types/tag";
import {
  GetTags,
  CreateTag,
  UpdateTag,
  DeleteTag,
  GetAllTagFileCounts,
  GetTagAliases,
  AddTagToFile,
  RemoveTagFromFile,
  GetFileTags,
} from "../api/backend";
import { useVaultStore } from "./vault";

export const useTagsStore = defineStore("tags", {
  state: () => ({
    tags: [] as Tag[],
    categories: [] as string[],
    tagCounts: {} as Record<number, number>,
    isLoading: false,
  }),
  getters: {
    /** Get all descendant tag IDs (children, grandchildren, etc.) — does NOT include the tag itself */
    getAllDescendantIds:
      () =>
      (tagId: number): number[] => {
        const allIds: number[] = [];
        const children = useTagsStore().tags.filter((t) => t.parent_id === tagId);
        for (const child of children) {
          allIds.push(child.id);
          // Recursively collect grandchildren
          allIds.push(...useTagsStore().getAllDescendantIds(child.id));
        }
        return allIds;
      },
    /** Get aggregate file count for a tag (direct count + all descendant counts) */
    getAggregateCount:
      () =>
      (tagId: number): number => {
        const counts = useTagsStore().tagCounts;
        let total = counts[tagId] ?? 0;
        const children = useTagsStore().tags.filter((t) => t.parent_id === tagId);
        for (const child of children) {
          total += useTagsStore().getAggregateCount(child.id);
        }
        return total;
      },
    /** Check if a tag has children */
    hasChildren:
      () =>
      (tagId: number): boolean => {
        return useTagsStore().tags.some((t) => t.parent_id === tagId);
      },
  },
  actions: {
    async loadTags() {
      this.isLoading = true;
      try {
        const result = await GetTags("");
        this.tags = Array.isArray(result) ? result : [];
        await this.loadTagCounts();
      } finally {
        this.isLoading = false;
      }
    },

    /** Reload tags whenever the vault changes */
    _watchVault() {
      const vaultStore = useVaultStore();
      const { currentVault } = storeToRefs(vaultStore);

      watch(currentVault, async (vault) => {
        if (vault) {
          await this.loadTags();
        } else {
          this.tags = [];
          this.tagCounts = {};
        }
      });
    },
    async loadTagCounts() {
      this.tagCounts = await GetAllTagFileCounts();
    },
    async createTag(tag: TagCreate) {
      await CreateTag(tag);
      await this.loadTags();
    },
    async updateTag(tag: TagUpdate) {
      await UpdateTag(tag);
      await this.loadTags();
    },
    async deleteTag(id: number) {
      await DeleteTag(id);
      await this.loadTags();
      // Remove deleted tag from active filters so the gallery doesn't query a stale tag
      const { useFiltersStore } = await import("./filters");
      const { useFilesStore } = await import("./files");
      const filtersStore = useFiltersStore();
      const idx = filtersStore.activeFilters.tagGroups.findIndex((g) => g.includes(id));
      if (idx >= 0) {
        filtersStore.activeFilters.tagGroups.splice(idx, 1);
        await useFilesStore().reloadFiles();
      }
    },
    async addTagToFile(fileID: number, tagID: number) {
      await AddTagToFile(fileID, tagID);
      await this.loadTagCounts();
    },
    async removeTagFromFile(fileID: number, tagID: number) {
      await RemoveTagFromFile(fileID, tagID);
      await this.loadTagCounts();
    },
    async getFileTags(fileID: number): Promise<Tag[]> {
      return await GetFileTags(fileID);
    },
    async getTagAliases(tagID: number): Promise<string[]> {
      return await GetTagAliases(tagID);
    },
  },
});
