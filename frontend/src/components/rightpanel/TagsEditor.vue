<template>
  <section class="tags-section">
    <label class="field-label">Tags</label>
    <div class="tags-container">
      <TagChip
        v-for="tag in previewStore.tags"
        :key="tag.id"
        :tag="tag"
        @remove="removeTag"
      />
      <button class="add-tag-btn" @click="showPicker = true">+</button>
    </div>
    <div v-if="showPicker" class="tag-picker">
      <input v-model="searchQuery" placeholder="Search or create tag…" class="picker-input" />
      <div class="picker-results">
        <div
          v-for="tag in filteredTags"
          :key="tag.id"
          class="picker-item"
          @click="addTag(tag.id)"
        >
          <span class="color-dot" :style="{ background: tag.color || '#666' }"></span>
          {{ tag.name }}
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import TagChip from '../common/TagChip.vue'
import { usePreviewStore } from '../../stores/preview'
import { useTagsStore } from '../../stores/tags'

const previewStore = usePreviewStore()
const tagsStore = useTagsStore()
const showPicker = ref(false)
const searchQuery = ref('')

const filteredTags = computed(() => {
  const q = searchQuery.value.toLowerCase()
  return tagsStore.tags.filter(t => t.name.toLowerCase().includes(q))
})

const addTag = (tagId: number) => {
  if (previewStore.currentFile) {
    tagsStore.addTagToFile(previewStore.currentFile.id, tagId)
    previewStore.loadFileDetails(previewStore.currentFile.id)
  }
  showPicker.value = false
  searchQuery.value = ''
}

const removeTag = (tagId: number) => {
  if (previewStore.currentFile) {
    tagsStore.removeTagFromFile(previewStore.currentFile.id, tagId)
    previewStore.loadFileDetails(previewStore.currentFile.id)
  }
}
</script>

<style scoped>
.tags-section { display: flex; flex-direction: column; gap: 4px; }
.field-label { color: #888; font-size: 11px; text-transform: uppercase; }
.tags-container { display: flex; flex-wrap: wrap; gap: 4px; }
.add-tag-btn {
  background: #2a2a2a; border: 1px dashed #444; color: #888;
  border-radius: 4px; width: 24px; height: 24px; cursor: pointer;
}
.tag-picker { margin-top: 6px; background: #2a2a2a; border-radius: 4px; padding: 6px; }
.picker-input {
  width: 100%; background: #1a1a1a; border: 1px solid #444; color: #fff;
  border-radius: 3px; padding: 4px 6px; font-size: 12px; outline: none; box-sizing: border-box;
}
.picker-results { max-height: 120px; overflow-y: auto; margin-top: 4px; }
.picker-item {
  display: flex; align-items: center; gap: 6px; padding: 3px 4px;
  cursor: pointer; border-radius: 3px; font-size: 12px; color: #ccc;
}
.picker-item:hover { background: #3a3a3a; }
.color-dot { width: 8px; height: 8px; border-radius: 50%; }
</style>
