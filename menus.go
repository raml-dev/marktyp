package main

import (
	"runtime"

	"github.com/wailsapp/wails/v2/pkg/menu"
)

func buildNativeMenu(app *App) *menu.Menu {
	mainMenu := menu.NewMenu()
	if runtime.GOOS == "darwin" {
		// Supplying a custom menu disables Wails' default macOS menu. Restore the
		// native application role so standard application shortcuts keep working.
		mainMenu.Append(menu.AppMenu())
	}

	fileMenu := mainMenu.AddSubmenu("File")
	fileMenu.AddText("New", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("file:new")
	})
	fileMenu.AddText("Open", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("file:open")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Save", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("file:save")
	})
	fileMenu.AddText("Save As", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("file:save-as")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Delete Note", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("file:delete")
	})

	if runtime.GOOS == "darwin" {
		// The native Edit role owns Cmd+X/C/V/A and the corresponding responder
		// chain. Without it, WebKit never receives Cmd+V in a packaged app.
		mainMenu.Append(menu.EditMenu())
	}

	exportMenu := mainMenu.AddSubmenu("Export")
	exportMenu.AddText("Export HTML", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("export:html")
	})
	exportMenu.AddText("Export PDF", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("export:pdf")
	})

	viewMenu := mainMenu.AddSubmenu("View")
	viewMenu.AddText("Document Mode", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("view:document")
	})
	viewMenu.AddText("Source Mode", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("view:source")
	})
	viewMenu.AddText("Dual Mode", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("view:dual")
	})
	viewMenu.AddSeparator()
	viewMenu.AddText("Toggle Autosave", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("view:toggle-autosave")
	})

	settingsMenu := mainMenu.AddSubmenu("Settings")
	settingsMenu.AddText("Theme: Marktyp", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("settings:theme-marktyp")
	})
	settingsMenu.AddText("Theme: Light", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("settings:theme-light")
	})
	settingsMenu.AddText("Theme: Dark", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("settings:theme-dark")
	})

	aboutMenu := mainMenu.AddSubmenu("About")
	aboutMenu.AddText("Marktyp", nil, func(_ *menu.CallbackData) {
		app.emitMenuAction("about:open")
	})

	if runtime.GOOS == "darwin" {
		mainMenu.Append(menu.WindowMenu())
	}

	return mainMenu
}
