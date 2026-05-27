package app

import (
	"context"

	"TagLoom/db"
)

// App is the main Wails application struct.
// It coordinates vault, scanner, thumbnailer, search, tags, and metadata operations.
type App struct {
	ctx context.Context
	db  *db.Database
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
	// TODO: Implement
	return nil
}
