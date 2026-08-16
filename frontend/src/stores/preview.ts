import { defineStore, storeToRefs } from "pinia";
import { watch } from "vue";
import type { File, FileMetadata } from "../types/file";
import type { Tag } from "../types/tag";
import { GetFileMetadata, GetFileTags, UpdateFile } from "../api/backend";
import { useFilesStore } from "./files";
import { logger } from "../utils/logger";

export const usePreviewStore = defineStore("preview", {
  state: () => ({
    currentFile: null as File | null,
    metadata: null as FileMetadata | null,
    tags: [] as Tag[],
    isLoading: false,
    previewModalOpen: false,
    _loadSeq: 0, // monotonic counter to discard stale loadFileDetails responses
    // In-flight edits per file ID — merged into later full-row payloads so
    // rapid consecutive saves (e.g. name edit + favorite toggle) don't
    // clobber each other with stale values.
    _pendingEdits: {} as Record<number, Record<string, any>>,
  }),
  actions: {
    async setFile(file: File | null) {
      // Skip if the same file is already loaded — prevents duplicate calls
      // from both handleClick and the _syncSelection watcher racing
      if (file && this.currentFile && this.currentFile.id === file.id) {
        return;
      }
      this.currentFile = file;
      this.metadata = null;
      this.tags = [];
      if (file) {
        await this.loadFileDetails(file.id);
      }
    },

    /** Sync with filesStore selection — auto-loads preview when single file is selected */
    _syncSelection() {
      const filesStore = useFilesStore();
      const { selectedFiles } = storeToRefs(filesStore);

      watch(selectedFiles, (files) => {
        if (files.length === 1) {
          this.setFile(files[0]);
        } else if (files.length === 0) {
          // Don't clear — keep last file visible until user explicitly deselects
        } else {
          // Multi-select: keep showing the last single-selected file's details
        }
      });
    },
    async loadFileDetails(fileID: number) {
      this.isLoading = true;
      const seq = ++this._loadSeq;
      // Fetch metadata and tags independently so one failure doesn't block the other
      const metadataPromise = GetFileMetadata(fileID).catch((e) => {
        logger.warn("preview.metadata", `metadata fetch failed for ${fileID}:`, e);
        return null;
      });
      const tagsPromise = GetFileTags(fileID).catch((e) => {
        logger.warn("preview.tags", `tags fetch failed for ${fileID}:`, e);
        return [];
      });
      const [metadata, tags] = await Promise.all([metadataPromise, tagsPromise]);
      // Only apply if no newer load has started
      if (seq === this._loadSeq) {
        this.metadata = metadata;
        this.tags = Array.isArray(tags) ? tags : [];
        this.isLoading = false;
      }
    },
    // Helper: build a full FileUpdate with defaults, then override the field
    _updateField(field: string, value: any) {
      if (!this.currentFile) return Promise.resolve();
      return this.updateFieldFor(this.currentFile, field, value);
    },

    /**
     * Save one field value against a specific file (not necessarily the
     * current one). Used to flush pending edits when the selection changes.
     */
    updateFieldFor(target: File, field: string, value: any) {
      // Track this edit so concurrent saves on the same file include it
      const pending = this._pendingEdits[target.id] ?? {};
      pending[field] = value;
      this._pendingEdits[target.id] = pending;

      // Coerce nullable string fields (name, notes, link) to empty string
      // so Go backend receives a valid string instead of undefined
      return UpdateFile({
        id: target.id,
        name: field === "name" ? value : (pending.name ?? target.name ?? ""),
        notes: field === "notes" ? value : (pending.notes ?? target.notes ?? ""),
        link: field === "link" ? value : (pending.link ?? target.link ?? ""),
        rating: pending.rating ?? target.rating,
        is_favorite: pending.is_favorite ?? target.is_favorite,
      })
        .then(() => {
          // Patch in-memory state by ID — a late response must only touch
          // the file it targeted, never whichever file is selected now.
          const patch = (f: File | null | undefined) => {
            if (f && f.id === target.id) (f as any)[field] = value;
          };
          patch(this.currentFile);
          const filesStore = useFilesStore();
          patch(filesStore.files.find((x) => x.id === target.id));
        })
        .finally(() => {
          delete this._pendingEdits[target.id];
        });
    },
    async updateName(value: string) {
      return this._updateField("name", value);
    },
    async updateNotes(value: string) {
      return this._updateField("notes", value);
    },
    async updateLink(value: string) {
      return this._updateField("link", value);
    },
    async setRating(rating: number) {
      return this._updateField("rating", rating);
    },
    async toggleFavorite() {
      if (!this.currentFile) return;
      const newFav = this.currentFile.is_favorite === 1 ? 0 : 1;
      return this._updateField("is_favorite", newFav);
    },

    openFullPreview() {
      this.previewModalOpen = true;
    },
    closeFullPreview() {
      this.previewModalOpen = false;
    },
  },
});
