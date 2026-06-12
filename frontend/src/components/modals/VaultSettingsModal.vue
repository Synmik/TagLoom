<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { X, FolderOpen, Plus, Pencil } from '@lucide/vue'
import { useVaultStore } from '../../stores/vault'
import { useToast } from '../../composables/useToast'
import ConfirmDialog from '../common/ConfirmDialog.vue'
import {
  GetExcludedFolders,
  AddExcludedFolder,
  RemoveExcludedFolder,
  SelectFolder,
} from '../../api/backend'

const vaultStore = useVaultStore()
const { success, error: toastError } = useToast()
defineEmits<{ close: [] }>()

// ── Excluded folders (loaded from backend) ──────────────────────────
const excludedFolders = ref<string[]>([])
const newExcludedPath = ref('')
const addError = ref('')
const isAdding = ref(false)

const loadExcludedFolders = async () => {
  try {
    const result = await GetExcludedFolders()
    excludedFolders.value = Array.isArray(result) ? result : []
  } catch (e) {
    console.error('Failed to load excluded folders:', e)
  }
}

const pickFolder = async () => {
  const dir = await SelectFolder()
  if (!dir) return
  // Store the full absolute path — the backend normalises it
  newExcludedPath.value = dir
}

const addExcluded = async () => {
  const path = newExcludedPath.value.trim()
  if (!path) return

  isAdding.value = true
  addError.value = ''
  try {
    await AddExcludedFolder(path)
    newExcludedPath.value = ''
    await loadExcludedFolders()
    success(`Excluded "${path}"`)
  } catch (e: any) {
    addError.value = e.message || 'Failed to add excluded folder'
    toastError(addError.value)
  } finally {
    isAdding.value = false
  }
}

const removeExcluded = async (path: string) => {
  try {
    await RemoveExcludedFolder(path)
    await loadExcludedFolders()
    success(`Un-excluded "${path}"`)
  } catch (e: any) {
    toastError('Failed to remove excluded folder')
  }
}

// ── Thumbnail quality ──────────────────────────────────────────────
const thumbnailQuality = ref(80)

// ── Rescan ─────────────────────────────────────────────────────────
const isRescanning = ref(false)

const showRescanConfirm = ref(false)

const rescanVault = () => {
  showRescanConfirm.value = true
}

const confirmRescan = async () => {
  showRescanConfirm.value = false
  isRescanning.value = true
  try {
    await vaultStore.rescanVault()
    success('Vault re-scan complete')
  } catch (e: any) {
    toastError('Re-scan failed: ' + (e.message || String(e)))
  } finally {
    isRescanning.value = false
  }
}

// ── Save config (quality + auto-tag) ───────────────────────────────
const isSaving = ref(false)

const saveNameAndSettings = async () => {
  if (!vaultStore.config) return
  isSaving.value = true
  const wasEditingName = editingName.value
  try {
    vaultStore.config.settings.thumbnail_quality = thumbnailQuality.value
    // Update vault name if editing
    const newNameTrimmed = newName.value.trim()
    if (wasEditingName && newNameTrimmed) {
      vaultStore.config.name = newNameTrimmed
      // Also update currentVault so TitleBar reflects the change
      if (vaultStore.currentVault) {
        vaultStore.currentVault.name = newNameTrimmed
      }
      editingName.value = false
    }
    await vaultStore.saveConfig(vaultStore.config)
    success('Settings saved')
  } catch (e: any) {
    const msg = e.message || String(e)
    if (wasEditingName) {
      toastError('Failed to rename vault: ' + msg)
    } else {
      toastError('Failed to save settings')
    }
  } finally {
    isSaving.value = false
  }
}

// ── Vault info ─────────────────────────────────────────────────────
const vaultName = computed(() => vaultStore.config?.name ?? vaultStore.currentVault?.name ?? '—')
const vaultPath = computed(() => vaultStore.currentVault?.path ?? '—')
const editingName = ref(false)
const newName = ref('')

const startEditingName = () => {
  newName.value = vaultStore.config?.name ?? vaultStore.currentVault?.name ?? ''
  editingName.value = true
}

const cancelEditingName = () => {
  editingName.value = false
  newName.value = ''
}



// ── Lifecycle ──────────────────────────────────────────────────────
onMounted(async () => {
  await loadExcludedFolders()
  // Initialise settings from current config
  if (vaultStore.config?.settings) {
    thumbnailQuality.value = vaultStore.config.settings.thumbnail_quality ?? 80
  }
})
</script>

