package tools

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const maxPreviewImageSize = 25 << 20

// ImagePreview returns a browser-safe data URL for a local raster image.
func ImagePreview(source, documentPath string) (string, error) {
	path, err := resolveImagePath(source, documentPath)
	if err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if info.IsDir() || info.Size() > maxPreviewImageSize {
		return "", errors.New("image is not a file or exceeds 25 MB")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxPreviewImageSize+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxPreviewImageSize {
		return "", errors.New("image exceeds 25 MB")
	}
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp":
	default:
		return "", fmt.Errorf("unsupported image type %q", contentType)
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func resolveImagePath(source, documentPath string) (string, error) {
	value := strings.TrimSpace(source)
	if value == "" {
		return "", errors.New("missing image path")
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme == "file" {
		value = parsed.Path
	}
	if !filepath.IsAbs(value) {
		if strings.TrimSpace(documentPath) == "" {
			return "", errors.New("relative image requires a saved document")
		}
		value = filepath.Join(filepath.Dir(documentPath), filepath.FromSlash(value))
	}
	return filepath.Clean(value), nil
}
