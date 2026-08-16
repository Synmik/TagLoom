<script setup lang="ts">
import { ref } from "vue";
import { FolderOpen, Plus, X } from "@lucide/vue";
import { SelectFolder } from "../../api/backend";

defineProps<{
  /** Folder paths to display. */
  folders: string[];
  /** Inline error under the add row (e.g. backend rejection). */
  error?: string;
  /** Disable the add button while a backend add is in flight. */
  busy?: boolean;
}>();

const emit = defineEmits<{
  /** A path was typed/picked and the user asked to add it. */
  add: [path: string];
  /** The user removed a path from the list. */
  remove: [path: string];
}>();

const newPath = ref("");

const submit = () => {
  const path = newPath.value.trim();
  if (!path) return;
  emit("add", path);
};

// Exposed so parents can clear the input after a successful add
// (keeps a failed path in the field for retry).
defineExpose({ clearInput: () => (newPath.value = "") });

const pick = async () => {
  const dir = await SelectFolder();
  if (!dir) return;
  // Store the full absolute path — the backend normalises it
  newPath.value = dir;
};
</script>

<template>
  <div class="excluded-editor">
    <div v-if="folders.length" class="excluded-list">
      <div v-for="path in folders" :key="path" class="excluded-item">
        <span class="excluded-path">{{ path }}</span>
        <button class="remove-btn" title="Remove" @click="emit('remove', path)">
          <X :size="14" />
        </button>
      </div>
    </div>
    <p v-else class="empty-hint">No excluded folders</p>

    <div class="add-row">
      <input
        v-model="newPath"
        placeholder="Folder path…"
        class="form-input"
        @keydown.enter="submit"
      />
      <button class="picker-btn" title="Browse…" @click="pick">
        <FolderOpen :size="14" />
      </button>
      <button class="add-btn" :disabled="busy || !newPath.trim()" @click="submit">
        <Plus v-if="!busy" :size="14" />
      </button>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
  </div>
</template>

<style scoped>
.excluded-editor {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

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
</style>
