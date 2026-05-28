import { defineStore } from 'pinia'
import type { VaultInfo, VaultConfig } from '../types/vault'
import {
  GetCurrentVault,
  OpenVault,
  CloseVault,
  SelectFolder,
  GetVaultConfig,
  SetVaultConfig,
  ScanVault,
  RescanVault,
} from '../api/backend'
// @ts-ignore - wails runtime is injected at build time
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

export const useVaultStore = defineStore('vault', {
  state: () => ({
    currentVault: null as VaultInfo | null,
    config: null as VaultConfig | null,
    isLoading: false,
    scanTotal: 0,
    scanCurrent: 0,
    isScanning: false,
  }),
  getters: {
    /** Derived from scanCurrent / scanTotal — always consistent */
    scanProgress: (state: any) => {
      if (state.scanTotal <= 0) return 0
      return Math.min(Math.round((state.scanCurrent / state.scanTotal) * 100), 100)
    },
  },
  actions: {
    /** Prompt user to pick a folder, then open it as a vault */
    async pickAndOpenVault() {
      this.isLoading = true
      try {
        const path = await SelectFolder()
        if (!path) return // User cancelled
        const vault = await OpenVault(path)
        this.currentVault = vault
        await this.loadConfig()

        // OpenVault only triggers an auto-scan when file count is 0.
        // Only show the scanning popup if this vault actually needs scanning.
        const needsScan = vault.file_count === 0
        this.isScanning = needsScan
        this.scanTotal = 0
        this.scanCurrent = 0

        if (needsScan) {
          // Clean up any stale listeners from previous opens
          EventsOff('scan:progress')
          EventsOff('scan:complete')
          EventsOff('scan:error')
          EventsOff('thumb:progress')
          EventsOff('thumb:complete')

          const autoProgressUnsub = EventsOn('scan:progress', (data: { current: number; total: number }) => {
            this.scanCurrent = data.current
            this.scanTotal = data.total
          })

          const thumbProgressUnsub = EventsOn('thumb:progress', (data: { current: number; total: number }) => {
            this.scanCurrent = data.current
            this.scanTotal = data.total
          })

          const autoCompleteUnsub = EventsOn('scan:complete', (count: number | { added: number; removed: number }) => {
            // Scan done — thumbnails are generated next by backend
            console.log('Auto-scan complete, generating thumbnails...')
          })

          const thumbCompleteUnsub = EventsOn('thumb:complete', (data: { generated: number; failed: number; total: number }) => {
            this.scanCurrent = data.total
            this.scanTotal = data.total
            this.isScanning = false
            this.isLoading = false
            this.refreshCurrentVault()
            console.log(`Auto-scan + thumbnails: ${data.generated} generated, ${data.failed} failed`)
          })

          const autoErrorUnsub = EventsOn('scan:error', (data: { error: string }) => {
            this.isScanning = false
            this.isLoading = false
            console.error('Auto-scan error:', data.error)
          })

          // Clean up listeners after 60 seconds
          setTimeout(() => {
            autoProgressUnsub()
            thumbProgressUnsub()
            autoCompleteUnsub()
            thumbCompleteUnsub()
            autoErrorUnsub()
          }, 60000)
        }
      } finally {
        this.isLoading = false
      }
    },
    async openVault(path: string) {
      this.isLoading = true
      try {
        const vault = await OpenVault(path)
        this.currentVault = vault
        await this.loadConfig()
      } finally {
        this.isLoading = false
      }
    },
    async closeVault() {
      await CloseVault()
      this.currentVault = null
      this.config = null
      // Always reset scanning state on close so the popup disappears
      this.isScanning = false
      this.scanTotal = 0
      this.scanCurrent = 0
    },
    async loadConfig() {
      this.config = await GetVaultConfig()
    },
    async saveConfig(cfg: VaultConfig) {
      await SetVaultConfig(cfg as any)
      this.config = cfg
    },
    async rescanVault() {
      this.isScanning = true
      this.isLoading = true
      this.scanTotal = 0
      this.scanCurrent = 0

      const diffUnsub = EventsOn('rescan:diff', (data: { added: number; removed: number; total: number }) => {
        this.scanTotal = data.total
        this.scanCurrent = 0
        console.log(`Rescan diff: +${data.added} -${data.removed} total=${data.total}`)
      })

      const progressUnsub = EventsOn('rescan:progress', (data: { phase: string; current: string; total: string }) => {
        this.scanCurrent = parseInt(data.current, 10)
        this.scanTotal = parseInt(data.total, 10)
      })

      const thumbProgressUnsub = EventsOn('thumb:progress', (data: { current: number; total: number }) => {
        this.scanCurrent = data.current
        this.scanTotal = data.total
      })

      const completeUnsub = EventsOn('rescan:complete', async (data: { added: number; removed: number }) => {
        console.log(`Rescan complete: +${data.added} -${data.removed}, generating thumbnails...`)
        try {
          await import('../api/backend').then(m => m.GenerateThumbnailsPool())
        } catch (e) {
          console.error('Thumbnail generation failed:', e)
        }
      })

      const thumbCompleteUnsub = EventsOn('thumb:complete', (data: { generated: number; failed: number; total: number }) => {
        this.scanCurrent = data.total
        this.scanTotal = data.total
        this.isScanning = false
        this.isLoading = false
        console.log(`Thumbnails: ${data.generated} generated, ${data.failed} failed out of ${data.total}`)
        this.refreshCurrentVault()
      })

      try {
        const added = await RescanVault()
        console.log('RescanVault result:', added, 'added')
      } catch (e) {
        console.error('RescanVault failed:', e)
        this.isScanning = false
        this.isLoading = false
      } finally {
        diffUnsub()
        progressUnsub()
        completeUnsub()
        thumbProgressUnsub()
        thumbCompleteUnsub()
      }
    },

    async scanVault() {
      this.isScanning = true
      this.isLoading = true
      this.scanTotal = 0
      this.scanCurrent = 0

      // Listen for scan progress events
      const progressUnsub = EventsOn('scan:progress', (data: { current: number; total: number }) => {
        this.scanCurrent = data.current
        this.scanTotal = data.total
      })

      const thumbProgressUnsub = EventsOn('thumb:progress', (data: { current: number; total: number }) => {
        this.scanCurrent = data.current
        this.scanTotal = data.total
      })

      const completeUnsub = EventsOn('scan:complete', async (count: number) => {
        this.scanCurrent = count
        this.scanTotal = count
        // Thumbnails are generated next — keep isScanning true
        console.log('Scan complete, generating thumbnails...')
        try {
          await import('../api/backend').then(m => m.GenerateThumbnailsPool())
        } catch (e) {
          console.error('Thumbnail generation failed:', e)
        }
      })

      const thumbCompleteUnsub = EventsOn('thumb:complete', (data: { generated: number; failed: number; total: number }) => {
        this.scanCurrent = data.total
        this.scanTotal = data.total
        this.isScanning = false
        this.isLoading = false
        console.log(`Thumbnails: ${data.generated} generated, ${data.failed} failed out of ${data.total}`)
        this.refreshCurrentVault()
      })

      try {
        const count = await ScanVault()
        console.log('ScanVault indexed', count, 'files')
      } catch (e) {
        console.error('ScanVault failed:', e)
        this.isScanning = false
        this.isLoading = false
      } finally {
        progressUnsub()
        completeUnsub()
        thumbProgressUnsub()
        thumbCompleteUnsub()
      }
    },
    stopScanning() {
      this.isScanning = false
      this.scanTotal = 0
      this.scanCurrent = 0
    },
    async refreshCurrentVault() {
      this.currentVault = await GetCurrentVault()
    },
  },
})
