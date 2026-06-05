<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { X, FolderOpen } from '@lucide/vue'
import { SelectFolder, CreateVault } from '../../api/backend'
import { useVaultStore } from '../../stores/vault'
import { useToast } from '../../composables/useToast'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'

const vaultStore = useVaultStore()
const { success, error: toastError } = useToast()
const emit = defineEmits<{ close: [] }>()

// ── Form state ─────────────────────────────────────────────────────
const selectedPath = ref('')
const thumbnailQuality = ref(80)
const excludedFolders = ref<string[]>([])
const newExcludedPath = ref('')
const addError = ref('')
const isCreating = ref(false)
const createError = ref('')



// ── Folder picker ──────────────────────────────────────────────────
const pickFolder = async () => {
  const dir = await SelectFolder()
  if (dir) {
    selectedPath.value = dir
    createError.value = ''
  }
}

// ── Excluded folders ──────────────────────────────────────────────
const addExcluded = () => {
  const path = newExcludedPath.value.trim()
  if (!path) return
  if (excludedFolders.value.includes(path)) {
    addError.value = 'Already in list'
    return
  }
  addError.value = ''
  excludedFolders.value.push(path)
  newExcludedPath.value = ''
}

const removeExcluded = (path: string) => {
  excludedFolders.value = excludedFolders.value.filter(p => p !== path)
}

const pickExcludedFolder = async () => {
  const dir = await SelectFolder()
  if (!dir) return
  newExcludedPath.value = dir
}

// ── Create vault ───────────────────────────────────────────────────
const createVault = async () => {
  if (!selectedPath.value) {
    createError.value = 'Please select a folder first'
    return
  }

  isCreating.value = true
  createError.value = ''

  try {
    const vault = await CreateVault(selectedPath.value, {
      thumbnail_quality: thumbnailQuality.value,
      excluded_folders: excludedFolders.value,
    } as any)

    vaultStore.currentVault = vault
    await vaultStore.loadConfig()

    // Wire up scan listeners (same as _doOpenVault)
    vaultStore.isScanning = true
    vaultStore.scanTotal = 0
    vaultStore.scanCurrent = 0

    EventsOff('scan:progress')
    EventsOff('scan:complete')
    EventsOff('scan:error')
    EventsOff('thumb:progress')
    EventsOff('thumb:complete')

    const autoProgressUnsub = EventsOn('scan:progress', (data: { current: number; total: number }) => {
      vaultStore.scanCurrent = data.current
      vaultStore.scanTotal = data.total
    })

    const thumbProgressUnsub = EventsOn('thumb:progress', (data: { current: number; total: number }) => {
      vaultStore.scanCurrent = data.current
      vaultStore.scanTotal = data.total
    })

    const thumbCompleteUnsub = EventsOn('thumb:complete', (data: { generated: number; failed: number; total: number }) => {
      vaultStore.scanCurrent = data.total
      vaultStore.scanTotal = data.total
      vaultStore.isScanning = false
      vaultStore.isLoading = false
      vaultStore.refreshCurrentVault()
    })

    setTimeout(() => {
      autoProgressUnsub()
      thumbProgressUnsub()
      thumbCompleteUnsub()
    }, 60000)

    success(`Vault "${vault.name}" created successfully`)
    emit('close')
  } catch (e: any) {
    createError.value = e.message || 'Failed to create vault'
    toastError(createError.value)
  } finally {
    isCreating.value = false
  }
}

// ── Lifecycle ──────────────────────────────────────────────────────
onMounted(() => {
  // Auto-focus on mount
})
</script>

