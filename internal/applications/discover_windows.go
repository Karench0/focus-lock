//go:build windows

package applications

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Karench0/focus-lock/internal/model"
	"golang.org/x/sys/windows/registry"
)

func Discover() []model.Application {
	out := []model.Application{}
	seen := map[string]bool{}
	add := func(name, path string) {
		if e := strings.Index(strings.ToLower(path), ".exe"); e >= 0 {
			path = path[:e+4]
		}
		path = strings.Trim(path, " \"")
		if expanded, err := registry.ExpandString(path); err == nil {
			path = expanded
		}
		p, err := Canonical(path)
		if err != nil || seen[strings.ToLower(p)] {
			return
		}
		seen[strings.ToLower(p)] = true
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		}
		out = append(out, model.Application{Name: name, Path: p, Icon: Icon(p)})
	}
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, view := range []uint32{registry.WOW64_64KEY, registry.WOW64_32KEY} {
			for _, base := range []string{`SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths`, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`} {
				key, err := registry.OpenKey(root, base, registry.READ|view)
				if err != nil {
					continue
				}
				names, _ := key.ReadSubKeyNames(-1)
				for _, name := range names {
					sub, err := registry.OpenKey(key, name, registry.READ|view)
					if err != nil {
						continue
					}
					if strings.HasSuffix(base, "App Paths") {
						p, _, _ := sub.GetStringValue("")
						add("", p)
					} else {
						n, _, _ := sub.GetStringValue("DisplayName")
						p, _, _ := sub.GetStringValue("DisplayIcon")
						if n != "" {
							add(n, p)
						}
					}
					_ = sub.Close()
				}
				_ = key.Close()
			}
		}
	}
	// Portable programs may be selected manually; only shallow common locations
	// are searched to avoid recursively inspecting the user's personal files.
	for _, dir := range []string{filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs"), os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
		entries, _ := os.ReadDir(dir)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			files, _ := filepath.Glob(filepath.Join(dir, entry.Name(), "*.exe"))
			for _, f := range files {
				base := strings.ToLower(filepath.Base(f))
				if strings.Contains(base, "unins") || strings.Contains(base, "setup") || strings.Contains(base, "update") || strings.Contains(base, "crash") {
					continue
				}
				add("", f)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}
