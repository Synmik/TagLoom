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
	if err := a.closeCurrentVault(); err != nil {
		return nil, fmt.Errorf("failed to close previous vault: %w", err)
	}

	// Open existing SQLite database. All slow I/O happens here, outside
	// the state lock; state is swapped in atomically at the end.
	dbPath := filepath.Join(tagloomDir, "tagloom.db")
	d, err := db.NewDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Seed default tags if new vault
	if err := d.SeedDefaultTags(); err != nil {
		d.Close()
		return nil, fmt.Errorf("failed to seed default tags: %w", err)
	}

	// Migrate absolute paths to relative paths for existing vaults.
	// This ensures the vault can be moved to a different location.
	if err := a.migrateToRelativePaths(d, path); err != nil {
		fmt.Printf("path migration warning: %v\n", err)
	}

	// Load or create config
	cfg, err := config.LoadConfig(path)
	if err != nil {
		d.Close()
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.CreatedAt == "" {
		cfg.Name = filepath.Base(path)
		cfg.CreatedAt = time.Now().Format(time.RFC3339)
		if err := config.SaveConfig(path, cfg); err != nil {
			d.Close()
			return nil, fmt.Errorf("failed to save config: %w", err)
		}
	}

	// Count indexed files once, at install time — the count is cached in
	// App state and maintained on mutations (see setFileCount and friends),
	// so GetCurrentVault never re-queries COUNT(*).
	fileCount := 0
	_ = d.Conn().QueryRow("SELECT COUNT(*) FROM files").Scan(&fileCount)

	// Atomically install as current vault (also saves last-vault path)
	a.setVault(d, path, cfg)
	a.setFileCount(fileCount)

	// Add to recent vaults list
	a.addToRecentVaults(path, cfg.Name)

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
	v := a.vault()
	if v.db == nil {
		return nil
	}

	// Clean up orphan thumbnails before closing. Runs on the snapshot with
	// no lock held — CleanupOrphanThumbnails takes its own read lock.
	if _, err := a.CleanupOrphanThumbnails(); err != nil {
		fmt.Printf("orphan thumbnail cleanup warning: %v\n", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.db != v.db {
		return nil // vault switched concurrently — leave the new one alone
	}

	// WAL checkpoint: flush pending changes to main DB and truncate WAL file.
	// This keeps the DB compact across sessions and prevents WAL file growth.
	_, _ = a.db.Conn().Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	if err := a.db.Close(); err != nil {
		return fmt.Errorf("failed to close vault: %w", err)
	}
	a.db = nil
	a.vaultPath = ""
	a.vaultCfg = nil
	a.fileCount = 0
	return nil
}

// GetVaultConfig returns the current vault configuration.
func (a *App) GetVaultConfig() (*config.VaultConfig, error) {
	v := a.vault()
	if v.cfg == nil {
		return nil, fmt.Errorf("no vault open")
	}
	return v.cfg, nil
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
	if err := a.closeCurrentVault(); err != nil {
		return nil, fmt.Errorf("failed to close previous vault: %w", err)
	}

	// Create .tagloom directory
	if err := os.MkdirAll(tagloomDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .tagloom directory: %w", err)
	}

	// Create SQLite database
	dbPath := filepath.Join(tagloomDir, "tagloom.db")
	d, err := db.NewDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Seed default tags
	if err := d.SeedDefaultTags(); err != nil {
		d.Close()
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
		d.Close()
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	// Insert excluded folders into DB
	for _, ef := range settings.ExcludedFolders {
		_, _ = d.Conn().Exec(
			"INSERT OR IGNORE INTO excluded_folders (path) VALUES (?)",
			ef,
		)
	}

	// Atomically install as current vault (also saves last-vault path)
	a.setVault(d, path, cfg)
	a.setFileCount(0) // new vault — the auto-scan below refreshes it

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
// Holds the write lock for the whole body (no other locking calls inside).
func (a *App) SetVaultConfig(cfg *config.VaultConfig) error {
	a.mu.Lock()
	if a.vaultPath == "" {
		a.mu.Unlock()
		return fmt.Errorf("no vault open")
	}
	path := a.vaultPath
	nameChanged := a.vaultCfg != nil && cfg.Name != a.vaultCfg.Name && cfg.Name != ""
	a.mu.Unlock()

	if err := config.SaveConfig(path, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// If the vault name changed, update recent vaults entry too
	if nameChanged && a.appCfg != nil {
		for i := range a.appCfg.RecentVaults {
			if a.appCfg.RecentVaults[i].Path == path {
				a.appCfg.RecentVaults[i].Name = cfg.Name
				break
			}
		}
		_ = config.SaveAppSettings(a.appCfg)
	}

	a.vaultCfg = cfg
	return nil
}

// migrateToRelativePaths converts absolute vault_path and folder_path values
// to relative paths from the vault root. This allows the vault to be moved
// to a different location without breaking file references.
// Also migrates excluded_folders paths to relative.
// No-op for new vaults (all paths already relative).
func (a *App) migrateToRelativePaths(d *db.Database, vaultPath string) error {
	if d == nil || vaultPath == "" {
		return nil
	}

	// Check if any files have absolute paths
	var count int
	err := d.Conn().QueryRow(`
		SELECT COUNT(*) FROM files WHERE vault_path LIKE '/%' OR vault_path LIKE '%:%'
	`).Scan(&count)
	if err != nil || count == 0 {
		return nil // No absolute paths found (new vault or already migrated)
	}

	fmt.Printf("migrating %d file records from absolute to relative paths\n", count)

	// Use a single UPDATE with SQLite string functions
	// vault_path: strip the vault prefix to get relative path
	// folder_path: strip the vault prefix to get relative path
	vaultPrefix := vaultPath + string(filepath.Separator)
	vaultPrefixLen := len(vaultPrefix)

	_, err = d.Conn().Exec(`
		UPDATE files SET
			vault_path = SUBSTR(vault_path, ?),
			folder_path = CASE
				WHEN SUBSTR(folder_path, ?) = '' THEN '.'
				ELSE SUBSTR(folder_path, ?)
			END
		WHERE vault_path LIKE ?
	`, vaultPrefixLen+1, vaultPrefixLen+1, vaultPrefixLen+1, vaultPrefix+"%")
	if err != nil {
		return fmt.Errorf("failed to migrate file paths: %w", err)
	}

	// Also migrate excluded_folders if they have absolute paths
	_, _ = d.Conn().Exec(`
		UPDATE excluded_folders SET path = SUBSTR(path, ?)
		WHERE path LIKE ?
	`, vaultPrefixLen+1, vaultPrefix+"%")

	return nil
}
