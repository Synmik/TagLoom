import { ref, onMounted, onUnmounted, type Ref, type ComputedRef, computed } from "vue";
import type { File } from "../types/file";

export interface VirtualCell {
  index: number;
  file: File;
  row: number;
  column: number;
  style: string;
}

export interface UseVirtualGridReturn {
  visibleCells: ComputedRef<VirtualCell[]>;
  totalHeight: ComputedRef<number>;
  numColumns: ComputedRef<number>;
  onScroll: (e: Event) => void;
  setContainerSize: (width: number, height: number) => void;
  attachScroll: (el: HTMLElement | null) => () => void;
  scrollTo: (y: number) => void;
}

/**
 * Virtual scrolling composable for a thumbnail grid.
 *
 * Layout model:
 *  - Fixed cell width (cellSize) with uniform gap between cells.
 *  - Each cell has a square thumbnail (aspect-ratio: 1) + a filename label (~28px).
 *  - Rows have a fixed estimated height derived from cellSize + label area.
 *  - Only cells in the visible viewport (+ overscan) are rendered as DOM nodes.
 */
export function useVirtualGrid(
  files: Ref<File[]>,
  cellSize: Ref<number>,
  gap: number = 8,
  overscan: number = 3,
  padding: number = 12,
  _totalCount?: Ref<number>, // kept for signature compat
): UseVirtualGridReturn {
  const scrollY = ref(0);
  const containerWidth = ref(800);
  const containerHeight = ref(600);

  // Number of columns that fit in the current container width (minus padding).
  const numColumns = computed(() => {
    const w = containerWidth.value - padding * 2;
    const s = cellSize.value;
    if (w <= 0 || s <= 0) return 1;
    return Math.max(1, Math.floor((w + gap) / (s + gap)));
  });

  // Row height = cellSize (square thumbnail) + filename label area.
  const rowHeight = computed(() => cellSize.value + 28);

  // Use only loaded files for scroll height so the scrollbar reflects what's
  // actually in memory. This prevents the user from overscrolling into unloaded
  // territory and ensures the infinite-scroll sentinel stays reachable.
  const logicalCount = computed(() => files.value.length);

  const numRows = computed(() => {
    const count = logicalCount.value;
    if (count === 0) return 0;
    return Math.ceil(count / numColumns.value);
  });

  const totalHeight = computed(() => numRows.value * rowHeight.value + padding * 2);

  const visibleCells = computed<VirtualCell[]>(() => {
    const count = files.value.length;
    if (count === 0) return [];

    const rows = numRows.value;
    const cols = numColumns.value;
    const rh = rowHeight.value;

    if (rows === 0) return [];

    // Clamp startRow so we never end up past the last row — prevents
    // the grid going blank when scrollY is stale after a layout change
    // (e.g. grid size, container resize, or page load).
    const startRow = Math.min(Math.floor(scrollY.value / rh), Math.max(0, rows - 1));
    const visibleRowCount = Math.ceil(containerHeight.value / rh);

    const start = Math.max(0, startRow - overscan);
    const end = Math.min(rows, startRow + visibleRowCount + overscan);

    const cells: VirtualCell[] = [];
    const s = cellSize.value;

    for (let row = start; row < end; row++) {
      const firstIndex = row * cols;
      const lastInRow = Math.min(firstIndex + cols, count);

      for (let col = 0; col < lastInRow - firstIndex; col++) {
        const idx = firstIndex + col;
        const file = files.value[idx];
        if (!file) continue;

        const left = padding + col * (s + gap);
        const topOffset = padding + row * rh;
        cells.push({
          index: idx,
          file,
          row,
          column: col,
          style: `position:absolute;top:${topOffset}px;left:${left}px;width:${s}px;`,
        });
      }
    }
    return cells;
  });

  const onScroll = (e: Event) => {
    scrollY.value = (e.target as HTMLElement).scrollTop;
  };

  const setContainerSize = (width: number, height: number) => {
    containerWidth.value = width;
    containerHeight.value = height;
  };

  const attachScroll = (el: HTMLElement | null) => {
    if (el) {
      el.addEventListener("scroll", onScroll, { passive: true });
      // Measure initial size
      containerWidth.value = el.clientWidth;
      containerHeight.value = el.clientHeight;
    }
    return () => {
      el?.removeEventListener("scroll", onScroll);
    };
  };

  const onResize = () => {
    // Triggered by window resize — the component should call setContainerSize
  };

  const scrollTo = (y: number) => {
    scrollY.value = y;
  };

  onMounted(() => {
    window.addEventListener("resize", onResize);
  });

  onUnmounted(() => {
    window.removeEventListener("resize", onResize);
  });

  return {
    visibleCells,
    totalHeight,
    numColumns,
    onScroll,
    setContainerSize,
    attachScroll,
    scrollTo,
  };
}
