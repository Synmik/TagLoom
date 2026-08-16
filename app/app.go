package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"TagLoom/config"
	"TagLoom/db"
	"TagLoom/utils"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const maxRecentVaults = 10

// App is the main Wails application struct.
// It coordinates vault, scanner, thumbnailer, search, tags, and metadata operations.
//
// Concurrency: Wails invokes binding methods from arbitrary goroutines, and the
// HTTP middleware serves thumbnails concurrently. mu guards all mutable state
// (vault state + app settings). Readers take a snapshot via vault() under the
// read lock and use it for the whole call; writers (OpenVault, CreateVault,
// CloseVault, SetVaultConfig) do slow I/O outside the lock and swap state
// under the write lock. Never call a locking method while holding the lock.
type App struct {
	ctx context.Context

	mu        sync.RWMutex
	db        *db.Database
	vaultPath string
	vaultCfg  *config.VaultConfig
	// fileCount is the cached files-table row count for the current vault,
	// maintained on vault install and every files-table mutation so
	// GetCurrentVault never has to run COUNT(*) on the (potentially huge)
	// files table.
	fileCount int
	appCfg    *config.AppSettings
	// thumbCancel cancels the currently running thumbnail worker pool, if
	// any (see CancelThumbnailGeneration). thumbCancelGen disambiguates
	// which pool registered it (Go functions are not comparable).
	thumbCancel    context.CancelFunc
	thumbCancelGen int
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// vault is an immutable snapshot of the vault state, taken under the read
// lock. Methods that need vault state take one snapshot at entry and use it
// throughout, so a concurrent vault switch can never mix two vaults mid-call.
type vault struct {
	db        *db.Database
	path      string
	cfg       *config.VaultConfig
	fileCount int
}

// vault returns the current vault state as a consistent snapshot.
func (a *App) vault() vault {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return vault{db: a.db, path: a.vaultPath, cfg: a.vaultCfg, fileCount: a.fileCount}
}

// setVault atomically installs a new vault as the current one.
// Callers must have completed all slow I/O (open DB, load config) beforehand.
// The cached file count is reset to 0; install sites set it via setFileCount
// immediately after.
func (a *App) setVault(d *db.Database, path string, cfg *config.VaultConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.db = d
	a.vaultPath = path
	a.vaultCfg = cfg
	a.fileCount = 0
	if a.appCfg != nil {
		a.appCfg.LastVaultPath = path
	}
}

// closeCurrentVault closes the current vault (if any) and clears vault state.
// Used when switching vaults; does NOT run the WAL checkpoint / thumbnail
// cleanup that user-initiated CloseVault performs.
func (a *App) closeCurrentVault() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.db == nil {
		return nil
	}
	err := a.db.Close()
	a.db = nil
	a.vaultPath = ""
	a.vaultCfg = nil
	a.fileCount = 0
	return err
}

// setFileCount updates the cached files-table row count for the current vault.
func (a *App) setFileCount(n int) {
	a.mu.Lock()
	a.fileCount = n
	a.mu.Unlock()
}

// adjustFileCount bumps the cached file count by delta (clamped at 0).
func (a *App) adjustFileCount(delta int) {
	a.mu.Lock()
	a.fileCount += delta
	if a.fileCount < 0 {
		a.fileCount = 0
	}
	a.mu.Unlock()
}

// refreshFileCount recomputes the cached file count with a single COUNT(*)
// query. Called after bulk operations (scans, imports) where incremental
// bookkeeping would be fragile.
func (a *App) refreshFileCount(v vault) {
	if v.db == nil {
		return
	}
	var n int
	if err := v.db.Conn().QueryRow("SELECT COUNT(*) FROM files").Scan(&n); err == nil {
		a.setFileCount(n)
	}
}

// resolvePath converts a relative path (stored in DB) to an absolute path.
// If the path is already absolute (legacy data), returns it as-is.
func (v vault) resolvePath(relPath string) string {
	if filepath.IsAbs(relPath) {
		return relPath
	}
	return filepath.Join(v.path, relPath)
}

// toRelativePath converts an absolute path to a relative path from the vault root.
// If the path is already relative, returns it as-is.
func (v vault) toRelativePath(absPath string) string {
	if !filepath.IsAbs(absPath) {
		return absPath
	}
	rel, err := filepath.Rel(v.path, absPath)
	if err != nil {
		return absPath
	}
	return rel
}

// generateThumbnailAbsolutePath returns the absolute path for a thumbnail
// given a relative file path. The hash is computed from the relative path
// so thumbnails remain valid when the vault is moved.
func (v vault) generateThumbnailAbsolutePath(relFilePath string) string {
	hash := utils.HashPath(relFilePath)
	subdir := utils.ThumbnailSubdir(hash)
	thumbDir := filepath.Join(v.path, ".tagloom", "thumbnails", subdir)
	os.MkdirAll(thumbDir, 0755)
	return filepath.Join(thumbDir, hash+".webp")
}

// emitEvent emits a Wails runtime event to the frontend.
// It is a no-op when no app context is set (e.g. in unit tests).
func (a *App) emitEvent(name string, data interface{}) {
	if a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, name, data)
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

	a.mu.Lock()
	a.appCfg = settings
	a.mu.Unlock()
}

// GetLastVaultPath returns the path of the last opened vault (from global app settings).
// The frontend calls this on mount to decide whether to auto-open.
func (a *App) GetLastVaultPath() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.appCfg == nil {
		return ""
	}
	return a.appCfg.LastVaultPath
}

// GetCurrentVault returns information about the currently open vault.
// The file count is served from the in-memory cache (see setFileCount /
// adjustFileCount / refreshFileCount) — no COUNT(*) per call.
func (a *App) GetCurrentVault() *db.VaultInfo {
	v := a.vault()
	if v.db == nil || v.path == "" {
		return nil
	}

	name, createdAt := "", ""
	if v.cfg != nil {
		name, createdAt = v.cfg.Name, v.cfg.CreatedAt
	}

	return &db.VaultInfo{
		Path:      v.path,
		Name:      name,
		CreatedAt: createdAt,
		FileCount: v.fileCount,
	}
}

// GetRecentVaults returns the list of recently opened vaults.
func (a *App) GetRecentVaults() []config.RecentVault {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.appCfg == nil || a.appCfg.RecentVaults == nil {
		return []config.RecentVault{}
	}
	return a.appCfg.RecentVaults
}

// addToRecentVaults adds a vault path to the recent vaults list.
// Moves existing entries to the back, keeps maxRecentVaults entries.
func (a *App) addToRecentVaults(path string, name string) {
	a.mu.Lock()
	if a.appCfg == nil {
		a.mu.Unlock()
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
		a.mu.Unlock()
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
	a.mu.Unlock()
}

// RemoveRecentVault removes a vault path from the recent vaults list.
func (a *App) RemoveRecentVault(path string) error {
	a.mu.Lock()
	if a.appCfg == nil {
		a.mu.Unlock()
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
	err := config.SaveAppSettings(a.appCfg)
	a.mu.Unlock()
	if err != nil {
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
const appVersion = "0.4.0"

func (a *App) GetVersion() string {
	return appVersion
}
