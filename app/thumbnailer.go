package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"TagLoom/utils"

	"github.com/disintegration/imaging"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

const (
	defaultThumbnailSize  = 256
	defaultThumbnailQuality = 80
	thumbnailWorkerCount  = 4
)

// GenerateThumbnail creates a thumbnail for the given file ID.
// Thumbnails are stored in .tagloom/thumbnails/{2char_hash}/{hash}.jpg
func (a *App) GenerateThumbnail(fileID int64) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("no vault open")
	}
	if a.vaultPath == "" {
		return "", fmt.Errorf("no vault path set")
	}

	// Fetch file info from DB
	row := a.db.Conn().QueryRow(`
		SELECT id, vault_path, thumbnail_path
		FROM files WHERE id = ?
	`, fileID)

	var id int64
	var vaultPath string
	var existingThumb *string // nullable in DB
	if err := row.Scan(&id, &vaultPath, &existingThumb); err != nil {
		return "", fmt.Errorf("failed to find file %d: %w", fileID, err)
	}

	// Dereference nullable thumbnail path
	thumbStr := ""
	if existingThumb != nil {
		thumbStr = *existingThumb
	}

	// Check if file actually exists on disk
	if _, err := os.Stat(vaultPath); err != nil {
		return "", fmt.Errorf("file not found on disk: %w", err)
	}

	// Determine target thumbnail path
	thumbPath := GenerateThumbnailPath(a.vaultPath, vaultPath)

	// If thumbnail already exists, check if source is newer
	if thumbStr == thumbPath {
		thumbStat, err1 := os.Stat(thumbPath)
		srcStat, err2 := os.Stat(vaultPath)
		if err1 == nil && err2 == nil && !srcStat.ModTime().After(thumbStat.ModTime()) {
			return thumbPath, nil // Thumbnail is up to date
		}
	}

	// Determine file category
	ext := strings.ToLower(filepath.Ext(vaultPath))
	category := utils.GetFileCategory(vaultPath)

	var img image.Image
	var err error

	switch category {
	case "image", "animated":
		img, err = decodeImage(vaultPath, ext)
		if err != nil {
			return "", fmt.Errorf("failed to decode image: %w", err)
		}
	default:
		// Video thumbnails require FFmpeg — skip for now
		return "", fmt.Errorf("thumbnail generation not supported for %s files yet", ext)
	}

	// Get config for thumbnail size/quality
	size := defaultThumbnailSize
	quality := defaultThumbnailQuality
	if a.vaultCfg != nil {
		if a.vaultCfg.Settings.ThumbnailSize > 0 {
			size = a.vaultCfg.Settings.ThumbnailSize
		}
		if a.vaultCfg.Settings.ThumbnailQuality > 0 {
			quality = a.vaultCfg.Settings.ThumbnailQuality
		}
	}

	// Resize to thumbnail (fit within size×size, maintain aspect ratio)
	resized := imaging.Fit(img, size, size, imaging.Lanczos)

	// Ensure output directory exists
	thumbDir := filepath.Dir(thumbPath)
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	// Write as JPEG
	outFile, err := os.Create(thumbPath)
	if err != nil {
		return "", fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer outFile.Close()

	if err := jpeg.Encode(outFile, resized, &jpeg.Options{Quality: quality}); err != nil {
		outFile.Close()
		os.Remove(thumbPath) // Clean up partial file
		return "", fmt.Errorf("failed to encode JPEG: %w", err)
	}

	// Update DB with thumbnail path
	_, err = a.db.Conn().Exec("UPDATE files SET thumbnail_path = ? WHERE id = ?", thumbPath, fileID)
	if err != nil {
		return "", fmt.Errorf("failed to update thumbnail_path: %w", err)
	}

	return thumbPath, nil
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
	if a.db == nil {
		return fmt.Errorf("no vault open")
	}
	if a.vaultPath == "" {
		return fmt.Errorf("no vault path set")
	}

	// Collect all file IDs that need thumbnails
	rows, err := a.db.Conn().Query(`
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
		// Skip if file doesn't exist on disk
		if _, err := os.Stat(f.vaultPath); err != nil {
			continue
		}
		pending = append(pending, f)
	}

	total := len(pending)
	if total == 0 {
		runtime.EventsEmit(a.ctx, "thumb:complete", map[string]int{
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

	// Worker function — does image I/O only, returns results for batched DB update
	worker := func(workerID int, jobs <-chan pendingFile, wg *sync.WaitGroup) {
		defer wg.Done()

		size := defaultThumbnailSize
		quality := defaultThumbnailQuality
		if a.vaultCfg != nil {
			if a.vaultCfg.Settings.ThumbnailSize > 0 {
				size = a.vaultCfg.Settings.ThumbnailSize
			}
			if a.vaultCfg.Settings.ThumbnailQuality > 0 {
				quality = a.vaultCfg.Settings.ThumbnailQuality
			}
		}

		for f := range jobs {
			// Check for cancellation
			select {
			case <-ctx.Done():
				return
			default:
			}

			thumbPath := GenerateThumbnailPath(a.vaultPath, f.vaultPath)

			// Skip if already generated (another worker may have done it)
			if _, err := os.Stat(thumbPath); err == nil {
				resultsMu.Lock()
				results = append(results, thumbResult{fileID: f.id, thumbPath: thumbPath, ok: true})
				resultsMu.Unlock()

				processedMu.Lock()
				processed++
				cur := processed
				processedMu.Unlock()

				if cur%10 == 0 || cur == total {
					runtime.EventsEmit(a.ctx, "thumb:progress", map[string]int{
						"current": cur,
						"total":   total,
					})
				}
				continue
			}

			// Decode image
			ext := strings.ToLower(filepath.Ext(f.vaultPath))
			category := utils.GetFileCategory(f.vaultPath)

			var img image.Image
			var decodeErr error

			switch category {
			case "image", "animated":
				img, decodeErr = decodeImage(f.vaultPath, ext)
			default:
				// Unsupported
				resultsMu.Lock()
				results = append(results, thumbResult{fileID: f.id, ok: false})
				resultsMu.Unlock()

				processedMu.Lock()
				processed++
				cur := processed
				processedMu.Unlock()

				if cur%10 == 0 || cur == total {
					runtime.EventsEmit(a.ctx, "thumb:progress", map[string]int{
						"current": cur,
						"total":   total,
					})
				}
				continue
			}

			if decodeErr != nil {
				resultsMu.Lock()
				results = append(results, thumbResult{fileID: f.id, ok: false})
				resultsMu.Unlock()

				processedMu.Lock()
				processed++
				cur := processed
				processedMu.Unlock()

				if cur%10 == 0 || cur == total {
					runtime.EventsEmit(a.ctx, "thumb:progress", map[string]int{
						"current": cur,
						"total":   total,
					})
				}
				continue
			}

			// Resize
			resized := imaging.Fit(img, size, size, imaging.Lanczos)

			// Write thumbnail
			thumbDir := filepath.Dir(thumbPath)
			if err := os.MkdirAll(thumbDir, 0755); err != nil {
				resultsMu.Lock()
				results = append(results, thumbResult{fileID: f.id, ok: false})
				resultsMu.Unlock()

				processedMu.Lock()
				processed++
				cur := processed
				processedMu.Unlock()

				if cur%10 == 0 || cur == total {
					runtime.EventsEmit(a.ctx, "thumb:progress", map[string]int{
						"current": cur,
						"total":   total,
					})
				}
				continue
			}

			outFile, err := os.Create(thumbPath)
			if err != nil {
				resultsMu.Lock()
				results = append(results, thumbResult{fileID: f.id, ok: false})
				resultsMu.Unlock()

				processedMu.Lock()
				processed++
				cur := processed
				processedMu.Unlock()

				if cur%10 == 0 || cur == total {
					runtime.EventsEmit(a.ctx, "thumb:progress", map[string]int{
						"current": cur,
						"total":   total,
					})
				}
				continue
			}

			if err := jpeg.Encode(outFile, resized, &jpeg.Options{Quality: quality}); err != nil {
				outFile.Close()
				os.Remove(thumbPath)
				resultsMu.Lock()
				results = append(results, thumbResult{fileID: f.id, ok: false})
				resultsMu.Unlock()

				processedMu.Lock()
				processed++
				cur := processed
				processedMu.Unlock()

				if cur%10 == 0 || cur == total {
					runtime.EventsEmit(a.ctx, "thumb:progress", map[string]int{
						"current": cur,
						"total":   total,
					})
				}
				continue
			}
			outFile.Close()

			// Record success — DB update is batched after all workers finish
			resultsMu.Lock()
			results = append(results, thumbResult{fileID: f.id, thumbPath: thumbPath, ok: true})
			resultsMu.Unlock()

			processedMu.Lock()
			processed++
			cur := processed
			processedMu.Unlock()

			if cur%10 == 0 || cur == total {
				runtime.EventsEmit(a.ctx, "thumb:progress", map[string]int{
					"current": cur,
					"total":   total,
				})
			}
		}
	}

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < thumbnailWorkerCount; i++ {
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
	tx, err := a.db.Conn().Begin()
	if err == nil {
		stmt, prepErr := tx.Prepare("UPDATE files SET thumbnail_path = ? WHERE id = ?")
		if prepErr == nil {
			for _, r := range results {
				if r.ok {
					_, _ = stmt.Exec(r.thumbPath, r.fileID)
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
	runtime.EventsEmit(a.ctx, "thumb:complete", map[string]int{
		"generated": generated,
		"failed":    failed,
		"total":     total,
	})

	return nil
}

// incrementCounters updates progress counters and emits events (thread-safe).
func incrementCounters(mu *sync.Mutex, generated, current *int, total int, ctx context.Context) {
	mu.Lock()
	defer mu.Unlock()

	*generated++
	*current++

	// Emit progress every 10 files
	if *current%10 == 0 || *current == total {
		runtime.EventsEmit(ctx, "thumb:progress", map[string]int{
			"current":   *current,
			"total":     total,
			"generated": *generated,
		})
	}
}

// CancelThumbnailGeneration cancels an ongoing thumbnail generation pool.
func (a *App) CancelThumbnailGeneration() {
	// This will be wired to a cancel context in the future
	runtime.EventsEmit(a.ctx, "thumb:cancelled", true)
}

// decodeImage opens and decodes an image file, supporting multiple formats.
func decodeImage(path string, ext string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	switch ext {
	case ".webp":
		return webp.Decode(f)
	case ".tiff", ".tif":
		return tiff.Decode(f)
	default:
		// jpeg, png, gif, bmp — all handled by image.Decode
		img, _, err := image.Decode(f)
		return img, err
	}
}

// GenerateThumbnailPath returns the expected thumbnail path for a given file path.
func GenerateThumbnailPath(vaultPath, filePath string) string {
	hash := utils.HashPath(filePath)
	subdir := utils.ThumbnailSubdir(hash)
	thumbDir := filepath.Join(vaultPath, ".tagloom", "thumbnails", subdir)
	os.MkdirAll(thumbDir, 0755)
	return filepath.Join(thumbDir, hash+".jpg")
}

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
// in the database. It walks .tagloom/thumbnails/ and deletes any .jpg not referenced
// by files.thumbnail_path. Returns the number of orphan files removed.
func (a *App) CleanupOrphanThumbnails() (int, error) {
	if a.db == nil {
		return 0, fmt.Errorf("no vault open")
	}
	if a.vaultPath == "" {
		return 0, fmt.Errorf("no vault path set")
	}

	thumbRoot := filepath.Join(a.vaultPath, ".tagloom", "thumbnails")
	if _, err := os.Stat(thumbRoot); os.IsNotExist(err) {
		return 0, nil // No thumbnails directory yet
	}

	// Collect all valid thumbnail paths from the database
	validThumbs := make(map[string]struct{})
	rows, err := a.db.Conn().Query("SELECT thumbnail_path FROM files WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''")
	if err != nil {
		return 0, fmt.Errorf("failed to query thumbnail paths: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tp string
		if err := rows.Scan(&tp); err == nil && tp != "" {
			validThumbs[tp] = struct{}{}
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
		if strings.HasSuffix(strings.ToLower(path), ".jpg") {
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

// GetThumbnailPath returns the thumbnail path for a file ID.
// Used by the HTTP handler to serve thumbnails.
func (a *App) GetThumbnailPath(fileID int64) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("no vault open")
	}

	row := a.db.Conn().QueryRow("SELECT thumbnail_path FROM files WHERE id = ?", fileID)
	var thumbPath *string // nullable in DB
	if err := row.Scan(&thumbPath); err != nil {
		return "", fmt.Errorf("no thumbnail for file %d: %w", fileID, err)
	}
	if thumbPath == nil || *thumbPath == "" {
		return "", fmt.Errorf("no thumbnail path for file %d", fileID)
	}

	// Verify file exists
	if _, err := os.Stat(*thumbPath); err != nil {
		return "", fmt.Errorf("thumbnail file not found: %w", err)
	}

	return *thumbPath, nil
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
	if a.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	row := a.db.Conn().QueryRow("SELECT thumbnail_path FROM files WHERE id = ?", fileID)
	var thumbPath *string // nullable in DB
	if err := row.Scan(&thumbPath); err != nil {
		return nil, fmt.Errorf("no thumbnail for file %d: %w", fileID, err)
	}

	info := &ThumbnailInfo{
		FileID: fileID,
	}

	if thumbPath != nil && *thumbPath != "" {
		info.ThumbnailPath = *thumbPath
		stat, err := os.Stat(*thumbPath)
		if err == nil {
			info.Exists = true
			info.SizeBytes = stat.Size()
		}
	}

	return info, nil
}

// GetThumbnailData reads a thumbnail file and returns it as a base64 data URL.
// The frontend uses this to display thumbnails in <img> tags.
func (a *App) GetThumbnailData(fileID int64) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("no vault open")
	}

	// Fetch thumbnail_path from DB
	row := a.db.Conn().QueryRow("SELECT thumbnail_path FROM files WHERE id = ?", fileID)
	var thumbPath *string // nullable in DB
	if err := row.Scan(&thumbPath); err != nil {
		return "", fmt.Errorf("no thumbnail for file %d: %w", fileID, err)
	}
	if thumbPath == nil || *thumbPath == "" {
		return "", fmt.Errorf("no thumbnail path for file %d", fileID)
	}

	data, err := os.ReadFile(*thumbPath)
	if err != nil {
		return "", fmt.Errorf("failed to read thumbnail: %w", err)
	}

	// Encode as base64 data URL
	b64 := base64.StdEncoding.EncodeToString(data)
	return "data:image/jpeg;base64," + b64, nil
}

// ServeThumbnail reads a thumbnail file and returns its bytes.
// Used by the frontend to display thumbnails via a Go handler.
func ServeThumbnail(thumbPath string) ([]byte, error) {
	if thumbPath == "" {
		return nil, fmt.Errorf("no thumbnail path")
	}
	data, err := os.ReadFile(thumbPath)
	if err != nil {
		return nil, err
	}
	return data, nil
}


