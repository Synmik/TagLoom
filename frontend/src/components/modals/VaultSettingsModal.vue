<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>Vault Settings</h3>
        <button class="close-btn" @click="$emit('close')">✕</button>
      </div>
      <div class="modal-body">
        <div class="form-group checkbox">
          <label>
            <input type="checkbox" v-model="settings.auto_tag_by_folder" />
            Auto-tag by folder name
          </label>
        </div>

        <div class="form-group">
          <label>Excluded Folders</label>
          <div class="excluded-list">
            <div v-for="(path, i) in settings.excluded_folders" :key="i" class="excluded-item">
              <span>{{ path }}</span>
              <button @click="removeExcluded(i)">×</button>
            </div>
          </div>
          <div class="add-row">
            <input v-model="newExcluded" placeholder="Folder path…" class="form-input" />
            <button @click="addExcluded" class="add-btn">+</button>
          </div>
        </div>

        <button class="save-btn" @click="save">Save Settings</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useVaultStore } from '../../stores/vault'
import type { VaultSettings } from '../../types/vault'

const vaultStore = useVaultStore()
defineEmits<{ close: [] }>()

const settings = ref<VaultSettings>(vaultStore.config?.settings || {
  auto_tag_by_folder: false,
  excluded_folders: [],
  thumbnail_size: 256,
  thumbnail_quality: 80,
  default_sort_field: 'indexed_at',
  default_sort_order: 'desc',
  grid_thumbnail_size: 'medium',
})

const newExcluded = ref('')

const addExcluded = () => {
  if (newExcluded.value.trim()) {
    settings.value.excluded_folders.push(newExcluded.value.trim())
    newExcluded.value = ''
  }
}

const removeExcluded = (index: number) => {
  settings.value.excluded_folders.splice(index, 1)
}

const save = async () => {
  if (vaultStore.config) {
    vaultStore.config.settings = settings.value
    await vaultStore.saveConfig(vaultStore.config)
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.6);
  display: flex; align-items: center; justify-content: center; z-index: 100;
}
.modal { background: #1e1e1e; border-radius: 8px; width: 400px; border: 1px solid #333; }
.modal-header { display: flex; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid #333; }
.modal-header h3 { margin: 0; color: #fff; font-size: 14px; }
.close-btn { background: none; border: none; color: #888; cursor: pointer; font-size: 16px; }
.modal-body { padding: 16px; display: flex; flex-direction: column; gap: 12px; }
.form-group { display: flex; flex-direction: column; gap: 6px; }
.form-group label { color: #888; font-size: 11px; }
.checkbox { flex-direction: row; align-items: center; gap: 6px; }
.excluded-list { display: flex; flex-direction: column; gap: 4px; }
.excluded-item { display: flex; justify-content: space-between; background: #2a2a2a; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: #ccc; }
.excluded-item button { background: none; border: none; color: #888; cursor: pointer; }
.add-row { display: flex; gap: 4px; margin-top: 4px; }
.form-input { flex: 1; background: #2a2a2a; border: 1px solid #444; color: #fff; border-radius: 4px; padding: 4px 8px; font-size: 12px; outline: none; }
.add-btn { background: #2a2a2a; border: 1px solid #444; color: #ccc; border-radius: 4px; width: 28px; cursor: pointer; }
.save-btn { background: #5b8af5; color: #fff; border: none; border-radius: 4px; padding: 8px; cursor: pointer; margin-top: 4px; }
</style>
