<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { Pencil, Wrench, Trash2 } from "@lucide/vue";
import { useVaultStore } from "../../stores/vault";
import { useFilesStore } from "../../stores/files";
import { useToast } from "../../composables/useToast";
import ConfirmDialog from "../common/ConfirmDialog.vue";
import ModalShell from "../common/ModalShell.vue";
import QualitySlider from "../common/QualitySlider.vue";
import ExcludedFoldersEditor from "../common/ExcludedFoldersEditor.vue";
import { EventsOn } from "../../../wailsjs/runtime/runtime";
import {
  GetExcludedFolders,
  AddExcludedFolder,
  RemoveExcludedFolder,
  RepairAllThumbnails,
  CleanupOrphanThumbnails,
} from "../../api/backend";
import { logger } from "../../utils/logger";

const vaultStore = useVaultStore();
const filesStore = useFilesStore();
const { success, error: toastError } = useToast();
defineEmits<{ close: [] }>();

// ── Vault info ─────────────────────────────────────────────────────
const vaultName = computed(() => vaultStore.config?.name ?? vaultStore.currentVault?.name ?? "—");
const vaultPath = computed(() => vaultStore.currentVault?.path ?? "—");
const fileCount = computed(() => vaultStore.currentVault?.file_count ?? "—");
const editingName = ref(false);
const newName = ref("");

const startEditingName = () => {
  newName.value = vaultStore.config?.name ?? vaultStore.currentVault?.name ?? "";
  editingName.value = true;
};

const cancelEditingName = () => {
  editingName.value = false;
  newName.value = "";
};

// ── Excluded folders (loaded from backend) ──────────────────────────
const excludedFolders = ref<string[]>([]);
const addError = ref("");
const isAdding = ref(false);
const excludedEditor = ref<InstanceType<typeof ExcludedFoldersEditor>>();

const loadExcludedFolders = async () => {
  try {
    const result = await GetExcludedFolders();
    excludedFolders.value = Array.isArray(result) ? result : [];
  } catch (e) {
    logger.error("VaultSettings.loadExcluded", e);
  }
};

const addExcluded = async (path: string) => {
  isAdding.value = true;
  addError.value = "";
  try {
    await AddExcludedFolder(path);
    excludedEditor.value?.clearInput();
    await loadExcludedFolders();
    success(`Excluded "${path}"`);
  } catch (e: any) {
    addError.value = e.message || "Failed to add excluded folder";
    toastError(addError.value);
  } finally {
    isAdding.value = false;
  }
};

const removeExcluded = async (path: string) => {
  try {
    await RemoveExcludedFolder(path);
    await loadExcludedFolders();
    success(`Un-excluded "${path}"`);
  } catch (e: any) {
    void e; // error details logged by backend
    toastError("Failed to remove excluded folder");
  }
};

// ── Thumbnails (quality + repair + cleanup) ────────────────────────
const thumbnailQuality = ref(80);
const isRepairing = ref(false);
const repairProgress = ref<{ current: number; total: number } | null>(null);
const showRepairConfirm = ref(false);

const confirmRepairAll = async () => {
  showRepairConfirm.value = false;
  isRepairing.value = true;
  repairProgress.value = null;
  try {
    const progressUnsub = EventsOn("thumb:progress", (data: { current: number; total: number }) => {
      repairProgress.value = data;
    });
    const completeUnsub = EventsOn(
      "thumb:complete",
      (data: { generated: number; failed: number; total: number }) => {
        if (data.failed > 0) {
          toastError(
            `Thumbnail repair: ${data.generated} fixed, ${data.failed} failed out of ${data.total}`,
          );
        } else {
          success(`Thumbnail repair complete — ${data.generated}/${data.total} checked`);
        }
      },
    );
    try {
      await RepairAllThumbnails();
    } finally {
      progressUnsub();
      completeUnsub();
    }
    // Re-render the gallery so freshly fixed thumbnails load
    await filesStore.reloadFiles();
  } catch (e: any) {
    toastError("Thumbnail repair failed: " + (e.message || String(e)));
  } finally {
    isRepairing.value = false;
    repairProgress.value = null;
  }
};

// ── Clean up orphaned thumbnails ───────────────────────────────────
const isCleaning = ref(false);
const showCleanupConfirm = ref(false);

const confirmCleanupOrphans = async () => {
  showCleanupConfirm.value = false;
  isCleaning.value = true;
  try {
    const removed = await CleanupOrphanThumbnails();
    if (removed > 0) {
      success(`Removed ${removed} orphan thumbnail${removed === 1 ? "" : "s"}`);
    } else {
      success("No orphaned thumbnails found");
    }
  } catch (e: any) {
    toastError("Thumbnail cleanup failed: " + (e.message || String(e)));
  } finally {
    isCleaning.value = false;
  }
};

// ── Rescan ─────────────────────────────────────────────────────────
const isRescanning = ref(false);
const showRescanConfirm = ref(false);

const rescanVault = () => {
  showRescanConfirm.value = true;
};

const confirmRescan = async () => {
  showRescanConfirm.value = false;
  isRescanning.value = true;
  try {
    await vaultStore.rescanVault();
    success("Vault re-scan complete");
  } catch (e: any) {
    toastError("Re-scan failed: " + (e.message || String(e)));
  } finally {
    isRescanning.value = false;
  }
};

// ── Save config (quality + auto-tag + name) ────────────────────────
const isSaving = ref(false);

