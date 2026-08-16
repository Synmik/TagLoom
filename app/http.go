package app

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"TagLoom/utils"
)

// AssetMiddleware returns an HTTP handler that serves the /api/ file routes
// (thumbnails, originals, thumbnail info) and passes every other request
// through to the next handler (the Wails asset server).
//
// Registered in main.go via assetserver.Middleware.
func (a *App) AssetMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/thumbnail/"):
			a.serveThumbnail(w, r.URL.Path[len("/api/thumbnail/"):])
		case strings.HasPrefix(r.URL.Path, "/api/generate-thumbnail/"):
			a.serveGeneratedThumbnail(w, r.URL.Path[len("/api/generate-thumbnail/"):])
		case strings.HasPrefix(r.URL.Path, "/api/original/"):
			a.serveOriginal(w, r, r.URL.Path[len("/api/original/"):])
		case strings.HasPrefix(r.URL.Path, "/api/thumbnail-info/"):
			a.serveThumbnailInfo(w, r.URL.Path[len("/api/thumbnail-info/"):])
		default:
			next.ServeHTTP(w, r)
		}
	})
}

// parseFileID parses a path segment as a positive file ID.
// On failure it writes a 400 response and returns ok=false.
func parseFileID(w http.ResponseWriter, s string) (int64, bool) {
	if s == "" {
		http.Error(w, "Missing file ID", http.StatusBadRequest)
		return 0, false
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// serveThumbnail streams an existing thumbnail (image/webp, 1h cache).
func (a *App) serveThumbnail(w http.ResponseWriter, idStr string) {
	id, ok := parseFileID(w, idStr)
	if !ok {
		return
	}
	thumbPath, err := a.GetThumbnailPath(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data, err := os.ReadFile(thumbPath)
	if err != nil {
		http.Error(w, "Thumbnail not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", "max-age=3600")
	w.Write(data)
}

// serveGeneratedThumbnail generates the thumbnail on demand if needed,
// then streams it (image/webp, 1h cache).
func (a *App) serveGeneratedThumbnail(w http.ResponseWriter, idStr string) {
	id, ok := parseFileID(w, idStr)
	if !ok {
		return
	}
	thumbPath, err := a.GenerateThumbnail(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := os.ReadFile(thumbPath)
	if err != nil {
		http.Error(w, "Thumbnail not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", "max-age=3600")
	w.Write(data)
}

// serveOriginal streams the original file with Range support (video
// seeking). http.ServeContent handles Range/Content-Range, ETag and
// If-Modified-Since; the content type comes from the extension map so
// video players get the exact type they need.
func (a *App) serveOriginal(w http.ResponseWriter, r *http.Request, idStr string) {
	id, ok := parseFileID(w, idStr)
	if !ok {
		return
	}
	filePath, err := a.GetOriginalFilePath(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	f, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "Failed to stat file", http.StatusInternalServerError)
		return
	}
	// Set before ServeContent so it won't try content sniffing.
	w.Header().Set("Content-Type", utils.MIMEType(filePath))
	http.ServeContent(w, r, filepath.Base(filePath), stat.ModTime(), f)
}

// serveThumbnailInfo returns thumbnail metadata as JSON.
func (a *App) serveThumbnailInfo(w http.ResponseWriter, idStr string) {
	id, ok := parseFileID(w, idStr)
	if !ok {
		return
	}
	info, err := a.GetThumbnailInfo(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
