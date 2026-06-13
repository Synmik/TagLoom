<template>
  <div class="tag-node-wrapper">
    <div
      class="tag-node"
      :class="{ selected: isSelected }"
      @click="handleClick"
      @contextmenu.prevent="handleContextMenu"
    >
      <span v-if="hasChildren" class="arrow" @click.stop="toggleExpand">
        <ChevronDown v-if="isExpanded" :size="10" />
        <ChevronRight v-else :size="10" />
      </span>
      <span v-else class="arrow spacer"></span>
      <span class="color-dot" :style="{ background: tag.color || '#666' }"></span>
      <span class="tag-name">{{ tag.name }}</span>
      <span class="file-count">{{ displayCount }}</span>
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
import { ChevronDown, ChevronRight } from '@lucide/vue'
import { useFiltersStore } from '../../stores/filters'
import { useTagsStore } from '../../stores/tags'
import type { Tag } from '../../types/tag'

const props = defineProps<{ tag: Tag }>()
const emit = defineEmits<{ edit: [tag: Tag] }>()

const filtersStore = useFiltersStore()
const tagsStore = useTagsStore()
const isExpanded = ref(true)

const children = computed(() => {
  const parentId = props.tag.id
  return tagsStore.tags
    .filter(t => t.parent_id === parentId)
    .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name))
})

const hasChildren = computed(() => children.value.length > 0)

// Aggregate count: direct files + all descendant files
const displayCount = computed(() => {
  return tagsStore.getAggregateCount(props.tag.id)
})

// Check if this tag is currently selected
const isSelected = computed(() => {
  const groups = filtersStore.activeFilters.tagGroups
  if (groups.length === 0) return false
  // Check if any group contains this tag's ID
  return groups.some(group => group.includes(props.tag.id))
})

const toggleExpand = () => { isExpanded.value = !isExpanded.value }

const handleClick = (event: MouseEvent) => {
  const allDescendants = tagsStore.getAllDescendantIds(props.tag.id)
  const groupIds = allDescendants.length > 0 ? [props.tag.id, ...allDescendants] : [props.tag.id]
  const accumulate = event.ctrlKey || event.metaKey  // Ctrl on Windows, Cmd on Mac

  filtersStore.toggleTagFilter(props.tag.id, groupIds, accumulate)
  // Gallery.vue watcher handles reload automatically
}

const handleContextMenu = () => {
  emit('edit', props.tag)
}
</script>

<style scoped>
.tag-node-wrapper { }

.tag-node {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 10px; cursor: pointer; border-radius: 6px; margin: 1px 0;
  font-size: 13px;
}
.tag-node:hover { background: #1e1e1e; }
.tag-node.selected { background: #14532d; color: #e8e8e8; }

.arrow {
  font-size: 8px; color: #666; width: 10px; text-align: center; flex-shrink: 0;
  cursor: pointer;
}
.arrow.spacer { cursor: default; }
.color-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.tag-name { flex: 1; color: #ccc; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-count { font-size: 11px; color: #555; background: #1a1a1a; padding: 1px 6px; border-radius: 10px; }

.tag-children { padding-left: 16px; }
</style>
