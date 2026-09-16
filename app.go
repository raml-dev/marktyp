package main

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"

	"marktyp/internal/appinfo"
	"marktyp/internal/document"
	"marktyp/internal/exporter"
	"marktyp/internal/tools"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// RPC aliases preserve Marktyp's JSON contracts while models live in the domain.
type AppInfo = document.AppInfo
type WorkspaceData = document.WorkspaceData
type DocumentState = document.DocumentState
type SaveDocumentRequest = document.SaveDocumentRequest
type RenameNoteRequest = document.RenameNoteRequest
type UpdatePreferencesRequest = document.UpdatePreferencesRequest
type AppConfig = document.AppConfig
type ExportHTMLRequest = document.ExportHTMLRequest
type ExportPDFRequest = document.ExportPDFRequest

const AppName = document.AppName

// App is the sole Wails RPC boundary.
type App struct {
	ctx         context.Context
	store       *document.Store
	exporter    *exporter.Exporter
	info        AppInfo
	updates     *appinfo.DiscoveryClient
	quitAllowed atomic.Bool
}

// NewApp constructs the Wails RPC facade and its domain services.
func NewApp() *App {
	info := appinfo.FromWailsConfig(wailsJSON)
	return &App{
		store:    document.NewStore(),
		exporter: exporter.NewExporter(),
		info:     info,
		updates:  appinfo.NewDiscoveryClient(info.GHLink),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) beforeClose(ctx context.Context) bool {
	if a.quitAllowed.Load() {
		return false
	}
	wruntime.EventsEmit(ctx, AppName+":request-close")
	return true
}

// ForceQuit closes the application after the frontend has preserved pending edits.
func (a *App) ForceQuit() {
	if a.ctx == nil {
		return
	}
	a.quitAllowed.Store(true)
	wruntime.Quit(a.ctx)
}

func (a *App) emitMenuAction(action string) {
	if a.ctx == nil {
		return
	}

	wruntime.EventsEmit(a.ctx, AppName+":menu-action", action)
}

// GetAppInfo returns immutable application metadata.
func (a *App) GetAppInfo() AppInfo {
	return a.info
}

// GetUpdatesFromRepo checks GitHub for a newer eligible Marktyp release.
func (a *App) GetUpdatesFromRepo() (*appinfo.GitHubResponse, error) {
	if a.updates == nil {
		return nil, errors.New("update discovery service not initialized")
	}
	if a.store == nil {
		return nil, errors.New("store not initialized")
	}

	config, err := a.store.GetConfig()
	if err != nil {
		return nil, err
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	response, err := a.updates.GetUpdatesFromRepo(ctx, a.info.ProductVersion, config.IncludePrereleaseUpdates)
	if err != nil {
		if a.ctx != nil {
			wruntime.EventsEmit(a.ctx, "updates:error", err.Error())
		}
		return nil, err
	}
	if response != nil && response.Release != nil && a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "updates:available", response)
	}

	return response, nil
}

// GetWorkspace loads configuration, recent notes, and the active document.
func (a *App) GetWorkspace() (WorkspaceData, error) {
	if a.store == nil {
		return WorkspaceData{}, errors.New("store not initialized")
	}
	return a.store.GetWorkspace(a.info)
}

// NewDocument creates and activates a distinct managed note.
func (a *App) NewDocument() (WorkspaceData, error) {
	if a.store == nil {
		return WorkspaceData{}, errors.New("store not initialized")
	}
	return a.store.CreateDocument(a.info)
}

// OpenDocument prompts for a Markdown file and makes it active.
func (a *App) OpenDocument() (WorkspaceData, error) {
	if a.store == nil {
		return WorkspaceData{}, errors.New("store not initialized")
	}
	if a.ctx == nil {
		return WorkspaceData{}, errors.New("ctx not initialized")
	}
	filePath, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Open Markdown file",
		Filters: []wruntime.FileFilter{{
			DisplayName: "Markdown",
			Pattern:     "*.md;*.markdown",
		}},
	})
	if err != nil {
		return WorkspaceData{}, err
	}

	if strings.TrimSpace(filePath) == "" {
		return a.store.GetWorkspace(a.info)
	}

	return a.store.OpenDocumentAtPath(filePath, a.info)
}

