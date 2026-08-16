<script setup lang="ts">
import { ref, onMounted } from "vue";
import { X, FolderOpen } from "@lucide/vue";
import { SelectFolder, CreateVault } from "../../api/backend";
import { useVaultStore } from "../../stores/vault";
import { useToast } from "../../composables/useToast";
import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";

const vaultStore = useVaultStore();
const { success, error: toastError } = useToast();
const emit = defineEmits<{ close: [] }>();

// ── Form state ─────────────────────────────────────────────────────
const selectedPath = ref("");
const thumbnailQuality = ref(80);
const excludedFolders = ref<string[]>([]);
const newExcludedPath = ref("");
const addError = ref("");
const isCreating = ref(false);
const createError = ref("");

// ── Folder picker ──────────────────────────────────────────────────
const pickFolder = async () => {
  const dir = await SelectFolder();
  if (dir) {
    selectedPath.value = dir;
    createError.value = "";
  }
};

// ── Excluded folders ──────────────────────────────────────────────
const addExcluded = () => {
  const path = newExcludedPath.value.trim();
  if (!path) return;
  if (excludedFolders.value.includes(path)) {
    addError.value = "Already in list";
    return;
  }
  addError.value = "";
  excludedFolders.value.push(path);
  newExcludedPath.value = "";
};

const removeExcluded = (path: string) => {
  excludedFolders.value = excludedFolders.value.filter((p) => p !== path);
};

const pickExcludedFolder = async () => {
  const dir = await SelectFolder();
  if (!dir) return;
  newExcludedPath.value = dir;
};

// ── Create vault ───────────────────────────────────────────────────
const createVault = async () => {
  if (!selectedPath.value) {
    createError.value = "Please select a folder first";
    return;
  }

  isCreating.value = true;
  createError.value = "";

  try {
    const vault = await CreateVault(selectedPath.value, {
      thumbnail_quality: thumbnailQuality.value,
      excluded_folders: excludedFolders.value,
    } as any);

    vaultStore.currentVault = vault;
    await vaultStore.loadConfig();

    // Wire up scan listeners (same as _doOpenVault)
    vaultStore.isScanning = true;
    vaultStore.scanTotal = 0;
    vaultStore.scanCurrent = 0;

    EventsOff("scan:progress");
    EventsOff("scan:complete");
    EventsOff("scan:error");
    EventsOff("thumb:progress");
    EventsOff("thumb:complete");

    const autoProgressUnsub = EventsOn(
      "scan:progress",
      (data: { current: number; total: number }) => {
        vaultStore.scanCurrent = data.current;
        vaultStore.scanTotal = data.total;
      },
    );

    const thumbProgressUnsub = EventsOn(
      "thumb:progress",
      (data: { current: number; total: number }) => {
        vaultStore.scanCurrent = data.current;
        vaultStore.scanTotal = data.total;
      },
    );

    const thumbCompleteUnsub = EventsOn(
      "thumb:complete",
      (data: { generated: number; failed: number; total: number }) => {
        vaultStore.scanCurrent = data.total;
        vaultStore.scanTotal = data.total;
        vaultStore.isScanning = false;
        vaultStore.isLoading = false;
        vaultStore.refreshCurrentVault();
      },
    );

    setTimeout(() => {
      autoProgressUnsub();
      thumbProgressUnsub();
      thumbCompleteUnsub();
    }, 60000);

    success(`Vault "${vault.name}" created successfully`);
    emit("close");
  } catch (e: any) {
    createError.value = e.message || "Failed to create vault";
    toastError(createError.value);
  } finally {
    isCreating.value = false;
  }
};

// ── Lifecycle ──────────────────────────────────────────────────────
onMounted(() => {
  // Auto-focus on mount
});
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
          <p class="hint">
            Select a folder to create a new vault. A <code>.tagloom</code> directory will be created
            inside it.
          </p>
          <div class="path-row">
            <input
              v-model="selectedPath"
              placeholder="Select a folder…"
              class="form-input path-input"
              readonly
            />
            <button class="picker-btn" title="Browse…" @click="pickFolder">
              <FolderOpen :size="14" />
            </button>
          </div>
        </section>

        <!-- Thumbnail quality -->
        <section class="section">
          <h4>Thumbnail Quality</h4>
          <div class="slider-row">
            <label>Quality</label>
            <input v-model.number="thumbnailQuality" type="range" min="10" max="100" step="5" />
            <span class="slider-value">{{ thumbnailQuality }}%</span>
          </div>
        </section>

        <!-- Excluded folders -->
        <section class="section">
          <h4>Excluded Folders</h4>
          <p class="hint">These folders will be skipped during indexing.</p>

          <div v-if="excludedFolders.length" class="excluded-list">
            <div v-for="path in excludedFolders" :key="path" class="excluded-item">
              <span class="excluded-path">{{ path }}</span>
              <button class="remove-btn" title="Remove" @click="removeExcluded(path)">
                <X :size="14" />
              </button>
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
            <button class="picker-btn" title="Browse…" @click="pickExcludedFolder">
              <FolderOpen :size="14" />
            </button>
            <button class="add-btn" :disabled="!newExcludedPath.trim()" @click="addExcluded">
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
            {{ isCreating ? "Creating…" : "Create Vault" }}
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
  width: 520px;
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
  font-family: "Inter", sans-serif;
}

.close-btn {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  font-size: 16px;
  padding: 4px;
  border-radius: 4px;
  transition:
    color 0.15s,
    background 0.15s;
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

.hint code {
  background: #1a1a1a;
  padding: 1px 6px;
  border-radius: 4px;
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
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #ccc;
  border-radius: 6px;
  padding: 8px 10px;
  font-size: 12px;
  outline: none;
  font-family: monospace;
  transition: border-color 0.15s;
}

.path-input:focus {
  border-color: #22c55e;
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

.slider-row input[type="range"] {
  flex: 1;
  height: 4px;
  -webkit-appearance: none;
  appearance: none;
  background: #2a2a2a;
  border-radius: 2px;
  outline: none;
  cursor: pointer;
}

.slider-row input[type="range"]::-webkit-slider-thumb {
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

.slider-row input[type="range"]::-moz-range-thumb {
  width: 12px;
  height: 12px;
  background: #22c55e;
  border: none;
  border-radius: 2px;
  cursor: pointer;
}

.slider-row input[type="range"]::-webkit-slider-runnable-track {
  height: 4px;
  background: #2a2a2a;
  border-radius: 2px;
}

.slider-row input[type="range"]::-moz-range-track {
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

/* ── Create error ───────────────────────────────────────────────── */
.create-error {
  margin: 0;
  color: #ef4444;
  font-size: 12px;
  background: rgba(239, 68, 68, 0.08);
  padding: 8px 10px;
  border-radius: 6px;
  border: 1px solid rgba(239, 68, 68, 0.15);
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

.cancel-btn,
.create-btn {
  border: none;
  border-radius: 6px;
  padding: 8px 18px;
  font-size: 12px;
  font-weight: 500;
  font-family: "Inter", sans-serif;
  cursor: pointer;
  transition: all 0.15s;
}

.cancel-btn {
  background: #1a1a1a;
  color: #999;
  border: 1px solid #2a2a2a;
}

.cancel-btn:hover {
  background: #222;
  color: #e8e8e8;
}

.create-btn {
  background: #22c55e;
  color: #000;
}

.create-btn:hover:not(:disabled) {
  background: #16a34a;
}

.create-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
