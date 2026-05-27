import { defineStore } from 'pinia'
import type { Tag, TagCreate, TagUpdate } from '../types/tag'

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
        // @ts-ignore
        this.tags = await window.go.main.app.GetTags('')
      } finally {
        this.isLoading = false
      }
    },
    async createTag(tag: TagCreate) {
      // @ts-ignore
      const newTag = await window.go.main.app.CreateTag(tag)
      this.tags.push(newTag)
      return newTag
    },
    async updateTag(tag: TagUpdate) {
      // @ts-ignore
      await window.go.main.app.UpdateTag(tag)
      const index = this.tags.findIndex(t => t.id === tag.id)
      if (index >= 0) {
        Object.assign(this.tags[index], tag)
      }
    },
    async deleteTag(id: number) {
      // @ts-ignore
      await window.go.main.app.DeleteTag(id)
      this.tags = this.tags.filter(t => t.id !== id)
    },
    async addTagToFile(fileID: number, tagID: number) {
      // @ts-ignore
      await window.go.main.app.AddTagToFile(fileID, tagID)
    },
    async removeTagFromFile(fileID: number, tagID: number) {
      // @ts-ignore
      await window.go.main.app.RemoveTagFromFile(fileID, tagID)
    },
    async getFileTags(fileID: number): Promise<Tag[]> {
      // @ts-ignore
      return await window.go.main.app.GetFileTags(fileID)
    },
  },
})
