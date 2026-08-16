<template>
  <div class="quick-actions">
    <button class="action-btn" title="New Vault" @click="onNewVault">
      <Plus :size="15" />
    </button>

    <!-- Vault switcher dropdown -->
    <div ref="dropdownRef" v-click-outside="closeDropdown" class="vault-switcher">
      <button
        class="action-btn vault-btn"
        :class="{ active: dropdownOpen }"
        title="Open Vault / Recent Vaults"
        @click="toggleDropdown"
      >
        <FolderOpen :size="15" />
      </button>
      <div v-if="dropdownOpen" class="dropdown-menu">
        <button class="dropdown-item" @click="onOpenVault">
          <FolderOpen :size="14" />
          <span>Browse…</span>
        </button>
        <div v-if="recentVaults.length > 0" class="dropdown-divider"></div>
        <button
          v-for="vault in recentVaults"
          :key="vault.path"
          class="dropdown-item recent-item"
          @click="openRecent(vault)"
        >
          <FolderOpen :size="12" class="recent-icon" />
          <div class="recent-text">
            <span class="recent-name">{{ vault.name }}</span>
            <span class="recent-path">{{ vault.path }}</span>
          </div>
        </button>
        <div v-if="recentVaults.length > 0" class="dropdown-divider"></div>
        <button class="dropdown-item" @click="onAppSettings">
          <Settings :size="14" />
          <span>Manage in Settings…</span>
        </button>
      </div>
    </div>

    <button class="action-btn" title="Rescan Folder" @click="onRescanVault">
      <RefreshCw :size="15" />
    </button>
    <button class="action-btn" title="Full Scan" @click="onFullScan">
      <FolderSearch :size="15" />
    </button>
    <button class="action-btn" title="Vault Settings" @click="onVaultSettings">
      <FolderCog :size="15" />
    </button>
    <button class="action-btn" title="App Settings" @click="onAppSettings">
      <Settings :size="15" />
    </button>
    <button class="action-btn" title="Tag Manager" @click="onTagManager">
      <Tags :size="15" />
    </button>
    <button class="action-btn" title="About TagLoom" @click="onAbout">
      <Info :size="15" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import {
  Plus,
  FolderOpen,
  RefreshCw,
  FolderSearch,
  Settings,
  FolderCog,
  Tags,
  Info,
} from "@lucide/vue";
import { useVaultStore } from "../../stores/vault";
import { useUIStore } from "../../stores/ui";
import { GetRecentVaults } from "../../api/backend";
import type { RecentVault } from "../../types/vault";
import { logger } from "../../utils/logger";

const vaultStore = useVaultStore();
const uiStore = useUIStore();

const dropdownOpen = ref(false);
const dropdownRef = ref<HTMLElement | null>(null);
const recentVaults = ref<RecentVault[]>([]);

const toggleDropdown = async () => {
  if (!dropdownOpen.value) {
    await loadRecentVaults();
  }
  dropdownOpen.value = !dropdownOpen.value;
};

const closeDropdown = () => {
  dropdownOpen.value = false;
};

const loadRecentVaults = async () => {
  try {
    recentVaults.value = await GetRecentVaults();
  } catch (e) {
    logger.warn("QuickActions.loadRecent", e);
  }
};

const openRecent = async (vault: RecentVault) => {
  dropdownOpen.value = false;
  await vaultStore.openVault(vault.path);
};

const onNewVault = () => uiStore.openNewVault();
const onOpenVault = () => {
  dropdownOpen.value = false;
  vaultStore.pickAndOpenVault();
};
const onRescanVault = () => vaultStore.rescanVault();
const onFullScan = () => vaultStore.scanVault();
const onVaultSettings = () => uiStore.openVaultSettings();
const onAppSettings = () => {
  dropdownOpen.value = false;
  uiStore.openAppSettings();
};
const onTagManager = () => uiStore.openTagManager();
const onAbout = () => uiStore.openAbout();
</script>

<style scoped>
.quick-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  border-radius: 4px;
  transition:
    color 0.15s,
    background 0.15s;
}

.action-btn:hover {
  background: #1a1a1a;
  color: #e8e8e8;
}

/* ── Vault switcher dropdown ────────────────────────────────────── */
.vault-switcher {
  position: relative;
}

.vault-btn.active {
  background: #1a1a1a;
  color: #22c55e;
}

.dropdown-menu {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  background: #111111;
  border: 1px solid #2a2a2a;
  border-radius: 8px;
  padding: 4px;
  min-width: 280px;
  max-width: 400px;
  max-height: 320px;
  overflow-y: auto;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
  z-index: 200;
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 10px;
  background: none;
  border: none;
  color: #ccc;
  font-size: 12px;
  font-family: "Inter", sans-serif;
  cursor: pointer;
  border-radius: 4px;
  text-align: left;
  transition: background 0.1s;
}

.dropdown-item:hover {
  background: #1a1a1a;
  color: #e8e8e8;
}

.dropdown-divider {
  height: 1px;
  background: #2a2a2a;
  margin: 4px 8px;
}

.recent-item {
  gap: 6px;
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
</style>
