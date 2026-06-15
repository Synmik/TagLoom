package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"TagLoom/config"
	"TagLoom/db"
	"TagLoom/utils"
)

const maxRecentVaults = 10

// App is the main Wails application struct.
// It coordinates vault, scanner, thumbnailer, search, tags, and metadata operations.
type App struct {
	ctx      context.Context
	db       *db.Database
	vaultPath string
	vaultCfg *config.VaultConfig
	appCfg   *config.AppSettings
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// resolvePath converts a relative path (stored in DB) to an absolute path.
// If the path is already absolute (legacy data), returns it as-is.
func (a *App) resolvePath(relPath string) string {
	if filepath.IsAbs(relPath) {
		return relPath
	}
	return filepath.Join(a.vaultPath, relPath)
}

// toRelativePath converts an absolute path to a relative path from the vault root.
// If the path is already relative, returns it as-is.
func (a *App) toRelativePath(absPath string) string {
	if !filepath.IsAbs(absPath) {
		return absPath
	}
	rel, err := filepath.Rel(a.vaultPath, absPath)
	if err != nil {
		return absPath
	}
	return rel
}

// generateThumbnailAbsolutePath returns the absolute path for a thumbnail
// given a relative file path. The hash is computed from the relative path
// so thumbnails remain valid when the vault is moved.
func (a *App) generateThumbnailAbsolutePath(relFilePath string) string {
	hash := utils.HashPath(relFilePath)
	subdir := utils.ThumbnailSubdir(hash)
	thumbDir := filepath.Join(a.vaultPath, ".tagloom", "thumbnails", subdir)
	os.MkdirAll(thumbDir, 0755)
	return filepath.Join(thumbDir, hash+".webp")
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Load global app settings (last vault path, etc.)
	settings, err := config.LoadAppSettings()
	if err != nil {
		// Non-fatal: proceed with empty settings
		settings = &config.AppSettings{}
	}
	a.appCfg = settings
}

// GetLastVaultPath returns the path of the last opened vault (from global app settings).
// The frontend calls this on mount to decide whether to auto-open.
func (a *App) GetLastVaultPath() string {
	if a.appCfg == nil {
		return ""
	}
	return a.appCfg.LastVaultPath
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

// GetRecentVaults returns the list of recently opened vaults.
func (a *App) GetRecentVaults() []config.RecentVault {
	if a.appCfg == nil {
		return nil
	}
	if a.appCfg.RecentVaults == nil {
		return []config.RecentVault{}
	}
	return a.appCfg.RecentVaults
}

// addToRecentVaults adds a vault path to the recent vaults list.
// Moves existing entries to the back, keeps maxRecentVaults entries.
func (a *App) addToRecentVaults(path string, name string) {
	if a.appCfg == nil {
		return
	}

	// Remove existing entry for this path
	var filtered []config.RecentVault
	for _, v := range a.appCfg.RecentVaults {
		if v.Path != path {
			filtered = append(filtered, v)
		}
	}

	// Check if .tagloom still exists (vault is still valid)
	tagloomDir := path + "/.tagloom"
	if _, err := os.Stat(tagloomDir); os.IsNotExist(err) {
		// Vault folder no longer exists — don't add to list
		// But also clean up from existing entries
		a.appCfg.RecentVaults = filtered
		_ = config.SaveAppSettings(a.appCfg)
		return
	}

	// Create new entry at front of list
	entry := config.RecentVault{
		Path:     path,
		Name:     name,
		OpenedAt: time.Now().Format(time.RFC3339),
	}

	// Prepend + limit to max
	result := append([]config.RecentVault{entry}, filtered...)
	if len(result) > maxRecentVaults {
		result = result[:maxRecentVaults]
	}

	a.appCfg.RecentVaults = result
	_ = config.SaveAppSettings(a.appCfg)
}

// RemoveRecentVault removes a vault path from the recent vaults list.
func (a *App) RemoveRecentVault(path string) error {
	if a.appCfg == nil {
		return fmt.Errorf("app settings not loaded")
	}

	var filtered []config.RecentVault
	for _, v := range a.appCfg.RecentVaults {
		if v.Path != path {
			filtered = append(filtered, v)
		}
	}

	// Also clear last_vault_path if it matches
	if a.appCfg.LastVaultPath == path {
		a.appCfg.LastVaultPath = ""
	}

	a.appCfg.RecentVaults = filtered
	if err := config.SaveAppSettings(a.appCfg); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
	return nil
}

// FFmpegStatus holds the install status of ffmpeg and ffprobe.
type FFmpegStatus struct {
	FFmpegPath  string `json:"ffmpeg_path"`
	FFprobePath string `json:"ffprobe_path"`
	FFmpegOK    bool   `json:"ffmpeg_ok"`
	FFprobeOK   bool   `json:"ffprobe_ok"`
}

// CheckFFmpeg returns the install status of ffmpeg and ffprobe.
func (a *App) CheckFFmpeg() *FFmpegStatus {
	status := &FFmpegStatus{}

	ffmpegPath, err := utils.FindFFmpeg()
	if err == nil {
		status.FFmpegPath = ffmpegPath
		status.FFmpegOK = true
	}

	ffprobePath, err := utils.FindFFprobe()
	if err == nil {
		status.FFprobePath = ffprobePath
		status.FFprobeOK = true
	}

	return status
}

// GetAppInfo returns basic application info for the About dialog.
func (a *App) GetAppInfo() map[string]string {
	return map[string]string{
		"os":   runtime.GOOS,
		"arch": runtime.GOARCH,
		"go":   runtime.Version(),
	}
}

// GetVersion returns the application version.
// Keep in sync with wails.json → info.productVersion.
const appVersion = "0.3.0"

func (a *App) GetVersion() string {
	return appVersion
}
