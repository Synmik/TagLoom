<template>
  <section class="metadata-section">
    <label class="field-label">Metadata</label>
    <div class="metadata-grid">
      <div class="meta-row">
        <span class="meta-label">Filename</span
        ><span
          class="meta-value"
          :class="{ truncated: displayFilenameTruncated }"
          :title="displayFilenameTruncated ? meta?.filename : undefined"
          >{{ displayFilename }}</span
        >
      </div>
      <div class="meta-row">
        <span class="meta-label">Date Created</span
        ><span class="meta-value">{{ meta?.date_created || "—" }}</span>
      </div>
      <div class="meta-row">
        <span class="meta-label">Date Modified</span
        ><span class="meta-value">{{ meta?.date_modified || "—" }}</span>
      </div>
      <div class="meta-row">
        <span class="meta-label">File Size</span><span class="meta-value">{{ formatSize }}</span>
      </div>
      <div class="meta-row">
        <span class="meta-label">File Format</span
        ><span class="meta-value">{{ meta?.format_name || "—" }}</span>
      </div>
      <div v-if="showResolution" class="meta-row">
        <span class="meta-label">Resolution</span><span class="meta-value">{{ resolution }}</span>
      </div>
      <div v-if="showDuration" class="meta-row">
        <span class="meta-label">Duration</span><span class="meta-value">{{ duration }}</span>
      </div>
      <div v-if="meta?.page_count" class="meta-row">
        <span class="meta-label">Pages</span><span class="meta-value">{{ meta.page_count }}</span>
      </div>
      <div v-if="meta?.title" class="meta-row">
        <span class="meta-label">Title</span
        ><span class="meta-value truncated" :title="meta.title">{{ meta.title }}</span>
      </div>
      <div v-if="meta?.author" class="meta-row">
        <span class="meta-label">Author</span
        ><span class="meta-value truncated" :title="meta.author">{{ meta.author }}</span>
      </div>
    </div>

    <div class="rating-row">
      <span class="meta-label">Rating</span>
      <StarRating :rating="previewStore.currentFile?.rating || 0" @change="setRating" />
    </div>

    <div class="favorite-row">
      <button
        class="favorite-btn"
        :class="{ active: previewStore.currentFile?.is_favorite === 1 }"
        @click="toggleFavorite"
      >
        <Heart
          :fill="previewStore.currentFile?.is_favorite === 1 ? 'currentColor' : 'none'"
          :size="14"
          class="fav-icon"
        />
        Favorite
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { Heart } from "@lucide/vue";
import StarRating from "../common/StarRating.vue";
import { usePreviewStore } from "../../stores/preview";

const previewStore = usePreviewStore();
const meta = computed(() => previewStore.metadata);

const displayFilenameTruncated = computed(() => {
  const fn = meta.value?.filename;
  return fn && fn.length > 40;
});

const displayFilename = computed(() => {
  const fn = meta.value?.filename;
  if (!fn) return "—";
  if (fn.length <= 40) return fn;
  return fn.slice(0, 37) + "…";
});

const formatSize = computed(() => {
  if (!meta.value?.size_bytes) return "—";
  const bytes = meta.value.size_bytes;
  if (bytes > 1_000_000) return (bytes / 1_000_000).toFixed(1) + " MB";
  if (bytes > 1_000) return (bytes / 1_000).toFixed(1) + " KB";
  return bytes + " B";
});

// Rows a format cannot have are left out rather than filled with "—": a still
// image has no duration, a PDF or EPUB has no resolution. The category comes
// from the backend (utils.GetFileCategory), which already labels GIF and WebM
// files "animated".
const RESOLUTION_CATEGORIES = new Set(["image", "animated", "video"]);
const DURATION_CATEGORIES = new Set(["animated", "video"]);

const category = computed(() => meta.value?.category ?? "");

const hasResolution = computed(
  () => (meta.value?.resolution_width ?? 0) > 0 && (meta.value?.resolution_height ?? 0) > 0,
);
const hasDuration = computed(() => (meta.value?.duration_seconds ?? 0) > 0);

// An empty category means an older binding or an unknown file: fall back to
// showing the row whenever a value was actually read.
const showResolution = computed(
  () => hasResolution.value && (category.value === "" || RESOLUTION_CATEGORIES.has(category.value)),
);
const showDuration = computed(
  () => hasDuration.value && (category.value === "" || DURATION_CATEGORIES.has(category.value)),
);

const resolution = computed(
  () => `${meta.value?.resolution_width} × ${meta.value?.resolution_height}`,
);

const duration = computed(() => {
  const total = Math.floor(meta.value?.duration_seconds ?? 0);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const mm = h > 0 ? String(m).padStart(2, "0") : String(m);
  const ss = String(s).padStart(2, "0");
  return h > 0 ? `${h}:${mm}:${ss}` : `${mm}:${ss}`;
});

const setRating = (r: number) => previewStore.setRating(r);
const toggleFavorite = () => previewStore.toggleFavorite();
</script>

<style scoped>
.metadata-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.field-label {
  color: #666;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.metadata-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;
}
.meta-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.meta-label {
  color: #555;
  font-size: 10px;
}
.meta-value {
  color: #bbb;
  font-size: 12px;
}
.meta-value.truncated {
  overflow: hidden;
  white-space: nowrap;
  max-width: 100%;
  display: block;
}
.rating-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}
.favorite-row {
  margin-top: 4px;
}
.favorite-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  color: #888;
  border-radius: 6px;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 12px;
  font-family: "Inter", sans-serif;
  transition: all 0.15s;
}
.favorite-btn:hover {
  background: #222;
  color: #ccc;
}
.favorite-btn.active {
  color: #ef4444;
  border-color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
}
.fav-icon {
  flex-shrink: 0;
}
</style>
