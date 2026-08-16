import { ref } from "vue";

export type ToastType = "success" | "error" | "info";

export interface ToastMessage {
  id: number;
  type: ToastType;
  message: string;
}

// Shared state — single source of truth across the app
const toasts = ref<ToastMessage[]>([]);
let nextId = 0;

const defaultDuration = 3500;

/**
 * Composable that provides a simple toast notification API.
 * Safe to call from any component or store — returns the same shared state.
 */
export function useToast() {
  /** Show a toast and auto-dismiss after `duration` ms. */
  const show = (message: string, type: ToastType = "info", duration = defaultDuration) => {
    const id = ++nextId;
    toasts.value.push({ id, type, message });

    if (duration > 0) {
      setTimeout(() => dismiss(id), duration);
    }
  };

  /** Dismiss a specific toast by id. */
  const dismiss = (id: number) => {
    const index = toasts.value.findIndex((t) => t.id === id);
    if (index !== -1) {
      toasts.value.splice(index, 1);
    }
  };

  /** Show a success toast. */
  const success = (message: string, duration?: number) => show(message, "success", duration);

  /** Show an error toast. */
  const error = (message: string, duration?: number) => show(message, "error", duration);

  /** Show an info toast. */
  const info = (message: string, duration?: number) => show(message, "info", duration);

  /** Clear all toasts. */
  const clear = () => {
    toasts.value = [];
  };

  return {
    toasts,
    show,
    dismiss,
    success,
    error,
    info,
    clear,
  };
}
