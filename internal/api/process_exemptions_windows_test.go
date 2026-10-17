//go:build windows

package api

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Karench0/focus-lock/internal/applications"
	"github.com/Karench0/focus-lock/internal/model"
	"github.com/Karench0/focus-lock/internal/session"
	"golang.org/x/sys/windows"
)

func TestProtectedProcessDoesNotInterruptRemainingStudyClosures(t *testing.T) {
	for _, denial := range []error{windows.ERROR_ACCESS_DENIED, windows.ERROR_PRIVILEGE_NOT_HELD, applications.ErrProcessProtected, applications.ErrProcessGone} {
		t.Run(denial.Error(), func(t *testing.T) {
			s := &Service{engine: session.New(time.Now), failed: map[uint32]int64{}}
			before, err := s.engine.StartExercise(model.Profile{ID: "test"})
			if err != nil {
				t.Fatal(err)
			}
			// Simulate protection appearing after consent. These are synthetic
			// records; no Defender, Steam or Telegram process is touched.
			processes := []applications.Process{
				{PID: 101, Created: 1001, Name: "DefenderSessionHelper.exe"},
				{PID: 102, Created: 1002, Name: "steam.exe"},
				{PID: 103, Created: 1003, Name: "Telegram.exe"},
			}
			var attempted []string
			err = s.closeStudyApplications(processes, func(p applications.Process) error {
				attempted = append(attempted, p.Name)
				if p.PID == 101 {
					return fmt.Errorf("OpenProcess: %w", denial)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("protected process cancelled startup: %v", err)
			}
			if !reflect.DeepEqual(attempted, []string{"DefenderSessionHelper.exe", "steam.exe", "Telegram.exe"}) {
				t.Fatalf("remaining ordinary apps were not closed: %v", attempted)
			}
			after := s.engine.Snapshot()
			if after.ID != before.ID || after.State != session.Active || after.AppBlocks != 2 || s.lastError != "" {
				t.Fatalf("exercise/counts changed after exemption: %+v, %q", after, s.lastError)
			}
			if s.failed[101] != 1001 {
				t.Fatal("denied process instance was not remembered for monitoring")
			}
			// Monitoring uses the same classification and must not turn a
			// later protected process into a session-ending error either.
			if err := s.recordTermination(applications.Process{PID: 104, Created: 1004}, denial); err != nil {
				t.Fatalf("monitoring did not exempt protected process: %v", err)
			}
			if s.engine.Snapshot().AppBlocks != 2 || s.failed[104] != 1004 {
				t.Fatal("protected process counted as closed or not remembered")
			}
		})
	}
}

func TestUnexpectedTerminationFailureStillReported(t *testing.T) {
	s := &Service{engine: session.New(time.Now), failed: map[uint32]int64{}}
	_, _ = s.engine.StartExercise(model.Profile{ID: "test"})
	failure := errors.New("unexpected termination failure")
	var attempted int
	err := s.closeStudyApplications([]applications.Process{{PID: 1, Created: 11, Name: "first.exe"}, {PID: 2, Created: 22}}, func(applications.Process) error {
		attempted++
		return failure
	})
	if !errors.Is(err, failure) || attempted != 1 || s.engine.Snapshot().AppBlocks != 0 {
		t.Fatalf("unexpected failure was swallowed: %v, attempts %d", err, attempted)
	}
}
