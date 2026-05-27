<template>
  <span class="tag-chip" :style="{ background: chipColor }">
    <span class="tag-text">{{ tag.name }}</span>
    <button class="remove-btn" @click="$emit('remove', tag.id)">×</button>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Tag } from '../../types/tag'

const props = defineProps<{ tag: Tag }>()
defineEmits<{ remove: [tagId: number] }>()

const chipColor = computed(() => {
  if (props.tag.color) return props.tag.color + '33' // 20% opacity
  return '#2a2a2a'
})
</script>

<style scoped>
.tag-chip {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 2px 6px; border-radius: 12px; font-size: 12px; color: #ddd;
}
.tag-text { }
.remove-btn {
  background: none; border: none; color: #888; cursor: pointer;
  font-size: 14px; line-height: 1; padding: 0;
}
.remove-btn:hover { color: #ff4444; }
</style>
