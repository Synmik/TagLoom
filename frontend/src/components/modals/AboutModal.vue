<script setup lang="ts">
import { ref, onMounted } from "vue";
import { X, ExternalLink, CheckCircle2, XCircle, Terminal } from "@lucide/vue";
import logoImg from "../../assets/appicon_small.png";
import { CheckFFmpeg, GetAppInfo, GetVersion } from "../../api/backend";
import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime";

defineEmits<{ close: [] }>();

const version = ref("dev");

// ── FFmpeg status ─────────────────────────────────────────────────
interface FFmpegStatus {
  ffmpeg_ok: boolean;
  ffprobe_ok: boolean;
  ffmpeg_path: string;
  ffprobe_path: string;
}

const ffmpeg = ref<FFmpegStatus | null>(null);

// ── System info ───────────────────────────────────────────────────
const appInfo = ref<Record<string, string> | null>(null);

const shortcuts: { key: string; desc: string }[] = [
  { key: "Ctrl+A", desc: "Select all files matching current filters" },
  { key: "Ctrl+B", desc: "Open batch edit modal for selected items" },
  { key: "Ctrl+C", desc: "Copy original image to clipboard, if possible" },
  { key: "Ctrl+F", desc: "Focus search bar" },
  { key: "Ctrl+R", desc: "Rescan the vault" },
  { key: "Ctrl+D", desc: "Toggle favorite on selected file(s)" },
  { key: "Enter", desc: "Open preview (same as double-click on thumbnail)" },
  {
    key: "Escape",
    desc: "Close open modals (priority order), then clear selection, unfocus search",
  },
  { key: "\u2190 / \u2192", desc: "Navigate gallery (previous / next file)" },
];

onMounted(async () => {
  try {
    ffmpeg.value = await CheckFFmpeg();
  } catch {
    ffmpeg.value = null;
  }
  try {
    appInfo.value = await GetAppInfo();
  } catch {
    appInfo.value = null;
  }
  try {
    version.value = await GetVersion();
  } catch {
    version.value = "dev";
  }
});
</script>

<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h3>About</h3>
        <button class="close-btn" @click="$emit('close')"><X :size="16" /></button>
      </div>

      <div class="modal-body">
        <!-- Logo + name + version -->
        <section class="hero">
          <img :src="logoImg" alt="TagLoom logo" class="hero-logo" />
          <div class="hero-text">
            <h1 class="hero-name">TagLoom</h1>
            <span class="hero-version">v{{ version }}</span>
          </div>
        </section>

        <!-- Badges -->
        <div class="badges">
          <span class="badge badge-license">MIT License</span>
          <button
            class="badge badge-github"
            @click="BrowserOpenURL('https://github.com/Synmik/TagLoom')"
          >
            <ExternalLink :size="12" /> GitHub
          </button>
        </div>

        <!-- Description -->
        <p class="description">
          TagLoom is a desktop multimedia tagging and library management application for images,
          videos, and GIFs. It indexes existing folders (vaults) without modifying any original
          files &mdash; all metadata and thumbnails live inside a <code>.tagloom</code> directory.
        </p>

        <!-- FFmpeg status -->
        <section class="section">
          <h4>Dependencies</h4>
          <div class="status-row">
            <CheckCircle2 v-if="ffmpeg?.ffmpeg_ok" :size="14" class="icon-ok" />
            <XCircle v-else :size="14" class="icon-fail" />
            <div class="status-text">
              <span class="status-label">FFmpeg</span>
              <span v-if="ffmpeg?.ffmpeg_path" class="status-path">{{ ffmpeg.ffmpeg_path }}</span>
              <span v-else class="status-missing">Not found</span>
            </div>
          </div>
          <div class="status-row">
            <CheckCircle2 v-if="ffmpeg?.ffprobe_ok" :size="14" class="icon-ok" />
            <XCircle v-else :size="14" class="icon-fail" />
            <div class="status-text">
              <span class="status-label">FFprobe</span>
              <span v-if="ffmpeg?.ffprobe_path" class="status-path">{{ ffmpeg.ffprobe_path }}</span>
              <span v-else class="status-missing">Not found</span>
            </div>
          </div>
        </section>

        <!-- Keyboard shortcuts -->
        <section class="section">
          <h4>Keyboard Shortcuts</h4>
          <div class="shortcuts">
            <div v-for="s in shortcuts" :key="s.key" class="shortcut-row">
              <kbd class="kbd">{{ s.key }}</kbd>
              <span class="shortcut-desc">{{ s.desc }}</span>
            </div>
          </div>
        </section>

        <!-- System info -->
        <section v-if="appInfo" class="section">
          <h4>System</h4>
          <div class="info-grid">
            <div class="info-item">
              <Terminal :size="12" class="info-icon" />
              <span class="info-label">OS</span>
              <span class="info-value">{{ appInfo.os }}</span>
            </div>
            <div class="info-item">
              <Terminal :size="12" class="info-icon" />
              <span class="info-label">Arch</span>
              <span class="info-value">{{ appInfo.arch }}</span>
            </div>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  background: #111111;
  border-radius: 12px;
  width: 520px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  border: 1px solid #222;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 18px;
  border-bottom: 1px solid #1a1a1a;
}

