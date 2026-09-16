package main

import (
	"embed"
	"log/slog"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsJSON []byte

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "marktyp",
		Width:     1440,
		Height:    900,
		MinWidth:  1100,
		MinHeight: 720,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 244, G: 240, B: 229, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		Menu:             buildNativeMenu(app),
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		slog.Error("Application failed", "error", err)
	}
}
