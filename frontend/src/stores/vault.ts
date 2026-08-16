import { defineStore } from "pinia";
import type { VaultInfo, VaultConfig } from "../types/vault";
import {
  GetCurrentVault,
  OpenVault,
  CloseVault,
  SelectFolder,
  GetLastVaultPath,
  GetVaultConfig,
  SetVaultConfig,
  ScanVault,
  RescanVault,
} from "../api/backend";
import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";
import { logger } from "../utils/logger";

export const useVaultStore = defineStore("vault", {
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
      if (state.scanTotal <= 0) return 0;
      return Math.min(Math.round((state.scanCurrent / state.scanTotal) * 100), 100);
    },
    /** True when scan is running but total is not yet known (single-pass walk phase) */
    scanProgressUnknown: (state: any) => {
      return state.isScanning && state.scanTotal <= 0 && state.scanCurrent > 0;
    },
  },
  actions: {
    /** Auto-open the last vault from global app settings */
    async autoOpenLastVault() {
      const lastPath = await GetLastVaultPath();
      if (!lastPath) return;
      await this._doOpenVault(lastPath);
    },

    /** Prompt user to pick a folder, then open it as a vault */
    async pickAndOpenVault() {
      this.isLoading = true;
      try {
        const path = await SelectFolder();
        if (!path) return; // User cancelled
        await this._doOpenVault(path);
      } catch (e: any) {
        const { useToast } = await import("../composables/useToast");
        useToast().error(e.message || "Failed to open vault");
      } finally {
        this.isLoading = false;
      }
    },

    /** Open a vault at the given path */
    async openVault(path: string) {
      await this._doOpenVault(path);
    },

    /** Internal: open a vault and wire up scan event listeners */
    async _doOpenVault(path: string) {
      this.isLoading = true;
      try {
        const vault = await OpenVault(path);
        this.currentVault = vault;
        await this.loadConfig();

        // OpenVault only triggers an auto-scan when file count is 0.
        // Only show the scanning popup if this vault actually needs scanning.
        const needsScan = vault.file_count === 0;
        this.isScanning = needsScan;
        this.scanTotal = 0;
        this.scanCurrent = 0;

        if (needsScan) {
          // Clean up any stale listeners from previous opens
          EventsOff("scan:progress");
          EventsOff("scan:complete");
          EventsOff("scan:error");
          EventsOff("thumb:progress");
          EventsOff("thumb:complete");

          const autoProgressUnsub = EventsOn(
            "scan:progress",
            (data: { current: number; total: number }) => {
              this.scanCurrent = data.current;
              this.scanTotal = data.total;
            },
          );

          const thumbProgressUnsub = EventsOn(
            "thumb:progress",
            (data: { current: number; total: number }) => {
              this.scanCurrent = data.current;
              this.scanTotal = data.total;
            },
          );

          const autoCompleteUnsub = EventsOn(
            "scan:complete",
            (_count: number | { added: number; removed: number }) => {
              // Scan done — thumbnails are generated next by backend
              logger.log("vault.openVault", "auto-scan complete, generating thumbnails...");
            },
          );

          const thumbCompleteUnsub = EventsOn(
            "thumb:complete",
            (data: { generated: number; failed: number; total: number }) => {
              this.scanCurrent = data.total;
              this.scanTotal = data.total;
              this.isScanning = false;
              this.isLoading = false;
              this.refreshCurrentVault();
              logger.log(
                "vault.openVault",
                `auto-scan + thumbnails: ${data.generated} generated, ${data.failed} failed`,
              );
            },
          );

          const autoErrorUnsub = EventsOn("scan:error", (data: { error: string }) => {
            this.isScanning = false;
            this.isLoading = false;
            logger.error("vault.openVault", "auto-scan error:", data.error);
          });

          // Clean up listeners after 60 seconds
          setTimeout(() => {
            autoProgressUnsub();
            thumbProgressUnsub();
            autoCompleteUnsub();
            thumbCompleteUnsub();
            autoErrorUnsub();
          }, 60000);
        }
      } finally {
        this.isLoading = false;
      }
    },
    async closeVault() {
      await CloseVault();
      this.currentVault = null;
      this.config = null;
      // Always reset scanning state on close so the popup disappears
      this.isScanning = false;
      this.scanTotal = 0;
      this.scanCurrent = 0;
    },
    async loadConfig() {
      this.config = await GetVaultConfig();
      // Sync persisted grid_thumbnail_size to the UI store
      if (this.config?.settings?.grid_thumbnail_size) {
        const { useUIStore } = await import("./ui");
        useUIStore().setGridSize(this.config.settings.grid_thumbnail_size as any);
      }
    },
    async saveConfig(cfg: VaultConfig) {
      await SetVaultConfig(cfg as any);
      this.config = cfg;
    },
    /** Persist a grid size change to the vault config */
    async persistGridSize(size: string) {
      if (!this.config?.settings) return;
      this.config.settings.grid_thumbnail_size = size;
      await SetVaultConfig(this.config as any);
    },
    async rescanVault() {
      this.isScanning = true;
      this.isLoading = true;
      this.scanTotal = 0;
      this.scanCurrent = 0;

      const diffUnsub = EventsOn(
        "rescan:diff",
        (data: { added: number; removed: number; total: number }) => {
          this.scanTotal = data.total;
          this.scanCurrent = 0;
          logger.log(
            "vault.rescanVault",
            `rescan diff: +${data.added} -${data.removed} total=${data.total}`,
          );
        },
      );

      const progressUnsub = EventsOn(
        "rescan:progress",
        (data: { phase: string; current: number; total: number }) => {
          this.scanCurrent = data.current;
          this.scanTotal = data.total;
        },
      );

      const thumbProgressUnsub = EventsOn(
        "thumb:progress",
        (data: { current: number; total: number }) => {
          this.scanCurrent = data.current;
          this.scanTotal = data.total;
        },
      );

      // Use a flag so thumb:complete only fires once for this rescan
      let thumbDone = false;

      const completeUnsub = EventsOn(
        "rescan:complete",
        async (data: { added: number; removed: number }) => {
          logger.log(
            "vault.rescanVault",
            `rescan complete: +${data.added} -${data.removed}, generating thumbnails...`,
          );
          // Mark scan as done (show 100%) using total from rescan:diff —
          // thumbnails will update progress further if there's work to do
          this.scanCurrent = this.scanTotal;
          try {
            await import("../api/backend").then((m) => m.GenerateThumbnailsPool());
          } catch (e) {
            logger.error("vault.rescanVault", "thumbnail generation failed:", e);
          }
        },
      );

      const thumbCompleteUnsub = EventsOn(
        "thumb:complete",
        (data: { generated: number; failed: number; total: number }) => {
          if (thumbDone) return;
          thumbDone = true;
          if (data.total > 0) {
            this.scanCurrent = data.total;
            this.scanTotal = data.total;
          }
          this.isScanning = false;
          this.isLoading = false;
          logger.log(
            "vault.rescanVault",
            `thumbnails: ${data.generated} generated, ${data.failed} failed out of ${data.total}`,
          );
          this.refreshCurrentVault();
          // Safe to clean up now that we're done
          thumbCompleteUnsub();
        },
      );

      try {
        const added = await RescanVault();
        logger.log("vault.rescanVault", added, "added");
      } catch (e) {
        logger.error("vault.rescanVault", e);
        this.isScanning = false;
        this.isLoading = false;
      } finally {
        diffUnsub();
        progressUnsub();
        completeUnsub();
        thumbProgressUnsub();
        // NOTE: thumbCompleteUnsub is NOT called here.
        // The rescan:complete callback is async and calls GenerateThumbnailsPool().
        // The finally block runs before that async callback finishes, so unsubscribing
        // thumb:complete here would prevent it from ever firing.
        // Instead, thumbCompleteUnsub calls itself inside its own callback.
      }
    },

    async scanVault() {
      this.isScanning = true;
      this.isLoading = true;
      this.scanTotal = 0;
      this.scanCurrent = 0;

      // Listen for scan progress events
      const progressUnsub = EventsOn(
        "scan:progress",
        (data: { current: number; total: number }) => {
          this.scanCurrent = data.current;
          this.scanTotal = data.total;
        },
      );

      const thumbProgressUnsub = EventsOn(
        "thumb:progress",
        (data: { current: number; total: number }) => {
          this.scanCurrent = data.current;
          this.scanTotal = data.total;
        },
      );

      // Use a flag so thumb:complete only fires once for this scan
      let thumbDone = false;

      const completeUnsub = EventsOn("scan:complete", async (count: number) => {
        this.scanCurrent = count;
        this.scanTotal = count;
        // Thumbnails are generated next — keep isScanning true
        logger.log("vault.scanVault", "scan complete, generating thumbnails...");
        try {
          await import("../api/backend").then((m) => m.GenerateThumbnailsPool());
        } catch (e) {
          logger.error("vault.scanVault", "thumbnail generation failed:", e);
        }
      });

      const thumbCompleteUnsub = EventsOn(
        "thumb:complete",
        (data: { generated: number; failed: number; total: number }) => {
          if (thumbDone) return;
          thumbDone = true;
          this.scanCurrent = data.total;
          this.scanTotal = data.total;
          this.isScanning = false;
          this.isLoading = false;
          logger.log(
            "vault.scanVault",
            `thumbnails: ${data.generated} generated, ${data.failed} failed out of ${data.total}`,
          );
          this.refreshCurrentVault();
          // Safe to clean up now that we're done
          thumbCompleteUnsub();
        },
      );

      try {
        const count = await ScanVault();
        logger.log("vault.scanVault", "indexed", count, "files");
      } catch (e) {
        logger.error("vault.scanVault", e);
        this.isScanning = false;
        this.isLoading = false;
      } finally {
        progressUnsub();
        completeUnsub();
        thumbProgressUnsub();
        // NOTE: thumbCompleteUnsub is NOT called here.
        // The scan:complete callback is async and calls GenerateThumbnailsPool().
        // The finally block runs before that async callback finishes, so unsubscribing
        // thumb:complete here would prevent it from ever firing.
        // Instead, thumbCompleteUnsub calls itself inside its own callback.
      }
    },
    stopScanning() {
      this.isScanning = false;
      this.scanTotal = 0;
      this.scanCurrent = 0;
    },
    async refreshCurrentVault() {
      this.currentVault = await GetCurrentVault();
    },
  },
});
