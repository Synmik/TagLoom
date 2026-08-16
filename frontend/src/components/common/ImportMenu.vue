<template>
  <Teleport to="body">
    <!-- Menu visibility is toggled by the parent via v-if on this component,
         so enter/leave animations run on mount/unmount — no inner toggle needed. -->
    <!-- eslint-disable vue/require-toggle-inside-transition -->
    <transition name="import-fade">
      <div ref="menuEl" class="import-menu" :style="positionStyle" @click.stop>
        <div class="import-menu-title">
          Import {{ files.length }} file{{ files.length > 1 ? "s" : "" }}
        </div>
        <div class="menu-item" @click="handleCopy">
          <Copy :size="14" class="menu-icon" />
          <span class="menu-label">Copy</span>
          <span class="menu-hint">Keep original</span>
        </div>
        <div class="menu-item" @click="handleMove">
          <Move :size="14" class="menu-icon" />
          <span class="menu-label">Move</span>
          <span class="menu-hint">Remove original</span>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch } from "vue";
import { Copy, Move } from "@lucide/vue";
import type { DroppedFile } from "../../composables/useDragDrop";

const props = defineProps<{
  files: DroppedFile[];
  x: number;
  y: number;
}>();

const emit = defineEmits<{
  close: [];
  copy: [];
  move: [];
}>();

const menuEl = ref<HTMLElement | null>(null);

// Clamp menu position within viewport
const positionStyle = computed(() => {
  const menuWidth = 220;
  const menuHeight = 120;
  let x = props.x;
  let y = props.y;

  // Clamp to viewport right edge
  if (x + menuWidth > window.innerWidth - 8) {
    x = window.innerWidth - menuWidth - 8;
  }
  // Clamp to viewport bottom edge
  if (y + menuHeight > window.innerHeight - 8) {
    y = window.innerHeight - menuHeight - 8;
  }
  // Ensure not off-screen left/top
  if (x < 8) x = 8;
  if (y < 8) y = 8;

  return { left: `${x}px`, top: `${y}px` };
});

function handleCopy() {
  emit("copy");
  emit("close");
}

function handleMove() {
  emit("move");
  emit("close");
}

function onGlobalClick(e: MouseEvent) {
  if (menuEl.value && !menuEl.value.contains(e.target as Node)) {
    emit("close");
  }
}

function onGlobalKeyDown(e: KeyboardEvent) {
  if (e.key === "Escape") {
    emit("close");
  }
}

watch(
  () => true,
  (v) => {
    if (v) {
      document.addEventListener("click", onGlobalClick);
      document.addEventListener("keydown", onGlobalKeyDown);
    } else {
      document.removeEventListener("click", onGlobalClick);
      document.removeEventListener("keydown", onGlobalKeyDown);
    }
  },
);

onMounted(() => {
  document.addEventListener("click", onGlobalClick);
  document.addEventListener("keydown", onGlobalKeyDown);
});

onUnmounted(() => {
  document.removeEventListener("click", onGlobalClick);
  document.removeEventListener("keydown", onGlobalKeyDown);
});
</script>

<style scoped>
.import-menu {
  position: fixed;
  z-index: 10001;
  min-width: 220px;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 8px;
  padding: 4px 0;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
}

.import-menu-title {
  padding: 6px 12px 4px;
  font-size: 11px;
  color: #888;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 600;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  font-size: 13px;
  color: #ccc;
  font-family: "Inter", sans-serif;
  user-select: none;
}

.menu-item:hover {
  background: #14532d;
  color: #e8e8e8;
}

.menu-icon {
  width: 16px;
  flex-shrink: 0;
  color: currentColor;
}

.menu-label {
  flex: 1;
  font-weight: 500;
}

.menu-hint {
  font-size: 11px;
  color: #888;
}

.menu-item:hover .menu-hint {
  color: #aaa;
}

/* ── Transition ── */
.import-fade-enter-active,
.import-fade-leave-active {
  transition: opacity 0.1s ease;
}

.import-fade-enter-from,
.import-fade-leave-to {
  opacity: 0;
}
</style>
