//go:build windows

package applications

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestProcessFixture(t *testing.T) {
	if os.Getenv("FOCUS_LOCK_TEST_HELPER") == "1" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
}
func TestOnlyOwnedDisposableFixtureIsTerminated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessFixture$")
	cmd.Env = append(os.Environ(), "FOCUS_LOCK_TEST_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	p, err := Inspect(uint32(cmd.Process.Pid), uint32(os.Getpid()))
	if err != nil {
		t.Fatal(err)
	}
	if p.Safe {
		t.Skip("Windows protects the test process in this environment (e.g. elevated or session 0)")
	}
	if Allowed(p, nil, nil, true, nil) {
		t.Fatal("unlisted process allowed")
	}
	protected := p
	protected.Safe = true
	if err := Terminate(protected); !errors.Is(err, ErrProcessProtected) {
		t.Fatalf("protected snapshot was not refused: %v", err)
	}
	if _, err := Inspect(p.PID, p.Parent); err != nil {
		t.Fatalf("protected fixture was terminated: %v", err)
	}
	stale := p
	stale.Created++
	if Terminate(stale) == nil {
		t.Fatal("recycled PID not rejected")
	}
	if err = Terminate(p); err != nil {
		t.Fatal(err)
	}
}

func TestProtectedProcessErrors(t *testing.T) {
	for _, err := range []error{ErrProcessProtected, windows.ERROR_ACCESS_DENIED, windows.ERROR_PRIVILEGE_NOT_HELD} {
		if !IsProtectedProcessError(fmt.Errorf("wrapped: %w", err)) {
			t.Errorf("protection error not recognized: %v", err)
		}
	}
	for _, err := range []error{nil, errors.New("Access is denied."), windows.ERROR_INVALID_PARAMETER, errors.New("process identity changed")} {
		if IsProtectedProcessError(err) {
			t.Errorf("unrelated error treated as protection: %v", err)
		}
	}
}

func TestSecurityProcessNamesStayExempt(t *testing.T) {
	for _, name := range []string{"DefenderSessionHelper.exe", "DEFENDERSESSIONHELPER.EXE", "MsMpEng.exe", "Taskmgr.exe"} {
		if !protectedProcessName(name) {
			t.Errorf("missing security exemption: %s", name)
		}
	}
	for _, name := range []string{"steam.exe", "Telegram.exe", "dota2.exe", "notdefendersessionhelper.exe"} {
		if protectedProcessName(name) {
			t.Errorf("ordinary app exempted: %s", name)
		}
	}
}
