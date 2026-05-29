import { defineStore, storeToRefs } from 'pinia'
import { watch } from 'vue'
import type { Tag, TagCreate, TagUpdate } from '../types/tag'
import {
  GetTags,
  CreateTag,
  UpdateTag,
  DeleteTag,
  GetAllTagFileCounts,
  AddTagToFile,
  RemoveTagFromFile,
  GetFileTags,
} from '../api/backend'
import { useVaultStore } from './vault'

export const useTagsStore = defineStore('tags', {
  state: () => ({
    tags: [] as Tag[],
    categories: [] as string[],
    tagCounts: {} as Record<number, number>,
    isLoading: false,
  }),
  actions: {
    async loadTags() {
      this.isLoading = true
      try {
        this.tags = await GetTags('')
        await this.loadTagCounts()
      } finally {
        this.isLoading = false
      }
    },

    /** Reload tags whenever the vault changes */
    _watchVault() {
      const vaultStore = useVaultStore()
      const { currentVault } = storeToRefs(vaultStore)

      watch(currentVault, async (vault) => {
        if (vault) {
          await this.loadTags()
        } else {
          this.tags = []
          this.tagCounts = {}
        }
      })
    },
    async loadTagCounts() {
      this.tagCounts = await GetAllTagFileCounts()
    },
    async createTag(tag: TagCreate) {
      await CreateTag(tag)
      await this.loadTags()
    },
    async updateTag(tag: TagUpdate) {
      await UpdateTag(tag)
      await this.loadTags()
    },
    async deleteTag(id: number) {
      await DeleteTag(id)
      await this.loadTags()
    },
    async addTagToFile(fileID: number, tagID: number) {
      await AddTagToFile(fileID, tagID)
      await this.loadTagCounts()
    },
    async removeTagFromFile(fileID: number, tagID: number) {
      await RemoveTagFromFile(fileID, tagID)
      await this.loadTagCounts()
    },
    async getFileTags(fileID: number): Promise<Tag[]> {
      return await GetFileTags(fileID)
    },
  },
})
