//go:build windows

package bridge

import (
	"net"
	"testing"
	"time"

	"github.com/Karench0/focus-lock/internal/applications"
)

func TestStudyCapabilityBelongsToTheBrowserRoot(t *testing.T) {
	old := applications.Process{PID: 100, Created: 1, Path: `C:\Browser\chrome.exe`}
	updated := applications.Process{PID: 200, Created: 2, Path: old.Path}
	child := applications.Process{PID: 201, Parent: 200, Created: 3, Path: old.Path}
	all := []applications.Process{old, updated, child}
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	s := &Server{clients: map[net.Conn]Status{
		a: {PID: old.PID, Created: old.Created, Path: old.Path, Connected: true, Applied: "study", LastSeen: time.Now()},
		b: {PID: updated.PID, Created: updated.Created, Path: updated.Path, Connected: true, StudyCapable: true, Applied: "study", LastSeen: time.Now()},
	}}
	if !s.Healthy(old, all, "study") || s.HealthyStudy(old, all, "study") {
		t.Fatal("another profile's upgraded extension authorized an old browser root")
	}
	if !s.HealthyStudy(updated, all, "study") || !s.HealthyStudy(child, all, "study") {
		t.Fatal("updated browser root or its child rejected")
	}
	if s.HealthyStudy(child, all, "other-session") {
		t.Fatal("stale session accepted")
	}
	all[1].Created++
	if s.HealthyStudy(child, all, "study") {
		t.Fatal("recycled browser PID accepted")
	}
}