const saveNameAndSettings = async () => {
  if (!vaultStore.config) return;
  isSaving.value = true;
  const wasEditingName = editingName.value;
  try {
    vaultStore.config.settings.thumbnail_quality = thumbnailQuality.value;
    // Update vault name if editing
    const newNameTrimmed = newName.value.trim();
    if (wasEditingName && newNameTrimmed) {
      vaultStore.config.name = newNameTrimmed;
      // Also update currentVault so TitleBar reflects the change
      if (vaultStore.currentVault) {
        vaultStore.currentVault.name = newNameTrimmed;
      }
      editingName.value = false;
    }
    await vaultStore.saveConfig(vaultStore.config);
    success("Settings saved");
  } catch (e: any) {
    const msg = e.message || String(e);
    if (wasEditingName) {
      toastError("Failed to rename vault: " + msg);
    } else {
      toastError("Failed to save settings");
    }
  } finally {
    isSaving.value = false;
  }
};

// ── Lifecycle ──────────────────────────────────────────────────────
onMounted(async () => {
  await loadExcludedFolders();
  // Initialise settings from current config
  if (vaultStore.config?.settings) {
    thumbnailQuality.value = vaultStore.config.settings.thumbnail_quality ?? 80;
  }
});
</script>

<template>
  <ConfirmDialog
    v-if="showCleanupConfirm"
    message="Remove orphaned thumbnails? WebP files whose source file is no longer in the vault will be deleted from disk."
    confirm-text="Clean up"
    @confirm="confirmCleanupOrphans"
    @cancel="showCleanupConfirm = false"
  />
  <ConfirmDialog
    v-if="showRepairConfirm"
    message="Repair all thumbnails? This re-checks every file in the vault and regenerates missing or stale thumbnails. Large vaults may take a while."
    confirm-text="Repair"
    @confirm="confirmRepairAll"
    @cancel="showRepairConfirm = false"
  />
  <ConfirmDialog
    v-if="showRescanConfirm"
    message="Re-scan the vault? This will detect added/removed files."
    confirm-text="Re-scan"
    @confirm="confirmRescan"
    @cancel="showRescanConfirm = false"
  />
  <ModalShell title="Vault Settings" @close="$emit('close')">
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
              autofocus
              @keydown.enter="saveNameAndSettings()"
              @keydown.escape="cancelEditingName()"
            />
          </template>
          <template v-else>
            <span class="value">{{ vaultName }}</span>
            <button class="edit-btn" title="Rename" @click="startEditingName">
              <Pencil :size="12" />
            </button>
          </template>
        </div>
      </div>
      <div class="info-row">
        <span class="label">Path</span>
        <span class="value path">{{ vaultPath }}</span>
      </div>
      <div class="info-row">
        <span class="label">Files Indexed</span>
        <span class="value">{{ fileCount }}</span>
      </div>
    </section>

    <!-- Thumbnails -->
    <section class="section">
      <h4>Thumbnails</h4>
      <QualitySlider v-model="thumbnailQuality" />
      <div class="thumb-actions">
        <button
          class="repair-btn"
          :disabled="isRepairing || isRescanning"
          @click="showRepairConfirm = true"
        >
          <Wrench :size="13" />
          {{ isRepairing ? "Repairing…" : "Repair all thumbnails" }}
        </button>
        <button
          class="repair-btn"
          :disabled="isCleaning || isRepairing || isRescanning"
          @click="showCleanupConfirm = true"
        >
          <Trash2 :size="13" />
          {{ isCleaning ? "Cleaning up…" : "Clean up orphaned thumbnails" }}
        </button>
      </div>
      <p v-if="repairProgress" class="repair-progress">
        {{ repairProgress.current }} / {{ repairProgress.total }}
      </p>
      <p class="hint">
        Repair fixes missing or stale thumbnails (old vaults with out-of-date paths). Clean up
        removes WebP files whose source file is gone.
      </p>
    </section>

    <!-- Excluded folders -->
    <section class="section">
      <h4>Excluded Folders</h4>
      <p class="hint">Folders listed here are skipped during indexing.</p>
      <ExcludedFoldersEditor
        ref="excludedEditor"
        :folders="excludedFolders"
        :error="addError"
        :busy="isAdding"
        @add="addExcluded"
        @remove="removeExcluded"
      />
    </section>

    <!-- Actions -->
    <div class="actions">
      <button class="rescan-btn" :disabled="isRescanning" @click="rescanVault">
        {{ isRescanning ? "Re-scanning…" : "Re-scan Vault" }}
      </button>
      <button class="save-btn" :disabled="isSaving" @click="saveNameAndSettings">
        {{ isSaving ? "Saving…" : "Save Settings" }}
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
  font-family: "Inter", sans-serif;
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

/* ── Repair / cleanup thumbnails ────────────────────────────────── */
.thumb-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.repair-btn {
  align-self: flex-start;
  display: flex;
  align-items: center;
  gap: 6px;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #999;
  border-radius: 6px;
  padding: 6px 12px;
  font-size: 12px;
  font-family: "Inter", sans-serif;
  cursor: pointer;
  transition: all 0.15s;
}

.repair-btn:hover:not(:disabled) {
  background: #222;
  border-color: #333;
  color: #e8e8e8;
}

.repair-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.repair-progress {
  margin: 0;
  color: #22c55e;
  font-size: 11px;
  font-weight: 600;
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
  font-family: "Inter", sans-serif;
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
