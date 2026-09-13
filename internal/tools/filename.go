package tools

import (
	"strings"
	"unicode"
)

// Filename creates a portable dialog filename from a document title.
func Filename(title, fallback, extension string) string {
	name := strings.TrimSpace(title)
	if name == "" {
		name = fallback
	}
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\\|?*`, r) {
			return '-'
		}
		return r
	}, name)
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	name = strings.Trim(name, ". -")
	if name == "" {
		name = fallback
	}
	const maxRunes = 120
	runes := []rune(name)
	if len(runes) > maxRunes {
		name = string(runes[:maxRunes])
	}
	return name + extension
}
