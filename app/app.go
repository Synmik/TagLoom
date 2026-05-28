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
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
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
