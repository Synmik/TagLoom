<script setup lang="ts">
import { ref, onMounted } from "vue";
import { Check, FolderOpen } from "@lucide/vue";
import {
  GetAppSettings,
  SetAppSettings,
  SelectFolder,
  GetRecentVaults,
  RemoveRecentVault,
} from "../../api/backend";
import { useToast } from "../../composables/useToast";
import type { RecentVault } from "../../types/vault";
import { useVaultStore } from "../../stores/vault";
import ModalShell from "../common/ModalShell.vue";
import RecentVaultsList from "../common/RecentVaultsList.vue";
import { logger } from "../../utils/logger";

const vaultStore = useVaultStore();

defineEmits<{ close: [] }>();

const { success, error: toastError } = useToast();

// ── Settings state ────────────────────────────────────────────────
const lastVaultPath = ref("");
const autoOpenLastVault = ref(true);
const confirmBeforeExit = ref(false);

const isSaving = ref(false);
const isLoaded = ref(false);

// ── Recent vaults ─────────────────────────────────────────────────
const recentVaults = ref<RecentVault[]>([]);

const loadRecentVaults = async () => {
  recentVaults.value = await GetRecentVaults();
};

const openRecentVault = async (vault: RecentVault) => {
  await vaultStore.openVault(vault.path);
};

const removeRecentVault = async (path: string) => {
  try {
    await RemoveRecentVault(path);
    await loadRecentVaults();
  } catch (e: any) {
    toastError("Failed to remove vault: " + (e.message || String(e)));
  }
};

// ── Load settings on mount ────────────────────────────────────────
onMounted(async () => {
  try {
    const settings = await GetAppSettings();
    if (settings) {
      lastVaultPath.value = settings.last_vault_path || "";
      autoOpenLastVault.value = settings.auto_open_last_vault ?? true;
      confirmBeforeExit.value = settings.confirm_before_exit ?? false;
    }
  } catch (e) {
    logger.warn("AppSettingsModal.load", e);
  }
  await loadRecentVaults();
  isLoaded.value = true;
});

// ── Save ──────────────────────────────────────────────────────────
const save = async () => {
  isSaving.value = true;
  try {
    await SetAppSettings({
      last_vault_path: lastVaultPath.value,
      recent_vaults: recentVaults.value,
      auto_open_last_vault: autoOpenLastVault.value,
      confirm_before_exit: confirmBeforeExit.value,
    } as any);
    success("Settings saved");
  } catch (e: any) {
    toastError("Failed to save settings: " + (e.message || String(e)));
  } finally {
    isSaving.value = false;
  }
};

// ── Pick last vault folder ────────────────────────────────────────
const pickLastVault = async () => {
  const dir = await SelectFolder();
  if (dir) lastVaultPath.value = dir;
};
</script>

<template>
  <ModalShell title="App Settings" width="440px" @close="$emit('close')">
    <!-- General -->
    <section v-if="isLoaded" class="section">
      <h4>General</h4>
      <label class="checkbox-row">
        <input v-model="autoOpenLastVault" type="checkbox" />
        <span>Auto-open last vault on startup</span>
      </label>
      <label class="checkbox-row">
        <input v-model="confirmBeforeExit" type="checkbox" />
        <span>Confirm before exiting the app</span>
      </label>
    </section>

    <!-- Last Vault -->
    <section v-if="isLoaded" class="section">
      <h4>Last Opened Vault</h4>
      <div class="path-row">
        <input v-model="lastVaultPath" placeholder="No vault set…" class="form-input" readonly />
        <button class="picker-btn" title="Browse…" @click="pickLastVault">
          <FolderOpen :size="14" />
        </button>
      </div>
      <p class="hint">Auto-opened when the app starts (if enabled above).</p>
    </section>

    <!-- Recent Vaults -->
    <section v-if="isLoaded" class="section">
      <h4>Recent Vaults ({{ recentVaults.length }})</h4>
      <RecentVaultsList
        :vaults="recentVaults"
        @open="openRecentVault"
        @remove="removeRecentVault"
      />
    </section>

    <!-- Actions -->
    <div v-if="isLoaded" class="actions">
      <button class="save-btn" :disabled="isSaving" @click="save">
        <Check v-if="!isSaving" :size="14" />
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

.checkbox-row input[type="checkbox"] {
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
  font-family: "Inter", sans-serif;
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
</style>
