import { defineStore } from 'pinia'
import type { Tag, TagCreate, TagUpdate } from '../types/tag'
import {
  GetTags,
  CreateTag,
  UpdateTag,
  DeleteTag,
  AddTagToFile,
  RemoveTagFromFile,
  GetFileTags,
} from '../api/backend'

export const useTagsStore = defineStore('tags', {
  state: () => ({
    tags: [] as Tag[],
    categories: [] as string[],
    isLoading: false,
  }),
  actions: {
    async loadTags() {
      this.isLoading = true
      try {
        this.tags = await GetTags('')
      } finally {
        this.isLoading = false
      }
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
    },
    async removeTagFromFile(fileID: number, tagID: number) {
      await RemoveTagFromFile(fileID, tagID)
    },
    async getFileTags(fileID: number): Promise<Tag[]> {
      return await GetFileTags(fileID)
    },
  },
})
