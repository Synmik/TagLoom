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

// OpenVault opens an existing vault at the given path.
// Returns an error if `.tagloom` does not exist — use CreateVault for new vaults.
func (a *App) OpenVault(path string) (*db.VaultInfo, error) {
	if path == "" {
		return nil, fmt.Errorf("vault path is empty")
	}

	// Verify .tagloom already exists
	tagloomDir := filepath.Join(path, ".tagloom")
	if _, err := os.Stat(tagloomDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no vault found at this location (.tagloom missing). Use \"New Vault\" to create one")
	}

	// Close existing vault if open
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			return nil, fmt.Errorf("failed to close previous vault: %w", err)
		}
	}

	// Open existing SQLite database
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

	// Save as last opened vault in global app settings
	if a.appCfg != nil {
		a.appCfg.LastVaultPath = path
	}

	// Add to recent vaults list
	a.addToRecentVaults(path, cfg.Name)

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
// Performs WAL checkpoint + orphan thumbnail cleanup before closing.
func (a *App) CloseVault() error {
	if a.db == nil {
		return nil
	}

	// WAL checkpoint: flush pending changes to main DB and truncate WAL file.
	// This keeps the DB compact across sessions and prevents WAL file growth.
	_, _ = a.db.Conn().Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	// Clean up orphan thumbnails before closing
	if _, err := a.CleanupOrphanThumbnails(); err != nil {
		fmt.Printf("orphan thumbnail cleanup warning: %v\n", err)
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

// NewVaultSettings holds the initial settings for creating a new vault.
type NewVaultSettings struct {
	ThumbnailQuality int      `json:"thumbnail_quality"`
	ExcludedFolders  []string `json:"excluded_folders"`
}

// CreateVault creates a brand-new vault at the given path.
// Returns an error if `.tagloom` already exists in the selected folder.
func (a *App) CreateVault(path string, settings NewVaultSettings) (*db.VaultInfo, error) {
	if path == "" {
		return nil, fmt.Errorf("vault path is empty")
	}

	// Check that .tagloom does NOT already exist
	tagloomDir := filepath.Join(path, ".tagloom")
	if _, err := os.Stat(tagloomDir); err == nil {
		return nil, fmt.Errorf("a vault already exists at this location (.tagloom found). Use \"Open Vault\" instead")
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("error checking folder: %w", err)
	}

	// Close existing vault if open
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			return nil, fmt.Errorf("failed to close previous vault: %w", err)
		}
	}

	// Create .tagloom directory
	if err := os.MkdirAll(tagloomDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .tagloom directory: %w", err)
	}

	// Create SQLite database
	dbPath := filepath.Join(tagloomDir, "tagloom.db")
	var err error
	a.db, err = db.NewDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Seed default tags
	if err := a.db.SeedDefaultTags(); err != nil {
		return nil, fmt.Errorf("failed to seed default tags: %w", err)
	}

	// Create config with user-provided settings
	cfg := config.DefaultConfig()
	cfg.Name = filepath.Base(path)
	cfg.CreatedAt = time.Now().Format(time.RFC3339)
	cfg.Settings.ThumbnailSize = 256
	cfg.Settings.ThumbnailQuality = settings.ThumbnailQuality
	cfg.Settings.ExcludedFolders = settings.ExcludedFolders

	if err := config.SaveConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	// Insert excluded folders into DB
	for _, ef := range settings.ExcludedFolders {
		_, _ = a.db.Conn().Exec(
			"INSERT OR IGNORE INTO excluded_folders (path) VALUES (?)",
			ef,
		)
	}

	// Store vault state
	a.vaultPath = path
	a.vaultCfg = cfg

	// Save as last opened vault
	if a.appCfg != nil {
		a.appCfg.LastVaultPath = path
	}

	// Add to recent vaults list
	a.addToRecentVaults(path, cfg.Name)

	// Auto-scan (vault is new, file count is 0)
	go func() {
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
			if err := a.GenerateThumbnailsPool(); err != nil {
				fmt.Printf("thumbnail generation warning: %v\n", err)
			}
		}
		runtime.EventsEmit(a.ctx, "scan:complete", count)
	}()

	return &db.VaultInfo{
		Path:      path,
		Name:      cfg.Name,
		CreatedAt: cfg.CreatedAt,
		FileCount: 0,
	}, nil
}

// SetVaultConfig updates the vault configuration.
func (a *App) SetVaultConfig(cfg *config.VaultConfig) error {
	if a.vaultPath == "" {
		return fmt.Errorf("no vault open")
	}
	if err := config.SaveConfig(a.vaultPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// If the vault name changed, update recent vaults entry too
	if a.vaultCfg != nil && cfg.Name != a.vaultCfg.Name && cfg.Name != "" {
		if a.appCfg != nil {
			for i := range a.appCfg.RecentVaults {
				if a.appCfg.RecentVaults[i].Path == a.vaultPath {
					a.appCfg.RecentVaults[i].Name = cfg.Name
					break
				}
			}
			_ = config.SaveAppSettings(a.appCfg)
		}
	}

	a.vaultCfg = cfg
	return nil
}
