package document

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func testStore(t *testing.T) (*Store, string) {
	t.Helper()
	root := t.TempDir()
	return &Store{root: filepath.Join(root, "config")}, root
}

func TestEmptyDraftSurvivesReloadAndDoesNotSwitchActiveNote(t *testing.T) {
	store, root := testStore(t)
	a, b := filepath.Join(root, "a.md"), filepath.Join(root, "b.md")
	for _, path := range []string{a, b} {
		if _, err := store.SaveDocument(SaveDocumentRequest{Path: path, Markdown: "# Saved"}, DefaultAppInfo()); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SaveDraft(a, ""); err != nil {
		t.Fatal(err)
	}
	workspace, err := store.GetWorkspace(DefaultAppInfo())
	if err != nil {
		t.Fatal(err)
	}
	if workspace.ActiveDoc.Path != b {
		t.Fatal("draft changed the selected note")
	}
	doc, err := store.GetDocument(a)
	if err != nil {
		t.Fatal(err)
	}
	if !doc.HasDraft || doc.DraftMarkdown != "" {
		t.Fatalf("empty draft lost: %+v", doc)
	}
	if _, err = store.SaveDocument(SaveDocumentRequest{Path: a, Markdown: ""}, DefaultAppInfo()); err != nil {
		t.Fatal(err)
	}
	doc, err = store.GetDocument(a)
	if err != nil || doc.HasDraft || doc.Markdown != "" {
		t.Fatalf("save did not clear draft: %+v, %v", doc, err)
	}
}

func TestSavePreservesExactContent(t *testing.T) {
	for _, content := range []string{"", "# Hello", "# Hello\n\n", "  text\r\n"} {
		t.Run(fmt.Sprintf("%q", content), func(t *testing.T) {
			store, root := testStore(t)
			path := filepath.Join(root, "note.md")
			if _, err := store.SaveDocument(SaveDocumentRequest{Path: path, Markdown: content}, DefaultAppInfo()); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != content {
				t.Fatalf("content changed: %q, %v", data, err)
			}
		})
	}
}

func TestConcurrentDraftsDoNotLoseNotes(t *testing.T) {
	store, root := testStore(t)
	const count = 16
	for i := 0; i < count; i++ {
		path := filepath.Join(root, fmt.Sprintf("%d.md", i))
		if _, err := store.SaveDocument(SaveDocumentRequest{Path: path, Markdown: "saved"}, DefaultAppInfo()); err != nil {
			t.Fatal(err)
		}
	}
	var group sync.WaitGroup
	for i := 0; i < count; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			if err := store.SaveDraft(filepath.Join(root, fmt.Sprintf("%d.md", i)), fmt.Sprintf("draft %d", i)); err != nil {
				t.Error(err)
			}
		}(i)
	}
	group.Wait()
	for i := 0; i < count; i++ {
		doc, err := store.GetDocument(filepath.Join(root, fmt.Sprintf("%d.md", i)))
		if err != nil || !doc.HasDraft || doc.DraftMarkdown != fmt.Sprintf("draft %d", i) {
			t.Errorf("draft %d lost: %+v, %v", i, doc, err)
		}
	}
}

func TestLegacyDraftFormatLoads(t *testing.T) {
	store, root := testStore(t)
	path := filepath.Join(root, "old.md")
	if _, err := store.SaveDocument(SaveDocumentRequest{Path: path, Markdown: "saved"}, DefaultAppInfo()); err != nil {
		t.Fatal(err)
	}
	dbPath, err := store.notesDBPath()
	if err != nil {
		t.Fatal(err)
	}
	legacy := fmt.Sprintf(`{"version":1,"notes":[{"path":%q,"draftMarkdown":"old draft"}]}`, path)
	if err = os.WriteFile(dbPath, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := store.GetDocument(path)
	if err != nil || !doc.HasDraft || doc.DraftMarkdown != "old draft" {
		t.Fatalf("legacy draft lost: %+v, %v", doc, err)
	}
}
