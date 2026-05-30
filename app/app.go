package app

import (
	"context"

	"TagLoom/config"
	"TagLoom/db"
)

// App is the main Wails application struct.
// It coordinates vault, scanner, thumbnailer, search, tags, and metadata operations.
type App struct {
	ctx      context.Context
	db       *db.Database
	vaultPath string
	vaultCfg *config.VaultConfig
	appCfg   *config.AppSettings
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Load global app settings (last vault path, etc.)
	settings, err := config.LoadAppSettings()
	if err != nil {
		// Non-fatal: proceed with empty settings
		settings = &config.AppSettings{}
	}
	a.appCfg = settings
}

// GetLastVaultPath returns the path of the last opened vault (from global app settings).
// The frontend calls this on mount to decide whether to auto-open.
func (a *App) GetLastVaultPath() string {
	if a.appCfg == nil {
		return ""
	}
	return a.appCfg.LastVaultPath
}

// GetCurrentVault returns information about the currently open vault.
func (a *App) GetCurrentVault() *db.VaultInfo {
	if a.db == nil || a.vaultPath == "" {
		return nil
	}

	var fileCount int
	_ = a.db.Conn().QueryRow("SELECT COUNT(*) FROM files").Scan(&fileCount)

	return &db.VaultInfo{
		Path:      a.vaultPath,
		Name:      a.vaultCfg.Name,
		CreatedAt: a.vaultCfg.CreatedAt,
		FileCount: fileCount,
	}
}
