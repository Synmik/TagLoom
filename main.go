package main

import (
	"embed"
	"net/http"

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
