package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
