package tools

import "testing"

func TestFilenameIsPortable(t *testing.T) {
	got := Filename(`  a/b:c*?"<>|\\d  `, "document", ".md")
	if got != "a-b-c-d.md" {
		t.Fatalf("unexpected filename %q", got)
	}
	if got := Filename("...", "document", ".pdf"); got != "document.pdf" {
		t.Fatalf("unexpected fallback %q", got)
	}
}
