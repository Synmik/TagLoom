<template>
  <aside class="right-panel" :style="panelStyle">
    <div v-if="!previewStore.currentFile" class="empty-state">
      <p>Select a file to view details</p>
    </div>
    <template v-else>
      <PreviewSection />
      <NameField />
      <LinkField />
      <NotesField />
      <TagsEditor />
      <MetadataSection />
    </template>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { usePreviewStore } from '../../stores/preview'
import { useUIStore } from '../../stores/ui'
import PreviewSection from './PreviewSection.vue'
import NameField from './NameField.vue'
import LinkField from './LinkField.vue'
import NotesField from './NotesField.vue'
import TagsEditor from './TagsEditor.vue'
import MetadataSection from './MetadataSection.vue'

const previewStore = usePreviewStore()
const uiStore = useUIStore()
const panelStyle = computed(() => ({ width: `${uiStore.rightPanelWidth}px` }))
</script>

<style scoped>
.right-panel {
  overflow-y: auto; padding: 12px;
  border-left: 1px solid #333; background: #1a1a1a;
  display: flex; flex-direction: column; gap: 12px;
  flex-shrink: 0;
}
.empty-state { text-align: center; color: #555; padding: 40px 0; }
</style>
