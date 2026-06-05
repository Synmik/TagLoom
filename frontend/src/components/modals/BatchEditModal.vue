<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>Batch Edit ({{ totalCount }} files)</h3>
        <button class="close-btn" @click="$emit('close')"><X :size="16" /></button>
      </div>

      <div class="modal-body">
        <!-- Add Tags -->
        <div class="form-group">
          <label>Add Tags</label>
          <div class="tags-container">
            <TagChip v-for="tag in tagsToAdd" :key="tag.id" :tag="tag" @remove="removeTagToAdd" />
            <button class="add-tag-btn" @click="openAddPicker"><Plus :size="14" /></button>
          </div>
          <div v-if="pickerMode === 'add'" class="tag-picker">
            <input
              ref="inputRef"
              v-model="searchQuery"
              placeholder="Search or create tag…"
              class="picker-input"
              @keydown.enter.prevent="handleEnter"
              @keydown.escape="closePicker"
            />
            <div class="picker-results">
              <div v-if="canCreate && !filteredTags.length" class="picker-item create-item" @click="createTag">
                <PlusCircle :size="14" class="create-icon" /> Create "<strong>{{ searchQuery }}</strong>"
              </div>
              <template v-else>
                <div v-for="tag in filteredTags" :key="tag.id" class="picker-item" @click="pickTag(tag)">
                  <span class="color-dot" :style="{ background: tag.color || '#666' }"></span>
                  {{ tag.name }}
                </div>
              </template>
            </div>
          </div>
        </div>

        <!-- Remove Tags -->
        <div class="form-group">
          <label>Remove Tags</label>
          <div class="tags-container">
            <TagChip v-for="tag in tagsToRemove" :key="tag.id" :tag="tag" @remove="removeTagToRemove" />
            <button class="add-tag-btn" @click="openRemovePicker"><Plus :size="14" /></button>
          </div>
          <div v-if="pickerMode === 'remove'" class="tag-picker">
            <input
              v-model="searchQuery"
              placeholder="Search tag to remove…"
              class="picker-input"
              @keydown.enter.prevent="handleEnter"
              @keydown.escape="closePicker"
            />
            <div class="picker-results">
              <div v-for="tag in filteredTags" :key="tag.id" class="picker-item" @click="pickTag(tag)">
                <span class="color-dot" :style="{ background: tag.color || '#666' }"></span>
                {{ tag.name }}
              </div>
            </div>
          </div>
        </div>

        <!-- Rating -->
        <div class="form-group">
          <label>Set Rating</label>
          <div class="rating-row">
            <StarRating :rating="rating" @change="onRatingChange" />
            <button
              v-if="applyRating"
              class="clear-btn"
              @click="rating = 0; applyRating = false"
              title="Clear rating"
            >
              <X :size="14" />
            </button>
          </div>
        </div>

        <!-- Favorite -->
        <div class="form-group checkbox">
          <label>
            <input type="radio" v-model="favoriteAction" :value="'unset'" />
            Leave favorites unchanged
          </label>
          <label>
            <input type="radio" v-model="favoriteAction" :value="'set'" />
            Set as favorite
          </label>
          <label>
            <input type="radio" v-model="favoriteAction" :value="'unset_flag'" />
            Remove favorite
          </label>
        </div>

        <!-- Actions -->
        <div class="actions-row">
          <button class="save-btn" @click="apply" :disabled="nothingToApply">
            Apply to {{ totalCount }} files
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { shallowRef, ref, computed, watch, nextTick, onMounted } from 'vue'
import { X, Plus, PlusCircle } from '@lucide/vue'
import TagChip from '../common/TagChip.vue'
import StarRating from '../common/StarRating.vue'
import { useFilesStore } from '../../stores/files'
import { useTagsStore } from '../../stores/tags'
import { useToast } from '../../composables/useToast'
import type { Tag } from '../../types/tag'

const emit = defineEmits<{ close: [] }>()
const { success, error: toastError } = useToast()