<template>
  <ConfirmDialog
    v-if="showRescanConfirm"
    message="Re-scan the vault? This will detect added/removed files."
    confirm-text="Re-scan"
    @confirm="confirmRescan"
    @cancel="showRescanConfirm = false"
  />
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>Vault Settings</h3>
        <button class="close-btn" @click="$emit('close')"><X :size="16" /></button>
      </div>

      <div class="modal-body">
        <!-- Vault info -->
        <section class="section">
          <h4>Vault</h4>
          <div class="info-row">
            <span class="label">Name</span>
            <div class="name-cell">
              <template v-if="editingName">
                <input
                  v-model="newName"
                  class="name-input"
                  @keydown.enter="saveNameAndSettings()"
                  @keydown.escape="cancelEditingName()"
                  autofocus
                />
              </template>
              <template v-else>
                <span class="value">{{ vaultName }}</span>
                <button class="edit-btn" @click="startEditingName" title="Rename"><Pencil :size="12" /></button>
              </template>
            </div>
          </div>
          <div class="info-row">
            <span class="label">Path</span>
            <span class="value path">{{ vaultPath }}</span>
          </div>
        </section>

        <!-- Thumbnail quality -->
        <section class="section">
          <h4>Thumbnails</h4>
          <div class="slider-row">
            <label>Quality</label>
            <input
              type="range"
              min="10"
              max="100"
              step="5"
              v-model.number="thumbnailQuality"
            />
            <span class="slider-value">{{ thumbnailQuality }}%</span>
          </div>
        </section>

        <!-- Excluded folders -->
        <section class="section">
          <h4>Excluded Folders</h4>
          <p class="hint">Folders listed here are skipped during indexing.</p>

          <div class="excluded-list" v-if="excludedFolders.length">
            <div v-for="path in excludedFolders" :key="path" class="excluded-item">
              <span class="excluded-path">{{ path }}</span>
              <button class="remove-btn" @click="removeExcluded(path)" title="Remove"><X :size="14" /></button>
            </div>
          </div>
          <p v-else class="empty-hint">No excluded folders</p>

          <div class="add-row">
            <input
              v-model="newExcludedPath"
              placeholder="Folder path…"
              class="form-input"
              @keydown.enter="addExcluded"
            />
            <button class="picker-btn" @click="pickFolder" title="Browse…"><FolderOpen :size="14" /></button>
            <button
              class="add-btn"
              :disabled="isAdding || !newExcludedPath.trim()"
              @click="addExcluded"
            >
              <Plus v-if="!isAdding" :size="14" />
            </button>
          </div>
          <p v-if="addError" class="error">{{ addError }}</p>
        </section>

        <!-- Actions -->
        <div class="actions">
          <button
            class="rescan-btn"
            :disabled="isRescanning"
            @click="rescanVault"
          >
            {{ isRescanning ? 'Re-scanning…' : 'Re-scan Vault' }}
          </button>
          <button
            class="save-btn"
            :disabled="isSaving"
            @click="saveNameAndSettings"
          >
            {{ isSaving ? 'Saving…' : 'Save Settings' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  background: #111111;
  border-radius: 12px;
  width: 480px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  border: 1px solid #222;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 18px;
  border-bottom: 1px solid #1a1a1a;
}

.modal-header h3 {
  margin: 0;
  color: #e8e8e8;
  font-size: 14px;
  font-weight: 600;
  font-family: 'Inter', sans-serif;
}

.close-btn {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  font-size: 16px;
  padding: 4px;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
}

.close-btn:hover {
  color: #e8e8e8;
  background: #1a1a1a;
}

.modal-body {
  padding: 18px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

/* ── Sections ───────────────────────────────────────────────────── */
.section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section h4 {
  margin: 0;
  color: #888;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.6px;
}

.hint {
  margin: 0;
  color: #555;
  font-size: 11px;
  line-height: 1.5;
}

/* ── Vault info ─────────────────────────────────────────────────── */
.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  padding: 4px 0;
}

.info-row .label {
  color: #666;
  font-size: 11px;
}

.info-row .value {
  color: #ccc;
  text-align: right;
  max-width: 85%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-direction: row-reverse;
  flex: 1;
  min-width: 0;
}

.name-input {
  background: #1a1a1a;
  border: 1px solid #22c55e;
  color: #e8e8e8;
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 12px;
  outline: none;
  width: 250px;
  text-align: right;
  font-family: 'Inter', sans-serif;
}

.edit-btn {
  background: none;
  border: none;
  color: #555;
  cursor: pointer;
  padding: 2px;
  transition: color 0.15s;
  flex-shrink: 0;
}

.edit-btn:hover {
  color: #22c55e;
}

.info-row .path {
  font-family: monospace;
  font-size: 11px;
}

/* ── Slider ─────────────────────────────────────────────────────── */
.slider-row {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: #ccc;
}

.slider-row label {
  min-width: 60px;
  font-size: 12px;
}

.slider-row input[type='range'] {
  flex: 1;
  height: 4px;
  -webkit-appearance: none;
  appearance: none;
  background: #2a2a2a;
  border-radius: 2px;
  outline: none;
  cursor: pointer;
}

.slider-row input[type='range']::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 12px;
  height: 12px;
  margin-top: -4px;
  background: #22c55e;
  border: none;
  border-radius: 2px;
  cursor: pointer;
}

