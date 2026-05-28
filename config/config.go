package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// VaultConfig holds vault-level settings.
type VaultConfig struct {
	Name      string   `json:"name"`
	Version   int      `json:"version"`
	CreatedAt string   `json:"created_at"`
	Settings  Settings `json:"settings"`
}

// Settings holds user-configurable options.
type Settings struct {
	AutoTagByFolder    bool   `json:"auto_tag_by_folder"`
	ExcludedFolders    []string `json:"excluded_folders"`
	ThumbnailSize      int    `json:"thumbnail_size"`
	ThumbnailQuality   int    `json:"thumbnail_quality"`
	DefaultSortField   string `json:"default_sort_field"`
	DefaultSortOrder   string `json:"default_sort_order"`
	GridThumbnailSize  string `json:"grid_thumbnail_size"`
}

// DefaultConfig returns a new VaultConfig with default values.
func DefaultConfig() *VaultConfig {
	return &VaultConfig{
		Version: 1,
		Settings: Settings{
			ThumbnailSize:      256,
			ThumbnailQuality:   80,
			DefaultSortField:   "indexed_at",
			DefaultSortOrder:   "desc",
			GridThumbnailSize:  "medium",
		},
	}
}

// LoadConfig reads the vault config from disk.
func LoadConfig(vaultPath string) (*VaultConfig, error) {
	configPath := filepath.Join(vaultPath, ".tagloom", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var config VaultConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveConfig writes the vault config to disk.
func SaveConfig(vaultPath string, config *VaultConfig) error {
	configDir := filepath.Join(vaultPath, ".tagloom")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.json")
	return os.WriteFile(configPath, data, 0644)
}
