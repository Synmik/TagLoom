<template>
  <div class="color-picker">
    <input type="color" v-model="selectedColor" @input="onChange" class="color-input" />
    <span class="color-preview" :style="{ background: selectedColor }"></span>
    <span class="color-hex">{{ selectedColor }}</span>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const selectedColor = ref(props.modelValue || '#888888')
const onChange = () => emit('update:modelValue', selectedColor.value)
</script>

<style scoped>
.color-picker { display: flex; align-items: center; gap: 8px; }
.color-input { width: 28px; height: 28px; border: 1px solid #2a2a2a; cursor: pointer; background: none; border-radius: 4px; padding: 0; }
.color-preview { width: 20px; height: 20px; border-radius: 4px; border: 1px solid #2a2a2a; }
.color-hex { color: #666; font-size: 12px; font-family: monospace; }
</style>
