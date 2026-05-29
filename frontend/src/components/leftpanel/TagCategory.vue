<template>
  <div class="tag-category">
    <div class="category-header" @click="toggle">
      <span class="arrow">{{ isExpanded ? '▼' : '▶' }}</span>
      <span class="category-name">{{ category.name }}</span>
    </div>
    <div v-if="isExpanded" class="category-tags">
      <TagNode v-for="tag in category.tags" :key="tag.id" :tag="tag" @edit="emit('edit', $event)" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import TagNode from './TagNode.vue'
import type { Tag } from '../../types/tag'

const props = defineProps<{
  category: { name: string; tags: Tag[] }
}>()
const emit = defineEmits<{ edit: [tag: Tag] }>()

const isExpanded = ref(true)
const toggle = () => { isExpanded.value = !isExpanded.value }
</script>

<style scoped>
.tag-category { margin-bottom: 4px; }
.category-header {
  display: flex; align-items: center; gap: 4px;
  padding: 4px 8px; cursor: pointer; color: #888; font-size: 11px; text-transform: uppercase;
}
.category-header:hover { color: #bbb; }
.arrow { font-size: 9px; }
.category-name { letter-spacing: 0.5px; }
.category-tags { padding-left: 8px; }
</style>
