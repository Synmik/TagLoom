<script setup lang="ts">
import { FolderOpen, Trash2 } from "@lucide/vue";
import type { RecentVault } from "../../types/vault";

defineProps<{
  vaults: RecentVault[];
}>();

const emit = defineEmits<{
  open: [vault: RecentVault];
  remove: [path: string];
}>();

const formatDate = (iso: string) => {
  try {
    const d = new Date(iso);
    return (
      d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" }) +
      " " +
      d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })
    );
  } catch {
    return iso;
  }
};
</script>

<template>
  <div v-if="vaults.length === 0" class="empty-recent">
    <p>No recent vaults.</p>
  </div>
  <div v-else class="recent-list">
    <div v-for="vault in vaults" :key="vault.path" class="recent-item" @click="emit('open', vault)">
      <div class="recent-info">
        <FolderOpen :size="14" class="recent-icon" />
        <div class="recent-text">
          <span class="recent-name">{{ vault.name }}</span>
          <span class="recent-path">{{ vault.path }}</span>
        </div>
      </div>
      <div class="recent-actions">
        <span class="recent-date">{{ formatDate(vault.opened_at) }}</span>
        <button
          class="remove-btn"
          title="Remove from list"
          @click.stop="emit('remove', vault.path)"
        >
          <Trash2 :size="12" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
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
