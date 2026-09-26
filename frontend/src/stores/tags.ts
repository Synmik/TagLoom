import { defineStore, storeToRefs } from "pinia";
import { watch } from "vue";
import type {
  ChildStrategy,
  Tag,
  TagAlias,
  TagCreate,
  TagMergeResult,
  TagUpdate,
  TagUsage,
} from "../types/tag";
import {
  GetTags,
  CreateTag,
  UpdateTag,
  GetAllTagFileCounts,
  GetTagAliases,
  GetAllTagAliases,
  GetTagUsage,
  MergeTags,
  MoveTag,
  DeleteTagsSafely,
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
    /**
     * Delete a single tag. Its children move up a level instead of being left
     * behind: PRAGMA foreign_keys is off, so removing a parent outright would
     * strand them out of sight. Callers that ask the user what to do about
     * children use deleteTagsSafely.
     */
    async deleteTag(id: number) {
      await this.deleteTagsSafely([id], "promote");
    },

    /**
     * Delete tags without orphaning children. `strategy` decides what happens to
     * the children; "block" makes the backend refuse when any child exists.
     */
    async deleteTagsSafely(tagIDs: number[], strategy: ChildStrategy) {
      if (tagIDs.length === 0) return;
      await DeleteTagsSafely(tagIDs, strategy);
      await this.loadTags();
      await this.dropFromActiveFilters(tagIDs);
      await this.refreshPreviewTags();
    },

    /** Fold source tags into target and report what changed. */
    async mergeTags(sourceIDs: number[], targetID: number): Promise<TagMergeResult | null> {
      const result = await MergeTags(sourceIDs, targetID);
      await this.loadTags();
      await this.dropFromActiveFilters(sourceIDs);
      await this.refreshPreviewTags();
      return result ?? null;
    },

    /**
     * Re-parent a tag and/or position it before `beforeID` in its new level.
     * `beforeID = 0` appends to the end of the level.
     */
    async moveTag(tagID: number, newParentID: number | null, beforeID = 0) {
      await MoveTag(tagID, newParentID, beforeID);
      await this.loadTags();
    },

    /** How the given tags are used, keyed by tag id: one query for the lot. */
    async fetchTagUsage(tagIDs: number[]): Promise<Record<number, TagUsage>> {
      if (tagIDs.length === 0) return {};
      const result = await GetTagUsage(tagIDs);
      const usage: Record<number, TagUsage> = {};
      for (const row of result || []) {
        usage[row.tag_id] = row;
      }
      return usage;
    },

    /** Aliases of every tag — the manager searches by alias, so it needs them all. */
    async loadAllTagAliases(): Promise<TagAlias[]> {
      const result = await GetAllTagAliases();
      return Array.isArray(result) ? result : [];
    },

    /**
     * Remove deleted tags from active filters so the gallery never queries a tag
     * that no longer exists, and reload once if anything was dropped.
     */
    async dropFromActiveFilters(tagIDs: number[]) {
      const { useFiltersStore } = await import("./filters");
      const { useFilesStore } = await import("./files");
      const filtersStore = useFiltersStore();
      const removed = new Set(tagIDs);
      const kept = filtersStore.activeFilters.tagGroups.filter(
        (g) => !g.some((id) => removed.has(id)),
      );
      if (kept.length !== filtersStore.activeFilters.tagGroups.length) {
        filtersStore.activeFilters.tagGroups = kept;
        await useFilesStore().reloadFiles();
      }
    },
    /**
     * The right panel lists the open file's tags; a merge or delete changes them
     * without the panel knowing. Refetch when a file is open.
     */
    async refreshPreviewTags() {
      const { usePreviewStore } = await import("./preview");
      const previewStore = usePreviewStore();
      const file = previewStore.currentFile;
      if (file) await previewStore.loadFileDetails(file.id);
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
