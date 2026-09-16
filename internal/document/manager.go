package document

// GetWorkspace runs a serialized storage transaction.
func (s *Store) GetWorkspace(appInfo AppInfo) (WorkspaceData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getWorkspace(appInfo)
}

// CreateDocument creates and activates a distinct note in managed storage.
func (s *Store) CreateDocument(appInfo AppInfo) (WorkspaceData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createDocument(appInfo)
}

// OpenDocumentAtPath runs a serialized storage transaction.
func (s *Store) OpenDocumentAtPath(path string, appInfo AppInfo) (WorkspaceData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.openDocumentAtPath(path, appInfo)
}

// SaveDocument runs a serialized storage transaction.
func (s *Store) SaveDocument(request SaveDocumentRequest, appInfo AppInfo) (WorkspaceData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveDocument(request, appInfo)
}

// RenameNote updates a note's Markdown title in a serialized storage transaction.
func (s *Store) RenameNote(request RenameNoteRequest, appInfo AppInfo) (WorkspaceData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.renameNote(request, appInfo)
}

// DeleteNote runs a serialized storage transaction.
func (s *Store) DeleteNote(path string, appInfo AppInfo) (WorkspaceData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteNote(path, appInfo)
}

// GetDocument runs a serialized storage transaction.
func (s *Store) GetDocument(path string) (DocumentState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getDocument(path)
}

// SaveDraft runs a serialized storage transaction.
func (s *Store) SaveDraft(path string, markdown string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveDraft(path, markdown)
}

// UpdatePreferences runs a serialized storage transaction.
func (s *Store) UpdatePreferences(request UpdatePreferencesRequest) (AppConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updatePreferences(request)
}

// GetConfig returns the persisted application configuration.
func (s *Store) GetConfig() (AppConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureStorage(); err != nil {
		return AppConfig{}, err
	}
	return s.readConfig()
}

// RegisterOpenedNote runs a serialized storage transaction.
func (s *Store) RegisterOpenedNote(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.registerOpenedNote(path)
}
