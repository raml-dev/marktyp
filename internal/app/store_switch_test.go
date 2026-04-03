package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenDocumentAtPathSwitchesActiveDocument(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	store := NewStore()
	info := DefaultAppInfo()

	pathA := filepath.Join(configRoot, "a.md")
	pathB := filepath.Join(configRoot, "b.md")

	if _, err := store.SaveDocument(SaveDocumentRequest{Path: pathA, Title: "A", Markdown: "# A\n"}, info); err != nil {
		t.Fatalf("save A: %v", err)
	}
	if _, err := store.SaveDocument(SaveDocumentRequest{Path: pathB, Title: "B", Markdown: "# B\n"}, info); err != nil {
		t.Fatalf("save B: %v", err)
	}

	workspace, err := store.OpenDocumentAtPath(pathA, info)
	if err != nil {
		t.Fatalf("open A: %v", err)
	}

	if workspace.ActiveDoc.Path != pathA {
		t.Fatalf("expected active path %q, got %q", pathA, workspace.ActiveDoc.Path)
	}

	if workspace.ActiveDoc.Title != "A" {
		t.Fatalf("expected active title A, got %q", workspace.ActiveDoc.Title)
	}

	if _, err := os.Stat(pathA); err != nil {
		t.Fatalf("expected note A to exist: %v", err)
	}
}
