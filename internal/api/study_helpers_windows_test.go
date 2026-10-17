//go:build windows

package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Karench0/focus-lock/internal/applications"
	"github.com/Karench0/focus-lock/internal/model"
)

func TestStudyAllowsOnlyTheInstalledIDEBundledRuntime(t *testing.T) {
	root := t.TempDir()
	ide := filepath.Join(root, "IDE", "bin", "idea64.exe")
	helper := filepath.Join(root, "IDE", "jbr", "bin", "java.exe")
	outside := filepath.Join(root, "OtherJDK", "bin", "java.exe")
	for _, path := range []string{ide, helper, outside} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("MZ fixture, never executed"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	p := studyProfile(model.Application{Name: "Browser", Path: `C:\Browser\chrome.exe`}, []model.Application{{Name: "IDE", Path: ide}})
	wanted, err := applications.Canonical(helper)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, app := range p.Applications {
		if strings.EqualFold(app.Path, wanted) {
			found = true
		}
	}
	if !found {
		t.Fatal("Kotlin IDE's bundled JVM was not allowed")
	}
	check := func(applications.Process, []applications.Process) bool { return true }
	if applications.Allowed(applications.Process{Path: outside}, nil, p.Applications, true, check) {
		t.Fatal("unrelated JDK received study permission")
	}
}
