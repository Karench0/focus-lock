//go:build windows

package desktop

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// ExtensionFolder resolves packaged assets first, then the source tree used by
// wails dev. It never silently falls back to the user's current directory.
func ExtensionFolder(executableDir string) (string, error) {
	candidates := []string{filepath.Join(executableDir, "browser-extension")}
	root := filepath.Clean(filepath.Join(executableDir, "..", ".."))
	if info, err := os.Stat(filepath.Join(root, "wails.json")); err == nil && !info.IsDir() {
		candidates = append(candidates, filepath.Join(root, "browser-extension"))
	}
	for _, candidate := range candidates {
		path, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(filepath.Join(path, "manifest.json")); err == nil && info.Mode().IsRegular() {
			return path, nil
		}
	}
	return "", fmt.Errorf("Папка расширения не найдена / Extension folder not found. Extract the complete release archive or reinstall Focus Lock")
}

func OpenExtensionFolder(executableDir string) error {
	path, err := ExtensionFolder(executableDir)
	if err != nil {
		return err
	}
	verb, _ := windows.UTF16PtrFromString("open")
	target, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	if err = windows.ShellExecute(0, verb, target, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("Не удалось открыть Проводник / Could not open Explorer: %w", err)
	}
	return nil
}
