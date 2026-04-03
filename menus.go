package main

import (
	internalapp "marktyp/internal/app"

	"github.com/wailsapp/wails/v2/pkg/menu"
)

func buildNativeMenu(app *internalapp.App) *menu.Menu {
	mainMenu := menu.NewMenu()

	fileMenu := mainMenu.AddSubmenu("File")
	fileMenu.AddText("New", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("file:new")
	})
	fileMenu.AddText("Open", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("file:open")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Save", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("file:save")
	})
	fileMenu.AddText("Save As", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("file:save-as")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Delete Note", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("file:delete")
	})

	exportMenu := mainMenu.AddSubmenu("Export")
	exportMenu.AddText("Export HTML", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("export:html")
	})
	exportMenu.AddText("Export PDF", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("export:pdf")
	})

	viewMenu := mainMenu.AddSubmenu("View")
	viewMenu.AddText("Document Mode", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("view:document")
	})
	viewMenu.AddText("Source Mode", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("view:source")
	})
	viewMenu.AddText("Dual Mode", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("view:dual")
	})
	viewMenu.AddSeparator()
	viewMenu.AddText("Toggle Autosave", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("view:toggle-autosave")
	})

	settingsMenu := mainMenu.AddSubmenu("Settings")
	settingsMenu.AddText("Theme: Marktyp", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("settings:theme-marktyp")
	})
	settingsMenu.AddText("Theme: Light", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("settings:theme-light")
	})
	settingsMenu.AddText("Theme: Dark", nil, func(_ *menu.CallbackData) {
		app.EmitMenuAction("settings:theme-dark")
	})

	return mainMenu
}
