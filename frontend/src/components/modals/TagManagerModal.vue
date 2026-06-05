<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>{{ isEditing ? 'Edit Tag' : 'Create Tag' }}</h3>
        <button class="close-btn" @click="$emit('close')"><X :size="16" /></button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>Name</label>
          <input v-model="form.name" placeholder="Tag name" class="form-input" :class="{ 'input-error': nameError }" @keyup.enter="saveTag" />
          <span v-if="nameError" class="error-text">{{ nameError }}</span>
        </div>
        <div class="form-group">
          <label>Color</label>
          <ColorPicker v-model="form.color" />
        </div>
        <div class="form-group">
          <label>Parent Tag</label>
          <select v-model="form.parentId" class="form-select">
            <option :value="null">None (root)</option>
            <option v-for="tag in parentTags" :key="tag.id" :value="tag.id">{{ tag.name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label>Aliases (comma-separated)</label>
          <input v-model="form.aliases" placeholder="alt1, alt2" class="form-input" />
        </div>
        <div class="form-group checkbox">
          <label>
            <input type="checkbox" v-model="form.isCategory" /> Is Category
          </label>
        </div>
        <div class="form-actions">
          <button class="save-btn" @click="saveTag">{{ isEditing ? 'Save' : 'Create' }}</button>
          <button v-if="isEditing" class="delete-btn" @click="deleteTag">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { X } from '@lucide/vue'
import ColorPicker from '../common/ColorPicker.vue'
import { useTagsStore } from '../../stores/tags'
import { useToast } from '../../composables/useToast'
import type { Tag } from '../../types/tag'

const tagsStore = useTagsStore()
const { success, error: toastError } = useToast()
const emit = defineEmits<{ close: [] }>()

const props = defineProps<{
  tag?: Tag | null
}>()

const isEditing = computed(() => !!props.tag)

const form = ref({
  name: '',
  color: '',
  parentId: null as number | null,
  aliases: '',
  isCategory: false,
})

// Populate form when editing an existing tag
watch(() => props.tag, (tag) => {
  if (tag) {
    form.value = {
      name: tag.name,
      color: tag.color || '',
      parentId: tag.parent_id ?? null,
      aliases: '',
      isCategory: tag.is_category === 1,
    }
  } else {
    form.value = { name: '', color: '', parentId: null, aliases: '', isCategory: false }
  }
}, { immediate: true })

// Exclude the current tag from parent dropdown to avoid self-reference
const parentTags = computed(() => {
  const allTags = tagsStore.tags || []
  if (!props.tag) return allTags
  return allTags.filter(t => t.id !== props.tag!.id)
})

// Check for case-insensitive name collision ("App" vs "app")
const nameError = computed(() => {
  const name = form.value.name.trim().toLowerCase()
  if (!name) return ''
  const allTags = tagsStore.tags || []
  const duplicate = allTags.find(
    t => t.name.toLowerCase() === name && t.id !== (props.tag?.id ?? -1)
  )
  if (duplicate) {
    return `Tag "${duplicate.name}" already exists (case-insensitive)`
  }
  return ''
})

const saveTag = async () => {
  if (!form.value.name.trim()) return
  if (nameError.value) return

  try {
    if (isEditing.value && props.tag) {
      await tagsStore.updateTag({
        id: props.tag.id,
        name: form.value.name.trim(),
        color: form.value.color,
        parent_id: form.value.parentId ?? undefined,
        is_category: form.value.isCategory ? 1 : 0,
        sort_order: 0,
        aliases: form.value.aliases,
      })
      success(`Tag "${form.value.name}" updated`)
    } else {
      await tagsStore.createTag({
        name: form.value.name.trim(),
        color: form.value.color,
        parent_id: form.value.parentId ?? undefined,
        is_category: form.value.isCategory ? 1 : 0,
        sort_order: 0,
        aliases: form.value.aliases,
      })
      success(`Tag "${form.value.name}" created`)
    }
  } catch (e: any) {
    toastError('Failed to save tag: ' + (e.message || String(e)))
    return
  }
  emit('close')
}

const deleteTag = async () => {
  if (!props.tag) return
  if (!confirm(`Delete tag "${props.tag.name}"? Files will lose this tag.`)) return
  try {
    await tagsStore.deleteTag(props.tag.id)
    success(`Tag "${props.tag.name}" deleted`)
  } catch (e: any) {
    toastError('Failed to delete tag: ' + (e.message || String(e)))
    return
  }
  emit('close')
}
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.7);
  display: flex; align-items: center; justify-content: center; z-index: 100;
}
.modal { background: #111111; border-radius: 12px; width: 360px; border: 1px solid #222; box-shadow: 0 16px 48px rgba(0,0,0,0.6); }
.modal-header { display: flex; justify-content: space-between; padding: 14px 18px; border-bottom: 1px solid #1a1a1a; }
.modal-header h3 { margin: 0; color: #e8e8e8; font-size: 14px; font-weight: 600; font-family: 'Inter', sans-serif; }
.close-btn { background: none; border: none; color: #666; cursor: pointer; font-size: 16px; padding: 4px; border-radius: 4px; transition: color 0.15s, background 0.15s; }
.close-btn:hover { color: #e8e8e8; background: #1a1a1a; }
.modal-body { padding: 18px; display: flex; flex-direction: column; gap: 12px; }
.form-group { display: flex; flex-direction: column; gap: 6px; }
.form-group label { color: #666; font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
.form-input, .form-select {
  background: #1a1a1a; border: 1px solid #2a2a2a; color: #e8e8e8;
  border-radius: 6px; padding: 7px 10px; font-size: 13px;
  font-family: 'Inter', sans-serif;
  outline: none; transition: border-color 0.15s;
}
.form-input:focus, .form-select:focus {
  border-color: #22c55e;
}
.form-input::placeholder { color: #444; }
.form-select option { background: #1a1a1a; color: #e8e8e8; }
.input-error {
  border-color: #ef4444;
}
.error-text {
  color: #ef4444; font-size: 11px; margin-top: 2px;
}
.checkbox { flex-direction: row; align-items: center; gap: 8px; }
.checkbox label {
  display: flex; align-items: center; gap: 8px;
  text-transform: none; font-size: 13px; color: #ccc; cursor: pointer;
}
.checkbox input[type="checkbox"] { accent-color: #22c55e; }
.form-actions { display: flex; gap: 8px; margin-top: 4px; padding-top: 12px; border-top: 1px solid #1a1a1a; }
.save-btn {
  flex: 1; background: #22c55e; color: #000; border: none; border-radius: 6px;
  padding: 9px; cursor: pointer; font-size: 13px; font-weight: 500;
  font-family: 'Inter', sans-serif;
  transition: background 0.15s;
}
.save-btn:hover { background: #16a34a; }
.delete-btn {
  background: rgba(239, 68, 68, 0.1); color: #ef4444; border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: 6px;
  padding: 9px 14px; cursor: pointer; font-size: 13px;
  font-family: 'Inter', sans-serif;
  transition: all 0.15s;
}
.delete-btn:hover { background: rgba(239, 68, 68, 0.2); }
</style>
