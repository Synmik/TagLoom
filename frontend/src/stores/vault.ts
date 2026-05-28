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
    scanProgress: 0,
    scanTotal: 0,
    scanCurrent: 0,
    isScanning: false,
  }),
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
      this.stopScanning()
      await CloseVault()
      this.currentVault = null
      this.config = null
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
      this.scanProgress = 0
      this.scanTotal = 0
      this.scanCurrent = 0

      const diffUnsub = EventsOn('rescan:diff', (data: { added: number; removed: number; total: number }) => {
        this.scanTotal = data.total
        this.scanCurrent = 0
        this.scanProgress = 0
        console.log(`Rescan diff: +${data.added} -${data.removed} total=${data.total}`)
      })

      const progressUnsub = EventsOn('rescan:progress', (data: { phase: string; current: string; total: string }) => {
        this.scanCurrent = parseInt(data.current, 10)
        this.scanTotal = parseInt(data.total, 10)
        if (this.scanTotal > 0) {
          this.scanProgress = Math.round((this.scanCurrent / this.scanTotal) * 100)
        }
      })

      const completeUnsub = EventsOn('rescan:complete', (data: { added: number; removed: number }) => {
        this.scanProgress = 100
        this.isScanning = false
        this.isLoading = false
        console.log(`Rescan complete: +${data.added} -${data.removed}`)
        this.refreshCurrentVault()
      })

      try {
        const [added, removed] = await RescanVault()
        console.log('RescanVault result:', added, 'added,', removed, 'removed')
      } catch (e) {
        console.error('RescanVault failed:', e)
        this.isScanning = false
        this.isLoading = false
      } finally {
        diffUnsub()
        progressUnsub()
        completeUnsub()
      }
    },

    async scanVault() {
      this.isScanning = true
      this.isLoading = true
      this.scanProgress = 0
      this.scanTotal = 0
      this.scanCurrent = 0

      // Listen for progress events
      const progressUnsub = EventsOn('scan:progress', (data: { current: number; total: number }) => {
        this.scanCurrent = data.current
        this.scanTotal = data.total
        if (data.total > 0) {
          this.scanProgress = Math.round((data.current / data.total) * 100)
        }
      })

      const completeUnsub = EventsOn('scan:complete', (count: number) => {
        this.scanCurrent = count
        this.scanTotal = count
        this.scanProgress = 100
        this.isScanning = false
        this.isLoading = false
        // Refresh vault info to get updated file count
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
      }
    },
    stopScanning() {
      this.isScanning = false
      EventsOff('scan:progress', 'scan:complete')
    },
    async refreshCurrentVault() {
      this.currentVault = await GetCurrentVault()
    },
  },
})
