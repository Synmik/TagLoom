<template>
  <section class="preview-section">
    <div class="preview-container" @dblclick="previewStore.openFullPreview">
      <!-- Image preview -->
      <img
        v-if="isImage"
        :src="imageUrl"
        alt="Preview"
        class="preview-media"
      />
      <!-- Video player -->
      <div v-else-if="isVideo" class="video-wrapper" @dblclick.stop>
        <video
          ref="videoEl"
          :src="videoUrl"
          preload="metadata"
          class="preview-media"
          @loadedmetadata="onVideoLoaded"
          @click="togglePlayPause"
          @timeupdate="onTimeUpdate"
          @ended="onVideoEnded"
          @progress="onProgress"
        />
        <!-- Play overlay when paused -->
        <div v-if="!isPlaying" class="play-overlay" @click="togglePlayPause">
          <PlayCircle :size="48" class="play-icon" />
        </div>
        <!-- Custom controls bar -->
        <div class="video-controls" @dblclick.stop>
          <div class="progress-bar" @click.stop="seekVideo($event)">
            <div class="progress-buffered" :style="{ width: bufferedPercent + '%' }"></div>
            <div class="progress-filled" :style="{ width: progressPercent + '%' }"></div>
          </div>
          <div class="controls-row">
            <button class="ctrl-btn" @click.stop="togglePlayPause" :title="isPlaying ? 'Pause' : 'Play'">
              <Pause v-if="isPlaying" :size="16" />
              <PlayCircle v-else :size="16" />
            </button>
            <span class="time-display">{{ currentTime }} / {{ totalTime }}</span>
            <div class="ctrl-spacer"></div>
            <button class="ctrl-btn" @click.stop="toggleMute" :title="isMuted ? 'Unmute' : 'Mute'">
              <Volume2 v-if="!isMuted && volume > 0.5" :size="16" />
              <Volume1 v-else-if="!isMuted && volume > 0" :size="16" />
              <VolumeX v-else :size="16" />
            </button>
            <button class="ctrl-btn" @click.stop="togglePiP" title="Picture in Picture">
              <PictureInPicture2 :size="16" />
            </button>
          </div>
        </div>
      </div>
      <!-- Fallback -->
      <span v-else class="no-preview">No preview</span>
    </div>
    <div class="format-info">
      <span class="format-badge">{{ formatName }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch, onUnmounted } from 'vue'
import { PlayCircle, Pause, Volume2, Volume1, VolumeX, PictureInPicture2 } from '@lucide/vue'
import { usePreviewStore } from '../../stores/preview'

const previewStore = usePreviewStore()

const videoEl = ref<HTMLVideoElement | null>(null)
const isPlaying = ref(false)
const isMuted = ref(true)
const volume = ref(0)
const currentTimeVal = ref(0)
const duration = ref(0)
const bufferedPercent = ref(0)

let watchTimer: ReturnType<typeof setTimeout> | null = null

watch(() => previewStore.currentFile, () => {
  // Reset state when file changes
  if (videoEl.value) {
    videoEl.value.pause()
    videoEl.value.currentTime = 0
  }
  isPlaying.value = false
  isMuted.value = true
  currentTimeVal.value = 0
  duration.value = 0
  bufferedPercent.value = 0
})

const togglePlayPause = () => {
  if (!videoEl.value) return
  if (isPlaying.value) {
    videoEl.value.pause()
  } else {
    videoEl.value.play()
  }
  isPlaying.value = !isPlaying.value
}

const onVideoLoaded = () => {
  if (videoEl.value) {
    duration.value = videoEl.value.duration
    isMuted.value = true
    videoEl.value.muted = true
  }
}

const onTimeUpdate = () => {
  if (videoEl.value) {
    currentTimeVal.value = videoEl.value.currentTime
  }
}

const onVideoEnded = () => {
  isPlaying.value = false
}

const onProgress = () => {
  if (!videoEl.value) return
  const buffer = videoEl.value.buffered
  if (buffer.length > 0 && duration.value > 0) {
    bufferedPercent.value = (buffer.end(buffer.length - 1) / duration.value) * 100
  }
}

const seekVideo = (event: MouseEvent) => {
  if (!videoEl.value || !duration.value) return
  const bar = event.currentTarget as HTMLDivElement
  const rect = bar.getBoundingClientRect()
  const ratio = Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width))
  videoEl.value.currentTime = ratio * duration.value
}

const toggleMute = () => {
  if (!videoEl.value) return
  isMuted.value = !isMuted.value
  videoEl.value.muted = isMuted.value
  if (!isMuted.value && volume.value === 0) {
    volume.value = 1
    videoEl.value.volume = 1
  }
}

