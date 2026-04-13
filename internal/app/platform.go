package app

import (
	"context"
	"errors"
	"net/url"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	store    *Store
	exporter *Exporter
	info     AppInfo
}

func New() *App {
	return &App{
		store:    NewStore(),
		exporter: NewExporter(),
		info:     DefaultAppInfo(),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) EmitMenuAction(action string) {
	if a.ctx == nil {
		return
	}

	wruntime.EventsEmit(a.ctx, AppName+":menu-action", action)
}

func (a *App) GetAppInfo() AppInfo {
	return a.info
}

func (a *App) GetWorkspace() (WorkspaceData, error) {
	return a.store.GetWorkspace(a.info)
}

func (a *App) NewDocument() DocumentState {
	return NewDocument()
}

func (a *App) OpenDocument() (WorkspaceData, error) {
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

func (a *App) OpenDocumentAtPath(path string) (WorkspaceData, error) {
	return a.store.OpenDocumentAtPath(path, a.info)
}

func (a *App) SaveDocument(request SaveDocumentRequest) (WorkspaceData, error) {
	if strings.TrimSpace(request.Path) == "" {
		return a.SaveDocumentAs(request)
	}

	return a.store.SaveDocument(request, a.info)
}

func (a *App) SaveDraft(path string, markdown string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return a.store.SaveDraft(path, markdown)
}

func (a *App) SaveDocumentAs(request SaveDocumentRequest) (WorkspaceData, error) {
	defaultFilename := request.Title
	if defaultFilename == "" {
		defaultFilename = "untitled"
	}

	filePath, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Save Markdown file",
		DefaultFilename: defaultFilename + ".md",
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

func (a *App) DeleteNote(path string) (WorkspaceData, error) {
	return a.store.DeleteNote(path, a.info)
}

func (a *App) GetDocument(path string) (DocumentState, error) {
	return a.store.GetDocument(path)
}

func (a *App) UpdatePreferences(request UpdatePreferencesRequest) (AppConfig, error) {
	return a.store.UpdatePreferences(request)
}

func (a *App) ExportHTML(request ExportHTMLRequest) error {
	if strings.TrimSpace(request.HTML) == "" {
		return errors.New("missing html content")
	}

	defaultFilename := strings.TrimSpace(request.Title)
	if defaultFilename == "" {
		defaultFilename = "document"
	}

	filePath, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Export HTML",
		DefaultFilename: defaultFilename + ".html",
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

func (a *App) ExportPDF(request ExportPDFRequest) error {
	if strings.TrimSpace(request.Markdown) == "" && strings.TrimSpace(request.HTML) == "" {
		return errors.New("missing document content")
	}

	defaultFilename := strings.TrimSpace(request.Title)
	if defaultFilename == "" {
		defaultFilename = "document"
	}

	filePath, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Export PDF",
		DefaultFilename: defaultFilename + ".pdf",
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

func (a *App) RevealNoteInFS(path string) error {
	target := strings.TrimSpace(path)
	if target == "" {
		return errors.New("missing path")
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}

	dir := filepath.Dir(absTarget)

	switch goruntime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", absTarget).Run()
	case "windows":
		return exec.Command("explorer", "/select,", filepath.Clean(absTarget)).Run()
	default:
		if err := runFirstAvailableCommand(
			[]string{"xdg-open", dir},
			[]string{"/usr/bin/xdg-open", dir},
			[]string{"gio", "open", dir},
			[]string{"nautilus", "--select", absTarget},
			[]string{"dolphin", "--select", absTarget},
		); err == nil {
			return nil
		}

		fileURL := (&url.URL{Scheme: "file", Path: dir}).String()
		wruntime.BrowserOpenURL(a.ctx, fileURL)
		return nil
	}
}

func runFirstAvailableCommand(candidates ...[]string) error {
	var lastErr error
	for _, candidate := range candidates {
		if len(candidate) == 0 {
			continue
		}

		binary := candidate[0]
		if !strings.Contains(binary, "/") {
			if _, err := exec.LookPath(binary); err != nil {
				lastErr = err
				continue
			}
		}

		cmd := exec.Command(binary, candidate[1:]...)
		if err := cmd.Start(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return errors.New("no reveal command available")
}
