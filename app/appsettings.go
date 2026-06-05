package app

import (
	"fmt"

	"TagLoom/config"
)

// GetAppSettings returns the current global application settings.
func (a *App) GetAppSettings() (*config.AppSettings, error) {
	if a.appCfg == nil {
		return nil, fmt.Errorf("app settings not loaded")
	}
	return a.appCfg, nil
}

// SetAppSettings updates the global application settings and persists them to disk.
func (a *App) SetAppSettings(settings *config.AppSettings) error {
	if settings == nil {
		return fmt.Errorf("settings is nil")
	}
	a.appCfg = settings
	if err := config.SaveAppSettings(a.appCfg); err != nil {
		return fmt.Errorf("failed to save app settings: %w", err)
	}
	return nil
}
