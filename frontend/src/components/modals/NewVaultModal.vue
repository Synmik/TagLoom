<script setup lang="ts">
import { ref } from "vue";
import { FolderOpen } from "@lucide/vue";
import { SelectFolder, CreateVault } from "../../api/backend";
import { useVaultStore } from "../../stores/vault";
import { useToast } from "../../composables/useToast";
import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
import ModalShell from "../common/ModalShell.vue";
import QualitySlider from "../common/QualitySlider.vue";
import ExcludedFoldersEditor from "../common/ExcludedFoldersEditor.vue";

const vaultStore = useVaultStore();
const { success, error: toastError } = useToast();
const emit = defineEmits<{ close: [] }>();

// ── Form state ─────────────────────────────────────────────────────
const selectedPath = ref("");
const thumbnailQuality = ref(80);
const excludedFolders = ref<string[]>([]);
const addError = ref("");
const isCreating = ref(false);
const createError = ref("");
const excludedEditor = ref<InstanceType<typeof ExcludedFoldersEditor>>();

// ── Folder picker ──────────────────────────────────────────────────
const pickFolder = async () => {
  const dir = await SelectFolder();
  if (dir) {
    selectedPath.value = dir;
    createError.value = "";
  }
};

// ── Excluded folders (local only until vault is created) ──────────
const addExcluded = (path: string) => {
  if (excludedFolders.value.includes(path)) {
    addError.value = "Already in list";
    return;
  }
  addError.value = "";
  excludedFolders.value.push(path);
  excludedEditor.value?.clearInput();
};

const removeExcluded = (path: string) => {
  excludedFolders.value = excludedFolders.value.filter((p) => p !== path);
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
</script>

<template>
  <ModalShell title="New Vault" width="520px" @close="emit('close')">
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
      <QualitySlider v-model="thumbnailQuality" />
    </section>

    <!-- Excluded folders -->
    <section class="section">
      <h4>Excluded Folders</h4>
      <p class="hint">These folders will be skipped during indexing.</p>
      <ExcludedFoldersEditor
        ref="excludedEditor"
        :folders="excludedFolders"
        :error="addError"
        @add="addExcluded"
        @remove="removeExcluded"
      />
    </section>

    <!-- Error -->
    <p v-if="createError" class="create-error">{{ createError }}</p>

    <!-- Actions -->
    <div class="actions">
      <button class="cancel-btn" @click="emit('close')">Cancel</button>
      <button
        class="create-btn"
        :disabled="isCreating || !selectedPath.trim()"
        @click="createVault"
      >
        {{ isCreating ? "Creating…" : "Create Vault" }}
      </button>
    </div>
  </ModalShell>
</template>

<style scoped>
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
