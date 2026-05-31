<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="preview-modal">
      <button class="close-btn" @click="$emit('close')">✕</button>
      <img v-if="imageUrl" :src="imageUrl" class="preview-img" alt="Full preview" />
      <div v-else class="no-preview">No preview available</div>
      <div class="nav-buttons">
        <button @click="navigate(-1)">←</button>
        <button @click="navigate(1)">→</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { usePreviewStore } from '../../stores/preview'
import { useFilesStore } from '../../stores/files'

const previewStore = usePreviewStore()
const filesStore = useFilesStore()
const emit = defineEmits<{ close: [] }>()

const imageUrl = computed(() => {
  const file = previewStore.currentFile
  return file ? `/api/original/${file.id}` : ''
})

const navigate = (direction: number) => {
  if (!previewStore.currentFile) return
  const index = filesStore.files.findIndex(f => f.id === previewStore.currentFile!.id)
  const nextIndex = (index + direction + filesStore.files.length) % filesStore.files.length
  previewStore.setFile(filesStore.files[nextIndex])
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'ArrowLeft') navigate(-1)
  else if (e.key === 'ArrowRight') navigate(1)
  else if (e.key === 'Escape') emit('close')
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.9);
  display: flex; align-items: center; justify-content: center; z-index: 200;
}
.preview-modal { position: relative; max-width: 90vw; max-height: 90vh; }
.preview-img { max-width: 100%; max-height: 85vh; object-fit: contain; }
.no-preview { color: #555; font-size: 18px; padding: 60px; }
.close-btn {
  position: absolute; top: -40px; right: 0; background: none; border: none;
  color: #fff; font-size: 24px; cursor: pointer;
}
.nav-buttons { display: flex; justify-content: center; gap: 20px; margin-top: 12px; }
.nav-buttons button {
  background: rgba(255,255,255,0.1); border: none; color: #fff;
  width: 40px; height: 40px; border-radius: 50%; cursor: pointer; font-size: 18px;
}
</style>
