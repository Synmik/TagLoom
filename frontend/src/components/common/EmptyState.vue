<template>
  <div class="empty-state">
    <div class="empty-icon">{{ icon }}</div>
    <div class="empty-content">
      <p class="empty-title">{{ title }}</p>
      <p class="empty-description">{{ description }}</p>
    </div>
    <button v-if="actionText" class="empty-action" :class="actionVariant" @click="$emit('action')">
      {{ actionText }}
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  icon: string
  title: string
  description: string
  actionText?: string
  actionVariant?: 'primary' | 'secondary'
}>()

defineEmits<{
  action: []
}>()
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 200px;
  gap: 12px;
  padding: 32px;
}

.empty-icon {
  font-size: 48px;
  line-height: 1;
}

.empty-content {
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.empty-title {
  font-size: 15px;
  color: #999;
  margin: 0;
}

.empty-description {
  font-size: 13px;
  color: #666;
  margin: 0;
}

.empty-action {
  margin-top: 4px;
  border: none;
  border-radius: 6px;
  padding: 8px 20px;
  cursor: pointer;
  font-size: 13px;
  transition: background-color 0.15s ease;
}

.empty-action.primary {
  background: #5b8af5;
  color: #fff;
}

.empty-action.primary:hover {
  background: #4a7ae5;
}

.empty-action.secondary {
  background: #2a2a2a;
  color: #ccc;
  border: 1px solid #444;
}

.empty-action.secondary:hover {
  background: #333;
  border-color: #555;
}
</style>
