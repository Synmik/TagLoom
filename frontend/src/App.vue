<template>
  <div class="app-layout">
    <TopBar />
    <div class="app-body">
      <LeftPanel />
      <Gallery />
      <RightPanel />
    </div>
  </div>
</template>

<script setup lang="ts">
import TopBar from './components/topbar/TopBar.vue'
import LeftPanel from './components/leftpanel/LeftPanel.vue'
import Gallery from './components/gallery/Gallery.vue'
import RightPanel from './components/rightpanel/RightPanel.vue'
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #121212; color: #ddd; }
</style>

<style scoped>
.app-layout { display: flex; flex-direction: column; height: 100vh; overflow: hidden; }
.app-body { display: flex; flex: 1; overflow: hidden; }
</style>
