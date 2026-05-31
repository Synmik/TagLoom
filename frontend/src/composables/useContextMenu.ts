import { ref } from 'vue'

export interface ContextMenuItem {
  type: 'item' | 'divider'
  label?: string
  icon?: string
  disabled?: boolean
  action?: () => void | Promise<void>
}

const visible = ref(false)
const x = ref(0)
const y = ref(0)
const items = ref<ContextMenuItem[]>([])

export function useContextMenu() {
  function open(event: MouseEvent, menuItems: ContextMenuItem[]) {
    event.preventDefault()
    event.stopPropagation()

    items.value = menuItems
    x.value = event.clientX
    y.value = event.clientY
    visible.value = true
  }

  function close() {
    visible.value = false
  }

  return { visible, x, y, items, open, close }
}
