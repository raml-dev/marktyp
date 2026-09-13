package document

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspacePrunesMissingNotes(t *testing.T) {
	store, root := testStore(t)
	path := filepath.Join(root, "removed.md")
	if _, err := store.SaveDocument(SaveDocumentRequest{Path: path, Markdown: "# Removed"}, DefaultAppInfo()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	workspace, err := store.GetWorkspace(DefaultAppInfo())
	if err != nil {
		t.Fatal(err)
	}
	if len(workspace.Notes) != 0 || workspace.ActiveDoc.Path != "" || workspace.Config.LastOpenedPath != "" {
		t.Fatalf("stale note was retained: %+v", workspace)
	}
}

func TestOpenMissingDocumentDoesNotChangeSelection(t *testing.T) {
	store, root := testStore(t)
	path := filepath.Join(root, "current.md")
	if _, err := store.SaveDocument(SaveDocumentRequest{Path: path, Markdown: "# Current"}, DefaultAppInfo()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.OpenDocumentAtPath(filepath.Join(root, "missing.md"), DefaultAppInfo()); !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
	workspace, err := store.GetWorkspace(DefaultAppInfo())
	if err != nil || workspace.ActiveDoc.Path != path {
		t.Fatalf("selection changed: %+v, %v", workspace, err)
	}
}
