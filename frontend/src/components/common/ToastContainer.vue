<script setup lang="ts">
import { useToast } from '../../composables/useToast'
import Toast from './Toast.vue'

const { toasts, dismiss } = useToast()
</script>

<template>
  <div class="toast-container">
    <transition-group name="toast-list">
      <Toast
        v-for="toast in toasts"
        :key="toast.id"
        :toast="toast"
        @dismiss="dismiss"
      />
    </transition-group>
  </div>
</template>

<style scoped>
.toast-container {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 8px;
  pointer-events: none;
}

.toast-container > * {
  pointer-events: auto;
}

/* ── Transition-group animations ────────────────────────────────── */
.toast-list-enter-active {
  animation: toast-in 0.25s ease-out;
}

.toast-list-leave-active {
  animation: toast-out 0.2s ease-in forwards;
}

@keyframes toast-in {
  from {
    opacity: 0;
    transform: translateX(20px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes toast-out {
  from {
    opacity: 1;
    transform: translateX(0);
  }
  to {
    opacity: 0;
    transform: translateX(20px);
  }
}
</style>
