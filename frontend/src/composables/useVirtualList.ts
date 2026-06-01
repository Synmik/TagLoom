import { ref, onMounted, onUnmounted, type Ref, type ComputedRef, computed, nextTick } from 'vue'
import type { File } from '../types/file'

export interface VirtualItem {
  index: number
  item: File
  style: string
}

export interface UseVirtualListReturn {
  visibleItems: ComputedRef<VirtualItem[]>
  totalHeight: ComputedRef<number>
  setContainerHeight: (h: number) => void
  attachScroll: (el: HTMLElement | null) => () => void
}

export function useVirtualList(
  files: Ref<File[]>,
  rowHeight: number = 44,
  overscan: number = 5,
  horizontalPadding: number = 12,
): UseVirtualListReturn {
  const scrollY = ref(0)
  const containerHeight = ref(600)

  const totalHeight = computed(() => files.value.length * rowHeight)

  const visibleItems = computed<VirtualItem[]>(() => {
    const count = files.value.length
    if (count === 0) return []

    const startIndex = Math.floor(scrollY.value / rowHeight)
    const visibleCount = Math.ceil(containerHeight.value / rowHeight)

    const start = Math.max(0, startIndex - overscan)
    const end = Math.min(count, startIndex + visibleCount + overscan)

    const items: VirtualItem[] = []
    for (let i = start; i < end; i++) {
      items.push({
        index: i,
        item: files.value[i],
        style: `position:absolute;top:${i * rowHeight}px;left:${horizontalPadding}px;right:${horizontalPadding}px;height:${rowHeight}px;`,
      })
    }
    return items
  })

  const onScroll = (e: Event) => {
    scrollY.value = (e.target as HTMLElement).scrollTop
  }

  const onResize = () => {
    // Will be called with the actual container element
  }

  let scrollCleanup: (() => void) | undefined

  onMounted(() => {
    window.addEventListener('resize', onResize)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', onResize)
    scrollCleanup?.()
  })

  return {
    visibleItems,
    totalHeight,
    setContainerHeight: (h: number) => { containerHeight.value = h },
    attachScroll: (el: HTMLElement | null) => {
      scrollCleanup?.()
      if (el) {
        el.addEventListener('scroll', onScroll, { passive: true })
        // Measure initial height
        containerHeight.value = el.clientHeight
      }
      return () => { el?.removeEventListener('scroll', onScroll) }
    },
  }
}
