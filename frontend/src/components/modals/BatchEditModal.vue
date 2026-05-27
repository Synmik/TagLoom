<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>Batch Edit ({{ filesStore.selectionCount }} files)</h3>
        <button class="close-btn" @click="$emit('close')">✕</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>Add Tags</label>
          <div class="tags-container">
            <TagChip v-for="tag in selectedTags" :key="tag.id" :tag="tag" @remove="removeSelectedTag" />
            <button class="add-tag-btn" @click="showPicker = true">+</button>
          </div>
        </div>

        <div class="form-group">
          <label>Set Rating</label>
          <StarRating :rating="rating" @change="onRatingChange" />
        </div>

        <div class="form-group checkbox">
          <label>
            <input type="checkbox" v-model="setFavorite" /> Set as Favorite
          </label>
        </div>

        <button class="save-btn" @click="apply">Apply to {{ filesStore.selectionCount }} files</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import TagChip from '../common/TagChip.vue'
import StarRating from '../common/StarRating.vue'
import { useFilesStore } from '../../stores/files'
import { useTagsStore } from '../../stores/tags'
import type { Tag } from '../../types/tag'

const filesStore = useFilesStore()
const tagsStore = useTagsStore()
defineEmits<{ close: [] }>()

const selectedTags = ref<Tag[]>([])
const rating = ref(0)
const setFavorite = ref(false)
const showPicker = ref(false)

const removeSelectedTag = (tagId: number) => {
  selectedTags.value = selectedTags.value.filter(t => t.id !== tagId)
}

const onRatingChange = (r: number) => { rating.value = r }

const apply = async () => {
  for (const file of filesStore.selectedFiles) {
    for (const tag of selectedTags.value) {
      await tagsStore.addTagToFile(file.id, tag.id)
    }
    if (rating.value > 0) {
      await filesStore.updateFile({ id: file.id, rating: rating.value })
    }
    if (setFavorite.value) {
      await filesStore.updateFile({ id: file.id, is_favorite: 1 })
    }
  }
  filesStore.clearSelection()
}
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.6);
  display: flex; align-items: center; justify-content: center; z-index: 100;
}
.modal { background: #1e1e1e; border-radius: 8px; width: 360px; border: 1px solid #333; }
.modal-header { display: flex; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid #333; }
.modal-header h3 { margin: 0; color: #fff; font-size: 14px; }
.close-btn { background: none; border: none; color: #888; cursor: pointer; font-size: 16px; }
.modal-body { padding: 16px; display: flex; flex-direction: column; gap: 12px; }
.form-group { display: flex; flex-direction: column; gap: 4px; }
.form-group label { color: #888; font-size: 11px; }
.tags-container { display: flex; flex-wrap: wrap; gap: 4px; }
.add-tag-btn { background: #2a2a2a; border: 1px dashed #444; color: #888; border-radius: 4px; width: 24px; height: 24px; cursor: pointer; }
.checkbox { flex-direction: row; align-items: center; gap: 6px; }
.save-btn { background: #5b8af5; color: #fff; border: none; border-radius: 4px; padding: 8px; cursor: pointer; margin-top: 4px; }
</style>
