<template>
  <Teleport to="body">
    <transition name="context-fade">
      <div
        v-if="visible"
        ref="menuEl"
        class="context-menu"
        :style="positionStyle"
        @click.stop
      >
        <template v-for="(item, i) in items" :key="i">
          <div
            v-if="item.type === 'item'"
            class="menu-item"
            :class="{ disabled: item.disabled }"
            @click="handleClick(item)"
          >
            <span v-if="item.icon" class="menu-icon">{{ item.icon }}</span>
            <span class="menu-label">{{ item.label }}</span>
          </div>
          <div v-else-if="item.type === 'divider'" class="menu-divider" />
        </template>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import type { ContextMenuItem } from '../../composables/useContextMenu'

const props = defineProps<{
  visible: boolean
  x: number
  y: number
  items: ContextMenuItem[]
}>()

const emit = defineEmits<{ close: [] }>()

const menuEl = ref<HTMLElement | null>(null)

const positionStyle = computed(() => {
  return {
    left: `${props.x}px`,
    top: `${props.y}px`,
  }
})

async function handleClick(item: ContextMenuItem) {
  if (item.type === 'divider' || item.disabled) return
  if (item.action) {
    await item.action()
  }
  emit('close')
}

function onGlobalClick(e: MouseEvent) {
  if (menuEl.value && !menuEl.value.contains(e.target as Node)) {
    emit('close')
  }
}

function onGlobalKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close')
  }
}

watch(
  () => props.visible,
  (v) => {
    if (v) {
      document.addEventListener('click', onGlobalClick)
      document.addEventListener('keydown', onGlobalKeyDown)
    } else {
      document.removeEventListener('click', onGlobalClick)
      document.removeEventListener('keydown', onGlobalKeyDown)
    }
  }
)

onUnmounted(() => {
  document.removeEventListener('click', onGlobalClick)
  document.removeEventListener('keydown', onGlobalKeyDown)
})
</script>

<style scoped>
.context-menu {
  position: fixed;
  z-index: 10000;
  min-width: 180px;
  background: #252525;
  border: 1px solid #444;
  border-radius: 6px;
  padding: 4px 0;

}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 12px;
  color: #ddd;
  user-select: none;
}

.menu-item:hover:not(.disabled) {
  background: #5b8af5;
  color: #fff;
}

.menu-item.disabled {
  opacity: 0.35;
  cursor: default;
}

.menu-icon {
  width: 16px;
  text-align: center;
  flex-shrink: 0;
  font-size: 13px;
}

.menu-label {
  flex: 1;
}

.menu-divider {
  height: 1px;
  background: #444;
  margin: 4px 8px;
}

/* ── Transition ── */
.context-fade-enter-active,
.context-fade-leave-active {
  transition: opacity 0.1s ease;
}

.context-fade-enter-from,
.context-fade-leave-to {
  opacity: 0;
}
</style>
