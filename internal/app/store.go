package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Store struct{}

type notesDB struct {
	Version        int          `json:"version"`
	LastOpenedPath string       `json:"lastOpenedPath"`
	Notes          []storedNote `json:"notes"`
}

type storedNote struct {
	Path      string `json:"path"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) GetWorkspace(appInfo AppInfo) (WorkspaceData, error) {
	if err := s.ensureStorage(); err != nil {
		return WorkspaceData{}, err
	}

	config, err := s.readConfig()
	if err != nil {
		return WorkspaceData{}, err
	}

	db, err := s.readNotesDB()
	if err != nil {
		return WorkspaceData{}, err
	}

	activePath := config.LastOpenedPath
	if activePath == "" {
		activePath = db.LastOpenedPath
	}
	if activePath == "" && len(db.Notes) > 0 {
		activePath = db.Notes[0].Path
	}

	var document DocumentState
	if activePath == "" {
		document = NewDocument()
		config.LastOpenedPath = ""
	} else {
		document, err = s.GetDocument(activePath)
		if err != nil {
			config.LastOpenedPath = ""
			if writeErr := s.writeConfig(config); writeErr != nil {
				return WorkspaceData{}, writeErr
			}
			return s.GetWorkspace(appInfo)
		}

		if err := s.RegisterOpenedNote(activePath); err != nil {
			return WorkspaceData{}, err
		}

		config.LastOpenedPath = activePath
	}

	if err := s.writeConfig(config); err != nil {
		return WorkspaceData{}, err
	}

	notes, err := s.listNotes()
	if err != nil {
		return WorkspaceData{}, err
	}

	return WorkspaceData{
		AppInfo:   appInfo,
		Config:    config,
		Notes:     notes,
		ActiveDoc: document,
	}, nil
}

func NewDocument() DocumentState {
	return DocumentState{
		ID:       "untitled",
		Title:    "Untitled",
		Path:     "",
		Markdown: "# Untitled\n\n",
	}
}

func (s *Store) OpenDocumentAtPath(path string, appInfo AppInfo) (WorkspaceData, error) {
	if err := s.ensureStorage(); err != nil {
		return WorkspaceData{}, err
	}

	if strings.TrimSpace(path) == "" {
		return s.GetWorkspace(appInfo)
	}

	if err := s.RegisterOpenedNote(path); err != nil {
		return WorkspaceData{}, err
	}

	config, err := s.readConfig()
	if err != nil {
		return WorkspaceData{}, err
	}

	config.LastOpenedPath = path
	if err := s.writeConfig(config); err != nil {
		return WorkspaceData{}, err
	}

	return s.GetWorkspace(appInfo)
}

func (s *Store) SaveDocument(request SaveDocumentRequest, appInfo AppInfo) (WorkspaceData, error) {
	if err := s.ensureStorage(); err != nil {
		return WorkspaceData{}, err
	}

	path := strings.TrimSpace(request.Path)
	if path == "" {
		return WorkspaceData{}, errors.New("missing document path")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return WorkspaceData{}, err
	}

	content := strings.TrimRight(request.Markdown, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return WorkspaceData{}, err
	}

	if err := s.upsertNote(path, true); err != nil {
		return WorkspaceData{}, err
	}

	config, err := s.readConfig()
	if err != nil {
		return WorkspaceData{}, err
	}

	config.LastOpenedPath = path
	if err := s.writeConfig(config); err != nil {
		return WorkspaceData{}, err
	}

	return s.GetWorkspace(appInfo)
}

func (s *Store) DeleteNote(path string, appInfo AppInfo) (WorkspaceData, error) {
	if err := s.ensureStorage(); err != nil {
		return WorkspaceData{}, err
	}

	target := strings.TrimSpace(path)
	if target == "" {
		return s.GetWorkspace(appInfo)
	}

	db, err := s.readNotesDB()
	if err != nil {
		return WorkspaceData{}, err
	}

	filteredNotes := make([]storedNote, 0, len(db.Notes))
	removed := false
	for _, note := range db.Notes {
		if note.Path == target {
			removed = true
			continue
		}
		filteredNotes = append(filteredNotes, note)
	}

	if !removed {
		return s.GetWorkspace(appInfo)
	}

	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return WorkspaceData{}, err
	}

	db.Notes = filteredNotes
	if db.LastOpenedPath == target {
		db.LastOpenedPath = ""
		if len(db.Notes) > 0 {
			db.LastOpenedPath = db.Notes[0].Path
		}
	}

	if err := s.writeNotesDB(db); err != nil {
		return WorkspaceData{}, err
	}

	config, err := s.readConfig()
	if err != nil {
		return WorkspaceData{}, err
	}

	config.LastOpenedPath = db.LastOpenedPath
	if err := s.writeConfig(config); err != nil {
		return WorkspaceData{}, err
	}

	return s.GetWorkspace(appInfo)
}

func (s *Store) GetDocument(path string) (DocumentState, error) {
	if err := s.ensureStorage(); err != nil {
		return DocumentState{}, err
	}

	markdownBytes, err := os.ReadFile(path)
	if err != nil {
		return DocumentState{}, err
	}

	markdown := string(markdownBytes)
	return DocumentState{
		ID:       noteID(path),
		Title:    titleFromMarkdown(path, markdown),
		Path:     path,
		Markdown: markdown,
	}, nil
}

func (s *Store) UpdatePreferences(request UpdatePreferencesRequest) (AppConfig, error) {
	if err := s.ensureStorage(); err != nil {
		return AppConfig{}, err
	}

	config, err := s.readConfig()
	if err != nil {
		return AppConfig{}, err
	}

	if strings.TrimSpace(request.PreferredMode) != "" {
		config.PreferredMode = request.PreferredMode
	}
	config.Autosave = request.Autosave
	switch strings.TrimSpace(request.Theme) {
	case "light", "dark", AppName:
		config.Theme = request.Theme
	}

	if err := s.writeConfig(config); err != nil {
		return AppConfig{}, err
	}

	return config, nil
}

func (s *Store) RegisterOpenedNote(path string) error {
	if err := s.ensureStorage(); err != nil {
		return err
	}

	return s.upsertNote(path, false)
}

func (s *Store) listNotes() ([]NoteSummary, error) {
	db, err := s.readNotesDB()
	if err != nil {
		return nil, err
	}

	notes := make([]NoteSummary, 0, len(db.Notes))
	for _, note := range db.Notes {
		title := titleFromPath(note.Path)
		if markdown, readErr := os.ReadFile(note.Path); readErr == nil {
			title = titleFromMarkdown(note.Path, string(markdown))
		}

		notes = append(notes, NoteSummary{
			ID:           noteID(note.Path),
			Title:        title,
			Path:         note.Path,
			CreatedAt:    note.CreatedAt,
			UpdatedAt:    note.UpdatedAt,
			UpdatedLabel: humanTime(note.UpdatedAt),
		})
	}

	sort.Slice(notes, func(i, j int) bool {
		return notes[i].UpdatedAt > notes[j].UpdatedAt
	})

	return notes, nil
}

func (s *Store) upsertNote(path string, markUpdated bool) error {
	db, err := s.readNotesDB()
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	found := false
	for index := range db.Notes {
		if db.Notes[index].Path == path {
			if db.Notes[index].CreatedAt == "" {
				db.Notes[index].CreatedAt = fileTimestampOrNow(path, now)
			}

			if markUpdated {
				db.Notes[index].UpdatedAt = now
			}

			if db.Notes[index].UpdatedAt == "" {
				db.Notes[index].UpdatedAt = fileTimestampOrNow(path, now)
			}

			found = true
			break
		}
	}

	if !found {
		createdAt := fileTimestampOrNow(path, now)
		updatedAt := createdAt
		if markUpdated {
			updatedAt = now
		}

		db.Notes = append(db.Notes, storedNote{
			Path:      path,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	db.LastOpenedPath = path

	sort.Slice(db.Notes, func(i, j int) bool {
		return db.Notes[i].UpdatedAt > db.Notes[j].UpdatedAt
	})

	return s.writeNotesDB(db)
}

func (s *Store) ensureStorage() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(configDir, AppName)
	notesDir := filepath.Join(baseDir, "notes")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return err
	}
	return os.MkdirAll(notesDir, 0o755)
}

func (s *Store) configDir() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(root, AppName), nil
}

func (s *Store) configPath() (string, error) {
	dir, err := s.configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func (s *Store) notesDBPath() (string, error) {
	dir, err := s.configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "notes.json"), nil
}

func (s *Store) readConfig() (AppConfig, error) {
	path, err := s.configPath()
	if err != nil {
		return AppConfig{}, err
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		config := AppConfig{
			Version:       1,
			PreferredMode: "Document",
			Autosave:      false,
			Theme:         AppName,
		}

		if writeErr := s.writeConfig(config); writeErr != nil {
			return AppConfig{}, writeErr
		}

		return config, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, err
	}

	config := AppConfig{}
	if err := json.Unmarshal(data, &config); err != nil {
		return AppConfig{}, err
	}
	if strings.TrimSpace(config.Theme) == "" {
		config.Theme = AppName
	}
	return config, nil
}

func (s *Store) writeConfig(config AppConfig) error {
	path, err := s.configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) readNotesDB() (notesDB, error) {
	path, err := s.notesDBPath()
	if err != nil {
		return notesDB{}, err
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		db := notesDB{Version: 1}
		if writeErr := s.writeNotesDB(db); writeErr != nil {
			return notesDB{}, writeErr
		}
		return db, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return notesDB{}, err
	}

	db := notesDB{}
	if err := json.Unmarshal(data, &db); err != nil {
		return notesDB{}, err
	}

	return db, nil
}

func (s *Store) writeNotesDB(db notesDB) error {
	path, err := s.notesDBPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func noteID(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strings.ReplaceAll(name, " ", "-")
}

func titleFromPath(path string) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if name == "" {
		return "Untitled"
	}
	return name
}

func titleFromMarkdown(path string, markdown string) string {
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			if title != "" {
				return title
			}
		}
	}
	return titleFromPath(path)
}

func humanTime(value string) string {
	if value == "" {
		return "Recently"
	}

	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "Recently"
	}

	now := time.Now()
	if now.Sub(timestamp) < 24*time.Hour {
		return "Today"
	}
	if now.Sub(timestamp) < 48*time.Hour {
		return "Yesterday"
	}
	return timestamp.Format("2006-01-02")
}

func fileTimestampOrNow(path string, fallback string) string {
	info, err := os.Stat(path)
	if err != nil {
		return fallback
	}
	return info.ModTime().Format(time.RFC3339)
}
