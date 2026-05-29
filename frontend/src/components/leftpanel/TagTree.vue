<template>
  <div class="tag-tree">
    <TagNode
      v-for="tag in rootTags"
      :key="tag.id"
      :tag="tag"
      @edit="emit('edit', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import TagNode from './TagNode.vue'
import { useTagsStore } from '../../stores/tags'
import type { Tag } from '../../types/tag'

const emit = defineEmits<{ edit: [tag: Tag] }>()

const tagsStore = useTagsStore()

const rootTags = computed(() => {
  return tagsStore.tags.filter(t => !t.parent_id)
    .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name))
})
</script>

<style scoped>
.tag-tree { font-size: 13px; }
</style>
