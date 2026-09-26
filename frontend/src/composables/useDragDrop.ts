import { ref } from "vue";
import { SUPPORTED_EXTENSIONS } from "../types/file";
import { OnFileDrop, OnFileDropOff } from "../../wailsjs/runtime/runtime";
import { logger } from "../utils/logger";

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
 * True while a drag started inside the app — re-parenting a tag in the Tag
 * Manager, for instance. Wails marks the root drop target during any drag over
 * the window, and its class is what shows the "Drop files to import" overlay, so
 * without this an internal drag would look like an incoming file drop.
 * Module level because the drag is started by one component and the overlay
 * lives in another.
 */
let internalDrag = false;

export function beginInternalDrag() {
  internalDrag = true;
}

export function endInternalDrag() {
  internalDrag = false;
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
    logger.log("dragDrop.onFileDrop", x, y, paths);

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
      logger.log("dragDrop.onFileDrop", "no supported files found");
      return;
    }

    logger.log("dragDrop.onFileDrop", `showing import menu for ${supported.length} file(s)`);
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
      const active = rootEl.classList.contains("wails-drop-target-active") && !internalDrag;
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