const togglePiP = async () => {
  if (!videoEl.value) return
  try {
    if (document.pictureInPictureElement) {
      await document.exitPictureInPicture()
    } else {
      await videoEl.value.requestPictureInPicture()
    }
  } catch {
    // PiP not supported or denied
  }
}

const progressPercent = computed(() => {
  if (!duration.value) return 0
  return (currentTimeVal.value / duration.value) * 100
})

const currentTime = computed(() => formatTime(currentTimeVal.value))
const totalTime = computed(() => formatTime(duration.value))

const formatTime = (seconds: number): string => {
  if (!seconds || isNaN(seconds)) return '0:00'
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

const imageExtensions = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg', '.avif'])
const videoExtensions = new Set(['.mp4', '.mov', '.avi', '.webm', '.mkv', '.wmv', '.flv', '.m4v', '.3gp', '.3g2', '.vob', '.ogv', '.mpg', '.mpeg', '.m2v', '.ts', '.mts', '.m2ts', '.asf', '.rm', '.amv', '.f4v', '.dv', '.mxf'])

const fileExt = computed(() => {
  const file = previewStore.currentFile
  if (!file) return ''
  return file.vault_path.split('.').pop()?.toLowerCase() || ''
})

const isImage = computed(() => imageExtensions.has('.' + fileExt.value))
const isVideo = computed(() => videoExtensions.has('.' + fileExt.value))

const imageUrl = computed(() => {
  const file = previewStore.currentFile
  if (!file) return ''
  return `/api/original/${file.id}`
})

const videoUrl = computed(() => {
  const file = previewStore.currentFile
  if (!file) return ''
  return `/api/original/${file.id}`
})

const formatName = computed(() => {
  const file = previewStore.currentFile
  if (!file) return ''
  const ext = file.vault_path.split('.').pop()?.toUpperCase() || ''
  return ext === 'JPG' || ext === 'JPEG' ? 'JPEG' : ext
})
</script>

<style scoped>
.preview-section { text-align: center; }
.preview-container {
  aspect-ratio: 1; background: #0a0a0a; border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; overflow: hidden; position: relative;
}
.preview-media {
  max-width: 100%; max-height: 100%; object-fit: contain;
  display: block;
}

/* Video wrapper */
.video-wrapper {
  width: 100%; height: 100%; position: relative;
  display: flex; flex-direction: column; background: #000;
}
.video-wrapper .preview-media {
  width: 100%; height: 100%; object-fit: contain;
  cursor: pointer;
}

/* Play overlay */
.play-overlay {
  position: absolute; inset: 0;
  display: flex; align-items: center; justify-content: center;
  background: rgba(0, 0, 0, 0.3); cursor: pointer;
  transition: opacity 0.2s;
}
.play-overlay:hover { background: rgba(0, 0, 0, 0.45); }
.play-icon { color: rgba(255, 255, 255, 0.85); }

/* Custom controls bar */
.video-controls {
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.85));
  padding: 8px 8px 4px; position: absolute;
  bottom: 0; left: 0; right: 0;
}
.progress-bar {
  height: 4px; background: rgba(255, 255, 255, 0.15);
  border-radius: 2px; cursor: pointer; position: relative;
  margin-bottom: 6px; overflow: hidden;
  transition: height 0.15s;
}
.progress-bar:hover { height: 6px; }
.progress-buffered {
  position: absolute; top: 0; left: 0; height: 100%;
  background: rgba(255, 255, 255, 0.15); border-radius: 2px;
}
.progress-filled {
  position: absolute; top: 0; left: 0; height: 100%;
  background: #22c55e; border-radius: 2px;
  transition: width 0.1s linear;
}
.controls-row {
  display: flex; align-items: center; gap: 8px;
}
.ctrl-btn {
  background: none; border: none; color: #ccc;
  cursor: pointer; padding: 4px; display: flex;
  align-items: center; justify-content: center;
  transition: color 0.15s;
}
.ctrl-btn:hover { color: #fff; }
.time-display {
  font-size: 11px; color: #999; font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.ctrl-spacer { flex: 1; }

.no-preview { color: #444; font-size: 12px; }
.format-info { margin-top: 8px; }
.format-badge {
  background: #1a1a1a; color: #999; font-size: 10px;
  font-weight: 600;
  padding: 3px 10px; border-radius: 10px; text-transform: uppercase;
  letter-spacing: 0.3px;
}
</style>