<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>New Vault</h3>
        <button class="close-btn" @click="$emit('close')"><X :size="16" /></button>
      </div>

      <div class="modal-body">
        <!-- Folder selection -->
        <section class="section">
          <h4>Vault Folder</h4>
          <p class="hint">Select a folder to create a new vault. A <code>.tagloom</code> directory will be created inside it.</p>
          <div class="path-row">
            <input
              v-model="selectedPath"
              placeholder="Select a folder…"
              class="form-input path-input"
              readonly
            />
            <button class="picker-btn" @click="pickFolder" title="Browse…"><FolderOpen :size="14" /></button>
          </div>
        </section>

        <!-- Thumbnail quality -->
        <section class="section">
          <h4>Thumbnail Quality</h4>
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
          <p class="hint">These folders will be skipped during indexing.</p>

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
            <button class="picker-btn" @click="pickExcludedFolder" title="Browse…"><FolderOpen :size="14" /></button>
            <button
              class="add-btn"
              :disabled="!newExcludedPath.trim()"
              @click="addExcluded"
            >
              +
            </button>
          </div>
          <p v-if="addError" class="error">{{ addError }}</p>
        </section>

        <!-- Error -->
        <p v-if="createError" class="create-error">{{ createError }}</p>

        <!-- Actions -->
        <div class="actions">
          <button class="cancel-btn" @click="$emit('close')">Cancel</button>
          <button
            class="create-btn"
            :disabled="isCreating || !selectedPath.trim()"
            @click="createVault"
          >
            {{ isCreating ? 'Creating…' : 'Create Vault' }}
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
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  background: #1e1e1e;
  border-radius: 8px;
  width: 520px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  border: 1px solid #333;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #333;
}

.modal-header h3 {
  margin: 0;
  color: #fff;
  font-size: 14px;
}

.close-btn {
  background: none;
  border: none;
  color: #888;
  cursor: pointer;
  font-size: 16px;
  padding: 4px;
}

.close-btn:hover {
  color: #fff;
}

.modal-body {
  padding: 16px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ── Sections ───────────────────────────────────────────────────── */
.section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section h4 {
  margin: 0;
  color: #aaa;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.hint {
  margin: 0;
  color: #666;
  font-size: 11px;
}

.hint code {
  background: #2a2a2a;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 11px;
  color: #999;
}

/* ── Path input ─────────────────────────────────────────────────── */
.path-row {
  display: flex;
  gap: 4px;
}

.path-input {
  flex: 1;
  background: #2a2a2a;
  border: 1px solid #444;
  color: #ccc;
  border-radius: 4px;
  padding: 8px;
  font-size: 12px;
  outline: none;
  font-family: monospace;
}

.path-input:focus {
  border-color: #5b8af5;
}

/* ── Slider ─────────────────────────────────────────────────────── */
.slider-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #ccc;
}

.slider-row label {
  min-width: 60px;
}

.slider-row input[type='range'] {
  flex: 1;
  accent-color: #5b8af5;
}

.slider-value {
  min-width: 40px;
  text-align: right;
  color: #5b8af5;
  font-weight: 600;
}

/* ── Excluded folders ───────────────────────────────────────────── */
.excluded-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 120px;
  overflow-y: auto;
}

.excluded-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #2a2a2a;
  padding: 6px 8px;
  border-radius: 4px;
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
  color: #888;
  cursor: pointer;
  font-size: 14px;
  padding: 0 4px;
  flex-shrink: 0;
}

.remove-btn:hover {
  color: #f55;
}

.empty-hint {
  margin: 0;
  color: #555;
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
  background: #2a2a2a;
  border: 1px solid #444;
  color: #fff;
  border-radius: 4px;
  padding: 6px 8px;
  font-size: 12px;
  outline: none;
  font-family: monospace;
}

.form-input:focus {
  border-color: #5b8af5;
}

.picker-btn,
.add-btn {
  background: #2a2a2a;
  border: 1px solid #444;
  color: #ccc;
  border-radius: 4px;
  padding: 4px 8px;
  cursor: pointer;
  font-size: 14px;
  min-width: 28px;
}

.picker-btn:hover,
.add-btn:hover:not(:disabled) {
  background: #333;
  border-color: #5b8af5;
}

.add-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.error {
  margin: 0;
  color: #f55;
  font-size: 11px;
}

/* ── Create error ───────────────────────────────────────────────── */
.create-error {
  margin: 0;
  color: #f55;
  font-size: 12px;
  background: rgba(255, 85, 85, 0.1);
  padding: 8px;
  border-radius: 4px;
  border: 1px solid rgba(255, 85, 85, 0.2);
}

/* ── Actions ────────────────────────────────────────────────────── */
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
  padding-top: 12px;
  border-top: 1px solid #333;
}

.cancel-btn,
.create-btn {
  border: none;
  border-radius: 4px;
  padding: 8px 16px;
  font-size: 12px;
  cursor: pointer;
  transition: opacity 0.15s;
}

.cancel-btn {
  background: #333;
  color: #ccc;
}

.cancel-btn:hover {
  background: #444;
}

.create-btn {
  background: #5b8af5;
  color: #fff;
}

.create-btn:hover:not(:disabled) {
  background: #4a7ae4;
}

.create-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
