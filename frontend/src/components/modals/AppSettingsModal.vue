<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { X, Check, FolderOpen, Trash2, FolderOpen as FolderOpenIcon } from '@lucide/vue'
import { GetAppSettings, SetAppSettings, SelectFolder, GetRecentVaults, RemoveRecentVault } from '../../api/backend'
import { useToast } from '../../composables/useToast'
import type { RecentVault } from '../../types/vault'
import { useVaultStore } from '../../stores/vault'

const vaultStore = useVaultStore()

defineEmits<{ close: [] }>()

const { success, error: toastError } = useToast()

// ── Settings state ────────────────────────────────────────────────
const lastVaultPath = ref('')
const autoOpenLastVault = ref(true)
const defaultGridSize = ref<'small' | 'medium' | 'large'>('medium')
const defaultSortField = ref('indexed_at')
const defaultSortOrder = ref('desc')
const confirmBeforeExit = ref(false)

const isSaving = ref(false)
const isLoaded = ref(false)

// ── Recent vaults ─────────────────────────────────────────────────
const recentVaults = ref<RecentVault[]>([])

const loadRecentVaults = async () => {
  recentVaults.value = await GetRecentVaults()
}

const openRecentVault = async (vault: RecentVault) => {
  await vaultStore.openVault(vault.path)
}

const removeRecentVault = async (path: string) => {
  try {
    await RemoveRecentVault(path)
    await loadRecentVaults()
  } catch (e: any) {
    toastError('Failed to remove vault: ' + (e.message || String(e)))
  }
}

const formatDate = (iso: string) => {
  try {
    const d = new Date(iso)
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }) +
      ' ' + d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  } catch {
    return iso
  }
}

// ── Load settings on mount ────────────────────────────────────────
onMounted(async () => {
  try {
    const settings = await GetAppSettings()
    if (settings) {
      lastVaultPath.value = settings.last_vault_path || ''
      autoOpenLastVault.value = settings.auto_open_last_vault ?? true
      defaultGridSize.value = (settings.default_grid_size as any) || 'medium'
      defaultSortField.value = settings.default_sort_field || 'indexed_at'
      defaultSortOrder.value = settings.default_sort_order || 'desc'
      confirmBeforeExit.value = settings.confirm_before_exit ?? false
    }
  } catch (e) {
    console.warn('[AppSettingsModal] failed to load settings:', e)
  }
  await loadRecentVaults()
  isLoaded.value = true
})

// ── Save ──────────────────────────────────────────────────────────
const save = async () => {
  isSaving.value = true
  try {
    await SetAppSettings({
      last_vault_path: lastVaultPath.value,
      recent_vaults: recentVaults.value,
      auto_open_last_vault: autoOpenLastVault.value,
      default_grid_size: defaultGridSize.value,
      default_sort_field: defaultSortField.value,
      default_sort_order: defaultSortOrder.value,
      confirm_before_exit: confirmBeforeExit.value,
    } as any)
    success('Settings saved')
  } catch (e: any) {
    toastError('Failed to save settings: ' + (e.message || String(e)))
  } finally {
    isSaving.value = false
  }
}

// ── Pick last vault folder ────────────────────────────────────────
const pickLastVault = async () => {
  const dir = await SelectFolder()
  if (dir) lastVaultPath.value = dir
}

// ── Options ───────────────────────────────────────────────────────
const gridSizeOptions = [
  { value: 'small', label: 'Small (128px)' },
  { value: 'medium', label: 'Medium (192px)' },
  { value: 'large', label: 'Large (256px)' },
] as const

const sortFieldOptions = [
  { value: 'indexed_at', label: 'Indexed At' },
  { value: 'filename', label: 'Filename' },
  { value: 'name', label: 'Name' },
  { value: 'file_size', label: 'File Size' },
  { value: 'rating', label: 'Rating' },
  { value: 'date_modified', label: 'Date Modified' },
  { value: 'date_created', label: 'Date Created' },
] as const

const sortOrderOptions = [
  { value: 'desc', label: 'Descending' },
  { value: 'asc', label: 'Ascending' },
] as const
</script>

