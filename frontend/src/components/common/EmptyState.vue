<template>
  <div class="empty-state">
    <div class="empty-icon">📂</div>
    <p class="empty-title">No files in this vault</p>
    <p class="empty-desc">Open or create a vault to get started</p>
    <button class="scan-btn" @click="openVault">Scan Folder</button>
  </div>
</template>

<script setup lang="ts">
const openVault = () => {
  // TODO: Trigger vault open dialog
}
</script>

<style scoped>
.empty-state {
  display: flex; flex-direction: column; align-items: center;
  justify-content: center; height: 100%; color: #555; gap: 8px;
}
.empty-icon { font-size: 48px; }
.empty-title { font-size: 16px; color: #777; margin: 0; }
.empty-desc { font-size: 13px; margin: 0; }
.scan-btn {
  margin-top: 8px; background: #5b8af5; color: #fff; border: none;
  border-radius: 6px; padding: 8px 20px; cursor: pointer; font-size: 13px;
}
.scan-btn:hover { background: #4a7ae5; }
</style>
