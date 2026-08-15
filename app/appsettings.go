package app

import (
	"fmt"

	"TagLoom/config"
)

// GetAppSettings returns the current global application settings.
// Returns a copy — the underlying settings may be replaced concurrently
// by SetAppSettings.
func (a *App) GetAppSettings() (*config.AppSettings, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.appCfg == nil {
		return nil, fmt.Errorf("app settings not loaded")
	}
	s := *a.appCfg
	return &s, nil
}

// SetAppSettings updates the global application settings and persists them to disk.
func (a *App) SetAppSettings(settings *config.AppSettings) error {
	if settings == nil {
		return fmt.Errorf("settings is nil")
	}
	a.mu.Lock()
	a.appCfg = settings
	a.mu.Unlock()
	if err := config.SaveAppSettings(settings); err != nil {
		return fmt.Errorf("failed to save app settings: %w", err)
	}
	return nil
}
