import { useFilesStore } from "../stores/files";
import type { File } from "../types/file";

export function useSelection() {
  const filesStore = useFilesStore();

  const select = (file: File, multi: boolean = false) => {
    filesStore.selectFile(file, multi);
  };

  const clear = () => {
    filesStore.clearSelection();
  };

  const isSelected = (file: File): boolean => {
    return filesStore.selectedFiles.some((f) => f.id === file.id);
  };

  const toggleSelection = (file: File, ctrlKey: boolean = false, shiftKey: boolean = false) => {
    if (shiftKey) {
      // Range select - TODO: implement range selection
      select(file, true);
    } else if (ctrlKey) {
      select(file, true);
    } else {
      select(file, false);
    }
  };

  return { select, clear, isSelected, toggleSelection, filesStore };
}
