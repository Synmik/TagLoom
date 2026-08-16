package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"TagLoom/utils"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var ffmpegAvailable = false

func init() {
	if _, err := utils.FindFFmpeg(); err == nil {
		ffmpegAvailable = true
	}
}

const (
	defaultThumbnailSize    = 256
	defaultThumbnailQuality = 80
	minWorkers              = 4
	maxWorkers              = 16
)

// thumbnailWorkerCount returns the number of thumbnail workers based on
// available CPU cores, clamped to [minWorkers, maxWorkers].
func thumbnailWorkerCount() int {
	n := runtime.NumCPU()
	if n < minWorkers {
		return minWorkers
	}
	if n > maxWorkers {
		return maxWorkers
	}
	return n
}

// GenerateThumbnail creates a thumbnail for the given file ID.
// Thumbnails are stored in .tagloom/thumbnails/{2char_hash}/{hash}.webp
func (a *App) GenerateThumbnail(fileID int64) (string, error) {
	v := a.vault()
	if v.db == nil {
		return "", fmt.Errorf("no vault open")
	}
	if v.path == "" {
		return "", fmt.Errorf("no vault path set")
	}

	// Fetch file info from DB (vault_path is relative)
	row := v.db.Conn().QueryRow(`
		SELECT id, vault_path, thumbnail_path
		FROM files WHERE id = ?
	`, fileID)

	var id int64
	var relVaultPath string
	var existingThumb *string // nullable in DB
	if err := row.Scan(&id, &relVaultPath, &existingThumb); err != nil {
		return "", fmt.Errorf("failed to find file %d: %w", fileID, err)
	}

	// Resolve relative path to absolute for file operations
	absPath := v.resolvePath(relVaultPath)

	// Dereference nullable thumbnail path (stored as relative to vault root)
	thumbStr := ""
	if existingThumb != nil {
		thumbStr = *existingThumb
	}

	// Check if file actually exists on disk
	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("file not found on disk: %w", err)
	}

	// Determine target thumbnail path
	// Hash is based on relative path (portable across vault moves)
	thumbPath := v.generateThumbnailAbsolutePath(relVaultPath)

	// If thumbnail already exists, check if source is newer
	// thumbStr is relative (from DB), resolve it for comparison
	if v.resolvePath(thumbStr) == thumbPath {
		thumbStat, err1 := os.Stat(thumbPath)
		srcStat, err2 := os.Stat(absPath)
		if err1 == nil && err2 == nil && !srcStat.ModTime().After(thumbStat.ModTime()) {
			return thumbPath, nil // Thumbnail is up to date
		}
	}

	// Determine file category
	ext := strings.ToLower(filepath.Ext(absPath))
	category := utils.GetFileCategory(absPath)

	// Get config for thumbnail size/quality (needed before switch for video case)
	size := defaultThumbnailSize
	quality := defaultThumbnailQuality
	if v.cfg != nil {
		if v.cfg.Settings.ThumbnailSize > 0 {
			size = v.cfg.Settings.ThumbnailSize
		}
		if v.cfg.Settings.ThumbnailQuality > 0 {
			quality = v.cfg.Settings.ThumbnailQuality
		}
	}

	switch category {
	case "image", "animated":
		if !ffmpegAvailable {
			return "", fmt.Errorf("ffmpeg not found — cannot generate thumbnail for %s", ext)
		}
		// Ensure output directory exists
		thumbDir := filepath.Dir(thumbPath)
		if err := os.MkdirAll(thumbDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create thumbnail directory: %w", err)
		}
		var encErr error
		if ext == ".svg" {
			encErr = utils.EncodeSVGToWebP(absPath, thumbPath, size, quality)
		} else {
			encErr = utils.EncodeImageToWebP(absPath, thumbPath, size, quality)
		}
		if encErr != nil {
			os.Remove(thumbPath)
			return "", fmt.Errorf("failed to encode image to WebP: %w", encErr)
		}
		// Update DB with thumbnail path (store relative to vault root)
		relThumbPath := v.toRelativePath(thumbPath)
		_, execErr := v.db.Conn().Exec("UPDATE files SET thumbnail_path = ? WHERE id = ?", relThumbPath, fileID)
		if execErr != nil {
			return "", fmt.Errorf("failed to update thumbnail_path: %w", execErr)
		}
		return thumbPath, nil
	case "video":
		if !ffmpegAvailable {
			return "", fmt.Errorf("ffmpeg not found — cannot generate video thumbnail for %s", ext)
		}
		// Ensure output directory exists
		thumbDir := filepath.Dir(thumbPath)
		if err := os.MkdirAll(thumbDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create thumbnail directory: %w", err)
		}
		if err := utils.ExtractVideoFrame(absPath, thumbPath, size, utils.DefaultVideoThumbTimestamp); err != nil {
			os.Remove(thumbPath)
			return "", fmt.Errorf("failed to extract video frame: %w", err)
		}
		// For video, skip the image resize/encode path — FFmpeg already produced the WebP
		// Update DB with thumbnail path (store relative to vault root)
		relThumbPath := v.toRelativePath(thumbPath)
		_, execErr := v.db.Conn().Exec("UPDATE files SET thumbnail_path = ? WHERE id = ?", relThumbPath, fileID)
		if execErr != nil {
			return "", fmt.Errorf("failed to update thumbnail_path: %w", execErr)
		}
		return thumbPath, nil
	default:
		return "", fmt.Errorf("thumbnail generation not supported for %s files", ext)
	}
}

