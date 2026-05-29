<template>
  <div class="tag-node-wrapper">
    <div
      class="tag-node"
      :class="{ selected: filtersStore.activeFilters.tagIds.includes(tag.id) }"
      @click="toggleFilter"
      @contextmenu.prevent="handleContextMenu"
    >
      <span v-if="hasChildren" class="arrow" @click.stop="toggleExpand">{{ isExpanded ? '▼' : '▶' }}</span>
      <span v-else class="arrow spacer"></span>
      <span class="color-dot" :style="{ background: tag.color || '#666' }"></span>
      <span class="tag-name">{{ tag.name }}</span>
      <span class="file-count">{{ fileCount }}</span>
    </div>
    <div v-if="hasChildren && isExpanded" class="tag-children">
      <TagNode
        v-for="child in children"
        :key="child.id"
        :tag="child"
        @edit="emit('edit', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useFiltersStore } from '../../stores/filters'
import { useFilesStore } from '../../stores/files'
import { useTagsStore } from '../../stores/tags'
import type { Tag } from '../../types/tag'

const props = defineProps<{ tag: Tag }>()
const emit = defineEmits<{ edit: [tag: Tag] }>()

const filtersStore = useFiltersStore()
const filesStore = useFilesStore()
const tagsStore = useTagsStore()
const isExpanded = ref(true)

const children = computed(() => {
  const parentId = props.tag.id
  return tagsStore.tags
    .filter(t => t.parent_id === parentId)
    .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name))
})

const hasChildren = computed(() => children.value.length > 0)

const fileCount = computed(() => {
  return tagsStore.tagCounts[props.tag.id] ?? 0
})

const toggleExpand = () => { isExpanded.value = !isExpanded.value }

const toggleFilter = () => {
  filtersStore.toggleTagFilter(props.tag.id)
  filesStore.loadFiles(filtersStore.asBackendFilter, { field: 'indexed_at', order: 'desc' })
}

const handleContextMenu = () => {
  emit('edit', props.tag)
}
</script>

<style scoped>
.tag-node-wrapper { }

.tag-node {
  display: flex; align-items: center; gap: 4px;
  padding: 3px 8px; cursor: pointer; border-radius: 4px; margin: 1px 0;
}
.tag-node:hover { background: #2a2a2a; }
.tag-node.selected { background: #2a3a5a; }

.arrow {
  font-size: 8px; color: #888; width: 10px; text-align: center; flex-shrink: 0;
  cursor: pointer;
}
.arrow.spacer { cursor: default; }
.color-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.tag-name { flex: 1; color: #ddd; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 13px; }
.file-count { font-size: 11px; color: #666; }

.tag-children { padding-left: 16px; }
</style>
