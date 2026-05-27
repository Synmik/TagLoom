import { defineStore } from 'pinia'
import type { VaultInfo, VaultConfig } from '../types/vault'

export const useVaultStore = defineStore('vault', {
  state: () => ({
    currentVault: null as VaultInfo | null,
    config: null as VaultConfig | null,
    isLoading: false,
    scanProgress: 0,
  }),
  actions: {
    async openVault(path: string) {
      this.isLoading = true
      try {
        // @ts-ignore - wailsjs auto-generated
        const vault = await window.go.main.app.OpenVault(path)
        this.currentVault = vault
        await this.loadConfig()
      } finally {
        this.isLoading = false
      }
    },
    async closeVault() {
      // @ts-ignore
      await window.go.main.app.CloseVault()
      this.currentVault = null
      this.config = null
    },
    async loadConfig() {
      // @ts-ignore
      this.config = await window.go.main.app.GetVaultConfig()
    },
    async saveConfig(config: VaultConfig) {
      // @ts-ignore
      await window.go.main.app.SetVaultConfig(config)
      this.config = config
    },
    async scanVault() {
      this.isLoading = true
      this.scanProgress = 0
      try {
        // @ts-ignore
        await window.go.main.app.ScanVault()
      } finally {
        this.isLoading = false
        this.scanProgress = 100
      }
    },
  },
})
