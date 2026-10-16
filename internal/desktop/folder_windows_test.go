//go:build windows

package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func manifest(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"manifest_version":3}`), 0600); err != nil {
		t.Fatal(err)
	}
}
func TestInstalledAndDevelopmentExtensionFolders(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "build", "bin")
	source := filepath.Join(root, "browser-extension")
	manifest(t, source)
	if err := os.WriteFile(filepath.Join(root, "wails.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := ExtensionFolder(bin)
	if err != nil || got != source {
		t.Fatal(got, err)
	}
	installed := filepath.Join(bin, "browser-extension")
	manifest(t, installed)
	got, err = ExtensionFolder(bin)
	if err != nil || got != installed {
		t.Fatal(got, err)
	}
}
func TestMissingFolderReturnsError(t *testing.T) {
	dir := t.TempDir()
	if _, err := ExtensionFolder(dir); err == nil {
		t.Fatal("missing folder silently succeeded")
	}
	if err := OpenExtensionFolder(dir); err == nil {
		t.Fatal("missing folder opened Explorer")
	}
}
func TestExplorerSmoke(t *testing.T) {
	dir := os.Getenv("FOCUS_LOCK_EXPLORER_SMOKE_DIR")
	if dir == "" {
		t.Skip("manual Explorer smoke test is opt-in")
	}
	if err := OpenExtensionFolder(dir); err != nil {
		t.Fatal(err)
	}
}
