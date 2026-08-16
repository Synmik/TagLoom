<script setup lang="ts">
defineProps<{
  label?: string;
}>();

const modelValue = defineModel<number>({ required: true });
</script>

<template>
  <div class="slider-row">
    <label>{{ label ?? "Quality" }}</label>
    <input v-model.number="modelValue" type="range" min="10" max="100" step="5" />
    <span class="slider-value">{{ modelValue }}%</span>
  </div>
</template>

<style scoped>
.slider-row {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: #ccc;
}

.slider-row label {
  min-width: 60px;
  font-size: 12px;
}

.slider-row input[type="range"] {
  flex: 1;
  height: 4px;
  -webkit-appearance: none;
  appearance: none;
  background: #2a2a2a;
  border-radius: 2px;
  outline: none;
  cursor: pointer;
}

.slider-row input[type="range"]::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 12px;
  height: 12px;
  margin-top: -4px;
  background: #22c55e;
  border: none;
  border-radius: 2px;
  cursor: pointer;
}

.slider-row input[type="range"]::-moz-range-thumb {
  width: 12px;
  height: 12px;
  background: #22c55e;
  border: none;
  border-radius: 2px;
  cursor: pointer;
}

.slider-row input[type="range"]::-webkit-slider-runnable-track {
  height: 4px;
  background: #2a2a2a;
  border-radius: 2px;
}

.slider-row input[type="range"]::-moz-range-track {
  height: 4px;
  background: #2a2a2a;
  border-radius: 2px;
}

.slider-value {
  min-width: 40px;
  text-align: right;
  color: #22c55e;
  font-weight: 600;
  font-size: 12px;
}
</style>
