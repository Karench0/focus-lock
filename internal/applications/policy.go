package applications

import (
	"path/filepath"
	"strings"

	"github.com/Karench0/focus-lock/internal/model"
)

type Process struct {
	PID     uint32 `json:"pid"`
	Parent  uint32 `json:"parent"`
	Path    string `json:"path"`
	Name    string `json:"name"`
	Created int64  `json:"-"`
	Safe    bool   `json:"-"`
	Visible bool   `json:"-"`
}

func (p Process) Key() string { return strings.ToLower(p.Path) }
func Browser(path string) string {
	switch strings.ToLower(filepath.Base(path)) {
	case "chrome.exe":
		return "chrome"
	case "msedge.exe":
		return "edge"
	case "firefox.exe", "brave.exe", "opera.exe", "vivaldi.exe", "arc.exe", "browser.exe", "waterfox.exe", "librewolf.exe", "chromium.exe":
		return "unsupported"
	}
	return ""
}

type BrowserCheck func(Process, []Process) bool

// Allowed never inherits permission from a browser. Other child permissions are
// opt-in and limited to the allowed program's installation directory.
func Allowed(p Process, all []Process, rules []model.Application, strict bool, healthy BrowserCheck) bool {
	if p.Safe {
		return true
	}
	if strict && Browser(p.Path) != "" && (healthy == nil || !healthy(p, all)) {
		return false
	}
	for _, a := range rules {
		if strings.EqualFold(p.Path, a.Path) {
			return true
		}
		if !a.AllowChildren || Browser(a.Path) != "" {
			continue
		}
		rel, err := filepath.Rel(filepath.Dir(a.Path), p.Path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			continue
		}
		pid := p.Parent
		visited := map[uint32]bool{}
		for depth := 0; depth < 32 && !visited[pid]; depth++ {
			visited[pid] = true
			found := false
			for _, parent := range all {
				if parent.PID == pid {
					if strings.EqualFold(parent.Path, a.Path) {
						return true
					}
					pid = parent.Parent
					found = true
					break
				}
			}
			if !found {
				break
			}
		}
	}
	return false
}
