import type { Directive, DirectiveBinding } from 'vue'

// Extend HTMLElement to store the handler
declare global {
  interface HTMLElement {
    _clickOutsideHandler?: (event: MouseEvent) => void
  }
}

export const clickOutside: Directive<HTMLElement, (event?: MouseEvent) => void> = {
  mounted(el: HTMLElement, binding: DirectiveBinding<(event?: MouseEvent) => void>) {
    el._clickOutsideHandler = (event: MouseEvent) => {
      if (!el.contains(event.target as Node)) {
        binding.value?.(event)
      }
    }
    document.addEventListener('click', el._clickOutsideHandler)
  },
  unmounted(el: HTMLElement) {
    if (el._clickOutsideHandler) {
      document.removeEventListener('click', el._clickOutsideHandler)
    }
  },
}
