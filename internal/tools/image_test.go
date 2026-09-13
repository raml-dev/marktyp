package tools

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImagePreviewResolvesRelativeRasterImage(t *testing.T) {
	root := t.TempDir()
	imagePath := filepath.Join(root, "pixel.png")
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(imagePath, png, 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := ImagePreview("pixel.png", filepath.Join(root, "note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(preview, "data:image/png;base64,") {
		t.Fatalf("unexpected preview %q", preview)
	}
}

func TestImagePreviewRejectsNonImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-image.png")
	if err := os.WriteFile(path, []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ImagePreview(path, ""); err == nil {
		t.Fatal("expected unsupported image error")
	}
}
