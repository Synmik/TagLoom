<template>
  <div
    class="tag-node"
    :class="{ selected: filtersStore.activeFilters.tagIds.includes(tag.id) }"
    @click="toggleFilter"
    @contextmenu.prevent="handleContextMenu"
  >
    <span class="color-dot" :style="{ background: tag.color || '#666' }"></span>
    <span class="tag-name">{{ displayName }}</span>
    <span class="file-count">{{ fileCount }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useFiltersStore } from '../../stores/filters'
import { useFilesStore } from '../../stores/files'
import { useTagsStore } from '../../stores/tags'
import type { Tag } from '../../types/tag'

const props = defineProps<{ tag: Tag }>()
const emit = defineEmits<{ edit: [tag: Tag] }>()

const filtersStore = useFiltersStore()
const filesStore = useFilesStore()
const tagsStore = useTagsStore()

const displayName = computed(() => {
  if (!props.tag.parent_id) return props.tag.name
  const parent = tagsStore.tags.find(t => t.id === props.tag.parent_id)
  if (!parent) return props.tag.name
  return `${props.tag.name} (${parent.name})`
})

const fileCount = computed(() => {
  return tagsStore.tagCounts[props.tag.id] ?? 0
})

const toggleFilter = () => {
  filtersStore.toggleTagFilter(props.tag.id)
  filesStore.loadFiles(filtersStore.asBackendFilter, { field: 'indexed_at', order: 'desc' })
}

const handleContextMenu = () => {
  emit('edit', props.tag)
}
</script>

<style scoped>
.tag-node {
  display: flex; align-items: center; gap: 6px;
  padding: 3px 8px; cursor: pointer; border-radius: 4px; margin: 1px 0;
}
.tag-node:hover { background: #2a2a2a; }
.tag-node.selected { background: #2a3a5a; }
.color-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.tag-name { flex: 1; color: #ddd; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-count { font-size: 11px; color: #666; }
</style>
