<template>
  <section class="field-section">
    <label class="field-label">Name</label>
    <input :value="localName" placeholder="Enter name…" class="field-input" @input="onInput" />
  </section>
</template>

<script setup lang="ts">
import { shallowRef, watch } from "vue";
import { usePreviewStore } from "../../stores/preview";

const previewStore = usePreviewStore();

// Track which file we are currently editing so debounce doesn't save to the wrong file
const editingFileId = shallowRef<number | null>(null);
const localName = shallowRef("");

// Watch for file changes — reset local state and mark editing file
watch(
  () => previewStore.currentFile,
  (file) => {
    editingFileId.value = file?.id ?? null;
    localName.value = file?.name || "";
  },
  { immediate: true },
);

// Use @input instead of v-model so we can distinguish user edits from resets
function onInput(event: Event) {
  localName.value = (event.target as HTMLInputElement).value;
}

// Debounced auto-save (500ms) — guarded by file ID + value comparison
watch(localName, (value, _prev, onCleanup) => {
  const currentFile = previewStore.currentFile;
  if (!currentFile) return;

  // If the value matches what's already in the store, this is a
  // programmatic reset (file switch), not a user edit — skip entirely.
  if (value === (currentFile.name || "")) return;

  const savedFileId = currentFile.id;

  const timer = setTimeout(() => {
    // Guard: only save if we're still on the same file
    if (previewStore.currentFile?.id !== savedFileId) return;
    previewStore.updateName(value);
  }, 500);
  onCleanup(() => clearTimeout(timer));
});
</script>

<style scoped>
.field-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field-label {
  color: #666;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.field-input {
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #e8e8e8;
  border-radius: 6px;
  padding: 7px 10px;
  font-size: 13px;
  font-family: "Inter", sans-serif;
  outline: none;
  transition: border-color 0.15s;
}
.field-input:focus {
  border-color: #22c55e;
}
.field-input::placeholder {
  color: #444;
}
</style>
