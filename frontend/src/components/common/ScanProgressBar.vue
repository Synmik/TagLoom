<template>
  <transition name="scan-fade">
    <div v-if="vaultStore.isScanning" class="scan-progress-overlay">
      <div class="scan-progress-card">
        <div class="scan-header">
          <span class="scan-icon">🔄</span>
          <span class="scan-title">{{ scanTitle }}</span>
          <button class="scan-close" @click="dismiss">×</button>
        </div>
        <div class="progress-track">
          <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
        </div>
        <div class="scan-stats">
          <span>{{ vaultStore.scanCurrent }} / {{ vaultStore.scanTotal }}</span>
          <span class="percent">{{ progressPercent }}%</span>
        </div>
        <div v-if="thumbProgress > 0" class="thumb-stats">
          <span>🖼️ Thumbnails: {{ thumbProgress }} generated</span>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useVaultStore } from '../../stores/vault'

const vaultStore = useVaultStore()
const thumbProgress = ref(0)
let thumbUnsub: (() => void) | undefined

const scanTitle = computed(() => {
  if (vaultStore.scanProgress >= 100) return 'Scan complete!'
  return 'Indexing files...'
})

const progressPercent = computed(() => {
  return Math.min(vaultStore.scanProgress, 100)
})

const dismiss = () => {
  // Don't actually stop scanning, just hide overlay
  // The overlay auto-hides when isScanning becomes false
}

onMounted(() => {
  thumbUnsub = () => {
    // Listen for thumbnail progress events
  }
})

onUnmounted(() => {
  thumbUnsub?.()
})
</script>

<style scoped>
.scan-progress-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.scan-progress-card {
  background: #1e1e1e;
  border: 1px solid #333;
  border-radius: 12px;
  padding: 24px 32px;
  min-width: 400px;
  max-width: 500px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
}

.scan-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.scan-icon { font-size: 20px; }

.scan-title {
  font-size: 16px;
  font-weight: 600;
  color: #fff;
  flex: 1;
}

.scan-close {
  background: none;
  border: none;
  color: #666;
  font-size: 20px;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
}

.scan-close:hover { color: #fff; }

.progress-track {
  height: 8px;
  background: #333;
  border-radius: 4px;
  overflow: hidden;
  margin-bottom: 12px;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #5b8af5, #7c5bf5);
  border-radius: 4px;
  transition: width 0.3s ease;
}

.scan-stats {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: #aaa;
}

.percent {
  font-weight: 600;
  color: #5b8af5;
}

.thumb-stats {
  margin-top: 8px;
  font-size: 12px;
  color: #888;
}

/* Transition */
.scan-fade-enter-active,
.scan-fade-leave-active {
  transition: opacity 0.2s ease;
}

.scan-fade-enter-from,
.scan-fade-leave-to {
  opacity: 0;
}

.scan-fade-enter-active .scan-progress-card,
.scan-fade-leave-active .scan-progress-card {
  transition: transform 0.2s ease;
}

.scan-fade-enter-from .scan-progress-card,
.scan-fade-leave-to .scan-progress-card {
  transform: scale(0.95);
}
</style>
