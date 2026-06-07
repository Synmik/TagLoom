import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  define: {
    // Injected at build time via APP_VERSION env var (set by Wails).
    // Falls back to "dev" during local dev.
    '__APP_VERSION__': JSON.stringify(process.env.APP_VERSION || 'dev'),
  },
})