// SelectImage prompts for a local raster image.
func (a *App) SelectImage() (string, error) {
	if a.ctx == nil {
		return "", errors.New("ctx not initialized")
	}
	path, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Insert Image",
		Filters: []wruntime.FileFilter{{
			DisplayName: "Images",
			Pattern:     "*.png;*.jpg;*.jpeg;*.gif;*.webp;*.bmp",
		}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	return path, nil
}

// GetImagePreview loads a safe local raster image for display in the webview.
func (a *App) GetImagePreview(source, documentPath string) (string, error) {
	return tools.ImagePreview(source, documentPath)
}

// OpenDocumentAtPath makes a recent Markdown file active.
func (a *App) OpenDocumentAtPath(path string) (WorkspaceData, error) {
	if a.store == nil {
		return WorkspaceData{}, errors.New("store not initialized")
	}
	return a.store.OpenDocumentAtPath(path, a.info)
}

// SaveDocument atomically saves a document, prompting for a path when needed.
func (a *App) SaveDocument(request SaveDocumentRequest) (WorkspaceData, error) {
	if a.store == nil {
		return WorkspaceData{}, errors.New("store not initialized")
	}
	if strings.TrimSpace(request.Path) == "" {
		return a.SaveDocumentAs(request)
	}

	return a.store.SaveDocument(request, a.info)
}

// RenameNote updates the Markdown title of a saved note.
func (a *App) RenameNote(request RenameNoteRequest) (WorkspaceData, error) {
	if a.store == nil {
		return WorkspaceData{}, errors.New("store not initialized")
	}
	return a.store.RenameNote(request, a.info)
}

// SaveDraft persists recoverable unsaved content for a saved note.
func (a *App) SaveDraft(path string, markdown string) error {
	if a.store == nil {
		return errors.New("store not initialized")
	}
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return a.store.SaveDraft(path, markdown)
}

// SaveDocumentAs prompts for a destination and atomically saves a document.
func (a *App) SaveDocumentAs(request SaveDocumentRequest) (WorkspaceData, error) {
	if a.store == nil {
		return WorkspaceData{}, errors.New("store not initialized")
	}
	if a.ctx == nil {
		return WorkspaceData{}, errors.New("ctx not initialized")
	}
	defaultFilename := tools.Filename(request.Title, "untitled", ".md")

	filePath, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Save Markdown file",
		DefaultFilename: defaultFilename,
		Filters: []wruntime.FileFilter{{
			DisplayName: "Markdown",
			Pattern:     "*.md",
		}},
	})
	if err != nil {
		return WorkspaceData{}, err
	}

	if strings.TrimSpace(filePath) == "" {
		return WorkspaceData{}, errors.New("save cancelled")
	}

	request.Path = filePath
	return a.store.SaveDocument(request, a.info)
}

// DeleteNote removes a note from disk and the recent-note database.
func (a *App) DeleteNote(path string) (WorkspaceData, error) {
	if a.store == nil {
		return WorkspaceData{}, errors.New("store not initialized")
	}
	return a.store.DeleteNote(path, a.info)
}

// GetDocument loads a saved note and any recoverable draft.
func (a *App) GetDocument(path string) (DocumentState, error) {
	if a.store == nil {
		return DocumentState{}, errors.New("store not initialized")
	}
	return a.store.GetDocument(path)
}

// UpdatePreferences validates and persists user preferences.
func (a *App) UpdatePreferences(request UpdatePreferencesRequest) (AppConfig, error) {
	if a.store == nil {
		return AppConfig{}, errors.New("store not initialized")
	}
	return a.store.UpdatePreferences(request)
}

// ExportHTML prompts for a destination and writes a standalone HTML document.
func (a *App) ExportHTML(request ExportHTMLRequest) error {
	if a.exporter == nil {
		return errors.New("exporter not initialized")
	}
	if a.ctx == nil {
		return errors.New("ctx not initialized")
	}
	if strings.TrimSpace(request.HTML) == "" {
		return errors.New("missing html content")
	}

	defaultFilename := tools.Filename(request.Title, "document", ".html")

	filePath, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Export HTML",
		DefaultFilename: defaultFilename,
		Filters: []wruntime.FileFilter{{
			DisplayName: "HTML",
			Pattern:     "*.html",
		}},
	})
	if err != nil {
		return err
	}

	if strings.TrimSpace(filePath) == "" {
		return errors.New("export cancelled")
	}

	return a.exporter.ExportHTMLToPath(request, filePath)
}

// ExportPDF prompts for a destination and writes a rendered PDF document.
func (a *App) ExportPDF(request ExportPDFRequest) error {
	if a.exporter == nil {
		return errors.New("exporter not initialized")
	}
	if a.ctx == nil {
		return errors.New("ctx not initialized")
	}
	if strings.TrimSpace(request.Markdown) == "" && strings.TrimSpace(request.HTML) == "" {
		return errors.New("missing document content")
	}

	defaultFilename := tools.Filename(request.Title, "document", ".pdf")

	filePath, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Export PDF",
		DefaultFilename: defaultFilename,
		Filters: []wruntime.FileFilter{{
			DisplayName: "PDF",
			Pattern:     "*.pdf",
		}},
	})
	if err != nil {
		return err
	}

	if strings.TrimSpace(filePath) == "" {
		return errors.New("export cancelled")
	}

	return a.exporter.ExportPDFToPath(request, filePath)
}

// RevealNoteInFS reveals the selected note using the platform file manager.
func (a *App) RevealNoteInFS(path string) error {
	if a.ctx == nil {
		return errors.New("ctx not initialized")
	}
	return tools.RevealNoteInFS(a.ctx, path)
}
