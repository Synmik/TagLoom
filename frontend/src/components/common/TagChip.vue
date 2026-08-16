<template>
  <span class="tag-chip" :style="{ background: chipColor }">
    <span class="tag-text">{{ displayName }}</span>
    <button class="remove-btn" @click.stop="$emit('remove', tag.id)"><X :size="14" /></button>
  </span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { X } from "@lucide/vue";
import { useTagsStore } from "../../stores/tags";
import type { Tag } from "../../types/tag";

const props = defineProps<{ tag: Tag }>();
defineEmits<{ remove: [tagId: number] }>();

const tagsStore = useTagsStore();

const displayName = computed(() => {
  if (!props.tag.parent_id) return props.tag.name;
  const parent = tagsStore.tags.find((t) => t.id === props.tag.parent_id);
  if (!parent) return props.tag.name;
  return `${props.tag.name} (${parent.name})`;
});

const chipColor = computed(() => {
  if (props.tag.color) return props.tag.color + "33"; // 20% opacity
  return "#2a2a2a";
});
</script>

<style scoped>
.tag-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 8px;
  border-radius: 10px;
  font-size: 12px;
  color: #ddd;
  min-height: 26px;
}
.tag-text {
}
.remove-btn {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0;
  transition: color 0.15s;
}
.remove-btn:hover {
  color: #ef4444;
}
</style>
