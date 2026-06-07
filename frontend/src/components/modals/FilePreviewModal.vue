<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="preview-modal">
      <button class="close-btn" @click="$emit('close')"><X :size="24" /></button>

      <!-- Image preview -->
      <img v-if="isImage" :src="mediaUrl" class="preview-img" alt="Full preview" />

      <!-- Video player -->
      <div v-else-if="isVideo" class="video-wrapper">
        <video
          ref="videoEl"
          :src="mediaUrl"
          class="preview-video"
          @loadedmetadata="onVideoLoaded"
          @click="togglePlayPause"
          @timeupdate="onTimeUpdate"
          @ended="onVideoEnded"
          @progress="onProgress"
        />
        <div v-if="!isPlaying" class="play-overlay" @click="togglePlayPause">
          <PlayCircle :size="64" class="play-icon" />
        </div>
        <div class="video-controls">
          <div class="progress-bar" @click="seekVideo($event)">
            <div class="progress-buffered" :style="{ width: bufferedPercent + '%' }"></div>
            <div class="progress-filled" :style="{ width: progressPercent + '%' }"></div>
          </div>
          <div class="controls-row">
            <button class="ctrl-btn" @click="togglePlayPause" :title="isPlaying ? 'Pause' : 'Play'">
              <Pause v-if="isPlaying" :size="18" />
              <PlayCircle v-else :size="18" />
            </button>
            <span class="time-display">{{ currentTime }} / {{ totalTime }}</span>
            <div class="ctrl-spacer"></div>
            <div class="volume-group" @mouseenter="showVolumeSlider = true" @mouseleave="showVolumeSlider = false">
              <button class="ctrl-btn" @click="toggleMute" :title="isMuted ? 'Unmute' : 'Mute'">
                <Volume2 v-if="!isMuted && vol > 0.5" :size="18" />
                <Volume1 v-else-if="!isMuted && vol > 0" :size="18" />
                <VolumeX v-else :size="18" />
              </button>
              <div class="volume-slider-wrap" :class="{ visible: showVolumeSlider || !isMuted }">
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.01"
                  :value="isMuted ? 0 : vol"
                  @input="onVolumeChange($event)"
                  class="volume-slider"
                />
              </div>
            </div>
            <button class="ctrl-btn" @click="togglePiP" title="Picture in Picture">
              <PictureInPicture2 :size="18" />
            </button>
            <button class="ctrl-btn" @click="toggleFullscreen" title="Fullscreen">
              <Maximize :size="18" />
            </button>
          </div>
        </div>
      </div>

      <!-- Fallback -->
      <div v-else class="no-preview">No preview available</div>

      <div class="nav-buttons">
        <button @click="navigate(-1)"><ChevronLeft :size="18" /></button>
        <button @click="navigate(1)"><ChevronRight :size="18" /></button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { X, ChevronLeft, ChevronRight, PlayCircle, Pause, Volume2, Volume1, VolumeX, Maximize, PictureInPicture2, Volume } from '@lucide/vue'
import { usePreviewStore } from '../../stores/preview'
import { useFilesStore } from '../../stores/files'

const previewStore = usePreviewStore()
const filesStore = useFilesStore()
const emit = defineEmits<{ close: [] }>()

const videoEl = ref<HTMLVideoElement | null>(null)
const isPlaying = ref(false)
const isMuted = ref(true)
const vol = ref(0)
const showVolumeSlider = ref(false)
const currentTimeVal = ref(0)
const duration = ref(0)
const bufferedPercent = ref(0)

const imageExtensions = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg', '.avif'])
const videoExtensions = new Set(['.mp4', '.mov', '.avi', '.webm', '.mkv', '.wmv', '.flv', '.m4v', '.3gp', '.3g2', '.vob', '.ogv', '.mpg', '.mpeg', '.m2v', '.ts', '.mts', '.m2ts', '.asf', '.rm', '.amv', '.f4v', '.dv', '.mxf'])

const fileExt = computed(() => {
  const file = previewStore.currentFile
  if (!file) return ''
  return file.vault_path.split('.').pop()?.toLowerCase() || ''
})

const isImage = computed(() => imageExtensions.has('.' + fileExt.value))
const isVideo = computed(() => videoExtensions.has('.' + fileExt.value))

const mediaUrl = computed(() => {
  const file = previewStore.currentFile
  return file ? `/api/original/${file.id}` : ''
})

watch(() => previewStore.currentFile, () => {
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
  if (!isMuted.value) {
    if (vol.value === 0) {
      vol.value = 1
      videoEl.value.volume = 1
    }
  }
}

const onVolumeChange = (event: Event) => {
  if (!videoEl.value) return
  const target = event.target as HTMLInputElement
  const newVol = parseFloat(target.value)
  vol.value = newVol
  videoEl.value.volume = newVol
  if (newVol > 0) {
    isMuted.value = false
    videoEl.value.muted = false
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

const toggleFullscreen = () => {
  if (!videoEl.value) return
  if (document.fullscreenElement) {
    document.exitFullscreen()
  } else {
    videoEl.value.requestFullscreen()
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

const navigate = (direction: number) => {
  if (!previewStore.currentFile) return
  const index = filesStore.files.findIndex(f => f.id === previewStore.currentFile!.id)
  const nextIndex = (index + direction + filesStore.files.length) % filesStore.files.length
  previewStore.setFile(filesStore.files[nextIndex])
}
</script>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.92);
  display: flex; align-items: center; justify-content: center; z-index: 200;
}
.preview-modal { position: relative; max-width: 90vw; max-height: 90vh; }
.preview-img { max-width: 100%; max-height: 85vh; object-fit: contain; display: block; }
.no-preview { color: #555; font-size: 18px; padding: 60px; text-align: center; }

/* Close button */
.close-btn {
  position: absolute; top: -40px; right: 0; background: none; border: none;
  color: #fff; cursor: pointer; transition: opacity 0.15s; z-index: 10;
}
.close-btn:hover { opacity: 0.7; }

/* Video wrapper */
.video-wrapper {
  position: relative; max-width: 90vw; max-height: 80vh;
  background: #000; border-radius: 8px; overflow: hidden;
  display: flex; flex-direction: column;
}
.preview-video {
  max-width: 100%; max-height: 80vh; object-fit: contain;
  display: block; cursor: pointer;
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
  padding: 8px 10px 6px; position: absolute;
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

/* Volume slider */
.volume-group {
  display: flex;
  align-items: center;
  gap: 4px;
}

.volume-slider-wrap {
  height: 16px;
  display: flex;
  align-items: center;
  width: 0;
  overflow: hidden;
  transition: width 0.2s ease;
}

.volume-slider-wrap.visible {
  width: 80px;
}

.volume-slider {
  width: 80px;
  height: 4px;
  -webkit-appearance: none;
  appearance: none;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 2px;
  outline: none;
  cursor: pointer;
}

.volume-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 12px;
  height: 12px;
  margin-top: -4px;
  background: #22c55e;
  border: none;
  border-radius: 50%;
  cursor: pointer;
}

.volume-slider::-moz-range-thumb {
  width: 12px;
  height: 12px;
  background: #22c55e;
  border: none;
  border-radius: 50%;
  cursor: pointer;
}

/* Navigation */
.nav-buttons { display: flex; justify-content: center; gap: 20px; margin-top: 12px; }
.nav-buttons button {
  background: rgba(255,255,255,0.1); border: none; color: #fff;
  width: 40px; height: 40px; border-radius: 50%; cursor: pointer;
  transition: background 0.15s;
}
.nav-buttons button:hover { background: rgba(255,255,255,0.2); }
</style>
