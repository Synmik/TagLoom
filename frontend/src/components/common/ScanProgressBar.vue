<template>
  <transition name="scan-fade">
    <div v-if="vaultStore.isScanning" class="scan-progress-overlay">
      <div class="scan-progress-card">
        <div class="scan-header">
          <RefreshCw :size="20" class="scan-icon" />
          <span class="scan-title">{{ scanTitle }}</span>
          <button class="scan-close" @click="dismiss"><X :size="20" /></button>
        </div>
        <div class="progress-track">
          <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
        </div>
        <div class="scan-stats">
          <span>{{ statsText }}</span>
          <span v-if="!vaultStore.scanProgressUnknown" class="percent">{{ progressPercent }}%</span>
        </div>
        <div v-if="thumbProgress > 0" class="thumb-stats">
          <span><ImageIcon :size="14" class="inline-icon" /> Thumbnails: {{ thumbProgress }} generated</span>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { RefreshCw, X, ImagePlus as ImageIcon } from '@lucide/vue'
import { useVaultStore } from '../../stores/vault'

const vaultStore = useVaultStore()
const thumbProgress = ref(0)
let thumbUnsub: (() => void) | undefined

const scanTitle = computed(() => {
  if (vaultStore.scanProgress >= 100) return 'Scan complete!'
  if (vaultStore.scanProgressUnknown) return 'Scanning files...'
  return 'Indexing files...'
})

const progressPercent = computed(() => {
  return Math.min(vaultStore.scanProgress, 100)
})

const statsText = computed(() => {
  if (vaultStore.scanProgressUnknown) {
    return `${vaultStore.scanCurrent} files found…`
  }
  return `${vaultStore.scanCurrent} / ${vaultStore.scanTotal}`
})

const dismiss = () => {
  // Hide the overlay — scanning continues in the background
  vaultStore.isScanning = false
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
  background: #111111;
  border: 1px solid #1a1a1a;
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

.scan-icon { color: #22c55e; }

.scan-title {
  font-size: 16px;
  font-weight: 600;
  color: #e8e8e8;
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

.scan-close:hover { color: #e8e8e8; }

.progress-track {
  height: 6px;
  background: #1a1a1a;
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 12px;
}

.progress-fill {
  height: 100%;
  background: #22c55e;
  border-radius: 3px;
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
  color: #22c55e;
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
