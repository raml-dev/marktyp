package app

const (
	AppName    = "marktyp"
	AppVersion = "0.1.0"
)

type AppInfo struct {
	Name    string   `json:"name"`
	Tagline string   `json:"tagline"`
	Version string   `json:"version"`
	Modes   []string `json:"modes"`
}

type AppConfig struct {
	Version        int    `json:"version"`
	PreferredMode  string `json:"preferredMode"`
	LastOpenedPath string `json:"lastOpenedPath"`
	Autosave       bool   `json:"autosave"`
	Theme          string `json:"theme"`
}

type NoteSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Path         string `json:"path"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	UpdatedLabel string `json:"updatedLabel"`
}

type DocumentState struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Path          string `json:"path"`
	Markdown      string `json:"markdown"`
	DraftMarkdown string `json:"draftMarkdown,omitempty"`
}

type WorkspaceData struct {
	AppInfo   AppInfo       `json:"appInfo"`
	Config    AppConfig     `json:"config"`
	Notes     []NoteSummary `json:"notes"`
	ActiveDoc DocumentState `json:"activeDoc"`
}

type SaveDocumentRequest struct {
	Path     string `json:"path"`
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
}

type UpdatePreferencesRequest struct {
	PreferredMode string `json:"preferredMode"`
	Autosave      bool   `json:"autosave"`
	Theme         string `json:"theme"`
}

type ExportHTMLRequest struct {
	Title string `json:"title"`
	HTML  string `json:"html"`
}

type ExportPDFRequest struct {
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
	HTML     string `json:"html"`
}

func DefaultAppInfo() AppInfo {
	return AppInfo{
		Name:    AppName,
		Tagline: "Editor documentale visuale con Markdown sotto il cofano",
		Version: AppVersion,
		Modes:   []string{"Document", "Source", "Dual"},
	}
}