<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>App Settings</h3>
        <button class="close-btn" @click="$emit('close')"><X :size="16" /></button>
      </div>

      <div class="modal-body" v-if="isLoaded">
        <!-- General -->
        <section class="section">
          <h4>General</h4>
          <label class="checkbox-row">
            <input type="checkbox" v-model="autoOpenLastVault" />
            <span>Auto-open last vault on startup</span>
          </label>
          <label class="checkbox-row">
            <input type="checkbox" v-model="confirmBeforeExit" />
            <span>Confirm before exiting the app</span>
          </label>
        </section>

        <!-- Last Vault -->
        <section class="section">
          <h4>Last Opened Vault</h4>
          <div class="path-row">
            <input
              v-model="lastVaultPath"
              placeholder="No vault set…"
              class="form-input"
              readonly
            />
            <button class="picker-btn" @click="pickLastVault" title="Browse…">
              <FolderOpen :size="14" />
            </button>
          </div>
          <p class="hint">Auto-opened when the app starts (if enabled above).</p>
        </section>

        <!-- Recent Vaults -->
        <section class="section">
          <h4>Recent Vaults ({{ recentVaults.length }})</h4>
          <div v-if="recentVaults.length === 0" class="empty-recent">
            <p>No recent vaults.</p>
          </div>
          <div v-else class="recent-list">
            <div
              v-for="vault in recentVaults"
              :key="vault.path"
              class="recent-item"
              @click="openRecentVault(vault)"
            >
              <div class="recent-info">
                <FolderOpenIcon :size="14" class="recent-icon" />
                <div class="recent-text">
                  <span class="recent-name">{{ vault.name }}</span>
                  <span class="recent-path">{{ vault.path }}</span>
                </div>
              </div>
              <div class="recent-actions">
                <span class="recent-date">{{ formatDate(vault.opened_at) }}</span>
                <button
                  class="remove-btn"
                  @click.stop="removeRecentVault(vault.path)"
                  title="Remove from list"
                >
                  <Trash2 :size="12" />
                </button>
              </div>
            </div>
          </div>
        </section>

        <!-- Display -->
        <section class="section">
          <h4>Display Defaults</h4>
          <div class="select-row">
            <label>Default Grid Size</label>
            <select v-model="defaultGridSize" class="form-select">
              <option v-for="opt in gridSizeOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>
          <div class="select-row">
            <label>Default Sort</label>
            <select v-model="defaultSortField" class="form-select">
              <option v-for="opt in sortFieldOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
            <select v-model="defaultSortOrder" class="form-select order-select">
              <option v-for="opt in sortOrderOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>
        </section>

        <!-- Actions -->
        <div class="actions">
          <button
            class="save-btn"
            :disabled="isSaving"
            @click="save"
          >
            <Check v-if="!isSaving" :size="14" />
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
  width: 440px;
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

/* ── Checkbox ───────────────────────────────────────────────────── */
.checkbox-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #ccc;
  cursor: pointer;
  padding: 4px 0;
}

.checkbox-row input[type='checkbox'] {
  accent-color: #22c55e;
}

/* ── Path row ───────────────────────────────────────────────────── */
.path-row {
  display: flex;
  gap: 4px;
}

.form-input {
  flex: 1;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #ccc;
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 12px;
  outline: none;
  font-family: monospace;
}

.form-input::placeholder {
  color: #444;
}

.picker-btn {
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #999;
  border-radius: 6px;
  padding: 5px 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.picker-btn:hover {
  background: #222;
  border-color: #333;
  color: #e8e8e8;
}

/* ── Select rows ────────────────────────────────────────────────── */
.select-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #ccc;
  padding: 4px 0;
}

.select-row label {
  min-width: 120px;
  font-size: 12px;
  color: #999;
}

.form-select {
  flex: 1;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #e8e8e8;
  border-radius: 6px;
  padding: 6px 8px;
  font-size: 12px;
  outline: none;
  font-family: 'Inter', sans-serif;
  cursor: pointer;
  transition: border-color 0.15s;
}

.form-select:focus {
  border-color: #22c55e;
}

.order-select {
  max-width: 120px;
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

.save-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  border: none;
  border-radius: 6px;
  padding: 8px 18px;
  font-size: 12px;
  font-weight: 500;
  font-family: 'Inter', sans-serif;
  cursor: pointer;
  background: #22c55e;
  color: #000;
  transition: all 0.15s;
}

.save-btn:hover:not(:disabled) {
  background: #16a34a;
}

.save-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ── Recent vaults ──────────────────────────────────────────────── */
.empty-recent {
  color: #555;
  font-size: 12px;
  padding: 8px 0;
}

.empty-recent p {
  margin: 0;
}

.recent-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.recent-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.recent-item:hover {
  background: #222;
  border-color: #333;
}

.recent-info {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.recent-icon {
  color: #22c55e;
  flex-shrink: 0;
}

.recent-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.recent-name {
  color: #e8e8e8;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.recent-path {
  color: #666;
  font-size: 10px;
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.recent-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.recent-date {
  color: #555;
  font-size: 10px;
  white-space: nowrap;
}

.remove-btn {
  background: none;
  border: none;
  color: #555;
  cursor: pointer;
  padding: 2px;
  border-radius: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.remove-btn:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
}
</style>
