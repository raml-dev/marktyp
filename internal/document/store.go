package document

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"marktyp/internal/tools"
)

// Store serializes complete load-mutate-save transactions.
type Store struct {
	mu   sync.Mutex
	root string
}

type notesDB struct {
	Version        int          `json:"version"`
	LastOpenedPath string       `json:"lastOpenedPath"`
	Notes          []storedNote `json:"notes"`
}

type storedNote struct {
	Path          string  `json:"path"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
	DraftMarkdown *string `json:"draftMarkdown,omitempty"`
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) getWorkspace(appInfo AppInfo) (WorkspaceData, error) {
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
	if pruned := pruneMissingNotes(&db); pruned {
		if err := s.writeNotesDB(db); err != nil {
			return WorkspaceData{}, err
		}
	}

	// Build a deduplicated list of candidate paths to try, in priority order.
	seen := make(map[string]bool)
	var candidates []string
	for _, p := range []string{config.LastOpenedPath, db.LastOpenedPath} {
		if p != "" && !seen[p] {
			seen[p] = true
			candidates = append(candidates, p)
		}
	}
	for _, note := range db.Notes {
		if note.Path != "" && !seen[note.Path] {
			seen[note.Path] = true
			candidates = append(candidates, note.Path)
		}
	}

	var document DocumentState
	activePath := ""
	for _, p := range candidates {
		doc, docErr := s.getDocument(p)
		if docErr == nil {
			document = doc
			activePath = p
			break
		}
	}
	if activePath == "" {
		document = NewDocument()
	}

	if activePath != "" {
		if err := s.registerOpenedNote(activePath); err != nil {
			return WorkspaceData{}, err
		}
	}

	if config.LastOpenedPath != activePath {
		config.LastOpenedPath = activePath
		if err := s.writeConfig(config); err != nil {
			return WorkspaceData{}, err
		}
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

func (s *Store) createDocument(appInfo AppInfo) (WorkspaceData, error) {
	if err := s.ensureStorage(); err != nil {
		return WorkspaceData{}, err
	}

	baseDir, err := s.configDir()
	if err != nil {
		return WorkspaceData{}, err
	}
	notesDir := filepath.Join(baseDir, "notes")
	var path string
	for index := 1; ; index++ {
		name := "Untitled.md"
		if index > 1 {
			name = "Untitled " + strconv.Itoa(index) + ".md"
		}
		candidate := filepath.Join(notesDir, name)
		file, openErr := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(openErr, os.ErrExist) {
			continue
		}
		if openErr != nil {
			return WorkspaceData{}, openErr
		}
		if _, writeErr := file.WriteString("# Untitled\n\n"); writeErr != nil {
			file.Close()
			os.Remove(candidate)
			return WorkspaceData{}, writeErr
		}
		if syncErr := file.Sync(); syncErr != nil {
			file.Close()
			os.Remove(candidate)
			return WorkspaceData{}, syncErr
		}
		if closeErr := file.Close(); closeErr != nil {
			os.Remove(candidate)
			return WorkspaceData{}, closeErr
		}
		path = candidate
		break
	}

	if err := s.upsertNote(path, true, ""); err != nil {
		os.Remove(path)
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
	return s.getWorkspace(appInfo)
}

func (s *Store) openDocumentAtPath(path string, appInfo AppInfo) (WorkspaceData, error) {
	if err := s.ensureStorage(); err != nil {
		return WorkspaceData{}, err
	}

	path = strings.TrimSpace(path)
	if path == "" {
		return s.getWorkspace(appInfo)
	}
	if _, err := os.Stat(path); err != nil {
		return WorkspaceData{}, err
	}

	if err := s.registerOpenedNote(path); err != nil {
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

	return s.getWorkspace(appInfo)
}

func (s *Store) saveDocument(request SaveDocumentRequest, appInfo AppInfo) (WorkspaceData, error) {
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

	if err := tools.WriteFileAtomic(path, []byte(request.Markdown), 0o644); err != nil {
		return WorkspaceData{}, err
	}

	if err := s.upsertNote(path, true, ""); err != nil {
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

	return s.getWorkspace(appInfo)
}

func (s *Store) renameNote(request RenameNoteRequest, appInfo AppInfo) (WorkspaceData, error) {
	if err := s.ensureStorage(); err != nil {
		return WorkspaceData{}, err
	}

	path := strings.TrimSpace(request.Path)
	title := normalizeTitle(request.Title)
	if path == "" {
		return WorkspaceData{}, errors.New("missing note path")
	}
	if title == "" {
		return WorkspaceData{}, errors.New("note title cannot be empty")
	}

	db, err := s.readNotesDB()
	if err != nil {
		return WorkspaceData{}, err
	}
	noteIndex := -1
	for index := range db.Notes {
		if db.Notes[index].Path == path {
			noteIndex = index
			break
		}
	}
	if noteIndex < 0 {
		return WorkspaceData{}, errors.New("note not found")
	}

	markdown, err := os.ReadFile(path)
	if err != nil {
		return WorkspaceData{}, err
	}
	if err := tools.WriteFileAtomic(path, []byte(replaceMarkdownTitle(string(markdown), title)), 0o644); err != nil {
		return WorkspaceData{}, err
	}

	if draft := db.Notes[noteIndex].DraftMarkdown; draft != nil {
		renamedDraft := replaceMarkdownTitle(*draft, title)
		db.Notes[noteIndex].DraftMarkdown = &renamedDraft
	}
	db.Notes[noteIndex].UpdatedAt = time.Now().Format(time.RFC3339)
	if err := s.writeNotesDB(db); err != nil {
		return WorkspaceData{}, err
	}

	return s.getWorkspace(appInfo)
}

func (s *Store) deleteNote(path string, appInfo AppInfo) (WorkspaceData, error) {
	if err := s.ensureStorage(); err != nil {
		return WorkspaceData{}, err
	}

	target := strings.TrimSpace(path)
	if target == "" {
		return s.getWorkspace(appInfo)
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
		return s.getWorkspace(appInfo)
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

	return s.getWorkspace(appInfo)
}

func (s *Store) getDocument(path string) (DocumentState, error) {
	if err := s.ensureStorage(); err != nil {
		return DocumentState{}, err
	}

	markdownBytes, err := os.ReadFile(path)
	if err != nil {
		return DocumentState{}, err
	}

	db, err := s.readNotesDB()
	if err != nil {
		return DocumentState{}, err
	}

	draft := ""
	hasDraft := false
	for _, note := range db.Notes {
		if note.Path == path {
			if note.DraftMarkdown != nil {
				draft = *note.DraftMarkdown
				hasDraft = true
			}
			break
		}
	}

	markdown := string(markdownBytes)
	return DocumentState{
		ID:            noteID(path),
		Title:         titleFromMarkdown(path, markdown),
		Path:          path,
		Markdown:      markdown,
		DraftMarkdown: draft,
		HasDraft:      hasDraft,
	}, nil
}

func (s *Store) saveDraft(path string, markdown string) error {
	if err := s.ensureStorage(); err != nil {
		return err
	}

	return s.upsertNote(path, false, markdown)
}

func (s *Store) updatePreferences(request UpdatePreferencesRequest) (AppConfig, error) {
	if err := s.ensureStorage(); err != nil {
		return AppConfig{}, err
	}

	config, err := s.readConfig()
	if err != nil {
		return AppConfig{}, err
	}
	previous := config

	if strings.TrimSpace(request.PreferredMode) != "" {
		config.PreferredMode = request.PreferredMode
	}
	config.Autosave = request.Autosave
	config.CheckForUpdates = request.CheckForUpdates
	config.IncludePrereleaseUpdates = request.IncludePrereleaseUpdates
	switch strings.TrimSpace(request.Theme) {
	case "light", "dark", AppName:
		config.Theme = request.Theme
	}

	if config != previous {
		if err := s.writeConfig(config); err != nil {
			return AppConfig{}, err
		}
	}

	return config, nil
}

func (s *Store) registerOpenedNote(path string) error {
	if err := s.ensureStorage(); err != nil {
		return err
	}

	db, err := s.readNotesDB()
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	found := false
	metadataChanged := false
	for index := range db.Notes {
		if db.Notes[index].Path == path {
			if db.Notes[index].CreatedAt == "" {
				db.Notes[index].CreatedAt = fileTimestampOrNow(path, now)
				metadataChanged = true
			}
			if db.Notes[index].UpdatedAt == "" {
				db.Notes[index].UpdatedAt = fileTimestampOrNow(path, now)
				metadataChanged = true
			}
			found = true
			break
		}
	}

	if !found {
		createdAt := fileTimestampOrNow(path, now)
		db.Notes = append(db.Notes, storedNote{
			Path:      path,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		})
	}

	if found && !metadataChanged && db.LastOpenedPath == path {
		return nil
	}
	db.LastOpenedPath = path

	sort.Slice(db.Notes, func(i, j int) bool {
		return db.Notes[i].UpdatedAt > db.Notes[j].UpdatedAt
	})

	return s.writeNotesDB(db)
}

func pruneMissingNotes(db *notesDB) bool {
	kept := db.Notes[:0]
	seen := make(map[string]struct{}, len(db.Notes))
	changed := false
	for _, note := range db.Notes {
		_, duplicate := seen[note.Path]
		if note.Path == "" || duplicate {
			changed = true
			continue
		}
		seen[note.Path] = struct{}{}
		if _, err := os.Stat(note.Path); errors.Is(err, os.ErrNotExist) {
			changed = true
			if db.LastOpenedPath == note.Path {
				db.LastOpenedPath = ""
			}
			continue
		}
		kept = append(kept, note)
	}
	db.Notes = kept
	return changed
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

func (s *Store) upsertNote(path string, markUpdated bool, draft string) error {
	db, err := s.readNotesDB()
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	found := false
	for index := range db.Notes {
		if db.Notes[index].Path == path {
			if !markUpdated && db.Notes[index].CreatedAt != "" && db.Notes[index].UpdatedAt != "" && db.Notes[index].DraftMarkdown != nil && *db.Notes[index].DraftMarkdown == draft {
				return nil
			}
			if db.Notes[index].CreatedAt == "" {
				db.Notes[index].CreatedAt = fileTimestampOrNow(path, now)
			}

			if markUpdated {
				db.Notes[index].UpdatedAt = now
			}

			if db.Notes[index].UpdatedAt == "" {
				db.Notes[index].UpdatedAt = fileTimestampOrNow(path, now)
			}

			if markUpdated {
				db.Notes[index].DraftMarkdown = nil
			} else {
				db.Notes[index].DraftMarkdown = &draft
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
			Path:          path,
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
			DraftMarkdown: draftPointer(draft, markUpdated),
		})
	}

	if markUpdated {
		db.LastOpenedPath = path
	}

	sort.Slice(db.Notes, func(i, j int) bool {
		return db.Notes[i].UpdatedAt > db.Notes[j].UpdatedAt
	})

	return s.writeNotesDB(db)
}

func draftPointer(draft string, saved bool) *string {
	if saved {
		return nil
	}
	return &draft
}

func (s *Store) ensureStorage() error {
	baseDir, err := s.configDir()
	if err != nil {
		return err
	}
	notesDir := filepath.Join(baseDir, "notes")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return err
	}
	return os.MkdirAll(notesDir, 0o755)
}

func (s *Store) configDir() (string, error) {
	if s.root != "" {
		return s.root, nil
	}
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
			Version:         2,
			PreferredMode:   "Document",
			Autosave:        false,
			Theme:           AppName,
			CheckForUpdates: true,
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
	if config.Version < 2 {
		config.Version = 2
		config.CheckForUpdates = true
		if err := s.writeConfig(config); err != nil {
			return AppConfig{}, err
		}
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
	return tools.WriteFileAtomic(path, data, 0o600)
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
	return tools.WriteFileAtomic(path, data, 0o600)
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
			title := normalizeTitle(strings.TrimPrefix(trimmed, "# "))
			if title != "" {
				return title
			}
		}
	}
	return titleFromPath(path)
}

func replaceMarkdownTitle(markdown string, title string) string {
	lineEnding := "\n"
	if strings.Contains(markdown, "\r\n") {
		lineEnding = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	for index, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			lines[index] = "# " + title
			return strings.Join(lines, lineEnding)
		}
	}

	body := strings.TrimLeft(strings.Join(lines, lineEnding), "\r\n")
	if body == "" {
		return "# " + title + lineEnding + lineEnding
	}
	return "# " + title + lineEnding + lineEnding + body
}

func normalizeTitle(title string) string {
	title = strings.Join(strings.Fields(title), " ")
	title = strings.TrimSpace(strings.TrimRight(title, "#"))
	// A block pasted into a heading can leave an unmatched HTML opener in old
	// documents. It is presentation noise, not part of the note title.
	title = strings.TrimSpace(strings.TrimLeft(title, "<"))
	return title
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
