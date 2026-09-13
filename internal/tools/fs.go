package tools

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic replaces a file only after its complete contents are on disk.
// Existing permissions and symbolic-link targets are preserved.
func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	if target, err := filepath.EvalSymlinks(path); err == nil {
		path = target
	}
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".marktyp-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err = file.Chmod(mode); err != nil {
		return err
	}
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return replaceFile(file.Name(), path)
}
