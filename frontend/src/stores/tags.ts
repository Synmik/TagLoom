import { defineStore } from 'pinia'
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
    async loadTagCounts() {
      this.tagCounts = await GetAllTagFileCounts()
    },
    async createTag(tag: TagCreate) {
      const newTag = await CreateTag(tag)
      this.tags.push(newTag)
      return newTag
    },
    async updateTag(tag: TagUpdate) {
      await UpdateTag(tag)
      const index = this.tags.findIndex(t => t.id === tag.id)
      if (index >= 0) {
        Object.assign(this.tags[index], tag)
      }
    },
    async deleteTag(id: number) {
      await DeleteTag(id)
      this.tags = this.tags.filter(t => t.id !== id)
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