.modal-header h3 {
  margin: 0;
  color: #e8e8e8;
  font-size: 14px;
  font-weight: 600;
  font-family: "Inter", sans-serif;
}

.close-btn {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  font-size: 16px;
  padding: 4px;
  border-radius: 4px;
  transition:
    color 0.15s,
    background 0.15s;
}

.close-btn:hover {
  color: #e8e8e8;
  background: #1a1a1a;
}

.modal-body {
  padding: 18px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ── Hero ─────────────────────────────────────────────────────── */
.hero {
  display: flex;
  align-items: center;
  gap: 14px;
}

.hero-logo {
  width: 56px;
  height: 56px;
  border-radius: 10px;
  object-fit: contain;
  background: #1a1a1a;
  flex-shrink: 0;
}

.hero-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.hero-name {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: #e8e8e8;
  font-family: "Inter", sans-serif;
}

.hero-version {
  font-size: 12px;
  color: #666;
  font-weight: 500;
}

/* ── Badges ───────────────────────────────────────────────────── */
.badges {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 500;
  text-decoration: none;
}

.badge-license {
  background: #14532d;
  color: #4ade80;
}

.badge-github {
  background: #1a1a1a;
  color: #999;
  border: 1px solid #2a2a2a;
  transition:
    color 0.15s,
    border-color 0.15s;
}

.badge-github:hover {
  color: #e8e8e8;
  border-color: #444;
}

/* ── Description ──────────────────────────────────────────────── */
.description {
  margin: 0;
  color: #888;
  font-size: 12px;
  line-height: 1.6;
}

.description code {
  background: #1a1a1a;
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 11px;
  color: #ccc;
  font-family: monospace;
}

/* ── Sections ─────────────────────────────────────────────────── */
.section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.section h4 {
  margin: 0;
  color: #888;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.6px;
}

/* ── Status rows ──────────────────────────────────────────────── */
.status-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}

.icon-ok {
  color: #22c55e;
  flex-shrink: 0;
}

.icon-fail {
  color: #ef4444;
  flex-shrink: 0;
}

.status-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.status-label {
  color: #ccc;
  font-size: 12px;
  font-weight: 500;
}

.status-path {
  color: #666;
  font-size: 10px;
  font-family: monospace;
}

.status-missing {
  color: #ef4444;
  font-size: 10px;
  font-style: italic;
}

/* ── Shortcuts ────────────────────────────────────────────────── */
.shortcuts {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.shortcut-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 3px 0;
}

.kbd {
  display: inline-block;
  background: #1a1a1a;
  border: 1px solid #2a2a2a;
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 11px;
  font-family: monospace;
  color: #ccc;
  min-width: 60px;
  text-align: center;
  flex-shrink: 0;
}

.shortcut-desc {
  color: #888;
  font-size: 11px;
  line-height: 1.4;
}

/* ── System info grid ─────────────────────────────────────────── */
.info-grid {
  display: flex;
  gap: 12px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 6px;
  background: #1a1a1a;
  padding: 5px 10px;
  border-radius: 6px;
  flex: 1;
}

.info-icon {
  color: #555;
  flex-shrink: 0;
}

.info-label {
  color: #666;
  font-size: 10px;
  text-transform: uppercase;
}

.info-value {
  color: #ccc;
  font-size: 11px;
  font-family: monospace;
  margin-left: auto;
}
</style>
