import { ref } from "vue";
import { SUPPORTED_EXTENSIONS } from "../types/file";
import { OnFileDrop, OnFileDropOff } from "../../wailsjs/runtime/runtime";

export interface DroppedFile {
  name: string;
  path: string;
}

export interface ImportMenuData {
  files: DroppedFile[];
  x: number;
  y: number;
}

/**
 * Uses Wails runtime OnFileDrop for drag-and-drop file import.
 *
 * Wails handles all native drag events at the window level.
 * The `--wails-drop-target: drop` CSS property on the root element
 * marks it as a valid drop zone. Wails adds/removes the
 * `wails-drop-target-active` class during drag-over — we watch for
 * that class to show/hide the overlay.
 */
export function useDragDrop() {
  const isDragging = ref(false);
  const showMenu = ref(false);
  const menuData = ref<ImportMenuData | null>(null);

  function closeMenu() {
    showMenu.value = false;
    menuData.value = null;
  }

  function onWailsDrop(x: number, y: number, paths: string[]) {
    console.log("[drag-drop] OnFileDrop:", x, y, paths);

    if (paths.length === 0) return;

    const droppedFiles: DroppedFile[] = paths.map((p) => ({
      name: p.split(/[\\/]/).pop() || p,
      path: p,
    }));

    const supported = droppedFiles.filter((f) => {
      const ext = "." + f.name.split(".").pop()?.toLowerCase();
      return SUPPORTED_EXTENSIONS.has(ext);
    });

    if (supported.length === 0) {
      console.log("[drag-drop] No supported files found");
      return;
    }

    console.log(`[drag-drop] Showing import menu for ${supported.length} file(s)`);
    menuData.value = { files: supported, x, y };
    showMenu.value = true;
  }

  function setupHandlers(rootEl: HTMLElement) {
    // Register Wails file-drop callback (useDropTarget=true = only fires on
    // elements with --wails-drop-target: drop CSS property)
    OnFileDrop(onWailsDrop, true);

    // Watch for Wails' active drop-target class to show the overlay.
    // Wails adds 'wails-drop-target-active' to elements during drag-over.
    const observer = new MutationObserver(() => {
      const active = rootEl.classList.contains("wails-drop-target-active");
      isDragging.value = active;
    });

    observer.observe(rootEl, { attributes: true, attributeFilter: ["class"] });

    // Store observer for cleanup
    (setupHandlers as any)._observer = observer;
  }

  function teardownHandlers(_rootEl: HTMLElement) {
    OnFileDropOff();

    const observer = (setupHandlers as any)._observer;
    if (observer) {
      observer.disconnect();
    }
  }

  return {
    isDragging,
    showMenu,
    menuData,
    closeMenu,
    setupHandlers,
    teardownHandlers,
  };
}
