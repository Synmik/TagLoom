package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"

	"TagLoom/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsJSON []byte

// productVersionFromWailsJSON extracts info.productVersion from a
// wails.json document. wails.json is the single source of truth for the
// application version — no Go constant is kept in sync by hand.
func productVersionFromWailsJSON(data []byte) (string, error) {
	var cfg struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse wails.json: %w", err)
	}
	if cfg.Info.ProductVersion == "" {
		return "", fmt.Errorf("wails.json has no info.productVersion")
	}
	return cfg.Info.ProductVersion, nil
}

func main() {
	version, err := productVersionFromWailsJSON(wailsJSON)
	if err != nil {
		println("Error:", err.Error())
		return
	}

	// Create an instance of the app structure
	appInstance := app.NewApp(version)

	// Create application with options
	err = wails.Run(&options.App{
		Title:     "TagLoom",
		Width:     1600,
		Height:    900,
		MinWidth:  900,
		MinHeight: 600,
		Frameless: true,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
			// Serves the /api/ file routes (thumbnails, originals with
			// Range support, thumbnail info); see app/http.go.
			Middleware: assetserver.Middleware(func(next http.Handler) http.Handler {
				return app.AssetMiddleware(appInstance, next)
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
