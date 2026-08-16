import { defineStore } from "pinia";
import type { FileFilter } from "../types/file";

export interface FilterState {
  folderPath: string;
  tagGroups: number[][]; // Each group = [tagId, ...descendants]; between groups = AND
  fileFormats: string[];
  minRating: number;
  favoritesOnly: boolean;
  untaggedOnly: boolean;
  searchQuery: string;
}

export const useFiltersStore = defineStore("filters", {
  state: () => ({
    activeFilters: {
      folderPath: "",
      tagGroups: [] as number[][],
      fileFormats: [] as string[],
      minRating: 0,
      favoritesOnly: false,
      untaggedOnly: false,
      searchQuery: "",
    } as FilterState,
  }),
  getters: {
    hasActiveFilters: (state: any) => {
      const f = state.activeFilters;
      return (
        f.folderPath !== "" ||
        f.tagGroups.length > 0 ||
        f.fileFormats.length > 0 ||
        f.minRating > 0 ||
        f.favoritesOnly ||
        f.untaggedOnly ||
        f.searchQuery.trim() !== ""
      );
    },
    hasActiveSearch: (state: any) => {
      return state.activeFilters.searchQuery.trim() !== "";
    },
    /** Returns a snake-case FileFilter compatible with the Go backend */
    asBackendFilter: (state: any): FileFilter => {
      const f = state.activeFilters;
      return {
        folder_path: f.folderPath,
        tag_groups: f.tagGroups,
        file_formats: f.fileFormats,
        min_rating: f.minRating,
        favorites_only: f.favoritesOnly,
        untagged_only: f.untaggedOnly,
      };
    },
  },
  actions: {
    setFolderFilter(path: string) {
      this.activeFilters.folderPath = path;
    },
    setTagFilter(groups: number[][]) {
      this.activeFilters.tagGroups = groups;
    },
    /**
     * Toggle a tag filter.
     * @param tagId - The primary tag ID (used for selection matching)
     * @param groupIds - Full group of IDs (tag + descendants) to query
     * @param accumulate - If true, add as a new AND group instead of replacing
     */
    toggleTagFilter(tagId: number, groupIds?: number[], accumulate = false) {
      const idsToUse = groupIds ?? [tagId];

      // Find if this tag's group is already in the selection
      const existingIndex = this.activeFilters.tagGroups.findIndex(
        (group) => group.length > 0 && group.includes(tagId),
      );

      if (existingIndex >= 0) {
        // This tag is already selected — remove its group
        this.activeFilters.tagGroups.splice(existingIndex, 1);
      } else if (accumulate) {
        // Add as a new AND group
        this.activeFilters.tagGroups.push(idsToUse);
      } else {
        // Replace all with just this group
        this.activeFilters.tagGroups = [idsToUse];
      }
    },
    setFormatFilter(formats: string[]) {
      this.activeFilters.fileFormats = formats;
    },
    setRatingFilter(minRating: number) {
      this.activeFilters.minRating = minRating;
    },
    setFavoritesFilter(favoritesOnly: boolean) {
      this.activeFilters.favoritesOnly = favoritesOnly;
    },
    setUntaggedFilter(untaggedOnly: boolean) {
      this.activeFilters.untaggedOnly = untaggedOnly;
    },
    clearFilters() {
      this.activeFilters.folderPath = "";
      this.activeFilters.tagGroups = [];
      this.activeFilters.fileFormats = [];
      this.activeFilters.minRating = 0;
      this.activeFilters.favoritesOnly = false;
      this.activeFilters.untaggedOnly = false;
      this.activeFilters.searchQuery = "";
    },
  },
});
