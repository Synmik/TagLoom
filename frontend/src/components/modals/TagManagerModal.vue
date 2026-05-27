<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>Tag Manager</h3>
        <button class="close-btn" @click="$emit('close')">✕</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>Name</label>
          <input v-model="form.name" placeholder="Tag name" class="form-input" />
        </div>
        <div class="form-group">
          <label>Color</label>
          <ColorPicker v-model="form.color" />
        </div>
        <div class="form-group">
          <label>Parent Tag</label>
          <select v-model="form.parentId" class="form-select">
            <option :value="null">None (root)</option>
            <option v-for="tag in tagsStore.tags" :key="tag.id" :value="tag.id">{{ tag.name }}</option>
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
        <button class="save-btn" @click="saveTag">Create Tag</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ColorPicker from '../common/ColorPicker.vue'
import { useTagsStore } from '../../stores/tags'

const tagsStore = useTagsStore()
defineEmits<{ close: [] }>()

const form = ref({
  name: '',
  color: '',
  parentId: null as number | null,
  aliases: '',
  isCategory: false,
})

const saveTag = async () => {
  if (!form.value.name.trim()) return
  await tagsStore.createTag({
    name: form.value.name,
    color: form.value.color,
    parent_id: form.value.parentId,
    is_category: form.value.isCategory ? 1 : 0,
    sort_order: 0,
    aliases: form.value.aliases,
  })
  form.value = { name: '', color: '', parentId: null, aliases: '', isCategory: false }
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
.save-btn {
  background: #5b8af5; color: #fff; border: none; border-radius: 4px;
  padding: 8px; cursor: pointer; font-size: 13px; margin-top: 4px;
}
</style>
