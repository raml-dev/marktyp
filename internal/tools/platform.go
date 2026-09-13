package tools

import (
	"context"
	"errors"
	"net/url"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func RevealNoteInFS(ctx context.Context, path string) error {
	if ctx == nil {
		return errors.New("ctx not initialized")
	}
	target := strings.TrimSpace(path)
	if target == "" {
		return errors.New("missing path")
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}

	dir := filepath.Dir(absTarget)
	commandContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch goruntime.GOOS {
	case "darwin":
		return exec.CommandContext(commandContext, "open", "-R", absTarget).Run()
	case "windows":
		return exec.CommandContext(commandContext, "explorer", "/select,"+filepath.Clean(absTarget)).Run()
	default:
		if err := runFirstAvailableCommand(commandContext,
			[]string{"xdg-open", dir},
			[]string{"/usr/bin/xdg-open", dir},
			[]string{"gio", "open", dir},
			[]string{"nautilus", "--select", absTarget},
			[]string{"dolphin", "--select", absTarget},
		); err == nil {
			return nil
		}

		fileURL := (&url.URL{Scheme: "file", Path: dir}).String()
		wruntime.BrowserOpenURL(ctx, fileURL)
		return nil
	}
}

func runFirstAvailableCommand(ctx context.Context, candidates ...[]string) error {
	var lastErr error
	for _, candidate := range candidates {
		if len(candidate) == 0 {
			continue
		}

		binary := candidate[0]
		if !strings.Contains(binary, "/") {
			if _, err := exec.LookPath(binary); err != nil {
				lastErr = err
				continue
			}
		}

		cmd := exec.CommandContext(ctx, binary, candidate[1:]...)
		if err := cmd.Run(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return errors.New("no reveal command available")
}
