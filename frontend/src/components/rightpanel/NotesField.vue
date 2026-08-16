<template>
  <section class="field-section">
    <label class="field-label">Notes</label>
    <textarea
      :value="localNotes"
      placeholder="Add notes…"
      class="field-textarea"
      rows="4"
      @input="onInput"
    />
  </section>
</template>

<script setup lang="ts">
import { shallowRef, watch } from "vue";
import { usePreviewStore } from "../../stores/preview";

const previewStore = usePreviewStore();

const editingFileId = shallowRef<number | null>(null);
const localNotes = shallowRef("");

watch(
  () => previewStore.currentFile,
  (file, prevFile) => {
    // Flush a pending (debounced) edit to the previous file before switching
    if (prevFile && localNotes.value !== (prevFile.notes || "")) {
      previewStore.updateFieldFor(prevFile, "notes", localNotes.value);
    }
    editingFileId.value = file?.id ?? null;
    localNotes.value = file?.notes || "";
  },
  { immediate: true },
);

function onInput(event: Event) {
  localNotes.value = (event.target as HTMLTextAreaElement).value;
}

// Debounced auto-save (500ms) — guarded by file ID + value comparison
watch(localNotes, (value, _prev, onCleanup) => {
  const currentFile = previewStore.currentFile;
  if (!currentFile) return;

  // If the value matches what's already in the store, this is a
  // programmatic reset (file switch), not a user edit — skip entirely.
  if (value === (currentFile.notes || "")) return;

  const savedFileId = currentFile.id;

  const timer = setTimeout(() => {
    // Guard: only save if we're still on the same file
    if (previewStore.currentFile?.id !== savedFileId) return;
    previewStore.updateNotes(value);
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
.field-textarea {
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #e8e8e8;
  border-radius: 6px;
  padding: 7px 10px;
  font-size: 13px;
  outline: none;
  resize: vertical;
  font-family: "Inter", sans-serif;
  transition: border-color 0.15s;
}
.field-textarea:focus {
  border-color: #22c55e;
}
.field-textarea::placeholder {
  color: #444;
}
</style>
