package document

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenameNoteUpdatesSavedDocumentAndDraft(t *testing.T) {
	root := t.TempDir()
	store := &Store{root: filepath.Join(root, "storage")}
	info := DefaultAppInfo()
	path := filepath.Join(root, "note.md")

	if _, err := store.SaveDocument(SaveDocumentRequest{Path: path, Markdown: "# Old title\n\nSaved body\n"}, info); err != nil {
		t.Fatalf("save note: %v", err)
	}
	if err := store.SaveDraft(path, "# Draft title\n\nDraft body\n"); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	workspace, err := store.RenameNote(RenameNoteRequest{Path: path, Title: "New title"}, info)
	if err != nil {
		t.Fatalf("rename note: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read renamed note: %v", err)
	}
	if got := string(data); got != "# New title\n\nSaved body\n" {
		t.Fatalf("unexpected saved markdown: %q", got)
	}
	if workspace.ActiveDoc.DraftMarkdown != "# New title\n\nDraft body\n" {
		t.Fatalf("unexpected renamed draft: %q", workspace.ActiveDoc.DraftMarkdown)
	}
	if workspace.ActiveDoc.Title != "New title" || workspace.Notes[0].Title != "New title" {
		t.Fatalf("renamed title not reflected in workspace: %#v", workspace)
	}
}

func TestTitleFromMarkdownRemovesStrayOpeningAngleBracket(t *testing.T) {
	if got := titleFromMarkdown("note.md", "# <Pasted title\n\nBody"); got != "Pasted title" {
		t.Fatalf("expected clean title, got %q", got)
	}
}
