package main

import (
	"embed"

	internalapp "marktyp/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := internalapp.New()

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
		OnStartup:        app.Startup,
		Menu:             buildNativeMenu(app),
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
