package api

import (
	"testing"
	"time"

	"github.com/Karench0/focus-lock/internal/applications"
)

func TestNativeHeartbeatDoesNotWaitForServiceWork(t *testing.T) {
	f := newStudyFixture(t)
	s := f.service
	plan, _ := s.PrepareStorozhenko()
	if err := s.StartStorozhenko(plan.Token, "storozhenko", true); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	done := make(chan bool, 1)
	go func() {
		rules := s.rules()
		s.siteBlocks(1)
		done <- rules.Active && rules.Exercise && rules.Language == "en"
	}()
	select {
	case ok := <-done:
		s.mu.Unlock()
		if !ok {
			t.Fatal("incorrect rules while service busy")
		}
	case <-time.After(500 * time.Millisecond):
		s.mu.Unlock()
		<-done
		t.Fatal("native heartbeat waits on process/discovery/service work")
	}
}

func TestStudyReviewAllowsRestartedInstancesButNotUnreviewedExecutables(t *testing.T) {
	reviewed := []applications.Process{{PID: 1, Created: 10, Path: `C:\Steam\steamwebhelper.exe`}}
	restarted := []applications.Process{{PID: 2, Created: 20, Path: `c:\steam\STEAMWEBHELPER.exe`}}
	if err := reviewedStudyApplications(reviewed, restarted); err != nil {
		t.Fatalf("reviewed helper restart forces another start attempt: %v", err)
	}
	for _, path := range []string{`D:\Other\steamwebhelper.exe`, `C:\Unreviewed\editor.exe`, ""} {
		if err := reviewedStudyApplications(reviewed, []applications.Process{{PID: 1, Created: 10, Path: path}}); err == nil {
			t.Fatalf("unreviewed executable authorized by old PID or basename: %q", path)
		}
	}
}
