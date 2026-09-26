<template>
  <ConfirmDialog
    v-if="showConfirm"
    :message="confirmMessage"
    confirm-text="Delete"
    @confirm="confirmDelete"
    @cancel="showConfirm = false"
  />
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>{{ isEditing ? "Edit Tag" : "Create Tag" }}</h3>
        <button class="close-btn" @click="$emit('close')"><X :size="16" /></button>
      </div>
      <div class="modal-body">
        <TagDetailForm
          :tag="tag"
          :preset-parent-id="presetParentId"
          @saved="emit('close')"
          @delete-request="requestDelete"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { X } from "@lucide/vue";
import TagDetailForm from "./tagmanager/TagDetailForm.vue";
import ConfirmDialog from "../common/ConfirmDialog.vue";
import { useTagsStore } from "../../stores/tags";
import { useToast } from "../../composables/useToast";
import type { Tag } from "../../types/tag";

const props = defineProps<{
  /** Tag to edit. Omit (or null) to create a new tag. */
  tag?: Tag | null;
  /** Pre-selected parent when creating. */
  presetParentId?: number | null;
}>();

const emit = defineEmits<{ close: [] }>();

const tagsStore = useTagsStore();
const { success, error: toastError } = useToast();

const isEditing = computed(() => !!props.tag);
const showConfirm = ref(false);

const confirmMessage = computed(() => {
  const name = props.tag?.name ?? "";
  return `Delete tag "${name}"? Files will lose this tag.`;
});

// Clear the confirmation when the modal is reused for a different tag.
watch(
  () => props.tag,
  () => {
    showConfirm.value = false;
  },
);

const requestDelete = () => {
  if (!props.tag) return;
  showConfirm.value = true;
};

const confirmDelete = async () => {
  const tag = props.tag;
  if (!tag) return;
  showConfirm.value = false;
  try {
    await tagsStore.deleteTag(tag.id);
    success(`Tag "${tag.name}" deleted`);
  } catch (e: any) {
    toastError("Failed to delete tag: " + (e.message || String(e)));
    return;
  }
  emit("close");
};
</script>

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
  width: 360px;
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
}
</style>
