package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewDesktopApp()
	if err := wails.Run(&options.App{
		Title:             "Kursomat",
		Width:             1180,
		Height:            700,
		MinWidth:          1100,
		MinHeight:         680,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	}); err != nil {
		log.Printf("nie udało się uruchomić Kursomatu: %v", err)
	}
}
