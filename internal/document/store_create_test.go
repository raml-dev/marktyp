package document

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateDocumentAlwaysCreatesADistinctManagedNote(t *testing.T) {
	root := t.TempDir()
	store := &Store{root: filepath.Join(root, "storage")}
	info := DefaultAppInfo()

	first, err := store.CreateDocument(info)
	if err != nil {
		t.Fatalf("create first note: %v", err)
	}
	second, err := store.CreateDocument(info)
	if err != nil {
		t.Fatalf("create second note: %v", err)
	}

	if first.ActiveDoc.Path == "" || second.ActiveDoc.Path == "" {
		t.Fatal("created notes must have persisted paths")
	}
	if first.ActiveDoc.Path == second.ActiveDoc.Path {
		t.Fatalf("expected distinct paths, got %q", first.ActiveDoc.Path)
	}
	if len(second.Notes) != 2 {
		t.Fatalf("expected two notes, got %d", len(second.Notes))
	}
	for _, path := range []string{first.ActiveDoc.Path, second.ActiveDoc.Path} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("created note %q is unavailable: %v", path, err)
		}
	}
}
