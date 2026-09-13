package document

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteNoteRemovesFileFromFilesystem(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	store := &Store{root: filepath.Join(configRoot, "storage")}
	info := DefaultAppInfo()

	targetPath := filepath.Join(configRoot, "external-note.md")
	if _, err := store.SaveDocument(SaveDocumentRequest{
		Path:     targetPath,
		Title:    "External",
		Markdown: "# External\n",
	}, info); err != nil {
		t.Fatalf("save note: %v", err)
	}

	if _, err := store.DeleteNote(targetPath, info); err != nil {
		t.Fatalf("delete note: %v", err)
	}

	if _, err := os.Stat(targetPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be removed, got err=%v", err)
	}
}
