<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>{{ isEditing ? 'Edit Tag' : 'Create Tag' }}</h3>
        <button class="close-btn" @click="$emit('close')">✕</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>Name</label>
          <input v-model="form.name" placeholder="Tag name" class="form-input" @keyup.enter="saveTag" />
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
import ColorPicker from '../common/ColorPicker.vue'
import { useTagsStore } from '../../stores/tags'
import type { Tag } from '../../types/tag'

const tagsStore = useTagsStore()
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
  const allTags = tagsStore.tags
  if (!props.tag) return allTags
  return allTags.filter(t => t.id !== props.tag!.id)
})

const saveTag = async () => {
  if (!form.value.name.trim()) return

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
  } else {
    await tagsStore.createTag({
      name: form.value.name.trim(),
      color: form.value.color,
      parent_id: form.value.parentId ?? undefined,
      is_category: form.value.isCategory ? 1 : 0,
      sort_order: 0,
      aliases: form.value.aliases,
    })
  }
  emit('close')
}

const deleteTag = async () => {
  if (!props.tag) return
  if (!confirm(`Delete tag "${props.tag.name}"? Files will lose this tag.`)) return
  await tagsStore.deleteTag(props.tag.id)
  emit('close')
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
.modal-body { padding: 16px; display: flex; flex-direction: column; gap: 10px; }
.form-group { display: flex; flex-direction: column; gap: 4px; }
.form-group label { color: #888; font-size: 11px; }
.form-input, .form-select {
  background: #2a2a2a; border: 1px solid #444; color: #fff;
  border-radius: 4px; padding: 6px 8px; font-size: 13px; outline: none;
}
.checkbox { flex-direction: row; align-items: center; gap: 6px; }
.form-actions { display: flex; gap: 8px; margin-top: 4px; }
.save-btn {
  flex: 1; background: #5b8af5; color: #fff; border: none; border-radius: 4px;
  padding: 8px; cursor: pointer; font-size: 13px;
}
.save-btn:hover { background: #4a7ae4; }
.delete-btn {
  background: #5a2a2a; color: #ff6b6b; border: none; border-radius: 4px;
  padding: 8px 12px; cursor: pointer; font-size: 13px;
}
.delete-btn:hover { background: #6a3a3a; }
</style>
