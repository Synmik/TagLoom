package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"TagLoom/config"
	"TagLoom/db"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SelectFolder opens a native folder picker dialog and returns the selected path.
// The frontend calls this to get a vault path, then passes it to OpenVault.
func (a *App) SelectFolder() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("app context not initialized")
	}

	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Vault Folder",
	})
	if err != nil {
		return "", fmt.Errorf("dialog error: %w", err)
	}
	if dir == "" {
		return "", nil // User cancelled
	}
	return dir, nil
}

// OpenVault opens an existing vault or creates a new one at the given path.
func (a *App) OpenVault(path string) (*db.VaultInfo, error) {
	if path == "" {
		return nil, fmt.Errorf("vault path is empty")
	}

	// Close existing vault if open
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			return nil, fmt.Errorf("failed to close previous vault: %w", err)
		}
	}

	// Ensure .tagloom directory exists
	tagloomDir := filepath.Join(path, ".tagloom")
	if err := os.MkdirAll(tagloomDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .tagloom directory: %w", err)
	}

	// Open or create SQLite database
	dbPath := filepath.Join(tagloomDir, "tagloom.db")
	var err error
	a.db, err = db.NewDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Seed default tags if new vault
	if err := a.db.SeedDefaultTags(); err != nil {
		return nil, fmt.Errorf("failed to seed default tags: %w", err)
	}

	// Load or create config
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.CreatedAt == "" {
		cfg.Name = filepath.Base(path)
		cfg.CreatedAt = time.Now().Format(time.RFC3339)
		if err := config.SaveConfig(path, cfg); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}
	}

	// Store vault state in app
	a.vaultPath = path
	a.vaultCfg = cfg

	// Count indexed files
	fileCount := 0
	_ = a.db.Conn().QueryRow("SELECT COUNT(*) FROM files").Scan(&fileCount)

	// Auto-scan if vault has no indexed files
	if fileCount == 0 {
		go func() {
			// Small delay to let UI update first
			time.Sleep(500 * time.Millisecond)
			runtime.EventsEmit(a.ctx, "scan:started", map[string]string{
				"vault_path": path,
			})
			count, err := a.ScanVault()
			if err != nil {
				runtime.EventsEmit(a.ctx, "scan:error", map[string]string{
					"error": err.Error(),
				})
			} else {
				// Generate thumbnails after scan completes
				if err := a.GenerateThumbnailsPool(); err != nil {
					fmt.Printf("thumbnail generation warning: %v\n", err)
				}
			}
			runtime.EventsEmit(a.ctx, "scan:complete", count)
		}()
	}

	return &db.VaultInfo{
		Path:      path,
		Name:      cfg.Name,
		CreatedAt: cfg.CreatedAt,
		FileCount: fileCount,
	}, nil
}

// CloseVault closes the current vault and releases resources.
func (a *App) CloseVault() error {
	if a.db == nil {
		return nil
	}
	if err := a.db.Close(); err != nil {
		return fmt.Errorf("failed to close vault: %w", err)
	}
	a.db = nil
	a.vaultPath = ""
	a.vaultCfg = nil
	return nil
}

// GetVaultConfig returns the current vault configuration.
func (a *App) GetVaultConfig() (*config.VaultConfig, error) {
	if a.vaultCfg == nil {
		return nil, fmt.Errorf("no vault open")
	}
	return a.vaultCfg, nil
}

// SetVaultConfig updates the vault configuration.
func (a *App) SetVaultConfig(cfg *config.VaultConfig) error {
	if a.vaultPath == "" {
		return fmt.Errorf("no vault open")
	}
	if err := config.SaveConfig(a.vaultPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	a.vaultCfg = cfg
	return nil
}