// thumbResult holds the outcome of processing one file's thumbnail.
type thumbResult struct {
	fileID    int64
	thumbPath string
	ok        bool // true if thumbnail was generated, false if failed
}

// GenerateThumbnailsPool generates thumbnails for all files that need them
// using a worker pool of 4 goroutines. Emits progress events to the frontend.
//
// DB writes are batched after all workers finish to avoid SQLITE_BUSY from
// concurrent writers — SQLite (even in WAL mode) only supports one writer
// at a time.
func (a *App) GenerateThumbnailsPool() error {
	v := a.vault()
	if v.db == nil {
		return fmt.Errorf("no vault open")
	}
	if v.path == "" {
		return fmt.Errorf("no vault path set")
	}

	// Collect all file IDs that need thumbnails.
	// No pre-flight os.Stat — workers handle missing files gracefully.
	rows, err := v.db.Conn().Query(`
		SELECT id, vault_path, thumbnail_path
		FROM files
		WHERE thumbnail_path = ''
		   OR thumbnail_path IS NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	type pendingFile struct {
		id        int64
		vaultPath string
	}

	var pending []pendingFile
	for rows.Next() {
		var f pendingFile
		var existingThumb *string // nullable in DB
		if err := rows.Scan(&f.id, &f.vaultPath, &existingThumb); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		pending = append(pending, f)
	}

	total := len(pending)
	if total == 0 {
		wailsruntime.EventsEmit(a.ctx, "thumb:complete", map[string]int{
			"generated": 0,
			"failed":    0,
			"total":     0,
		})
		return nil
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	// Channel for distributing work
	jobs := make(chan pendingFile, total)

	// Collect results from workers
	var (
		resultsMu   sync.Mutex
		results     []thumbResult
		processedMu sync.Mutex
		processed   int
	)

	// finishFile appends a result and updates progress (thread-safe).
	finishFile := func(fileID int64, ok bool, thumbPath string) {
		resultsMu.Lock()
		results = append(results, thumbResult{fileID: fileID, thumbPath: thumbPath, ok: ok})
		resultsMu.Unlock()

		processedMu.Lock()
		processed++
		cur := processed
		processedMu.Unlock()

		if cur%10 == 0 || cur == total {
			wailsruntime.EventsEmit(a.ctx, "thumb:progress", map[string]int{
				"current": cur,
				"total":   total,
			})
		}
	}

	// Worker function — does image I/O only, returns results for batched DB update
	worker := func(workerID int, jobs <-chan pendingFile, wg *sync.WaitGroup) {
		defer wg.Done()

		size := defaultThumbnailSize
		quality := defaultThumbnailQuality
		if v.cfg != nil {
			if v.cfg.Settings.ThumbnailSize > 0 {
				size = v.cfg.Settings.ThumbnailSize
			}
			if v.cfg.Settings.ThumbnailQuality > 0 {
				quality = v.cfg.Settings.ThumbnailQuality
			}
		}

		for f := range jobs {
			// Check for cancellation
			select {
			case <-ctx.Done():
				return
			default:
			}

			// f.vaultPath is relative — resolve for file operations
			absPath := v.resolvePath(f.vaultPath)
			// Hash based on relative path (portable across vault moves)
			thumbPath := v.generateThumbnailAbsolutePath(f.vaultPath)

			// Skip if already generated (another worker may have done it)
			if _, err := os.Stat(thumbPath); err == nil {
				finishFile(f.id, true, thumbPath)
				continue
			}

			category := utils.GetFileCategory(absPath)

			switch category {
			case "image", "animated":
				if !ffmpegAvailable {
					finishFile(f.id, false, "")
					continue
				}
				// Encode via FFmpeg (or Go-side SVG rasterization)
				thumbDir := filepath.Dir(thumbPath)
				if mkErr := os.MkdirAll(thumbDir, 0755); mkErr != nil {
					finishFile(f.id, false, "")
					continue
				}
				ext := strings.ToLower(filepath.Ext(absPath))
				var encErr error
				if ext == ".svg" {
					encErr = utils.EncodeSVGToWebP(absPath, thumbPath, size, quality)
				} else {
					encErr = utils.EncodeImageToWebP(absPath, thumbPath, size, quality)
				}
				if encErr != nil {
					os.Remove(thumbPath)
					finishFile(f.id, false, "")
					continue
				}
				// Image thumbnail generated successfully
				finishFile(f.id, true, thumbPath)
				continue
			case "video":
				if !ffmpegAvailable {
					finishFile(f.id, false, "")
					continue
				}
				// Generate via FFmpeg
				thumbDir := filepath.Dir(thumbPath)
				if mkErr := os.MkdirAll(thumbDir, 0755); mkErr != nil {
					finishFile(f.id, false, "")
					continue
				}
				if ffErr := utils.ExtractVideoFrame(absPath, thumbPath, size, utils.DefaultVideoThumbTimestamp); ffErr != nil {
					os.Remove(thumbPath)
					finishFile(f.id, false, "")
					continue
				}
				// Video thumbnail generated successfully
				finishFile(f.id, true, thumbPath)
				continue
			default:
				// Unsupported
				finishFile(f.id, false, "")
				continue
			}
		}
	}

	// Start workers — count scales with CPU cores
	workers := thumbnailWorkerCount()
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker(i, jobs, &wg)
	}

	// Feed jobs
	for _, f := range pending {
		jobs <- f
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()

	// Batch DB update — sequential writes avoid SQLITE_BUSY entirely
	// Store thumbnail_path as relative to vault root
	tx, err := v.db.Conn().Begin()
	if err == nil {
		stmt, prepErr := tx.Prepare("UPDATE files SET thumbnail_path = ? WHERE id = ?")
		if prepErr == nil {
			for _, r := range results {
				if r.ok {
					relThumbPath := v.toRelativePath(r.thumbPath)
					_, _ = stmt.Exec(relThumbPath, r.fileID)
				}
			}
			stmt.Close()
		}
		if commitErr := tx.Commit(); commitErr != nil {
			fmt.Printf("thumbnail batch commit warning: %v\n", commitErr)
		}
	}

	// Compute final counts
	generated := 0
	failed := 0
	for _, r := range results {
		if r.ok {
			generated++
		} else {
			failed++
		}
	}

	// Emit completion event
	wailsruntime.EventsEmit(a.ctx, "thumb:complete", map[string]int{
		"generated": generated,
		"failed":    failed,
		"total":     total,
	})

	return nil
}



// CancelThumbnailGeneration cancels an ongoing thumbnail generation pool.
func (a *App) CancelThumbnailGeneration() {
	// This will be wired to a cancel context in the future
	wailsruntime.EventsEmit(a.ctx, "thumb:cancelled", true)
}

// decodeImage opens and decodes an image file, supporting multiple formats.

// GenerateThumbnailsForFiles generates thumbnails for a list of file IDs.
// Returns the number of thumbnails generated.
func (a *App) GenerateThumbnailsForFiles(fileIDs []int64) int {
	count := 0
	for _, id := range fileIDs {
		_, err := a.GenerateThumbnail(id)
		if err == nil {
			count++
		}
	}
	return count
}

// CleanupOrphanThumbnails removes thumbnail files whose source file is no longer
// in the database. It walks .tagloom/thumbnails/ and deletes any .webp not referenced
// by files.thumbnail_path. Returns the number of orphan files removed.
func (a *App) CleanupOrphanThumbnails() (int, error) {
	v := a.vault()
	if v.db == nil {
		return 0, fmt.Errorf("no vault open")
	}
	if v.path == "" {
		return 0, fmt.Errorf("no vault path set")
	}

	thumbRoot := filepath.Join(v.path, ".tagloom", "thumbnails")
	if _, err := os.Stat(thumbRoot); os.IsNotExist(err) {
		return 0, nil // No thumbnails directory yet
	}

	// Collect all valid thumbnail paths from the database
	// thumbnail_path is stored as relative to vault root; resolve to absolute
	// for comparison with absolute paths from filepath.WalkDir
	validThumbs := make(map[string]struct{})
	rows, err := v.db.Conn().Query("SELECT thumbnail_path FROM files WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''")
	if err != nil {
		return 0, fmt.Errorf("failed to query thumbnail paths: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tp string
		if err := rows.Scan(&tp); err == nil && tp != "" {
			absThumb := v.resolvePath(tp)
			validThumbs[absThumb] = struct{}{}
		}
	}

	// Walk the thumbnails directory and remove orphans
	removed := 0
	err = filepath.WalkDir(thumbRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}
		if d.IsDir() {
			return nil // Will remove empty dirs after the walk
		}
		if strings.HasSuffix(strings.ToLower(path), ".webp") {
			if _, ok := validThumbs[path]; !ok {
				if err := os.Remove(path); err == nil {
					removed++
				}
			}
		}
		return nil
	})
	if err != nil {
		return removed, fmt.Errorf("failed to walk thumbnails directory: %w", err)
	}

	// Remove empty subdirectories (left behind after orphan deletion)
	filepath.WalkDir(thumbRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == thumbRoot {
			return nil
		}
		entries, _ := os.ReadDir(path)
		if len(entries) == 0 {
			os.Remove(path)
		}
		return nil
	})

	if removed > 0 {
		fmt.Printf("cleaned up %d orphan thumbnails\n", removed)
	}
	return removed, nil
}

// GetThumbnailPath returns the absolute thumbnail path for a file ID.
// Used by the HTTP handler to serve thumbnails.
// The thumbnail_path in DB is relative; this resolves it to an absolute path.
func (a *App) GetThumbnailPath(fileID int64) (string, error) {
	v := a.vault()
	if v.db == nil {
		return "", fmt.Errorf("no vault open")
	}

	row := v.db.Conn().QueryRow("SELECT thumbnail_path FROM files WHERE id = ?", fileID)
	var thumbPath *string // nullable in DB
	if err := row.Scan(&thumbPath); err != nil {
		return "", fmt.Errorf("no thumbnail for file %d: %w", fileID, err)
	}
	if thumbPath == nil || *thumbPath == "" {
		return "", fmt.Errorf("no thumbnail path for file %d", fileID)
	}

	// Resolve relative path to absolute
	absThumb := v.resolvePath(*thumbPath)

	// Verify file exists
	if _, err := os.Stat(absThumb); err != nil {
		return "", fmt.Errorf("thumbnail file not found: %w", err)
	}

	return absThumb, nil
}

// ThumbnailInfo holds metadata about a thumbnail.
type ThumbnailInfo struct {
	FileID        int64  `json:"file_id"`
	ThumbnailPath string `json:"thumbnail_path"`
	Exists        bool   `json:"exists"`
	SizeBytes     int64  `json:"size_bytes"`
}

// GetThumbnailInfo returns metadata about a thumbnail.
func (a *App) GetThumbnailInfo(fileID int64) (*ThumbnailInfo, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	row := v.db.Conn().QueryRow("SELECT thumbnail_path FROM files WHERE id = ?", fileID)
	var thumbPath *string // nullable in DB
	if err := row.Scan(&thumbPath); err != nil {
		return nil, fmt.Errorf("no thumbnail for file %d: %w", fileID, err)
	}

	info := &ThumbnailInfo{
		FileID: fileID,
	}

	if thumbPath != nil && *thumbPath != "" {
		absThumb := v.resolvePath(*thumbPath)
		info.ThumbnailPath = absThumb
		stat, err := os.Stat(absThumb)
		if err == nil {
			info.Exists = true
			info.SizeBytes = stat.Size()
		}
	}

	return info, nil
}



