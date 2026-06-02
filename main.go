package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"TagLoom/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	appInstance := app.NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:       "TagLoom",
		Width:       1280,
		Height:      800,
		MinWidth:    900,
		MinHeight:   600,
		AssetServer: &assetserver.Options{
			Assets: assets,
			Middleware: assetserver.Middleware(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Check if this is a thumbnail request
					if strings.HasPrefix(r.URL.Path, "/api/thumbnail/") {
						fileIDStr := strings.TrimPrefix(r.URL.Path, "/api/thumbnail/")
						if fileIDStr == "" {
							http.Error(w, "Missing file ID", http.StatusBadRequest)
							return
						}

						var fileID int64
						fmt.Sscanf(fileIDStr, "%d", &fileID)

						thumbPath, err := appInstance.GetThumbnailPath(fileID)
						if err != nil {
							http.Error(w, err.Error(), http.StatusNotFound)
							return
						}

						data, err := os.ReadFile(thumbPath)
						if err != nil {
							http.Error(w, "Thumbnail not found", http.StatusNotFound)
							return
						}

						w.Header().Set("Content-Type", "image/jpeg")
						w.Header().Set("Cache-Control", "max-age=3600")
						w.Write(data)
						return
					}

					// Check if this is a thumbnail generation request
					if strings.HasPrefix(r.URL.Path, "/api/generate-thumbnail/") {
						fileIDStr := strings.TrimPrefix(r.URL.Path, "/api/generate-thumbnail/")
						if fileIDStr == "" {
							http.Error(w, "Missing file ID", http.StatusBadRequest)
							return
						}

						var fileID int64
						fmt.Sscanf(fileIDStr, "%d", &fileID)

						thumbPath, err := appInstance.GenerateThumbnail(fileID)
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}

						data, err := os.ReadFile(thumbPath)
						if err != nil {
							http.Error(w, "Thumbnail not found", http.StatusNotFound)
							return
						}

						w.Header().Set("Content-Type", "image/jpeg")
						w.Header().Set("Cache-Control", "max-age=3600")
						w.Write(data)
						return
					}

					// Check if this is an original file request (with Range support for video seeking)
					if strings.HasPrefix(r.URL.Path, "/api/original/") {
						fileIDStr := strings.TrimPrefix(r.URL.Path, "/api/original/")
						if fileIDStr == "" {
							http.Error(w, "Missing file ID", http.StatusBadRequest)
							return
						}

						var fileID int64
						fmt.Sscanf(fileIDStr, "%d", &fileID)

						filePath, err := appInstance.GetOriginalFilePath(fileID)
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
						fileSize := stat.Size()

						ext := strings.ToLower(filepath.Ext(filePath))
						contentType := "application/octet-stream"
						switch ext {
						case ".jpg", ".jpeg":
							contentType = "image/jpeg"
						case ".png":
							contentType = "image/png"
						case ".gif":
							contentType = "image/gif"
						case ".webp":
							contentType = "image/webp"
						case ".bmp":
							contentType = "image/bmp"
						case ".tiff", ".tif":
							contentType = "image/tiff"
						case ".svg":
							contentType = "image/svg+xml"
						case ".mp4", ".m4v", ".f4v":
							contentType = "video/mp4"
						case ".mov":
							contentType = "video/quicktime"
						case ".avi":
							contentType = "video/x-msvideo"
						case ".webm":
							contentType = "video/webm"
						case ".mkv":
							contentType = "video/x-matroska"
						case ".wmv":
							contentType = "video/x-ms-wmv"
						case ".flv":
							contentType = "video/x-flv"
						case ".3gp":
							contentType = "video/3gpp"
						case ".3g2":
							contentType = "video/3gpp2"
						case ".vob", ".mpg", ".mpeg", ".m2v":
							contentType = "video/mpeg"
						case ".ogv":
							contentType = "video/ogg"
						case ".ts", ".mts", ".m2ts":
							contentType = "video/mp2t"
						case ".asf":
							contentType = "video/x-ms-asf"
						case ".rm":
							contentType = "application/vnd.rn-realmedia"
						case ".amv":
							contentType = "video/avi"
						case ".dv":
							contentType = "video/dv"
						case ".mxf":
							contentType = "application/mxf"
						}

						w.Header().Set("Content-Type", contentType)
						w.Header().Set("Accept-Ranges", "bytes")
						w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))

						// Handle Range requests (video seeking)
						rangeHeader := r.Header.Get("Range")
						if rangeHeader != "" {
							// Parse range: "bytes=START-END" or "bytes=START-"
							if strings.HasPrefix(rangeHeader, "bytes=") {
								bytesPart := strings.TrimPrefix(rangeHeader, "bytes=")
								parts := strings.SplitN(bytesPart, "-", 2)
								if len(parts) == 2 {
									var start, end int64
									if parts[0] != "" {
										fmt.Sscanf(parts[0], "%d", &start)
									}
									if parts[1] != "" {
										fmt.Sscanf(parts[1], "%d", &end)
									} else {
										end = fileSize - 1
									}
									if start > end || start >= fileSize {
										http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
										return
									}

									// Seek to start position
									f.Seek(start, 0)
									length := end - start + 1

									w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
									w.Header().Set("Content-Length", fmt.Sprintf("%d", length))
									w.WriteHeader(http.StatusPartialContent)
									io.CopyN(w, f, length)
									return
								}
							}
						}

						// Full file request
						w.WriteHeader(http.StatusOK)
						io.Copy(w, f)
						return
					}

					// Check if this is a thumbnail metadata request
					if strings.HasPrefix(r.URL.Path, "/api/thumbnail-info/") {
						fileIDStr := strings.TrimPrefix(r.URL.Path, "/api/thumbnail-info/")
						if fileIDStr == "" {
							http.Error(w, "Missing file ID", http.StatusBadRequest)
							return
						}

						var fileID int64
						fmt.Sscanf(fileIDStr, "%d", &fileID)

						info, err := appInstance.GetThumbnailInfo(fileID)
						if err != nil {
							http.Error(w, err.Error(), http.StatusNotFound)
							return
						}

						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(info)
						return
					}

					// Pass to next handler
					next.ServeHTTP(w, r)
				})
			}),
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 18, A: 1},
		OnStartup:        appInstance.Startup,
		Bind: []interface{}{
			appInstance,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
