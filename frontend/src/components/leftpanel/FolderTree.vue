<template>
  <div class="folder-tree">
    <div v-if="foldersStore.isLoading" class="loading">Loading folders…</div>
    <template v-else>
      <FolderNode v-for="node in foldersStore.tree" :key="node.path" :node="node" :depth="0" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from "vue";
import FolderNode from "./FolderNode.vue";
import { useFoldersStore } from "../../stores/folders";

const foldersStore = useFoldersStore();
onMounted(() => foldersStore.loadTree());
</script>

<style scoped>
.folder-tree {
  font-size: 13px;
}
.loading {
  color: #666;
  padding: 8px;
}
</style>