const filesStore = useFilesStore()
const tagsStore = useTagsStore()

// Total count (handles both normal selection and folder bulk edit mode)
const totalCount = ref(filesStore.selectionCount)

onMounted(async () => {
  totalCount.value = await filesStore.getTotalSelectedCount()
})

// ── State ──
const tagsToAdd = ref<Tag[]>([])
const tagsToRemove = ref<Tag[]>([])
const rating = shallowRef(0)
const applyRating = shallowRef(false)
const favoriteAction = shallowRef<'unset' | 'set' | 'unset_flag'>('unset')

// ── Tag picker state ──
const pickerMode = shallowRef<'none' | 'add' | 'remove'>('none')
const searchQuery = ref('')
const inputRef = ref<HTMLInputElement | null>(null)

const filteredTags = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return tagsStore.tags
  return tagsStore.tags.filter(t => t.name.toLowerCase().includes(q))
})

const canCreate = computed(() => {
  const q = searchQuery.value.trim()
  return q.length > 0 && filteredTags.value.length === 0
})

const nothingToApply = computed(() => {
  return (
    tagsToAdd.value.length === 0 &&
    tagsToRemove.value.length === 0 &&
    !applyRating.value &&
    favoriteAction.value === 'unset'
  )
})

// ── Tag picker ──
const openAddPicker = async () => {
  await tagsStore.loadTags()
  pickerMode.value = 'add'
  searchQuery.value = ''
  await nextTick()
  inputRef.value?.focus()
}

const openRemovePicker = async () => {
  await tagsStore.loadTags()
  pickerMode.value = 'remove'
  searchQuery.value = ''
  await nextTick()
  inputRef.value?.focus()
}

const closePicker = () => {
  pickerMode.value = 'none'
  searchQuery.value = ''
}

// Close picker when mode changes
watch(pickerMode, (mode) => {
  if (mode === 'none') {
    searchQuery.value = ''
  }
})

const handleEnter = () => {
  if (canCreate.value) {
    createTag()
  } else if (filteredTags.value.length > 0) {
    pickTag(filteredTags.value[0])
  }
}

const pickTag = (tag: Tag) => {
  if (pickerMode.value === 'add') {
    if (!tagsToAdd.value.find(t => t.id === tag.id)) {
      tagsToAdd.value.push(tag)
    }
  } else if (pickerMode.value === 'remove') {
    if (!tagsToRemove.value.find(t => t.id === tag.id)) {
      tagsToRemove.value.push(tag)
    }
  }
  // Keep picker open for multi-select — close on Escape
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
  const newTag = tagsStore.tags.find(t => t.name.toLowerCase() === name.toLowerCase())
  if (newTag) {
    pickTag(newTag)
  }
  searchQuery.value = ''
}

// ── Tag chip removal ──
const removeTagToAdd = (tagId: number) => {
  tagsToAdd.value = tagsToAdd.value.filter(t => t.id !== tagId)
}

const removeTagToRemove = (tagId: number) => {
  tagsToRemove.value = tagsToRemove.value.filter(t => t.id !== tagId)
}

// ── Rating ──
const onRatingChange = (r: number) => {
  rating.value = r
  applyRating.value = r > 0
}

