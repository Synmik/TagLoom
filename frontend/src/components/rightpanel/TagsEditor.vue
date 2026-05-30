<template>
  <section class="tags-section" ref="sectionRef">
    <label class="field-label">Tags</label>
    <div class="tags-container">
      <TagChip
        v-for="tag in fileTags"
        :key="tag.id"
        :tag="tag"
        @remove="removeTag"
      />
      <button class="add-tag-btn" @click="openPicker">+</button>
    </div>
    <div v-if="showPicker" class="tag-picker">
      <input
        ref="inputRef"
        v-model="searchQuery"
        placeholder="Search or create tag…"
        class="picker-input"
        @keydown.enter.prevent="handleEnter"
        @keydown.escape="closePicker"
      />
      <div class="picker-results">
        <div
          v-if="canCreate && !filteredTags.length"
          class="picker-item create-item"
          @click="createTag"
        >
          <span class="create-icon">+</span>
          Create "<strong>{{ searchQuery }}</strong>"
        </div>
        <template v-else>
          <div
            v-for="tag in filteredTags"
            :key="tag.id"
            class="picker-item"
            :class="{ disabled: alreadyAttached(tag.id) }"
            @click="addTag(tag.id)"
          >
            <span class="color-dot" :style="{ background: tag.color || '#666' }"></span>
            {{ tag.name }}
            <span v-if="alreadyAttached(tag.id)" class="attached-badge">✓</span>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import TagChip from '../common/TagChip.vue'
import { usePreviewStore } from '../../stores/preview'
import { useTagsStore } from '../../stores/tags'

const previewStore = usePreviewStore()
const tagsStore = useTagsStore()
const showPicker = ref(false)
const searchQuery = ref('')
const inputRef = ref<HTMLInputElement | null>(null)
const sectionRef = ref<HTMLElement | null>(null)

// Ensure tags are loaded when picker opens
watch(() => showPicker.value, async (open) => {
  if (open) {
    await tagsStore.loadTags()
    await nextTick()
    inputRef.value?.focus()
  }
})

const filteredTags = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return tagsStore.tags
  return tagsStore.tags.filter(t => t.name.toLowerCase().includes(q))
})

const canCreate = computed(() => {
  const q = searchQuery.value.trim()
  return q.length > 0 && !filteredTags.value.length
})

const fileTags = computed(() => previewStore.tags || [])

const alreadyAttached = (tagId: number) => {
  return fileTags.value.some(t => t.id === tagId)
}

const openPicker = () => {
  searchQuery.value = ''
  showPicker.value = true
}

const closePicker = () => {
  showPicker.value = false
  searchQuery.value = ''
}

const handleEnter = () => {
  if (canCreate.value) {
    createTag()
  } else if (filteredTags.value.length) {
    const first = filteredTags.value.find(t => !alreadyAttached(t.id)) || filteredTags.value[0]
    addTag(first.id)
  }
}

const addTag = async (tagId: number) => {
  if (!previewStore.currentFile) return
  // Skip if already attached
  if (fileTags.value.some(t => t.id === tagId)) {
    closePicker()
    return
  }
  await tagsStore.addTagToFile(previewStore.currentFile.id, tagId)
  await previewStore.loadFileDetails(previewStore.currentFile.id)
  closePicker()
}

const removeTag = async (tagId: number) => {
  if (previewStore.currentFile) {
    await tagsStore.removeTagFromFile(previewStore.currentFile.id, tagId)
    await previewStore.loadFileDetails(previewStore.currentFile.id)
  }
}

const createTag = async () => {
  const name = searchQuery.value.trim()
  if (!name) return
  await tagsStore.createTag({
    name,
    color: '',
    is_category: 0,
    sort_order: 0,
    aliases: '',
  })
  // After creating (or finding existing case-insensitive match), attach to current file
  // Use case-insensitive lookup since the backend may return an existing tag
  const newTag = tagsStore.tags.find(t => t.name.toLowerCase() === name.toLowerCase())
  if (newTag && previewStore.currentFile) {
    await tagsStore.addTagToFile(previewStore.currentFile.id, newTag.id)
    await previewStore.loadFileDetails(previewStore.currentFile.id)
  }
  closePicker()
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
.picker-item.disabled { opacity: 0.4; cursor: default; }
.picker-item.disabled:hover { background: transparent; }
.create-item { color: #5b8af5; font-style: italic; }
.create-icon { color: #5b8af5; font-weight: bold; }
.color-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.attached-badge { margin-left: auto; color: #5b8af5; font-size: 11px; }
</style>