.slider-row input[type='range']::-moz-range-thumb {
  width: 12px;
  height: 12px;
  background: #22c55e;
  border: none;
  border-radius: 2px;
  cursor: pointer;
}

.slider-row input[type='range']::-webkit-slider-runnable-track {
  height: 4px;
  background: #2a2a2a;
  border-radius: 2px;
}

.slider-row input[type='range']::-moz-range-track {
  height: 4px;
  background: #2a2a2a;
  border-radius: 2px;
}

.slider-value {
  min-width: 40px;
  text-align: right;
  color: #22c55e;
  font-weight: 600;
  font-size: 12px;
}

/* ── Checkbox ───────────────────────────────────────────────────── */
.checkbox-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #ccc;
  cursor: pointer;
}

.checkbox-row input[type='checkbox'] {
  accent-color: #22c55e;
}

/* ── Excluded folders ───────────────────────────────────────────── */
.excluded-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 160px;
  overflow-y: auto;
}

.excluded-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #1a1a1a;
  padding: 6px 10px;
  border-radius: 6px;
  font-size: 12px;
}

.excluded-path {
  color: #ccc;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  margin-right: 8px;
  font-family: monospace;
  font-size: 11px;
}

.remove-btn {
  background: none;
  border: none;
  color: #555;
  cursor: pointer;
  font-size: 14px;
  padding: 0 4px;
  flex-shrink: 0;
  transition: color 0.15s;
}

.remove-btn:hover {
  color: #ef4444;
}

.empty-hint {
  margin: 0;
  color: #444;
  font-size: 11px;
  font-style: italic;
}

.add-row {
  display: flex;
  gap: 4px;
  margin-top: 4px;
}

.form-input {
  flex: 1;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #e8e8e8;
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 12px;
  outline: none;
  font-family: monospace;
  transition: border-color 0.15s;
}

.form-input:focus {
  border-color: #22c55e;
}

.form-input::placeholder {
  color: #444;
}

.picker-btn,
.add-btn {
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #999;
  border-radius: 6px;
  padding: 5px 8px;
  cursor: pointer;
  font-size: 14px;
  min-width: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.picker-btn:hover,
.add-btn:hover:not(:disabled) {
  background: #222;
  border-color: #333;
  color: #e8e8e8;
}

.add-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.error {
  margin: 0;
  color: #ef4444;
  font-size: 11px;
}

/* ── Actions ────────────────────────────────────────────────────── */
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
  padding-top: 14px;
  border-top: 1px solid #1a1a1a;
}

.rescan-btn,
.save-btn {
  border: none;
  border-radius: 6px;
  padding: 8px 18px;
  font-size: 12px;
  font-weight: 500;
  font-family: 'Inter', sans-serif;
  cursor: pointer;
  transition: all 0.15s;
}

.rescan-btn {
  background: #1a1a1a;
  color: #999;
  border: 1px solid #2a2a2a;
}

.rescan-btn:hover:not(:disabled) {
  background: #222;
  color: #e8e8e8;
}

.save-btn {
  background: #22c55e;
  color: #000;
}

.save-btn:hover:not(:disabled) {
  background: #16a34a;
}

.rescan-btn:disabled,
.save-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