// ── Apply ──
const apply = async () => {
  if (nothingToApply.value) return

  try {
    if (tagsToAdd.value.length > 0) {
      const tagIDs = tagsToAdd.value.map(t => t.id)
      await filesStore.batchAddTags(tagIDs)
    }

    if (tagsToRemove.value.length > 0) {
      const tagIDs = tagsToRemove.value.map(t => t.id)
      await filesStore.batchRemoveTags(tagIDs)
    }

    if (applyRating.value) {
      await filesStore.batchSetRating(rating.value)
    }

    if (favoriteAction.value === 'set') {
      await filesStore.batchSetFavorite(true)
    } else if (favoriteAction.value === 'unset_flag') {
      await filesStore.batchSetFavorite(false)
    }

    filesStore.clearSelection()
    // Reload gallery to reflect optimistic updates
    await filesStore.reloadFiles()
    // Reload tags tree
    await tagsStore.loadTags()

    success(`Changes applied to ${totalCount.value} files`)
  } catch (e: any) {
    toastError('Batch edit failed: ' + (e.message || String(e)))
    return
  }

  closePicker()
  emit('close')
}
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.7);
  display: flex; align-items: center; justify-content: center; z-index: 100;
}
.modal { background: #111111; border-radius: 12px; width: 400px; border: 1px solid #222; box-shadow: 0 16px 48px rgba(0,0,0,0.6); }
.modal-header { display: flex; justify-content: space-between; padding: 14px 18px; border-bottom: 1px solid #1a1a1a; }
.modal-header h3 { margin: 0; color: #e8e8e8; font-size: 14px; font-weight: 600; font-family: 'Inter', sans-serif; }
.close-btn { background: none; border: none; color: #666; cursor: pointer; font-size: 16px; padding: 4px; border-radius: 4px; transition: color 0.15s, background 0.15s; }
.close-btn:hover { color: #e8e8e8; background: #1a1a1a; }
.modal-body { padding: 18px; display: flex; flex-direction: column; gap: 14px; }
.form-group { display: flex; flex-direction: column; gap: 6px; }
.form-group label { color: #666; font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
.tags-container { display: flex; flex-wrap: wrap; gap: 4px; }
.add-tag-btn {
  background: #1a1a1a; border: 1px dashed #333; color: #666;
  border-radius: 6px; width: 26px; height: 26px; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: all 0.15s;
}
.add-tag-btn:hover { border-color: #22c55e; color: #22c55e; background: #222; }
.rating-row { display: flex; align-items: center; gap: 8px; }
.clear-btn {
  background: none; border: none; color: #555; cursor: pointer; font-size: 14px; transition: color 0.15s;
}
.clear-btn:hover { color: #ef4444; }
.checkbox { flex-direction: column; align-items: flex-start; gap: 6px; }
.checkbox label {
  flex-direction: row; display: flex; align-items: center; gap: 8px;
  text-transform: none; font-size: 13px; color: #ccc;
}
.checkbox input[type="radio"] { accent-color: #22c55e; }
.actions-row { display: flex; gap: 8px; margin-top: 4px; padding-top: 12px; border-top: 1px solid #1a1a1a; }
.save-btn {
  flex: 1; background: #22c55e; color: #000; border: none; border-radius: 6px;
  padding: 9px 16px; cursor: pointer; font-size: 13px; font-weight: 500;
  font-family: 'Inter', sans-serif;
  transition: background 0.15s;
}
.save-btn:hover { background: #16a34a; }
.save-btn:disabled { opacity: 0.4; cursor: not-allowed; }

/* ── Tag picker ── */
.tag-picker {
  margin-top: 6px; background: #1a1a1a;
  border: 1px solid #2a2a2a; border-radius: 6px; padding: 6px;
}
.picker-input {
  width: 100%; background: #111; border: 1px solid #2a2a2a; color: #e8e8e8;
  border-radius: 5px; padding: 6px 8px; font-size: 12px;
  font-family: 'Inter', sans-serif;
  outline: none; box-sizing: border-box; transition: border-color 0.15s;
}
.picker-input:focus { border-color: #22c55e; }
.picker-input::placeholder { color: #444; }
.picker-results { max-height: 150px; overflow-y: auto; margin-top: 6px; }
.picker-item {
  display: flex; align-items: center; gap: 6px; padding: 4px 6px;
  cursor: pointer; border-radius: 4px; font-size: 12px; color: #ccc;
}
.picker-item:hover { background: #2a2a2a; }
.create-item { color: #22c55e; font-style: italic; }
.create-icon { color: #22c55e; font-weight: bold; }
.color-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
</style>
