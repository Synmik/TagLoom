<template>
  <div class="tag-tree">
    <template v-for="category in categories" :key="category.name">
      <TagCategory :category="category" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import TagCategory from './TagCategory.vue'
import { useTagsStore } from '../../stores/tags'
import type { Tag } from '../../types/tag'

const tagsStore = useTagsStore()
onMounted(() => tagsStore.loadTags())

const categories = computed(() => {
  const metaTags: Tag[] = []
  const colorTags: Tag[] = []
  const userTags: Tag[] = []

  for (const tag of tagsStore.tags) {
    if (tag.name.includes('★') || tag.name.includes('♥')) {
      metaTags.push(tag)
    } else if (tag.color || tag.name.match(/^(Red|Blue|Green|Yellow|Purple|Black|White)/)) {
      colorTags.push(tag)
    } else {
      userTags.push(tag)
    }
  }

  return [
    { name: 'Meta Tags', tags: metaTags },
    { name: 'Colors', tags: colorTags },
    { name: 'Tags', tags: userTags },
  ]
})
</script>

<style scoped>
.tag-tree { font-size: 13px; }
</style>
