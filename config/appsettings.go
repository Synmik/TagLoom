package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppSettings holds global application preferences stored outside any vault.
type AppSettings struct {
	LastVaultPath      string `json:"last_vault_path"`
	AutoOpenLastVault  bool   `json:"auto_open_last_vault"`
	DefaultGridSize    string `json:"default_grid_size"`
	DefaultSortField   string `json:"default_sort_field"`
	DefaultSortOrder   string `json:"default_sort_order"`
	ConfirmBeforeExit  bool   `json:"confirm_before_exit"`
}

// AppSettingsDir returns the directory for global app settings.
// Windows: %APPDATA%/TagLoom
// macOS:   ~/Library/Application Support/TagLoom
// Linux:   ~/.config/TagLoom
func AppSettingsDir() (string, error) {
	// Wails runtime provides the user's home directory via the App struct,
	// but we can use os.UserConfigDir which maps to the right location per OS.
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "TagLoom"), nil
}

// LoadAppSettings reads global app settings from disk.
// Returns a zero-value AppSettings if the file doesn't exist yet.
func LoadAppSettings() (*AppSettings, error) {
	settingsDir, err := AppSettingsDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(settingsDir, "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AppSettings{}, nil
		}
		return nil, err
	}

	var settings AppSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// SaveAppSettings writes global app settings to disk.
func SaveAppSettings(settings *AppSettings) error {
	settingsDir, err := AppSettingsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(settingsDir, "settings.json")
	return os.WriteFile(path, data, 0644)
}
