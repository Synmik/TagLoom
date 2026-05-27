<template>
  <div class="star-rating">
    <span
      v-for="i in 5"
      :key="i"
      class="star"
      :class="{ filled: i <= rating }"
      @click="$emit('change', i)"
    >
      ★
    </span>
  </div>
</template>

<script setup lang="ts">
defineProps<{ rating: number }>()
defineEmits<{ change: [rating: number] }>()
</script>

<style scoped>
.star-rating { display: flex; gap: 2px; cursor: pointer; }
.star { font-size: 18px; color: #444; transition: color 0.15s; }
.star.filled { color: #f5c542; }
.star:hover { color: #f5c542; }
</style>
