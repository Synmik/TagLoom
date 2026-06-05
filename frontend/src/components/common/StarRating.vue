<template>
  <div class="star-rating">
    <span
      v-for="i in 5"
      :key="i"
      class="star"
      :class="{ filled: i <= rating }"
      @click="$emit('change', i)"
    >
      <Star :size="15" :fill="i <= rating ? 'currentColor' : 'none'" />
    </span>
  </div>
</template>

<script setup lang="ts">
import { Star } from '@lucide/vue'
defineProps<{ rating: number }>()
defineEmits<{ change: [rating: number] }>()
</script>

<style scoped>
.star-rating { display: flex; gap: 2px; cursor: pointer; }
.star { font-size: 16px; color: #333; transition: color 0.15s; }
.star.filled { color: #C5AF22; }
.star:hover { color: #C5AF22; }
</style>
