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
} from '../api/backend'

export const useVaultStore = defineStore('vault', {
  state: () => ({
    currentVault: null as VaultInfo | null,
    config: null as VaultConfig | null,
    isLoading: false,
    scanProgress: 0,
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
    async scanVault() {
      this.isLoading = true
      this.scanProgress = 0
      try {
        await ScanVault(null)
      } finally {
        this.isLoading = false
        this.scanProgress = 100
      }
    },
    async refreshCurrentVault() {
      this.currentVault = await GetCurrentVault()
    },
  },
})
