<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useVaultStore } from '../../stores/vault'
import {
  GetExcludedFolders,
  AddExcludedFolder,
  RemoveExcludedFolder,
  SelectFolder,
} from '../../api/backend'

const vaultStore = useVaultStore()
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
  } catch (e: any) {
    addError.value = e.message || 'Failed to add excluded folder'
  } finally {
    isAdding.value = false
  }
}

const removeExcluded = async (path: string) => {
  try {
    await RemoveExcludedFolder(path)
    await loadExcludedFolders()
  } catch (e) {
    console.error('Failed to remove excluded folder:', e)
  }
}

// ── Thumbnail quality ──────────────────────────────────────────────
const thumbnailQuality = ref(80)

// ── Auto-tag by folder ─────────────────────────────────────────────
const autoTagByFolder = ref(false)

// ── Rescan ─────────────────────────────────────────────────────────
const isRescanning = ref(false)

const rescanVault = async () => {
  if (!confirm('Re-scan the vault? This will detect added/removed files.')) return
  isRescanning.value = true
  try {
    await vaultStore.rescanVault()
  } finally {
    isRescanning.value = false
  }
}

// ── Save config (quality + auto-tag) ───────────────────────────────
const isSaving = ref(false)

const save = async () => {
  if (!vaultStore.config) return
  isSaving.value = true
  try {
    vaultStore.config.settings.thumbnail_quality = thumbnailQuality.value
    vaultStore.config.settings.auto_tag_by_folder = autoTagByFolder.value
    await vaultStore.saveConfig(vaultStore.config)
  } catch (e) {
    console.error('Failed to save config:', e)
  } finally {
    isSaving.value = false
  }
}

// ── Vault info (read-only display) ─────────────────────────────────
const vaultName = computed(() => vaultStore.currentVault?.name ?? '—')
const vaultPath = computed(() => vaultStore.currentVault?.path ?? '—')

// ── Lifecycle ──────────────────────────────────────────────────────
onMounted(async () => {
  await loadExcludedFolders()
  // Initialise settings from current config
  if (vaultStore.config?.settings) {
    thumbnailQuality.value = vaultStore.config.settings.thumbnail_quality ?? 80
    autoTagByFolder.value = vaultStore.config.settings.auto_tag_by_folder ?? false
  }
})
</script>

<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>Vault Settings</h3>
        <button class="close-btn" @click="$emit('close')">✕</button>
      </div>

      <div class="modal-body">
        <!-- Vault info -->
        <section class="section">
          <h4>Vault</h4>
          <div class="info-row">
            <span class="label">Name</span>
            <span class="value">{{ vaultName }}</span>
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

        <!-- Auto-tag by folder -->
        <section class="section">
          <h4>Auto-Tagging</h4>
          <label class="checkbox-row">
            <input type="checkbox" v-model="autoTagByFolder" />
            <span>Auto-tag files by folder name</span>
          </label>
        </section>

        <!-- Excluded folders -->
        <section class="section">
          <h4>Excluded Folders</h4>
          <p class="hint">Folders listed here are skipped during indexing.</p>

          <div class="excluded-list" v-if="excludedFolders.length">
            <div v-for="path in excludedFolders" :key="path" class="excluded-item">
              <span class="excluded-path">{{ path }}</span>
              <button class="remove-btn" @click="removeExcluded(path)" title="Remove">✕</button>
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
            <button class="picker-btn" @click="pickFolder" title="Browse…">📁</button>
            <button
              class="add-btn"
              :disabled="isAdding || !newExcludedPath.trim()"
              @click="addExcluded"
            >
              {{ isAdding ? '…' : '+' }}
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
            @click="save"
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
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  background: #1e1e1e;
  border-radius: 8px;
  width: 480px;
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

/* ── Vault info ─────────────────────────────────────────────────── */
.info-row {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
}

.info-row .label {
  color: #888;
}

.info-row .value {
  color: #ccc;
  text-align: right;
  max-width: 70%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.info-row .path {
  font-family: monospace;
  font-size: 11px;
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
  accent-color: #5b8af5;
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

/* ── Actions ────────────────────────────────────────────────────── */
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
  padding-top: 12px;
  border-top: 1px solid #333;
}

.rescan-btn,
.save-btn {
  border: none;
  border-radius: 4px;
  padding: 8px 16px;
  font-size: 12px;
  cursor: pointer;
  transition: opacity 0.15s;
}

.rescan-btn {
  background: #333;
  color: #ccc;
}

.rescan-btn:hover:not(:disabled) {
  background: #444;
}

.save-btn {
  background: #5b8af5;
  color: #fff;
}

.save-btn:hover:not(:disabled) {
  background: #4a7ae4;
}

.rescan-btn:disabled,
.save-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
