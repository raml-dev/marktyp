package exporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTMLTitleIsEscaped(t *testing.T) {
	html := wrapHTMLDocument(`</title><script>alert(1)</script>`, "<p>Body</p>")
	if strings.Contains(html, "<script>") {
		t.Fatal("export title injected active HTML")
	}
	if !strings.Contains(html, "&lt;/title&gt;") || !strings.Contains(html, "<p>Body</p>") {
		t.Fatal("title escaping or body changed")
	}
}

func TestPDFFallsBackWhenBrowserIsUnavailable(t *testing.T) {
	original := browserExecutable
	browserExecutable = func() (string, bool) { return "", false }
	t.Cleanup(func() { browserExecutable = original })
	path := filepath.Join(t.TempDir(), "document.pdf")
	err := NewExporter().ExportPDFToPath(ExportPDFRequest{Title: "Fallback", Markdown: "# Fallback", HTML: "<h1>Fallback</h1>"}, path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		t.Fatalf("PDF not created: %v", err)
	}
}
