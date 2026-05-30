<template>
  <div class="thumbnail-slider">
    <span class="label">Size</span>
    <input
      type="range"
      min="0"
      max="2"
      step="1"
      :value="sizeIndex"
      @input="onChange($event)"
      class="slider"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useUIStore } from '../../stores/ui'
import { useFilesStore } from '../../stores/files'
import { useVaultStore } from '../../stores/vault'

const uiStore = useUIStore()
const filesStore = useFilesStore()
const vaultStore = useVaultStore()

const sizes = ['small', 'medium', 'large'] as const
const sizeIndex = computed(() => sizes.indexOf(uiStore.gridSize))

const onChange = (e: Event) => {
  const index = Number((e.target as HTMLInputElement).value)
  const size = sizes[index] as 'small' | 'medium' | 'large'
  uiStore.setGridSize(size)
  vaultStore.persistGridSize(size)
  filesStore.loadFiles()
}
</script>

<style scoped>
.thumbnail-slider { display: flex; align-items: center; gap: 6px; }
.label { color: #888; font-size: 11px; }
.slider { width: 80px; accent-color: #5b8af5; }
</style>
