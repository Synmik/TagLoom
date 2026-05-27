package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"TagLoom/config"
	"TagLoom/db"
)

// OpenVault opens an existing vault or creates a new one at the given path.
func (a *App) OpenVault(path string) (*db.VaultInfo, error) {
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
		cfg.CreatedAt = time.Now().Format(time.RFC3339)
		if err := config.SaveConfig(path, cfg); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}
	}

	// Get vault name from path
	vaultName := filepath.Base(path)

	// Count indexed files
	fileCount := 0
	if a.db != nil {
		a.db.Conn().QueryRow("SELECT COUNT(*) FROM files").Scan(&fileCount)
	}

	return &db.VaultInfo{
		Path:      path,
		Name:      vaultName,
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
	return nil
}

// GetVaultConfig returns the current vault configuration.
func (a *App) GetVaultConfig() (*config.VaultConfig, error) {
	// TODO: Implement - read from current vault path
	return nil, fmt.Errorf("not implemented")
}

// SetVaultConfig updates the vault configuration.
func (a *App) SetVaultConfig(config *config.VaultConfig) error {
	// TODO: Implement - write to current vault path
	return fmt.Errorf("not implemented")
}
